package dashboard

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
)

// verifyLikeReceiver recomputes the receiver-side HMAC over the delivered
// "<timestamp>.<raw body>" material and compares constant-time, exactly as
// the WithWebhookHMAC doc contract describes.
func verifyLikeReceiver(secret []byte, t *testing.T, r *http.Request, rawBody []byte) bool {
	t.Helper()

	sig := r.Header.Get("X-Health-Dashboard-Signature")
	hexSig, found := strings.CutPrefix(sig, "sha256=")
	if !found {
		return false
	}

	delivered, err := hex.DecodeString(hexSig)
	if err != nil {
		return false
	}

	ts := r.Header.Get("X-Health-Dashboard-Timestamp")
	if _, err := strconv.ParseInt(ts, 10, 64); err != nil {
		return false
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(ts))
	mac.Write([]byte("."))
	mac.Write(rawBody)

	return hmac.Equal(delivered, mac.Sum(nil))
}

func TestWebhookHMAC_SignsDeliveries(t *testing.T) {
	t.Parallel()

	const secret = "receiver-only-secret"

	var (
		mu     sync.Mutex
		valid  bool
		body   []byte
		header http.Header
		seen   bool
	)

	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)

		mu.Lock()
		valid = verifyLikeReceiver([]byte(secret), t, r, raw)
		body = raw
		header = r.Header.Clone()
		seen = true
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer hook.Close()

	notifier := newWebhookNotifier(Config{
		WebhookURL:        hook.URL,
		WebhookHMACSecret: secret,
	})

	notifier.fireOnChange(health.Response{
		Status: health.StatusPass,
		Checks: map[string]health.Check{"svc/db": {Status: health.StatusPass}},
	})

	deadline := time.Now().Add(3 * time.Second)
	for {
		mu.Lock()
		done := seen
		mu.Unlock()

		if done {
			break
		}

		if time.Now().After(deadline) {
			t.Fatal("no signed delivery arrived within 3s")
		}

		time.Sleep(10 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()

	if !valid {
		t.Fatalf("receiver rejected the signature on a signed delivery (sig %q, ts %q)",
			header.Get("X-Health-Dashboard-Signature"),
			header.Get("X-Health-Dashboard-Timestamp"))
	}

	if len(body) == 0 {
		t.Fatal("delivered body empty")
	}
}

func TestWebhookHMAC_Table(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		configured  string
		wantHeaders bool
	}{
		{name: "unsigned when no secret", configured: "", wantHeaders: false},
		{name: "signed when secret set", configured: "s3cret", wantHeaders: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var (
				mu        sync.Mutex
				signature string
				timestamp string
				custom    string
				seen      bool
			)

			hook := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					signature = r.Header.Get("X-Health-Dashboard-Signature")
					timestamp = r.Header.Get("X-Health-Dashboard-Timestamp")
					custom = r.Header.Get("X-Custom-Auth")
					seen = true
					mu.Unlock()

					w.WriteHeader(http.StatusOK)
				}),
			)
			defer hook.Close()

			notifier := newWebhookNotifier(Config{
				WebhookURL:        hook.URL,
				WebhookHMACSecret: tc.configured,
				WebhookHeaders:    map[string]string{"X-Custom-Auth": "bearer"},
			})

			notifier.fireOnChange(health.Response{
				Status: health.StatusFail,
				Checks: map[string]health.Check{
					"svc/queue": {Status: health.StatusFail, Error: "down"},
				},
			})

			deadline := time.Now().Add(3 * time.Second)
			for {
				mu.Lock()
				op := seen
				mu.Unlock()

				if op {
					break
				}

				if time.Now().After(deadline) {
					t.Fatal("no delivery arrived within 3s")
				}

				time.Sleep(10 * time.Millisecond)
			}

			mu.Lock()
			defer mu.Unlock()

			if tc.wantHeaders {
				if signature == "" || timestamp == "" {
					t.Fatalf("signed delivery missing headers: sig %q ts %q", signature, timestamp)
				}

				if _, err := strconv.ParseInt(timestamp, 10, 64); err != nil {
					t.Fatalf("non-integer timestamp %q: %v", timestamp, err)
				}

				if !strings.HasPrefix(signature, "sha256=") {
					t.Fatalf("signature lacks sha256= prefix: %q", signature)
				}
			} else {
				if signature != "" || timestamp != "" {
					t.Fatalf(
						"unsigned delivery carries signature headers: sig %q ts %q",
						signature,
						timestamp,
					)
				}
			}

			if custom != "bearer" {
				t.Fatalf("custom header lost when HMAC configured: %q", custom)
			}
		})
	}
}

func TestWebhookHMAC_WrongSecretAndTamperFailVerification(t *testing.T) {
	t.Parallel()

	const secret = "the-real-secret"

	var (
		mu   sync.Mutex
		hdr  http.Header
		raw  []byte
		seen bool
	)

	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		mu.Lock()
		hdr = r.Header.Clone()
		raw = body
		seen = true
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer hook.Close()

	notifier := newWebhookNotifier(Config{
		WebhookURL:        hook.URL,
		WebhookHMACSecret: secret,
	})

	notifier.fireOnChange(health.Response{
		Status: health.StatusWarn,
		Checks: map[string]health.Check{"svc/cache": {Status: health.StatusWarn}},
	})

	deadline := time.Now().Add(3 * time.Second)
	for {
		mu.Lock()
		op := seen
		mu.Unlock()

		if op {
			break
		}

		if time.Now().After(deadline) {
			t.Fatal("no delivery arrived within 3s")
		}

		time.Sleep(10 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()

	sig := hdr.Get("X-Health-Dashboard-Signature")
	ts := hdr.Get("X-Health-Dashboard-Timestamp")

	// Wrong secret: a receiver configured with a different key must reject.
	mac := hmac.New(sha256.New, []byte("wrong-secret"))
	mac.Write([]byte(ts))
	mac.Write([]byte("."))
	mac.Write(raw)

	delivered, err := hex.DecodeString(strings.TrimPrefix(sig, "sha256="))
	if err != nil {
		t.Fatalf("delivered signature not hex: %v", err)
	}

	if hmac.Equal(delivered, mac.Sum(nil)) {
		t.Fatal("signature verified under a WRONG receiver secret")
	}

	// Sanity: the unmodified material verifies under the real secret.
	realMac := hmac.New(sha256.New, []byte(secret))
	realMac.Write([]byte(ts))
	realMac.Write([]byte("."))
	realMac.Write(raw)
	if !hmac.Equal(delivered, realMac.Sum(nil)) {
		t.Fatal("delivered signature does not verify under the real secret")
	}

	// Tamper: flipping one payload byte must invalidate verification.
	tampered := append([]byte(nil), raw...)
	tampered[len(tampered)-1] ^= 0x20

	tamperMac := hmac.New(sha256.New, []byte(secret))
	tamperMac.Write([]byte(ts))
	tamperMac.Write([]byte("."))
	tamperMac.Write(tampered)

	if hmac.Equal(delivered, tamperMac.Sum(nil)) {
		t.Fatal("tampered body still verifies against the delivered signature")
	}
}
