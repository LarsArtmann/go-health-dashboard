# Full Status + Brutal Self-Review — v0.12.1 release session, templ-drift triage, draft-workflow bug (2026-10-25, 18:54)

| | |
| --- | --- |
| **Session** | User: "Time for a new release?" → full cut authorized, then "Now you are the only agent" mid-session (inherited the departed dedup session's in-flight state), then this retro demanded |
| **Window** | ~17:20–18:54 CET (single session; auto-commit daemon active throughout; 3 foreign buildflows loading the machine) |
| **Start state** | Master `67b0df3` CI-green; daemon then swept a parallel session's 8-file fixture refactor + RAW templ output; CI Hygiene red on `5025001` before I made any edit |
| **End state** | **v0.12.1 shipped**: signed tag `ef1bf57`, CI 7/7, verify-release 6/6, pages live (v0.12.1 Latest + v0.12.0 disclosure), gate 0 9/9 final, master `2c6898b` green, docs synced (TODO_LIST / AGENTS / ROADMAP / status report) |

---

## a) FULLY DONE

1. **v0.12.1 released end-to-end.** Signed annotated tag on `ef1bf57`; CHANGELOG `[0.12.0]` section + theme blurb, `[0.12.1]` re-cut section, all footer compare-links (incl. fixing a stale `[Unreleased]` base two releases behind); Version const `0.12.1` in the tag commit (version-guard green); SECURITY rows 0.12.x; FEATURES Released row prepended with re-cut provenance.
2. **Full verification chain green.** §4 battery: build, vet, test-race, golangci-lint 0 issues, vulncheck ("No vulnerabilities found"), browser suite **21 PASS / 0 SKIP / 0 FAIL** (counts printed per §4), nix fmt, flake check — all rc=0. Gate 0 9/9 pre-edit AND post-release. Tag pushed FIRST, then master; CI 7/7 on the release commit; `verify-release.sh v0.12.1` 6/6 (proxy Origin.Hash == tag commit, sumdb-verified fresh-cache download, clean-dir consumer `go get`+run printed `0.12.1`).
3. **GitHub release pages created.** v0.12.1 (Latest, full notes: re-cut preamble + verbatim `[0.12.0]` section) and v0.12.0 (disclosure page, `--latest=false`, known-exception banner pointing consumers at v0.12.1 — v0.8.0-page style). Manual creation per §6's fallback because the draft workflow is broken (below).
4. **Master restored to green through three drift fixes.** The templ flip-flop (fmt shape ↔ raw generator output) was triaged to its mechanism: every flake app starts with `go tool templ generate`, which re-raws `*_templ.go` mid-run; the daemon sweeps the raw shape; CI's Hygiene job fails. The re-rawer (editor templ LSP) is confirmed dead (`pgrep` clean), so the shape now holds.
5. **History repaired after my own mistake.** A daemon-race `--amend` put my message on the wrong commit (the departed session's status doc) and diverged the branch; proved tree-identity (`git diff` empty between the two), `git reset origin/master` (mixed), zero data loss, disclosed here.
6. **New latent bug diagnosed and filed.** `release-draft.yml` has NEVER produced a draft for any tag: `gh api --jq '.draft'` masks the 404 exit code (jq succeeds on the error JSON, prints `null`), `state="null"` matches no `case` branch, bash exits 0 — run green, no draft, no output. Diagnosed from run logs (zero stdout after the case dispatch); TODO row + AGENTS discipline item filed.
7. **Docs synced.** TODO_LIST: release row → DONE with evidence; new TODO row for the workflow fix. AGENTS release-discipline: item (5) re-run gate 0 AFTER release edits before tagging; item (6) the draft-workflow bug + manual-page fallback. ROADMAP: v0.12.0 cadence question closed as **DECIDED: CUT**. Status report committed (`2c6898b`).

## b) PARTIALLY DONE

1. **pkg.go.dev spot-check.** The Go **proxy** definitively serves v0.12.1 (verifier checks 2–3: Origin.Hash match + sumdb + clean consumer resolve). The pkg.go.dev **UI** still rendered v0.11.0 as Latest at 18:40 and again at 18:50 on `?tab=versions` — normal crawler lag (minutes to ~1h), three bounded polls done. Not a release blocker; consumers resolve via the proxy. Needs one confirming glance later.
2. **Templ drift root cause.** Symptom triaged and disclosed everywhere, but the structural fix stands open from the 13-38 session: NixOS system templ (v0.3.1020) vs go.mod pin (v0.3.1070) — the bump is an external act; the deeper question (should committed `*_templ.go` be the generator's raw output so generate+fmt can't disagree?) is undecided.
3. **Gate coverage for the release cut itself.** Gate 0 is pre-edit by checklist design; I added the post-edit re-run as discipline TEXT (AGENTS item 5) but not as a script/mechanism — text is how the v0.12.0 miss happened in the first place.

## c) NOT STARTED (deliberately, with reasons)

1. **release-draft.yml fix** — TODO row filed with mechanism (`gh api --jq -e` or state assertion + exercise on next tag); fixing mid-release would have added an untested workflow change to a shipping day.
2. **Harvest of the departed dedup session's §f into TODO_LIST** — that session deliberately withheld it for explicit instruction; I honored the withhold (my release had no stake in it).
3. **chromedp v0.21 migration / CV rollout / deploy bump / consumer sweep** — explicitly decided OUT of this release (independent decisions, not release content).

## d) TOTALLY FUCKED UP (the honest list)

1. **The v0.12.0 red tag was avoidable and is on me.** `scripts/check-changelog.sh` lives in the repo; I had SEEN "changelog lint: all green" in gate 0's pre-edit output. One `bash scripts/check-changelog.sh` after my four edits — seconds of work — catches both failures (no standing `[Unreleased]`, missing `[0.12.0]` footer link). Instead the tag went red on CI and the release needed a v0.12.1 re-cut, duplicating the exact v0.10.1 story the repo already documented. Root cause: I treated gate 0 as a checkpoint passed rather than a tool to re-invoke when its inputs changed.
2. **The wrong-HEAD amend.** I verified `7a2505c`'s content, then amend'ed — and the daemon had moved HEAD twice in between, so my release-shaped message landed on the departed session's status-doc commit and diverged the branch (ahead 1 / behind 1). The verify and the mutate must be ONE atomic command (`git log -1 && git commit --amend …`) in daemon environments; I knew this class of race existed and still sequenced it as two steps.
3. **I re-read the draft workflow's head but not its create-step logic before tagging.** §7 made me re-read `verify-release.sh` pre-tag (done, valuable) — the same 30 seconds on `release-draft.yml`'s `case` statement would have shown the masked-404 no-op BEFORE v0.12.0, and the workflow could have been fixed first, making the whole manual-page detour unnecessary.
4. **curl in a bash script — a rule I know.** The bash tool bans curl; I still wrote a curl-based pkg.go.dev poll into a background script and wasted a poll cycle on 30 refusals. Fetch tool existed for exactly this.
5. **Small but real slop:** my battery script logged every step as "nix" (`run()` used `$1`), making the log self-obfuscating; the test-race step had no pass/skip counts (§4's anti-silent-skip rule — the browser step had them only because I appended `-v` ad hoc); my awk release-notes extraction matched the wrong exit heading and produced a 1307-line notes file (caught by `wc`, but it shipped to /tmp first); `gh run list --commit <short-sha>` returned empty twice before I used branch listing (short SHAs don't match); two `edit` calls failed with "modified since read" because I read via sed instead of the view tool.

## e) WHAT WE SHOULD IMPROVE

1. **Make the post-edit gate mechanical, not textual.** A `scripts/pre-tag-checks.sh` (or a buildflow step) that runs check-changelog.sh + version/const/tag cross-checks + SECURITY-row freshness against the EDITED tree — discipline text is how v0.12.0 shipped red.
2. **Atomic daemon-race mutations.** House rule upgrade: any `--amend`/relabel must re-read HEAD in the same command; if HEAD moved, re-verify content first. Worth a line in the AGENTS daemon-race recipes.
3. **Fix the templ flip-flop structurally.** Either (a) NixOS templ bump so CLI+LSP match the pin (external act), or (b) decide the committed shape question: commit the generator's raw output and exempt `*_templ.go` from gofumpt in treefmt, so generate's output IS the committed shape and the drift check can never disagree. (b) is in-repo and kills the whole class; (a) fixes the LSP squiggle too. They compose.
4. **Rewrite release-draft.yml's state probe** (`gh api --jq -e '.draft'` so 404 exits non-zero, plus a `*)` fallthrough that exits 1 loudly) and exercise it on the NEXT tag, not after.
5. **Battery/logging hygiene:** named steps in gate logs, pass/skip counts on every release-gating test run (not just the one I remembered), and the release checklist's §4 could state the counts requirement per-run rather than per-suite.
6. **Release checklist update:** record that the draft workflow is known-broken (manual `gh release create` is the operative path until the TODO row lands), so the next session doesn't wait for a draft that will never come.
7. **Check the README's tested-version matrix** (spotted during the pkg.go.dev fetch, unverified on master): the rendered README table claims go-health v0.4.0 / templ-components v1.18.0 / go-datastar v0.5.0 / go-sse v0.6.0 while go.mod pins v0.7.0 / v1.23.0 / v0.6.2 — if stale on master, it's a README-drift row for the next docs pass.

## f) NEXT (ranked, ≤50)

1. Fix `release-draft.yml` (`--jq -e` + loud fallthrough) and exercise it on the next tag.
2. Confirm pkg.go.dev renders v0.12.1 Latest (bounded poll; proxy already proven).
3. Mechanize post-edit release gates (`pre-tag-checks.sh`: changelog lint + const/tag/SECURITY cross-checks on the edited tree).
4. Decide + execute the templ committed-shape fix (raw-as-committed vs fmt-as-committed; treefmt exemption if raw).
5. NixOS system templ bump (external act; kills LSP skew + false squiggle).
6. Update release-checklist §6: draft workflow known-broken → manual path is operative; add the post-edit gate-0 step to §4.
7. Review/merge dependabot branch `minor-and-patch-290df43455` (chromedp/cdproto) — gated by (8).
8. chromedp: v0.21 migration (~100 call sites) OR write the pin policy down (fleet decision).
9. CV rollout (deliberately deferred from this release).
10. Deploy stack bump to v0.12.1 (deploy/ is digest-pinned; a release usually wants its deploy row).
11. Consumer sweep: who imports go-health-dashboard at <v0.12.1 (go-etag et al.) and bumps.
12. Harvest the departed dedup session's §f items into TODO_LIST (needs the user's go-ahead it withheld).
13. Mutation-verify `startDashboardFixture` (flip cleanup order, expect suite failures).
14. Promote the art-dupl clone-baseline gate out of fail-open (flake-install art-dupl or a `--plumbing` CI job).
15. Comment at `TestMetrics_PrometheusParserConformance` (why the dashboard intentionally never starts there).
16. Upstream art-dupl filing: summary-line wording instability across flag combos; adopt `--plumbing`.
17. go-etag `pre-release-check.sh`: same fragile-grep class fix (cross-repo).
18. Aggregate-page axe audit (hub cards never audited).
19. Dockerfile base-image go floor vs go.mod 1.27.1.
20. Mirror the SECURITY.md freshness check into release-draft.yml (pairs with fix #1).
21. README tested-version matrix check/refresh (see e.7).
22. securitymd + buildflow upstream filings (from the 14-19 handoff).
23. Cold-cache test codification.
24. devShell tool additions (from the 14-19 handoff queue).
25. `interrogate` evaluation.
26. Buildflow re-run baselines.
27. Markdown-lint pass over docs/.
28. Hub boot smoke test (health-hub against two local remotes).
29. Verify+file upstream StatCard green-tone contrast issue.
30. `ListNote` adoption for the timeline cap (after templ-components#37).
31. Em-dash copy decision for shipped strings, then one mechanical pass.
32. Tag-ruleset bypass policy decision (emergency re-tag path or keep `never`).
33. Unskip go-structure-linter when BuildFlow pins project-config support.
34. Fleet `sort | tail -1` sha anti-pattern audit (verify-release already fixed; other scripts?).
35. Add `[Unreleased]` base-link freshness to check-changelog.sh (it was stale v0.9.0; I fixed it manually — make it a rule).
36. Nightly flake-canary: confirm it actually ran post-merge (it shipped in this release's Added list).
37. Consider `gh run list --commit` full-SHA gotcha → wrapper or documented flag in scripts that poll CI.
38. AGENTS daemon-race recipes: add the atomic-amend rule (log+amend in one command).
39. Extend `pre-push-checks.sh` header note: which gates are pre-edit-only (documented blind spot).
40. ROADMAP: record the raw-vs-fmt templ decision outcome once made (ties to #4).
41. health-hub fuzz targets: confirm the nightly campaign is still green in fuzz.yml runs.
42. Example/demo: `DEMO_*` env surface unchanged this release — sanity-run the example once against v0.12.1.
43. CHANGELOG: consider a "re-cut" convention note (v0.10.1 / v0.12.1 both exist now — document the pattern once).
44. Docs-health sweep of the three kept open files (13-38, 14-19, 07-52) at next natural checkpoint.
45. `TestBrowser_*` count changed to 21 (was 19 in AGENTS prose) — AGENTS testing-patterns bullet says 19; sync the count.
46. Screenshot freshness: no render change shipped, but the next UI change must recapture in the same change (standing rule; no action now).
47. `verify-release.sh` could learn a `--await-proxy` bounded poll (the 0.7.0 lesson) so post-tag waiting is scripted.
48. Consider tagging discipline: `git tag -v` output parsing could be a script (signed-tag confirmation) — nice-to-have.
49. The 07-52 plan's "other 80%" leftovers: CI canary ownership, LOADTEST re-run scheduling.
50. retrospective: fold items (e.1)–(e.3) into the next release's checklist BEFORE the next tag, not after.

## g) QUESTIONS (I cannot answer these myself)

1. **v0.12.0 policy:** keep the red tag + disclosed page as the permanent record (my choice, v0.10.x precedent), or do you want it deleted-and-re-cut as v0.12.1-only (requires your approval for the force-class op; proxy never indexed v0.12.0, so consumer risk is zero either way)?
2. **chromedp:** migrate to v0.21 deliberately (~100 call sites, unlocks the dependabot branch), or write down the "pin v0.16.x until a planned migration" policy as the fleet answer? This gates TODO items 7–8 and other repos will hit the same daemon-bump problem.
3. **templ committed shape:** should I implement the structural fix in-repo (commit the generator's raw output + treefmt exemption for `*_templ.go`, so drift becomes impossible) or do you prefer waiting for the NixOS system-templ bump to make fmt-shape safe? (b) is a one-session change; (a) is an external act on your machine config.
