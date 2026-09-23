# Bisectability Audit — `071c251..HEAD`

Date: 2026-09-04 19:15 CEST
Method: for every commit in `git rev-list 071c251..HEAD` (91 commits),
create a detached worktree, run `GOEXPERIMENT=jsonv2 GOWORK=off go build
./...`, record the result, remove the worktree.

## Result

**86 of 91 commits build. Five do not** — all five are auto-commit-daemon
snapshots taken mid-edit (pushed history, immutable). Tagged releases
(`v0.4.0` = `8f63d85`, `v0.5.0` = `ed650bf`) build.

| Commit    | Date       | Build error                                                                        |
| --------- | ---------- | ---------------------------------------------------------------------------------- |
| `fb9d0de` | 2026-09-03 | `status.go:264` — `health.Status` passed where `string` expected                   |
| `72783fc` | 2026-09-03 | missing `sync/atomic` import (previously known, AGENTS.md)                         |
| `61f18a3` | 2026-09-03 | `trend.go` — `time` import declared twice                                          |
| `49f4eb9` | 2026-09-04 | `dashboard.go` — `webhookNotifier` undefined; unused import in `di.go`             |
| `ed2b759` | 2026-09-04 | `wantsJSON` declared in both `dashboard.go` and `handlers.go` (mid-split snapshot) |

## Guidance

- ~~`git bisect skip` those five SHAs; the range bisects cleanly otherwise.~~ done — the five SHAs are documented in AGENTS.md (bisect-wall gotcha) for `git bisect skip`
- Root cause class: the auto-commit daemon commits whatever the working
  tree looks like when it fires. Mitigation for future sessions: run
  `go build ./...` before stepping away from a half-wired refactor.
- The audit was rerun implicitly by every later daemon commit landing on
  a green tree; `187b92f` (current HEAD at audit time) builds.

## Extension — 2026-09-23: `187b92f..HEAD` (405 commits, same method)

Re-ran mechanically with the module's CURRENT toolchain floor in mind:
go.mod moved to `go 1.27.1` on 09-19, so the earlier ambient-go method
(1.26.7 + GOTOOLCHAIN=local) now false-fails every commit after the
bump — the rerun pins the devShell's go 1.27.1 binary
(`GOTOOLCHAIN=local GOEXPERIMENT=jsonv2 GOWORK=off`).

**386 of 405 build; 19 do not** — again ALL auto-daemon mid-edit
snapshots (one is the revert of one). Two failure classes:

1. Classic torn-tree carriers: code referencing symbols/registers that
   land in a LATER commit (`8efd66c` calls `dashboard.DefaultPushInterval`
   before options.go declares it; `f208a4b` carries raw templ output
   against an old `badgeForStatus` signature; `0f2d035` uses
   `aggregate.NamedSource` before the dep bump).
2. NEW class — the 2026-09-22 normalize-downgrade scars: commits whose
   `go` directive was swept to `go 1.27` (or whose go.sum is torn) fail
   `go mod` loading entirely ("updates to go.mod needed"; `ecccf63`,
   `2ff040d`). The check-go-directive guard exists since 09-22 because
   of exactly this class; it caught the live repeat on 09-23.

| Commit    | Date       | Class      | Build error                                                        |
| --------- | ---------- | ---------- | ------------------------------------------------------------------ |
| `0f2d035` | 2026-09-05 | carrier    | `aggregate.NamedSource` undefined (example ahead of dep bump)      |
| `fbf5e37` | 2026-09-09 | carrier    | mid-edit snapshot                                                  |
| `c2a8b25` | 2026-09-09 | carrier    | mid-edit snapshot                                                  |
| `597c108` | 2026-09-10 | carrier    | mid-edit snapshot                                                  |
| `f208a4b` | 2026-09-17 | carrier    | raw templ output vs old `badgeForStatus` signature                 |
| `ac4f846` | 2026-09-19 | carrier    | mid-edit snapshot                                                  |
| `a61685f` | 2026-09-22 | carrier    | 16-file sweep snapshot                                             |
| `d97232b` | 2026-09-22 | revert     | revert of a torn carrier (reverts are not self-consistent)         |
| `d6ced4c` | 2026-09-22 | carrier    | mid-edit snapshot                                                  |
| `492a799` | 2026-09-22 | carrier    | mid-edit snapshot                                                  |
| `2ff040d` | 2026-09-22 | go.mod tear| "updates to go.mod needed" (torn directive/sum state)              |
| `0676e0c` | 2026-09-22 | carrier    | mid-edit snapshot                                                  |
| `bfaa5c5` | 2026-09-22 | carrier    | mid-edit snapshot                                                  |
| `3fb0960` | 2026-09-22 | carrier    | mid-edit snapshot                                                  |
| `efe0ab0` | 2026-09-22 | carrier    | mid-edit snapshot                                                  |
| `7b6de61` | 2026-09-22 | carrier    | mid-edit snapshot                                                  |
| `d6b0f7a` | 2026-09-22 | carrier    | mid-edit snapshot                                                  |
| `ecccf63` | 2026-09-23 | go.mod tear| directive swept to `go 1.27`; CI's directive guard failed on it    |
| `8efd66c` | 2026-09-23 | carrier    | `dashboard.DefaultPushInterval` used before it was declared        |

Guidance unchanged: `git bisect skip` these 19; everything else in the
range bisects. The root cause and mitigation are the same as the
original audit — and the 09-22/09-23 concentration says the
commit-at-first-green discipline slipped during release weeks; the
per-batch commit rule in the release checklist is the fix.
