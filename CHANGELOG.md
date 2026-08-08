# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- `WithCSSPath` option to use compiled CSS instead of the Tailwind Play CDN
- `WithHeartbeatInterval` option for configurable SSE keepalive interval (default 15s)
- `WithMaxSSEConnections` option to limit concurrent SSE clients (DoS prevention)
- `SubscriberCount()` method for SSE connection observability
- Favicon endpoint (`/favicon.svg`) with embedded SVG green-heart icon
- Dark mode toggle button in dashboard header
- `docs/DOMAIN_LANGUAGE.md` with ubiquitous language glossary
- CI/CD GitHub Actions workflow: build, test with race detector, lint, vulncheck
- Dependabot config for Go modules and GitHub Actions
- Comprehensive `.golangci.yml` (80+ linters, pragmatic test/example exclusions)
- SSE integration tests: change detection, resilience, connection limit
- CSP nonce verification tests
- Content negotiation q-value tests (RFC 7231 wildcards, equal-q-value defaults)
- README badges: CI status, Go Reference, Go Report Card, License
- Example app `PORT` environment variable support
- MIT LICENSE (replaced PROPRIETARY)

### Changed

- **Breaking (internal):** `Dashboard.push` field changed from `*pusher` to
  `atomic.Pointer[pusher]` to fix a production data race
- Content negotiation rewritten with full RFC 7231 §5.3.2 q-value parsing,
  replacing naive `strings.Contains` check
- `RegisterRoutes` now conditionally registers favicon route (skips when empty)

### Fixed

- Production data race: concurrent `Shutdown()` writing `push = nil` while
  `sseHandler` read `d.push` without synchronization — now uses atomic
  pointer
- CSP nonce rendering: script tags no longer render empty `nonce=""` attribute
  when no nonce is configured (fixed in templ-components SDKScript,
  ThemeScript, and ThemeToggle)
- `godoclint` false positive on `doc.go` excluded from lint config

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
