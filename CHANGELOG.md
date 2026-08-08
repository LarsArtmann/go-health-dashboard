# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.0] - 2026-08-08

First proper release. A real-time, browser-friendly health dashboard that
composes [go-health](https://github.com/larsartmann/go-health),
[templ-components](https://github.com/larsartmann/templ-components),
[go-datastar](https://github.com/larsartmann/go-datastar), and
[go-sse](https://github.com/larsartmann/go-sse). All module dependencies now
resolve from published versions; the local `replace` directives used during
development have been removed.

### Added

- `WithCSSPath` option to use compiled CSS instead of the Tailwind Play CDN
  (`dashboard.go:77`)
- `WithHeartbeatInterval` option for configurable SSE keepalive interval
  (default 15s) (`dashboard.go:84`)
- `WithMaxSSEConnections` option to limit concurrent SSE clients, returning
  HTTP 503 when exceeded (DoS prevention) (`dashboard.go:91`)
- `SubscriberCount()` method for SSE connection observability
  (`dashboard.go:269`)
- Favicon endpoint (`/favicon.svg`) with embedded SVG green-heart icon
  (`favicon.go:13`)
- Dark mode toggle button in dashboard header (`view.templ:36`)
- `docs/DOMAIN_LANGUAGE.md` with ubiquitous language glossary
- CI/CD GitHub Actions workflow: build, test with race detector, lint,
  vulncheck (`.github/workflows/ci.yml`)
- Dependabot config for Go modules and GitHub Actions
  (`.github/dependabot.yml`)
- Comprehensive `.golangci.yml` (80+ linters, pragmatic test/example
  exclusions, 0 issues)
- SSE integration tests: change detection, resilience, connection limit,
  multi-client fan-out (`sse_integration_test.go` — 10 tests)
- CSP nonce verification tests (`csp_test.go` — 9 tests)
- Content negotiation q-value tests (RFC 7231 wildcards, equal-q-value
  defaults) (`dashboard_test.go`)
- README badges: CI status, Go Reference, Go Report Card, License
- Example app `PORT` environment variable support
- MIT LICENSE (replaced PROPRIETARY)

### Changed

- **Breaking (internal):** `Dashboard.push` field changed from `*pusher` to
  `atomic.Pointer[pusher]` to fix a production data race (`dashboard.go:107`)
- Content negotiation rewritten with full RFC 7231 §5.3.2 q-value parsing,
  replacing naive `strings.Contains` check (`dashboard.go:190`)
- `RegisterRoutes` now conditionally registers favicon route (skips when empty)
  (`dashboard.go:301`)

### Fixed

- Production data race: concurrent `Shutdown()` writing `push = nil` while
  `sseHandler` read `d.push` without synchronization — now uses atomic
  pointer
- CSP nonce rendering: script tags no longer render empty `nonce=""` attribute
  when no nonce is configured (fixed in templ-components SDKScript,
  ThemeScript, and ThemeToggle)

## [0.1.0-alpha] - 2026-08-08

### Added

- Initial project structure: `go.mod`, `doc.go`, `example/`
- `status.go` — status mapping layer: `mapStatusToBadge`,
  `mapStatusToFeedback`, `mapStatusToText`, `groupChecks`, `buildViewModel`,
  `rowsToTableRows`, `fingerprintChecks`
- `view.templ` — full HTML dashboard with `layout.Base`, StatCards grid,
  severity-grouped Cards with Tables, Alert banner
- `routes.go` — `Routes` struct and `DefaultRoutes()`
- `dashboard.go` — `Dashboard` struct, `Config`, `Option` type, `New()`,
  `Handler()`, `RegisterRoutes()`, `Start()`, `Shutdown()`
- `pusher.go` — SSE pusher goroutine with broadcaster fan-out,
  `PushOnChange`/`PushAlways` modes, Datastar `ElementsFromTempl` patches
- Content negotiation on `/health`: `Accept: application/json` returns JSON
  health response (200 for pass/warn, 503 for fail)
- Kubernetes probe endpoints wired from go-health: `/healthz`, `/readyz`,
  `/startupz`
- 37 tests (12 status + 25 dashboard), all passing with `-race`
- `README.md`, `AGENTS.md`, `CONTRIBUTING.md`, `CHANGELOG.md`

### Changed

- **Breaking:** Migrated from HTMX polling to Datastar SSE as the sole
  real-time transport. Removed `RefreshMode` type, `WithRefreshMode` option,
  `Partial` route, `partial.templ`, and `handlers.go` content-negotiation
  handler
- **Breaking:** Renamed `mapStatusToAlert` to `mapStatusToFeedback`; now uses
  canonical `feedback.FeedbackType` instead of deprecated
  `feedback.AlertType`
- `viewModel` struct: removed `PartialURL`, `Every`, `SSE`, `AlertType`;
  added `SSEURL`, `DatastarNonce`, `TailwindNonce`, `FeedbackType`
- `Routes` struct: removed `Partial` field; now has Dashboard, SSE,
  Liveness, Readiness, Startup
- `layout.Base` called with `HTMXVersion: ""` to disable HTMX injection
  entirely
- `buildViewModel` signature changed from
  `(resp, title, partialURL, every)` to `(resp, title, sseURL)`

### Removed

- `partial.templ` and `partial_templ.go` — HTMX polling fragment
- `handlers.go` — old content-negotiation handler
- `realtime.go` — old SSE pusher (replaced by `pusher.go`)
- `hashChecks` function — non-deterministic, replaced by `fingerprintChecks`

### Fixed

- Non-deterministic change detection: `hashChecks` iterated a Go map
  (randomized order), causing `PushOnChange` to broadcast every tick.
  Replaced by `fingerprintChecks` which sorts keys before concatenating
  (`status.go:215`)
