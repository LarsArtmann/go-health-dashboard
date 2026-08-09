# Status Report: Nonce Architecture Follow-up Resolution

Follows up `2026-08-09_01-47_per-request-nonce-architecture-and-httputil-v0.11.0.md`.
All critical blockers from that report are resolved except one pre-existing CQRS gap.

> **RESOLVED — go-health-dashboard v0.2.0 is the final state.** The dashboard's
> per-request nonce feature shipped at tag `v0.2.0` (`61c6718`); the CHANGELOG now
> documents it. The one CLI-unverifiable item — live browser CSP runtime check
> (final section) — is recorded as a known gap in `FEATURES.md` and a candidate
> headless-browser test in `ROADMAP.md`. The CQRS / consumer (`file-and-image-renamer`)
> items below are **OUT OF SCOPE** for go-health-dashboard.

## Done this session

### go-health-dashboard — published v0.2.0 (tag `v0.2.0`, rev `61c6718`)

- **Verified clean**: build, vet, `go test` (+ `-race`), `golangci-lint` (0 issues), `nix flake check` all pass.
- **Pushed + tagged**: `feat(dashboard)` per-request nonce (`a22ef06`), extractor tests (`022d09d`), docs (`3eaf210`), status report (`7a61017`), and new **render-cleanliness regression guards** (`61c6718`):
  - `TestRender_AllScriptsCarryNonce` — every `<script>` in the `/health` output carries the nonce.
  - `TestRender_NoInlineStyles` — zero `<style>` blocks and zero inline `style=` attributes.
- **v0.2.0 = minor bump** for the new public API: `Config.NonceExtractor` + `WithNonceExtractor(fn)`.

### file-and-image-renamer — split-brain resolved, nonce flow verified

- **Split-brain fixed**: flake input pinned to `refs/tags/v0.2.0` (was stale `master@0ad2cee`); `replace … => /local/path` dropped from `go.mod`; `require` bumped `v0.1.0 → v0.2.0`; `vendorHash` recomputed. `nix build .#file-and-image-renamer` succeeds and the binary runs.
- **Lint fix in nonce code**: `middleware.go` + `csp_test.go` switched to `httputil.DefaultNonceConfig()` + `CSPBuilder` override (resolves `exhaustruct` — `NonceConfig.Size` was omitted). Nonce files are lint-clean.
- **End-to-end nonce integration test** (`TestNonceFlow_CSPHeaderMatchesHTMLNonce`): wires the real `httputil.Nonce` → `WithNonceExtractor(httputil.NonceFromRequest)` flow, then asserts the CSP-header nonce is the SAME nonce on every inline `<script>`, and that two requests yield distinct nonces. **This is the CLI-verifiable replacement for the browser-console check flagged in three prior reports.**
- Auto-git committed the nonce work as `92e28d0` (cleanly separated from the parallel CQRS stream `5d89a29`).

## Decisions

### `style-src 'unsafe-inline'` — KEPT (deliberate, evidence-backed)

Investigated removing `'unsafe-inline'` from the consumer's global CSP. Findings:

- **templ-components** emits **zero** `<style>` blocks, zero `templ.CSS`/`CSSClass`, and no inline `style=` attributes in its library code (only in demo examples) — it's pure Tailwind utility classes.
- The **`/health` page** renders 5 nonce'd `<script>` tags, **0** `<style>` blocks, **0** inline styles (asserted by `TestRender_NoInlineStyles`).

So `/health` does **not** need `'unsafe-inline'`. However, the consumer's CSP is **global** (all routes, incl. the operations dashboard), and:

1. CSP nonces **cannot** cover inline `style="…"` attributes — only `<style>` elements and `<script>`. Removing `'unsafe-inline'` would break any inline style attribute.
2. Runtime JS (Datastar/HTMX DOM patching) may set styles via `setAttribute('style', …)`, which CSP blocks without `'unsafe-inline'`. This is unverifiable from the CLI.

**Conclusion**: keep `'unsafe-inline'` for the global CSP. A future per-route stricter CSP for `/health` alone is *possible* (the evidence supports it) but would reintroduce route-specific CSP complexity that the nonce refactor deliberately removed — not worth it. Revisit only if a security audit demands it.

### httputil `server_timing` — not adopted (unchanged)

Confirmed: it's just a header string; the dashboard already surfaces latency; the consumer doesn't use it. No dependency added.

## Remaining (NOT mine — parallel CQRS stream)

`nix flake check` fails in the **checkPhase** (vet/test of all packages) on `pkg/cqrs/`, which imports `go-cqrs-lite/{decider,event,id,…}/v4` sub-modules that are **not wired into the flake `deps` map** (`mkPreparedSource` only maps `idempotency/v4`). `nix build .#file-and-image-renamer` still works because `subPackages = ["cmd/file-renamer"]` doesn't compile `pkg/cqrs`.

Fix (for the CQRS stream owner): add each missing sub-module to the `deps` map in `flake.nix`, e.g.
```nix
"github.com/larsartmann/go-cqrs-lite/decider/v4" = "${inputs.go-cqrs-lite-src}/decider";
"github.com/larsartmann/go-cqrs-lite/event/v4"    = "${inputs.go-cqrs-lite-src}/event";
"github.com/larsartmann/go-cqrs-lite/id/v4"       = "${inputs.go-cqrs-lite-src}/id";
# …plus codec, metadata, otel, record, snapshot as imported
```
then recompute `vendorHash`. This is out of scope for the nonce work.

## Cannot verify from CLI

- **Live browser check at `https://renamer.home.lan/health`** — still requires a human to open the deployed dashboard and confirm zero CSP console errors. The integration test (`TestNonceFlow_CSPHeaderMatchesHTMLNonce`) proves the nonce wiring is correct end-to-end, which is the strongest CLI-verifiable substitute.
