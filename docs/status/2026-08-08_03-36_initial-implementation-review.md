# Status Report — go-health-dashboard

**Date:** 2026-08-08 03:36
**Session:** Initial implementation (P1–P16 execution plan)
**Reporter:** Crush (self-review)

---

## A) FULLY DONE

### P1: go-health Accessors
- `Probe.CachedResponse() Response` — reads atomic cache, overlays shutdown flag (including when no cache exists)
- `Probe.RefreshInterval() time.Duration` — returns configured interval
- 6 tests in go-health, all passing
- AGENTS.md updated with new methods

### P2: Repo Scaffold
- `go.mod` with module, dependencies, replace directives
- `doc.go` with package comment and quick-start example
- `.gitignore` (pre-existing, verified)
- `example/` directory

### P3: Status Mapping Layer
- `mapStatusToBadge` — pass→Success, warn→Warning, fail→Error, unknown→Neutral
- `mapStatusToAlert` — pass→Success, warn→Warning, fail→Error, unknown→Info
- `mapStatusToText` — human-readable status strings
- `groupChecks` — partitions by severity (fail/warn/pass), sorted alphabetically
- `rowsToTableRows` — converts to templ-components TableRow with Badge components
- `buildViewModel` — assembles full view model with shutdown overlay
- 9 tests, all passing

### P4: Dashboard Templ Pages
- `view.templ` — full HTML page with layout.Base, StatCards grid, PolledRegion, Card+Table per group
- `partial.templ` — PolledRegion wrapper for HTMX polling
- Both generated to `*_templ.go` via `templ generate`
- Tailwind CDN loaded via head script
- Dark mode classes present (verified)

### P5: Content Negotiation Handler
- `acceptsHTML(r)` — parses Accept header for text/html or */*
- HTML request → render full page; JSON request → delegate to probe.ReadinessHandler()
- Missing Accept header → JSON (kubelet default); */* → HTML (browser default)
- Tested with 5 Accept header variants

### P6: Public API + Options
- `Dashboard` struct, `Config`, `Option` type
- `New(probe, opts...)` with defaults
- `WithTitle`, `WithRefreshInterval`, `WithRefreshMode`, `WithRoutes`, `WithNonce`, `WithSSEPush`
- `Handler()`, `PartialHandler()`, `SSEHandler()`, `RegisterRoutes()`, `Start()`, `Shutdown()`

### P7: HTMX Polling
- PolledRegion wraps dashboard content with outerHTML swap
- Partial endpoint returns fresh PolledRegion + content on each poll
- Interval configurable, defaults to probe's RefreshInterval

### P8: StatCards + Card Grouping
- 3 StatCards: Version, Uptime, Check Latency
- Checks grouped into severity Cards: Critical Failures, Non-Critical Issues, Healthy Services
- Each Card uses Flush table (no padding border)
- Empty checks map shows "No registered services" empty state

### P9: Routes
- `Routes` struct with 6 fields (Dashboard, Partial, SSE, Liveness, Readiness, Startup)
- `DefaultRoutes()` — conventional paths
- `RegisterRoutes` wires all endpoints including probe handlers

### P10: Comprehensive Tests
- 31 tests total (9 status + 22 dashboard), all passing
- Content negotiation dispatch, HTML output validation, options, routes, shutdown, SSE mode
- 2 benchmarks (HTML rendering, partial rendering)

### P11: Example App
- Mock services: always-healthy, flapping (15s cycle), always-failing
- Critical (postgres, redis) and non-critical (metrics-exporter) classification
- Full demo server with RegisterRoutes

### P12: flake.nix
- flake-parts + treefmt-nix pattern matching go-health
- Apps: generate, test, test-race, build, vet, lint, coverage, vulncheck, security, example, clean
- devShell with templ CLI, GOEXPERIMENT=jsonv2 set
- templ generate as pre-build step

### P13: Documentation
- `README.md` — quick start, routes table, options, refresh modes, SSE, status mapping
- `AGENTS.md` — architecture, design decisions, data flow, testing patterns, gotchas
- `doc.go` — package comment with example

---

## B) PARTIALLY DONE

### P14: SSE Push Mode — 70% done
- **Done:** ssePusher struct, start goroutine, broadcast loop, SSE connection handler, heartbeat, initial state push
- **Missing:**
  - Does NOT use go-datastar (ElementsFromTempl) — uses hand-rolled EventSource + innerHTML replacement instead. The plan specified Datastar SDK integration. go-datastar is in replace directives but NEVER imported.
  - No `datastar.SDKScript` in page head when SSE is active
  - Client-side JavaScript is inline and manual, not Datastar-powered

### P15: Status Change Detection — 60% done
- **Done:** `PushMode` type defined (PushOnChange, PushAlways), `shouldBroadcast` method, lastStatus tracking
- **CRITICAL BUG:** `hashChecks()` iterates over a Go map (randomized iteration order) and concatenates into a string. The hash is **non-deterministic** — the same checks map produces different strings on different calls. This means PushOnChange mode ALWAYS detects "changes" and broadcasts every tick, defeating the purpose.
- **Missing:** `WithPushMode(PushMode)` option function was planned (P15.3) but never implemented. The pusher hardcodes `PushOnChange` with no way to change it.

### P16: Polish — 70% done
- **Done:** Build passes, vet passes, tests pass, dark mode classes verified present, responsive grid verified, Tailwind CDN loaded, badges render with correct colors
- **Missing:**
  - golangci-lint never actually run (only `go vet`)
  - No flake.lock generated
  - go-health's `content-negotiation-design.md` not updated to link to this repo (P16.4)
  - No LICENSE file (README says MIT but no file exists)
  - No .golangci.yml

---

## C) NOT STARTED

1. **go-datastar integration** — The plan explicitly designed SSE around `datastar.ElementsFromTempl()` for patch protocol. Implemented with vanilla EventSource instead. go-datastar is a dead dependency.
2. **Datastar SDK script in head** — P14.7: Include `datastar.SDKScript` in page `<head>` when SSE mode is active. Not done.
3. **`WithPushMode` option** — P15.3. Not implemented.
4. **Q-value sorting in acceptsHTML** — P5.4 specified handling `Accept: application/json;q=1.0, text/html;q=0.1`. Current implementation uses naive string `Contains` which returns HTML for this case (wrong).
5. **`WithCSSPath` option** — No way to override Tailwind CDN with compiled CSS for production.
6. **flake.lock** — Never generated. `nix build` will fail without it.
7. **GOWORK=off in flake.nix** — go-health's flake sets this; I didn't. The workspace `go.work` in `~/projects` will interfere with Nix builds.
8. **LICENSE file** — Missing entirely.
9. **.golangci.yml** — Missing.
10. **Version constant** — No exported version string.
11. **govulncheck / gosec** — Never run.

---

## D) TOTALLY FUCKED UP

### D1: `hashChecks` is non-deterministic (CRITICAL BUG)

```go
func hashChecks(checks map[string]health.Check) string {
    var buf bytes.Buffer
    for name, check := range checks { // MAP ITERATION IS RANDOMIZED IN GO
        buf.WriteString(name)
        // ...
    }
    return buf.String()
}
```

Go randomizes map iteration order. The same checks map produces a different hash string every call. `shouldBroadcast` compares against `lastChecks` — which is always different. **PushOnChange mode broadcasts every single tick.** The status change detection is completely broken.

**Fix:** Sort keys before concatenating, or use a sorted data structure.

### D2: GOEXPERIMENT=jsonv2 requirement is a major friction point

The go-sse dependency imports `encoding/json/v2` which requires `GOEXPERIMENT=jsonv2`. This means:
- Every `go build`, `go test`, `go run` needs the env var
- Every consumer of go-health-dashboard needs to set it
- The flake.nix devShell sets it, but plain `go build` without Nix fails
- This is an experimental Go feature that could change

This should have been flagged as a design decision earlier. Options:
- Make SSE mode a build-tag-gated separate package
- Fork go-sse to remove json/v2 dependency
- Accept the requirement and document loudly (current approach, but discovered late)

### D3: go-datastar is a dead dependency

The plan designed SSE around go-datastar's `ElementsFromTempl`. I imported go-sse but never used go-datastar. The `go-datastar` replace directive and module requirement exist but nothing imports it. Dead code path in dependency management.

### D4: flake.nix won't work for `nix build`

Two problems:
1. **No GOWORK=off** — The workspace `~/projects/go.work` includes replace directives pointing to local paths. `nix build` will try to use these local paths which don't exist in the Nix sandbox.
2. **No flake.lock** — Never generated.
3. **templ generate modifies source** — The build apps run `templ generate` before `go build`, but in a pure Nix build the source is read-only.

### D5: `WithPushMode` option promised by type but not implemented

`PushMode` type and constants are defined and exported. Users will expect `WithPushMode(dashboard.PushOnChange)` to exist. It doesn't.

---

## E) WHAT WE SHOULD IMPROVE

1. **Fix `hashChecks`** — Sort map keys before concatenating. This is a one-line fix but it's a correctness bug in the headline SSE feature.
2. **Implement `WithPushMode`** — The type exists; the option doesn't. 5-minute fix.
3. **Decide on go-datastar** — Either use it for SSE patches (as planned) or remove it from dependencies entirely. Currently it's dead weight.
4. **Fix acceptsHTML q-values** — Use `mime.ParseAccept` or similar for proper content negotiation. The naive string search is wrong for edge cases.
5. **Set GOWORK=off in flake.nix** — Copy the pattern from go-health. Without this, Nix builds are broken.
6. **Generate flake.lock** — Run `nix flake lock` or `nix build` once.
7. **Add LICENSE file** — MIT, matching README claim.
8. **Add .golangci.yml** — Copy from go-health, adapt for this project.
9. **Actually run golangci-lint** — Never ran it. Could be hiding issues.
10. **Add `WithCSSPath` option** — Let production users swap Tailwind CDN for compiled CSS.
11. **Add version constant** — `const Version = "0.1.0"`.
12. **Test the example app actually runs** — Never started it. Could have runtime errors.
13. **Remove unused `r *http.Request` parameter in `buildData`** — The request is passed but never used.
14. **Benchmark setup is hacky** — Uses `&testing.T{}` which is wrong. Should use proper benchmark initialization.
15. **SSE handler test is fragile** — Uses 100ms context timeout. Could flake on slow CI.
16. **No integration test for SSE change detection** — The headline SSE feature has no test verifying that status changes trigger broadcasts and unchanged state doesn't.
17. **The `min` builtin usage in test** — `body[strings.Index(body, "hx-trigger"):min(len(body), strings.Index(body, "hx-trigger")+40)]` — if `strings.Index` returns -1, this slices incorrectly. Fragile error message construction.
18. **Dark mode toggle missing** — layout.Base includes theme script, but no toggle button rendered. Users can't switch themes.
19. **No mobile-specific testing** — Plan says verify mobile responsive. Only checked that `grid` class exists.
20. **Documentation says "go-datastar" in dependency table** — README lists go-datastar as unused. Should clarify it's only needed for planned Datastar SSE mode.

---

## F) Up to 50 Things to Do Next

### Critical (must fix before any release)
1. **Fix `hashChecks` non-deterministic bug** — sort keys before hashing
2. **Implement `WithPushMode(PushMode)` option**
3. **Set `GOWORK=off` in flake.nix devShell**
4. **Generate `flake.lock`**
5. **Actually run `nix build` and fix whatever breaks**
6. **Run `golangci-lint` and fix issues**
7. **Run `govulncheck`**
8. **Remove go-datastar from dependencies** (if not going to use it) OR implement Datastar SSE mode
9. **Add LICENSE file**
10. **Test the example app by actually running it**

### High Priority
11. **Fix `acceptsHTML` to use proper Accept header parsing with q-values**
12. **Add `WithCSSPath(string)` option for production CSS**
13. **Add `.golangci.yml`**
14. **Add version constant**
15. **Remove unused `r *http.Request` parameter from `buildData`**
16. **Fix benchmark helper to not use `&testing.T{}`**
17. **Add SSE change detection integration test**
18. **Update go-health `content-negotiation-design.md` to link to this repo** (P16.4)
19. **Add `Content-Type` constants**
20. **Verify mobile responsive layout with actual viewport testing**

### Medium Priority
21. **Add `WithNonce` support to templ HeadContent** (CSP compliance)
22. **Add dark mode toggle button to dashboard**
23. **Add auto-generated refresh timestamp display** (PolledRegion ShowTimestamp)
24. **Add status history/sparkline** (templ-components has Sparkline)
25. **Add favicon**
26. **Add OG metadata for dashboard page**
27. **Consider build-tag gating for SSE code** (so consumers who don't need SSE don't pull go-sse)
28. **Add graceful SSE client reconnection** (EventSource auto-reconnects, but verify)
29. **Add SSE connection count metrics** (Broadcaster.SubscriberCount)
30. **Add request logging middleware option**
31. **Add custom CSS class option for dashboard container**
32. **Add option to hide StatCards** (minimal mode)
33. **Add option to hide specific services from the dashboard**
34. **Add service grouping by custom tags/labels** (not just severity)
35. **Add WebSocket alternative to SSE** (for environments where SSE is blocked)

### Low Priority / Polish
36. **Add CONTRIBUTING.md**
37. **Add CHANGELOG.md**
38. **Add FEATURES.md**
39. **Add TODO_LIST.md**
40. **Add ROADMAP.md**
41. **Add docs/DOMAIN_LANGUAGE.md**
42. **Add screenshot to README** (run example, capture browser screenshot)
43. **Add CI/CD GitHub Actions workflow**
44. **Add Dependabot config**
45. **Add pre-commit hooks**
46. **Add release process documentation**
47. **Add semantic versioning tags**
48. **Add `go test -race` to CI**
49. **Add coverage badge to README**
50. **Consider extracting SSE pusher into interface for testability**

---

## G) Questions I Cannot Answer Myself

### G1: Should go-datastar be used for SSE, or removed entirely?

The plan designed SSE around `datastar.ElementsFromTempl()` for patch protocol. I implemented SSE with vanilla EventSource + innerHTML replacement instead, which works but doesn't use Datastar's reconnection replay, signal management, or patch protocol. go-datastar is currently a dead dependency.

**Decision needed:** Full Datastar integration (more powerful, more complex), or remove go-datastar and keep vanilla SSE (simpler, fewer deps)?

### G2: Is the GOEXPERIMENT=jsonv2 requirement acceptable?

go-sse depends on `encoding/json/v2` which requires `GOEXPERIMENT=jsonv2`. This propagates to every consumer. Options:
- Accept it (document loudly, set in flake.nix)
- Fork go-sse to remove json/v2 (maintenance burden)
- Build-tag gate the entire SSE feature so non-SSE consumers don't need it
- Wait for Go 1.27 where json/v2 may be stable

**Decision needed:** Which approach? This affects the package's usability.

### G3: Should replace directives stay or go?

The `go.mod` has 6 `replace` directives pointing to local sibling repos (`../go-health`, `../templ-components`, etc.). These are needed for local development but must be removed before `go get` works for external consumers. The upstream repos aren't tagged on GitHub yet.

**Decision needed:** Tag and publish the upstream repos first (go-health, templ-components, go-sse, go-error-family, go-branded-id), or keep replace directives and document the local-dev-only status?
