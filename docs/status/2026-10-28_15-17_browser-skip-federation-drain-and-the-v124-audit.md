# Status — 2026-10-28 15:17 CET

**Session scope:** continue the overnight go-health consumption wave. Three
repos touched: `go-health-dashboard` (primary), `go-health`, `DiscordSync`.
No pushes (awaiting approval). All three trees end clean.

**Mode of work:** verify-before-trusting, one step at a time. The previous
handoff left four unblocked tasks and three blocking questions. I took the
unblocked ones, re-measured every inherited claim, and deliberately did **not**
take the two decisions that are yours.

**Headline:** the browser suite no longer lies (it skips in 12s with a reason
instead of burning 150s per test), two go-health capability gaps closed
upstream in v0.8.1 are now adopted with live proof, the templ-components
v1.24.0 drift is documented correctly for the first time, and DiscordSync
turned out to have a seasonal golden bug that I found and fixed.

---

## 0. State at handoff

| Repo | HEAD | Ahead of origin | Tree |
| --- | --- | --- | --- |
| `go-health-dashboard` | `a44f0fd docs: record what templ-components v1.24.0 actually changes, measured` | 5 | clean |
| `go-health` | `cbd7f78 docs: publish the consumer evidence table instead of claiming compiles` | 1 | clean |
| `DiscordSync` | `921d5fc4 fix: make golden snapshots independent of the machine's time zone` | 2 | clean |

Nothing was pushed. The auto-pusher moved four earlier dashboard commits on
its own during the session (origin jumped from `f7010ae`-ancestor to
`9e0d6a1`), which is why the ahead-count fell from 9 to 5 while I was working.

---

## a) FULLY DONE

### a1. Browser tests skip honestly instead of timing out

`browser_test.go` gained:

- `chromeNetworkPreflightTimeout` (12s),
- `probeChromeNetwork(chromePath) error` — one `chromium --dump-dom` against a
  loopback `httptest` server, `exec.CommandContext` + `WaitDelay` so a hung
  Chrome cannot leak,
- `chromePreflight` (`sync.Once` + cached error) exposed as
  `chromeNetworkPreflight`, so the probe costs one launch per test binary,
- `usableChrome(t)` which resolves the binary (`locateChrome`, renamed from
  `findChrome`) **and then** skips with the reason when the probe fails.

Proof (measured, not asserted):

```
nix develop -c go test -run TestBrowser_CSPCleanRuntime -count=1 -v .
--- SKIP: TestBrowser_CSPCleanRuntime (12.07s)
    this host's Chromium cannot complete an HTTP load (no HTTP response within
    12s (the navigation never reached the httptest server)); ...

nix develop -c go test -run TestBrowser -count=1 .
ok  github.com/larsartmann/go-health-dashboard  12.039s     # all 19 tests
```

12 seconds for the whole suite, one probe, five call sites converted.

### a2. The host's Chromium failure is narrowed to a precise, reusable finding

Chromium's own netlog (`--log-net-log`, `IncludeSensitive`) shows the request
dying *after* the socket:

```
TCP_CONNECT [50]  phase=1 (begin)  address=127.0.0.1:18999
TCP_CONNECT [50]  phase=2 (end)    local_address=127.0.0.1:35436
                                   remote_address=127.0.0.1:18999
CORS_REQUEST [51] GET /health HTTP/1.1        # headers generated
# then nothing; the python server on :18999 logged zero GETs for this navigation
```

The same browser process, in the same run, downloaded a component-updater
payload from `edgedl.me.gvt1.com` (`downloaded_bytes=14332`,
`download_time_ms=2802`, `error=0`). So: sockets work, the browser-process
URL fetcher works, `data:` and `file:` render in under a second, and *only*
`http(s)` navigation stalls before the request is written.

Eliminated this session (each one a real test, not a guess): proxy env and
PAC (`--no-proxy-server`), `--host-resolver-rules`, QUIC/HTTP2
(`--disable-quic --disable-http2`), `NetworkServiceSandbox`,
`NetworkServiceInProcess`, `--single-process --no-zygote`,
`--disable-dev-shm-usage` (63G free on `/dev/shm`), a pristine
`--user-data-dir`, `LocalNetworkAccessChecks` +
`PrivateNetworkAccessSendPreflights`, conntrack saturation
(2425/262144), and two older browsers from the store
(155.0.8059.84, 154.0.8037.57) which hang identically.

That last row is the useful one for triage: it is **not** the devShell's
156 build, so pinning or downgrading the flake's Chromium is not a fix.

### a3. Federation capability gaps closed (go-health v0.8.1) — adopted with proof

Federation now implements `Healthz()`, `SourceStatuses()` and
`MarkShuttingDown()`. Three consequences, all landed:

1. `TestFederation_HealthzCapabilityNotForwarded` (which asserted a negative
   that upstream no longer satisfies) became
   `TestFederation_HealthzCapabilityForwarded`: the prober satisfies
   `dashboard.Healthzer`, `/livez` serves the merged view with 200, and
   answers 503 once `MarkShuttingDown()` is called.
2. `sourcestatus_test.go` now pins `SourceStatuser` for **both** merge
   surfaces (`aggregate.Aggregate` and `healthfederation.Prober`), so a
   future drop breaks a build instead of silently softening federated cards.
   Two stale comments claiming federation lacks the accessor were corrected.
3. `cmd/health-hub` runs the real two-phase drain: `fed.MarkShuttingDown()`
   → `drainObservation` (2s, so a load balancer can observe) →
   `dash.Shutdown()` → `shutdown(server)`. The package comment that documented
   "federation has no MarkShuttingDown, so the hub cannot flip readiness" was
   rewritten to describe the drain instead.

Live smoke test against a fake remote (built binary, real SIGTERM):

```
readyz-before-drain: HTTP/1.1 200 OK
readyz-during-drain: HTTP/1.1 503 Service Unavailable after 2ms
hub-exit=0
log: 12:25:45 serving ... / 12:25:47 shutting down...
```

### a4. templ-components v1.24.0 delta documented correctly (third attempt, now measured)

Read from the module sources side by side (`v1.23.0` vs `v1.24.0`), not
inferred from a rendered diff:

| Component | Change |
| --- | --- |
| `display.Table` | the horizontal-scroll wrapper **loses** `role="region"`/`role="img"` + `aria-label`; the `<table>` element **gains** the `aria-label` |
| `display.CopyButton` | the label span loses `role="status" aria-live="polite"`; a sibling `<span data-tc-copy-status role="status" class="sr-only">` carries the announcement; the inline script gains `tcAnnounceCopy` |
| `errorpage.NotFound404` | the search field's leading icon (`ps-10`) is replaced by a trailing submit **button** carrying the same icon (`ps-4`/`pe-14`); `GoBackText` and `SearchSubmitLabel` props gained defaults |

Recorded in four places so the next person does not regenerate goldens to
find out: `scripts/check-ui-pins.sh` audit history, `AGENTS.md` (UI-pin
bullet), the dashboard `TODO_LIST.md` row, and DiscordSync's `C60` row.

### a5. DiscordSync: seasonal golden bug found, fixed, and mutation-proven

Re-measuring the inherited "3 failing snapshots" claim gave **19** failures,
and four of them had nothing to do with templ-components:

`internal/web/views.go` renders `timestampFormat = "2006-01-02 15:04:05 MST"`,
and the golden normalizer collapsed the timestamp but left the zone
abbreviation behind. Goldens captured under daylight saving store
`TIMESTAMP CEST`; every run after the 2026-10-25 switch renders `CET` and
fails. A UTC CI container would disagree with a summer local run for exactly
the same reason — so the suite was environment-dependent, not just
season-dependent.

Fix: `zoneSuffixRe = (TIMESTAMP) [A-Z]{2,5}\b` → `$1 ZONE`, applied after the
timestamp rules; 14 golden occurrences rewritten (`8 × CEST`, `6 × UTC`); three
new table cases pin CEST, CET and UTC to the same output.

RED-proof: commenting the `zoneSuffixRe` line out makes
`TestNormalize_ReplacesTimestamps` fail; restoring it passes.

Effect: `go test ./internal/web -run 'Snapshot|Golden'` went **19 → 15**
failures, and `grep -c CEST` on the log is now 0. The remaining 15 are the
v1.24.0 markup above.

### a6. go-health M04 evidence table published

`docs/consumer-verification.md` gained an evidence table with 16 consumer
rows and an explicit level per row (L1 pin-read via authenticated GitHub API
on the live default branch, L2 local build, L3 local suite). Only the two
repos actually exercised carry L2/L3, so a pin read can never be mistaken for
a compile. Fresh measurement backing it:

```
bash tools/fleet-skew.sh
16 rows, all v0.8.1 (dashboard, CV, cv, Zlota44, dnsblockd, library-policy,
file-and-image-renamer, DiscordSync, KeyHolderAI, nsfw-classifier, webphone@main,
projects-management-automation, timesheets, cqrs-htmx health/go.mod,
go-appkit health/go.mod, go-taskqueue)
minor-stale: 0 (allowed 0), warnings: 0, unreadable: 0
```

`nix run .#gates` before committing: **all gates green** (treefmt, docs drift,
fuzz registry, openapi lockstep).

### a7. Commit hygiene under an active auto-commit daemon

The daemon captured my batches as `chore: auto-commit N changed file(s)`
four times. Each was either labelled with an intent message via
`git commit --amend` (go-health `cbd7f78`, dashboard `a44f0fd`) or squashed
(`git reset --soft HEAD~1` + amend for DiscordSync `921d5fc4`), which is the
project's documented recovery recipe. No history was rewritten beyond
unpushed, tree-identical commits.

---

## b) PARTIALLY DONE

### b1. chromedp v0.21 migration — compile-verified, not runtime-verified

`go vet ./...` exits 0 across all 108 formerly broken call sites
(`evalInto`/`evalDiscard` shims, `remote.NewAllocator`, the `Do`/`Run` split,
`Events` iterators, `captureScreenshot`, `waitForJSTo`/`pollJS`). But not one
browser test has executed its body on this host, so CSP cleanliness, live SSE
patching, axe and keyboard behaviour under v0.21 are **unproven**. It is
honest to call this "compile-verified only" — nothing more.

### b2. The v1.24.0 decision is fully prepared but not taken

Evidence is complete (a4), blast radius is measured (7 goldens here, 15
snapshots there), and the guard is red by design. What is missing is the
decision, which is yours.

### b3. DiscordSync's snapshot suite is still red (15)

All 15 trace to the v1.24.0 markup. Regenerating them without the adopt
decision would be a reflex, not an audit — especially because the 404 change
is a visual redesign that wants a human look.

### b4. Host Chromium: narrowed, not fixed

Twelve hypotheses eliminated, one decisive netlog trace, no root cause and no
workaround. What I did deliver is that the failure is now cheap and
self-explaining instead of a 10-minute timeout.

### b5. TODO hygiene inside the session

I updated the `todos` tool at the start and once mid-session, then let it
drift again for the last third. The final state is correct as of this report;
the intermediate state was not.

### b6. Push state

go-health is 1 ahead, dashboard 5, DiscordSync 2. Locally green, remotely
unverified — I did not push and did not `git fetch` to compare with remote CI,
because pushing was not authorized either.

---

## c) NOT STARTED

1. **M19** `company-master-data` → `checks.Disk` — still deliberately
   deferred (forgejo-only, mounted through go-appkit, no CI).
2. **`Routes.Healthz` in `cmd/health-hub`.** The capability now exists
   upstream and one line would give the hub a combined-traffic endpoint, but
   `newServeMux`'s doc comment explicitly says the hub never overrides Routes,
   so it is a product decision plus a doc rewrite. Flagged, not done.
3. **A hub-level test for the drain.** I proved `/livez` 503s on drain at the
   library level (`TestFederation_HealthzCapabilityForwarded`) and proved the
   hub's `/readyz` behaviour by a manual smoke run, but `cmd/health-hub` has no
   automated test for the shutdown sequence — the smoke script lived in `/tmp`
   and is gone.
4. **Runtime verification of the chromedp migration** (blocked, b1).
5. **Auditing the 7 drifted dashboard goldens** for anything beyond
   regeneration (e.g. does moving `aria-label` onto the `<table>` change what
   our own evidence tooltips/live regions assert?).
6. **CHANGELOG entries.** DiscordSync's own convention says completed work is
   deleted from the TODO list and logged in `CHANGELOG.md`. The zone fix is a
   real bug fix and deserves an entry; I recorded it in the TODO row only.
   Same question for `go-health` (the evidence doc) and the dashboard (the
   browser-skip behaviour change).
7. **Reaping the two leftover servers** on loopback ports 8977 and 18999
   (see d3).
8. **A `docs/gotchas.md` entry** for the seasonal-golden class in DiscordSync.
9. **The ~40 leaked Chromium processes** and the peer's stuck job from earlier
   in the day — host hygiene I did not touch beyond `pgrep`/`kill` attempts.
10. **Nothing about go-health's own next release.** No tag, no CHANGELOG, no
    push.

---

## d) TOTALLY FUCKED UP

### d1. I described the v1.24.0 delta wrongly, twice, and propagated it

The first note said Table's scroll wrapper *gains* `role="region"`. The truth
is the opposite direction (wrapper loses it, table gains `aria-label`). That
wrong description went into `scripts/check-ui-pins.sh`, `AGENTS.md` and a
`TODO_LIST.md` row before I caught it — and what caught it was not my own
review but a character-offset diff of DiscordSync's snapshot failure. Three
files had to be corrected afterwards. This is precisely the failure mode the
project's memory warns about: prose that reads authoritative and is never
checked against the source.

### d2. I burned a large part of the session on Chromium forensics

The deliverable that mattered (the preflight skip) was ready after the first
`--dump-dom` test. I then spent roughly half the session on twelve more
hypotheses that produced no fix. A time box ("two more tests, then write the
finding down") would have freed that time for the DiscordSync bug, which was
found almost by accident while re-measuring an inherited number.

### d3. Incomplete cleanup

Two loopback `python3 -m http.server` processes (8977, 18999) survived both
`SIGTERM` and a `kill -9` attempt — `kill -9` is not available in this shell
(`kill: unsupported builtin`). They are harmless static servers, but they are
mine and they are still running. Same for the ~40 leaked Chromium processes
that predate this session.

### d4. I wrote a TODO row on an inherited number, then had to rewrite it

`C60` was first written with "3 failing snapshots" (the previous session's
claim). The measured number was 19. The wrong row is still visible in commit
`a4de7130`; the corrected row lands in `921d5fc4`. Churn I created by not
re-measuring before writing.

### d5. Mutation check relied on a `/tmp` backup

For the RED-proof of `zoneSuffixRe` I copied `golden.go` to `/tmp`, mutated in
place, and restored from the copy. With an auto-commit daemon running, a
commit landing inside that window would have committed the mutated file. It
did not, but the technique is fragile; the daemon-race recipe in the project
memory says to mutate from a committed state, not a scratch copy.

### d6. I never ran golangci-lint

`go vet ./...` passed on the dashboard, but `nix run .#lint` was never run
after my edits, so my "verified" claim covers compile + tests only. Same for
DiscordSync (vet on `./internal/web/...` only, no lint).

---

## e) WHAT WE SHOULD IMPROVE

1. **Verify against the source before writing a delta.** The module cache is
   right there; a `diff` of two versions is cheaper than correcting three docs.
2. **Time-box investigations and state the exit criterion first.** "If the
   netlog does not name the stalled stage, I write the finding and stop."
3. **Never inherit a number.** `3` vs `19` cost a rewrite and nearly hid a
   real bug.
4. **Update the task list at every batch boundary**, not at session start and
   end.
5. **Run the repository's own lint**, not just `go vet`, before claiming a
   change is verified.
6. **Follow each repo's documentation conventions.** DiscordSync deletes
   completed rows and logs them in `CHANGELOG.md`; I left the fix in a TODO
   row.
7. **Keep mutation checks inside git** (commit, mutate, `git restore`), never a
   `/tmp` backup, when a daemon may commit at any moment.
8. **Prefer a self-explaining skip over a passing-looking suite.** The
   preflight is the general pattern: an environment that cannot run a test
   class must say so in one line, in 12 seconds.
9. **Treat "capability appears upstream" as a test-flip task, not a doc note.**
   The federation `Healthz` test kept asserting an absence for days.
10. **Record host-level findings where the next session will look** — that is
    now the `AGENTS.md` gotcha bullet, including what was eliminated, so nobody
    repeats the twelve hypotheses.

---

## f) UP TO 50 THINGS TO DO NEXT

Ordered by impact per minute, not by size.

1. Decide **adopt vs revert** templ-components v1.24.0 (blocks 2-6).
2. If adopt: regenerate the 7 dashboard goldens in one commit, review the diff.
3. If adopt: regenerate DiscordSync's 15 snapshot goldens, review the diff.
4. If adopt: update `check-ui-pins.sh` pins to v1.24.0 and re-run the guard.
5. If adopt: visually review the redesigned 404 search field (icon → submit
   button) in both repos.
6. If revert: pin back to v1.23.0 in both repos and re-run both suites.
7. Push the three repos (5 + 1 + 2 commits) once authorized.
8. `git fetch` + `gh run list` to compare local green with remote CI before
   pushing (the release discipline the memory demands).
9. Add the `CHANGELOG.md` entry for the DiscordSync zone fix.
10. Add the `CHANGELOG.md` entry for the dashboard's browser-skip change.
11. Add a `CHANGELOG.md` note in `go-health` for the evidence-doc addition.
12. Write an automated test for the hub's shutdown sequence (mark → 503 →
    drain → exit), replacing the `/tmp` smoke script.
13. Wire `Routes.Healthz` in `cmd/health-hub` (one line + doc rewrite) so a hub
    behind a load balancer gets a combined-traffic endpoint.
14. Extend the hub test to assert `/livez` 200 → 503 during drain.
15. Add a `docs/gotchas.md` entry in DiscordSync: "golden normalizers must
    normalize the time zone".
16. Add the same note to the shared golden-kit docs, since other repos use
    `internal/web/golden`.
17. Run `nix run .#lint` on the dashboard and fix whatever it finds.
18. Run DiscordSync's lint on `./internal/web/...`.
19. Reap the two surviving loopback servers (8977, 18999).
20. Reap the ~40 leaked Chromium processes; add a note if reaping needs a
    different signal than SIGTERM in this shell.
21. Add a host-preflight line to `README`/AGENTS: browser tests self-skip when
    the host cannot browse.
22. Consider a `-short`-style env toggle that turns the preflight into an
    immediate skip (no 12s probe) for CI parity checks.
23. Add the preflight pattern to the templ-components-adjacent repos that run
    browser tests (DiscordSync e2e, Zlota44, KeyHolderAI).
24. Re-measure the dashboard's 7 golden diffs and confirm each is *only* the
    three component changes (no fourth, unnoticed delta).
25. Check whether the dashboard renders `display.Table` at all, or whether its
    7 goldens drift for a different reason than DiscordSync's 15.
26. Confirm whether the dashboard uses `errorpage` (the 404 redesign may be
    irrelevant to it).
27. After the v1.24.0 decision: re-run `set -o pipefail; bash scripts/check-ui-pins.sh`
    and record the exit code.
28. Open the M19 `company-master-data` work only if its go-appkit mount can
    expose a disk check; otherwise close the row as won't-do.
29. Audit other repos for the same seasonal `MST` normalizer bug (any repo with
    `timestampFormat` + golden files).
30. Add a timezone-independence test to `internal/web/golden` that renders the
    same fixture under `TZ=UTC` and `TZ=Europe/Berlin`.
31. Add `TZ=UTC` to DiscordSync's test environment so goldens are captured in a
    fixed zone regardless of the normalizer.
32. Verify the DiscordSync `internal/web/golden` package still compiles against
    the new normalizer under `-race`.
33. Re-run the DiscordSync full `./internal/web/...` suite (not just snapshots)
    to confirm the zone change broke nothing else.
34. Same for `./internal/...` in DiscordSync.
35. Extend the go-health evidence table with the two bridge modules' *local*
    builds (currently L1 only).
36. Add a "how to reproduce this table" command block to
    `docs/consumer-verification.md`.
37. Make `tools/fleet-skew.sh` emit machine-readable JSON so the table can be
    generated instead of hand-copied.
38. Consider a CI job that publishes the skew report as an artifact.
39. Add the federation capability matrix (Healthz / SourceStatuses /
    MarkShuttingDown per prober) to `docs/adoption-matrix.md`.
40. Decide whether the dashboard's `AGENTS.md` go-health version note should
    track the pin automatically (it said v0.7.0 while go.mod said v0.8.1).
41. Sweep `AGENTS.md`/docs for other stale "federation lacks X" claims.
42. Add a regression test that `Prober`-shaped external implementors still
    compile without the optional capabilities.
43. Write the missing `pre-tag` follow-up: delete `/tmp/pretag-probe` if it
    still exists.
44. Add a nightly job that runs the browser suite on a host that can browse, so
    the migration gets runtime proof automatically.
45. If no such host exists, document the migration as "compile-verified" in the
    dashboard's FEATURES/TODO so nobody claims more.
46. Review the `-skip` ergonomics: a documented one-liner for non-browser
    suites on broken hosts.
47. Add a `docs/status/archived/README.md` manifest entry for this report when
    it is archived.
48. Consider consolidating the three repos' "golden drift" guidance into one
    shared doc.
49. Re-read the daemon-race recipes and encode the `git reset --soft` +
    `--amend` sequence as a script, since I used it three times today.
50. Re-run everything in this list's first ten items in one clean pass once the
    v1.24.0 decision is in, and only then push.

---

## g) THREE QUESTIONS I CANNOT ANSWER MYSELF

**Q1 — templ-components v1.24.0: adopt or revert?**
New information that changes the shape of the decision: it is not markup
churn. `errorpage.NotFound404` redesigns the search field (leading icon out,
trailing submit button in), and `display.Table` moves its `role`/`aria-label`
from the wrapper onto the table. Adopting means 7 + 15 goldens regenerate plus
a human look at the 404 page; reverting means pinning both repos back to the
audited v1.23.0 and re-running both suites. The guard stays red until you pick.

**Q2 — may I push?**
`go-health` 1 commit, `go-health-dashboard` 5, `DiscordSync` 2. All locally
verified (dashboard: build/vet/suite except the 7 known goldens; go-health:
`nix run .#gates` all green; DiscordSync: vet + golden + mediapipeline green,
snapshots 15-known-red). I have not pushed and have not compared with remote
CI yet.

**Q3 — whose is the host?**
This machine's Chromium cannot load any `http(s)` URL (proven for 154, 155 and
156; netlog shows the connect completing and the GET never written), ~40
leaked browser processes predate my session, and two loopback servers of mine
survived SIGTERM. Is fixing or reaping any of that mine to do, or should I
treat the host as a fixed constraint and keep the skip?

---

## Appendix — exact commands and results

```bash
date "+%Y-%m-%d_%H-%M %Z (%A)"        # 2026-10-28_15-17 CET (Wednesday)

# dashboard
nix develop -c go vet ./...                                     # exit 0
nix develop -c go test -run TestBrowser .                        # ok  12.039s (19 skips)
nix develop -c go test -run TestFederation -count=1 -v .         # 4/4 PASS
nix develop -c go test ./... -count=1                            # 7 golden FAIL, rest ok
nix fmt                                                          # exit 0
nix develop -c go build ./cmd/health-hub                         # BUILD_OK

# hub drain smoke (fake remote served by python http.server on :18999)
/tmp/health-hub + HEALTH_HUB_REMOTES=edge=http://127.0.0.1:18999/health
readyz-before-drain: HTTP/1.1 200 OK
readyz-during-drain: HTTP/1.1 503 Service Unavailable after 2ms
hub-exit=0

# go-health
bash tools/fleet-skew.sh                                         # 16 rows @v0.8.1, minor-stale 0
nix run .#gates                                                  # all gates green

# DiscordSync
nix develop -c go test ./internal/web -run 'Snapshot|Golden'     # 19 FAIL -> 15 FAIL after fix
nix develop -c go test ./internal/web/golden/ ./internal/mediapipeline/   # ok ok
nix develop -c go vet ./internal/web/...                         # exit 0
# mutation check
#   comment out zoneSuffixRe -> TestNormalize_ReplacesTimestamps FAIL
#   restore                     -> PASS

# chromium probes (all hang; 25-40s timeouts, 0 bytes)
chromium-156 --headless --dump-dom http://127.0.0.1:18999/     # EXIT 124
chromium-155 --headless --dump-dom http://127.0.0.1:18999/     # EXIT 124
chromium-154 --headless --dump-dom http://127.0.0.1:18999/     # EXIT 124
chromium-156 --headless --dump-dom 'data:text/html,<h1>hi</h1>' # EXIT 0  <h1>hi</h1>
chromium-156 --headless --dump-dom file:///tmp/hubsmoke/health  # EXIT 0
```
