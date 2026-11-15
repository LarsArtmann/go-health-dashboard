package dashboard

import (
	"context"
	"crypto/hmac"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Native OpenID Connect login for deployments that cannot sit behind a
// reverse-proxy auth chain (WithOIDC). The middleware owns the whole
// browser dance against any OIDC-compliant issuer: authorization code
// + PKCE S256, id_token verification via the issuer's JWKS, a userinfo
// fallback for issuers whose tokens carry no profile claims, and a
// stateless HMAC-signed session cookie — zero new runtime
// dependencies, everything from the standard library.
//
// Routes the middleware answers (mounted inside the wrapped handler
// space, so WithBasePath prefixes them like every dashboard route):
//
//	GET <LoginPath>     start the dance (default /oidc/login)
//	GET <CallbackPath>  finish it (default /oidc/callback)
//	GET <LogoutPath>    clear the LOCAL session cookie only (default
//	                    /oidc/logout) — the issuer session is never
//	                    ended, a shared-IdP deployment keeps its other
//	                    logins
//
// Kubernetes probe endpoints are not wrapped by WithMiddleware at all,
// so they stay reachable without a session — a dead dashboard is
// observable by design. Unauthenticated browser navigations redirect
// to the login route; requests negotiating JSON (wantsJSON) get a
// clean 401 document instead, so scrapers and health fetchers never
// chase a redirect into the issuer.

const (
	// oidcSessionCookieName carries the HMAC-signed session (subject,
	// email, expiry). HttpOnly + SameSite=Lax: top-level redirects from
	// the issuer callback ride, cross-site embeds do not.
	oidcSessionCookieName = "health_oidc_session"

	// oidcAttemptCookieName carries one in-flight login attempt (state,
	// nonce, PKCE verifier, expiry) — the only login state; there is no
	// server-side store.
	oidcAttemptCookieName = "health_oidc_attempt"

	// oidcDefaultSessionTTL bounds a session.
	oidcDefaultSessionTTL = 12 * time.Hour

	// oidcAttemptTTL bounds one login attempt. Ten minutes covers a
	// slow human at the issuer without leaving replayable state lying
	// around.
	oidcAttemptTTL = 10 * time.Minute

	// oidcDiscoveryTTL bounds the cached discovery document. Endpoints
	// move rarely; a restart re-fetches anyway.
	oidcDiscoveryTTL = 10 * time.Minute

	// oidcJWKSTTL bounds the cached JWKS; a kid miss forces one refetch
	// before failing (issuer key rotation).
	oidcJWKSTTL = 10 * time.Minute

	// oidcHTTPTimeout bounds every call to the issuer (discovery, JWKS,
	// token exchange, userinfo). An issuer outage costs a login, never
	// the process.
	oidcHTTPTimeout = 10 * time.Second

	// oidcClockSkew is the leeway on id_token expiry checks.
	oidcClockSkew = 2 * time.Minute

	// oidcMinSessionSecretBytes is the floor on the HMAC key: below it
	// WithOIDC refuses to construct — a guessable key is no gate.
	oidcMinSessionSecretBytes = 16

	// oidcMaxIssuerBody bounds every issuer response read (discovery,
	// JWKS, token, userinfo) — a hostile or broken issuer cannot
	// exhaust memory.
	oidcMaxIssuerBody = 1 << 20

	// oidcRandomBytes is the entropy per state/nonce/verifier value.
	oidcRandomBytes = 32
)

// Typed login-failure codes, rendered on the error page and stable for
// monitoring (the sso_error family other dashboards in this fleet use).
const (
	oidcErrDiscovery  = "discovery"  // issuer unreachable / bad discovery document
	oidcErrStale      = "stale"      // attempt cookie missing, expired, or state mismatch
	oidcErrExchange   = "exchange"   // token endpoint refused the code
	oidcErrToken      = "token"      // id_token failed verification or claims
	oidcErrUnverified = "unverified" // email claim present but not verified
	oidcErrGroup      = "group"      // configured group not granted
	oidcErrConfig     = "config"     // configuration or issuer self-inconsistency
)

// OIDCConfig configures the native OIDC login (WithOIDC). Issuer,
// ClientID, ClientSecret, RedirectURL, and SessionSecret are required;
// everything else has a default. The client is CONFIDENTIAL (a
// server-rendered page exchanges the code server-side); PKCE S256
// rides every attempt anyway.
type OIDCConfig struct {
	// Issuer is the OIDC issuer base URL (e.g. https://auth.example.com,
	// no trailing slash, no /.well-known suffix). Must match the
	// discovery document's issuer field byte-for-byte.
	Issuer string

	// ClientID and ClientSecret identify the confidential client
	// registered at the issuer.
	ClientID     string
	ClientSecret string

	// RedirectURL is the absolute callback URL registered at the issuer
	// (scheme + host + CallbackPath).
	RedirectURL string

	// Group optionally names a group the identity must carry (id_token
	// groups claim, userinfo fallback). Empty disables the gate — then
	// the issuer-side client restriction is the only membership check.
	Group string

	// SessionSecret HMACs the attempt and session cookies. At least 16
	// bytes; rotating it invalidates every session and attempt.
	SessionSecret []byte

	// SessionTTL bounds a session (default 12h). Non-positive selects
	// the default.
	SessionTTL time.Duration

	// Scopes requested at the authorize endpoint. Nil selects
	// "openid email profile groups" — the groups claim is what the
	// Group gate reads.
	Scopes []string

	// CookieSecure stamps the Secure attribute on both cookies. Default
	// (zero = true; set false explicitly only for plain-HTTP local
	// development) matches a TLS-served dashboard.
	CookieSecure bool

	// LoginPath, CallbackPath, LogoutPath place the OIDC routes.
	// Defaults: /oidc/login, /oidc/callback, /oidc/logout.
	LoginPath    string
	CallbackPath string
	LogoutPath   string

	// httpClient replaces the issuer-calling client (tests). Nil
	// selects a client with the oidcHTTPTimeout timeout.
	httpClient *http.Client

	// nowFunc replaces the clock (tests). Nil selects time.Now.
	nowFunc func() time.Time
}

// WithOIDC enables the native OIDC login: every dashboard-owned handler
// (HTML, SSE, favicon, metrics, trend, export) requires a valid session
// cookie; the middleware serves the login/callback/logout routes and a
// self-contained error page. Kubernetes probe endpoints stay public
// (WithMiddleware never wraps them). Configuration is validated up
// front — a malformed OIDC setup fails New loudly, not the first login.
func WithOIDC(cfg OIDCConfig) Option {
	return func(c *Config) {
		mw, err := newOIDCMiddleware(cfg)
		if err != nil {
			panic(fmt.Sprintf("dashboard.WithOIDC: %v", err))
		}

		c.Middleware = mw
	}
}

// newOIDCMiddleware validates the configuration and returns the wired
// middleware. Every failure is an option-time error so a broken setup
// never reaches first traffic.
func newOIDCMiddleware(cfg OIDCConfig) (func(http.Handler) http.Handler, error) {
	cfg, err := normalizeOIDCConfig(cfg)
	if err != nil {
		return nil, err
	}

	oidc := &oidcLogin{cfg: cfg}

	return oidc.middleware, nil
}

// normalizeOIDCConfig fills defaults and validates the required fields.
func normalizeOIDCConfig(cfg OIDCConfig) (OIDCConfig, error) {
	issuer, err := url.Parse(cfg.Issuer)
	if cfg.Issuer == "" || err != nil || issuer.Host == "" || (issuer.Scheme != "http" && issuer.Scheme != "https") {
		return OIDCConfig{}, fmt.Errorf("issuer must be an absolute http(s) URL, got %q", cfg.Issuer)
	}

	if strings.HasSuffix(cfg.Issuer, "/") {
		return OIDCConfig{}, errors.New("issuer must not carry a trailing slash")
	}

	if cfg.ClientID == "" {
		return OIDCConfig{}, errors.New("client id is required")
	}

	if cfg.ClientSecret == "" {
		return OIDCConfig{}, errors.New("client secret is required (the native login is a confidential client)")
	}

	redirect, err := url.Parse(cfg.RedirectURL)
	if cfg.RedirectURL == "" || err != nil || redirect.Host == "" || (redirect.Scheme != "http" && redirect.Scheme != "https") {
		return OIDCConfig{}, fmt.Errorf("redirect URL must be absolute http(s), got %q", cfg.RedirectURL)
	}

	if len(cfg.SessionSecret) < oidcMinSessionSecretBytes {
		return OIDCConfig{}, fmt.Errorf(
			"session secret must be at least %d bytes, got %d",
			oidcMinSessionSecretBytes,
			len(cfg.SessionSecret),
		)
	}

	cfg = withOIDCDefaults(cfg)

	return cfg, nil
}

// withOIDCDefaults fills the defaulted fields of an already-validated
// configuration.
func withOIDCDefaults(cfg OIDCConfig) OIDCConfig {
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = oidcDefaultSessionTTL
	}

	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{"openid", "email", "profile", "groups"}
	}

	if cfg.LoginPath == "" {
		cfg.LoginPath = "/oidc/login"
	}

	if cfg.CallbackPath == "" {
		cfg.CallbackPath = "/oidc/callback"
	}

	if cfg.LogoutPath == "" {
		cfg.LogoutPath = "/oidc/logout"
	}

	return cfg
}

// oidcLogin holds the middleware state: configuration, caches for the
// issuer's discovery document and JWKS, and the injectable clock and
// HTTP client.
type oidcLogin struct {
	cfg     OIDCConfig
	client  *http.Client
	nowFunc func() time.Time
	discoMu sync.Mutex
	disco   *oidcDiscovery
	discoAt time.Time
	jwksMu  sync.Mutex
	jwks    map[string]*rsa.PublicKey
	jwksURI string
	jwksAt  time.Time
}

type oidcDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

// middleware is the WithMiddleware entry: OIDC-owned paths answer
// directly, everything else requires a session.
func (o *oidcLogin) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case o.cfg.LoginPath:
			o.handleLogin(w, r)
		case o.cfg.CallbackPath:
			o.handleCallback(w, r)
		case o.cfg.LogoutPath:
			o.handleLogout(w, r)
		default:
			o.requireSession(w, r, next)
		}
	})
}

// requireSession passes valid sessions through; JSON-negotiating
// requests get a 401 document, browser navigations redirect to the
// login route.
func (o *oidcLogin) requireSession(w http.ResponseWriter, r *http.Request, next http.Handler) {
	if cookie, err := r.Cookie(oidcSessionCookieName); err == nil && o.validSession(cookie.Value) {
		next.ServeHTTP(w, r)

		return
	}

	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthenticated"}` + "\n"))

		return
	}

	http.Redirect(w, r, o.cfg.LoginPath, http.StatusTemporaryRedirect)
}

// handleLogout clears the LOCAL session cookie only. The issuer's
// session is deliberately untouched: a shared-IdP deployment keeps its
// other logins (ending the IdP session on one dashboard's sign-out
// silently signs out every sibling surface).
func (o *oidcLogin) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, o.expiredCookie(oidcSessionCookieName))
	http.Redirect(w, r, o.cfg.LoginPath, http.StatusTemporaryRedirect)
}

// handleLogin starts the dance: fresh state/nonce/verifier, S256
// challenge, attempt cookie, redirect into the issuer's authorize
// endpoint.
func (o *oidcLogin) handleLogin(w http.ResponseWriter, r *http.Request) {
	disco, errCode := o.discovery(r.Context())
	if errCode != "" {
		o.serveError(w, oidcErrDiscovery, "the issuer's discovery document is unreachable or malformed")

		return
	}

	state, err := oidcRandomToken()
	if err != nil {
		o.serveError(w, oidcErrConfig, "generating login state failed")

		return
	}

	nonce, err := oidcRandomToken()
	if err != nil {
		o.serveError(w, oidcErrConfig, "generating login nonce failed")

		return
	}

	verifier, err := oidcRandomToken()
	if err != nil {
		o.serveError(w, oidcErrConfig, "generating the PKCE verifier failed")

		return
	}

	authorize, err := url.Parse(disco.AuthorizationEndpoint)
	if err != nil || authorize.Host == "" {
		o.serveError(w, oidcErrConfig, "the discovery document's authorization endpoint is not an absolute URL")

		return
	}

	query := authorize.Query()
	query.Set("client_id", o.cfg.ClientID)
	query.Set("redirect_uri", o.cfg.RedirectURL)
	query.Set("response_type", "code")
	query.Set("scope", strings.Join(o.cfg.Scopes, " "))
	query.Set("state", state)
	query.Set("nonce", nonce)
	query.Set("code_challenge", oidcS256Challenge(verifier))
	query.Set("code_challenge_method", "S256")
	authorize.RawQuery = query.Encode()

	http.SetCookie(w, &http.Cookie{
		Name:     oidcAttemptCookieName,
		Value:    o.seal(oidcAttempt{State: state, Nonce: nonce, Verifier: verifier, Expires: o.clock().Add(oidcAttemptTTL).Unix()}),
		Path:     "/",
		MaxAge:   int(oidcAttemptTTL.Seconds()),
		HttpOnly: true,
		Secure:   o.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, authorize.String(), http.StatusTemporaryRedirect)
}

// handleCallback finishes the dance: state check, code exchange,
// id_token verification (JWKS), claim extraction with the userinfo
// fallback, the group gate, and the session cookie.
func (o *oidcLogin) handleCallback(w http.ResponseWriter, r *http.Request) {
	if issuerErr := r.URL.Query().Get("error"); issuerErr != "" {
		o.serveError(w, oidcErrExchange, "the issuer refused the login: "+issuerErr)

		return
	}

	attempt, ok := o.attemptFromRequest(w, r)
	if !ok {
		return // attemptFromRequest already served the error page
	}

	if state := r.URL.Query().Get("state"); state == "" || !hmac.Equal([]byte(state), []byte(attempt.State)) {
		o.serveError(w, oidcErrStale, "state mismatch — the callback does not belong to your login attempt")

		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		o.serveError(w, oidcErrExchange, "the issuer redirected back without a code")

		return
	}

	claims, ok := o.claimsFromIssuer(w, r.Context(), code, attempt.Nonce, attempt.Verifier)
	if !ok {
		return // claimsFromIssuer already served the error page
	}

	if !o.grantSession(w, claims) {
		return // grantSession already served the error page
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// attemptFromRequest reopens and validates the attempt cookie. The
// stale error page is served in-place on every failure.
func (o *oidcLogin) attemptFromRequest(w http.ResponseWriter, r *http.Request) (oidcAttempt, bool) {
	cookie, err := r.Cookie(oidcAttemptCookieName)
	if err != nil {
		o.serveError(w, oidcErrStale, "no login attempt in flight (cookie missing) — start again")

		return oidcAttempt{}, false
	}

	attempt, ok := o.unseal[oidcAttempt](cookie.Value)
	if !ok || attempt.Expires <= o.clock().Unix() {
		o.serveError(w, oidcErrStale, "the login attempt expired — start again")

		return oidcAttempt{}, false
	}

	return attempt, true
}

// claimsFromIssuer exchanges the code, verifies the id_token, and
// applies the userinfo fallback for missing profile claims. The PKCE
// verifier rides the exchange (confidential client: one factor in the
// Basic header, one in the form body).
func (o *oidcLogin) claimsFromIssuer(w http.ResponseWriter, ctx context.Context, code, nonce, verifier string) (oidcClaims, bool) {
	tokens, errCode := o.exchange(ctx, code, verifier)
	if errCode != "" {
		o.serveError(w, errCode, "exchanging the authorization code failed")

		return oidcClaims{}, false
	}

	claims, err := o.verifyIDToken(ctx, tokens.IDToken, nonce)
	if err != nil {
		o.serveError(w, oidcErrToken, "the id_token failed verification: "+err.Error())

		return oidcClaims{}, false
	}

	if claims.Email == "" || (o.cfg.Group != "" && claims.Groups == nil) {
		fallback, err := o.userinfo(ctx, tokens.AccessToken)
		if err != nil {
			o.serveError(w, oidcErrToken, "the id_token lacks profile claims and the userinfo fallback failed: "+err.Error())

			return oidcClaims{}, false
		}

		claims.merge(fallback)
	}

	return claims, true
}

// grantSession enforces the email/group gates and sets the session
// cookie. The matching error page is served in-place on failure.
func (o *oidcLogin) grantSession(w http.ResponseWriter, claims oidcClaims) bool {
	if claims.Email == "" {
		o.serveError(w, oidcErrToken, "no email claim on the identity")

		return false
	}

	if !claims.EmailVerified {
		o.serveError(w, oidcErrUnverified, "the issuer reports this email as not verified")

		return false
	}

	if o.cfg.Group != "" && !claims.hasGroup(o.cfg.Group) {
		o.serveError(w, oidcErrGroup, "your account is not in the required group "+o.cfg.Group)

		return false
	}

	http.SetCookie(w, o.expiredCookie(oidcAttemptCookieName))
	http.SetCookie(w, &http.Cookie{
		Name:     oidcSessionCookieName,
		Value:    o.seal(oidcSession{Subject: claims.Subject, Email: claims.Email, Expires: o.clock().Add(o.cfg.SessionTTL).Unix()}),
		Path:     "/",
		MaxAge:   int(o.cfg.SessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   o.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	return true
}

// exchange posts the code + PKCE verifier to the token endpoint with
// HTTP Basic client authentication: the confidential secret rides the
// header, the PKCE proof rides the form body.
func (o *oidcLogin) exchange(ctx context.Context, code, verifier string) (oidcTokens, string) {
	disco, errCode := o.discovery(ctx)
	if errCode != "" {
		return oidcTokens{}, oidcErrDiscovery
	}

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {o.cfg.RedirectURL},
		"code_verifier": {verifier},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, disco.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return oidcTokens{}, oidcErrConfig
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(o.cfg.ClientID, o.cfg.ClientSecret)

	resp, err := o.httpClient().Do(req)
	if err != nil {
		return oidcTokens{}, oidcErrExchange
	}

	defer resp.Body.Close() //nolint:errcheck // read-side close after the bounded read

	body, err := io.ReadAll(io.LimitReader(resp.Body, oidcMaxIssuerBody))
	if err != nil || resp.StatusCode != http.StatusOK {
		return oidcTokens{}, oidcErrExchange
	}

	var tokens oidcTokens
	if err := json.Unmarshal(body, &tokens); err != nil || tokens.IDToken == "" {
		return oidcTokens{}, oidcErrExchange
	}

	return tokens, ""
}

type oidcTokens struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
}

// discovery fetches and caches the issuer's discovery document.
func (o *oidcLogin) discovery(ctx context.Context) (*oidcDiscovery, string) {
	o.discoMu.Lock()
	defer o.discoMu.Unlock()

	if o.disco != nil && o.clock().Sub(o.discoAt) < oidcDiscoveryTTL {
		return o.disco, ""
	}

	body, err := o.fetchJSON(ctx, o.cfg.Issuer+"/.well-known/openid-configuration")
	if err != nil {
		return nil, oidcErrDiscovery
	}

	var disco oidcDiscovery
	if err := json.Unmarshal(body, &disco); err != nil {
		return nil, oidcErrDiscovery
	}

	if disco.Issuer != o.cfg.Issuer {
		return nil, oidcErrConfig
	}

	if disco.AuthorizationEndpoint == "" || disco.TokenEndpoint == "" || disco.JWKSURI == "" {
		return nil, oidcErrDiscovery
	}

	o.disco = &disco
	o.discoAt = o.clock()

	return o.disco, ""
}

// jwksFor resolves the RSA public key for a key id, refetching the JWKS
// once on a miss (rotation).
func (o *oidcLogin) jwksFor(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	o.jwksMu.Lock()
	defer o.jwksMu.Unlock()

	if o.jwks == nil || o.clock().Sub(o.jwksAt) >= oidcJWKSTTL || o.jwks[kid] == nil {
		if err := o.fetchJWKS(ctx); err != nil {
			return nil, err
		}
	}

	key := o.jwks[kid]
	if key == nil {
		return nil, fmt.Errorf("no RSA key for kid %q in the issuer's JWKS", kid)
	}

	return key, nil
}

// fetchJWKS replaces the cached key set. Callers hold jwksMu.
func (o *oidcLogin) fetchJWKS(ctx context.Context) error {
	if o.jwksURI == "" {
		disco, errCode := o.discovery(ctx)
		if errCode != "" {
			return errors.New("discovery unavailable")
		}

		o.jwksURI = disco.JWKSURI
	}

	body, err := o.fetchJSON(ctx, o.jwksURI)
	if err != nil {
		return fmt.Errorf("fetching the JWKS: %w", err)
	}

	var set struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}

	if err := json.Unmarshal(body, &set); err != nil {
		return fmt.Errorf("parsing the JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(set.Keys))

	for _, jwk := range set.Keys {
		if jwk.Kty != "RSA" || jwk.N == "" || jwk.E == "" {
			continue
		}

		key, err := oidcRSAKey(jwk.N, jwk.E)
		if err != nil {
			continue
		}

		keys[jwk.Kid] = key
	}

	if len(keys) == 0 {
		return errors.New("the JWKS carries no usable RSA keys")
	}

	o.jwks = keys
	o.jwksAt = o.clock()

	return nil
}

// fetchJSON GETs an issuer URL with the bounded client and returns the
// body.
func (o *oidcLogin) fetchJSON(ctx context.Context, target string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := o.httpClient().Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close() //nolint:errcheck // read-side close after the bounded read

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("issuer answered %s", resp.Status)
	}

	return io.ReadAll(io.LimitReader(resp.Body, oidcMaxIssuerBody))
}

// userinfo fetches the profile claims with the access token — the
// fallback for issuers whose id_tokens carry no email/groups claims.
func (o *oidcLogin) userinfo(ctx context.Context, accessToken string) (oidcClaims, error) {
	disco, errCode := o.discovery(ctx)
	if errCode != "" || disco.UserinfoEndpoint == "" {
		return oidcClaims{}, errors.New("no userinfo endpoint")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, disco.UserinfoEndpoint, nil)
	if err != nil {
		return oidcClaims{}, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := o.httpClient().Do(req)
	if err != nil {
		return oidcClaims{}, err
	}

	defer resp.Body.Close() //nolint:errcheck // read-side close after the bounded read

	body, err := io.ReadAll(io.LimitReader(resp.Body, oidcMaxIssuerBody))
	if err != nil {
		return oidcClaims{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return oidcClaims{}, fmt.Errorf("userinfo answered %s", resp.Status)
	}

	var claims oidcClaims
	if err := json.Unmarshal(body, &claims); err != nil {
		return oidcClaims{}, err
	}

	return claims, nil
}

// httpClient lazily builds the default issuer client.
func (o *oidcLogin) httpClient() *http.Client {
	if o.client != nil {
		return o.client
	}

	return &http.Client{Timeout: oidcHTTPTimeout}
}

// clock returns the current time through the injectable clock.
func (o *oidcLogin) clock() time.Time {
	if o.nowFunc != nil {
		return o.nowFunc()
	}

	return time.Now()
}
