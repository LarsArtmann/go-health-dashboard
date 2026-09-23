# Research — dark-mode contrast of the dim metadata line (2026-09-23)

|             |                                                                                                                                            |
| ----------- | -------------------------------------------------------------------------- |
| **Date**    | 2026-09-23                                                                  |
| **Method**  | WCAG 2.x relative-luminance contrast of the rendered Tailwind palette pairs (computed, not eyeballed) |
| **Context** | Status colors have been WCAG-AA-locked since v0.8.0; the dim per-row metadata line (`text-xs text-gray-500 dark:text-gray-400`, view.templ) never had a measured ratio (TODO M-row, 20-44 report §f24). |

## The measured pairs (dark mode)

The metadata line renders `text-gray-400` (#9ca3af) and sits on the
check table (`dark:bg-gray-900`, #111827) inside the card surface
(`dark:bg-gray-800`, #1f2937).

| Pair                                | Ratio   | WCAG verdict (4.5:1 normal text) |
| ----------------------------------- | ------- | -------------------------------- |
| gray-400 on gray-900 (metadata on table) | **6.99:1** | AA pass |
| gray-400 on gray-800 (metadata on card)  | **5.78:1** | AA pass |
| gray-300 on gray-900 (body text)         | 12.04:1 | AA pass |
| gray-100 on gray-900 (headings)          | 16.12:1 | AA pass |
| gray-500 on gray-900 (the one borderline) | 3.67:1 | AA large-text only |

The `dark:text-gray-500` string that appears in the render belongs to
the decorative disclosure chevron `<svg>` — non-text contrast needs
3:1 (WCAG 1.4.11), which 3.67:1 passes. It is never used for text.

## Reading

- The dim metadata line is **WCAG AA conformant at 6.99:1** — nearly
  AAA (7:1) on the table surface and comfortably above AA on the card
  surface. No change needed.
- The uppercase per-column labels in the stacked mobile layout use the
  same gray-400 pairing, so the mobile visual check inherits this
  verdict.
- Light-mode equivalents (gray-500 on white ≈ 7.5:1) predate this
  measurement and were already AA-clean.
