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
returns a `Status` and an optional error message. The dashboard renders
checks grouped by severity (default) or by source (`WithGrouping`).

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

| Route                 | Purpose                                        | Content Type                          |
| --------------------- | ---------------------------------------------- | ------------------------------------- |
| `/health`             | HTML dashboard (or JSON via Accept)            | text/html or JSON                     |
| `/health/sse`         | SSE patch stream                               | text/event-stream                     |
| `/favicon.svg`        | Dashboard favicon                              | image/svg+xml                         |
| `/health/metrics`     | Prometheus exposition (opt-in)                 | text/plain                            |
| `/health/trend`       | History samples + transitions (opt-in)         | application/json                      |
| `/health/export`      | History export, JSON/CSV/NDJSON (opt-in)       | application/json, text/csv, x-ndjson  |
| `/health/introspect`  | Resolved-config document (opt-in)              | application/json                      |
| `/health/datastar.js` | Embedded SDK bundle (opt-in)                   | text/javascript                       |
| `/healthz`            | Kubernetes liveness probe                      | application/json                      |
| `/readyz`             | Kubernetes readiness probe                     | application/json                      |
| `/startupz`           | Kubernetes startup probe                       | application/json                      |

## Presentation Terms (0.7.x UI)

Terms the rendered page and its copy lean on; load-bearing in tests too.

### Group

One card (or collapsible section) of related checks. Two grouping axes
(`WithGrouping`): **severity** — Critical Failures, Non-Critical Issues,
Healthy Services (default); **source** — one card per aggregate `source/check`
prefix, worst-of status per card, plain keys in a fallback Services card.

### source/check

The namespaced key an aggregate produces for a merged probe's check (e.g.
`api/postgres`). The source prefix is load-bearing: short-name rendering and
source grouping both preserve it verbatim.

### Short display name

The condensed table label for a check (`handlers.Handlers` instead of the
fully-qualified type). Presentation-only and lossless: the raw key stays
recoverable via the title attribute and a monospace details line. Generic
type parameters (`store.Store[string]`) pass through unchanged.

### Healthy-group collapse

The healthy group renders as a native `<details>` section and auto-collapses
at `WithHealthyGroupCollapse(n)` rows (default 8) so a wall of green never
buries problems. Severity mode only — source-grouped pages have no single
healthy group.

### Collapse persistence

`WithPersistCollapse` stores the healthy group's open/closed state in
localStorage and re-applies it after every SSE patch and reload. Storage keys
off summary clicks (user intent); a scoped MutationObserver re-applies after
patch merges.

### Client-side filter

The search box (embedded-SDK setups): rows hide via `data-class:hidden` while
the query signal matches neither short nor raw name. The no-match hint is a
`role="status"` region so screen readers announce it.

### Connection pill

The live / reconnecting / offline indicator, driven by the SDK's
`datastar-fetch` lifecycle events. Equal-width states prevent layout shift;
state changes are announced via `aria-live`.

### Jump-to-problems

The anchor from the status banner to `#group-problems` (the first
failing/warning card), so operators land on what hurts.

### Sample / Transition

A **sample** is one recorded overall-status point (`{At, Value, Status}`);
**transitions** are derived flips between statuses. Samples power the trend
sparkline, `/health/trend`, `/health/export`, and the timeline card.
