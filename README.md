# go-health-dashboard

A browser-friendly health dashboard for [go-health](https://github.com/larsartmann/go-health).
Drops one handler into your mux and gets a live status page with green/yellow/red
badges, auto-refresh, and content negotiation for kubelet compatibility.

## What It Does

- **Browser visits `/health`**: sees a rich dashboard with status banners, service
  tables, and badges — auto-refreshing every few seconds.
- **Kubelet hits `/health`**: gets the JSON readiness response (zero overhead).
- **Content negotiation via Accept header**: `text/html` renders the dashboard,
  `application/json` delegates to go-health's readiness handler.

## Why a Separate Repo?

go-health is a **single-dependency** library (`samber/do` only). Pulling in templ,
templ-components, go-datastar, and go-sse as transitive dependencies would destroy
that value proposition. The dashboard lives in its own module so consumers who
only want JSON health probes pay zero dependency cost.

## Quick Start

```go
package main

import (
    "context"
    "net/http"
    "time"

    "github.com/larsartmann/go-health-dashboard"
    health "github.com/larsartmann/go-health"
    "github.com/samber/do/v2"
)

func main() {
    ctx := context.Background()
    injector := do.New()

    // Register your services with samber/do...
    // do.ProvideNamed(injector, "database", NewDatabase)

    probe := health.New(injector,
        health.WithVersion("1.2.3"),
        health.WithCriticalServices("database"),
        health.WithRefreshInterval(2*time.Second),
    )
    _ = probe.Start(ctx)
    defer probe.Shutdown()

    dash := dashboard.New(probe,
        dashboard.WithTitle("My Service"),
    )

    mux := http.NewServeMux()
    dash.RegisterRoutes(mux, dashboard.DefaultRoutes())

    http.ListenAndServe(":8080", mux)
}
```

Open `http://localhost:8080/health` in a browser. Done.

## Routes

| Path              | Method | Content-Type | What It Does                                            |
| ----------------- | ------ | ------------ | ------------------------------------------------------- |
| `/health`         | GET    | HTML or JSON | Content-negotiated dashboard (Accept header)            |
| `/health/partial` | GET    | HTML         | HTMX polling partial (auto-refresh fragment)            |
| `/healthz`        | GET    | JSON         | Liveness probe (always 200, no dependency checks)       |
| `/readyz`         | GET    | JSON         | Readiness probe (503 when critical services fail)       |
| `/startupz`       | GET    | JSON         | Startup probe (latched once all critical services pass) |

## Options

```go
dash := dashboard.New(probe,
    dashboard.WithTitle("My Service"),                    // Page title
    dashboard.WithRefreshInterval(5*time.Second),          // Polling interval
    dashboard.WithRefreshMode(dashboard.RefreshModePoll),  // HTMX polling (default)
    // dashboard.WithRefreshMode(dashboard.RefreshModeSSE),  // SSE push mode
    // dashboard.WithRefreshMode(dashboard.RefreshModeOff),  // Static (no auto-refresh)
    dashboard.WithRoutes(dashboard.Routes{
        Dashboard: "/status",
        Readiness: "/ready",
        // ...
    }),
)
```

## Refresh Modes

| Mode                        | Mechanism               | Latency    | Use Case                       |
| --------------------------- | ----------------------- | ---------- | ------------------------------ |
| `RefreshModePoll` (default) | HTMX polling            | 2-5s       | Operators watching a dashboard |
| `RefreshModeSSE`            | Server-Sent Events push | Sub-second | NOC monitors, wall displays    |
| `RefreshModeOff`            | None (manual reload)    | N/A        | Debugging, static snapshots    |

### SSE Push Mode

For sub-second updates, enable SSE push:

```go
dash := dashboard.New(probe,
    dashboard.WithRefreshMode(dashboard.RefreshModeSSE),
)
if err := dash.Start(ctx); err != nil {
    log.Fatal(err)
}
defer dash.Shutdown()
```

A background goroutine reads the probe's cached response and pushes updates to
all connected SSE clients. Status change detection (default) means traffic only
flows when something actually changes.

## Build

**Requires `GOEXPERIMENT=jsonv2`** (the go-sse dependency uses `encoding/json/v2`).

```bash
# Using Nix (recommended)
nix run .#build
nix run .#test
nix run .#lint

# Manual
GOEXPERIMENT=jsonv2 go build ./...
GOEXPERIMENT=jsonv2 go test ./...
```

## Run the Example

```bash
GOEXPERIMENT=jsonv2 go run ./example
# Open http://localhost:8080/health
```

The example includes mock services: one always healthy, one flapping (alternates
pass/fail every 15s), and one always failing. Watch the dashboard update live.

## Status Mapping

| go-health Status | Badge Color      | Alert Banner                     |
| ---------------- | ---------------- | -------------------------------- |
| `pass`           | Green (success)  | "All Systems Operational"        |
| `warn`           | Yellow (warning) | "Degraded — Non-Critical Issues" |
| `fail`           | Red (error)      | "Unhealthy — Critical Failures"  |

## Dependencies

| Dependency                                                          | Purpose                                           |
| ------------------------------------------------------------------- | ------------------------------------------------- |
| [go-health](https://github.com/larsartmann/go-health)               | Health-check Response, Probe, CachedResponse      |
| [templ-components](https://github.com/larsartmann/templ-components) | Alert, Table, Badge, StatCard, Card, PolledRegion |
| [go-sse](https://github.com/larsartmann/go-sse)                     | SSE transport (push mode only)                    |

## License

MIT
