package dicomweb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

// memStore is an in-memory StoreBackend/RetrieveBackend for the in-process round-trip
// tests. It keys instances by SOP Instance UID.
type memStore struct {
	mu        sync.Mutex
	instances map[string]*dicom.DataSet
	failUID   string // a SOP Instance UID this backend rejects, to exercise fail-closed
	failCode  uint16
}

func newMemStore() *memStore {
	return &memStore{instances: make(map[string]*dicom.DataSet)}
}

func (m *memStore) Store(_ context.Context, ds *dicom.DataSet) error {
	uid, _ := ds.GetString(dicom.TagSOPInstanceUID)
	if m.failUID != "" && uid == m.failUID {
		return &FailureReasonError{Reason: m.failCode, Msg: "configured to fail"}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.instances[uid] = ds.Clone()
	return nil
}

func (m *memStore) RetrieveInstance(_ context.Context, p ResourcePath) (*dicom.DataSet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ds, ok := m.instances[string(p.Instance)]
	if !ok {
		return nil, ErrNotFound
	}
	return ds, nil
}

// sampleInstance builds a minimal storable instance with the identity tags STOW/WADO
// reference, plus a patient-name element so the round-trip has content to compare.
func sampleInstance(study, series, sop string) *dicom.DataSet {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.4") // MR Image Storage
	ds.SetString(dicom.TagSOPInstanceUID, sop)
	ds.SetString(dicom.TagStudyInstanceUID, study)
	ds.SetString(dicom.TagSeriesInstanceUID, series)
	ds.Set(dicom.Element{Tag: dicom.TagPatientName, VR: dicom.VRPN, Value: dicom.NewStrings(dicom.VRPN, "Doe^Jane")})
	return ds
}

func newTestServerClient(t *testing.T, store *memStore) *Client {
	t.Helper()
	srv, err := NewServer(WithStoreBackend(store), WithRetrieveBackend(store))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestStoreThenComplete(t *testing.T) {
	store := newMemStore()
	c := newTestServerClient(t, store)

	ds := sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5")
	resp, err := c.Store(context.Background(), ds)
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	if !resp.IsComplete() {
		t.Fatalf("Store response not complete: %+v", resp)
	}
	if len(resp.Referenced) != 1 {
		t.Fatalf("Referenced len = %d, want 1", len(resp.Referenced))
	}
	if resp.Referenced[0].SOPInstanceUID != "1.2.3.4.5" {
		t.Fatalf("Referenced SOPInstanceUID = %q", resp.Referenced[0].SOPInstanceUID)
	}
}

// TestStoreFailClosed is the named fail-closed regression: a backend that rejects one
// instance yields a non-nil error AND a StoreResponse with a populated Failed list, so
// the partial failure is never silently dropped (PRD §9.2).
func TestStoreFailClosed(t *testing.T) {
	store := newMemStore()
	store.failUID = "1.2.3.4.6"
	store.failCode = 0xA700
	c := newTestServerClient(t, store)

	ok := sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5")
	bad := sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.6")

	resp, err := c.Store(context.Background(), ok, bad)
	if err == nil {
		t.Fatal("Store of a failing instance returned nil error, want a fail-closed error")
	}
	if _, ok := errors.AsType[*StoreError](err); !ok {
		t.Fatalf("Store error = %v, want *StoreError", err)
	}
	if resp == nil {
		t.Fatal("Store returned a nil response alongside the error; the partial result was dropped")
	}
	if len(resp.Failed) != 1 {
		t.Fatalf("Failed len = %d, want 1", len(resp.Failed))
	}
	if resp.Failed[0].FailureReason != 0xA700 {
		t.Fatalf("Failed FailureReason = 0x%04X, want 0xA700", resp.Failed[0].FailureReason)
	}
	if resp.IsComplete() {
		t.Fatal("IsComplete() = true on a partial store")
	}
}

// TestRetrieveInstanceRoundTrip stores an instance and WADO-RS retrieves it, asserting
// the dataset matches.
func TestRetrieveInstanceRoundTrip(t *testing.T) {
	store := newMemStore()
	c := newTestServerClient(t, store)

	orig := sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5")
	if _, err := c.Store(context.Background(), orig); err != nil {
		t.Fatalf("Store: %v", err)
	}

	got, err := c.RetrieveInstance(context.Background(), NewInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	if err != nil {
		t.Fatalf("RetrieveInstance: %v", err)
	}
	gotName, _ := got.GetString(dicom.TagPatientName)
	if gotName != "Doe^Jane" {
		t.Fatalf("retrieved PatientName = %q, want Doe^Jane", gotName)
	}
	gotSOP, _ := got.GetString(dicom.TagSOPInstanceUID)
	if gotSOP != "1.2.3.4.5" {
		t.Fatalf("retrieved SOPInstanceUID = %q, want 1.2.3.4.5", gotSOP)
	}
}

// TestQIDORouteReachable is the named regression for the QIDO-RS wiring: with a query
// backend registered, a /studies search is routed to the query handler and returns a
// conformant application/dicom+json result, not the 501 the route answered before QIDO
// was implemented. The no-backend 501 contract is covered by
// TestQIDONotImplementedWhenNoBackend in qido_test.go.
func TestQIDORouteReachable(t *testing.T) {
	backend := &memQuery{candidates: []*dicom.DataSet{
		studyRecord("1.2.3", "12345", "Doe^Jane", "20200101", "ACC1", "CT"),
	}}
	srv, err := NewServer(WithQueryBackend(backend))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	resp, err := hs.Client().Get(hs.URL + "/studies?PatientID=12345")
	if err != nil {
		t.Fatalf("GET /studies: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("QIDO status = %d, want 200 (route reachable, one match)", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != mediaTypeDICOMJSON {
		t.Fatalf("QIDO Content-Type = %q, want %q", ct, mediaTypeDICOMJSON)
	}
	var results []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Fatalf("decode QIDO results: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("QIDO results = %d, want 1", len(results))
	}
}

// TestServerBindsLoopbackByDefault is the named regression: with no address option the
// server binds a loopback address (PRD §9.1).
func TestServerBindsLoopbackByDefault(t *testing.T) {
	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if !loopbackOnly(srv.Addr()) {
		t.Fatalf("default Addr() = %q is not loopback", srv.Addr())
	}
}

func TestServerStoreNotImplementedWhenNoBackend(t *testing.T) {
	srv, err := NewServer(WithRetrieveBackend(newMemStore()))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	resp, err := hs.Client().Post(hs.URL+"/studies", `multipart/related; type="application/dicom"; boundary=x`, http.NoBody)
	if err != nil {
		t.Fatalf("POST /studies: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("store-without-backend status = %d, want 501", resp.StatusCode)
	}
}

func TestRetrieveUnacceptableReturns406(t *testing.T) {
	store := newMemStore()
	srv, err := NewServer(WithRetrieveBackend(store))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	req, _ := http.NewRequest(http.MethodGet, hs.URL+"/studies/1.2.3/series/1.2.3.4/instances/1.2.3.4.5", http.NoBody)
	req.Header.Set("Accept", "application/dicom+xml")
	resp, err := hs.Client().Do(req)
	if err != nil {
		t.Fatalf("GET instance: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotAcceptable {
		t.Fatalf("unacceptable Accept status = %d, want 406", resp.StatusCode)
	}
}

// TestStorePartialReturns202 asserts the STOW-RS status follows PS3.18 §10.5.3: a batch
// with one accepted and one rejected instance answers 202 Accepted, not 409.
func TestStorePartialReturns202(t *testing.T) {
	store := newMemStore()
	store.failUID = "1.2.3.4.6"
	store.failCode = 0xA700
	srv, err := NewServer(WithStoreBackend(store))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	// The client returns a *StoreError on a partial store; the status assertion is on the
	// server, so drive the raw status through a direct request.
	body, ct := encodeStoreBody(t,
		sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"),
		sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.6"),
	)
	resp, err := c.httpClient.Post(hs.URL+"/studies", ct, body)
	if err != nil {
		t.Fatalf("POST /studies: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("partial store status = %d, want 202 Accepted", resp.StatusCode)
	}
}

// TestStoreAllFailedReturns409 asserts a batch in which every instance is rejected
// answers 409 Conflict.
func TestStoreAllFailedReturns409(t *testing.T) {
	store := newMemStore()
	store.failUID = "1.2.3.4.5"
	store.failCode = 0xA700
	srv, err := NewServer(WithStoreBackend(store))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	body, ct := encodeStoreBody(t, sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	resp, err := hs.Client().Post(hs.URL+"/studies", ct, body)
	if err != nil {
		t.Fatalf("POST /studies: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("all-failed store status = %d, want 409 Conflict", resp.StatusCode)
	}
}

// TestStoreRejectsStudyMismatch asserts that a POST to /studies/{study} rejects an
// instance whose StudyInstanceUID does not match the targeted study, recording it in the
// Failed SOP Sequence rather than storing it under the wrong hierarchy.
func TestStoreRejectsStudyMismatch(t *testing.T) {
	store := newMemStore()
	c := newTestServerClient(t, store)

	// Drive a /studies/{study} target directly: the high-level Store always posts to the
	// root /studies, so exercise the constrained target through a raw request.
	body, ct := encodeStoreBody(t, sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	resp, err := c.httpClient.Post(c.baseURL+"/studies/9.9.9", ct, body)
	if err != nil {
		t.Fatalf("POST /studies/9.9.9: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("study-mismatch status = %d, want 409 (nothing accepted)", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	ds, err := UnmarshalJSON(raw)
	if err != nil {
		t.Fatalf("decode store response: %v", err)
	}
	parsed := parseStoreResponse(ds)
	if len(parsed.Failed) != 1 || parsed.Failed[0].FailureReason != failureReasonNotInStudy {
		t.Fatalf("study-mismatch Failed = %+v, want one 0x%04X failure", parsed.Failed, failureReasonNotInStudy)
	}
	if len(store.instances) != 0 {
		t.Fatalf("backend stored %d instances despite study mismatch, want 0", len(store.instances))
	}
}

// encodeStoreBody frames the given instances as a multipart/related application/dicom
// body and returns the body reader and its Content-Type.
func encodeStoreBody(t *testing.T, instances ...*dicom.DataSet) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := NewMultipartWriter(&buf, mediaTypeDICOM)
	for _, ds := range instances {
		raw, err := encodeInstance(ds, defaultStorageTransferSyntax)
		if err != nil {
			t.Fatalf("encode instance: %v", err)
		}
		if err := mw.AddPart(mediaTypeDICOM, bytes.NewReader(raw)); err != nil {
			t.Fatalf("add part: %v", err)
		}
	}
	if _, err := mw.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}
	return &buf, mw.ContentType()
}

// TestStoreRejectsPartWithoutSOPIdentity asserts a STOW-RS part whose dataset omits the
// SOP Class/Instance UID is rejected (400) rather than stored or reported with empty UIDs.
func TestStoreRejectsPartWithoutSOPIdentity(t *testing.T) {
	store := newMemStore()
	srv, err := NewServer(WithStoreBackend(store))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	// Build a Part 10 object that parses but carries no SOP identity. encodeInstance would
	// reject it, so frame the raw bytes directly via dicom.Write with a meta whose UIDs are
	// set only in the meta group, not the main dataset.
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagStudyInstanceUID, "1.2.3")
	var part bytes.Buffer
	f := &dicom.File{
		Meta: &dicom.FileMeta{
			MediaStorageSOPClassUID:    "1.2.840.10008.5.1.4.1.1.4",
			MediaStorageSOPInstanceUID: "1.2.3.4.5",
			TransferSyntaxUID:          dicom.ExplicitVRLittleEndian,
		},
		DataSet: ds,
	}
	if err := dicom.Write(&part, f); err != nil {
		t.Fatalf("write part: %v", err)
	}

	var body bytes.Buffer
	mw := NewMultipartWriter(&body, mediaTypeDICOM)
	if err := mw.AddPart(mediaTypeDICOM, bytes.NewReader(part.Bytes())); err != nil {
		t.Fatalf("add part: %v", err)
	}
	if _, err := mw.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}

	resp, err := hs.Client().Post(hs.URL+"/studies", mw.ContentType(), &body)
	if err != nil {
		t.Fatalf("POST /studies: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("no-SOP-identity part status = %d, want 400", resp.StatusCode)
	}
	if len(store.instances) != 0 {
		t.Fatalf("backend stored %d instances despite missing SOP identity, want 0", len(store.instances))
	}
}

// TestStoreRejectsMalformedTargetStudy asserts a /studies/{study} target whose UID is not
// conformant is rejected as an invalid resource (400), not treated as a study mismatch.
func TestStoreRejectsMalformedTargetStudy(t *testing.T) {
	store := newMemStore()
	c := newTestServerClient(t, store)
	body, ct := encodeStoreBody(t, sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	resp, err := c.httpClient.Post(c.baseURL+"/studies/not-a-uid", ct, body)
	if err != nil {
		t.Fatalf("POST /studies/not-a-uid: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed target study status = %d, want 400", resp.StatusCode)
	}
}

func TestLoopbackOnly(t *testing.T) {
	cases := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1:8080", true},
		{"localhost:8080", true},
		{"[::1]:8080", true},
		{"0.0.0.0:8080", false},
		{"192.168.1.10:8080", false},
	}
	for _, tc := range cases {
		if got := loopbackOnly(tc.addr); got != tc.want {
			t.Errorf("loopbackOnly(%q) = %v, want %v", tc.addr, got, tc.want)
		}
	}
}
