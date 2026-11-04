package dashboard_test

import (
	"strings"
	"testing"

	dashboard "github.com/larsartmann/go-health-dashboard"
)

func TestRecommendedCSP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		nonce string
		want  string
	}{
		{
			name:  "with nonce",
			nonce: "abc123",
			want: "default-src 'self'; " +
				"script-src 'self' 'nonce-abc123' 'unsafe-eval'; " +
				"style-src 'self'; " +
				"img-src 'self' data:; " +
				"connect-src 'self'; " +
				"font-src 'self'; " +
				"object-src 'none'; " +
				"base-uri 'self'",
		},
		{
			name:  "empty nonce omits nonce source",
			nonce: "",
			want: "default-src 'self'; " +
				"script-src 'self' 'unsafe-eval'; " +
				"style-src 'self'; " +
				"img-src 'self' data:; " +
				"connect-src 'self'; " +
				"font-src 'self'; " +
				"object-src 'none'; " +
				"base-uri 'self'",
		},
		{
			name:  "nonce with base64 padding and url-safe characters",
			nonce: "aB+/=_-9z",
			want: "default-src 'self'; " +
				"script-src 'self' 'nonce-aB+/=_-9z' 'unsafe-eval'; " +
				"style-src 'self'; " +
				"img-src 'self' data:; " +
				"connect-src 'self'; " +
				"font-src 'self'; " +
				"object-src 'none'; " +
				"base-uri 'self'",
		},
		{
			name:  "malicious nonce is omitted, policy stays well-formed",
			nonce: "x'; script-src *; x",
			want: "default-src 'self'; " +
				"script-src 'self' 'unsafe-eval'; " +
				"style-src 'self'; " +
				"img-src 'self' data:; " +
				"connect-src 'self'; " +
				"font-src 'self'; " +
				"object-src 'none'; " +
				"base-uri 'self'",
		},
		{
			name:  "whitespace-only nonce is omitted",
			nonce: "  ",
			want: "default-src 'self'; " +
				"script-src 'self' 'unsafe-eval'; " +
				"style-src 'self'; " +
				"img-src 'self' data:; " +
				"connect-src 'self'; " +
				"font-src 'self'; " +
				"object-src 'none'; " +
				"base-uri 'self'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := dashboard.RecommendedCSP(tt.nonce)
			if got != tt.want {
				t.Errorf("RecommendedCSP(%q) =\n%q\nwant:\n%q", tt.nonce, got, tt.want)
			}
			if strings.Contains(got, "unsafe-inline") {
				t.Errorf(
					"RecommendedCSP(%q) contains 'unsafe-inline', policy must stay strict",
					tt.nonce,
				)
			}
		})
	}
}

func TestRecommendedCSP_MatchesBrowserTestPolicy(t *testing.T) {
	t.Parallel()

	got := dashboard.RecommendedCSP("abc123")
	for _, directive := range []string{
		"default-src 'self'",
		"script-src 'self' 'nonce-abc123' 'unsafe-eval'",
		"style-src 'self'",
		"img-src 'self' data:",
		"connect-src 'self'",
		"font-src 'self'",
		"object-src 'none'",
		"base-uri 'self'",
	} {
		if !strings.Contains(got, directive) {
			t.Errorf("RecommendedCSP missing verified directive %q in %q", directive, got)
		}
	}
}

func TestRecommendedCSPStrict(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		nonce string
		want  string
	}{
		{
			name:  "with nonce",
			nonce: "abc123",
			want: "default-src 'self'; " +
				"script-src 'self' 'nonce-abc123' 'unsafe-eval'; " +
				"style-src 'self'; " +
				"img-src 'self' data:; " +
				"connect-src 'self'; " +
				"font-src 'self'; " +
				"object-src 'none'; " +
				"base-uri 'none'; " +
				"frame-ancestors 'none'; " +
				"form-action 'none'",
		},
		{
			name:  "empty nonce omits nonce source",
			nonce: "",
			want: "default-src 'self'; " +
				"script-src 'self' 'unsafe-eval'; " +
				"style-src 'self'; " +
				"img-src 'self' data:; " +
				"connect-src 'self'; " +
				"font-src 'self'; " +
				"object-src 'none'; " +
				"base-uri 'none'; " +
				"frame-ancestors 'none'; " +
				"form-action 'none'",
		},
		{
			name:  "malicious nonce is omitted, policy stays well-formed",
			nonce: "x'; script-src *; x",
			want: "default-src 'self'; " +
				"script-src 'self' 'unsafe-eval'; " +
				"style-src 'self'; " +
				"img-src 'self' data:; " +
				"connect-src 'self'; " +
				"font-src 'self'; " +
				"object-src 'none'; " +
				"base-uri 'none'; " +
				"frame-ancestors 'none'; " +
				"form-action 'none'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := dashboard.RecommendedCSPStrict(tt.nonce)
			if got != tt.want {
				t.Errorf("RecommendedCSPStrict(%q) =\n%q\nwant:\n%q", tt.nonce, got, tt.want)
			}
			if strings.Contains(got, "unsafe-inline") {
				t.Errorf(
					"RecommendedCSPStrict(%q) contains 'unsafe-inline', policy must stay strict",
					tt.nonce,
				)
			}
			if strings.Contains(got, "base-uri 'self'") {
				t.Errorf(
					"RecommendedCSPStrict(%q) ships the permissive base-uri 'self'; strict pins 'none'",
					tt.nonce,
				)
			}
		})
	}
}

func TestRecommendedCSPStrict_IsTightenedBasePolicy(t *testing.T) {
	t.Parallel()

	base := dashboard.RecommendedCSP("abc123")
	strict := dashboard.RecommendedCSPStrict("abc123")

	// Structural pin: strict is exactly the base policy with base-uri
	// tightened to 'none' plus the two non-fallback directives appended —
	// the go-taskqueue local patch, expressed in-library. Any other delta
	// between the two variants must be a deliberate, browser-verified
	// change, not an accident.
	want := strings.Replace(base, "base-uri 'self'", "base-uri 'none'", 1) +
		"; frame-ancestors 'none'; form-action 'none'"
	if strict != want {
		t.Errorf(
			"RecommendedCSPStrict drifted from the tightened-base structural pin:\n%q\nwant:\n%q",
			strict,
			want,
		)
	}
}
