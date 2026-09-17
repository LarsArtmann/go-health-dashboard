package dashboard

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
)

// goldenRenderResponse is the frozen fixture the golden files capture:
// every shape the renderer distinguishes (severity groups, an aggregate
// source/check key, a fully-qualified type name, a generic type, a long
// error past the truncation threshold) in one deterministic response.
// The go-health v0.2.0 per-check metadata is exercised in all three
// shapes: since+duration, since-only, and duration-only.
func goldenRenderResponse() health.Response {
	return health.Response{
		Status:       health.StatusFail,
		Version:      "1.2.3",
		Uptime:       "1h0m0s",
		Checks:       goldenChecks(),
		ShuttingDown: false,
	}
}

// goldenSince is the fixed state-entry time for fixture checks; paired
// with goldenNow below it yields a stable "17m" age in the golden bytes.
var goldenSince = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)

// goldenNow is the pinned build clock for the golden view model.
var goldenNow = time.Date(2026, 9, 16, 12, 17, 0, 0, time.UTC)

func goldenChecks() map[string]health.Check {
	longError := strings.Repeat("x", 120)

	return map[string]health.Check{
		"api/postgres": {Status: health.StatusPass, DurationNanos: int64(823 * time.Microsecond)},
		"api/mail": {
			Status: health.StatusWarn,
			Error:  "smtp timeout after 5s",
			Since:  goldenSince,
		},
		"worker/queue": {
			Status:        health.StatusFail,
			Error:         "connection refused",
			Since:         goldenSince,
			DurationNanos: int64(42 * time.Millisecond),
		},
		"*github.com/x/y/handlers.Handlers": {Status: health.StatusPass},
		"*github.com/x/repo/store.Store[string]": {
			Status: health.StatusWarn,
			Error:  "cache pressure",
		},
		"database": {Status: health.StatusFail, Error: longError},
	}
}

// goldenViewModel builds the fully deterministic view model: every volatile
// input (timestamps, uptime, latency, routes) is pinned so the rendered
// bytes only change when the template or mapping logic changes.
func goldenViewModel(t *testing.T) viewModel {
	t.Helper()

	return goldenViewModelForMode(t, GroupBySeverity)
}

// goldenViewModelForMode builds the deterministic view model for the given
// grouping axis: every volatile input (timestamps, since-ages, uptime,
// latency, routes) is pinned so the rendered bytes only change when the
// template or mapping logic changes.
func goldenViewModelForMode(t *testing.T, mode GroupMode) viewModel {
	t.Helper()

	vm := buildViewModelAt(goldenRenderResponse(), "Golden Service", "/health/sse", mode, goldenNow)
	vm.LastUpdated = "12:00:00 UTC"
	// Zeroed on purpose: the Updated-age span renders from the REAL clock
	// (view.templ calls time.Now directly), so a real LastUpdatedTime would
	// age past formatAge's "just now" bucket and time-bomb the golden. The
	// row-level since-ages are safe: they are frozen into SinceText at
	// build time from the pinned goldenNow clock.
	vm.LastUpdatedTime = time.Time{}
	vm.LatencyMs = 42
	vm.CSSPath = "/static/app.css"
	vm.DatastarSrc = "/static/datastar.js"
	vm.ShowStatCards = true

	applyCollapsePolicy(&vm, 8)

	return vm
}

var goldenUpdate = flag.Bool("golden-update", false, "rewrite golden render files")

// TestGoldenRender_Severity captures the full page render for the frozen
// fixture. Run `go test ./... -run TestGoldenRender -golden-update` after
// an INTENTIONAL markup change; the diff in review then shows exactly what
// the renderer now emits.
func TestGoldenRender_Severity(t *testing.T) {
	t.Parallel()

	goldenRender(t, "severity", goldenViewModel(t))
}

// TestGoldenRender_Source captures the same fixture under WithGrouping's
// source axis, so grouping regressions are as reviewable as markup churn.
func TestGoldenRender_Source(t *testing.T) {
	t.Parallel()

	goldenRender(t, "source", goldenViewModelForMode(t, GroupBySource))
}

// goldenEvidenceStart is the fixed observation-window start for the
// evidence goldens; paired with goldenNow it yields stable strip wording.
var goldenEvidenceStart = time.Date(2026, 9, 16, 11, 0, 0, 0, time.UTC)

// TestGoldenRender_ZeroProvenWarning locks the exact wording of the
// health-washing warning: a fully green table whose evidence log has never
// seen a deviation must render the explicit "green rows are unproven"
// strip, and every pass badge must carry the unproven disclosure tooltip.
func TestGoldenRender_ZeroProvenWarning(t *testing.T) {
	t.Parallel()

	vm := goldenViewModel(t)
	vm.Evidence = evidenceSummary{
		Since: goldenEvidenceStart,
		Total: len(goldenChecks()),
	}

	goldenRender(t, "zeroproven", vm)
}

// TestGoldenRender_PublicMode locks the anonymized render exactly as
// production builds it: anonymizeViewModel runs BEFORE the evidence
// summary is attached, so names and errors are masked while the strip
// keeps its (non-identifying) counts — and, keyed by the original names,
// every pass badge honestly renders as unproven in public mode.
func TestGoldenRender_PublicMode(t *testing.T) {
	t.Parallel()

	vm := goldenViewModel(t)
	anonymizeViewModel(&vm)
	vm.Evidence = evidenceSummary{
		Since:  goldenEvidenceStart,
		Total:  len(goldenChecks()),
		Proven: 1,
		lastNonPassBy: map[string]time.Time{
			"api/mail": goldenSince,
		},
	}

	goldenRender(t, "publicmode", vm)
}

func goldenRender(t *testing.T, name string, vm viewModel) {
	t.Helper()

	var buf bytes.Buffer

	if err := View(vm).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}

	goldenPath := filepath.Join("testdata", "golden", name+".html")

	if *goldenUpdate {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o750); err != nil {
			t.Fatalf("mkdir golden: %v", err)
		}

		if err := os.WriteFile(goldenPath, buf.Bytes(), 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}

		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("golden file missing (run with -golden-update once): %v", err)
	}

	if !bytes.Equal(bytes.TrimSpace(want), bytes.TrimSpace(buf.Bytes())) {
		t.Errorf(
			"render drifted from golden %s — if intentional, regenerate with `go test -run TestGoldenRender -golden-update` and review the diff",
			goldenPath,
		)
	}
}
