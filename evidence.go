package dashboard

import (
	"fmt"
	"sync"
	"time"

	health "github.com/larsartmann/go-health"
)

// The dashboard's value equals the fraction of its checks that can actually
// fail (the health-washing lesson, samber-linter README §1): a sweep row
// that has never deviated from pass may verify nothing, yet it renders the
// same green badge as a check that just survived a real failure. The
// evidence log closes the observability half of that gap. It cannot know
// WHY a check passes — go-health's Check is only {Status, Error} — so it
// records what is honestly observable: whether each check has ever reported
// a non-pass status under this pusher, and when that last happened. The UI
// then distinguishes proven greens from unproven ones instead of claiming
// verification it does not have.

// maxEvidenceChecks bounds the evidence map. Check names come from the
// probe's response (consumer-controlled, UTF-8-sanitized); the cap keeps a
// pathologically churning key space from growing memory without bound.
// Past the cap, new names stop accruing evidence and honestly report
// unproven; existing entries keep updating.
const maxEvidenceChecks = 10000

// checkEvidence accumulates one check's non-pass observations.
type checkEvidence struct {
	NonPassCount int
	LastNonPass  time.Time
}

// evidenceLog tracks, per check name, the non-pass observations seen by the
// pusher. One log lives per pusher: it starts when Start launches the push
// loop and resets on restart, which is exactly the honest observation
// window ("since monitoring started").
type evidenceLog struct {
	mu        sync.Mutex
	startedAt time.Time
	checks    map[string]*checkEvidence
}

func newEvidenceLog() *evidenceLog {
	return &evidenceLog{
		startedAt: time.Now(),
		checks:    make(map[string]*checkEvidence),
	}
}

// observe records one tick's checks. Pass results add nothing — only a
// non-pass observation proves a check can deviate from green (the
// healthaudit "errored is the only proof" semantics, generalized to warn,
// which equally proves real logic ran).
func (e *evidenceLog) observe(resp health.Response, at time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for name, check := range resp.Checks {
		if check.Status == health.StatusPass {
			continue
		}

		ev := e.checks[name]
		if ev == nil {
			if len(e.checks) >= maxEvidenceChecks {
				continue
			}

			ev = &checkEvidence{}
			e.checks[name] = ev
		}

		ev.NonPassCount++
		ev.LastNonPass = at
	}
}

// evidenceSummary is the template-ready snapshot of the log. Total and
// Proven are computed against the CURRENT response's check set, so a check
// that proved itself and then disappeared cannot inflate the ratio.
type evidenceSummary struct {
	// Since is the UTC observation-window start (pusher creation).
	Since time.Time
	// Total is the number of checks in the current response.
	Total int
	// Proven is how many of those checks have a recorded non-pass.
	Proven int

	lastNonPassBy map[string]time.Time
}

// everNonPass reports whether the named check has ever been observed in a
// non-pass status during the window.
func (s evidenceSummary) everNonPass(name string) bool {
	return !s.lastNonPassBy[name].IsZero()
}

// lastNonPassOf returns the most recent non-pass observation time for the
// named check, or the zero time when it has never deviated.
func (s evidenceSummary) lastNonPassOf(name string) time.Time {
	return s.lastNonPassBy[name]
}

// snapshot copies the log into a summary. Total and Proven are filled by
// populateEvidence, which knows the current response's check set.
func (e *evidenceLog) snapshot() evidenceSummary {
	e.mu.Lock()
	defer e.mu.Unlock()

	byName := make(map[string]time.Time, len(e.checks))
	for name, ev := range e.checks {
		if ev.NonPassCount > 0 {
			byName[name] = ev.LastNonPass
		}
	}

	return evidenceSummary{
		Since:         e.startedAt,
		lastNonPassBy: byName,
	}
}

// populateEvidence annotates the view model with observational evidence:
// the summary counts (against the current response's checks) feed the truth
// strip, and badgeForStatus reads the same summary for per-row tooltips.
// Both render paths (initial HTML and SSE patches) call this so they always
// agree. A nil log (pusher not started) leaves the viewModel untouched and
// the strip unrendered — no observations, no evidence claims.
func populateEvidence(vm *viewModel, log *evidenceLog, resp health.Response) {
	if log == nil {
		return
	}

	sum := log.snapshot()
	sum.Total = len(resp.Checks)

	for name := range resp.Checks {
		if sum.everNonPass(name) {
			sum.Proven++
		}
	}

	vm.Evidence = sum
}

// evidenceSummaryText renders the truth strip's one-line verdict. The three
// cases are deliberately distinct: zero proven checks is the health-washing
// warning (all-green may mean "cannot fail"), a split names both classes,
// and a full house is the only situation where "all pass" reads as
// well-evidenced.
func evidenceSummaryText(s evidenceSummary) string {
	if s.Total == 0 {
		return ""
	}

	since := s.Since.UTC().Format(updatedStampFormat)

	switch {
	case s.Proven == 0:
		return fmt.Sprintf(
			"Failure evidence: 0 of %d checks has ever deviated from pass since %s — green rows are unproven, not verified; they may be unable to fail.",
			s.Total, since)
	case s.Proven == s.Total:
		return fmt.Sprintf(
			"Failure evidence: all %d checks have deviated from pass at least once since %s.",
			s.Total, since)
	default:
		return fmt.Sprintf(
			"Failure evidence: %d of %d checks have deviated from pass at least once since %s; the other %d green rows are unproven.",
			s.Proven, s.Total, since, s.Total-s.Proven)
	}
}

// evidenceTooltip is the native hover explanation behind the truth strip:
// what a pass badge does and does not mean, and why the window resets.
const evidenceTooltip = "A pass badge means only that no failure was observed. Checks that have never reported a non-pass status may be unable to fail (for example, registered without a real health check). Evidence accrues while the dashboard pushes and resets when it restarts."

// badgeEvidenceTitle builds the per-row tooltip for a pass badge: unproven
// greens disclose their lack of evidence; proven greens cite the last
// observed non-pass, turning a green row into backed claims. Non-pass rows
// need no tooltip — the badge itself is the deviation.
func badgeEvidenceTitle(row checkRow, s evidenceSummary) string {
	if row.Status != health.StatusPass || s.Since.IsZero() {
		return ""
	}

	if !s.everNonPass(row.Name) {
		return fmt.Sprintf(
			"pass — never deviated from pass since %s; this green is unproven (the check may be unable to fail)",
			s.Since.UTC().Format(updatedStampFormat))
	}

	return fmt.Sprintf(
		"pass — last non-pass observed at %s; this green is backed by a check that has demonstrably deviated",
		s.lastNonPassOf(row.Name).UTC().Format(updatedStampFormat))
}
