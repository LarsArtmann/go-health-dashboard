# Metrics Reference

The dashboard exposes Prometheus-compatible metrics as a consumer-facing
reference for operators scraping `/health/metrics`. Everything here is
produced by `metrics.go` (hand-rolled text exposition, Prometheus format
0.0.4) — one page instead of code spelunking.

## Enabling the endpoint

```go
dash := dashboard.New(probe,
    dashboard.WithMetrics(true), // default: disabled
)
dash.RegisterRoutes(mux) // serves Config.Routes.Metrics (default /health/metrics)
```

- The route is configured via `WithRoutes` / `WithBasePath`. An **empty**
  `Routes.Metrics` disables the endpoint even when `WithMetrics(true)` is
  set.
- `WithBasePath("/admin")` prefixes a non-empty metrics route
  (`/admin/health/metrics`).
- The route is dashboard-owned: `WithMiddleware` wraps it, so it sits
  behind whatever auth/CSP policy you attach there.

## Wire format

- **Prometheus text exposition format 0.0.4.** No OpenMetrics negotiation,
  no exemplars, no timestamps, no `Created` series.
- **Deterministic output.** Check families are emitted in sorted
  `check`-label order; successive scrapes with an unchanged response are
  byte-identical except for histogram `_sum`/counts, which legitimately
  move with each pusher tick.
- **Label values are escaped** per the exposition format (backslash,
  double quote, newline) so hostile check names cannot break the document.
- Aggregate (`aggregate.New`) and federated (`federation.New`) probes
  land their checks as namespaced keys (`source/check`, `name/check`) in
  the same `check` label — no extra metric families.

## Metric families

All names carry the `dashboard_` prefix.

| Metric | Type | Labels | Meaning |
| --- | --- | --- | --- |
| `dashboard_health_up` | gauge | — | 1 when the overall status is `pass`, else 0 |
| `dashboard_health_status` | gauge | — | Overall status: 2 pass, 1 warn, 0 fail, −1 unknown |
| `dashboard_health_check` | gauge | `check`, `status` | 1 when that check passes, else 0 |
| `dashboard_health_check_last_duration_seconds` | gauge | `check` | Duration of the check's most recent execution |
| `dashboard_health_latency_ms` | gauge | — | Wall-clock time of the last check batch, milliseconds |
| `dashboard_health_shutting_down` | gauge | — | 1 once the probe is marked for shutdown |
| `dashboard_sse_connections` | gauge | — | Currently connected SSE clients |
| `dashboard_pusher_active` | gauge | — | 1 while the SSE pusher goroutine is running |
| `dashboard_health_check_duration_seconds` | histogram | — | Batch durations across ticks (see buckets below) |
| `dashboard_webhook_deliveries_total` | counter | `result` | Webhook deliveries by result (`ok` / `error`) |
| `dashboard_webhook_delivery_duration_seconds` | histogram | — | Webhook delivery duration |

Conditional families:

- `dashboard_health_check_duration_seconds` renders whenever metrics are
  enabled (the histogram is observed once per pusher tick).
- The two `dashboard_webhook_*` families render **only when
  `WithWebhook` is configured**; without a webhook URL they are absent
  from the document entirely.

## Encoding details

- `dashboard_health_status` mirrors go-health's `Status.Rank` severity
  ladder (fail 0 < warn 1 < pass 2). An unreadable status maps to −1 —
  deliberately stricter than the merge-time "unknown → pass" default, so
  a scrape can never claim a broken response is healthy and can never
  collide with fail's 0.
- The `status` label on `dashboard_health_check` carries the raw
  go-health status text (`pass` / `warn` / `fail`), not the numeric
  encoding.
- `dashboard_health_check_last_duration_seconds` is a **gauge** (last
  observation), not a summary; use the histogram for distribution
  questions.

## Histogram buckets

Both histograms use the same fixed cumulative bounds (seconds):

```
0.005 0.01 0.025 0.05 0.1 0.25 0.5 1 2.5 5 10 +Inf
```

chosen to span fast local checks through slow timeouts. `+Inf`, `_sum`,
and `_count` series are emitted per the format.

## Cardinality guarantees

- Every label is **bounded by configuration, not traffic**:
  - `check` = one series per registered health check (plus its `status`
    twin on `dashboard_health_check`).
  - `result` = exactly two series (`ok`, `error`).
  - Everything else is label-free.
- There are no per-client, per-connection, per-remote-status, or
  timestamp-derived labels. Series count grows only when you add health
  checks.
- Under `WithPublicMode`, masked `check-N` names replace raw check names
  in labels, so the cardinality is unchanged and the label values carry
  no identifying information.

## Scrape guidance

- Scrape interval ≥ the push interval is a natural fit: every pusher tick
  refreshes all gauges and observes the histogram once.
- `dashboard_pusher_active == 0` with `dashboard_health_up == 1` means
  the page is stale by definition (no pusher → no patches): alert on it.
- See `deploy/` for a working Grafana dashboard provisioned against
  these series, including a commented threshold-alert example.
