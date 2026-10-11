> [!NOTE]
> This filing was drafted by GLM-5.3 via Crush from an AI-run investigation, not at my request. When the failure traces to an external report, that source is linked in the body.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

**TL;DR:** Every `ListNote` variant synthesizes its message from count
semantics, but some lists have honest message shapes none of the three
variants can produce — a rolling window needs "Showing the latest 5 of 12
status changes.", and the truncated variant's "Narrow your search to see
more." is false advice on a searchless list. Ask: a `CustomMessage` escape
hatch that reuses the shared shell (class/ARIA source of truth) while the
consumer owns the sentence.

## Problem

`go-health-dashboard`'s Status Changes card caps at 5 timeline entries and
needs to disclose the cap honestly. None of the three variants fits
(`display/list_note.templ`, master `a30aa6324c94`):

| variant     | message                                           | why it doesn't fit the timeline                                    |
| ----------- | ------------------------------------------------- | ------------------------------------------------------------------ |
| `truncated` | "Showing N of M. Narrow your search to see more." | there is no search; older entries age out by design, not by filter |
| `count`     | "Showing N items."                                | hides the total — the dropped-entries fact is the whole message    |
| `range`     | "Showing X–Y of Z."                               | reads as pagination; hides that the dropped side is the OLDEST     |

So the dashboard hand-rolls the note (`view.templ` `timelineNote`) and
copies the shell verbatim — `border-t border-gray-200 dark:border-gray-700
px-4 py-2.5 text-xs text-gray-500 dark:text-gray-400` + `role="status"` —
which is exactly the duplication `listNoteShell` exists to prevent: the
component restyles, the consumer's copy of the classes silently drifts.

## Goal

As a consumer with a message shape that is honest but not a count shape, I
want to keep `ListNote`'s shell, ARIA, and render/no-render discipline while
supplying my own sentence and my own visibility rule:

```go
@display.ListNote(display.ListNoteProps{
    BaseProps:     utils.BaseProps{ID: "timeline-cap"},
    CustomMessage: fmt.Sprintf("Showing the latest %d of %d status changes.", shown, total),
    Visible:       total > shown,
})
```

Naming and mechanism are the maintainer's call — the seam is: one field (or
variant) that bypasses `listNoteMessage`'s synthesis but keeps
`listNoteShell` as the single class/ARIA source. A `Visible bool` (or
"empty CustomMessage renders nothing", mirroring the truncated variant's
vanish behavior) keeps the caller's condition in one place.

The closest existing behavior is the truncated variant's
vanish-when-everything-fits, which is the right template: custom message,
same discipline.

Happy to contribute it.

💘 Generated with Crush

<!--
internal (strip before posting):
repo: LarsArtmann/templ-components (own repo)
title: display.ListNote: custom-message escape hatch that keeps the shared shell
status: verified 2026-10-11 against master a30aa6324c94 + v1.21.0; pending Lars review
publish: gh issue create -R LarsArtmann/templ-components --title "..." --body-file <extracted body>
related: #20 (count/range variants, shipped v1.20.0 — different gap)
-->
