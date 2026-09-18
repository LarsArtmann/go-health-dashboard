# TODO List

> Short-term, actionable, bounded work items, verified against the actual
> code (docs-health HARVEST passes 2026-09-03, 2026-09-04, 2026-09-10,
> 2026-09-17 morning, and 2026-09-17 evening — closed items live in
> `CHANGELOG.md`, never here). For long-term vision and unrefined ideas,
> see ROADMAP.md.

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                               |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed. |

## Next Up

Harvested 2026-09-17 (morning docs-health pass + evening post-v0.9.0
batch executed by two parallel sessions). Ground truth:
`Version = "0.9.0"`, go-health v0.2.0, 281 top-level test/benchmark/fuzz
functions across 39 test files
(`rg -c '^func (Test\|Benchmark\|Fuzz)' *_test.go`).

| Task                                             | Status    | Why it matters / notes                                                                                                                                                        | Evidence                                                     |
| ------------------------------------------------ | --------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------ |
| CHANGELOG per-bullet tag audit for v0.3.0–v0.8.0 | 🔴 `TODO` | The 2026-09-17 audit verified the `[0.1.0]`/`[0.2.0]` claims against their tags (all TRUE — including dark mode and connection limits, contrary to the old stray-bullets premise), fixed the `[0.1.0-alpha]` scope note (its link target `01277d3` is the bare module-init commit), added the missing `[0.8.0]`/`[0.8.1]`/`[0.9.0]` link defs, re-pointed `[Unreleased]` to `v0.9.0...HEAD`, and normalized heading dashes. The remaining sections' bullets are spot-checked only, not diffed feature-by-feature against each tag. | CHANGELOG alpha audit note + footer; `git grep`/`git cat-file` tag checks |

Shipped since the last sweep (2026-09-10): v0.8.0 + v0.8.1 (evidence strip,
mobile stacking, WCAG AA, single bootstrap, templ-components v1.17.0,
verify-release.sh, changelog lint, release-lesson codification), the
BuildFlow red→green triage (erraudit 13→0, three skip gates with rationale,
treefmt snapshot-race root cause + templ-generate skip), the go-health
v0.2.0 consumer upgrade (per-check since/duration UI + the new duration
gauge), and v0.9.0 (2026-09-17: signed tag, tag-first push,
`verify-release.sh v0.9.0` 9/9 green, vulncheck clean, coverage 82.0% vs
the 78% floor, browser suite 15/15 — Latest).

Shipped in the 2026-09-17 evening post-v0.9.0 batch (two parallel
sessions, per the pareto plan's 4% + 20% tiers): the metadata test-gap
proofs (SSE patch payloads carry the metadata line and the zero-proven
evidence-strip warning — connect-time AND broadcast; aggregate transit
with namespaced keys; latency-histogram bucket/bounds pairing at the
value level), `/health/export` JSON per-check `checks` object, evidence
tooltips citing the probe-side `Check.Since` stamp, fuzz targets for the
two duration formatters with hostile seeds plus clean 60s campaigns on
four targets (results in `docs/research/2026-09-10_benchmarks.md`
alongside the stamping-loop benchmark: ~0.21µs/6 allocs per row),
regenerated light/dark/degraded screenshots (all three eyeballed:
metadata line + evidence strip visible), golden fixtures for the
zero-proven warning and public-mode render (dark mode needs none —
CSS-class-only theming), the dark-mode axe re-audit (one Chrome launch,
both themes, zero serious/critical findings), `DEMO_DETAILED=1` example
toggle, `scripts/verify-dep-bump.sh` (per-repo gate chain; the fleet-level
placement question stays open), a provisioned Grafana service in
`deploy/` (Prometheus datasource + 8-panel dashboard including the new
duration gauge; booted and verified against Grafana 11.5.2), and the
docs batch (SECURITY.md, README "Upgrading" + security link,
CONTRIBUTING guard-scripts section, ROADMAP v1.0 criteria, CHANGELOG
audit fixes), and the `protect-release-tags` ruleset on GitHub (id
23637848: `refs/tags/v*` deletion + non-fast-forward blocked, active,
bypass never — belt-and-suspenders under proxy immutability; not
live-probed because a working block makes the probe tag itself
undeletable). All rows closed here ship in `CHANGELOG.md` `[Unreleased]`.

Everything else from the v0.3.x–0.9.0 cycles either shipped (see
`CHANGELOG.md`), was closed with a reason in the annotated reports under
`docs/status/` (fully-executed reports move to `archived/` — 2026-09-17
docs-health pass), or lives in ROADMAP.md as raw ideas. v0.9.0 (2026-09-17,
commit `5f71fa6`) is Latest.
Session-closing gate: `scripts/pre-push-checks.sh`; dependency bumps now
have `scripts/verify-dep-bump.sh`. Known-broken-commit SHAs for
`git bisect skip`: see AGENTS.md and
`docs/status/archived/2026-09-04_19-15_bisectability-audit.md`.

## Blocked (needs user decision)

| Task                                                    | Status       | Why blocked                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     | Evidence                                                                                                 |
| ------------------------------------------------------- | ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| CV-side adoption: bump go.mod, deploy, verify live page | 🔵 `BLOCKED` | Deploy pipeline + rollout order are the user's call (report g Q2). Verified 2026-09-10 (read-only): CV serves a CSP-safe mini-client, not the Datastar SDK — a safe bump needs `dashboard.WithNoDatastarRuntime()` (shipped in v0.7.0) or the filter/pill would render dead                                                                                                                                                                                                                                                     | CV `internal/di/health_dashboard.go` pins v0.6.1                                                         |
| Copy affordance for raw check keys                      | 🔵 `BLOCKED` | Product decision (report g Q3): title-attr + select-text vs a Datastar clipboard action                                                                                                                                                                                                                                                                                                                                                                                                                                         | status report 2026-09-09 question 3                                                                      |
| Pin-guard keep sign-off                                 | 🔵 `BLOCKED` | Guard rewritten to v1.17.0 pins per the keep decision (sixth sweep caught 2026-09-16: the daemon swept the bump unguarded); the deviation from the original removal condition wants sign-off                                                                                                                                                                                                                                                                                                                                    | `scripts/check-ui-pins.sh` header                                                                        |
| Build-tag gating for SSE                                | 🔵 `BLOCKED` | Consumers who only want HTML shouldn't need GOEXPERIMENT=jsonv2. Requires decision: accept, fork go-sse, or gate.                                                                                                                                                                                                                                                                                                                                                                                                               | `ROADMAP.md` Open Questions                                                                              |
| Fingerprint format stability                            | 🔵 `BLOCKED` | Length-prefix fix changed fingerprint values; documented as accepted in CHANGELOG pending a versioning decision.                                                                                                                                                                                                                                                                                                                                                                                                                | `ROADMAP.md` Open Questions                                                                              |
| Evidence strip machine contract + persistence           | 🔵 `BLOCKED` | Two product decisions from the evidence-strip design (report g Q2/Q3): should proven/unproven counts surface in metrics + trend/export JSON (HTML-only contract today), and should the observation window persist across restarts (opt-in store) instead of resetting?                                                                                                                                                                                                                                                          | `docs/status/2026-09-10_03-24_evidence-strip-health-washing-status.md` §g; `ROADMAP.md` Open Questions   |
| Unskip go-structure-linter in BuildFlow                 | 🔵 `BLOCKED` | BuildFlow pins go-structure-linter v0.10.0, whose SDK predates project-config support — the committed `.go-structure-linter.yaml` (`flat` preset for this deliberately flat root-package library) is inert, so the step is skipped in `.buildflow.yml`. Unskip when BuildFlow pins a release where `Lint` calls `LoadProjectConfig`.                                                                                                                                                                                            | go-structure-linter v0.10.0 `pkg/sdk/sdk.go` (no `LoadProjectConfig`) vs current checkout `sdk.go:132`   |
| Unskip branching-flow in BuildFlow                      | 🔵 `BLOCKED` | The step gates on 24 PHANTOM_TYPE findings spanning the public option API (breaking redesign → versioning decision) and 2 BOOL_BLIND bit-flag demands; no scoping exists (`//nolint` ignored by phantom/boolblind, `.branching-flow.yml` has no rule exclusion, BuildFlow's provider hardcodes `analysis.RunAll`). Unskip after a fleet decision: per-project rule config in branching-flow honored through BuildFlow, severity tuning, or the phantom-type adoption in a major version.                                        | `.buildflow.yml` skip rationale; branching-flow `pkg/core/ignore_comments.go` consumers (roleak/do only) |
| Unskip templ-generate in BuildFlow (fleet DAG ordering) | 🔵 `BLOCKED` | BuildFlow lets nix-evaluating steps run concurrently with tree-mutating generators; a nix `git+file` source snapshot ingested during templ's raw-output window made treefmt-check fail on a tree state that no longer existed (verified 2026-09-16: files formatted at 15:19:12, check still failed at 15:19:57 on the stale snapshot). Skipped locally; unskip when BuildFlow orders all generators before all nix-evaluating steps. Also decide upstream-file vs local-patch (report question 3). | `/tmp/bf-final.log` timeline; `.buildflow.yml` skip rationale; status report 2026-09-16 §c               |
