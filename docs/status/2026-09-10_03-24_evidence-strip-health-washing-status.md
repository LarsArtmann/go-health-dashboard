# Status Report — Evidence Strip (Health-Washing Truth UI)

**Date**: 2026-09-10 03:24 CEST
**Session scope**: "What can we learn from `~/projects/samber-linter` so our UI/UX better represents the Truth to end users?"
**Repo**: `github.com/larsartmann/go-health-dashboard` @ `6b1bea9` (master)
**Verdict**: Shipped. Feature complete, all gates green, docs updated. Known gaps are deliberate scope cuts or small test additions, listed below.

---

## Executive Summary

`/home/lars/projects/samber-linter` taught one thesis: **a health dashboard's value equals the fraction of its checks that can actually fail** (README §1, the CV incident: 60 green rows, ~6 fail-capable). samber/do's sweep returns identical `nil` for "checked and passed" and "not checkable", so no renderer can distinguish them from a single response — and this dashboard rendered all greens identically, amplifying the lie.

What IS honestly observable at runtime is whether a check has **ever deviated from pass** (the `healthaudit` package's "errored is the only runtime proof" semantics). The dashboard now records that per pusher tick and separates **proven** greens from **unproven** greens:

- A **truth strip** under the Updated stamp: "Failure evidence: 6 of 60 checks have deviated from pass since 14:02:05 MST; the other 54 green rows are unproven." Zero-proven renders the explicit health-washing warning; all-proven renders the full-house line.
- **Per-badge tooltips**: every `pass` badge discloses its evidence basis — unproven greens say "may be unable to fail", proven greens cite their last observed non-pass.

The log claims only what it can know (`health.Check` is `{Status, Error}` — no "checked vs skipped" sentinel exists upstream). Honesty guards: observation window = pusher lifetime (resets on restart, stated in the tooltip); zero observations render nothing; proven counts intersect the current check set; 10k-name cap.

**All quality gates green at session end**: full test suite `ok`, lint exit 0, vet exit 0, `go test -race` on evidence paths `ok`, headless-Chrome browser suite (strict CSP + axe audit) `ok`, golden renders regenerated + diff-reviewed, generate→fmt stability loop confirmed.

---

## a) FULLY DONE

| Item | Evidence |
|---|---|
| **Research**: samber-linter README (§1–§11), AGENTS.md, `pkg/healthaudit/healthaudit.go` read in full; no UI recommendations existed anywhere in that repo (sub-agent sweep of docs/, TODO_LIST.md, FEATURES.md) — the display gap was mine to design | Report §Executive Summary; sub-agent search results |
| **Gap analysis**: go-health `Check` = `{Status, Error}` (types.go:19–24); dashboard rendered all greens identically; history tracked overall status only | Read of `~/projects/go-health/types.go`, `status.go`, `history.go` |
| **`evidence.go`** (new): `evidenceLog` (mutex-guarded per-check non-pass observations), `evidenceSummary`, `populateEvidence` (nil-log guard, intersection with current checks), `evidenceSummaryText` (0/split/all-house verdicts), `evidenceTooltip`, `badgeEvidenceTitle` (zero-`Since` silence guard) | Passing unit tests in `evidence_test.go`; lint exit 0 |
| **Pusher wiring**: `evidence *evidenceLog` on pusher, created in `newPusher`, `observe(resp, now)` on every tick **before** change detection (evidence accrues in PushOnChange mode, same discipline as the trend ring), `populateEvidence` in `renderPatch` so SSE initial patches carry the strip | `pusher.go:52,80,116–118,174`; `TestIntegration_EvidenceStripReflectsObservations` passes |
| **`buildData` wiring**: initial HTML render populates `viewModel.Evidence` via the started pusher | `handlers.go:169–171` |
| **view.templ**: truth strip (renders only when `!Since.IsZero() && Total > 0`, native `title` tooltip, CSP-clean classes only) + badge tooltips; `evidenceSummary` threaded explicitly through `groupTable`/`groupRows` (templ has no closure scope — first build failed with `undefined: data`, fixed by explicit param threading) | `TestRender_EvidenceStrip` passes; browser suite CSP-clean |
| **`viewModel.Evidence` field** added with doc comment stating the no-claims-without-observations contract | `status.go:149–156` |
| **`badgeForStatus` signature change**: `(health.Status)` → `(health.Status, evidenceSummary, string)`; sets `title` attr **only** when a title exists (empty-`title=""` noise found in first golden diff and eliminated) | Golden diffs reviewed; `TestBadgeForStatus_CarriesTitle` |
| **Unit tests** (`evidence_test.go`, internal package): observes warn+fail not pass; recovery keeps proof; latest-timestamp wins; 10k cap; current-checks intersection; nil log; badge title cases (unproven/proven/non-pass/zero-window); strip text cases; render-level strip presence/absence | `go test -run 'TestEvidence\|TestPopulateEvidence\|TestBadge\|TestRender_Evidence'` → ok |
| **Integration tests** (`evidence_integration_test.go`): real probe+pusher through failing fixtures — split ratio ("2 of 3 … the other 1 green rows are unproven"), unproven tooltip present; all-failing fixture reaches "all 2 checks" with no "unproven" text; `setupAllFailing` helper added | `TestIntegration_EvidenceStrip*` pass via `waitForBody` polling (no fixed sleeps) |
| **Golden render tests regenerated twice** and diffs reviewed (first regen exposed the `title=""` defect; second confirmed clean) | `go test -run TestGoldenRender -golden-update` → PASS, `rg 'title=""' testdata/golden/` → 0 hits |
| **Lint remediation** (was exit 1 → exit 0): exhaustive switch case, varnamelen (`at`→`observedAt`, `ev`→`evidence`, `sb`→`builder`), QF1008 (`props.Attrs`), QF1002 (tagless switch → early-return ifs), stringsseq (`strings.SplitSeq`), gci import order (manual — treefmt does not enforce gci, found the hard way) | `nix run .#lint` → exit 0 |
| **Race + vet + browser gates**: `go test -race` on evidence/SSE/render/golden paths ok; `nix run .#vet` exit 0; `nix develop -c go test -run TestBrowser` ok 13.5s (strict-CSP DOM, zero `<style>`, axe audit) | CLI outputs in session log |
| **generate→fmt canonical order** verified stable (AGENTS release-discipline gotcha #1 respected) | `nix run .#generate` → `nix fmt` → clean tree, lint 0 |
| **Docs**: AGENTS.md — file list (+`evidence.go`, +missing `introspect.go` drift fixed, test count 20→34), new design-decision bullet ("Greens are observations, not verification"); CHANGELOG `[Unreleased]`/Added entry; README "Reading the Dashboard" bullet | Commits `27cd8fa`, `6b1bea9` |
| **Parallel-session conflict handling**: foreign view.templ change (Updated-line span removal) preserved, my strip adapted onto it; their half-wired contrast test failure diagnosed as theirs (fixtures can't produce the asserted raw-key line) via baseline worktree at `2770011`, not "fixed" (theirs to finish — they did, later) | Worktree run: "no tests to run" at baseline; suite green at end |

## b) PARTIALLY DONE

| Item | Works now | Open | Effort |
|---|---|---|---|
| **Evidence under PublicMode** | Counts reasoned non-identifying; `anonymizeViewModel` untouched (masks names only) | **No test** asserting the strip renders and tooltips carry no check names under `WithPublicMode` — the reasoning is unproven by CI | S |
| **SSE patch path** | `renderPatch` calls `populateEvidence`; initial GET asserted in integration tests | No explicit assertion that the **SSE stream's initial patch body** contains "Failure evidence" (the `sseHandler`→`renderPatch` seam) | S |
| **Machine-readable evidence** | HTML-only by design (documented in CHANGELOG/AGENTS, mirroring the Updated-age decision) | Metrics gauges (`dashboard_health_checks_total`/`_proven`) and trend/export JSON fields not built and the "deliberately out" decision is only implied, not ADR'd | M |
| **Docs completeness** | AGENTS/CHANGELOG/README updated | `docs/status/archived/` convention not used for this report's placement decision; FEATURES.md/TODO_LIST.md not yet harvested; no ADR-0003 | S |
| **HARVEST handoff** | Section (f) below written harvest-ready | `docs-health` HARVEST into TODO_LIST.md/ROADMAP.md NOT yet run (user said WAIT — flagged as next step) | S |

## c) NOT STARTED

| Item | Why | Still wanted? |
|---|---|---|
| **README screenshot refresh** — `docs/screenshot.png` (+ dark variant) predate the strip; needs `SCREENSHOT_OUTPUT=docs/screenshot.png` runs | Needs env-guarded capture runs; cosmetic | Yes |
| **Metrics gauges / trend+export JSON evidence fields** (the healthwash `registered/checked/errored` pattern, HTML-vs-machine decision) | Byte-stability contract question — see Question 2 | Yes, pending decision |
| **Evidence persistence across restarts** (opt-in file store) | Product decision on window semantics — see Question 3 | Unknown |
| **v0.8.0 release cut** with this feature (release discipline: tag-first, `verify-release.sh`) | Feature just landed; CHANGELOG section ready | Yes |
| **Upstream samber/do proposal** — "checked vs skipped" sentinel on sweep results (samber-linter ISSUE_DRAFT.md direction 2) | Cross-repo, requires verify-before-filing discipline; fixes the ROOT cause (dashboard currently infers from observations only) | Yes — this is the real fix |
| **go-health `Check` metadata** (e.g. check runtime info) so evidence could distinguish checked-vs-skipped at the source | Upstream design change | Yes, long-term |
| **Example demo server narration** (`DEMO_*` env toggle explaining the strip, `safeBasePath` pattern) | Nice-to-have | Yes |
| **DOMAIN_LANGUAGE.md** entries: evidence / proven / unproven / observation window | Not blocking | Yes |

## d) TOTALLY FUCKED UP

Nothing user-facing is broken — all gates green at `6b1bea9`. Radical-honesty items:

1. **The auto-daemon committed a non-compiling tree mid-session — again.** At `597c108` the tree contains my new `badgeForStatus(health.Status, evidenceSummary, string)` in `status.go` next to the stale 1-arg `view_templ.go` call → `go build` fails at that commit. This is the documented "bisect wall" class (AGENTS.md gotcha: "the daemon snapshots half-wired trees"), **recurring and still unfixed by process**. Mitigation: none — anyone bisecting through this window must `git bisect skip` daemon commits. Root cause: no compile guard in the daemon's commit path.
2. **Session-collision churn cost real time.** The parallel session's mid-edit commits invalidated my read-state three times (`file has been modified since read`), one foreign edit landed inside my target region (Required re-reading + adapting), and their half-wired `TestRender_ContrastSafeStatusColors` red the shared suite mid-session (their fixtures — `cache`/`queue` plain names — can never produce the shortened raw-key line the test asserts). No data was lost and nothing was reverted that wasn't mine, but two agents on one working tree without coordination markers is a standing hazard.
3. **I wrote garbage into a test once**: `TestPopulateEvidence_CountsOnlyCurrentChecks` initially contained a nonsense expression (`respWithChecks(map[...]{...}[0:0][0].Status)`) from a mid-thought edit; also a dead `name := ...; _ = name` block in the cap test. Caught by golangci typecheck diagnostics and fixed immediately — but it shipped to disk in that state.

## e) WHAT WE SHOULD IMPROVE

1. **`nix fmt` and golangci disagree on import order.** treefmt (gofumpt+goimports) reports "0 changed" while golangci's gci formatter fails the file. I had to fix gci grouping manually (single non-std group, path-sorted). Fix: add golangci-lint's `fmt` (or gci) to the treefmt pipeline so `nix fmt` actually fixes what `nix run .#lint` flags. Impact: every new file risks a lint-only failure loop.
2. **LSP staleness burned a round trip.** gopls kept flagging `view_templ.go:955 WrongArgCount` after the file was regenerated and `go build` was green. Per the global "independently verify tool output" rule I trusted the CLI — but the editor noise persisted all session. Improvement: `lsp_restart` after templ regen as a habit.
3. **Edit-tool read-state vs the daemon.** Three edits bounced with "file modified since read" because `sed` reads don't refresh the tracker and the daemon bumps mtimes. Improvement: always View (not sed) immediately before editing in this repo.
4. **Test-first discipline slipped on the integration seam.** I wrote unit tests immediately but the integration tests (which caught the real wiring) came after the feature was "done". The integration test is what proved evidence accrues through the real push loop — it should have been drafted alongside `evidence.go`.
5. **Product decisions were defaulted silently.** Default-on strip, HTML-only contract, per-pusher window — each defensible, none user-confirmed. They're now Questions 1–3 instead of footguns discovered later.
6. **Recurring class worth a skill note**: "templ has no closure scope — thread view data explicitly through every templ component in the call chain" (cost me the first failed build). Belongs in AGENTS.md gotchas or the templ-components skill.

## f) Up to 50 things we should get done next

Ranked by impact; Effort: S <30min, M 30min–2hr, L >2hr. **This section is HARVEST input** — TODO_LIST.md/ROADMAP.md pull from here.

| # | Task | Impact | Effort | Category |
|---|---|---|---|---|
| 1 | Run docs-health HARVEST: move section (f) items into TODO_LIST.md/ROADMAP.md | High | S | Documentation |
| 2 | PublicMode evidence test: strip renders, tooltips contain no check names under `WithPublicMode` | High | S | Quality |
| 3 | Assert SSE initial patch body contains "Failure evidence" (sseHandler→renderPatch seam) | High | S | Quality |
| 4 | Add `dashboard_health_checks_total`/`_proven` gauges to hand-rolled metrics (opt-in `WithMetrics`) | High | M | Feature |
| 5 | Expose evidence summary in `/health/trend` + `/health/export` JSON (shared wire mapping) | High | M | Feature |
| 6 | Write ADR-0003: evidence strip design (HTML-only contract, window semantics, intersection rule) | High | S | Documentation |
| 7 | Cut v0.8.0 with the evidence strip (generate→fmt, tag-first, `scripts/verify-release.sh`) | High | M | Release |
| 8 | Regenerate README screenshots (light + dark) showing the strip (`SCREENSHOT_OUTPUT`) | Medium | S | Documentation |
| 9 | Propose upstream samber/do "checked vs skipped" sentinel (coordinate with samber-linter ISSUE_DRAFT; verify-before-filing) | High | L | Feature |
| 10 | Propose go-health `Check` runtime metadata so dashboards can distinguish checked-vs-skipped at the source | High | L | Feature |
| 11 | Persist evidence across restarts (opt-in atomic-write file, modeled on go-atomic-write/baseline pattern) | Medium | L | Feature |
| 12 | Browser test: pass-badge `title` text present in runtime DOM (tooltip currently only render-tested) | Medium | S | Quality |
| 13 | GroupBySource render test: strip + tooltips in source mode | Medium | S | Quality |
| 14 | Accessibility: `title` tooltips are hover-only — add visually-hidden evidence text or `aria-describedby` for screen readers | Medium | M | Quality |
| 15 | Touch devices: `title` tooltips don't fire — decide fallback (`<details>` expansion or accept loss) | Medium | M | Feature |
| 16 | Fuzz target for `evidenceLog.observe` (hostile/churning name maps vs cap) + run `-fuzz` 60s once | Medium | S | Quality |
| 17 | Bench `evidenceLog.observe` at 10k names + `BenchmarkHandler_HTMLRendering` before/after strip cost | Medium | S | Quality |
| 18 | Cache evidence snapshots with a dirty flag (currently copies the name→time map every render/tick) | Medium | M | Quality |
| 19 | Example server: `DEMO_*` toggle narrating the evidence strip (`safeBasePath` validation pattern) | Medium | S | Feature |
| 20 | Add golangci fmt/gci to `nix fmt` so formatting and linting agree (kill the manual-import-order loop) | Medium | S | Quality |
| 21 | Auto-daemon compile guard: run `go build ./...` before heuristic commits (stops the bisect-wall class at `597c108`) | High | M | Quality |
| 22 | FEATURES.md: add evidence strip under DONE | Medium | S | Documentation |
| 23 | DOMAIN_LANGUAGE.md: evidence / proven / unproven / observation window / health-washing | Medium | S | Documentation |
| 24 | AGENTS.md gotcha: "templ has no closure scope — thread view data explicitly" (this session's failed first build) | Medium | S | Documentation |
| 25 | Document PushOnChange staleness reasoning: evidence counts only change on fingerprint change, so no missed broadcasts | Low | S | Documentation |
| 26 | Document per-replica semantics behind a load balancer (each replica has its own observation window) | Medium | S | Documentation |
| 27 | Per-source evidence rollup on GroupBySource cards (proven ratio per source card) | Medium | M | Feature |
| 28 | Per-check history ring → per-row "last non-pass" timeline entries | Medium | L | Feature |
| 29 | Reconnect-path regression test: strip present in the patch a reconnecting client receives | Medium | M | Quality |
| 30 | Strip text review for the "0 of N" case with 1 check ("0 of 1 checks has ever deviated") — grammar sweep of all three verdicts × singular/plural | Low | S | Quality |
| 31 | Troubleshooting FAQ: "Why does my dashboard say unproven?" in README | Low | S | Documentation |
| 32 | Cross-link: samber-linter README §7 should cite go-health-dashboard's implemented strip as the runtime-companion UI lesson | Low | S | Documentation |
| 33 | Decide + implement strip dismissal or opt-out (`WithHideEvidence`) per Question 1 | High | S | Feature |
| 34 | Decide machine-contract exposure per Question 2 (feeds items 4–5) | High | S | Feature |
| 35 | Decide persistence semantics per Question 3 (feeds item 11) | High | S | Feature |
| 36 | `docs/status/` archival policy: move this report to `archived/` when superseded (existing convention) | Low | S | Documentation |
| 37 | Consider `role="note"`/`aria-label` on the strip `<p>` for AT announcement without hover | Medium | S | Quality |
| 38 | Verify fuzz seed corpus still passes with the new viewModel field (ran green in suite; confirm fuzz targets unaffected) | Low | S | Quality |
| 39 | Check `check-changelog.sh` compliance for the next release section ordering | Low | S | Release |
| 40 | Update `scripts/check-ui-pins.sh` docs if the strip ever needs templ-components changes (none today — pure dashboard markup) | Low | S | Documentation |
| 41 | Add evidence wording to the demo screenshot captions in README | Low | S | Documentation |
| 42 | Sweep other LarsArtmann dashboards (CV production) for upgrade to a version with the strip — the original incident audience | High | L | Feature |
| 43 | Evaluate naming: "deviated from pass" vs samber-linter's "fail-capable" — unify ecosystem vocabulary | Low | S | Documentation |
| 44 | Add race-soak test: concurrent observe (push loop) + snapshot (renders) under `-race` with high tick rate | Low | M | Quality |
| 45 | Roadmap: evidence-based grouping — auto-collapse "unproven greens" separately from "proven greens" | Low | L | Feature |

## g) Questions I cannot figure out myself

1. **Noise tolerance / product posture**: is the evidence strip default-on for every consumer the right call, or do you want an opt-out (`WithHideEvidence`) and/or a dismissible UI? For a large all-green fleet the zero-proven warning reads as a permanent scolding — I cannot judge whether that truth-telling or that noise is wanted for your NOC audience.
2. **Machine contracts**: should evidence surface in Prometheus metrics and trend/export JSON (healthaudit's `registered/checked/errored` pattern), or stay HTML-only like the `LastUpdatedTime` decision? This hinges on your byte-stability philosophy for the JSON contract, which I can only infer from one AGENTS.md bullet.
3. **Window semantics**: evidence resets when the dashboard restarts — with frequent deploys the strip will perpetually say "unproven". Is per-pusher-lifetime the intended truth ("since monitoring started"), or should evidence persist across restarts (opt-in file store), trading freshness for continuity?

---

*Point-in-time snapshot at `6b1bea9`. Section (f) is HARVEST-ready; per the status-report skill, TODO_LIST.md/ROADMAP.md should be updated from it when work resumes.*
