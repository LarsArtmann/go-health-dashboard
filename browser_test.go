package dashboard_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
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
		if _, err := os.Stat(p); err == nil {
			return p
		}

		t.Fatalf("GO_HEALTH_DASHBOARD_CHROME is set but the file does not exist: %s", p)
	}

	for _, name := range []string{"google-chrome-stable", "google-chrome", "chromium", "chromium-browser"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}

	t.Skip("no Chrome/Chromium binary found; set GO_HEALTH_DASHBOARD_CHROME to enable browser tests")

	return ""
}

// strictCSPMiddleware serves a locked-down CSP: no unsafe-inline for scripts
// or styles, everything self-hosted, only the given nonce may inline scripts.
func strictCSPMiddleware(nonce string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'nonce-"+nonce+"'; "+
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

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.ExecPath(chromePath),
			chromedp.Flag("no-sandbox", true),
		)...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	if err := chromedp.Run(ctx, chromedp.Navigate(server.URL+"/health")); err != nil {
		t.Fatalf("browser navigate: %v", err)
	}

	deadline := time.Now().Add(15 * time.Second)
	for s.dash.SubscriberCount() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("Datastar SDK never connected via SSE; CSP blocked the inline script or the connection")
		}

		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(250 * time.Millisecond) // allow at least one SSE patch to apply

	var styleAttrs, styleTags int64

	var bodyText string

	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[style]').length`, &styleAttrs),
		chromedp.Evaluate(`document.querySelectorAll('style').length`, &styleTags),
		chromedp.Evaluate(`document.body.innerText`, &bodyText),
	); err != nil {
		t.Fatalf("browser evaluate: %v", err)
	}

	if styleAttrs != 0 {
		t.Errorf("runtime DOM contains %d elements with inline style attributes; want 0", styleAttrs)
	}

	if styleTags != 0 {
		t.Errorf("runtime DOM contains %d <style> elements; want 0", styleTags)
	}

	if !strings.Contains(bodyText, "All Systems Operational") {
		t.Errorf("health content missing from live DOM; got: %.200s", bodyText)
	}
}
