package version

import (
	"regexp"
	"runtime/debug"
	"testing"
)

// nixShapeRe locks parity with the nix shortRev/dirtyShortRev shape so a
// stamped binary and a flake-built binary report identically.
var nixShapeRe = regexp.MustCompile(`^[0-9a-f]{7}(-dirty)?$`)

func TestResolvePrefersInjection(t *testing.T) {
	t.Parallel()

	got := resolve("v9.9.9", nil)
	if got != "v9.9.9" {
		t.Fatalf("resolve(injected, nil) = %q, want the injected string verbatim", got)
	}
}

func TestResolveFallsBackToDev(t *testing.T) {
	t.Parallel()

	got := resolve("", &debug.BuildInfo{})
	if got != "dev" {
		t.Fatalf("resolve(\"\", empty BuildInfo) = %q, want %q", got, "dev")
	}
}

func TestVCSStamp(t *testing.T) {
	t.Parallel()

	const fullRevision = "10bd9acdeadbeefdeadbeefdeadbeefdeadbeef"

	tests := []struct {
		name string
		bi   *debug.BuildInfo
		want string
	}{
		{
			name: "nil build info",
			bi:   nil,
			want: "",
		},
		{
			name: "no vcs settings (nix or go install)",
			bi:   &debug.BuildInfo{},
			want: "",
		},
		{
			name: "clean tree",
			bi: &debug.BuildInfo{Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: fullRevision},
				{Key: "vcs.modified", Value: "false"},
			}},
			want: "10bd9ac",
		},
		{
			name: "dirty tree",
			bi: &debug.BuildInfo{Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: fullRevision},
				{Key: "vcs.modified", Value: "true"},
			}},
			want: "10bd9ac-dirty",
		},
		{
			name: "unknown modified flag is not treated as dirty",
			bi: &debug.BuildInfo{Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: fullRevision},
			}},
			want: "10bd9ac",
		},
		{
			name: "revision shorter than shortRevLength stays intact",
			bi: &debug.BuildInfo{Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abc"},
				{Key: "vcs.modified", Value: "true"},
			}},
			want: "abc-dirty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := vcsStamp(tt.bi); got != tt.want {
				t.Fatalf("vcsStamp() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVersionMatchesNixShape(t *testing.T) {
	t.Parallel()

	stamp := vcsStamp(readBuildInfo())
	if stamp != "" && !nixShapeRe.MatchString(stamp) {
		t.Fatalf("vcsStamp() = %q, want nix shortRev/dirtyShortRev shape", stamp)
	}
}

func TestVersionIsNeverEmpty(t *testing.T) {
	t.Parallel()

	if Version == "" {
		t.Fatal("Version must never be empty")
	}
}
