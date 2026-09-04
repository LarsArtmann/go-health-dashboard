package dashboard_test

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"strings"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	"github.com/samber/do/v2"
)

// TestSelfMonitoring_RegisteredDashboardAppearsInOwnTable pins the
// empirically-verified self-monitoring behavior: `dashboard.Register`
// stores the Dashboard in the same injector the probe evaluates on every
// tick, and because *Dashboard implements do.HealthcheckerWithContext the
// dashboard appears as a check row in its own table. This is the documented
// consequence behind the ROADMAP self-monitoring decision — if you do not
// want self-monitoring, construct with New instead of Register (or use a
// separate injector scope).
func TestSelfMonitoring_RegisteredDashboardAppearsInOwnTable(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "database")
	invoke[*healthyService](t, injector, "database")

	probe := health.New(injector, health.WithRefreshInterval(20*time.Millisecond))

	dash := dashboard.Register(injector, probe)

	startCtx, cancelStart := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStart()

	if err := probe.Start(startCtx); err != nil {
		t.Fatalf("probe start: %v", err)
	}
	defer func() { probe.Shutdown() }()

	if err := dash.Start(startCtx); err != nil {
		t.Fatalf("dash start: %v", err)
	}
	defer func() { dash.Shutdown() }()
	<-time.After(80 * time.Millisecond)

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	w := doRequestWithAccept(t, mux, "/health", "application/json")
	if w.Code != http.StatusOK {
		t.Fatalf("/health JSON: want 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Checks map[string]struct {
			Status string `json:"status"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode /health JSON: %v", err)
	}

	found := false
	for name, check := range resp.Checks {
		if strings.Contains(name, "Dashboard") {
			found = true
			if check.Status != "pass" {
				t.Errorf("self-reported dashboard status: want pass, got %q", check.Status)
			}
		}
	}

	if !found {
		t.Errorf(
			"registered Dashboard did not appear in its own health table; checks: %v",
			resp.Checks,
		)
	}
}
