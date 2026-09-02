package dashboard

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	health "github.com/larsartmann/go-health"
	"github.com/larsartmann/templ-components/display"
	"github.com/larsartmann/templ-components/feedback"
)

// mapStatusToBadge converts a go-health Status to the corresponding
// templ-components BadgeType.
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

// mapStatusToFeedback converts a go-health Status to the corresponding
// templ-components FeedbackType for the overall status banner.
func mapStatusToFeedback(s health.Status) feedback.FeedbackType {
	switch s {
	case health.StatusPass:
		return feedback.FeedbackSuccess
	case health.StatusWarn:
		return feedback.FeedbackWarning
	case health.StatusFail:
		return feedback.FeedbackError
	default:
		return feedback.FeedbackInfo
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
type viewModel struct {
	Title         string
	Status        health.Status
	FeedbackType  feedback.FeedbackType
	StatusText    string
	Version       string
	Uptime        string
	LatencyMs     int64
	Groups        []checkGroup
	SSEURL        string
	FaviconURL    string
	CSSPath       string
	DatastarSrc   string
	DatastarNonce string
	TailwindNonce string
	// History holds recent overall-status samples for the trend sparkline
	// (pass=1, warn=0.5, fail=0, oldest first). Nil when the trend is
	// disabled (default) or no samples recorded yet.
	History     []float64
	Timeline    []TimelineEntry
	LastUpdated string
	Description string
	// ShowStatCards renders the version/uptime/latency card grid.
	// Enabled by default; disabled via WithHideStatCards.
	ShowStatCards bool
}

// buildViewModel transforms a health.Response into a template-ready viewModel.
// Checks are sorted alphabetically by name and grouped by severity:
// failing (critical) first, then warnings (non-critical), then healthy.
func buildViewModel(resp health.Response, title, sseURL string) viewModel {
	groups := groupChecks(resp.Checks)

	feedbackType := mapStatusToFeedback(resp.Status)
	statusText := mapStatusToText(resp.Status)

	if resp.ShuttingDown {
		feedbackType = feedback.FeedbackWarning
		statusText = "Shutting Down — Draining Traffic"
	}

	return viewModel{
		LastUpdated:   time.Now().UTC().Format("15:04:05 MST"),
		Title:         title,
		Status:        resp.Status,
		FeedbackType:  feedbackType,
		StatusText:    statusText,
		Version:       resp.Version,
		Uptime:        resp.Uptime,
		LatencyMs:     resp.TotalLatencyMs,
		Groups:        groups,
		SSEURL:        sseURL,
		ShowStatCards: true,
	}
}

// Trend scale values for the sparkline.
const (
	trendPassValue = 1
	trendWarnValue = 0.5
	trendFailValue = 0
)

// statusValue maps a status to the 0..1 trend scale used by the sparkline:
// pass=1, warn=0.5, fail=0. Unknown statuses plot as fail — the trend line
// dips on anything that is not provably healthy.
func statusValue(s health.Status) float64 {
	switch s {
	case health.StatusPass:
		return trendPassValue
	case health.StatusWarn:
		return trendWarnValue
	case health.StatusFail:
		return trendFailValue
	default:
		return trendFailValue
	}
}

const (
	groupTitleFailing = "Critical Failures"
	groupTitleWarning = "Non-Critical Issues"
	groupTitleHealthy = "Healthy Services"
)

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
		case health.StatusPass:
			healthy = append(healthy, row)
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
			Title:  groupTitleFailing,
			Status: health.StatusFail,
			Rows:   failing,
		})
	}

	if len(warning) > 0 {
		groups = append(groups, checkGroup{
			Title:  groupTitleWarning,
			Status: health.StatusWarn,
			Rows:   warning,
		})
	}

	if len(healthy) > 0 {
		groups = append(groups, checkGroup{
			Title:  groupTitleHealthy,
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

// badgeForStatus creates a display.BadgeProps for the given status.
func badgeForStatus(s health.Status) display.BadgeProps {
	return display.BadgeProps{
		Text: string(s),
		Type: mapStatusToBadge(s),
	}
}

// rowsToTableRows converts check rows to templ-components TableRows with
// badge components in the status column.
func rowsToTableRows(rows []checkRow) []display.TableRow {
	tableRows := make([]display.TableRow, 0, len(rows))

	for _, row := range rows {
		errorText := row.Error
		if errorText == "" {
			errorText = "—"
		}

		tableRows = append(tableRows, display.TableRow{
			Cells: []display.TableCell{
				{Text: row.Name},
				{Content: display.Badge(badgeForStatus(row.Status))},
				{Text: errorText},
			},
		})
	}

	return tableRows
}

// fingerprintChecks creates a deterministic string fingerprint of the checks
// map for change detection. Keys are sorted to ensure the same input always
// produces the same output (Go map iteration order is randomized). Each
// field is length-prefixed so names containing delimiter characters can
// never collide with a different split across name, status, and error.
func fingerprintChecks(checks map[string]health.Check) string {
	keys := make([]string, 0, len(checks))

	for k := range checks {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var buf []byte

	for _, name := range keys {
		check := checks[name]
		buf = appendField(buf, name)
		buf = appendField(buf, string(check.Status))
		buf = appendField(buf, check.Error)
	}

	return string(buf)
}

// appendField appends "<length>:<value>;" so field boundaries are explicit
// regardless of the value's content.
func appendField(buf []byte, value string) []byte {
	buf = append(buf, strconv.Itoa(len(value))...)
	buf = append(buf, ':')
	buf = append(buf, value...)

	return append(buf, ';')
}

// TimelineEntry is one recent status flip rendered in the dashboard's
// status-change timeline.
type TimelineEntry struct {
	At       string // HH:MM:SS render timestamp
	Status   string
	Degraded bool
}

// anonymizeViewModel replaces identifying details with generic labels so
// the rendered page can be shared with untrusted audiences. Group titles,
// check names, and error messages are masked; statuses remain visible.
func anonymizeViewModel(vm *viewModel) {
	for gi := range vm.Groups {
		group := &vm.Groups[gi]

		for ri := range group.Rows {
			row := &group.Rows[ri]
			row.Name = fmt.Sprintf("check-%d", gi*100+ri+1)
			row.Error = ""
		}
	}
}
