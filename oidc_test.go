package dashboard

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// stubIssuer is a minimal OIDC issuer for the middleware tests: one RSA
// key, discovery + JWKS + token + userinfo endpoints, and knobs for the
// claim shapes under test (missing email, unverified email, groups in
// token vs userinfo, a hostile second key for signature checks).
type stubIssuer struct {
	server *httptest.Server
	key    *rsa.PrivateKey
	kid    string

	email           string
	emailVerified   bool
	groups          []string
	emailInToken    bool
	groupsInToken   bool
	nonce           string
	tokenAudience   string
	tokenIssuer     string
	tokenTTL        time.Duration
	signWithForeign bool

	foreignKey *rsa.PrivateKey
}

// newStubIssuer starts the issuer on a loopback test server.
func newStubIssuer(t *testing.T) *stubIssuer {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating the issuer key: %v", err)
	}

	foreignKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating the foreign key: %v", err)
	}

	stub := &stubIssuer{
		key:           key,
		foreignKey:    foreignKey,
		kid:           "test-key-1",
		email:         "lars@example.test",
		emailVerified: true,
		groups:        []string{"pbx-health"},
		emailInToken:  true,
		groupsInToken: true,
		tokenAudience: "health",
		tokenTTL:      time.Minute,
	}

	mux := http.NewServeMux()
	mux.HandleFunc(
		"GET /.well-known/openid-configuration",
		func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, map[string]any{
				"issuer":                 stub.server.URL,
				"authorization_endpoint": stub.server.URL + "/authorize",
				"token_endpoint":         stub.server.URL + "/token",
				"userinfo_endpoint":      stub.server.URL + "/userinfo",
				"jwks_uri":               stub.server.URL + "/jwks.json",
			})
		},
	)
	mux.HandleFunc("GET /jwks.json", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"keys": []any{stub.jwk(stub.key.PublicKey)},
		})
	})
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"access_token": "stub-access-token",
			"id_token":     stub.idToken(stub.nonce),
		})
	})
	mux.HandleFunc("GET /userinfo", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"email":          stub.email,
			"email_verified": stub.emailVerified,
			"groups":         stub.groups,
		})
	})

	stub.server = httptest.NewServer(mux)
	t.Cleanup(stub.server.Close)

	return stub
}

// jwk renders a public key as a JWKS entry.
func (s *stubIssuer) jwk(pub rsa.PublicKey) map[string]any {
	return map[string]any{
		"kty": "RSA",
		"kid": s.kid,
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(bigToIntBytes(pub.E)),
	}
}

// idToken signs a full id_token for the current knobs.
func (s *stubIssuer) idToken(nonce string) string {
	claims := map[string]any{
		"iss": s.tokenIssuerURL(),
		"sub": "user-1",
		"aud": s.tokenAudience,
		"exp": time.Now().Add(s.tokenTTL).Unix(),
		"iat": time.Now().Unix(),
	}

	if nonce != "" {
		claims["nonce"] = nonce
	}

	if s.emailInToken {
		claims["email"] = s.email
		claims["email_verified"] = s.emailVerified
	}

	if s.groupsInToken {
		claims["groups"] = s.groups
	}

	header := base64.RawURLEncoding.EncodeToString(
		mustJSONForStub(map[string]any{"alg": "RS256", "kid": s.kid}),
	)
	payload := base64.RawURLEncoding.EncodeToString(mustJSONForStub(claims))

	key := s.key
	if s.signWithForeign {
		key = s.foreignKey
	}

	signed := []byte(header + "." + payload)
	digest := sha256.Sum256(signed)

	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		panic(fmt.Sprintf("signing the stub id_token: %v", err))
	}

	return header + "." + payload + "." + base64.RawURLEncoding.EncodeToString(signature)
}

// tokenIssuerURL answers the issuer URL the token should claim (the
// knob exists to test issuer mismatch).
func (s *stubIssuer) tokenIssuerURL() string {
	if s.tokenIssuer != "" {
		return s.tokenIssuer
	}

	return s.server.URL
}

// newTestMiddleware wires a middleware for the given stub with the
// default test configuration. The returned login gives in-package
// access for driving the dance.
func newTestMiddleware(
	t *testing.T,
	stub *stubIssuer,
	mutate ...func(*OIDCConfig),
) (*oidcLogin, http.Handler) {
	t.Helper()

	cfg := OIDCConfig{
		Issuer:        stub.server.URL,
		ClientID:      "health",
		ClientSecret:  "stub-client-secret",
		RedirectURL:   "https://health.example.test/oidc/callback",
		SessionSecret: []byte("0123456789abcdef0123456789abcdef"),
	}

	for _, m := range mutate {
		if m != nil {
			m(&cfg)
		}
	}

	normalized, err := normalizeOIDCConfig(cfg)
	if err != nil {
		t.Fatalf("normalizing the test config: %v", err)
	}

	login := &oidcLogin{cfg: normalized, client: stub.server.Client()}

	reached := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	_ = reached

	return login, login.middleware(next)
}

// driveLogin walks the browser dance against the stub: login redirect,
// synthesized authorize round trip, callback with the attempt cookie.
// It returns the session cookie on success.
func driveLogin(
	t *testing.T,
	login *oidcLogin,
	handler http.Handler,
	stub *stubIssuer,
) *http.Cookie {
	t.Helper()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/oidc/login", nil))

	if recorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("login answered %d, want 307: %s", recorder.Code, recorder.Body.String())
	}

	authorize, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parsing the authorize redirect: %v", err)
	}

	state := authorize.Query().Get("state")
	if state == "" {
		t.Fatal("the authorize redirect carries no state")
	}

	attemptCookie := findCookie(recorder, oidcAttemptCookieName)
	if attemptCookie == nil {
		t.Fatal("the login response sets no attempt cookie")
	}

	attempt, ok := login.unseal[oidcAttempt](attemptCookie.Value)
	if !ok {
		t.Fatal("the attempt cookie does not unseal")
	}

	// The stub mints its id_token with the attempt's nonce bound.
	stub.nonce = attempt.Nonce

	callback := httptest.NewRequest(
		http.MethodGet,
		"/oidc/callback?code=stub-code&state="+url.QueryEscape(state),
		nil,
	)
	callback.AddCookie(attemptCookie)

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, callback)

	if recorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("callback answered %d, want 307: %s", recorder.Code, recorder.Body.String())
	}

	session := findCookie(recorder, oidcSessionCookieName)
	if session == nil {
		t.Fatal("the callback sets no session cookie")
	}

	return session
}

func findCookie(recorder *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}

	return nil
}

func mustJSONForStub(value any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("encoding stub JSON: %v", err))
	}

	return encoded
}

func bigToIntBytes(value int) []byte {
	return []byte{
		byte(value >> 24), byte(value >> 16), byte(value >> 8), byte(value),
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		panic(fmt.Sprintf("encoding stub JSON: %v", err))
	}
}

// --- configuration validation -------------------------------------------

func TestOIDC_ConfigValidation(t *testing.T) {
	t.Parallel()

	valid := OIDCConfig{
		Issuer:        "https://auth.example.test",
		ClientID:      "health",
		ClientSecret:  "secret",
		RedirectURL:   "https://health.example.test/oidc/callback",
		SessionSecret: []byte("0123456789abcdef"),
	}

	cases := map[string]func(*OIDCConfig){
		"missing issuer":          func(c *OIDCConfig) { c.Issuer = "" },
		"non-absolute issuer":     func(c *OIDCConfig) { c.Issuer = "auth.example.test" },
		"trailing-slash issuer":   func(c *OIDCConfig) { c.Issuer = "https://auth.example.test/" },
		"missing client id":       func(c *OIDCConfig) { c.ClientID = "" },
		"missing client secret":   func(c *OIDCConfig) { c.ClientSecret = "" },
		"missing redirect":        func(c *OIDCConfig) { c.RedirectURL = "" },
		"relative redirect":       func(c *OIDCConfig) { c.RedirectURL = "/oidc/callback" },
		"short session secret":    func(c *OIDCConfig) { c.SessionSecret = []byte("short") },
		"public client refused":   func(c *OIDCConfig) { c.ClientSecret = "" },
		"garbage scheme issuer":   func(c *OIDCConfig) { c.Issuer = "ftp://auth.example.test" },
		"garbage scheme redirect": func(c *OIDCConfig) { c.RedirectURL = "ftp://health/oidc/callback" },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cfg := valid
			mutate(&cfg)

			if _, err := newOIDCMiddleware(cfg); err == nil {
				t.Fatalf("expected a validation error for %q", name)
			}
		})
	}

	if _, err := newOIDCMiddleware(valid); err != nil {
		t.Fatalf("the valid config must pass: %v", err)
	}
}

// --- the dance -----------------------------------------------------------

func TestOIDC_LoginRedirectShape(t *testing.T) {
	t.Parallel()

	stub := newStubIssuer(t)
	_, handler := newTestMiddleware(t, stub)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/oidc/login", nil))

	if recorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("login answered %d", recorder.Code)
	}

	location, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parsing the redirect: %v", err)
	}

	if !strings.HasPrefix(location.String(), stub.server.URL+"/authorize") {
		t.Fatalf("redirect targets %q, want the issuer authorize endpoint", location.String())
	}

	query := location.Query()
	for key, want := range map[string]string{
		"client_id":             "health",
		"redirect_uri":          "https://health.example.test/oidc/callback",
		"response_type":         "code",
		"code_challenge_method": "S256",
	} {
		if got := query.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}

	if challenge := query.Get("code_challenge"); len(challenge) != 43 {
		t.Errorf("code_challenge is %d chars, want the S256 43-char shape", len(challenge))
	}

	attempt := findCookie(recorder, oidcAttemptCookieName)
	if attempt == nil || !attempt.HttpOnly || attempt.SameSite != http.SameSiteLaxMode {
		t.Fatalf("attempt cookie shape wrong: %+v", attempt)
	}
}

func TestOIDC_FullDanceGrantsSession(t *testing.T) {
	t.Parallel()

	stub := newStubIssuer(t)
	login, handler := newTestMiddleware(t, stub)

	session := driveLogin(t, login, handler, stub)

	if !login.validSession(session.Value) {
		t.Fatal("the granted session cookie does not validate")
	}

	authed := httptest.NewRequest(http.MethodGet, "/health", nil)
	authed.AddCookie(session)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, authed)

	if recorder.Code != http.StatusOK {
		t.Fatalf("an authed request answered %d, want the wrapped 200", recorder.Code)
	}
}

func TestOIDC_UnauthenticatedBrowserRedirectsToLogin(t *testing.T) {
	t.Parallel()

	stub := newStubIssuer(t)
	_, handler := newTestMiddleware(t, stub)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	if recorder.Code != http.StatusTemporaryRedirect ||
		recorder.Header().Get("Location") != "/oidc/login" {
		t.Fatalf(
			"unauthed browser visit answered %d -> %q",
			recorder.Code,
			recorder.Header().Get("Location"),
		)
	}
}

func TestOIDC_UnauthenticatedJSONGetsClean401(t *testing.T) {
	t.Parallel()

	stub := newStubIssuer(t)
	_, handler := newTestMiddleware(t, stub)

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Accept", "application/json")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthed JSON visit answered %d, want 401", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), "unauthenticated") {
		t.Fatalf("401 body = %q", recorder.Body.String())
	}
}

// --- failure taxonomy -----------------------------------------------------

func TestOIDC_FailureTaxonomy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		code   int
		want   string
		stub   func(*stubIssuer)
		mutate func(*OIDCConfig)
	}{
		{
			name: "unverified email",
			code: http.StatusForbidden,
			want: oidcErrUnverified,
			stub: func(s *stubIssuer) { s.emailVerified = false },
		},
		{
			name: "group denied",
			code: http.StatusForbidden,
			want: oidcErrGroup,
			stub: func(s *stubIssuer) { s.groups = []string{"somebody-else"} },
			mutate: func(c *OIDCConfig) {
				c.Group = "pbx-health"
			},
		},
		{
			name: "foreign key signature",
			code: http.StatusUnauthorized,
			want: oidcErrToken,
			stub: func(s *stubIssuer) { s.signWithForeign = true },
		},
		{
			name: "issuer mismatch inside token",
			code: http.StatusUnauthorized,
			want: oidcErrToken,
			stub: func(s *stubIssuer) { s.tokenIssuer = "https://evil.example.test" },
		},
		{
			name: "audience mismatch",
			code: http.StatusUnauthorized,
			want: oidcErrToken,
			stub: func(s *stubIssuer) { s.tokenAudience = "other-client" },
		},
		{
			name: "expired id_token",
			code: http.StatusUnauthorized,
			want: oidcErrToken,
			stub: func(s *stubIssuer) { s.tokenTTL = -2 * oidcClockSkew },
		},
		{
			name: "email missing everywhere",
			code: http.StatusUnauthorized,
			want: oidcErrToken,
			stub: func(s *stubIssuer) { s.emailInToken = false; s.email = "" },
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stub := newStubIssuer(t)
			if tt.stub != nil {
				tt.stub(stub)
			}

			login, handler := newTestMiddleware(t, stub, tt.mutate)

			// Drive to the callback, then inspect the served page: every
			// failure mode must render its typed code in-place.
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/oidc/login", nil))

			attemptCookie := findCookie(recorder, oidcAttemptCookieName)
			if attemptCookie == nil {
				t.Fatal("login set no attempt cookie")
			}

			attempt, ok := login.unseal[oidcAttempt](attemptCookie.Value)
			if !ok {
				t.Fatal("attempt cookie does not unseal")
			}

			stub.nonce = attempt.Nonce

			callback := httptest.NewRequest(
				http.MethodGet,
				"/oidc/callback?code=stub-code&state="+url.QueryEscape(attempt.State),
				nil,
			)
			callback.AddCookie(attemptCookie)

			recorder = httptest.NewRecorder()
			handler.ServeHTTP(recorder, callback)

			if recorder.Code != tt.code {
				t.Fatalf(
					"callback answered %d, want %d: %s",
					recorder.Code,
					tt.code,
					recorder.Body.String(),
				)
			}

			if !strings.Contains(recorder.Body.String(), tt.want) {
				t.Fatalf("error page lacks the code %q: %s", tt.want, recorder.Body.String())
			}
		})
	}
}

func TestOIDC_StateMismatchIsStale(t *testing.T) {
	t.Parallel()

	stub := newStubIssuer(t)
	_, handler := newTestMiddleware(t, stub)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/oidc/login", nil))

	attemptCookie := findCookie(recorder, oidcAttemptCookieName)
	if attemptCookie == nil {
		t.Fatal("login set no attempt cookie")
	}

	callback := httptest.NewRequest(
		http.MethodGet,
		"/oidc/callback?code=stub-code&state=forged-state",
		nil,
	)
	callback.AddCookie(attemptCookie)

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, callback)

	if recorder.Code != http.StatusUnauthorized ||
		!strings.Contains(recorder.Body.String(), oidcErrStale) {
		t.Fatalf("forged state answered %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestOIDC_MissingAttemptCookieIsStale(t *testing.T) {
	t.Parallel()

	stub := newStubIssuer(t)
	_, handler := newTestMiddleware(t, stub)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/oidc/callback?code=x&state=y", nil),
	)

	if recorder.Code != http.StatusUnauthorized ||
		!strings.Contains(recorder.Body.String(), oidcErrStale) {
		t.Fatalf("cookieless callback answered %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestOIDC_IssuerErrorRedirectIsExchange(t *testing.T) {
	t.Parallel()

	stub := newStubIssuer(t)
	_, handler := newTestMiddleware(t, stub)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/oidc/callback?error=access_denied", nil),
	)

	if recorder.Code != http.StatusBadGateway ||
		!strings.Contains(recorder.Body.String(), oidcErrExchange) {
		t.Fatalf("issuer error answered %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestOIDC_DeadIssuerIsDiscovery(t *testing.T) {
	t.Parallel()

	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(dead.Close)

	_, handler := newTestMiddleware(t, &stubIssuer{server: dead, tokenAudience: "health"},
		func(c *OIDCConfig) { c.Issuer = dead.URL },
	)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/oidc/login", nil))

	if recorder.Code != http.StatusBadGateway ||
		!strings.Contains(recorder.Body.String(), oidcErrDiscovery) {
		t.Fatalf("dead issuer answered %d: %s", recorder.Code, recorder.Body.String())
	}
}

// --- session + logout ----------------------------------------------------

func TestOIDC_TamperedSessionCookieRejected(t *testing.T) {
	t.Parallel()

	stub := newStubIssuer(t)
	login, handler := newTestMiddleware(t, stub)

	session := driveLogin(t, login, handler, stub)

	forged := *session
	if len(forged.Value) > 0 {
		forged.Value = forged.Value[:len(forged.Value)-2] + "xx"
	}

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.AddCookie(&forged)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("a tampered session reached the handler (code %d)", recorder.Code)
	}
}

func TestOIDC_ExpiredSessionRejected(t *testing.T) {
	t.Parallel()

	stub := newStubIssuer(t)
	login, _ := newTestMiddleware(t, stub)

	expired := login.seal(
		oidcSession{
			Subject: "user-1",
			Email:   stub.email,
			Expires: time.Now().Add(-time.Minute).Unix(),
		},
	)

	if login.validSession(expired) {
		t.Fatal("an expired session validated")
	}
}

func TestOIDC_LogoutClearsOnlyTheLocalCookie(t *testing.T) {
	t.Parallel()

	stub := newStubIssuer(t)
	_, handler := newTestMiddleware(t, stub)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/oidc/logout", nil))

	cleared := findCookie(recorder, oidcSessionCookieName)
	if cleared == nil || cleared.MaxAge != -1 {
		t.Fatalf("logout did not clear the session cookie: %+v", cleared)
	}
}

// --- the userinfo fallback -------------------------------------------------

func TestOIDC_UserinfoFallbackCarriesProfileClaims(t *testing.T) {
	t.Parallel()

	stub := newStubIssuer(t)
	// Token carries NO email/groups; userinfo carries both — the Pocket
	// ID doctrine (access-token-side claims are absent).
	stub.emailInToken = false
	stub.groupsInToken = false

	login, handler := newTestMiddleware(t, stub, func(c *OIDCConfig) { c.Group = "pbx-health" })

	session := driveLogin(t, login, handler, stub)

	if !login.validSession(session.Value) {
		t.Fatal("the fallback dance never granted a session")
	}
}
