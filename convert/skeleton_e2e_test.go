package convert

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicomweb"
	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/fhir/r5"
	"github.com/codeninja55/go-radx/hl7v2"
)

// recordingStoreSCP is an in-process DIMSE C-STORE SCP that records the received
// dataset, so the skeleton can assert the C-STORE leg transferred the instance.
type recordingStoreSCP struct {
	mu sync.Mutex
	ds *dicom.DataSet
}

func (h *recordingStoreSCP) Store(_ context.Context, ds *dicom.DataSet, _ dimse.OpInfo) dimse.Status {
	h.mu.Lock()
	h.ds = ds.Clone()
	h.mu.Unlock()
	return dimse.StatusStoreSuccess
}

func (h *recordingStoreSCP) received() *dicom.DataSet {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.ds
}

// memWebStore is an in-process DICOMweb StoreBackend/RetrieveBackend keyed by SOP
// Instance UID, so the skeleton can STOW an instance and WADO-read it back.
type memWebStore struct {
	mu        sync.Mutex
	instances map[string]*dicom.DataSet
}

func newMemWebStore() *memWebStore {
	return &memWebStore{instances: make(map[string]*dicom.DataSet)}
}

func (m *memWebStore) Store(_ context.Context, ds *dicom.DataSet) error {
	uid, _ := ds.GetString(dicom.TagSOPInstanceUID)
	m.mu.Lock()
	m.instances[uid] = ds.Clone()
	m.mu.Unlock()
	return nil
}

func (m *memWebStore) RetrieveInstance(_ context.Context, p dicomweb.ResourcePath) (*dicom.DataSet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ds, ok := m.instances[string(p.Instance)]
	if !ok {
		return nil, dicomweb.ErrUnsupported
	}
	return ds, nil
}

// startDIMSEStoreSCP serves an in-process C-STORE SCP on loopback and returns its
// address and the recording handler.
func startDIMSEStoreSCP(t *testing.T) (string, *recordingStoreSCP) {
	t.Helper()
	ae, err := dimse.NewAE(dimse.AETitle("SKEL-SCP"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	h := &recordingStoreSCP{}
	srv := dimse.NewServer(ae, dimse.StorageContexts(), h)

	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && srv.Addr() == nil {
		time.Sleep(time.Millisecond)
	}
	if srv.Addr() == nil {
		t.Fatal("DIMSE SCP did not bind within the deadline")
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		<-served
	})
	return srv.Addr().String(), h
}

// TestSkeletonEndToEnd is the M2 acceptance proof: it drives all six skeleton
// legs in-process (no containers) and asserts each output, proving every
// subsystem connects.
//
//  1. read a vendored .dcm
//  2. C-STORE it via an in-process DIMSE SCU -> SCP
//  3. STOW it to an in-process DICOMweb server
//  4. WADO-read it back from that server
//  5. parse a vendored ORM -> ServiceRequest, and an SR -> DiagnosticReport
//  6. convert the DICOM instance -> ImagingStudy
func TestSkeletonEndToEnd(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Leg 1: read the vendored DICOM instance.
	file, err := dicom.ReadFile(filepath.Join("..", "testdata", "dicom", "MR2_UNCI.dcm"))
	if err != nil {
		t.Fatalf("leg 1 read .dcm: %v", err)
	}
	src := file.DataSet
	sentSOP, _ := src.GetString(dicom.TagSOPInstanceUID)
	sentStudy, _ := src.GetString(dicom.TagStudyInstanceUID)
	sentSeries, _ := src.GetString(dicom.TagSeriesInstanceUID)
	if sentSOP == "" || sentStudy == "" || sentSeries == "" {
		t.Fatalf("leg 1 fixture missing identity: sop=%q study=%q series=%q", sentSOP, sentStudy, sentSeries)
	}

	// Leg 2: C-STORE the instance via an in-process DIMSE SCU -> SCP.
	scpAddr, scpHandler := startDIMSEStoreSCP(t)
	scu, err := dimse.NewAE(dimse.AETitle("SKEL-SCU"))
	if err != nil {
		t.Fatalf("leg 2 NewAE: %v", err)
	}
	assoc, err := scu.Associate(ctx, scpAddr, dimse.AETitle("SKEL-SCP"), dimse.StorageContexts())
	if err != nil {
		t.Fatalf("leg 2 Associate: %v", err)
	}
	status, err := assoc.Store(ctx, src)
	if err != nil {
		t.Fatalf("leg 2 Store: %v", err)
	}
	if !status.IsSuccess() {
		t.Fatalf("leg 2 Store status = %s, want success", status)
	}
	if err := assoc.Release(ctx); err != nil {
		t.Fatalf("leg 2 Release: %v", err)
	}
	if got := scpHandler.received(); got == nil {
		t.Fatal("leg 2 SCP recorded no dataset")
	} else if gotSOP, _ := got.GetString(dicom.TagSOPInstanceUID); gotSOP != sentSOP {
		t.Errorf("leg 2 SCP received SOP Instance UID = %q, want %q", gotSOP, sentSOP)
	}

	// Legs 3 & 4: STOW to and WADO-read from an in-process DICOMweb server.
	store := newMemWebStore()
	web, err := dicomweb.NewServer(
		dicomweb.WithStoreBackend(store),
		dicomweb.WithRetrieveBackend(store),
	)
	if err != nil {
		t.Fatalf("legs 3-4 NewServer: %v", err)
	}
	hs := httptest.NewServer(web.Handler())
	t.Cleanup(hs.Close)
	client, err := dicomweb.NewClient(hs.URL, dicomweb.WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("legs 3-4 NewClient: %v", err)
	}

	storeResp, err := client.Store(ctx, src)
	if err != nil {
		t.Fatalf("leg 3 STOW: %v", err)
	}
	if !storeResp.IsComplete() {
		t.Fatalf("leg 3 STOW response not complete: %+v", storeResp)
	}

	retrieved, err := client.RetrieveInstance(ctx,
		dicomweb.NewInstance(dicom.UID(sentStudy), dicom.UID(sentSeries), dicom.UID(sentSOP)))
	if err != nil {
		t.Fatalf("leg 4 WADO: %v", err)
	}
	if gotSOP, _ := retrieved.GetString(dicom.TagSOPInstanceUID); gotSOP != sentSOP {
		t.Errorf("leg 4 WADO retrieved SOP Instance UID = %q, want %q", gotSOP, sentSOP)
	}

	// Leg 5a: parse a vendored ORM and emit a ServiceRequest.
	msg, err := hl7v2.Parse([]byte(canonicalORM))
	if err != nil {
		t.Fatalf("leg 5a parse ORM: %v", err)
	}
	sr, srReport, err := ORMToServiceRequestR5(msg)
	if err != nil {
		t.Fatalf("leg 5a ORMToServiceRequestR5: %v", err)
	}
	if sr.Status == nil || *sr.Status != r5.RequestStatusActive ||
		sr.Intent == nil || *sr.Intent != r5.RequestIntentOrder {
		t.Errorf("leg 5a ServiceRequest status/intent = %v/%v, want active/order", sr.Status, sr.Intent)
	}
	if len(sr.Identifier) == 0 {
		t.Error("leg 5a ServiceRequest has no identifiers")
	}
	if !hasDefault(srReport, "ServiceRequest.intent", "order") {
		t.Error("leg 5a ServiceRequest report does not record the intent default")
	}

	// Leg 5b: produce a DiagnosticReport from a vendored SR document.
	srFile, err := dicom.ReadFile(filepath.Join("..", "testdata", "dicom", "basic-text-sr.dcm"))
	if err != nil {
		t.Fatalf("leg 5b read SR: %v", err)
	}
	dr, _, _, err := SRToDiagnosticReportR5(srFile.DataSet)
	if err != nil {
		t.Fatalf("leg 5b SRToDiagnosticReportR5: %v", err)
	}
	if len(dr.Identifier) == 0 || dr.Identifier[0].System == nil || *dr.Identifier[0].System != "urn:dicom:uid" {
		t.Errorf("leg 5b DiagnosticReport identifier = %+v, want a urn:dicom:uid identifier", dr.Identifier)
	}
	if dr.Status == nil {
		t.Error("leg 5b DiagnosticReport has no status")
	}

	// Leg 6: convert the DICOM instance to an ImagingStudy.
	study, studyReport, err := DICOMToImagingStudyR5([]*dicom.DataSet{src})
	if err != nil {
		t.Fatalf("leg 6 DICOMToImagingStudyR5: %v", err)
	}
	if study.NumberOfSeries == nil || *study.NumberOfSeries != 1 {
		t.Errorf("leg 6 NumberOfSeries = %v, want 1", study.NumberOfSeries)
	}
	if study.NumberOfInstances == nil || *study.NumberOfInstances != 1 {
		t.Errorf("leg 6 NumberOfInstances = %v, want 1", study.NumberOfInstances)
	}
	// The identity rule end-to-end: the study identifier and subject are logical,
	// never a fabricated Reference URL.
	if len(study.Identifier) == 0 || study.Identifier[0].Value == nil ||
		*study.Identifier[0].Value != "urn:oid:"+sentStudy {
		t.Errorf("leg 6 study identifier = %+v, want urn:oid:%s", study.Identifier, sentStudy)
	}
	if study.Subject != nil && study.Subject.Reference != nil {
		t.Errorf("leg 6 study subject is a Reference URL %q, want a logical identifier", *study.Subject.Reference)
	}
	_ = studyReport
}
