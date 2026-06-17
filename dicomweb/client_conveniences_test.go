package dicomweb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
)

// TestRetryOnTransientStatus asserts a flaky origin that returns 503 a fixed number of times then
// succeeds is retried to success when WithRetry is configured (matching the dicomweb-client
// set_http_retry_params behaviour).
func TestRetryOnTransientStatus(t *testing.T) {
	var calls atomic.Int32
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("try again"))
			return
		}
		w.Header().Set("Content-Type", mediaTypeDICOMJSON)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(hs.Close)

	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()),
		WithRetry(RetryPolicy{MaxRetries: 3, BaseDelay: time.Millisecond}))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	results, err := c.SearchStudies(context.Background(), SearchQuery{})
	if err != nil {
		t.Fatalf("SearchStudies with retry: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("results = %d, want 0", len(results))
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("origin saw %d calls, want 3 (2 failures then success)", got)
	}
}

// TestRetryExhaustionReturnsLastFailure asserts that when every attempt fails the retryable
// status, the client surfaces the failure rather than masking it, and makes exactly
// MaxRetries+1 attempts.
func TestRetryExhaustionReturnsLastFailure(t *testing.T) {
	var calls atomic.Int32
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(hs.Close)

	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()),
		WithRetry(RetryPolicy{MaxRetries: 2, BaseDelay: time.Millisecond}))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.SearchStudies(context.Background(), SearchQuery{})
	if err == nil {
		t.Fatal("SearchStudies against an always-503 origin returned nil error")
	}
	var he *HTTPError
	if !errors.As(err, &he) || he.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("error = %v, want *HTTPError 503", err)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("origin saw %d calls, want 3 (1 + 2 retries)", got)
	}
}

// TestNoRetryOnDeterministicStatus asserts a 404 is never retried: a deterministic client fault
// cannot be fixed by a replay, so the client must not waste requests.
func TestNoRetryOnDeterministicStatus(t *testing.T) {
	var calls atomic.Int32
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(hs.Close)

	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()),
		WithRetry(RetryPolicy{MaxRetries: 3, BaseDelay: time.Millisecond}))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, _ = c.SearchStudies(context.Background(), SearchQuery{})
	if got := calls.Load(); got != 1 {
		t.Fatalf("origin saw %d calls, want 1 (no retry on 404)", got)
	}
}

// TestByteRangeBulkDataRequestsRange asserts RetrieveBulkDataRange sends the Range header and
// returns the requested span when the origin honours it with 206 Partial Content.
func TestByteRangeBulkDataRequestsRange(t *testing.T) {
	const full = "0123456789ABCDEF"
	var seenRange string
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenRange = r.Header.Get("Range")
		body := full
		status := http.StatusOK
		if rng := r.Header.Get("Range"); rng != "" {
			start, end := parseTestRange(rng, len(full))
			body = full[start : end+1]
			status = http.StatusPartialContent
		}
		var buf bytes.Buffer
		mw := NewMultipartWriter(&buf, mediaTypeOctet)
		if err := mw.AddPart(mediaTypeOctet, strings.NewReader(body)); err != nil {
			t.Errorf("AddPart: %v", err)
		}
		if _, err := mw.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
		w.Header().Set("Content-Type", mw.ContentType())
		w.WriteHeader(status)
		_, _ = w.Write(buf.Bytes())
	}))
	t.Cleanup(hs.Close)

	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	p := NewInstance("1.2.3", "1.2.3.4", "1.2.3.4.5")
	parts, err := c.RetrieveBulkDataRange(context.Background(), p, ByteRange{Start: 4, End: Int64Ptr(7)})
	if err != nil {
		t.Fatalf("RetrieveBulkDataRange: %v", err)
	}
	if seenRange != "bytes=4-7" {
		t.Fatalf("origin saw Range = %q, want bytes=4-7", seenRange)
	}
	if len(parts) != 1 || string(parts[0]) != "4567" {
		t.Fatalf("range body = %q, want 4567", parts)
	}
}

// TestByteRangeFirstByteProbe asserts ByteRange{Start: 0, End: Int64Ptr(0)} sends "bytes=0-0" and
// returns exactly the first byte, the inclusive first-byte probe that the old End<=0 open-ended
// sentinel could not express (RFC 7233 §2.1, PS3.18 §8.7.4).
func TestByteRangeFirstByteProbe(t *testing.T) {
	const full = "0123456789ABCDEF"
	var seenRange string
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenRange = r.Header.Get("Range")
		body := full
		status := http.StatusOK
		if rng := r.Header.Get("Range"); rng != "" {
			start, end := parseTestRange(rng, len(full))
			body = full[start : end+1]
			status = http.StatusPartialContent
		}
		var buf bytes.Buffer
		mw := NewMultipartWriter(&buf, mediaTypeOctet)
		if err := mw.AddPart(mediaTypeOctet, strings.NewReader(body)); err != nil {
			t.Errorf("AddPart: %v", err)
		}
		if _, err := mw.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
		w.Header().Set("Content-Type", mw.ContentType())
		w.WriteHeader(status)
		_, _ = w.Write(buf.Bytes())
	}))
	t.Cleanup(hs.Close)

	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	p := NewInstance("1.2.3", "1.2.3.4", "1.2.3.4.5")
	parts, err := c.RetrieveBulkDataRange(context.Background(), p, ByteRange{Start: 0, End: Int64Ptr(0)})
	if err != nil {
		t.Fatalf("RetrieveBulkDataRange: %v", err)
	}
	if seenRange != "bytes=0-0" {
		t.Fatalf("origin saw Range = %q, want bytes=0-0", seenRange)
	}
	if len(parts) != 1 || string(parts[0]) != "0" {
		t.Fatalf("first-byte probe body = %q, want a single byte %q", parts, "0")
	}
}

// TestByteRangeValidation asserts a degenerate range (end before start) is rejected before any
// request.
func TestByteRangeValidation(t *testing.T) {
	c, err := NewClient("https://pacs.example.org")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	p := NewInstance("1.2.3", "1.2.3.4", "1.2.3.4.5")
	if _, err := c.RetrieveBulkDataRange(context.Background(), p, ByteRange{Start: 10, End: Int64Ptr(4)}); !errors.Is(err, ErrInvalidResource) {
		t.Fatalf("inverted range error = %v, want ErrInvalidResource", err)
	}
}

// TestByteRangeHeaderForms covers the header rendering for closed, open-ended, and suffix ranges.
func TestByteRangeHeaderForms(t *testing.T) {
	cases := []struct {
		br   ByteRange
		want string
	}{
		{ByteRange{Start: 0, End: Int64Ptr(99)}, "bytes=0-99"},
		{ByteRange{Start: 0, End: Int64Ptr(0)}, "bytes=0-0"},
		{ByteRange{Start: 5, End: Int64Ptr(5)}, "bytes=5-5"},
		{ByteRange{Start: 100}, "bytes=100-"},
		{ByteRange{Start: 100, End: nil}, "bytes=100-"},
		{ByteRange{Start: -256}, "bytes=-256"},
	}
	for _, tc := range cases {
		if got := tc.br.header(); got != tc.want {
			t.Errorf("ByteRange%+v header = %q, want %q", tc.br, got, tc.want)
		}
	}
}

// TestSearchAllStudiesPaginates asserts SearchAllStudies pages through the whole result set with a
// small page size, yielding every match exactly once across multiple pages.
func TestSearchAllStudiesPaginates(t *testing.T) {
	candidates := make([]*dicom.DataSet, 0, 5)
	for i := range 5 {
		candidates = append(candidates, studyRecord(
			fmt.Sprintf("1.2.%d", i), fmt.Sprintf("PID%d", i), "Doe^Jane", "20200101", "ACC", "CT"))
	}
	hs := newQueryTestServer(t, &memQuery{candidates: candidates})
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	seen := make(map[string]int)
	var count int
	for r, err := range c.SearchAllStudies(context.Background(), SearchQuery{Limit: 2}) {
		if err != nil {
			t.Fatalf("SearchAllStudies yielded error: %v", err)
		}
		uid, _ := r.DataSet.GetString(dicom.TagStudyInstanceUID)
		seen[uid]++
		count++
	}
	if count != 5 {
		t.Fatalf("paged %d results, want 5", count)
	}
	for uid, n := range seen {
		if n != 1 {
			t.Errorf("study %s yielded %d times, want once", uid, n)
		}
	}
}

// TestSearchAllStudiesEmpty asserts an empty result set yields nothing without error.
func TestSearchAllStudiesEmpty(t *testing.T) {
	hs := newQueryTestServer(t, &memQuery{})
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	var count int
	for _, err := range c.SearchAllStudies(context.Background(), SearchQuery{Limit: 2}) {
		if err != nil {
			t.Fatalf("SearchAllStudies yielded error: %v", err)
		}
		count++
	}
	if count != 0 {
		t.Fatalf("empty search paged %d results, want 0", count)
	}
}

// TestSearchAllStudiesPagesPastServerCap asserts the auto-paginating iterator yields every match
// even when the origin caps each page below the requested page size (here WithMaxQueryResults(2)
// against a Limit of 10) and signals the truncation with a Warning: 299 header (PS3.18 §10.6.1.4).
// Stopping on the first short page would drop the matches beyond the cap; advancing the offset by
// the rows actually returned reaches all of them and terminates on the empty final page.
func TestSearchAllStudiesPagesPastServerCap(t *testing.T) {
	candidates := make([]*dicom.DataSet, 0, 5)
	for i := range 5 {
		candidates = append(candidates, studyRecord(
			fmt.Sprintf("1.2.%d", i), fmt.Sprintf("PID%d", i), "Doe^Jane", "20200101", "ACC", "CT"))
	}
	hs := newQueryTestServer(t, &memQuery{candidates: candidates}, WithMaxQueryResults(2))
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	seen := make(map[string]int)
	var count int
	for r, err := range c.SearchAllStudies(context.Background(), SearchQuery{Limit: 10}) {
		if err != nil {
			t.Fatalf("SearchAllStudies yielded error: %v", err)
		}
		uid, _ := r.DataSet.GetString(dicom.TagStudyInstanceUID)
		seen[uid]++
		count++
	}
	if count != 5 {
		t.Fatalf("server-capped paging yielded %d results, want all 5", count)
	}
	for uid, n := range seen {
		if n != 1 {
			t.Errorf("study %s yielded %d times, want once", uid, n)
		}
	}
}

// parseTestRange parses a "bytes=start-end" header into inclusive 0-based offsets for the test
// origin, clamping to the value length. It handles the closed-range form the byte-range test uses.
func parseTestRange(h string, length int) (int, int) {
	h = strings.TrimPrefix(h, "bytes=")
	parts := strings.SplitN(h, "-", 2)
	start := 0
	end := length - 1
	if parts[0] != "" {
		_, _ = fmt.Sscanf(parts[0], "%d", &start)
	}
	if len(parts) == 2 && parts[1] != "" {
		_, _ = fmt.Sscanf(parts[1], "%d", &end)
	}
	if end >= length {
		end = length - 1
	}
	return start, end
}
