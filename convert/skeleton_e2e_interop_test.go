//go:build interop

// This file holds the interop variant of the M2 walking-skeleton end-to-end
// proof. It drives the DIMSE C-STORE leg and the DICOMweb STOW/WADO legs against
// a real Orthanc container, then runs the three pure-conversion legs in-process.
// It is behind the interop build tag so the default build and test run are
// unaffected and the testcontainers dependency stays out of the default build
// graph. The orchestrator runs it when Docker is free; it MUST compile under
// `go build -tags interop ./...` and `go vet -tags interop`.
package convert

import (
	"context"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicomweb"
	dwebOrthanc "github.com/codeninja55/go-radx/dicomweb/integration/orthanc"
	"github.com/codeninja55/go-radx/dimse"
	dimseOrthanc "github.com/codeninja55/go-radx/dimse/integration/orthanc"
	"github.com/codeninja55/go-radx/fhir/r5"
	"github.com/codeninja55/go-radx/hl7v2"
)

// skeletonInteropFixture is the vendored, uncompressed MR Image Storage instance
// the interop skeleton stores and retrieves.
const skeletonInteropFixture = "../testdata/dicom/MR2_UNCI.dcm"

// TestSkeletonEndToEndInterop is the interop variant of the M2 acceptance proof.
// It exercises the same six legs as the in-process test, but the DIMSE and
// DICOMweb legs run against real containers (DIMSE C-STORE to an Orthanc DIMSE
// SCP, then STOW/WADO against a separate Orthanc DICOMweb endpoint).
func TestSkeletonEndToEndInterop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Leg 1: read the vendored DICOM instance.
	file, err := dicom.ReadFile(skeletonInteropFixture)
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

	// Leg 2: C-STORE the instance to a real Orthanc DIMSE SCP.
	dimseOrth, err := dimseOrthanc.Start(ctx)
	if err != nil {
		t.Fatalf("leg 2 start Orthanc (DIMSE): %v", err)
	}
	t.Cleanup(func() { _ = dimseOrth.Stop(context.Background()) })

	calling, err := dimse.ParseAETitle("SKEL-SCU")
	if err != nil {
		t.Fatalf("leg 2 parse calling AE: %v", err)
	}
	called, err := dimse.ParseAETitle(dimseOrthanc.AETitle)
	if err != nil {
		t.Fatalf("leg 2 parse called AE: %v", err)
	}
	scu, err := dimse.NewAE(calling)
	if err != nil {
		t.Fatalf("leg 2 NewAE: %v", err)
	}
	assoc, err := scu.Associate(ctx, dimseOrth.DICOMAddr(), called, dimse.StorageContexts())
	if err != nil {
		t.Fatalf("leg 2 Associate: %v", err)
	}
	status, err := assoc.Store(ctx, src)
	if err != nil {
		t.Fatalf("leg 2 Store: %v", err)
	}
	if !status.IsSuccess() || status.Code != dimse.StatusStoreSuccess.Code {
		t.Fatalf("leg 2 Store status = %s, want StatusStoreSuccess", status)
	}
	if err := assoc.Release(ctx); err != nil {
		t.Fatalf("leg 2 Release: %v", err)
	}
	if !waitForInteropInstance(ctx, t, dimseOrth, sentSOP) {
		t.Fatalf("leg 2 Orthanc did not persist the stored instance %s", sentSOP)
	}

	// Legs 3 & 4: STOW to and WADO-read from a real Orthanc DICOMweb endpoint.
	dwebOrth, err := dwebOrthanc.Start(ctx)
	if err != nil {
		t.Fatalf("legs 3-4 start Orthanc (DICOMweb): %v", err)
	}
	t.Cleanup(func() { _ = dwebOrth.Stop(context.Background()) })

	client, err := dicomweb.NewClient(dwebOrth.DICOMWebBaseURL())
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
	sr, _, err := ORMToServiceRequestR5(msg)
	if err != nil {
		t.Fatalf("leg 5a ORMToServiceRequestR5: %v", err)
	}
	if sr.Intent == nil || *sr.Intent != r5.RequestIntentOrder {
		t.Errorf("leg 5a ServiceRequest intent = %v, want order", sr.Intent)
	}

	// Leg 5b: produce a DiagnosticReport from a vendored SR document.
	srFile, err := dicom.ReadFile("../testdata/dicom/basic-text-sr.dcm")
	if err != nil {
		t.Fatalf("leg 5b read SR: %v", err)
	}
	dr, _, err := SRToDiagnosticReportR5(srFile.DataSet)
	if err != nil {
		t.Fatalf("leg 5b SRToDiagnosticReportR5: %v", err)
	}
	if dr.Status == nil {
		t.Error("leg 5b DiagnosticReport has no status")
	}

	// Leg 6: convert the DICOM instance to an ImagingStudy.
	study, _, err := DICOMToImagingStudyR5([]*dicom.DataSet{src})
	if err != nil {
		t.Fatalf("leg 6 DICOMToImagingStudyR5: %v", err)
	}
	if study.NumberOfInstances == nil || *study.NumberOfInstances != 1 {
		t.Errorf("leg 6 NumberOfInstances = %v, want 1", study.NumberOfInstances)
	}
	if study.Subject != nil && study.Subject.Reference != nil {
		t.Errorf("leg 6 study subject is a Reference URL %q, want a logical identifier", *study.Subject.Reference)
	}
}

// waitForInteropInstance polls Orthanc's REST API until the SOP Instance UID
// appears or the deadline elapses, accommodating Orthanc's asynchronous indexing.
func waitForInteropInstance(ctx context.Context, t *testing.T, orth *dimseOrthanc.Container, sopInstanceUID string) bool {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		found, err := orth.HasInstanceWithSOPUID(ctx, sopInstanceUID)
		if err != nil {
			t.Fatalf("query Orthanc instances: %v", err)
		}
		if found {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}
