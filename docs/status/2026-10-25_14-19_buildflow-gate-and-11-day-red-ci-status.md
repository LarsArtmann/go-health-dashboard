# Status + Brutal Self-Review — BuildFlow error-gate closeout and the 11-day red CI nobody saw (2026-10-25, 14:19)

|                 |                                                                                                                                                                                                                                              |
| --------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Session**     | Triggered by the user pasting a `buildflow --fix --build-mode=full` output that exited 69 (one error-severity finding) with the directive READ/UNDERSTAND/RESEARCH/REFLECT, execute-and-verify stepwise, "keep going until everything works" |
| **Window**      | ~13:41–14:19 CET (single session, no concurrent sessions observed after the 13:38 templ-skew report)                                                                                                                                         |
| **Start state** | Clean tree at `e9f687d`; master ahead of origin (auto-push had synced earlier); CI failing on every run of the day                                                                                                                           |
| **End state**   | Clean tree at `e8eeefa`, pushed; **CI 7/7 jobs green**; every local gate green; BuildFlow gate findings (error + the fixable warnings) at zero                                                                                               |
| **Commits**     | `c0991da` fix(flake+security): clear BuildFlow error gate with full flake meta (amend-relabel onto daemon commit `25b31de`/`31ca567` carriers) · `e8eeefa` fix(ci): pin guard downloads the SDK module on a cold cache                       |

---

## §0 Brutal self-review

### What did I forget?

1. **I pushed a red commit to master myself (`c0991da`, CI failure 13:02).** The worst one. I verified every LOCAL gate meticulously — lint, tests, build, flake check, pre-push 8/8 — but never ran `gh run list` BEFORE the first push, even though "verify the check-ui-pins CI wiring / 10-day-red mystery" was literally item 10 on my own todo list. One command, knowable in advance, would have reordered the work (guard fix first, single push) and kept my name off a red master commit. The prior session HELD a push over lint redness; I repeated the same failure class one layer up: local-green ≠ shippable when remote CI state is stale-red. Discovered only afterward, fixed within the session, but the process failure is real and mine.
2. **I pushed before independently verifying the nix FOD build.** The vendorHash had been rewritten mid-day by buildflow's nix-hash-fix; the run summary said "nix-build-verify healed nix-build" and I took the tool's self-report plus `nix flake check` (which does not build the FOD on the default system set) as sufficient. `nix build .#health-hub` only ran — in the background — AFTER both pushes. It passed, but had it not, I'd have pushed a broken flake on tool faith. One independent command owed before shipping.
3. **Close-the-loop is half-kept.** The buildflow skill's step 7 demands every problem ends fixed, filed, or documented. I fixed in-repo and documented, but two upstream-fileable oddities are only sitting in my head/notes: securitymd reporting `filesScanned: 0` in the same JSON where its finding fires (reporting bug candidate), and nix-hash-fix double-WARNing "stale hash not found verbatim" before healing itself in the same run (transient, repro unclear). Neither is filed; "noticed" is not a terminal state.
4. **I never told the other sessions their owed gates are now paid.** The 13:38 templ-skew report listed as owed: full test suite, lint, flake check on the restored tree. I ran all three green — their debt is cleared — but their report dangles unannotated. The docs-health discipline says annotate, and I skipped it as "unrelated"; it was related by one line each.
5. **`./result` left stale.** I built the FOD to `/tmp/gohd-result` while buildflow's vulnix step expects `./result` and had already warned it "predates flake.lock". Next vulnix scan inherits a stale-target warn I could have cleared for free.

### What could you have done better?

1. **Sequence: remote state → fix everything → single push.** The correct order was knowable at minute one: fetch, `gh run list`, see red, root-cause (the log grep I eventually did at 13:0x), fix guard + flake meta + securitymd together, verify, one push, one CI round-trip. Instead: push → discover red → second fix commit → second CI wait. Cost: one wasted CI cycle and a temporarily-red master under my intent commit.
2. **Read the checker's source before iterating on its findings.** For securitymd I went to the local source (found the exact accepted phrase list — good), but for flake-meta-checker I found out about `maintainers`/`platforms` the slow way: one edit, re-run, next info, next edit. Three round-trips where one source read (or even reading the info messages as a checklist) would have been one.
3. **Commit the guard fix at first green, separately.** Release discipline says commit intent-bearing changes right after each verified batch. My guard fix, gotchas entry, and TODO row went in one commit — correct — but the whole batch only became possible because I hadn't committed the flake fix earlier. Smaller batches = smaller red windows. (The daemon won two races anyway; amend-relabel worked both times, splits disclosed.)
4. **State the ambiguity I resolved.** The inherited session had three open questions (push ownership / v0.12.0 cadence / second-session policy) and I acted on the "fix and push" branch based on "keep going until everything works" without flagging the interpretation. It matched the user's intent, but the call was mine and unannounced — it belongs in the handoff, not just in my head.

### What could you still improve?

1. **Red-main detection is still manual.** The 11-day red happened because nobody watches `gh run list`. I documented and opened a TODO row, but no alerting exists. Until wired, every session inherits this blind spot.
2. **CI cannot see the failure class that actually bit.** The FOD/vendorHash build and (today) the cold-cache guard path are invisible to CI's plain-go jobs. A nix-build job would have caught the 10-15..10-25 vendorHash drift class; the guard cold-cache class is now fixed at source, but the general lesson stands: CI should exercise the flake at least occasionally.
3. **The guard's self-heal is tested by simulation only.** My cold-cache proof (`GOMODCACHE=/tmp/...`) faithfully mirrors a fresh runner, but the real CI proof is the green run itself — which passed. Fine. But a future guard change re-learns nothing automatically; the simulation isn't codified anywhere.
4. **AGENTS.md release-discipline bullet not extended.** "Verify batch → commit → push" should say "…and confirm remote CI state + latest run conclusion before pushing." It's the project's own doctrine file and the lesson is exactly in its domain. I put it in gotchas instead; the doctrine bullet is the higher-leverage home.

---

## a) FULLY DONE

1. **BuildFlow error gate cleared.** Root cause: `flake-meta-checker` demanded a meta block, the package carried only `meta.mainProgram`. Fixed with a full block (description, homepage, license, inline maintainers, platforms, mainProgram) at flake.nix:113 and `meta.description` on all 11 apps via a new `mkApp` parameter (flake.nix:44). Targeted step re-run: **0 findings** (was 1 error + 2 infos).
2. **11 `nix-flake-check` app-description warnings cleared.** Same fix; step now exits clean (verified `-s nix-flake-check` → 0).
3. **securitymd warning cleared with rule-level evidence.** Its `missing-response-time` rule (source read at `~/projects/securitymd/pkg/policy/validate.go:60-72`) accepts only conventional phrasings; SECURITY.md's accurate "acknowledgement within 3 business days" matched none. Reworded to "initial response within 3 business days" — commitment unchanged, wording now recognized. Step: 0 findings.
4. **varnamelen findings (`tc`, `cw`) verified resolved.** Already renamed (`testCase`, `csvWriter`) by the concurrent 13:20 session after buildflow ran; I confirmed zero remnants by grep and `nix run .#lint` → **0 issues**.
5. **UI-pin sweep excluded.** The run's `go-mod-update` rewrote go.sum; verified go.mod untouched where it matters — templ-components v1.23.0 ×4, go-datastar v0.6.2/static v0.6.1, chromedp v0.16.0 — and `check-ui-pins.sh` green pre-fix. (gopls's chromedp-v0.21 diagnostics remain stale-cache noise, as documented.)
6. **Full local gate battery green on the exact shipped tree:** `nix run .#lint` 0 issues · `nix develop -c go test ./...` green incl. browser suite (17.7s) · `nix run .#build` · `nix flake check` all checks passed · `scripts/pre-push-checks.sh` 8/8 · `nix build .#health-hub` FOD built OK on the refreshed vendorHash.
7. **Canonical templ generate→fmt state restored.** `nix fmt` exposed that a daemon commit had captured raw generator output (separate imports / `var x =`); ran the sanctioned `nix develop -c go tool templ generate` → `nix fmt` cycle, stamps v0.3.1070 both files, fmt-idempotent (0 changed on re-run).
8. **The CI mystery root-caused with evidence.** Guard IS wired (ci.yml:49 Build, ci.yml:77 Test). Last green run: **2026-10-14T10:15Z**. Two stacked causes: (1) 10-15 pin drift v1.21.0→v1.23.0 (guard updated today by the prior session); (2) today's new SDK-bundle hash check failing on every runner — cold module cache at guard time, script errored with "run 'go mod download' first" instead of downloading (log: `SDK bundle not found at /home/runner/go/pkg/mod/...`).
9. **Guard self-heal implemented and triple-verified.** scripts/check-ui-pins.sh: shared `go_run` helper (existing devShell-fallback pattern) + `go mod download github.com/larsartmann/go-datastar/static` on missing bundle, fail only if still absent. Verified: warm cache (no download, green), cold-cache simulation via `GOMODCACHE=/tmp/...` (downloads into the empty dir, green — mirrors a fresh runner), `shellcheck` clean (CI lints gate scripts).
10. **Shipped and watched green.** `c0991da` + `e8eeefa` pushed; CI run on `e8eeefa`: **7/7 jobs success** (Lint, Hygiene, Vulnerability Scan, Build, Version const, Browser CSP, Test go1.27.1 with race+coverage).
11. **Lessons durably recorded.** docs/gotchas.md "CI was red for 11 days and nobody looked" (both causes, dates, the local-green≠CI-green lesson, securitymd phrasing note); TODO_LIST open row: red-main notification path.

## b) PARTIALLY DONE

1. **Close-the-loop on run problems** — in-repo fixes done and documented (see a8/a9/a11), but two upstream candidates unfiled: securitymd's `filesScanned: 0`-alongside-a-finding reporting oddity; buildflow's nix-hash-fix failing to apply twice before the hash-mismatch case healed it (needs a repro before filing).
2. **Sibling sessions' owed gates** — the 13:38 report's owed full-suite/lint/flake-check are now green on the restored tree (my runs), but neither of today's sibling reports is annotated as satisfied. Reports dangle by one line each.
3. **Red-main prevention** — incident documented + TODO row opened; no actual alerting mechanism exists yet.
4. **vulnix scan freshness** — FOD verified but to `/tmp/gohd-result`; the conventional `./result` (which vulnix scans) still predates flake.lock.

## c) NOT STARTED

1. v0.12.0 release: not cut (cadence question still unanswered; tree is now fully green and releasable).
2. Second-session T13+ ownership question: unanswered; their in-flight files (`cmd/measure-patch`, `trend.go`/`dashboard.go` changes, test sweep) sit in history unreviewed by me.
3. Dependabot branch `minor-and-patch-290df43455` (chromedp/cdproto bump, 08:11): not reviewed — its CI run failed on the same guard red, so it needs a fresh look post-fix.
4. CI failure notification wiring (the TODO row).
5. CI nix-build job (vendorHash/FOD class invisible to CI).
6. Upstream filings (securitymd, buildflow) per b1.
7. The 13:38 report's external queue (NixOS templ bump ≥ v0.3.1070, editor LSP repoint, stamp-guard script, devShell templ wrapper) — theirs to harvest, untouched by me per the do-not-research boundary.
8. `interrogate` tool not installed; shellcheck/dprint/lychee/hadolint/vulnix absent from devShell (buildflow falls back to `nix run nixpkgs#…` every run) — fleet/devShell decision unmade.

## d) TOTALLY FUCKED UP

1. **Shipped a red master commit under my own name.** `c0991da` failed CI (13:02) on the pin guard I was _simultaneously_ tasked to investigate. The knowledge was one `gh run list` away before the push; I checked local gates obsessively and remote state not at all. Mitigated within the session (`e8eeefa` green, red window ~1h), no harm to consumers — but this is precisely the failure class this repo's push-hold discipline exists to prevent, and I re-derived it from the other side.
2. **Trusted a tool's self-reported "healed" for a ship decision.** "nix-build-verify healed nix-build" plus a green `nix flake check` stood in for the actual `nix build` until after the pushes. Nothing broke — the hash fix was correct — but the verification order was backwards and luck-neutral, not skill-based.
3. Nothing else. No data loss, no reverts of others' work, no daemon conflicts beyond the two known amend-relabel recoveries (both disclosed in commit messages).

## e) WHAT WE SHOULD IMPROVE

1. **Make remote state part of the pre-push gate, not the operator's memory.** `git fetch` + ahead/behind + last master CI conclusion should be checked (even warning-only) by `pre-push-checks.sh` or the release-discipline doctrine in AGENTS.md. Today's red-main incident is the argument; b2's "CI cannot see the FOD" is the same principle upstream.
2. **Gate-script changes need CI-green proof on the carrying commit.** Now in gotchas; the stronger form is mechanical: any commit touching `scripts/*` gates should be pushed alone and watched before more work stacks on it.
3. **Never accept a repair tool's "healed" verdict as ship evidence.** One independent command (`nix build`) before push; tools self-report, gates verify.
4. **Read a checker's source before hand-fixing to its findings.** The accepted-pattern list (securitymd) and required meta attributes (flake-meta-checker) were both locally knowable; reading first turns N edit-verify round-trips into one.
5. **Annotate sibling reports when you accidentally pay their debts.** Cross-session bookkeeping is cheap and keeps the docs-health chain unbroken.

## f) Next tasks (ranked)

| #  | Task                                                                                                                                                              | Impact | Effort | Category |
| -- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | -------- |
| 1  | Decide v0.12.0 cadence and cut it if now (tree + CI + FOD all green; version const currently matches v0.11.0 tag so a release needs the const-bump + tag dance)   | High   | M      | Release  |
| 2  | Wire red-main CI notifications (TODO_LIST row): pick GitHub-native vs webhook vs fleet mechanism, implement, test with a deliberate red                           | High   | S–M    | Infra    |
| 3  | Add remote-state pre-push gate: fetch + ahead-count + last CI conclusion (warning-only) in `pre-push-checks.sh` + one-line AGENTS.md release-discipline extension | High   | S      | Process  |
| 4  | Review/merge dependabot branch `minor-and-patch-290df43455` (its only failure was the guard red, now fixed)                                                       | Medium | S      | Deps     |
| 5  | Add a CI nix-build job (or periodic) so vendorHash/FOD breakage is caught remotely                                                                                | Medium | M      | Infra    |
| 6  | Reproduce + file securitymd upstream: `filesScanned: 0` reported alongside an active finding                                                                      | Low    | S      | Upstream |
| 7  | Repro + file buildflow upstream: nix-hash-fix "stale hash not found verbatim" double-WARN before self-heal                                                        | Low    | M      | Upstream |
| 8  | Refresh `./result` (`nix build .# -o result`) so the next vulnix scan target is fresh                                                                             | Low    | S      | Hygiene  |
| 9  | Annotate the 13:38 templ-skew report: its owed gates (test/lint/flake) now green on the restored tree                                                             | Low    | S      | Docs     |
| 10 | Harvest the 13:38 §f queue into TODO_LIST: stamp-guard script (f2), devShell templ wrapper (f3) — both in-repo, S                                                 | Medium | S      | Quality  |
| 11 | Resolve the chromedp question: v0.21 migration (dedicated change + browser suite) or pin-policy decision (TODO_LIST row exists)                                   | Medium | M      | Deps     |
| 12 | Decide dependabot docker entries: drop (images built elsewhere) or add Dockerfiles the repo owns — kills 2 info findings per run                                  | Low    | S      | Config   |
| 13 | README "Installation" section (go-structure-linter info) — 3 lines, clears a finding                                                                              | Low    | S      | Docs     |
| 14 | Record `assets/` suggestion + README-installation as deliberate non-fixes if 13 is skipped (gotchas entry)                                                        | Low    | S      | Docs     |
| 15 | Fleet sweep: propagate the pin-guard self-heal pattern to sibling repos whose guards assume a warm module cache                                                   | Medium | M      | Fleet    |
| 16 | Fleet sweep: propagate the flake meta-block convention (other flakes may fail the same checker error)                                                             | Medium | M      | Fleet    |
| 17 | Codify the guard cold-cache test: a tiny script or CI step that simulates `GOMODCACHE` fresh so future guard edits re-learn it                                    | Low    | S      | Quality  |
| 18 | Decide devShell additions (shellcheck, dprint, lychee, hadolint, vulnix) vs documented nix-run fallback — silences 6 buildflow health warns per run               | Low    | S      | DevEx    |
| 19 | Install or permanently silence `interrogate` (buildflow health warn every run)                                                                                    | Low    | S      | DevEx    |
| 20 | Full `buildflow` re-run post-fixes: confirm 0 errors, rebuild timing baselines (today's regression verdicts are cold-cache noise)                                 | Medium | M      | Quality  |
| 21 | On-demand `buildflow -s gitleaks -s codespell -s markdown-lint` once (8 markdown files were dprint-rewritten today)                                               | Low    | S      | Quality  |
| 22 | AGENTS.md size reduction (406 lines vs 220 budget) — the standing warn; narratives already in gotchas                                                             | Medium | M      | Docs     |
| 23 | Health-hub boot smoke with empty `HEALTH_HUB_REMOTES` (bounded, verifies the FOD binary end-to-end beyond "it built")                                             | Low    | S      | Testing  |
| 24 | Watch `pkg.go.dev` still renders v0.11.0 Latest at release time; re-check after v0.12.0                                                                           | Low    | S      | Release  |
| 25 | At v0.12.0: re-verify release-draft workflow (v0.10.1 dual-cut lessons) before tagging                                                                            | Medium | S      | Release  |

(25 real items; honest ceiling of this session's blast radius — padding toward 50 would be brainstorm, not commitments. Items 7/11/22 are the only M+ candidates beyond the top five.)

## g) Questions I cannot answer myself

1. **Cut v0.12.0 now?** The tree, CI (7/7), FOD, and pre-push gates are all green, so "release when green" is satisfied — but I don't know whether you wanted the second session's T13+ work (measure-patch tool, trend/dashboard changes) inside v0.12.0 or releasable-as-is. Cut now, or wait for that scope?
2. **Red-main alerting: which channel do you actually watch?** GitHub-native email, a webhook (Slack/Discord/ntfy), or a fleet-level solution in crush-config/home-manager. I can wire any of them but cannot see where you'd notice a 3 a.m. failure — and "notice it next session" is exactly what failed these 11 days.
3. **The dependabot docker entries** (`.github/dependabot.yml`: `docker` in `/` and `/deploy`) match nothing in this repo. Are the deploy images built in a sibling repo (drop the entries here), or should `deploy/` gain Dockerfiles this repo owns? I deliberately did not research deploy/ beyond this session's scope.

---

_Written per the operator's explicit `.md`-at-path demand (standing exception to the HTML-canonical status format; consistent with every existing file in docs/status/). Point-in-time snapshot at `e8eeefa`, CI green: annotate, never rewrite (docs-health ANNOTATE)._
