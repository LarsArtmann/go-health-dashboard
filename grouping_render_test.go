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
