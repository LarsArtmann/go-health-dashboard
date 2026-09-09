package dashboard

import (
	"fmt"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	"github.com/larsartmann/templ-components/display"
	"github.com/larsartmann/templ-components/feedback"
)

func TestMapStatusToBadge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status health.Status
		want   display.BadgeType
	}{
		{"pass", health.StatusPass, display.BadgeSuccess},
		{"warn", health.StatusWarn, display.BadgeWarning},
		{"fail", health.StatusFail, display.BadgeError},
		{"unknown", health.Status("unknown"), display.BadgeNeutral},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := mapStatusToBadge(tt.status); got != tt.want {
				t.Errorf("mapStatusToBadge(%s): want %s, got %s", tt.status, tt.want, got)
			}
		})
	}
}

func TestMapStatusToFeedback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status health.Status
		want   feedback.FeedbackType
	}{
		{"pass", health.StatusPass, feedback.FeedbackSuccess},
		{"warn", health.StatusWarn, feedback.FeedbackWarning},
		{"fail", health.StatusFail, feedback.FeedbackError},
		{"unknown", health.Status("unknown"), feedback.FeedbackInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := mapStatusToFeedback(tt.status); got != tt.want {
				t.Errorf("mapStatusToFeedback(%s): want %s, got %s", tt.status, tt.want, got)
			}
		})
	}
}

func TestMapStatusToText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status health.Status
		want   string
	}{
		{"pass", health.StatusPass, "All Systems Operational"},
		{"warn", health.StatusWarn, "Degraded — Non-Critical Issues"},
		{"fail", health.StatusFail, "Unhealthy — Critical Failures"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := mapStatusToText(tt.status); got != tt.want {
				t.Errorf("mapStatusToText(%s): want %q, got %q", tt.status, tt.want, got)
			}
		})
	}
}

func TestGroupChecks_PartitionsBySeverity(t *testing.T) {
	t.Parallel()

	checks := map[string]health.Check{
		"database": {Status: health.StatusFail, Error: "connection refused"},
		"redis":    {Status: health.StatusWarn, Error: "high latency"},
		"api":      {Status: health.StatusPass},
		"cache":    {Status: health.StatusPass},
		"queue":    {Status: health.StatusFail, Error: "timeout"},
		"exporter": {Status: health.StatusWarn, Error: "slow"},
	}

	groups := groupChecks(checks)

	if len(groups) != 3 {
		t.Fatalf("want 3 groups (failing, warning, healthy), got %d", len(groups))
	}

	if groups[0].Title != "Critical Failures" || len(groups[0].Rows) != 2 {
		t.Errorf("group 0: want 'Critical Failures' with 2 rows, got %q with %d",
			groups[0].Title, len(groups[0].Rows))
	}

	if groups[1].Title != "Non-Critical Issues" || len(groups[1].Rows) != 2 {
		t.Errorf("group 1: want 'Non-Critical Issues' with 2 rows, got %q with %d",
			groups[1].Title, len(groups[1].Rows))
	}

	if groups[2].Title != "Healthy Services" || len(groups[2].Rows) != 2 {
		t.Errorf("group 2: want 'Healthy Services' with 2 rows, got %q with %d",
			groups[2].Title, len(groups[2].Rows))
	}
}

func TestGroupChecks_SortedAlphabetically(t *testing.T) {
	t.Parallel()

	checks := map[string]health.Check{
		"zebra": {Status: health.StatusFail},
		"alpha": {Status: health.StatusFail},
		"mongo": {Status: health.StatusFail},
	}

	groups := groupChecks(checks)

	if len(groups) != 1 {
		t.Fatalf("want 1 group, got %d", len(groups))
	}

	names := make([]string, 0, len(groups[0].Rows))
	for _, row := range groups[0].Rows {
		names = append(names, row.Name)
	}

	want := []string{"alpha", "mongo", "zebra"}
	for i, name := range want {
		if names[i] != name {
			t.Errorf("row %d: want %s, got %s", i, name, names[i])
		}
	}
}

func TestGroupChecks_EmptyMap(t *testing.T) {
	t.Parallel()

	groups := groupChecks(map[string]health.Check{})
	if len(groups) != 0 {
		t.Errorf("empty checks: want 0 groups, got %d", len(groups))
	}
}

func TestGroupChecks_OnlyOneSeverity(t *testing.T) {
	t.Parallel()

	checks := map[string]health.Check{
		"db":    {Status: health.StatusPass},
		"cache": {Status: health.StatusPass},
	}

	groups := groupChecks(checks)

	if len(groups) != 1 {
		t.Fatalf("want 1 group (healthy only), got %d", len(groups))
	}

	if groups[0].Title != "Healthy Services" {
		t.Errorf("want 'Healthy Services', got %q", groups[0].Title)
	}
}

func TestBuildViewModel(t *testing.T) {
	t.Parallel()

	resp := health.Response{
		Status:         health.StatusWarn,
		Version:        "1.2.3",
		Uptime:         "5m0s",
		TotalLatencyMs: 42,
		Checks: map[string]health.Check{
			"db":    {Status: health.StatusPass},
			"cache": {Status: health.StatusWarn, Error: "slow"},
		},
	}

	vm := buildViewModel(resp, "My Service", "/health/sse")

	if vm.Title != "My Service" {
		t.Errorf("title: want 'My Service', got %q", vm.Title)
	}

	if vm.Status != health.StatusWarn {
		t.Errorf("status: want warn, got %s", vm.Status)
	}

	if vm.FeedbackType != feedback.FeedbackWarning {
		t.Errorf("feedback type: want warning, got %s", vm.FeedbackType)
	}

	if vm.SSEURL != "/health/sse" {
		t.Errorf("SSE URL: want '/health/sse', got %q", vm.SSEURL)
	}

	if vm.Version != "1.2.3" {
		t.Errorf("version: want '1.2.3', got %q", vm.Version)
	}

	if vm.LatencyMs != 42 {
		t.Errorf("latency: want 42, got %d", vm.LatencyMs)
	}

	if len(vm.Groups) != 2 {
		t.Errorf("groups: want 2 (warning + healthy), got %d", len(vm.Groups))
	}
}

func TestBuildViewModel_ShuttingDown(t *testing.T) {
	t.Parallel()

	resp := health.Response{
		Status:       health.StatusPass,
		ShuttingDown: true,
	}

	vm := buildViewModel(resp, "Test", "/health/sse")

	if vm.FeedbackType != feedback.FeedbackWarning {
		t.Errorf("shutdown feedback: want warning, got %s", vm.FeedbackType)
	}

	if vm.StatusText != "Shutting Down — Draining Traffic" {
		t.Errorf("shutdown text: want 'Shutting Down — Draining Traffic', got %q", vm.StatusText)
	}
}

func TestFingerprintChecks_Deterministic(t *testing.T) {
	t.Parallel()

	checks := map[string]health.Check{
		"db":    {Status: health.StatusPass},
		"cache": {Status: health.StatusWarn, Error: "slow"},
		"queue": {Status: health.StatusFail, Error: "timeout"},
	}

	fp1 := fingerprintChecks(checks)
	fp2 := fingerprintChecks(checks)

	if fp1 != fp2 {
		t.Errorf("fingerprint must be deterministic:\n  fp1=%q\n  fp2=%q", fp1, fp2)
	}
}

func TestFingerprintChecks_DetectsChanges(t *testing.T) {
	t.Parallel()

	before := map[string]health.Check{
		"db": {Status: health.StatusPass},
	}

	after := map[string]health.Check{
		"db": {Status: health.StatusFail, Error: "connection refused"},
	}

	if fingerprintChecks(before) == fingerprintChecks(after) {
		t.Error("fingerprint should differ when checks change")
	}
}

func TestFingerprintChecks_EmptyMap(t *testing.T) {
	t.Parallel()

	if fp := fingerprintChecks(map[string]health.Check{}); fp != "" {
		t.Errorf("empty checks fingerprint: want empty string, got %q", fp)
	}
}

func TestFingerprintChecks_NoDelimiterCollision(t *testing.T) {
	t.Parallel()

	// A check name containing delimiter characters must not alias a
	// different split of the same bytes across name/status/error. The
	// historical ":...;" separator encoding collided on exactly this pair.
	aliased := map[string]health.Check{
		"a:b": {Status: health.StatusPass, Error: "c"},
	}
	separate := map[string]health.Check{
		"a": {Status: health.StatusPass, Error: "b:c"},
	}

	if fingerprintChecks(aliased) == fingerprintChecks(separate) {
		t.Error("fingerprint collision: delimiter-bearing name aliases a different field split")
	}
}

// healthyChecks builds an n-entry all-pass checks map.
func healthyChecks(n int) map[string]health.Check {
	checks := make(map[string]health.Check, n)
	for i := range n {
		checks[fmt.Sprintf("svc-%02d", i)] = health.Check{Status: health.StatusPass}
	}

	return checks
}

func TestApplyCollapsePolicy_ThresholdBoundary(t *testing.T) {
	t.Parallel()

	const threshold = 8

	tests := []struct {
		name          string
		healthyRows   int
		wantCollapsed bool
	}{
		{"below threshold", 7, false},
		{"at threshold", 8, true},
		{"above threshold", 9, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vm := buildViewModel(health.Response{
				Status: health.StatusPass,
				Checks: healthyChecks(tt.healthyRows),
			}, "Test", "/health/sse")

			applyCollapsePolicy(&vm, threshold)

			if vm.HealthyCount != tt.healthyRows {
				t.Errorf("HealthyCount: want %d, got %d", tt.healthyRows, vm.HealthyCount)
			}

			if vm.HealthyCollapsed != tt.wantCollapsed {
				t.Errorf("HealthyCollapsed with %d healthy rows (threshold %d): want %v, got %v",
					tt.healthyRows, threshold, tt.wantCollapsed, vm.HealthyCollapsed)
			}
		})
	}
}

func TestApplyCollapsePolicy_NoHealthyGroup(t *testing.T) {
	t.Parallel()

	vm := buildViewModel(health.Response{
		Status: health.StatusFail,
		Checks: map[string]health.Check{
			"db": {Status: health.StatusFail, Error: "down"},
		},
	}, "Test", "/health/sse")

	applyCollapsePolicy(&vm, 8)

	if vm.HealthyCount != 0 {
		t.Errorf("HealthyCount with no healthy group: want 0, got %d", vm.HealthyCount)
	}

	if vm.HealthyCollapsed {
		t.Error("HealthyCollapsed must be false when no healthy group exists")
	}
}

func TestApplyCollapsePolicy_ZeroThresholdNeverCollapses(t *testing.T) {
	t.Parallel()

	vm := buildViewModel(health.Response{
		Status: health.StatusPass,
		Checks: healthyChecks(50),
	}, "Test", "/health/sse")

	applyCollapsePolicy(&vm, 0)

	if vm.HealthyCollapsed {
		t.Error("HealthyCollapsed must be false with threshold 0")
	}

	if vm.HealthyCount != 50 {
		t.Errorf("HealthyCount: want 50, got %d", vm.HealthyCount)
	}
}

func TestHealthyGroupSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		group checkGroup
		want  string
	}{
		{
			name: "all pass",
			group: checkGroup{
				Title: groupTitleHealthy,
				Rows: []checkRow{
					{Name: "db", Status: health.StatusPass},
					{Name: "cache", Status: health.StatusPass},
				},
			},
			want: "Healthy Services · 2 · all pass",
		},
		{
			name: "unknown status suppresses all-pass claim",
			group: checkGroup{
				Title: groupTitleHealthy,
				Rows: []checkRow{
					{Name: "db", Status: health.StatusPass},
					{Name: "odd", Status: health.Status("unknown")},
				},
			},
			want: "Healthy Services · 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := healthyGroupSummary(tt.group); got != tt.want {
				t.Errorf("healthyGroupSummary: want %q, got %q", tt.want, got)
			}
		})
	}
}

func TestShortDisplayName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"empty", "", ""},
		{"single word", "database", "database"},
		{"already short", "database.Service", "database.Service"},
		{"stdlib", "fmt.Stringer", "fmt.Stringer"},
		{
			"cv handler",
			"*github.com/LarsArtmann/CV/internal/features/healthdash/handlers.Handlers",
			"handlers.Handlers",
		},
		{"go-health probe", "github.com/larsartmann/go-health.Probe", "go-health.Probe"},
		{
			"aggregate",
			"github.com/larsartmann/go-health/aggregate.Aggregate",
			"aggregate.Aggregate",
		},
		{"stdlib net/http", "net/http.Client", "http.Client"},
		{"versioned module", "samber/do/v2.injector", "v2.injector"},
		{"aggregate key unchanged", "cv/database", "cv/database"},
		{"source check unchanged", "source/check", "source/check"},
		{"bare dot", ".", "."},
		{"leading dot", ".weird", ".weird"},
		{"trailing dot", "pkg.", "pkg."},
		{"public mode masked name", "check-12", "check-12"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := shortDisplayName(tt.raw); got != tt.want {
				t.Errorf("shortDisplayName(%q): want %q, got %q", tt.raw, tt.want, got)
			}
		})
	}
}

func TestGroupChecks_DerivesShortDisplayName(t *testing.T) {
	t.Parallel()

	groups := groupChecks(map[string]health.Check{
		"*github.com/x/y/handlers.Handlers": {Status: health.StatusPass},
		"database":                          {Status: health.StatusFail, Error: "down"},
	})

	var long, plain *checkRow

	for gi := range groups {
		for ri := range groups[gi].Rows {
			row := &groups[gi].Rows[ri]

			switch row.Name {
			case "*github.com/x/y/handlers.Handlers":
				long = row
			case "database":
				plain = row
			}
		}
	}

	if long == nil || long.Display != "handlers.Handlers" {
		t.Errorf("long type name should shorten to handlers.Handlers, got %+v", long)
	}

	if plain == nil || plain.Display != "database" {
		t.Errorf("short name should pass through unchanged, got %+v", plain)
	}
}

func TestFormatAge(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		at   time.Time
		want string
	}{
		{"just now", now.Add(-5 * time.Second), "just now"},
		{"future clamps", now.Add(5 * time.Minute), "just now"},
		{"seconds", now.Add(-42 * time.Second), "just now"},
		{"exactly a minute", now.Add(-61 * time.Second), "1m ago"},
		{"minutes", now.Add(-3 * time.Minute), "3m ago"},
		{"hours", now.Add(-2 * time.Hour), "2h ago"},
		{"long", now.Add(-26 * time.Hour), "26h ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := formatAge(tt.at, now); got != tt.want {
				t.Errorf("formatAge: want %q, got %q", tt.want, got)
			}
		})
	}
}
