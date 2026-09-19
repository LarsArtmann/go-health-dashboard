package main

import (
	"reflect"
	"strings"
	"testing"
	"time"

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
