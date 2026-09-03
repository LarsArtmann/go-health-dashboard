# Domain Language

Ubiquitous language for the go-health-dashboard package.

## Core Concepts

### Probe

The central health-checking engine from [go-health](https://github.com/larsartmann/go-health).
Runs registered service checks on a refresh interval, caches the aggregated
result, and exposes JSON probe handlers for Kubernetes. The dashboard reads
from the probe but never writes to it.

### Check

A single named service health check registered with the probe. Each check
returns a `Status` and an optional error message. The dashboard renders checks
grouped by severity.

### Status

The health state of a check or the overall system. Three values:

| Value  | Meaning                                    |
| ------ | ------------------------------------------ |
| `pass` | Service is healthy and responding normally |
| `warn` | Non-critical issue detected                |
| `fail` | Critical failure — service is unavailable  |

### Response

The probe's cached health snapshot, containing: overall `Status`, per-service
`Checks` map, `Version`, `Uptime`, `TotalLatencyMs`, and `ShuttingDown` flag.

## Dashboard-Specific Terms

### Dashboard

The top-level type that renders a browser-friendly health view from a `Probe`.
Owns the pusher lifecycle and route registration. Created via `New(probe, opts...)`.

### Config

Construction-only configuration populated by `Option` functions (`WithTitle`,
`WithPushInterval`, `WithPushMode`, `WithNonce`, `WithCSSPath`, etc.).
Immutable after `New()` returns.

### Pusher

An internal goroutine that periodically reads `probe.CachedResponse()`, renders
the dashboard content as a Datastar patch, and broadcasts it to all connected
SSE clients via a `Broadcaster`. Only one pusher runs per Dashboard instance.

### PushMode

Controls when the pusher sends updates:

| Value          | Behavior                                                     |
| -------------- | ------------------------------------------------------------ |
| `PushOnChange` | Broadcast only when status or check results change (default) |
| `PushAlways`   | Broadcast on every tick, regardless of changes               |

### Fingerprint

A deterministic string hash of all check names, statuses, and error messages.
Used by `PushOnChange` to detect whether anything changed between ticks. Keys
are sorted before hashing to ensure deterministic output (Go map iteration
order is randomized).

### SSE Handler

The HTTP handler at `/health/sse` that upgrades to a Server-Sent Events
connection, sends the initial dashboard state as a Datastar patch, then
forwards broadcaster events to the client. Includes a heartbeat keepalive to
prevent proxy timeouts.

### Broadcaster

A fan-out hub from [go-sse](https://github.com/larsartmann/go-sse) that
distributes `sse.Event` values to all subscribed SSE streams. The pusher
broadcasts to it; each SSE connection subscribes to it.

### Datastar Patch

A DOM mutation instruction in the Datastar protocol. The dashboard uses
`ElementsFromTempl` with `WithModeInner` to replace the content inside the
`#health-region` element on each tick.

### LiveRegion

A templ-components wrapper (`datastar.LiveRegion`) around the dashboard
content that auto-connects to the SSE endpoint and applies incoming patches.
Has an `aria-live` attribute for screen-reader accessibility.

### ViewModel

The template-ready representation of a health `Response`. The `buildViewModel`
function transforms a `Response` into a `viewModel` by mapping statuses to
display types, grouping checks by severity, and sorting alphabetically.

### Sample

One timestamped status observation in the pusher's history: `sample{At, Value,
Status}` where `Value` maps pass=1, warn=0.5, fail=0 (unknown=0). Recorded on
every pusher tick when `WithTrend` is enabled, before change detection.

### History Buffer

The mutex-guarded ring buffer in the pusher that retains the last n samples.
Powers the sparkline card, `/health/trend`, `/health/export`, the Status
Changes timeline card, and the `Updated <time>` stamp. Transitions are derived
on demand (`historyBuffer.transitions()`), never stored.

### Status Transition

A derived change point between consecutive samples (e.g. pass → fail with a
timestamp). The raw data behind the timeline card and the `/health/trend`
transitions array.

### Latency Histogram

`dashboard_health_check_duration_seconds` — a hand-rolled, fixed-bucket,
cumulative Prometheus histogram (with `_sum` and `_count`) of check-batch
durations in the metrics exposition. Zero-dependency by design.

### Shutdown Drain

`WithShutdownDrain(d)` behavior: `Shutdown()` first swaps the pusher to nil
(new SSE connections get an immediate 503), then waits up to `d` for existing
subscribers to disconnect before closing the broadcaster. Bounded by design.

### Rate Limiter

`WithRateLimit(max, window)` — a hand-rolled shared token bucket across
dashboard-owned routes. Returns 429 with `Retry-After`; Kubernetes probes are
exempt. Deliberately not `golang.org/x/time/rate` to keep zero runtime deps.

### Public Mode

`WithPublicMode()` — presentation-only anonymization: check names and error
details become generic `check-N` labels in the rendered HTML and metric
labels. The `/health` JSON response and kubelet probes intentionally stay
verbatim.

### Content Negotiation

The `/health` endpoint inspects the `Accept` header using RFC 7231 q-value
parsing. `Accept: application/json` returns the raw health response as JSON;
any other value renders the HTML dashboard. Equal q-values default to HTML.

### BasePath

`WithBasePath(prefix)` — a route prefix applied to all configured routes so
the dashboard can mount under a non-root path (e.g. `/admin` produces
`/admin/health`, `/admin/health/sse`). Call it after `WithRoutes`; a later
`WithRoutes` replaces the prefixed set.

### RetryInterval

`WithRetryInterval(d)` — the SSE `retry` field (milliseconds) sent to
browsers so they know how long to wait before reconnecting. Zero (default)
uses the browser's built-in delay; negatives are clamped to zero.

### SubscriberCount

`Dashboard.SubscriberCount()` — the number of active SSE connections, from
the pusher's atomic counter. Returns 0 when the pusher has not been started.

## Route Layout

| Route             | Purpose                             | Content Type      |
| ----------------- | ----------------------------------- | ----------------- |
| `/health`         | HTML dashboard (or JSON via Accept) | text/html or JSON |
| `/health/sse`     | SSE patch stream                    | text/event-stream |
| `/favicon.svg`    | Dashboard favicon                   | image/svg+xml     |
| `/health/metrics` | Prometheus exposition (opt-in)      | text/plain        |
| `/health/trend`   | History samples + transitions (opt-in) | application/json |
| `/health/export`  | History export, JSON or CSV (opt-in)   | application/json or text/csv |
| `/healthz`        | Kubernetes liveness probe           | application/json  |
| `/readyz`         | Kubernetes readiness probe          | application/json  |
| `/startupz`       | Kubernetes startup probe            | application/json  |
