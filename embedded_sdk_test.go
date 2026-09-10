package dashboard_test

import (
	"net/http"
	"strings"
	"testing"

	dstarstatic "github.com/larsartmann/go-datastar/static"
	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
)

// TestEmbeddedSDK_ServesPinnedBundle verifies WithEmbeddedDatastarSDK
// registers Routes.DatastarJS serving byte-identical SDK bytes and points
// the HTML script tag at it.
func TestEmbeddedSDK_ServesPinnedBundle(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithEmbeddedDatastarSDK())
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health/datastar.js")
	if w.Code != http.StatusOK {
		t.Fatalf("embedded SDK: want 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/javascript") {
		t.Errorf("Content-Type: want text/javascript, got %q", ct)
	}

	if got := w.Body.Bytes(); string(got) != string(dstarstatic.Bytes()) {
		t.Errorf("served bytes differ from the pinned go-datastar/static bundle")
	}

	html := doRequest(t, s.mux, "/health").Body.String()
	if !strings.Contains(html, `src="/health/datastar.js"`) {
		t.Errorf("HTML script tag does not reference the embedded SDK path:\n%.400s", html)
	}
}

// TestEmbeddedSDK_DisabledByDefault pins the opt-in contract.
func TestEmbeddedSDK_DisabledByDefault(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	if w := doRequest(t, s.mux, "/health/datastar.js"); w.Code != http.StatusNotFound {
		t.Errorf("embedded SDK without the option: want 404, got %d", w.Code)
	}
}

// TestEmbeddedSDK_RespectsBasePath verifies the SDK route participates in
// WithBasePath resolution (the order-independent kind).
func TestEmbeddedSDK_RespectsBasePath(t *testing.T) {
	t.Parallel()

	d := dashboard.New(
		newStubProber(health.Response{}),
		dashboard.WithEmbeddedDatastarSDK(),
		dashboard.WithBasePath("/admin"),
	)

	if got := d.Routes().DatastarJS; got != "/admin/health/datastar.js" {
		t.Fatalf("DatastarJS: want /admin/health/datastar.js, got %q", got)
	}
}
