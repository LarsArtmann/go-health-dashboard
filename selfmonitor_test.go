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

// selfMonitorJSON is the slice of /health the self-monitoring assertions
// need: per-check status and error.
type selfMonitorJSON struct {
	Checks map[string]struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	} `json:"checks"`
}

// fetchSelfMonitor decodes one /health JSON snapshot.
func fetchSelfMonitor(t *testing.T, mux *http.ServeMux) selfMonitorJSON {
	t.Helper()

	w := doRequestWithAccept(t, mux, "/health", "application/json")
	if w.Code != http.StatusOK {
		t.Fatalf("/health JSON: want 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp selfMonitorJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode /health JSON: %v", err)
	}

	return resp
}

// TestSelfMonitoring_RegisteredDashboardAppearsInOwnTable pins the
// empirically-verified self-monitoring behavior: `dashboard.Register`
// stores the Dashboard in the same injector the probe evaluates on every
// tick, and because *Dashboard implements do.HealthcheckerWithContext the
// dashboard appears as a check row in its own table. This is the documented
// consequence behind the ROADMAP self-monitoring decision — if you do not
// want self-monitoring, construct with New instead of Register (or use a
// separate injector scope).
//
// The row's arrival is POLLED, not slept for: a fixed 80ms margin was the
// 2026-10-11 flake (want pass, got warn — the first probe tick had not
// evaluated the dashboard yet under -race parallel load). The status and
// error assertions run only after the row exists.
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

	mux := http.NewServeMux()
	dash.RegisterRoutes(mux)

	deadline := time.Now().Add(5 * time.Second)

	// The row legitimately ARRIVES as warn ("Start has not been called"):
	// the probe's first tick can beat dash.Start. Poll until it flips to
	// pass — the steady state the old fixed sleep waited for.
	var (
		row   selfMonitorJSON
		name  string
		check struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		}
	)

	for {
		row = fetchSelfMonitor(t, mux)

		for checkName, c := range row.Checks {
			if strings.Contains(checkName, "Dashboard") {
				name, check = checkName, c

				break
			}
		}

		if name != "" && check.Status == "pass" {
			break
		}

		if time.Now().After(deadline) {
			if name == "" {
				t.Fatalf("registered Dashboard did not appear in its own health table within 5s; checks: %v", row.Checks)
			}

			t.Fatalf("self-reported dashboard status never reached pass within 5s: %q (error %q)", check.Status, check.Error)
		}

		time.Sleep(20 * time.Millisecond)
	}

	if check.Error != "" {
		t.Errorf("passing dashboard %q self-reports a non-empty error: %q", name, check.Error)
	}
}
