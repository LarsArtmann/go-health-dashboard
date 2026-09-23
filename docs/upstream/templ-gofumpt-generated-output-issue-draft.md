# Upstream issue draft — a-h/templ: generated Go is not gofumpt-clean

> **Status: verified, awaiting filing authorization (2026-09-23).**
> Prepared per verify-before-filing: all claims reproduced locally against
> templ `v0.3.1020` (the version pinned by this repo's `go.mod` `tool`
> directive) before any prose was written. Gate 5 (duplicate search over
> a-h/templ issues, open + closed, keywords "gofumpt" / "gofmt generated")
> found nothing on generated-code formatting.

## Verification record (Gate 1-2, local reproduction)

`go tool templ generate` emits Go that `gofumpt` rewrites — so every
generator run is followed by a formatter run, and any consumer that
formats generated output cannot treat `templ generate` as producing a
stable final artifact:

```console
$ go tool templ generate . && gofumpt -l .
view_templ.go
```

Concrete diffs on real output (this repo's `view_templ.go`, ~1k lines):

```diff
 import "strings"
-import "github.com/a-h/templ"
-import templruntime "github.com/a-h/templ/runtime"
+
+	"github.com/a-h/templ"
+	templruntime "github.com/a-h/templ/runtime"
```

(import grouping: third-party imports emitted in the stdlib block) and:

```diff
-		var templ_7745c5c3_Var37 = []any{...}
+		templ_7745c5c3_Var37 := []any{...}
```

(`var x = ...` where gofumpt wants `x := ...`).

Gate 2: `templ generate --help` exposes no formatting flag, and
`templ fmt` formats `.templ` sources only, not the generated Go — there
is no built-in path to gofumpt-conformant output today.

## Proposed issue body (external bug report register)

### Problem

`templ generate` writes Go that `gofumpt` (and partly `goimports`)
rewrites, so generated output is never formatter-stable. Any repo that
enforces `gofumpt` in CI has to run `templ generate` followed by a
formatter on every regeneration, and "generated file matches what the
generator emits" checks (e.g. a drift job that re-runs the generator and
diffs) fail without that extra formatter step.

Reproduced with templ `v0.3.1020`:

```console
$ go tool templ generate . && gofumpt -l .
view_templ.go
```

Two recurring shapes: third-party imports (`github.com/a-h/templ`,
`templ/runtime`) are emitted into the stdlib import block, and
`var x = []any{...}` where gofumpt wants `x := []any{...}`.

### Proposal

Emit gofumpt-conformant Go from `templ generate` directly (import
grouping + `:=` over `var =`), or add a `--fumpt`-style flag to
`templ generate` so consumers get formatter-stable output in one step.

### Why here, not in our config

Our CI hygiene job currently regenerates and formats as one atomic step
because of this; when the step was skipped the raw output landed in git
verbatim and broke `gofumpt` gates downstream (three separate incidents
in this repo before the atomic step existed). The gap is in the
generator's output, not in how we call it.

💘 Generated with Crush
