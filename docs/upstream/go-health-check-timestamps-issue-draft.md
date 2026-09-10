# Upstream issue draft — go-health: per-check `Since`/`Duration` metadata

> **Filed 2026-09-10 as [go-health#2](https://github.com/LarsArtmann/go-health/issues/2)** — claims re-verified against the v0.1.3 module cache before filing. Kept as the reasoning record.
>
> Draft provenance: prepared per the verify-before-filing discipline:
> every claim below was re-read from `types.go` in go-health v0.1.3 (module
> cache, 2026-09-09) before being written.

## What go-health exposes today (verified)

`Check` is the per-service health result served by all probe handlers:

```go
// types.go (v0.1.3)
type Check struct {
	// Status is the health status of this individual check.
	Status Status `json:"status"`
	// Error contains the failure message when Status is not pass.
	Error string `json:"error,omitempty"`
}
```

The `Response` carries aggregate metadata (`version`, `instance_id`,
`uptime`, timestamps on the response level), but the per-check record has
exactly two fields: a status enum and an error string.

## Why the dashboard misses it

The health dashboard renders one row per check. Operators consistently ask
three questions about a failing service: **what** is wrong (covered by
`Error`), **how long** has it been wrong, and **when** was it last checked.
The second and third questions are unanswerable from the wire shape — the
dashboard currently renders a placeholder for both, and the status-changes
timeline can only be derived from the dashboard's own sampling clock (a
client-side approximation that resets on restart and lies across restarts
of the dashboard process, not the checked service).

## Proposal

Two optional, additive fields on `Check`:

```go
type Check struct {
	Status Status `json:"status"`
	Error  string `json:"error,omitempty"`

	// Since is the time the check entered its current status. Zero means
	// unknown (the service did not report it); consumers must treat zero
	// as absent, not as a timestamp.
	Since time.Time `json:"since,omitempty"`
	// Duration is how long the most recent check execution took.
	Duration time.Duration `json:"duration,omitempty"`
}
```

Services already wrapping their checks in a closure can set both for free;
`omitempty` keeps every existing consumer and golden payload unchanged
(zero `time.Time` marshals to `"0001-01-01T00:00:00Z"`, which is why the
field should be a pointer or explicitly omitted by the encoder if strict
absence is preferred — open question for the maintainer).

## Adoption path for the dashboard

Once `Check.Since` ships, the dashboard can render a "failing since 14:02
(17m)" column, convert the status-changes timeline from "observed by the
dashboard" to "reported by the service", and collapse long-stable healthy
groups with an honest "stable for 6h" summary line. `Duration` feeds the
per-check latency display without instrumenting every service.
