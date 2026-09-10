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

The 2026-09-10 late-session sweep emptied both tables below: all five
release-hygiene rows and all six polish rows shipped or closed. See
`CHANGELOG.md` `[Unreleased]` for what shipped (verify-release.sh, the
CHANGELOG structural lint in CI, the release-lesson codification, the
v0.6.1 GitHub Release backfill, mobile row stacking, the single nonce'd
page bootstrap + `page_scripts.templ` split, and the WCAG AA contrast
pass) and the notes here for the two closed-without-code rows:

- Dependabot reds (PRs #13/#12/#4): root-caused — stale branches created
  before the v1.16.0 pin ceremony and the v0.7.0 tag, so the pin guard
  and version guard failed on their own branches exactly as designed.
  `gh pr update-branch` fixed all three; each is now 10/10 green and
  merge is a one-command user decision (`gh pr merge <n>`).
- `rg -r` habit guard: already satisfied machine-level by
  `~/.config/fish/conf.d/01-rg-replace-guard.fish` (2026-08-30,
  deliberate warn-only tripwire: replacement is a legitimate rg feature).
  A duplicate written this session was removed.
- Full `aria-live` filter-count announcer: closed without code — the
  row's own gate ("only if a screen-reader user asks") is unmet; the
  no-match hint already announces via its `role="status"` region.
  Reopen on the first screen-reader user request.

## Blocked (needs user decision)

| Task                                                    | Status       | Why blocked                                                                                                                                                                                                                                                                 | Evidence                                                  |
| ------------------------------------------------------- | ------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| CV-side adoption: bump go.mod, deploy, verify live page | 🔵 `BLOCKED` | Deploy pipeline + rollout order are the user's call (report g Q2). Verified 2026-09-10 (read-only): CV serves a CSP-safe mini-client, not the Datastar SDK — a safe bump needs `dashboard.WithNoDatastarRuntime()` (shipped in v0.7.0) or the filter/pill would render dead | CV `internal/di/health_dashboard.go` pins v0.6.1          |
| Copy affordance for raw check keys                      | 🔵 `BLOCKED` | Product decision (report g Q3): title-attr + select-text vs a Datastar clipboard action                                                                                                                                                                                     | status report 2026-09-09 question 3                       |
| Pin-guard keep sign-off                                 | 🔵 `BLOCKED` | Guard rewritten to v1.16.0 pins per the keep decision (fifth sweep caught 2026-09-10); the deviation from the original removal condition wants sign-off                                                                                                                     | `scripts/check-ui-pins.sh` header                         |
| Per-check latency metric labels (F5 remainder)          | 🔵 `BLOCKED` | go-health `Check` carries no per-check duration; unblocked when [go-health#2](https://github.com/LarsArtmann/go-health/issues/2) ships `Check.Since`/`Duration`                                                                                                             | `docs/upstream/go-health-check-timestamps-issue-draft.md` |
| Build-tag gating for SSE                                | 🔵 `BLOCKED` | Consumers who only want HTML shouldn't need GOEXPERIMENT=jsonv2. Requires decision: accept, fork go-sse, or gate.                                                                                                                                                           | `ROADMAP.md` Open Questions                               |
| Fingerprint format stability                            | 🔵 `BLOCKED` | Length-prefix fix changed fingerprint values; documented as accepted in CHANGELOG pending a versioning decision.                                                                                                                                                            | `ROADMAP.md` Open Questions                               |

Everything else from the v0.3.x–0.7.0 cycles either shipped (see
`CHANGELOG.md` `[0.7.0]` for the 2026-09-10 session: v1.16.0 pin
ceremony + axe tolerance retirement, PersistCollapse patch-survival fix,
introspection completeness, NDJSON export, `WithGrouping(BySource)`,
`WithNoDatastarRuntime`, example toggles, the RetryAlways×503 interplay
test, golden renders, the load-test harness with recorded numbers, and the
upstream go-health#2 filing), was closed with a reason in the annotated
reports under `docs/status/` (fully-executed reports move to `archived/`),
or lives in ROADMAP.md as raw ideas. v0.7.0 was released 2026-09-10
(tagged, pushed, proxy-verified, GitHub Release published; CI fully
green including version-guard). Session-closing gate:
`scripts/pre-push-checks.sh`. Known-broken-commit SHAs for
`git bisect skip`: see AGENTS.md and
`docs/status/archived/2026-09-04_19-15_bisectability-audit.md`.
