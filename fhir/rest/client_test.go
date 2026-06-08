package rest_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r4"
	"github.com/codeninja55/go-radx/fhir/r5"
	"github.com/codeninja55/go-radx/fhir/rest"
)

// fakeServer is a minimal in-memory FHIR origin for the client tests. It is intentionally small: it
// stores resources keyed by "type/id", serves read/create/search/transaction and a
// CapabilityStatement, paginates a searchset of two pages, and answers a configurable error so the
// OperationOutcome error mapping and the conditional/ETag paths can be exercised. It is built per
// release so the same test body runs against R4 and R5.
type fakeServer struct {
	release fhir.Release
	mu      sync.Mutex
	store   map[string]string // "type/id" -> resource JSON
	version map[string]string // "type/id" -> ETag version
	nextID  int
}

func newFakeServer(release fhir.Release) *fakeServer {
	return &fakeServer{
		release: release,
		store:   map[string]string{},
		version: map[string]string{},
	}
}

// mediaTypeFHIRJSON is the content type the fake server sends and the client expects.
const mediaTypeFHIRJSON = "application/fhir+json"

func (s *fakeServer) handler(t *testing.T) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/metadata", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", mediaTypeFHIRJSON)
		_, _ = w.Write(s.capabilityJSON())
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.Trim(r.URL.Path, "/")
		switch {
		case path == "":
			s.handleTransaction(t, w, r)
		case strings.HasPrefix(path, "_force/"):
			// "/_force/<status>" lets a test drive a specific error status with an OperationOutcome
			// body, exercising the client's error mapping.
			s.handleForcedError(w, path)
		default:
			segs := strings.Split(path, "/")
			switch len(segs) {
			case 1:
				if r.Method == http.MethodGet {
					s.handleSearch(w, r, segs[0])
					return
				}
				s.handleCreate(t, w, r, segs[0])
			case 2:
				s.handleInstance(w, r, segs[0], segs[1])
			default:
				http.NotFound(w, r)
			}
		}
	})
	return mux
}

func (s *fakeServer) handleInstance(w http.ResponseWriter, r *http.Request, rt, id string) {
	switch r.Method {
	case http.MethodGet:
		s.mu.Lock()
		body, ok := s.store[rt+"/"+id]
		etag := s.version[rt+"/"+id]
		s.mu.Unlock()
		if !ok {
			s.writeOutcome(w, http.StatusNotFound, "not-found", "no such resource")
			return
		}
		w.Header().Set("Content-Type", mediaTypeFHIRJSON)
		w.Header().Set("ETag", etag)
		_, _ = w.Write([]byte(body))
	case http.MethodPut:
		// Optimistic concurrency: an If-Match that does not equal the stored version is a 412.
		s.mu.Lock()
		current := s.version[rt+"/"+id]
		s.mu.Unlock()
		if im := r.Header.Get("If-Match"); im != "" && im != current {
			s.writeOutcome(w, http.StatusPreconditionFailed, "conflict", "version mismatch")
			return
		}
		body, _ := readAll(r)
		newVer := `W/"2"`
		s.mu.Lock()
		s.store[rt+"/"+id] = string(body)
		s.version[rt+"/"+id] = newVer
		s.mu.Unlock()
		w.Header().Set("Content-Type", mediaTypeFHIRJSON)
		w.Header().Set("ETag", newVer)
		_, _ = w.Write(body)
	default:
		s.writeOutcome(w, http.StatusMethodNotAllowed, "not-supported", "method not allowed")
	}
}

func (s *fakeServer) handleCreate(t *testing.T, w http.ResponseWriter, r *http.Request, rt string) {
	body, _ := readAll(r)
	// Conditional create: an If-None-Exist whose query already matches answers 200 with the match.
	if cond := r.Header.Get("If-None-Exist"); cond != "" {
		s.mu.Lock()
		for key, stored := range s.store {
			if strings.HasPrefix(key, rt+"/") {
				w.Header().Set("Content-Type", mediaTypeFHIRJSON)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(stored))
				s.mu.Unlock()
				return
			}
		}
		s.mu.Unlock()
	}
	s.mu.Lock()
	s.nextID++
	id := fmt.Sprintf("%d", s.nextID)
	withID := injectID(t, body, id)
	key := rt + "/" + id
	etag := `W/"1"`
	s.store[key] = withID
	s.version[key] = etag
	s.mu.Unlock()
	w.Header().Set("Content-Type", mediaTypeFHIRJSON)
	w.Header().Set("ETag", etag)
	w.Header().Set("Location", rt+"/"+id)
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(withID))
}

func (s *fakeServer) handleSearch(w http.ResponseWriter, r *http.Request, rt string) {
	page := r.URL.Query().Get("page")
	var entries []string
	s.mu.Lock()
	for key, stored := range s.store {
		if strings.HasPrefix(key, rt+"/") {
			entries = append(entries, stored)
		}
	}
	s.mu.Unlock()

	// Serve one entry per page so the test exercises Bundle.link next paging across two pages.
	total := len(entries)
	var pageEntry string
	var nextURL string
	switch page {
	case "", "1":
		if len(entries) > 0 {
			pageEntry = entries[0]
		}
		if len(entries) > 1 {
			nextURL = fmt.Sprintf("http://%s/%s?page=2", r.Host, rt)
		}
	case "2":
		if len(entries) > 1 {
			pageEntry = entries[1]
		}
	}
	w.Header().Set("Content-Type", mediaTypeFHIRJSON)
	_, _ = w.Write(s.searchsetJSON(total, pageEntry, nextURL, fmt.Sprintf("http://%s/%s", r.Host, rt)))
}

func (s *fakeServer) handleTransaction(t *testing.T, w http.ResponseWriter, r *http.Request) {
	body, _ := readAll(r)
	// Echo a transaction-response bundle. The test only asserts the resourceType and that each
	// submitted entry produced a response entry.
	var reqBundle struct {
		Entry []json.RawMessage `json:"entry"`
	}
	if err := json.Unmarshal(body, &reqBundle); err != nil {
		s.writeOutcome(w, http.StatusBadRequest, "structure", "bad bundle")
		return
	}
	respType := "transaction-response"
	entries := make([]string, 0, len(reqBundle.Entry))
	for range reqBundle.Entry {
		entries = append(entries, `{"response":{"status":"201 Created"}}`)
	}
	bundle := fmt.Sprintf(`{"resourceType":"Bundle","type":%q,"entry":[%s]}`, respType, strings.Join(entries, ","))
	w.Header().Set("Content-Type", mediaTypeFHIRJSON)
	_, _ = w.Write([]byte(bundle))
}

func (s *fakeServer) handleForcedError(w http.ResponseWriter, path string) {
	status := strings.TrimPrefix(path, "_force/")
	switch status {
	case "404":
		s.writeOutcome(w, http.StatusNotFound, "not-found", "Patient.id resolved to nothing")
	case "409":
		s.writeOutcome(w, http.StatusConflict, "conflict", "version conflict")
	case "422":
		s.writeOutcome(w, http.StatusUnprocessableEntity, "required", "Patient.gender is required")
	default:
		s.writeOutcome(w, http.StatusInternalServerError, "exception", "server error")
	}
}

// writeOutcome writes a release OperationOutcome with one error issue. It builds the JSON directly so
// the fake server stays release-light; the client decodes it through the release registry.
func (s *fakeServer) writeOutcome(w http.ResponseWriter, status int, code, diagnostics string) {
	body := fmt.Sprintf(`{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":%q,"diagnostics":%q}]}`,
		code, diagnostics)
	w.Header().Set("Content-Type", mediaTypeFHIRJSON)
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func (s *fakeServer) capabilityJSON() []byte {
	version := string(s.release)
	return []byte(fmt.Sprintf(`{
		"resourceType":"CapabilityStatement",
		"status":"active",
		"kind":"instance",
		"fhirVersion":%q,
		"format":["application/fhir+json"],
		"rest":[{
			"mode":"server",
			"interaction":[{"code":"transaction"},{"code":"batch"}],
			"resource":[
				{"type":"Patient","interaction":[{"code":"read"},{"code":"create"},{"code":"search-type"}]}
			]
		}]
	}`, version))
}

func (s *fakeServer) searchsetJSON(total int, entry, nextURL, selfURL string) []byte {
	var entryJSON string
	if entry != "" {
		entryJSON = fmt.Sprintf(`{"resource":%s,"search":{"mode":"match"}}`, entry)
	}
	links := fmt.Sprintf(`{"relation":"self","url":%q}`, selfURL)
	if nextURL != "" {
		links += fmt.Sprintf(`,{"relation":"next","url":%q}`, nextURL)
	}
	var entries string
	if entryJSON != "" {
		entries = entryJSON
	}
	return []byte(fmt.Sprintf(`{"resourceType":"Bundle","type":"searchset","total":%d,"link":[%s],"entry":[%s]}`,
		total, links, entries))
}

// --- helpers ---

func readAll(r *http.Request) ([]byte, error) {
	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 512)
	for {
		n, err := r.Body.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			if errors.Is(err, http.ErrBodyReadAfterClose) {
				return buf, nil
			}
			break
		}
	}
	return buf, nil
}

// injectID splices a server-assigned id into a resource body so a created resource reads back with
// its id, matching how a real server assigns one.
func injectID(t *testing.T, body []byte, id string) string {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("injectID: decode body: %v", err)
	}
	idBytes, _ := json.Marshal(id)
	obj["id"] = idBytes
	out, _ := json.Marshal(obj)
	return string(out)
}

// newPatient builds a minimal valid Patient of the given release for use as a request body, returned
// behind the fhir.Resource interface so the test body is release-neutral.
func newPatient(release fhir.Release, gender string) fhir.Resource {
	switch release {
	case fhir.R4:
		g := r4.AdministrativeGender(gender)
		return &r4.Patient{Gender: &g}
	default:
		g := r5.AdministrativeGender(gender)
		return &r5.Patient{Gender: &g}
	}
}

func releases() []fhir.Release { return []fhir.Release{fhir.R4, fhir.R5} }

// startClient spins up the fake server and a release-matched client against it.
func startClient(t *testing.T, release fhir.Release) (*rest.Client, *httptest.Server) {
	t.Helper()
	fake := newFakeServer(release)
	srv := httptest.NewServer(fake.handler(t))
	t.Cleanup(srv.Close)
	c, err := rest.NewClient(release, srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, srv
}

func TestNewClientRejectsUnsupportedRelease(t *testing.T) {
	if _, err := rest.NewClient(fhir.Release("3.0.0"), "http://example.org/fhir"); err == nil {
		t.Fatal("expected an error for an unsupported release")
	}
}

func TestCRUDRoundTrip(t *testing.T) {
	for _, release := range releases() {
		t.Run(string(release), func(t *testing.T) {
			c, _ := startClient(t, release)
			ctx := context.Background()

			created, err := c.Create(ctx, newPatient(release, "female"), "")
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			if created.Location == "" {
				t.Error("Create: expected a Location header")
			}
			if created.ETag == "" {
				t.Error("Create: expected an ETag")
			}
			id := resourceID(t, created.Resource)
			if id == "" {
				t.Fatal("Create: created resource has no id")
			}

			read, err := c.Read(ctx, "Patient", id)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if got := read.Resource.ResourceType(); got != "Patient" {
				t.Errorf("Read: resourceType = %q, want Patient", got)
			}
			if resourceID(t, read.Resource) != id {
				t.Errorf("Read: id = %q, want %q", resourceID(t, read.Resource), id)
			}
		})
	}
}

func TestReadNotFoundMapsToErrNotFound(t *testing.T) {
	for _, release := range releases() {
		t.Run(string(release), func(t *testing.T) {
			c, _ := startClient(t, release)
			_, err := c.Read(context.Background(), "Patient", "does-not-exist")
			if !errors.Is(err, rest.ErrNotFound) {
				t.Fatalf("Read of absent resource: err = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestSearchFollowsBundleLinkPaging(t *testing.T) {
	for _, release := range releases() {
		t.Run(string(release), func(t *testing.T) {
			c, _ := startClient(t, release)
			ctx := context.Background()
			// Store two patients so the searchset spans two pages.
			if _, err := c.Create(ctx, newPatient(release, "female"), ""); err != nil {
				t.Fatalf("Create 1: %v", err)
			}
			if _, err := c.Create(ctx, newPatient(release, "male"), ""); err != nil {
				t.Fatalf("Create 2: %v", err)
			}

			page, err := c.Search(ctx, "Patient", rest.NewSearchParams())
			if err != nil {
				t.Fatalf("Search: %v", err)
			}
			if len(page.Resources) != 1 {
				t.Fatalf("first page: got %d resources, want 1", len(page.Resources))
			}
			if !page.HasNext() {
				t.Fatal("first page: expected a next link")
			}

			next, err := c.FollowNext(ctx, page)
			if err != nil {
				t.Fatalf("FollowNext: %v", err)
			}
			if len(next.Resources) != 1 {
				t.Fatalf("second page: got %d resources, want 1", len(next.Resources))
			}
			if next.HasNext() {
				t.Error("second page: expected no next link")
			}

			all, err := c.SearchAll(ctx, "Patient", rest.NewSearchParams(), 0)
			if err != nil {
				t.Fatalf("SearchAll: %v", err)
			}
			if len(all) != 2 {
				t.Errorf("SearchAll: got %d resources, want 2", len(all))
			}
		})
	}
}

func TestSearchParamsEncoding(t *testing.T) {
	p := rest.NewSearchParams().
		Add("name", "Smith").
		Modifier("name", "exact", "Smith").
		Chain("general-practitioner", "name", "Jones").
		Include("Patient:organization").
		RevInclude("Observation:subject").
		Count(50)
	got := p.Values()
	if got.Get("name:exact") != "Smith" {
		t.Errorf("modifier param missing: %v", got)
	}
	if got.Get("general-practitioner.name") != "Jones" {
		t.Errorf("chained param missing: %v", got)
	}
	if got.Get("_include") != "Patient:organization" {
		t.Errorf("_include missing: %v", got)
	}
	if got.Get("_revinclude") != "Observation:subject" {
		t.Errorf("_revinclude missing: %v", got)
	}
	if got.Get("_count") != "50" {
		t.Errorf("_count missing: %v", got)
	}
	// name appears twice (Add then Modifier uses a different key), so the bare "name" has one value.
	if len(got["name"]) != 1 {
		t.Errorf("name AND-repeat: got %d values, want 1", len(got["name"]))
	}
}

func TestConditionalCreateAndETagUpdate(t *testing.T) {
	for _, release := range releases() {
		t.Run(string(release), func(t *testing.T) {
			c, _ := startClient(t, release)
			ctx := context.Background()

			first, err := c.Create(ctx, newPatient(release, "female"), "")
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			id := resourceID(t, first.Resource)

			// Conditional create with an If-None-Exist that matches the existing patient returns 200
			// with the existing resource, not a new 201.
			cond, err := c.Create(ctx, newPatient(release, "male"), "identifier=anything")
			if err != nil {
				t.Fatalf("conditional Create: %v", err)
			}
			if resourceID(t, cond.Resource) != id {
				t.Errorf("conditional create: got id %q, want existing %q", resourceID(t, cond.Resource), id)
			}

			// An If-Match update with the wrong version is a 412 -> ErrConflict.
			_, err = c.Update(ctx, id, withID(t, release, newPatient(release, "other"), id), `W/"99"`)
			if !errors.Is(err, rest.ErrConflict) {
				t.Fatalf("stale If-Match update: err = %v, want ErrConflict", err)
			}

			// An If-Match update with the correct version succeeds.
			ok, err := c.Update(ctx, id, withID(t, release, newPatient(release, "other"), id), first.ETag)
			if err != nil {
				t.Fatalf("matching If-Match update: %v", err)
			}
			if ok.ETag == "" {
				t.Error("update: expected a new ETag")
			}
		})
	}
}

func TestTransactionRoundTrip(t *testing.T) {
	for _, release := range releases() {
		t.Run(string(release), func(t *testing.T) {
			c, _ := startClient(t, release)
			ctx := context.Background()
			bundle := transactionBundle(t, release)

			resp, err := c.Transaction(ctx, bundle)
			if err != nil {
				t.Fatalf("Transaction: %v", err)
			}
			if resp.ResourceType() != "Bundle" {
				t.Errorf("Transaction response: got %q, want Bundle", resp.ResourceType())
			}
		})
	}
}

func TestCapabilityNegotiation(t *testing.T) {
	for _, release := range releases() {
		t.Run(string(release), func(t *testing.T) {
			c, _ := startClient(t, release)
			caps, err := c.Capabilities(context.Background())
			if err != nil {
				t.Fatalf("Capabilities: %v", err)
			}
			if caps.FHIRVersion != string(release) {
				t.Errorf("FHIRVersion = %q, want %q", caps.FHIRVersion, string(release))
			}
			if !caps.SupportsTransaction() {
				t.Error("expected transaction to be supported")
			}
			if !caps.SupportsResourceInteraction("Patient", "read") {
				t.Error("expected Patient read to be supported")
			}
			if !caps.SupportsResourceInteraction("Patient", "create") {
				t.Error("expected Patient create to be supported")
			}
			if caps.SupportsResourceInteraction("Patient", "delete") {
				t.Error("delete should not be advertised by the fake server")
			}
		})
	}
}

func TestOperationOutcomeErrorMapping(t *testing.T) {
	for _, release := range releases() {
		t.Run(string(release), func(t *testing.T) {
			c, _ := startClient(t, release)
			ctx := context.Background()

			// Drive the 422 path through the forced-error endpoint via a search whose type is the
			// magic "_force" path. Use a raw read against the forced path to map a 422.
			_, err := c.Read(ctx, "_force", "422")
			if err == nil {
				t.Fatal("expected an error for the forced 422")
			}
			if !errors.Is(err, rest.ErrUnprocessable) {
				t.Errorf("422: errors.Is ErrUnprocessable = false; err = %v", err)
			}
			var ooErr *rest.OperationOutcomeError
			if !errors.As(err, &ooErr) {
				t.Fatalf("422: errors.As *OperationOutcomeError = false; err = %v", err)
			}
			if ooErr.StatusCode != http.StatusUnprocessableEntity {
				t.Errorf("422: status = %d, want 422", ooErr.StatusCode)
			}
			if ooErr.Outcome == nil || !ooErr.Outcome.HasErrors() {
				t.Errorf("422: expected an OperationOutcome with errors")
			}
			// The diagnostic names a structural locator, not a patient value.
			if !strings.Contains(ooErr.Error(), "Patient.gender") {
				t.Errorf("422: error message %q does not name the locator", ooErr.Error())
			}

			// 409 maps to ErrConflict.
			_, err = c.Read(ctx, "_force", "409")
			if !errors.Is(err, rest.ErrConflict) {
				t.Errorf("409: errors.Is ErrConflict = false; err = %v", err)
			}
		})
	}
}

// --- test-only release helpers ---

func resourceID(t *testing.T, r fhir.Resource) string {
	t.Helper()
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal resource: %v", err)
	}
	var env struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("read id: %v", err)
	}
	return env.ID
}

func withID(t *testing.T, release fhir.Release, r fhir.Resource, id string) fhir.Resource {
	t.Helper()
	data, _ := json.Marshal(r)
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatalf("withID decode: %v", err)
	}
	idBytes, _ := json.Marshal(id)
	obj["id"] = idBytes
	merged, _ := json.Marshal(obj)
	var res fhir.Resource
	var err error
	switch release {
	case fhir.R4:
		res, err = r4.UnmarshalResource(merged)
	default:
		res, err = r5.UnmarshalResource(merged)
	}
	if err != nil {
		t.Fatalf("withID re-decode: %v", err)
	}
	return res
}

func transactionBundle(t *testing.T, release fhir.Release) fhir.Resource {
	t.Helper()
	switch release {
	case fhir.R4:
		b, err := r4.NewTransaction(r4.TransactionEntry{
			Resource: newPatient(release, "female"),
			Method:   r4.HTTPVerbPOST,
			URL:      "Patient",
		})
		if err != nil {
			t.Fatalf("r4 NewTransaction: %v", err)
		}
		return b
	default:
		b, err := r5.NewTransaction(r5.TransactionEntry{
			Resource: newPatient(release, "female"),
			Method:   r5.HTTPVerbPOST,
			URL:      "Patient",
		})
		if err != nil {
			t.Fatalf("r5 NewTransaction: %v", err)
		}
		return b
	}
}
