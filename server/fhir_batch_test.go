package server

// Tests for the FHIR role's batch processing at the base endpoint (POST [base] with a batch Bundle,
// FHIR R5 http.html#brules). Batch entries execute independently — a failing entry never rolls back
// or aborts its siblings, the contrast with the transaction path's atomicity — and every entry flows
// through exactly the same per-interaction pipeline as a standalone request: release validation on
// writes, conditional-header semantics, the workflow-type whitelist, and the version-store
// compare-and-swap. The mixed success/failure test is the regression guard for the validation-bypass
// class of defect: a validation-rejected resource inside a batch must yield a per-entry error outcome
// and must not persist, while a valid sibling in the same batch must persist.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r4"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// batchResponseEnvelope is the release-neutral JSON view of a batch-response Bundle the tests decode
// into: per-entry response.status/location/etag plus the OperationOutcome a failed entry carries in
// response.outcome.
type batchResponseEnvelope struct {
	ResourceType string `json:"resourceType"`
	Type         string `json:"type"`
	Entry        []struct {
		Resource json.RawMessage `json:"resource"`
		Response *struct {
			Status       string `json:"status"`
			Location     string `json:"location"`
			Etag         string `json:"etag"`
			LastModified string `json:"lastModified"`
			Outcome      *struct {
				ResourceType string `json:"resourceType"`
				Issue        []struct {
					Severity string `json:"severity"`
					Code     string `json:"code"`
				} `json:"issue"`
			} `json:"outcome"`
		} `json:"response"`
	} `json:"entry"`
}

// postBatch POSTs a batch Bundle to the base endpoint and decodes the batch-response Bundle,
// asserting the overall envelope: HTTP 200 whatever the per-entry results (batch failures live in
// the entries, not the HTTP status), resourceType Bundle, type batch-response, and one response
// entry per request entry.
func postBatch(t *testing.T, base string, bundle []byte, wantEntries int) batchResponseEnvelope {
	t.Helper()
	status, body, _ := httpDo(t, http.MethodPost, base, "application/fhir+json", bundle)
	if status != http.StatusOK {
		t.Fatalf("batch POST status = %d, want 200; body=%s", status, body)
	}
	var resp batchResponseEnvelope
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("batch response decode: %v; body=%s", err, body)
	}
	if resp.ResourceType != "Bundle" || resp.Type != "batch-response" {
		t.Fatalf("batch response = %s/%s, want Bundle/batch-response; body=%s", resp.ResourceType, resp.Type, body)
	}
	if len(resp.Entry) != wantEntries {
		t.Fatalf("batch response entries = %d, want %d; body=%s", len(resp.Entry), wantEntries, body)
	}
	for i, e := range resp.Entry {
		if e.Response == nil || e.Response.Status == "" {
			t.Fatalf("batch response entry[%d] has no response.status; body=%s", i, body)
		}
	}
	return resp
}

// assertBatchEntryStatus asserts one response entry's status line starts with the given HTTP code.
func assertBatchEntryStatus(t *testing.T, resp batchResponseEnvelope, i int, wantCode string) {
	t.Helper()
	got := resp.Entry[i].Response.Status
	if !strings.HasPrefix(got, wantCode) {
		t.Errorf("batch entry[%d] status = %q, want %s", i, got, wantCode)
	}
}

// assertBatchEntryOutcome asserts a failed entry carries an error-severity OperationOutcome in
// response.outcome, the per-entry error surface FHIR R5 bundle.html defines for a batch-response.
func assertBatchEntryOutcome(t *testing.T, resp batchResponseEnvelope, i int) {
	t.Helper()
	outcome := resp.Entry[i].Response.Outcome
	if outcome == nil {
		t.Errorf("batch entry[%d] has no response.outcome; a failed entry must carry one", i)
		return
	}
	if outcome.ResourceType != "OperationOutcome" {
		t.Errorf("batch entry[%d] outcome resourceType = %q, want OperationOutcome", i, outcome.ResourceType)
		return
	}
	if len(outcome.Issue) == 0 || outcome.Issue[0].Severity != "error" {
		t.Errorf("batch entry[%d] outcome carries no error issue: %+v", i, outcome.Issue)
	}
}

// batchBundleBytes builds a batch Bundle from the release's typed builder.
func batchBundleBytes(t *testing.T, release fhir.Release, r4Entries []r4.TransactionEntry, r5Entries []r5.TransactionEntry) []byte {
	t.Helper()
	switch release {
	case fhir.R4:
		b, err := r4.NewBatch(r4Entries...)
		if err != nil {
			t.Fatalf("r4 NewBatch: %v", err)
		}
		out, _ := json.Marshal(b)
		return out
	default:
		b, err := r5.NewBatch(r5Entries...)
		if err != nil {
			t.Fatalf("r5 NewBatch: %v", err)
		}
		out, _ := json.Marshal(b)
		return out
	}
}

// TestFHIRRoleBatchMixedSuccessAndFailure is the batch independence and validation-lockstep proof: a
// batch carrying a valid Patient create, an invalid Encounter create (missing its required status,
// which the standalone create rejects 422), a GET of an absent resource, and a second valid Patient
// create answers 200 with a batch-response whose entries line up one-to-one, in request order, with
// per-entry statuses. The invalid Encounter gets a per-entry 422 error outcome and is NOT persisted;
// both valid Patient siblings ARE persisted — a failing entry neither rolls back nor aborts the
// others, the exact contrast with the transaction test where the same invalid Encounter rejects the
// whole bundle.
func TestFHIRRoleBatchMixedSuccessAndFailure(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			g4 := r4.AdministrativeGender("female")
			g5 := r5.AdministrativeGender("female")
			bundle := batchBundleBytes(t, release,
				[]r4.TransactionEntry{
					{Resource: &r4.Patient{Gender: &g4}, Method: r4.HTTPVerbPOST, URL: "Patient"},
					{Resource: &r4.Encounter{}, Method: r4.HTTPVerbPOST, URL: "Encounter"},
					{Method: r4.HTTPVerbGET, URL: "Patient/does-not-exist"},
					{Resource: &r4.Patient{Gender: &g4}, Method: r4.HTTPVerbPOST, URL: "Patient"},
				},
				[]r5.TransactionEntry{
					{Resource: &r5.Patient{Gender: &g5}, Method: r5.HTTPVerbPOST, URL: "Patient"},
					{Resource: &r5.Encounter{}, Method: r5.HTTPVerbPOST, URL: "Encounter"},
					{Method: r5.HTTPVerbGET, URL: "Patient/does-not-exist"},
					{Resource: &r5.Patient{Gender: &g5}, Method: r5.HTTPVerbPOST, URL: "Patient"},
				})

			resp := postBatch(t, base, bundle, 4)

			assertBatchEntryStatus(t, resp, 0, "201")
			if loc := resp.Entry[0].Response.Location; !strings.Contains(loc, "Patient/") || !strings.Contains(loc, "/_history/") {
				t.Errorf("batch entry[0] location = %q, want a versioned Patient location", loc)
			}
			if etag := resp.Entry[0].Response.Etag; etag == "" {
				t.Error("batch entry[0] has no response.etag; a created entry must report its version ETag")
			}

			assertBatchEntryStatus(t, resp, 1, "422")
			assertBatchEntryOutcome(t, resp, 1)

			assertBatchEntryStatus(t, resp, 2, "404")
			assertBatchEntryOutcome(t, resp, 2)

			assertBatchEntryStatus(t, resp, 3, "201")

			// Independence: the invalid Encounter did not persist, and its failure did not roll back or
			// abort either valid Patient sibling.
			assertWorkflowCount(t, base, "Patient", 2)
			assertWorkflowCount(t, base, "Encounter", 0)
		})
	}
}

// TestFHIRRoleBatchEntryConditionalHeaders proves a batch entry's request.ifNoneExist and
// request.ifMatch ride into the same conditional-header semantics the standalone paths enforce: a
// POST entry carrying ifNoneExist fails closed with a per-entry 400 (the standalone If-None-Exist
// stance — nothing persisted, never a silent duplicate), a PUT entry with a stale ifMatch is a
// per-entry 412 through the repository's compare-and-swap without modifying the resource, and a PUT
// entry with the current version's ifMatch commits (200) and bumps the version — all in one batch,
// each entry independent.
func TestFHIRRoleBatchEntryConditionalHeaders(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			// Seed one Patient at version 1.
			status, body, _ := httpDo(t, http.MethodPost, base+"/Patient", "application/fhir+json",
				patientJSON(release, "female"))
			if status != http.StatusCreated {
				t.Fatalf("seed create status = %d; body=%s", status, body)
			}
			var created struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(body, &created); err != nil || created.ID == "" {
				t.Fatalf("seed create id decode: %v; body=%s", err, body)
			}
			id := created.ID

			g4 := r4.AdministrativeGender("male")
			g5 := r5.AdministrativeGender("male")
			bundle := batchBundleBytes(t, release,
				[]r4.TransactionEntry{
					{Resource: &r4.Patient{Gender: &g4}, Method: r4.HTTPVerbPOST, URL: "Patient", IfNoneExist: "gender=male"},
					{Resource: &r4.Patient{Gender: &g4}, Method: r4.HTTPVerbPUT, URL: "Patient/" + id, IfMatch: `W/"99"`},
					{Resource: &r4.Patient{Gender: &g4}, Method: r4.HTTPVerbPUT, URL: "Patient/" + id, IfMatch: `W/"1"`},
				},
				[]r5.TransactionEntry{
					{Resource: &r5.Patient{Gender: &g5}, Method: r5.HTTPVerbPOST, URL: "Patient", IfNoneExist: "gender=male"},
					{Resource: &r5.Patient{Gender: &g5}, Method: r5.HTTPVerbPUT, URL: "Patient/" + id, IfMatch: `W/"99"`},
					{Resource: &r5.Patient{Gender: &g5}, Method: r5.HTTPVerbPUT, URL: "Patient/" + id, IfMatch: `W/"1"`},
				})

			resp := postBatch(t, base, bundle, 3)

			// The conditional create fails closed per-entry, exactly like the standalone header.
			assertBatchEntryStatus(t, resp, 0, "400")
			assertBatchEntryOutcome(t, resp, 0)

			// The stale If-Match is a per-entry 412 through the same compare-and-swap.
			assertBatchEntryStatus(t, resp, 1, "412")
			assertBatchEntryOutcome(t, resp, 1)

			// The current-version If-Match commits and reports the bumped version.
			assertBatchEntryStatus(t, resp, 2, "200")
			if etag := resp.Entry[2].Response.Etag; etag != `W/"2"` {
				t.Errorf("batch entry[2] etag = %q, want W/\"2\" (the stale sibling must not have consumed a version)", etag)
			}

			// The failed conditional create persisted nothing: still exactly one Patient.
			assertWorkflowCount(t, base, "Patient", 1)
		})
	}
}

// TestFHIRRoleBatchUnsupportedEntryShapes proves malformed and unsupported batch entries fail
// per-entry through the same gates the standalone paths apply, without aborting valid siblings: an
// entry with no request at all is a per-entry 400 (bdl-3a), an entry addressing the base endpoint
// (an empty request.url — a nested batch/transaction) is a per-entry 400, an absolute request.url is
// a per-entry 400 (entry URLs are relative to base), an unsupported method on a served route is a
// per-entry 405, a PATCH entry is a per-entry 415 (the standalone patch accepts JSON Patch bodies
// only, which a Bundle entry resource is not), an out-of-scope resource type is the whitelist's
// per-entry 404, and the valid Patient sibling still commits.
func TestFHIRRoleBatchUnsupportedEntryShapes(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			bundle := rawBatchBundleBytes(t, release)
			resp := postBatch(t, base, bundle, 7)

			for _, i := range []int{0, 1, 2} {
				assertBatchEntryStatus(t, resp, i, "400")
				assertBatchEntryOutcome(t, resp, i)
			}
			assertBatchEntryStatus(t, resp, 3, "405")
			assertBatchEntryOutcome(t, resp, 3)
			assertBatchEntryStatus(t, resp, 4, "415")
			assertBatchEntryOutcome(t, resp, 4)
			assertBatchEntryStatus(t, resp, 5, "404")
			assertBatchEntryOutcome(t, resp, 5)
			assertBatchEntryStatus(t, resp, 6, "201")

			assertWorkflowCount(t, base, "Patient", 1)
		})
	}
}

// rawBatchBundleBytes hand-builds the malformed-entry batch Bundle the unsupported-shapes test
// POSTs; the typed NewBatch builder correctly refuses these shapes, so the test constructs the
// Bundle structs directly, standing in for a non-conforming client on the wire.
func rawBatchBundleBytes(t *testing.T, release fhir.Release) []byte {
	t.Helper()
	switch release {
	case fhir.R4:
		g := r4.AdministrativeGender("female")
		bt := r4.BundleTypeBatch
		post, head, patch := r4.HTTPVerbPOST, r4.HTTPVerb("HEAD"), r4.HTTPVerbPATCH
		emptyURL, patientURL, patientInstanceURL := "", "Patient", "Patient/anything"
		absoluteURL := "http://elsewhere.example/Patient"
		medicationURL := "Medication"
		var patientRes fhir.Resource = &r4.Patient{Gender: &g}
		var medicationRes fhir.Resource = &r4.Medication{}
		b := &r4.Bundle{Type: &bt, Entry: []r4.BundleEntry{
			{Resource: &patientRes},
			{Request: &r4.BundleEntryRequest{Method: &post, URL: &emptyURL}, Resource: &patientRes},
			{Request: &r4.BundleEntryRequest{Method: &post, URL: &absoluteURL}, Resource: &patientRes},
			{Request: &r4.BundleEntryRequest{Method: &head, URL: &patientURL}},
			{Request: &r4.BundleEntryRequest{Method: &patch, URL: &patientInstanceURL}, Resource: &patientRes},
			{Request: &r4.BundleEntryRequest{Method: &post, URL: &medicationURL}, Resource: &medicationRes},
			{Request: &r4.BundleEntryRequest{Method: &post, URL: &patientURL}, Resource: &patientRes},
		}}
		out, _ := json.Marshal(b)
		return out
	default:
		g := r5.AdministrativeGender("female")
		bt := r5.BundleTypeBatch
		post, head, patch := r5.HTTPVerbPOST, r5.HTTPVerb("HEAD"), r5.HTTPVerbPATCH
		emptyURL, patientURL, patientInstanceURL := "", "Patient", "Patient/anything"
		absoluteURL := "http://elsewhere.example/Patient"
		medicationURL := "Medication"
		var patientRes fhir.Resource = &r5.Patient{Gender: &g}
		var medicationRes fhir.Resource = &r5.Medication{}
		b := &r5.Bundle{Type: &bt, Entry: []r5.BundleEntry{
			{Resource: &patientRes},
			{Request: &r5.BundleEntryRequest{Method: &post, URL: &emptyURL}, Resource: &patientRes},
			{Request: &r5.BundleEntryRequest{Method: &post, URL: &absoluteURL}, Resource: &patientRes},
			{Request: &r5.BundleEntryRequest{Method: &head, URL: &patientURL}},
			{Request: &r5.BundleEntryRequest{Method: &patch, URL: &patientInstanceURL}, Resource: &patientRes},
			{Request: &r5.BundleEntryRequest{Method: &post, URL: &medicationURL}, Resource: &medicationRes},
			{Request: &r5.BundleEntryRequest{Method: &post, URL: &patientURL}, Resource: &patientRes},
		}}
		out, _ := json.Marshal(b)
		return out
	}
}

// TestFHIRRoleBatchDeleteAndReadEntries proves the non-create verbs flow through their standalone
// pipelines inside a batch: a DELETE entry of an existing resource answers the standalone delete's
// 200, a GET entry of the now-deleted resource answers the standalone read's 410 Gone (the entries
// run sequentially in request order, so the read observes the delete), and a DELETE of an absent
// resource answers the idempotent 204.
func TestFHIRRoleBatchDeleteAndReadEntries(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			status, body, _ := httpDo(t, http.MethodPost, base+"/Patient", "application/fhir+json",
				patientJSON(release, "female"))
			if status != http.StatusCreated {
				t.Fatalf("seed create status = %d; body=%s", status, body)
			}
			var created struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(body, &created); err != nil || created.ID == "" {
				t.Fatalf("seed create id decode: %v; body=%s", err, body)
			}
			id := created.ID

			bundle := batchBundleBytes(t, release,
				[]r4.TransactionEntry{
					{Method: r4.HTTPVerbDELETE, URL: "Patient/" + id},
					{Method: r4.HTTPVerbGET, URL: "Patient/" + id},
					{Method: r4.HTTPVerbDELETE, URL: "Patient/never-existed"},
				},
				[]r5.TransactionEntry{
					{Method: r5.HTTPVerbDELETE, URL: "Patient/" + id},
					{Method: r5.HTTPVerbGET, URL: "Patient/" + id},
					{Method: r5.HTTPVerbDELETE, URL: "Patient/never-existed"},
				})

			resp := postBatch(t, base, bundle, 3)
			assertBatchEntryStatus(t, resp, 0, "200")
			assertBatchEntryStatus(t, resp, 1, "410")
			assertBatchEntryOutcome(t, resp, 1)
			assertBatchEntryStatus(t, resp, 2, "204")
		})
	}
}

// TestFHIRRoleBatchReadReturnsResource is the merge-blocker regression: a successful batch GET must
// carry its resource in the batch-response entry. A read entry returns the Patient in entry.resource,
// and a search entry returns its searchset Bundle — a 200 with an empty entry would be non-conformant
// and unusable.
func TestFHIRRoleBatchReadReturnsResource(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			status, body, _ := httpDo(t, http.MethodPost, base+"/Patient", "application/fhir+json",
				patientJSON(release, "female"))
			if status != http.StatusCreated {
				t.Fatalf("seed create status = %d; body=%s", status, body)
			}
			var created struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(body, &created); err != nil || created.ID == "" {
				t.Fatalf("seed create id decode: %v; body=%s", err, body)
			}
			id := created.ID

			bundle := batchBundleBytes(t, release,
				[]r4.TransactionEntry{
					{Method: r4.HTTPVerbGET, URL: "Patient/" + id},
					{Method: r4.HTTPVerbGET, URL: "Patient"},
				},
				[]r5.TransactionEntry{
					{Method: r5.HTTPVerbGET, URL: "Patient/" + id},
					{Method: r5.HTTPVerbGET, URL: "Patient"},
				})

			resp := postBatch(t, base, bundle, 2)
			assertBatchEntryStatus(t, resp, 0, "200")
			assertBatchEntryStatus(t, resp, 1, "200")

			// The read entry carries the Patient itself.
			var readEntry struct {
				ResourceType string `json:"resourceType"`
				ID           string `json:"id"`
			}
			if err := json.Unmarshal(resp.Entry[0].Resource, &readEntry); err != nil {
				t.Fatalf("read entry.resource decode: %v; raw=%s", err, resp.Entry[0].Resource)
			}
			if readEntry.ResourceType != "Patient" || readEntry.ID != id {
				t.Errorf("read entry.resource = %s/%s, want Patient/%s", readEntry.ResourceType, readEntry.ID, id)
			}

			// The search entry carries a searchset Bundle.
			var searchEntry struct {
				ResourceType string `json:"resourceType"`
				Type         string `json:"type"`
			}
			if err := json.Unmarshal(resp.Entry[1].Resource, &searchEntry); err != nil {
				t.Fatalf("search entry.resource decode: %v; raw=%s", err, resp.Entry[1].Resource)
			}
			if searchEntry.ResourceType != "Bundle" || searchEntry.Type != "searchset" {
				t.Errorf("search entry.resource = %s/%s, want Bundle/searchset", searchEntry.ResourceType, searchEntry.Type)
			}
		})
	}
}

// TestFHIRRoleBatchPatchEntry proves a batch PATCH entry applies: a JSON Patch conveyed as a Binary
// (contentType application/json-patch+json, data base64) flows through the same standalone PATCH
// pipeline. Without the Binary payload adaptation a PATCH entry could never satisfy the standalone
// PATCH media gate.
func TestFHIRRoleBatchPatchEntry(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			status, body, _ := httpDo(t, http.MethodPost, base+"/Patient", "application/fhir+json",
				patientJSON(release, "female"))
			if status != http.StatusCreated {
				t.Fatalf("seed create status = %d; body=%s", status, body)
			}
			var created struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(body, &created); err != nil || created.ID == "" {
				t.Fatalf("seed create id decode: %v; body=%s", err, body)
			}
			id := created.ID

			patchDoc := []byte(`[{"op":"replace","path":"/gender","value":"male"}]`)
			bundle := batchPatchBundleBytes(t, release, id, patchDoc)
			resp := postBatch(t, base, bundle, 1)
			assertBatchEntryStatus(t, resp, 0, "200")

			// The patch took effect: a follow-up read shows gender male.
			status, body, _ = httpDo(t, http.MethodGet, base+"/Patient/"+id, "", nil)
			if status != http.StatusOK {
				t.Fatalf("read after batch patch status = %d; body=%s", status, body)
			}
			var patched struct {
				Gender string `json:"gender"`
			}
			if err := json.Unmarshal(body, &patched); err != nil || patched.Gender != "male" {
				t.Errorf("gender after batch patch = %q (err %v), want male", patched.Gender, err)
			}
		})
	}
}

// batchPatchBundleBytes builds a single-entry batch whose PATCH entry carries a JSON Patch as a
// Binary resource (contentType application/json-patch+json, data base64), the FHIR-conformant way to
// convey a patch inside a bundle entry.
func batchPatchBundleBytes(t *testing.T, release fhir.Release, id string, patchDoc []byte) []byte {
	t.Helper()
	encoded := base64.StdEncoding.EncodeToString(patchDoc)
	ct := mediaTypeJSONPatch
	switch release {
	case fhir.R4:
		bt := r4.BundleTypeBatch
		method := r4.HTTPVerbPATCH
		url := "Patient/" + id
		var bin fhir.Resource = &r4.Binary{ContentType: &ct, Data: &encoded}
		b := &r4.Bundle{Type: &bt, Entry: []r4.BundleEntry{
			{Request: &r4.BundleEntryRequest{Method: &method, URL: &url}, Resource: &bin},
		}}
		out, _ := json.Marshal(b)
		return out
	default:
		bt := r5.BundleTypeBatch
		method := r5.HTTPVerbPATCH
		url := "Patient/" + id
		var bin fhir.Resource = &r5.Binary{ContentType: &ct, Data: &encoded}
		b := &r5.Bundle{Type: &bt, Entry: []r5.BundleEntry{
			{Request: &r5.BundleEntryRequest{Method: &method, URL: &url}, Resource: &bin},
		}}
		out, _ := json.Marshal(b)
		return out
	}
}

// TestFHIRRoleBatchEntryPathCanonicalisation proves a batch entry cannot smuggle an un-cleaned path
// past the routing edge: a "../" or "//" in an entry request.url is a per-entry 400, and the valid
// sibling still commits — dispatching internally must not bypass the path canonicalisation the
// standalone edge would apply.
func TestFHIRRoleBatchEntryPathCanonicalisation(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			bundle := rawBatchPathBundleBytes(t, release)
			resp := postBatch(t, base, bundle, 3)
			assertBatchEntryStatus(t, resp, 0, "400")
			assertBatchEntryOutcome(t, resp, 0)
			assertBatchEntryStatus(t, resp, 1, "400")
			assertBatchEntryOutcome(t, resp, 1)
			assertBatchEntryStatus(t, resp, 2, "201")
			assertWorkflowCount(t, base, "Patient", 1)
		})
	}
}

// rawBatchPathBundleBytes hand-builds a batch whose first two entries carry non-canonical paths
// ("Patient/../Secret" and "Patient//1") and whose third is a valid create.
func rawBatchPathBundleBytes(t *testing.T, release fhir.Release) []byte {
	t.Helper()
	switch release {
	case fhir.R4:
		g := r4.AdministrativeGender("female")
		bt := r4.BundleTypeBatch
		get, post := r4.HTTPVerbGET, r4.HTTPVerbPOST
		dotdot, doubleslash, patientURL := "Patient/../Secret", "Patient//1", "Patient"
		var patientRes fhir.Resource = &r4.Patient{Gender: &g}
		b := &r4.Bundle{Type: &bt, Entry: []r4.BundleEntry{
			{Request: &r4.BundleEntryRequest{Method: &get, URL: &dotdot}},
			{Request: &r4.BundleEntryRequest{Method: &get, URL: &doubleslash}},
			{Request: &r4.BundleEntryRequest{Method: &post, URL: &patientURL}, Resource: &patientRes},
		}}
		out, _ := json.Marshal(b)
		return out
	default:
		g := r5.AdministrativeGender("female")
		bt := r5.BundleTypeBatch
		get, post := r5.HTTPVerbGET, r5.HTTPVerbPOST
		dotdot, doubleslash, patientURL := "Patient/../Secret", "Patient//1", "Patient"
		var patientRes fhir.Resource = &r5.Patient{Gender: &g}
		b := &r5.Bundle{Type: &bt, Entry: []r5.BundleEntry{
			{Request: &r5.BundleEntryRequest{Method: &get, URL: &dotdot}},
			{Request: &r5.BundleEntryRequest{Method: &get, URL: &doubleslash}},
			{Request: &r5.BundleEntryRequest{Method: &post, URL: &patientURL}, Resource: &patientRes},
		}}
		out, _ := json.Marshal(b)
		return out
	}
}

// TestFHIRRoleBatchEntryCap proves the base endpoint caps the entry count on both submission paths: an
// over-cap batch and an over-cap transaction are each rejected with an OperationOutcome before any
// entry runs, so a hostile 8 MB bundle cannot force unbounded sequential writes.
func TestFHIRRoleBatchEntryCap(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			for _, bundleType := range []string{"batch", "transaction"} {
				bundle := overCapBundleBytes(t, release, bundleType, maxBundleEntries+1)
				status, body, _ := httpDo(t, http.MethodPost, base, "application/fhir+json", bundle)
				if status != http.StatusRequestEntityTooLarge {
					t.Fatalf("%s over-cap status = %d, want 413; body=%s", bundleType, status, body)
				}
				assertOperationOutcome(t, body, "error")
			}
			// Nothing was created: the cap rejects before any entry runs.
			assertWorkflowCount(t, base, "Patient", 0)
		})
	}
}

// overCapBundleBytes builds a batch or transaction Bundle carrying n POST-Patient entries, for the
// entry-cap test.
func overCapBundleBytes(t *testing.T, release fhir.Release, bundleType string, n int) []byte {
	t.Helper()
	switch release {
	case fhir.R4:
		g := r4.AdministrativeGender("female")
		entries := make([]r4.TransactionEntry, 0, n)
		for i := 0; i < n; i++ {
			entries = append(entries, r4.TransactionEntry{Resource: &r4.Patient{Gender: &g}, Method: r4.HTTPVerbPOST, URL: "Patient"})
		}
		var b *r4.Bundle
		var err error
		if bundleType == "batch" {
			b, err = r4.NewBatch(entries...)
		} else {
			b, err = r4.NewTransaction(entries...)
		}
		if err != nil {
			t.Fatalf("r4 build %s: %v", bundleType, err)
		}
		out, _ := json.Marshal(b)
		return out
	default:
		g := r5.AdministrativeGender("female")
		entries := make([]r5.TransactionEntry, 0, n)
		for i := 0; i < n; i++ {
			entries = append(entries, r5.TransactionEntry{Resource: &r5.Patient{Gender: &g}, Method: r5.HTTPVerbPOST, URL: "Patient"})
		}
		var b *r5.Bundle
		var err error
		if bundleType == "batch" {
			b, err = r5.NewBatch(entries...)
		} else {
			b, err = r5.NewTransaction(entries...)
		}
		if err != nil {
			t.Fatalf("r5 build %s: %v", bundleType, err)
		}
		out, _ := json.Marshal(b)
		return out
	}
}

// TestFHIRRoleBatchContextCancellation proves the batch loop honours request-context cancellation: a
// batch dispatched with an already-cancelled context stops before mutating the repository, so a
// disconnected client cannot drive unbounded work. It drives the handler directly so the cancellation
// is deterministic.
func TestFHIRRoleBatchContextCancellation(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			repo, err := NewMemoryRepository(release)
			if err != nil {
				t.Fatalf("NewMemoryRepository: %v", err)
			}
			adapter, _ := adapterForRelease(release)
			h := &fhirHandler{
				repo:            repo,
				adapter:         adapter,
				basePath:        "/fhir",
				maxRequestBytes: defaultFHIRMaxRequestBytes,
				logger:          zap.NewNop(),
			}
			bundle := batchBundleBytes(t, release,
				[]r4.TransactionEntry{{Resource: &r4.Patient{}, Method: r4.HTTPVerbPOST, URL: "Patient"}},
				[]r5.TransactionEntry{{Resource: &r5.Patient{}, Method: r5.HTTPVerbPOST, URL: "Patient"}})
			decoded, err := adapter.unmarshalResource(bundle)
			if err != nil {
				t.Fatalf("decode bundle: %v", err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			req := httptest.NewRequest(http.MethodPost, "/fhir", nil).WithContext(ctx)
			rec := httptest.NewRecorder()
			h.handleBatch(rec, req, decoded)

			if rec.Code == http.StatusOK {
				t.Errorf("a cancelled batch should not answer 200 OK, got %d", rec.Code)
			}
			// The cancelled batch mutated nothing.
			searchBundle, serr := repo.Search(context.Background(), "Patient", nil)
			if serr != nil {
				t.Fatalf("Search: %v", serr)
			}
			if total := searchsetTotal(t, searchBundle); total != 0 {
				t.Errorf("a cancelled batch created %d Patients, want 0", total)
			}
		})
	}
}

// TestFHIRRoleBatchWriteCarriesLastModified proves a batch write entry reports response.lastModified,
// mirroring the Last-Modified header the standalone write path emits.
func TestFHIRRoleBatchWriteCarriesLastModified(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			g4 := r4.AdministrativeGender("female")
			g5 := r5.AdministrativeGender("female")
			bundle := batchBundleBytes(t, release,
				[]r4.TransactionEntry{{Resource: &r4.Patient{Gender: &g4}, Method: r4.HTTPVerbPOST, URL: "Patient"}},
				[]r5.TransactionEntry{{Resource: &r5.Patient{Gender: &g5}, Method: r5.HTTPVerbPOST, URL: "Patient"}})
			resp := postBatch(t, base, bundle, 1)
			assertBatchEntryStatus(t, resp, 0, "201")
			if resp.Entry[0].Response.LastModified == "" {
				t.Error("a created batch entry should carry response.lastModified")
			}
		})
	}
}
