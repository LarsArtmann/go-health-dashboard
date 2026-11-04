package dashboard

import (
	"regexp"
	"strings"
)

// cspNonceValue matches a CSP nonce token: base64 or base64url characters
// after the "nonce-" prefix, per the CSP nonce-source grammar
// (1*( ALPHA / DIGIT / "+" / "/" / "-" / "_" / "=" )).
var cspNonceValue = regexp.MustCompile(`^[A-Za-z0-9+/=_-]+$`)

// cspPolicy assembles the dashboard's verified Content-Security-Policy.
// hardened swaps base-uri 'self' for 'none' and adds frame-ancestors and
// form-action 'none' for embedders that never frame the dashboard and
// accept no form posts. Both variants ship the exact directive set the
// browser test verifies.
func cspPolicy(nonce string, hardened bool) string {
	scriptSrc := "script-src 'self'"
	if cspNonceValue.MatchString(nonce) {
		scriptSrc += " 'nonce-" + nonce + "'"
	}

	scriptSrc += " 'unsafe-eval'"

	baseURI := "base-uri 'self'"
	if hardened {
		baseURI = "base-uri 'none'"
	}

	directives := []string{
		"default-src 'self'",
		scriptSrc,
		"style-src 'self'",
		"img-src 'self' data:",
		"connect-src 'self'",
		"font-src 'self'",
		"object-src 'none'",
		baseURI,
	}

	if hardened {
		// Neither directive falls back to default-src, so omitting them
		// leaves framing and form-action permissive; 'none' closes both.
		directives = append(directives, "frame-ancestors 'none'", "form-action 'none'")
	}

	return strings.Join(directives, "; ")
}

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
//
// The returned bytes are pinned by tests and by scripts/check-ui-pins.sh;
// treat any change as a dedicated, browser-verified change.
func RecommendedCSP(nonce string) string {
	return cspPolicy(nonce, false)
}

// RecommendedCSPStrict returns the hardened variant of RecommendedCSP for
// consumers that own the whole page around the dashboard and know the
// dashboard is never framed and never posts forms: base-uri tightens from
// 'self' to 'none', and frame-ancestors and form-action are pinned to
// 'none' (neither falls back to default-src, so the base policy leaves
// them permissive). go-taskqueue carried this exact triple as a local
// strip-and-reappend patch — CSP is first-occurrence-wins, so appending
// directives to RecommendedCSP's output would not work; use this variant
// instead of patching the header.
//
// The page itself runs unmodified under it: it has no forms, no frames,
// and no <base> tag, and every runtime-verified directive is identical.
// An empty or invalid nonce behaves exactly as in RecommendedCSP.
func RecommendedCSPStrict(nonce string) string {
	return cspPolicy(nonce, true)
}
