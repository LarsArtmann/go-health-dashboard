# Status Report: samber/do v2 Integration Improvement

**Date:** 2026-08-09 04:41\
**Session Goal:** "How can we better leverage samber/do v2?"\
**Verifier:** Self-review (brutal)

---

## What Was Done

### Problem Statement

The dashboard library depended on `samber/do/v2` (required by go-health for service discovery) but was completely decoupled from it internally. The `Dashboard` struct held a pusher goroutine and SSE broadcaster — resources with real lifecycle — yet implemented none of samber/do's lifecycle interfaces. The example had a DO-2 anti-pattern (`do.New()` without `do.Shutdown()`). There was zero convenience for DI-integrated consumers.

### Changes Made (5 files touched, 2 new)

| File                      | Change                                                                                                                                      | Lines       |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- | ----------- |
| `dashboard.go`            | Added `HealthCheck(ctx) error` method, compile-time `do.HealthcheckerWithContext` + `do.Shutdowner` assertions, imported `errors` + `do/v2` | +22         |
| `di.go` (new)             | `Register(injector, probe, opts...)` — creates Dashboard + `do.ProvideValue`                                                                | +35         |
| `example/main.go`         | Fixed DO-2, switched to `Register`, added SIGINT/SIGTERM graceful shutdown                                                                  | ~30 changed |
| `lifecycle_test.go` (new) | 9 tests: HealthCheck states, Register cascades, idempotent shutdown, interface satisfaction                                                 | +140        |
| `AGENTS.md`               | Updated file listing, design decisions, data flow, testing patterns, dependency notes                                                       | ~40 changed |

### Test Results

```
go build ./...        — PASS
go test -race ./...   — PASS (all 9 new tests + all pre-existing tests)
```

---

## a) FULLY DONE

1. **`HealthCheck(ctx) error` method** — Returns error when pusher not active (not started or shut down). Fast atomic read, no blocking.
2. **Compile-time lifecycle assertions** — `var _ do.HealthcheckerWithContext = (*Dashboard)(nil)` and `var _ do.Shutdowner = (*Dashboard)(nil)`.
3. **`Register(injector, probe, opts...)` convenience function** — Zero-boilerplate DI registration via `do.ProvideValue`.
4. **Example rewritten** — DO-2 fixed (`defer injector.Shutdown()`), uses `Register`, graceful HTTP server shutdown on SIGINT/SIGTERM, proper deferred cleanup ordering.
5. **9 lifecycle tests** — All pass with race detector. Cover not-started, started, post-shutdown, cancelled-context, container cascade, idempotent, interface satisfaction, same-instance.
6. **AGENTS.md updated** — File listing, design decision, data flow step 8, testing patterns, samber/do dependency notes section.

---

## b) PARTIALLY DONE

### `HealthCheck` error is not a sentinel

Used `errors.New("dashboard: SSE pusher is not active")` inline. Consumers cannot programmatically distinguish this error from others with `errors.Is`. Should be:

```go
var ErrPusherNotActive = errors.New("dashboard: SSE pusher is not active")
```

### Example still uses `log.Fatalf` in `registerService`

The `registerService` helper calls `log.Fatalf` on `do.InvokeNamed` failure. This is technically acceptable (setup code in `main`), but a `Register`-style approach returning errors would be cleaner and more testable. Left as-is to keep the example focused on the Dashboard changes.

### doc.go not updated

The package doc comment's Quick Start example still uses `dashboard.New(probe, ...)` + `defer dash.Shutdown()`. No mention of `Register` or the lifecycle interfaces. A second example block showing the DI-integrated path would be valuable.

---

## c) NOT STARTED

1. ~~**Sentinel error `ErrPusherNotActive`** — See (b).~~ done (done - ErrPusherNotActive shipped; family extended at db8621f)
2. ~~**doc.go Quick Start update** — See (b).~~ done at `db8621f`
3. ~~**Linter run** (`nix run .#lint`) — Never ran golangci-lint. Only ran `go build` + `go test -race`.~~ done (done - lint 0 issues; latest full run at ad2447e)
4. ~~**Formatter run** (`nix fmt`) — Never ran treefmt. Files may not conform to project formatting rules.~~ done (done - nix fmt/treefmt clean 2026-09-04)
5. ~~**`nix flake check`** — Never validated the flake.~~ done (done - nix flake check green 2026-09-04)
6. ~~**Benchmark for `HealthCheck`** — Trivially fast (atomic load) but no benchmark exists to prove it.~~ done (routed to ROADMAP benchmarks 2026-09-04)
7. ~~**Verify Dashboard self-monitoring behavior** — When registered via `do.ProvideValue`, go-health's Probe may discover the Dashboard as a health-checkable service (it implements `HealthcheckerWithContext`). This could cause the dashboard to appear in its own health table. Untested — could be a feature or a confusion.~~ done (routed to ROADMAP self-monitoring decision 2026-09-04)
8. ~~**`ShutdownerWithError` consideration** — `Dashboard.Shutdown()` calls `broadcaster.Close()` (instant). The broadcaster also has `Shutdown(ctx) error` (graceful drain). The current `do.Shutdowner` interface is correct for the instant path, but a context-aware graceful shutdown was not explored.~~ done (resolved by WithShutdownDrain (v0.4.0 8f63d85) - graceful drain ships in Shutdown)
9. ~~**`do.Package` grouping** — The skill recommends grouping related providers with `do.Package`. Not explored for the Dashboard.~~ done (routed to ROADMAP 2026-09-04)
10. ~~**Test for shutdown ordering in example** — The deferred shutdown order (probe → injector → cancel) has a brief window where the pusher goroutine is alive after the broadcaster is closed. This is safe (broadcasts after Close are silently dropped) but untested.~~ done (routed to ROADMAP shutdown-ordering test 2026-09-04)

---

## d) TOTALLY FUCKED UP

### `TestHealthCheck_ErrorIsDetectable` is a meaningless test

```go
if !errors.Is(err, err) {
    t.Fatal("errors.Is should return true for the same error")
}
```

`errors.Is(err, err)` is trivially true for every error in Go. This test asserts nothing useful. It should test `errors.Is(err, dashboard.ErrPusherNotActive)` — but that sentinel doesn't exist yet, which is why the test devolved into this.

**Fix:** Add `ErrPusherNotActive` sentinel, then rewrite the test to use `errors.Is`.

~~**Fix:** Add `ErrPusherNotActive` sentinel, then rewrite the test to use `errors.Is`.~~ done 2026-09-03 (docs-health pass): sentinel exists at `dashboard.go:668` and the test now asserts `errors.Is(err, ErrPusherNotActive)`.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture-Level

1. ~~**Sentinel errors** — All exported error-returning methods should use package-level sentinel errors or typed errors so consumers can handle them programmatically.~~ done (done - sentinel family ErrPusherNotActive/NotStarted/ShutDown (db8621f))
2. ~~**Consider `ShutdownerWithContext`** — For graceful SSE drain during shutdown. The broadcaster supports it; the Dashboard just doesn't expose it.~~ done (resolved by WithShutdownDrain (v0.4.0 8f63d85))
3. ~~**Dashboard self-monitoring** — When registered in the injector, the Dashboard may appear in its own health table. Decide: feature (the dashboard monitors its own SSE health) or filter it out (don't show the monitor in the monitored).~~ done (routed to ROADMAP self-monitoring decision 2026-09-04)
4. ~~**`do.Package` integration** — For consumers who want to `injector.Inject(dashboard.Package(probe, opts...))` as a single unit.~~ done (routed to ROADMAP 2026-09-04)

### Quality-Level

5. ~~**Run `nix run .#lint`** — Never ran the linter. May have lint violations.~~ done (done - lint 0 issues (latest ad2447e))
6. ~~**Run `nix fmt`** — Never formatted. Code may not match project style.~~ done (done - treefmt clean 2026-09-04)
7. ~~**Remove the meaningless test** — `TestHealthCheck_ErrorIsDetectable` should be rewritten or removed.~~ done (done in the 2026-09-03 docs-health pass - asserts ErrPusherNotActive now)
8. ~~**Add `doc.go` example for `Register`** — The DI-integrated path should be documented alongside the manual path.~~ done at `db8621f`

### Process-Level

9. ~~**Read the skill's references** — The skill has `references/samber-do-best-practices-report.md` and `references/anti-pattern-examples.md`. I loaded the SKILL.md but never read these. May have missed nuance.~~ **Won't implement — session-process note - the shipped integration passed subsequent review and gates.**
10. ~~**Test the example binary** — The example compiles but was never run to verify the dashboard actually renders and the graceful shutdown works end-to-end.~~ routed to ROADMAP 2026-09-04 docs-health pass (example-binary e2e run)

---

## f) NEXT 50 THINGS TO DO

### High Priority (do first)

1. ~~Add `ErrPusherNotActive` sentinel error in `dashboard.go`~~ done (sentinel exists (dashboard.go:668), extracted in the 2026-08-09 defect-fix session)
2. ~~Rewrite `TestHealthCheck_ErrorIsDetectable` to use `errors.Is(err, ErrPusherNotActive)`~~ done (fixed 2026-09-03 — test now asserts errors.Is against ErrPusherNotActive)
3. ~~Run `nix run .#lint` and fix all violations~~ done (golangci-lint 0 issues at HEAD; CI Lint job green 2026-09-03)
4. ~~Run `nix fmt` to format all new/changed files~~ done (treefmt enforced by nix flake check — clean)
5. ~~Run `nix flake check` to validate the flake~~ done (green (v0.3.x cycle and 2026-09-03))
6. ~~Update `doc.go` Quick Start with `Register` example~~ done at `db8621f`
7. ~~Verify whether Dashboard appears in its own health table when registered (test it)~~ routed to ROADMAP 2026-09-04 docs-health pass (self-monitoring)
8. ~~If it does appear, decide: keep (feature) or filter (confusing)~~ routed to ROADMAP 2026-09-04 docs-health pass (self-monitoring decision)

### Medium Priority

9. ~~Explore `ShutdownerWithContext` for graceful SSE drain~~ done (resolved as WithShutdownDrain option (3022fbf) — bounded drain inside Shutdown instead of a context-aware Shutdowner interface)
10. ~~Add benchmark: `BenchmarkHealthCheck`~~ routed to ROADMAP 2026-09-04 docs-health pass (benchmarks)
11. ~~Add benchmark: `BenchmarkRegister`~~ routed to ROADMAP 2026-09-04 docs-health pass (benchmarks)
12. ~~Consider `do.Package` wrapper for one-call injection~~ routed to ROADMAP 2026-09-04 docs-health pass (DI surface)
13. ~~Add integration test: full lifecycle via `Register` → `Start` → serve → `do.Shutdown`~~ done - lifecycle_test.go cascade coverage
14. ~~Add integration test: `do.HealthCheck[*Dashboard]` in a realistic container with other services~~ done - lifecycle_test.go container-cascade coverage
15. ~~Test the example binary end-to-end (start, curl `/health`, curl `/readyz`, send SIGTERM)~~ done (example v2 functionally smoke-tested over HTTP (401/200/metrics/probes) in the v0.3.x cycle)
16. ~~Document the shutdown ordering in the example (why probe before injector)~~ routed to ROADMAP 2026-09-04 docs-health pass (shutdown-ordering docs/test)
17. ~~Consider `WithInjector` option as alternative to `Register` (evaluate and dismiss or implement)~~ routed to ROADMAP 2026-09-04 docs-health pass (API ergonomics)
18. ~~Add `Provider` function (lazy variant using `do.Provide` instead of `do.ProvideValue`) — evaluate tradeoff~~ routed to ROADMAP 2026-09-04 docs-health pass (API ergonomics)
19. ~~Review whether `Register` should return `(*Dashboard, func())` for cleanup without injector~~ **Won't implement** - Register already hands lifecycle to the injector; a second cleanup path would duplicate ownership.
20. ~~Add godoc examples for `Register` and `HealthCheck`~~ routed to ROADMAP 2026-09-04 docs-health pass (doc.go examples)

### Documentation

21. ~~Update `FEATURES.md` with DI lifecycle integration as a feature~~ done (FEATURES row added 2026-09-03 (samber/do lifecycle under Configuration))
22. ~~Update `CHANGELOG.md` with the new `Register`, `HealthCheck`, lifecycle interfaces~~ done (v0.3.0 CHANGELOG section documents Register, HealthCheck, and the lifecycle interfaces)
23. ~~Add section to AGENTS.md about the `ShutdownReport` gotcha (always non-nil)~~ done (present in AGENTS.md samber/do dependency notes)
24. ~~Document that `HealthCheck` ignores context intentionally (fast atomic read)~~ routed to ROADMAP 2026-09-04 docs-health pass (docs)
25. ~~Add architecture decision record for "why `ProvideValue` not `Provide`"~~ routed to ROADMAP 2026-09-04 docs-health pass (ProvideValue-vs-Provide ADR)

### Testing Improvements

26. ~~Add test: `HealthCheck` concurrent calls (100 goroutines)~~ routed to ROADMAP 2026-09-04 docs-health pass (robustness tests)
27. ~~Add test: `Register` with nil injector panics with clear message~~ routed to ROADMAP 2026-09-04 docs-health pass (robustness tests)
28. ~~Add test: `Register` with nil probe panics with clear message~~ routed to ROADMAP 2026-09-04 docs-health pass (robustness tests)
29. ~~Add test: multiple `Register` calls on same injector (override behavior)~~ routed to ROADMAP 2026-09-04 docs-health pass (robustness tests)
30. ~~Add test: `do.Shutdown` cascade order (Dashboard before or after other services?)~~ routed to ROADMAP 2026-09-04 docs-health pass (robustness tests)
31. ~~Add test: `Register` + `Start` + `Shutdown` + `Start` again (restart after shutdown)~~ routed to ROADMAP 2026-09-04 docs-health pass (robustness tests)
32. ~~Add test: `HealthCheck` during active SSE connections (returns nil)~~ routed to ROADMAP 2026-09-04 docs-health pass (robustness tests)
33. ~~Add test: `SubscriberCount` after `Register` + `Start`~~ routed to ROADMAP 2026-09-04 docs-health pass (robustness tests)

### Code Quality

34. ~~Review `di.go` for naming — is `Register` the best name or should it be `RegisterDashboard`?~~ **Won't implement** - `Register` is the documented API name (README, AGENTS.md, doc.go).
35. ~~Consider whether `di.go` should be named `container.go` or `injector.go`~~ **Won't implement** - di.go is single-purpose; rename churn without value.
36. ~~Add `//go:generate` instruction if needed~~ **Won't implement** - no generate directives needed.
37. ~~Run `govulncheck` (`nix run .#vulncheck`) — never ran it~~ done (nix run .#vulncheck — no vulnerabilities 2026-09-03)
38. ~~Run `go vet` (`nix run .#vet`) — never ran it separately~~ done (go vet clean at HEAD)
39. ~~Check coverage: `nix run .#coverage` — measure lifecycle test coverage~~ done (coverage baseline 76.9% recorded 2026-09-03)
40. ~~Consider whether `HealthCheck` should also check probe health (not just pusher)~~ routed to ROADMAP 2026-09-04 docs-health pass (HealthCheck-probe aggregation)

### Future Features

41. ~~`WithInjector` option for auto-registration during `New`~~ routed to ROADMAP 2026-09-04 docs-health pass (DI surface)
42. ~~`Dashboard.RegisterNamed(injector, name, opts...)` for multiple dashboard instances~~ routed to ROADMAP 2026-09-04 docs-health pass (DI surface - RegisterNamed multi-instance)
43. ~~Health check aggregation: Dashboard.HealthCheck calls probe.HealthCheck~~ routed to ROADMAP 2026-09-04 docs-health pass (HealthCheck-probe aggregation)
44. ~~`do.Healthcheck` integration: expose pusher metrics (connection count, broadcast count)~~ routed to ROADMAP 2026-09-04 docs-health pass (pusher-metrics exposure)
45. ~~SSE connection health as a named service in the injector~~ routed to ROADMAP 2026-09-04 docs-health pass (DI surface - SSE health as a named service)
46. ~~Consider `samber-do-auditlog` integration for registration/shutdown observability~~ routed to ROADMAP 2026-09-04 docs-health pass (auditlog observability)
47. ~~Explore child scopes for per-tenant dashboard isolation~~ routed to ROADMAP 2026-09-04 docs-health pass (DI surface - child scopes)
48. ~~Add `Dashboard.Explain()` method using `do.Explain` output format~~ routed to ROADMAP 2026-09-04 docs-health pass (DI surface - Explain output)
49. ~~Consider provider function pattern: `func Provider(probe *health.Probe, opts ...Option) func(do.Injector) (*Dashboard, error)`~~ routed to ROADMAP 2026-09-04 docs-health pass (DI surface - provider-func pattern)
50. ~~Evaluate whether the Dashboard should implement `do.Healthchecker` (no-context variant) in addition to `HealthcheckerWithContext`~~ routed to ROADMAP 2026-09-04 docs-health pass (DI surface - do.Healthchecker variant)

---

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Should the Dashboard appear in its own health table?**
   **ROUTED 2026-09-04:** design decision parked on ROADMAP (self-monitoring decision, Pipeline/Testing theme). No behavior change shipped.

When registered via `do.ProvideValue`, go-health's Probe iterates the injector and may discover the Dashboard as a health-checkable service. This would make the dashboard monitor its own SSE pusher health — which is either a cool self-monitoring feature or a confusing recursive display. I cannot determine the intended UX without your input. (I can test this empirically, but the _design decision_ is yours.)

2. **Should `Shutdown()` expose graceful drain?**
   **ANSWERED:** shipped as the opt-in `WithShutdownDrain(d)` in v0.4.0 (`8f63d85`) — no breaking signature change; `Shutdown()` keeps its shape and drains only when configured.

The broadcaster has both `Close()` (instant, current) and `Shutdown(ctx) error` (graceful drain). Switching to graceful drain would require either changing `Shutdown()` to `Shutdown(ctx) error` (breaking API change) or adding a new method. I cannot decide this without knowing your API stability constraints for v0.2.x vs v0.3.0.

3. **Is `Register` the right abstraction, or should it be a `WithInjector` option?**
   **ANSWERED:** `Register` stayed (documented in README/doc.go/AGENTS.md); the `WithInjector` alternative is routed to ROADMAP (DI-surface ideas).

I chose a standalone function to keep `New()` decoupled from DI. But some consumers might prefer `dashboard.New(probe, dashboard.WithInjector(injector))` as a single-call API. This is a taste question I can't resolve without knowing your preferred integration style.
