Three separate causes, all fixed:

- Test job on `aa38ece` (docs-only commit): FEATURES.md claimed 49 test files, the CI gate counts 50 glob files — `chromedp_eval_test.go` has zero top-level test funcs so `rg -c` match-files undercounts by one. `1175a09`.
- `TestShutdown_StopsPusherLoopGoroutine` (race job on `2ca9eca`): 4 loop goroutines still unscheduled after the full 5s poll under race + full-suite load on the 2-4 vCPU runner — same test green on `aa38ece` and in the nightly canary, so no production leak path. Settle deadline 5s→15s, failure path now dumps surviving loop stacks. `lifecycle_test.go:348`.
- `TestBrowser_CollapseInteract` (browser job on `2ca9eca`): a PushAlways patch (200ms) re-collapsed the group between click and state read — two CDP round-trips apart, and runner load lands a patch inside that window. Toggle-listener flag on `window` + the table check force-opens inside one synchronous evaluation. Mutation-checked: the old read fails an injected mid-window patch with the exact CI message, the flag read survives it. `browser_test.go:1460`.

All fixed on `1903a7a`: run 43161705684, all 8 jobs green.
