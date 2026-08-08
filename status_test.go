package dashboard

import (
	"testing"

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

func TestMapStatusToAlert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status health.Status
		want   feedback.AlertType
	}{
		{"pass", health.StatusPass, feedback.AlertSuccess},
		{"warn", health.StatusWarn, feedback.AlertWarning},
		{"fail", health.StatusFail, feedback.AlertError},
		{"unknown", health.Status("unknown"), feedback.AlertInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := mapStatusToAlert(tt.status); got != tt.want {
				t.Errorf("mapStatusToAlert(%s): want %s, got %s", tt.status, tt.want, got)
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
		"database":  {Status: health.StatusFail, Error: "connection refused"},
		"redis":     {Status: health.StatusWarn, Error: "high latency"},
		"api":       {Status: health.StatusPass},
		"cache":     {Status: health.StatusPass},
		"queue":     {Status: health.StatusFail, Error: "timeout"},
		"exporter":  {Status: health.StatusWarn, Error: "slow"},
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
		"zebra":  {Status: health.StatusFail},
		"alpha":  {Status: health.StatusFail},
		"mongo":  {Status: health.StatusFail},
	}

	groups := groupChecks(checks)

	if len(groups) != 1 {
		t.Fatalf("want 1 group, got %d", len(groups))
	}

	names := make([]string, len(groups[0].Rows))
	for i, row := range groups[0].Rows {
		names[i] = row.Name
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
		"db":   {Status: health.StatusPass},
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
		Status:        health.StatusWarn,
		Version:       "1.2.3",
		Uptime:        "5m0s",
		TotalLatencyMs: 42,
		Checks: map[string]health.Check{
			"db":      {Status: health.StatusPass},
			"cache":   {Status: health.StatusWarn, Error: "slow"},
		},
	}

	vm := buildViewModel(resp, "My Service", "/health/partial", "2s")

	if vm.Title != "My Service" {
		t.Errorf("title: want 'My Service', got %q", vm.Title)
	}

	if vm.Status != health.StatusWarn {
		t.Errorf("status: want warn, got %s", vm.Status)
	}

	if vm.AlertType != feedback.AlertWarning {
		t.Errorf("alert type: want warning, got %s", vm.AlertType)
	}

	if vm.Version != "1.2.3" {
		t.Errorf("version: want '1.2.3', got %q", vm.Version)
	}

	if vm.LatencyMs != 42 {
		t.Errorf("latency: want 42, got %d", vm.LatencyMs)
	}

	if vm.PartialURL != "/health/partial" {
		t.Errorf("partial URL: want '/health/partial', got %q", vm.PartialURL)
	}

	if len(vm.Groups) != 2 {
		t.Fatalf("want 2 groups (warning + healthy), got %d", len(vm.Groups))
	}
}

func TestRowsToTableRows_BuildsCellsWithBadges(t *testing.T) {
	t.Parallel()

	rows := []checkRow{
		{Name: "db", Status: health.StatusPass},
		{Name: "cache", Status: health.StatusFail, Error: "connection refused"},
	}

	tableRows := rowsToTableRows(rows)

	if len(tableRows) != 2 {
		t.Fatalf("want 2 table rows, got %d", len(tableRows))
	}

	for _, row := range tableRows {
		if len(row.Cells) != 3 {
			t.Errorf("want 3 cells (name, status, error), got %d", len(row.Cells))
		}
	}

	if tableRows[0].Cells[1].Content == nil {
		t.Error("status cell should have a badge component, got nil")
	}

	if tableRows[0].Cells[2].Text != "—" {
		t.Errorf("passing check error: want '—', got %q", tableRows[0].Cells[2].Text)
	}

	if tableRows[1].Cells[2].Text != "connection refused" {
		t.Errorf("failing check error: want 'connection refused', got %q",
			tableRows[1].Cells[2].Text)
	}
}
