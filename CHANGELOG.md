# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Content negotiation on `/health`: `Accept: application/json` returns the full
  health response as JSON (200 for pass/warn, 503 for fail); all other Accept
  values render the HTML dashboard (`dashboard.go:127-175`)
- SSE change-detection via deterministic `fingerprintChecks` with sorted map keys
  (`status.go:202-224`)
- `PushMode` type with `PushOnChange` (default) and `PushAlways` modes
  (`pusher.go:14-27`)
- SSE heartbeat every 15s to prevent proxy timeouts (`pusher.go:31-34`)
- Initial state push on SSE connect so clients don't wait for the next tick
  (`pusher.go:147-153`)
- Functional options: `WithTitle`, `WithPushInterval`, `WithPushMode`,
  `WithNonce`, `WithRoutes` (`dashboard.go:36-62`)
- `Version = "0.1.0"` exported constant (`dashboard.go:20`)
- Example app with mock services: always-healthy, flapping (15s cycle),
  always-failing (`example/main.go`)
- Graceful shutdown state display: "Shutting Down — Draining Traffic"
  (`status.go:94-97`)
- flake.nix with full devShell: `GOWORK=off`, `GOEXPERIMENT=jsonv2`,
  golangci-lint, govulncheck, gosec, templ CLI (`flake.nix`)
- Nix apps: generate, test, test-race, build, vet, lint, coverage, vulncheck,
  security, example, clean (`flake.nix:87-138`)

### Changed

- **Breaking:** Migrated from HTMX polling to Datastar SSE as the sole real-time
  transport. Removed `RefreshMode` type, `WithRefreshMode` option, `Partial`
  route, `partial.templ`, and `handlers.go` content-negotiation handler
- **Breaking:** Renamed `mapStatusToAlert` to `mapStatusToFeedback`; now uses
  canonical `feedback.FeedbackType` instead of deprecated `feedback.AlertType`
- `viewModel` struct: removed `PartialURL`, `Every`, `SSE`, `AlertType`; added
  `SSEURL`, `DatastarNonce`, `TailwindNonce`, `FeedbackType`
- `Routes` struct: removed `Partial` field; now has Dashboard, SSE, Liveness,
  Readiness, Startup
- `layout.Base` called with `HTMXVersion: ""` to disable HTMX injection entirely
- `buildViewModel` signature changed from `(resp, title, partialURL, every)` to
  `(resp, title, sseURL)`

### Removed

- `partial.templ` and `partial_templ.go` — HTMX polling fragment
- `handlers.go` — old content-negotiation handler
- `realtime.go` — old SSE pusher (replaced by `pusher.go`)
- `hashChecks` function — non-deterministic, replaced by `fingerprintChecks`

### Fixed

- Non-deterministic change detection: `hashChecks` iterated a Go map (randomized
  order), causing `PushOnChange` to broadcast every tick. Replaced by
  `fingerprintChecks` which sorts keys before hashing (`status.go:202-224`)

## [0.1.0-alpha] - 2026-08-08

### Added

- Initial project structure: `go.mod`, `doc.go`, `example/`
- `status.go` — status mapping layer: `mapStatusToBadge`, `mapStatusToFeedback`,
  `mapStatusToText`, `groupChecks`, `buildViewModel`, `rowsToTableRows`
- `view.templ` — full HTML dashboard with `layout.Base`, StatCards grid,
  severity-grouped Cards with Tables, Alert banner
- `routes.go` — `Routes` struct and `DefaultRoutes()`
- `dashboard.go` — `Dashboard` struct, `Config`, `Option` type, `New()`,
  `Handler()`, `RegisterRoutes()`, `Start()`, `Shutdown()`
- Kubernetes probe endpoints wired from go-health: `/healthz`, `/readyz`,
  `/startupz`
- 37 tests (12 status + 25 dashboard), all passing with `-race`
- `README.md`, `AGENTS.md`, `CONTRIBUTING.md`, `CHANGELOG.md`
