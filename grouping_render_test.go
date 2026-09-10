package dashboard_test

import (
	"net/http"
	"strings"
	"testing"

	dashboard "github.com/larsartmann/go-health-dashboard"
)

// TestWithGrouping_RendersSourceCards wires WithGrouping end-to-end: the
// plain (un-namespaced) fixture checks land in the fallback Services card
// instead of the severity cards, proving the option reaches the renderer.
func TestWithGrouping_RendersSourceCards(t *testing.T) {
	t.Parallel()

	s := setupDashboardWithFailures(t, dashboard.WithGrouping(dashboard.GroupBySource))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")
	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Fatalf("/health: unexpected status %d", w.Code)
	}

	body := w.Body.String()

	if !strings.Contains(body, ">Services<") {
		t.Error("GroupBySource should render the fallback Services card for plain check names")
	}

	if strings.Contains(body, "Critical Failures") {
		t.Error("GroupBySource must not render severity cards")
	}
}

// TestWithNoDatastarRuntime_OmitsSDKDependentUI pins the contract consumers
// like CV need: with a custom patch client instead of the Datastar SDK, the
// filter box, no-match hint, and connection pill are omitted (all three
// would render dead without the SDK's expression engine and fetch events),
// while server-rendered behavior is untouched.
func TestWithNoDatastarRuntime_OmitsSDKDependentUI(t *testing.T) {
	t.Parallel()

	s := setupDashboardWithHealthyServices(t, 3,
		dashboard.WithDatastarSrc("/custom/mini-client.js"),
		dashboard.WithNoDatastarRuntime(),
	)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")
	if w.Code != http.StatusOK {
		t.Fatalf("/health: unexpected status %d", w.Code)
	}

	body := w.Body.String()

	// The bootstrap script's SOURCE mentions the pill element ids (its pill
	// section self-guards on their absence), so assert on rendered markup,
	// not bare substrings.
	for _, dead := range []string{"id=\"conn-state-live\"", "id=\"health-filter\"", "data-filter-empty"} {
		if strings.Contains(body, dead) {
			t.Errorf("SDK-dependent UI %q must not render without the Datastar runtime", dead)
		}
	}

	if !strings.Contains(body, "<details") {
		t.Error("server-rendered collapse must keep working without the Datastar runtime")
	}
}

// TestMobileRowStacking_Markup pins the mobile responsive contract: below
// the sm breakpoint (640px) the service table stops laying out as a table —
// the header hides, each row becomes a stacked card, and every cell shows
// its label. Pure Tailwind variants on our own markup (no stylesheet, no
// script), so SSE patches carry the identical classes. The visual side is
// covered by TestBrowser_MobileViewport against the harness CSS stub; the
// real Tailwind pipeline (Play CDN or consumer build) is the generator of
// these utilities by construction of the class names.
func TestMobileRowStacking_Markup(t *testing.T) {
	t.Parallel()

	s := setupDashboardWithHealthyServices(t, 2)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")
	body := w.Body.String()

	// The upstream-rendered <thead> has no class hook, so the hide rule
	// rides an arbitrary variant on the table element (HTML-escaped in the
	// output: & -> &amp;).
	if !strings.Contains(body, `[&amp;_thead]:max-sm:hidden`) {
		t.Error("table must hide its header row below the sm breakpoint")
	}

	if !strings.Contains(body, `max-sm:[&:not(.hidden)]:block`) {
		t.Error("rows must stack into blocks below the sm breakpoint")
	}

	// The :not(.hidden) guard is load-bearing: the client-side filter hides
	// rows by toggling the hidden class, and without the guard the stacking
	// display:block would out-cascade it on mobile (same specificity, later
	// sort order in generated sheets).
	for _, want := range []string{
		`max-sm:border-b`,
		`max-sm:last:border-b-0`,
		`dark:max-sm:border-gray-700`,
		`>Service</span>`,
		`>Status</span>`,
		`>Details</span>`,
		`sm:hidden">Service`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("mobile stacking markup missing %q", want)
		}
	}

	if strings.Count(body, `sm:hidden">Service`) != 2 {
		t.Errorf("each row must carry a mobile-only Service label, got %d for 2 rows",
			strings.Count(body, `sm:hidden">Service`))
	}
}

// TestRender_ContrastSafeStatusColors locks the WCAG AA color decisions for
// status-bearing text on the live page. Measured ratios (WCAG relative
// luminance, normal text needs >= 4.5:1; values computed 2026-09-10):
//
//	green-600 on white 3.30 / amber-600 on white 3.19 / gray-400 on white 2.54  -> FAIL, replaced
//	green-700 on white 5.02 / amber-700 on white 5.02 / red-600 on white 4.83   -> pass
//	gray-500 on white 4.83 / gray-400 on gray-800 5.78 (raw keys, labels)       -> pass
//	dark variants (gray-400/green-400/amber-400/red-400/blue-400 on gray-800): 5.31-8.79 -> pass
//
// Deliberately out of scope: the upstream CollapsibleSection chevron keeps
// its gray-400 idle class (decorative disclosure icon, upstream-owned
// markup), and the Badge/Alert/StatCard chip palettes are upstream's.
func TestRender_ContrastSafeStatusColors(t *testing.T) {
	t.Parallel()

	s := setupDashboardWithFailures(t,
		dashboard.WithEmbeddedDatastarSDK(),
		dashboard.WithDatastarSrc("/static/datastar.js"),
	)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")
	body := w.Body.String()

	for _, failing := range []string{
		`text-green-600`, // 3.30:1 on white
		`text-amber-600`, // 3.19:1 on white
	} {
		if strings.Contains(body, failing) {
			t.Errorf("rendered page uses contrast-failing color class %q", failing)
		}
	}

	for _, want := range []string{
		`text-green-700 dark:text-green-400`,
		`text-amber-700 dark:text-amber-400`,
		`text-red-600 dark:text-red-400`, // 4.83:1 on white — passes
		// Raw check keys and the details dash: 2.54:1 -> 4.83:1 on white,
		// and 3.04:1 -> 5.78:1 on the dark card surface.
		`font-mono text-xs text-gray-500 dark:text-gray-400`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("rendered page missing contrast-safe color class %q", want)
		}
	}
}
