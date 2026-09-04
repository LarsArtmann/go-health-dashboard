# Roadmap

> Long-term direction and raw ideas. Items here are NOT actionable tasks.
> When an idea is refined into bounded work, it moves to TODO_LIST.md.

## Themes

### 1. Production Hardening

Make the dashboard bulletproof for production NOC deployments: always-on
monitoring walls, proxy environments, strict CSP, and high-connection scenarios.

Raw ideas:

- Per-route stricter CSP for `/health` alone (no `style-src 'unsafe-inline'`) if
  a security audit demands it (the default render is already style-clean; see
  README CSP section)
- Metrics endpoint security headers / `noindex` — plan M27
- Rate limiter response headers: `X-RateLimit-Limit` / `X-RateLimit-Remaining`
  / `X-RateLimit-Reset` (basic 429 + Retry-After shipped in v0.3.x cycle)
- Optional jitter on `WithMaxConnectionLifetime` to avoid reconnect herds
- Watchdog gauge `dashboard_pusher_last_tick_seconds` and an opt-in
  auto-restart hook (current watchdog is report-only by design)
- Request logging middleware option (slog) for dashboard routes

### 2. Multi-Service and Federation

In-process multi-service shipped in v0.5.0 (go-health `aggregate` + the
`Prober` interface). What remains:

Raw ideas:

- `WithGrouping(BySource)` view option: per-service cards by splitting
  namespaced `source/check` keys (severity grouping already renders
  aggregates fine)
- Per-source staleness surface: last-refresh age per source (needs a
  timestamp API in go-health)
- Service grouping by custom tags or labels (not just severity)
- Aggregate status across multiple instances or clusters
- Federation: pull health from remote go-health instances via HTTP
  (spike conclusion: preferred home is a `FederatedProber` option in
  go-health, not the dashboard — see Design Spikes)
- `Register` auto-start: wire `Start(ctx)` into the samber/do container
  lifecycle so `Register` + container start needs no manual `Start` call
- Example/demo: aggregate + webhook demo modes in `nix run .#example`
  (it still demos a single probe)

### 3. Observability and History

Move from point-in-time status to temporal awareness: what happened, when,
and what's trending.

Raw ideas:

- History retention window (`WithTrendWindow(duration)`) and `?since=`
  incremental polling on `/health/trend`; `WithTrend` ships an in-memory ring
  whose size is sample-count-based
- Conditional requests on `/health/export` (ETag / If-None-Match)
- Trend visualization transition markers (dots) on the sparkline SVG —
  transitions ship as data via `/health/trend`; the visual markers are
  unwritten
- Per-check sparklines grouped by severity
- `dashboard_health_check_last_transition_seconds` metric and
  `dashboard_build_info{version=...}` / `dashboard_health_checks_total`
- Optional `client_golang` bridge package for consumers with existing
  registries (current exposition is hand-rolled, zero-dep)
- Public mode hardening: leak-scanner test (grep rendered HTML for registered
  service names), optional redaction of the `/health` JSON (which stays
  verbatim today — documented in AGENTS.md)
- Webhook delivery observability: `dashboard_webhook_deliveries_total{result}`
  - duration histogram behind `WithMetrics` (deliveries are silently
    best-effort today — "it didn't arrive" is undebuggable)
- Webhook hardening: HMAC signing (`WithWebhookSecret` → `X-Signature`)
  and a payload `"schema":1` version field before external consumers
  freeze the format (see Open Questions)
- Introspection endpoint (JSON: enabled routes, limits, modes) for ops
- Rate-limit 429 body: include Retry-After as JSON for API clients
- PushMode: PushOnChange with TTL (re-assert state every N intervals)
- Timeline card: cap entries by age as well as count (5 entries can
  span days)
- Per-check latency histogram series in metrics (currently total only)
- NDJSON export format option on `/health/export`
- Incident tracking (annotate status changes with context) — deferred; needs
  product thought beyond the timeline card that shipped in the v0.3.x cycle

### 4. Deployment Flexibility

Support environments where SSE is blocked, where the dashboard must be
embedded, or where strict network policies apply.

Raw ideas:

- WebSocket alternative transport — spike rejected for now (see Design
  Spikes); revisit only if SSE-blocked environments turn out to be real
- Build-tag gating for SSE code (consumers who only want HTML pay no SSE
  cost) — BLOCKED on the `GOEXPERIMENT=jsonv2` decision (Open Questions)
- Screenshot or PDF export for incident reports — screenshots are covered by
  env-guarded tests; PDF remains out of scope
- Embedded Datastar SDK serving helper (a `WithCSSPath` analog) so
  CSP-'self' deployments don't hand-roll static wiring
- Browser test: CSP-clean runtime check for an aggregate-rendered page
- Load test: 20-source aggregate under concurrent SSE + scrape

### 5. Pipeline, Testing and Docs Tooling

Raw ideas:

- CI: `templ generate` drift check (fail when generated files differ)
- CI: `actionlint` step for both workflow files; add a `nix flake check` job;
  Go version matrix (latest two 1.26.x patches)
- CI: upload browser screenshots as artifacts for visual diffs;
  Dependabot/renovate for GitHub Action SHA bumps (pins rot)
- devShell: Chrome/Chromium in the flake so the browser suite runs locally
- Coverage: push total > 80% (currently ~77%), then raise the CI floor
- Nightly fuzztime budget review (4×60s → target the hottest target);
  rehearse the fuzz issue-on-failure path with a deliberately failing run
- Fuzz targets: CSV exporter, `RecommendedCSP` injection attempts, webhook
  payload marshal, aggregate merge (follow the `fuzz_test.go` pattern)
- Browser golden-screenshot diff test (catch visual drift)
- Keyboard-navigation a11y smoke in the browser suite; browser-test the
  metrics endpoint under strict CSP
- Unit-test the version-guard grep logic (script drift protection)
- Boundary/negotiation tests: `WithTrend(1)`, `ExportHandler` with
  `Accept: text/csv;q=0.8`, `WithBasePath` edge cases (`""`, `"/"`,
  trailing slash, nested `/a/b`), SSE retry large values
- Race-stress: 50 concurrent SSE clients vs `SubscriberCount` consistency;
  webhook delivery-ordering test under concurrent transitions
- Verify no heartbeat-goroutine leak on Shutdown/broadcaster close;
  fuzz the `safeBasePath` validator once it moves into the library
- Benchmarks: `renderPatch` retry stamping; `BenchmarkHealthCheck`
- Mutation-test spot check on change-detection (fingerprint) logic
- DI-surface ideas (from the 04-41 NEXT-50, still open): example-binary e2e
  run, `BenchmarkRegister`, `do.Package` wrapper, `WithInjector` evaluation,
  lazy `Provider` variant, the self-monitoring decision (should a registered
  Dashboard appear in its own health table?), robustness tests (nil
  injector/probe panics, double-`Register` override, restart-after-shutdown,
  cascade order, `HealthCheck` under live SSE, `SubscriberCount` after
  `Register`), HealthCheck↔probe aggregation, pusher-metrics exposure,
  `RegisterNamed` multi-instance, child scopes, `Explain()` output, the
  `do.Healthchecker` no-context variant, and the ProvideValue-vs-Provide ADR
- Decide the Dashboard self-monitoring question (a registered Dashboard may
  appear in its own health table): feature or filter?
- Shutdown-ordering test for the example (pusher alive vs broadcaster closed)
- API ergonomics: `Routes()` accessor for external mux wiring; store a
  `BasePath` field and resolve routes after all options run (kills the
  `WithRoutes`/`WithBasePath` ordering footgun)
- docs/release-checklist.md (reconcile → changelog → version → tag → push →
  proxy-verify → CI watch)
- ADR: the options/handlers/history split + the error-sentinel family
- README: coverage badge; "protect probes via network policy" note;
  screenshot regenerate one-liner + caption for the light screenshot
- doc.go: webhook + public-mode combo example; `WithBasePath` example
- templ-components UI follow-ups (PageHeader, Stack, StatCard icons, Dot) —
  verify not already shipped upstream, then adopt
- Deprecate `WithNonce` in favor of `WithNonceExtractor` (long-term)
- `nix run .#ci` local mirror of the GitHub Actions steps
- Investigate why 14 gopls stdversion warnings persist despite the committed
  `.vscode/settings.json` (tooling split brain)
- Consider signing tags (release hardening) + Keep-a-Changelog compare links
  in CHANGELOG footers

## Non-goals

Things we are deliberately NOT pursuing and why:

- **HTMX polling mode:** Removed in the SSE-first rewrite. SSE is strictly better
  in this ecosystem. If a real user needs polling, it can be added then, but we
  will not maintain two modes.
- **Content negotiation on probe endpoints:** `/healthz`, `/readyz`, `/startupz`
  are JSON-only. Only `/health` does Accept-based negotiation. We will not add
  Accept parsing to the kubelet-style endpoints.
- **Alternative template engines:** The dashboard is built on templ. Supporting
  html/template or text/template would double the rendering surface for no
  benefit.
- **Database-backed health history:** The dashboard is a view layer. Historical
  data storage is a separate concern that belongs in the application, not the
  dashboard library. Note: the pusher's in-memory trend ring (`WithTrend`)
  is bounded, ephemeral view state — not a history store — so this non-goal
  still holds.

## Open Questions

These require user decisions and cannot be resolved by reading code:

- **GOEXPERIMENT=jsonv2:** Every Go command requires this env var because go-sse
  uses `encoding/json/v2`. Accept it (and document loudly), fork go-sse, or
  build-tag gate the SSE code?
- **Fingerprint compatibility:** the v0.3.x length-prefix fix changed
  fingerprint values (one spurious "change" after upgrade if anyone persisted
  them). Accept + document as-is, or version/stabilize the fingerprint
  format?
- **Webhook hardening appetite:** is `Authorization`-header auth enough for
  your ingests, or do you want HMAC signing (`WithWebhookSecret` →
  `X-Signature`) and an explicit payload `"schema":1` version field before
  any external consumer freezes the format?

## Design Spikes (v0.3.x cycle, not implemented)

Summaries in `docs/planning/archived/2026-09-03_v03-cycle-decisions-notes.md`.

- **Federation spike** — expose one instance's health to another as a
  synthetic check. Preferred home: go-health (`FederatedProber` option),
  not the dashboard; the dashboard renders, it does not source.
- **WebSocket transport spike** — rejected for now: SSE already covers
  one-way push with browser-managed reconnection; WebSocket would add a
  second protocol surface (auth, proxies, reconnect logic) with no
  dashboard use case for bidirectional traffic.
