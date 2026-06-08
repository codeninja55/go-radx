package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r4"
	"github.com/codeninja55/go-radx/fhir/r5"
	"github.com/codeninja55/go-radx/fhir/rest"
)

// startFHIRDaemon mounts a FHIR role over an in-memory repository of the given release on a loopback
// OS-assigned port and returns the running daemon's base URL and a cleanup. It is the shared harness
// for the role HTTP tests, so each test exercises the real handler over real HTTP rather than a
// direct method call.
func startFHIRDaemon(t *testing.T, release fhir.Release) (string, func()) {
	t.Helper()
	repo, err := NewMemoryRepository(release)
	if err != nil {
		t.Fatalf("NewMemoryRepository: %v", err)
	}
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

// httpDo issues a request and returns the status, body, and Location header, the small client the
// role HTTP tests share.
func httpDo(t *testing.T, method, url, contentType string, body []byte) (int, []byte, http.Header) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/fhir+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do %s %s: %v", method, url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, out, resp.Header
}

func patientJSON(release fhir.Release, gender string) []byte {
	switch release {
	case fhir.R4:
		g := r4.AdministrativeGender(gender)
		b, _ := json.Marshal(&r4.Patient{Gender: &g})
		return b
	default:
		g := r5.AdministrativeGender(gender)
		b, _ := json.Marshal(&r5.Patient{Gender: &g})
		return b
	}
}

func fhirReleases() []fhir.Release { return []fhir.Release{fhir.R4, fhir.R5} }

// patientJSONWithID builds a Patient carrying a client-supplied id, the create payload that proves the
// server assigns its own id and never honours the client's on create.
func patientJSONWithID(release fhir.Release, id string) []byte {
	switch release {
	case fhir.R4:
		g := r4.AdministrativeGender("female")
		b, _ := json.Marshal(&r4.Patient{DomainResource: r4.DomainResource{ID: &id}, Gender: &g})
		return b
	default:
		g := r5.AdministrativeGender("female")
		b, _ := json.Marshal(&r5.Patient{DomainResource: r5.DomainResource{ID: &id}, Gender: &g})
		return b
	}
}

// createPatientWithClientID POSTs a Patient carrying the given client id and returns the
// server-assigned id from the response body, for the create-never-overwrites guard.
func createPatientWithClientID(t *testing.T, base string, release fhir.Release, clientID string) string {
	t.Helper()
	status, body, _ := httpDo(t, http.MethodPost, base+"/Patient", "application/fhir+json",
		patientJSONWithID(release, clientID))
	if status != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", status, body)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("create body decode: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("create body has no id: %s", body)
	}
	return created.ID
}

// collidingCreateTransactionBundle builds a two-POST transaction whose entries both carry the same
// client id "1", so the test can prove a transaction create assigns server ids and never overwrites.
func collidingCreateTransactionBundle(t *testing.T, release fhir.Release) []byte {
	t.Helper()
	id := "1"
	switch release {
	case fhir.R4:
		g := r4.AdministrativeGender("female")
		b, err := r4.NewTransaction(
			r4.TransactionEntry{Resource: &r4.Patient{DomainResource: r4.DomainResource{ID: &id}, Gender: &g}, Method: r4.HTTPVerbPOST, URL: "Patient"},
			r4.TransactionEntry{Resource: &r4.Patient{DomainResource: r4.DomainResource{ID: &id}, Gender: &g}, Method: r4.HTTPVerbPOST, URL: "Patient"},
		)
		if err != nil {
			t.Fatalf("r4 NewTransaction: %v", err)
		}
		out, _ := json.Marshal(b)
		return out
	default:
		g := r5.AdministrativeGender("female")
		b, err := r5.NewTransaction(
			r5.TransactionEntry{Resource: &r5.Patient{DomainResource: r5.DomainResource{ID: &id}, Gender: &g}, Method: r5.HTTPVerbPOST, URL: "Patient"},
			r5.TransactionEntry{Resource: &r5.Patient{DomainResource: r5.DomainResource{ID: &id}, Gender: &g}, Method: r5.HTTPVerbPOST, URL: "Patient"},
		)
		if err != nil {
			t.Fatalf("r5 NewTransaction: %v", err)
		}
		out, _ := json.Marshal(b)
		return out
	}
}

// instanceURLCreateTransactionBundle builds a transaction whose single POST entry targets an instance
// URL ("Patient/123") rather than the type endpoint ("Patient"). A FHIR transaction create must
// target the type, so the role must reject this bundle rather than silently creating a Patient and
// discarding the id. The entry's resource carries no id, so the only thing wrong is the request.url.
func instanceURLCreateTransactionBundle(t *testing.T, release fhir.Release) []byte {
	t.Helper()
	switch release {
	case fhir.R4:
		g := r4.AdministrativeGender("female")
		b, err := r4.NewTransaction(
			r4.TransactionEntry{Resource: &r4.Patient{Gender: &g}, Method: r4.HTTPVerbPOST, URL: "Patient/123"},
		)
		if err != nil {
			t.Fatalf("r4 NewTransaction: %v", err)
		}
		out, _ := json.Marshal(b)
		return out
	default:
		g := r5.AdministrativeGender("female")
		b, err := r5.NewTransaction(
			r5.TransactionEntry{Resource: &r5.Patient{Gender: &g}, Method: r5.HTTPVerbPOST, URL: "Patient/123"},
		)
		if err != nil {
			t.Fatalf("r5 NewTransaction: %v", err)
		}
		out, _ := json.Marshal(b)
		return out
	}
}

// TestFHIRRoleTransactionRejectsInstanceURLCreate proves a transaction POST entry whose request.url
// names an instance ("Patient/123") is rejected with a 400 OperationOutcome and commits nothing: a
// create must target the type endpoint, so the role must not silently create a Patient under a
// server id while discarding the malformed instance URL. The store stays empty (atomic), and a POST
// entry url of the bare type still creates, so the rejection is the instance URL, not the verb.
func TestFHIRRoleTransactionRejectsInstanceURLCreate(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			bundle := instanceURLCreateTransactionBundle(t, release)
			status, body, _ := httpDo(t, http.MethodPost, base, "application/fhir+json", bundle)
			if status != http.StatusBadRequest {
				t.Fatalf("instance-url create transaction status = %d, want 400; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")
			// Atomic: nothing was created.
			assertWorkflowCount(t, base, "Patient", 0)

			// A POST entry targeting the bare type still creates, so the guard rejects the instance URL,
			// not the POST verb.
			ok := transactionBundleResource(t, release)
			okBody, _ := json.Marshal(ok)
			status, body, _ = httpDo(t, http.MethodPost, base, "application/fhir+json", okBody)
			if status != http.StatusOK {
				t.Fatalf("type-endpoint create transaction status = %d, want 200; body=%s", status, body)
			}
			assertWorkflowCount(t, base, "Patient", 1)
		})
	}
}

// startFHIRDaemonAt mounts a FHIR role on the given base path and returns the daemon's bound HTTP
// address (host:port, with no base path appended) and a cleanup, so a test can build URLs against an
// arbitrary mount point including the root ("/"). It is the base-path-aware twin of startFHIRDaemon.
func startFHIRDaemonAt(t *testing.T, release fhir.Release, basePath string) (string, func()) {
	t.Helper()
	repo, err := NewMemoryRepository(release)
	if err != nil {
		t.Fatalf("NewMemoryRepository: %v", err)
	}
	role, err := NewFHIRRole(repo, WithFHIRPort(0), WithFHIRRelease(release), WithFHIRBasePath(basePath))
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
	roleName := role.name()
	waitForAddrs(t, d, roleName)
	host := "http://" + d.Addrs()[roleName].String()
	cleanup := func() {
		cancelRun()
		select {
		case <-runErr:
		case <-time.After(5 * time.Second):
			t.Error("daemon did not stop within 5s")
		}
	}
	return host, cleanup
}

// TestFHIRRoleRootMountLocationIsSingleSlash proves a create against a root-mounted ("/") FHIR role
// returns a Location with a single leading slash ("/Patient/{id}"), not "//Patient/{id}". The
// double-slash form parses as a network-path reference (host "Patient"), so it is not a valid
// relative Location; the single-slash form parses as an absolute-path reference with an empty host.
// A "/fhir"-mounted role keeps its "/fhir/Patient/{id}" Location, so the join is correct at both
// mount points.
func TestFHIRRoleRootMountLocationIsSingleSlash(t *testing.T) {
	cases := []struct {
		name     string
		basePath string
		wantBase string // the Location prefix before "/Patient/"
	}{
		{"root mount", "/", ""},
		{"fhir mount", "/fhir", "/fhir"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			host, cleanup := startFHIRDaemonAt(t, fhir.R5, tc.basePath)
			defer cleanup()

			createURL := host + tc.wantBase + "/Patient"
			status, body, header := httpDo(t, http.MethodPost, createURL, "application/fhir+json",
				patientJSON(fhir.R5, "female"))
			if status != http.StatusCreated {
				t.Fatalf("create status = %d, want 201; body=%s", status, body)
			}
			var created struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(body, &created); err != nil {
				t.Fatalf("create body decode: %v", err)
			}
			loc := header.Get("Location")
			want := tc.wantBase + "/Patient/" + created.ID
			if loc != want {
				t.Fatalf("Location = %q, want %q", loc, want)
			}
			// The Location parses as an absolute-path reference with an empty host: a leading "//" would
			// instead parse as a network-path reference whose host is the resource type.
			u, err := url.Parse(loc)
			if err != nil {
				t.Fatalf("Location %q does not parse: %v", loc, err)
			}
			if u.Host != "" {
				t.Fatalf("Location %q parsed with host %q; a valid relative Location has an empty host", loc, u.Host)
			}
		})
	}
}

// TestDaemonExposesTwoFHIRRolesDistinctly proves the documented dual-release setup works: mounting
// two FHIRRoles (R4 and R5) on different base paths exposes BOTH bound addresses through
// Daemon.Addrs(), under two distinct keys, neither overwritten. Before the role names incorporated
// the base path, both roles reported "fhir" and Addrs() (a map keyed by role name) dropped one.
func TestDaemonExposesTwoFHIRRolesDistinctly(t *testing.T) {
	r4Repo, err := NewMemoryRepository(fhir.R4)
	if err != nil {
		t.Fatalf("NewMemoryRepository R4: %v", err)
	}
	r5Repo, err := NewMemoryRepository(fhir.R5)
	if err != nil {
		t.Fatalf("NewMemoryRepository R5: %v", err)
	}
	r4Role, err := NewFHIRRole(r4Repo, WithFHIRPort(0), WithFHIRRelease(fhir.R4), WithFHIRBasePath("/fhir/r4"))
	if err != nil {
		t.Fatalf("NewFHIRRole R4: %v", err)
	}
	r5Role, err := NewFHIRRole(r5Repo, WithFHIRPort(0), WithFHIRRelease(fhir.R5), WithFHIRBasePath("/fhir/r5"))
	if err != nil {
		t.Fatalf("NewFHIRRole R5: %v", err)
	}
	d, err := New(WithFHIR(r4Role), WithFHIR(r5Role))
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

	r4Name, r5Name := r4Role.name(), r5Role.name()
	if r4Name == r5Name {
		t.Fatalf("two FHIR roles share the name %q; Addrs() would overwrite one", r4Name)
	}
	waitForAddrs(t, d, r4Name)
	waitForAddrs(t, d, r5Name)

	addrs := d.Addrs()
	a4, a5 := addrs[r4Name], addrs[r5Name]
	if a4 == nil {
		t.Fatalf("Addrs() missing the R4 role under key %q: %v", r4Name, addrs)
	}
	if a5 == nil {
		t.Fatalf("Addrs() missing the R5 role under key %q: %v", r5Name, addrs)
	}
	if a4.String() == a5.String() {
		t.Fatalf("both FHIR roles bound to the same address %q; the two roles collapsed", a4)
	}
}

func TestFHIRRoleCreateReadSearch(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			// Create.
			status, body, header := httpDo(t, http.MethodPost, base+"/Patient", "application/fhir+json",
				patientJSON(release, "female"))
			if status != http.StatusCreated {
				t.Fatalf("create status = %d, want 201; body=%s", status, body)
			}
			if header.Get("Location") == "" {
				t.Error("create: expected a Location header")
			}
			var created struct {
				ResourceType string `json:"resourceType"`
				ID           string `json:"id"`
			}
			if err := json.Unmarshal(body, &created); err != nil {
				t.Fatalf("create body decode: %v", err)
			}
			if created.ResourceType != "Patient" || created.ID == "" {
				t.Fatalf("create body = %s, want a Patient with an id", body)
			}

			// Read it back.
			status, body, _ = httpDo(t, http.MethodGet, base+"/Patient/"+created.ID, "", nil)
			if status != http.StatusOK {
				t.Fatalf("read status = %d, want 200; body=%s", status, body)
			}

			// Search by _id finds it.
			status, body, _ = httpDo(t, http.MethodGet, base+"/Patient?_id="+created.ID, "", nil)
			if status != http.StatusOK {
				t.Fatalf("search status = %d, want 200; body=%s", status, body)
			}
			var bundle struct {
				ResourceType string `json:"resourceType"`
				Type         string `json:"type"`
				Total        int    `json:"total"`
				Entry        []json.RawMessage
			}
			if err := json.Unmarshal(body, &bundle); err != nil {
				t.Fatalf("search bundle decode: %v", err)
			}
			if bundle.ResourceType != "Bundle" || bundle.Type != "searchset" {
				t.Errorf("search bundle = %s, want a searchset Bundle", body)
			}
			if bundle.Total != 1 || len(bundle.Entry) != 1 {
				t.Errorf("search bundle total=%d entries=%d, want 1/1", bundle.Total, len(bundle.Entry))
			}
		})
	}
}

// TestFHIRRoleCreateAssignsIDAndNeverOverwrites proves a create with a client-supplied id never
// clobbers an existing resource: FHIR create makes a new resource (it is not the update path, which
// this role does not expose), so the server assigns the id and ignores the client's. Two creates that
// both carry the same non-numeric client id must yield two distinct resources under server-assigned
// ids, with the first still present — never a single overwritten resource that bypasses concurrency
// control. The client id is deliberately non-numeric so it cannot coincide with the monotonic
// server-assigned id, making "the server ignored the client id" observable.
func TestFHIRRoleCreateAssignsIDAndNeverOverwrites(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			const clientID = "client-supplied-id"
			firstID := createPatientWithClientID(t, base, release, clientID)
			secondID := createPatientWithClientID(t, base, release, clientID)

			// The server assigned both ids; neither is the client's id and the two differ, so the second
			// create made a new resource rather than overwriting the first.
			if firstID == clientID || secondID == clientID {
				t.Fatalf("server honoured the client id: got first=%q second=%q, want server-assigned ids", firstID, secondID)
			}
			if firstID == secondID {
				t.Fatalf("two creates collapsed onto one id %q; the second overwrote the first", firstID)
			}

			// Both resources are present: the first survived the second create.
			for _, id := range []string{firstID, secondID} {
				status, body, _ := httpDo(t, http.MethodGet, base+"/Patient/"+id, "", nil)
				if status != http.StatusOK {
					t.Fatalf("read Patient/%s status = %d, want 200; body=%s", id, status, body)
				}
			}
			assertWorkflowCount(t, base, "Patient", 2)
		})
	}
}

// TestFHIRRoleTransactionCreateNeverOverwrites is the transaction twin of the direct-create guard: a
// transaction whose two POST entries both carry id "1" must commit two distinct resources under
// server-assigned ids, atomically, never a single overwritten Patient/1. The transaction shares the
// create path, so the server-assigns-id rule holds for transaction creates exactly as for direct ones.
func TestFHIRRoleTransactionCreateNeverOverwrites(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			bundle := collidingCreateTransactionBundle(t, release)
			status, body, _ := httpDo(t, http.MethodPost, base, "application/fhir+json", bundle)
			if status != http.StatusOK {
				t.Fatalf("transaction status = %d, want 200; body=%s", status, body)
			}
			var resp struct {
				Entry []struct {
					Response struct {
						Status   string `json:"status"`
						Location string `json:"location"`
					} `json:"response"`
				} `json:"entry"`
			}
			if err := json.Unmarshal(body, &resp); err != nil {
				t.Fatalf("transaction response decode: %v", err)
			}
			if len(resp.Entry) != 2 {
				t.Fatalf("transaction response entries = %d, want 2; body=%s", len(resp.Entry), body)
			}
			for i, e := range resp.Entry {
				if !strings.HasPrefix(e.Response.Status, "201") {
					t.Errorf("entry %d status = %q, want a 201", i, e.Response.Status)
				}
			}
			if loc0, loc1 := resp.Entry[0].Response.Location, resp.Entry[1].Response.Location; loc0 == loc1 {
				t.Fatalf("both transaction creates landed at %q; the second overwrote the first", loc0)
			}

			// Both Patients are committed under distinct server ids, proving no silent overwrite and that
			// the atomic transaction created two resources.
			assertWorkflowCount(t, base, "Patient", 2)
		})
	}
}

func TestFHIRRoleReadNotFoundIsOperationOutcome(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()
			status, body, _ := httpDo(t, http.MethodGet, base+"/Patient/missing", "", nil)
			if status != http.StatusNotFound {
				t.Fatalf("read missing status = %d, want 404; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")
		})
	}
}

func TestFHIRRoleCreateInvalidResourceIsOperationOutcome(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()
			// A body that is not a FHIR resource (no resourceType) is a 400 with an OperationOutcome.
			status, body, _ := httpDo(t, http.MethodPost, base+"/Patient", "application/fhir+json",
				[]byte(`{"not":"a resource"}`))
			if status != http.StatusBadRequest {
				t.Fatalf("create invalid status = %d, want 400; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")

			// A body whose resourceType does not match the endpoint is a 400.
			status, body, _ = httpDo(t, http.MethodPost, base+"/Patient", "application/fhir+json",
				[]byte(`{"resourceType":"Observation"}`))
			if status != http.StatusBadRequest {
				t.Fatalf("create type-mismatch status = %d, want 400; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")
		})
	}
}

func TestFHIRRoleDeferredInteractionIsNotImplemented(t *testing.T) {
	base, cleanup := startFHIRDaemon(t, fhir.R5)
	defer cleanup()
	// A PUT (update) is a deferred interaction: a 501 OperationOutcome, never a silent no-op.
	status, body, _ := httpDo(t, http.MethodPut, base+"/Patient/1", "application/fhir+json",
		patientJSON(fhir.R5, "female"))
	if status != http.StatusNotImplemented {
		t.Fatalf("update status = %d, want 501; body=%s", status, body)
	}
	assertOperationOutcome(t, body, "error")

	// history and vread are recognized FHIR interactions this role defers; per the contract they
	// answer 501 (deferred), not the 405 used for an unknown route.
	for _, path := range []string{"/Patient/1/_history", "/Patient/1/_history/2"} {
		status, body, _ := httpDo(t, http.MethodGet, base+path, "", nil)
		if status != http.StatusNotImplemented {
			t.Errorf("GET %s status = %d, want 501; body=%s", path, status, body)
		}
		assertOperationOutcome(t, body, "error")
	}
}

func TestFHIRRoleUnservedResourceType(t *testing.T) {
	base, cleanup := startFHIRDaemon(t, fhir.R5)
	defer cleanup()
	// A type outside the workflow set is rejected, never routed to the repository.
	status, body, _ := httpDo(t, http.MethodGet, base+"/Medication/1", "", nil)
	if status != http.StatusNotFound {
		t.Fatalf("unserved type status = %d, want 404; body=%s", status, body)
	}
	assertOperationOutcome(t, body, "error")
}

func TestFHIRRoleTransaction(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()
			bundle := transactionRequestBundle(t, release)
			status, body, _ := httpDo(t, http.MethodPost, base, "application/fhir+json", bundle)
			if status != http.StatusOK {
				t.Fatalf("transaction status = %d, want 200; body=%s", status, body)
			}
			var resp struct {
				ResourceType string `json:"resourceType"`
				Type         string `json:"type"`
				Entry        []struct {
					Response struct {
						Status string `json:"status"`
					} `json:"response"`
				} `json:"entry"`
			}
			if err := json.Unmarshal(body, &resp); err != nil {
				t.Fatalf("transaction response decode: %v", err)
			}
			if resp.ResourceType != "Bundle" || resp.Type != "transaction-response" {
				t.Errorf("transaction response = %s, want a transaction-response Bundle", body)
			}
			if len(resp.Entry) != 1 || !strings.HasPrefix(resp.Entry[0].Response.Status, "201") {
				t.Errorf("transaction response entries = %+v, want one 201 response", resp.Entry)
			}
		})
	}
}

// TestFHIRRoleClientTransactionAtBase drives the real fhir/rest client's Transaction against the
// in-process role, proving the client and the server agree on the transaction endpoint: the client
// POSTs to the exact base path (no trailing slash), and the role must route that to the transaction
// handler rather than answering a subtree-redirect 405. This is the end-to-end guard for the base
// path registration — a client cannot run a transaction against its own server unless the exact base
// is served.
func TestFHIRRoleClientTransactionAtBase(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			// A raw POST to the exact base (no trailing slash, no redirect-following) must reach the
			// transaction handler directly: a 200 with a transaction-response, never a 3xx subtree
			// redirect. Redirect-following clients would mask the registration gap, so this asserts the
			// exact-base pattern serves the POST without a bounce.
			noRedirect := &http.Client{
				CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
			}
			req, err := http.NewRequest(http.MethodPost, base, bytes.NewReader(transactionRequestBundle(t, release)))
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}
			req.Header.Set("Content-Type", "application/fhir+json")
			rawResp, err := noRedirect.Do(req)
			if err != nil {
				t.Fatalf("raw POST to exact base: %v", err)
			}
			_ = rawResp.Body.Close()
			if rawResp.StatusCode != http.StatusOK {
				t.Fatalf("raw POST to exact base status = %d, want 200 (a 3xx means the exact base redirects instead of serving the transaction)", rawResp.StatusCode)
			}

			client, err := rest.NewClient(release, base)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			bundle := transactionBundleResource(t, release)

			resp, err := client.Transaction(context.Background(), bundle)
			if err != nil {
				t.Fatalf("client.Transaction against the in-process role: %v", err)
			}
			if resp.ResourceType() != "Bundle" {
				t.Fatalf("transaction response resourceType = %q, want Bundle", resp.ResourceType())
			}
			out, _ := json.Marshal(resp)
			var decoded struct {
				Type  string `json:"type"`
				Entry []struct {
					Response struct {
						Status string `json:"status"`
					} `json:"response"`
				} `json:"entry"`
			}
			if err := json.Unmarshal(out, &decoded); err != nil {
				t.Fatalf("transaction response decode: %v", err)
			}
			if decoded.Type != "transaction-response" {
				t.Errorf("transaction response type = %q, want transaction-response", decoded.Type)
			}
			if len(decoded.Entry) != 1 || !strings.HasPrefix(decoded.Entry[0].Response.Status, "201") {
				t.Errorf("transaction response entries = %+v, want one 201 response", decoded.Entry)
			}
		})
	}
}

// TestFHIRRoleRejectsNonTransactionBundleAtBase proves the system endpoint processes only a
// transaction Bundle: a collection Bundle POSTed to the base is rejected with a 400 OperationOutcome
// before the repository is touched, never run as a transaction and answered with a 200
// transaction-response. The role advertises only the transaction system interaction, so accepting a
// collection or searchset Bundle would silently process a request it does not implement.
func TestFHIRRoleRejectsNonTransactionBundleAtBase(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			collection := collectionBundleBytes(t, release)
			status, body, _ := httpDo(t, http.MethodPost, base, "application/fhir+json", collection)
			if status != http.StatusBadRequest {
				t.Fatalf("collection Bundle at base status = %d, want 400; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")

			// A transaction Bundle at the same endpoint still succeeds, so the type check rejects only the
			// unsupported Bundle types rather than breaking the supported one.
			txn := transactionRequestBundle(t, release)
			status, body, _ = httpDo(t, http.MethodPost, base, "application/fhir+json", txn)
			if status != http.StatusOK {
				t.Fatalf("transaction Bundle at base status = %d, want 200; body=%s", status, body)
			}
		})
	}
}

// TestMemoryRepositoryTransactionRollback proves the in-memory repository's transaction is atomic: a
// two-entry transaction whose second entry fails leaves the first resource absent (no partial
// commit), and a fully valid transaction commits every entry. A partial commit on a clinical store
// would leave an orphaned resource a later read could surface, which the role's advertised
// all-or-nothing transaction interaction forbids.
func TestMemoryRepositoryTransactionRollback(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			repo, err := NewMemoryRepository(release)
			if err != nil {
				t.Fatalf("NewMemoryRepository: %v", err)
			}
			ctx := context.Background()

			// A two-entry transaction: a valid create followed by a GET of a resource that does not
			// exist, which fails the transaction. Nothing must be committed.
			failing := failingTransactionBundle(t, release)
			if _, err := repo.Transaction(ctx, failing); err == nil {
				t.Fatal("failing transaction returned nil error, want a transaction failure")
			}

			// The repository must hold no Patient: the first entry's create was rolled back.
			searched, err := repo.Search(ctx, "Patient", nil)
			if err != nil {
				t.Fatalf("Search after rollback: %v", err)
			}
			if total := searchsetTotal(t, searched); total != 0 {
				t.Fatalf("after a failed transaction the repository holds %d Patients, want 0 (no partial commit)", total)
			}

			// A fully valid single-entry transaction commits.
			ok := transactionBundleResource(t, release)
			if _, err := repo.Transaction(ctx, ok); err != nil {
				t.Fatalf("valid transaction: %v", err)
			}
			searched, err = repo.Search(ctx, "Patient", nil)
			if err != nil {
				t.Fatalf("Search after commit: %v", err)
			}
			if total := searchsetTotal(t, searched); total != 1 {
				t.Fatalf("after a valid transaction the repository holds %d Patients, want 1", total)
			}
		})
	}
}

// TestMemoryRepositoryTransactionPreservesConcurrentCreates proves the transaction's atomic rollback
// does not discard concurrent writes. A failing transaction runs alongside many concurrent Creates;
// after the transaction fails, every concurrently created resource must still be present and the
// failed transaction's own entry must be absent. The earlier design snapshotted the store without
// holding the lock and, on failure, restored that older snapshot — silently dropping any Create that
// committed after the snapshot was taken, unrelated clinical-data loss. Run under -race, this also
// guards the lock discipline of the snapshot/apply/restore window.
func TestMemoryRepositoryTransactionPreservesConcurrentCreates(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			repo, err := NewMemoryRepository(release)
			if err != nil {
				t.Fatalf("NewMemoryRepository: %v", err)
			}
			ctx := context.Background()

			const concurrentCreates = 50
			createdIDs := make(chan string, concurrentCreates)

			var wg sync.WaitGroup
			// One goroutine runs a transaction that ultimately fails (its second entry GETs a missing
			// resource); the rest run independent Creates of an Encounter. The failing transaction must
			// not roll back any of the concurrent Encounters.
			wg.Add(1)
			go func() {
				defer wg.Done()
				failing := failingTransactionBundle(t, release)
				if _, terr := repo.Transaction(ctx, failing); terr == nil {
					t.Error("failing transaction returned nil error, want a transaction failure")
				}
			}()

			for i := 0; i < concurrentCreates; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					enc := newEncounterResource(release)
					stored, cerr := repo.Create(ctx, enc)
					if cerr != nil {
						t.Errorf("concurrent Create: %v", cerr)
						return
					}
					createdIDs <- resourceLogicalIDForTest(t, stored)
				}()
			}

			wg.Wait()
			close(createdIDs)

			// Every concurrently created Encounter must still be readable: the failed transaction discarded
			// none of them.
			present := 0
			for id := range createdIDs {
				if id == "" {
					t.Error("concurrent Create returned a resource with no id")
					continue
				}
				if _, rerr := repo.Read(ctx, "Encounter", id); rerr != nil {
					t.Errorf("Encounter/%s lost after a failed concurrent transaction: %v", id, rerr)
					continue
				}
				present++
			}
			if present != concurrentCreates {
				t.Fatalf("after a failed transaction %d of %d concurrent Creates survive, want all", present, concurrentCreates)
			}

			// The failed transaction's own Patient entry must be absent (atomic rollback of its own work).
			searched, err := repo.Search(ctx, "Patient", nil)
			if err != nil {
				t.Fatalf("Search Patient after failed transaction: %v", err)
			}
			if total := searchsetTotal(t, searched); total != 0 {
				t.Fatalf("failed transaction left %d Patients, want 0 (its own entries must not commit)", total)
			}

			// A subsequent valid transaction still commits all its entries.
			ok := transactionBundleResource(t, release)
			if _, err := repo.Transaction(ctx, ok); err != nil {
				t.Fatalf("valid transaction after the failed one: %v", err)
			}
			searched, err = repo.Search(ctx, "Patient", nil)
			if err != nil {
				t.Fatalf("Search Patient after valid transaction: %v", err)
			}
			if total := searchsetTotal(t, searched); total != 1 {
				t.Fatalf("valid transaction committed %d Patients, want 1", total)
			}
		})
	}
}

// TestFHIRRoleWriteRequiresFHIRMediaType proves a write must declare the FHIR JSON content type before
// the body is decoded. A create sent as text/plain is a 415 OperationOutcome and does not mutate the
// store; application/fhir+json and the generic application/json both succeed.
func TestFHIRRoleWriteRequiresFHIRMediaType(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			// text/plain is unsupported on a write: 415 with an OperationOutcome, before any decode/store.
			status, body, _ := httpDo(t, http.MethodPost, base+"/Patient", "text/plain",
				patientJSON(release, "female"))
			if status != http.StatusUnsupportedMediaType {
				t.Fatalf("create with text/plain status = %d, want 415; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")
			// No mutation: the store still holds no Patient.
			assertWorkflowCount(t, base, "Patient", 0)

			// A missing Content-Type is likewise unsupported on a write.
			status, body, _ = httpDo(t, http.MethodPost, base+"/Patient", "", patientJSON(release, "female"))
			if status != http.StatusUnsupportedMediaType {
				t.Fatalf("create with no Content-Type status = %d, want 415; body=%s", status, body)
			}
			assertWorkflowCount(t, base, "Patient", 0)

			// application/fhir+json succeeds.
			status, body, _ = httpDo(t, http.MethodPost, base+"/Patient", "application/fhir+json",
				patientJSON(release, "female"))
			if status != http.StatusCreated {
				t.Fatalf("create with application/fhir+json status = %d, want 201; body=%s", status, body)
			}

			// The generic application/json (with a charset parameter) also succeeds, as FHIR permits.
			status, body, _ = httpDo(t, http.MethodPost, base+"/Patient", "application/json; charset=utf-8",
				patientJSON(release, "male"))
			if status != http.StatusCreated {
				t.Fatalf("create with application/json status = %d, want 201; body=%s", status, body)
			}

			assertWorkflowCount(t, base, "Patient", 2)
		})
	}
}

// TestFHIRRoleTransactionValidatesEntries proves a transaction's POST entries go through the same
// create-validation gate a direct create POST does, so a transaction cannot commit a resource the
// direct path would reject. An Encounter missing its required status is a 422 OperationOutcome via
// POST /Encounter; the same Encounter inside a transaction (alongside a valid Patient) must reject the
// whole transaction and commit nothing — the valid sibling must be absent too, because the transaction
// is atomic. A transaction whose entries are all valid commits. An out-of-scope resource type inside a
// transaction is rejected the same way handleCreate rejects an out-of-scope create.
func TestFHIRRoleTransactionValidatesEntries(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			// A direct create of an Encounter with no status is a 422 (well-formed but unprocessable).
			status, body, _ := httpDo(t, http.MethodPost, base+"/Encounter", "application/fhir+json",
				encounterNoStatusJSON(t, release))
			if status != http.StatusUnprocessableEntity {
				t.Fatalf("direct create of an invalid Encounter status = %d, want 422; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")

			// The same invalid Encounter inside a transaction (with a valid Patient sibling) must reject the
			// whole transaction with an error OperationOutcome and commit nothing.
			status, body, _ = httpDo(t, http.MethodPost, base, "application/fhir+json",
				invalidEntryTransactionBundle(t, release))
			if status != http.StatusUnprocessableEntity {
				t.Fatalf("transaction with an invalid Encounter status = %d, want 422; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")

			// Atomic: the valid Patient sibling must not have committed.
			assertWorkflowCount(t, base, "Patient", 0)
			assertWorkflowCount(t, base, "Encounter", 0)

			// A transaction whose entries are all valid commits.
			status, body, _ = httpDo(t, http.MethodPost, base, "application/fhir+json",
				transactionRequestBundle(t, release))
			if status != http.StatusOK {
				t.Fatalf("all-valid transaction status = %d, want 200; body=%s", status, body)
			}
			assertWorkflowCount(t, base, "Patient", 1)

			// An out-of-scope resource type inside a transaction is rejected (400), the same class of
			// rejection handleCreate gives an out-of-scope create, and commits nothing.
			status, body, _ = httpDo(t, http.MethodPost, base, "application/fhir+json",
				outOfScopeTransactionBundle(t, release))
			if status != http.StatusBadRequest {
				t.Fatalf("transaction with an out-of-scope type status = %d, want 400; body=%s", status, body)
			}
			assertOperationOutcome(t, body, "error")
			// The Patient count is unchanged: the out-of-scope transaction committed nothing.
			assertWorkflowCount(t, base, "Patient", 1)
		})
	}
}

// TestFHIRRoleAuthRejectionIsOperationOutcome proves the FHIR role's auth rejection flows through the
// FHIR error contract: a non-loopback bind with a real Authenticator that rejects a request answers
// 401 with an application/fhir+json OperationOutcome body, not net/http's plain-text "unauthorized". An
// accepted request proceeds to the handler. The auth decision is the Authenticator's; only the
// rejection response format is the FHIR one.
func TestFHIRRoleAuthRejectionIsOperationOutcome(t *testing.T) {
	repo, err := NewMemoryRepository(fhir.R5)
	if err != nil {
		t.Fatalf("NewMemoryRepository: %v", err)
	}
	role, err := NewFHIRRole(repo, WithFHIRPort(0), WithFHIRRelease(fhir.R5))
	if err != nil {
		t.Fatalf("NewFHIRRole: %v", err)
	}
	// A non-loopback bind (permitted only with an explicit Authenticator) with an Authenticator that
	// rejects every HTTP request unless it carries the sentinel bearer token.
	d, err := New(WithFHIR(role), WithBind("0.0.0.0"),
		WithAuthenticator(bearerAuth{token: "sentinel-token"}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	waitForAddrs(t, d, "fhir@/fhir")
	defer func() {
		cancelRun()
		select {
		case <-runErr:
		case <-time.After(5 * time.Second):
			t.Error("daemon did not stop within 5s")
		}
	}()

	bound := d.Addrs()["fhir@/fhir"]
	if bound == nil {
		t.Fatal("daemon reported no FHIR address after start")
	}
	_, port, err := net.SplitHostPort(bound.String())
	if err != nil {
		t.Fatalf("bound addr %q not host:port: %v", bound, err)
	}
	base := "http://127.0.0.1:" + port + "/fhir"

	// A request with no credentials is rejected: 401 with an application/fhir+json OperationOutcome.
	req, err := http.NewRequest(http.MethodGet, base+"/metadata", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do (unauthenticated): %v", err)
	}
	rejectedBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401; body=%s", resp.StatusCode, rejectedBody)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/fhir+json" {
		t.Errorf("401 Content-Type = %q, want application/fhir+json (not plain text)", ct)
	}
	assertOperationOutcome(t, rejectedBody, "error")

	// A request carrying the sentinel token is admitted and reaches the handler.
	req, err = http.NewRequest(http.MethodGet, base+"/metadata", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer sentinel-token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do (authenticated): %v", err)
	}
	okBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("authenticated metadata status = %d, want 200; body=%s", resp.StatusCode, okBody)
	}
	var cs struct {
		ResourceType string `json:"resourceType"`
	}
	if err := json.Unmarshal(okBody, &cs); err != nil {
		t.Fatalf("metadata decode: %v", err)
	}
	if cs.ResourceType != "CapabilityStatement" {
		t.Errorf("authenticated request resourceType = %q, want CapabilityStatement", cs.ResourceType)
	}
}

func TestFHIRRoleCapabilityStatement(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()
			status, body, _ := httpDo(t, http.MethodGet, base+"/metadata", "", nil)
			if status != http.StatusOK {
				t.Fatalf("metadata status = %d, want 200; body=%s", status, body)
			}
			var cs struct {
				ResourceType string `json:"resourceType"`
				FhirVersion  string `json:"fhirVersion"`
				Rest         []struct {
					Mode        string `json:"mode"`
					Interaction []struct {
						Code string `json:"code"`
					} `json:"interaction"`
					Resource []struct {
						Type        string `json:"type"`
						Interaction []struct {
							Code string `json:"code"`
						} `json:"interaction"`
					} `json:"resource"`
				} `json:"rest"`
			}
			if err := json.Unmarshal(body, &cs); err != nil {
				t.Fatalf("metadata decode: %v", err)
			}
			if cs.ResourceType != "CapabilityStatement" {
				t.Fatalf("metadata resourceType = %q, want CapabilityStatement", cs.ResourceType)
			}
			// The served metadata must be a VALID CapabilityStatement for its release: a client or
			// conformance test that validates /metadata will reject the role otherwise (this caught a
			// missing required CapabilityStatement.date). Decode and validate through the release.
			switch release {
			case fhir.R4:
				res, err := r4.UnmarshalResource(body)
				if err != nil {
					t.Fatalf("metadata is not a decodable R4 resource: %v", err)
				}
				if oo := r4.Validate(res); oo.HasErrors() {
					t.Errorf("served CapabilityStatement fails R4 validation: %s", oo.Error())
				}
			case fhir.R5:
				res, err := r5.UnmarshalResource(body)
				if err != nil {
					t.Fatalf("metadata is not a decodable R5 resource: %v", err)
				}
				if oo := r5.Validate(res); oo.HasErrors() {
					t.Errorf("served CapabilityStatement fails R5 validation: %s", oo.Error())
				}
			}
			if cs.FhirVersion != string(release) {
				t.Errorf("fhirVersion = %q, want %q", cs.FhirVersion, string(release))
			}
			if len(cs.Rest) == 0 || cs.Rest[0].Mode != "server" {
				t.Fatalf("metadata rest = %+v, want one server-mode rest", cs.Rest)
			}
			if !advertisesResourceInteraction(cs.Rest[0].Resource, "Patient", "read") {
				t.Error("metadata: expected Patient read to be advertised")
			}
			if !advertisesResourceInteraction(cs.Rest[0].Resource, "Patient", "create") {
				t.Error("metadata: expected Patient create to be advertised")
			}
			// The served metadata must list only the system interactions the handler implements:
			// transaction (the base POST) and nothing else. The handler does not return a batch-response
			// and answers GET at the base with a 405, so advertising batch or search-system would
			// over-advertise. A client preflighting via /metadata must see an accurate picture.
			system := map[string]bool{}
			for _, i := range cs.Rest[0].Interaction {
				system[i.Code] = true
			}
			if !system["transaction"] {
				t.Error("metadata: expected the transaction system interaction to be advertised")
			}
			if system["batch"] {
				t.Error("metadata: batch is advertised but the handler does not return a batch-response")
			}
			if system["search-system"] {
				t.Error("metadata: search-system is advertised but the handler does not implement it")
			}
			if len(cs.Rest[0].Interaction) != 1 {
				t.Errorf("metadata: system interactions = %+v, want only transaction", cs.Rest[0].Interaction)
			}
		})
	}
}

func TestFHIRRoleNonLoopbackBindWithoutAuthIsInsecure(t *testing.T) {
	repo, err := NewMemoryRepository(fhir.R5)
	if err != nil {
		t.Fatalf("NewMemoryRepository: %v", err)
	}
	role, err := NewFHIRRole(repo, WithFHIRPort(0))
	if err != nil {
		t.Fatalf("NewFHIRRole: %v", err)
	}
	// A non-loopback bind with no explicit Authenticator must fail closed with ErrInsecureBind.
	if _, err := New(WithFHIR(role), WithBind("0.0.0.0")); !errors.Is(err, ErrInsecureBind) {
		t.Fatalf("New(non-loopback FHIR bind, no auth) = %v, want ErrInsecureBind", err)
	}
	// The same bind with an explicit Authenticator is accepted.
	if _, err := New(WithFHIR(role), WithBind("0.0.0.0"), WithAuthenticator(AllowAll())); err != nil {
		t.Fatalf("New(non-loopback FHIR bind, AllowAll) = %v, want nil", err)
	}
	// The default loopback bind is accepted with no Authenticator.
	if _, err := New(WithFHIR(role)); err != nil {
		t.Fatalf("New(loopback FHIR bind) = %v, want nil", err)
	}
}

func TestNewFHIRRoleRejectsNilRepoAndBadRelease(t *testing.T) {
	if _, err := NewFHIRRole(nil); err == nil {
		t.Error("NewFHIRRole(nil) should error")
	}
	repo, _ := NewMemoryRepository(fhir.R5)
	if _, err := NewFHIRRole(repo, WithFHIRRelease(fhir.Release("3.0.0"))); err == nil {
		t.Error("NewFHIRRole with an unsupported release should error")
	}
	if _, err := NewMemoryRepository(fhir.Release("3.0.0")); err == nil {
		t.Error("NewMemoryRepository with an unsupported release should error")
	}
}

// --- helpers ---

func assertOperationOutcome(t *testing.T, body []byte, wantSeverity string) {
	t.Helper()
	var oo struct {
		ResourceType string `json:"resourceType"`
		Issue        []struct {
			Severity string `json:"severity"`
			Code     string `json:"code"`
		} `json:"issue"`
	}
	if err := json.Unmarshal(body, &oo); err != nil {
		t.Fatalf("OperationOutcome decode: %v; body=%s", err, body)
	}
	if oo.ResourceType != "OperationOutcome" {
		t.Fatalf("error body resourceType = %q, want OperationOutcome; body=%s", oo.ResourceType, body)
	}
	if len(oo.Issue) == 0 {
		t.Fatalf("OperationOutcome has no issues; body=%s", body)
	}
	if oo.Issue[0].Severity != wantSeverity {
		t.Errorf("issue severity = %q, want %q", oo.Issue[0].Severity, wantSeverity)
	}
}

func advertisesResourceInteraction(resources []struct {
	Type        string `json:"type"`
	Interaction []struct {
		Code string `json:"code"`
	} `json:"interaction"`
}, resourceType, code string) bool {
	for _, r := range resources {
		if r.Type != resourceType {
			continue
		}
		for _, i := range r.Interaction {
			if i.Code == code {
				return true
			}
		}
	}
	return false
}

// transactionBundleResource builds a single-entry transaction Bundle (one Patient create) as a
// fhir.Resource, for the client Transaction and the repository rollback tests.
func transactionBundleResource(t *testing.T, release fhir.Release) fhir.Resource {
	t.Helper()
	switch release {
	case fhir.R4:
		g := r4.AdministrativeGender("female")
		b, err := r4.NewTransaction(r4.TransactionEntry{
			Resource: &r4.Patient{Gender: &g},
			Method:   r4.HTTPVerbPOST,
			URL:      "Patient",
		})
		if err != nil {
			t.Fatalf("r4 NewTransaction: %v", err)
		}
		return b
	default:
		g := r5.AdministrativeGender("female")
		b, err := r5.NewTransaction(r5.TransactionEntry{
			Resource: &r5.Patient{Gender: &g},
			Method:   r5.HTTPVerbPOST,
			URL:      "Patient",
		})
		if err != nil {
			t.Fatalf("r5 NewTransaction: %v", err)
		}
		return b
	}
}

// failingTransactionBundle builds a two-entry transaction whose first entry is a valid Patient create
// and whose second entry is a GET of a resource that does not exist, so the transaction fails on the
// second entry. It is the input for the rollback test: the first create must not survive the failure.
func failingTransactionBundle(t *testing.T, release fhir.Release) fhir.Resource {
	t.Helper()
	switch release {
	case fhir.R4:
		g := r4.AdministrativeGender("female")
		b, err := r4.NewTransaction(
			r4.TransactionEntry{Resource: &r4.Patient{Gender: &g}, Method: r4.HTTPVerbPOST, URL: "Patient"},
			r4.TransactionEntry{Method: r4.HTTPVerbGET, URL: "Patient/does-not-exist"},
		)
		if err != nil {
			t.Fatalf("r4 NewTransaction: %v", err)
		}
		return b
	default:
		g := r5.AdministrativeGender("female")
		b, err := r5.NewTransaction(
			r5.TransactionEntry{Resource: &r5.Patient{Gender: &g}, Method: r5.HTTPVerbPOST, URL: "Patient"},
			r5.TransactionEntry{Method: r5.HTTPVerbGET, URL: "Patient/does-not-exist"},
		)
		if err != nil {
			t.Fatalf("r5 NewTransaction: %v", err)
		}
		return b
	}
}

// searchsetTotal reads the total of a searchset Bundle returned by the repository, so the rollback
// test can assert the Patient count without depending on a release's concrete Bundle type.
func searchsetTotal(t *testing.T, bundle fhir.Resource) int {
	t.Helper()
	data, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("marshal searchset: %v", err)
	}
	var env struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("decode searchset total: %v", err)
	}
	return env.Total
}

// newEncounterResource builds a minimal Encounter of the release for the concurrency test. The
// repository stores what it is given without validating (validation is the role handler's job), so a
// status-only Encounter is enough to exercise concurrent Creates.
func newEncounterResource(release fhir.Release) fhir.Resource {
	switch release {
	case fhir.R4:
		s := r4.EncounterStatus("finished")
		return &r4.Encounter{Status: &s}
	default:
		s := r5.EncounterStatus("completed")
		return &r5.Encounter{Status: &s}
	}
}

// resourceLogicalIDForTest reads a stored resource's logical id by peeking its JSON "id" key, so the
// concurrency test can read each concurrently created Encounter back by id without depending on a
// release's concrete type.
func resourceLogicalIDForTest(t *testing.T, r fhir.Resource) string {
	t.Helper()
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal resource for id: %v", err)
	}
	var env struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("decode resource id: %v", err)
	}
	return env.ID
}

// collectionBundleBytes builds an empty collection Bundle of the release as JSON, the input for the
// non-transaction-Bundle rejection test: a collection is a valid Bundle the system endpoint must
// reject rather than process as a transaction.
func collectionBundleBytes(t *testing.T, release fhir.Release) []byte {
	t.Helper()
	switch release {
	case fhir.R4:
		b, err := r4.NewCollection()
		if err != nil {
			t.Fatalf("r4 NewCollection: %v", err)
		}
		out, _ := json.Marshal(b)
		return out
	default:
		b, err := r5.NewCollection()
		if err != nil {
			t.Fatalf("r5 NewCollection: %v", err)
		}
		out, _ := json.Marshal(b)
		return out
	}
}

// bearerAuth is an Authenticator that admits an HTTP request only when it carries the configured
// bearer token, the test seam proving the FHIR role's rejection flows through the OperationOutcome
// adapter rather than net/http's plain-text body. It admits every DIMSE association (this test
// exercises only the HTTP path).
type bearerAuth struct {
	token string
}

func (a bearerAuth) AuthenticateHTTP(_ context.Context, r *http.Request) (Principal, error) {
	if r.Header.Get("Authorization") != "Bearer "+a.token {
		return Principal{}, errors.New("missing or invalid bearer token")
	}
	return Principal{ID: "test-subject"}, nil
}

func (a bearerAuth) AuthenticateDIMSE(_ context.Context, calling dimse.AETitle) (Principal, error) {
	return Principal{ID: string(calling)}, nil
}

// assertWorkflowCount asserts the role serves exactly want resources of resourceType, read through the
// role's own search so it observes the committed store rather than the repository directly.
func assertWorkflowCount(t *testing.T, base, resourceType string, want int) {
	t.Helper()
	status, body, _ := httpDo(t, http.MethodGet, base+"/"+resourceType, "", nil)
	if status != http.StatusOK {
		t.Fatalf("search %s status = %d, want 200; body=%s", resourceType, status, body)
	}
	var bundle struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(body, &bundle); err != nil {
		t.Fatalf("search %s bundle decode: %v", resourceType, err)
	}
	if bundle.Total != want {
		t.Fatalf("store holds %d %s, want %d", bundle.Total, resourceType, want)
	}
}

// encounterNoStatusJSON builds an Encounter with no status (its required element) as JSON, the
// well-formed-but-unprocessable resource the create-validation gate rejects with a 422.
func encounterNoStatusJSON(t *testing.T, release fhir.Release) []byte {
	t.Helper()
	switch release {
	case fhir.R4:
		b, _ := json.Marshal(&r4.Encounter{})
		return b
	default:
		b, _ := json.Marshal(&r5.Encounter{})
		return b
	}
}

// invalidEntryTransactionBundle builds a two-POST transaction whose first entry is a valid Patient and
// whose second entry is an Encounter missing its required status, so the transaction must reject the
// whole bundle (atomic) on the invalid entry and commit neither.
func invalidEntryTransactionBundle(t *testing.T, release fhir.Release) []byte {
	t.Helper()
	switch release {
	case fhir.R4:
		g := r4.AdministrativeGender("female")
		b, err := r4.NewTransaction(
			r4.TransactionEntry{Resource: &r4.Patient{Gender: &g}, Method: r4.HTTPVerbPOST, URL: "Patient"},
			r4.TransactionEntry{Resource: &r4.Encounter{}, Method: r4.HTTPVerbPOST, URL: "Encounter"},
		)
		if err != nil {
			t.Fatalf("r4 NewTransaction: %v", err)
		}
		out, _ := json.Marshal(b)
		return out
	default:
		g := r5.AdministrativeGender("female")
		b, err := r5.NewTransaction(
			r5.TransactionEntry{Resource: &r5.Patient{Gender: &g}, Method: r5.HTTPVerbPOST, URL: "Patient"},
			r5.TransactionEntry{Resource: &r5.Encounter{}, Method: r5.HTTPVerbPOST, URL: "Encounter"},
		)
		if err != nil {
			t.Fatalf("r5 NewTransaction: %v", err)
		}
		out, _ := json.Marshal(b)
		return out
	}
}

// outOfScopeTransactionBundle builds a single-POST transaction whose entry creates a Medication, a
// resource type the role does not serve, so the create-validation gate rejects it the same way
// handleCreate rejects an out-of-scope direct create.
func outOfScopeTransactionBundle(t *testing.T, release fhir.Release) []byte {
	t.Helper()
	switch release {
	case fhir.R4:
		b, err := r4.NewTransaction(
			r4.TransactionEntry{Resource: &r4.Medication{}, Method: r4.HTTPVerbPOST, URL: "Medication"},
		)
		if err != nil {
			t.Fatalf("r4 NewTransaction: %v", err)
		}
		out, _ := json.Marshal(b)
		return out
	default:
		b, err := r5.NewTransaction(
			r5.TransactionEntry{Resource: &r5.Medication{}, Method: r5.HTTPVerbPOST, URL: "Medication"},
		)
		if err != nil {
			t.Fatalf("r5 NewTransaction: %v", err)
		}
		out, _ := json.Marshal(b)
		return out
	}
}

func transactionRequestBundle(t *testing.T, release fhir.Release) []byte {
	t.Helper()
	switch release {
	case fhir.R4:
		g := r4.AdministrativeGender("female")
		b, err := r4.NewTransaction(r4.TransactionEntry{
			Resource: &r4.Patient{Gender: &g},
			Method:   r4.HTTPVerbPOST,
			URL:      "Patient",
		})
		if err != nil {
			t.Fatalf("r4 NewTransaction: %v", err)
		}
		out, _ := json.Marshal(b)
		return out
	default:
		g := r5.AdministrativeGender("female")
		b, err := r5.NewTransaction(r5.TransactionEntry{
			Resource: &r5.Patient{Gender: &g},
			Method:   r5.HTTPVerbPOST,
			URL:      "Patient",
		})
		if err != nil {
			t.Fatalf("r5 NewTransaction: %v", err)
		}
		out, _ := json.Marshal(b)
		return out
	}
}
