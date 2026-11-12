# AGENTS.md restructure outline (T30, 2026-11-12)

Target: the fleet 220-line budget (today: 553). Decision rides batch B
(B8). This outline is the one-approval change: every move below is a
mechanical cut-and-paste with a one-line pointer left behind, verified by
the docs battery and a fresh-session readability pass.

## What STAYS (operative, every-session context)

| Section                                                                                                                                                                                                                                           | Lines today    | Why it stays                                            |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------- | ------------------------------------------------------- |
| Commands                                                                                                                                                                                                                                          | 9–30           | The gate table is the first thing every session touches |
| Architecture file map                                                                                                                                                                                                                             | 31–68          | Navigation; one paragraph per file                      |
| Key Design Decisions — the ADR-index bullets only (Prober interface, SSE-first, content negotiation, sentinel family, atomic pusher pointer, single source of truth, middleware philosophy, lifecycle)                                            | ~40 of 69–135  | Architectural invariants a session must not violate     |
| Testing Patterns — the fixture doctrine, wall-clock margins, goroutine-scan doctrine, browser-environment rules                                                                                                                                   | ~50 of 149–173 | Breaking these breaks the suite                         |
| Gotchas — the ~14 load-bearing bullets (GOEXPERIMENT, GOWORK, templ pins + stamp guard, a-h/parse pairing, daemon races + recovery recipes, UI-pin guard, Datastar colon grammar, Chrome preflight, probe/dashboard Start, no replace directives) | ~90 of 174–430 | The ones that change HOW a session acts today           |
| Dependency Notes — one paragraph per dependency (wire surface + current pin + pairing rule)                                                                                                                                                       | ~25 of 431–552 | What consumers/integrators touch                        |

Estimated post-cut size: ≈ 250–270 lines (the honest target; B8's
recommendation accepts the residual warn).

## What MOVES (already duplicated or derivable)

| Content                                                                                                         | Lines      | Moves to                                                                                             | Pointer left                                                             |
| --------------------------------------------------------------------------------------------------------------- | ---------- | ---------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| Long-form gotcha narratives (the 30 bullets that END with "Details: `docs/gotchas.md`" or restate a war story)  | ~170       | `docs/gotchas.md` is ALREADY the long-form home — the AGENTS copies shrink to one sentence + pointer | `See docs/gotchas.md "<entry title>"`                                    |
| Per-dependency API inventory (go-health's method list, aggregate/federation capability notes, the v0.x history) | ~70        | NEW `docs/dependency-notes.md`                                                                       | One line per dependency in AGENTS                                        |
| Data Flow walkthrough (8 numbered steps)                                                                        | 136–148    | `docs/architecture.md` (new) — with the file map's detail level                                      | Two sentences in Architecture                                            |
| Historical decision rationale inside bullets (v0.x bump stories, the twelve UI movements, sweep/revert history) | interwoven | Already in `docs/gotchas.md` / CHANGELOG — delete duplication, keep the RULE                         | —                                                                        |
| Release-discipline war stories (v0.10.1/v0.12.x dual-cuts, workflow saga)                                       | interwoven | `docs/gotchas.md` (release section)                                                                  | One rule line: tag-first-then-master, gate 0 after edits, verify-release |

## Mechanics of the move

1. One change, generate → fmt NOT involved (docs only); docs battery +
   markdown lint + freshness gate must stay green — every moved bullet
   leaves either a pointer or is genuinely duplicated content.
2. The freshness gate parses TODO_LIST/ROADMAP/CHANGELOG, not AGENTS —
   no gate risk. The annotate gate only scans docs/status/**.
3. After the cut, run a fresh-session readability probe: can a new
   session answer "how do I run tests / what must I never do / where is
   the long form" from AGENTS.md alone in under a minute?

## Explicitly NOT moving

- The Commands table (fleet-standard shape).
- The pill-bullet, evidence-strip bullet, and filter-chips bullet —
  these encode invariants whose loss has already cost sessions (#31,
  health-washing, dead chips).
- The daemon-race recovery recipes — they are exercised weekly and live
  closest to where daemon commits happen.
