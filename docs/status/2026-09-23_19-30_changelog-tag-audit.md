# CHANGELOG Per-Bullet Tag Audit — v0.3.0 … v0.8.1

**Date:** 2026-09-23 · **Method:** every bullet in the `[0.3.0]`–`[0.8.1]`
CHANGELOG sections was checked against its tag (`git grep` / `git
cat-file` at the tag ref for the code-level claim; `gh release view` for
the one external-state claim). Commit-log matching was deliberately NOT
used — the auto-daemon's heuristic commits make commit-to-bullet mapping
unreliable, while tag-tree state is ground truth.

## Result: all bullets verified TRUE, one historical finding

Every code-verifiable bullet in the audited sections resolves to real,
present state at its tag. Representative verified claims per section:

- **[0.3.0]** `WithDatastarSrc`, `Register`/`di.go`, `WithRetryInterval`,
  `WithBasePath`, SSE `retry:` field, `SubscriberCount`, the breaking
  `RegisterRoutes(mux *http.ServeMux)` signature — all present at tag.
- **[0.3.1]** `WithMiddleware`, `WithMetrics`/`WithTrend`/
  `WithHideStatCards`, both fuzz targets, `browser_test.go`,
  `screenshot_test.go`, README `unsafe-eval` doc — all present.
- **[0.4.0]** `RecommendedCSP` (csp.go), `FuzzEscapeLabelValue`,
  `.github/workflows/fuzz.yml`, CI browser job, `WithDescription`,
  `WithShutdownDrain`, `WithMaxConnectionLifetime`, Prometheus parser
  dep, templ-components pinned v1.11.0 — all present.
- **[0.5.0]** `Prober` interface, `WithWebhook`/`WithWebhookHeaders`,
  `docs/integrations.md`, go-health v0.1.0, README integrations
  sections — all present.
- **[0.6.0]** sentinel family, version-guard job, 75% coverage floor,
  fuzz dedup-issue script, CI concurrency group, golangci-lint pin,
  `DEMO_PUBLIC`, trend not-started 503 wording, `options.go` split,
  go-health v0.1.3, `SanitizeResponse` choke point, bisect audit doc,
  doc.go sentinel examples, axe tolerance scoping — all present.
- **[0.6.1]** the `DatastarVersion1_0_3` rename — TRUE in source
  (`view.templ:87` at the tag).
- **[0.7.0]** `Routes()`, `WithIntrospection`, `WithPushOnChangeTTL`,
  `WithTimelineMaxAge`, `WithPersistCollapse`,
  `WithEmbeddedDatastarSDK`, `WithNoDatastarRuntime`, `GroupBySource`,
  429 JSON negotiation, drain `Retry-After`, example toggles, ndjson
  export, templ-components v1.16.0, jump-to-problems anchor — all
  present.
- **[0.8.0]** mobile `sm:` stacking, `page_scripts.templ`,
  `scripts/verify-release.sh`, evidence strip (`evidence.go`),
  templ-components v1.17.0, deploy compose stack, AGENTS
  release-discipline section, the v0.6.1 Release backfill (verified
  live: `gh release view v0.6.1` → published, not draft) — all present.
- **[0.8.1]** FEATURES recount (248) — present at tag.

## Finding (historical, no action needed)

**The v0.6.1 tag shipped a stale generated file.** At `v0.6.1`,
`view.templ` correctly references `datastar.DatastarVersion1_0_3`, but
the committed generated `view_templ.go` still referenced
`DatastarVersion1_0_2` — the rename-fix commit (`b8a81eb`) updated the
template source without running `go tool templ generate`, and the tag
was cut on that tree. It compiled only because go.mod still pinned
templ-components v1.13.0, where the old constant still exists as an
alias. This is the same generated-file-staleness class the CI hygiene
drift check now catches, and the "generate → fmt" canonical order plus
the pinned tool directive were adopted later in response to exactly
this class. Tags are immutable; the discrepancy self-healed in every
later release. Recorded here so the audit is honest about the one thing
that did not verify byte-perfectly.
