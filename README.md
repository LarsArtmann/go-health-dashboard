# go-health-dashboard

<p align="center">
<a href="https://pkg.go.dev/github.com/larsartmann/go-health-dashboard"><img src="https://pkg.go.dev/badge/github.com/larsartmann/go-health-dashboard.svg" alt="Go Reference"></a>
<a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License: MIT"></a>
</p>

A real-time health dashboard for [go-health](https://github.com/larsartmann/go-health),
powered by [Datastar](https://data-star.dev) SSE. Drops into your mux with one
call and gives you a live status page with green/yellow/red badges, severity
grouping, and sub-second updates.

## What It Does

- **Browser visits `/health`**: sees a rich dashboard with status banners, service
  tables, and badges — updating in real-time via Datastar SSE.
- **Kubelet hits `/readyz`**: gets the JSON readiness response from go-health.
- **Content negotiation on `/health`**: browsers get the HTML dashboard, `Accept: application/json` gets JSON. Kubelet probes (`/readyz`, `/healthz`, `/startupz`) are JSON-only.

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

    health "github.com/larsartmann/go-health"
    dashboard "github.com/larsartmann/go-health-dashboard"
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
    _ = dash.Start(ctx)
    defer dash.Shutdown()

    mux := http.NewServeMux()
    dash.RegisterRoutes(mux, dashboard.DefaultRoutes())

    http.ListenAndServe(":8080", mux)
}
```

Open `http://localhost:8080/health` in a browser. Done.

## Routes

| Path          | Method | Content-Type                  | What It Does                                                                                                              |
| ------------- | ------ | ----------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| `/health`     | GET    | text/html or application/json | HTML dashboard (default) or JSON health response (Accept: application/json). JSON returns 503 when critical services fail |
| `/health/sse` | GET    | text/event-stream             | SSE endpoint (Datastar patch protocol)                                                                                    |
| `/healthz`    | GET    | application/json              | Liveness probe (always 200, no dependency checks)                                                                         |
| `/readyz`     | GET    | application/json              | Readiness probe (503 when critical services fail)                                                                         |
| `/startupz`   | GET    | application/json              | Startup probe (latched once all critical services pass)                                                                   |

## Options

```go
dash := dashboard.New(probe,
    dashboard.WithTitle("My Service"),                        // Page title
    dashboard.WithPushInterval(5*time.Second),                 // SSE push interval
    dashboard.WithPushMode(dashboard.PushOnChange),            // Only push on change (default)
    // dashboard.WithPushMode(dashboard.PushAlways),            // Push on every tick
    dashboard.WithNonce("abc123"),                             // CSP nonce
    dashboard.WithRoutes(dashboard.Routes{
        Dashboard: "/status",
        SSE:       "/status/sse",
        Readiness: "/ready",
        // ...
    }),
)
```

## How Real-Time Works

The dashboard uses [Datastar](https://data-star.dev) for real-time DOM updates:

1. The HTML page loads the Datastar SDK via a `<script>` tag
2. A `datastar.LiveRegion` div wraps the health content with `data-init="@get('/health/sse')"`
3. The Datastar SDK opens an SSE connection to `/health/sse`
4. A background pusher goroutine reads `probe.CachedResponse()` at the configured interval
5. On each update, the pusher renders the content as a Datastar element patch and broadcasts it
6. The Datastar SDK applies the patch, replacing the inner HTML of the LiveRegion

By default, `PushOnChange` mode only sends updates when the health status actually changes — minimizing SSE traffic for NOC monitors that stay connected for long periods.

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

| Dependency                                                          | Purpose                                          |
| ------------------------------------------------------------------- | ------------------------------------------------ |
| [go-health](https://github.com/larsartmann/go-health)               | Health-check Response, Probe, CachedResponse     |
| [templ-components](https://github.com/larsartmann/templ-components) | LiveRegion, SDKScript, Alert, Table, Badge, Card |
| [go-datastar](https://github.com/larsartmann/go-datastar)           | Datastar SSE patch protocol (ElementsFromTempl)  |
| [go-sse](https://github.com/larsartmann/go-sse)                     | SSE transport (Broadcaster, Stream)              |

## Dark Mode

The dashboard respects the user's OS dark-mode preference and includes a
toggle button for manual switching. The preference is persisted in
`localStorage`.

## License

MIT
