# TODO List

> Short-term, actionable, bounded work items, verified against the actual
> code (docs-health HARVEST pass 2026-09-03; sweep 2026-09-04 closed the
> v0.3.x backlog — closed items live in `CHANGELOG.md`, never here). For
> long-term vision and unrefined ideas, see ROADMAP.md.

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                              |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed. |

## Next Up

### Features & polish

| Task                                                                                                 | Status    | Impact | Effort | Notes                                                                                                |
| ---------------------------------------------------------------------------------------------------- | --------- | ------ | ------ | ---------------------------------------------------------------------------------------------------- |
| Upstream PR to templ-components: StatCard `<dl>` fix (+ goldens); then remove the axe tolerance here | 🔴 `TODO` | Low    | 60min  | templ-components#6 still open. Local side done: axe tolerance scoped to the StatCard signature (2026-09-04), so a fix upstream + bump retires it cleanly |

## Blocked (needs user decision)

| Task                     | Status       | Why blocked                                                                                                       | Evidence                    |
| ------------------------ | ------------ | ----------------------------------------------------------------------------------------------------------------- | --------------------------- |
| Build-tag gating for SSE | 🔵 `BLOCKED` | Consumers who only want HTML shouldn't need GOEXPERIMENT=jsonv2. Requires decision: accept, fork go-sse, or gate.  | `ROADMAP.md` Open Questions |
| Fingerprint format stability | 🔵 `BLOCKED` | Length-prefix fix changed fingerprint values; documented as accepted in CHANGELOG pending a versioning decision. | `ROADMAP.md` Open Questions |

Resolved 2026-09-04: "Next release version (v0.4.0 vs v0.3.2)" — v0.4.0 and
v0.5.0 shipped; the CI `version-guard` job now enforces const↔tag parity.

Everything else from the v0.3.x cycle brainstorms and the 2026-09-04 sweep
either shipped (see `CHANGELOG.md` [Unreleased] and 0.5.0), was closed with
a reason in the annotated reports under `docs/status/`, or lives in
`ROADMAP.md` as raw ideas.
