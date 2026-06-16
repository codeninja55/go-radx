package server

// Tests for the FHIR role's write interactions beyond create: update (PUT), patch (PATCH), delete
// (DELETE), and their conditional forms. The normative anchors are the FHIR R5 HTTP page
// (hl7.org/fhir/R5/http.html): #update, #patch, #delete, #cond-update, #cond-patch, #cond-delete,
// #concurrency. Status codes match HAPI's defaults (update-as-create allowed; idempotent delete).

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r4"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// TestMemoryRepositoryUpdateAndDelete proves the version store appends versions on update and delete:
// create writes version 1, update writes version 2 (a non-deleted entry), delete writes version 3
// (a deletion entry), Read then reports ErrGone, the prior versions stay VRead-able, and a later
// update resurrects the resource as version 4. It is the repository-level companion to the HTTP
// delete-sequence test.
func TestMemoryRepositoryUpdateAndDelete(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			repo, err := NewMemoryRepository(release)
			if err != nil {
				t.Fatalf("NewMemoryRepository: %v", err)
			}
			ctx := context.Background()

			created, err := repo.Create(ctx, newPatientResource(release))
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			id := resourceLogicalIDForTest(t, created)

			// Update: version 2, created=false.
			updated, wasCreated, err := repo.Update(ctx, "Patient", id, newPatientResource(release), "")
			if err != nil {
				t.Fatalf("Update: %v", err)
			}
			if wasCreated {
				t.Error("Update of existing resource reported created=true")
			}
			if vid, _ := resourceVersionViaJSON(updated); vid != "2" {
				t.Errorf("update versionId = %q, want 2", vid)
			}

			// Delete: existed=true; Read is now ErrGone; prior versions still VRead-able.
			existed, err := repo.Delete(ctx, "Patient", id, "")
			if err != nil || !existed {
				t.Fatalf("Delete = (%v, %v), want (true, nil)", existed, err)
			}
			if _, err := repo.Read(ctx, "Patient", id); err == nil || !isGone(err) {
				t.Errorf("Read after delete = %v, want ErrGone", err)
			}
			if _, err := repo.VRead(ctx, "Patient", id, "1"); err != nil {
				t.Errorf("VRead version 1 after delete = %v, want it to remain readable", err)
			}

			// History shows three versions, newest first, with the deletion newest.
			versions, err := repo.History(ctx, "Patient", id)
			if err != nil {
				t.Fatalf("History: %v", err)
			}
			if len(versions) != 3 || !versions[0].Deleted || versions[0].VersionID != "3" {
				t.Fatalf("History = %+v, want three versions with a deleted version 3 newest", versions)
			}

			// Idempotent re-delete: existed=false, no error.
			if existed, err := repo.Delete(ctx, "Patient", id, ""); err != nil || existed {
				t.Errorf("re-delete = (%v, %v), want (false, nil)", existed, err)
			}

			// Resurrect via update: version 4, created=true.
			res, wasCreated, err := repo.Update(ctx, "Patient", id, newPatientResource(release), "")
			if err != nil {
				t.Fatalf("resurrect Update: %v", err)
			}
			if !wasCreated {
				t.Error("update after delete reported created=false; a resurrection is a create")
			}
			if vid, _ := resourceVersionViaJSON(res); vid != "4" {
				t.Errorf("resurrect versionId = %q, want 4 (monotonic)", vid)
			}
		})
	}
}

// isGone reports whether err is the ErrGone sentinel, the repository's deleted-version signal.
func isGone(err error) bool { return err != nil && errorsIsGone(err) }

// patientJSONGender builds a Patient carrying the given gender, the conditional-update/patch payload
// that has no id (the server resolves the target from the criteria, not the body).
func patientJSONGender(release fhir.Release, gender string) []byte {
	return patientJSON(release, gender)
}

// patientJSONWithIDGender builds a Patient carrying both a client id and a gender, the update payload
// that addresses an existing resource by id and changes a field.
func patientJSONWithIDGender(release fhir.Release, id, gender string) []byte {
	switch release {
	case fhir.R4:
		g := r4.AdministrativeGender(gender)
		b, _ := json.Marshal(&r4.Patient{DomainResource: r4.DomainResource{ID: &id}, Gender: &g})
		return b
	default:
		g := r5.AdministrativeGender(gender)
		b, _ := json.Marshal(&r5.Patient{DomainResource: r5.DomainResource{ID: &id}, Gender: &g})
		return b
	}
}

// encounterJSON builds a valid Encounter of the release (with its required elements), for the
// type-mismatch and validation tests. R4 requires status and class; R5 requires status only.
func encounterJSON(t *testing.T, release fhir.Release) []byte {
	t.Helper()
	switch release {
	case fhir.R4:
		s := r4.EncounterStatus("finished")
		code := "AMB"
		b, _ := json.Marshal(&r4.Encounter{Status: &s, Class: &r4.Coding{Code: &code}})
		return b
	default:
		s := r5.EncounterStatus("completed")
		b, _ := json.Marshal(&r5.Encounter{Status: &s})
		return b
	}
}

// httpDoHeaders issues a request with arbitrary headers and returns the status, body, and response
// headers — the write tests need to set If-Match, which httpDo does not.
func httpDoHeaders(t *testing.T, method, url, contentType string, body []byte, headers map[string]string) (int, []byte, http.Header) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		rdr = strings.NewReader(string(body))
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do %s %s: %v", method, url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, out, resp.Header
}

// TestFHIRRoleUpdateExisting proves a PUT to an existing resource replaces it and bumps the version:
// the response is 200 with the new ETag/Last-Modified, and a subsequent read sees version 2.
func TestFHIRRoleUpdateExisting(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()
			id := createPatient(t, base, release)

			status, body, header := httpDoHeaders(t, http.MethodPut, base+"/Patient/"+id,
				"application/fhir+json", patientJSONWithIDGender(release, id, "male"), nil)
			if status != http.StatusOK {
				t.Fatalf("update status = %d, want 200; body=%s", status, body)
			}
			if etag := header.Get("ETag"); etag != `W/"2"` {
				t.Errorf("update ETag = %q, want W/\"2\"", etag)
			}
			if header.Get("Last-Modified") == "" {
				t.Error("update response is missing Last-Modified")
			}

			// The read sees the updated version and the new gender.
			status, body, header = httpDo(t, http.MethodGet, base+"/Patient/"+id, "", nil)
			if status != http.StatusOK {
				t.Fatalf("read after update status = %d, want 200; body=%s", status, body)
			}
			if etag := header.Get("ETag"); etag != `W/"2"` {
				t.Errorf("read ETag after update = %q, want W/\"2\"", etag)
			}
			var got struct {
				Gender string `json:"gender"`
			}
			_ = json.Unmarshal(body, &got)
			if got.Gender != "male" {
				t.Errorf("gender after update = %q, want male", got.Gender)
			}
		})
	}
}

// TestFHIRRoleUpdateResourceTypeIDMismatch proves the update body's resourceType and id must match
// the URL: a body whose resourceType differs from the endpoint, or whose id contradicts the URL id,
// is a 400 OperationOutcome — never a silent overwrite of the wrong resource.
func TestFHIRRoleUpdateResourceTypeIDMismatch(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()
			id := createPatient(t, base, release)

			// Body id contradicts the URL id.
			status, body, _ := httpDoHeaders(t, http.MethodPut, base+"/Patient/"+id,
				"application/fhir+json", patientJSONWithID(release, "other-id"), nil)
			if status != http.StatusBadRequest {
				t.Fatalf("update id-mismatch status = %d, want 400; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")

			// Body resourceType differs from the endpoint (an Encounter PUT to a Patient URL).
			status, body, _ = httpDoHeaders(t, http.MethodPut, base+"/Patient/"+id,
				"application/fhir+json", encounterJSON(t, release), nil)
			if status != http.StatusBadRequest {
				t.Fatalf("update type-mismatch status = %d, want 400; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")
		})
	}
}

// TestFHIRRolePatchJSONPatch proves a JSON Patch (RFC 6902) PATCH applies to the current version,
// is re-validated, and bumps the version (200). A patch with the wrong content type is a 415; a
// patch that does not apply (a bad path) is a 422.
func TestFHIRRolePatchJSONPatch(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()
			id := createPatient(t, base, release)

			patch := []byte(`[{"op":"replace","path":"/gender","value":"male"}]`)
			status, body, header := httpDoHeaders(t, http.MethodPatch, base+"/Patient/"+id,
				"application/json-patch+json", patch, nil)
			if status != http.StatusOK {
				t.Fatalf("patch status = %d, want 200; body=%s", status, body)
			}
			if etag := header.Get("ETag"); etag != `W/"2"` {
				t.Errorf("patch ETag = %q, want W/\"2\"", etag)
			}
			var got struct {
				Gender string `json:"gender"`
			}
			_ = json.Unmarshal(body, &got)
			if got.Gender != "male" {
				t.Errorf("gender after patch = %q, want male", got.Gender)
			}

			// Wrong content type is a 415.
			status, body, _ = httpDoHeaders(t, http.MethodPatch, base+"/Patient/"+id,
				"application/fhir+json", patch, nil)
			if status != http.StatusUnsupportedMediaType {
				t.Fatalf("patch wrong content type status = %d, want 415; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")

			// A patch that does not apply (a path into a non-existent member) is a 422.
			badPatch := []byte(`[{"op":"replace","path":"/nope/0","value":"x"}]`)
			status, body, _ = httpDoHeaders(t, http.MethodPatch, base+"/Patient/"+id,
				"application/json-patch+json", badPatch, nil)
			if status != http.StatusUnprocessableEntity {
				t.Fatalf("patch bad-path status = %d, want 422; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")
		})
	}
}

// TestFHIRRolePatchValidatesResult proves the patched resource is re-validated through the same
// release validator that gates create: a patch that removes a required element (Encounter.status)
// is a 422, and nothing is persisted (the resource keeps its prior version).
func TestFHIRRolePatchValidatesResult(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			// Create an Encounter (with its required status) so there is something to patch.
			status, body, _ := httpDo(t, http.MethodPost, base+"/Encounter", "application/fhir+json",
				encounterJSON(t, release))
			if status != http.StatusCreated {
				t.Fatalf("create Encounter status = %d, want 201; body=%s", status, body)
			}
			var created struct {
				ID string `json:"id"`
			}
			_ = json.Unmarshal(body, &created)

			// Removing the required status must fail validation: a 422, nothing persisted.
			patch := []byte(`[{"op":"remove","path":"/status"}]`)
			status, body, _ = httpDoHeaders(t, http.MethodPatch, base+"/Encounter/"+created.ID,
				"application/json-patch+json", patch, nil)
			if status != http.StatusUnprocessableEntity {
				t.Fatalf("patch removing required status = %d, want 422; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")

			// The resource still reads at its original (version 1) state.
			status, _, header := httpDo(t, http.MethodGet, base+"/Encounter/"+created.ID, "", nil)
			if status != http.StatusOK {
				t.Fatalf("read after failed patch status = %d, want 200", status)
			}
			if etag := header.Get("ETag"); etag != `W/"1"` {
				t.Errorf("ETag after failed patch = %q, want W/\"1\" (nothing persisted)", etag)
			}
		})
	}
}

// TestFHIRRoleDeleteSequence is the full delete lifecycle: a delete retires the current version
// (200), a subsequent read is 410 Gone, a vread of a prior version still returns 200, the history
// shows the deletion entry, and a second delete is idempotent (204, no error).
func TestFHIRRoleDeleteSequence(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()
			id := createPatient(t, base, release)

			// Delete the live resource: 200, the resource existed.
			status, body, _ := httpDo(t, http.MethodDelete, base+"/Patient/"+id, "", nil)
			if status != http.StatusOK {
				t.Fatalf("delete status = %d, want 200; body=%s", status, body)
			}

			// A read of a deleted resource is 410 Gone.
			status, body, _ = httpDo(t, http.MethodGet, base+"/Patient/"+id, "", nil)
			if status != http.StatusGone {
				t.Fatalf("read after delete status = %d, want 410 Gone; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")

			// A vread of the prior (version 1) still returns 200.
			status, body, _ = httpDo(t, http.MethodGet, base+"/Patient/"+id+"/_history/1", "", nil)
			if status != http.StatusOK {
				t.Fatalf("vread prior version after delete status = %d, want 200; body=%s", status, body)
			}

			// The history shows the deletion as a resource-less DELETE entry, newest first.
			status, body, _ = httpDo(t, http.MethodGet, base+"/Patient/"+id+"/_history", "", nil)
			if status != http.StatusOK {
				t.Fatalf("history after delete status = %d, want 200; body=%s", status, body)
			}
			var hist struct {
				Total int `json:"total"`
				Entry []struct {
					Resource json.RawMessage `json:"resource"`
					Request  struct {
						Method string `json:"method"`
					} `json:"request"`
				} `json:"entry"`
			}
			if err := json.Unmarshal(body, &hist); err != nil {
				t.Fatalf("history decode: %v; body=%s", err, body)
			}
			if hist.Total != 2 {
				t.Errorf("history total after delete = %d, want 2 (create + delete)", hist.Total)
			}
			if len(hist.Entry) == 0 || hist.Entry[0].Request.Method != "DELETE" || len(hist.Entry[0].Resource) != 0 {
				t.Errorf("newest history entry = %+v, want a resource-less DELETE entry", hist.Entry)
			}

			// A second delete is idempotent: 204, no error.
			status, body, _ = httpDo(t, http.MethodDelete, base+"/Patient/"+id, "", nil)
			if status != http.StatusNoContent {
				t.Fatalf("idempotent re-delete status = %d, want 204; body=%s", status, body)
			}

			// A delete of a never-existing resource is also idempotent: 204.
			status, _, _ = httpDo(t, http.MethodDelete, base+"/Patient/never", "", nil)
			if status != http.StatusNoContent {
				t.Fatalf("delete of absent resource status = %d, want 204", status)
			}
		})
	}
}

// TestFHIRRoleUpdateResurrectsDeleted proves a PUT after a DELETE brings the resource back as a new
// version (FHIR R5 http.html#delete: a server may allow a deleted resource to be brought back by a
// subsequent update). The resurrected resource reads 200 again and its history is monotonic.
func TestFHIRRoleUpdateResurrectsDeleted(t *testing.T) {
	base, cleanup := startFHIRDaemon(t, fhir.R5)
	defer cleanup()
	id := createPatient(t, base, fhir.R5)

	if status, body, _ := httpDo(t, http.MethodDelete, base+"/Patient/"+id, "", nil); status != http.StatusOK {
		t.Fatalf("delete status = %d, want 200; body=%s", status, body)
	}
	// PUT after delete: a create-on-update resurrection answering 201.
	status, body, _ := httpDoHeaders(t, http.MethodPut, base+"/Patient/"+id,
		"application/fhir+json", patientJSONWithID(fhir.R5, id), nil)
	if status != http.StatusCreated {
		t.Fatalf("resurrect via PUT status = %d, want 201; body=%s", status, body)
	}
	// The resource reads 200 again at version 3 (create=1, delete=2, resurrect=3).
	status, _, header := httpDo(t, http.MethodGet, base+"/Patient/"+id, "", nil)
	if status != http.StatusOK {
		t.Fatalf("read after resurrect status = %d, want 200", status)
	}
	if etag := header.Get("ETag"); etag != `W/"3"` {
		t.Errorf("ETag after resurrect = %q, want W/\"3\" (monotonic history)", etag)
	}
}

// TestFHIRRoleConditionalUpdate proves a conditional update (PUT [type]?_id=...) resolves the
// criteria: one match updates that resource (200), no match creates (201), and many matches is a
// 412. Resolution uses the MemoryRepository search, which matches _id, so the criteria here are an
// _id search — the documented conditional scope against the dev repository.
func TestFHIRRoleConditionalUpdate(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()
			id := createPatient(t, base, release)

			// One match: PUT [type]?_id={id} updates the matched resource (200).
			status, body, _ := httpDoHeaders(t, http.MethodPut, base+"/Patient?_id="+id,
				"application/fhir+json", patientJSONGender(release, "male"), nil)
			if status != http.StatusOK {
				t.Fatalf("conditional update one-match status = %d, want 200; body=%s", status, body)
			}
			var got struct {
				ID     string `json:"id"`
				Gender string `json:"gender"`
			}
			_ = json.Unmarshal(body, &got)
			if got.ID != id || got.Gender != "male" {
				t.Errorf("conditional update result = %+v, want id %s gender male", got, id)
			}

			// No match: PUT [type]?_id=absent creates (201).
			status, body, _ = httpDoHeaders(t, http.MethodPut, base+"/Patient?_id=absent",
				"application/fhir+json", patientJSONGender(release, "female"), nil)
			if status != http.StatusCreated {
				t.Fatalf("conditional update no-match status = %d, want 201; body=%s", status, body)
			}

			// Missing criteria: a conditional interaction with no query is a 400.
			status, body, _ = httpDoHeaders(t, http.MethodPut, base+"/Patient",
				"application/fhir+json", patientJSONGender(release, "female"), nil)
			if status != http.StatusBadRequest {
				t.Fatalf("conditional update no-criteria status = %d, want 400; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")
		})
	}
}

// TestFHIRRoleConditionalUpdateMultipleMatch proves a conditional update whose criteria match more
// than one resource is a 412 (the criteria are not selective enough). The criteria here select
// all-of-type (no _id), which against the dev repository's all-of-type fallback resolves to the two
// created resources.
func TestFHIRRoleConditionalUpdateMultipleMatch(t *testing.T) {
	base, cleanup := startFHIRDaemon(t, fhir.R5)
	defer cleanup()
	createPatient(t, base, fhir.R5)
	createPatient(t, base, fhir.R5)

	// A non-_id criterion the dev repository ignores resolves to all-of-type: two matches.
	status, body, _ := httpDoHeaders(t, http.MethodPut, base+"/Patient?gender=female",
		"application/fhir+json", patientJSONGender(fhir.R5, "male"), nil)
	if status != http.StatusPreconditionFailed {
		t.Fatalf("conditional update multi-match status = %d, want 412; body=%s", status, body)
	}
	assertOperationOutcome(t, body, "error")
}

// TestFHIRRoleConditionalDelete proves a conditional delete (DELETE [type]?_id=...): one match is
// deleted (200), no match is an idempotent 204, and many matches is a 412.
func TestFHIRRoleConditionalDelete(t *testing.T) {
	base, cleanup := startFHIRDaemon(t, fhir.R5)
	defer cleanup()
	id := createPatient(t, base, fhir.R5)

	// One match: deleted (200).
	status, body, _ := httpDo(t, http.MethodDelete, base+"/Patient?_id="+id, "", nil)
	if status != http.StatusOK {
		t.Fatalf("conditional delete one-match status = %d, want 200; body=%s", status, body)
	}
	// The resource is now gone.
	if status, _, _ := httpDo(t, http.MethodGet, base+"/Patient/"+id, "", nil); status != http.StatusGone {
		t.Errorf("read after conditional delete status = %d, want 410", status)
	}

	// No match: idempotent 204.
	status, body, _ = httpDo(t, http.MethodDelete, base+"/Patient?_id=absent", "", nil)
	if status != http.StatusNoContent {
		t.Fatalf("conditional delete no-match status = %d, want 204; body=%s", status, body)
	}

	// Many matches: 412.
	createPatient(t, base, fhir.R5)
	createPatient(t, base, fhir.R5)
	status, body, _ = httpDo(t, http.MethodDelete, base+"/Patient?gender=female", "", nil)
	if status != http.StatusPreconditionFailed {
		t.Fatalf("conditional delete multi-match status = %d, want 412; body=%s", status, body)
	}
	assertOperationOutcome(t, body, "error")
}

// TestFHIRRoleConditionalPatch proves a conditional patch (PATCH [type]?_id=...): one match is
// patched (200), no match is a 404, and many matches is a 412.
func TestFHIRRoleConditionalPatch(t *testing.T) {
	base, cleanup := startFHIRDaemon(t, fhir.R5)
	defer cleanup()
	id := createPatient(t, base, fhir.R5)
	patch := []byte(`[{"op":"replace","path":"/gender","value":"male"}]`)

	// One match: patched (200).
	status, body, _ := httpDoHeaders(t, http.MethodPatch, base+"/Patient?_id="+id,
		"application/json-patch+json", patch, nil)
	if status != http.StatusOK {
		t.Fatalf("conditional patch one-match status = %d, want 200; body=%s", status, body)
	}

	// No match: 404.
	status, body, _ = httpDoHeaders(t, http.MethodPatch, base+"/Patient?_id=absent",
		"application/json-patch+json", patch, nil)
	if status != http.StatusNotFound {
		t.Fatalf("conditional patch no-match status = %d, want 404; body=%s", status, body)
	}
	assertOperationOutcome(t, body, "error")

	// Many matches: 412.
	createPatient(t, base, fhir.R5)
	status, body, _ = httpDoHeaders(t, http.MethodPatch, base+"/Patient?gender=female",
		"application/json-patch+json", patch, nil)
	if status != http.StatusPreconditionFailed {
		t.Fatalf("conditional patch multi-match status = %d, want 412; body=%s", status, body)
	}
	assertOperationOutcome(t, body, "error")
}

// TestApplyRFC6902 unit-tests the JSON Patch applier against the RFC 6902 operation set: add,
// replace, remove, move, copy, and test, plus the failure modes (a failed test, a bad path, an
// out-of-range array index). It is a direct test of the applier, independent of the HTTP role.
func TestApplyRFC6902(t *testing.T) {
	doc := []byte(`{"a":1,"b":{"c":2},"d":["x","y"]}`)
	cases := []struct {
		name    string
		patch   string
		want    string
		wantErr bool
	}{
		{"replace", `[{"op":"replace","path":"/a","value":9}]`, `{"a":9,"b":{"c":2},"d":["x","y"]}`, false},
		{"add member", `[{"op":"add","path":"/e","value":true}]`, `{"a":1,"b":{"c":2},"d":["x","y"],"e":true}`, false},
		{"add nested", `[{"op":"add","path":"/b/f","value":3}]`, `{"a":1,"b":{"c":2,"f":3},"d":["x","y"]}`, false},
		{"add array index", `[{"op":"add","path":"/d/1","value":"z"}]`, `{"a":1,"b":{"c":2},"d":["x","z","y"]}`, false},
		{"add array append", `[{"op":"add","path":"/d/-","value":"z"}]`, `{"a":1,"b":{"c":2},"d":["x","y","z"]}`, false},
		{"remove member", `[{"op":"remove","path":"/a"}]`, `{"b":{"c":2},"d":["x","y"]}`, false},
		{"remove array elem", `[{"op":"remove","path":"/d/0"}]`, `{"a":1,"b":{"c":2},"d":["y"]}`, false},
		{"move", `[{"op":"move","from":"/a","path":"/b/moved"}]`, `{"b":{"c":2,"moved":1},"d":["x","y"]}`, false},
		{"copy", `[{"op":"copy","from":"/a","path":"/b/cp"}]`, `{"a":1,"b":{"c":2,"cp":1},"d":["x","y"]}`, false},
		{"test ok", `[{"op":"test","path":"/a","value":1},{"op":"replace","path":"/a","value":5}]`, `{"a":5,"b":{"c":2},"d":["x","y"]}`, false},
		{"test fail", `[{"op":"test","path":"/a","value":2}]`, "", true},
		{"bad path", `[{"op":"replace","path":"/nope","value":1}]`, "", true},
		{"array out of range", `[{"op":"remove","path":"/d/9"}]`, "", true},
		{"unknown op", `[{"op":"frobnicate","path":"/a"}]`, "", true},
		{"empty patch", `[]`, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := applyRFC6902(doc, []byte(c.patch))
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %s", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !jsonEqual(t, got, []byte(c.want)) {
				t.Errorf("patch result = %s, want %s", got, c.want)
			}
		})
	}
}

// TestApplyRFC6902PreservesNumericLexicals proves the JSON Patch applier does not rewrite numbers it
// did not touch (the P1-B regression): a FHIR decimal carries its lexical form (1.00, 1.230 — the
// trailing zeros are significant precision FHIR mandates be preserved) and a 64-bit integer exceeds
// float64's exact range (2^53). Patching an unrelated field must leave both byte-for-byte intact;
// decoding the whole document through float64 (the prior bug) would round 1.00 to 1 and lose
// precision past 2^53. It also confirms an add/replace operand keeps its own decimal lexical form.
func TestApplyRFC6902PreservesNumericLexicals(t *testing.T) {
	// valueQuantity.value is a FHIR decimal (1.00); a Money.value 1.230; an explicit 64-bit integer
	// beyond 2^53. All sit beside the field the patch changes.
	const big = "9223372036854775807" // math.MaxInt64, past float64's exact range
	doc := []byte(`{` +
		`"resourceType":"Observation",` +
		`"status":"final",` +
		`"valueQuantity":{"value":1.00,"unit":"mg"},` +
		`"extra":{"money":1.230,"i64":` + big + `}` +
		`}`)

	patch := []byte(`[{"op":"replace","path":"/status","value":"amended"}]`)
	got, err := applyRFC6902(doc, patch)
	if err != nil {
		t.Fatalf("applyRFC6902: %v", err)
	}

	// The patched field changed; every number is byte-for-byte preserved.
	if !jsonContainsRaw(t, got, `"value":1.00`) {
		t.Errorf("decimal 1.00 was rewritten; result=%s", got)
	}
	if !jsonContainsRaw(t, got, `"money":1.230`) {
		t.Errorf("decimal 1.230 was rewritten; result=%s", got)
	}
	if !jsonContainsRaw(t, got, `"i64":`+big) {
		t.Errorf("int64 %s lost precision; result=%s", big, got)
	}
	if !jsonContainsRaw(t, got, `"status":"amended"`) {
		t.Errorf("patched field not applied; result=%s", got)
	}

	// An add/replace operand keeps its own decimal lexical form too.
	got2, err := applyRFC6902(doc, []byte(`[{"op":"replace","path":"/valueQuantity/value","value":2.50}]`))
	if err != nil {
		t.Fatalf("applyRFC6902 operand: %v", err)
	}
	if !jsonContainsRaw(t, got2, `"value":2.50`) {
		t.Errorf("operand decimal 2.50 was rewritten; result=%s", got2)
	}
	// The untouched int64 still round-trips after a different patch.
	if !jsonContainsRaw(t, got2, `"i64":`+big) {
		t.Errorf("int64 %s lost precision on operand patch; result=%s", big, got2)
	}
}

// jsonContainsRaw reports whether the compact form of doc contains the literal substring want, used
// to assert a number's exact lexical form survived (json.Marshal emits no insignificant whitespace,
// so a "key":number substring is stable).
func jsonContainsRaw(t *testing.T, doc []byte, want string) bool {
	t.Helper()
	return strings.Contains(string(doc), want)
}

// TestFHIRRoleUpdateAsCreateReservesID proves update-as-create reserves the server id counter (the
// P1-A regression): a PUT to /Patient/1 stores a resource at the client-chosen numeric id 1, and a
// later POST must mint a fresh id (not 1), never overwriting the PUT-created resource. Before the
// fix the counter was not advanced, so the POST re-minted "1" and clobbered the PUT resource with a
// second version-1 history entry.
func TestFHIRRoleUpdateAsCreateReservesID(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			// PUT /Patient/1: update-as-create at the client-chosen id 1 (201).
			status, body, _ := httpDoHeaders(t, http.MethodPut, base+"/Patient/1", "application/fhir+json",
				patientJSONWithIDGender(release, "1", "male"), nil)
			if status != http.StatusCreated {
				t.Fatalf("PUT /Patient/1 status = %d, want 201; body=%s", status, body)
			}

			// POST /Patient: the server mints an id; it must not be "1" (the reserved id).
			status, body, _ = httpDo(t, http.MethodPost, base+"/Patient", "application/fhir+json",
				patientJSON(release, "female"))
			if status != http.StatusCreated {
				t.Fatalf("POST /Patient status = %d, want 201; body=%s", status, body)
			}
			var posted struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(body, &posted); err != nil {
				t.Fatalf("POST body decode: %v", err)
			}
			if posted.ID == "1" {
				t.Fatalf("POST re-minted reserved id 1, overwriting the PUT resource; body=%s", body)
			}

			// The PUT resource at /Patient/1 is intact: still version 1, still gender male, history has
			// exactly one version (no clobbering second version-1 entry).
			status, body, _ = httpDo(t, http.MethodGet, base+"/Patient/1", "", nil)
			if status != http.StatusOK {
				t.Fatalf("GET /Patient/1 after POST status = %d, want 200; body=%s", status, body)
			}
			var got struct {
				ID     string `json:"id"`
				Gender string `json:"gender"`
				Meta   struct {
					VersionID string `json:"versionId"`
				} `json:"meta"`
			}
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("GET body decode: %v", err)
			}
			if got.ID != "1" || got.Gender != "male" || got.Meta.VersionID != "1" {
				t.Fatalf("PUT resource was disturbed: id=%q gender=%q version=%q; body=%s",
					got.ID, got.Gender, got.Meta.VersionID, body)
			}
			status, body, _ = httpDo(t, http.MethodGet, base+"/Patient/1/_history", "", nil)
			if status != http.StatusOK {
				t.Fatalf("history status = %d, want 200; body=%s", status, body)
			}
			if total, ok := bundleTotalForTest(body); !ok || total != 1 {
				t.Fatalf("history total = %d (ok=%v), want 1 (no clobbering version); body=%s", total, ok, body)
			}
		})
	}
}

// bundleTotalForTest reads a Bundle's total element from its JSON, for asserting a history length.
func bundleTotalForTest(body []byte) (int, bool) {
	var env struct {
		Total *int `json:"total"`
	}
	if err := json.Unmarshal(body, &env); err != nil || env.Total == nil {
		return 0, false
	}
	return *env.Total, true
}

// TestFHIRRoleIfMatchIsAtomicCompareAndSwap proves the If-Match precondition is a compare-and-swap
// atomic with the write (the P1-C regression): two concurrent writers presenting the SAME valid
// If-Match (the current version) must not both succeed — exactly one wins (200) and the other gets a
// 412. Before the fix the version check was a separate read before the write, so both passed the
// read and the later write silently overwrote the earlier (a lost update). Run under -race.
func TestFHIRRoleIfMatchIsAtomicCompareAndSwap(t *testing.T) {
	base, cleanup := startFHIRDaemon(t, fhir.R5)
	defer cleanup()
	id := createPatient(t, base, fhir.R5) // version 1

	const writers = 8 // more than two, to stress the CAS under -race
	var wg sync.WaitGroup
	statuses := make([]int, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req, err := http.NewRequest(http.MethodPut, base+"/Patient/"+id,
				strings.NewReader(string(patientJSONWithIDGender(fhir.R5, id, "male"))))
			if err != nil {
				statuses[idx] = -1
				return
			}
			req.Header.Set("Content-Type", "application/fhir+json")
			req.Header.Set("If-Match", `W/"1"`) // every writer claims version 1
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				statuses[idx] = -1
				return
			}
			_ = resp.Body.Close()
			statuses[idx] = resp.StatusCode
		}(i)
	}
	wg.Wait()

	ok, conflict, other := 0, 0, 0
	for _, s := range statuses {
		switch s {
		case http.StatusOK:
			ok++
		case http.StatusPreconditionFailed:
			conflict++
		default:
			other++
		}
	}
	if ok != 1 || conflict != writers-1 || other != 0 {
		t.Fatalf("concurrent If-Match writers: ok=%d conflict=%d other=%d (statuses=%v); want exactly one 200 and %d 412",
			ok, conflict, other, statuses, writers-1)
	}

	// The resource advanced to exactly version 2 (one committed write), never further (no lost update
	// where a second writer also committed against version 1).
	_, _, hdr := httpDo(t, http.MethodGet, base+"/Patient/"+id, "", nil)
	if etag := hdr.Get("ETag"); etag != `W/"2"` {
		t.Fatalf("ETag after concurrent writes = %q, want W/\"2\" (exactly one commit)", etag)
	}
}

// pagedSearchRepo is a Repository whose Search returns a searchset reporting total=3 but carrying a
// single entry — a paged result. It exercises the conditional-resolution count decision (the P2-C
// regression): a conditional update/delete must use Bundle.total, not len(entry), so a multi-match
// search resolves as "many" (a 412), never a wrong single-resource write. Only Search and the write
// methods are needed; the rest report an error so a stray call is loud.
type pagedSearchRepo struct {
	Repository
	adapter releaseAdapter
}

func (p pagedSearchRepo) Search(context.Context, string, url.Values) (fhir.Resource, error) {
	g := r5.AdministrativeGender("female")
	one := &r5.Patient{DomainResource: r5.DomainResource{ID: strptr("page-1")}, Gender: &g}
	// total 3, one entry on the page.
	return p.adapter.newSearchSet(3, []fhir.Resource{one})
}

func (p pagedSearchRepo) Update(context.Context, string, string, fhir.Resource, string) (fhir.Resource, bool, error) {
	return nil, false, fmt.Errorf("pagedSearchRepo: update must not be reached on a multi-match resolution")
}

func (p pagedSearchRepo) Delete(context.Context, string, string, string) (bool, error) {
	return false, fmt.Errorf("pagedSearchRepo: delete must not be reached on a multi-match resolution")
}

// TestFHIRRoleConditionalWriteCountsBundleTotal proves conditional resolution uses Bundle.total, not
// the page's entry count (the P2-C regression): a searchset reporting total=3 with one entry must
// resolve as multiple matches, so a conditional update and a conditional delete each answer 412 and
// never reach the repository's Update/Delete with a single (wrong) id.
func TestFHIRRoleConditionalWriteCountsBundleTotal(t *testing.T) {
	repo := pagedSearchRepo{adapter: r5Adapter{}}
	role, err := NewFHIRRole(repo, WithFHIRPort(0), WithFHIRRelease(fhir.R5))
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

	// Conditional update against a total=3 searchset: 412 multiple matches, not a single write.
	status, body, _ := httpDoHeaders(t, http.MethodPut, base+"/Patient?gender=female",
		"application/fhir+json", patientJSONGender(fhir.R5, "male"), nil)
	if status != http.StatusPreconditionFailed {
		t.Fatalf("conditional update with total=3 status = %d, want 412; body=%s", status, body)
	}
	assertOperationOutcome(t, body, "error")

	// Conditional delete against the same searchset: 412 too.
	status, body, _ = httpDo(t, http.MethodDelete, base+"/Patient?gender=female", "", nil)
	if status != http.StatusPreconditionFailed {
		t.Fatalf("conditional delete with total=3 status = %d, want 412; body=%s", status, body)
	}
	assertOperationOutcome(t, body, "error")
}

// jsonEqual reports whether two JSON documents are structurally equal (key order and whitespace
// insensitive), for asserting a patch result without depending on member order.
func jsonEqual(t *testing.T, a, b []byte) bool {
	t.Helper()
	var va, vb any
	if err := json.Unmarshal(a, &va); err != nil {
		t.Fatalf("decode a: %v", err)
	}
	if err := json.Unmarshal(b, &vb); err != nil {
		t.Fatalf("decode b: %v", err)
	}
	ja, _ := json.Marshal(va)
	jb, _ := json.Marshal(vb)
	return string(ja) == string(jb)
}
