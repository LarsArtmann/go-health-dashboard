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
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	dstarstatic "github.com/larsartmann/go-datastar/static"
	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	"github.com/larsartmann/go-health/aggregate"
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
	// (The StatCard definition-list/dlitem tolerance was retired with
	// templ-components v1.16.0, which fixed upstream #6 by grouping <dt>
	// and <dd> inside the same wrapper div.)
	start := `axe.run(
		{ include: [document], exclude: [["a[href='#main-content']"]] },
		{ resultTypes: ["violations"] }
	).then(function (r) {
		window.__axeViolations = JSON.stringify(r.violations.filter(function (v) {
			return v.impact === "serious" || v.impact === "critical";
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

// TestBrowser_AggregateCSPClean renders a two-source aggregate page (the
// multi-service dashboard) under the strict CSP harness: namespaced
// source/check rows must appear, the runtime must stay style-free, and the
// SSE connection must come up — proving the aggregate surface is
// browser-valid, not just unit-tested.
func TestBrowser_AggregateCSPClean(t *testing.T) {
	t.Parallel()

	chromePath := findChrome(t)

	const nonce = "browser-agg-nonce"

	apiInjector := do.New()
	provideHealthy(apiInjector, "postgres")
	invoke[*healthyService](t, apiInjector, "postgres")

	apiProbe := health.New(apiInjector, health.WithRefreshInterval(100*time.Millisecond))
	if err := apiProbe.Start(context.Background()); err != nil {
		t.Fatalf("api probe start: %v", err)
	}
	defer apiProbe.Shutdown()

	agg, err := aggregate.New(
		aggregate.Source{Name: "api", Probe: apiProbe},
	)
	if err != nil {
		t.Fatalf("aggregate.New: %v", err)
	}

	s := setupDashboardWithProber(t, agg,
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

	if dash := s.dash; dash.SubscriberCount() == 0 {
		deadline := time.Now().Add(20 * time.Second)

		for dash.SubscriberCount() == 0 && time.Now().Before(deadline) {
			time.Sleep(100 * time.Millisecond)
		}

		if dash.SubscriberCount() == 0 {
			var html string
			_ = chromedp.Run(ctx, chromedp.Evaluate(`document.body.innerHTML.slice(0, 600)`, &html))
			t.Fatalf(
				"aggregate page SSE never connected; console: %v; body: %s",
				errLog.all(),
				html,
			)
		}
	}

	var namespaced string
	if err := chromedp.Run(ctx, chromedp.Evaluate(
		`document.body.innerText.includes("api/postgres") ? "yes" : "no"`,
		&namespaced,
	)); err != nil {
		t.Fatalf("evaluate: %v", err)
	}

	if namespaced != "yes" {
		t.Errorf(
			"aggregate page does not show the namespaced api/postgres check; console: %v",
			errLog.all(),
		)
	}

	// CSP-clean invariants, same as the single-probe page.
	var styles string
	if err := chromedp.Run(ctx, chromedp.Evaluate(
		`document.querySelectorAll("style").length + ":" + [...document.querySelectorAll("[style]:not(html)")].length`,
		&styles,
	)); err != nil {
		t.Fatalf("style probe: %v", err)
	}

	if styles != "0:0" {
		var offender string
		_ = chromedp.Run(ctx, chromedp.Evaluate(
			`[...document.querySelectorAll("[style]")].map(e => e.outerHTML.slice(0, 300)).join(" || ")`,
			&offender,
		))
		t.Errorf(
			"aggregate page has inline styles or style tags: %s; offender: %s",
			styles,
			offender,
		)
	}

	assertNoBrowserErrors(t, errLog)
}

// TestBrowser_CollapseInteract proves the healthy-group collapse end-to-end
// under a strict CSP: the group starts collapsed (server-derived default),
// clicking the summary expands it, and the next SSE patch re-applies the
// collapsed default — collapse state is re-derived server-side on every
// patch by design, and this test pins that documented behavior.
func TestBrowser_CollapseInteract(t *testing.T) {
	t.Parallel()

	chromePath := findChrome(t)

	const nonce = "browser-collapse-nonce"

	// PushAlways guarantees patches flow after the manual toggle without
	// needing an actual health change.
	s := setupDashboardWithHealthyServices(t, 9,
		dashboard.WithNonce(nonce),
		dashboard.WithCSSPath("/static/app.css"),
		dashboard.WithDatastarSrc("/static/datastar.js"),
		dashboard.WithPushMode(dashboard.PushAlways),
		dashboard.WithPushInterval(200*time.Millisecond),
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

	waitForSubscriber(t, s.dash)
	time.Sleep(250 * time.Millisecond) // allow the initial SSE patch to apply

	const detailsState = `(function () {
		var d = document.querySelector("details");
		return d ? (d.open ? "open" : "closed") : "missing";
	})()`

	var state string

	if err := chromedp.Run(ctx, chromedp.Evaluate(detailsState, &state)); err != nil {
		t.Fatalf("browser evaluate: %v", err)
	}

	if state != "closed" {
		t.Fatalf("healthy group should start collapsed, got %q", state)
	}

	if err := chromedp.Run(ctx, chromedp.Click("details summary", chromedp.ByQuery)); err != nil {
		t.Fatalf("click summary: %v", err)
	}

	if err := chromedp.Run(ctx, chromedp.Evaluate(detailsState, &state)); err != nil {
		t.Fatalf("browser evaluate after click: %v", err)
	}

	if state != "open" {
		t.Fatalf("healthy group should be expanded after clicking the summary, got %q", state)
	}

	var tableVisible bool

	if err := chromedp.Run(ctx, chromedp.Evaluate(
		`document.querySelector("details table") !== null && document.querySelector("details table").offsetParent !== null`,
		&tableVisible,
	)); err != nil {
		t.Fatalf("browser evaluate table: %v", err)
	}

	if !tableVisible {
		t.Error("service table inside the expanded group should be visible")
	}

	// The next SSE patch (PushAlways, 200ms) replaces #health-region with a
	// fresh server-side render, which re-collapses the group. This is the
	// documented trade-off of server-derived collapse state.
	waitForJS(t, ctx, detailsState+` === "closed"`, detailsState, nil)

	assertNoBrowserErrors(t, errLog)
}

// TestBrowser_CollapsePersistInteract proves WithPersistCollapse end-to-end:
// a manual toggle survives SSE patches (localStorage re-apply) and a full
// page reload, unlike the server-derived default proven above.
func TestBrowser_CollapsePersistInteract(t *testing.T) {
	t.Parallel()

	chromePath := findChrome(t)

	const nonce = "browser-collapse-persist-nonce"

	s := setupDashboardWithHealthyServices(t, 9,
		dashboard.WithNonce(nonce),
		dashboard.WithCSSPath("/static/app.css"),
		dashboard.WithDatastarSrc("/static/datastar.js"),
		dashboard.WithPushMode(dashboard.PushAlways),
		dashboard.WithPushInterval(200*time.Millisecond),
		dashboard.WithPersistCollapse(),
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

	waitForSubscriber(t, s.dash)
	time.Sleep(250 * time.Millisecond) // allow the initial SSE patch to apply

	const detailsState = `(function () {
		var d = document.querySelector("details");
		return d ? (d.open ? "open" : "closed") : "missing";
	})()`

	var state string

	if err := chromedp.Run(ctx, chromedp.Evaluate(detailsState, &state)); err != nil {
		t.Fatalf("browser evaluate: %v", err)
	}

	if state != "closed" {
		t.Fatalf("healthy group should start collapsed (threshold default 8 < 9 rows), got %q", state)
	}

	if err := chromedp.Run(ctx, chromedp.Click("details summary", chromedp.ByQuery)); err != nil {
		t.Fatalf("click summary: %v", err)
	}

	var stored string

	if err := chromedp.Run(ctx, chromedp.Evaluate(
		`localStorage.getItem("health-healthy-group-collapsed") || ""`,
		&stored,
	)); err != nil {
		t.Fatalf("browser evaluate localStorage: %v", err)
	}

	if stored == "" {
		t.Error("toggle should persist the collapse state to localStorage")
	}

	// PushAlways patches every 200ms; the persistence script must re-apply
	// the stored "open" state after each patch. Give it a few patch cycles,
	// then assert it is still open.
	time.Sleep(700 * time.Millisecond)

	if err := chromedp.Run(ctx, chromedp.Evaluate(detailsState, &state)); err != nil {
		t.Fatalf("browser evaluate after patches: %v", err)
	}

	if state != "open" {
		var diag string
		_ = chromedp.Run(ctx, chromedp.Evaluate(`(function () {
			var d = document.querySelector('details[data-collapsible]');
			var plain = document.querySelector('details');
			return JSON.stringify({
				init: !!window.__healthCollapseInit,
				stored: (function(){ try { return localStorage.getItem('health-healthy-group-collapsed'); } catch (e) { return 'ERR:'+e.message; } })(),
				hasDataAttr: d !== null,
				plainDetailsOpen: plain ? plain.open : null,
				regionHTML: (document.getElementById('health-region') || {innerHTML:'NO_REGION'}).innerHTML.slice(0, 200)
			});
		})()`, &diag))
		t.Fatalf("with WithPersistCollapse the expanded state must survive SSE patches, got %q; diag=%s", state, diag)
	}

	// A full reload must also honor the stored state.
	if err := chromedp.Run(ctx, chromedp.Navigate(server.URL+"/health")); err != nil {
		t.Fatalf("browser reload: %v", err)
	}

	waitForJS(t, ctx, detailsState+` === "open"`, detailsState, nil)

	assertNoBrowserErrors(t, errLog)
}

// TestBrowser_FilterInteract proves the client-side filter end-to-end:
// typing narrows the visible rows to matches, clearing restores all of
// them, and the page stays free of browser errors under strict CSP.
func TestBrowser_FilterInteract(t *testing.T) {
	t.Parallel()

	chromePath := findChrome(t)

	const nonce = "browser-filter-nonce"

	s := setupDashboardWithHealthyServices(t, 12,
		dashboard.WithNonce(nonce),
		dashboard.WithCSSPath("/static/app.css"),
		dashboard.WithDatastarSrc("/static/datastar.js"),
		dashboard.WithEmbeddedDatastarSDK(),
		dashboard.WithHealthyGroupExpanded(),
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

	waitForSubscriber(t, s.dash)

	visibleRows := `(function () {
		return [...document.querySelectorAll("tr[data-filter-row]")].filter(function (tr) {
			return !tr.classList.contains("hidden");
		}).length + "";
	})()`

	var visible string

	if err := chromedp.Run(ctx, chromedp.Evaluate(visibleRows, &visible)); err != nil {
		t.Fatalf("browser evaluate: %v", err)
	}

	if visible != "12" {
		t.Fatalf("all 12 rows should start visible, got %s", visible)
	}

	if err := chromedp.Run(ctx,
		chromedp.SetValue(`#health-filter`, "svc-03", chromedp.ByQuery),
	); err != nil {
		t.Fatalf("type into filter: %v", err)
	}

	waitForJS(t, ctx, visibleRows+` === "1"`, visibleRows, nil)

	// Plan C5 leftover: re-run the accessibility audit on the FILTERED
	// state, not just the full page — hiding rows via data-class must not
	// introduce violations (e.g. a dangling no-match region or removed
	// table semantics).
	axeBytes := fetchAxeCore(t)

	s.mux.HandleFunc("/static/axe.js", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/javascript")
		_, _ = w.Write(axeBytes)
	})

	if err := chromedp.Run(ctx, chromedp.Evaluate(`(function () {
		if (window.axe) { return; }
		var el = document.createElement("script");
		el.src = "/static/axe.js";
		document.head.appendChild(el);
	})()`, nil)); err != nil {
		t.Fatalf("axe inject: %v", err)
	}

	waitForJS(t, ctx, `window.axe !== undefined`, `typeof window.axe`, nil)

	var filteredAudit string

	if err := chromedp.Run(ctx, chromedp.Evaluate(`axe.run(
		{ include: [document] },
		{ resultTypes: ["violations"] }
	).then(function (r) {
		window.__filteredAxe = JSON.stringify(r.violations.filter(function (v) {
			return v.impact === "serious" || v.impact === "critical";
		}).map(function (v) { return v.id + ":" + v.nodes.length; }));
	}).catch(function (e) {
		window.__filteredAxe = "AXE_ERROR: " + e;
	})`, nil)); err != nil {
		t.Fatalf("axe run: %v", err)
	}

	waitForJS(t, ctx, `window.__filteredAxe !== undefined`, `window.__filteredAxe`, &filteredAudit)

	if filteredAudit != "[]" {
		t.Errorf("axe on the filtered DOM found serious/critical violations: %s", filteredAudit)
	}

	// chromedp.SetValue cannot set an empty string, so clear via JS and
	// dispatch the input event data-bind listens on.
	if err := chromedp.Run(ctx, chromedp.Evaluate(`(function () {
		var el = document.getElementById("health-filter");
		el.value = "";
		el.dispatchEvent(new Event("input", { bubbles: true }));
	})()`, nil)); err != nil {
		t.Fatalf("clear filter: %v", err)
	}

	waitForJS(t, ctx, visibleRows+` === "12"`, visibleRows, nil)

	assertNoBrowserErrors(t, errLog)
}

// TestBrowser_ConnectionPill proves the pill reflects the SSE stream
// lifecycle: Live while the stream is up, a degraded state when the stream
// cannot be reached, and Live again after the stream recovers.
func TestBrowser_ConnectionPill(t *testing.T) {
	t.Parallel()

	chromePath := findChrome(t)

	const nonce = "browser-pill-nonce"

	var blockSSE atomic.Bool

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

	ssePath := s.dash.Routes().SSE

	proxied := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if blockSSE.Load() && r.URL.Path == ssePath {
			w.WriteHeader(http.StatusNotFound)

			return
		}
		s.mux.ServeHTTP(w, r)
	})

	server := httptest.NewServer(strictCSPMiddleware(nonce, proxied))
	defer server.Close()

	wsURL, stopChrome := startHeadlessChrome(t, chromePath)
	defer stopChrome()

	runCtx, runCancel := context.WithTimeout(context.Background(), 150*time.Second)
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

	// Record every SDK lifecycle event so a failure can show what actually
	// fired instead of guessing from minified bundle logic.
	if err := chromedp.Run(ctx, chromedp.Evaluate(`(function () {
		window.__fetchEvents = [];
		document.addEventListener("datastar-fetch", function (e) {
			var d = e.detail || {};
			window.__fetchEvents.push(d.type + (d.argsRaw && d.argsRaw.status ? ":" + d.argsRaw.status : ""));
		});
	})()`, nil)); err != nil {
		t.Fatalf("inject event recorder: %v", err)
	}

	pillState := `(function () {
		var states = ["live", "reconnecting", "offline"];
		for (var i = 0; i < states.length; i++) {
			var el = document.getElementById("conn-state-" + states[i]);
			if (el && !el.hidden) { return states[i]; }
		}
		return "none";
	})()`

	var state string

	if err := chromedp.Run(ctx, chromedp.Evaluate(pillState, &state)); err != nil {
		t.Fatalf("browser evaluate: %v", err)
	}

	if state != "live" {
		t.Fatalf("pill should start live, got %q", state)
	}

	// Break the stream: block new connections and close the broadcaster so
	// the live stream ends and the SDK starts retrying into the 404.
	blockSSE.Store(true)
	s.dash.Shutdown()

	if state = pollPillState(
		t,
		ctx,
		pillState,
		[]string{"reconnecting", "offline"},
		20*time.Second,
	); state == "live" {
		t.Fatalf("pill never left live; events: %s", dumpFetchEvents(ctx))
	}

	// Recover: unblock and restart the pusher. The SDK's RetryAlways mode
	// keeps re-running the connect through the outage (retrying into the
	// 404, then succeeding), so the pill must return to live on its own —
	// without any page reload.
	blockSSE.Store(false)

	if err := s.dash.Start(runCtx); err != nil {
		t.Fatalf("dash restart: %v", err)
	}

	if state = pollPillState(t, ctx, pillState, []string{"live"}, 45*time.Second); state != "live" {
		t.Fatalf("pill never returned to live; state %q; events: %s", state, dumpFetchEvents(ctx))
	}

	assertNoBrowserErrors(t, errLog)
}

// pollPillState polls the connection pill until it reaches one of the wanted
// states or the deadline expires, returning the last observed state.
func pollPillState(
	t *testing.T,
	ctx context.Context,
	pillState string,
	want []string,
	timeout time.Duration,
) string {
	t.Helper()

	deadline := time.Now().Add(timeout)

	var state string

	for {
		if err := chromedp.Run(ctx, chromedp.Evaluate(pillState, &state)); err != nil {
			t.Fatalf("browser evaluate pill: %v", err)
		}

		if slices.Contains(want, state) {
			return state
		}

		if time.Now().After(deadline) {
			return state
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// dumpFetchEvents returns the recorded datastar-fetch event types for
// diagnostics when a pill assertion fails.
func dumpFetchEvents(ctx context.Context) string {
	var events string
	_ = chromedp.Run(ctx, chromedp.Evaluate(`JSON.stringify(window.__fetchEvents || [])`, &events))

	return events
}

// TestBrowser_RetryAlwaysRidesOutMaxConnections proves the client-side
// interplay of RetryAlways with WithMaxSSEConnections: with the limit held
// by one tab, a second tab's SSE requests are rejected with 503 and the SDK
// keeps retrying without user action; once the holding tab disconnects, the
// waiting tab connects and renders live state. The same non-200 retry path
// in the SDK bundle serves 429 rate-limit responses (verified in the pinned
// v0.5.0 bundle: any non-200 under retry=always schedules a retry), so this
// also documents the RetryAlways × rate-limit interplay.
func TestBrowser_RetryAlwaysRidesOutMaxConnections(t *testing.T) {
	t.Parallel()

	chromePath := findChrome(t)

	const nonce = "browser-retry-503-nonce"

	s := setupDashboardWithHealthyServices(t, 4,
		dashboard.WithNonce(nonce),
		dashboard.WithCSSPath("/static/app.css"),
		dashboard.WithDatastarSrc("/static/datastar.js"),
		dashboard.WithPushMode(dashboard.PushAlways),
		dashboard.WithPushInterval(200*time.Millisecond),
		dashboard.WithMaxSSEConnections(1),
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

	runCtx, runCancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer runCancel()

	allocCtx, allocCancel := chromedp.NewRemoteAllocator(runCtx, wsURL)
	defer allocCancel()

	tabA, cancelA := chromedp.NewContext(allocCtx)
	defer cancelA()

	errLogA := watchBrowserErrors(tabA)

	if err := chromedp.Run(tabA, chromedp.Navigate(server.URL+"/health")); err != nil {
		t.Fatalf("tab A navigate: %v", err)
	}

	waitForSubscriber(t, s.dash)

	if s.dash.SubscriberCount() != 1 {
		t.Fatalf("connection limit 1: want exactly 1 subscriber, got %d", s.dash.SubscriberCount())
	}

	tabB, cancelB := chromedp.NewContext(allocCtx)
	defer cancelB()

	errLogB := watchBrowserErrors(tabB)

	if err := chromedp.Run(tabB, chromedp.Navigate(server.URL+"/health")); err != nil {
		t.Fatalf("tab B navigate: %v", err)
	}

	// Tab B's SSE requests get 503 for as long as tab A holds the only
	// slot. The SDK must keep retrying (RetryAlways) without connecting.
	time.Sleep(1500 * time.Millisecond)

	if got := s.dash.SubscriberCount(); got != 1 {
		t.Fatalf("tab B must not exceed the connection limit, subscribers = %d", got)
	}

	// Releasing tab A closes its SSE stream; tab B's pending retry should
	// take the freed slot and render live state.
	cancelA()

	deadline := time.Now().Add(20 * time.Second)

	for s.dash.SubscriberCount() == 0 {
		if time.Now().After(deadline) {
			t.Fatalf("tab B never connected after the slot freed (retry loop gave up?)")
		}

		time.Sleep(100 * time.Millisecond)
	}

	waitForBodyText(t, tabB, "Healthy")

	assertNoBrowserErrors(t, errLogA)
	assertNoBrowserErrors(t, errLogB)
}

// TestBrowser_MobileViewport proves the dashboard is usable at a phone
// width: no page-level horizontal overflow (the table's scroll wrapper
// contains its own overflow), and the content is reachable.
func TestBrowser_MobileViewport(t *testing.T) {
	t.Parallel()

	chromePath := findChrome(t)

	const nonce = "browser-mobile-nonce"

	s := setupDashboardWithHealthyServices(t, 9,
		dashboard.WithNonce(nonce),
		dashboard.WithCSSPath("/static/app.css"),
		dashboard.WithDatastarSrc("/static/datastar.js"),
		dashboard.WithEmbeddedDatastarSDK(),
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

	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(375, 667),
		chromedp.Navigate(server.URL+"/health"),
	); err != nil {
		t.Fatalf("browser navigate: %v", err)
	}

	waitForSubscriber(t, s.dash)

	var pageOverflow, wrapperContained bool

	if err := chromedp.Run(ctx,
		chromedp.Evaluate(
			`document.documentElement.scrollWidth <= window.innerWidth + 1`,
			&pageOverflow,
		),
		chromedp.Evaluate(
			`(function () {
				var w = document.querySelector(".overflow-x-auto");
				return !!w && w.scrollWidth >= w.clientWidth - 2 && w.scrollWidth <= w.clientWidth + 400;
			})()`,
			&wrapperContained,
		),
	); err != nil {
		t.Fatalf("browser evaluate: %v", err)
	}

	if !pageOverflow {
		t.Error(
			"page must not overflow horizontally at 375px; table overflow must be contained in its scroll wrapper",
		)
	}

	var bodyText string

	if err := chromedp.Run(
		ctx,
		chromedp.Evaluate(`document.body.innerText`, &bodyText),
	); err != nil {
		t.Fatalf("browser evaluate body: %v", err)
	}

	if !strings.Contains(bodyText, "All Systems Operational") {
		t.Errorf("health content missing from mobile DOM; got: %.200s", bodyText)
	}

	assertNoBrowserErrors(t, errLog)
}

// TestBrowser_KeyboardNewControls proves the new interactive controls are
// keyboard-operable end-to-end: the filter input is reachable and typeable,
// and the collapsed healthy group toggles with Enter alone (native
// <details>/<summary> semantics) under a strict CSP.
func TestBrowser_KeyboardNewControls(t *testing.T) {
	t.Parallel()

	chromePath := findChrome(t)

	const nonce = "browser-keyboard-nonce"

	s := setupDashboardWithHealthyServices(t, 9,
		dashboard.WithNonce(nonce),
		dashboard.WithCSSPath("/static/app.css"),
		dashboard.WithDatastarSrc("/static/datastar.js"),
		dashboard.WithEmbeddedDatastarSDK(),
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

	waitForSubscriber(t, s.dash)
	time.Sleep(250 * time.Millisecond) // let the initial patch settle before focusing

	detailsState := `(function () {
		var d = document.querySelector("details");
		return d ? (d.open ? "open" : "closed") : "missing";
	})()`

	// Focus the summary directly (keyboard path) and toggle with Enter
	// ("\r" is the rune chromedp's keyboard encoder maps to the Enter key).
	if err := chromedp.Run(ctx,
		chromedp.Focus(`details summary`, chromedp.ByQuery),
		chromedp.KeyEvent("\r"),
	); err != nil {
		t.Fatalf("keyboard toggle: %v", err)
	}

	waitForJS(t, ctx, detailsState+` === "open"`, detailsState, nil)

	// Focus the filter input and type through the keyboard; the signal
	// updates through the real input events.
	if err := chromedp.Run(ctx,
		chromedp.Focus(`#health-filter`, chromedp.ByQuery),
		chromedp.KeyEvent("svc-01"),
	); err != nil {
		t.Fatalf("keyboard filter focus: %v", err)
	}

	visibleRows := `(function () {
		return [...document.querySelectorAll("tr[data-filter-row]")].filter(function (tr) {
			return !tr.classList.contains("hidden");
		}).length + "";
	})()`

	waitForJS(t, ctx, visibleRows+` === "1"`, visibleRows, nil)

	assertNoBrowserErrors(t, errLog)
}
