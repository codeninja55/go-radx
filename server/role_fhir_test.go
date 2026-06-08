package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r4"
	"github.com/codeninja55/go-radx/fhir/r5"
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
