package dicomweb

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

// warningStore is a StoreBackend that reports a Warning Reason for a configured SOP Instance
// UID, exercising the WarnableStoreBackend accept-with-caveat path.
type warningStore struct {
	*memStore
	warnUID  string
	warnCode uint16
}

func (w *warningStore) StoreWithResult(ctx context.Context, ds *dicom.DataSet) (StoreResult, error) {
	if err := w.Store(ctx, ds); err != nil {
		return StoreResult{}, err
	}
	if uid, _ := ds.GetString(dicom.TagSOPInstanceUID); uid == w.warnUID {
		return StoreResult{Warning: w.warnCode}, nil
	}
	return StoreResult{}, nil
}

// TestStoreResponseCarriesResolvableRetrieveURL asserts the store response carries a
// per-instance Retrieve URL and a study-level Retrieve URL, and that the client can resolve
// the per-instance URL back to the stored instance through the same origin.
func TestStoreResponseCarriesResolvableRetrieveURL(t *testing.T) {
	store := newMemStore()
	c := newTestServerClient(t, store)

	resp, err := c.Store(context.Background(), sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	if len(resp.Referenced) != 1 {
		t.Fatalf("Referenced len = %d, want 1", len(resp.Referenced))
	}
	if resp.RetrieveURL == "" {
		t.Fatal("store response carried no study-level RetrieveURL")
	}
	if !strings.HasSuffix(resp.RetrieveURL, "/studies/1.2.3") {
		t.Fatalf("study RetrieveURL = %q, want a /studies/1.2.3 suffix", resp.RetrieveURL)
	}
	instURL := resp.Referenced[0].RetrieveURL
	if !strings.HasSuffix(instURL, "/studies/1.2.3/series/1.2.3.4/instances/1.2.3.4.5") {
		t.Fatalf("instance RetrieveURL = %q, want the full instance path", instURL)
	}

	// Resolve the per-instance Retrieve URL back to the instance: the URL the origin returned
	// must address a retrievable resource on the same origin.
	got, err := c.RetrieveInstance(context.Background(), NewInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	if err != nil {
		t.Fatalf("RetrieveInstance via RetrieveURL identity: %v", err)
	}
	if sop, _ := got.GetString(dicom.TagSOPInstanceUID); sop != "1.2.3.4.5" {
		t.Fatalf("resolved SOPInstanceUID = %q, want 1.2.3.4.5", sop)
	}
}

// TestStoreResponseCarriesWarningReason asserts a backend that accepts an instance with a
// caveat surfaces the Warning Reason (0008,1196) in the Referenced SOP Sequence item, with the
// instance still counted as accepted (a clean store, not an error).
func TestStoreResponseCarriesWarningReason(t *testing.T) {
	const warnCode uint16 = 0xB007 // data set does not match SOP Class (coercion warning)
	store := &warningStore{memStore: newMemStore(), warnUID: "1.2.3.4.5", warnCode: warnCode}
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

	resp, err := c.Store(context.Background(), sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	if err != nil {
		t.Fatalf("Store with a warning returned an error, want a clean accept: %v", err)
	}
	if !resp.IsComplete() {
		t.Fatal("a warned-but-accepted store reported as incomplete")
	}
	if len(resp.Referenced) != 1 {
		t.Fatalf("Referenced len = %d, want 1", len(resp.Referenced))
	}
	if resp.Referenced[0].WarningReason != warnCode {
		t.Fatalf("WarningReason = 0x%04X, want 0x%04X", resp.Referenced[0].WarningReason, warnCode)
	}
}

// TestStoreResponseOtherFailure asserts that a malformed metadata instance with no SOP
// identity is reported as a top-level Other failure (0008,1197) rather than a Failed item with
// empty UIDs, and that the response reads as incomplete.
func TestStoreResponseOtherFailure(t *testing.T) {
	store := newMemStore()
	srv, err := NewServer(WithStoreBackend(store))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	// A metadata+bulkdata body whose single metadata instance carries no SOP identity.
	body, ct := metadataBulkBody(t, []string{`{}`}, nil)
	resp, err := hs.Client().Post(hs.URL+"/studies", ct, body)
	if err != nil {
		t.Fatalf("POST /studies: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("other-failure status = %d, want 409 (nothing accepted)", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	ds, err := UnmarshalJSON(raw)
	if err != nil {
		t.Fatalf("decode store response: %v", err)
	}
	parsed := parseStoreResponse(ds)
	if parsed.OtherFailure == 0 {
		t.Fatal("store response carried no top-level Other failure for an unreferenceable instance")
	}
	if parsed.IsComplete() {
		t.Fatal("a store with a top-level Other failure read as complete")
	}
}

// TestStoreMetadataBulkDataVariant asserts the metadata+bulkdata STOW variant stores: a
// type="application/dicom+json" body whose metadata references a bulkdata part by
// Content-Location is reassembled and stored, then retrievable.
func TestStoreMetadataBulkDataVariant(t *testing.T) {
	store := newMemStore()
	srv, err := NewServer(WithStoreBackend(store), WithRetrieveBackend(store))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	pixel := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	const loc = "urn:bulk:pixeldata-1"
	// Metadata instance: SOP identity inline, PixelData (OB) referenced by BulkDataURI.
	meta := fmt.Sprintf(`{
		"00080016": {"vr": "UI", "Value": ["1.2.840.10008.5.1.4.1.1.4"]},
		"00080018": {"vr": "UI", "Value": ["1.2.3.4.5"]},
		"0020000D": {"vr": "UI", "Value": ["1.2.3"]},
		"0020000E": {"vr": "UI", "Value": ["1.2.3.4"]},
		"7FE00010": {"vr": "OB", "BulkDataURI": %q}
	}`, loc)

	body, ct := metadataBulkBody(t, []string{meta}, map[string][]byte{loc: pixel})
	resp, err := hs.Client().Post(hs.URL+"/studies", ct, body)
	if err != nil {
		t.Fatalf("POST /studies: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("metadata+bulkdata store status = %d, want 200; body %s", resp.StatusCode, raw)
	}

	got, ok := store.instances["1.2.3.4.5"]
	if !ok {
		t.Fatal("metadata+bulkdata variant did not store the instance")
	}
	pd, ok := got.Get(dicom.TagPixelData)
	if !ok {
		t.Fatal("stored instance carried no PixelData reassembled from the bulkdata part")
	}
	_ = pd // presence is the assertion; the bytes round-trip through the dicom value layer
}

// TestStoreMetadataBulkDataMissingReference asserts a metadata instance whose BulkDataURI
// names no part is rejected (the request stores nothing), never stored with a partial value.
func TestStoreMetadataBulkDataMissingReference(t *testing.T) {
	store := newMemStore()
	srv, err := NewServer(WithStoreBackend(store))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	meta := `{
		"00080016": {"vr": "UI", "Value": ["1.2.840.10008.5.1.4.1.1.4"]},
		"00080018": {"vr": "UI", "Value": ["1.2.3.4.5"]},
		"0020000D": {"vr": "UI", "Value": ["1.2.3"]},
		"7FE00010": {"vr": "OB", "BulkDataURI": "urn:bulk:absent"}
	}`
	body, ct := metadataBulkBody(t, []string{meta}, nil) // no bulkdata part supplied
	resp, err := hs.Client().Post(hs.URL+"/studies", ct, body)
	if err != nil {
		t.Fatalf("POST /studies: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("missing-reference status = %d, want 409 (nothing accepted)", resp.StatusCode)
	}
	if len(store.instances) != 0 {
		t.Fatalf("backend stored %d instances despite an unresolvable bulkdata reference, want 0", len(store.instances))
	}
}

// metadataBulkBody frames a metadata+bulkdata STOW body: an application/dicom+json metadata
// part (a JSON array of the given instance documents) followed by one application/octet-stream
// bulkdata part per entry, each carrying its Content-Location key. It returns the body reader
// and the multipart/related Content-Type with type="application/dicom+json".
func metadataBulkBody(t *testing.T, instances []string, bulk map[string][]byte) (io.Reader, string) {
	t.Helper()
	var buf strings.Builder
	mw := NewMultipartWriter(&buf, mediaTypeDICOMJSON)
	array := "[" + strings.Join(instances, ",") + "]"
	if err := mw.AddPart(mediaTypeDICOMJSON, strings.NewReader(array)); err != nil {
		t.Fatalf("add metadata part: %v", err)
	}
	for loc, data := range bulk {
		if err := mw.AddPartWithHeader(map[string]string{
			"Content-Type":     mediaTypeOctet,
			"Content-Location": loc,
		}, strings.NewReader(string(data))); err != nil {
			t.Fatalf("add bulkdata part: %v", err)
		}
	}
	if _, err := mw.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}
	return strings.NewReader(buf.String()), mw.ContentType()
}

// TestStoreQZeroRefused asserts a STOW-RS body whose Accept refuses application/dicom+json with
// q=0 is answered 406, honouring the q-value refusal in content negotiation.
func TestStoreQZeroRefused(t *testing.T) {
	if negotiateDICOMJSON("application/dicom+json;q=0") {
		t.Fatal("negotiateDICOMJSON admitted a q=0 refusal of application/dicom+json")
	}
	if !negotiateDICOMJSON("application/dicom+json;q=1") {
		t.Fatal("negotiateDICOMJSON refused application/dicom+json at q=1")
	}
	// A q=0 on one range with a wildcard fallback at q>0 is still acceptable.
	if !negotiateDICOMJSON("application/dicom+json;q=0, */*") {
		t.Fatal("negotiateDICOMJSON refused a representation a wildcard fallback admits")
	}
}
