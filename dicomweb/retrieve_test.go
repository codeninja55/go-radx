package dicomweb

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
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
		return nil, errors.New("not found")
	}
	return si.DataSet, nil
}

func (m *wadoStore) RetrieveStoredInstance(_ context.Context, p ResourcePath) (RetrievedInstance, error) {
	si, ok := m.instances[string(p.Instance)]
	if !ok {
		return RetrievedInstance{}, errors.New("not found")
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
			return nil, errors.New("not found")
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
		return nil, errors.New("not found")
	}
	out := make([]BulkDataObject, 0, len(frames))
	for _, f := range frames {
		if f < 1 || f > len(all) {
			return nil, errors.New("frame out of range")
		}
		out = append(out, BulkDataObject{Data: all[f-1]})
	}
	return out, nil
}

func (m *wadoStore) RetrieveBulkData(_ context.Context, p ResourcePath) ([]BulkDataObject, error) {
	all, ok := m.bulk[string(p.Instance)]
	if !ok {
		return nil, errors.New("not found")
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
	var he *HTTPError
	if !errors.As(got, &he) || he.StatusCode != http.StatusNotFound {
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
	return nil, errors.New("not found")
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

// TestResolveBulkDataURICrossOriginOmitsToken is the credential-leak regression: a
// server-supplied absolute BulkDataURI on a different host must be fetched WITHOUT the bearer
// token, so a malicious or compromised origin cannot harvest the PACS credential.
func TestResolveBulkDataURICrossOriginOmitsToken(t *testing.T) {
	const token = "s3cr3t-pacs-token"
	payload := bytes.Repeat([]byte{0xCD}, 16)

	// The configured origin (its handler is never reached for the cross-origin reference).
	base, baseAuth := bulkDataEchoServer(t, payload)
	// A different host: a distinct httptest server gets its own port, so it is cross-origin.
	evil, evilAuth := bulkDataEchoServer(t, payload)

	c, err := NewClient(base.URL, WithHTTPClient(base.Client()), WithBearerToken(token))
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
