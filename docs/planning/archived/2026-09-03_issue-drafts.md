# Upstream Issue Drafts (M30)

Drafts for issues to file against sibling projects. Each diagnosis is
verified against the upstream source before filing.

**RESOLVED 2026-09-04:** the draft below was filed as
[templ-components#6](https://github.com/LarsArtmann/templ-components/issues/6);
the second upstream issue (#7, LiveRegion busy-script `nonce=""`) was filed
2026-09-04 during the v0.4.0 cycle. Follow-up PRs are tracked in
`TODO_LIST.md`. File archived — fully executed.

## 1. templ-components — StatCard `<dl>` structure trips axe `definition-list`

**Repo**: LarsArtmann/templ-components · **Component**: `display.StatCard`
(`display/card.templ`, `statCardFigures`)

**Evidence (verified at source, v1.11.0):**

```html
<dl>
  <dt class="...">Version</dt>
  <div class="mt-1 flex items-baseline">
    <dd ...>1.0.0</dd>
  </div>
</dl>
```

The `<div>` wraps only the `<dd>` while its `<dt>` sibling stays outside
the div. HTML allows `<div>` inside `<dl>` as a _group_ wrapper, but the
group must contain the dt/dd pairs it belongs to. axe-core flags this as a
serious `definition-list` violation; screen readers lose the dt→dd
association.

**Found via**: go-health-dashboard's axe-core browser audit
(`TestBrowser_Accessibility`), reproduced on the stock StatCard markup.

**Suggested fix**: move the `<div class="mt-1 flex items-baseline">`
outside so it wraps both `<dt>` and `<dd>`, or drop the div and put the
flex classes on the `<dl>`.

**Status**: FILED — https://github.com/LarsArtmann/templ-components/issues/6
