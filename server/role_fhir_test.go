package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

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
	waitForAddrs(t, d, "fhir")

	base := "http://" + d.Addrs()["fhir"].String() + "/fhir"
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
