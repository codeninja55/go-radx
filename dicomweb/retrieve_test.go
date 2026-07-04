package dicomweb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

// wadoStore is an in-memory WADO-RS retrieval backend implementing every optional retrieve
// interface, for the in-process study/series/metadata/frames/bulkdata round-trip tests. It
// keys instances by SOP Instance UID and groups them by study and series.
type wadoStore struct {
	instances map[string]RetrievedInstance
	frames    map[string][][]byte // SOP Instance UID -> ordered frame octets
	bulk      map[string][][]byte // SOP Instance UID -> ordered bulk-data octets
}

func newWADOStore() *wadoStore {
	return &wadoStore{
		instances: make(map[string]RetrievedInstance),
		frames:    make(map[string][][]byte),
		bulk:      make(map[string][][]byte),
	}
}

func (m *wadoStore) put(si RetrievedInstance) {
	uid, _ := si.DataSet.GetString(dicom.TagSOPInstanceUID)
	m.instances[uid] = si
}

func (m *wadoStore) RetrieveInstance(_ context.Context, p ResourcePath) (*dicom.DataSet, error) {
	si, ok := m.instances[string(p.Instance)]
	if !ok {
		return nil, ErrNotFound
	}
	return si.DataSet, nil
}

func (m *wadoStore) RetrieveStoredInstance(_ context.Context, p ResourcePath) (RetrievedInstance, error) {
	si, ok := m.instances[string(p.Instance)]
	if !ok {
		return RetrievedInstance{}, ErrNotFound
	}
	return si, nil
}

func (m *wadoStore) RetrieveStudy(_ context.Context, study dicom.UID) ([]RetrievedInstance, error) {
	var out []RetrievedInstance
	for _, si := range m.instances {
		if v, _ := si.DataSet.GetString(dicom.TagStudyInstanceUID); v == string(study) {
			out = append(out, si)
		}
	}
	return out, nil
}

func (m *wadoStore) RetrieveSeries(_ context.Context, study, series dicom.UID) ([]RetrievedInstance, error) {
	var out []RetrievedInstance
	for _, si := range m.instances {
		st, _ := si.DataSet.GetString(dicom.TagStudyInstanceUID)
		se, _ := si.DataSet.GetString(dicom.TagSeriesInstanceUID)
		if st == string(study) && se == string(series) {
			out = append(out, si)
		}
	}
	return out, nil
}

func (m *wadoStore) RetrieveMetadata(_ context.Context, p ResourcePath) ([]RetrievedInstance, error) {
	switch p.Level() {
	case LevelInstance:
		si, ok := m.instances[string(p.Instance)]
		if !ok {
			return nil, ErrNotFound
		}
		return []RetrievedInstance{si}, nil
	case LevelSeries:
		return m.RetrieveSeries(context.Background(), p.Study, p.Series)
	default:
		return m.RetrieveStudy(context.Background(), p.Study)
	}
}

func (m *wadoStore) RetrieveFrames(_ context.Context, p ResourcePath, frames []int) ([]BulkDataObject, error) {
	all, ok := m.frames[string(p.Instance)]
	if !ok {
		return nil, ErrNotFound
	}
	out := make([]BulkDataObject, 0, len(frames))
	for _, f := range frames {
		if f < 1 || f > len(all) {
			return nil, fmt.Errorf("%w: frame out of range", ErrNotFound)
		}
		out = append(out, BulkDataObject{Data: all[f-1]})
	}
	return out, nil
}

func (m *wadoStore) RetrieveBulkData(_ context.Context, p ResourcePath) ([]BulkDataObject, error) {
	all, ok := m.bulk[string(p.Instance)]
	if !ok {
		return nil, ErrNotFound
	}
	out := make([]BulkDataObject, 0, len(all))
	for _, b := range all {
		out = append(out, BulkDataObject{Data: b})
	}
	return out, nil
}

func newWADOServerClient(t *testing.T, store *wadoStore) *Client {
	t.Helper()
	srv, err := NewServer(WithRetrieveBackend(store))
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

// TestRetrieveStudyStreamsInstances stores two instances of one study and asserts
// RetrieveStudy streams both back as decoded datasets.
func TestRetrieveStudyStreamsInstances(t *testing.T) {
	store := newWADOStore()
	store.put(RetrievedInstance{DataSet: sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5")})
	store.put(RetrievedInstance{DataSet: sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.6")})
	c := newWADOServerClient(t, store)

	seen := make(map[string]bool)
	for ds, err := range c.RetrieveStudy(context.Background(), "1.2.3") {
		if err != nil {
			t.Fatalf("RetrieveStudy yielded error: %v", err)
		}
		uid, _ := ds.GetString(dicom.TagSOPInstanceUID)
		seen[uid] = true
	}
	if !seen["1.2.3.4.5"] || !seen["1.2.3.4.6"] {
		t.Fatalf("RetrieveStudy returned %v, want both instances", seen)
	}
}

// TestRetrieveSeriesStreamsInstances asserts series retrieval scopes to one series.
func TestRetrieveSeriesStreamsInstances(t *testing.T) {
	store := newWADOStore()
	store.put(RetrievedInstance{DataSet: sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5")})
	store.put(RetrievedInstance{DataSet: sampleInstance("1.2.3", "9.9.9", "1.2.3.4.7")})
	c := newWADOServerClient(t, store)

	var count int
	for ds, err := range c.RetrieveSeries(context.Background(), "1.2.3", "1.2.3.4") {
		if err != nil {
			t.Fatalf("RetrieveSeries yielded error: %v", err)
		}
		uid, _ := ds.GetString(dicom.TagSOPInstanceUID)
		if uid != "1.2.3.4.5" {
			t.Fatalf("RetrieveSeries returned instance %q outside the series", uid)
		}
		count++
	}
	if count != 1 {
		t.Fatalf("RetrieveSeries returned %d instances, want 1", count)
	}
}

// TestDecodeRetrievedInstanceCaptureBytes asserts the two decode paths the dataset-only and
// object-returning retrieves use: with captureBytes false (the dataset-only path) the decoder streams
// the part and allocates NO encoded buffer (Encoded stays nil), so a large study does not pay the
// doubled per-instance memory of buffering bytes the caller discards; with captureBytes true (the
// object-returning path) Encoded holds the byte-exact Part 10 object and TransferSyntax reports the
// origin's syntax. Both yield the same decoded dataset.
func TestDecodeRetrievedInstanceCaptureBytes(t *testing.T) {
	const sop = "1.2.3.4.5"
	raw, err := encodeInstance(sampleInstance("1.2.3", "1.2.3.4", sop), dicom.ExplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("encodeInstance: %v", err)
	}

	// Dataset-only path: no encoded bytes captured.
	plain, err := decodeRetrievedInstance(bytes.NewReader(raw), false)
	if err != nil {
		t.Fatalf("decodeRetrievedInstance(captureBytes=false): %v", err)
	}
	if plain.Encoded != nil {
		t.Errorf("dataset-only decode captured %d encoded bytes; want nil (no buffer allocated)", len(plain.Encoded))
	}
	if uid, _ := plain.DataSet.GetString(dicom.TagSOPInstanceUID); uid != sop {
		t.Errorf("dataset-only decode SOPInstanceUID = %q, want %q", uid, sop)
	}

	// Object-returning path: byte-exact representation and transfer syntax preserved.
	obj, err := decodeRetrievedInstance(bytes.NewReader(raw), true)
	if err != nil {
		t.Fatalf("decodeRetrievedInstance(captureBytes=true): %v", err)
	}
	if !bytes.Equal(obj.Encoded, raw) {
		t.Errorf("object decode captured %d encoded bytes, want the %d-byte source", len(obj.Encoded), len(raw))
	}
	if obj.TransferSyntax != dicom.ExplicitVRLittleEndian {
		t.Errorf("object decode transfer syntax = %q, want %q", obj.TransferSyntax, dicom.ExplicitVRLittleEndian)
	}
	if uid, _ := obj.DataSet.GetString(dicom.TagSOPInstanceUID); uid != sop {
		t.Errorf("object decode SOPInstanceUID = %q, want %q", uid, sop)
	}
}

// TestRetrieveInstanceObjectPreservesTransferSyntax asserts the client's object-retrieve path keeps
// the instance's transfer syntax (and exact bytes) rather than discarding them, so a caller can write
// the object back in the origin's syntax instead of transcoding. The instance is stored in Implicit
// VR LE; the retrieved object must report that syntax and a byte-exact Part 10 representation.
func TestRetrieveInstanceObjectPreservesTransferSyntax(t *testing.T) {
	store := newWADOStore()
	store.put(RetrievedInstance{
		DataSet:        sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"),
		TransferSyntax: dicom.ImplicitVRLittleEndian,
	})
	c := newWADOServerClient(t, store)

	si, err := c.RetrieveInstanceObject(context.Background(), NewInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	if err != nil {
		t.Fatalf("RetrieveInstanceObject: %v", err)
	}
	if si.TransferSyntax != dicom.ImplicitVRLittleEndian {
		t.Errorf("retrieved transfer syntax = %q, want %q (origin's syntax, not forced Explicit VR LE)",
			si.TransferSyntax, dicom.ImplicitVRLittleEndian)
	}
	// The captured bytes must decode back to a Part 10 object whose file meta carries the same syntax.
	f, err := dicom.Read(bytes.NewReader(si.Encoded))
	if err != nil {
		t.Fatalf("captured bytes do not parse as Part 10: %v", err)
	}
	if f.Meta.TransferSyntaxUID != dicom.ImplicitVRLittleEndian {
		t.Errorf("captured bytes' file-meta transfer syntax = %q, want %q",
			f.Meta.TransferSyntaxUID, dicom.ImplicitVRLittleEndian)
	}
}

// TestRetrieveInstanceObjectCompressedFailsClosed asserts an encapsulated (compressed) instance is a
// fail-closed parse error on retrieve, not a silent transcode: go-radx reads only the four
// uncompressed syntaxes, so a compressed object cannot be decoded faithfully. The honest outcome is a
// typed parse error (the caller surfaces exit 3), never a corrupted Explicit VR LE file. The stored
// bytes here are a passthrough placeholder; the object-retrieve path decodes the part, which the
// reader rejects for an encapsulated syntax.
func TestRetrieveInstanceObjectCompressedFailsClosed(t *testing.T) {
	const sop = "1.2.3.4.5"
	store := newWADOStore()
	store.put(RetrievedInstance{
		DataSet:        sampleInstance("1.2.3", "1.2.3.4", sop),
		TransferSyntax: dicom.JPEGBaseline8Bit,
		Encoded:        bytes.Repeat([]byte{0x4A, 0x50, 0x45, 0x47}, 32),
	})
	c := newWADOServerClient(t, store)

	if _, err := c.RetrieveInstanceObject(context.Background(), NewInstance("1.2.3", "1.2.3.4", sop)); err == nil {
		t.Fatal("RetrieveInstanceObject of a compressed instance returned nil error; want a fail-closed parse error, never a silent transcode")
	}
}

// TestRetrieveStudyNotFoundIs404 asserts an empty study answers 404 (the iterator yields a
// single HTTPError), never a silent empty stream read as a complete study.
func TestRetrieveStudyNotFoundIs404(t *testing.T) {
	store := newWADOStore()
	c := newWADOServerClient(t, store)

	var got error
	for _, err := range c.RetrieveStudy(context.Background(), "9.9.9") {
		got = err
		break
	}
	he, ok := errors.AsType[*HTTPError](got)
	if !ok || he.StatusCode != http.StatusNotFound {
		t.Fatalf("empty-study retrieve error = %v, want HTTPError 404", got)
	}
}

// TestRetrieveMetadataRoundTrip asserts instance metadata round-trips: the returned dataset
// carries the identity and patient-name attributes.
func TestRetrieveMetadataRoundTrip(t *testing.T) {
	store := newWADOStore()
	store.put(RetrievedInstance{DataSet: sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5")})
	c := newWADOServerClient(t, store)

	metas, err := c.RetrieveMetadata(context.Background(), NewInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	if err != nil {
		t.Fatalf("RetrieveMetadata: %v", err)
	}
	if len(metas) != 1 {
		t.Fatalf("RetrieveMetadata returned %d objects, want 1", len(metas))
	}
	name, _ := metas[0].GetString(dicom.TagPatientName)
	if name != "Doe^Jane" {
		t.Fatalf("metadata PatientName = %q, want Doe^Jane", name)
	}
}

// TestRetrieveMetadataBulkDataResolvable is the acceptance regression: an instance with a
// pixel-data value emits a BulkDataURI in its metadata that the client resolves to the
// original octets through the same origin's bulkdata sub-resource.
func TestRetrieveMetadataBulkDataResolvable(t *testing.T) {
	const sop = "1.2.3.4.5"
	pixels := bytes.Repeat([]byte{0xAB}, 64)

	ds := sampleInstance("1.2.3", "1.2.3.4", sop)
	ds.Set(dicom.Element{Tag: dicom.TagPixelData, VR: dicom.VROW, Value: dicom.NewBytes(dicom.VROW, pixels)})

	store := newWADOStore()
	store.put(RetrievedInstance{DataSet: ds})
	store.bulk[sop] = [][]byte{pixels}
	c := newWADOServerClient(t, store)

	metas, err := c.RetrieveMetadata(context.Background(), NewInstance("1.2.3", "1.2.3.4", sop))
	if err != nil {
		t.Fatalf("RetrieveMetadata: %v", err)
	}
	uris := BulkDataURIs(metas[0])
	if len(uris) == 0 {
		t.Fatal("metadata carried no BulkDataURI for the pixel data")
	}

	resolved, err := c.ResolveBulkDataURI(context.Background(), uris[0])
	if err != nil {
		t.Fatalf("ResolveBulkDataURI: %v", err)
	}
	if !bytes.Equal(resolved, pixels) {
		t.Fatalf("resolved bulk data (%d bytes) does not match the stored pixels (%d bytes)", len(resolved), len(pixels))
	}
}

// TestRetrieveFramesRoundTrip stores three frames and asserts the requested 1-based frames
// come back in request order.
func TestRetrieveFramesRoundTrip(t *testing.T) {
	const sop = "1.2.3.4.5"
	store := newWADOStore()
	store.put(RetrievedInstance{DataSet: sampleInstance("1.2.3", "1.2.3.4", sop)})
	store.frames[sop] = [][]byte{{0x01}, {0x02}, {0x03}}
	c := newWADOServerClient(t, store)

	frames, err := c.RetrieveFrames(context.Background(), NewInstance("1.2.3", "1.2.3.4", sop), 3, 1)
	if err != nil {
		t.Fatalf("RetrieveFrames: %v", err)
	}
	if len(frames) != 2 || frames[0][0] != 0x03 || frames[1][0] != 0x01 {
		t.Fatalf("RetrieveFrames returned %v, want frames 3 then 1", frames)
	}
}

// TestRetrieveBulkDataRoundTrip asserts the instance-level bulkdata sub-resource returns the
// stored payloads.
func TestRetrieveBulkDataRoundTrip(t *testing.T) {
	const sop = "1.2.3.4.5"
	store := newWADOStore()
	store.put(RetrievedInstance{DataSet: sampleInstance("1.2.3", "1.2.3.4", sop)})
	store.bulk[sop] = [][]byte{{0xDE, 0xAD}, {0xBE, 0xEF}}
	c := newWADOServerClient(t, store)

	objs, err := c.RetrieveBulkData(context.Background(), NewInstance("1.2.3", "1.2.3.4", sop))
	if err != nil {
		t.Fatalf("RetrieveBulkData: %v", err)
	}
	if len(objs) != 2 || !bytes.Equal(objs[0], []byte{0xDE, 0xAD}) || !bytes.Equal(objs[1], []byte{0xBE, 0xEF}) {
		t.Fatalf("RetrieveBulkData returned %v, want the two stored payloads", objs)
	}
}

// TestRetrieveBulkDataMalformedUIDIs400 is the named regression for the published WADO-RS
// guarantee that a malformed UID in the path is rejected with 400 Bad Request: a bulkdata
// request whose study/series/instance UID does not parse is rejected before the backend
// lookup, while a well-formed-UID request still reaches the backend.
func TestRetrieveBulkDataMalformedUIDIs400(t *testing.T) {
	const sop = "1.2.3.4.5"
	store := newWADOStore()
	store.put(RetrievedInstance{DataSet: sampleInstance("1.2.3", "1.2.3.4", sop)})
	store.bulk[sop] = [][]byte{{0xDE, 0xAD}}

	srv, err := NewServer(WithRetrieveBackend(store))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	get := func(path string) int {
		req, _ := http.NewRequest(http.MethodGet, hs.URL+path, http.NoBody)
		req.Header.Set("Accept", `multipart/related; type="application/octet-stream"`)
		resp, derr := hs.Client().Do(req)
		if derr != nil {
			t.Fatalf("GET %s: %v", path, derr)
		}
		defer func() { _ = resp.Body.Close() }()
		return resp.StatusCode
	}

	if got := get("/studies/not-a-uid/series/bad/instances/bad/bulkdata"); got != http.StatusBadRequest {
		t.Fatalf("malformed-UID bulkdata status = %d, want 400 (the published guarantee)", got)
	}
	if got := get("/studies/1.2.3/series/1.2.3.4/instances/" + sop + "/bulkdata"); got != http.StatusOK {
		t.Fatalf("well-formed-UID bulkdata status = %d, want 200 (request reaches the backend)", got)
	}
}

// TestRetrieveInstanceTransferSyntaxPassthrough asserts an instance stored in Explicit VR LE
// is served unchanged when the client accepts that syntax (passthrough).
func TestRetrieveInstanceTransferSyntaxPassthrough(t *testing.T) {
	store := newWADOStore()
	store.put(RetrievedInstance{
		DataSet:        sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"),
		TransferSyntax: dicom.ExplicitVRLittleEndian,
	})
	srv, err := NewServer(WithRetrieveBackend(store))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	req, _ := http.NewRequest(http.MethodGet, hs.URL+"/studies/1.2.3/series/1.2.3.4/instances/1.2.3.4.5", http.NoBody)
	req.Header.Set("Accept", acceptInstances(dicom.ExplicitVRLittleEndian))
	resp, err := hs.Client().Do(req)
	if err != nil {
		t.Fatalf("GET instance: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("passthrough retrieve status = %d, want 200", resp.StatusCode)
	}
}

// TestRetrieveInstanceUnservableSyntaxIs406 is the honest-406 regression: an instance stored
// in JPEG Baseline is requested with an Accept naming only Explicit VR LE, which the server
// cannot transcode to, so it answers 406 rather than silently re-encode (the old
// unconditional behaviour).
func TestRetrieveInstanceUnservableSyntaxIs406(t *testing.T) {
	store := newWADOStore()
	store.put(RetrievedInstance{
		DataSet:        sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"),
		TransferSyntax: dicom.JPEGBaseline8Bit,
	})
	srv, err := NewServer(WithRetrieveBackend(store))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	req, _ := http.NewRequest(http.MethodGet, hs.URL+"/studies/1.2.3/series/1.2.3.4/instances/1.2.3.4.5", http.NoBody)
	req.Header.Set("Accept", acceptInstances(dicom.ExplicitVRLittleEndian))
	resp, err := hs.Client().Do(req)
	if err != nil {
		t.Fatalf("GET instance: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotAcceptable {
		t.Fatalf("unservable-syntax retrieve status = %d, want 406", resp.StatusCode)
	}
}

// TestRetrieveInstanceCompressedPassthroughDefaultAccept is the compressed-passthrough
// regression: an instance stored in an encapsulated (compressed) syntax, retrieved with the
// default Accept (no transfer-syntax preference), is served byte-exact from its stored bytes
// rather than 500ing on a doomed dicom.Write of an encapsulated syntax. The default Accept
// selects passthrough, and go-radx writes no encapsulated syntax, so the stored bytes must be
// returned unchanged.
func TestRetrieveInstanceCompressedPassthroughDefaultAccept(t *testing.T) {
	const sop = "1.2.3.4.5"
	// A byte-exact stored object the server must echo verbatim. The bytes need not be a valid
	// JPEG stream: passthrough copies them unchanged, and the test asserts that copy is exact.
	storedBytes := bytes.Repeat([]byte{0x4A, 0x50, 0x45, 0x47}, 32)

	store := newWADOStore()
	store.put(RetrievedInstance{
		DataSet:        sampleInstance("1.2.3", "1.2.3.4", sop),
		TransferSyntax: dicom.JPEGBaseline8Bit,
		Encoded:        storedBytes,
	})
	srv, err := NewServer(WithRetrieveBackend(store))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	// Default Accept with no transfer-syntax preference: the passthrough path that previously
	// 500ed when the stored syntax was encapsulated.
	req, _ := http.NewRequest(http.MethodGet, hs.URL+"/studies/1.2.3/series/1.2.3.4/instances/"+sop, http.NoBody)
	req.Header.Set("Accept", acceptInstances())
	resp, err := hs.Client().Do(req)
	if err != nil {
		t.Fatalf("GET instance: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("compressed passthrough status = %d, want 200 (must not 500 on dicom.Write of an encapsulated syntax)", resp.StatusCode)
	}
	if !isMultipartRelated(resp.Header.Get("Content-Type")) {
		t.Fatalf("response Content-Type = %q, want multipart/related", resp.Header.Get("Content-Type"))
	}

	mr, err := NewMultipartReader(resp.Body, resp.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("NewMultipartReader: %v", err)
	}
	_, part, err := mr.NextPart()
	if err != nil {
		t.Fatalf("NextPart: %v", err)
	}
	got, err := io.ReadAll(part)
	if err != nil {
		t.Fatalf("read part: %v", err)
	}
	if !bytes.Equal(got, storedBytes) {
		t.Fatalf("passthrough body (%d bytes) is not byte-exact with the stored bytes (%d bytes)", len(got), len(storedBytes))
	}
}

// TestRetrieveInstanceCompressedNoBytesIs406 asserts the honest-406 fallback: an instance
// stored in an encapsulated syntax with no byte-exact bytes to pass through cannot be served
// (go-radx writes no encapsulated syntax), so the default-Accept passthrough answers 406 rather
// than a 500 from a doomed re-encode.
func TestRetrieveInstanceCompressedNoBytesIs406(t *testing.T) {
	const sop = "1.2.3.4.5"
	store := newWADOStore()
	store.put(RetrievedInstance{
		DataSet:        sampleInstance("1.2.3", "1.2.3.4", sop),
		TransferSyntax: dicom.JPEG2000Lossless,
	})
	srv, err := NewServer(WithRetrieveBackend(store))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	req, _ := http.NewRequest(http.MethodGet, hs.URL+"/studies/1.2.3/series/1.2.3.4/instances/"+sop, http.NoBody)
	req.Header.Set("Accept", acceptInstances())
	resp, err := hs.Client().Do(req)
	if err != nil {
		t.Fatalf("GET instance: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotAcceptable {
		t.Fatalf("encapsulated-no-bytes status = %d, want 406 (never 500 from writing an encapsulated syntax)", resp.StatusCode)
	}
}

// TestRetrieveStudyNotImplementedWhenBaseBackend asserts a backend implementing only the base
// RetrieveBackend answers 501 for study retrieval, never a silent 200 no-op.
func TestRetrieveStudyNotImplementedWhenBaseBackend(t *testing.T) {
	srv, err := NewServer(WithRetrieveBackend(baseOnlyBackend{}))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	resp, err := hs.Client().Get(hs.URL + "/studies/1.2.3")
	if err != nil {
		t.Fatalf("GET study: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("study retrieve without StudyRetriever status = %d, want 501", resp.StatusCode)
	}
}

// baseOnlyBackend implements only RetrieveBackend, exercising the optional-interface fallback.
type baseOnlyBackend struct{}

func (baseOnlyBackend) RetrieveInstance(context.Context, ResourcePath) (*dicom.DataSet, error) {
	return nil, ErrNotFound
}

// TestParseFrameListRejectsMalformed asserts a malformed frame list is rejected and the
// offending text never appears in the error (PRD §9.1).
func TestParseFrameListRejectsMalformed(t *testing.T) {
	cases := []string{"", "0", "1,bad", "-1"}
	for _, list := range cases {
		t.Run(list, func(t *testing.T) {
			if _, err := parseFrameList(list); !errors.Is(err, ErrInvalidResource) {
				t.Fatalf("parseFrameList(%q) error = %v, want ErrInvalidResource", list, err)
			}
		})
	}
	frames, err := parseFrameList("1,4,5")
	if err != nil {
		t.Fatalf("parseFrameList valid error = %v", err)
	}
	if len(frames) != 3 || frames[0] != 1 || frames[2] != 5 {
		t.Fatalf("parseFrameList(1,4,5) = %v", frames)
	}
}

// TestAbsoluteBulkDataURL asserts a relative reference is joined to the configured base while
// an absolute one is fetched as given.
func TestAbsoluteBulkDataURL(t *testing.T) {
	c, err := NewClient("https://pacs.example.org/dicom-web", WithClientBulkDataBaseURL("https://bulk.example.org/store/"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	cases := []struct {
		ref  string
		want string
	}{
		{"frames/1", "https://bulk.example.org/store/frames/1"},
		{"/studies/x/bulkdata", "https://bulk.example.org/store/studies/x/bulkdata"},
		{"https://other.example.org/b/1", "https://other.example.org/b/1"},
	}
	for _, tc := range cases {
		t.Run(tc.ref, func(t *testing.T) {
			if got := c.absoluteBulkDataURL(tc.ref); got != tc.want {
				t.Fatalf("absoluteBulkDataURL(%q) = %q, want %q", tc.ref, got, tc.want)
			}
		})
	}
}

// bulkDataEchoServer is a test origin that records the Authorization header of the request it
// received and answers a one-part multipart/related application/octet-stream body, so a test
// can assert whether ResolveBulkDataURI attached the bearer token.
func bulkDataEchoServer(t *testing.T, payload []byte) (*httptest.Server, *string) {
	t.Helper()
	var gotAuth string
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var buf bytes.Buffer
		mw := NewMultipartWriter(&buf, mediaTypeOctet)
		if err := mw.AddPart(mediaTypeOctet, bytes.NewReader(payload)); err != nil {
			t.Errorf("AddPart: %v", err)
		}
		if _, err := mw.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
		w.Header().Set("Content-Type", mw.ContentType())
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}))
	t.Cleanup(hs.Close)
	return hs, &gotAuth
}

// TestResolveBulkDataURISameOriginSendsToken asserts the bearer token is attached when the
// resolved absolute BulkDataURI is same-origin with the client's configured base URL.
func TestResolveBulkDataURISameOriginSendsToken(t *testing.T) {
	const token = "s3cr3t-pacs-token"
	payload := bytes.Repeat([]byte{0xAB}, 16)
	origin, gotAuth := bulkDataEchoServer(t, payload)

	c, err := NewClient(origin.URL, WithHTTPClient(origin.Client()), WithBearerToken(token))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	// An absolute BulkDataURI on the same origin (host:port) as the base URL.
	uri := BulkDataURI(origin.URL + "/studies/1.2.3/series/1.2.3.4/instances/1.2.3.4.5/bulkdata/0")
	got, err := c.ResolveBulkDataURI(context.Background(), uri)
	if err != nil {
		t.Fatalf("ResolveBulkDataURI: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("resolved bulk data does not match the payload")
	}
	if *gotAuth != "Bearer "+token {
		t.Fatalf("same-origin Authorization = %q, want the bearer token to be sent", *gotAuth)
	}
}

// TestResolveBulkDataURICrossOriginOmitsToken is the credential-leak regression: when
// cross-origin bulk-data fetching is explicitly opted in, a server-supplied absolute
// BulkDataURI on a different host must still be fetched WITHOUT the bearer token, so a
// malicious or compromised origin cannot harvest the PACS credential.
func TestResolveBulkDataURICrossOriginOmitsToken(t *testing.T) {
	const token = "s3cr3t-pacs-token"
	payload := bytes.Repeat([]byte{0xCD}, 16)

	// The configured origin (its handler is never reached for the cross-origin reference).
	base, baseAuth := bulkDataEchoServer(t, payload)
	// A different host: a distinct httptest server gets its own port, so it is cross-origin.
	evil, evilAuth := bulkDataEchoServer(t, payload)

	c, err := NewClient(base.URL, WithHTTPClient(base.Client()), WithBearerToken(token),
		WithAllowCrossOriginBulkData())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	uri := BulkDataURI(evil.URL + "/harvested/bulkdata/0")
	got, err := c.ResolveBulkDataURI(context.Background(), uri)
	if err != nil {
		t.Fatalf("ResolveBulkDataURI: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("resolved bulk data does not match the cross-origin payload")
	}
	if *evilAuth != "" {
		t.Fatalf("cross-origin Authorization = %q, want NO credential sent (token must not leak cross-origin)", *evilAuth)
	}
	if *baseAuth != "" {
		t.Fatalf("base origin saw Authorization = %q, but the cross-origin reference should never reach it", *baseAuth)
	}
}

// TestResolveBulkDataURICrossOriginRefusedByDefault is the SSRF regression: a server-supplied
// absolute BulkDataURI on a host other than the configured origin must be refused before any
// request leaves the process, so a hostile or compromised origin cannot steer the client at an
// internal address. The opt-in (a blanket allow or a host allowlist) re-enables the fetch; a
// same-origin and a relative reference always resolve.
func TestResolveBulkDataURICrossOriginRefusedByDefault(t *testing.T) {
	payload := bytes.Repeat([]byte{0xEF}, 16)

	base, _ := bulkDataEchoServer(t, payload)
	other, _ := bulkDataEchoServer(t, payload)

	// The reference's host carries no UID, so naming it in the error leaks no PHI; the
	// UID-bearing path must never appear.
	crossRef := BulkDataURI(other.URL + "/studies/1.2.3/series/4.5.6/instances/7.8.9/bulkdata/0")

	t.Run("refused by default", func(t *testing.T) {
		c, err := NewClient(base.URL, WithHTTPClient(base.Client()))
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		_, err = c.ResolveBulkDataURI(context.Background(), crossRef)
		if !errors.Is(err, ErrCrossOriginBulkData) {
			t.Fatalf("cross-origin ResolveBulkDataURI: err = %v, want ErrCrossOriginBulkData", err)
		}
		coe, ok := errors.AsType[*CrossOriginBulkDataError](err)
		if !ok {
			t.Fatalf("err = %v, want a *CrossOriginBulkDataError", err)
		}
		otherURL, _ := url.Parse(other.URL)
		if coe.Host != otherURL.Host {
			t.Fatalf("error host = %q, want the rejected host %q", coe.Host, otherURL.Host)
		}
		if strings.Contains(err.Error(), "1.2.3") || strings.Contains(err.Error(), "bulkdata") {
			t.Fatalf("error message leaked the UID-bearing path: %q", err.Error())
		}
	})

	t.Run("allowed under blanket opt-in", func(t *testing.T) {
		c, err := NewClient(base.URL, WithHTTPClient(base.Client()), WithAllowCrossOriginBulkData())
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		got, err := c.ResolveBulkDataURI(context.Background(), crossRef)
		if err != nil {
			t.Fatalf("ResolveBulkDataURI under opt-in: %v", err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("resolved bulk data does not match the payload")
		}
	})

	t.Run("allowed for an allowlisted host", func(t *testing.T) {
		otherURL, _ := url.Parse(other.URL)
		c, err := NewClient(base.URL, WithHTTPClient(base.Client()),
			WithBulkDataHostAllowlist(otherURL.Host))
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		got, err := c.ResolveBulkDataURI(context.Background(), crossRef)
		if err != nil {
			t.Fatalf("ResolveBulkDataURI for allowlisted host: %v", err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("resolved bulk data does not match the payload")
		}
	})

	t.Run("same-origin still resolves", func(t *testing.T) {
		c, err := NewClient(base.URL, WithHTTPClient(base.Client()))
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		sameRef := BulkDataURI(base.URL + "/studies/1.2.3/series/4.5.6/instances/7.8.9/bulkdata/0")
		got, err := c.ResolveBulkDataURI(context.Background(), sameRef)
		if err != nil {
			t.Fatalf("same-origin ResolveBulkDataURI: %v", err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("resolved bulk data does not match the payload")
		}
	})

	t.Run("relative reference still resolves", func(t *testing.T) {
		c, err := NewClient(base.URL, WithHTTPClient(base.Client()))
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		got, err := c.ResolveBulkDataURI(context.Background(), BulkDataURI("studies/1.2.3/bulkdata/0"))
		if err != nil {
			t.Fatalf("relative ResolveBulkDataURI: %v", err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("resolved bulk data does not match the payload")
		}
	})
}

// faultingRetrieveBackend returns a NON-sentinel error from every retriever method, standing in
// for a broken catalogue or object store behind the backend.
type faultingRetrieveBackend struct{}

func (faultingRetrieveBackend) RetrieveInstance(context.Context, ResourcePath) (*dicom.DataSet, error) {
	return nil, errors.New("backend fault")
}

func (faultingRetrieveBackend) RetrieveStudy(context.Context, dicom.UID) ([]RetrievedInstance, error) {
	return nil, errors.New("backend fault")
}

func (faultingRetrieveBackend) RetrieveMetadata(context.Context, ResourcePath) ([]RetrievedInstance, error) {
	return nil, errors.New("backend fault")
}

// TestRetrieveBackendFaultIs500Never404 is the seam regression for the retriever error contract:
// only an error wrapping ErrNotFound answers 404; any other backend error is an internal fault
// answered 500, so a catalogue/store failure is never disguised as an absent resource (PRD §9.2).
func TestRetrieveBackendFaultIs500Never404(t *testing.T) {
	srv, err := NewServer(WithRetrieveBackend(faultingRetrieveBackend{}))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	cases := []struct {
		name   string
		path   string
		accept string
	}{
		{"study", "/studies/1.2.3", `multipart/related; type="application/dicom"`},
		{"instance", "/studies/1.2.3/series/1.2.3.4/instances/1.2.3.4.5", `multipart/related; type="application/dicom"`},
		{"metadata", "/studies/1.2.3/metadata", "application/dicom+json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, hs.URL+tc.path, nil)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}
			req.Header.Set("Accept", tc.accept)
			resp, err := hs.Client().Do(req)
			if err != nil {
				t.Fatalf("GET %s: %v", tc.path, err)
			}
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusInternalServerError {
				t.Errorf("%s backend fault status = %d, want 500 (never a 404 that hides the fault)", tc.name, resp.StatusCode)
			}
		})
	}

	// Control: the sentinel still answers 404 (the wadoStore fakes return ErrNotFound).
	notFoundSrv, err := NewServer(WithRetrieveBackend(newWADOStore()))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	nfs := httptest.NewServer(notFoundSrv.Handler())
	t.Cleanup(nfs.Close)
	req, err := http.NewRequest(http.MethodGet, nfs.URL+"/studies/9.9.9/series/9.9.9.1/instances/9.9.9.1.1", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Accept", `multipart/related; type="application/dicom"`)
	resp, err := nfs.Client().Do(req)
	if err != nil {
		t.Fatalf("GET control: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("ErrNotFound control status = %d, want 404", resp.StatusCode)
	}
}

// TestBulkDataURIReturnsExactlyTheReferencedAttribute is the per-attribute locator regression
// (PS3.18 §10.4.4): an instance carrying several binary attributes emits one BulkDataURI per
// attribute in its metadata, and resolving each URI returns exactly ITS OWN octets — never the
// whole instance's bulk data — including a binary value nested in a sequence item. A bogus
// attribute path under the bulkdata sub-resource answers 404.
func TestBulkDataURIReturnsExactlyTheReferencedAttribute(t *testing.T) {
	const sop = "1.2.3.4.5"
	pixels := bytes.Repeat([]byte{0xAB}, 64)
	doc := bytes.Repeat([]byte{0xCD}, 32)
	nested := bytes.Repeat([]byte{0xEF}, 16)

	ds := sampleInstance("1.2.3", "1.2.3.4", sop)
	ds.Set(dicom.Element{Tag: dicom.TagPixelData, VR: dicom.VROB, Value: dicom.NewBytes(dicom.VROB, pixels)})
	ds.Set(dicom.Element{Tag: dicom.TagEncapsulatedDocument, VR: dicom.VROB, Value: dicom.NewBytes(dicom.VROB, doc)})
	item := dicom.NewDataSet()
	item.Set(dicom.Element{Tag: dicom.TagEncapsulatedDocument, VR: dicom.VROB, Value: dicom.NewBytes(dicom.VROB, nested)})
	ds.Set(dicom.Element{
		Tag: dicom.TagIconImageSequence, VR: dicom.VRSQ,
		Value: dicom.NewSequenceValue(dicom.NewSequence(item)),
	})

	store := newWADOStore()
	store.put(RetrievedInstance{DataSet: ds})
	c := newWADOServerClient(t, store)

	metas, err := c.RetrieveMetadata(context.Background(), NewInstance("1.2.3", "1.2.3.4", sop))
	if err != nil {
		t.Fatalf("RetrieveMetadata: %v", err)
	}
	uris := BulkDataURIs(metas[0])
	if len(uris) != 3 {
		t.Fatalf("metadata carried %d BulkDataURIs, want 3 (pixel data, document, nested document): %v", len(uris), uris)
	}

	// Each emitted URI's locator suffix names its attribute; each must resolve to its own bytes.
	want := map[string][]byte{
		"/bulkdata/7FE00010":            pixels,
		"/bulkdata/00420011":            doc,
		"/bulkdata/00880200/0/00420011": nested,
	}
	for _, uri := range uris {
		var matched bool
		for suffix, expect := range want {
			if !strings.HasSuffix(string(uri), suffix) {
				continue
			}
			matched = true
			got, rerr := c.ResolveBulkDataURI(context.Background(), uri)
			if rerr != nil {
				t.Fatalf("ResolveBulkDataURI(%s): %v", suffix, rerr)
			}
			if !bytes.Equal(got, expect) {
				t.Errorf("URI %s resolved to %d bytes that are not the referenced attribute's own value (want %d bytes)",
					suffix, len(got), len(expect))
			}
		}
		if !matched {
			t.Errorf("metadata emitted an unexpected BulkDataURI %q", uri)
		}
	}

	// A locator naming a non-binary attribute, and a malformed locator, are 404 — never wrong bytes.
	base := strings.TrimSuffix(string(uris[0]), "/7FE00010")
	for _, bogus := range []string{base + "/00100010", base + "/ZZZZ"} {
		_, rerr := c.ResolveBulkDataURI(context.Background(), BulkDataURI(bogus))
		he, ok := errors.AsType[*HTTPError](rerr)
		if !ok || he.StatusCode != http.StatusNotFound {
			t.Errorf("bogus locator %q error = %v, want HTTPError 404", bogus, rerr)
		}
	}
}

// metadataOnlyBackend implements the base RetrieveBackend and MetadataRetriever but NOT
// BulkDataRetriever, so the per-attribute locator regression can prove the URIs metadata emits
// are resolvable without the optional return-all interface.
type metadataOnlyBackend struct {
	ds *dicom.DataSet
}

func (b metadataOnlyBackend) RetrieveInstance(_ context.Context, p ResourcePath) (*dicom.DataSet, error) {
	if uid, _ := b.ds.GetString(dicom.TagSOPInstanceUID); uid != string(p.Instance) {
		return nil, ErrNotFound
	}
	return b.ds, nil
}

func (b metadataOnlyBackend) RetrieveMetadata(_ context.Context, p ResourcePath) ([]RetrievedInstance, error) {
	if _, err := b.RetrieveInstance(context.Background(), p); err != nil {
		return nil, err
	}
	return []RetrievedInstance{{DataSet: b.ds}}, nil
}

// TestBulkDataLocatorServedWithoutBulkDataRetriever asserts a locator-suffixed bulkdata target
// needs only the base RetrieveBackend: a backend implementing MetadataRetriever but not
// BulkDataRetriever still serves the per-attribute URIs its own metadata emits (200 with the
// referenced octets), while the bare return-all ".../bulkdata" stays gated on BulkDataRetriever
// and answers 501.
func TestBulkDataLocatorServedWithoutBulkDataRetriever(t *testing.T) {
	const sop = "1.2.3.4.5"
	pixels := bytes.Repeat([]byte{0xAB}, 64)
	ds := sampleInstance("1.2.3", "1.2.3.4", sop)
	ds.Set(dicom.Element{Tag: dicom.TagPixelData, VR: dicom.VROB, Value: dicom.NewBytes(dicom.VROB, pixels)})

	srv, err := NewServer(WithRetrieveBackend(metadataOnlyBackend{ds: ds}))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	metas, err := c.RetrieveMetadata(context.Background(), NewInstance("1.2.3", "1.2.3.4", sop))
	if err != nil {
		t.Fatalf("RetrieveMetadata: %v", err)
	}
	uris := BulkDataURIs(metas[0])
	if len(uris) != 1 {
		t.Fatalf("metadata carried %d BulkDataURIs, want 1 (the pixel data): %v", len(uris), uris)
	}
	got, err := c.ResolveBulkDataURI(context.Background(), uris[0])
	if err != nil {
		t.Fatalf("ResolveBulkDataURI without BulkDataRetriever: %v", err)
	}
	if !bytes.Equal(got, pixels) {
		t.Errorf("locator URI resolved %d bytes, want the %d referenced pixel bytes", len(got), len(pixels))
	}

	// The bare return-all sub-resource still requires the optional BulkDataRetriever.
	req, err := http.NewRequest(http.MethodGet,
		hs.URL+"/studies/1.2.3/series/1.2.3.4/instances/"+sop+"/bulkdata", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Accept", `multipart/related; type="application/octet-stream"`)
	resp, err := hs.Client().Do(req)
	if err != nil {
		t.Fatalf("GET bare bulkdata: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("bare bulkdata without BulkDataRetriever status = %d, want 501", resp.StatusCode)
	}
}
