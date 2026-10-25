# Status: templ CLI version skew — phantom syntax error, repo restored twice, root cause still outside the repo

- **Date**: 2026-10-25 13:38 (Sunday)
- **Scope**: this session only (the `templ generate ./...` failure, its fix, and the recurrence ~1h later). Per instruction, no research into unrelated work. Neighboring reports from today (`13-20`, `12-26/12-27`, `10-45`) belong to other sessions and were not read.
- **Repo state at write time**: clean tree, both `*_templ.go` stamped v0.3.1070, `go build ./...` green. The auto-commit daemon will snapshot the latest regen within seconds.
- **Env state**: `/run/current-system/sw/bin/templ` = **v0.3.1020** (NixOS system package, outside this repo). go.mod tool pin = **v0.3.1070**.

---

## Self-review (what did I forget / do worse / can improve)

1. **What did you forget?** That `templ generate` is **not atomic**: it regenerates every file it _can_ parse, then errors on the one it can't. The user's "failed" run still rewrote `page_scripts_templ.go` with the stale v0.3.1020 generator (`updates=1` in their own output), and the daemon committed the regressed stamp. I verified "both stamps 1070" mid-session but had ended my turn; the regression landed after and was only caught on this prompt's re-inspection.
2. **What could you have done better?** Two things. (a) My headline was "Fixed" — but I fixed the _repo state_, not the _environment_. The failure boundary (stale PATH binary + editor LSP, unfixable from inside this repo) deserved the headline, not a sub-bullet. The user's re-run of the same command then predictably re-failed → "fucking wTF?!" is a fair reaction. (b) One of my stamp checks used `grep -h` (filenames suppressed) — weak verification; the mixed-stamp tree would have been obvious with `-H`.
3. **What could you still improve?** Make the sanctioned path the _lazy_ path (devShell `templ` wrapper proxying `go tool templ`) instead of fighting muscle memory with documentation, and add a stamp-vs-go.mod guard so a regressed `*_templ.go` can never be committed silently again. Both are in (f).
4. **Did you lie?** No, but "Fixed — view.templ was never broken" oversold completion. The accurate claim: "repo repaired; your `templ` binary is 50 patch versions stale and will keep failing until the NixOS package is bumped."
5. **Ghost systems?** None — no new code was written this session (docs + regenerated output only).
6. **Split brains?** One transient, now healed: the generated output existed in two version-shapes simultaneously (page_scripts @ v0.3.1020, view @ v0.3.1070) for ~1h because a daemon committed the stale generator's partial write. The guard in (f) item 2 makes this class structurally impossible.
7. **Tests?** Not run this session — build + vet only (justification: all changes were generated-comment stamps and `nix fmt` churn; weak justification given fmt touched two test files). Owed: full `nix run .#test` incl. the browser suite.

---

## a) FULLY DONE

1. **Root cause diagnosed with proof (not guesswork)** — the PATH `templ` (v0.3.1020) cannot parse syntax the go.mod-pinned tool (v0.3.1070) supports. `view.templ:168` is valid; the error is a phantom from the old parser. Evidence: reproduced on an isolated copy in `/tmp/templ-repro` (stale CLI fails identically, no repo mutation); `nix develop -c go tool templ generate` succeeds (87ms).
2. **Repo state restored (pass 1)** — all `*_templ.go` regenerated with the pinned tool; stamps moved v0.3.1020 → v0.3.1070. The daemon had previously committed stale-CLI output (the tell: old `// templ: version:` stamp inside committed output). Canonical order honored: generate → `nix fmt` (4 files normalized: browser_test.go, page_scripts_templ.go, status_test.go, view_templ.go; daemon commit `07472d2`).
3. **Gates green on restored state** — `nix run .#build` ✓, `nix run .#vet` ✓.
4. **AGENTS.md gotcha extended** — the templ tool-directive bullet now documents "miss #3": symptom signature (bogus `missing ',' in argument list` at a valid line), the stale-stamp tell, recovery recipe, and the root-fix pointer.
5. **Repo state restored (pass 2, this prompt)** — the user's re-run had regressed `page_scripts_templ.go` back to a v0.3.1020 stamp (partial write before the error; daemon-committed as `1822ef2`/`3e1293d` window). Re-ran the pinned generator: both stamps v0.3.1070, `go build ./...` green. Evidence: `grep -H 'templ: version' *_templ.go` before/after in this session's transcript.

## b) PARTIALLY DONE

1. **Root cause remediation** — nothing; `/run/current-system/sw/bin/templ` is still v0.3.1020. It lives in the NixOS system config, outside this repo; I cannot bump it from here. Until bumped, **every bare `templ generate` will (a) fail on view.templ and (b) silently regenerate the other files with stale output** — exactly what happened on the recurrence. Blocker: external config access. Effort: S once the config repo is known.
2. **Editor templ LSP** — still stale (same v0.3.1020 class): the phantom `view.templ:168` error persists in project diagnostics. No `.vscode/settings.json` repoint attempted. Effort: S–M (depends on extension, see question g3).
   ~~3. **Post-change validation** — build ✓ vet ✓, but the full test suite was never run after the fmt churn touched two test files. Effort: S (one command, ~minutes incl. browser suite in devShell).~~ satisfied — full test/lint/flake battery green on the restored tree by the 14:19 session (its report a6)

## c) NOT STARTED

1. **Stamp guard script** — assert every `*_templ.go` `// templ: version:` equals go.mod's tool pin; wire into `scripts/pre-push-checks.sh` + CI (precedent: `check-ui-pins.sh`). Priority: HIGH — today's regression was invisible to every existing gate.
2. **devShell `templ` wrapper** — shadow the stale binary inside `nix develop` with a proxy to `go tool templ`. Priority: HIGH — makes the sanctioned invocation the lazy one.
3. **NixOS config templ bump** — the actual root fix. Priority: CRITICAL, external.
   ~~4. **Full gates** — `nix run .#test`, `nix run .#lint`, `nix flake check` on the restored tree. Priority: MEDIUM.~~ satisfied — same session: nix run .#test, .#lint, nix flake check all green (14-19 a6)
4. **docs/gotchas.md miss #3** — AGENTS.md was extended; `docs/gotchas.md` ("history of the two misses") was not. Priority: LOW-MEDIUM.

## d) TOTALLY FUCKED UP

1. **Mixed-version generated output sat at HEAD for ~1h** — `page_scripts_templ.go` @ v0.3.1020 beside `view_templ.go` @ v0.3.1070, committed by the daemon. Severity: low immediate (outputs byte-identical modulo the stamp — proven by this session's first diff), high repetition risk (every bare `templ` run re-creates it). Root cause: non-atomic generate + stale PATH binary + daemon committing partial writes. Mitigation: restored twice; permanent fix = (f) 1–3. **Status: healed at write time.**
2. **My "Fixed" handoff** — I repaired the repo, said so in the headline, and left the actual failure source (user's shell + editor) untouched with no warning that re-running the same command would fail identically and _also_ regress the tree. That is a communication failure, and it manufactured today's "wTF" moment. Mitigation: this report + (f) items.
3. **No stamp guard exists at all** — three templ-related protections (go.mod tool directive, UI pins check, generate→fmt discipline) yet none notices a committed stamp/go.mod pin mismatch. Process gap, not a bug.

## e) WHAT WE SHOULD IMPROVE

1. **Warn about partial writes when a generator "fails"** — templ generate writes what it can before erroring; a failed run is still a mutating run. Check `git status` after ANY failed generate, not just successful ones. Impact: prevents exactly today's regression class.
2. **Make the correct path the lazy path** — documentation (AGENTS.md bullet) does not stop muscle-memory `templ generate`; a devShell wrapper does. Prefer environment fixes over policy fixes for recurring human actions.
3. **Verification rigor** — `grep -h` hid filenames during a stamp check; use flags that preserve identity of what you're verifying. Also: re-verify state at end-of-turn, not just mid-turn, in daemon-driven repos.

## f) Next tasks (ranked)

| #  | Task                                                                                                                                                                          | Impact   | Effort            | Category      |
| -- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ----------------- | ------------- |
| 1  | Bump templ in the NixOS config to ≥ v0.3.1070 (matches go.mod pin; fixes CLI + likely LSP)                                                                                    | Critical | S (external repo) | Bug           |
| 2  | Add `scripts/check-templ-stamps.sh`: every `*_templ.go` stamp == go.mod tool pin; wire into pre-push-checks.sh + CI Build job                                                 | High     | S                 | Quality       |
| 3  | Add devShell `templ` wrapper proxying `go tool templ` so bare `templ generate` is safe inside `nix develop`                                                                   | High     | S                 | Quality       |
| 4  | Full `nix run .#test` (devShell, incl. browser suite) on the restored tree                                                                                                    | High     | S                 | Quality       |
| 5  | Repoint/upgrade the editor templ LSP (`.vscode/settings.json` or extension update) to kill the phantom squiggle                                                               | Medium   | S–M               | Bug           |
| 6  | `nix run .#lint` + `nix flake check` on the restored tree                                                                                                                     | Medium   | S                 | Quality       |
| 7  | Extend `docs/gotchas.md` templ section with miss #3 (AGENTS.md already updated)                                                                                               | Medium   | S                 | Documentation |
| 8  | Check upstream templ for releases beyond v0.3.1070; bump pin deliberately if warranted (dedicated change: bump → generate → fmt → green browser suite, per UI-pin discipline) | Medium   | M                 | Maintenance   |
| 9  | HARVEST this (f) into TODO_LIST.md via docs-health — carefully: TODO_LIST was closed out by another session TODAY (`cb2b347`), don't resurrect stale rows                     | Medium   | S                 | Documentation |
| 10 | After the NixOS bump: re-verify editor diagnostics cleared + one bare `templ generate` round-trip, close the loop                                                             | Medium   | S                 | Bug           |
| 11 | Consider a daemon-side guard (pre-commit) for stale stamps — likely redundant once #2 lands; decide then                                                                      | Low      | S                 | Quality       |

(11 real items; padding to 50 would be brainstorm, not commitments — the honest ceiling for this session's blast radius. ROADMAP candidates: #8, #11.)

## g) Questions I cannot answer myself

1. **Where is the NixOS config that installs templ v0.3.1020** (`/run/current-system/sw/bin/templ` — repo path / channel), and do you want the bump to track go.mod's pin (v0.3.1070) or latest upstream? I cannot see or edit that config from this repo.
2. **In-repo mitigations now or after the NixOS bump?** I can implement (f)2 + (f)3 (stamp guard + devShell wrapper) immediately, but if you bump the system package first they may be redundant belt-and-suspenders — your call on how much layering you want.
3. **Is your editor's templ LSP the VS Code extension's bundled binary or the PATH one?** Determines whether a `.vscode/settings.json` repoint fixes the phantom `view.templ:168` squiggle or whether the extension itself must be updated — I can't inspect your editor's configuration from here.

---

_Written per the operator's explicit `.md`-at-path demand (standing exception to the HTML-canonical status format; consistent with every existing file in docs/status/). Report is a point-in-time snapshot: annotate, never rewrite (docs-health ANNOTATE)._
