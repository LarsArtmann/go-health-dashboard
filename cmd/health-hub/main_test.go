package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
	dashboard "github.com/larsartmann/go-health-dashboard"
	healthfederation "github.com/larsartmann/go-health/federation"
)

func TestParseRemotes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		spec    string
		want    []healthfederation.Remote
		wantErr string
	}{
		{
			name: "single remote",
			spec: "cv=http://127.0.0.1:8098/health",
			want: []healthfederation.Remote{{Name: "cv", URL: "http://127.0.0.1:8098/health"}},
		},
		{
			name: "multiple remotes",
			spec: "cv=http://127.0.0.1:8098/health,forgejo=http://forgejo.home.lan:3000/health",
			want: []healthfederation.Remote{
				{Name: "cv", URL: "http://127.0.0.1:8098/health"},
				{Name: "forgejo", URL: "http://forgejo.home.lan:3000/health"},
			},
		},
		{
			name: "surrounding whitespace is trimmed",
			spec: " cv = http://127.0.0.1:8098/health , db = https://db.home.lan/health ",
			want: []healthfederation.Remote{
				{Name: "cv", URL: "http://127.0.0.1:8098/health"},
				{Name: "db", URL: "https://db.home.lan/health"},
			},
		},
		{
			name: "equals signs inside the URL survive the first cut",
			spec: "cv=http://127.0.0.1:8098/health?token=a=b",
			want: []healthfederation.Remote{
				{Name: "cv", URL: "http://127.0.0.1:8098/health?token=a=b"},
			},
		},
		{
			name: "userinfo in the URL is accepted",
			spec: "cv=http://metrics@127.0.0.1:8098/health",
			want: []healthfederation.Remote{
				{Name: "cv", URL: "http://metrics@127.0.0.1:8098/health"},
			},
		},
		{
			name: "uppercase scheme is accepted (url.Parse normalizes)",
			spec: "cv=HTTP://127.0.0.1:8098/health",
			want: []healthfederation.Remote{{Name: "cv", URL: "HTTP://127.0.0.1:8098/health"}},
		},
		{
			name:    "empty spec",
			spec:    "",
			wantErr: "at least one name=url entry is required",
		},
		{
			name:    "whitespace-only spec is an empty spec",
			spec:    "   ",
			wantErr: "at least one name=url entry is required",
		},
		{
			name:    "entry without equals sign names the offender",
			spec:    "cv=http://127.0.0.1:8098/health,forgejo",
			wantErr: `entry 1 ("forgejo")`,
		},
		{
			name:    "empty name",
			spec:    "=http://127.0.0.1:8098/health",
			wantErr: "want name=url with a non-empty name",
		},
		{
			name:    "empty URL",
			spec:    "cv=",
			wantErr: "want name=url with a non-empty name",
		},
		{
			name:    "URL without host",
			spec:    "cv=http://",
			wantErr: "URL must be absolute http(s)",
		},
		{
			name:    "non-http scheme",
			spec:    "cv=ftp://127.0.0.1/health",
			wantErr: "URL must be absolute http(s)",
		},
		{
			name:    "relative URL",
			spec:    "cv=x/health",
			wantErr: "URL must be absolute http(s)",
		},
		{
			name:    "unparseable URL",
			spec:    "cv=http://[::1",
			wantErr: "invalid URL",
		},
		{
			name:    "slash in name",
			spec:    "a/b=http://127.0.0.1:8098/health",
			wantErr: `name must not contain "/"`,
		},
		{
			name:    "space inside name",
			spec:    "cv backup=http://127.0.0.1:8098/health",
			wantErr: "must not contain whitespace",
		},
		{
			name:    "tab inside name",
			spec:    "cv\tbackup=http://127.0.0.1:8098/health",
			wantErr: "must not contain whitespace",
		},
		{
			name:    "duplicate remote name",
			spec:    "cv=http://127.0.0.1:8098/health,cv=http://127.0.0.1:8099/health",
			wantErr: `duplicate remote name "cv"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseRemotes(tt.spec)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf(
						"parseRemotes(%q) error = nil, want containing %q",
						tt.spec,
						tt.wantErr,
					)
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf(
						"parseRemotes(%q) error = %q, want containing %q",
						tt.spec,
						err,
						tt.wantErr,
					)
				}

				if got != nil {
					t.Fatalf("parseRemotes(%q) remotes = %v, want nil on error", tt.spec, got)
				}

				return
			}

			if err != nil {
				t.Fatalf("parseRemotes(%q) unexpected error: %v", tt.spec, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseRemotes(%q) = %+v, want %+v", tt.spec, got, tt.want)
			}
		})
	}
}

func TestParseTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    time.Duration
		wantErr bool
	}{
		{name: "seconds", raw: "5s", want: 5 * time.Second},
		{name: "milliseconds", raw: "500ms", want: 500 * time.Millisecond},
		{name: "compound", raw: "1m30s", want: 90 * time.Second},
		{name: "hours", raw: "2h", want: 2 * time.Hour},
		{name: "zero is rejected", raw: "0", wantErr: true},
		{name: "negative is rejected", raw: "-3s", wantErr: true},
		{name: "bare number is rejected", raw: "5", wantErr: true},
		{name: "unitless words are rejected", raw: "five seconds", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseTimeout(tt.raw)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseTimeout(%q) error = nil, want failure", tt.raw)
				}

				if !strings.Contains(err.Error(), tt.raw) {
					t.Fatalf(
						"parseTimeout(%q) error = %q, want it to name the offending value",
						tt.raw,
						err,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("parseTimeout(%q) unexpected error: %v", tt.raw, err)
			}

			if got != tt.want {
				t.Fatalf("parseTimeout(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestOptionsFromEnv(t *testing.T) {
	tests := []struct {
		name      string
		trend     string
		metrics   string
		wantCount int
	}{
		{name: "both off by default", wantCount: 0},
		{name: "trend only", trend: "1", wantCount: 1},
		{name: "metrics only", metrics: "1", wantCount: 1},
		{name: "both on", trend: "1", metrics: "1", wantCount: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.trend != "" {
				t.Setenv(trendEnvVar, tt.trend)
			}

			if tt.metrics != "" {
				t.Setenv(metricsEnvVar, tt.metrics)
			}

			if got := len(optionsFromEnv()); got != tt.wantCount {
				t.Fatalf("optionsFromEnv() returned %d options, want %d", got, tt.wantCount)
			}
		})
	}
}

func TestEnvOrDefault(t *testing.T) {
	t.Setenv(portEnvVar, "9090")

	if got := envOrDefault(portEnvVar, "8080"); got != "9090" {
		t.Fatalf("envOrDefault with a set variable = %q, want %q", got, "9090")
	}

	if got := envOrDefault(addrEnvVar, ":8080"); got != ":8080" {
		t.Fatalf("envOrDefault with an unset variable = %q, want the fallback %q", got, ":8080")
	}
}

// stubProber satisfies dashboard.Prober with canned handlers, so the mux
// wiring can be exercised without a running probe.
type stubProber struct{}

func (stubProber) CachedResponse() health.Response { return health.Response{} }
func (stubProber) RefreshInterval() time.Duration  { return time.Minute }

func (stubProber) LivenessHandler() http.HandlerFunc  { return serveOK }
func (stubProber) ReadinessHandler() http.HandlerFunc { return serveOK }
func (stubProber) StartupHandler() http.HandlerFunc   { return serveOK }

func serveOK(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestNewServeMux(t *testing.T) {
	t.Parallel()

	mux := newServeMux(dashboard.New(stubProber{}))

	server := httptest.NewServer(mux)
	// Not defer: parallel subtests resume only after this function returns,
	// and t.Cleanup waits for them, unlike a deferred close.
	t.Cleanup(server.Close)

	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	tests := []struct {
		name         string
		path         string
		wantStatus   int
		wantLocation string
	}{
		{
			name:         "root redirects to the dashboard",
			path:         "/",
			wantStatus:   http.StatusTemporaryRedirect,
			wantLocation: "/health",
		},
		{
			name:       "registered liveness route wins over the root pattern",
			path:       "/healthz",
			wantStatus: http.StatusOK,
		},
		{
			name:       "registered favicon route wins over the root pattern",
			path:       "/favicon.svg",
			wantStatus: http.StatusOK,
		},
		{
			name:       "unknown path keeps 404ing",
			path:       "/definitely-not-a-route",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp, err := client.Get(server.URL + tt.path)
			if err != nil {
				t.Fatalf("GET %s: %v", tt.path, err)
			}

			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("GET %s status = %d, want %d", tt.path, resp.StatusCode, tt.wantStatus)
			}

			if tt.wantLocation != "" && resp.Header.Get("Location") != tt.wantLocation {
				t.Errorf(
					"GET %s Location = %q, want %q",
					tt.path,
					resp.Header.Get("Location"),
					tt.wantLocation,
				)
			}
		})
	}
}

func TestParsePushInterval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    time.Duration
		wantErr bool
	}{
		{name: "empty means library default", raw: "", want: 0},
		{name: "seconds", raw: "10s", want: 10 * time.Second},
		{name: "minutes", raw: "1m", want: time.Minute},
		{name: "zero is rejected", raw: "0", wantErr: true},
		{name: "negative is rejected", raw: "-5s", wantErr: true},
		{name: "bare number is rejected", raw: "10", wantErr: true},
		{name: "unitless words are rejected", raw: "slow", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parsePushInterval(tt.raw)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("parsePushInterval(%q) error = nil, want failure", tt.raw)
				}

				if !strings.Contains(err.Error(), tt.raw) {
					t.Fatalf(
						"parsePushInterval(%q) error %q does not name the offending value",
						tt.raw,
						err,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("parsePushInterval(%q) error = %v, want nil", tt.raw, err)
			}

			if got != tt.want {
				t.Fatalf("parsePushInterval(%q) = %s, want %s", tt.raw, got, tt.want)
			}
		})
	}
}

func TestResolvedPushInterval(t *testing.T) {
	t.Parallel()

	if got := resolvedPushInterval(0); got != dashboard.DefaultPushInterval {
		t.Fatalf(
			"resolvedPushInterval(0) = %s, want the library default %s",
			got,
			dashboard.DefaultPushInterval,
		)
	}

	if got := resolvedPushInterval(15 * time.Second); got != 15*time.Second {
		t.Fatalf("resolvedPushInterval(15s) = %s, want 15s", got)
	}
}

// TestConfigFromEnv_HappyPaths pins the full environment contract of the
// hub end to end (the per-knob parsers have their own tables; this proves
// configFromEnv assembles them): defaults when only remotes are set, and
// every knob honored — timeout, push interval, drain window, TLS CA path,
// proxy — with trimmed values. Not parallel: t.Setenv forbids it, and env
// isolation matters more here than scheduling freedom.
func TestConfigFromEnv_HappyPaths(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want func(t *testing.T, cfg hubConfig)
	}{
		{
			name: "only remotes: library defaults everywhere",
			env:  map[string]string{"HEALTH_HUB_REMOTES": "cv=http://127.0.0.1:8098/health"},
			want: wantConfigDefaults,
		},
		{
			name: "every knob set",
			env: map[string]string{
				"HEALTH_HUB_REMOTES":       "cv=http://127.0.0.1:8098/health,forgejo=http://forgejo.home.lan:3000/health",
				"HEALTH_HUB_TIMEOUT":       "3s",
				"HEALTH_HUB_PUSH_INTERVAL": "10s",
				"HEALTH_HUB_SSE_DRAIN":     "2s",
				"HEALTH_HUB_CLIENT_TLS_CA": " /etc/ssl/corp-ca.pem ",
				"HEALTH_HUB_HTTP_PROXY":    " http://proxy.lan:3128 ",
			},
			want: wantConfigEveryKnob,
		},
		{
			name: "empty optional knobs read as unset",
			env: map[string]string{
				"HEALTH_HUB_REMOTES":       "db=https://db.home.lan/health",
				"HEALTH_HUB_TIMEOUT":       "",
				"HEALTH_HUB_PUSH_INTERVAL": "",
				"HEALTH_HUB_SSE_DRAIN":     "",
				"HEALTH_HUB_CLIENT_TLS_CA": "",
				"HEALTH_HUB_HTTP_PROXY":    "",
			},
			want: wantConfigDefaults,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			cfg, err := configFromEnv()
			if err != nil {
				t.Fatalf("configFromEnv: %v", err)
			}

			tt.want(t, cfg)
		})
	}
}

// wantConfigDefaults asserts the library-default shape: one remote, the
// default fetch timeout, no push/drain override, no TLS/proxy override.
func wantConfigDefaults(t *testing.T, cfg hubConfig) {
	t.Helper()

	if len(cfg.remotes) != 1 || cfg.remotes[0].Name != "cv" && cfg.remotes[0].Name != "db" {
		t.Errorf("remotes = %+v", cfg.remotes)
	}

	if cfg.fetchExpiry != defaultFetchExpiry {
		t.Errorf("fetchExpiry = %s, want default %s", cfg.fetchExpiry, defaultFetchExpiry)
	}

	if cfg.pushInterval != 0 || cfg.sseDrain != 0 {
		t.Errorf("push=%s drain=%s, want zero (library defaults)", cfg.pushInterval, cfg.sseDrain)
	}

	if cfg.tlsCAPath != "" || cfg.httpProxy != "" {
		t.Errorf("tlsCA=%q proxy=%q, want empty", cfg.tlsCAPath, cfg.httpProxy)
	}
}

// wantConfigEveryKnob asserts the fully-populated shape, including value
// trimming on the free-form path/URL knobs.
func wantConfigEveryKnob(t *testing.T, cfg hubConfig) {
	t.Helper()

	if len(cfg.remotes) != 2 {
		t.Fatalf("remotes = %+v, want 2", cfg.remotes)
	}

	if cfg.fetchExpiry != 3*time.Second {
		t.Errorf("fetchExpiry = %s, want 3s", cfg.fetchExpiry)
	}

	if cfg.pushInterval != 10*time.Second {
		t.Errorf("pushInterval = %s, want 10s", cfg.pushInterval)
	}

	if cfg.sseDrain != 2*time.Second {
		t.Errorf("sseDrain = %s, want 2s", cfg.sseDrain)
	}

	if cfg.tlsCAPath != "/etc/ssl/corp-ca.pem" {
		t.Errorf("tlsCAPath = %q, want trimmed path", cfg.tlsCAPath)
	}

	if cfg.httpProxy != "http://proxy.lan:3128" {
		t.Errorf("httpProxy = %q, want trimmed URL", cfg.httpProxy)
	}
}
