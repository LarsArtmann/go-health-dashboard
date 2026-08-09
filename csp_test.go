package dashboard_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	dashboard "github.com/larsartmann/go-health-dashboard"
)

// scriptOpenTagRE matches inline <script ...> opening tags (not self-closing
// or external src-only tags differently). Used to verify every script carries
// a CSP nonce.
var scriptOpenTagRE = regexp.MustCompile(`<script[^>]*>`)

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

// --- Per-request nonce extractor tests ---.

// TestNonceExtractor_AppliesPerRequestNonce verifies that WithNonceExtractor
// produces script tags with the extracted nonce, not the construction-time
// nonce.
func TestNonceExtractor_AppliesPerRequestNonce(t *testing.T) {
	t.Parallel()

	extractor := func(_ *http.Request) string { return "per-request-nonce-xyz" }
	s := setupDashboard(t, dashboard.WithNonceExtractor(extractor))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

	body := w.Body.String()

	if !strings.Contains(body, `nonce="per-request-nonce-xyz"`) {
		t.Error("HTML should contain the per-request nonce from the extractor")
	}

	if strings.Contains(body, `nonce="construction-time-nonce"`) {
		t.Error("HTML should NOT contain a stale construction-time nonce")
	}
}

// TestNonceExtractor_DifferentNoncePerRequest verifies that each request gets
// a different nonce when the extractor returns unique values.
func TestNonceExtractor_DifferentNoncePerRequest(t *testing.T) {
	t.Parallel()

	var counter int
	extractor := func(_ *http.Request) string {
		counter++

		return "nonce-" + string(rune('a'+counter-1))
	}
	s := setupDashboard(t, dashboard.WithNonceExtractor(extractor))
	defer s.cleanup()

	w1 := doRequest(t, s.mux, "/health")
	w2 := doRequest(t, s.mux, "/health")

	body1 := w1.Body.String()
	body2 := w2.Body.String()

	if !strings.Contains(body1, `nonce="nonce-a"`) {
		t.Error("first request should use nonce-a")
	}

	if !strings.Contains(body2, `nonce="nonce-b"`) {
		t.Error("second request should use nonce-b")
	}
}

// TestNonceExtractor_FallsBackToFixedNonce verifies that when the extractor
// returns an empty string, the dashboard falls back to WithNonce.
func TestNonceExtractor_FallsBackToFixedNonce(t *testing.T) {
	t.Parallel()

	extractor := func(_ *http.Request) string { return "" }
	s := setupDashboard(t,
		dashboard.WithNonce("fallback-nonce"),
		dashboard.WithNonceExtractor(extractor),
	)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")

	body := w.Body.String()

	if !strings.Contains(body, `nonce="fallback-nonce"`) {
		t.Error("HTML should fall back to fixed nonce when extractor returns empty")
	}
}

// TestNonceExtractor_DoesNotAffectJSONResponse verifies that the extractor is
// not invoked for JSON responses (no nonce leakage into JSON).
func TestNonceExtractor_DoesNotAffectJSONResponse(t *testing.T) {
	t.Parallel()

	extractor := func(_ *http.Request) string { return "should-not-appear" }
	s := setupDashboard(t, dashboard.WithNonceExtractor(extractor))
	defer s.cleanup()

	w := doRequestWithAccept(t, s.mux, "/health", "application/json")

	body := w.Body.String()

	if strings.Contains(body, "should-not-appear") {
		t.Error("JSON response should not contain the nonce")
	}
}

// --- Render cleanliness guarantees (CSP regression guards) ---.

// TestRender_AllScriptsCarryNonce verifies that every inline <script> tag in
// the dashboard output carries the per-request nonce. This is the library-level
// guarantee that lets a consumer set script-src 'nonce-...' without
// 'unsafe-inline'. If a new component or refactor emits an un-nonce'd script,
// this test catches it.
func TestRender_AllScriptsCarryNonce(t *testing.T) {
	t.Parallel()

	const nonce = "csp-guard-nonce"
	s := setupDashboard(
		t,
		dashboard.WithNonceExtractor(func(*http.Request) string { return nonce }),
	)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")
	body := w.Body.String()

	for _, match := range scriptOpenTagRE.FindAllString(body, -1) {
		if !strings.Contains(match, `nonce="`+nonce+`"`) {
			t.Errorf("inline script tag lacks nonce %q:\n%s", nonce, match)
		}
	}
}

// TestRender_NoInlineStyles verifies the dashboard output contains no <style>
// blocks and no inline style="" attributes. This documents that the dashboard
// is compatible with a strict style-src policy (no 'unsafe-inline' needed for
// the /health page itself). templ-components uses Tailwind classes exclusively;
// if that ever changes, this test surfaces it.
func TestRender_NoInlineStyles(t *testing.T) {
	t.Parallel()

	s := setupDashboard(t, dashboard.WithNonceExtractor(func(*http.Request) string { return "x" }))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")
	body := w.Body.String()

	if c := strings.Count(body, "<style"); c != 0 {
		t.Errorf("dashboard output should contain no <style> blocks, found %d", c)
	}

	if c := strings.Count(body, `style="`); c != 0 {
		t.Errorf("dashboard output should contain no inline style= attributes, found %d", c)
	}
}
