# ADR-0006: Stat-Card Two-Family Tone System (Blue/Purple, No Green Tile)

**Date:** decided 2026-10-09 (UI revamp); ADR recorded 2026-10-25
**Status:** Accepted (shipped; pinned by `TestRender_ContrastSafeStatusColors`)
**Covers:** TODO_LIST UI-revamp harvest row "ADR: stat-card two-family tone
system" (report §f13/§f32); folds the tone rule into the dark-mode checklist

## Context

The 2026-10-09 UI revamp introduced icon stat cards (uptime verdict,
latency, freshness) in the page header. Upstream templ-components' StatCard
ships a green tone family whose tile text is `text-green-600` — a color we
measured at **3.30:1 on white** (WCAG relative luminance; normal text needs
4.5:1, computed 2026-09-10). We also measured `amber-600` at 3.19:1 and
`gray-400` at 2.54:1 — all three fail AA and were replaced page-wide by
`green-700`/`amber-700`/`red-600` (+ their `dark:*-400` variants, 5.31-8.79:1
on gray-800).

Every attempt to use the upstream green tile would either fail AA or force
us to locally override upstream markup per call site.

## Decision

Stat cards use exactly two tone families, both information-bearing rather
than verdict-bearing:

1. **Blue** (`text-blue-500/600`, `dark:text-blue-400`) — neutral telemetry:
   the uptime verdict icon, the latency bolt, endpoint links. Blue reads as
   "information" and carries no pass/fail connotation.
2. **Purple/violet accents** — secondary emphasis where a second hue is
   needed; same discipline: never a verdict.

Verdict colors (green/amber/red) remain reserved for the check rows,
badges, and alert surfaces where they ARE the verdict — and only in the
AA-passing shades (`green-700`, `amber-700`, `red-600` light; `*-400`
dark). The upstream green tone family is **banned page-wide**; this is
pinned by `TestRender_ContrastSafeStatusColors`, which fails the build on
`text-green-600`/`text-amber-600` reappearing anywhere in the rendered
page.

## Consequences

- The stat-card strip stays visually calm: problems carry color, telemetry
  stays blue. This mirrors the "only problems carry color" rule already
  used for group count chips (`groupCountTone`: warn = amber, fail = red,
  pass/off = neutral gray).
- Upstream's green tile remains unusable here until upstream fixes the
  contrast (verified + filed upstream separately; `docs/upstream/`).
- The dark-mode checklist rule: any new status-bearing text must use the
  measured pair (`*-700` light / `*-400` dark) and add a line to
  `TestRender_ContrastSafeStatusColors` in the same change.
