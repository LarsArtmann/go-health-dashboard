package dashboard

import (
	"bytes"
	"context"
	"strings"
	"testing"

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
		{name: "both cards share the grid", samples: 3, transitions: 1, wantGrid: true, wantTrend: true, wantTimeline: true},
		{name: "trend alone renders full-width", samples: 3, transitions: 0, wantGrid: false, wantTrend: true, wantTimeline: false},
		{name: "timeline alone renders full-width", samples: 0, transitions: 1, wantGrid: false, wantTrend: false, wantTimeline: true},
		{name: "neither renders the row", samples: 0, transitions: 0, wantGrid: false, wantTrend: false, wantTimeline: false},
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
