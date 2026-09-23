# Session Status Report — Full TODO_LIST execution sweep

**Generated:** 2026-09-23 18:09 CEST
**Head:** `6776877` (master, clean tree) · **35 commits ahead of origin, unpushed**
**Scope of this report:** THIS SESSION ONLY (the 2026-09-18/22 TODO_LIST queue,
harvested from the pasted list). User directive: "GET SHIT DONE — the WHOLE
TODO LIST." Report format: markdown at `docs/status/` per explicit standing
instruction.

---

## Session Summary

Executed every row of the 2026-09-18/22 TODO_LIST queue: 33 TODO rows
closed (implemented, verified, or verified-already-done), 2 blocked rows
closed as overtaken, 11 blocked rows re-triaged with fresh premise
checks. Four real bugs were found and fixed along the way (two of them
only because the rows demanded *actually running* things instead of
trusting prose). All gates green at my last run; **a parallel session
landed edits at 17:42–17:47, after my last green gate** — the current
tip is therefore NOT verified by me (see d7).

---

## a) FULLY DONE

- **go.mod directive hotfix** (`83daf78`): the daemon had swept `go
  1.27.1` → `go 1.27`; CI was red on `ecccf63` (Build/Browser/Lint/Test/
  Vuln all failing the directive guard). Restored atomically per the
  documented pattern; guard green.
- **verify-dep-bump.sh hardening** (`eb59eab`): coverage floor parsed
  from ci.yml (single source), fmt dirty-tree check retries once to ride
  out daemon races, coverage parse runs via `nix develop -c`. Shellcheck
  clean (it caught an SC2027 in my own first draft).
- **Gate coverage** (`b1c6504` + daemon carries): `pre-push-checks.sh`
  now diffs `fuzz.yml`'s `-fuzz` steps against actual `Fuzz*` definitions
  (incl. per-package placement) and rejects bare `templ generate` /
  `pkgs.templ` anywhere in flake/CI/scripts; CI build job runs shellcheck
  over all six gate scripts. Root-fixed `check-ui-pins.sh` to self-select
  the toolchain (probe + `nix develop -c go` fallback) and moved the
  devShell banner to stderr so `nix develop -c` stdout is parseable.
- **`shutting_down:true` webhook wire pin** (`52bfacf`): true branch
  locked as a real JSON bool.
- **`testdata/golden/source.html` row closed by verification**: the
  `TestGoldenRender_SourceProvenTooltips` fixture (`sourceproven.html`)
  already pins the grouped-card proven tooltip pairing; all goldens pass
  against the current render; `source.html` correctly pins the
  no-evidence state (no tooltips by design). No code change needed.
- **M103** (`4847840`): `TestVersionConst_MatchesCIGrep` reproduces the
  CI grep+sed pipeline verbatim and fails with a precise message if the
  const's text shape would break it.
- **M105 + hub cadence** (`49c4c47`): doc.go examples for webhook+public-
  mode and `WithBasePath`; `HEALTH_HUB_PUSH_INTERVAL` env knob (validated
  up front, effective value in the startup log); library exports
  `DefaultPushInterval`; hub unit tests for both parsers.
- **`docs/metrics.md`** (`10b1c00`): all eleven metric families, 0.0.4
  wire constraints, buckets, status-encoding rationale, cardinality
  guarantee.
- **Compose batch** (`dc99291` + daemon): images digest-pinned
  (manifest-LIST digests via buildx), dependabot `docker` ecosystem
  entries for `/` and `/deploy`, Prometheus `/-/ready` healthcheck gating
  Grafana, inert alert example.
- **Compose e2e boot** (`412ed3f`): first real boot ever — caught the
  Dockerfile `golang:1.26` base failing `go mod download` against the
  1.27.1 floor (fixed, base digest-pinned); port collisions with other
  projects solved via `!override` compose port lists (documented — plain
  overrides MERGE); live scrape verified (`dashboard_health_up`
  queryable, target up); all 8 panels rendering against real data;
  screenshot captured (`docs/screenshot-grafana.png`); alert example
  activated → **caught a real schema bug** (`relativeTimeRange` wants
  integer seconds, `"5m"` 422s provisioning), fixed, re-verified to
  `Pending` (firing) against the failing demo probe, then deactivated.
- **`deploy/README.md`**: ports, layout, anonymous-viewer posture,
  teardown, dependabot policy, the alert activation recipe, boot history.
- **M91** (`9c0fd46`): env-guarded `TestMeasureChromeLaunchLatency`;
  launch = avg 155ms / max 254ms vs the 45s announce timeout (~0.6%
  utilization); full 15-test browser suite ~15s wall. Verdict recorded:
  keep serialization. Also instrumented `startHeadlessChrome` with the
  shared `lastChromeLaunch` atomic.
- **M92 + aggregate load re-run** (`3a471e1`): `LOADTEST_*` env knobs;
  default fixture p50 1.05ms → 2.24ms (evidence-era render), doubled
  shape (30×4/40 clients) p50 5.7ms, zero event loss; research doc
  updated.
- **CHANGELOG per-bullet tag audit v0.3.0–v0.8.1** (`1b18e01`): every
  bullet checked against its tag via tag-tree greps (NOT commit matching
  — the daemon makes commit attribution unreliable). All verified TRUE;
  one honest finding recorded: v0.6.1 shipped a stale generated
  `view_templ.go` next to a corrected `view.templ` (compiled only
  because the old constant still existed as an alias).
- **Dark-mode contrast** (`194d87b`): metadata line gray-400 = 6.99:1 on
  the table surface, 5.78:1 on cards — AA pass, no change needed; the
  lone gray-500 is the decorative chevron (3.67:1 ≥ 3:1 non-text rule).
- **M107** (`ccad792`): nightly `release-check.yml` walking every tag for
  a Release page; actionlint clean.
- **`scripts/set-tag-protection.sh`**: rulesets-as-code encoding the
  three 422 quirks; name-resolved idempotent create-or-update; read-only
  `--check` verified the live ruleset 23637848 matches byte-for-byte.
  Mutating path NOT executed (repo settings = your call).
- **M106** (`ae1acdb`): bisectability audit extended over 405 commits
  with the toolchain PINNED to go 1.27.1 — 386 build, 19 broken, all
  daemon carriers in two classes (classic mid-edit tears + the new
  go-directive-sweep tears). First audit attempt used the ambient go and
  produced garbage (see d3).
- **M78 artifacts** (`35873004856`): screenshots artifact downloaded and
  eyeballed — light/dark/degraded PNGs, correct render, CI browser job
  works end to end.
- **M88/M89/mobile rows closed as already-done**: `TestBrowser_Keyboard
  Navigation`, `KeyboardNewControls`, `KeyboardLinks`,
  `MetricsUnderStrictCSP`, `MobileViewport` all exist and pass (v0.7.0
  era + the 09-23 dedup session); plus my own 390px-viewport eyeball pass
  on the compose dashboard (metadata line legible, stacked layout clean).
- **ROADMAP v1.0 sweep**: status table added (criterion 4 largely met; 1
  and 2 not met; 3 partial).
- **Blocked-row premise triage**: 11 premises re-verified HOLDS with
  evidence; branching-flow (unskipped upstream 09-22) and the 1.26.x
  toolchain row CLOSED as overtaken.
- **TODO_LIST/AGENTS/FEATURES/CHANGELOG swept** (`5ee9e3d`): queue
  emptied, ground truth updated (297 funcs / 42 root test files),
  AGENTS gotchas refreshed (erraudit claim refuted, devShell
  requirements retired, deploy-stack facts, bisect extension), full
  `[Unreleased]` section written.
- **Upstream verification** (`9132d84`): templ gofumpt claim REPRODUCED
  (raw output ≠ gofumpt; no format flag; no dup issues) → draft ready;
  `nolint-audit ./...` silent no-op REPRODUCED → draft ready; the old
  "strings.Cut FP" claim **REFUTED** on erraudit 1c6809a (stripping the
  directive changes nothing) → NOT filed, stale `status.go` directive
  removed, and the mismatch between nolint-audit's NEEDED verdict and
  the analyzer's actual findings documented as the real bug. Voice check
  passed on the external draft.

## b) PARTIALLY DONE

- **Intent-commit discipline**: 6+ batches still lost their message to
  daemon heuristic commits (`52bfacf`, `4cf1a88`, `6776877`-adjacent,
  etc.) despite same-chain verify+commit — my chains were green but
  multi-second, and the daemon is faster. Code always landed; intent
  often didn't (same failure mode as the 09-23 dedup session, g1 still
  unanswered).
- **erraudit directive audit**: I strip-tested ONLY the `status.go`
  directive. The other three (`handlers.go`, `pusher.go`, `webhook.go`)
  are still trusted on nolint-audit's word — which I *proved* can be
  wrong. Each needs the same strip-and-rerun treatment.
- **Final gate coverage of the tip**: my green chain (build/test/race/
  lint/vet/pre-push) ran through commit ~`4cf1a88`. The parallel
  session's 17:42–17:47 commits (`51e9143`, `db0666c`, `6776877` —
  `sse_integration_test.go` +23/−9) are AFTER it; tree is clean and
  their diff looks contained, but nothing of mine has verified the tip.
- **Alert example "commented example" deviation**: shipped as an inert
  `.yaml.example` + README recipe instead of comments (JSON can't carry
  comments; a real rule file that isn't loaded is more useful than dead
  comments). Documented, but it IS a deviation from the row's wording.
- **verify-dep-bump.sh full chain not executed end-to-end this session**:
  I unit-verified each fix (floor grep, shellcheck, syntax) but never ran
  the whole 12-gate script (no dep bump happened; build/test/race/lint/
  vet were run individually instead).

## c) NOT STARTED

- **Push + CI-on-tip verification**: 35 commits ahead, unpushed (needs
  your authorization). CI is green on `a36a146`, red on `ecccf63` (the
  swept directive — my fix landed after); the current tip has no CI run.
- **The 09-23 dedup session's follow-up list** (their section f): poll
  split-brain (`startByteStableScrapeTarget` vs `waitForTrendSamples`),
  shared JS consts, unified poll primitive, `browser_test.go` file split,
  their HARVEST — untouched (out of my session's scope by your framing).
- **Open dependabot PR**: "Configured Graph Update: go_modules" ran
  successfully against a PR (#1588885104) — I never looked at what PR
  that is or whether it wants merging.
- **`erraudit nolint-audit` upstream filing**: verified + drafted, not
  filed (same as templ — blocked on authorization).
- **vulncheck / coverage floor runs**: not executed this session (new
  tests added; coverage presumably ≥ floor but unmeasured).

## d) TOTALLY FUCKED UP

Nothing irreversible — every mistake was caught and fixed in-session,
and the final state I verified was green. Honest list:

1. **First bisect audit run was methodologically wrong**: I ran 374
   builds with the ambient go 1.26.7 while go.mod's floor is 1.27.1 —
   the exact gotcha I had FIXED in `check-ui-pins.sh` earlier the same
   session. Result: 97 false "broken" commits and ~20 wasted minutes.
   Redone with the pinned toolchain before any of it entered a document.
2. **FEATURES count off-by-one** after M103: I bumped 294→295 forgetting
   the webhook test (+1) from the same session; the drift guard caught
   it (296 actual); fixed in a follow-up commit. Then the M91 test made
   it 297/42 — caught only because I recounted before rewriting
   TODO_LIST.
3. **Lint debt left in-tree for hours**: `browser_latency_test.go`
   shipped with paralleltest/gci/golines findings and full lint only ran
   in the FINAL gate chain — three findings sat in history for ~10
   commits. `nix fmt` was also skipped after individual test-file edits
   (canonical generate→fmt order only honored at the end).
4. **My gate false-positived on itself** (pkgs.templ self-scan matched
   the gate's own comments/error message) — took two fix rounds; and the
   devShell MOTD polluted `nix develop -c` stdout, which I only noticed
   because the UI-pin check printed garbled versions.
5. **set-tag-protection.sh first draft**: used `gh api --slurp --jq`
   (unsupported combination), then compared jq outputs with different
   key ordering (live vs desired could never match). Two iterations
   before `--check` went green against the live ruleset.
6. **The alert example shipped a schema error** (`from: 5m` vs `from:
   300`) — my first activation attempt failed provisioning. Caught by
   exactly the live verification the row demanded, but it's still a
   fabricated-schema bug in a first draft.
7. **Intent commits lost to the daemon ~6 times** despite knowing the
   rule before starting. The same-chain discipline was applied only from
   mid-session onward.
8. **Package mistake in the M91 test** (`package dashboard` vs
   `dashboard_test` — the rig lives in the external test package);
   compiler caught it immediately.

## e) WHAT WE SHOULD IMPROVE (brutal self-review)

- **Q1 What did I forget:** the toolchain floor when designing bulk
  operations (d1) — knowing a gotcha and applying it are different
  things; the webhook test when bumping the FEATURES count (d2); lint
  per-batch instead of at the end (d3); checking for a parallel session's
  fresh commits between my last gate and finishing.
- **Q2 Stupid things we do anyway:** multi-second verify chains that
  give the daemon a window; drafting schemas/API shapes from memory and
  letting "live verification" be the first real check (d6 — right
  safety net, wrong first attempt); trusting a tool's own verdict
  (nolint-audit's "NEEDED") until forced to test it.
- **Q3 Could have done better:** run `golangci-lint` + `nix fmt` after
  every file-level change, not in the tail gate; design long loops
  (audit) against the END state of the toolchain, not the ambient one;
  put the verify+commit in ONE tool call from the first batch; checked
  `git log` for fresh foreign commits immediately before my final gate
  claim instead of at report time.
- **Q4 Could still improve:** strip-test the remaining three erraudit
  directives; run `nix run .#vulncheck` + coverage once over the final
  tree; hand the parallel session's diff a quick test run.
- **Q5 Did I lie:** no. Two claims needed care: (a) "alert example
  live-verified" is true for the FIXED file, not my first draft;
  (b) "all gates green" was true at my last run, not at the current tip
  (parallel session landed after). Both stated as such.
- **Q6 Less stupid:** tag-tree greps instead of commit-log matching for
  the CHANGELOG audit (the daemon makes attribution unreliable — this
  choice is what made the audit trustworthy); live-booting the compose
  stack instead of trusting `compose config` (it caught two real bugs);
  read-only `--check` mode on the ruleset script so verification never
  needed the mutating path.
- **Q7 Ghost systems:** none created; everything wired. One removal
  (stale nolint directive) — verified suppresses nothing before
  removing.
- **Q8 Scope creep:** held to the TODO list + direct findings; did not
  touch the dedup session's follow-up list despite it sitting in
  `docs/status/`.
- **Q9 Removed something useful:** the `status.go` nolint directive —
  deliberate, verified, and recorded; AGENTS.md updated so the old
  guidance doesn't resurrect it.
- **Q10 Split brains:** the compose port-collision knowledge now lives
  in deploy/README.md AND AGENTS.md (short pointer, not a full copy).
  The alert-schema quirk is documented in README + the example file
  header — mild duplication, accepted.
- **Q11 Testing:** full suite + race + lint + vet green post-changes;
  browser suite green pre-parallel-session; the M91 measurement test has
  no assertions by design (informational) — noted in the doc.
- **Q12 What I'd flag for you:** the daemon eating intent messages is
  now the top process debt — it lost 6 messages THIS session and a
  parallel session is committing into the same tree simultaneously.

## f) NEXT — up to 50 things to get done (sorted by impact)

| #   | Task                                                                                                                                                  | Impact |
| --- | ----------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 1   | Verify the tip AFTER the parallel session finishes: `nix run .#build` + `.#test` + `.#lint` on `HEAD`                                                    | High   |
| 2   | Push the 35 commits (needs your go-ahead) and confirm CI green on the tip — esp. the shellcheck job and the new gates                                    | High   |
| 3   | File the templ gofumpt issue (draft ready, voice-checked, `docs/upstream/`) — external repo, needs authorization                                          | High   |
| 4   | File the erraudit nolint-audit issues (./... no-op + NEEDED-mismatch) on larsartmann/erraudit — own repo, needs authorization                             | High   |
| 5   | Strip-test the remaining three `//nolint:erraudit` directives (handlers/pusher/webhook) the same way as status.go                                        | High   |
| 6   | Run `nix run .#vulncheck` + `nix run .#coverage` over the final tree (new tests unmeasured)                                                              | High   |
| 7   | Answer the commit-policy question (g1, asked two sessions running): harness rule vs AGENTS commit-at-first-green — 6 more intent messages lost this session | High   |
| 8   | Compose-stack CI policy decision (g3 below): the e2e boot caught a real Dockerfile break — argues for at least a build-only job                          | High   |
| 9   | HARVEST the 09-23 dedup session's section f into TODO_LIST/ROADMAP (their list is still entombed)                                                         | High   |
| 10  | Kill their known split-brain: `startByteStableScrapeTarget` poll loop vs `waitForTrendSamples`                                                           | High   |
| 11  | Investigate the open dependabot PR behind "Configured Graph Update: go_modules #1588885104"                                                              | Med    |
| 12  | Cut v0.11.0 when you want the hub cadence knob + deploy stack released (`[Unreleased]` is loaded)                                                        | Med    |
| 13  | Fleet-replicate `set-tag-protection.sh` to sibling repos (it was built for this)                                                                         | Med    |
| 14  | Fleet-replicate the fuzz-registry + templ-pin pre-push checks + shellcheck CI job                                                                        | Med    |
| 15  | Re-verify the mobile metadata line against THIS session's render after the parallel session's SSE changes                                                | Med    |
| 16  | Extend the dark-contrast measurement to light mode + badge colors mechanically (script it, don't hand-compute)                                           | Med    |
| 17  | Add the M91 latency numbers as a soft regression assert (e.g. avg < 2s) so a Chrome/launcher regression fails loudly                                      | Med    |
| 18  | Parameterize the load-test doc baseline into a small table the harness can diff against                                                                   | Med    |
| 19  | Alert example: add a second rule draft for `dashboard_pusher_active < 1` (stale-page detection)                                                           | Med    |
| 20  | Grafana: consider provisioning a "degraded view" panel set from `dashboard_health_status` banding                                                        | Med    |
| 21  | Digest-pin the Dockerfile base bump policy note into dependabot config review (does dependabot update `repo:tag@sha256` pins correctly?)                  | Med    |
| 22  | Measure the full verify-dep-bump.sh wall time and record it (operators plan around it)                                                                    | Med    |
| 23  | erraudit upstream: propose the strip-and-rerun mode as a built-in (`nolint-audit --verify`) so the NEEDED mismatch class dies                              | Med    |
| 24  | templ upstream: check whether master already formats output (pin is v0.3.1020; the issue draft should cite the pinned version explicitly)                 | Med    |
| 25  | Add a session-start "verify tip before claiming green" step to the release checklist (this session's d7 class)                                            | Med    |
| 26  | Document the `!override` compose port trick in the fleet lessons file (cross-project reusable)                                                            | Low    |
| 27  | Record a `references/lessons.md` entry: "pin the toolchain in bulk loops; ambient go lies after a floor bump"                                             | Low    |
| 28  | Their f4: extract shared inline-JS consts (`detailsState`, `visibleRows`) in browser_test.go                                                              | Low    |
| 29  | Their f5: unify the five bespoke poll loops behind one deadline-parameterized primitive                                                                   | Low    |
| 30  | Their f6: replace 250ms settle sleeps with a deterministic initial-patch wait                                                                             | Low    |
| 31  | Their f8: split `browser_test.go` (session helpers → own file, ADR-0001 pattern)                                                                          | Low    |
| 32  | Their f9: guard the duplicate-route class (`mustHandle` helper that fatals on re-registration)                                                            | Low    |
| 33  | Re-run art-dupl at `-t 3` after this session's additions (browser_latency_test.go may have introduced near-duplicates with screenshot_test.go)            | Low    |
| 34  | Give `TestMeasureChromeLaunchLatency` a stable machine-tag (skip on CI) so nobody runs it in a shared runner by accident                                  | Low    |
| 35  | Compose: add a `docker compose config` lint to CI hygiene (catches YAML drift without Docker)                                                             | Low    |
| 36  | Compose: prometheus + grafana data-volume mounts for longer local observation windows (opt-in profile)                                                    | Low    |
| 37  | Grafana dashboard JSON: pin panel datasources explicitly (uid "prometheus") instead of inheriting                                                         | Low    |
| 38  | deploy/README: add a "what good looks like" annotated screenshot section (panel-by-panel expected values)                                                 | Low    |
| 39  | docs/metrics.md: add the hub's federation card (`name/reachable`) to the cardinality table with an example                                                | Low    |
| 40  | ROADMAP: write the explicit support-policy section (criterion 3's named gap)                                                                              | Low    |
| 41  | CHANGELOG audit: extend the method to v0.9.0–v0.10.1 (verified only through 0.8.1 + spot checks)                                                          | Low    |
| 42  | Bisect audit: scriptify the pinned-toolchain method into `scripts/` so the next extension is one command                                                   | Low    |
| 43  | pre-push-checks: add check 7 — FEATURES count includes cmd/health-hub test files (currently only root files are guarded)                                  | Low    |
| 44  | Consider `LOADTEST_EVIDENCE=1` knob to toggle evidence explicitly (currently always-on; the knob would make the old-vs-new comparison one-flag)           | Low    |
| 45  | Release-check workflow: also verify the Release notes match the tag's CHANGELOG section (draft-drift class from v0.10.x)                                  | Low    |
| 46  | set-tag-protection: add a CI drift job running `--check` nightly (the script exists; nothing schedules it)                                                | Low    |
| 47  | AGENTS: fold the "verify tip before claiming green" rule into the parallel-session handshake bullet                                                       | Low    |
| 48  | Sweep ROADMAP Open Questions for rows this session made stale (evidence contract wording references HTML-only "today")                                    | Low    |
| 49  | Consider exporting the demo stack's prometheus scrape config as a library example (consumers copy-paste from deploy/)                                     | Low    |
| 50  | Reserved: triage bucket — revisit when fleet tooling changes (BuildFlow DAG ordering, go-structure-linter pin)                                            | Low    |

## g) Questions I can NOT figure out myself

1. **Commit authority conflict (asked two sessions running, still
   unanswered):** AGENTS.md mandates commit-at-first-green; my harness
   forbids committing without your explicit say; the daemon ate six
   intent messages this session either way. Which wins — should I
   always commit at the first green gate in the same tool call?
2. **Filing + push authorization:** may I (a) file the templ issue on
   a-h/templ and the two erraudit issues on your own tracker as drafted,
   and (b) push the 35 commits so CI runs on the tip?
3. **Compose-stack CI policy:** the first manual e2e boot caught a real
   Dockerfile break — does that change your earlier "manual-only demo"
   lean toward at least a build-only CI job (config-validate +
   Dockerfile build, no containers), or does manual-only stand?

---

**Waiting for instructions.**
