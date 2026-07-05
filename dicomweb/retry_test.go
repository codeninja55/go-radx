package dicomweb

import (
	"context"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestRetryDefaultOff asserts the client behaviour is unchanged without WithRetry: a
// transient 500 is returned to the caller after a single attempt.
func TestRetryDefaultOff(t *testing.T) {
	var hits atomic.Int32
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.SearchStudies(context.Background(), SearchQuery{})
	var he *HTTPError
	if !errors.As(err, &he) || he.StatusCode != http.StatusInternalServerError {
		t.Fatalf("SearchStudies = %v, want *HTTPError 500", err)
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("origin saw %d requests, want 1 (retry must be off by default)", got)
	}
}

// TestRetryRecoversIdempotentGET asserts an opted-in client retries a GET through
// transient 5xx answers and succeeds once the origin recovers.
func TestRetryRecoversIdempotentGET(t *testing.T) {
	var hits atomic.Int32
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if hits.Add(1) <= 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()),
		WithRetry(RetryPolicy{MaxRetries: 3, Backoff: time.Millisecond, MaxBackoff: 5 * time.Millisecond}))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	results, err := c.SearchStudies(context.Background(), SearchQuery{})
	if err != nil {
		t.Fatalf("SearchStudies with retry = %v, want success after recovery", err)
	}
	if results != nil {
		t.Fatalf("results = %v, want the empty 204 result", results)
	}
	if got := hits.Load(); got != 3 {
		t.Fatalf("origin saw %d requests, want 3 (two failures then success)", got)
	}
}

// TestRetryExhaustsBudget asserts the retry budget is bounded: a persistently failing
// origin yields the final failure after exactly MaxRetries+1 attempts.
func TestRetryExhaustsBudget(t *testing.T) {
	var hits atomic.Int32
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()),
		WithRetry(RetryPolicy{MaxRetries: 2, Backoff: time.Millisecond, MaxBackoff: 5 * time.Millisecond}))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.SearchStudies(context.Background(), SearchQuery{})
	var he *HTTPError
	if !errors.As(err, &he) || he.StatusCode != http.StatusInternalServerError {
		t.Fatalf("SearchStudies = %v, want *HTTPError 500 after exhaustion", err)
	}
	if got := hits.Load(); got != 3 {
		t.Fatalf("origin saw %d requests, want 3 (initial + 2 retries)", got)
	}
}

// TestRetryNeverRepeatsStorePOST asserts a STOW POST is never retried, whatever the
// origin answers: replaying a store body could double-store instances.
func TestRetryNeverRepeatsStorePOST(t *testing.T) {
	var hits atomic.Int32
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()),
		WithRetry(RetryPolicy{MaxRetries: 3, Backoff: time.Millisecond, MaxBackoff: 5 * time.Millisecond}))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ds := sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5")
	if _, err := c.Store(context.Background(), ds); err == nil {
		t.Fatal("Store against a 503 origin returned nil error")
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("origin saw %d POSTs, want exactly 1 (STOW is never replayed)", got)
	}
}

// TestRetryHonorsRetryAfter asserts a 429 answer with a Retry-After delay is retried
// after that delay rather than the backoff schedule.
func TestRetryHonorsRetryAfter(t *testing.T) {
	var hits atomic.Int32
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if hits.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()),
		WithRetry(RetryPolicy{MaxRetries: 1, Backoff: time.Millisecond, MaxBackoff: 2 * time.Second}))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	start := time.Now()
	if _, err := c.SearchStudies(context.Background(), SearchQuery{}); err != nil {
		t.Fatalf("SearchStudies = %v, want success after the Retry-After wait", err)
	}
	if elapsed := time.Since(start); elapsed < time.Second {
		t.Fatalf("retry fired after %v, want at least the 1s Retry-After delay", elapsed)
	}
	if got := hits.Load(); got != 2 {
		t.Fatalf("origin saw %d requests, want 2", got)
	}
}

// TestRetryGivesUpOnExcessiveRetryAfter asserts a Retry-After beyond MaxBackoff returns
// the throttling answer instead of either hammering earlier than instructed or stalling
// past the policy's bound.
func TestRetryGivesUpOnExcessiveRetryAfter(t *testing.T) {
	var hits atomic.Int32
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()),
		WithRetry(RetryPolicy{MaxRetries: 3, Backoff: time.Millisecond, MaxBackoff: 50 * time.Millisecond}))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.SearchStudies(context.Background(), SearchQuery{})
	var he *HTTPError
	if !errors.As(err, &he) || he.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("SearchStudies = %v, want *HTTPError 503 without retrying", err)
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("origin saw %d requests, want 1 (Retry-After exceeded MaxBackoff)", got)
	}
}

// TestRetryStopsOnContextCancel asserts a cancelled context aborts the backoff wait: no
// further attempt is made once the caller has given up.
func TestRetryStopsOnContextCancel(t *testing.T) {
	var hits atomic.Int32
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()),
		WithRetry(RetryPolicy{MaxRetries: 5, Backoff: 5 * time.Second, MaxBackoff: 10 * time.Second}))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := c.SearchStudies(ctx, SearchQuery{}); err == nil {
		t.Fatal("SearchStudies with a cancelled context returned nil error")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("cancelled search took %v, want a prompt abort of the backoff wait", elapsed)
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("origin saw %d requests, want 1 (no retry after cancellation)", got)
	}
}

// countingTransport fails its first fail attempts with a transport error, then returns a
// canned 204, counting every attempt.
type countingTransport struct {
	attempts atomic.Int32
	fail     int32
}

func (t *countingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	if t.attempts.Add(1) <= t.fail {
		return nil, errors.New("simulated transport fault")
	}
	return &http.Response{
		StatusCode: http.StatusNoContent,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil
}

// TestRetryRecoversTransportError asserts a transport-level failure (connection reset,
// refused) is retried for an idempotent request.
func TestRetryRecoversTransportError(t *testing.T) {
	tr := &countingTransport{fail: 2}
	c, err := NewClient("https://pacs.example.org", WithHTTPClient(&http.Client{Transport: tr}),
		WithRetry(RetryPolicy{MaxRetries: 3, Backoff: time.Millisecond, MaxBackoff: 5 * time.Millisecond}))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.SearchStudies(context.Background(), SearchQuery{}); err != nil {
		t.Fatalf("SearchStudies = %v, want success after transport recovery", err)
	}
	if got := tr.attempts.Load(); got != 3 {
		t.Fatalf("transport saw %d attempts, want 3", got)
	}
}

// TestParseRetryAfter covers the two header forms RFC 9110 permits: delay seconds and an
// HTTP-date.
func TestParseRetryAfter(t *testing.T) {
	if d, ok := parseRetryAfter("2"); !ok || d != 2*time.Second {
		t.Fatalf("parseRetryAfter(2) = %v %v, want 2s true", d, ok)
	}
	future := time.Now().Add(30 * time.Second).UTC().Format(http.TimeFormat)
	if d, ok := parseRetryAfter(future); !ok || d <= 0 || d > 31*time.Second {
		t.Fatalf("parseRetryAfter(http-date) = %v %v, want ~30s true", d, ok)
	}
	if _, ok := parseRetryAfter("soon"); ok {
		t.Fatal("parseRetryAfter(soon) parsed, want false")
	}
	if _, ok := parseRetryAfter(""); ok {
		t.Fatal("parseRetryAfter(empty) parsed, want false")
	}
}

// TestRetryGivesUpOnOverflowingRetryAfter asserts an absurd Retry-After (large enough to
// overflow a naive seconds-to-Duration multiply) is clamped, exceeds MaxBackoff, and so
// returns the throttling answer after a single attempt - never a burst of instant
// retries from a wrapped-negative wait.
func TestRetryGivesUpOnOverflowingRetryAfter(t *testing.T) {
	var hits atomic.Int32
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Retry-After", "10000000000")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()),
		WithRetry(RetryPolicy{MaxRetries: 3, Backoff: time.Millisecond, MaxBackoff: 50 * time.Millisecond}))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.SearchStudies(context.Background(), SearchQuery{})
	var he *HTTPError
	if !errors.As(err, &he) || he.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("SearchStudies = %v, want *HTTPError 503 without retrying", err)
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("origin saw %d requests, want 1", got)
	}
}

// TestParseRetryAfterClampsOverflow asserts a delay-seconds value beyond the Duration
// range is clamped to the maximum duration rather than wrapping negative.
func TestParseRetryAfterClampsOverflow(t *testing.T) {
	d, ok := parseRetryAfter("10000000000")
	if !ok || d != time.Duration(math.MaxInt64) {
		t.Fatalf("parseRetryAfter(1e10) = %v %v, want the clamped maximum duration", d, ok)
	}
}

// TestRetrySkipsDeterministic5xx asserts only the transient statuses (500, 502, 503,
// 504, and 429) are retried: a 501 Not Implemented is deterministic and returns after a
// single attempt.
func TestRetrySkipsDeterministic5xx(t *testing.T) {
	var hits atomic.Int32
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusNotImplemented)
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()),
		WithRetry(RetryPolicy{MaxRetries: 3, Backoff: time.Millisecond, MaxBackoff: 5 * time.Millisecond}))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.SearchStudies(context.Background(), SearchQuery{})
	var he *HTTPError
	if !errors.As(err, &he) || he.StatusCode != http.StatusNotImplemented {
		t.Fatalf("SearchStudies = %v, want *HTTPError 501", err)
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("origin saw %d requests, want 1 (501 is not transient)", got)
	}
}
