# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [0.5.0] - 2026-09-04

Integrate, don't compete. The dashboard now feeds monitoring platforms
(Prometheus, SigNoz, Gatus, Uptime Kuma) instead of trying to replace
them, and natively renders go-health's new multi-probe aggregate.
go-health is bumped to v0.1.0, which introduces the `aggregate`
sub-package this release consumes.

### Compatibility

- `dashboard.New` now accepts the new `Prober` interface instead of the
  concrete `*health.Probe`. Source-compatible: every existing caller
  keeps compiling unchanged (structural typing), and
  `*aggregate.Aggregate` satisfies the same interface.
- The dependency bumps to `github.com/larsartmann/go-health v0.1.0`.

### Added

- `WithWebhook(url)` and `WithWebhookHeaders(map[string]string)`:
  push a JSON status snapshot on every health transition (initial
  state included), independent of `PushMode`. Deliveries are
  best-effort (10s timeout, no retries, no logging) with bounded
  in-flight goroutines, so a slow receiver can never block the SSE
  push loop. `WithPublicMode` masks check names (`check-N`) and
  strips error strings from the payload. Docs:
  `docs/integrations.md`
- `Prober` interface: the minimal probe surface the dashboard
  renders, defined on the consumer side (Go convention). Satisfied
  by `*health.Probe` and go-health's `*aggregate.Aggregate`, enabling
  one dashboard to serve N in-process probes with `source/check`
  namespaced checks and worst-of status (`integration_test.go`
  verifies the aggregate end-to-end).
- Integration cookbook `docs/integrations.md`: validated recipes for
  Prometheus/SigNoz alerting (including the SigNoz `target=0` and
  `{{$value}}` traps), Gatus and Uptime Kuma endpoint configuration
  against go-health semantics, and the webhook payload contract.

### Changed

- `go.mod`: `github.com/larsartmann/go-health` v0.0.2 → v0.1.0.
- README: new "Monitoring Integrations" and "Multi-Service
  Dashboards (aggregate)" sections.

## [0.4.0] - 2026-09-03

Harden and expose. The dashboard gains opt-in SSE hardening (graceful
drain, connection lifetime, rate limiting, a pusher watchdog),
timestamped history with JSON and CSV export endpoints, a status
change timeline, a latency histogram, public status-page mode, and
a verified CSP helper.

### Compatibility

- The change-detection fingerprint encoding changed from
  delimiter-separated fields to length-prefixed fields (fixing a real
  collision).
  Fingerprints are in-memory and short-lived, so upgrades are seamless
  in practice; but a fingerprint persisted across an upgrade produces
  exactly one spurious change detection on the first tick.
- templ-components is pinned at v1.11.0: v1.12.0's new LiveRegion
  busy-script emits `nonce=""` (unguarded attribute), breaking no-nonce
  and strict-CSP renders. Reported upstream (templ-components#7); the
  pin lifts when the guard ships.

### Added

- `RecommendedCSP(nonce string) string` helper: returns the exact
  Content-Security-Policy value the headless-browser test verifies — strict,
  self-hosted, no `unsafe-inline`, with the `'unsafe-eval'` the Datastar SDK
  requires. Invalid nonce tokens (outside the CSP base64 alphabet) are
  omitted so the header can never be malformed (`csp.go`,
  `csp_policy_test.go`)
- Fuzz targets `FuzzEscapeLabelValue` (Prometheus label escaper: no raw
  newlines, lossless round-trip) and `FuzzFingerprintChecks` (change-detection
  fingerprint: deterministic, distinguishes any single-field mutation)
  (`fuzz_test.go`)
- Nightly fuzz workflow (`.github/workflows/fuzz.yml`): 60s per target on a
  03:00 UTC schedule plus manual dispatch, printing crasher inputs on failure
- CI browser job: the runtime CSP test now runs on every push/PR against a
  real Chrome (`GO_HEALTH_DASHBOARD_CHROME`), and the test job reports
  coverage totals (`.github/workflows/ci.yml`)
- Browser hardening: every headless-browser test now captures
  `console.error` calls and uncaught exceptions and fails on them (this
  catches CSP violations at runtime), a live-patch test proves a real
  service failure reaches the rendered DOM through the SSE stream without a
  reload under strict CSP (`TestBrowser_LiveSSEPatch`), and an axe-core
  audit (downloaded same-origin, skipped offline) enforces serious/critical
  accessibility violations plus targeted ARIA/landmark checks
  (`TestBrowser_Accessibility`)
- Dashboard presentation options: `WithDescription(description)` renders
  the meta description plus Open Graph tags (omitted by default), and
  `WithPublicMode()` anonymizes the rendered HTML and the metrics labels
  for untrusted audiences — check names and error details become generic
  `check-N` labels while statuses stay visible (`status.go`, `metrics.go`,
  `publicmode_test.go`)
- History features built on timestamped trend samples
  (`sample{At,Value,Status}`): `TrendHandler` serving samples plus derived
  status transitions as JSON at `Routes.Trend` (default `/health/trend`),
  `ExportHandler` for JSON/CSV export at `Routes.Export` (default
  `/health/export`, `?format=csv` or `Accept: text/csv`), a status-change
  timeline card in the UI, an "Updated <time>" refresh stamp, and a
  hand-rolled `dashboard_health_check_duration_seconds` histogram in the
  metrics exposition (`pusher.go`, `trend.go`, `metrics.go`, `view.templ`)
- SSE hardening options: `WithShutdownDrain(d)` (Shutdown rejects new
  connections immediately and waits up to d for existing clients before
  closing the broadcaster), `WithMaxConnectionLifetime(d)` (server closes
  streams past the cap; browsers reconnect and receive fresh state),
  `WithRateLimit(max, window)` (shared hand-rolled token bucket across
  dashboard-owned routes with 429 + Retry-After; probes exempt), and a
  report-only pusher watchdog in `HealthCheck` returning
  `ErrPusherStale` when the push loop stops ticking for three intervals
  (`ratelimit.go`, `hardening_test.go`, `watchdog_test.go`)
- Metrics conformance test: the exposition is parsed with the official
  `prometheus/common` text-format parser under strict legacy name
  validation, covering all seven metric families and label values with
  quotes/backslashes/newlines; a second test pipes a scrape through
  `promtool check metrics` when a promtool binary is on PATH
  (`metrics_test.go`). Note: nixpkgs' prometheus 3.x package no longer
  ships promtool, so the lint pass is opt-in via PATH while the parser
  check always runs

### Fixed

- `fingerprintChecks` delimiter collision: a check name containing `:` or
  `;` could alias a different split of the same bytes across name, status,
  and error (e.g. name `a:b` with status `c` collided with name `a` and
  status `b:c`), causing missed change detection. Fields are now
  length-prefixed so boundaries are unambiguous (`status.go`,
  `status_test.go`)
  **Compatibility:** this changes fingerprint values. Fingerprints are
  in-memory change-detection state only — nothing persisted or exposed — so
  the only visible effect after upgrading is at most one spurious broadcast
  on the first tick. Code that persisted fingerprints across versions (not a
  supported use) will see one false "changed" report.

## [0.3.1] - 2026-09-02

Protect, measure, trend. The dashboard gains consumer-supplied auth middleware,
an opt-in zero-dependency Prometheus metrics endpoint, a live health trend
sparkline, and runtime CSP verification in a headless browser — plus fuzzing
for the Accept parser and JSON serialization.

### Added

- `WithMiddleware(fn)` option: wraps every dashboard-owned route (dashboard
  HTML, SSE, favicon, metrics) with consumer-supplied middleware for auth
  integration. Kubernetes probe endpoints are deliberately not wrapped —
  the kubelet cannot authenticate (`dashboard.go`, `middleware_test.go`)
- `WithMetrics(enabled)` option and `Dashboard.MetricsHandler()`: opt-in
  Prometheus metrics endpoint at `Routes.Metrics` (default `/health/metrics`)
  serving hand-rolled text exposition 0.0.4 — zero new runtime dependencies,
  deterministic sorted output, proper label escaping (`metrics.go`,
  `metrics_test.go`)
- `WithTrend(n)` option: health trend sparkline card driven by a mutex-guarded
  ring buffer of status samples in the pusher (pass=1, warn=0.5, fail=0),
  rendered with `display.Sparkline` and updated via the normal SSE stream
  (`pusher.go`, `view.templ`, `trend_test.go`)
- `WithHideStatCards()` option: hides the version/uptime/latency stat card
  grid for compact dashboards (`dashboard.go`, `trend_test.go`)
- Fuzz targets `FuzzWantsJSON` (Accept-header parser) and
  `FuzzHealthResponseSerialization` (JSON round-trip + idempotence invariants)
  in `fuzz_test.go`
- Headless-browser runtime CSP test (`browser_test.go`): loads the dashboard
  under a strict CSP in Chromium via chromedp, proves the Datastar SDK
  connects via SSE, and asserts the runtime DOM stays free of CSP-relevant
  inline styles. Skips automatically when no Chrome is available; point it at
  a binary with `GO_HEALTH_DASHBOARD_CHROME`
- Screenshot capture test (`screenshot_test.go`): env-guarded
  (`SCREENSHOT_OUTPUT`) chromedp capture used to generate `docs/screenshot.png`
- README: embedded dashboard screenshot, "Protecting the Dashboard",
  "Prometheus Metrics", "Health Trend", and "Content-Security-Policy" sections

### Changed

- **Documented CSP requirement:** the Datastar SDK compiles its `data-*`
  expressions with the `Function` constructor, so `script-src` needs
  `'unsafe-eval'` next to the nonce; without it the bundle throws
  `GenerateExpression` during init and SSE never connects (found by the
  headless-browser test)
- Test dependencies: chromedp and `go-datastar/static` (embedded SDK bundle
  for hermetic browser tests) added to `go.mod`
- Test count: 78 → 120 top-level test functions across 12 test files

### Fixed

- `WithBasePath` no longer converts an intentionally empty `Routes.Metrics`
  into the bare prefix

## [0.3.0] - 2026-08-10

Self-hosted Datastar SDK, SSE reconnection, sub-path mounting, and samber/do
lifecycle integration. Note: this release was tagged on 2026-08-10 while the
CHANGELOG still read `[Unreleased]` and the Version constant still said
`0.2.0`; this section was written afterwards to document what the tag
actually contains.

### Added

- `WithDatastarSrc(src string)` option: serve a local copy of the Datastar
  runtime instead of the pinned jsdelivr CDN URL, keeping the dashboard fully
  offline and CSP-compliant under `script-src 'self'`. Falls back to the CDN
  URL when unset (`dashboard.go`, `csp_test.go`)
- samber/do lifecycle integration: `Register(injector, probe, opts...)` plus
  `do.HealthcheckerWithContext` / `do.Shutdowner` assertions, so the Dashboard
  participates in container-wide `do.HealthCheck` and `do.Shutdown` cascades
  (`di.go`, `lifecycle_test.go`)
- `WithRetryInterval(d time.Duration)` option and `Config.RetryInterval` field:
  sets the SSE retry field so browsers know how long to wait before reconnecting
  after a disconnect. Default zero uses the browser's built-in ~3s delay
  (`dashboard.go:121`)
- `WithBasePath(prefix string)` option: prefixes all dashboard and probe routes
  for mounting under a non-root path (e.g. `/admin` produces `/admin/health`,
  `/admin/health/sse`). Applied to whatever routes are currently configured
  (`dashboard.go:132`)
- SSE reconnection support: events carry the `retry` field when configured, and
  the SSE handler always sends current state on connect, so reconnecting clients
  immediately see the latest health status (`pusher.go:105`)
- `SubscriberCount()` tracking test: verifies count increments/decrements with
  connections and returns 0 when pusher not started (`sse_integration_test.go`)
- `WithHeartbeatInterval` keepalive test: verifies custom heartbeat sends SSE
  comment frames at the configured interval (`sse_integration_test.go`)
- SSE CSP nonce flow test: verifies SSE patches contain no `<script>` tags,
  confirming they are CSP-safe inner-HTML replacements (`sse_integration_test.go`)
- Sub-path mounting tests: verify routes are prefixed, default routes not
  registered, and HTML references the prefixed SSE URL (`dashboard_test.go`)
- `WithBasePath` / `WithRoutes` ordering tests: verify that `WithBasePath`
  after `WithRoutes` prefixes custom routes, and `WithRoutes` after
  `WithBasePath` replaces the prefixed set entirely (`dashboard_test.go`)
- `WithRetryInterval` default test: verifies that zero (the default) omits the
  `retry:` field from SSE events (`sse_integration_test.go`)

### Changed

- **Breaking:** `RegisterRoutes` signature changed from
  `(mux *http.ServeMux, routes Routes)` to `(mux *http.ServeMux)`. Routes are
  now read from `Config` as the single source of truth (`dashboard.go:364`)
- Test count: 67 → 78 top-level test functions; coverage 80.0% → 79.7%

### Fixed

- `Version` constant updated from `"0.1.0"` to `"0.2.0"` to match the released
  tag — it was left stale when v0.2.0 was tagged (`dashboard.go:27`)
- `RegisterRoutes` latent bug: the `routes` parameter could diverge from
  `Config.Routes`, causing the HTML to reference an SSE URL that didn't match
  the registered handler — now uses `Config.Routes` as single source of truth
- `WithRetryInterval` negative-duration guard: negative values are now clamped
  to zero instead of producing a nonsensical `uint` underflow
- Reconnection test race condition: replaced fixed `time.Sleep(150ms)` with
  deterministic event-driven waiting (keep stream open until state confirmed)

## [0.2.0] - 2026-08-09

Per-request CSP nonce extraction. A single construction-time nonce is
incompatible with strict per-request CSP policies in long-running services;
per-request nonces defend against nonce reuse. Backward compatible.

### Added

- `WithNonceExtractor(func(*http.Request) string)` option and
  `Config.NonceExtractor` field: host applications supply a per-request CSP
  nonce (e.g. `httputil.NonceFromRequest`) instead of a single
  construction-time nonce. The extractor takes precedence over `WithNonce`,
  with graceful fallback when it returns an empty string (`dashboard.go:84`)
- Per-request nonce extractor tests: applies per-request nonce, distinct nonce
  per request, falls back to fixed nonce, does not affect JSON response
  (`csp_test.go`)
- Render-cleanliness regression guards: `TestRender_AllScriptsCarryNonce`
  (every inline `<script>` carries the nonce) and `TestRender_NoInlineStyles`
  (zero `<style>` blocks, zero inline `style=` attributes) (`csp_test.go`)

### Changed

- **Breaking (internal):** `buildData` signature changed from no arguments to
  `(r *http.Request)` so the nonce can be read per-request (`dashboard.go:299`)

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
