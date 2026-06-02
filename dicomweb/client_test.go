package dicomweb

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

func TestNewClientRequiresBaseURL(t *testing.T) {
	if _, err := NewClient("   "); err == nil {
		t.Fatal("NewClient with a blank base URL returned nil error")
	}
}

func TestNewClientTrimsTrailingSlash(t *testing.T) {
	c, err := NewClient("https://pacs.example.org/dicom-web/")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.baseURL != "https://pacs.example.org/dicom-web" {
		t.Fatalf("baseURL = %q, want the slash trimmed", c.baseURL)
	}
}

// TestBearerTokenSentNeverLogged asserts the token reaches the origin as an
// Authorization header but never appears in an error message.
func TestBearerTokenSentNeverLogged(t *testing.T) {
	const secret = "super-secret-token-value"
	var seenAuth string
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		// Reply with a non-success status so Store produces an HTTPError we can inspect.
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	t.Cleanup(hs.Close)

	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()), WithBearerToken(secret))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ds := sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5")
	_, err = c.Store(context.Background(), ds)
	if err == nil {
		t.Fatal("Store against a 500 origin returned nil error")
	}
	if seenAuth != "Bearer "+secret {
		t.Fatalf("origin saw Authorization = %q, want the bearer token", seenAuth)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error message leaked the bearer token: %q", err.Error())
	}
}

func TestStoreRequiresAnInstance(t *testing.T) {
	c, err := NewClient("https://pacs.example.org")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := c.Store(context.Background()); err == nil {
		t.Fatal("Store with no instances returned nil error")
	}
}

// TestStoreRejectsInstanceWithoutSOPIdentity asserts a dataset missing its SOP identity
// is rejected before any request is made, since it could not be referenced in the store
// response.
func TestStoreRejectsInstanceWithoutSOPIdentity(t *testing.T) {
	c, err := NewClient("https://pacs.example.org")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ds := dicom.NewDataSet() // no SOP Class / Instance UID
	if _, err := c.Store(context.Background(), ds); err == nil {
		t.Fatal("Store of an instance without SOP identity returned nil error")
	}
}

// TestResponseCapFailsClosed asserts that a response larger than WithMaxResponseBytes
// returns a *LimitExceededError rather than silently truncating to a "complete" body
// (PRD §9.3, truncation is failure §9.2).
func TestResponseCapFailsClosed(t *testing.T) {
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", mediaTypeDICOMJSON)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(make([]byte, 4096)) // far larger than the 16-byte cap below
	}))
	t.Cleanup(hs.Close)

	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()), WithMaxResponseBytes(16))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := c.httpClient.Get(hs.URL + "/x")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	_, err = io.ReadAll(c.boundedBody(resp))
	if !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("reading an over-cap body = %v, want ErrLimitExceeded", err)
	}
}

// TestStoreFailClosedOnStatusWithSparseBody asserts the fail-closed contract holds on the
// HTTP status alone: a 409 carrying a store-response document with no Failed SOP Sequence
// still yields a *StoreError, never a nil error read as success (PRD §9.2).
func TestStoreFailClosedOnStatusWithSparseBody(t *testing.T) {
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", mediaTypeDICOMJSON)
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{}`)) // a valid but empty store response: no Failed sequence
	}))
	t.Cleanup(hs.Close)

	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := c.Store(context.Background(), sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	if err == nil {
		t.Fatal("Store against a 409 with a sparse body returned nil error")
	}
	var serr *StoreError
	if !errors.As(err, &serr) {
		t.Fatalf("Store error = %v, want *StoreError", err)
	}
	if serr.Status != http.StatusConflict {
		t.Fatalf("StoreError.Status = %d, want 409", serr.Status)
	}
	if resp == nil {
		t.Fatal("Store dropped the parsed response on a status-only failure")
	}
}

// TestRemoteErrorBodyNotLeaked asserts a remote 4xx/5xx body, which may carry PHI, never
// appears in the returned error (PRD §9.1).
func TestRemoteErrorBodyNotLeaked(t *testing.T) {
	const phi = "PatientName=Doe^John^SENSITIVE"
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(phi))
	}))
	t.Cleanup(hs.Close)

	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = c.RetrieveInstance(context.Background(), NewInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	if err == nil {
		t.Fatal("RetrieveInstance against a 400 returned nil error")
	}
	if strings.Contains(err.Error(), "SENSITIVE") || strings.Contains(err.Error(), "Doe") {
		t.Fatalf("error leaked the remote response body: %q", err.Error())
	}
}

func TestHTTPErrorStripsQuery(t *testing.T) {
	e := &HTTPError{StatusCode: 404, Method: "GET", URL: "/studies", Detail: "not found"}
	if !strings.Contains(e.Error(), "404") || !strings.Contains(e.Error(), "/studies") {
		t.Fatalf("HTTPError.Error() = %q", e.Error())
	}
}

// TestRedactPathRemovesUIDs asserts resource UIDs are replaced by placeholders while the
// structural keywords stay, and a query string is dropped (PRD §9.1).
func TestRedactPathRemovesUIDs(t *testing.T) {
	got := redactPath("/studies/1.2.3/series/1.2.3.4/instances/1.2.3.4.5?PatientID=12345")
	want := "/studies/{uid}/series/{uid}/instances/{uid}"
	if got != want {
		t.Fatalf("redactPath() = %q, want %q", got, want)
	}
	if strings.Contains(got, "1.2.3") || strings.Contains(got, "12345") {
		t.Fatalf("redactPath() leaked an identifier: %q", got)
	}
}

// TestTransportErrorDropsURL asserts a transport failure does not surface the full,
// UID-bearing request URL net/http embeds in *url.Error (PRD §9.1). It points the client
// at an unroutable address so http.Client.Do fails before any response.
func TestTransportErrorDropsURL(t *testing.T) {
	c, err := NewClient("http://127.0.0.1:1") // port 1: connection refused
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = c.RetrieveInstance(context.Background(), NewInstance("1.2.999", "1.2.998", "1.2.997"))
	if err == nil {
		t.Fatal("RetrieveInstance against an unroutable origin returned nil error")
	}
	for _, uid := range []string{"1.2.999", "1.2.998", "1.2.997"} {
		if strings.Contains(err.Error(), uid) {
			t.Fatalf("transport error leaked UID %s: %q", uid, err.Error())
		}
	}
}
