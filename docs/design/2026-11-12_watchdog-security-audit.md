# Master-red-alert security audit + hardening (M30, 2026-11-12)

The pager must not have its own blind spot. This is the workflow_run/fork
security audit of `.github/workflows/master-red-alert.yml`, the self-watch
check, and the watched-list dedupe (plan T23/M30).

## workflow_run / fork security (F105)

| Surface       | Finding                                                                                                                | Verdict                                                                                                              |
| ------------- | ---------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| Trigger       | `workflow_run` on CI completed, `branches: [master]`                                                                   | SAFE — fork PR runs never carry the master branch ref, so fork activity cannot trigger the watchdog                  |
| Permissions   | `actions: read` + `issues: write`                                                                                      | LEAST-PRIVILEGE — exactly what `gh run list` + issue create need; no contents/pull-requests                          |
| Checkout      | Pinned SHA (`v7.0.1`), `persist-credentials: false`                                                                    | SAFE — no credential persists into the script step                                                                   |
| Script source | `scripts/check-master-green.sh` checked out from the master tip                                                        | INHERENT — the workflow's event checks out the default branch, so the executed guard is always master-tested content |
| Injection     | Watched names enter `gh run list --workflow "$wf"` quoted; issue bodies interpolate API results via jq, not shell eval | SAFE — no unquoted expansion; names are repo-controlled content                                                      |
| GITHUB_TOKEN  | Ephemeral, repo-scoped                                                                                                 | SAFE                                                                                                                 |

## Self-watch (F106)

"Master Red Alert" is IN `.github/watched-workflows.txt`: a red of the
watchdog itself is caught by the next nightly sweep (the green-recency
backstop) and by CI-completion triggers. Verified — the drift guard's
master/scheduled detection includes it (5 watched, 5 master/scheduled,
consistent).

## Watched-list dedupe (F107/F108)

`scripts/check-watched-workflows.sh` gained two failure modes, both
mutation-checked:

1. **Duplicate watched entries** — the list is a set; a dup means someone
   appended without reading. Appending "CI" twice trips
   `lists "CI" 3 times — the watched list is a set`.
2. **Name collision across files** — `declare -A real_names` previously
   silently overwrote a second file declaring the same workflow name; a
   shadow "CI" workflow was invisible to both guard views. Now a hard
   error (`two workflow files declare the name "CI" …`), verified by
   dropping a shadow copy next to `ci.yml`.
