package dashboard_test

import (
	"os"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// TestCaptureREADME_ScreenshotDark captures the dashboard in dark mode for
// documentation. Env-guarded: run with
//
//	SCREENSHOT_OUTPUT_DARK=docs/screenshot-dark.png GO_HEALTH_DASHBOARD_CHROME=/path/to/chromium \
//	  go test -run TestCaptureREADME_ScreenshotDark
func TestCaptureREADME_ScreenshotDark(t *testing.T) {
	if os.Getenv("SCREENSHOT_OUTPUT_DARK") == "" {
		t.Skip("SCREENSHOT_OUTPUT_DARK not set; dark screenshot capture is opt-in")
	}

	chromePath := findChrome(t)

	s := setupDashboard(t,
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

	server := httptest.NewServer(s.mux)
	defer server.Close()

	wsURL, stopChrome := startHeadlessChrome(t, chromePath)
	defer stopChrome()

	runCtx, runCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer runCancel()

	allocCtx, allocCancel := chromedp.NewRemoteAllocator(runCtx, wsURL)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	waitForSubscriber(t, s.dash)

	// Force the dark class before the shot; the UI stores the preference in
	// localStorage and applies it on load.
	if err := chromedp.Run(ctx,
		chromedp.Navigate(server.URL+"/health"),
		chromedp.Evaluate(`document.documentElement.classList.add("dark"); localStorage.setItem("theme", "dark")`, nil),
		chromedp.Sleep(300*time.Millisecond),
		chromedp.FullScreenshot(&buf, 92),
	); err != nil {
		t.Fatalf("browser screenshot: %v", err)
	}

	out := os.Getenv("SCREENSHOT_OUTPUT_DARK")
	if err := os.WriteFile(out, buf, 0o600); err != nil {
		t.Fatalf("write screenshot: %v", err)
	}
}
