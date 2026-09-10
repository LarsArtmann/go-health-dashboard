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
