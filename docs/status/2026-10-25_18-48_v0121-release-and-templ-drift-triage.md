# Status — v0.12.1 release cut (green-CI re-cut of v0.12.0) + templ-drift triage (2026-10-25, 18:48)

|                 |                                                                                                                                                                                                                                                                                                 |
| --------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Session**     | User: "Time for a new release?" → cut authorized. Mid-session the parallel dedup session ended and this became the only agent ("Now you are the only agent"), inheriting its in-flight state                                                                                                    |
| **Window**      | ~17:20–18:48 CET (single session; auto-commit daemon active throughout)                                                                                                                                                                                                                         |
| **Start state** | Master `67b0df3` (TODO_LIST rebuild) CI-green; daemon then swept a parallel session's test-fixture refactor + RAW templ output; CI Hygiene red on `5025001`                                                                                                                                     |
| **End state**   | **v0.12.1 shipped end-to-end**: tag `ef1bf57` (signed) on origin; CI 7/7 green on the release commit; verify-release.sh 6/6 (proxy hash == tag commit, sumdb, clean consumer prints `0.12.1`); pages live (v0.12.1 Latest, v0.12.0 disclosure); final pre-push gate 9/9; master `e7cb45f` green |

---

## §0 Brutal self-review

### What went wrong (and was caught)

1. **I amended the wrong commit.** Racing the daemon, I ran `git commit --amend`
   to relabel `7a2505c` (my templ fmt restore) — but the daemon had already
   committed twice more; the amend relabeled a NEWER commit holding the
   departed session's status doc, putting my message on the wrong content and
   diverging the branch (ahead 1, behind 1). Recovery: tree-identity proof
   (`git diff ad0b0f9 b451207` empty), `git reset origin/master` (mixed),
   zero loss. The intent message for the fmt restore is lost to a pushed
   heuristic commit — accepted, disclosed here.
2. **The templ fmt drift is a cycle, and I fed it twice.** Committed shape =
   fmt output, but EVERY flake app (`build`, `test-race`, …) starts with
   `go tool templ generate`, which re-raws the files mid-run; the daemon then
   sweeps the raw shape into a commit and CI's Hygiene job goes red. I
   restored the shape twice (`7a2505c` pushed red-window; `01a5f32` held) —
   the cycle only ended because the battery finished and no generate ran
   before the release commit. Root causes stay open (13-38 session): system
   templ skew feeds the LSP that re-raws; the structural fix is making the
   pinned generator's output byte-match the committed shape, not human
   discipline.
3. **Gate 0 ran only PRE-edit.** The release checklist's own step order has
   gate 0 "before any edits" — so the changelog structural lint validated the
   UNEDITED file and v0.12.0's tag sailed to a red CI (no standing
   `[Unreleased]` + missing footer link). v0.10.1 fixed the identical class
   and I still repeated it. Fixed in the discipline bullet: gate 0 again
   AFTER the release edits, BEFORE tagging.
4. **The release-draft workflow has never worked.** Both tags' draft runs went
   "success" in ~9s and produced nothing: `gh api --jq '.draft'` masks the
   404 (jq succeeds on the error JSON, prints null) → `state="null"` matches
   no case branch → silent exit 0. Diagnosed from the run logs (zero output
   lines after the case dispatch); pages created manually per §6's old-way
   fallback; TODO row + AGENTS bullet filed.

### Honesty checklist

- **Did I lie?** No; every green claim reproduces: `verify-release.sh
  v0.12.1` exit 0, CI `gh run view` 7/7 success, gate 0 exit 0 twice.
- **Ghost systems?** None. The v0.12.0 tag is disclosed, not hidden.
- **Reverted others' work?** No — the departed session's fixture refactor,
  vendorHash fix, and status doc were preserved (verified compiling+green
  before building on them); the one wrong-message commit was repaired, not
  rewritten away.

## a) FULLY DONE

1. **v0.12.1 released**: signed annotated tag on `ef1bf57`; CHANGELOG
   `[0.12.0]` section + theme blurb + `[0.12.1]` re-cut section + all footer
   links; Version const `0.12.1` in the tag commit (version-guard green);
   SECURITY rows 0.12.x; FEATURES Released row prepended.
2. **Verification chain green**: §4 battery (build, vet, test-race, lint
   0 issues, vulncheck clean, browser 21 PASS / 0 SKIP / 0 FAIL, nix fmt,
   flake check — all rc=0); gate 0 9/9 pre-edit AND post-release; tag-first
   push; CI 7/7 on the release commit; verify-release.sh 6/6.
3. **GitHub pages**: v0.12.1 (Latest, full notes) + v0.12.0 (disclosure
   page, `--latest=false`, known-exception banner per the v0.8.0 style).
4. **Master restored to green** after three drift commits (`74b4a78`,
   `01a5f32`, plus the re-cut) and one history repair.
5. **Docs synced**: TODO_LIST release row DONE + new workflow-bug TODO row;
   AGENTS release-discipline items (5) and (6); ROADMAP cadence question
   marked DECIDED: CUT.

## b) PARTIALLY DONE

1. **pkg.go.dev spot-check**: proxy + sumdb prove v0.12.1 is served
   (verifier checks 2–3), but pkg.go.dev's UI still rendered v0.11.0 at
   18:40 — normal minutes-scale lag, single re-poll pending. Not a release
   blocker (consumers resolve via the proxy).
2. **Templ drift root cause**: symptom triaged and disclosed everywhere;
   the structural fix (generator emits the committed shape / NixOS templ
   bump) remains the 13-38 session's open row.

## c) NOT STARTED

1. **release-draft.yml fix** — TODO row filed with the exact mechanism
   (`gh api --jq -e` or state assertion); exercise it on the next tag.
2. **Harvest of the departed dedup session's §f** into TODO_LIST — that
   session deliberately withheld it for an explicit instruction; still
   withheld.

## d) Evidence

- Tag: `git tag -v v0.12.1` (Good signature, object ef1bf57).
- CI: run 40217514253, 7/7 success on `ef1bf57`.
- Verifier: `bash scripts/verify-release.sh v0.12.1` → 6/6 OK, exit 0.
- Gates: `/tmp/release-gates.log` (8 steps rc=0), `/tmp/gate0-final.log`
  (9/9, SECURITY row 0.12.x ↔ tag v0.12.1).
- Proxy: `@v/list` gained v0.12.1; `Origin.Hash` == `ef1bf57…`;
  consumer run printed `0.12.1`.

## e) Decisions

- **v0.12.1 re-cut over tag deletion** (v0.10.1 precedent): pushed tags are
  never moved; the red v0.12.0 stays, disclosed, with consumers pointed at
  v0.12.1. Proxy had NOT indexed v0.12.0 (checked via `@v/list` before
  deciding) — zero consumer exposure either way.
- **Manual release pages** over blocking on the broken draft workflow.
- **chromedp v0.21 + CV rollout stay out** of the release (independent
  decisions, not release content).

## f) Next tasks (unowned)

1. Fix + exercise release-draft.yml (TODO row, mechanism documented).
2. pkg.go.dev Latest-banner confirmation for v0.12.1 (bounded poll).
3. The standing chromedp / CV / templ-toolchain rows (unchanged priority).

## g) Handoff

Master is green at `e7cb45f`, tree clean, all gates green, v0.12.1 is
Latest everywhere that matters. Next session can start normal work after
the standard handshake; nothing is mid-flight except the pkg.go.dev UI
lag self-resolving.
