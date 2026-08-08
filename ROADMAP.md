# Roadmap

> Long-term direction and raw ideas. Items here are NOT actionable tasks.
> When an idea is refined into bounded work, it moves to TODO_LIST.md.

## Themes

### 1. Production Hardening

Make the dashboard bulletproof for production NOC deployments: always-on
monitoring walls, proxy environments, strict CSP, and high-connection scenarios.

Raw ideas:

- Configurable SSE connection limit to prevent connection-exhaustion DoS
- SSE reconnection support via `Last-Event-ID` header
- Graceful shutdown: wait for in-flight SSE connections to drain
- Configurable heartbeat interval (currently hardcoded 15s)
- SSE connection timeout to prevent infinite-lived connections
- Pusher health self-check (is the goroutine alive?)
- Request logging middleware option

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

- Health check history with retention window
- Trend visualization (templ-components has Sparkline)
- Status change timeline (when did each service flip?)
- Incident tracking (annotate status changes with context)
- Auto-generated refresh timestamp display
- Prometheus-compatible metrics endpoint for the dashboard itself
- Export health data as JSON/CSV

### 4. Deployment Flexibility

Support environments where SSE is blocked, where the dashboard must be
embedded, or where strict network policies apply.

Raw ideas:

- WebSocket alternative transport (for environments where SSE is blocked)
- Build-tag gating for SSE code (consumers who only want HTML pay no SSE cost)
- Embeddable dashboard component (mount under a sub-path, not root)
- Authentication middleware integration
- OG metadata and social preview for dashboard page
- Screenshot or PDF export for incident reports

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

- **License:** The LICENSE file says PROPRIETARY (all rights reserved). The
  README says MIT. Which is correct? This must be resolved before any release.
- **Replace directives:** go.mod has 6 `replace` directives pointing to local
  sibling repos. Should these stay (local-dev-only) or should upstream repos be
  tagged on GitHub first?
- **GOEXPERIMENT=jsonv2:** Every Go command requires this env var because go-sse
  uses `encoding/json/v2`. Accept it, fork go-sse, or build-tag gate the SSE
  code?
