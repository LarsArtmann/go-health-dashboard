package dashboard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	"github.com/larsartmann/go-health/aggregate"
	healthfederation "github.com/larsartmann/go-health/federation"
)

// The dashboard's Prober interface is narrow on purpose; per-source roll-up
// statuses ride an optional capability instead (SourceStatuser, mirroring
// Healthzer). go-health's aggregate gained the accessor in v0.6.0 and
// federation followed in v0.8.x — both pins keep the capability satisfiable
// by real merge surfaces, so a future drop breaks a build instead of
// silently softening every federated source card.
var (
	_ SourceStatuser = (*aggregate.Aggregate)(nil)
	_ SourceStatuser = (*healthfederation.Prober)(nil)
)

// sourceStatusProber is a Prober fake that also implements SourceStatuser:
// the shape of go-health's aggregate.Aggregate after the v0.7.0 bump. The
// statuses map is independent of the response on purpose — the whole point
// of the capability is verdicts the rows cannot show (a shutting-down
// source keeps its own all-pass checks under a merged fail).
type sourceStatusProber struct {
	resp     health.Response
	statuses map[string]health.Status
}

func (p *sourceStatusProber) CachedResponse() health.Response { return p.resp }

func (p *sourceStatusProber) RefreshInterval() time.Duration { return 20 * time.Millisecond }

func (p *sourceStatusProber) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
}

func (p *sourceStatusProber) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
}

func (p *sourceStatusProber) StartupHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
}

func (p *sourceStatusProber) SourceStatuses() map[string]health.Status { return p.statuses }

// plainProber is the minimal Prober WITHOUT the SourceStatuser capability —
// the shape of every pre-v0.6.0 probe. Both of go-health's merge surfaces
// (aggregate, federation as of v0.8.x) now carry the accessor, so it stands
// for third-party Probers rather than for anything in this fleet.
type plainProber struct {
	resp health.Response
}

func (p *plainProber) CachedResponse() health.Response { return p.resp }

func (p *plainProber) RefreshInterval() time.Duration { return 20 * time.Millisecond }

func (p *plainProber) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
}

func (p *plainProber) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
}

func (p *plainProber) StartupHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
}

// TestApplySourceStatuses_EscalatesButNeverSoftens pins the merge rule
// between the two truth sources: the probe's per-source accessor may
// worsen a card's verdict, never soften it — the card must not read
// healthier than either signal agrees to.
func TestApplySourceStatuses_EscalatesButNeverSoftens(t *testing.T) {
	t.Parallel()

	group := func(rowStatus health.Status) checkGroup {
		return checkGroup{
			Title:  "api",
			Status: worstGroupStatus([]checkRow{{Name: "api/db", Status: rowStatus}}),
			Rows:   []checkRow{{Name: "api/db", Status: rowStatus}},
		}
	}

	tests := []struct {
		name      string
		rows      health.Status
		accessor  health.Status
		nilMap    bool
		wantGroup health.Status
	}{
		{
			name:      "draining source escalates all-pass rows",
			rows:      health.StatusPass,
			accessor:  health.StatusFail,
			wantGroup: health.StatusFail,
		},
		{
			name:      "accessor warn escalates pass rows",
			rows:      health.StatusPass,
			accessor:  health.StatusWarn,
			wantGroup: health.StatusWarn,
		},
		{
			name:      "accessor pass never softens warn rows",
			rows:      health.StatusWarn,
			accessor:  health.StatusPass,
			wantGroup: health.StatusWarn,
		},
		{
			name:      "accessor pass never softens fail rows",
			rows:      health.StatusFail,
			accessor:  health.StatusPass,
			wantGroup: health.StatusFail,
		},
		{
			name:      "off in accessor keeps pass floor",
			rows:      health.StatusPass,
			accessor:  health.StatusOff,
			wantGroup: health.StatusPass,
		},
		{
			name:      "source absent from map keeps row fold",
			rows:      health.StatusWarn,
			wantGroup: health.StatusWarn,
		},
		{
			name:      "nil map keeps row fold",
			rows:      health.StatusWarn,
			nilMap:    true,
			wantGroup: health.StatusWarn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			groups := []checkGroup{group(tt.rows)}
			var statuses map[string]health.Status
			if !tt.nilMap {
				statuses = map[string]health.Status{}
				if tt.accessor != "" || tt.accessor == health.StatusPass {
					statuses["api"] = tt.accessor
				}
			}

			applySourceStatuses(groups, statuses)

			if groups[0].Status != tt.wantGroup {
				t.Errorf("group status = %v, want %v", groups[0].Status, tt.wantGroup)
			}
		})
	}
}

// TestBuildViewModelAtWithSourceStatuses_Modes pins the wiring contract: the
// map escalates cards in source grouping mode, is ignored in severity mode,
// and a nil map renders identically to buildViewModelAt.
func TestBuildViewModelAtWithSourceStatuses_Modes(t *testing.T) {
	t.Parallel()

	resp := health.Response{
		Status: health.StatusFail,
		Checks: map[string]health.Check{
			"api/db":    {Status: health.StatusPass},
			"api/cache": {Status: health.StatusPass},
		},
	}
	statuses := map[string]health.Status{"api": health.StatusFail}
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

	t.Run("source mode escalates the api card", func(t *testing.T) {
		t.Parallel()

		vm := buildViewModelAtWithSourceStatuses(resp, "t", "/sse", GroupBySource, now, statuses)

		if len(vm.Groups) != 1 {
			t.Fatalf("groups = %d, want 1", len(vm.Groups))
		}

		if vm.Groups[0].Status != health.StatusFail {
			t.Errorf("api card status = %v, want fail", vm.Groups[0].Status)
		}
	})

	t.Run("source mode nil map matches buildViewModelAt", func(t *testing.T) {
		t.Parallel()

		want := buildViewModelAt(resp, "t", "/sse", GroupBySource, now)
		got := buildViewModelAtWithSourceStatuses(resp, "t", "/sse", GroupBySource, now, nil)

		if got.Groups[0].Status != want.Groups[0].Status {
			t.Errorf(
				"nil-map status = %v, want row fold %v",
				got.Groups[0].Status,
				want.Groups[0].Status,
			)
		}
	})

	t.Run("severity mode ignores the map", func(t *testing.T) {
		t.Parallel()

		want := buildViewModelAt(resp, "t", "/sse", GroupBySeverity, now)
		got := buildViewModelAtWithSourceStatuses(resp, "t", "/sse", GroupBySeverity, now, statuses)

		for i := range want.Groups {
			if got.Groups[i].Status != want.Groups[i].Status {
				t.Errorf(
					"severity group %d status = %v, want %v",
					i,
					got.Groups[i].Status,
					want.Groups[i].Status,
				)
			}
		}
	})
}

// TestDashboard_SourceStatuses_EscalateCardStatus exercises the production
// wiring end to end at the view-model level: buildData consults the probe's
// capability and the draining source's card reports fail while its own rows
// all pass.
func TestDashboard_SourceStatuses_EscalateCardStatus(t *testing.T) {
	t.Parallel()

	probe := &sourceStatusProber{
		resp: health.Response{
			Status: health.StatusFail,
			Checks: map[string]health.Check{
				"api/db":    {Status: health.StatusPass},
				"api/cache": {Status: health.StatusPass},
			},
		},
		statuses: map[string]health.Status{"api": health.StatusFail},
	}

	dash := New(probe, WithGrouping(GroupBySource))

	vm := dash.buildData(httptest.NewRequest(http.MethodGet, "/health", nil))

	if len(vm.Groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(vm.Groups))
	}

	if vm.Groups[0].Status != health.StatusFail {
		t.Errorf("api card status = %v, want fail", vm.Groups[0].Status)
	}
}

// TestDashboard_SourceStatuses_RenderedCardLeavesHealthyGroup pins the user
// observable: without the capability, an all-pass source renders inside the
// collapsible healthy section ("all pass" summary). With it, the same rows
// render as a problem card — a draining source must never read as healthy.
func TestDashboard_SourceStatuses_RenderedCardLeavesHealthyGroup(t *testing.T) {
	t.Parallel()

	resp := health.Response{
		Status: health.StatusFail,
		Checks: map[string]health.Check{
			"api/db":    {Status: health.StatusPass},
			"api/cache": {Status: health.StatusPass},
		},
	}

	t.Run("capability escalates the card out of the healthy section", func(t *testing.T) {
		t.Parallel()

		probe := &sourceStatusProber{
			resp:     resp,
			statuses: map[string]health.Status{"api": health.StatusFail},
		}
		dash := New(probe, WithGrouping(GroupBySource))

		rec := httptest.NewRecorder()
		dash.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

		body := rec.Body.String()
		if strings.Contains(body, "all pass") {
			t.Error("draining source rendered inside the healthy section (\"all pass\" found)")
		}

		if !strings.Contains(body, "group-problems") {
			t.Error("draining source did not render as a problem card")
		}
	})

	t.Run("absent capability keeps the row fold", func(t *testing.T) {
		t.Parallel()

		dash := New(&plainProber{resp: resp}, WithGrouping(GroupBySource))

		rec := httptest.NewRecorder()
		dash.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

		if !strings.Contains(rec.Body.String(), "all pass") {
			t.Error("all-pass rows without the capability should render in the healthy section")
		}
	})
}

// TestPusher_SourceStatuses_PatchEscalates covers the SSE patch path: the
// pusher's render must agree with the page HTML, so a draining source's
// patch carries the problem card too.
func TestPusher_SourceStatuses_PatchEscalates(t *testing.T) {
	t.Parallel()

	probe := &sourceStatusProber{
		resp: health.Response{
			Status: health.StatusFail,
			Checks: map[string]health.Check{
				"api/db":    {Status: health.StatusPass},
				"api/cache": {Status: health.StatusPass},
			},
		},
		statuses: map[string]health.Status{"api": health.StatusFail},
	}

	dash := New(probe, WithGrouping(GroupBySource))
	p := newPusher(dash)

	evt, ok := p.renderPatch(probe.resp)
	if !ok {
		t.Fatal("renderPatch failed")
	}

	if strings.Contains(evt.Data, "all pass") {
		t.Error("patch rendered the draining source inside the healthy section")
	}

	if !strings.Contains(evt.Data, "group-problems") {
		t.Error("patch did not render the draining source as a problem card")
	}
}
