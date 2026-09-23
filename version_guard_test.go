package dashboard

import (
	"os"
	"regexp"
	"testing"
)

// TestVersionConst_MatchesCIGrep pins the exact text shape the CI
// version-guard job's shell pipeline depends on:
//
//	grep -E 'const Version = ' dashboard.go | sed -E 's/.*"([^"]+)".*/\1/'
//
// The job greps the raw file, so a declaration-shape drift (extra quoted
// string on the line, a second matching line, a commented-out copy) makes
// CI extract garbage and compare it against the latest tag. This test
// reproduces both pipeline stages verbatim and fails at the desk with a
// precise message instead.
func TestVersionConst_MatchesCIGrep(t *testing.T) {
	t.Parallel()

	src, err := os.ReadFile("dashboard.go")
	if err != nil {
		t.Fatalf("read dashboard.go: %v", err)
	}

	grepExpr := regexp.MustCompile(`(?m).*const Version = .*`)
	lines := grepExpr.FindAllString(string(src), -1)
	if len(lines) != 1 {
		t.Fatalf(
			"the version-guard job greps 'const Version = ' and expects exactly one matching line, found %d in dashboard.go — the job would extract garbage; keep exactly one declaration and no commented copies",
			len(lines),
		)
	}

	sedExpr := regexp.MustCompile(`.*"([^"]+)".*`)
	extracted := sedExpr.ReplaceAllString(lines[0], "${1}")
	if extracted != Version {
		t.Fatalf(
			"the version-guard pipeline extracts %q from dashboard.go, but the compiled const is %q — the job would compare the wrong value against the latest git tag",
			extracted,
			Version,
		)
	}
}
