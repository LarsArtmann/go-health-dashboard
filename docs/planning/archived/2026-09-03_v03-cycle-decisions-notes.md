# v0.3.x Cycle — Decision Notes

Point-in-time record of the design decisions made while executing the
v0.3.1 cycle. Status reports are snapshots; these are the "why" behind the
code.

## Release engineering

### The stray v0.3.0 tag (2026-09-02)

A v0.3.0 tag existed on the remote pointing at `3b9abd8` (2026-08-10), a
commit whose `Version` constant still said `0.2.0` and whose CHANGELOG
still read `[Unreleased]`. The Go module proxy had already cached it, so
the tag could not be moved — Go module versions are immutable once
cached. Resolution:

- Kept the stray tag (remote and proxy stay consistent).
- Documented what that tag actually contains under `## [0.3.0] - 2026-08-10`
  in the CHANGELOG, including an explicit note that the section was
  written afterwards.
- Released the current cycle as v0.3.1.

Lesson: re-head the CHANGELOG and bump `Version` **in the release commit
that gets tagged**, not after.

## SSE hardening

- **Drain is opt-in and bounded** (`WithShutdownDrain`): Shutdown swaps the
  pusher to nil first (new connections get 503 immediately), then waits up
  to the drain window for subscribers to drop, then closes the
  broadcaster. Unbounded drains are how shutdowns miss deploy deadlines.
- **Connection lifetime** (`WithMaxConnectionLifetime`): server-side close
  only; the browser reconnects via SSE semantics and receives fresh state
  on connect, so no server-side session state is needed.
- **Watchdog is report-only** (`ErrPusherStale`): a wedged pusher is
  surfaced through `HealthCheck` but never restarted. Self-healing here
  would hide the underlying bug from operators; a container orchestrator
  restart is the right recovery level.
- **Rate limiting is a hand-rolled token bucket** (`WithRateLimit`):
  golang.org/x/time/rate would be the first runtime dependency of the
  module. ~40 lines of mutex + float tokens achieve the same semantics at
  this scale, and probe endpoints stay exempt either way.

## History / trend

- **Samples carry timestamps** (`sample{At,Value,Status}`): every downstream
  feature (trend JSON, CSV export, status timeline, refresh stamp) wanted
  timestamps; the refactor happened once, before any of them landed.
- **Transitions are derived, not stored**: `historyBuffer.transitions()`
  computes flips from samples on demand. Storing them would double the
  bookkeeping for data that is a pure function of the samples.
- **Latency histogram is fixed-bucket and hand-rolled**: official client
  semantics (cumulative buckets, `_sum`, `_count`) without the dependency.
  `_sum` accumulates atomically scaled by 1e6 to stay in int64.

## Accessibility

- The axe audit excludes the sr-only skip link from color-contrast (no real
  stylesheet in the harness) and tolerates the `definition-list` violation
  coming from templ-components `StatCard` — filed upstream, tracked in
  `docs/planning/archived/2026-09-03_issue-drafts.md`.
- Browser tests serialize on a mutex: parallel headless-Chrome startups
  were timing out on loaded machines. Reliability over a little wall time.

## Testing

- Fuzzing found a real fingerprint collision: a check name containing `:`
  or `;` could alias a different split of name/status/error bytes. Fixed
  with length-prefixed fields — the fuzz target plus a deterministic
  regression test both guard it.
- Metrics conformance runs against the official `prometheus/common` parser
  on every test run; `promtool check metrics` runs only when a promtool
  binary is on PATH (nixpkgs' prometheus 3.x no longer ships it).

## Design spikes (not implemented)

See the spikes section in `ROADMAP.md`; summary:

- **Federation**: expose the dashboard's own health to another instance's
  probe as a synthetic check. Preferred shape: a `FederatedProber` option
  in go-health, not the dashboard — the dashboard is a renderer.
- **WebSocket transport**: rejected for now. SSE covers one-way push with
  automatic browser reconnection; WebSocket adds bidirectional traffic the
  dashboard has no use for, plus a second protocol surface to secure.
