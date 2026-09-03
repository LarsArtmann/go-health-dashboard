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

Extend beyond a single go-health Probe to aggregate multiple services or
clusters on one dashboard.

Raw ideas:

- Multi-probe support: dashboard for multiple services on one page
- Service grouping by custom tags or labels (not just severity)
- Aggregate status across multiple instances or clusters
- Federation: pull health from remote go-health instances via HTTP
  (spike conclusion: preferred home is a `FederatedProber` option in
  go-health, not the dashboard — see Design Spikes)
- `Register` auto-start: wire `Start(ctx)` into the samber/do container
  lifecycle so `Register` + container start needs no manual `Start` call

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
- **Next release policy:** the post-v0.3.1 batch in CHANGELOG `[Unreleased]`
  is purely additive (new options, endpoints) — semver suggests v0.4.0;
  alternatively batch more and cut v0.3.2.
- **Fingerprint compatibility:** the v0.3.x length-prefix fix changed
  fingerprint values (one spurious "change" after upgrade if anyone persisted
  them). Accept + document as-is, or version/stabilize the fingerprint
  format?

## Design Spikes (v0.3.x cycle, not implemented)

Summaries in `docs/planning/2026-09-03_v03-cycle-decisions-notes.md`.

- **Federation spike** — expose one instance's health to another as a
  synthetic check. Preferred home: go-health (`FederatedProber` option),
  not the dashboard; the dashboard renders, it does not source.
- **WebSocket transport spike** — rejected for now: SSE already covers
  one-way push with browser-managed reconnection; WebSocket would add a
  second protocol surface (auth, proxies, reconnect logic) with no
  dashboard use case for bidirectional traffic.
