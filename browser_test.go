package dashboard_test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	dstarstatic "github.com/larsartmann/go-datastar/static"
	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	"github.com/samber/do/v2"
)

// findChrome returns a usable Chrome/Chromium executable or skips the test.
// Resolution order: GO_HEALTH_DASHBOARD_CHROME env var, then well-known
// binary names on PATH.
func findChrome(t *testing.T) string {
	t.Helper()

	if p := os.Getenv("GO_HEALTH_DASHBOARD_CHROME"); p != "" {
		if _, err := os.Stat(p); err == nil { //nolint:gosec // operator-provided test binary path
			return p
		}

		t.Fatalf("GO_HEALTH_DASHBOARD_CHROME is set but the file does not exist: %s", p)
	}

	for _, name := range []string{"google-chrome-stable", "google-chrome", "chromium", "chromium-browser"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}

	t.Skip(
		"no Chrome/Chromium binary found; set GO_HEALTH_DASHBOARD_CHROME to enable browser tests",
	)

	return ""
}

// freePort reserves an ephemeral TCP port and immediately releases it, so
// Chrome can be pinned to a concrete debugging port — this Chromium build
// never announces a DevTools websocket when asked for port 0.
func freePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve free port: %v", err)
	}

	defer listener.Close()

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("reserved address is not TCP: %v", listener.Addr())
	}

	return tcpAddr.Port
}

// browserSerial gates all browser tests: headless Chrome startups are
// heavyweight and contend for CPU with parallel launches on loaded machines,
// which historically pushed startup past the announce timeout. Running them
// one at a time trades a little wall time for reliability.
var browserSerial sync.Mutex

// startHeadlessChrome launches Chrome manually with a concrete DevTools port
// and returns the websocket debugger URL parsed from stderr. chromedp's own
// launcher queries the debugger over 127.0.0.1, which hangs when Chrome
// binds the DevTools listener to IPv6 ::1 only — parsing the announced URL
// avoids that failure mode entirely.
func startHeadlessChrome(t *testing.T, chromePath string) (string, func()) {
	t.Helper()

	browserSerial.Lock()

	// t.TempDir cleanup races Chrome's renderer children, which keep writing
	// into the profile after the browser process exits — removal is retried
	// in stopChrome instead.
	//nolint:usetesting // see above
	profileDir, err := os.MkdirTemp("", "go-health-dashboard-chrome-")
	if err != nil {
		browserSerial.Unlock()

		t.Fatalf("chrome profile dir: %v", err)
	}

	//nolint:gosec // chromePath comes from the operator's env or PATH, test-only
	cmd := exec.Command(chromePath,
		"--headless",
		"--no-sandbox",
		"--disable-gpu",
		"--remote-debugging-port="+strconv.Itoa(freePort(t)),
		"--user-data-dir="+profileDir,
		"about:blank",
	)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("chrome stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		browserSerial.Unlock()

		t.Fatalf("chrome start: %v", err)
	}

	lines := make(chan string)

	go func() {
		defer close(lines)

		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()

	timeout := time.After(45 * time.Second)

	// stopChrome terminates the browser and removes the profile. Renderer
	// child processes may outlive the browser process for a few milliseconds
	// and keep writing into the profile, so removal retries briefly before
	// giving up (the OS temp dir is the final safety net).
	stopChrome := func() {
		defer browserSerial.Unlock()

		_ = cmd.Process.Signal(os.Interrupt)
		_ = cmd.Wait()

		for range 3 {
			if err := os.RemoveAll(profileDir); err == nil {
				return
			}

			time.Sleep(100 * time.Millisecond)
		}
	}

	for {
		select {
		case <-timeout:
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			_ = os.RemoveAll(profileDir)
			browserSerial.Unlock()

			t.Fatal("chrome did not announce a DevTools websocket within 45s")
		case line, ok := <-lines:
			if !ok {
				_ = cmd.Wait()
				_ = os.RemoveAll(profileDir)
				browserSerial.Unlock()

				t.Fatal("chrome exited before announcing a DevTools websocket")
			}

			if url, found := strings.CutPrefix(line, "DevTools listening on "); found {
				return url, stopChrome
			}
		}
	}
}

// strictCSPMiddleware serves a locked-down CSP: no unsafe-inline for scripts
// or styles, everything self-hosted. 'unsafe-eval' is required because the
// Datastar SDK compiles its data-* expressions with the Function
// constructor — without it the bundle throws "GenerateExpression" during
// init and the SSE connection never opens.
func strictCSPMiddleware(nonce string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'nonce-"+nonce+"' 'unsafe-eval'; "+
				"style-src 'self'; "+
				"img-src 'self' data:; "+
				"connect-src 'self'; "+
				"font-src 'self'; "+
				"object-src 'none'; "+
				"base-uri 'self'")
		next.ServeHTTP(w, r)
	})
}

// TestBrowser_CSPCleanRuntime closes the runtime-CSP verification loop the
// CLI tests cannot: it loads the real page in a headless browser under a
// strict CSP and verifies that
//
//  1. the Datastar SDK executes and connects via SSE (a wrong or missing
//     nonce would get the inline script blocked and the connection would
//     never open),
//  2. the DOM stays free of inline style attributes and <style> elements at
//     runtime, including after Datastar applies SSE patches.
//
// The page is fully self-hosted (compiled CSS + embedded Datastar bundle), so
// the test runs hermetically without CDN access.
func TestBrowser_CSPCleanRuntime(t *testing.T) {
	t.Parallel()

	chromePath := findChrome(t)

	const nonce = "browser-test-nonce"

	s := setupDashboard(t,
		dashboard.WithNonce(nonce),
		dashboard.WithCSSPath("/static/app.css"),
		dashboard.WithDatastarSrc("/static/datastar.js"),
	)
	defer s.cleanup()

	s.mux.HandleFunc("/static/app.css", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_, _ = w.Write([]byte("body { margin: 0; }"))
	})

	s.mux.HandleFunc("/static/datastar.js", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/javascript")
		_, _ = w.Write(dstarstatic.Bytes())
	})

	server := httptest.NewServer(strictCSPMiddleware(nonce, s.mux))
	defer server.Close()

	wsURL, stopChrome := startHeadlessChrome(t, chromePath)
	defer stopChrome()

	runCtx, runCancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer runCancel()

	allocCtx, allocCancel := chromedp.NewRemoteAllocator(runCtx, wsURL)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	errLog := watchBrowserErrors(ctx)

	if err := chromedp.Run(ctx, chromedp.Navigate(server.URL+"/health")); err != nil {
		t.Fatalf("browser navigate: %v", err)
	}

	deadline := time.Now().Add(15 * time.Second)
	for s.dash.SubscriberCount() == 0 {
		if time.Now().After(deadline) {
			t.Fatal(
				"Datastar SDK never connected via SSE; CSP blocked the inline script or the connection",
			)
		}

		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(250 * time.Millisecond) // allow at least one SSE patch to apply

	var styleViolations, styleTags int64

	var bodyText string

	// The dark-mode pre-paint script sets `color-scheme` on <html> via the
	// CSSOM (el.style.*), which CSP does not restrict — only style markup in
	// the served HTML is. So <html> is the one allowed carrier of a style
	// attribute; any other styled element is a real leak that would break a
	// style-src policy without 'unsafe-inline'.
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(
			`[...document.querySelectorAll('[style]')].filter(e => e !== document.documentElement).length`,
			&styleViolations,
		),
		chromedp.Evaluate(`document.querySelectorAll('style').length`, &styleTags),
		chromedp.Evaluate(`document.body.innerText`, &bodyText),
	); err != nil {
		t.Fatalf("browser evaluate: %v", err)
	}

	if styleViolations != 0 {
		var styledHTML string

		if err := chromedp.Run(ctx,
			chromedp.Evaluate(
				`[...document.querySelectorAll('[style]')].map(e => e.outerHTML).join("\n---\n")`,
				&styledHTML,
			)); err != nil {
			styledHTML = "could not fetch styled elements: " + err.Error()
		}

		t.Errorf(
			"runtime DOM contains %d CSP-relevant elements with inline style attributes; want 0:\n%s",
			styleViolations,
			styledHTML,
		)
	}

	if styleTags != 0 {
		t.Errorf("runtime DOM contains %d <style> elements; want 0", styleTags)
	}

	if !strings.Contains(bodyText, "All Systems Operational") {
		t.Errorf("health content missing from live DOM; got: %.200s", bodyText)
	}

	assertNoBrowserErrors(t, errLog)
}

// --- Console / CSP-violation observation ---

// browserErrorLog records console.error calls and uncaught exceptions from
// the page. CSP violations surface as console errors ("Refused to ..."),
// so watching this channel catches both broken scripts and policy breaches.
type browserErrorLog struct {
	mu      sync.Mutex
	entries []string
}

// watchBrowserErrors attaches a target-event listener that collects page
// errors. It must be called before the first navigation.
func watchBrowserErrors(ctx context.Context) *browserErrorLog {
	log := &browserErrorLog{}

	chromedp.ListenTarget(ctx, func(ev any) {
		switch event := ev.(type) {
		case *runtime.EventConsoleAPICalled:
			if event.Type != "error" {
				return
			}

			var parts []string

			for _, arg := range event.Args {
				parts = append(parts, arg.Description)
			}

			log.mu.Lock()
			log.entries = append(log.entries, "console.error: "+strings.Join(parts, " "))
			log.mu.Unlock()
		case *runtime.EventExceptionThrown:
			if event.ExceptionDetails == nil {
				return
			}

			text := event.ExceptionDetails.Text

			if event.ExceptionDetails.Exception != nil {
				text += ": " + event.ExceptionDetails.Exception.Description
			}

			log.mu.Lock()
			log.entries = append(log.entries, "uncaught exception: "+text)
			log.mu.Unlock()
		}
	})

	return log
}

// all returns the collected entries so far.
func (l *browserErrorLog) all() []string {
	l.mu.Lock()
	defer l.mu.Unlock()

	return append([]string(nil), l.entries...)
}

// assertNoBrowserErrors fails the test when the page logged errors or threw.
func assertNoBrowserErrors(t *testing.T, log *browserErrorLog) {
	t.Helper()

	if entries := log.all(); len(entries) != 0 {
		t.Errorf(
			"browser logged %d error(s); want 0:\n%s",
			len(entries),
			strings.Join(entries, "\n"),
		)
	}
}

// --- Live SSE patch verification ---

// TestBrowser_LiveSSEPatch proves the dashboard's headline behavior
// end-to-end in a real browser: the page starts green, a service actually
// breaks, and the DOM updates to the degraded banner through the normal
// Datastar SSE patch stream — no reload, under a strict CSP, with a clean
// console throughout.
func TestBrowser_LiveSSEPatch(t *testing.T) {
	t.Parallel()

	chromePath := findChrome(t)

	const nonce = "browser-live-nonce"

	toggle := &toggleService{}
	toggle.healthy.Store(true)

	injector := do.New()
	provideToggleService(injector, "database", toggle)
	provideHealthy(injector, "redis")
	invoke[*healthyService](t, injector, "redis")

	probe := health.New(injector,
		health.WithVersion("2.1.0"),
		health.WithRefreshInterval(100*time.Millisecond),
	)
	dash := dashboard.New(probe,
		dashboard.WithNonce(nonce),
		dashboard.WithCSSPath("/static/app.css"),
		dashboard.WithDatastarSrc("/static/datastar.js"),
	)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	mux.HandleFunc("/static/app.css", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_, _ = w.Write([]byte("body { margin: 0; }"))
	})
	mux.HandleFunc("/static/datastar.js", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/javascript")
		_, _ = w.Write(dstarstatic.Bytes())
	})

	if err := probe.Start(t.Context()); err != nil {
		t.Fatalf("probe.Start: %v", err)
	}
	defer probe.Shutdown()

	if err := dash.Start(t.Context()); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}
	defer dash.Shutdown()

	server := httptest.NewServer(strictCSPMiddleware(nonce, mux))
	defer server.Close()

	wsURL, stopChrome := startHeadlessChrome(t, chromePath)
	defer stopChrome()

	runCtx, runCancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer runCancel()

	allocCtx, allocCancel := chromedp.NewRemoteAllocator(runCtx, wsURL)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	errLog := watchBrowserErrors(ctx)

	if err := chromedp.Run(ctx, chromedp.Navigate(server.URL+"/health")); err != nil {
		t.Fatalf("browser navigate: %v", err)
	}

	waitForSubscriber(t, dash)

	waitForBodyText(t, ctx, "All Systems Operational")

	toggle.healthy.Store(false)

	waitForBodyText(t, ctx, "Degraded")

	assertNoBrowserErrors(t, errLog)
}

// waitForSubscriber blocks until an SSE client connects or the test times out.
func waitForSubscriber(t *testing.T, dash *dashboard.Dashboard) {
	t.Helper()

	deadline := time.Now().Add(15 * time.Second)

	for dash.SubscriberCount() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("Datastar SDK never connected via SSE")
		}

		time.Sleep(50 * time.Millisecond)
	}
}

// waitForBodyText polls the live DOM until it contains the given text.
func waitForBodyText(t *testing.T, ctx context.Context, want string) {
	t.Helper()

	const query = `document.body.innerText`

	deadline := time.Now().Add(15 * time.Second)

	for {
		var bodyText string

		if err := chromedp.Run(ctx, chromedp.Evaluate(query, &bodyText)); err != nil {
			t.Fatalf("browser evaluate: %v", err)
		}

		if strings.Contains(bodyText, want) {
			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("live DOM never showed %q; got: %.300s", want, bodyText)
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// --- Accessibility ---

// axeCoreCDN is the pinned axe-core build injected into the page for the
// accessibility audit (verified available at this URL).
const axeCoreCDN = "https://cdnjs.cloudflare.com/ajax/libs/axe-core/4.10.2/axe.min.js"

// fetchAxeCore downloads the axe-core runtime so the audit runs fully
// same-origin (strict CSP blocks third-party script). It skips the test
// when the machine is offline; the targeted ARIA checks below still run.
func fetchAxeCore(t *testing.T) []byte {
	t.Helper()

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(axeCoreCDN)
	if err != nil {
		t.Skipf("axe-core unavailable (offline?): %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("axe-core CDN returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Skipf("axe-core download failed: %v", err)
	}

	return body
}

// TestBrowser_Accessibility runs two layers of accessibility checks:
//
//  1. targeted, hermetic assertions (html lang, landmarks, named buttons,
//     labelled live region) that always run when Chrome is available;
//  2. a full axe-core audit served same-origin, skipped when offline.
func TestBrowser_Accessibility(t *testing.T) {
	t.Parallel()

	chromePath := findChrome(t)

	axeBytes := fetchAxeCore(t)

	const nonce = "browser-a11y-nonce"

	s := setupDashboard(t,
		dashboard.WithNonce(nonce),
		dashboard.WithCSSPath("/static/app.css"),
		dashboard.WithDatastarSrc("/static/datastar.js"),
	)
	defer s.cleanup()

	s.mux.HandleFunc("/static/app.css", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_, _ = w.Write([]byte("body { margin: 0; }"))
	})
	s.mux.HandleFunc("/static/datastar.js", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/javascript")
		_, _ = w.Write(dstarstatic.Bytes())
	})
	s.mux.HandleFunc("/static/axe.js", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/javascript")
		_, _ = w.Write(axeBytes)
	})

	server := httptest.NewServer(s.mux)
	defer server.Close()

	wsURL, stopChrome := startHeadlessChrome(t, chromePath)
	defer stopChrome()

	runCtx, runCancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer runCancel()

	allocCtx, allocCancel := chromedp.NewRemoteAllocator(runCtx, wsURL)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	errLog := watchBrowserErrors(ctx)

	if err := chromedp.Run(ctx, chromedp.Navigate(server.URL+"/health")); err != nil {
		t.Fatalf("browser navigate: %v", err)
	}

	waitForSubscriber(t, s.dash)

	var lang, missingNames, missingRegions string

	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`document.documentElement.lang || ""`, &lang),
		chromedp.Evaluate(
			`[...document.querySelectorAll("button,a")].filter(e => !e.textContent.trim() && !e.getAttribute("aria-label") && !e.getAttribute("title")).length + ""`,
			&missingNames,
		),
		chromedp.Evaluate(
			`document.querySelector("main,[role=main]") ? "main" : (document.querySelector("[aria-label],[role=region]") ? "region" : "none")`,
			&missingRegions,
		),
	); err != nil {
		t.Fatalf("browser evaluate: %v", err)
	}

	if lang == "" {
		t.Error("html element has no lang attribute")
	}

	if missingNames != "0" {
		t.Errorf("%s button(s)/link(s) have no accessible name", missingNames)
	}

	if missingRegions == "none" {
		t.Error("page has no main landmark or labelled region")
	}

	if axeBytes == nil {
		return
	}

	var audit string

	inject := `(function () {
		if (window.axe) { return; }
		var s = document.createElement("script");
		s.src = "/static/axe.js";
		document.head.appendChild(s);
	})()`

	if err := chromedp.Run(ctx, chromedp.Evaluate(inject, nil)); err != nil {
		t.Fatalf("axe inject: %v", err)
	}

	waitForJS(
		t, ctx,
		`window.axe !== undefined`,
		`typeof window.axe`,
		nil,
	)

	// The skip link is sr-only until keyboard focus and this harness serves
	// no real Tailwind stylesheet, so axe cannot compute meaningful contrast
	// for it; production colors (blue-600 on white) pass WCAG AA.
	// definition-list is tolerated ONLY for the StatCard figure markup
	// (upstream templ-components#6: the <dd> sits in an
	// "items-baseline" <div> — the only items-baseline user on this page);
	// any other definition-list violation still fails.
	start := `axe.run(
		{ include: [document], exclude: [["a[href='#main-content']"]] },
		{ resultTypes: ["violations"] }
	).then(function (r) {
		window.__axeViolations = JSON.stringify(r.violations.filter(function (v) {
			if (v.impact !== "serious" && v.impact !== "critical") { return false; }
			if (v.id === "definition-list") {
				return !(v.nodes.length > 0 && v.nodes.every(function (n) {
					return n.html.indexOf("items-baseline") !== -1 && n.html.indexOf("<dd") !== -1;
				}));
			}
			return true;
		}).map(function (v) { return v.id + ":" + v.impact + ":" + v.nodes.length + ":" + v.nodes.map(function (n) { return n.html; }).join(" | ").slice(0, 200); }));
	}).catch(function (e) {
		window.__axeViolations = "AXE_ERROR: " + e;
	})`

	if err := chromedp.Run(ctx, chromedp.Evaluate(start, nil)); err != nil {
		t.Fatalf("axe start: %v", err)
	}

	waitForJS(
		t, ctx,
		`window.__axeViolations !== undefined`,
		`window.__axeViolations`,
		&audit,
	)

	if audit != "[]" {
		t.Errorf("axe-core found serious/critical violations: %s", audit)
	}

	assertNoBrowserErrors(t, errLog)
}

// waitForJS polls a JavaScript predicate until it is truthy (bounded by a
// 30s deadline) and then unmarshals the value expression into res when res
// is non-nil. This avoids chromedp.Poll's predicate-value unmarshalling
// semantics, which complicate boolean-gated string results.
func waitForJS(t *testing.T, ctx context.Context, predicate, valueExpr string, res any) {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)

	for {
		var truthy bool

		if err := chromedp.Run(ctx, chromedp.Evaluate("!!("+predicate+")", &truthy)); err != nil {
			t.Fatalf("browser evaluate %q: %v", predicate, err)
		}

		if truthy {
			if res == nil {
				return
			}

			if err := chromedp.Run(ctx, chromedp.Evaluate(valueExpr, res)); err != nil {
				t.Fatalf("browser fetch %q: %v", valueExpr, err)
			}

			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("condition never became true within 30s: %s", predicate)
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// browserStaticHandlers wires the minimal static assets the strict-CSP
// harness pages reference (compiled CSS stand-in + embedded SDK).
func browserStaticHandlers(t *testing.T, s *probeSetup) {
	t.Helper()

	s.mux.HandleFunc("/static/app.css", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_, _ = w.Write([]byte("body { margin: 0; }"))
	})
	s.mux.HandleFunc("/static/datastar.js", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/javascript")
		_, _ = w.Write(dstarstatic.Bytes())
	})
}

// TestBrowser_KeyboardNavigation walks the page with real Tab keystrokes
// and asserts the focus ring stays on visible, meaningful targets in a
// classifyFocusStop interprets a focus descriptor: the first result reports
// whether the stop is an interactive element (anything but <body>), the
// second whether it was visible with a non-none focus outline.
func classifyFocusStop(desc string) (bool, bool) {
	if desc == "body" {
		return false, false
	}

	parts := strings.SplitN(desc, "|", 4)
	if len(parts) != 4 {
		return true, false
	}

	w, h := 0, 0
	if _, err := fmt.Sscanf(parts[2], "%dx%d", &w, &h); err != nil || w <= 0 || h <= 0 {
		return true, false
	}

	return true, !strings.HasPrefix(parts[3], "none/")
}

// sane order: it must reach at least two distinct interactive elements,
// never die on <body>, and every stop must be visible with a non-none
// focus outline (Chrome's default ring — the harness loads no Tailwind).
func TestBrowser_KeyboardNavigation(t *testing.T) {
	t.Parallel()

	chromePath := findChrome(t)

	const nonce = "browser-kbd-nonce"

	s := setupDashboard(t,
		dashboard.WithNonce(nonce),
		dashboard.WithCSSPath("/static/app.css"),
		dashboard.WithDatastarSrc("/static/datastar.js"),
	)
	defer s.cleanup()

	browserStaticHandlers(t, s)

	server := httptest.NewServer(s.mux)
	defer server.Close()

	wsURL, stopChrome := startHeadlessChrome(t, chromePath)
	defer stopChrome()

	runCtx, runCancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer runCancel()

	allocCtx, allocCancel := chromedp.NewRemoteAllocator(runCtx, wsURL)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	errLog := watchBrowserErrors(ctx)

	if err := chromedp.Run(ctx, chromedp.Navigate(server.URL+"/health")); err != nil {
		t.Fatalf("browser navigate: %v", err)
	}

	waitForSubscriber(t, s.dash)

	const describeFocus = `(() => {
		const a = document.activeElement;
		if (!a || a === document.body) { return "body"; }
		const r = a.getBoundingClientRect();
		const cs = getComputedStyle(a);
		return [a.tagName, a.id || a.getAttribute("aria-label") || a.textContent.trim().slice(0, 30),
			Math.round(r.width) + "x" + Math.round(r.height),
			cs.outlineStyle + "/" + cs.outlineWidth].join("|");
	})()`

	var desc string
	stops := make([]string, 0, 15)
	visibleWithOutline := 0

	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`document.activeElement && document.activeElement.blur(); "ok"`, &desc),
	); err != nil {
		t.Fatalf("blur initial focus: %v", err)
	}

	for i := range 15 {
		if err := chromedp.Run(ctx, chromedp.KeyEvent("\t")); err != nil {
			t.Fatalf("tab keypress %d: %v", i, err)
		}
		if err := chromedp.Run(ctx, chromedp.Evaluate(describeFocus, &desc)); err != nil {
			t.Fatalf("describe focus after tab %d: %v", i, err)
		}
		stops = append(stops, desc)
		if _, visible := classifyFocusStop(desc); visible {
			visibleWithOutline++
		}
	}

	distinct := map[string]bool{}
	bodyDeadEnds := 0
	for _, stop := range stops {
		if stop == "body" {
			bodyDeadEnds++

			continue
		}
		distinct[stop] = true
	}

	if len(distinct) < 2 {
		t.Errorf(
			"keyboard walk reached %d distinct interactive targets (%d body dead ends), want >= 2; stops: %v",
			len(distinct),
			bodyDeadEnds,
			stops,
		)
	}

	if visibleWithOutline < 2 {
		t.Errorf("only %d focused targets were visible with a focus outline, want >= 2; stops: %v",
			visibleWithOutline, stops)
	}

	assertNoBrowserErrors(t, errLog)
}

// TestBrowser_MetricsUnderStrictCSP fetches /health/metrics from inside
// the strict-CSP dashboard page: same-origin fetch must be allowed by
// connect-src 'self', the scrape must parse as Prometheus exposition, and
// the CSP sandbox must stay silent (no console errors).
func TestBrowser_MetricsUnderStrictCSP(t *testing.T) {
	t.Parallel()

	chromePath := findChrome(t)

	const nonce = "browser-metrics-nonce"

	s := setupDashboard(t,
		dashboard.WithNonce(nonce),
		dashboard.WithCSSPath("/static/app.css"),
		dashboard.WithDatastarSrc("/static/datastar.js"),
		dashboard.WithMetrics(true),
	)
	defer s.cleanup()

	browserStaticHandlers(t, s)

	server := httptest.NewServer(s.mux)
	defer server.Close()

	wsURL, stopChrome := startHeadlessChrome(t, chromePath)
	defer stopChrome()

	runCtx, runCancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer runCancel()

	allocCtx, allocCancel := chromedp.NewRemoteAllocator(runCtx, wsURL)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	errLog := watchBrowserErrors(ctx)

	if err := chromedp.Run(ctx, chromedp.Navigate(server.URL+"/health")); err != nil {
		t.Fatalf("browser navigate: %v", err)
	}

	waitForSubscriber(t, s.dash)

	fetch := `(() => {
		fetch("/health/metrics")
			.then(function (r) { return r.text(); })
			.then(function (text) { window.__scrape = text; })
			.catch(function (e) { window.__scrape = "FETCH_ERROR:" + e; });
		return "started";
	})()`

	var status string
	if err := chromedp.Run(ctx, chromedp.Evaluate(fetch, &status)); err != nil {
		t.Fatalf("metrics fetch evaluate: %v", err)
	}

	var scrape string
	waitForJS(
		t,
		ctx,
		`window.__scrape !== undefined`,
		`window.__scrape === undefined ? "pending" : (String(window.__scrape).indexOf("FETCH_ERROR") === 0 ? window.__scrape : "loaded")`,
		&scrape,
	)

	if scrape == "pending" {
		t.Fatal("in-page metrics fetch never completed")
	}

	if scrape != "loaded" {
		t.Fatalf("in-page metrics fetch failed under strict CSP: %s", scrape)
	}

	var body string
	if err := chromedp.Run(ctx, chromedp.Evaluate(`window.__scrape`, &body)); err != nil {
		t.Fatalf("read scrape text: %v", err)
	}

	if !strings.Contains(body, "dashboard_health") {
		t.Errorf(
			"scrape does not look like dashboard exposition (no dashboard_health series): %.200s",
			body,
		)
	}

	assertNoBrowserErrors(t, errLog)
}
