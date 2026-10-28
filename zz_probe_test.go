package dashboard_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/remote"
	dashboard "github.com/larsartmann/go-health-dashboard"
)

func scratchLaunch(t *testing.T) (string, func()) {
	t.Helper()

	chrome := os.Getenv("GO_HEALTH_DASHBOARD_CHROME")
	if chrome == "" {
		t.Skip("no chrome")
	}

	profileDir := t.TempDir()

	cmd := exec.Command(chrome,
		"--headless",
		"--no-sandbox",
		"--disable-gpu",
		"--remote-debugging-port=0",
		"--user-data-dir="+profileDir,
		"about:blank",
	)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	stop := func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}

	lines := make(chan string, 64)

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()

	deadline := time.After(30 * time.Second)

	for {
		select {
		case line := <-lines:
			if i := strings.Index(line, "DevTools listening on "); i >= 0 {
				return strings.TrimSpace(line[i+len("DevTools listening on "):]), stop
			}
		case <-deadline:
			stop()
			t.Fatal("no devtools url")
		}
	}
}

func scratchSession(t *testing.T, wsURL string) context.Context {
	t.Helper()

	runCtx, runCancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(runCancel)

	allocCtx, allocCancel := remote.NewAllocator(runCtx, wsURL)
	t.Cleanup(allocCancel)

	ctx, cancel := chromedp.NewContext(allocCtx)
	t.Cleanup(cancel)

	return ctx
}

func TestScratchPlainPage(t *testing.T) {
	wsURL, stop := scratchLaunch(t)
	defer stop()

	ctx := scratchSession(t, wsURL)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<h1>ok</h1>"))
	}))
	defer server.Close()

	start := time.Now()

	if err := chromedp.Do(ctx, chromedp.Navigate(server.URL)); err != nil {
		t.Fatalf("plain navigate after %s: %v", time.Since(start), err)
	}

	t.Logf("plain navigate in %s", time.Since(start))
}

func scratchHealthServer(t *testing.T) (*httptest.Server, func()) {
	t.Helper()

	s := setupDashboard(t,
		dashboard.WithNonce("scratch-nonce"),
		dashboard.WithCSSPath("/static/app.css"),
		dashboard.WithDatastarSrc("/static/datastar.js"),
	)

	browserStaticHandlers(t, s)

	server := httptest.NewServer(s.mux)

	return server, func() {
		server.Close()
		s.cleanup()
	}
}

func TestScratchHealthPage(t *testing.T) {
	wsURL, stop := scratchLaunch(t)
	defer stop()

	ctx := scratchSession(t, wsURL)

	server, cleanup := scratchHealthServer(t)
	defer cleanup()

	start := time.Now()

	if err := chromedp.Do(ctx, chromedp.Navigate(server.URL+"/health")); err != nil {
		t.Fatalf("health navigate after %s: %v", time.Since(start), err)
	}

	t.Logf("health navigate in %s", time.Since(start))
}

func TestScratchHealthPageWithErrLog(t *testing.T) {
	wsURL, stop := scratchLaunch(t)
	defer stop()

	ctx := scratchSession(t, wsURL)

	server, cleanup := scratchHealthServer(t)
	defer cleanup()

	errLog := watchBrowserErrors(ctx)

	start := time.Now()

	if err := chromedp.Do(ctx, chromedp.Navigate(server.URL+"/health")); err != nil {
		t.Fatalf("health navigate after %s: %v (console: %v)", time.Since(start), err, errLog.all())
	}

	t.Logf("health navigate in %s", time.Since(start))
}
