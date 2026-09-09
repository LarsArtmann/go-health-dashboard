# TODO List

> Short-term, actionable, bounded work items, verified against the actual
> code (docs-health HARVEST passes 2026-09-03, 2026-09-04, and 2026-09-10 —
> closed items live in `CHANGELOG.md`, never here). For long-term vision and
> unrefined ideas, see ROADMAP.md.

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                               |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed. |

## Next Up

Everything below survived the 2026-09-10 full-execution session (Pareto plan
`docs/planning/2026-09-10_00-26_ci-green-and-backlog-pareto.html`, closing
report `docs/status/2026-09-10_01-15_full-execution-session.md`).

### Polish & cleanup (Low impact)

| Task                                                                    | Status    | Impact | Effort | Notes                                                                                                |
| ----------------------------------------------------------------------- | --------- | ------ | ------ | ---------------------------------------------------------------------------------------------------- |
| Mobile: design decision + true row stacking (not just overflow containment) | 🔴 `TODO` | Medium | M    | Plan T9's original intent; visual design decision first (report 2026-09-09 b2)                         |
| Extract the three inline page scripts (pill, persistence, theme) into one nonce'd bootstrap | 🔴 `TODO` | Low | M | Saves two script tags; deliberately deferred 2026-09-10 — three small tested scripts beat one merged risky refactor mid-session; revisit with a dedicated CSP-suite run |
| `view.templ` split for pill/persistence if the file keeps growing       | 🔴 `TODO` | Low    | M      | ADR-0001 file-split rationale; conditional on growth (~500 lines now)                                 |
| Dark-mode contrast pass of pill/badge/link colors on the live page      | 🔴 `TODO` | Low    | S      | Rides the CV rollout (axe already covers the harness-representable parts)                             |
| Full `aria-live` filter-count announcer (scripted match counts)         | 🔴 `TODO` | Low    | M      | The no-match hint is already a `role="status"` region; counts need a script — only if a screen-reader user asks |
| `rg -r` habit guard (machine-level ripgrep config)                      | 🔴 `TODO` | Low    | S      | Personal tooling, not project code — decide the mechanism (alias/config) yourself                     |

## Blocked (needs user decision)

| Task                                                     | Status       | Why blocked                                                                                                                                                                | Evidence                                  |
| -------------------------------------------------------- | ------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------- |
| Tag `v0.7.0` + push `--follow-tags` (greens version-guard) | 🔵 `BLOCKED` | Release decision + publish act; also decides whether `[Unreleased]` (now substantial: introspection, NDJSON, grouping, ceremony, PersistCollapse fix) folds into 0.7.0 or ships as 0.8.0 (report g Q1) | CI version-guard red; no `v0.7.0` tag     |
| CV-side adoption: bump go.mod, deploy, verify live page  | 🔵 `BLOCKED` | Deploy pipeline + rollout order are the user's call (report g Q2). Verified 2026-09-10 (read-only): CV serves a CSP-safe mini-client, not the Datastar SDK — a safe bump needs `dashboard.WithNoDatastarRuntime()` (shipped in `[Unreleased]`) or the filter/pill would render dead | CV `internal/di/health_dashboard.go` pins v0.6.1 |
| Copy affordance for raw check keys                      | 🔵 `BLOCKED` | Product decision (report g Q3): title-attr + select-text vs a Datastar clipboard action                                                                                      | status report 2026-09-09 question 3       |
| Pin-guard keep sign-off                                  | 🔵 `BLOCKED` | Guard rewritten to v1.16.0 pins per the keep decision (fifth sweep caught 2026-09-10); the deviation from the original removal condition wants sign-off                        | `scripts/check-ui-pins.sh` header         |
| Per-check latency metric labels (F5 remainder)           | 🔵 `BLOCKED` | go-health `Check` carries no per-check duration; unblocked when [go-health#2](https://github.com/LarsArtmann/go-health/issues/2) ships `Check.Since`/`Duration`               | `docs/upstream/go-health-check-timestamps-issue-draft.md` |
| Build-tag gating for SSE                                 | 🔵 `BLOCKED` | Consumers who only want HTML shouldn't need GOEXPERIMENT=jsonv2. Requires decision: accept, fork go-sse, or gate.                                                           | `ROADMAP.md` Open Questions                |
| Fingerprint format stability                             | 🔵 `BLOCKED` | Length-prefix fix changed fingerprint values; documented as accepted in CHANGELOG pending a versioning decision.                                                            | `ROADMAP.md` Open Questions                |

Everything else from the v0.3.x–0.7.0 cycles either shipped (see
`CHANGELOG.md` `[Unreleased]` for the 2026-09-10 session: v1.16.0 pin
ceremony + axe tolerance retirement, PersistCollapse patch-survival fix,
introspection completeness, NDJSON export, `WithGrouping(BySource)`,
`WithNoDatastarRuntime`, example toggles, the RetryAlways×503 interplay
test, golden renders, the load-test harness with recorded numbers, and the
upstream go-health#2 filing), was closed with a reason in the annotated
reports under `docs/status/` (fully-executed reports move to `archived/`),
or lives in ROADMAP.md as raw ideas. Session-closing gate:
`scripts/pre-push-checks.sh` (only the version-vs-tag check is red, pending
the release decision above). Known-broken-commit SHAs for
`git bisect skip`: see AGENTS.md and
`docs/status/archived/2026-09-04_19-15_bisectability-audit.md`.
