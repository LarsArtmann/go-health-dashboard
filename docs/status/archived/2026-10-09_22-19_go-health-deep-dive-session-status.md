# Session Status — go-health Library Deep Dive (2026-10-09 22:19 CEST)

**Task given:** "What are we not yet leveraging from ../go-health?" — executed as a
library-deep-dive audit (skill-loaded, all 7 phases), then a full status pass.

**One-paragraph verdict:** The dashboard is go-health's richest and most idiomatic
consumer (15/26 capabilities fully leveraged, zero anti-patterns, on the latest tag
v0.5.1). The real gaps are two unused observation seams (`WithEvaluationHook`,
two-phase drain), four never-produced/demoed features (`checks` batteries,
`health.Off`, `VersionHandler`, `WithInstanceID` producer), one design decision
(probe verdict in the do cascade), and the unreleased `aggregate.SourceStatuses`.
Deliverable: committed HTML report `docs/research/2026-10-09_go-health-deep-dive.html`
(`38f8c76`). Session discipline had three self-inflicted wounds: a daemon commit race
(recovered via amend-relabel), two HTML defects introduced in one multiedit (one
fixed, one still live: "a endpoint"), and a skipped pre-write handshake.

---

## a) FULLY DONE

~~1. **Full API enumeration of go-health at the pin.** `go doc -all` for all four~~ done this session — evidence cited in the line
   packages (root: 806 lines; aggregate; federation; checks) under the devShell
   against go.mod's v0.5.1 — the exact audited version, not training data.
~~2. **Version currency.** Pin v0.5.1 == newest upstream tag. Unreleased v0.6 train~~ done this session — evidence cited in the line
   identified from CHANGELOG + source: `aggregate.SourceStatuses()`
   (aggregate.go:170-177), `aggregate.New` error-joining, never-started
   grace-window fix. Three commits of unpushed upstream work correctly excluded
   from "available" claims.
~~3. **Consumer-side usage inventory.** 90 go-health import sites; alias-safe symbol~~ done this session — evidence cited in the line
   counts (`health.*`, `aggregate.*`, `healthfederation.*`, `checks.*` = **zero**);
   config audits of `example/main.go` and `cmd/health-hub/main.go` (options set,
   shutdown order, env knobs).
~~4. **26-capability gap analysis, every claim cited.** 15 fully leveraged, 6~~ done this session — evidence cited in the line
   partial, 5 missed, 0 misused. Verified highlights: `SanitizeResponse` at the
   single choke point (dashboard.go:171-173); `Healthzer` capability forwarding
   with a pinned capability matrix (handlers.go:249-256, dashboard_test.go:1132,
   federation_test.go:195); `Status.Rank()` in trend+metrics (status.go:399,
   metrics.go:203); all four probe constructors exercised somewhere in-repo.
~~5. **Ten graded finding cards** with before/after code: evaluation hook,~~ done this session — evidence cited in the line
   two-phase drain, cascade verdict, checks batteries, `health.Off` producer,
   `VersionHandler`, `Routes.Healthz` config, `SourceStatuses`, `federation.WithClient`,
   `AwaitReady`.
~~6. **Scoring + Pareto.** 78/100 adoption; priorities table (impact × ease, 20 → 3).~~ done this session — evidence cited in the line
~~7. **HTML report written and verified.** Zero template placeholders left; tag~~ done this session — evidence cited in the line
   balance exact (40/40 spans, all other pairs clean); all 7 nav anchors resolve;
   headless-Chromium render confirmed for hero/scorecard/summary; parse check
   clean (the two "errors" were self-closing `<meta/>` parser artifacts).
~~8. **Committed with intent message** `38f8c76` after daemon-race recovery (see d1).~~ done this session — evidence cited in the line
   Not pushed — master ahead 5, push needs authorization.

## b) PARTIALLY DONE

~~1. **Visual verification of the report.** Only hero/scorecard/summary confirmed in~~ closed by later sessions (2026-10-11..25) — the residue rode the harvest chain into TODO_LIST
   a real browser. Findings/currency/appendix verified by parse + grep only: the
   anchor-fragment screenshot silently rendered the page top (headless
   `--screenshot` does not scroll to fragments), and the full-page capture hit the
   200 KB view limit; the crop retry failed because `nix develop` ran from /tmp
   (no flake there).
~~2. **AGENTS.md currency.** Noticed dependency-notes staleness vs. reality —~~ closed by later sessions (2026-10-11..25) — the residue rode the harvest chain into TODO_LIST
   "v0.2.0 in go.mod", "go.mod pins the v0.4.0 tag", "federation (unreleased;
   v0.3.0 vehicle)" while the pin is v0.5.1 and federation shipped in v0.3.0 —
   NOT fixed (session ended into the status request; user constrained research).
~~3. **Findings → TODO_LIST harvest.** The report's priorities are entombed in a~~ closed by later sessions (2026-10-11..25) — the residue rode the harvest chain into TODO_LIST
   timestamped HTML; TODO_LIST.md was not updated (status-report skill: (f) is
   HARVEST input — pending instructions).
~~4. **Some render claims verified only to viewModel level.** `Uptime`/`Version`~~ closed by later sessions (2026-10-11..25) — the residue rode the harvest chain into TODO_LIST
   shown to land in the viewModel (status.go:261-262) but not traced into
   view.templ display; the "fully surfaced" strength phrasing is slightly ahead
   of the evidence.
~~5. **Proofread.** One grammar defect shipped ("not on a endpoint",~~ closed by later sessions (2026-10-11..25) — the residue rode the harvest chain into TODO_LIST
   finding-version h3). Everything else passed a mechanical grep pass only.
~~6. **Phase 6 (cross-skill references)** handled implicitly inside the report, no~~ closed by later sessions (2026-10-11..25) — the residue rode the harvest chain into TODO_LIST
   dedicated follow-up actions created.

## c) NOT STARTED

~~1. **All ten prioritized opportunities** — this was an audit by request; no~~ closed or routed — 2026-10-13 HARVEST → TODO_LIST; executed or consciously closed in the 2026-10-25 closeout
   production code was changed (the repo diff is docs-only).
~~2. TODO_LIST.md / FEATURES.md updates from the findings.~~ closed or routed — 2026-10-13 HARVEST → TODO_LIST; executed or consciously closed in the 2026-10-25 closeout
~~3. Upstream federation.Healthz proposal (would require verify-before-filing gate).~~ closed or routed — 2026-10-13 HARVEST → TODO_LIST; executed or consciously closed in the 2026-10-25 closeout
~~4. Push to origin / green-CI confirmation on the release-style commit chain.~~ closed or routed — 2026-10-13 HARVEST → TODO_LIST; executed or consciously closed in the 2026-10-25 closeout
~~5. Same deep-dive for the other three UI/SSE dependencies~~ closed or routed — 2026-10-13 HARVEST → TODO_LIST; executed or consciously closed in the 2026-10-25 closeout
   (templ-components, go-datastar, go-sse).

## d) TOTALLY FUCKED UP

~~1. **The daemon commit race — the documented failure mode, again.** The daemon~~ resolved — superseded by later sessions (see TODO_LIST/CHANGELOG 2026-10-25)
   swept the report file twice mid-write: `537de6f` captured the raw 57-line
   template (a garbage intermediate, immutable history) and `86c06ed` captured the
   final 534-line content with a heuristic message. My explicit `git add` +
   `git commit` chain arrived seconds late ("nothing to commit, working tree
   clean"). AGENTS.md release discipline §0/§2 documents exactly this; I still
   lost the intent message. Recovery: amend-relabel of HEAD → `38f8c76` (same
   diff, real message). The amend itself raced the daemon — had a newer daemon
   commit landed first, I would have relabeled the WRONG commit. Worked this
   time; not a safe default.
~~2. **Two HTML defects introduced in the big findings multiedit:** a stray `>`~~ resolved — superseded by later sessions (see TODO_LIST/CHANGELOG 2026-10-25)
   splitting "nothing" across a tag boundary and a broken `</n>` close tag.
   Self-caught next tool call and fixed — but they prove the multiedit was not
   proofread before submission.
~~3. **"a endpoint" typo is live in the committed report.** Shipped because the~~ resolved — superseded by later sessions (see TODO_LIST/CHANGELOG 2026-10-25)
   final proofread was mechanical (placeholder/balance greps), not a read.
~~4. **Pre-write handshake skipped.** AGENTS.md's two-session protocol (git log +~~ resolved — superseded by later sessions (see TODO_LIST/CHANGELOG 2026-10-25)
   docs/status tail + git status BEFORE any write) ran only before the COMMIT,
   not before the first write. Low harm this time (new file), wrong discipline.

## e) WHAT WE SHOULD IMPROVE

~~1. **Commit intent in the same chain as the first successful write** of a new~~ resolved — superseded by later sessions (see TODO_LIST/CHANGELOG 2026-10-25)
   artifact — write → verify → commit, before any next step. Never let a
   verified artifact sit uncommitted across tool calls.
~~2. **Proofread generated HTML as text, not just structure:** grep for~~ resolved — superseded by later sessions (see TODO_LIST/CHANGELOG 2026-10-25)
   `\ba [aeiou]`, stray `>` artifacts, `</[a-z]{1,2}>` oddities, before
   declaring done.
~~3. **Visual-verify every section:** one devShell call from the repo dir (never~~ resolved — superseded by later sessions (see TODO_LIST/CHANGELOG 2026-10-25)
   /tmp), full-page render, downscale + crop inside that call, view all crops.
   Anchor fragments don't scroll in headless screenshots — scroll with JS or crop.
~~4. **Handshake before the first write, always** — not just before commits.~~ resolved — superseded by later sessions (see TODO_LIST/CHANGELOG 2026-10-25)
~~5. **Fix noticed doc staleness on sight** (AGENTS.md go-health notes) or ticket it~~ resolved — superseded by later sessions (see TODO_LIST/CHANGELOG 2026-10-25)
   immediately; "noticed but not fixed" is how the three-times-burned go-floor
   gotcha class happened.
~~6. **Fix the scoring rubric before writing cards.** The 26-row denominator was~~ resolved — superseded by later sessions (see TODO_LIST/CHANGELOG 2026-10-25)
   constructed while writing; it held together, but the method should be pinned
   first.
~~7. **Keep strength claims at verified strength** ("demoed" vs "exercised in~~ resolved — superseded by later sessions (see TODO_LIST/CHANGELOG 2026-10-25)
   tests" are different claims; NewWithHealthCheck is tests-only).
~~8. **Recovery recipe worth standardizing:** when the daemon eats an intent~~ resolved — superseded by later sessions (see TODO_LIST/CHANGELOG 2026-10-25)
   commit, amend-relabel only if `git log -1 --format=%H` confirms HEAD is still
   the daemon commit carrying your content; otherwise commit an empty-ish
   follow-up note instead of racing.

## f) UP TO 50 THINGS TO GET DONE NEXT

_Pareto top from the audit first (P = impact × ease), then repo hygiene,
upstream watch, process._

**Library leverage (the actual answer to the original question):**

~~1. Two-phase drain in example: `probe.MarkShuttingDown()` before `server.Shutdown` (P20).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~2. Same reorder in health-hub `shutdown()` (P20).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~3. `/version` endpoint in example via `health.VersionHandler(version.Version)` (P15).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~4. Same endpoint in health-hub (P15).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~5. `health.Off(...)` demo row in the example probe (P15).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~6. `checks.Disk` + `checks.Memory` rows in the example (P15) — real data, warn-by-default.~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~7. ADR: `WithEvaluationSink(func(health.Response))` — probe-cadence ingest for trend/evidence (P10).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~8. Implement the sink + wire `WithEvaluationHook` in example/hub (P10).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~9. `health.WithInstanceID(hostname)` in example + hub (P10).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~10. Adopt `aggregate.SourceStatuses()` for hub per-source cards when go-health v0.6 tags (P9).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~11. Decide + implement the `Healthchecker` optional capability (probe verdict into do cascade) (P8).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~12. `Routes.Healthz` demo in the example's aggregate mode (P8).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~13. `AwaitReady(ctx)` before `ListenAndServe` in example (P4).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~14. First-fetch gate (`AwaitReady`-style) in health-hub (P4).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~15. `federation.WithClient` env knob (TLS CA / egress proxy) (P3).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~16. Refresh README screenshots after 5/6/9 land (off badge + battery rows visible).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~17. Browser test: off-badge renders and stays out of the evidence strip.~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~18. README: document the consumer-side `WithEvaluationHook` pattern.~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~19. Hub README: state federation has no Healthz (LB story needs probe/aggregate-backed deploy).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7

**Repo hygiene:**
~~20. Fix AGENTS.md stale go-health dependency notes (pin v0.5.1; federation released; StatusOff adopted).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~21. HARVEST this report's items 1-19 into TODO_LIST.md with owners/status.~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~22. Fix "a endpoint" → "an endpoint" in the committed report; commit the fix.~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~23. Visual verification pass over the report's lower sections (findings/currency/appendix).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~24. Trace `Uptime`/`Version` from viewModel into view.templ; annotate the report if "rendered" overclaims.~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~25. Investigate how the evidence log treats synthetic rows ("health-check", "evaluation-hook").~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~26. Guard the go-health pin like the UI pins (go-mod-update sweeps hit go-health before: 68ac162, b1128ae).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~27. Link-check pass: re-verify every file:line citation in the report in one sweep.~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~28. FEATURES.md: record the leverage baseline the audit established.~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~29. Re-run the rubric after items 1-9 land; update the report's score annotation.~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~30. Consider a README-screenshot freshness gate so demo-data changes (5/6) can't silently rot images.~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~31. Decide whether the evidence strip should distinguish probe-cadence vs push-cadence observations (feeds ADR in 7).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~32. Consider `NewWithHealthCheck` coverage in the example (currently tests-only) or reframe docs.~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7

**Upstream watch / contributions:**
~~33. Watch go-health for the v0.6 tag; adopt the trio (SourceStatuses, error-join, grace fix) in one train.~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~34. verify-before-filing, then propose federation.Healthz (single-endpoint LB story) upstream.~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~35. Upstream docs nit: adoption-matrix lists this repo as `WithInstanceID` adopter — we render but never produce; annotate.~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7

**Process / meta:**
~~36. Full gate chain (`nix fmt`/`build`/`test`) before the next push; master is ahead 5 with CI blind.~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~37. Push master at the next authorized milestone.~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~38. Same deep-dive rubric for templ-components, go-datastar, go-sse.~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~39. Document the amend-relabel daemon-race recovery (worked; guard conditions) in AGENTS.md release discipline.~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7
~~40. Process fix: handshake-before-first-write; proofread-greps before "done"; both recorded here and in e).~~ routed — 2026-10-13 HARVEST ledger; dispositions live in TODO_LIST (closed 2026-10-25) and ROADMAP themes 5-7

_(40 items — within the "up to 50" budget; items 1-15 are the audit's actionable core.)_

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

~~1. **Cascade semantics (gates item 11):** Should the dashboard forward the~~ routed — standing TODO_LIST BLOCKED row or ROADMAP Open Question
   probe's roll-up verdict into the samber/do `do.HealthCheck` cascade
   (optional `Healthchecker` capability), or must the cascade stay
   pusher-scoped because federation fail-over staging wants "surface alive"
   and "fleet healthy" as separate signals?
~~2. **API scope (gates items 7-8):** Should `WithEvaluationSink` land as a~~ routed — standing TODO_LIST BLOCKED row or ROADMAP Open Question
   dashboard library API now, or stay a documented consumer-side pattern until
   go-health v0.6 ships (hook maturity + `SourceStatuses` train)?
~~3. **Authorization:** May I push master (5 commits, CI currently blind), and/or~~ routed — standing TODO_LIST BLOCKED row or ROADMAP Open Question
   start the verify-before-filing gate for the upstream federation.Healthz
   proposal?

---

**Artifacts this session:** `docs/research/2026-10-09_go-health-deep-dive.html`
(committed `38f8c76`; garbage template intermediate `537de6f` in history),
`docs/status/2026-10-09_22-19_go-health-deep-dive-session-status.md` (this file).
No production code changed. Working tree at write time: clean.
