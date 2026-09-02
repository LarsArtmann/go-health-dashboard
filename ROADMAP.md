# Roadmap

> Long-term direction and raw ideas. Items here are NOT actionable tasks.
> When an idea is refined into bounded work, it moves to TODO_LIST.md.

## Themes

### 1. Production Hardening

Make the dashboard bulletproof for production NOC deployments: always-on
monitoring walls, proxy environments, strict CSP, and high-connection scenarios.

Raw ideas:

- Per-route stricter CSP for `/health` alone (no `style-src 'unsafe-inline'`) if
  a security audit demands it
- Latency histogram (prometheus buckets) — session 2026-09 plan M8
- Metrics endpoint security headers / `noindex` — plan M27
- Rate limiting on dashboard HTML route (per-IP token bucket) — plan M12
- SSE connection max-lifetime option — plan M10
- Pusher watchdog (last-tick recency surfaced via `HealthCheck`) — plan M11
- Graceful shutdown: wait for in-flight SSE connections to drain — plan M9

### 2. Multi-Service and Federation

Extend beyond a single go-health Probe to aggregate multiple services or
clusters on one dashboard.

Raw ideas:

- Multi-probe support: dashboard for multiple services on one page
- Service grouping by custom tags or labels (not just severity)
- Aggregate status across multiple instances or clusters
- Public-facing status page mode (no internal details exposed)
- Federation: pull health from remote go-health instances via HTTP

### 3. Observability and History

Move from point-in-time status to temporal awareness: what happened, when,
and what's trending.

Raw ideas:

- Health check history with retention window — partially shipped as `WithTrend`
  (in-memory ring); retention/export extensions are plan M23/M25
- Trend visualization (templ-components has Sparkline) — **shipped** as
  `WithTrend` in v0.3.0; markers/timeline extensions are plan M22/M29
- Status change timeline (when did each service flip?) — plan M29
- Auto-generated refresh timestamp display — plan M26
- Prometheus-compatible metrics endpoint for the dashboard itself — **shipped**
  as `WithMetrics` in v0.3.0
- Export health data as JSON/CSV — plan M25
- Public-facing status page mode (no internal details exposed) — plan M28
- axe-core accessibility verification in the browser test — plan M21

### 4. Deployment Flexibility

Support environments where SSE is blocked, where the dashboard must be
embedded, or where strict network policies apply.

Raw ideas:

- WebSocket alternative transport (for environments where SSE is blocked)
- Build-tag gating for SSE code (consumers who only want HTML pay no SSE cost)
- Embeddable dashboard component (mount under a sub-path, not root) — **DONE** in v0.2.0 via `WithBasePath`
- Authentication middleware integration — **shipped** as `WithMiddleware` in
  v0.3.0
- OG metadata and social preview for dashboard page — plan M27 (og:description)
- `RecommendedCSP()` helper so consumers get a correct CSP without hand-rolling
  — plan M3
- Screenshot or PDF export for incident reports — partially covered by
  `screenshot_test.go`; PDF out of scope
- Request logging middleware option (slog) — candidate, not scheduled
- Incident tracking (annotate status changes with context) — deferred; needs
  product thought beyond the M29 timeline

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
- **Database-backed health history:** The dashboard is a stateless view layer.
  Historical data storage is a separate concern that belongs in the application,
  not the dashboard library.

## Open Questions

These require user decisions and cannot be resolved by reading code:

- **GOEXPERIMENT=jsonv2:** Every Go command requires this env var because go-sse
  uses `encoding/json/v2`. Accept it (and document loudly), fork go-sse, or
  build-tag gate the SSE code?
