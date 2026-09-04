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

- `git bisect skip` those five SHAs; the range bisects cleanly otherwise.
- Root cause class: the auto-commit daemon commits whatever the working
  tree looks like when it fires. Mitigation for future sessions: run
  `go build ./...` before stepping away from a half-wired refactor.
- The audit was rerun implicitly by every later daemon commit landing on
  a green tree; `187b92f` (current HEAD at audit time) builds.
