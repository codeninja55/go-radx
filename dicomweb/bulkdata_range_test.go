package dicomweb

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// parseTestRange parses a bytes=start-end (or bytes=start-) header for the test origin.
func parseTestRange(header string, size int) (start, end int, ok bool) {
	spec, found := strings.CutPrefix(header, "bytes=")
	if !found {
		return 0, 0, false
	}
	lo, hi, found := strings.Cut(spec, "-")
	if !found {
		return 0, 0, false
	}
	start, err := strconv.Atoi(lo)
	if err != nil {
		return 0, 0, false
	}
	if hi == "" {
		return start, size - 1, true
	}
	end, err = strconv.Atoi(hi)
	if err != nil {
		return 0, 0, false
	}
	if end >= size {
		end = size - 1
	}
	return start, end, true
}

// rangeOrigin is an httptest handler serving one bulk-data value as a multipart/related
// octet-stream body, honouring a bytes=start-end Range header with a 206 when honour is
// true and ignoring it (200, full value) otherwise. It records the Range header it saw.
func rangeOrigin(t *testing.T, value []byte, honour bool, sawRange *string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		*sawRange = r.Header.Get("Range")
		body := value
		status := http.StatusOK
		if honour && *sawRange != "" {
			if start, end, ok := parseTestRange(*sawRange, len(value)); ok {
				body = value[start : end+1]
				status = http.StatusPartialContent
			}
		}
		var buf bytes.Buffer
		mw := NewMultipartWriter(&buf, mediaTypeOctet)
		if err := mw.AddPart(mediaTypeOctet, bytes.NewReader(body)); err != nil {
			t.Errorf("AddPart: %v", err)
		}
		if _, err := mw.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
		w.Header().Set("Content-Type", mw.ContentType())
		w.WriteHeader(status)
		_, _ = w.Write(buf.Bytes())
	}
}

// TestResolveBulkDataURIRangePartialContent asserts the Range header is sent in the
// bytes=start-end form and a 206 answer's octets are returned.
func TestResolveBulkDataURIRangePartialContent(t *testing.T) {
	value := []byte("0123456789")
	var sawRange string
	hs := httptest.NewServer(rangeOrigin(t, value, true, &sawRange))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.ResolveBulkDataURIRange(context.Background(),
		BulkDataURI(hs.URL+"/studies/1/series/2/instances/3/bulkdata/7FE00010"),
		ByteRange{Start: 2, End: i64(5)})
	if err != nil {
		t.Fatalf("ResolveBulkDataURIRange: %v", err)
	}
	if sawRange != "bytes=2-5" {
		t.Fatalf("origin saw Range = %q, want bytes=2-5", sawRange)
	}
	if !bytes.Equal(got, []byte("2345")) {
		t.Fatalf("octets = %q, want the 2-5 slice", got)
	}
}

// TestResolveBulkDataURIRangeOpenEnded asserts an open-ended range renders as
// bytes=start- and returns the tail of the value.
func TestResolveBulkDataURIRangeOpenEnded(t *testing.T) {
	value := []byte("0123456789")
	var sawRange string
	hs := httptest.NewServer(rangeOrigin(t, value, true, &sawRange))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.ResolveBulkDataURIRange(context.Background(),
		BulkDataURI(hs.URL+"/bulk"), ByteRange{Start: 7})
	if err != nil {
		t.Fatalf("ResolveBulkDataURIRange: %v", err)
	}
	if sawRange != "bytes=7-" {
		t.Fatalf("origin saw Range = %q, want bytes=7-", sawRange)
	}
	if !bytes.Equal(got, []byte("789")) {
		t.Fatalf("octets = %q, want the tail from 7", got)
	}
}

// TestResolveBulkDataURIRangeFullBodyFallback asserts an origin that ignores Range and
// answers 200 with the whole value is tolerated: the full octets are returned, matching
// dicomweb-client's byte_range semantics.
func TestResolveBulkDataURIRangeFullBodyFallback(t *testing.T) {
	value := []byte("0123456789")
	var sawRange string
	hs := httptest.NewServer(rangeOrigin(t, value, false, &sawRange))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.ResolveBulkDataURIRange(context.Background(),
		BulkDataURI(hs.URL+"/bulk"), ByteRange{Start: 2, End: i64(5)})
	if err != nil {
		t.Fatalf("ResolveBulkDataURIRange: %v", err)
	}
	if sawRange != "bytes=2-5" {
		t.Fatalf("origin saw Range = %q, want bytes=2-5", sawRange)
	}
	if !bytes.Equal(got, value) {
		t.Fatalf("octets = %q, want the full value from the 200 answer", got)
	}
}

// TestResolveBulkDataURIRangeRawPartialBody asserts a 206 whose body is raw
// application/octet-stream (not multipart-framed, the form some origins serve a single
// range in) yields the raw octets.
func TestResolveBulkDataURIRangeRawPartialBody(t *testing.T) {
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", mediaTypeOctet)
		w.Header().Set("Content-Range", "bytes 2-5/10")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("2345"))
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.ResolveBulkDataURIRange(context.Background(),
		BulkDataURI(hs.URL+"/bulk"), ByteRange{Start: 2, End: i64(5)})
	if err != nil {
		t.Fatalf("ResolveBulkDataURIRange: %v", err)
	}
	if !bytes.Equal(got, []byte("2345")) {
		t.Fatalf("octets = %q, want the raw 206 body", got)
	}
}

// TestResolveBulkDataURIRangeRejectsInvalidRange asserts a malformed range is rejected
// before any request reaches the wire.
func TestResolveBulkDataURIRangeRejectsInvalidRange(t *testing.T) {
	requested := false
	hs := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requested = true
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	for _, br := range []ByteRange{{Start: -1}, {Start: 5, End: i64(2)}, {Start: 0, End: i64(-3)}} {
		if _, err := c.ResolveBulkDataURIRange(context.Background(), BulkDataURI(hs.URL+"/bulk"), br); !errors.Is(err, ErrInvalidResource) {
			t.Fatalf("ResolveBulkDataURIRange(%+v) = %v, want ErrInvalidResource", br, err)
		}
	}
	if requested {
		t.Fatal("an invalid range reached the wire")
	}
}

// TestResolveBulkDataURIWithoutRangeStaysRangeless asserts the plain resolver never sends
// a Range header and refuses a stray 206, so an origin cannot silently hand back less
// than the whole value.
func TestResolveBulkDataURIWithoutRangeStaysRangeless(t *testing.T) {
	var sawRange string
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawRange = r.Header.Get("Range")
		var buf bytes.Buffer
		mw := NewMultipartWriter(&buf, mediaTypeOctet)
		_ = mw.AddPart(mediaTypeOctet, bytes.NewReader([]byte("23")))
		_, _ = mw.Close()
		w.Header().Set("Content-Type", mw.ContentType())
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(buf.Bytes())
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.ResolveBulkDataURI(context.Background(), BulkDataURI(hs.URL+"/bulk")); err == nil {
		t.Fatal("ResolveBulkDataURI accepted an unrequested 206 partial answer")
	}
	if sawRange != "" {
		t.Fatalf("plain ResolveBulkDataURI sent Range = %q, want none", sawRange)
	}
}

// i64 builds the pointer form ByteRange.End takes.
func i64(v int64) *int64 { return &v }

// TestByteRangeHeaderForms asserts the rendered Range header for the three shapes the
// API expresses: a closed range (including the single first byte, bytes=0-0), and the
// open-ended tail.
func TestByteRangeHeaderForms(t *testing.T) {
	cases := []struct {
		br   ByteRange
		want string
	}{
		{ByteRange{Start: 0, End: i64(0)}, "bytes=0-0"},
		{ByteRange{Start: 2, End: i64(5)}, "bytes=2-5"},
		{ByteRange{Start: 7}, "bytes=7-"},
	}
	for _, tc := range cases {
		got, err := tc.br.header()
		if err != nil {
			t.Fatalf("header(%+v): %v", tc.br, err)
		}
		if got != tc.want {
			t.Fatalf("header(%+v) = %q, want %q", tc.br, got, tc.want)
		}
	}
}

// TestResolveBulkDataURIRangeRawFullBody asserts a rangeful request answered 200 with a
// raw octet-stream body (an origin that ignores Range and skips the multipart framing)
// yields the full value, per the reference client's tolerance of a full-body answer.
func TestResolveBulkDataURIRangeRawFullBody(t *testing.T) {
	value := []byte("0123456789")
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", mediaTypeOctet)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(value)
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.ResolveBulkDataURIRange(context.Background(),
		BulkDataURI(hs.URL+"/bulk"), ByteRange{Start: 2, End: i64(5)})
	if err != nil {
		t.Fatalf("ResolveBulkDataURIRange: %v", err)
	}
	if !bytes.Equal(got, value) {
		t.Fatalf("octets = %q, want the full raw value", got)
	}
}
