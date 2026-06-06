//go:build interop

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicomweb"
)

// TestInteropQIDOOrthanc STOWs a vendored instance to a real Orthanc origin, then exercises
// the QIDO-RS client at the study, series, and instance levels against Orthanc's QIDO-RS
// endpoint, asserting the stored study/series/instance are found by their identity UIDs.
// It proves the client's query encoding and application/dicom+json result parsing against a
// compliant origin (PRD §11.1) — the cross-implementation gate for QIDO-RS, the HTTP
// counterpart to the DIMSE C-FIND interop test.
func TestInteropQIDOOrthanc(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	orth := startOrthanc(ctx, t)
	ds, study, series, sop := readFixture(t)

	client, err := dicomweb.NewClient(orth.DICOMWebBaseURL())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if _, err := client.Store(ctx, ds); err != nil {
		t.Fatalf("STOW-RS store (seed for QIDO): %v", err)
	}

	// Study level: find the stored study by its StudyInstanceUID.
	studies, err := client.SearchStudies(ctx, dicomweb.SearchQuery{
		Match: map[string]string{"StudyInstanceUID": study},
	})
	if err != nil {
		t.Fatalf("QIDO-RS SearchStudies: %v", err)
	}
	if !containsUID(studies, dicom.TagStudyInstanceUID, study) {
		t.Fatalf("QIDO-RS study search did not return the stored study")
	}

	// Series level, scoped to the stored study.
	seriesResults, err := client.SearchSeries(ctx, dicom.UID(study), dicomweb.SearchQuery{
		Match: map[string]string{"SeriesInstanceUID": series},
	})
	if err != nil {
		t.Fatalf("QIDO-RS SearchSeries: %v", err)
	}
	if !containsUID(seriesResults, dicom.TagSeriesInstanceUID, series) {
		t.Fatalf("QIDO-RS series search did not return the stored series")
	}

	// Instance level, scoped to the stored study and series.
	instances, err := client.SearchInstances(ctx, dicom.UID(study), dicom.UID(series), dicomweb.SearchQuery{
		Match: map[string]string{"SOPInstanceUID": sop},
	})
	if err != nil {
		t.Fatalf("QIDO-RS SearchInstances: %v", err)
	}
	if !containsUID(instances, dicom.TagSOPInstanceUID, sop) {
		t.Fatalf("QIDO-RS instance search did not return the stored instance")
	}
}

// containsUID reports whether any QIDO-RS result carries want at tag. The search level
// dictates which identity tag is present, so each call asserts the tag for its level.
func containsUID(results []dicomweb.SearchResult, tag dicom.Tag, want string) bool {
	for _, r := range results {
		if r.DataSet == nil {
			continue
		}
		if v, ok := r.DataSet.GetString(tag); ok && v == want {
			return true
		}
	}
	return false
}
