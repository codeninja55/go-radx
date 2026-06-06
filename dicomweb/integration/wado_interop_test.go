//go:build interop

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicomweb"
)

// TestInteropWADOStudySeriesOrthanc STOWs a vendored instance to a real Orthanc origin, then
// retrieves it back at the study and series levels through the streaming WADO-RS iterators,
// asserting the stored instance is present in each. It proves the client's multipart/related
// study and series retrieval and application/dicom part decoding against a compliant origin
// (PRD §11.1).
func TestInteropWADOStudySeriesOrthanc(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	orth := startOrthanc(ctx, t)
	ds, study, series, sop := readFixture(t)

	client, err := dicomweb.NewClient(orth.DICOMWebBaseURL())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if _, err := client.Store(ctx, ds); err != nil {
		t.Fatalf("STOW-RS store (seed for WADO): %v", err)
	}

	studyHas := false
	for got, err := range client.RetrieveStudy(ctx, dicom.UID(study)) {
		if err != nil {
			t.Fatalf("WADO-RS RetrieveStudy: %v", err)
		}
		if v, _ := got.GetString(dicom.TagSOPInstanceUID); v == sop {
			studyHas = true
		}
	}
	if !studyHas {
		t.Fatal("WADO-RS study retrieve did not return the stored instance")
	}

	seriesHas := false
	for got, err := range client.RetrieveSeries(ctx, dicom.UID(study), dicom.UID(series)) {
		if err != nil {
			t.Fatalf("WADO-RS RetrieveSeries: %v", err)
		}
		if v, _ := got.GetString(dicom.TagSOPInstanceUID); v == sop {
			seriesHas = true
		}
	}
	if !seriesHas {
		t.Fatal("WADO-RS series retrieve did not return the stored instance")
	}
}

// TestInteropWADOMetadataBulkDataOrthanc STOWs a vendored instance, retrieves its
// instance-level metadata as application/dicom+json, then resolves a BulkDataURI the metadata
// emits (the pixel data) back to its octets through the bulkdata sub-resource. It proves the
// metadata-with-bulkdata round trip against a compliant origin: the reference Orthanc emits
// is resolvable by the client.
func TestInteropWADOMetadataBulkDataOrthanc(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	orth := startOrthanc(ctx, t)
	ds, study, series, sop := readFixture(t)

	client, err := dicomweb.NewClient(orth.DICOMWebBaseURL())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if _, err := client.Store(ctx, ds); err != nil {
		t.Fatalf("STOW-RS store (seed for WADO metadata): %v", err)
	}

	metas, err := client.RetrieveMetadata(ctx, dicomweb.NewInstance(dicom.UID(study), dicom.UID(series), dicom.UID(sop)))
	if err != nil {
		t.Fatalf("WADO-RS RetrieveMetadata: %v", err)
	}
	if len(metas) == 0 {
		t.Fatal("WADO-RS metadata returned no objects")
	}

	uris := dicomweb.BulkDataURIs(metas[0])
	if len(uris) == 0 {
		t.Skip("Orthanc inlined every binary value; no BulkDataURI to resolve in this fixture")
	}
	resolved, err := client.ResolveBulkDataURI(ctx, uris[0])
	if err != nil {
		t.Fatalf("WADO-RS ResolveBulkDataURI: %v", err)
	}
	if len(resolved) == 0 {
		t.Fatal("WADO-RS resolved bulk data is empty")
	}
}

// TestInteropWADOFramesOrthanc STOWs a vendored instance and retrieves its first frame as a
// multipart/related application/octet-stream body, asserting non-empty octets come back. It
// proves the client's frame retrieval and octet-stream part parsing against a compliant
// origin (PS3.18 §10.4.3).
func TestInteropWADOFramesOrthanc(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	orth := startOrthanc(ctx, t)
	ds, study, series, sop := readFixture(t)

	client, err := dicomweb.NewClient(orth.DICOMWebBaseURL())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if _, err := client.Store(ctx, ds); err != nil {
		t.Fatalf("STOW-RS store (seed for WADO frames): %v", err)
	}

	frames, err := client.RetrieveFrames(ctx, dicomweb.NewInstance(dicom.UID(study), dicom.UID(series), dicom.UID(sop)), 1)
	if err != nil {
		t.Fatalf("WADO-RS RetrieveFrames: %v", err)
	}
	if len(frames) != 1 || len(frames[0]) == 0 {
		t.Fatalf("WADO-RS frames retrieve returned %d frames, want 1 non-empty frame", len(frames))
	}
}
