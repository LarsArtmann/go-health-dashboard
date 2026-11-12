# Upstream filings — re-verification packet (T26/M25, 2026-11-12)

Status: PACKET ONLY. Nothing filed; filing stays owner-gated (the Blocked
rows). Each draft re-verified TODAY per verify-before-filing against the
CURRENT sources, with a verdict: FILE-WORTHY, STALE, or REFUTED.

## 1. templ gofumpt output — STALE, do not file

`docs/upstream/templ-gofumpt-generated-output-issue-draft.md` claims
`go tool templ generate` emits non-gofumpt-clean Go (verified 2026-09-23
against templ v0.3.1020).

**Re-verification 2026-11-12:** on the CURRENT pin (templ v0.3.1100 per
go.mod's tool directive + current gofumpt), `go tool templ generate`
followed by `gofumpt -l .` reports NOTHING — generated output is
gofumpt-clean. Whether upstream fixed it or gofumpt relaxed, the claim
does not reproduce. Filing would embarrass.

**Disposition:** retire the draft (keep as a historical note with this
verdict appended); drop the pending-filing framing.

## 2. erraudit nolint-audit directory shape — FILE-WORTHY

`docs/upstream/erraudit-nolint-audit-pattern-noop-issue-draft.md` claims
`erraudit nolint-audit ./...` reports zero directives while
`erraudit nolint-audit .` finds them (own-repo tracker, verified
2026-09-23 against erraudit 1c6809a).

**Re-verification 2026-11-12:** REPRODUCED on erraudit `5c841cf` (newer
than the draft's pin): `./...` → "No //nolint:erraudit directives
found.", `.` → 4 NEEDED (handlers.go:138 et al). The `./...` shape is the
idiomatic one every Go tool accepts, so the silent zero stays a trap.
The AGENTS gotcha ("takes a DIRECTORY (.), not ./...") documents the
workaround; the bug itself persists upstream.

**Disposition:** valid; file when authorized (own repo, bug-report
register, draft body as written).

## 3. StatCard green-tone contrast — REFUTED in current framing

TODO row + ADR-0006 framing: "Upstream's green tile ships `text-green-600`,
which fails WCAG on white and locks every consumer out of the green tone."

**Re-verification 2026-11-12:**
- The upstream source DOES ship it (templ-components
  `display/card_templ.go:782`, `StatToneGreen` →
  `text-green-600 dark:text-green-400`) — the source claim stands.
- The WCAG claim does not: the tone colors a DECORATIVE aria-hidden icon
  glyph, and computed ratios put it ABOVE the applicable floor —
  green-600 on green-50 = 3.15:1, on white = 3.30:1 (WCAG 1.4.11
  non-text floor 3:1: PASS; only the 4.5:1 TEXT floor would fail, and
  that floor governs status-bearing text, not decorative glyphs). The
  2026-11-04 eleventh-movement re-audit already moved this class
  (decorative aria-hidden icons → non-text floor) when it re-scoped the
  contrast scan.

**Disposition:** do not file. The remaining consumer-side truth ("a
consumer that wants a green STAT TEXT tile would inherit a failing
color") is a consumer decision, not an upstream defect. Close the Blocked
row as refuted; `TestRender_ContrastSafeStatusColors` stays as the local
ban (this dashboard keeps its own safer palette).

## Filing state (F121)

| Draft | Verdict | Filed? |
| ----- | ------- | ------ |
| templ gofumpt | STALE | No — retired |
| erraudit nolint-audit | Valid | No — awaits authorization |
| StatCard contrast | REFUTED | No — row closes as refuted |
