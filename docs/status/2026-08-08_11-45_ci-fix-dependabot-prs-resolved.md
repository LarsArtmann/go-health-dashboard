# Status Report: CI Fix — Dependabot PRs Resolved, Lint Green

**Date:** 2026-08-08 11:45
**Session scope:** Fix the 3 open Dependabot PRs on go-health-dashboard, get CI green.
**Repo:** go-health-dashboard (primary), templ-components (observed), go-health (observed)

---

## (a) FULLY DONE

### Dependabot PRs resolved (all 3)

| PR  | Title                                          | Resolution                                         |
| --- | ---------------------------------------------- | -------------------------------------------------- |
| #1  | bump actions/setup-go from 5 to 7              | Applied in `101ea3d`, PR auto-closed by Dependabot |
| #2  | bump actions/checkout from 4 to 7              | Applied in `101ea3d`, PR auto-closed by Dependabot |
| #3  | bump golangci/golangci-lint-action from 6 to 9 | Applied in `101ea3d`, PR auto-closed by Dependabot |

All three modified the same file (`.github/workflows/ci.yml`), so I combined them into a single commit rather than merging sequentially.

### Dashboard CI: ALL GREEN

Latest run (`b9eacb8`): **4/4 jobs passing**

| Job                | Status | Notes                               |
| ------------------ | ------ | ----------------------------------- |
| Build              | ✅     |                                     |
| Test               | ✅     | Race detector enabled               |
| Lint               | ✅     | golangci-lint v2.12.2 via action v9 |
| Vulnerability Scan | ✅     | govulncheck clean                   |

### Root cause identified and fixed

**The critical failure:** `golangci-lint-action@v6` installs golangci-lint v1.64.8 (built with Go 1.24). This binary cannot load configs targeting Go 1.26.5. Error: `can't load config: the Go language version (go1.24) used to build golangci-lint is lower than the targeted Go version (1.26.5)`. This was failing every Lint job for 5+ commits before this session.

**The fix:** Bump to `golangci-lint-action@v9`, which installs golangci-lint v2.12.2 (built with a recent Go). This matches the local Nix-provided version.

### Secondary fix: duplicate `run` argument

The v9 action automatically prepends the `run` subcommand. Passing `args: run ./...` produced `golangci-lint run run ./...`, which failed with a phantom directory error (`stat ./run: directory not found`). Fixed by changing `args: run ./...` → `args: ./...` in commit `b9eacb8`.

### Commits this session

| Commit    | Description                                      |
| --------- | ------------------------------------------------ |
| `101ea3d` | ci: bump GitHub Actions to latest major versions |
| `b9eacb8` | ci: drop duplicate 'run' from golangci-lint args |

---

## (b) PARTIALLY DONE

Nothing was partially done this session. Every task was completed to green CI.

---

## (c) NOT STARTED

These are items from the previous session's self-review that I did NOT touch this session (scope was CI fix only):

1. **TODO_LIST.md cleanup** — Still marks "Remove replace directives" and "Tag v0.1.0" as BLOCKED, but both are DONE (v0.1.0 tagged, zero replace directives in go.mod). Needs updating.
2. **Verify pkg.go.dev indexing** — v0.1.0 tag pushed but never confirmed pkg.go.dev shows the module.
3. **templ-components CI fix** — Still RED (see section d below).
4. **go-health CI** — No CI workflow runs observed (repo may not have CI configured, or it only runs on PRs).
5. **templ-components Dependabot alerts** — 4 vulnerabilities (2 high, 2 moderate) uninvestigated.

---

## (d) TOTALLY FUCKED UP

### 1. Wasted a CI round trip on the duplicate `run` bug

**What happened:** I read the v9 release notes in the Dependabot PR body. They clearly document major version changes. But I did not read the v8 release notes which state `Requires golangci-lint version >= v2.1.0` and the v9 changes about Node runtime. More importantly, I did not check whether the action's invocation interface changed between v6 and v9.

**What I should have done:** After editing ci.yml, I should have noticed that the `args: run ./...` field already had a `run` subcommand in it. The v6 action ran `golangci-lint run run ./...` too — but v6's golangci-lint v1.x was more lenient. v9's v2.x is stricter. I could have caught this by simply reading the args line critically: "if the action is called `golangci-lint-action` and its core job is to run the linter, why am I also passing `run` as an arg?"

**Impact:** One wasted CI round trip (~90 seconds). Not catastrophic, but unnecessary.

### 2. Did NOT fix templ-components CI (which I broke last session)

**What happened:** Last session I committed the CSP nonce fix to templ-components and tagged v1.8.0. But the CI on that repo has been RED for 8+ consecutive pushes. The failure is: `./cmd/tc/_sources/display/popover_templ.go is not tracked. Generated files MUST be committed.` — 30+ untracked `*_templ.go` files under `cmd/tc/_sources/`.

**Why this matters:** I released templ-components v1.8.0 with RED CI. The tag points at a broken state. Anyone consuming v1.8.0 is using code from a repo where CI doesn't pass.

**What I should have done:** Either fixed this before tagging v1.8.0, or at minimum flagged it prominently in the release notes. I did neither. I focused only on go-health-dashboard's CI this session because that's where the PRs were, but templ-components is the actual dependency and its CI is still broken.

### 3. Did NOT update TODO_LIST.md

The TODO_LIST still has two items marked BLOCKED that are actually DONE:

- "Remove replace directives" — BLOCKED → actually DONE (go.mod has zero replaces)
- "Tag v0.1.0 in git" — BLOCKED → actually DONE (tag v0.1.0 exists, pushed, GitHub Release created)

This is a docs drift I introduced by not cleaning up after the release work.

---

## (e) WHAT WE SHOULD IMPROVE

### Process improvements

1. **Read action docs before bumping major versions** — When Dependabot bumps a GitHub Action major version, read the release notes for breaking changes before merging. The `run` duplication would have been caught by understanding the v8/v9 interface.

2. **Check ALL repos in the dependency chain, not just the one with PRs** — I only looked at go-health-dashboard CI because the PRs were there. templ-components CI is still broken and I didn't even mention it until writing this report.

3. **Close the loop on docs** — Every completed task should immediately update TODO_LIST.md. Leaving stale BLOCKED items undermines trust in the tracking system.

4. **Never tag a release on a repo with RED CI** — The templ-components v1.8.0 tag is on a commit where CI fails. This is the #1 lesson from the previous session's self-review, and it's still unaddressed.

5. **CI should match local tooling** — Local has golangci-lint v2.12.2 (Nix). CI had v1.64.8 (action v6). This mismatch was the root cause of months of failures. Now fixed, but the principle stands: pin CI tool versions to match local.

### Technical improvements

6. **Pin golangci-lint version explicitly** — Currently `version: latest` in ci.yml. This means the version drifts over time. Should pin to a specific version (e.g., `v2.12.2`) for reproducibility.

7. **templ-components needs `cmd/tc/_sources/` committed or gitignored** — The CI guard catches untracked `*_templ.go` files, which is good. But the fix (committing them or adding to `.gitignore`) has not been done.

---

## (f) Up to 50 things to do next

### Critical (block release integrity)

1. Fix templ-components CI: commit the 30+ untracked `*_templ.go` files under `cmd/tc/_sources/` or add them to `.gitignore`
2. Verify templ-components CI goes green after fix
3. Consider re-tagging templ-components v1.8.0 (or v1.8.1) on a green-CI commit
4. Update dashboard go.mod if templ-components is re-tagged
5. Verify pkg.go.dev indexes go-health-dashboard@v0.1.0

### High impact (docs + tracking)

6. Update TODO_LIST.md: mark "Remove replace directives" as DONE
7. Update TODO_LIST.md: mark "Tag v0.1.0 in git" as DONE
8. Update TODO_LIST.md: remove the third BLOCKED item if resolved, or clarify current blocker
9. Add test for `SubscriberCount()` (TODO, 15min)
10. Add test for `WithHeartbeatInterval` (TODO, 15min)
11. Pin golangci-lint version in ci.yml instead of `latest`
12. Add `GOTOOLCHAIN: local` to CI env (v9 action sets this, but should be explicit)

### Medium impact (release polish)

13. Add screenshot/GIF to README for visual preview
14. Verify the `v0.1.0` GitHub Release notes are accurate and complete
15. Investigate whether v0.1.0 should have been v0.0.1 (user originally asked for v0.0.1)
16. Consider adding a `CODEOWNERS` file
17. Add Dependabot config to group minor updates (avoid 3 separate PRs for same file)
18. Investigate templ-components Dependabot alerts (2 high, 2 moderate)
19. Check if go-health repo has CI workflows at all
20. Add CI status badge to README pointing to the now-green pipeline
21. Document the `GOEXPERIMENT=jsonv2` requirement in CI comments
22. Review whether `args: ./...` is correct or should be just the dashboard package
23. Consider adding a `make ci-local` or `nix run .#ci` that mirrors CI steps locally

### Low impact / future

24. SSE reconnection support (Last-Event-ID header)
25. Embeddable dashboard mode (mount under sub-path)
26. Auth middleware integration
27. Prometheus metrics endpoint
28. Health history / sparkline visualization
29. UI flexibility options (WithHideStatCards, etc.)
30. Fuzzing for Accept header parsing
31. Fuzzing for health response serialization
32. Build-tag gating for SSE (so consumers who only want HTML don't need GOEXPERIMENT=jsonv2)
33. Add integration test that exercises the full SSE lifecycle end-to-end
34. Consider adding a `nix run .#ci` that runs the same steps as GitHub Actions
35. Review whether the `templ generate ./...` step in CI matches the local `nix run .#generate` behavior
36. Document the CI pipeline in AGENTS.md (what jobs run, what they check)
37. Consider adding a pre-commit hook that runs `golangci-lint` locally
38. Review the `.golangci.yml` config for any CI-specific vs local-specific differences
39. Add a test that verifies zero replace directives in go.mod (prevent accidental reintroduction)
40. Consider automating the release process (tag + GitHub Release + CHANGELOG) via a workflow

### templ-components repo (sibling)

41. Fix the `cmd/tc/_sources/` untracked files issue
42. Fix the "CSS Freshness" and "Visual Regression" CI jobs that fail at setup
43. Investigate the breadcrumbs template regeneration that seems to be causing drift
44. Review whether `cmd/tc/_sources/` should exist at all (is it a demo artifact?)
45. Re-run visual tests after golden regeneration to confirm they pass

### go-health repo (sibling)

46. Verify CI exists and is green (no push-triggered CI runs observed)
47. Add CI if missing
48. Verify v0.0.2 tag is on a green commit

### Cross-repo

49. Verify the entire dependency chain (go-health v0.0.2 → templ-components v1.8.0 → dashboard v0.1.0) resolves from the Go proxy without local replaces
50. Document the release dependency chain in a RELEASE.md or AGENTS.md section

---

## (g) Questions I cannot answer myself

### 1. Should I fix templ-components CI now, or is that out of scope for this session?

templ-components CI is RED (8+ consecutive failures). I released v1.8.0 from that broken state last session. The fix is known (commit the untracked `*_templ.go` files). But that's a different repo, and you pointed me specifically at the dashboard PRs. Should I cross-repo-fix it now, or do you want to handle it separately?

### 2. Should the v0.1.0 tag stay, or should we re-release?

Last session I steered you from v0.0.1 (your request) to v0.1.0 (my recommendation). The tag is pushed, GitHub Release exists, CI is now green. Releasing as v0.0.1 would mean deleting the tag (irreversible-ish — tag is public). Do you want to keep v0.1.0, or re-cut as v0.0.1?

### 3. Should I pin `golangci-lint` version in CI, or leave it at `latest`?

Currently `version: latest` resolves to v2.12.2 today but will drift. Pinning gives reproducibility but creates maintenance burden (manual bumps). Your other repos use `latest` — should I keep consistency, or pin for safety?
