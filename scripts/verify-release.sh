#!/usr/bin/env bash
# Post-push release verification — the mechanical tail of docs/release-checklist.md §6.
#
# Usage: bash scripts/verify-release.sh <version>    # v0.7.0 or 0.7.0
#
# Verifies, in order:
#   1. the tag exists locally AND on origin, pointing at the same commit
#   2. the Go proxy serves the version and its Origin.Hash equals the tag
#      commit (this is how "proxy hash == tag commit" is checked; learned
#      empirically from the v0.7.0 cut — proxy.golang.org .info documents
#      the origin VCS hash and ref it indexed)
#   3. the download passes the checksum database (GOSUMDB, default on) in
#      a throwaway module cache, so a stale local cache cannot mask drift
#   4. a clean consumer directory can go get + compile + run the published
#      module and prints the expected dashboard.Version (GOEXPERIMENT=jsonv2,
#      GOWORK=off — same contract as the flake devShell)
#   5. the GitHub Release exists and is not a draft
#   6. CI on the release commit finished green (all runs for that SHA)
#   7. when HEAD is the release commit: the Version const matches the tag
#      (the version-guard contract, checked locally)
#
# Exits non-zero on the first red check. Network + gh required.
set -euo pipefail

if [ $# -ne 1 ]; then
	echo "usage: $0 <version>   (e.g. v0.7.0 or 0.7.0)"
	exit 2
fi

TAG="v${1#v}"
MODULE=$(go list -m)
fail=0
step() {
	echo
	echo "== $1"
}
ok() { echo "   OK  $1"; }
bad() {
	echo "   FAIL $1"
	fail=1
}

TMP=$(mktemp -d)
# Go marks downloaded module-cache files read-only; restore write bits before rm.
trap 'chmod -R u+w "$TMP" 2>/dev/null; rm -rf "$TMP"' EXIT

step "Module: $MODULE  Tag: $TAG"

# 1. Local + remote tag consistency.
step "1. Tag exists locally and on origin"
if local_commit=$(git rev-parse -q --verify "$TAG^{commit}" 2>/dev/null); then
	ok "local $TAG -> $local_commit"
else
	bad "tag $TAG not found locally"
	local_commit=""
fi

if [ -n "$local_commit" ]; then
	remote_peeled=$(git ls-remote origin "refs/tags/$TAG" "refs/tags/$TAG^{}" | awk '{print $1}' | sort -u | tail -1)
	if [ "$remote_peeled" = "$local_commit" ]; then
		ok "origin $TAG -> $remote_peeled"
	else
		bad "origin $TAG resolves to '$remote_peeled', expected $local_commit (push the tag first: git push origin $TAG)"
	fi
fi

# 2+3. Proxy serves the version; Origin.Hash matches the tag commit; sumdb verified.
# go mod download verifies against the checksum database by default (GOSUMDB),
# so a passing download here IS the sumdb check.
step "2. Proxy serves $TAG; Origin.Hash == tag commit; sumdb verified (fresh cache)"
if [ -n "$local_commit" ]; then
	dl_json=$(GOWORK=off GOFLAGS= GOMODCACHE="$TMP/gomodcache" go mod download -json "$MODULE@$TAG")
	proxy_info=$(echo "$dl_json" | sed -n 's/^[[:space:]]*"Info": "\(.*\)",/\1/p')
	proxy_hash=$(grep -o '"Hash":"[^"]*"' "$proxy_info" | head -1 | cut -d'"' -f4)
	proxy_ref=$(grep -o '"Ref":"[^"]*"' "$proxy_info" | head -1 | cut -d'"' -f4)
	if [ "$proxy_hash" = "$local_commit" ]; then
		ok "proxy Origin.Hash $proxy_hash == tag commit"
	else
		bad "proxy Origin.Hash '$proxy_hash' != tag commit '$local_commit' — the proxy indexed a different commit; never move a pushed tag, cut a new version"
	fi
	if [ "$proxy_ref" = "refs/tags/$TAG" ]; then
		ok "proxy Origin.Ref $proxy_ref"
	else
		bad "proxy Origin.Ref '$proxy_ref' != refs/tags/$TAG"
	fi
	content_hash=$(echo "$dl_json" | sed -n 's/^[[:space:]]*"Sum": "\(.*\)",/\1/p')
	ok "sumdb-verified download; content hash ${content_hash:-<none>}"
else
	echo "   skipped (no local tag)"
fi

# 4. Clean-dir consumer: get + compile + run, expect the version string.
step "3. Clean consumer: go get + build + run prints ${TAG#v}"
if [ -n "$local_commit" ]; then
	consumer="$TMP/consumer"
	mkdir -p "$consumer"
	(
		cd "$consumer"
		export GOWORK=off GOFLAGS= GOEXPERIMENT=jsonv2 GOMODCACHE="$TMP/gomodcache2"
		go mod init consumer >/dev/null 2>&1
		printf 'package main\n\nimport (\n\t"fmt"\n\n\t"github.com/larsartmann/go-health-dashboard"\n)\n\nfunc main() { fmt.Println(dashboard.Version) }\n' >main.go
		go mod tidy >/dev/null
		got=$(go run .)
		if [ "$got" = "${TAG#v}" ]; then
			ok "consumer run printed $got"
		else
			bad "consumer run printed '$got', expected '${TAG#v}'"
		fi
	) || bad "consumer go get/tidy/build/run failed (see output above)"
else
	echo "   skipped (no local tag)"
fi

# 5. GitHub Release state.
step "4. GitHub Release $TAG published (not draft)"
if release_json=$(gh release view "$TAG" --json name,isDraft,isPrerelease 2>/dev/null); then
	is_draft=$(echo "$release_json" | sed -n 's/.*"isDraft": *\([a-z]*\).*/\1/p')
	is_pre=$(echo "$release_json" | sed -n 's/.*"isPrerelease": *\([a-z]*\).*/\1/p')
	if [ "$is_draft" = "false" ]; then
		ok "release exists (prerelease: $is_pre)"
	else
		bad "release $TAG is a draft — publish it"
	fi
else
	bad "no GitHub Release for $TAG — create it from the CHANGELOG section (backfill convention)"
fi

# 6. CI green on the release commit.
step "5. CI green on release commit"
if [ -n "$local_commit" ]; then
	if ! conclusions=$(gh run list --commit "$local_commit" --json conclusion --limit 50 2>/dev/null); then
		conclusions=""
	fi
	if [ -z "$conclusions" ]; then
		bad "could not list CI runs for $local_commit (gh auth? network?)"
	else
		runs=$(echo "$conclusions" | grep -o '"conclusion":"[^"]*"' | cut -d'"' -f4)
		if [ -z "$runs" ]; then
			bad "no CI runs recorded for $local_commit"
		elif echo "$runs" | grep -qvE '^(success|skipped|neutral)$'; then
			echo "$runs" | sed 's/^/       run: /'
			bad "CI on $local_commit has non-green runs (expected success/skipped/neutral only)"
		else
			ok "all CI runs green ($(echo "$runs" | wc -l | tr -d ' ') runs)"
		fi
	fi
else
	echo "   skipped (no local tag)"
fi

# 7. Version const matches the tag — only meaningful when HEAD is the release commit.
step "6. Version const vs tag (HEAD-on-tag check)"
head_commit=$(git rev-parse HEAD)
if [ "$head_commit" = "$local_commit" ]; then
	const_version=$(grep -E 'const Version = ' dashboard.go | sed -E 's/.*"([^"]+)".*/\1/')
	if [ "$const_version" = "${TAG#v}" ]; then
		ok "Version const $const_version == ${TAG#v} (HEAD is the release commit)"
	else
		bad "Version const ($const_version) != ${TAG#v} although HEAD is the release commit"
	fi
else
	echo "   skipped (HEAD is not $TAG — const check only applies on the release commit)"
fi

echo
if [ "$fail" -ne 0 ]; then
	echo "verify-release: FAILED for $TAG"
	exit 1
fi
echo "verify-release: all green for $TAG"
