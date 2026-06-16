package server

// Tests for the FHIR role's versioning surface: the repository version store, vread,
// history-instance, the ETag / Last-Modified version headers, the If-Match precondition, and the
// $validate operation. The normative anchors cited inline are sections of the FHIR R5 HTTP page
// (hl7.org/fhir/R5/http.html): #read, #vread, #history, #create, #concurrency.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r4"
	"github.com/codeninja55/go-radx/fhir/r5"
	"github.com/codeninja55/go-radx/fhir/rest"
)

// startFHIRDaemonRepo mounts a FHIR role over the given Repository on a loopback OS-assigned port,
// the custom-repository twin of startFHIRDaemon for tests that need a repository the in-memory one
// cannot express yet (a deleted version, a multi-version history).
func startFHIRDaemonRepo(t *testing.T, release fhir.Release, repo Repository) (string, func()) {
	t.Helper()
	role, err := NewFHIRRole(repo, WithFHIRPort(0), WithFHIRRelease(release))
	if err != nil {
		t.Fatalf("NewFHIRRole: %v", err)
	}
	d, err := New(WithFHIR(role))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	waitForAddrs(t, d, "fhir@/fhir")
	base := "http://" + d.Addrs()["fhir@/fhir"].String() + "/fhir"
	cleanup := func() {
		cancelRun()
		select {
		case <-runErr:
		case <-time.After(5 * time.Second):
			t.Error("daemon did not stop within 5s")
		}
	}
	return base, cleanup
}

// createPatient POSTs a Patient and returns its server-assigned id.
func createPatient(t *testing.T, base string, release fhir.Release) string {
	t.Helper()
	status, body, _ := httpDo(t, http.MethodPost, base+"/Patient", "application/fhir+json",
		patientJSON(release, "female"))
	if status != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", status, body)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &created); err != nil || created.ID == "" {
		t.Fatalf("create body has no id (%v): %s", err, body)
	}
	return created.ID
}

// TestMemoryRepositoryVersionStore proves the repository writes version 1 on create — the stored
// resource carries meta.versionId "1" and meta.lastUpdated from the repository clock — and serves
// it back through VRead and History. The clock is pinned so the instant is asserted exactly.
func TestMemoryRepositoryVersionStore(t *testing.T) {
	at := time.Date(2026, 6, 11, 9, 30, 0, 250e6, time.UTC)
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			repo, err := NewMemoryRepository(release)
			if err != nil {
				t.Fatalf("NewMemoryRepository: %v", err)
			}
			repo.now = func() time.Time { return at }
			ctx := context.Background()

			created, err := repo.Create(ctx, newPatientResource(release))
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			id := resourceLogicalIDForTest(t, created)
			vid, lastUpdated := resourceVersionViaJSON(created)
			if vid != "1" {
				t.Errorf("created meta.versionId = %q, want %q", vid, "1")
			}
			if want := "2026-06-11T09:30:00.250Z"; lastUpdated != want {
				t.Errorf("created meta.lastUpdated = %q, want %q", lastUpdated, want)
			}

			// VRead of version 1 returns the same stored version.
			got, err := repo.VRead(ctx, "Patient", id, "1")
			if err != nil {
				t.Fatalf("VRead: %v", err)
			}
			if gotVid, _ := resourceVersionViaJSON(got); gotVid != "1" {
				t.Errorf("VRead meta.versionId = %q, want %q", gotVid, "1")
			}

			// An unknown version of an existing resource and any version of an unknown resource are
			// both ErrNotFound (the 404 split; ErrGone is reserved for deleted versions).
			if _, err := repo.VRead(ctx, "Patient", id, "99"); !errorsIsNotFound(err) {
				t.Errorf("VRead unknown version = %v, want ErrNotFound", err)
			}
			if _, err := repo.VRead(ctx, "Patient", "missing", "1"); !errorsIsNotFound(err) {
				t.Errorf("VRead unknown resource = %v, want ErrNotFound", err)
			}

			// History returns the single create version; a never-existing resource is ErrNotFound.
			versions, err := repo.History(ctx, "Patient", id)
			if err != nil {
				t.Fatalf("History: %v", err)
			}
			if len(versions) != 1 || versions[0].VersionID != "1" || versions[0].Deleted {
				t.Fatalf("History = %+v, want one non-deleted version 1", versions)
			}
			if !versions[0].LastUpdated.Equal(at) {
				t.Errorf("History lastUpdated = %v, want %v", versions[0].LastUpdated, at)
			}
			if _, err := repo.History(ctx, "Patient", "missing"); !errorsIsNotFound(err) {
				t.Errorf("History unknown resource = %v, want ErrNotFound", err)
			}
		})
	}
}

// TestMemoryRepositoryTransactionRollsBackVersions proves the version store is staged with the
// current-version map: a failed transaction leaves no orphan history behind, so a later history or
// vread cannot surface a version of a resource the transaction never committed.
func TestMemoryRepositoryTransactionRollsBackVersions(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			repo, err := NewMemoryRepository(release)
			if err != nil {
				t.Fatalf("NewMemoryRepository: %v", err)
			}
			ctx := context.Background()

			failing := failingTransactionBundle(t, release)
			if _, err := repo.Transaction(ctx, failing); err == nil {
				t.Fatal("failing transaction returned nil error, want a transaction failure")
			}
			// The failed transaction's create must have left no history entry: the staged version map
			// was discarded with the staged store. The created id would have been "1".
			if _, err := repo.History(ctx, "Patient", "1"); !errorsIsNotFound(err) {
				t.Errorf("History after rolled-back transaction = %v, want ErrNotFound (no orphan versions)", err)
			}

			// A committed transaction's create IS versioned.
			if _, err := repo.Transaction(ctx, transactionBundleResource(t, release)); err != nil {
				t.Fatalf("valid transaction: %v", err)
			}
			versions, err := repo.History(ctx, "Patient", "2")
			if err != nil {
				t.Fatalf("History after committed transaction: %v", err)
			}
			if len(versions) != 1 || versions[0].VersionID != "1" {
				t.Fatalf("committed transaction history = %+v, want one version 1", versions)
			}
		})
	}
}

// TestFHIRRoleReadAndCreateEmitVersionHeaders proves the version headers on the write and read
// paths: a create answers 201 with ETag W/"1" and Last-Modified (http.html#create: "the response
// SHALL include ... an ETag header with the versionId"), and a read of the created resource
// answers 200 with the same headers (http.html#read: "Servers SHOULD return an ETag header with
// the versionId ... and a Last-Modified header"). The ETag is the weak form W/"versionId"
// (http.html#concurrency: FHIR ETags are weak).
func TestFHIRRoleReadAndCreateEmitVersionHeaders(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			status, body, header := httpDo(t, http.MethodPost, base+"/Patient", "application/fhir+json",
				patientJSON(release, "female"))
			if status != http.StatusCreated {
				t.Fatalf("create status = %d, want 201; body=%s", status, body)
			}
			if etag := header.Get("ETag"); etag != `W/"1"` {
				t.Errorf("create ETag = %q, want %q (http.html#create)", etag, `W/"1"`)
			}
			// A versioning server's create Location names the created version:
			// [base]/[type]/[id]/_history/[vid] (http.html#create).
			if loc := header.Get("Location"); !strings.HasPrefix(loc, "/fhir/Patient/") ||
				!strings.HasSuffix(loc, "/_history/1") {
				t.Errorf("create Location = %q, want /fhir/Patient/{id}/_history/1 (http.html#create)", loc)
			}
			if lm := header.Get("Last-Modified"); lm == "" {
				t.Error("create: no Last-Modified header (http.html#create)")
			} else if _, err := time.Parse(http.TimeFormat, lm); err != nil {
				t.Errorf("create Last-Modified %q is not an HTTP date: %v", lm, err)
			}
			var created struct {
				ID   string `json:"id"`
				Meta struct {
					VersionID   string `json:"versionId"`
					LastUpdated string `json:"lastUpdated"`
				} `json:"meta"`
			}
			if err := json.Unmarshal(body, &created); err != nil {
				t.Fatalf("create body decode: %v", err)
			}
			if created.Meta.VersionID != "1" || created.Meta.LastUpdated == "" {
				t.Errorf("create body meta = %+v, want versionId 1 with lastUpdated", created.Meta)
			}

			status, _, header = httpDo(t, http.MethodGet, base+"/Patient/"+created.ID, "", nil)
			if status != http.StatusOK {
				t.Fatalf("read status = %d, want 200", status)
			}
			if etag := header.Get("ETag"); etag != `W/"1"` {
				t.Errorf("read ETag = %q, want %q (http.html#read)", etag, `W/"1"`)
			}
			if header.Get("Last-Modified") == "" {
				t.Error("read: no Last-Modified header (http.html#read)")
			}
		})
	}
}

// TestFHIRRoleVRead proves the vread interaction (http.html#vread): GET {type}/{id}/_history/{vid}
// returns that version with its version headers; an unknown version of an existing resource and a
// version of an unknown resource are both 404 OperationOutcomes. The 200-vs-404 split is driven
// through the real repository over HTTP.
func TestFHIRRoleVRead(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()
			id := createPatient(t, base, release)

			status, body, header := httpDo(t, http.MethodGet, base+"/Patient/"+id+"/_history/1", "", nil)
			if status != http.StatusOK {
				t.Fatalf("vread status = %d, want 200 (http.html#vread); body=%s", status, body)
			}
			if etag := header.Get("ETag"); etag != `W/"1"` {
				t.Errorf("vread ETag = %q, want %q", etag, `W/"1"`)
			}
			if header.Get("Last-Modified") == "" {
				t.Error("vread: no Last-Modified header")
			}
			var got struct {
				ResourceType string `json:"resourceType"`
				ID           string `json:"id"`
				Meta         struct {
					VersionID string `json:"versionId"`
				} `json:"meta"`
			}
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("vread body decode: %v", err)
			}
			if got.ResourceType != "Patient" || got.ID != id || got.Meta.VersionID != "1" {
				t.Errorf("vread body = %s, want Patient/%s version 1", body, id)
			}

			// An unknown version of an existing resource is a 404 OperationOutcome.
			status, body, _ = httpDo(t, http.MethodGet, base+"/Patient/"+id+"/_history/99", "", nil)
			if status != http.StatusNotFound {
				t.Fatalf("vread unknown version status = %d, want 404; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")

			// A version of an unknown resource is a 404 OperationOutcome.
			status, body, _ = httpDo(t, http.MethodGet, base+"/Patient/missing/_history/1", "", nil)
			if status != http.StatusNotFound {
				t.Fatalf("vread unknown resource status = %d, want 404; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")
		})
	}
}

// TestFHIRRoleVReadDeletedVersionIsGone proves the 410 path of http.html#vread ("If the version
// referred to is actually one where the resource was deleted, the server should return a 410
// status code"): a Repository that reports ErrGone for a version is answered 410 with a
// deleted-coded OperationOutcome — distinct from the 404 an unknown version gets. The in-memory
// repository cannot record a deletion yet (delete is deferred), so the deleted version comes from
// a stub Repository honouring the documented ErrGone contract.
func TestFHIRRoleVReadDeletedVersionIsGone(t *testing.T) {
	repo := &stubVersionRepo{release: fhir.R5}
	base, cleanup := startFHIRDaemonRepo(t, fhir.R5, repo)
	defer cleanup()

	status, body, _ := httpDo(t, http.MethodGet, base+"/Patient/p1/_history/3", "", nil)
	if status != http.StatusGone {
		t.Fatalf("vread deleted version status = %d, want 410 (http.html#vread); body=%s", status, body)
	}
	assertOperationOutcome(t, body, "error")
	var oo struct {
		Issue []struct {
			Code string `json:"code"`
		} `json:"issue"`
	}
	if err := json.Unmarshal(body, &oo); err != nil || len(oo.Issue) == 0 {
		t.Fatalf("410 body decode (%v): %s", err, body)
	}
	if oo.Issue[0].Code != "deleted" {
		t.Errorf("410 issue code = %q, want %q", oo.Issue[0].Code, "deleted")
	}

	// The live version still vreads 200, so the 410 is the deleted version, not the resource.
	status, _, _ = httpDo(t, http.MethodGet, base+"/Patient/p1/_history/1", "", nil)
	if status != http.StatusOK {
		t.Fatalf("vread live version status = %d, want 200", status)
	}
}

// TestFHIRRoleHistoryInstance proves the history-instance interaction (http.html#history): GET
// {type}/{id}/_history returns a history Bundle whose total counts the versions and whose single
// create version carries entry.request (POST against the type — "the request component ...
// indicates the HTTP action that would mimic the change") and entry.response (the original "201
// Created" status, the version's weak ETag, and lastModified). The Bundle must validate under the
// release validator (bdl-1 permits total on history; bdl-3 requires entry.request). A resource
// that has never existed is a 404 OperationOutcome.
func TestFHIRRoleHistoryInstance(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()
			id := createPatient(t, base, release)

			status, body, _ := httpDo(t, http.MethodGet, base+"/Patient/"+id+"/_history", "", nil)
			if status != http.StatusOK {
				t.Fatalf("history status = %d, want 200; body=%s", status, body)
			}
			var bundle struct {
				ResourceType string `json:"resourceType"`
				Type         string `json:"type"`
				Total        int    `json:"total"`
				Entry        []struct {
					FullURL  string `json:"fullUrl"`
					Resource *struct {
						ResourceType string `json:"resourceType"`
						ID           string `json:"id"`
					} `json:"resource"`
					Request *struct {
						Method string `json:"method"`
						URL    string `json:"url"`
					} `json:"request"`
					Response *struct {
						Status       string `json:"status"`
						Etag         string `json:"etag"`
						LastModified string `json:"lastModified"`
					} `json:"response"`
				} `json:"entry"`
			}
			if err := json.Unmarshal(body, &bundle); err != nil {
				t.Fatalf("history bundle decode: %v", err)
			}
			if bundle.ResourceType != "Bundle" || bundle.Type != "history" {
				t.Fatalf("history bundle = %s, want a history Bundle", body)
			}
			if bundle.Total != 1 || len(bundle.Entry) != 1 {
				t.Fatalf("history total=%d entries=%d, want 1/1", bundle.Total, len(bundle.Entry))
			}
			e := bundle.Entry[0]
			// Every history entry carries the absolute [base]/[type]/[id] fullUrl (R5 bundle.html:
			// the same fullUrl for every version of the resource).
			if want := base + "/Patient/" + id; e.FullURL != want {
				t.Errorf("history entry.fullUrl = %q, want %q (bundle.html)", e.FullURL, want)
			}
			if e.Request == nil || e.Request.Method != "POST" || e.Request.URL != "Patient" {
				t.Errorf("history entry.request = %+v, want POST Patient (the create that produced version 1)", e.Request)
			}
			if e.Response == nil || e.Response.Status != "201 Created" {
				t.Errorf("history entry.response = %+v, want status 201 Created", e.Response)
			}
			if e.Response != nil && e.Response.Etag != `W/"1"` {
				t.Errorf("history entry.response.etag = %q, want %q", e.Response.Etag, `W/"1"`)
			}
			if e.Response != nil && e.Response.LastModified == "" {
				t.Error("history entry.response.lastModified is empty")
			}
			if e.Resource == nil || e.Resource.ResourceType != "Patient" || e.Resource.ID != id {
				t.Errorf("history entry.resource = %+v, want Patient/%s", e.Resource, id)
			}

			// The served history Bundle must validate under the release validator (bdl-1/bdl-3).
			switch release {
			case fhir.R4:
				res, err := r4.UnmarshalResource(body)
				if err != nil {
					t.Fatalf("history bundle is not a decodable R4 resource: %v", err)
				}
				if oo := r4.Validate(res); oo.HasErrors() {
					t.Errorf("history Bundle fails R4 validation: %s", oo.Error())
				}
			case fhir.R5:
				res, err := r5.UnmarshalResource(body)
				if err != nil {
					t.Fatalf("history bundle is not a decodable R5 resource: %v", err)
				}
				if oo := r5.Validate(res); oo.HasErrors() {
					t.Errorf("history Bundle fails R5 validation: %s", oo.Error())
				}
			}

			// History of a resource that has never existed is a 404 OperationOutcome.
			status, body, _ = httpDo(t, http.MethodGet, base+"/Patient/missing/_history", "", nil)
			if status != http.StatusNotFound {
				t.Fatalf("history of unknown resource status = %d, want 404; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")
		})
	}
}

// TestFHIRRoleHistoryRendersUpdatesAndDeletes proves the history Bundle's per-version interaction
// derivation over a multi-version history (http.html#history: each entry mimics the change that
// produced the version): newest first, an update version renders as PUT {type}/{id} with "200 OK",
// a deleted version as DELETE {type}/{id} with "204 No Content" and no resource body, and the
// oldest version as the POST create. The multi-version history comes from a stub Repository: the
// in-memory store writes only create versions until update/delete land (wave 3).
func TestFHIRRoleHistoryRendersUpdatesAndDeletes(t *testing.T) {
	repo := &stubVersionRepo{release: fhir.R5}
	base, cleanup := startFHIRDaemonRepo(t, fhir.R5, repo)
	defer cleanup()

	status, body, _ := httpDo(t, http.MethodGet, base+"/Patient/p1/_history", "", nil)
	if status != http.StatusOK {
		t.Fatalf("history status = %d, want 200; body=%s", status, body)
	}
	var bundle struct {
		Type  string `json:"type"`
		Total int    `json:"total"`
		Entry []struct {
			FullURL  string           `json:"fullUrl"`
			Resource *json.RawMessage `json:"resource"`
			Request  struct {
				Method string `json:"method"`
				URL    string `json:"url"`
			} `json:"request"`
			Response struct {
				Status string `json:"status"`
				Etag   string `json:"etag"`
			} `json:"response"`
		} `json:"entry"`
	}
	if err := json.Unmarshal(body, &bundle); err != nil {
		t.Fatalf("history bundle decode: %v", err)
	}
	if bundle.Type != "history" || bundle.Total != 3 || len(bundle.Entry) != 3 {
		t.Fatalf("history bundle type=%q total=%d entries=%d, want history/3/3", bundle.Type, bundle.Total, len(bundle.Entry))
	}
	// Every version of the resource carries the same absolute fullUrl (R5 bundle.html), including
	// the deleted version whose entry has no resource body.
	for i, e := range bundle.Entry {
		if want := base + "/Patient/p1"; e.FullURL != want {
			t.Errorf("entry[%d].fullUrl = %q, want %q (identical across versions, present on deleted entries)", i, e.FullURL, want)
		}
	}
	// Newest first: the delete (version 3), the update (version 2), the create (version 1).
	if e := bundle.Entry[0]; e.Request.Method != "DELETE" || e.Request.URL != "Patient/p1" ||
		e.Response.Status != "204 No Content" || e.Resource != nil {
		t.Errorf("entry[0] = %+v, want a resource-less DELETE Patient/p1 204", e)
	}
	if e := bundle.Entry[1]; e.Request.Method != "PUT" || e.Request.URL != "Patient/p1" ||
		e.Response.Status != "200 OK" || e.Response.Etag != `W/"2"` || e.Resource == nil {
		t.Errorf("entry[1] = %+v, want PUT Patient/p1 200 with version 2 resource", e)
	}
	if e := bundle.Entry[2]; e.Request.Method != "POST" || e.Request.URL != "Patient" ||
		e.Response.Status != "201 Created" || e.Response.Etag != `W/"1"` || e.Resource == nil {
		t.Errorf("entry[2] = %+v, want POST Patient 201 with version 1 resource", e)
	}
}

// TestFHIRRoleHistoryInstanceHonoursCount proves the _count parameter on history-instance
// (http.html#history): the Bundle returns at most _count of the newest entries while total still
// reports the full version count. Paging links are deferred, so _count is a cap, not a page size.
// The truncation must happen after the per-version interaction derivation: with _count=1 over a
// three-version history the single entry is the newest version's DELETE — were the version list
// truncated first, the oldest survivor would be mis-rendered as the create. A _count larger than
// the history is a no-op, _count=0 returns an empty page with the honest total, and a non-numeric
// or negative _count is ignored rather than erroring (the parameter is a hint).
func TestFHIRRoleHistoryInstanceHonoursCount(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()
			id := createPatient(t, base, release)

			total, entries := historyTotals(t, base+"/Patient/"+id+"/_history?_count=5")
			if total != 1 || entries != 1 {
				t.Errorf("_count=5 over 1 version: total=%d entries=%d, want 1/1", total, entries)
			}
		})
	}

	t.Run("multi-version stub", func(t *testing.T) {
		repo := &stubVersionRepo{release: fhir.R5}
		base, cleanup := startFHIRDaemonRepo(t, fhir.R5, repo)
		defer cleanup()

		status, body, _ := httpDo(t, http.MethodGet, base+"/Patient/p1/_history?_count=1", "", nil)
		if status != http.StatusOK {
			t.Fatalf("history _count=1 status = %d, want 200; body=%s", status, body)
		}
		var bundle struct {
			Total int `json:"total"`
			Entry []struct {
				Request struct {
					Method string `json:"method"`
				} `json:"request"`
			} `json:"entry"`
		}
		if err := json.Unmarshal(body, &bundle); err != nil {
			t.Fatalf("history bundle decode: %v", err)
		}
		if bundle.Total != 3 || len(bundle.Entry) != 1 {
			t.Fatalf("_count=1 over 3 versions: total=%d entries=%d, want total 3 with 1 entry", bundle.Total, len(bundle.Entry))
		}
		// The kept entry is the newest version's DELETE, not a mis-derived create.
		if got := bundle.Entry[0].Request.Method; got != "DELETE" {
			t.Errorf("_count=1 kept entry method = %q, want DELETE (the newest version)", got)
		}

		if total, entries := historyTotals(t, base+"/Patient/p1/_history?_count=0"); total != 3 || entries != 0 {
			t.Errorf("_count=0: total=%d entries=%d, want total 3 with 0 entries", total, entries)
		}
		for _, q := range []string{"_count=abc", "_count=-1"} {
			if total, entries := historyTotals(t, base+"/Patient/p1/_history?"+q); total != 3 || entries != 3 {
				t.Errorf("%s: total=%d entries=%d, want the unfiltered 3/3", q, total, entries)
			}
		}
	})
}

// historyTotals GETs a history URL and returns the Bundle's total and entry count.
func historyTotals(t *testing.T, url string) (total, entries int) {
	t.Helper()
	status, body, _ := httpDo(t, http.MethodGet, url, "", nil)
	if status != http.StatusOK {
		t.Fatalf("history status = %d, want 200; body=%s", status, body)
	}
	var bundle struct {
		Total int               `json:"total"`
		Entry []json.RawMessage `json:"entry"`
	}
	if err := json.Unmarshal(body, &bundle); err != nil {
		t.Fatalf("history bundle decode: %v", err)
	}
	return bundle.Total, len(bundle.Entry)
}

// TestFHIRRoleIfMatchPrecondition proves the If-Match version precondition on the write
// interactions (http.html#concurrency: a version-aware update quotes the version it is based on in
// If-Match, and "if the version id given in the If-Match header does not match, the server returns
// a 412 Precondition Failed status code instead of updating the resource"). A stale If-Match is a
// 412 conflict OperationOutcome; an If-Match against an absent resource is a 412 (the named version
// cannot be the current one); a current If-Match (weak or strong form) passes the precondition and
// the update succeeds with 200. The same gate covers PUT, PATCH, and DELETE, the three version-aware
// writes.
func TestFHIRRoleIfMatchPrecondition(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()
			id := createPatient(t, base, release)

			put := func(target, ifMatch string) (int, []byte) {
				t.Helper()
				req, err := http.NewRequest(http.MethodPut, base+"/Patient/"+target,
					strings.NewReader(string(patientJSONWithID(release, target))))
				if err != nil {
					t.Fatalf("NewRequest: %v", err)
				}
				req.Header.Set("Content-Type", "application/fhir+json")
				if ifMatch != "" {
					req.Header.Set("If-Match", ifMatch)
				}
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatalf("Do: %v", err)
				}
				defer func() { _ = resp.Body.Close() }()
				var buf [4096]byte
				n, _ := resp.Body.Read(buf[:])
				return resp.StatusCode, buf[:n]
			}

			// A stale If-Match is a 412 Precondition Failed with a conflict OperationOutcome.
			status, body := put(id, `W/"999"`)
			if status != http.StatusPreconditionFailed {
				t.Fatalf("PUT stale If-Match status = %d, want 412 (http.html#concurrency); body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")

			// An If-Match against a resource that does not exist is a 412: the named version cannot be
			// the current one (there is no current version to match).
			status, body = put("missing", `W/"1"`)
			if status != http.StatusPreconditionFailed {
				t.Fatalf("PUT If-Match on missing resource status = %d, want 412; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")

			// A current If-Match passes the precondition; the update succeeds (the strong form a
			// non-FHIR-aware client sends is accepted too). The first succeeds against version 1; the
			// second names version 2 (the version the first update minted).
			for _, etag := range []string{`W/"1"`, `"2"`} {
				status, body = put(id, etag)
				if status != http.StatusOK {
					t.Fatalf("PUT current If-Match %q status = %d, want 200; body=%s", etag, status, body)
				}
			}

			// A stale If-Match on DELETE and PATCH is a 412 the same way. The PATCH carries the JSON
			// Patch content type (a PATCH with the wrong content type is a 415 before the precondition
			// is reached, so the media type is declared to exercise the precondition path).
			cases := []struct {
				method, contentType string
				body                []byte
			}{
				{http.MethodDelete, "", nil},
				{http.MethodPatch, "application/json-patch+json", []byte(`[{"op":"replace","path":"/gender","value":"male"}]`)},
			}
			for _, c := range cases {
				var rdr *strings.Reader
				if c.body != nil {
					rdr = strings.NewReader(string(c.body))
				}
				var req *http.Request
				var err error
				if rdr != nil {
					req, err = http.NewRequest(c.method, base+"/Patient/"+id, rdr)
				} else {
					req, err = http.NewRequest(c.method, base+"/Patient/"+id, nil)
				}
				if err != nil {
					t.Fatalf("NewRequest: %v", err)
				}
				if c.contentType != "" {
					req.Header.Set("Content-Type", c.contentType)
				}
				req.Header.Set("If-Match", `W/"999"`)
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatalf("Do %s: %v", c.method, err)
				}
				_ = resp.Body.Close()
				if resp.StatusCode != http.StatusPreconditionFailed {
					t.Errorf("%s stale If-Match status = %d, want 412", c.method, resp.StatusCode)
				}
			}
		})
	}
}

// TestFHIRRoleValidateOperation proves the server-side $validate (POST {type}/$validate): the body
// runs through the same release validator that gates create and the findings come back as an
// OperationOutcome with nothing persisted. A resource with no findings answers 200 with one
// informational issue (OperationOutcome.issue is 1..*); a structurally invalid resource still
// answers 200 — the validation executed; its findings are the result — carrying the error issues; a
// body that cannot be decoded, or whose type does not match the endpoint, is a 400 (the operation
// could not run); the operation is POST-only and respects the workflow-type whitelist.
func TestFHIRRoleValidateOperation(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			// A valid Patient: 200 with an all-clear informational OperationOutcome.
			status, body, _ := httpDo(t, http.MethodPost, base+"/Patient/$validate", "application/fhir+json",
				patientJSON(release, "female"))
			if status != http.StatusOK {
				t.Fatalf("$validate valid resource status = %d, want 200; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "information")

			// An Encounter missing its required status: 200 (validation ran) with error issues.
			status, body, _ = httpDo(t, http.MethodPost, base+"/Encounter/$validate", "application/fhir+json",
				encounterNoStatusJSON(t, release))
			if status != http.StatusOK {
				t.Fatalf("$validate invalid resource status = %d, want 200; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")

			// Nothing was persisted by either call.
			assertWorkflowCount(t, base, "Patient", 0)
			assertWorkflowCount(t, base, "Encounter", 0)

			// A body that is not a FHIR resource is a 400: the operation could not be performed.
			status, body, _ = httpDo(t, http.MethodPost, base+"/Patient/$validate", "application/fhir+json",
				[]byte(`{"not":"a resource"}`))
			if status != http.StatusBadRequest {
				t.Fatalf("$validate malformed body status = %d, want 400; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")

			// A type mismatch between body and endpoint is a 400, matching the create gate.
			status, body, _ = httpDo(t, http.MethodPost, base+"/Patient/$validate", "application/fhir+json",
				encounterNoStatusJSON(t, release))
			if status != http.StatusBadRequest {
				t.Fatalf("$validate type mismatch status = %d, want 400; body=%s", status, body)
			}

			// GET is a 405; an out-of-scope type is a 404; a non-FHIR content type is a 415.
			status, _, _ = httpDo(t, http.MethodGet, base+"/Patient/$validate", "", nil)
			if status != http.StatusMethodNotAllowed {
				t.Fatalf("$validate GET status = %d, want 405", status)
			}
			status, _, _ = httpDo(t, http.MethodPost, base+"/Medication/$validate", "application/fhir+json",
				[]byte(`{"resourceType":"Medication"}`))
			if status != http.StatusNotFound {
				t.Fatalf("$validate out-of-scope type status = %d, want 404", status)
			}
			status, _, _ = httpDo(t, http.MethodPost, base+"/Patient/$validate", "text/plain",
				patientJSON(release, "female"))
			if status != http.StatusUnsupportedMediaType {
				t.Fatalf("$validate text/plain status = %d, want 415", status)
			}
		})
	}
}

// TestFHIRRoleTransactionResponseCarriesVersionedLocationAndETag proves the transaction-response
// entries report the create like the direct create path does (http.html#transaction-response): each
// POST entry's response carries the versioned location ([type]/[id]/_history/[vid]) and the
// version's weak ETag, so a transaction client learns the created version exactly as a direct
// create client does from the Location and ETag headers.
func TestFHIRRoleTransactionResponseCarriesVersionedLocationAndETag(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			bundleBody, _ := json.Marshal(transactionBundleResource(t, release))
			status, body, _ := httpDo(t, http.MethodPost, base, "application/fhir+json", bundleBody)
			if status != http.StatusOK {
				t.Fatalf("transaction status = %d, want 200; body=%s", status, body)
			}
			var resp struct {
				Entry []struct {
					Response struct {
						Status   string `json:"status"`
						Location string `json:"location"`
						Etag     string `json:"etag"`
					} `json:"response"`
				} `json:"entry"`
			}
			if err := json.Unmarshal(body, &resp); err != nil {
				t.Fatalf("transaction response decode: %v", err)
			}
			if len(resp.Entry) != 1 {
				t.Fatalf("transaction response entries = %d, want 1; body=%s", len(resp.Entry), body)
			}
			e := resp.Entry[0].Response
			if e.Status != "201 Created" {
				t.Errorf("entry response status = %q, want 201 Created", e.Status)
			}
			if !strings.HasPrefix(e.Location, "Patient/") || !strings.HasSuffix(e.Location, "/_history/1") {
				t.Errorf("entry response location = %q, want Patient/{id}/_history/1 (http.html#transaction-response)", e.Location)
			}
			if e.Etag != `W/"1"` {
				t.Errorf("entry response etag = %q, want %q (http.html#transaction-response)", e.Etag, `W/"1"`)
			}
		})
	}
}

// TestFHIRRoleClientCreateResolvesVersionedLocation is the cross-component guard for the versioned
// create Location: the fhir/rest client's Create against the in-process role must resolve the
// created id from the [base]/[type]/[id]/_history/[vid] Location (idFromLocation strips the
// _history suffix) and the version from the ETag, so the versioned Location never breaks the
// shipped client's id parsing.
func TestFHIRRoleClientCreateResolvesVersionedLocation(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()
			client, err := rest.NewClient(release, base)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}

			res, err := client.Create(context.Background(), newPatientResource(release), "")
			if err != nil {
				t.Fatalf("client.Create against the in-process role: %v", err)
			}
			if res.ID == "" {
				t.Fatal("client.Create resolved no id from the versioned Location")
			}
			if !strings.HasSuffix(res.Location, "/Patient/"+res.ID+"/_history/1") {
				t.Errorf("client.Create Location = %q, want .../Patient/%s/_history/1", res.Location, res.ID)
			}
			if res.VersionID != "1" {
				t.Errorf("client.Create VersionID = %q, want %q", res.VersionID, "1")
			}

			// The resolved id addresses the created resource: the round trip proves the id is real,
			// not a mis-parsed "_history" segment.
			got, err := client.Read(context.Background(), "Patient", res.ID)
			if err != nil {
				t.Fatalf("client.Read of the created id: %v", err)
			}
			if got.Resource.ResourceType() != "Patient" {
				t.Errorf("read-back resourceType = %q, want Patient", got.Resource.ResourceType())
			}
		})
	}
}

// TestFHIRRoleClientVReadAndHistory drives the real fhir/rest client's VRead and History against
// the in-process role, the end-to-end proof that the shipped client and the shipped server agree
// on the vread and history wire shapes (the same interop guard the transaction test provides).
func TestFHIRRoleClientVReadAndHistory(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()
			id := createPatient(t, base, release)

			client, err := rest.NewClient(release, base)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}

			got, err := client.VRead(context.Background(), "Patient", id, "1")
			if err != nil {
				t.Fatalf("client.VRead against the in-process role: %v", err)
			}
			if got.Resource.ResourceType() != "Patient" {
				t.Errorf("VRead resourceType = %q, want Patient", got.Resource.ResourceType())
			}

			history, err := client.History(context.Background(), "Patient", id)
			if err != nil {
				t.Fatalf("client.History against the in-process role: %v", err)
			}
			out, _ := json.Marshal(history)
			var decoded struct {
				Type  string `json:"type"`
				Total int    `json:"total"`
			}
			if err := json.Unmarshal(out, &decoded); err != nil {
				t.Fatalf("history decode: %v", err)
			}
			if decoded.Type != "history" || decoded.Total != 1 {
				t.Errorf("client history type=%q total=%d, want history/1", decoded.Type, decoded.Total)
			}
		})
	}
}

// --- helpers ---

func errorsIsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

func errorsIsGone(err error) bool { return errors.Is(err, ErrGone) }

// newPatientResource builds a minimal Patient of the release for repository-level tests.
func newPatientResource(release fhir.Release) fhir.Resource {
	switch release {
	case fhir.R4:
		g := r4.AdministrativeGender("female")
		return &r4.Patient{Gender: &g}
	default:
		g := r5.AdministrativeGender("female")
		return &r5.Patient{Gender: &g}
	}
}

// stubVersionRepo is a Repository with a fixed three-version R5 history for Patient/p1 — version 1
// (create), version 2 (update), version 3 (deleted) — so the handler paths the in-memory
// repository cannot reach yet (a 410 vread, an update/delete history entry) are tested against the
// documented Repository contract. Write and search methods are unused by these tests.
type stubVersionRepo struct {
	release fhir.Release
}

func (s *stubVersionRepo) patientVersion(versionID, lastUpdated string) fhir.Resource {
	g := r5.AdministrativeGender("female")
	id := "p1"
	return &r5.Patient{
		DomainResource: r5.DomainResource{
			ID:   &id,
			Meta: &r5.Meta{VersionId: &versionID, LastUpdated: &lastUpdated},
		},
		Gender: &g,
	}
}

func (s *stubVersionRepo) versions() []ResourceVersion {
	t1 := time.Date(2026, 6, 11, 8, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Hour)
	t3 := t2.Add(time.Hour)
	return []ResourceVersion{
		{VersionID: "3", LastUpdated: t3, Deleted: true},
		{Resource: s.patientVersion("2", fhirInstant(t2)), VersionID: "2", LastUpdated: t2},
		{Resource: s.patientVersion("1", fhirInstant(t1)), VersionID: "1", LastUpdated: t1},
	}
}

func (s *stubVersionRepo) Read(_ context.Context, resourceType, id string) (fhir.Resource, error) {
	if resourceType != "Patient" || id != "p1" {
		return nil, fmt.Errorf("%w: %s/%s", ErrNotFound, resourceType, id)
	}
	return s.patientVersion("2", fhirInstant(time.Date(2026, 6, 11, 9, 0, 0, 0, time.UTC))), nil
}

func (s *stubVersionRepo) VRead(_ context.Context, resourceType, id, versionID string) (fhir.Resource, error) {
	if resourceType != "Patient" || id != "p1" {
		return nil, fmt.Errorf("%w: %s/%s", ErrNotFound, resourceType, id)
	}
	for _, v := range s.versions() {
		if v.VersionID != versionID {
			continue
		}
		if v.Deleted {
			return nil, fmt.Errorf("%w: %s/%s/_history/%s", ErrGone, resourceType, id, versionID)
		}
		return v.Resource, nil
	}
	return nil, fmt.Errorf("%w: %s/%s/_history/%s", ErrNotFound, resourceType, id, versionID)
}

func (s *stubVersionRepo) History(_ context.Context, resourceType, id string) ([]ResourceVersion, error) {
	if resourceType != "Patient" || id != "p1" {
		return nil, fmt.Errorf("%w: %s/%s", ErrNotFound, resourceType, id)
	}
	return s.versions(), nil
}

func (s *stubVersionRepo) Create(context.Context, fhir.Resource) (fhir.Resource, error) {
	return nil, fmt.Errorf("stubVersionRepo: create is not supported")
}

func (s *stubVersionRepo) Update(context.Context, string, string, fhir.Resource, string) (fhir.Resource, bool, error) {
	return nil, false, fmt.Errorf("stubVersionRepo: update is not supported")
}

func (s *stubVersionRepo) Delete(context.Context, string, string, string) (bool, error) {
	return false, fmt.Errorf("stubVersionRepo: delete is not supported")
}

func (s *stubVersionRepo) Search(context.Context, string, url.Values) (fhir.Resource, error) {
	return nil, fmt.Errorf("stubVersionRepo: search is not supported")
}

func (s *stubVersionRepo) Transaction(context.Context, fhir.Resource) (fhir.Resource, error) {
	return nil, fmt.Errorf("stubVersionRepo: transactions are not supported")
}
