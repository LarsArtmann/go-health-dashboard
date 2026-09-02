package dashboard_test

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	dstarstatic "github.com/larsartmann/go-datastar/static"
	dashboard "github.com/larsartmann/go-health-dashboard"
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

// startHeadlessChrome launches Chrome manually with a concrete DevTools port
// and returns the websocket debugger URL parsed from stderr. chromedp's own
// launcher queries the debugger over 127.0.0.1, which hangs when Chrome
// binds the DevTools listener to IPv6 ::1 only — parsing the announced URL
// avoids that failure mode entirely.
func startHeadlessChrome(t *testing.T, chromePath string) (string, func()) {
	t.Helper()

	// t.TempDir cleanup races Chrome's renderer children, which keep writing
	// into the profile after the browser process exits — removal is retried
	// in stopChrome instead.
	//nolint:usetesting // see above
	profileDir, err := os.MkdirTemp("", "go-health-dashboard-chrome-")
	if err != nil {
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

	timeout := time.After(20 * time.Second)

	// stopChrome terminates the browser and removes the profile. Renderer
	// child processes may outlive the browser process for a few milliseconds
	// and keep writing into the profile, so removal retries briefly before
	// giving up (the OS temp dir is the final safety net).
	stopChrome := func() {
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
			t.Fatal("chrome did not announce a DevTools websocket within 20s")
		case line, ok := <-lines:
			if !ok {
				_ = cmd.Wait()
				_ = os.RemoveAll(profileDir)
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
}
