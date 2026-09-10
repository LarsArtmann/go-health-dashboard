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
func goldenRenderResponse() health.Response {
	return health.Response{
		Status:       health.StatusFail,
		Version:      "1.2.3",
		Uptime:       "1h0m0s",
		Checks:       goldenChecks(),
		ShuttingDown: false,
	}
}

func goldenChecks() map[string]health.Check {
	longError := strings.Repeat("x", 120)

	return map[string]health.Check{
		"api/postgres": {Status: health.StatusPass},
		"api/mail": {
			Status: health.StatusWarn,
			Error:  "smtp timeout after 5s",
		},
		"worker/queue": {
			Status: health.StatusFail,
			Error:  "connection refused",
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

	vm := buildViewModel(goldenRenderResponse(), "Golden Service", "/health/sse", GroupBySeverity)
	vm.LastUpdated = "12:00:00 UTC"
	// Zeroed on purpose: a real timestamp would age past formatAge's
	// "just now" bucket and time-bomb the golden. The zero-value guard
	// omits the age span; formatAge has its own unit tests.
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

	vm := buildViewModel(goldenRenderResponse(), "Golden Service", "/health/sse", GroupBySource)
	vm.LastUpdated = "12:00:00 UTC"
	vm.LastUpdatedTime = time.Time{} // zeroed: see severity golden
	vm.LatencyMs = 42
	vm.CSSPath = "/static/app.css"
	vm.DatastarSrc = "/static/datastar.js"
	vm.ShowStatCards = true

	goldenRender(t, "source", vm)
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
