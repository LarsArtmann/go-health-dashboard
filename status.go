package dashboard

import (
	"encoding/json/v2"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

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

// checkRow is a single service row in the dashboard table. Name is the raw
// check key (often a fully-qualified Go type name); Display is the shortened
// form shown in the table (see shortDisplayName).
type checkRow struct {
	Name    string
	Display string
	Status  health.Status
	Error   string
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
	History  []float64
	Timeline []TimelineEntry
	// LastUpdated is the "Updated <time>" stamp. With trend history enabled
	// it is the observation time of the most recent sample (when the health
	// state was actually seen); without it, the render time.
	LastUpdated string
	// LastUpdatedTime is the machine timestamp behind LastUpdated, used to
	// render the coarse "2m ago" age next to the absolute stamp.
	LastUpdatedTime time.Time
	// ExportURL, TrendURL, and MetricsURL are the non-empty endpoints
	// surfaced in the header links row: export/history JSON, trend JSON, and
	// Prometheus metrics. Empty means the endpoint is disabled.
	ExportURL   string
	TrendURL    string
	MetricsURL  string
	Description string
	// ShowStatCards renders the version/uptime/latency card grid.
	// Enabled by default; disabled via WithHideStatCards.
	ShowStatCards bool
	// HealthyCount is the number of rows in the healthy group (0 when no
	// healthy group exists). Rendered in the group's summary line.
	HealthyCount int
	// HealthyCollapsed collapses the healthy group behind a native
	// <details> element on initial render. Derived from HealthyCount and
	// the configured collapse threshold by applyCollapsePolicy; both render
	// paths (initial HTML and SSE patches) re-derive it, so a patch always
	// restores the default collapse state.
	HealthyCollapsed bool
	// PersistCollapse enables the localStorage persistence script for the
	// healthy group's open/closed state (WithPersistCollapse).
	PersistCollapse bool
}

// updatedStampFormat is the wall-clock format of the viewModel LastUpdated
// stamp. UTC is used so the stamp is stable across server timezones.
const updatedStampFormat = "15:04:05 MST"

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
		LastUpdated:     time.Now().UTC().Format(updatedStampFormat),
		LastUpdatedTime: time.Now().UTC(),
		Title:           title,
		Status:          resp.Status,
		FeedbackType:    feedbackType,
		StatusText:      statusText,
		Version:         resp.Version,
		Uptime:          resp.Uptime,
		LatencyMs:       resp.TotalLatencyMs,
		Groups:          groups,
		SSEURL:          sseURL,
		ShowStatCards:   true,
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
		row := checkRow{
			Name:    name,
			Display: shortDisplayName(name),
			Status:  check.Status,
			Error:   check.Error,
		}

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

// shortDisplayName condenses a fully-qualified Go type name for table
// display. The module path is the noise: a service registered as
//
//	*github.com/host/repo/internal/features/healthdash/handlers.Handlers
//
// reads as "handlers.Handlers", and "github.com/larsartmann/go-health.Probe"
// as "go-health.Probe". Names that are already short ("database.Service"),
// single words, and aggregate source/check keys pass through unchanged.
// The raw name stays available on the row (title attribute and details
// cell), so shortening is presentation-only and lossless.
func shortDisplayName(raw string) string {
	name := strings.TrimPrefix(raw, "*")

	dot := strings.LastIndex(name, ".")
	if dot <= 0 || dot == len(name)-1 {
		return raw
	}

	pkgPath, typeName := name[:dot], name[dot+1:]

	if slash := strings.LastIndex(pkgPath, "/"); slash >= 0 {
		pkgPath = pkgPath[slash+1:]
	}

	return pkgPath + "." + typeName
}

// hasShortDisplay reports whether the row's raw key was shortened for
// display — when false, the details column need not repeat it.
func (row checkRow) hasShortDisplay() bool {
	return row.Display != "" && row.Display != row.Name
}

// hasProblems reports whether any check group is failing or warning — used
// to decide whether the "jump to problems" anchor link renders.
func hasProblems(vm viewModel) bool {
	for _, group := range vm.Groups {
		if group.Status != health.StatusPass {
			return true
		}
	}

	return false
}

// filterHaystack returns the lowercase search text a row is matched against
// by the client-side filter: short display name plus raw check key.
func filterHaystack(row checkRow) string {
	return strings.ToLower(row.displayName() + " " + row.Name)
}

// jsStringLiteral encodes s as a double-quoted JavaScript string literal.
// JSON strings are valid ES2019+ literals (the JSON-superset proposal), so
// encoding/json's output is used directly — it handles quotes, control
// characters, and non-ASCII safely. Marshal cannot fail for strings; the
// fallback exists for exhaustiveness.
func jsStringLiteral(s string) string {
	encoded, err := json.Marshal(s)
	if err != nil {
		return strconv.Quote(s)
	}

	return string(encoded)
}

// filterEmptyExpr builds the data-class-hidden expression for the
// "no services match" hint: visible only while a query is active and no
// row anywhere on the page matches it.
func filterEmptyExpr(vm viewModel) string {
	all := &strings.Builder{}

	for _, group := range vm.Groups {
		for _, row := range group.Rows {
			all.WriteString(filterHaystack(row))
			all.WriteString("\n")
		}
	}

	return "$query !== '' || !" + jsStringLiteral(all.String()) + ".includes($query)"
}

// errorSummaryMax bounds the error text shown before the details expansion.
const errorSummaryMax = 80

// truncateError shortens long error text for the collapsed summary,
// rune-safe so multi-byte characters are never split.
func truncateError(s string) string {
	if utf8.RuneCountInString(s) <= errorSummaryMax {
		return s
	}

	runes := []rune(s)

	return string(runes[:errorSummaryMax]) + "…"
}

// formatAge renders a coarse human age for the Updated stamp: "just now",
// "42s ago", "3m ago", or "2h ago". Future timestamps clamp to "just now".
func formatAge(observation, now time.Time) string {
	d := now.Sub(observation)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
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

// displayName returns the shortened display name, falling back to the raw
// name when no short form was derived.
func (row checkRow) displayName() string {
	if row.Display != "" {
		return row.Display
	}

	return row.Name
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
			row.Display = row.Name
			row.Error = ""
		}
	}
}

// applyCollapsePolicy derives the healthy-group collapse state from the view
// model's groups and the configured threshold: the group collapses when at
// least threshold rows are healthy. A threshold of zero or less never
// collapses. Both render paths (initial HTML and SSE patches) call this, so
// a patch re-applies the default collapse state by design.
func applyCollapsePolicy(vm *viewModel, threshold int) {
	for _, group := range vm.Groups {
		if group.Status == health.StatusPass {
			vm.HealthyCount = len(group.Rows)

			break
		}
	}

	vm.HealthyCollapsed = threshold > 0 && vm.HealthyCount >= threshold
}

// healthyGroupSummary builds the healthy group's summary line, e.g.
// "Healthy Services \u00b7 57 \u00b7 all pass". The "all pass" suffix is appended only
// when every row in the group is a pass: unknown statuses are grouped as
// healthy too and must not be claimed as passing.
func healthyGroupSummary(group checkGroup) string {
	summary := fmt.Sprintf("%s \u00b7 %d", group.Title, len(group.Rows))

	for _, row := range group.Rows {
		if row.Status != health.StatusPass {
			return summary
		}
	}

	return summary + " \u00b7 all pass"
}
