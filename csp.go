package dashboard

import (
	"regexp"
	"strings"
)

// cspNonceValue matches a CSP nonce token: base64 or base64url characters
// after the "nonce-" prefix, per the CSP nonce-source grammar
// (1*( ALPHA / DIGIT / "+" / "/" / "-" / "_" / "=" )).
var cspNonceValue = regexp.MustCompile(`^[A-Za-z0-9+/=_-]+$`)

// RecommendedCSP returns a Content-Security-Policy header value that the
// /health dashboard page is verified to work under, suitable for
//
//	w.Header().Set("Content-Security-Policy", dashboard.RecommendedCSP(nonce))
//
// The policy is strict: everything self-hosted, no 'unsafe-inline' for
// scripts or styles. 'unsafe-eval' is required because the Datastar SDK
// compiles its data-* expressions with the Function constructor — without
// it the bundle throws "GenerateExpression" during init and the SSE
// connection never opens (verified by the headless-browser test). Pass the
// same nonce you gave the Dashboard via WithNonce or WithNonceExtractor so
// the page's inline scripts are allowed to execute. An empty or invalid
// nonce (anything outside the CSP base64 alphabet) is omitted from the
// policy rather than producing a malformed header.
func RecommendedCSP(nonce string) string {
	scriptSrc := "script-src 'self'"
	if cspNonceValue.MatchString(nonce) {
		scriptSrc += " 'nonce-" + nonce + "'"
	}

	scriptSrc += " 'unsafe-eval'"

	directives := []string{
		"default-src 'self'",
		scriptSrc,
		"style-src 'self'",
		"img-src 'self' data:",
		"connect-src 'self'",
		"font-src 'self'",
		"object-src 'none'",
		"base-uri 'self'",
	}

	return strings.Join(directives, "; ")
}
