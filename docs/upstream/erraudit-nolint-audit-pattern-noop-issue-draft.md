# Upstream issue draft — erraudit: `nolint-audit` accepts a package pattern but only walks a directory

> **Status: verified, awaiting filing authorization (2026-09-23).**
> erraudit is Lars's own repo (`larsartmann/erraudit`, erraudit `1c6809a`),
> so this is an own-repo tracker item rather than an external filing;
> verify-before-filing still applied and both commands below were run in
> this repo before writing. Gate 5: no existing erraudit issue covers
> `nolint-audit` behavior (the only open one, #2, is about a suppression
> directive *namespace*, unrelated).

## Verification record (local reproduction, erraudit 1c6809a)

`nolint-audit` accepts `./...` and answers confidently with the wrong
result — the same repo, same commit, two invocations:

```console
$ erraudit nolint-audit ./...
No //nolint:erraudit directives found.

$ erraudit nolint-audit .
4 directives: 4 needed, 0 stale
```

The repo does carry four `//nolint:erraudit` directives (handlers.go,
pusher.go, status.go, webhook.go); the `./...` run silently reports zero.
`./...` is the idiomatic shape every Go tool accepts, so the failure is
invisible — you only notice when comparing against a hand count.

## Proposed issue body (own-repo, bug report register)

### Problem

`erraudit nolint-audit ./...` exits 0 with "No //nolint:erraudit
directives found." on a repository that has directives. The flag value is
treated as a filesystem path; since `./...` is not a directory the
best-effort walk matches nothing and reports the empty result as success.

```console
$ erraudit nolint-audit ./...
No //nolint:erraudit directives found.
$ erraudit nolint-audit .
4 directives: 4 needed, 0 stale
```

### Proposal

Either resolve the argument with `go/packages` like the analysis path
does (patterns become package lists), or reject non-directory arguments
with a usage error so the silent no-op becomes a loud one.

## Second verified finding for the same file: nolint-audit vs analyzer mismatch

While refuting the old "strings.Cut false positive" TODO note, a second
mismatch surfaced: `nolint-audit` classifies the directive at
`status.go:340` as NEEDED ("suppresses: ignored — Error may be ignored
using blank identifier"), but the analyzer produces **no** finding at
that site — stripping the directive and re-running `erraudit .` changes
nothing (7 violations before, 7 after; zero on that line; confirmed also
in a standalone reproduction package). So the audit's "needed" verdict
does not prove the directive suppresses anything — the mapping between
`nolint-audit`'s suppression classes and the analyzer's actual findings
is looser than the output implies. Proposal: derive "needed/stale" by
re-running the finding pipeline with and without the directive (or by
matching exact finding keys), not by classifying the suppressed line's
syntax.

(Directive removal landed in this repo alongside this draft — the
directive was suppressing nothing on erraudit 1c6809a.)

💘 Generated with Crush
