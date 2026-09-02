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
				t.Errorf("RecommendedCSP(%q) contains 'unsafe-inline', policy must stay strict", tt.nonce)
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
