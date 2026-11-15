package dashboard

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
)

// The stateless cookie machinery: attempt and session values are
// JSON payloads sealed as base64url(payload).hex(HMAC-SHA256(secret,
// payload)) — integrity by MAC, no server-side store, rotation of the
// secret invalidates everything at once.

// oidcAttempt is one in-flight login: the OAuth state, the OIDC nonce,
// the PKCE verifier, and the expiry. Sealed into the attempt cookie.
type oidcAttempt struct {
	State    string `json:"state"`
	Nonce    string `json:"nonce"`
	Verifier string `json:"verifier"`
	Expires  int64  `json:"exp"`
}

// oidcSession is the signed-in identity: subject, email, expiry.
// Sealed into the session cookie.
type oidcSession struct {
	Subject string `json:"sub"`
	Email   string `json:"email"`
	Expires int64  `json:"exp"`
}

// seal MACs a cookie payload with the session secret.
func (o *oidcLogin) seal(payload any) string {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return ""
	}

	body := base64.RawURLEncoding.EncodeToString(encoded)

	return body + "." + o.mac(body)
}

// unseal verifies the MAC and decodes the payload. A bad MAC, bad
// base64, or bad JSON all answer ok=false — cookie values are
// attacker-controlled input, no failure deserves a distinct error.
func (o *oidcLogin) unseal[T any](value string) (T, bool) {
	var zero T

	body, mac, found := cutLastDot(value)
	if !found || !hmac.Equal([]byte(mac), []byte(o.mac(body))) {
		return zero, false
	}

	decoded, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return zero, false
	}

	var payload T
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return zero, false
	}

	return payload, true
}

// validSession reopens and expiry-checks a session cookie value.
func (o *oidcLogin) validSession(value string) bool {
	session, ok := o.unseal[oidcSession](value)

	return ok && session.Expires > o.clock().Unix()
}

// mac returns the hex HMAC-SHA256 of body under the session secret.
func (o *oidcLogin) mac(body string) string {
	mac := hmac.New(sha256.New, o.cfg.SessionSecret)
	mac.Write([]byte(body))

	return hex.EncodeToString(mac.Sum(nil))
}

func cutLastDot(value string) (string, string, bool) {
	for i := len(value) - 1; i >= 0; i-- {
		if value[i] == '.' {
			return value[:i], value[i+1:], true
		}
	}

	return "", "", false
}

// expiredCookie builds a same-name cookie that clears the browser's
// copy.
func (o *oidcLogin) expiredCookie(name string) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   o.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
}

// oidcRandomToken returns a fresh 32-byte base64url token (state,
// nonce, or PKCE verifier — 43 chars, exactly the RFC 7636 verifier
// length).
func oidcRandomToken() (string, error) {
	buf := make([]byte, oidcRandomBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("reading %d random bytes: %w", oidcRandomBytes, err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// oidcS256Challenge derives the PKCE code_challenge from a verifier.
func oidcS256Challenge(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))

	return base64.RawURLEncoding.EncodeToString(digest[:])
}

// oidcErrorStatus maps a typed failure code onto the HTTP status the
// error page answers — auth rejections stay in the 4xx family, issuer
// trouble reads as bad gateway.
func oidcErrorStatus(code string) int {
	switch code {
	case oidcErrDiscovery, oidcErrExchange:
		return http.StatusBadGateway
	case oidcErrConfig:
		return http.StatusInternalServerError
	default:
		return http.StatusUnauthorized
	}
}

// serveError renders the self-contained login-failure page. In-place on
// purpose: redirecting to the dashboard would bounce straight back
// into the login route (the redirect-loop hazard), and the page carries
// the typed code verbatim so monitoring matches on it.
func (o *oidcLogin) serveError(w http.ResponseWriter, code, detail string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(oidcErrorStatus(code))

	fmt.Fprintf(
		w,
		`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Login failed</title></head>
<body style="font-family: system-ui, sans-serif; max-width: 40rem; margin: 4rem auto; padding: 0 1rem;">
<h1>Login failed</h1>
<p>Code: <code>%s</code></p>
<p>%s</p>
<p><a href="%s">Try again</a></p>
</body>
</html>
`,
		html.EscapeString(code),
		html.EscapeString(detail),
		html.EscapeString(o.cfg.LoginPath),
	)
}
