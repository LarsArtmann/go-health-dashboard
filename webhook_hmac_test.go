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

// webhookCapture collects one delivery's headers and raw body from the
// receiver handler under a lock, so polls and assertions read a consistent
// snapshot.
type webhookCapture struct {
	mu       sync.Mutex
	headers  http.Header
	rawBody  []byte
	received bool
}

func (c *webhookCapture) record(r *http.Request) {
	raw, _ := io.ReadAll(r.Body)

	c.mu.Lock()
	c.headers = r.Header.Clone()
	c.rawBody = raw
	c.received = true
	c.mu.Unlock()
}

func (c *webhookCapture) header(name string) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.headers.Get(name)
}

func (c *webhookCapture) body() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.rawBody
}

func (c *webhookCapture) delivered() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.received
}

// waitForHookDelivery polls until the receiver has captured a delivery or
// the deadline passes, so webhook goroutine scheduling never races the
// assertions.
func waitForHookDelivery(t *testing.T, capture *webhookCapture) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for {
		if capture.delivered() {
			return
		}

		if time.Now().After(deadline) {
			t.Fatal("no webhook delivery arrived within 3s")
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// verifyLikeReceiver recomputes the receiver-side HMAC over the delivered
// "<timestamp>.<raw body>" material and compares constant-time, exactly as
// the WithWebhookHMAC doc contract describes.
func verifyLikeReceiver(t *testing.T, secret []byte, r *http.Request, rawBody []byte) bool {
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

	stamp := r.Header.Get("X-Health-Dashboard-Timestamp")
	if _, err := strconv.ParseInt(stamp, 10, 64); err != nil {
		return false
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(stamp))
	mac.Write([]byte("."))
	mac.Write(rawBody)

	return hmac.Equal(delivered, mac.Sum(nil))
}

// recomputeSignature is the receiver-side recompute over an explicit
// timestamp and raw body (for wrong-secret and tamper cases).
func recomputeSignature(secret []byte, stamp string, rawBody []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(stamp))
	mac.Write([]byte("."))
	mac.Write(rawBody)

	return mac.Sum(nil)
}

func TestWebhookHMAC_SignsDeliveries(t *testing.T) {
	t.Parallel()

	const hookSecret = "receiver-only-secret"

	capture := &webhookCapture{}

	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capture.record(r)
		w.WriteHeader(http.StatusOK)
	}))
	defer hook.Close()

	notifier := newWebhookNotifier(Config{
		WebhookURL:        hook.URL,
		WebhookHMACSecret: hookSecret,
	})

	notifier.fireOnChange(health.Response{
		Status: health.StatusPass,
		Checks: map[string]health.Check{"svc/db": {Status: health.StatusPass}},
	})

	waitForHookDelivery(t, capture)

	if len(capture.body()) == 0 {
		t.Fatal("delivered body empty")
	}

	stamp := capture.header("X-Health-Dashboard-Timestamp")
	if _, err := strconv.ParseInt(stamp, 10, 64); err != nil {
		t.Fatalf("non-integer timestamp %q: %v", stamp, err)
	}

	delivered, err := hex.DecodeString(
		strings.TrimPrefix(capture.header("X-Health-Dashboard-Signature"), "sha256="),
	)
	if err != nil {
		t.Fatalf("signature not sha256=-prefixed hex: %v", err)
	}

	if !hmac.Equal(delivered, recomputeSignature([]byte(hookSecret), stamp, capture.body())) {
		t.Fatal("receiver-side verification failed on a correctly-signed delivery")
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

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			capture := &webhookCapture{}

			hook := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					capture.record(r)
					w.WriteHeader(http.StatusOK)
				}),
			)
			defer hook.Close()

			notifier := newWebhookNotifier(Config{
				WebhookURL:        hook.URL,
				WebhookHMACSecret: testCase.configured,
				WebhookHeaders:    map[string]string{"X-Custom-Auth": "bearer"},
			})

			notifier.fireOnChange(health.Response{
				Status: health.StatusFail,
				Checks: map[string]health.Check{
					"svc/queue": {Status: health.StatusFail, Error: "down"},
				},
			})

			waitForHookDelivery(t, capture)

			signature := capture.header("X-Health-Dashboard-Signature")
			stamp := capture.header("X-Health-Dashboard-Timestamp")

			if testCase.wantHeaders {
				assertSignedDelivery(t, signature, stamp)
			} else {
				assertUnsignedDelivery(t, signature, stamp)
			}

			if custom := capture.header("X-Custom-Auth"); custom != "bearer" {
				t.Fatalf("custom header lost when HMAC configured: %q", custom)
			}
		})
	}
}

func assertSignedDelivery(t *testing.T, signature, stamp string) {
	t.Helper()

	if signature == "" || stamp == "" {
		t.Fatalf("signed delivery missing headers: sig %q ts %q", signature, stamp)
	}

	if _, err := strconv.ParseInt(stamp, 10, 64); err != nil {
		t.Fatalf("non-integer timestamp %q: %v", stamp, err)
	}

	if !strings.HasPrefix(signature, "sha256=") {
		t.Fatalf("signature lacks sha256= prefix: %q", signature)
	}
}

func assertUnsignedDelivery(t *testing.T, signature, stamp string) {
	t.Helper()

	if signature != "" || stamp != "" {
		t.Fatalf(
			"unsigned delivery carries signature headers: sig %q ts %q",
			signature,
			stamp,
		)
	}
}

func TestWebhookHMAC_WrongSecretAndTamperFailVerification(t *testing.T) {
	t.Parallel()

	const hookSecret = "the-real-secret"

	capture := &webhookCapture{}

	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capture.record(r)
		w.WriteHeader(http.StatusOK)
	}))
	defer hook.Close()

	notifier := newWebhookNotifier(Config{
		WebhookURL:        hook.URL,
		WebhookHMACSecret: hookSecret,
	})

	notifier.fireOnChange(health.Response{
		Status: health.StatusWarn,
		Checks: map[string]health.Check{"svc/cache": {Status: health.StatusWarn}},
	})

	waitForHookDelivery(t, capture)

	sig := capture.header("X-Health-Dashboard-Signature")
	stamp := capture.header("X-Health-Dashboard-Timestamp")

	delivered, err := hex.DecodeString(strings.TrimPrefix(sig, "sha256="))
	if err != nil {
		t.Fatalf("delivered signature not hex: %v", err)
	}

	// Wrong secret: a receiver configured with a different key must reject.
	if hmac.Equal(delivered, recomputeSignature([]byte("wrong-secret"), stamp, capture.body())) {
		t.Fatal("signature verified under a WRONG receiver secret")
	}

	// Sanity: the unmodified material verifies under the real secret.
	if !hmac.Equal(delivered, recomputeSignature([]byte(hookSecret), stamp, capture.body())) {
		t.Fatal("delivered signature does not verify under the real secret")
	}

	// Tamper: flipping one payload byte must invalidate verification.
	tampered := append([]byte(nil), capture.body()...)
	tampered[len(tampered)-1] ^= 0x20

	if hmac.Equal(delivered, recomputeSignature([]byte(hookSecret), stamp, tampered)) {
		t.Fatal("tampered body still verifies against the delivered signature")
	}
}
