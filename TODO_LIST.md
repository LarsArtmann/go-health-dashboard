# TODO List

> Short-term, actionable, bounded work items, verified against the actual code
> (docs-health HARVEST pass 2026-09-03: sources cited per item; closed items
> live in `CHANGELOG.md`, never here). For long-term vision and unrefined
> ideas, see ROADMAP.md.

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                               |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed. |

## Next Up

Harvested from the v0.3.x retrospective
(`docs/status/2026-09-03_12-32_v03x-cycle-retrospective.md` §f) and older
status reports, each verified against the code on 2026-09-03.

### Release & history

| Task                                                                        | Status    | Impact | Effort | Notes                                                                             |
| --------------------------------------------------------------------------- | --------- | ------ | ------ | --------------------------------------------------------------------------------- |
| Cut the next release: re-head CHANGELOG `[Unreleased]`, bump `Version`, tag | 🔴 `TODO` | High   | 45min  | Retrospective f1. Pending version choice (Blocked below)                          |
| CI guard: `Version` const must match the latest git tag                     | 🔴 `TODO` | High   | 30min  | Retrospective f2. Stale-const bug has bitten twice (v0.2.0 era, stray v0.3.0 tag) |
| Bisectability audit `071c251..HEAD`; document any non-building commits      | 🔴 `TODO` | Medium | 45min  | Retrospective f3. `72783fc` already documented in AGENTS.md (bisect-skip note)    |

### CI & verification

| Task                                                    | Status    | Impact | Effort | Notes                                                                                       |
| ------------------------------------------------------- | --------- | ------ | ------ | ------------------------------------------------------------------------------------------- |
| Validate `fuzz.yml` end-to-end via `workflow_dispatch`  | 🔴 `TODO` | Medium | 15min  | Retrospective f7. Confirm the crasher-print step works                                      |
| Pin golangci-lint version (currently `version: latest`) | 🔴 `TODO` | Medium | 10min  | Retrospective f10; `.github/workflows/ci.yml` lint job                                      |
| Pin templ CLI in CI (currently `@latest`)               | 🔴 `TODO` | Medium | 10min  | Retrospective f11; four `go install` steps in `ci.yml`                                      |
| Add CI concurrency group to cancel superseded runs      | 🔴 `TODO` | Low    | 10min  | Retrospective f12                                                                           |
| Nightly fuzz: open an issue on failure                  | 🔴 `TODO` | Low    | 20min  | Retrospective f13. Currently only prints crashers                                           |
| Consider coverage artifact upload + optional CI floor   | 🔴 `TODO` | Low    | 30min  | Retrospective f14. Verify `actions/upload-artifact` SHA before adding (no unpinned actions) |

Verified green on 2026-09-03, so no action needed: CI browser job ran on a
real runner (all 5 jobs pass, `gh run view 33763955031`), `nix run
.#vulncheck` finds no vulnerabilities, coverage baseline is 76.9% total
(retrospective f6/f8/f9 closed).

### Code quality

| Task                                                                       | Status    | Impact | Effort | Notes                                         |
| -------------------------------------------------------------------------- | --------- | ------ | ------ | --------------------------------------------- |
| Split `dashboard.go` (~700 lines): config/options vs lifecycle vs handlers | 🔴 `TODO` | Medium | 90min  | Retrospective f15                             |
| Extract `historyBuffer` into `history.go`                                  | 🔴 `TODO` | Low    | 20min  | Retrospective f16. `pusher.go` is growing     |
| Deduplicate sample→JSON mapping shared by Trend/Export handlers            | 🔴 `TODO` | Low    | 20min  | Retrospective f17; `trend.go`                 |
| Fix `TrendHandler` 503 message (not-started vs not-enabled)                | 🔴 `TODO` | Low    | 10min  | Retrospective f18; `trend.go`                 |
| Rename `BenchmarkDashboard_PatchRender` (it serves full HTML)              | 🔴 `TODO` | Low    | 5min   | Retrospective f20; `metrics_bench_test.go:82` |
| Inline the `maxRequestsInvalid` one-liner in the example                   | 🔴 `TODO` | Low    | 5min   | Retrospective f21; `example/main.go:251`      |

### Testing gaps

| Task                                                                                    | Status    | Impact | Effort | Notes                                                          |
| --------------------------------------------------------------------------------------- | --------- | ------ | ------ | -------------------------------------------------------------- |
| Replace flaky `time.Sleep` in SSE tests with event-driven waits                         | 🔴 `TODO` | Medium | 30min  | Defect-fix report d4/d5; `sse_integration_test.go:265`, `:492` |
| Test `WithMaxSSEConnections(0)` allows unlimited connections                            | 🔴 `TODO` | Low    | 15min  | Defect-fix report f17                                          |
| Test SSE handler when the probe hasn't started (degraded render)                        | 🔴 `TODO` | Low    | 20min  | Defect-fix report f18                                          |
| Assert no `style=` attributes in SSE patch content                                      | 🔴 `TODO` | Low    | 15min  | TODO-implementation report f10; patches are inner-HTML         |
| Distinguish not-started vs shut-down in `ErrPusherNotActive` (or add a second sentinel) | 🔴 `TODO` | Low    | 20min  | Defect-fix report f16; `dashboard.go:666`                      |

### Features & polish

| Task                                                                                                 | Status    | Impact | Effort | Notes                                                                                               |
| ---------------------------------------------------------------------------------------------------- | --------- | ------ | ------ | --------------------------------------------------------------------------------------------------- |
| Refresh stamp: use last sample timestamp (observation time), not render time                         | 🔴 `TODO` | Medium | 30min  | Retrospective b6/f23. Initial HTML can be up to one probe interval stale                            |
| Scope the axe `definition-list` tolerance to the StatCard nodes                                      | 🔴 `TODO` | Low    | 20min  | Retrospective b7/f19. Currently the whole rule ID is filtered in `browser_test.go`                  |
| Upstream PR to templ-components: StatCard `<dl>` fix (+ goldens); then remove the axe tolerance here | 🔴 `TODO` | Low    | 60min  | Retrospective f47/f48; diagnosis in `docs/planning/2026-09-03_issue-drafts.md` (templ-components#6) |
| Example toggles: `DEMO_PUBLIC=1`, `DEMO_BASE_PATH=/status`                                           | 🔴 `TODO` | Low    | 20min  | Retrospective f45/f46; `example/main.go`                                                            |
| Document rate-limiter shared-bucket semantics in the README options list                             | 🔴 `TODO` | Low    | 10min  | Retrospective f25. Per-route buckets rejected for now (ROADMAP)                                     |

### Docs

| Task                                                                                   | Status    | Impact | Effort | Notes                                                            |
| -------------------------------------------------------------------------------------- | --------- | ------ | ------ | ---------------------------------------------------------------- |
| `doc.go`: add `Register` + `ErrPusherNotActive` examples                               | 🔴 `TODO` | Low    | 20min  | Defect-fix report f21/f22. README Register note added 2026-09-03 |
| CONTRIBUTING.md: how to run browser + screenshot tests; mention the `Register` DI path | 🔴 `TODO` | Low    | 20min  | Sweep report f31; defect-fix report f35                          |

## Blocked (needs user decision)

| Task                                    | Status       | Why blocked                                                                                                       | Evidence                    |
| --------------------------------------- | ------------ | ----------------------------------------------------------------------------------------------------------------- | --------------------------- |
| Build-tag gating for SSE                | 🔵 `BLOCKED` | Consumers who only want HTML shouldn't need GOEXPERIMENT=jsonv2. Requires decision: accept, fork go-sse, or gate. | `ROADMAP.md` Open Questions |
| Next release version (v0.4.0 vs v0.3.2) | 🔵 `BLOCKED` | Post-v0.3.1 batch is purely additive; semver suggests v0.4.0, but 0.x is loose. Gates the release task above.     | `ROADMAP.md` Open Questions |
| Fingerprint format stability            | 🔵 `BLOCKED` | Length-prefix fix changed fingerprint values; documented as accepted in CHANGELOG pending a versioning decision.  | `ROADMAP.md` Open Questions |

Everything else from the v0.3.x cycle brainstorms either shipped (see
`CHANGELOG.md`), was closed with a reason in the annotated reports under
`docs/status/`, or lives in `ROADMAP.md` as raw ideas.
