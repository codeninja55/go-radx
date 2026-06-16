package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/fhir"
)

// PHI sentinels for the audit no-leak assertions, the internal/phisweep convention:
// synthetic, never real, shaped like the values they stand in for.
const (
	auditPHIName = "SENTINEL^PHI^DONOTLOG"
	auditPHIID   = "ZZZTEST-MRN-PHI-SENTINEL"
)

// auditCollector is a concurrency-safe AuditFunc sink for the tests.
type auditCollector struct {
	mu     sync.Mutex
	events []AuditEvent
}

func (c *auditCollector) fn() AuditFunc {
	return func(ev AuditEvent) {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.events = append(c.events, ev)
	}
}

func (c *auditCollector) all() []AuditEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]AuditEvent(nil), c.events...)
}

// assertNoSentinel scans an event's full rendered form for the PHI sentinels, so a
// value smuggled into any current or future field is caught. Object-identity UIDs
// are permitted event fields by contract (see AuditEvent — values never, object
// identity always); the fixture UIDs are plain numeric dotted strings ("7.1.1.1",
// "1.2.840...") that embed no sentinel, so any hit is a genuine value leak.
func assertNoSentinel(t *testing.T, ev AuditEvent) {
	t.Helper()
	rendered := fmt.Sprintf("%+v", ev)
	for _, sentinel := range []string{auditPHIName, auditPHIID} {
		if strings.Contains(rendered, sentinel) {
			t.Errorf("PHI sentinel %q leaked through the audit event: %s", sentinel, rendered)
		}
	}
}

// newAuditTestObject is newTestObject plus sentinel-bearing patient attributes, so
// the no-PHI assertion exercises a dataset that actually carries patient values.
func newAuditTestObject(study, series, instance string) *dicom.DataSet {
	ds := newTestObject(study, series, instance)
	ds.SetString(dicom.TagPatientName, auditPHIName)
	ds.SetString(dicom.TagPatientID, auditPHIID)
	return ds
}

// TestDIMSEStoreAuditEvent asserts the C-STORE write path emits one structural
// AuditEvent per committed store — operation, outcome, timestamp, and the object's
// hierarchy UIDs (carried by contract) — and that no attribute value leaks through
// any event field.
func TestDIMSEStoreAuditEvent(t *testing.T) {
	t.Parallel()
	store, cat := newTestBackends(t)
	collector := &auditCollector{}
	h := &dimseHandler{store: store, cat: cat, logger: zap.NewNop(), audit: collector.fn()}

	const study, series, instance = "7.1", "7.1.1", "7.1.1.1"
	ds := newAuditTestObject(study, series, instance)
	status := h.Store(context.Background(), ds, dimse.OpInfo{SOPClassUID: "1.2.840.10008.5.1.4.1.1.7"})
	if status != dimse.StatusStoreSuccess {
		t.Fatalf("Store status = %v, want success", status)
	}

	events := collector.all()
	if len(events) != 1 {
		t.Fatalf("audit events = %d, want exactly 1 per committed C-STORE", len(events))
	}
	ev := events[0]
	if ev.Op != AuditOpDIMSEStore {
		t.Errorf("Op = %q, want %q", ev.Op, AuditOpDIMSEStore)
	}
	if ev.Time.IsZero() {
		t.Error("Time is zero, want the commit time")
	}
	if ev.StudyInstanceUID != study || ev.SeriesInstanceUID != series || ev.SOPInstanceUID != instance {
		t.Errorf("event identifiers = %q/%q/%q, want %q/%q/%q",
			ev.StudyInstanceUID, ev.SeriesInstanceUID, ev.SOPInstanceUID, study, series, instance)
	}
	if ev.SOPClassUID == "" {
		t.Error("SOPClassUID is empty, want the stored object's SOP Class")
	}
	if ev.Outcome != AuditOutcomeStoredIndexed {
		t.Errorf("Outcome = %q, want %q", ev.Outcome, AuditOutcomeStoredIndexed)
	}
	assertNoSentinel(t, ev)
}

// failingStore is an ObjectStore whose Put always fails, so the failure path of the
// store handlers can be exercised.
type failingStore struct{}

var errPutFailed = errors.New("put failed")

func (failingStore) Put(context.Context, *dicom.DataSet) error { return errPutFailed }
func (failingStore) Get(context.Context, dicom.SOPInstanceUID) (*dicom.DataSet, error) {
	return nil, ErrNotFound
}
func (failingStore) Exists(context.Context, dicom.SOPInstanceUID) (bool, error) { return false, nil }
func (failingStore) Delete(context.Context, dicom.SOPInstanceUID) error         { return ErrNotFound }

// TestStoreAuditNotEmittedOnFailure asserts the hook reports committed writes only:
// a failed C-STORE or STOW store emits no event (PRD §9.2 — never report work that
// did not happen).
func TestStoreAuditNotEmittedOnFailure(t *testing.T) {
	t.Parallel()
	_, cat := newTestBackends(t)
	collector := &auditCollector{}
	ds := newAuditTestObject("8.1", "8.1.1", "8.1.1.1")

	h := &dimseHandler{store: failingStore{}, cat: cat, logger: zap.NewNop(), audit: collector.fn()}
	if status := h.Store(context.Background(), ds, dimse.OpInfo{}); status == dimse.StatusStoreSuccess {
		t.Fatal("Store against a failing ObjectStore should not succeed")
	}

	b := &dicomwebStore{store: failingStore{}, cat: cat, logger: zap.NewNop(), audit: collector.fn()}
	if err := b.Store(context.Background(), ds); err == nil {
		t.Fatal("STOW Store against a failing ObjectStore should not succeed")
	}

	if events := collector.all(); len(events) != 0 {
		t.Errorf("audit events after failed writes = %d, want 0", len(events))
	}
}

// failingCatalogue is a Catalogue whose Index always fails, so the stored-but-
// un-indexed path of the store handlers can be exercised.
type failingCatalogue struct{}

var errIndexFailed = errors.New("index failed")

func (failingCatalogue) Index(context.Context, *dicom.DataSet) error { return errIndexFailed }
func (failingCatalogue) Query(context.Context, CatalogueQuery) iter.Seq2[*dicom.DataSet, error] {
	return func(func(*dicom.DataSet, error) bool) {}
}
func (failingCatalogue) Remove(context.Context, dicom.SOPInstanceUID) error { return ErrNotFound }

// TestStoreAuditEmittedOnIndexFailure asserts the durable write is the audited
// modification: when ObjectStore.Put commits but Catalogue.Index fails, the object
// is durably stored, so the event must still fire — with the un-indexed outcome —
// for both the DIMSE C-STORE and STOW-RS paths. A silent gap here would be a
// durable, unaudited modification.
func TestStoreAuditEmittedOnIndexFailure(t *testing.T) {
	t.Parallel()
	store, _ := newTestBackends(t)
	collector := &auditCollector{}
	const study, series, instance = "10.1", "10.1.1", "10.1.1.1"

	h := &dimseHandler{store: store, cat: failingCatalogue{}, logger: zap.NewNop(), audit: collector.fn()}
	ds := newAuditTestObject(study, series, instance)
	if status := h.Store(context.Background(), ds, dimse.OpInfo{}); status == dimse.StatusStoreSuccess {
		t.Fatal("Store with a failing Catalogue should not report clean success")
	}

	b := &dicomwebStore{store: store, cat: failingCatalogue{}, logger: zap.NewNop(), audit: collector.fn()}
	if err := b.Store(context.Background(), newAuditTestObject(study, series, instance)); err == nil {
		t.Fatal("STOW Store with a failing Catalogue should return the index error")
	}

	events := collector.all()
	if len(events) != 2 {
		t.Fatalf("audit events = %d, want 2 (one per durably stored object)", len(events))
	}
	for _, ev := range events {
		if ev.Outcome != AuditOutcomeStoredUnindexed {
			t.Errorf("%s Outcome = %q, want %q", ev.Op, ev.Outcome, AuditOutcomeStoredUnindexed)
		}
		if ev.SOPInstanceUID != instance {
			t.Errorf("%s SOPInstanceUID = %q, want %q", ev.Op, ev.SOPInstanceUID, instance)
		}
		assertNoSentinel(t, ev)
	}
}

// TestSTOWStoreAuditEvent asserts the STOW-RS write path emits one structural
// AuditEvent per stored instance — object-identity UIDs carried by contract, no
// attribute value in any field.
func TestSTOWStoreAuditEvent(t *testing.T) {
	t.Parallel()
	store, cat := newTestBackends(t)
	collector := &auditCollector{}
	b := &dicomwebStore{store: store, cat: cat, logger: zap.NewNop(), audit: collector.fn()}

	const study, series, instance = "9.1", "9.1.1", "9.1.1.1"
	if err := b.Store(context.Background(), newAuditTestObject(study, series, instance)); err != nil {
		t.Fatalf("Store: %v", err)
	}

	events := collector.all()
	if len(events) != 1 {
		t.Fatalf("audit events = %d, want exactly 1 per stored instance", len(events))
	}
	ev := events[0]
	if ev.Op != AuditOpSTOWStore {
		t.Errorf("Op = %q, want %q", ev.Op, AuditOpSTOWStore)
	}
	if ev.SOPInstanceUID != instance {
		t.Errorf("SOPInstanceUID = %q, want %q", ev.SOPInstanceUID, instance)
	}
	if ev.Outcome != AuditOutcomeStoredIndexed {
		t.Errorf("Outcome = %q, want %q", ev.Outcome, AuditOutcomeStoredIndexed)
	}
	assertNoSentinel(t, ev)
}

// TestFHIRCreateAuditEvent drives a real daemon with WithAudit through a sentinel-
// bearing FHIR create and asserts one structural event: the resource type plus the
// server-assigned id and version, never a value from the resource body. It exercises
// the daemon-level WithAudit threading end to end.
func TestFHIRCreateAuditEvent(t *testing.T) {
	t.Parallel()
	collector := &auditCollector{}

	repo, err := NewMemoryRepository(fhir.R5)
	if err != nil {
		t.Fatalf("NewMemoryRepository: %v", err)
	}
	role, err := NewFHIRRole(repo, WithFHIRPort(0), WithFHIRRelease(fhir.R5))
	if err != nil {
		t.Fatalf("NewFHIRRole: %v", err)
	}
	d, err := New(WithFHIR(role), WithAudit(collector.fn()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	defer func() {
		cancelRun()
		select {
		case <-runErr:
		case <-time.After(5 * time.Second):
			t.Error("daemon did not stop within 5s")
		}
	}()
	waitForAddrs(t, d, "fhir@/fhir")
	base := "http://" + d.Addrs()["fhir@/fhir"].String() + "/fhir"

	patient := map[string]any{
		"resourceType": "Patient",
		"name":         []map[string]any{{"family": auditPHIName}},
		"identifier":   []map[string]any{{"value": auditPHIID}},
	}
	body, err := json.Marshal(patient)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, base+"/Patient", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/fhir+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST Patient: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", resp.StatusCode)
	}

	events := collector.all()
	if len(events) != 1 {
		t.Fatalf("audit events = %d, want exactly 1 per committed create", len(events))
	}
	ev := events[0]
	if ev.Op != AuditOpFHIRCreate {
		t.Errorf("Op = %q, want %q", ev.Op, AuditOpFHIRCreate)
	}
	if ev.ResourceType != "Patient" {
		t.Errorf("ResourceType = %q, want Patient", ev.ResourceType)
	}
	if ev.ResourceID == "" {
		t.Error("ResourceID is empty, want the server-assigned id")
	}
	if ev.VersionID != "1" {
		t.Errorf("VersionID = %q, want 1 (a create is version 1)", ev.VersionID)
	}
	if ev.Outcome != AuditOutcomeStoredIndexed {
		t.Errorf("Outcome = %q, want %q (a repository create is atomic)", ev.Outcome, AuditOutcomeStoredIndexed)
	}
	if loc := resp.Header.Get("Location"); !strings.Contains(loc, "Patient/"+ev.ResourceID) {
		t.Errorf("Location %q does not name the audited resource id %q", loc, ev.ResourceID)
	}
	assertNoSentinel(t, ev)
}

// TestFHIRUpdateDeleteAuditEvents drives a create, an update, and a delete through a daemon with
// WithAudit and asserts the structural events: the update is audited as an update (version 2), the
// delete as a delete (the deleted outcome), each carrying only the resource type and the
// server-known id and version, never a value from the sentinel-bearing body.
func TestFHIRUpdateDeleteAuditEvents(t *testing.T) {
	t.Parallel()
	collector := &auditCollector{}

	repo, err := NewMemoryRepository(fhir.R5)
	if err != nil {
		t.Fatalf("NewMemoryRepository: %v", err)
	}
	role, err := NewFHIRRole(repo, WithFHIRPort(0), WithFHIRRelease(fhir.R5))
	if err != nil {
		t.Fatalf("NewFHIRRole: %v", err)
	}
	d, err := New(WithFHIR(role), WithAudit(collector.fn()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	defer func() {
		cancelRun()
		select {
		case <-runErr:
		case <-time.After(5 * time.Second):
			t.Error("daemon did not stop within 5s")
		}
	}()
	waitForAddrs(t, d, "fhir@/fhir")
	base := "http://" + d.Addrs()["fhir@/fhir"].String() + "/fhir"

	sentinelPatient := func(id string) []byte {
		patient := map[string]any{
			"resourceType": "Patient",
			"name":         []map[string]any{{"family": auditPHIName}},
			"identifier":   []map[string]any{{"value": auditPHIID}},
		}
		if id != "" {
			patient["id"] = id
		}
		b, _ := json.Marshal(patient)
		return b
	}

	do := func(method, path, contentType string, body []byte) *http.Response {
		req, err := http.NewRequest(method, base+path, bytes.NewReader(body))
		if err != nil {
			t.Fatalf("NewRequest: %v", err)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		_ = resp.Body.Close()
		return resp
	}

	resp := do(http.MethodPost, "/Patient", "application/fhir+json", sentinelPatient(""))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", resp.StatusCode)
	}
	id := idFromLocationHeader(resp.Header.Get("Location"))

	resp = do(http.MethodPut, "/Patient/"+id, "application/fhir+json", sentinelPatient(id))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update status = %d, want 200", resp.StatusCode)
	}
	resp = do(http.MethodDelete, "/Patient/"+id, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d, want 200", resp.StatusCode)
	}

	events := collector.all()
	if len(events) != 3 {
		t.Fatalf("audit events = %d, want 3 (create, update, delete)", len(events))
	}
	create, update, del := events[0], events[1], events[2]
	if create.Op != AuditOpFHIRCreate || create.VersionID != "1" {
		t.Errorf("create event = %+v, want fhir.create version 1", create)
	}
	if update.Op != AuditOpFHIRUpdate || update.VersionID != "2" || update.Outcome != AuditOutcomeStoredIndexed {
		t.Errorf("update event = %+v, want fhir.update version 2 stored-indexed", update)
	}
	if del.Op != AuditOpFHIRDelete || del.Outcome != AuditOutcomeDeleted || del.ResourceID != id {
		t.Errorf("delete event = %+v, want fhir.delete deleted id %s", del, id)
	}
	for _, ev := range events {
		assertNoSentinel(t, ev)
	}
}

// idFromLocationHeader extracts the resource id from a versioned FHIR Location header
// ([base]/Patient/{id}/_history/{vid}), for the audit test to address the created resource.
func idFromLocationHeader(loc string) string {
	if i := strings.Index(loc, "/_history/"); i >= 0 {
		loc = loc[:i]
	}
	segs := strings.Split(strings.Trim(loc, "/"), "/")
	if len(segs) == 0 {
		return ""
	}
	return segs[len(segs)-1]
}
