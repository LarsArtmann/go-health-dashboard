# TODO List

> Short-term, actionable, bounded work items, verified against the actual
> code (docs-health HARVEST passes 2026-09-03 and 2026-09-04 — closed items
> live in `CHANGELOG.md`, never here). For long-term vision and unrefined
> ideas, see ROADMAP.md.

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                               |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed. |

## Next Up

### Release

| Task                                                                                                                                                 | Status    | Impact | Effort | Notes                                                                                                                                                                                                                                                                        |
| ---------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | ------ | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Cut v0.6.0: re-head CHANGELOG `[Unreleased]` (already written), bump `Version` in the same commit as the tag, push `--follow-tags`, verify the proxy | 🟢 `DONE` | High   | 30min  | Done 2026-09-04: commit `26b85c5`, annotated tag `v0.6.0`, `--follow-tags` push; proxy resolved (`go list -m @v0.6.0`); all 6 CI jobs green on their first real run (run 33916788241: Test, Browser CSP, Lint, Build, Version-guard, Vuln scan) + coverage artifact uploaded |
| Create GitHub Releases pages for v0.2.0–v0.5.0 from the CHANGELOG sections                                                                           | 🟢 `DONE` | Low    | 20min  | Done 2026-09-04: `gh release create` for v0.2.0, v0.3.0, v0.3.1, v0.4.0, v0.5.0, v0.6.0 — all six pages populated from CHANGELOG sections and verified (`gh release list`)                                                                                                   |
| 🆕 CI pin-guard: fail CI while templ-components ≠ v1.11.0 (until #7 lands)                                                                           | 🔴 `TODO` | High   | 30min  | New from the v0.6.0 cycle plan (`docs/planning/2026-09-04_22-27_v060-release-and-hardening-cycle.md` G1) — mechanical enforcement after the dep sweep landed twice on 2026-09-04                                                                                             |

### Features & polish

| Task                                                                                                                                                                             | Status    | Impact | Effort | Notes                                                                                                                                                                                                   |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | ------ | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Upstream PR to templ-components: StatCard `<dl>` fix (+ goldens); then remove the axe tolerance here                                                                             | 🔴 `TODO` | Low    | 60min  | templ-components#6 still open. Local side done: axe tolerance scoped to the StatCard signature (2026-09-04), so a fix upstream + bump retires it cleanly                                                |
| Upstream PR to templ-components: guard the LiveRegion busy-script `nonce=""` (issue #7); then bump off the v1.11.0 pin and re-validate the Datastar bundle via the browser suite | 🔴 `TODO` | Medium | 60min  | The v1.12.0 regression was re-verified live on 2026-09-04 (three failing CSP tests before the pin restore — `CHANGELOG.md` `[Unreleased]`); until the guard ships, UI-dep bumps stay blocked by the pin |

## Blocked (needs user decision)

| Task                         | Status       | Why blocked                                                                                                       | Evidence                    |
| ---------------------------- | ------------ | ----------------------------------------------------------------------------------------------------------------- | --------------------------- |
| Build-tag gating for SSE     | 🔵 `BLOCKED` | Consumers who only want HTML shouldn't need GOEXPERIMENT=jsonv2. Requires decision: accept, fork go-sse, or gate. | `ROADMAP.md` Open Questions |
| Fingerprint format stability | 🔵 `BLOCKED` | Length-prefix fix changed fingerprint values; documented as accepted in CHANGELOG pending a versioning decision.  | `ROADMAP.md` Open Questions |

Everything else from the v0.3.x cycle brainstorms, the 2026-09-04 sweep, and
the integration-pivot reports either shipped (see `CHANGELOG.md` [Unreleased]
and 0.5.0), was closed with a reason in the annotated reports under
`docs/status/` (fully-executed reports are moved to `archived/`), or lives in
`ROADMAP.md` as raw ideas. Known-broken-commit SHAs for `git bisect skip`:
see AGENTS.md and
`docs/status/archived/2026-09-04_19-15_bisectability-audit.md`.
