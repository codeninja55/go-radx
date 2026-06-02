//go:build interop

// Package integration holds the DICOMweb interop regression net: a STOW-RS store
// followed by a WADO-RS retrieve against a real Orthanc container, proving the
// round-trip end-to-end against a compliant origin (PRD §11.1). Every test is behind
// the interop build tag so the default build and test run are unaffected and the
// testcontainers dependency stays out of the default build graph.
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicomweb"
	"github.com/codeninja55/go-radx/dicomweb/integration/orthanc"
)

// storeFixture is a vendored, uncompressed (Explicit VR Little Endian) MR Image Storage
// instance. MR Image Storage is universally accepted by Orthanc, so the STOW leg
// exercises a real SOP Class without transcoding.
const storeFixture = "../../testdata/dicom/MR2_UNCI.dcm"

// readFixture loads the vendored .dcm and returns its main dataset plus the identity
// UIDs the STOW/WADO round-trip references.
func readFixture(t *testing.T) (ds *dicom.DataSet, study, series, sop string) {
	t.Helper()
	f, err := dicom.ReadFile(storeFixture)
	if err != nil {
		t.Fatalf("read fixture %s: %v", storeFixture, err)
	}
	get := func(tag dicom.Tag, name string) string {
		v, ok := f.DataSet.GetString(tag)
		if !ok || v == "" {
			t.Fatalf("fixture %s has no %s", storeFixture, name)
		}
		return v
	}
	study = get(dicom.TagStudyInstanceUID, "StudyInstanceUID")
	series = get(dicom.TagSeriesInstanceUID, "SeriesInstanceUID")
	sop = get(dicom.TagSOPInstanceUID, "SOPInstanceUID")
	return f.DataSet, study, series, sop
}

// startOrthanc starts an Orthanc container with the DICOMweb plugin and registers its
// teardown. Container start pulls the image on first run, so the caller's context should
// allow several minutes.
func startOrthanc(ctx context.Context, t *testing.T) *orthanc.Container {
	t.Helper()
	c, err := orthanc.Start(ctx)
	if err != nil {
		t.Fatalf("start Orthanc: %v", err)
	}
	t.Cleanup(func() {
		if err := c.Stop(context.Background()); err != nil {
			t.Logf("stop Orthanc: %v", err)
		}
	})
	return c
}

// TestInteropStowThenWadoOrthanc STOWs a vendored .dcm to a real Orthanc origin, asserts
// the store response is complete, confirms via Orthanc's REST API that the exact SOP
// Instance UID landed, then WADO-RS retrieves the instance and asserts the dataset
// round-trips. This is the M2 DICOMweb-leg acceptance gate.
func TestInteropStowThenWadoOrthanc(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	orth := startOrthanc(ctx, t)
	ds, study, series, sop := readFixture(t)

	client, err := dicomweb.NewClient(orth.DICOMWebBaseURL())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.Store(ctx, ds)
	if err != nil {
		t.Fatalf("STOW-RS store: %v", err)
	}
	if !resp.IsComplete() {
		t.Fatalf("STOW-RS store not complete: %d referenced, %d failed", len(resp.Referenced), len(resp.Failed))
	}
	if len(resp.Referenced) != 1 {
		t.Fatalf("STOW-RS referenced %d instances, want 1", len(resp.Referenced))
	}
	if got := string(resp.Referenced[0].SOPInstanceUID); got != sop {
		t.Fatalf("STOW-RS referenced SOPInstanceUID = %q, want %q", got, sop)
	}

	ok, err := orth.HasInstanceWithSOPUID(ctx, sop)
	if err != nil {
		t.Fatalf("verify stored instance via REST: %v", err)
	}
	if !ok {
		t.Fatalf("Orthanc does not hold the stored instance %s after STOW-RS", sop)
	}

	got, err := client.RetrieveInstance(ctx, dicomweb.NewInstance(dicom.UID(study), dicom.UID(series), dicom.UID(sop)))
	if err != nil {
		t.Fatalf("WADO-RS retrieve: %v", err)
	}
	gotSOP, _ := got.GetString(dicom.TagSOPInstanceUID)
	if gotSOP != sop {
		t.Fatalf("WADO-RS retrieved SOPInstanceUID = %q, want %q", gotSOP, sop)
	}
	gotStudy, _ := got.GetString(dicom.TagStudyInstanceUID)
	if gotStudy != study {
		t.Fatalf("WADO-RS retrieved StudyInstanceUID = %q, want %q", gotStudy, study)
	}
}
