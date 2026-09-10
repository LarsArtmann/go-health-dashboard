package dashboard

import (
	"strings"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	"github.com/a-h/templ"
	"github.com/larsartmann/templ-components/display"
)

// respWithChecks builds a response whose checks all carry the given status.
func respWithChecks(statuses map[string]health.Status) health.Response {
	checks := make(map[string]health.Check, len(statuses))
	for name, status := range statuses {
		checks[name] = health.Check{Status: status}
	}

	return health.Response{Status: worstOf(statuses), Checks: checks}
}

// worstOf rolls per-check statuses up the way go-health does: fail beats
// warn beats pass.
func worstOf(statuses map[string]health.Status) health.Status {
	status := health.StatusPass

	for _, s := range statuses {
		switch s {
		case health.StatusFail:
			return health.StatusFail
		case health.StatusWarn:
			status = health.StatusWarn
		}
	}

	return status
}

func TestEvidenceLog_ObservesNonPass(t *testing.T) {
	t.Parallel()

	log := newEvidenceLog()
	at := time.Now()

	pass := health.Response{Status: health.StatusPass, Checks: map[string]health.Check{
		"db": {Status: health.StatusPass},
	}}

	log.observe(pass, at)

	if got := log.snapshot(); len(got.lastNonPassBy) != 0 {
		t.Fatalf("pass-only tick recorded evidence: %v", got.lastNonPassBy)
	}

	degraded := health.Response{Status: health.StatusWarn, Checks: map[string]health.Check{
		"db":    {Status: health.StatusPass},
		"cache": {Status: health.StatusWarn, Error: "slow"},
		"queue": {Status: health.StatusFail, Error: "timeout"},
	}}

	log.observe(degraded, at)
	log.observe(degraded, at.Add(time.Second))

	sum := log.snapshot()

	if !sum.everNonPass("cache") || !sum.everNonPass("queue") {
		t.Fatalf("warn/fail checks not proven: %v", sum.lastNonPassBy)
	}

	if sum.everNonPass("db") {
		t.Error("pass-only check must stay unproven")
	}

	if got := sum.lastNonPassOf("cache"); !got.Equal(at.Add(time.Second)) {
		t.Errorf("LastNonPass = %v, want %v (latest observation wins)", got, at.Add(time.Second))
	}

	// Recovery keeps the evidence: a green that has deviated before stays
	// proven — that is the entire point.
	log.observe(pass, at.Add(2*time.Second))

	if sum2 := log.snapshot(); !sum2.everNonPass("cache") {
		t.Error("recovered check lost its proven status")
	}
}

func TestEvidenceLog_Cap(t *testing.T) {
	t.Parallel()

	log := newEvidenceLog()

	for i := range maxEvidenceChecks + 10 {
		log.observe(health.Response{Checks: map[string]health.Check{
			intToName(i): {Status: health.StatusFail},
		}}, time.Now())
	}

	if got := len(log.checks); got != maxEvidenceChecks {
		t.Fatalf("evidence map grew to %d entries, want capped at %d", got, maxEvidenceChecks)
	}
}

// intToName renders i as a distinct check name.
func intToName(i int) string {
	return "check-" + strings.Repeat("n", i%13) + "-" + time.Unix(int64(i), 0).Format("0102150405")
}

func TestPopulateEvidence_CountsOnlyCurrentChecks(t *testing.T) {
	t.Parallel()

	log := newEvidenceLog()
	at := time.Now()

	log.observe(health.Response{Checks: map[string]health.Check{
		"db":      {Status: health.StatusFail},
		"removed": {Status: health.StatusFail},
	}}, at)

	// "removed" disappeared from the response; its proof must not inflate
	// the ratio against the current check set.
	resp := respWithChecks(map[string]health.Status{
		"db":    health.StatusPass,
		"fresh": health.StatusPass,
	})

	vm := buildViewModel(resp, "", "", GroupBySeverity)
	populateEvidence(&vm, log, resp)

	if vm.Evidence.Total != 2 {
		t.Errorf("Total = %d, want 2 (current checks)", vm.Evidence.Total)
	}

	if vm.Evidence.Proven != 1 {
		t.Errorf("Proven = %d, want 1 (db proven, removed check not counted, fresh unproven)", vm.Evidence.Proven)
	}
}

func TestPopulateEvidence_NilLog(t *testing.T) {
	t.Parallel()

	vm := buildViewModel(respWithChecks(map[string]health.Status{"db": health.StatusPass}), "", "", GroupBySeverity)
	populateEvidence(&vm, nil, respWithChecks(map[string]health.Status{"db": health.StatusPass}))

	if !vm.Evidence.Since.IsZero() {
		t.Error("nil log must leave Evidence zero-valued so the strip stays unrendered")
	}
}

func TestBadgeEvidenceTitle(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)

	sum := evidenceSummary{
		Since:  at,
		Total:  2,
		Proven: 1,
		lastNonPassBy: map[string]time.Time{
			"db": at,
		},
	}

	tests := []struct {
		name string
		row  checkRow
		want string
		zero bool
	}{
		{
			name: "unproven pass discloses",
			row:  checkRow{Name: "cache", Status: health.StatusPass},
			want: "unproven",
		},
		{
			name: "proven pass cites last non-pass",
			row:  checkRow{Name: "db", Status: health.StatusPass},
			want: "last non-pass observed at 12:00:00 UTC",
		},
		{
			name: "non-pass needs no tooltip",
			row:  checkRow{Name: "db", Status: health.StatusFail},
			want: "",
		},
		{
			name: "zero observation window stays silent",
			row:  checkRow{Name: "cache", Status: health.StatusPass},
			want: "",
			zero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			summary := sum
			if tt.zero {
				summary = evidenceSummary{}
			}

			got := badgeEvidenceTitle(tt.row, summary)

			if tt.want == "" {
				if got != "" {
					t.Fatalf("title = %q, want empty", got)
				}

				return
			}

			if !strings.Contains(got, tt.want) {
				t.Fatalf("title = %q, want it to contain %q", got, tt.want)
			}
		})
	}
}

func TestBadgeForStatus_CarriesTitle(t *testing.T) {
	t.Parallel()

	at := time.Now()

	sum := evidenceSummary{
		Since:         at,
		lastNonPassBy: map[string]time.Time{"db": at},
	}

	props := badgeForStatus(health.StatusPass, sum, "cache")

	title, ok := props.BaseProps.Attrs["title"].(string)
	if !ok || !strings.Contains(title, "unproven") {
		t.Fatalf("badge title = %v, want an unproven disclosure", props.BaseProps.Attrs["title"])
	}

	if props.Text != string(health.StatusPass) || props.Type != display.BadgeSuccess {
		t.Errorf("badge = %q/%v, want pass/Success", props.Text, props.Type)
	}
}

func TestEvidenceSummaryText(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		summary  evidenceSummary
		contains string
	}{
		{
			name:     "no checks renders empty",
			summary:  evidenceSummary{Since: since},
			contains: "",
		},
		{
			name: "zero proven is the health-washing warning",
			summary: evidenceSummary{
				Since: since, Total: 60,
			},
			contains: "0 of 60 checks has ever deviated from pass",
		},
		{
			name: "split names the unproven remainder",
			summary: evidenceSummary{
				Since: since, Total: 60, Proven: 6,
			},
			contains: "6 of 60 checks have deviated from pass",
		},
		{
			name: "full house",
			summary: evidenceSummary{
				Since: since, Total: 60, Proven: 60,
			},
			contains: "all 60 checks have deviated from pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := evidenceSummaryText(tt.summary)

			if tt.contains == "" {
				if got != "" {
					t.Fatalf("text = %q, want empty", got)
				}

				return
			}

			if !strings.Contains(got, tt.contains) {
				t.Fatalf("text = %q, want it to contain %q", got, tt.contains)
			}

			if !strings.Contains(got, "08:00:00 UTC") {
				t.Errorf("text = %q, want the UTC observation-window start", got)
			}
		})
	}
}

func TestRender_EvidenceStrip(t *testing.T) {
	t.Parallel()

	at := time.Now()

	provenVM := buildViewModel(respWithChecks(map[string]health.Status{
		"db":    health.StatusPass,
		"cache": health.StatusWarn,
	}), "Test", "/health/sse", GroupBySeverity)

	provenVM.Evidence = evidenceSummary{
		Since: at, Total: 2, Proven: 1,
		lastNonPassBy: map[string]time.Time{"cache": at},
	}

	html := renderToString(t, dashboardContent(provenVM))

	if !strings.Contains(html, "Failure evidence: 1 of 2 checks") {
		t.Errorf("rendered HTML missing truth strip:\n%s", html)
	}

	if !strings.Contains(html, evidenceTooltip) {
		t.Error("truth strip missing the native tooltip")
	}

	zeroVM := buildViewModel(respWithChecks(map[string]health.Status{"db": health.StatusPass}), "Test", "/health/sse", GroupBySeverity)
	zeroHTML := renderToString(t, dashboardContent(zeroVM))

	if strings.Contains(zeroHTML, "Failure evidence") {
		t.Errorf("unpopulated viewModel must not render the strip:\n%s", zeroHTML)
	}
}

// renderToString renders a templ component to a string, failing the test on
// error.
func renderToString(t *testing.T, c templ.Component) string {
	t.Helper()

	sb := &strings.Builder{}

	if err := c.Render(t.Context(), sb); err != nil {
		t.Fatalf("render: %v", err)
	}

	return sb.String()
}
