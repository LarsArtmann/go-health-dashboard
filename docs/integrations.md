# Integration Cookbook

go-health and go-health-dashboard are designed to **feed** your existing
monitoring stack — Prometheus/Grafana, SigNoz, Gatus, Uptime Kuma, custom
ingests — not to replace it. This document encodes the health semantics once,
so every integration expresses the same meaning:

| Concept | Meaning | Wire signal |
|---|---|---|
| `pass` | Healthy | `200` on `/readyz`, `dashboard_health_status{2}` |
| `warn` | Degraded but serving (non-critical failure) | `200` on `/readyz` — **deliberately not an error code** |
| `fail` | Not serving (critical failure) | `503` on `/readyz` |
| shutting down | Graceful drain in progress | `503` on `/readyz`, `dashboard_health_shutting_down 1` |
| liveness | Process alive, zero dependency checks | Always `200` on `/healthz` (by design — no restart cascades) |

---

## Prometheus (and Grafana, SigNoz, Mimir, VictoriaMetrics)

Enable the endpoint:

```go
dash := dashboard.New(probe,
    dashboard.WithMetrics(true), // serves /health/metrics (Prometheus text 0.0.4)
)
```

Scrape it, then alert on:

```promql
# Any service failing (page)
dashboard_health_status != 2

# Actually down (fail or shutting down — the 503 condition)
max(dashboard_health_status) == 0

# A specific dependency is unhealthy
dashboard_health_check{check="postgres"} == 0

# Degraded-but-serving trend (warn count over time)
count(dashboard_health_check{status="warn"} == 1)

# Health-check batch is getting slow
histogram_quantile(0.99, sum(rate(dashboard_health_check_duration_seconds_bucket[5m])) by (le))
```

**SigNoz-specific traps** (learned the hard way — see SystemNix
`_signoz-alerts.nix`):

- `target=0` with `above_or_equal` fires when the metric is `>= 0` — always
  true for non-negative metrics. To alert "at least one failing", use
  `target=1` with `above_or_equal`, or `dashboard_health_status` `below 2`.
- `{{$value}}` must have zero spaces inside the braces; `{{ $value }}` gets
  rewritten by SigNoz's template preprocessor and renders empty.

Grafana users: the same expressions work directly; a minimal stat panel on
`dashboard_health_status` with value mappings `2→Pass, 1→Warn, 0→Fail`
reproduces the dashboard's status banner.

---

## Gatus

Gatus polls endpoints over HTTP, so the readiness endpoint's status-code
contract does all the work. Encode go-health semantics **once** — for NixOS,
as a library function:

```nix
# lib/go-health.nix (SystemNix)
lib: {
  # A go-health readiness endpoint as a Gatus endpoint.
  # warn stays 200 by design, so Gatus alerts only on real failures;
  # degraded state is visible on /health and in metrics.
  goHealthReady =
    { name, url, interval ? "30s", description ? "${name} readiness failing" }:
    {
      inherit name url interval description;
      group = "go-health";
      conditions = [
        "[STATUS] == 200"
        "[RESPONSE_TIME] < 1000"
      ];
      alerts = [
        {
          type = "discord";
          failure-threshold = 3;
          success-threshold = 2;
          send-on-resolved = true;
        }
      ];
    };
}
```

Plain YAML equivalent:

```yaml
endpoints:
  - name: billing-api
    url: http://billing-api.internal:8080/readyz
    interval: 30s
    conditions:
      - "[STATUS] == 200"
      - "[RESPONSE_TIME] < 1000"
    alerts:
      - type: discord
        failure-threshold: 3
        success-threshold: 2
        send-on-resolved: true
```

Notes:

- Point at `/readyz`, not `/healthz`: liveness is dependency-blind on
  purpose, so it makes a weak monitoring signal.
- Want degraded (`warn`) visibility in Gatus? Add a body condition on the
  dashboard's JSON instead of alerting on 503:
  `"[BODY] == pat(*\"status\":\"pass\"*)"` against `/health` with
  `Accept: application/json` — but expect noise; warn is designed to be
  visible, not paged.

---

## Uptime Kuma

Add an **HTTP(s) monitor**:

- URL: `http://service:8080/readyz`
- Heartbeat: 30–60s (matches the probe's default 1s cache; no dependency
  hammering)
- Accepted status codes: `200` (leave `200-299` defaults; 503 = down)

Keyword monitors: `GET /health` with `Accept: application/json` and keyword
`"status":"pass"` also works, but the status-code monitor is cheaper and
never lies about drain state.

---

## Webhooks (push-based integration)

For egress-restricted deployments (NAT, serverless, locked-down clusters) or
custom JSON ingests, the dashboard can push every transition instead of being
polled:

```go
dash := dashboard.New(probe,
    dashboard.WithWebhook("https://ingest.internal.example/health-events"),
    dashboard.WithWebhookHeaders(map[string]string{
        "Authorization": "Bearer " + os.Getenv("INGEST_TOKEN"),
    }),
    dashboard.WithPublicMode(), // optional: mask names/errors before they leave
)
```

Every state transition — including the initial state on `Start` — is POSTed
as JSON:

```json
{
  "status": "warn",
  "shutting_down": false,
  "checks": {
    "cache": {"status": "warn", "error": "connection refused"}
  },
  "changed_at": "2026-09-04T00:30:00Z"
}
```

Delivery contract:

- **Change-only**: fires when the overall status or any check's fingerprint
  changes, independent of `PushMode` (SSE may stream every tick; webhooks
  never spam).
- **Best-effort**: 10s timeout, no retries, no logging. The receiver (or the
  platform in front of it) owns alert thresholds and dedup.
- **Masked under `WithPublicMode`**: check names become `check-N` (sorted)
  and error strings are dropped.
- **Secrets stay in config**: the URL may embed a token; it is never logged.

This is the recommended path for feeding event ingests (PapDashboard-style
`/api/ingest` endpoints, n8n, custom responders). It removes the fragile
`[PLACEHOLDER]` template layer that tool-specific custom alerting providers
require — the payload shape is owned and versioned here.

---

## What we deliberately do NOT build

- **Remote federation** (pulling health from remote instances): Kuma, Gatus,
  and Grafana are purpose-built for multi-service aggregation, history, and
  alerting. Use them. For several probes inside *one* process, use
  go-health's `aggregate` package and hand the merged view to this dashboard.
- **A public status page**: `WithPublicMode` anonymizes the rendered
  dashboard and metrics; hosting, whitelisting, and incident workflows stay
  with your status-page provider.
