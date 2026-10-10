package dashboard_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
)

// verdictProber is a stubProber that additionally carries a probe-side
// cascade verdict (the method shape of go-health's Probe.HealthCheck), so
// the WithCascadeProbeVerdict forwarding tests exercise the optional
// capability detection without a real probe.
type verdictProber struct {
	*stubProber
	verdict func(context.Context) error
}

func (v *verdictProber) HealthCheck(ctx context.Context) error {
	return v.verdict(ctx)
}

// healthResponse builds a minimal health response with one check.
func healthResponse(status health.Status, checkName string, checkErr string) health.Response {
	resp := health.Response{Status: status, Checks: map[string]health.Check{}}

	if checkName != "" {
		c := health.Check{Status: status}
		if checkErr != "" {
			c.Error = checkErr
		}
		resp.Checks[checkName] = c
	}

	return resp
}

// renderDashboardHTML serves the dashboard's HTML once through the mux.
func renderDashboardHTML(t *testing.T, d *dashboard.Dashboard) string {
	t.Helper()

	mux := http.NewServeMux()
	d.RegisterRoutes(mux)

	w := httptest.NewRecorder()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("/health returned %d", w.Code)
	}

	return w.Body.String()
}

// TestObserve_CapturesSubIntervalFlap pins the ADR-0003 contract end to
// end: a flap that happens entirely BETWEEN pusher ticks — a fail
// observation followed by a pass, with no Start/ tick in between — shows
// up in both the trend ring (via /health/trend) and the evidence strip
// ("1 of 1 checks have deviated").
func TestObserve_CapturesSubIntervalFlap(t *testing.T) {
	t.Parallel()

	probe := newStubProber(healthResponse(health.StatusPass, "db", ""))
	d := dashboard.New(probe,
		dashboard.WithTrend(10),
		dashboard.WithPushInterval(time.Hour), // no tick during the test
	)

	fail := healthResponse(health.StatusFail, "db", "connection refused")
	pass := healthResponse(health.StatusPass, "db", "")

	d.Observe(fail)
	d.Observe(pass)

	// Trend ring: two probe-cadence samples recorded without any pusher tick.
	mux := http.NewServeMux()
	d.RegisterRoutes(mux)

	w := httptest.NewRecorder()

	req := httptest.NewRequest(http.MethodGet, "/health/trend", nil)
	mux.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, `"status":"fail"`) || !strings.Contains(body, `"status":"pass"`) {
		t.Errorf("trend samples missing the flap pair; body: %.400s", body)
	}

	// Evidence strip: the fail observation proves the check can deviate.
	html := renderDashboardHTML(t, d)
	if !strings.Contains(html, "1 of 1 checks have deviated") {
		t.Errorf("evidence strip missing the proven deviation; html: %.400s", html)
	}
}

// TestObserve_NilSafeWithoutTrend: Observe must not panic without WithTrend
// and must still accrue evidence.
func TestObserve_NilSafeWithoutTrend(t *testing.T) {
	t.Parallel()

	probe := newStubProber(healthResponse(health.StatusPass, "db", ""))
	d := dashboard.New(probe)

	d.Observe(healthResponse(health.StatusFail, "db", "boom"))
	d.Observe(healthResponse(health.StatusPass, "db", ""))

	html := renderDashboardHTML(t, d)
	if !strings.Contains(html, "1 of 1 checks have deviated") {
		t.Errorf("evidence strip missing the proven deviation without trend; html: %.400s", html)
	}
}

// TestHealthCheck_CascadeVerdictForwarding pins the ADR-0004 contract:
// default HealthCheck stays pusher-scoped; WithCascadeProbeVerdict forwards
// the prober's verdict once the pusher-state checks pass.
func TestHealthCheck_CascadeVerdictForwarding(t *testing.T) {
	t.Parallel()

	probeErr := errors.New("probe: roll-up fail")

	newVerdictProber := func() *verdictProber {
	return &verdictProber{
		stubProber: newStubProber(healthResponse(health.StatusPass, "db", "")),
		verdict:    func(context.Context) error { return probeErr },
	}
	}

	t.Run("default stays pusher-scoped", func(t *testing.T) {
		t.Parallel()

		vp := newVerdictProber()
		d := dashboard.New(vp)

		if err := d.Start(t.Context()); err != nil {
			t.Fatalf("Start: %v", err)
		}
		defer d.Shutdown()

		if err := d.HealthCheck(t.Context()); err != nil {
			t.Errorf("default HealthCheck forwarded the probe verdict: %v", err)
		}
	})

	t.Run("opt-in forwards the verdict", func(t *testing.T) {
		t.Parallel()

		vp := newVerdictProber()
		d := dashboard.New(vp, dashboard.WithCascadeProbeVerdict())

		if err := d.Start(t.Context()); err != nil {
			t.Fatalf("Start: %v", err)
		}
		defer d.Shutdown()

		err := d.HealthCheck(t.Context())
		if !errors.Is(err, probeErr) {
			t.Errorf("opt-in HealthCheck = %v, want the probe verdict %v", err, probeErr)
		}
	})

	t.Run("pusher-state errors take precedence", func(t *testing.T) {
		t.Parallel()

		vp := newVerdictProber()
		d := dashboard.New(vp, dashboard.WithCascadeProbeVerdict())

		// Never started: ErrPusherNotStarted must win over the (failing)
		// probe verdict.
		err := d.HealthCheck(t.Context())
		if !errors.Is(err, dashboard.ErrPusherNotStarted) {
			t.Errorf("pre-Start HealthCheck = %v, want ErrPusherNotStarted", err)
		}
	})

	t.Run("prober without the capability falls back silently", func(t *testing.T) {
		t.Parallel()

		probe := newStubProber(healthResponse(health.StatusPass, "db", ""))
		d := dashboard.New(probe, dashboard.WithCascadeProbeVerdict())

		if err := d.Start(t.Context()); err != nil {
			t.Fatalf("Start: %v", err)
		}
		defer d.Shutdown()

		if err := d.HealthCheck(t.Context()); err != nil {
			t.Errorf("capability-less prober forwarded an error: %v", err)
		}
	})
}
