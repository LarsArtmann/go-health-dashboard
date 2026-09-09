package dashboard_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	"github.com/samber/do/v2"
)

// setupDashboardWithHealthyServices builds a started probe/dashboard pair
// with n healthy services named svc-01..svc-n.
func setupDashboardWithHealthyServices(t *testing.T, n int, opts ...dashboard.Option) *probeSetup {
	t.Helper()

	injector := do.New()
	for i := range n {
		name := fmt.Sprintf("svc-%02d", i+1)

		provideHealthy(injector, name)
		invoke[*healthyService](t, injector, name)
	}

	probe := health.New(injector,
		health.WithVersion("1.0.0"),
		health.WithRefreshInterval(100*time.Millisecond),
	)

	dash := dashboard.New(probe, opts...)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	if err := probe.Start(t.Context()); err != nil {
		t.Fatalf("probe.Start: %v", err)
	}

	if err := dash.Start(t.Context()); err != nil {
		t.Fatalf("dash.Start: %v", err)
	}

	return &probeSetup{
		probe: probe,
		dash:  dash,
		mux:   mux,
		cleanup: func() {
			dash.Shutdown()
			probe.Shutdown()
		},
	}
}

// detailsOpenTag returns the first <details ...> opening tag in body.
func detailsOpenTag(body string) string {
	start := strings.Index(body, "<details")
	if start == -1 {
		return ""
	}

	end := strings.Index(body[start:], ">")
	if end == -1 {
		return ""
	}

	return body[start : start+end+1]
}

// assertCollapseState requests the dashboard HTML and asserts the healthy
// group's <details> render state.
func assertCollapseState(t *testing.T, s *probeSetup, wantCollapsed bool) string {
	t.Helper()

	w := doRequest(t, s.mux, "/health")
	body := w.Body.String()

	tag := detailsOpenTag(body)
	if tag == "" {
		t.Fatalf("healthy group should render as a <details> section, body:\n%s", body)
	}

	hasOpen := strings.Contains(tag, " open")
	if hasOpen == wantCollapsed {
		t.Errorf("collapsed=%v but details open-attribute present=%v (tag %q)", wantCollapsed, hasOpen, tag)
	}

	return body
}

func TestCollapse_DefaultThresholdCollapsesAt8(t *testing.T) {
	t.Parallel()

	s := setupDashboardWithHealthyServices(t, 9)
	defer s.cleanup()

	body := assertCollapseState(t, s, true)

	if !strings.Contains(body, "Healthy Services · 9 · all pass") {
		t.Errorf("collapsed summary should state count and pass status, body:\n%s", body)
	}
}

func TestCollapse_BelowThresholdStaysExpanded(t *testing.T) {
	t.Parallel()

	s := setupDashboardWithHealthyServices(t, 7)
	defer s.cleanup()

	assertCollapseState(t, s, false)
}

func TestCollapse_WithHealthyGroupExpandedOverridesThreshold(t *testing.T) {
	t.Parallel()

	s := setupDashboardWithHealthyServices(t, 9, dashboard.WithHealthyGroupExpanded())
	defer s.cleanup()

	assertCollapseState(t, s, false)
}

func TestCollapse_CustomThreshold(t *testing.T) {
	t.Parallel()

	t.Run("collapsed at threshold", func(t *testing.T) {
		t.Parallel()

		s := setupDashboardWithHealthyServices(t, 4, dashboard.WithHealthyGroupCollapse(4))
		defer s.cleanup()

		assertCollapseState(t, s, true)
	})

	t.Run("expanded below threshold", func(t *testing.T) {
		t.Parallel()

		s := setupDashboardWithHealthyServices(t, 4, dashboard.WithHealthyGroupCollapse(5))
		defer s.cleanup()

		assertCollapseState(t, s, false)
	})

	t.Run("negative threshold behaves as expanded", func(t *testing.T) {
		t.Parallel()

		s := setupDashboardWithHealthyServices(t, 4, dashboard.WithHealthyGroupCollapse(-3))
		defer s.cleanup()

		assertCollapseState(t, s, false)
	})
}

func TestCollapse_FailingAndWarningGroupsStayCards(t *testing.T) {
	t.Parallel()

	s := setupDashboardWithFailures(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")
	body := w.Body.String()

	if c := strings.Count(body, "<details"); c != 1 {
		t.Fatalf("only the healthy group may render as <details>, found %d", c)
	}

	if !strings.Contains(detailsOpenTag(body), " open") {
		t.Error("single healthy row must stay expanded below the default threshold")
	}

	// The warning group heading is a Card, not the details summary.
	summaryIdx := strings.Index(body, "<details")
	warningIdx := strings.Index(body, "Non-Critical Issues")
	if warningIdx == -1 {
		t.Fatal("warning group heading missing")
	}

	if warningIdx > summaryIdx {
		t.Error("warning group heading should render before (outside) the healthy details section")
	}
}

func TestCollapse_CollapsedRenderKeepsCSPInvariants(t *testing.T) {
	t.Parallel()

	s := setupDashboardWithHealthyServices(t, 10, dashboard.WithNonce("collapse-nonce"))
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")
	body := w.Body.String()

	if c := strings.Count(body, "<style"); c != 0 {
		t.Errorf("collapsed render must contain no <style> blocks, found %d", c)
	}

	if c := strings.Count(body, `style="`); c != 0 {
		t.Errorf("collapsed render must contain no inline style= attributes, found %d", c)
	}

	if !strings.Contains(body, "Healthy Services · 10 · all pass") {
		t.Errorf("summary should state count and pass status, body:\n%s", body)
	}
}
