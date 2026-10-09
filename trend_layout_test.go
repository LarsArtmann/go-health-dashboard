package dashboard

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
)

// renderDashboardHTML renders the full page for markup assertions without
// the probe/pusher rig — the layout branching under test is a pure function
// of the viewModel's History and Timeline slices.
func renderDashboardHTML(t *testing.T, vm viewModel) string {
	t.Helper()

	var buf bytes.Buffer

	if err := View(vm).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}

	return buf.String()
}

// trendLayoutViewModel returns a minimal viewModel with the given number of
// trend samples and timeline entries.
func trendLayoutViewModel(samples, transitions int) viewModel {
	vm := viewModel{Title: "Layout", ShowStatCards: false}

	for range samples {
		vm.History = append(vm.History, trendPassValue)
	}

	for range transitions {
		vm.Timeline = append(vm.Timeline, TimelineEntry{
			At:       "12:00:00",
			Status:   string(health.StatusPass),
			Degraded: false,
		})
	}

	return vm
}

// TestTrendTimelineLayout pins the row layout branching: the two cards share
// a half/half grid only when BOTH have content — a lone card renders
// full-width instead of stranding half a row of dead space (caught visually
// on the 2026-10-09 README recapture: a trend-only dashboard showed the
// sparkline squeezed into the left half). Timeline-only is unreachable in
// production (the timeline derives from the trend ring) but asserted for
// symmetry.
func TestTrendTimelineLayout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		samples      int
		transitions  int
		wantGrid     bool
		wantTrend    bool
		wantTimeline bool
	}{
		{
			name:         "both cards share the grid",
			samples:      3,
			transitions:  1,
			wantGrid:     true,
			wantTrend:    true,
			wantTimeline: true,
		},
		{
			name:         "trend alone renders full-width",
			samples:      3,
			transitions:  0,
			wantGrid:     false,
			wantTrend:    true,
			wantTimeline: false,
		},
		{
			name:         "timeline alone renders full-width",
			samples:      0,
			transitions:  1,
			wantGrid:     false,
			wantTrend:    false,
			wantTimeline: true,
		},
		{
			name:         "neither renders the row",
			samples:      0,
			transitions:  0,
			wantGrid:     false,
			wantTrend:    false,
			wantTimeline: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			body := renderDashboardHTML(t, trendLayoutViewModel(tt.samples, tt.transitions))

			const gridClass = "sm:grid-cols-2"
			if got := strings.Contains(body, gridClass); got != tt.wantGrid {
				t.Errorf("grid class %q present = %v, want %v", gridClass, got, tt.wantGrid)
			}

			if got := strings.Contains(body, "Health Trend"); got != tt.wantTrend {
				t.Errorf("trend card present = %v, want %v", got, tt.wantTrend)
			}

			if got := strings.Contains(body, "Status Changes"); got != tt.wantTimeline {
				t.Errorf("timeline card present = %v, want %v", got, tt.wantTimeline)
			}
		})
	}
}

// TestTimelineNoteDisclosesCap pins the truncation notice: it renders only
// when the ring dropped transitions (TimelineTotal > shown), with copy that
// is honest about the window semantics — "latest", not a search-filter
// promise (the upstream ListNote truncated variant's advice line would lie
// here).
func TestTimelineNoteDisclosesCap(t *testing.T) {
	t.Parallel()

	capped := trendLayoutViewModel(4, 5)
	capped.TimelineTotal = 7

	body := renderDashboardHTML(t, capped)
	if !strings.Contains(body, "Showing the latest 5 of 7 status changes.") {
		t.Error("capped timeline must disclose the window: showing-the-latest line missing")
	}

	uncapped := trendLayoutViewModel(4, 5)

	body = renderDashboardHTML(t, uncapped)
	if strings.Contains(body, "status changes.") {
		t.Error("uncapped timeline must not render the truncation notice")
	}
}

// TestUpdatedAgePrecisionKeysOffPublicMode pins the age-precision decision
// (2026-10-09, ROADMAP Q2): private deployments render the client-ticked
// <time> element (initial page installs the singleton refresh script,
// patches emit the bare attribute-carrying element the script keeps
// ticking), public mode keeps the coarse formatAge stamp — precise ages
// fingerprint a deployment (exact restart times, instance uptime), so the
// stance follows the WithPublicMode audience switch.
func TestUpdatedAgePrecisionKeysOffPublicMode(t *testing.T) {
	t.Parallel()

	stamp := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	wantDatetime := `datetime="` + stamp.Format(time.RFC3339) + `"`

	initial := trendLayoutViewModel(0, 0)
	initial.LastUpdated = "12:00:00 UTC"
	initial.LastUpdatedTime = stamp
	initial.InitialRender = true

	body := renderDashboardHTML(t, initial)
	if !strings.Contains(body, "data-tc-relative") {
		t.Error("private initial render must emit the ticking element")
	}

	if !strings.Contains(body, wantDatetime) {
		t.Errorf("relative time must carry the pinned RFC3339 datetime %s", wantDatetime)
	}

	if !strings.Contains(body, "<script") {
		t.Error("private initial render must install the auto-refresh script")
	}

	patch := initial
	patch.InitialRender = false

	// A patch carries only dashboardContent — the page scripts live in the
	// shell — so the script-free assertion must scope to the patched
	// region, exactly what the wire carries.
	var content strings.Builder
	if err := dashboardContent(patch).Render(context.Background(), &content); err != nil {
		t.Fatalf("render dashboardContent: %v", err)
	}

	body = content.String()
	if !strings.Contains(body, "data-tc-relative") || !strings.Contains(body, wantDatetime) {
		t.Error("patch render must keep the ticking element with its datetime")
	}

	if strings.Contains(body, "<script") {
		t.Error("patch render must stay script-free (patch purity invariant)")
	}

	public := initial
	public.PublicMode = true

	body = renderDashboardHTML(t, public)
	if strings.Contains(body, "data-tc-relative") {
		t.Error("public mode must keep the coarse stamp (no ticking element)")
	}

	if !strings.Contains(body, "Updated 12:00:00 UTC") {
		t.Error("public mode must keep the absolute stamp line")
	}
}
