# Status Report — 2026-09-22 17:39 — Health Hub Root Redirect + Fleet Federation Inventory

**Session scope:** Diagnose `https://health.home.lan/` (404) and `/health` (only CV); fix the hub in
`go-health-dashboard`; correct SystemNix docs; repair a dependency-pin sweep my own tooling caused.
Two repos touched: `go-health-dashboard` (code + tests + docs), `SystemNix` (docs only).
Self-review questions (forgot / better / improve) are answered inline across (b), (d), (e).

---

## a) FULLY DONE (verifiable, with evidence)

| #  | What                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        | Evidence                                                                                             |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
~~| A1 | Root `/` 404 root-caused and fixed: the hub never registered `/`; Caddy proxies every path, Go's mux 404s the rest. Fix: `newServeMux` (`cmd/health-hub/main.go:171`) registers exact-root `"/{$}"` → 307 → `DefaultRoutes().Dashboard`; unknown paths still 404, registered routes still win                                                                                                                                                                                               | commits `2579772` (feature — message lost to daemon, see D2) + `a61685f` (go-fix reformat)           |~~ done this session — evidence cited in the line
~~| A2 | Unit coverage for the mux wiring: `TestNewServeMux` — 4 subtests: `/`→307+`Location: /health`; `/healthz`→200; `/favicon.svg`→200; `/nope`→404                                                                                                                                                                                                                                                                                                                                              | PASS at HEAD, re-verified 17:40 (`go test ./cmd/health-hub/ -run TestNewServeMux -v`)                |~~ done this session — evidence cited in the line
~~| A3 | Live end-to-end verification: patched binary on `127.0.0.1:18999` against the REAL cv remote (`:8098`): `/`→307 `/health` ✓, `/health`→200 ✓, `/healthz`→200 ✓, `/nope`→404 ✓ — 4/4 OK                                                                                                                                                                                                                                                                                                      | probe output in session log                                                                          |~~ done this session — evidence cited in the line
~~| A4 | Fleet federation inventory (the definitive "why only CV"): live-probed 14 endpoints across the fleet — **CV `:8098` is the only go-health speaker**; inboxclean/overview/browser-history/papdashboard/manifest/geometrikks = custom JSON, renamer/discordsync/tq = HTML, llama-rag = llama.cpp shape, monitor365/crush-daily/pma = down/404. Recorded in SystemNix `docs/services/health-dashboard.md` ("Why only CV federates today") and the `[decision]` item in `docs/todo/services.md` | SystemNix commit `3123cbfb`                                                                          |~~ done this session — evidence cited in the line
~~| A5 | Dependency-pin sweep incident fully repaired: swept pins reverted (`d97232b`), `go 1.27.1` floor restored (`3f15936`), both traps documented in AGENTS.md (`51157da`)                                                                                                                                                                                                                                                                                                                       | full root suite + cmd tests green post-repair; golangci-lint + test-race green in buildflow dev gate |~~ done this session — evidence cited in the line
~~| A6 | SystemNix deployability verified after the foreign `flake.lock` sweep: `services.health-dashboard.enable` evals `true`, `remotes` evals to exactly `["cv=http://127.0.0.1:8098/health"]`                                                                                                                                                                                                                                                                                                    | `nix eval` output in session log                                                                     |~~ done this session — evidence cited in the line

## b) PARTIALLY DONE

~~1. **The fix is not live for users.** go-health-dashboard master is **6 commits ahead, unpushed** (push needs~~ closed by later sessions — the 2026-09-23 full-list sweep and the 2026-10-x chains executed or consciously closed the residue
   authorization); SystemNix input not bumped; evo-x2 still serves 404 on `/`.
   _Blocker:_ authorization + sudo deploy. _Effort:_ S.
~~2. **Pin protection is reactive, not preventive.** The trap is documented (AGENTS.md gotcha), but~~ closed by later sessions — the 2026-09-23 full-list sweep and the 2026-10-x chains executed or consciously closed the residue
   `go-mod-update` is still active in `.buildflow.yml` — the next unguarded local buildflow run can sweep again.
   _Remaining:_ one `skip_steps` entry + local pre-push pin guard. _Effort:_ S.
~~3. **Redirect verification depth.** Unit + loopback e2e done; NOT verified through the real Caddy vhost /~~ closed by later sessions — the 2026-09-23 full-list sweep and the 2026-10-x chains executed or consciously closed the residue
   TLS / oauth2 chain (needs deploy). Query-string behavior on `/` (dropped by `http.RedirectHandler`) is
   neither tested nor documented. _Effort:_ S.
~~4. **Hub documentation.** Package doc comment updated; **CHANGELOG.md entry NOT added** (forgot — the file~~ closed by later sessions — the 2026-09-23 full-list sweep and the 2026-10-x chains executed or consciously closed the residue
   exists); FEATURES.md untouched (the daemon's `a61685f` diff there was pure whitespace reflow — verified).
   _Effort:_ S.
~~5. **Inventory boundary (honesty caveat).** I probed all services whose modules reference go-health-style~~ closed by later sessions — the 2026-09-23 full-list sweep and the 2026-10-x chains executed or consciously closed the residue
   checks plus health-named ports (14 targets). Did **not** probe forgejo/miniflux/twenty/bank-sync/
   openseo/taskchampion/mr-sync/searxng (no go-health signal in their modules). "Every candidate port"
   means every _credible_ candidate — not every port in `ports.nix`. _Effort to close:_ M.

## c) NOT STARTED

- **go-health adoption by fleet services** — the actual fix for "only CV". Blocked on owner decision
  (which service first; alert semantics). This is the real work behind the user's original complaint.
- **go-health v0.3.0 re-pin** — the sweep revealed the federation tag now exists (the swept go.mod resolved
  it); AGENTS.md pre-authorizes re-pinning from the master pseudo-version once tagged. Not started.
- **Guarded evaluation of templ-components v1.19.1 / go-datastar v0.6.0 / go-sse v0.6.1** — swept versions
  were reverted, not assessed.
- **monitor365-server `:3001` connection-refused investigation**; **pma-health `:9190` serves 404 on
  `/health`** (port named `-health`, no health endpoint). Both noticed, not investigated.
- **SystemNix AGENTS.md smoke-probe rationale update** ("hub 404s /" → "redirects") — post-deploy.
- **HARVEST** of this report's section (f) into `TODO_LIST.md`/`ROADMAP.md` — post-report, per skill contract.

## d) TOTALLY FUCKED UP (radical honesty — the valuable section)

~~1. **I caused the dependency-pin sweep.** My buildflow invocation ran `go-mod-update`, which bumped~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
   templ-components 1.18.0→1.19.1, go-datastar 0.5.0→0.6.0, go-health→v0.3.0, go-sse→0.6.1; the auto-daemon
   committed it (`68ac162`) minutes later. Golden tests caught the drift only after the fact; a revert was
   required; the broken commit is permanent history. _Severity:_ an unguarded UI bump nearly landed — the
   exact failure class AGENTS.md devotes a full gotcha to. _Root cause:_ no `--exclude go-mod-update`, no
   post-run `git diff go.mod go.sum` check. _Mitigation now:_ gotcha documented + f-items 7/8/44. The daemon
   is not an excuse — it was my run.
~~2. **I violated the repo's #1 rule (commit beats the daemon).** The feature sat uncommitted through~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
   verification; the daemon committed it as `2579772` "chore: auto-commit 2 changed file(s)" — the feature's
   intent message is permanently lost. AGENTS.md documents this exact loss class from the v0.7.0 release,
   and I re-committed it anyway. _Mitigation:_ none available (history policy); rule re-internalized.
~~3. **First buildflow run executed outside the devshell** — 7 failures, 9 unavailable tools (system go 1.26.7~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
   vs the 1.27.1 floor that AGENTS.md states explicitly). A wasted full run.
~~4. **`"{$}"` ServeMux pattern panic** — wrong exact-root syntax (`"/{$}"`). Test panicked on first run;~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
   sloppy recall, should have checked before running.
~~5. **Parallel-subtest + `defer server.Close()` race** — textbook Go gotcha; server closed before parallel~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
   subtests resumed; first run failed with connection refused. Fixed with `t.Cleanup`.
~~6. **Two failed `edit` calls** ("read the file first") — I had read those files via bash `sed`/`rg` instead~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
   of the View tool. Two wasted round trips.
~~7. **Commit message typo** ("restore it.go") → amend. Sloppy heredoc assembly.~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
~~8. **Noisy eval attempts** — `--raw` on a boolean, wrong `packages.x86_64-linux.health-hub` attr path.~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
   Cosmetic, but sloppy.

## e) WHAT WE SHOULD IMPROVE

~~1. **Buildflow invocation protocol:** always `nix develop -c buildflow --build-mode dev --exclude~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
   go-mod-update --exclude nix-hash-fix --exclude nix-build-verify` locally — better: encode the excludes
   in `.buildflow.yml` `skip_steps` with rationale so nobody re-learns it (f7).
~~2. **Post-buildflow invariant:** `git diff -- go.mod go.sum` must be empty or explained BEFORE walking away —~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
   the daemon swept the mutation into a commit within ~2 minutes.
~~3. **Intent-commit timing:** commit the moment tests pass, BEFORE gate chains. Re-internalized the hard way (D2).~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
~~4. **Local UI-pin guard:** `check-ui-pins.sh` fires only in CI — after the sweep already landed locally. Wire~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
   it into `pre-push-checks.sh` / pre-buildflow (f8).
~~5. **Deliberate doc split-brain (disclosed):** dashboard AGENTS.md now says the hub redirects `/`; SystemNix~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
   AGENTS.md still says it 404s. True-vs-true until deploy; resolve in the deploy change (f5).
~~6. **Fleet health-surface fragmentation:** 9+ services expose bespoke `/health` JSON. The hub is fine — the~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
   fleet is. Consolidating on go-health is what makes "federation" actually mean something (f13-16).
~~7. **Did I lie anywhere?** One imprecision self-caught and now disclosed: "live-probed every candidate port"~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
   = every _credible_ candidate (14 targets), not every port in `ports.nix` (see B5).
~~8. **Ghost-system candidates flagged, not invented:** pma-health port with no health endpoint; monitor365 down.~~ resolved — executed in the 2026-09/10 cycles (see CHANGELOG)
   No ghost code created; nothing useful removed (the revert removed only the sweep — correctly).

## f) NEXT TASKS (45 concrete items, ranked; Impact / Effort / Category — HARVEST input)

**Ship the fix (Critical path):**

~~1. Push go-health-dashboard master (6 commits) — High / S / Chore _(blocked:push)_~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~2. Bump SystemNix flake input `go-health-dashboard` after push — High / S / Chore~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~3. Deploy evo-x2 — High / S / Deploy _(blocked:user — sudo)_~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~4. Post-deploy smoke: `https://health.home.lan/` → 307 → dashboard; `/healthz` 200 — High / S / Verification~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~5. Update SystemNix AGENTS.md smoke-probe rationale (404s → redirects) — Medium / S / Docs~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~6. Add redirect assertion to `scripts/post-deploy-check.sh` hub smoke section — Medium / S / Quality~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)

**Pin protection (prevent recurrence):**
~~7. Add `go-mod-update` to `.buildflow.yml` `skip_steps` with rationale — High / S / Config~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~8. Wire `scripts/check-ui-pins.sh` into `pre-push-checks.sh` (local, pre-CI) — High / S / Quality~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~9. Add `git diff -- go.mod go.sum` gate to `pre-push-checks.sh` — High / S / Quality~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~10. Decide: re-pin go-health to the v0.3.0 tag (AGENTS.md pre-authorized on tag; verify tag contents first) — High / S / Deps~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~11. Guarded eval of templ-components v1.19.1 (dedicated change + green browser suite) — Medium / M / Deps~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~12. Guarded eval of go-datastar v0.6.0 (retry-field semantics are pinned-by-notes — extra care) — Medium / M / Deps~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~13. go-sse v0.6.1 changelog review — Low / S / Deps~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~14. File upstream (BuildFlow): go-mod-normalize downgrading `go 1.27.1`→`1.27` — verify-before-filing first — Medium / S / Upstream~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~15. Encode the dev-gate invocation (excludes) in AGENTS.md commands table — Medium / S / Docs~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)

**Federation growth (the real answer to "only CV"):**
~~16. Owner decision: first service to adopt go-health (inboxclean / browser-history / monitor365) — High / S / Decision~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~17. Design adapter pattern: `health.NewWithHealthCheck` wrapper mapping custom `/health` JSON → go-health checks — Medium / M / Design~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~18. Implement adoption #1 + wire its remote into `services.health-dashboard.remotes` — Medium / M / Feature~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~19. Update runbook + `[decision]` item with the rollout plan — Low / S / Docs~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~20. Revisit `WithPublicMode` masking needs as remote count grows — Low / S / Quality~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~21. Close the inventory gap: probe forgejo/miniflux/twenty/bank-sync/openseo/taskchampion/mr-sync/searxng health surfaces — Low / S / Inventory~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~22. Extend go-health README federation section with the fleet-inventory finding — Low / S / Docs~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)

**Incidents noticed this session:**
~~23. Investigate monitor365-server `:3001` connection refused (down? socket-activated? retired?) — Medium / S / Investigation~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~24. Investigate pma-health `:9190` 404 on `/health` (dead port or wrong path?) — Low / S / Investigation~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~25. Establish provenance of SystemNix `flake.lock` sweep `e2f78a01` (16:30) — Medium / S / Investigation~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~26. Root-cause nix-sandbox DNS failure for `proxy.golang.org` (`[::1]:53` refused) — fixes local flake gates — Medium / M / Infra~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~27. Rogue hermes llamas decision (pre-existing; my llama probe result feeds it) — Medium / S / Decision~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)

**Hygiene from this session:**
~~28. CHANGELOG.md entry: root-redirect feature + pin-sweep incident — Medium / S / Docs~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~29. FEATURES.md row for the hub root redirect — Low / S / Docs~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~30. Test/document `/` redirect query-string behavior (currently dropped by RedirectHandler) — Low / S / Quality~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~31. Consider 301 vs 307 for `/` (bookmark/caching semantics) — Low / S / Decision~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~32. Consider serving the dashboard AT `/` via a library option (alternative to redirect) — Low / M / Decision~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~33. Consider a `Dashboard.Routes()` accessor to decouple the hub from `DefaultRoutes()` (noted in `newServeMux` comment) — Low / S / Refactor~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~34. Add `TestNewServeMux` case: POST `/` stays 307 (method-preserving) — Low / S / Quality~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~35. Check `/health/` trailing-slash behavior (mux subtree semantics) — Low / S / Quality~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~36. ADR-0003 for the root-redirect decision — Low / S / Docs~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~37. Verify Gatus "Health Hub" checks unaffected by the redirect post-deploy (they use `/healthz`) — Low / S / Verification~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~38. Verify health-hub version stamp post-deploy (ldflags rev vs flake.lock node) — Medium / S / Verification~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~39. Re-run the full browser suite after ANY future pin bump (standing guard rule) — High / M / Quality~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~40. HARVEST this report into TODO_LIST/ROADMAP — High / S / Process~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~41. ANNOTATE the 2026-09-19 hub-build status report with the redirect once deployed — Low / S / Docs~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~42. SystemNix CI darkness (`NIX_GITHUB_RO_TOKEN` missing; 120+ dark runs, pre-existing) — Medium / S / Infra~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~43. Expose `WithPushInterval` / fetch-timeout as module options (pre-existing parked rows) — Low / M / Feature~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~44. Runbook phrasing check: hub always sends `Accept: application/json` to remotes — Low / S / Docs~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)
~~45. Post-adoption: re-capture README screenshots (redirect + multi-remote dashboard) — Low / M / Docs~~ routed — swept in the 2026-09-23 full-list execution; dispositions in CHANGELOG and TODO_LIST (closed 2026-10-25)

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

**Q1 — The SystemNix `flake.lock` sweep:** commit `e2f78a01` (16:30 today) moved 74 input pairs
(nixpkgs + several fleet flakes). I checked `git log`, the diff shape, and `docs/status/` — no handoff file
covers it. Was that you, another agent session, or a scheduled updater? This decides whether the lock is
trusted for the next deploy or should be reverted first.

**Q2 — The swept upstream releases:** the sweep revealed go-health **v0.3.0** (the federation tag the
module has been waiting for), templ-components v1.19.1, go-datastar v0.6.0, go-sse v0.6.1. I reverted all of
them. Do you want them adopted via the guarded process (dedicated change + green browser suite) — and is the
v0.3.0 re-pin wanted NOW (it's pre-authorized by AGENTS.md, only the timing is yours)?

**Q3 — First go-health adoption target:** which service becomes the second federation remote —
inboxclean (largest service), browser-history, or monitor365 (currently down — revive first)? The capability
inventory is done; the priority is an owner call.

---

_Format note: skill default is a styled HTML dashboard; user explicitly requested `.md` — override honored.
Section (f) is HARVEST input for `docs-health` (TODO_LIST/ROADMAP), not entombed here._
