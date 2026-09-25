# TODO List

> Short-term, actionable, bounded work items, verified against the actual
> code. Closed items live in `CHANGELOG.md`, never here. For long-term
> vision and unrefined ideas, see ROADMAP.md.

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                               |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed. |

## Next Up

Swept 2026-09-23 (full-list execution session): every row harvested
2026-09-18/22 was executed, verified, closed into CHANGELOG, or
re-triaged — nothing is left in the 🔴 queue. Ground truth re-verified
against the tree that day: `Version = "0.10.1"`, go-health v0.4.0,
templ-components v1.18.0, 297 top-level test/benchmark/fuzz functions
across 42 root test files (+2 in `cmd/health-hub`) — enforced by the
FEATURES-count drift guard and `pre-push-checks.sh` (now 6 checks incl.
the fuzz-registry diff and the templ tool-pin grep; CI adds shellcheck).

| Task                | Status      | Why it matters / notes                                                                                  | Evidence                                      |
| ------------------- | ----------- | ------------------------------------------------------------------------------------------------------- | --------------------------------------------- |
| ~~Entire 2026-09-18/22 queue~~ | ✅ closed 2026-09-23 | Boot e2e, deploy README, tag-protection script, verify-dep-bump hardening (×3), CHANGELOG tag audit, `shutting_down:true` pin, source-mode golden, M88/M89/M91/M92/M103/M105/M106/M107, mobile visual pass, dark contrast, aggregate load re-run, premise triage, CI artifact inspection, ROADMAP v1.0 sweep, fuzz registry + templ pins gates, digest pins + healthchecks + alert example, metrics reference, upstream verification, `HEALTH_HUB_PUSH_INTERVAL` | `CHANGELOG.md` `[Unreleased]`; `docs/status/2026-09-23_19-30_changelog-tag-audit.md`; `docs/research/` (M91 + load re-runs + dark contrast); `docs/upstream/` (verified drafts) |

Open follow-ups noticed during the sweep (bounded, low priority):

| Task                                                                          | Status      | Why it matters / notes                                                                                                     | Evidence                                        |
| ----------------------------------------------------------------------------- | ----------- | -------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------- |
| File the two verified upstream drafts (templ gofumpt output; erraudit nolint-audit) | 🔵 `BLOCKED` | Verification done, drafts ready in `docs/upstream/`; filing is a remote action awaiting authorization. The third claim (erraudit `strings.Cut` FP) was REFUTED on erraudit 1c6809a and must not be filed — the real erraudit bug is nolint-audit's NEEDED verdict on a directive that suppresses nothing | `docs/upstream/*.md` |
| The auto-daemon still wins the edit→commit race regularly                     | 🔴 `TODO`   | 6 intent commits lost to heuristic sweeps in the 09-23 session alone; harness-level fix (commit at first green IN THE SAME tool call) is behavioral, not codeable | `git log --grep 'chore: auto-commit'` around any session |

Everything else from the v0.3.x–v0.10.x cycles either shipped (see
`CHANGELOG.md`), was closed with a reason in the annotated reports under
`docs/status/` (fully-executed reports move to `archived/`), or lives in
ROADMAP.md as raw ideas. v0.10.1 (2026-09-22) is Latest.
Session-closing gate: `scripts/pre-push-checks.sh`; dependency bumps have
`scripts/verify-dep-bump.sh`; the templ-components v1.18.0 sweep
(2026-09-18) was adopted with the pin-guard update and a green browser
suite. Known-broken-commit SHAs for `git bisect skip`: see AGENTS.md and
`docs/status/archived/2026-09-04_19-15_bisectability-audit.md` (extended
2026-09-23: 19 carriers + the go-directive-sweep class).

## Blocked (needs user decision)

Premises re-verified against the 2026-09-23 tree (triage row executed);
rows whose premise dissolved were closed, not carried.

| Task                                                    | Status       | Why blocked                                                                                                                                                                                                                                                                                                                                                                 | 2026-09-23 premise check                        |
| ------------------------------------------------------- | ------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------- |
| CV-side adoption: bump go.mod, deploy, verify live page | 🔵 `BLOCKED` | Deploy pipeline + rollout order are the user's call. CV `go.mod:31` still pins v0.6.1 (re-verified); a safe bump needs `dashboard.WithNoDatastarRuntime()` (shipped v0.7.0) or the filter/pill would render dead                                                                                                                                                             | HOLDS                                           |
| Copy affordance for raw check keys                      | 🔵 `BLOCKED` | Product decision: title-attr + select-text vs a Datastar clipboard action                                                                                                                                                                                                                                                                                                    | HOLDS (no code changed either way)              |
| Pin-guard keep sign-off                                 | 🔵 `BLOCKED` | Guard maintained through the seventh sweep (v1.18.0, 2026-09-18); the deviation from the original removal condition still wants formal sign-off — `scripts/check-ui-pins.sh` header unchanged, pins v1.18.0/v0.5.0 verified green today                                                                                                                                      | HOLDS                                           |
| Build-tag gating for SSE                                | 🔵 `BLOCKED` | Consumers who only want HTML shouldn't need GOEXPERIMENT=jsonv2. Requires decision: accept, fork go-sse, or gate. go-sse v0.6.0 + jsonv2 still in the module graph (re-verified)                                                                                                                                                                                              | HOLDS                                           |
| Fingerprint format stability                            | 🔵 `BLOCKED` | Length-prefix fix changed fingerprint values; documented as accepted in CHANGELOG (§0.4.0) pending a versioning decision                                                                                                                                                                                                                                                     | HOLDS                                           |
| Evidence strip machine contract + persistence           | 🔵 `BLOCKED` | Should proven/unproven counts surface in metrics + trend/export JSON (HTML-only contract today — re-verified: no evidence fields in metrics.go/trend.go/webhook.go), and should the observation window persist across restarts (opt-in store) instead of resetting?                                                                                                           | HOLDS                                           |
| Compose-stack CI policy                                 | 🔵 `BLOCKED` | Manual-only demo (now DOCUMENTED + once boot-verified manually), build-only CI job, or full boot+probe CI? Costs CI minutes and Docker-in-CI surface. Note: the e2e boot caught a real Dockerfile break on its first manual run, which argues for at least a build-only job                                                                                                    | HOLDS, evidence strengthened                    |
| Tag-ruleset bypass policy                               | 🔵 `BLOCKED` | `protect-release-tags` live with `bypass: never` — re-verified byte-for-byte via `set-tag-protection.sh --check` (id 23637848). Keep `never`, or add repository-admin as a bypass actor for emergency re-tags?                                                                                                                                                                | HOLDS                                           |
| Enable private vulnerability reporting                  | 🔵 `BLOCKED` | SECURITY.md references PVR (re-verified); repo API shows no PVR enablement surface without admin authorization. Repo-admin action.                                                                                                                                                                                                                                            | HOLDS                                           |
| Unskip go-structure-linter in BuildFlow                 | 🔵 `BLOCKED` | BuildFlow's pinned go-structure-linter still predates project-config support; the `.buildflow.yml` skip (with the `flat` preset committed) remains — unskip when BuildFlow pins a release whose `Lint` calls `LoadProjectConfig`                                                                                                                                              | HOLDS                                           |
| Unskip templ-generate in BuildFlow (fleet DAG ordering) | 🔵 `BLOCKED` | Still skipped in `.buildflow.yml`: BuildFlow lets nix-evaluating steps run concurrently with tree-mutating generators; unskip when BuildFlow orders all generators before all nix-evaluating steps; also decide upstream-file vs local-patch                                                                                                                                  | HOLDS                                           |
| ~~Unskip branching-flow in BuildFlow~~                  | ✅ `CLOSED`  | Overtaken: UNSKIPPED 2026-09-22 under the branching-flow v0.6.2+ SDK severity cap — findings land as warnings/observations, gate stays green, real-bug analyzers (panic/split-brain/ro-leak) are back. Design disagreement recorded in `.buildflow.yml`                                                                                                                       | premise dissolved                               |
| ~~Toolchain bump go1.26.7 → 1.26.8 via verify-dep-bump.sh~~ | ✅ `CLOSED` | Overtaken by events: go.mod's floor is `go 1.27.1` (dep-driven, 2026-09-19) and CI's matrix pins 1.27.1 — there is no 1.26.x alignment left to do; `check-go-directive.sh` guarded the floor (and caught the live 09-23 sweep). RESOLVED 2026-09-25: go-health v0.4.1 lowered its floor to minor form, the directive settled at `go 1.27` via go-version-auto-configure's fix (gate-verified), `who-forces` reports clean, and the guard was deleted as obsolete — the fleet filing question about go-mod-normalize downgrades is closed fixed-upstream (gvac v0.2.0+ dep-floor gate) | stale                                           |
