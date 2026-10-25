# Status Report — Pareto Plan Execution (T1–T9 Complete)

**Date**: 2026-09-22 22:35 CEST · **Session**: go-health superb-consumption (arc: deep-dive audit → wire fixes → Pareto plan → **this: plan execution**)
**Repo state**: master synced with origin through `837f12e`, working tree clean, all local gates green.
**Scope**: this report covers THIS session run only (plan execution T1–T9 + self-critique). The earlier arc phases (audit report, wire fixes, plan authoring) are context, not re-audited.

---

## Direct answers first

**What did you forget?**

~~1. `scripts/pre-push-checks.sh` — the AGENTS.md session-closing desk gate — was never run before the push. Local gates (build/test/lint/vet/guards) were green and CI runs the same checks, but the gate exists precisely to catch this class of omission before `git push`.~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
~~2. `FEATURES.md` feature rows for the two new capabilities: I updated the test-suite count in lockstep but never added "InstanceID stat card" / "Healthzer route" rows to the feature inventory itself.~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
~~3. The aggregate-path Healthz test named in plan T7.3 ("probe stub with/without Healthz; **aggregate path**") — with/without/default shipped, the aggregate-fed case did not.~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
~~4. The Pareto plan doc itself is now 100% executed but still reads as future work — no completion annotation.~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)

**What could you have done better?**

- Ran Go tooling once **while a buildflow run was active** (first federation test invocation) — a direct violation of a documented gotcha; got lucky with no torn state.
- Five edit-quality stumbles that each cost a fix cycle: consumed the export test's closing brace, swallowed its trailing assertion into the new test, referenced a non-existent `muxOrStub(t)` helper, mis-ordered imports, left a `( federation` typo in CHANGELOG.
- The plan itself contained an unverified external claim (`*healthfederation.Federation`) — the actual type is `*Prober`. Caught only by reading the sibling source at execution time; a verify-external-claims pass at plan-authoring time would have caught it.
- Sequential-doc churn: T2's CHANGELOG text declared the trend endpoint out of scope for `Deterministic`; T6 reversed that ~40 minutes later. Closeout prose written too early.

**What could you still improve?** — see (e); the systemic answer is "gate myself with the same checklists I wrote" (desk gate, FEATURES registry, verify-before-writing-plans).

---

## a) FULLY DONE (this session run)

| #  | Work | Evidence                                                                                                                                                                                                                                                                                                                                                                                                        |
| -- | ---- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~ | 1    | **T1 Go-directive guard**: `scripts/check-go-directive.sh` (exact `go 1.27.1` assertion, restore pattern in header), wired into CI Build **and** Test jobs, negative-tested (downgraded + missing directive → exit 1 with `::error::`), AGENTS.md buildflow-trap bullet cross-linked                                                                                                                            |
| ~~ | 2    | **T2 Docs closeout**: superseded `Re-pin go-health v0.3.0` BLOCKED row closed, federation-row premise refreshed, audit rows added to Next Up, CHANGELOG `[Unreleased]` Fixed (two `Deterministic` fixes + test names), Changed (v0.4.0 ratification + go-floor note), Added (guard)                                                                                                                             |
| ~~ | 3    | **T3 Federation insurance batch**: compile-time `var _ dashboard.Prober = (*healthfederation.Prober)(nil)`; end-to-end evidence test (unreachable remote's `dark/reachable` fail row → "Failure evidence: 1 of 2 checks" + unproven green); seam spot-checks with a quote-hostile remote name through hand-rolled Prometheus label escaping AND webhook announce payload; health-hub-suffices decision recorded |
| ~~ | 4    | **T4 Rank adoption, behavior-locked**: `numericHealthStatus` derives known statuses from `health.Status.Rank()` (unknown stays −1), `statusValue` computes `Rank()/2` (unknown stays plot-0), `worstGroupStatus` keeps early-return switch with Rank citation + explicit stricter-than-Rank rationale. Baseline test run before, identical run after                                                            |
| ~~ | 5    | **T5 InstanceID surfacing**: `viewModel.InstanceID` wired from `resp.InstanceID`, conditional "Instance" stat card in `view.templ`, `anonymizeViewModel` clears it (public mode), positive/negative/masking tests, golden ceremony (regenerated + reviewed by eye: exactly one whitespace byte across 5 files — templ conditional separator, zero semantic drift), browser suite green                          |
| ~~ | 6    | **T6 Trend Deterministic**: `/health/trend` marshals with `json.Deterministic(true)` (slices-only today, future-proofing rationale in comment), `TestTrendHandler_JSONIsByteStable` pins two scrapes byte-identical                                                                                                                                                                                             |
| ~~ | 7    | **T7 Healthz passthrough**: `Healthzer` optional interface (exported `Prober` untouched), `Routes.Healthz` default-disabled with kubelet-collision rationale, `WithBasePath` prefixing, `registerHealthz` wiring (unwrapped like probe routes), 3 tests (registered-when-implemented / skipped-when-absent / disabled-by-default with `/healthz` liveness intact)                                               |
| ~~ | 8    | **T8 Example → NewChecks**: default probe builds services via `health.NewChecks` (plain funcs, timed), verified LIVE: rows render real µs durations (22µs/11µs/9µs), `/readyz` JSON carries `duration_ns` per check; samber/do lifecycle demo preserved (`Register` still hosts the dashboard)                                                                                                                  |
| ~~ | 9    | **T9 Final gates**: `nix run .#build` → `nix fmt` (canonical order) → full test suite (main/hub/version all ok) → lint **0 issues** (after fixing 2 self-introduced findings: cyclop 14>12 via `registerHealthz` extraction; unparam unused `injector` param) → vet → check-ui-pins OK → check-go-directive OK → pushed `cdea97b..837f12e`                                                                      |
| ~~ | 10   | **FEATURES registry kept honest**: test-suite count updated in same change per task (281→286→287→289→292), including rescuing a pre-existing +2 drift from the earlier byte-stability session                                                                                                                                                                                                                   |
| ~~ | 11   | **Daemon discipline held**: every task committed immediately after verification; ~5 daemon sweeps folded forward, none rebased; go.mod verified `go 1.27.1` + v0.4.0 after every buildflow pass (untouched at every check)                                                                                                                                                                                      |

## b) PARTIALLY DONE

| #  | Work | What exists                                                                     | What's missing                                                              |
| -- | ---- | ------------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| ~~ | 1    | **FEATURES.md inventory**                                                       | Test-count row accurate (292/40)                                            |
| ~~ | 2    | **T7.3 aggregate-path Healthz test**                                            | Stub with/without capability + default-route test                           |
| ~~ | 3    | **InstanceID on the hub**                                                       | Response-level stat card (serving replica disambiguated)                    |
| ~~ | 4    | **Audit report currency** (`docs/research/2026-09-22_go-health-deep-dive.html`) | Version currency updated to v0.4.0 earlier today                            |
| ~~ | 5    | **Pareto plan doc**                                                             | Complete, correct execution graph                                           |
| ~~ | 6    | **Hub binary × Healthz**                                                        | Library capability ships; hub consumers could configure it via `WithRoutes` |

## c) NOT STARTED (all deliberately out of this plan's scope — none regressed)

~~1. **v0.10.0 release cut** — `[Unreleased]` now carries two features (Healthzer, InstanceID), the guard, NewChecks example, and three fix/test batches; release timing is an explicit user decision (Blocked row).~~ closed or routed — swept 2026-09-23 (TODO_LIST sweep row); dispositions in CHANGELOG [0.10.x]/[0.11.0]
~~2. Fleet-level BuildFlow fixes (go-mod-normalize downgrade, DAG ordering, structure-linter/branching-flow/templ-generate unskips) — live in the BuildFlow repo, not here.~~ closed or routed — swept 2026-09-23 (TODO_LIST sweep row); dispositions in CHANGELOG [0.10.x]/[0.11.0]
~~3. All ~30 pre-existing TODO_LIST rows (deploy e2e, a11y keyboard smoke, metrics doc, compose CI policy, upstream filings, …) — untouched by design, still open.~~ closed or routed — swept 2026-09-23 (TODO_LIST sweep row); dispositions in CHANGELOG [0.10.x]/[0.11.0]
~~4. Evidence persistence / evidence-in-metrics product decisions — still Blocked, user session needed.~~ closed or routed — swept 2026-09-23 (TODO_LIST sweep row); dispositions in CHANGELOG [0.10.x]/[0.11.0]
~~5. Upstream filings (templ gofumpt conflict, erraudit false positives) — verify-before-filing gates each; not started.~~ closed or routed — swept 2026-09-23 (TODO_LIST sweep row); dispositions in CHANGELOG [0.10.x]/[0.11.0]

## d) TOTALLY FUCKED UP

Nothing shipped broken — final state is green everywhere (build, full suite, race-free CI matrix locally implied, lint 0, both guards, browser suite, goldens reviewed, pushed). But these process failures are real and worth naming:

~~1. **Ran `go test` while buildflow was mid-run** (first federation test invocation, ~22:05). Documented gotcha, explicit rule, violated it anyway. Outcome was benign (no go.mod/templ tearing) — luck, not discipline. Also caught it only AFTER the run.~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
~~2. **Pushed without `scripts/pre-push-checks.sh`** — the AGENTS.md §0 release-discipline desk gate. The plan's own T9 forgot to include it, and I executed the plan instead of the repo's rules. The repo rule wins; the plan was wrong; I propagated the error.~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
~~3. **Edit-tool sloppiness with real rework cost**: one insertion consumed a test function's closing brace AND its trailing `checks`-sorted assertion (compile error + wrong-test-home failure, two fix cycles); one appended test referenced a helper I never wrote; imports mis-ordered once; CHANGELOG typo. All caught within one tool call each, but five stumbles in one session is a pattern, not noise.~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
~~4. **The FEATURES +2 drift was my own earlier work**: this same session-arc shipped two byte-stability tests in the morning without updating FEATURES, then T3 "discovered" the drift. Found it by accident while counting for the new tests.~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
~~5. **Two content errors in things I wrote fast**: the plan's federation type name (`*Federation`) was fabricated from memory; T2's CHANGELOG "trend stays alone" sentence was wrong within the hour. Both corrected, both were avoidable by reading instead of recalling.~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)

## e) WHAT WE SHOULD IMPROVE

~~1. **Execute the repo's gates, not the plan's gates.** Plans are derived work; AGENTS.md/session-closing rules are authoritative. `pre-push-checks.sh` belongs in T9 of every future plan (or in a fleet default).~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
~~2. **verify-before-filing applies to plans too** — any external symbol/version/flag in a plan gets read from source before the plan commits. The `*Federation` miss is the exact failure mode the skill names.~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
~~3. **Registry discipline as a checklist item per task**: FEATURES rows for features, counts for tests, CHANGELOG for user-visible change — one line per task, not an afterthought found by counting.~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
~~4. **Doc-vs-sequence churn**: when tasks are planned sequentially (T2 before T6), write closeout prose that doesn't pre-declare later tasks' outcomes, or update in the same commit that changes the fact.~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
~~5. **Buildflow check placement**: `pgrep -fa buildflow` belongs immediately before EVERY Go-tooling invocation in a session, not once per phase. Cheap, mechanizable (a wrapper or a hook).~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
~~6. **Daemon-commit hygiene is working but fragile**: intent messages kept getting separated from content (5 sweeps this session). Faster commit cadence after each edit batch — or accept the fold-forward pattern as canonical and stop tracking it manually.~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
~~7. **gopls ambient noise** (118 false diagnostics from the 1.27.0-vs-1.27.1 module-load failure) is read 30+ times per session. The `.vscode/settings.json` dismissal exists; the Crush LSP integration needs the same treatment or an explicit ignore.~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)

## f) Up to 50 things to get done next

_Brainstorm list, impact-ordered, marked: [NEW] born this session, [ROW] existing TODO_LIST item, [FLEET] lives outside this repo, [DECISION] needs user input._

| #  | Thing | Why / source                                                                                                                             |
| -- | ----- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| ~~ | 1     | [NEW] Run `scripts/pre-push-checks.sh` now as belt-and-braces                                                                            |
| ~~ | 2     | [NEW] Add FEATURES.md rows: InstanceID stat card, Healthzer route, NewChecks example                                                     |
| ~~ | 3     | [NEW] Aggregate-path Healthz test (T7.3 remainder)                                                                                       |
| ~~ | 4     | [NEW] Annotate the Pareto plan doc: T1–T9 executed, links to commits                                                                     |
| ~~ | 5     | [NEW] ANNOTATE the deep-dive HTML report: both Deterministic findings fixed today                                                        |
| ~~ | 6     | [DECISION] Cut v0.10.0 — Unreleased carries Healthzer + InstanceID + guard + fixes; tag-first checklist                                  |
| ~~ | 7     | [NEW] `HEALTH_HUB_HEALTHZ` env knob on the hub binary                                                                                    |
| ~~ | 8     | [NEW] AGENTS.md: document Healthzer + Routes.Healthz in Key Design Decisions                                                             |
| ~~ | 9     | [NEW] AGENTS.md: document InstanceID card + public-mode masking                                                                          |
| ~~ | 10    | [NEW] README: Instance card + Healthz route for end-users                                                                                |
| ~~ | 11    | [NEW] Golden fixture WITH InstanceID set (positive pin of the card render)                                                               |
| ~~ | 12    | [NEW] Introspection: document the deliberate Healthz-route omission (or include it)                                                      |
| ~~ | 13    | [NEW] `scripts/demo-smoke.sh`: scripted example-server scrape (curl-shaped)                                                              |
| ~~ | 14    | [FLEET] File buildflow go-mod-normalize downgrade (1.27.1→1.27 twice in a day) upstream                                                  |
| ~~ | 15    | [ROW] shellcheck for `scripts/*.sh` in CI — new guard script raises the stakes                                                           |
| ~~ | 16    | [DECISION] Evidence persistence + evidence-in-metrics                                                                                    |
| ~~ | 17    | [ROW] Exercise `release-draft.yml` on the next tag                                                                                       |
| ~~ | 18    | [ROW] verify-dep-bump.sh: read coverage floor from ci.yml                                                                                |
| ~~ | 19    | [ROW] verify-dep-bump.sh: daemon-proof fmt check                                                                                         |
| ~~ | 20    | [ROW] verify-dep-bump.sh: devShell-independent `go tool cover`                                                                           |
| ~~ | 21    | [ROW] Compose-stack CI policy                                                                                                            |
| ~~ | 22    | [ROW] Tag-ruleset bypass policy (`bypass: never` vs admin)                                                                               |
| ~~ | 23    | [ROW] Enable private vulnerability reporting                                                                                             |
| ~~ | 24    | [FLEET] Unskip go-structure-linter when BuildFlow pins config-capable SDK                                                                |
| ~~ | 25    | [FLEET] Unskip branching-flow (phantom-type API decision)                                                                                |
| ~~ | 26    | [FLEET] Unskip templ-generate (DAG ordering)                                                                                             |
| ~~ | 27    | [ROW] Boot full compose stack e2e + screenshot                                                                                           |
| ~~ | 28    | [ROW] Write `deploy/README.md`                                                                                                           |
| ~~ | 29    | [ROW] `scripts/set-tag-protection.sh` rulesets-as-code                                                                                   |
| ~~ | 30    | [ROW] CHANGELOG per-bullet tag audit v0.3.0–v0.8.0                                                                                       |
| ~~ | 31    | [ROW] Pin `shutting_down:true` webhook payload case                                                                                      |
| ~~ | 32    | [ROW] `testdata/golden/source.html` vs since-tooltips check                                                                              |
| ~~ | 33    | [ROW] M88 keyboard-navigation a11y smoke                                                                                                 |
| ~~ | 34    | [ROW] M89 metrics endpoint under strict CSP browser test                                                                                 |
| ~~ | 35    | [ROW] M91 browser-suite startup latency measurement                                                                                      |
| ~~ | 36    | [ROW] M92 load-test env knobs beyond 20×3 fixture                                                                                        |
| ~~ | 37    | [ROW] M103 unit-test the version-guard grep logic                                                                                        |
| ~~ | 38    | [ROW] M105 doc.go combo examples (webhook+public, WithBasePath)                                                                          |
| ~~ | 39    | [ROW] M106 bisect-wall audit amendment (2026-09-17/22 scars)                                                                             |
| ~~ | 40    | [ROW] M107 nightly Release-page-exists job                                                                                               |
| ~~ | 41    | [ROW] Mobile visual check of the metadata line                                                                                           |
| ~~ | 42    | [ROW] Dark-mode contrast measurement (dim metadata line)                                                                                 |
| ~~ | 43    | [ROW] Re-run aggregate load test on current render                                                                                       |
| ~~ | 44    | [ROW] Toolchain bump via verify-dep-bump.sh when 1.27.x patch ships                                                                      |
| ~~ | 45    | [ROW] Digest-pin compose images + dependabot                                                                                             |
| ~~ | 46    | [ROW] Compose healthchecks (prometheus→grafana readiness)                                                                                |
| ~~ | 47    | [ROW] Write `docs/metrics.md` consumer reference                                                                                         |
| ~~ | 48    | [ROW] Grafana threshold-alert example                                                                                                    |
| ~~ | 49    | [ROW] Upstream filings (templ formatting, erraudit ×2)                                                                                   |
| ~~ | 50    | [NEW] Consider InstanceID in metrics labels (replica disambiguation vs cardinality) — needs a written tradeoff before anyone asks for it |

_Items 1–5 + 7–13 are the true "next session" queue; 6 gates most of them. Items 16+ are the pre-existing parallel backlog — do not re-plan here (docs-health HARVEST routes them)._

## g) Questions I can NOT figure out myself

~~1. **Release timing**: `[Unreleased]` now carries two features (Healthzer capability, InstanceID card), the CI guard, the NewChecks example, and three fix batches. Cut **v0.10.0 now** (tag-first checklist, it's feature+floor breaking as pre-declared), or keep accruing until a consumer (CV deploy? hub rollout?) forces the cut?~~ routed — standing TODO_LIST BLOCKED row or a later decision record (ROADMAP/AGENTS)
~~2. **Healthz default-route policy**: I shipped `Routes.Healthz` disabled-by-default (kubelet owns `/healthz`; combined handler 503s during boot). Stay disabled-forever as the permanent contract, or should it eventually default to `/health/healthz` for discoverability? This is an API contract decision that gets expensive to change after external implementors exist.~~ routed — standing TODO_LIST BLOCKED row or a later decision record (ROADMAP/AGENTS)
~~3. **Fleet filing scope**: the buildflow `go-mod-normalize` step downgrading `go 1.27.1→1.27` twice in one day is now guarded here but still fires fleet-wide every run. Do you want it filed/fixed in the BuildFlow repo (and if so, is that worth a session soon?), or does the per-repo guard make it a non-priority?~~ routed — standing TODO_LIST BLOCKED row or a later decision record (ROADMAP/AGENTS)

---

**Handshake note for the next session**: before ANY write — `git log --oneline -10`, `ls docs/status/ | tail`, `git status`. This report's section (f) items 1–5 are the immediate queue; TODO_LIST.md already carries the audit rows closed this session. Latest on origin: `837f12e`.
