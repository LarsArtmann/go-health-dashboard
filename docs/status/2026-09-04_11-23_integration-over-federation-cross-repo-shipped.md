# Status Report — Integration over Federation, Cross-Repo (go-health v0.1.0 + dashboard v0.5.0)

**Date**: 2026-09-04 11:23
**Scope of this report**: the 2026-09-03/04 session that pivoted both repos from
"federation ambitions" to "integrate, don't compete", shipped the `aggregate` package,
the `Prober` interface, webhook push, the integration cookbook, and two releases.
**Plan of record**: `docs/planning/archived/2026-09-04_00-14_integration-over-federation.md`

---

## Headline

Two releases shipped, verified from a clean consumer: **go-health v0.1.0** (aggregate
sub-package) and **go-health-dashboard v0.5.0** (Prober interface, webhook push,
cookbook). All gates green: build, full tests, race, vet, lint (0 issues), proxy
`go get` + `go mod verify` from a fresh module. Nothing is broken.

---

## a) FULLY DONE

| Item                                 | Detail                                                                                                                                                                                                                                                                             |
| ------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Strategy decision                    | Integrate (SigNoz/Gatus/Kuma/Grafana) instead of building federation; multi-service = `aggregate` in-process only. Federation and status-page mode explicitly rejected and documented.                                                                                             |
| go-health `aggregate` package        | `Source{Name, Probe}`, `New` validation (empty/dup/nil), merge-on-read `CachedResponse` (worst-of, `source/check` keys, shutdown overlay, max latency), `RefreshInterval` (slowest), `StartupComplete` (AND latch), liveness/readiness/startup handlers, `RegisterRoutes`.         |
| aggregate test suite                 | Construction rejections, worst-of matrix (pass/warn/fail), shutdown-forces-fail, slowest-refresh, handler status codes, startup latch flow, RegisterRoutes content-type check.                                                                                                     |
| dashboard `Prober` interface         | Consumer-side interface; `New`, `Register`, `resolvePushInterval` switched; proven source-compatible (entire existing suite passes untouched) + stub-prober test.                                                                                                                  |
| `WithWebhook` / `WithWebhookHeaders` | Change-only (independent of PushMode), initial-state announce, 10s timeout, bounded in-flight (8), best-effort silent, public-mode masking (`check-N`, no error strings), secret-never-logged. Tests: announce+transition, headers/content-type, silence-when-unchanged, masking.  |
| Aggregate e2e test                   | 2 injectors → probes → aggregate → dashboard HTML renders `api/dependency` + `web/dependency`, worst-of warn banner, `/readyz` 200-on-warn verified.                                                                                                                               |
| Integration cookbook                 | `docs/integrations.md`: semantics table, Prometheus/SigNoz PromQL incl. `target=0` and `{{$value}}` traps, Gatus YAML + Nix snippet, Kuma config, webhook payload contract + security notes.                                                                                       |
| Cross-repo docs                      | go-health: CHANGELOG cut, AGENTS.md (aggregate architecture/design/gotchas, stale jsonv2 gotcha fixed), README "Aggregating Multiple Probes". dashboard: README (Integrations + Multi-Service sections), AGENTS.md (Prober, webhook, aggregate dependency notes), CHANGELOG 0.5.0. |
| Plan file                            | `docs/planning/archived/2026-09-04_00-14_integration-over-federation.md` — Pareto (1%/4%/20%), comprehensive + micro task tables, mermaid execution graph, frozen design contracts. Committed and pushed.                                                                          |
| Releases                             | go-health **v0.1.0** and dashboard **v0.5.0**: CHANGELOG-cut, annotated tags, pushed; consumer resolution verified in `/tmp` module (`go get` both, `go mod verify` clean).                                                                                                        |
| Verification gates                   | Both repos: build ✓, full tests ✓, race ✓, vet ✓, golangci-lint 0 issues ✓.                                                                                                                                                                                                        |

## b) PARTIALLY DONE

| Item                  | What exists                                           | What's missing                                                                                                                                             |
| --------------------- | ----------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Release lifecycle     | Tags + pushes + CHANGELOG cuts                        | GitHub Releases pages (`gh release create`) never created for v0.1.0/v0.5.0; pkg.go.dev doc-fetch not triggered; post-push CI status not re-checked.       |
| Webhook observability | Silent-by-design drop policy                          | Zero operator signal: no delivery counters/duration in `/health/metrics`, no debug hook. "It didn't arrive" is undebuggable today.                         |
| Cookbook validation   | Recipes written from real semantics + SystemNix scars | Nothing wired into the real Gatus/SigNoz stack yet; the Nix `mkHttpCheck`-style snippet is illustrative, not a tested `lib/go-health.nix`.                 |
| Docs hygiene          | CHANGELOG/AGENTS/README updated                       | `FEATURES.md` + `TODO_LIST.md` NOT updated in either repo (both exist). dashboard CHANGELOG has no empty `[Unreleased]` placeholder section.               |
| Performance claims    | "Merge-on-read is cheap" asserted in docs             | No benchmark for `aggregate.CachedResponse` (1/5/20 sources) or `webhook.buildPayload`.                                                                    |
| Plan vs execution     | Plan file frozen pre-code (design contracts held)     | Plan never updated for the in-flight scope addition (dashboard v0.5.0 release was not in the micro-task table); no execution log/tick-off in the plan doc. |

## c) NOT STARTED

| Item                                              | Notes                                                                                                       |
| ------------------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| `WithGrouping(BySource)` view option              | Per-service cards from namespaced keys; severity grouping still renders aggregates fine.                    |
| SystemNix `lib/go-health.nix` generator           | Consumer side; collapses hand-written Gatus blocks.                                                         |
| `go-health-otel` OTLP module                      | Deferred until SigNoz-native demand; Prometheus path covers SigNoz today.                                   |
| `/health/monitor.json` manifest                   | Rejected for now (permanent API surface vs marginal value).                                                 |
| statuspage.io / PagerDuty / Slack / ntfy adapters | Consumer-land payload mappers; docs snippets only, not written.                                             |
| Grafana dashboard JSON panel in docs              | Not started.                                                                                                |
| Fuzz targets for new code                         | No fuzz for webhook payload marshal or aggregate merge (repo has existing fuzz pattern to follow).          |
| Example server demo update                        | `nix run .#example` still demos single probe; no aggregate/webhook demo mode.                               |
| Browser test with aggregate page                  | CSP-clean runtime check exists for single probe; not run against a rendered aggregate page.                 |
| First real webhook consumer                       | PapDashboard ingest has not been switched from the fragile Gatus custom-provider template to `WithWebhook`. |

## d) TOTALLY FUCKED UP

Nothing shipped is broken — all gates green, releases verified. But two process
failures were real:

1. **I violated "read before you write" on go-health's Gotchas.** The
   samber/do lazy-invoke trap (never-invoked services health-check as pass) was
   documented in go-health's AGENTS.md:87 _before_ I wrote the aggregate tests — I
   hit the failure, burned a debug cycle building a throwaway probe test, and only
   then found the gotcha documented. Cost: one build-fail + one failing-suite round
   trip. Lesson: read the target repo's Gotchas section, not just its architecture,
   before writing tests.
2. **First-draft sloppiness that gates had to catch**: `fmt.Copy` (nonexistent API),
   a hand-rolled `contains` helper next to `strings.Contains`, a test awaiting a
   third webhook delivery that change-only logic correctly never sends, and
   `wsl`/`mnd` lint churn across three lint rounds. Each was caught and fixed, but
   the round trips were avoidable with slower first drafts.

## e) WHAT WE SHOULD IMPROVE

| Improvement                                                                 | Why                                                                                                                                                                      |
| --------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Read target-repo Gotchas before writing its tests                           | Both failures this session came from documented-but-unread gotchas.                                                                                                      |
| Commit deliberate milestones before the auto-commit daemon slices them      | 8 heuristic commits in dashboard, 3 in go-health fragment the narrative; the detailed release messages only cover the tail. Stage + commit per logical unit immediately. |
| Keep webhook failure semantics consistent with "best user feedback"         | Silent drop is a defensible library default, but ship the observability (metrics counters) in the same release as the feature, not later.                                |
| Add fuzz/benchmark coverage in the same change as new parse/merge hot paths | Repo already has the pattern; new code skipped it.                                                                                                                       |
| Update the plan file when scope grows mid-execution                         | v0.5.0 release happened outside the plan; the plan should stay the single truthful record.                                                                               |
| Run the full doc surface (FEATURES/TODO_LIST) at release time               | CHANGELOG alone under-reports; docs-health HARVEST/VERIFY exists for this.                                                                                               |
| Check CI after pushing tags                                                 | A red tag-CI run is frozen forever (go-release Phase 4.4/6).                                                                                                             |

## f) Next — up to 50 items (brainstorm, sorted roughly by impact; most are ROADMAP fuel)

| #  | Item                                                                                                                                                                            | Repo      |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- |
| 1  | Wire the first real Lars-stack service into Gatus + SigNoz using the cookbook (pilot validation)                                                                                | infra     |
| 2  | SystemNix `lib/go-health.nix` generator consuming cookbook semantics                                                                                                            | SystemNix |
| 3  | Webhook delivery metrics: `dashboard_webhook_deliveries_total{result}` + duration histogram behind `WithMetrics`                                                                | dashboard |
| 4  | Retroactive GitHub Releases for v0.1.0 and v0.5.0 from CHANGELOG sections                                                                                                       | both      |
| 5  | Verify post-push CI green on both tags; fix red if found                                                                                                                        | both      |
| 6  | Trigger + review pkg.go.dev docs for `aggregate` package                                                                                                                        | go-health |
| 7  | HMAC request signing: `WithWebhookSecret` → `X-Signature` header                                                                                                                | dashboard |
| 8  | Payload schema version field (`"schema":1`) before external consumers appear                                                                                                    | dashboard |
| 9  | Fuzz targets: webhook payload marshal, aggregate merge (follow `fuzz_test.go` pattern)                                                                                          | dashboard |
| 10 | Benchmarks: `aggregate.CachedResponse` at 1/5/20 sources; `buildPayload`                                                                                                        | go-health |
| 11 | `WithGrouping(BySource)`: per-service cards by splitting namespaced keys                                                                                                        | dashboard |
| 12 | Switch PapDashboard ingest from Gatus custom-provider template to `WithWebhook`                                                                                                 | infra     |
| 13 | Update `FEATURES.md` + `TODO_LIST.md` in both repos (aggregate, webhook, Prober)                                                                                                | both      |
| 14 | Add empty `[Unreleased]` placeholder to dashboard CHANGELOG                                                                                                                     | dashboard |
| 15 | docs-health HARVEST: route this report's section (f) into TODO_LIST/ROADMAP                                                                                                     | dashboard |
| 16 | Example server: aggregate + webhook demo mode (`nix run .#example`)                                                                                                             | dashboard |
| 17 | Browser test: CSP-clean runtime for an aggregate-rendered page                                                                                                                  | dashboard |
| 18 | Kuma section of cookbook validated against a live Kuma instance                                                                                                                 | docs      |
| 19 | Ship a Grafana dashboard JSON (status panel + per-check) in docs/                                                                                                               | dashboard |
| 20 | Prometheus alert rules starter pack (rules file, not just PromQL snippets)                                                                                                      | dashboard |
| 21 | Regenerate README screenshots (new sections exist; `screenshot_test` env-guarded)                                                                                               | dashboard |
| 22 | `nix flake check` both repos (not run this session)                                                                                                                             | both      |
| 23 | govulncheck + gosec over new code                                                                                                                                               | both      |
| 24 | CI: assert `GOEXPERIMENT=jsonv2` env explicitly in workflows for both repos                                                                                                     | both      |
| 25 | Webhook: configurable retry (attempts + backoff), demand-driven                                                                                                                 | dashboard |
| 26 | Aggregate staleness surface: per-source last-refresh age (needs go-health timestamp API)                                                                                        | go-health |
| 27 | Document wire-shape difference: aggregate liveness omits `uptime`/`version` that Probe liveness includes                                                                        | go-health |
| 28 | Verify trend/export endpoints with an aggregate source (logic is source-agnostic; test it anyway)                                                                               | dashboard |
| 29 | e2e: aggregate with an empty-checks source renders sanely                                                                                                                       | dashboard |
| 30 | statuspage.io / ntfy / Slack adapter examples in cookbook (consumer-land snippets)                                                                                              | docs      |
| 31 | `promtool check rules` (or lint) for cookbook PromQL if tooling available                                                                                                       | docs      |
| 32 | AGENTS.md cross-link: "one project split in two" note in both repos                                                                                                             | both      |
| 33 | go-health ROADMAP: mark federation/rejected items with rationale pointers to the plan doc                                                                                       | go-health |
| 34 | Decide GET-only guard parity for aggregate handlers (Probe has `WithGETOnly`; aggregate doesn't)                                                                                | go-health |
| 35 | Consider `go-health` README install snippet bump to v0.1.0 (check pinned versions in README)                                                                                    | go-health |
| 36 | dashboard README: pin example imports to v0.5.0 in Quick Start                                                                                                                  | dashboard |
| 37 | Webhook headers: document Go canonicalization (`authorization` → `Authorization`)                                                                                               | dashboard |
| 38 | Load test: 20-source aggregate under concurrent SSE + scrape                                                                                                                    | dashboard |
| 39 | Mark executed steps in the plan doc (execution log)                                                                                                                             | dashboard |
| 40 | Docker example image rebuild check (recent commit added Docker setup)                                                                                                           | dashboard |
| 41 | Retraction safety: none needed, but record tag→commit mapping in CHANGELOG footers                                                                                              | both      |
| 42 | Review `WithTrend` sample values against aggregate worst-of (warn=0.5) for multi-service semantics                                                                              | dashboard |
| 43 | Dashboard version constant → also surface in JSON `/health`? (currently Response.Version comes from probe; aggregate yields empty — decide what multi-service page should show) | both      |
| 44 | Deprecated-tag risk: none; document tag discipline in CONTRIBUTING                                                                                                              | both      |
| 45 | Integration test: webhook against HTTP/1.0-style picky receiver (Go client default is fine; skip unless a consumer appears)                                                     | dashboard |
| 46 | `aggregate`: option to include per-source headers in Readiness body? (probably reject — document why)                                                                           | go-health |
| 47 | Cookbook: add "Kubernetes probe wiring for aggregate" section                                                                                                                   | docs      |
| 48 | Split cookbook into per-platform pages if it grows                                                                                                                              | docs      |
| 49 | Cross-repo e2e in CI: build dashboard against released go-health (catch drift early)                                                                                            | both      |
| 50 | Postmortem the daemon-commit fragmentation into a session-convention note in AGENTS.md                                                                                          | both      |

## g) Questions I cannot answer myself

1. **Webhook hardening appetite**: is `Authorization`-header auth enough for your
   ingests, or do you want HMAC request signing (`WithWebhookSecret` → `X-Signature`)
   and an explicit payload `"schema":1` version field now, before any external
   consumer freezes the format?
2. **Pilot service**: which Lars-stack service should be the first wired into Gatus +
   SigNoz via the cookbook? (You know which service hurts most; the 2283-line
   `gatus-config.nix` doesn't say which endpoints you care about most.)
3. **Multi-service UI priority**: for the aggregate page, is per-service card grouping
   (`WithGrouping(BySource)`) the feature that would make it a daily driver, or is
   severity-grouped with `source/check` names already enough for how you'd use it?
