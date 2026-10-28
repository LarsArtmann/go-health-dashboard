# Status: the chromedp v0.21 migration compiles, the fleet is at zero, and the host cannot browse

**Session:** 2026-10-28, 10:33 → 11:25 CET (continuation of the go-health consumption wave that ran overnight)
**Repos touched:** `go-health-dashboard` (primary), `go-health`, `DiscordSync` (read-only verification)
**Verdict in one line:** the two long-standing reds in this repo are now *diagnosed to the root* — one is a compile migration that is finished and unverifiable here, the other is a UI dependency whose markup really did move — but neither is green, and I will not pretend otherwise.

---

## 0. Direct answers to the three questions

### What did I forget?

1. **Test the environment before testing the code.** The browser suite hung twice (10 min, then 2 min) before I checked `chromium --dump-dom` on a bare URL. Thirty seconds of `--dump-dom` at 10:40 would have saved ~20 minutes of goroutine-dump reading.
2. **The daemon commits whatever is on disk.** My throwaway probe file (`zz_probe_test.go`) went into history three times (`a4079e0`, `680d182`, `f50a439`) before I deleted it in `fd0c000`. I knew this behaviour and still left scratch files in the tree.
3. **`GO_HEALTH_DASHBOARD_CHROME=` does not skip browser tests.** `findChrome` falls back to a `PATH` lookup, so the devShell's `chromium` was picked up anyway and the "skip the browser suite" run still hung for 10 minutes. The skip lever is `-skip`, not the env var.
4. **I nearly filed two phantom bugs.** The prior session's notes said DiscordSync had "6 failing snapshot goldens" and a QuickTime mime failure. I verified instead: the mime red is gone, the goldens are red for a completely different reason than recorded. Had I trusted the note, the TODO row would have described the wrong defect.
5. **The golden update I ran to *measure* the drift left 7 regenerated files in the tree** for a few minutes. I restored them from a `/tmp` copy, but I should have copied the directory *and* confirmed the restore before doing anything else.

### What could I have done better?

- **Run `git status` immediately before every write.** Two `edit` calls were rejected with "file has been modified since it was last read" because the daemon had committed or my own script had rewritten the file. I burned four tool calls on that dance and eventually reached for `sed -i` for one-line fixes.
- **Write the migration script correctly the first time.** My first pass treated every Go apostrophe in a comment (`Datastar's`, `chromedp's`) as a rune literal, which put the scanner into a string state for the rest of the file: the script silently rewrote part of `browser_test.go` and none of `screenshot_test.go`, and reported success both times. A scanner that owns a language it only half-parses is a trap; I should have dropped rune tracking from the start.
- **Not let the daemon own my intent messages.** The migration landed as three `chore: auto-commit N changed file(s)` commits; only the cleanup got a real message. Recovering that (`reset --soft` + amend) is exactly what `AGENTS.md` describes, and I skipped it because the report was requested.
- **Decide the UI-pin question before reporting,** not after. Right now the guard is red and I have the evidence to justify either direction, but I have not applied either.

### What can I still improve?

1. Make the UI pin guard's failure *actionable*: today it prints four `::error::` lines and stops. It should also print the golden command (`go test -run TestGoldenRender -golden-update`) and the exact ceremony required, so the next person does not have to reverse-engineer the rule from 200 lines of comments.
2. Record the v1.24.0 markup delta (`role="region" aria-label="…"` on scroll wrappers; the CopyButton status span + `tcAnnounceCopy` script) in the guard's audit history, because that is the actual reason the goldens moved in two repos at once.
3. Add a `-short`-style escape for browser tests that does not depend on `PATH`, so a broken host produces one clean skip instead of a 10-minute timeout.

---

## a) FULLY DONE

### a1. `nix run .#pre-tag` proven end-to-end (first time ever)

In a throwaway clone (`/tmp/pretag-probe`) of `go-health` at `a9d349f`:

```
=== pre-tag: gate sweep (.#gates) ===   → all gates green
=== pre-tag: fleet version skew ===     → minor-stale: 0 (allowed 2), warnings: 0, unreadable: 0
pre-tag: ready to tag <version>
PRETAG_EXIT=0
```

This is the path a tagger actually walks and it had never been executed. It passed *because* of the allow-file work from the previous session; that work is now proven rather than reasoned about.

### a2. The fleet is genuinely at zero skew, and the allowance is gone

`bash tools/fleet-skew.sh --source api` (16 consumers, live default branches):

```
minor-stale: 0 (allowed 0), warnings: 0, unreadable: 0 (allowed 0)
```

All sixteen read `v0.8.1`, including `CV`, `cv`, `nsfw-classifier` and `go-taskqueue` — the four that were stale or in flight when the previous session ended. Following `tools/fleet-skew.allow`'s own rule ("once the backlog is zero, delete this file"), the file is **deleted**, so every caller — the weekly workflow and `nix run .#pre-tag` — now defaults to a zero allowance with no literal anywhere.

### a3. chromedp v0.21 migration: complete, `go vet` clean

The browser suite had 108 compile errors across two files (93 + 15) after the pin moved to v0.21.0 in an unreviewed sweep. All of them are gone.

| Mechanical change | Sites | How |
| --- | --- | --- |
| `chromedp.Run(ctx, steps...)` → `chromedp.Do(...)` | 71 | scripted |
| `chromedp.Evaluate(expr, &out)` / `(expr, nil)` | 57 | scripted → `evalInto[T]` / `evalDiscard` |
| `chromedp.ByQuery` lookup option | 14 | scripted → `chromedp.CSS(...)` selector type |
| `chromedp.NewRemoteAllocator` | 4 | `remote.NewAllocator` (`remote` moved into the core module) |
| `chromedp.ListenTarget(ctx, switch on `any`)` | 1 | rewritten as two `chromedp.Events` iterators |
| `chromedp.FullScreenshot(&png, 95)` | 3 | `captureScreenshot(ctx)` helper (`Run` returns `[]byte`) |

Two small shims carry the shape (`chromedp_eval_test.go`, 36 lines): `evalInto[T]` and `evalDiscard`. `waitForJS` was split into `waitForJS` (wait only, 3 args) and `waitForJSTo[T]` (wait + decode), because the old `res any` parameter cannot express "pointer or nil" once `Evaluate` is generic — seventeen call sites lost a meaningless `, nil`.

`nix develop -c go vet ./...` → **no output, exit 0**. `nix fmt` clean.

### a4. The v1.24.0 UI bump is proven to be behaviour-changing (not just version drift)

This is the finding that matters most for the repo's own red CI. `scripts/check-ui-pins.sh` fails on four modules (`templ-components` root/datastar/icons/utils, `v1.24.0` vs pinned `v1.23.0`). I regenerated the goldens to see *what* moved, then restored them. The delta:

- `display.Table`'s scroll wrapper gains `role="region" aria-label="…"` (an a11y improvement — but new markup).
- `display.CopyButton` gains a `data-tc-copy-status` live-region span, moves `role="status" aria-live="polite"` onto the label, and its inline script gains `tcAnnounceCopy`.

In this repo that drifts **7 golden files** and fails 3 tests (`TestGoldenRender_ZeroProvenWarning`, `_Source`, `_Severity`). In `DiscordSync` the same bump fails its snapshot suite — the diff there shows the identical `role="region" aria-label="Voice states"` insertion and the same `tcAnnounceCopy` script rewrite. One dependency release, two repos, same cause. The guard's ceremony rule was right.

### a5. The DiscordSync "two reds" re-verified, one of them retired

| Prior claim | Re-run 2026-10-28 | Verdict |
| --- | --- | --- |
| 6 failing snapshot goldens after templ-components v1.24 | `go test ./internal/web/ -run 'Snapshot\|Golden'` → **FAIL** (activity/command-audit/voice-states snapshots) | **real**, root cause = v1.24.0 markup (a4) |
| mediapipeline `QuickTime .mov` served as `application/octet-stream` | `go test ./internal/mediapipeline/` → **ok** (1.0 s) | **stale claim**, no longer reproducible |

`internal/web/golden/` (the separate golden package) is green; the failures live in `internal/web`'s own snapshots.

### a6. Housekeeping that would otherwise rot

- Throwaway probe deleted (`fd0c000`); working tree clean.
- 5 leaked Chrome processes from timed-out test runs killed (profile dirs `TestScratch*`).
- Redirected `go-health`'s TODO_LIST row for the stragglers from `BLOCKED` to `DONE` with the measured numbers, and corrected the neighbouring row that still blamed the pins for the CV clone pair (the pins are aligned; what remains is local lineage).

---

## b) PARTIALLY DONE

### b1. The migration is compile-verified but **not** runtime-verified

`go vet` is clean and the API mapping is right by construction, but no browser test has executed the new code. Everything below in (d1) explains why. Claiming "migrated" without runtime proof is exactly the class of statement this repo's own history warns about.

### b2. The UI-pin decision is prepared but not applied

Two defensible end states:

- **Revert** `templ-components` to `v1.23.0` (the set audited green on 2026-10-25): guard goes green immediately, goldens pass, drift disappears, and the v1.24.0 adoption becomes a dedicated change.
- **Adopt** `v1.24.0`: regenerate the 7 goldens, update the pin block with the markup delta in a4, and accept that the browser-suite half of the ceremony could not run today.

I have the evidence for both and applied neither.

### b3. `go-health`'s skew commit is unpushed

`15fad10` is one commit ahead of `origin/master`. It is small and self-contained (allow-file deletion, workflow comment, TODO row) but until it is pushed the fleet's zero-skew state is recorded only locally.

### b4. DiscordSync TODO rows not written

I verified the reds (a5) but did not add the two rows to `DiscordSync/TODO_LIST.md`, because one of the two claims dissolved and the surviving one is a v1.24.0 decision that belongs to the same call as b2 — filing it as an unrelated DiscordSync task would split the fix.

---

## c) NOT STARTED

1. **M04 evidence table** (`docs/fleet-consumption-evidence.md` or similar): repo, module, pin, result, command, date. The data exists (a2's 16-row dump is version evidence; six consumers were build-verified in the previous session) but nothing is published. The honest version needs a *re-built* row per consumer, not a carried claim.
2. **M19** (`company-master-data` → `checks.Disk`). Still deferred for the same structural reason: the probe is mounted through `go-appkit` (`*appkithealth.Mounted`), the repo is forgejo-only with no CI, and it is deliberately absent from `tools/fleet-repos.txt`.
3. **`cv` push.** `~/projects/cv` still carries `34b219f22` (three unpushed commits' worth of divergence) and `~/projects/CV` has an uncommitted pin bump. Both need the owner's lineage decision.
4. **go-taskqueue `NewChecks` port (`M06`).** Still blocked on the two API gaps recorded in `go-health/TODO_LIST.md`.
5. **Fleet-wide golden regeneration** for the v1.24.0 markup (this repo + DiscordSync + any other templ-components consumer with snapshots).
6. **A browser-test skip switch that survives a broken `PATH`.**

---

## d) TOTALLY FUCKED UP

### d1. This host's Chromium cannot load any HTTP URL, so the browser suite cannot run at all

Not my code — but it blocks the half of the migration that matters, and it took me 20 minutes to notice.

```
$ chromium --headless --no-sandbox --disable-gpu --dump-dom http://127.0.0.1:8977/
EXIT=124 (timeout 25s), 0 bytes on stdout; the local http.server log shows ZERO requests
$ chromium --headless --no-sandbox --disable-gpu --dump-dom https://example.com
EXIT=124 (timeout 30s), 0 bytes
```

Inside the same failing session, a `data:` navigation via CDP succeeds in **512 ms** — so the transport, the allocator, `Do`, and the two `Events` subscriptions are all exercised and working; only networked loads hang. Corroboration: an unrelated agent's Chrome screenshot job (`--screenshot` against `127.0.0.1:8491`) has been stuck for 32+ minutes, and the host is at load average 51 with ~95 Chromium processes. `--single-process --no-zygote` and `--headless=old` behave identically.

Consequence: `go test ./...` inside the devShell hangs for the full 10-minute timeout; the "skip browser tests" run (`GO_HEALTH_DASHBOARD_CHROME=`) hangs too, because `findChrome` falls back to `PATH`.

### d2. The migration landed as three daemon commits

`5f716da`, `3fa29d2`, `5eae4ac` — all `chore: auto-commit N changed file(s)`. The intent (what the three shims are for, why `waitForJS` split) exists only in this report. Recoverable with `git reset --soft` + amend; not done because the report was requested first.

### d3. My first migration script silently half-migrated a file and reported success

Apostrophes in Go comments flipped the scanner into rune-string state; `browser_test.go` was partially rewritten and `screenshot_test.go` untouched, both reported as done. Caught by grepping for the old symbols afterwards — which is the only reason it did not become a mystery later.

### d4. Two `edit` calls were applied to stale snapshots and silently did nothing

The multiedit that added the `remote` import and `captureScreenshot` reported success, but the file on disk never changed (the daemon had committed between my read and my write). I only noticed because the next `go vet` still failed. Re-reads and `sed -i` fixed it, at the cost of four wasted calls and one confused minute.

### d5. The repo is still red in two independent ways

`scripts/check-ui-pins.sh` fails by design (4 modules at v1.24.0) and 3 golden tests fail from that same markup change. The chromedp compile break is gone; the CI job will now fail *later* in the chain instead of at compile time. Net: I moved the failure, and the second failure is now better understood — but master is not green.

---

## e) WHAT WE SHOULD IMPROVE

1. **Environment canary first.** A one-line `chromium --dump-dom data:text/html,ok` + `… --dump-dom <local server>` preflight at the top of the browser suite would turn today's 10-minute timeout into a 2-second skip with a named reason.
2. **Scratch files out of the tree.** Any probe file must live under `/tmp` or be git-ignored; the daemon will commit whatever it finds.
3. **Scripts that parse a language must not half-parse it.** Either use `go/ast`/`gofmt -r` or restrict the transform to line-oriented, unambiguous patterns and *assert* the post-condition (zero remaining old symbols) inside the script itself.
4. **The pin guard should teach.** Print the exact ceremony (goldens command, browser command, where to record the audit) in its failure output.
5. **One decision per dependency bump.** The v1.24.0 markup change touches two repos' goldens; it deserves a single, dated decision record rather than two independent "why is this red" investigations.
6. **`retention` for the reports.** `docs/status/` in this repo now holds 5 files from 2026-10-25 plus this one; the corpus budget gate exists for the *other* repo. Rotate older ones to `archived/` on a schedule instead of at 20 files.
7. **Ask the host question earlier.** Load 51 / 95 Chromium processes is a shared-resource problem across concurrent agent sessions; a cheap preflight (`nproc`, loadavg, chromedp launch latency) before starting browser work would flag it.

---

## f) Up to 50 things to get done next

**Immediate (this repo, this branch)**

1. Decide the v1.24.0 question (b2) and apply it: revert to v1.23.0, or regenerate the 7 goldens + update the guard's pin block with the a4 delta.
2. Re-run `scripts/check-ui-pins.sh` and prove exit 0 (note: the current run's `PINS_EXIT` was masked by a pipe — use `set -o pipefail`).
3. Get one green browser-suite run on the migrated code — on a host whose Chrome can load a URL, or in CI — before calling a3 done.
4. Recover the daemon commits' intent: `git reset --soft HEAD~3` + one amend with a real message (or a `docs:` commit that carries the rationale).
5. Push `go-health` `15fad10` after a `nix run .#gates` pass.
6. Add the "v1.24.0 markup delta" paragraph to `scripts/check-ui-pins.sh`'s audit history.
7. Add a Chrome preflight to `browser_test.go` (env canary + `t.Skipf` with the reason).
8. Make browser tests skippable without `PATH` gymnastics (`-skip` documented in `AGENTS.md`, or an explicit `GO_HEALTH_DASHBOARD_SKIP_BROWSER=1`).
9. File the DiscordSync golden row referencing the shared v1.24.0 cause (not as an isolated mime bug).
10. Write the M04 evidence table, honestly labelled "pin verified remotely / build verified locally on <date>".

**Near term (fleet hygiene)**

11. Re-derive the six consumer builds from scratch (not from the previous session's notes) and record commands + exit codes.
12. Ask/decide the `cv` vs `CV` canonical-lineage question and push the health adapter work.
13. Land the go-taskqueue `NewChecks` port once the pass-detail API question is answered.
14. Re-run `nix run .#skew` weekly and keep the number at zero; a reappearing stale repo is drift, not a tolerance.
15. Delete the now-dead `--allow` documentation in `tools/fleet-skew.sh`'s header? (No — it still describes `--allow` correctly; confirm and leave.)
16. Promote `#pre-tag` from "proven once in a clone" to "run before every tag", and note the exact command in `CONTRIBUTING.md`.
17. Add the `.#pre-tag` app to CI as a weekly dry run with a dummy version.
18. Check whether other templ-components consumers (`timesheets`, `Zlota44`, `webphone`, `projects-management-automation`) have snapshot tests that will drift on v1.24.0.
19. Archive `docs/status/2026-10-25_*` into `archived/`.
20. Add this session's lesson ("prove the environment before the code") to the repo's AGENTS.md gotchas.

**Medium term**

21. Reconsider the browser harness: chromedp v0.21's default `ExecAllocator` uses a **pipe**, which sidesteps this machine's IPv6/DevTools-port problem entirely and removes ~60 lines of manual launch code.
22. If the pipe transport is adopted, delete `startHeadlessChrome`'s stderr parsing and the profile-dir retry logic.
23. Replace `chromedp.Events`+goroutine error collection with `chromedp.WaitEvent` where a single event matters.
24. Revisit `evalInto`/`evalDiscard`: with `Action[T]` composition, some step lists could return values directly instead of writing through pointers.
25. Type the test helpers (`waitForJSTo` already is; `evalInto` could take a decoder).
26. Add a fuzz/property test for the JS-expression builders (`visibleRows`, `detailsState`, `pillState`) so a selector typo fails fast.
27. Re-examine the 3 failing goldens' *content* — do we want `role="region"` at all, or should the wrapper use the table's own `aria-label` only?
28. Upstream: ask templ-components for a changelog entry per markup change, so a sweep can be triaged without regenerating goldens.
29. Add a `make goldens` (or `nix run .#goldens`) app that regenerates and prints the semantic diff.
30. Record the golden-drift policy in `docs/adr/` (accept + regenerate is a decision, not a chore).

**Longer term**

31. Cross-repo dependency sweep automation that opens a PR per consumer when goldens drift.
32. Fleet-wide "markup fingerprint" check: hash the rendered table wrapper of a golden fixture in every consumer.
33. Replace the hand-maintained `tools/fleet-repos.txt` with a query of the GitHub API (org listing + go.mod probe).
34. Make `fleet-skew.sh` able to *bump* the stale repos (`--fix`), not just report.
35. Reconcile the two `cv` clones with a documented canonical path.
36. Add `go-health`'s pass-detail channel (`Check.Detail`) so `NewChecks` adopters keep their green-row notes.
37. Revisit `WithCriticalService[T]` naming/behaviour for recorder-renamed keys.
38. Document `Response.Clone`'s nil-map behaviour in `types.go` + a test.
39. Add a `WithDeploymentFreshness` helper so consumers stop comparing wall clocks (`CachedResponseFresh` is the seam).
40. Publish `docs/consumer-verification.md` as the single evidence surface for the adoption wave.
41. Add CI job that runs the browser suite on a runner with a working Chrome (today's host cannot).
42. Measure Chrome launch latency again once the host is healthy (`TestMeasureChromeLaunchLatency`).
43. Re-run the 20-source load harness in the new evidence era.
44. Clean the stale `docs/gotchas.md` chromedp entry (it describes the v0.16-era launch problem) and add the v0.21 pipe note.
45. Reduce `browser_test.go`'s 2452 lines by extracting fixtures (`toggleService`, `provideHealthy`) into a helpers file.
46. Add a `TestBrowser_*` count assertion so a silently skipped suite is visible (19 expected).
47. Land the `zz_probe`-style isolation recipe as a documented debugging playbook ("is it me or the host?").
48. Backfill `docs/status/INDEX.md` with this report.
49. Verify the fleet skew from a *clean* environment (no `gh` token) at least once, to exercise the `--source local` path.
50. Retire the `--allow` mechanism entirely if the backlog stays at zero for a month.

---

## g) Three questions I cannot answer myself

1. **v1.24.0: revert or adopt?** Its markup delta is an a11y improvement (`role="region"` on scrollable table wrappers, a `role="status"` copy announcement) but it landed unguarded, drifts 7 goldens here and breaks DiscordSync's snapshots, and the ceremony's other half — a green browser suite — cannot run on this host today. Revert restores the audited baseline; adopt accepts an unaudited-at-runtime set. I will not pick for you.
2. **Is the broken host mine to fix?** Chromium here loads no http/https URL at all (bare `--dump-dom` times out; load average 51; ~95 Chromium processes including another agent's 32-minute-stuck screenshot job). If this is a shared-resource problem, restarting things could break a peer's run; if it is mine, I should chase it now. Which?
3. **Who owns the `cv` / `CV` lineage?** Two clones of one repository, each ahead of `origin` with no shared tip; pins are now aligned on both remotes, but three commits' worth of health-adapter work sits unpushed in `~/projects/cv`. Deciding this unblocks the last two fleet rows and the `M08` evidence.

---

## Appendix — commands and exit codes from this session

```bash
# A. release gate, end to end, in a throwaway clone
git clone ~/projects/go-health /tmp/pretag-probe && cd /tmp/pretag-probe
nix run .#pre-tag            # all gates green; minor-stale: 0 (allowed 2); EXIT=0

# B. fleet skew after the allow file was deleted
cd ~/projects/go-health && bash tools/fleet-skew.sh
# minor-stale: 0 (allowed 0), warnings: 0, unreadable: 0 (allowed 0)   EXIT=0

# C. migration compile gate
cd ~/projects/go-health-dashboard && nix develop -c go vet ./...
# (no output) EXIT=0

# D. UI pin guard (masked exit is a pipe artefact; the ::error:: lines are the signal)
nix develop -c bash scripts/check-ui-pins.sh
# ::error:: templ-components{,,/datastar,/utils,/icons} is v1.24.0, expected v1.23.0

# E. goldens
nix develop -c go test ./... -count=1 -skip 'TestBrowser|TestCaptureREADME|TestMeasureChromeLaunchLatency'
# --- FAIL: TestGoldenRender_ZeroProvenWarning / _Source / _Severity  (markup drift)
# ok: cmd/health-hub, pkg/version

# F. the host, not the code
chromium --headless --no-sandbox --disable-gpu --dump-dom http://127.0.0.1:8977/   # EXIT=124, 0 bytes
chromium --headless --no-sandbox --disable-gpu --dump-dom https://example.com      # EXIT=124, 0 bytes
# same session, via CDP: Navigate("data:text/html,…") → 512ms, PASS

# G. DiscordSync re-verification
cd ~/projects/DiscordSync
nix develop -c go test ./internal/web/golden/ -count=1        # ok
nix develop -c go test ./internal/mediapipeline/ -count=1     # ok  (QuickTime red retired)
nix develop -c go test ./internal/web/ -run 'Snapshot|Golden' -count=1   # FAIL (v1.24.0 markup)
```

**Commits made**

| Repo | Commit | Subject |
| --- | --- | --- |
| go-health | `15fad10` | chore: drop the fleet-skew allowance now that the backlog is zero (unpushed) |
| go-health-dashboard | `5f716da`, `3fa29d2`, `5eae4ac` | daemon captures of the chromedp migration (no intent message) |
| go-health-dashboard | `fd0c000` | chore: drop the throwaway chromedp probe |

💘 Generated with Crush

Assisted-by: Crush:hf:deepseek-ai/DeepSeek-V4.1-Flash-1017
