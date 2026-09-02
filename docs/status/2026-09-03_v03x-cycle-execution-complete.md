# v0.3.x Cycle — Execution Complete

Date: 2026-09-03
Base: 071c251 (post TODO-sweep) → HEAD at time of writing
Verdict: **every actionable item from the Pareto plan and TODO_LIST is
implemented, tested, and verified.** Remaining ideas live in `ROADMAP.md`.

## Release

- **v0.3.1 shipped** (`d453c52`, proxy-verified). The stray v0.3.0 tag was
  already cached by the Go module proxy and could not be moved; it is now
  honestly documented in the CHANGELOG (`## [0.3.0] - 2026-08-10`) and the
  cycle shipped as v0.3.1.

## What shipped this cycle

| Area | Deliverables |
| ---- | ------------ |
| API | `RecommendedCSP`, `WithShutdownDrain`, `WithMaxConnectionLifetime`, `WithRateLimit`, `WithDescription`, `WithPublicMode` |
| Endpoints | `/health/trend` (samples + transitions JSON), `/health/export` (JSON/CSV), latency histogram in `/health/metrics` |
| UI | Status Changes timeline card, `Updated <time>` refresh stamp, OG tags |
| Tests | 4 fuzz targets + nightly workflow, official-parser metrics conformance (+promtool passthrough), live SSE patch browser test, console/CSP error capture, axe-core a11y audit, delimiter-collision fingerprint fix |
| CI | Browser job (real Chrome), coverage totals, nightly fuzz |
| Tooling | gopls env fix (.vscode), benchmarks (metrics/patch render), dark-mode screenshot, example app env toggles, Dockerfile + docker-compose Prometheus demo |
| Docs | README (toggles, routes, CSP helper), FEATURES/ROADMAP/AGENTS/TODO_LIST sweep, decision notes, upstream issue filed (templ-components#6) |

## Verification (all green)

- `nix flake check` — all checks passed
- `nix run .#build` / `.#test` / `.#test-race` / `.#vet` / `.#lint` — 0 issues
- Fuzz smoke: 4 targets × 5s — no failures
- Browser suite ×3 consecutive runs — stable
- v0.3.1 live on the Go module proxy

## Notable finds during execution

1. Real fingerprint collision (delimiter aliasing in `fingerprintChecks`)
   — found by writing the fuzz target, fixed with length-prefixing.
2. Headless-Chrome startups contend when parallelized on loaded machines —
   browser tests now serialize through a mutex.
3. nixpkgs' prometheus 3.x no longer ships promtool — conformance is
   anchored on the official Go parser instead, promtool stays optional.
4. templ-components StatCard `<dl>` structure violates axe's
   `definition-list` rule — verified at source, filed upstream
   (templ-components#6), tolerated (documented) in this repo's audit.

## Deliberately not done (with reasons)

- Trend sparkline transition *markers* (visual): transitions are exposed
  via the trend JSON; drawing markers on the SVG sparkline is polish left
  to a future cycle.
- WebSocket transport: rejected in the design spike (SSE suffices; see
  decision notes).
- Build-tag gating for SSE (BLOCKED on user decision, unchanged —
  requires accepting the jsonv2 requirement, forking go-sse, or gating).
