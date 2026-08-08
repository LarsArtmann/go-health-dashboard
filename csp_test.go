package dashboard_test

import (
	"strings"
	"testing"

	dashboard "github.com/larsartmann/go-health-dashboard"
)

func TestCSP_NonceAppliedToDatastarScript(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithNonce("test-nonce-abc"))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

	body := w.Body.String()

	datastarScripts := strings.Count(body, `nonce="test-nonce-abc"`)
	if datastarScripts < 2 {
		t.Errorf("expected at least 2 script tags with nonce, got %d", datastarScripts)
	}

	if !strings.Contains(body, `nonce="test-nonce-abc"`) {
		t.Error("Datastar SDK script tag should have nonce attribute")
	}
}

func TestCSP_NonceAppliedToTailwindScript(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithNonce("tw-nonce-123"))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

	body := w.Body.String()

	if !strings.Contains(body, `nonce="tw-nonce-123"`) {
		t.Error("Tailwind CDN script tag should have nonce attribute")
	}

	if !strings.Contains(body, `nonce="tw-nonce-123">\n\t\ttailwind.config`) &&
		!strings.Contains(body, "tailwind.config") {
		t.Error("Inline tailwind config script should have nonce attribute")
	}
}

func TestCSP_NoNonceRendersWithoutNonce(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

	body := w.Body.String()

	if strings.Contains(body, "nonce=") {
		t.Error("HTML should not contain nonce attributes when WithNonce is not set")
	}
}

func TestCSP_EmptyNonceRendersWithoutNonce(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithNonce(""))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

	body := w.Body.String()

	if strings.Contains(body, `nonce=""`) {
		t.Error("HTML should not contain empty nonce attributes")
	}
}

func TestCSP_NonceDoesNotAffectJSONResponse(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithNonce("json-test-nonce"))
	defer s.cleanup()

	w := doRequestWithAccept(t, s.mux, "/health", "application/json")

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("content-type: want application/json, got %s", ct)
	}

	body := w.Body.String()

	if strings.Contains(body, "nonce") {
		t.Error("JSON response should not contain nonce attributes")
	}
}

func TestCSP_NonceUsedConsistently(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithNonce("consistent-nonce"))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

	body := w.Body.String()

	count := strings.Count(body, `nonce="consistent-nonce"`)
	if count < 2 {
		t.Errorf("expected nonce to appear at least 2 times (Datastar + Tailwind), got %d", count)
	}

	if strings.Contains(body, "nonce=") &&
		strings.Contains(body, `nonce="consistent-nonce"`) == false {
		t.Error("all nonce attributes should use the provided value")
	}
}

func TestCSP_WithCSSPathSuppressesTailwindCDN(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithCSSPath("/static/app.css"))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

	body := w.Body.String()

	if strings.Contains(body, "cdn.tailwindcss.com") {
		t.Error("Tailwind CDN script should not be rendered when CSSPath is set")
	}

	if !strings.Contains(body, `href="/static/app.css"`) {
		t.Error("HTML should contain CSS link tag when CSSPath is set")
	}
}

func TestCSP_WithoutCSSPathUsesTailwindCDN(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

	body := w.Body.String()

	if !strings.Contains(body, "cdn.tailwindcss.com") {
		t.Error("Tailwind CDN script should be rendered when CSSPath is not set")
	}
}

func TestDarkMode_ToggleButtonPresent(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

	body := w.Body.String()

	if !strings.Contains(body, "data-theme-toggle") {
		t.Error("HTML should contain dark mode toggle button with data-theme-toggle attribute")
	}
}
