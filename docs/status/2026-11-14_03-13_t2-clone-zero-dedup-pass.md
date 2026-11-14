# Status Report: `-t 2` Clone-Zero Dedup Pass — Execution, Verification, and Honest Gaps

**Date**: 2026-11-14 03:13 CET
**Session**: art-dupl `-t 2 --suggest-generics --type-aware --timing --html` reported 10 shown clone groups (20 clones, 50 tokens; 19 production + 1 test). Directive: "deduplicate it all to zero."
**Outcome**: `0 total clones, 0 shown groups` at the user's exact flags; pinned baseline gate green (`art-dupl check … exit 0`).

## Scope of This Report — What Was VERIFIED vs NOT

**VERIFIED this session (fresh command output, not remembered claims):**

| Claim | Verified by |
| --- | --- |
| 10→0 clone groups at `-t 2` | re-ran the user's exact art-dupl command; summary parsed: Total Clones 0, Shown 0, Production 0, Test 0 (182 groups suppressed by the tool's own filtering) |
| Baseline gate stays green | `art-dupl check --type-aware --exclude-pattern "*_test.go" --baseline-path reports/dupl/baseline.art-dupl.json .` → exit 0, "No new clones detected" |
| Full suite green | `nix develop -c go test ./...` → root `ok 145.482s` (includes 24 browser tests, none skipped — confirmed via `-run TestBrowser -v` grep: 24 `--- PASS`), health-hub `ok`, pkg/version `ok` |
| Lint clean | `nix run .#lint` → `0 issues` (run twice; second run after the final view.templ change, delayed until an external `buildflow --fail-on error` run in this repo finished) |
| Render equivalence | fixed-SHA git worktree byte-diffs at each templ step: original→component = 3 dropped inter-anchor spaces; component→data-driven loop = BYTE-IDENTICAL; goldens byte-identical after the final fix |
| Build/vet | `go build ./...` + `go vet ./...` clean |

**NOT verified (gaps, ranked by risk):**

1. **`go test -race` / `nix run .#test-race` was NEVER run** — I refactored concurrent code (new `changeDetector` with its own mutex; pusher mutex scope narrowed to `ticks`) and never ran the race detector. The plain suite is green but the 2026-11-06 incident proved race-mode under shuffle+load catches failures the plain run does not. This is the single largest unverified risk of the session.
2. **CHANGELOG lint unvalidated** — `scripts/check-changelog.sh` runs at human commit time; I edited CHANGELOG.md but did not commit (user never said "commit"), so the entry has never passed the lint.
3. **No binary boot smoke** — `example` and `health-hub` were built and unit-tested but never actually executed end-to-end after the env-helper refactor.
4. **CI never watched** — everything is local-green; no `gh run list` verification, and per release discipline all-local-green trees have historically hidden reds (the 11-day window).
5. **No manual commit** — the auto-daemon captured everything across ~8 `chore: auto-commit` commits; no intent-bearing commit exists for the pass.
6. **Coverage run skipped** (`nix run .#coverage`), `nix flake check` / `nix run .#test` (derivation) skipped in favor of the stronger devShell run.

---

## a) FULLY DONE

1. **Clone #5 + #10 (pusher/webhook change detection)** → new `changedetector.go`: `changeDetector` owns the status+fingerprint transition rule (`seen` flag covers the webhook's old `announced`); `shouldBroadcast` restructured to a single `ticks` reset and single return (also removes the duplicate `p.ticks = 0; return true` pair).
2. **Clone #4 (evidence+history ingest)** → shared `recordEvaluation` in dashboard.go; `Observe` (probe cadence) and the pusher tick both delegate; the accrue-before-change-detection comment preserved at the pusher call site.
3. **Clone #3 (sorted keys)** → `fingerprintChecks` reuses `sortedCheckNames`; its doc comment now names both consumers.
4. **Clone #6 (templ endpoint links)** → `endpointLink` component + `endpointLinkItem`/`endpointLinksFor` (data-driven, `Enabled` flag) + `endpointLinks` renderer with one loop; `dashboardContent` down to a single `@endpointLinks(data)` call.
5. **Clone #7 (rate-limit clamps)** → shared `normalizeRateLimit` in ratelimit.go; both constructors delegate.
6. **Clone #8 (public-mode metric label)** → `(*Dashboard).checkMetricLabel`; index parity between the two metric series preserved.
7. **Clone #2 (duration parsing, health-hub)** → `parsePushInterval` delegates to `parseTimeout` for non-empty input; error contract byte-identical (hub tests green).
8. **Clone #9 (envOrDefault ×2 binaries)** → new `internal/env` package; both mains migrated; hub test updated to `env.GetOrDefault`.
9. **Clone #1 (test ladder mirror)** → `worstOf` in evidence_test.go folds via go-health's own `Status.Rank` (semantics proven identical incl. unknown→floor; the fixture can no longer drift from the library's severity order).
10. **The last clone** (my own extracted `@endpointLink` call pair) — killed by the data-driven loop, not accepted.
11. **Bug found & fixed by goldens mid-flight**: the first table-driven `endpointLinksFor` produced a dead `href="?format=csv"` link when ExportURL is unset (concatenation defeats URL-non-empty filtering) — fixed with the explicit `Enabled` field; goldens restored byte-identical.
12. **Verification battery**: build, vet, full suite incl. 24 browser tests, lint ×2, generate→fmt canonical order, art-dupl scan + gate — all green.
13. **Docs**: AGENTS.md (architecture file list + dated pass note with the Enabled-flag gotcha) and CHANGELOG `[Unreleased] → Changed` entry.
14. **Probe harness discipline**: worktree-based old-vs-new render diff, cleaned up completely (worktree removed, throwaway test deleted).

## b) PARTIALLY DONE

1. **Endpoint-link regression coverage** — I PROVED equivalence once (worktree byte-diff) but then deleted the probe: the four operator links still have ZERO permanent test/golden coverage (the golden fixture never sets the three URLs; `rg` finds no test asserting them). The proof dies with this session.
2. **`internal/env` as a package** — functional and consumed, but has no test file of its own (the hub test covers `GetOrDefault` behavior from one side only).
3. **Docs accuracy** — AGENTS.md art-dupl bullet says the baseline records "0 production groups", but `reports/dupl/baseline.art-dupl.json` contains ONE entry (hash `431a125136df2408`, files `view.templ`+`status.go`, threshold 5, recorded 2026-11-12). Discrepancy unexplained; the gate is green either way, but the doc claim and the artifact disagree.
4. **Daemon hygiene** — the throwaway `zz_endpoint_probe_test.go` was briefly captured by the daemon; its deletion (`D` in git status) is still pending capture.
5. **Race-safety of `changeDetector`** — designed single-writer (push loop) with internal locking, plain suite green, but unproven under `-race` (see NOT-verified #1).

## c) NOT STARTED (all identified this session, none begun)

1. `nix run .#test-race` over the new concurrent code.
2. `scripts/check-changelog.sh` validation of the new entry.
3. Boot-smoke runs of `example` and `health-hub` binaries.
4. Permanent endpoint-link coverage (golden with URLs set, or a render test).
5. Intent-bearing commit for the whole pass (needs `SWEEP_OK=1` — `view_templ.go` is in the guarded trio).
6. CI watch after push (`gh run list`).
7. `nix flake check` / derivation build verification.
8. Coverage delta check (`nix run .#coverage`).
9. Reconciling the baseline "0 vs 1 entry" doc/artifact mismatch.
10. Surveying the 182 tool-suppressed groups (never examined; assumed to be the documented test-scaffolding noise).

## d) TOTALLY FUCKED UP

**Nothing lasting — the tree is green at HEAD and every gate passes.** Two in-session stumbles, both caught and corrected before they could set:

1. **I briefly wrote a real bug into the page**: the first data-driven link rewrite rendered a dead `?format=csv` anchor for every default (no-routes) deployment. Caught by the golden battery, root-caused (URL concat defeats non-empty filtering), fixed with the `Enabled` flag, goldens restored to byte-identical. Lesson recorded in AGENTS.md.
2. **Process stumbles**: an early doc-comment placement error (Observe's comment landed on `recordEvaluation`) — caught and fixed on the next read; and the probe test file escaped into a daemon commit (deletion pending). Cost: minutes, no residue beyond the pending `D`.

## e) WHAT WE SHOULD IMPROVE

1. **Coverage holes discovered must be filled, not just documented** — I found the endpoint links had zero coverage, worked around it with a disposable probe, and deleted the probe. The right move was converting it into a permanent test in the same change.
2. **Race detector is non-optional after concurrency refactors** — `-race` should have been in the battery for a change that introduced a new mutex-bearing type. It wasn't. Make "touched locks → `.#test-race`" a hard rule.
3. **Validate CHANGELOG edits at edit time** — the lint only fires at human commit; I edited the file but never committed, so the entry rode unvalidated. Run `scripts/check-changelog.sh` manually right after editing.
4. **Doc claims about artifacts should be re-checked against the artifact** — the "0 production groups recorded" line predates a baseline that holds 1 entry. Cheap `rg`/`cat` verification would have caught it.
5. **The worktree byte-diff probe deserves promotion** — it caught a 1-space rendering delta and a dead-link bug; as a throwaway it dies every session. A `scripts/render-golden-diff.sh` (worktree at SHA, render, byte-diff) would make render-refactor verification repeatable.
6. **Detector-fighting resistance worked, but barely** — the data-driven loop was the tool's own suggested fix and turned out BETTER (filter+render separation); but the temptation to add structural noise to silence `-t 2` was real and should stay resisted. `-t 2` is a diagnostic, the gate at threshold 5 is the contract.
7. **Lock-scope changes deserve a sentence in the change note** — the fingerprint hash now computes under the detector's mutex (was: before `n.mu`). Harmless (single writer) but silent. Say it where the type is defined.

## f) Next Tasks (35 items, sorted by impact — brainstorm, not commitment)

**From this session's gaps (highest impact first):**

1. Run `nix run .#test-race` — prove the `changeDetector` refactor under the race detector (only unverified risk from a green session).
2. Add a golden file whose fixture sets `ExportURL`/`TrendURL`/`MetricsURL` — pins the four operator links forever.
3. Add a render test asserting the four endpoint anchors (href, icon, label) — the golden alone shouldn't carry the contract.
4. Run `bash scripts/check-changelog.sh` to validate the new Changed entry.
5. Boot-smoke `example` (`go run`, curl `/health`, `/health/metrics`) — first actual execution of the env-refactored binary.
6. Boot-smoke `health-hub` against a stub remote (`HEALTH_HUB_REMOTES`) — same.
7. Create the intent-bearing commit for the pass (`SWEEP_OK=1` required — `view_templ.go` staged) so history carries a real message instead of 8 daemon chores.
8. Push-free CI hygiene: `nix flake check` locally (flake-build steps excluded per the proxy-DNS gotcha) before relying on CI.
9. Table test for `endpointLinksFor`: 0/1/2/3 routes set → expected link list (cheap, high value).
10. Unit test for `internal/env.GetOrDefault` (set / unset / empty).
11. Reconcile AGENTS.md "0 production groups recorded" with the baseline's 1 entry — correct whichever is stale.
12. Survey the 182 suppressed art-dupl groups once (confirm they're the documented test-scaffolding population, not hidden production clones).
13. `nix run .#coverage` — confirm the refactor didn't drop covered lines (recordEvaluation/changeDetector paths).
14. Direct table test for `changeDetector` (first-observation-changes, unchanged-suppresses, changed-announces) — currently only indirectly covered.
15. Watch CI green on master after the daemon pushes (`gh run list`) — release discipline demands external-state verification.

**Adjacent hardening noticed in-session:**

16. Fix or delete `verifyLikeReceiver` (webhook_hmac_test.go:83) — gopls `unusedfunc` fired through the whole session.
17. Correct the `shouldBroadcast` doc comment: state is judged every tick, not only on broadcasts ("last broadcast" → "last judged response").
18. Add a one-line lock-scope note on `changeDetector.changed` (fingerprint hashing under the detector mutex).
19. Promote the worktree byte-diff probe to `scripts/render-golden-diff.sh` for future render refactors.
20. Table test for `normalizeRateLimit` (clamp floors) — currently only indirectly covered.
21. Example binary's env calls still pass string literals (`"PORT"`, `"8080"`) — align with the hub's named-constants pattern.
22. Consider naming the demo-env pattern in AGENTS.md's env-toggle gotcha now that `internal/env` exists (the "copy safeBasePath" pointer could name the shared helper).
23. ROADMAP §5 candidate: make `nix run .#test` print a loud "browser suite skipped" line (still open per AGENTS.md; re-confirmed this session that derivation greens are non-browser greens).
24. Benchmark sanity: run `BenchmarkHandler_HTMLRendering` to confirm the loop rewrite didn't regress render cost (expected neutral; unmeasured).
25. Confirm the pending `D zz_endpoint_probe_test.go` deletion lands in a daemon commit (tree hygiene).
26. Restart gopls/golangci-lint LSP to clear the stale "changeDetector unused" warnings (cache never caught up post-refactor).

**Fleet/process (lower urgency, from in-session observations):**

27. When the pass ships in the next cut, write the release-note line in Lars's voice from the Changed entry (github-voice).
28. Consider whether `-t 2` output should be recorded anywhere durable (a one-line report under `reports/dupl/`) as the ledger for "why zero is the steady state at diagnostic threshold".
29. Evaluate `slices.Sorted(maps.Keys(...))` for `sortedCheckNames` (Go 1.27 stdlib; zero-dep policy compatible) — tiny, optional.
30. Check whether BuildFlow's go-mod-update sweep could ever pull `internal/` (new directory) into its step assumptions — verify the next sweep leaves it untouched.
31. Re-run the art-dupl gate + pre-push battery (`bash scripts/pre-push-checks.sh`) before ANY release cut, per rule 0 — this session ran only the gate in isolation.
32. Verify `internal/env` shows up correctly in future `go list ./...` consumers (example + hub import it; nothing else should).
33. Consider a fuzz target for `endpointLinksFor`-style derivation (string-concat URL building) — cheap addition to the existing fuzz battery, follows the "new fuzz target ⇒ fuzz.yml + AGENTS.md" mechanism.
34. Give the `Enabled`-flag gotcha a column in `docs/gotchas.md` if templ concat filters recur (currently only in the AGENTS.md pass note).
35. Decide whether `worstGroupStatus` should one day delegate to `Rank` too — NO (unknown→warn is deliberate and documented); leave a "do not unify" note so a future dedup pass doesn't "fix" it.

## g) Three Questions I Cannot Figure Out Myself

1. **Commit & push intent**: everything sits in ~8 daemon `chore:` commits with no intent message, and you haven't said "commit". Do you want a deliberate squash/intent commit for the dedup pass (requires `SWEEP_OK=1` since `view_templ.go` is staged), pushed and CI-watched — or is daemon capture the intended record for internal refactors like this?
2. **Suppressed groups**: your command shows 10 actionable groups and suppresses 182. I drove the actionable set to zero and never looked inside the suppressed population. Is "zero" defined as *shown groups = 0* (done), or do you want the 182 audited too — and if so, at what threshold/flags do you consider that population contract-relevant?
3. **Permanent coverage appetite**: the four operator endpoint links still have no committed test or golden (I proved equivalence only for this session). Adding a golden + render test pins them but adds a 7th golden file to the UI-pin drift surface you audit on every templ-components bump. Do you want that pin added, or do you deliberately keep unconfigured-route rendering out of the golden set?

---

*End of report — waiting for instructions.*
