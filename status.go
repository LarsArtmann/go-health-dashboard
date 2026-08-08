package dashboard

import (
	"fmt"
	"sort"

	health "github.com/larsartmann/go-health"
	"github.com/larsartmann/templ-components/display"
	"github.com/larsartmann/templ-components/feedback"
)

// mapStatusToBadge converts a go-health Status to the corresponding
// templ-components BadgeType. We map directly to constants rather than relying
// on StatusBadge's string map (which recognises "healthy"/"degraded"/
// "unhealthy" but not "pass"/"warn"/"fail").
func mapStatusToBadge(s health.Status) display.BadgeType {
	switch s {
	case health.StatusPass:
		return display.BadgeSuccess
	case health.StatusWarn:
		return display.BadgeWarning
	case health.StatusFail:
		return display.BadgeError
	default:
		return display.BadgeNeutral
	}
}

// mapStatusToAlert converts a go-health Status to the corresponding
// templ-components AlertType for the overall status banner.
func mapStatusToAlert(s health.Status) feedback.AlertType {
	switch s {
	case health.StatusPass:
		return feedback.AlertSuccess
	case health.StatusWarn:
		return feedback.AlertWarning
	case health.StatusFail:
		return feedback.AlertError
	default:
		return feedback.AlertInfo
	}
}

// mapStatusToText returns human-readable display text for each status.
func mapStatusToText(s health.Status) string {
	switch s {
	case health.StatusPass:
		return "All Systems Operational"
	case health.StatusWarn:
		return "Degraded — Non-Critical Issues"
	case health.StatusFail:
		return "Unhealthy — Critical Failures"
	default:
		return fmt.Sprintf("Unknown status: %s", s)
	}
}

// checkRow is a single service row in the dashboard table.
type checkRow struct {
	Name   string
	Status health.Status
	Error  string
}

// checkGroup groups checks by severity for card-based layout.
type checkGroup struct {
	Title  string
	Status health.Status
	Rows   []checkRow
}

// viewModel is the template-ready representation of a health.Response.
// All component construction (badges, alerts) is done during buildViewModel
// so the templ templates only iterate and render.
type viewModel struct {
	Title      string
	Status     health.Status
	AlertType  feedback.AlertType
	StatusText string
	Version    string
	Uptime     string
	LatencyMs  int64
	Groups     []checkGroup
	PartialURL string
	Every      string
}

// buildViewModel transforms a health.Response into a template-ready viewModel.
// Checks are sorted alphabetically by name and grouped by severity:
// failing (critical) first, then warnings (non-critical), then healthy.
func buildViewModel(resp health.Response, title, partialURL, every string) viewModel {
	groups := groupChecks(resp.Checks)

	alertType := mapStatusToAlert(resp.Status)
	statusText := mapStatusToText(resp.Status)

	if resp.ShuttingDown {
		alertType = feedback.AlertWarning
		statusText = "Shutting Down — Draining Traffic"
	}

	return viewModel{
		Title:      title,
		Status:     resp.Status,
		AlertType:  alertType,
		StatusText: statusText,
		Version:    resp.Version,
		Uptime:     resp.Uptime,
		LatencyMs:  resp.TotalLatencyMs,
		Groups:     groups,
		PartialURL: partialURL,
		Every:      every,
	}
}

// groupChecks partitions checks into severity-ordered groups: failing,
// warning, and healthy. Each group is sorted alphabetically by name.
// Empty groups are omitted.
func groupChecks(checks map[string]health.Check) []checkGroup {
	var failing, warning, healthy []checkRow

	for name, check := range checks {
		row := checkRow{Name: name, Status: check.Status, Error: check.Error}

		switch check.Status {
		case health.StatusFail:
			failing = append(failing, row)
		case health.StatusWarn:
			warning = append(warning, row)
		default:
			healthy = append(healthy, row)
		}
	}

	sortByName(failing)
	sortByName(warning)
	sortByName(healthy)

	var groups []checkGroup

	if len(failing) > 0 {
		groups = append(groups, checkGroup{
			Title:  "Critical Failures",
			Status: health.StatusFail,
			Rows:   failing,
		})
	}

	if len(warning) > 0 {
		groups = append(groups, checkGroup{
			Title:  "Non-Critical Issues",
			Status: health.StatusWarn,
			Rows:   warning,
		})
	}

	if len(healthy) > 0 {
		groups = append(groups, checkGroup{
			Title:  "Healthy Services",
			Status: health.StatusPass,
			Rows:   healthy,
		})
	}

	return groups
}

// sortByName sorts check rows alphabetically by service name.
func sortByName(rows []checkRow) {
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Name < rows[j].Name
	})
}

// badgeForStatus creates a display.Badge component for the given status.
func badgeForStatus(s health.Status) display.BadgeProps {
	return display.BadgeProps{
		Text: string(s),
		Type: mapStatusToBadge(s),
	}
}

// rowsToTableRows converts check rows to templ-components TableRows with
// badge components in the status column.
func rowsToTableRows(rows []checkRow) []display.TableRow {
	tableRows := make([]display.TableRow, len(rows))

	for i, row := range rows {
		errorText := row.Error
		if errorText == "" {
			errorText = "—"
		}

		tableRows[i] = display.TableRow{
			Cells: []display.TableCell{
				{Text: row.Name},
				{Content: display.Badge(badgeForStatus(row.Status))},
				{Text: errorText},
			},
		}
	}

	return tableRows
}
