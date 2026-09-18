// Package version holds the single source of truth for the example
// server's version string. Every build self-identifies with the git
// commit it was built from, plus a "-dirty" suffix when the working tree
// had uncommitted changes:
//
//   - Builds that inject at link time (flake ldflags or release scripts)
//     override everything via the injected variable.
//   - Plain go build / go install binaries read the VCS stamp the Go
//     toolchain embeds (debug.BuildInfo vcs.revision and vcs.modified),
//     yielding "10bd9ac" or "10bd9ac-dirty". The toolchain only stamps
//     build/install binaries — go run and go test builds are unstamped
//     and fall back to "dev".
//   - Builds from sources without VCS metadata (e.g. go install from a
//     module proxy) also fall back to "dev".
package version

import (
	"runtime/debug"
)

// injected is set at link time, e.g.:
//
//	-X github.com/larsartmann/go-health-dashboard/pkg/version.injected=<shortRev>
//
// It is a separate variable because -X can only replace statically
// initialized strings; Version's dynamic initializer would clobber it.
//
//nolint:gochecknoglobals // ldflags injection requires a package-level var
var injected string

// Version is the example server version: the ldflags-injected string when
// present, otherwise the git-derived VCS stamp, otherwise "dev".
//
//nolint:gochecknoglobals // resolved once at package init; consumers read it
var Version = resolve(injected, readBuildInfo())

// resolve picks the version. Precedence: ldflags injection (nix shortRev
// or an explicit release string), then the toolchain's VCS stamp, then
// "dev".
func resolve(injected string, bi *debug.BuildInfo) string {
	if injected != "" {
		return injected
	}
	if stamp := vcsStamp(bi); stamp != "" {
		return stamp
	}
	return "dev"
}

func readBuildInfo() *debug.BuildInfo {
	bi, _ := debug.ReadBuildInfo()
	return bi
}

// vcsStamp derives "<shortrev>" or "<shortrev>-dirty" from the build's
// embedded VCS settings. It returns the empty string when the binary
// carries no VCS metadata (nix builds, go install).
func vcsStamp(bi *debug.BuildInfo) string {
	if bi == nil {
		return ""
	}

	var revision, modified string
	for _, setting := range bi.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value
		}
	}

	if revision == "" {
		return ""
	}

	short := revision
	if len(short) > shortRevLength {
		short = short[:shortRevLength]
	}

	if modified == "true" {
		return short + dirtySuffix
	}
	return short
}

const (
	// shortRevLength matches the nix shortRev width so plain and nix
	// builds report identically shaped versions.
	shortRevLength = 7
	// dirtySuffix matches the nix dirtyShortRev suffix.
	dirtySuffix = "-dirty"
)
