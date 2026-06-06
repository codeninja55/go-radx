package dimse

import (
	"context"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

// TestCGetCommandFieldValues pins the C-GET command field constants to their PS3.7 wire values.
func TestCGetCommandFieldValues(t *testing.T) {
	if CommandCGetRQ != 0x0010 {
		t.Errorf("CommandCGetRQ = %#04x, want 0x0010", uint16(CommandCGetRQ))
	}
	if CommandCGetRSP != 0x8010 {
		t.Errorf("CommandCGetRSP = %#04x, want 0x8010", uint16(CommandCGetRSP))
	}
}

// TestDefaultGetModel pins the C-GET default information model per level: Patient-level uses Patient
// Root GET (Study Root has no Patient level); every other level uses Study Root GET.
func TestDefaultGetModel(t *testing.T) {
	if got := defaultGetModel(QueryLevelPatient); got != patientRootGetSOPClass {
		t.Errorf("defaultGetModel(Patient) = %s, want Patient Root GET", got)
	}
	for _, lvl := range []QueryLevel{QueryLevelStudy, QueryLevelSeries, QueryLevelImage} {
		if got := defaultGetModel(lvl); got != studyRootGetSOPClass {
			t.Errorf("defaultGetModel(%s) = %s, want Study Root GET", lvl, got)
		}
	}
}

// TestGetPreflightFailClosedOnNilAssociation verifies a C-GET on a nil/unestablished association
// yields a single terminal Failure status and never panics (DIMSE-017). The sink is non-nil so the
// failure is attributable to the association, not the sink guard.
func TestGetPreflightFailClosedOnNilAssociation(t *testing.T) {
	var a *Association
	sink := &recordingGetSink{}

	var statuses []Status
	for status := range a.Get(context.Background(), nil, QueryLevelStudy, sink) {
		statuses = append(statuses, status)
	}
	if len(statuses) != 1 || !statuses[0].IsFailure() {
		t.Fatalf("Get on a nil association yielded %v, want one terminal Failure", statuses)
	}
	// LastError on a nil association is nil-safe and returns nil (no slot to record on).
	if err := a.LastError(); err != nil {
		t.Errorf("LastError on a nil association = %v, want nil", err)
	}
}

// TestGetPreflightRejectsNonGetModel verifies WithQueryModel naming a non-GET (a FIND or MOVE) model
// fails the C-GET pre-flight closed with a single terminal Failure, never sending a C-GET-RQ whose
// Affected SOP Class is not a retrieve information model.
func TestGetPreflightRejectsNonGetModel(t *testing.T) {
	var a *Association // nil is fine: the model check runs before the association check
	sink := &recordingGetSink{}

	var statuses []Status
	for status := range a.Get(context.Background(), nil, QueryLevelStudy, sink,
		WithQueryModel(studyRootFindSOPClass)) {
		statuses = append(statuses, status)
	}
	if len(statuses) != 1 || !statuses[0].IsFailure() {
		t.Fatalf("Get with a FIND model yielded %v, want one terminal Failure", statuses)
	}
}

// TestGetPreflightRejectsNilSink verifies a C-GET with a nil StoreHandler sink fails closed: there is
// nothing to receive the sub-operation instances, so the runtime must not proceed.
func TestGetPreflightRejectsNilSink(t *testing.T) {
	var a *Association

	var statuses []Status
	for status := range a.Get(context.Background(), dicom.NewDataSet(), QueryLevelStudy, nil) {
		statuses = append(statuses, status)
	}
	if len(statuses) != 1 || !statuses[0].IsFailure() {
		t.Fatalf("Get with a nil sink yielded %v, want one terminal Failure", statuses)
	}
}
