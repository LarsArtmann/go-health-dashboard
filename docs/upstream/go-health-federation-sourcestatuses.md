> [!NOTE]
> This filing was drafted by GLM-5.3 via Crush from an AI-run investigation, not at my request. When the failure traces to an external report, that source is linked in the body.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

**TL;DR:** `federation.Prober` cannot answer "is remote `api` healthy
overall?" — `aggregate.Aggregate` got `SourceStatuses()` for exactly that
question (v0.6.0), and the merge unification (v0.7.0) made federation the
odd one out. The concrete cost: a shutting-down remote contributes its own
all-pass checks and forces the merged roll-up to fail, so a per-remote card
folded from rows renders green under a red page, and no accessor exists to
fix it. Ask: the same `SourceStatuses() map[string]health.Status` on
`federation.Prober`.

## Problem

The merged view carries only namespaced checks (`name/check`), and a remote
can be unhealthy in a way no row shows. From `internal/merge/merge.go` at
master (`cf1238ca252b`), the drain case folds into the roll-up only:

```go
	if src.Response.ShuttingDown {
		shuttingDown = true
	}
	...
	if shuttingDown {
		status = health.StatusFail
	}
```

No fail row is added for that source — its checks pass through unchanged.
Measured in `go-health-dashboard` (v0.7.0, dashboard commit `1473fc2`): with
one source reporting all-pass checks under a shutting-down flag, a card that
folds rows renders pass while the merged response status is fail. The
dashboard's new `SourceStatuser` capability escalates card verdicts from the
accessor, which fixes aggregate consumers — federation consumers (the
`health-hub`) have no accessor to wire, so their draining-remote cards keep
lying until this lands.

Unreachable remotes are already honest: the merge adds the synthetic
`name/reachable` fail row, so a row fold sees the failure. The gap is
precisely the drain case (and any future per-source state that rides the
roll-up without a row).

## Goal

As a hub operator, I want to badge each remote card with the remote's own
roll-up: when remote `api` is draining, its card reads fail — same as the
page — without me refolding `api/*` keys and without the wire format
changing.

|                                 | `aggregate.Aggregate` (v0.6.0+) | `federation.Prober` (master) |
| ------------------------------- | ------------------------------- | ---------------------------- |
| Per-source roll-up accessor     | ✅ `SourceStatuses()`           | ❌                           |
| Shutting-down source reports    | `fail` for that source          | merged fail, no row          |
| Dark remote surfaces as         | n/a (in-process, trusted)       | `name/reachable` fail row    |

## Notes on shape

The design note (`docs/aggregate-per-source-visibility-design.md`, decided
2026-09-04) chose the programmatic map accessor over wire-format changes and
set a promotion condition: "If a second consumer appears earlier, promote
it." The dashboard is that second consumer now, and it consumes the accessor
through a capability interface (`SourceStatuses() map[string]health.Status`)
— identical signature on federation would let one interface cover both
merge surfaces.

One acquisition difference matters: aggregate does one atomic cache load per
source; federation's reads fetch. A federation `SourceStatuses()` would
trigger the same one-parallel-fetch-per-remote pass as `CachedResponse()`
(or share a single fetch — caller's choice), so its doc comment should state
the cost the same way the package doc states merge-on-read costs. Dark
remotes should report `fail` for themselves (mirroring the `reachable` row),
not error, to keep the map total.

Happy to contribute it — the aggregate implementation
(`aggregate/aggregate.go:155`, `v0.7.0`) is the template and the
`internal/merge` fold makes the two implementations stay in lockstep.

💘 Generated with Crush

<!--
internal (strip before posting):
repo: LarsArtmann/go-health (own repo)
title: federation: per-source roll-up accessor (SourceStatuses) to match aggregate
status: verified 2026-10-11 against v0.7.0 + master cf1238ca252b; pending Lars review
publish: gh issue create -R LarsArtmann/go-health --title "..." --body-file <extracted body>
precedent: docs/upstream/go-health-federation-mark-shutting-down.md (go-health#5)
-->
