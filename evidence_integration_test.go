package dashboard_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	do "github.com/samber/do/v2"
)

// setupAllFailing builds a probe whose every check is non-pass, so the
// evidence ratio reaches the full house (all proven) once the pusher has
// observed one tick.
func setupAllFailing(t *testing.T, opts ...dashboard.Option) *probeSetup {
	t.Helper()

	injector := do.New()
	provideUnhealthy(injector, "database", "down")
	provideUnhealthy(injector, "cache", "slow")
	invoke[*unhealthyService](t, injector, "database")
	invoke[*unhealthyService](t, injector, "cache")

	probe := health.New(injector,
		health.WithCriticalServices("database"),
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
		},
	}
}

// TestIntegration_EvidenceStripReflectsObservations drives a real probe +
// pusher through the failing fixture and verifies the truth strip reports
// the observed non-pass ratio live: cache and queue deviate, database
// stays pass, so 2 of 3 checks must be proven and the pass badge must
// disclose its lack of evidence.
func TestIntegration_EvidenceStripReflectsObservations(t *testing.T) {
	t.Parallel()

	s := setupDashboardWithFailures(t,
		dashboard.WithPushInterval(5*time.Millisecond),
		dashboard.WithEmbeddedDatastarSDK(),
		dashboard.WithDatastarSrc("/static/datastar.js"),
	)
	defer s.cleanup()

	body := waitForBody(t, s.mux, "/health", "Failure evidence: 2 of 3 checks")

	if !strings.Contains(body, "the other 1 green rows are unproven") {
		t.Errorf(
			"strip should name the unproven remainder: %s",
			extractLine(body, "Failure evidence"),
		)
	}

	if !strings.Contains(body, "never deviated from pass") {
		t.Error("the healthy check's pass badge should carry the unproven disclosure tooltip")
	}
}

// TestIntegration_EvidenceStrip_AllProven verifies the full-house wording:
// when every check has been observed non-pass, the strip must stop warning
// about unproven greens.
func TestIntegration_EvidenceStrip_AllProven(t *testing.T) {
	t.Parallel()

	s := setupAllFailing(t,
		dashboard.WithPushInterval(5*time.Millisecond),
		dashboard.WithEmbeddedDatastarSDK(),
		dashboard.WithDatastarSrc("/static/datastar.js"),
	)
	defer s.cleanup()

	body := waitForBody(t, s.mux, "/health", "Failure evidence: all 2 checks")

	if strings.Contains(body, "unproven") {
		t.Errorf(
			"all-proven dashboard must not warn about unproven rows: %s",
			extractLine(body, "Failure evidence"),
		)
	}
}

// extractLine returns the first line of body containing substr (for
// focused failure messages).
func extractLine(body, substr string) string {
	for line := range strings.SplitSeq(body, "\n") {
		if strings.Contains(line, substr) {
			return line
		}
	}

	return ""
}
