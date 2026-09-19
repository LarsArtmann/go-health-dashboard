package main

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
	"unicode"

	healthfederation "github.com/larsartmann/go-health/federation"
)

// FuzzParseRemotes exercises the remote-spec parser with arbitrary input.
// Invariants: the parser never panics, is deterministic, and every accepted
// remote carries a unique whitespace-free slash-free name plus an absolute
// http(s) URL with a host — the same contract federation.New enforces
// upstream, applied one boundary earlier so a bad environment fails fast
// with a named entry instead of surfacing at serve time.
func FuzzParseRemotes(f *testing.F) {
	for _, seed := range []string{
		"cv=http://127.0.0.1:8098/health",
		"cv=http://127.0.0.1:8098/health,forgejo=http://forgejo.home.lan:3000/health",
		" cv = http://127.0.0.1:8098/health ",
		"cv=http://127.0.0.1:8098/health?token=a=b",
		"cv=http://monitor:secret@127.0.0.1:8098/health",
		"cv=HTTP://127.0.0.1:8098/health",
		"héllo=http://127.0.0.1:8098/health",
		"",
		",",
		",,",
		"cv",
		"=",
		"a=b=c",
		"=http://127.0.0.1:8098/health",
		"cv=",
		"cv=http://",
		"cv=ftp://127.0.0.1/health",
		"cv=x/health",
		"a/b=http://127.0.0.1:8098/health",
		"cv backup=http://127.0.0.1:8098/health",
		"cv\tbackup=http://127.0.0.1:8098/health",
		"cv=http://a/health,cv=http://b/health",
		"cv=http://[::1",
		strings.Repeat("a=http://127.0.0.1/health,", 64),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, spec string) {
		got, err := parseRemotes(spec)

		if err != nil {
			if got != nil {
				t.Fatalf("parseRemotes(%q) returned remotes alongside error %v", spec, err)
			}

			return
		}

		if len(got) == 0 {
			t.Fatalf("parseRemotes(%q) accepted an empty remote list", spec)
		}

		seen := make(map[string]bool, len(got))
		for _, remote := range got {
			if remote.Name == "" ||
				strings.Contains(remote.Name, "/") ||
				strings.ContainsFunc(remote.Name, unicode.IsSpace) {
				t.Fatalf("parseRemotes(%q) accepted invalid remote name %q", spec, remote.Name)
			}

			if seen[remote.Name] {
				t.Fatalf("parseRemotes(%q) accepted duplicate remote name %q", spec, remote.Name)
			}
			seen[remote.Name] = true

			parsed, parseErr := url.Parse(remote.URL)
			if parseErr != nil ||
				(parsed.Scheme != "http" && parsed.Scheme != "https") ||
				parsed.Host == "" {
				t.Fatalf("parseRemotes(%q) accepted invalid remote URL %q", spec, remote.URL)
			}
		}

		again, againErr := parseRemotes(spec)
		if againErr != nil || !reflect.DeepEqual(got, again) {
			t.Fatalf(
				"parseRemotes(%q) is not deterministic: first %+v/%v, then %+v/%v",
				spec, got, err, again, againErr,
			)
		}
	})
}
