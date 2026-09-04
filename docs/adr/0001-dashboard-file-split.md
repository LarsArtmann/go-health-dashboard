# ADR 0001: Split dashboard.go into options, handlers, and history files

Date: 2026-09-04
Status: Accepted

## Context

`dashboard.go` had grown to hold four distinct concerns: the `Dashboard`
type and its lifecycle, the `Config`/`Option` pattern with its dozens of
`With*` functions, HTTP handler wiring and middleware wrapping, and the
trend-history sample machinery. At ~700 lines the file had four reasons
to change and no single section a reader could hold in their head. Every
new option forced a diff in the same file as every handler change, which
made review noisy and merge conflicts routine.

## Decision

Split by concern, keeping the single `dashboard` package (no new module
or package boundary):

- `dashboard.go` — the `Dashboard` type, lifecycle (`Start`,
  `Shutdown`, `HealthCheck`), the `Prober` interface, and the `Version`
  const.
- `options.go` — `Config`, `Option`, and every `With*` function. One
  file where all configuration surface lives; options are the package's
  public contract and deserve one readable home.
- `handlers.go` — HTTP handler construction, `RegisterRoutes`,
  middleware wrapping, and content negotiation.
- `history.go` — the `historyBuffer` ring, `sample` type, and the
  sample→JSON mapping shared by the trend/export endpoints.

## Consequences

- Adding an option no longer touches lifecycle code; the diff is one
  file (or two with tests).
- `dashboard.go` now reads top-to-bottom in one sitting: version,
  interface, type, lifecycle.
- The package count stays at one: no public import paths changed, and
  the split is invisible to consumers. We explicitly rejected a
  `dashboard/options` sub-package — it would break the
  `dashboard.WithX()` call sites for zero functional gain.
- File order in AGENTS.md documents the layout; contributors should
  keep new options in `options.go`, not `dashboard.go`.
