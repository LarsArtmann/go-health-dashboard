# Decision Batch B — eight open calls (parked for the owner)

Written 2026-11-12 (plan T25). Same shape as batch A: context, options,
recommendation, blast radius. One-word replies unblock rows.

## B1. Evidence strip machine contract + persistence

**Context.** The evidence strip (proven/unproven greens) is HTML-only by
design; the JSON/webhook/metrics contracts stayed untouched. Should the
counts surface in metrics + trend/export JSON, and should the observation
window persist across restarts? The window currently resets on pusher
restart (stated in the tooltip).

**Options.**

1. **Stay HTML-only (status quo):** the strip is presentation truth;
   machine surfaces stay go-health-shaped.
2. **Metrics only:** `dashboard_evidence_proven`/`_unproven` gauges —
   operators alert on "0 proven greens" (the health-washing alarm).
   Non-identifying counts, public-mode safe.
3. **Metrics + export:** also add evidence to `/health/export` JSON
   (the dashboard-owned wire shape that may grow).

**Recommendation.** Option 2 — the health-washing alarm belongs in
monitoring; export stays per-sample. Persistence stays restart-scoped
(the log's claim is observational, and a persisted window invites
stale-unproven rot).

**Blast radius.** metrics.go, docs/metrics.md; export/JSON contracts
untouched.

## B2. Build-tag gating for SSE

**Context.** Consumers who only want the HTML page still pull go-sse (and
its `GOEXPERIMENT=jsonv2` requirement) into their module graph. See batch
A3.

**Options.**

1. **No gate (status quo):** one module, one contract.
2. **`healthdashboard_sse` build tag:** `dashboard.New` compiles without
   SSE (SSEHandler 503s, pusher stubbed) when the tag is absent.
3. **Separate `dashboardlite` module:** fork the package — rejected
   (split-brain factory).

**Recommendation.** Option 1 until a second consumer actually hurts;
option 2 is the designed escape (the A3 posture).

**Blast radius.** Build matrix in CI would double for the gated shape.

## B3. Fleet CI-lint pin + own-repo filing banner

**Context.** golangci-lint CI pin policy (manual vs dependabot vs latest):
one staleness break already cost a day of red CI. Second half: do AI-drafted
filings to OWN repos skip the "MANUALLY REVIEWED" banner?

**Options (lint pin).**

1. **Pin to minor (`v2.x`), dependabot minor PRs:** predictable; the one
   break came from a major jump.
2. **Latest always:** catches regressions early; costs surprise reds.
3. **Manual review per bump:** safest, slowest.

**Options (banner).** a) banner on ALL AI filings including own repos;
b) own-repo AI filings skip the banner (the repo owner is the reviewer by
construction).

**Recommendation.** Pin-to-minor + dependabot PRs (option 1); banner
skipped for OWN repos only (option b) — third-party filings keep it.

**Blast radius.** Fleet-wide CI workflows; the upstream-drafts doctrine.

## B4. PVR enablement (the compatibility-policy blocker)

**Context.** The v1.0 criteria sweep marks the compatibility policy MET
except PVR (the release-verification robot?) enablement, which needs an
admin act on the repo settings.

**Options.**

1. **Enable now:** the policy claims it; the checker enforces it.
2. **Defer:** the manual verify-release.sh procedure carries v1.x fine.

**Recommendation.** Option 2 until v1.0; the manual procedure has a
perfect record, and one more moving CI part before v1.0 is risk without
consumer demand.

**Blast radius.** Repo settings (admin act); release-check workflow.

## B5. Issue #14 contract wording

**Context.** #14 (push-loop lifetime) is closed, but its thread carries a
contract question: does "Shutdown stops the pusher loop" promise GOROUTINE
EXIT or BROADCASTER CLOSE? The pinned test counts `.start` frames
(issue-#14 fix), the heartbeat scan counts go-sse goroutines separately.

**Options.**

1. **Wording = goroutine exit:** matches the pinned test; broadcaster
   close is an implementation detail documented in Shutdown's doc.
2. **Wording = full resource quiescence:** stronger promise, would need
   the heartbeat scan as the pinned contract (it currently pins the
   5-heartbeat leak signature instead).

**Recommendation.** Option 1; add one clarifying sentence to Shutdown's
doc comment in the next docs-touching change.

**Blast radius.** One doc comment; no contract change.

## B6. Pin-guard keep sign-off

**Context.** `scripts/check-ui-pins.sh` maintained through twelve
dependency movements (deliberate + swept). The original removal condition
("delete when templ-components stabilizes") never fired; the guard's
value is proven (it caught ten unguarded sweeps). Formal sign-off: keep
permanently?

**Options.**

1. **Keep permanently:** the pin + audit history is the UI-change contract.
2. **Sunset condition:** delete when consumers stop drifting (measurable
   via the MVS fleet-watch, B7).

**Recommendation.** Option 1 — the twelfth movement was a DELIBERATE
adoption and the guard still carried it; sunset conditions rot.

**Blast radius.** None (status quo confirmation).

## B7. Consumer-side MVS drift watch

**Context.** 8 consumer graphs resolve templ-components ≥ this repo's
pin; CI tested v1.24.0 while consumers RAN v1.26.0 (the amber-rename
incident). The guard sees only OUR go.mod.

**Options.**

1. **tc doctor in consumer CI (fleet ask):** each consumer runs
   `tc doctor` against their compiled CSS.
2. **Central watch:** this repo's CI checks sibling go.mods via the meta
   inventory.
3. **Adoption-on-drift doctrine (status quo):** every observed drift got
   adopted within a day; the hazard is consumers' STALE artifacts
   (health.css, library-classes.txt), which the remediation path
   (README's two failure signatures + tc doctor) covers.

**Recommendation.** Option 3 now; file option 1 as a fleet-level ask in
each consumer's backlog rather than building central infra here.

**Blast radius.** Consumer repos (their CI), fleet coordination.

## B8. AGENTS.md size

**Context.** 537+ lines vs the fleet 220-line budget; the preflight warns
every run. T30 produced the restructure prep (what moves to docs/); the
decision is accept vs restructure.

**Options.**

1. **Restructure per the T30 outline:** dependency-note long-form +
   gotcha detail move to docs/, AGENTS keeps the operative rules.
   Target ≈ 260 lines. One focused session, mechanical, goldens-free.
2. **Accept the overage formally:** document WHY (the module's doctrine
   density beats the budget) and silence the warn in buildflow config.

**Recommendation.** Option 1 — the outline exists, the split is
mechanical, and a lean AGENTS.md is what a fresh session can actually
load. Accept the residual warn if the target misses 220 (260 is honest).

**Blast radius.** AGENTS.md + docs/ moves; session-start context loads.
