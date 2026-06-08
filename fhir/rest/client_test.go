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
		// "_empty" models a misbehaving server or proxy that answers a read with a 200 and no body. A
		// read must return data, so the client treats this as an error, never a nil-resource success.
		if id == "_empty" {
			w.Header().Set("Content-Type", mediaTypeFHIRJSON)
			w.WriteHeader(http.StatusOK)
			return
		}
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
	// The client asks for the stored resource back with Prefer: return=representation on every write.
	if pref := r.Header.Get("Prefer"); pref != "return=representation" {
		t.Errorf("create: Prefer header = %q, want return=representation", pref)
	}
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
		// The diagnostics carries a synthetic patient-name sentinel a real server might echo back
		// from the submitted resource; the expression carries the structural FHIRPath locator. The
		// client error must surface the locator but never the diagnostics (the PHI-free error
		// contract), so the test can assert the locator is present and the sentinel is absent.
		s.writeOutcomeWithExpression(w, http.StatusUnprocessableEntity, "required",
			"rejected value for patient SENTINEL-DETliff-PHI", "Patient.gender")
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

// writeOutcomeWithExpression writes a one-issue OperationOutcome carrying both a free-text diagnostics
// and a structural expression locator, so a test can assert the client error surfaces the locator but
// not the diagnostics.
func (s *fakeServer) writeOutcomeWithExpression(w http.ResponseWriter, status int, code, diagnostics, expression string) {
	body := fmt.Sprintf(
		`{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":%q,"diagnostics":%q,"expression":[%q]}]}`,
		code, diagnostics, expression)
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

// TestCreateBodylessSuccess proves a compliant server honouring return=minimal — a 2xx with no body,
// carrying Location and ETag — is read as a success, not the old "no resource body" failure. The
// client recovers the assigned id and version from the headers; the resource is absent because the
// server returned none. A real release Patient is posted so the request passes the client's
// release check, and a dedicated mux answers the create with a bodyless 201.
func TestCreateBodylessSuccess(t *testing.T) {
	for _, release := range releases() {
		t.Run(string(release), func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/Patient", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("ETag", `W/"7"`)
				w.Header().Set("Location", "Patient/min-42/_history/7")
				w.WriteHeader(http.StatusCreated)
			})
			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)
			c, err := rest.NewClient(release, srv.URL)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}

			res, err := c.Create(context.Background(), newPatient(release, "female"), "")
			if err != nil {
				t.Fatalf("Create with a bodyless 201: %v", err)
			}
			if res.Resource != nil {
				t.Errorf("bodyless create: expected a nil Resource, got %T", res.Resource)
			}
			if res.ID != "min-42" {
				t.Errorf("bodyless create: id = %q, want min-42 (parsed from Location)", res.ID)
			}
			if res.VersionID != "7" {
				t.Errorf("bodyless create: versionID = %q, want 7 (parsed from ETag)", res.VersionID)
			}
			if res.ETag != `W/"7"` {
				t.Errorf("bodyless create: ETag = %q, want W/\"7\"", res.ETag)
			}
			if res.Location != "Patient/min-42/_history/7" {
				t.Errorf("bodyless create: Location = %q, want the versioned Location", res.Location)
			}
		})
	}
}

// TestCreateRepresentationSuccess proves the representation path still works: a 201 carrying the
// resource body yields the decoded resource together with its id, ETag, and Location.
func TestCreateRepresentationSuccess(t *testing.T) {
	for _, release := range releases() {
		t.Run(string(release), func(t *testing.T) {
			c, _ := startClient(t, release)
			res, err := c.Create(context.Background(), newPatient(release, "female"), "")
			if err != nil {
				t.Fatalf("Create with a representation body: %v", err)
			}
			if res.Resource == nil {
				t.Fatal("representation create: expected a decoded resource")
			}
			if res.ID == "" {
				t.Error("representation create: expected an id")
			}
			if res.ETag == "" {
				t.Error("representation create: expected an ETag")
			}
			if res.Location == "" {
				t.Error("representation create: expected a Location")
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

func TestReadEmptyBodyIsError(t *testing.T) {
	for _, release := range releases() {
		t.Run(string(release), func(t *testing.T) {
			c, _ := startClient(t, release)
			ctx := context.Background()

			// A 200 with no body must be an error for a read: a read promises a resource, so a bodyless
			// 200 is a misbehaving server, not a return=minimal success that yields a nil resource.
			if _, err := c.Read(ctx, "Patient", "_empty"); err == nil {
				t.Fatal("Read of a bodyless 200 returned nil error, want a missing-body error")
			}

			// A normal read still returns the resource, so requiring a body does not break the happy path.
			created, err := c.Create(ctx, newPatient(release, "female"), "")
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			id := resourceID(t, created.Resource)
			read, err := c.Read(ctx, "Patient", id)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if read.Resource == nil {
				t.Fatal("Read returned a nil resource on a normal 200")
			}
			if got := read.Resource.ResourceType(); got != "Patient" {
				t.Errorf("Read: resourceType = %q, want Patient", got)
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

// TestFollowNextResolvesLinkForms proves the paging loop resolves a Bundle "next" link by its URL
// type against the URL of the request that produced the page (RFC 3986), with a base URL that carries
// a path prefix ("/fhir"): an absolute link is used verbatim, an origin-relative (leading-slash) link
// resolves against the scheme and host only — it must not double the "/fhir" prefix — a relative link
// joins to the search path, and a query-only link ("?page=2") keeps the search path and replaces only
// the query (it must not page into the service root). The server is mounted under "/fhir" so the
// path-doubling bug (http://host/fhir/fhir/Patient) and the query-only root bug (http://host/fhir/?page=2)
// are observable: the test asserts the exact path the second request hits.
func TestFollowNextResolvesLinkForms(t *testing.T) {
	const basePath = "/fhir"
	cases := []struct {
		name     string
		nextLink func(host string) string // built from the server host so an absolute link is well-formed
		wantPath string                   // the path the FollowNext request must hit
	}{
		{
			name:     "absolute",
			nextLink: func(host string) string { return "http://" + host + "/fhir/Patient?page=2" },
			wantPath: "/fhir/Patient",
		},
		{
			name:     "origin-relative leading slash",
			nextLink: func(string) string { return "/fhir/Patient?page=2" },
			wantPath: "/fhir/Patient",
		},
		{
			name:     "relative",
			nextLink: func(string) string { return "Patient?page=2" },
			wantPath: "/fhir/Patient",
		},
		{
			name:     "query only",
			nextLink: func(string) string { return "?page=2" },
			wantPath: "/fhir/Patient",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotSecondPath, gotSecondQuery string
			nextLink := tc.nextLink
			mux := http.NewServeMux()
			mux.HandleFunc(basePath+"/Patient", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", mediaTypeFHIRJSON)
				switch r.URL.Query().Get("page") {
				case "2":
					// The second page: record where the client landed and return a terminal page.
					gotSecondPath = r.URL.Path
					gotSecondQuery = r.URL.RawQuery
					_, _ = w.Write([]byte(
						`{"resourceType":"Bundle","type":"searchset","total":2,"link":[],"entry":[{"resource":{"resourceType":"Patient"},"search":{"mode":"match"}}]}`))
				default:
					// The first page: one match plus a "next" link in the form under test.
					body := fmt.Sprintf(
						`{"resourceType":"Bundle","type":"searchset","total":2,"link":[{"relation":"next","url":%q}],"entry":[{"resource":{"resourceType":"Patient"},"search":{"mode":"match"}}]}`,
						nextLink(r.Host))
					_, _ = w.Write([]byte(body))
				}
			})
			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)

			// The client base URL carries the "/fhir" path prefix, the non-root deployment shape that
			// triggers the path-doubling bug.
			c, err := rest.NewClient(fhir.R5, srv.URL+basePath)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			ctx := context.Background()

			page, err := c.Search(ctx, "Patient", rest.NewSearchParams())
			if err != nil {
				t.Fatalf("Search: %v", err)
			}
			if !page.HasNext() {
				t.Fatal("first page: expected a next link")
			}
			next, err := c.FollowNext(ctx, page)
			if err != nil {
				t.Fatalf("FollowNext (%s link): %v", tc.name, err)
			}
			if len(next.Resources) != 1 {
				t.Errorf("second page: got %d resources, want 1", len(next.Resources))
			}
			if gotSecondPath != tc.wantPath {
				t.Errorf("FollowNext hit path %q, want %q (path must not double the base prefix)", gotSecondPath, tc.wantPath)
			}
			if gotSecondQuery != "page=2" {
				t.Errorf("FollowNext hit query %q, want page=2", gotSecondQuery)
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
			// The error string names the structural FHIRPath locator (expression), which is safe to log.
			if !strings.Contains(ooErr.Error(), "Patient.gender") {
				t.Errorf("422: error message %q does not name the structural locator", ooErr.Error())
			}
			// The error string must NOT carry the server's free-text diagnostics: it can echo a
			// submitted patient value, so leaking it into the error would leak PHI into logs.
			const diagnosticsSentinel = "SENTINEL-DETliff-PHI"
			if strings.Contains(ooErr.Error(), diagnosticsSentinel) {
				t.Errorf("422: error message %q leaks the server diagnostics (PHI); it must stay off the string", ooErr.Error())
			}
			// The diagnostics is still reachable on the structured Outcome for a caller that explicitly
			// chooses to inspect it.
			if len(ooErr.Outcome.Issue) == 0 || !strings.Contains(ooErr.Outcome.Issue[0].Diagnostics, diagnosticsSentinel) {
				t.Errorf("422: expected the diagnostics to remain on the structured Outcome field")
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

// transactionBundleNoResources builds a transaction Bundle whose only entry is a GET (no resource
// payload). A cross-release Bundle of this shape has an empty resource view, so it exercises the
// release check on the Bundle resource itself rather than on entry resources.
func transactionBundleNoResources(t *testing.T, release fhir.Release) fhir.Resource {
	t.Helper()
	switch release {
	case fhir.R4:
		b, err := r4.NewTransaction(r4.TransactionEntry{Method: r4.HTTPVerbGET, URL: "Patient/1"})
		if err != nil {
			t.Fatalf("r4 NewTransaction: %v", err)
		}
		return b
	default:
		b, err := r5.NewTransaction(r5.TransactionEntry{Method: r5.HTTPVerbGET, URL: "Patient/1"})
		if err != nil {
			t.Fatalf("r5 NewTransaction: %v", err)
		}
		return b
	}
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

// TestWriteBodylessSuccessStatuses confirms create/update/patch accept the full set of FHIR write
// success statuses — 200, 201, and a return=minimal 204 No Content — rather than reporting a
// completed server-side write as a client error (which would break retry/concurrency logic).
func TestWriteBodylessSuccessStatuses(t *testing.T) {
	patient := newPatient(fhir.R5, "female")
	jsonPatch := []byte(`[{"op":"add","path":"/active","value":true}]`)
	cases := []struct {
		name   string
		status int
	}{
		{"200 OK", http.StatusOK},
		{"201 Created", http.StatusCreated},
		{"204 No Content", http.StatusNoContent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/Patient/1", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("ETag", `W/"2"`)
				w.Header().Set("Location", "/Patient/1/_history/2")
				w.WriteHeader(tc.status)
				if tc.status != http.StatusNoContent {
					_, _ = w.Write([]byte(`{"resourceType":"Patient","id":"1"}`))
				}
			})
			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)
			c, err := rest.NewClient(fhir.R5, srv.URL)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			ctx := context.Background()
			if _, err := c.Update(ctx, "1", patient, ""); err != nil {
				t.Errorf("Update with %s reported a failure: %v", tc.name, err)
			}
			if _, err := c.Patch(ctx, "Patient", "1", jsonPatch, ""); err != nil {
				t.Errorf("Patch with %s reported a failure: %v", tc.name, err)
			}
		})
	}
}

// countingTransport records how many requests reached the transport, so a test can assert a write
// was rejected client-side without ever issuing an HTTP request.
type countingTransport struct {
	mu    sync.Mutex
	count int
}

func (t *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.count++
	t.mu.Unlock()
	return nil, fmt.Errorf("countingTransport: request should not have been issued (%s %s)", req.Method, req.URL.Path)
}

func (t *countingTransport) requests() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.count
}

// otherRelease returns the v1 release that is not r.
func otherRelease(r fhir.Release) fhir.Release {
	if r == fhir.R4 {
		return fhir.R5
	}
	return fhir.R4
}

// TestWriteRejectsCrossReleaseResource confirms a release-fixed client rejects a resource of the
// other release on create, update, and a transaction entry, returning ErrReleaseMismatch without
// ever issuing an HTTP request — a cross-release mix-up is a client-side error, not a wrong-shape
// payload on the wire. A matching-release resource still reaches the transport.
func TestWriteRejectsCrossReleaseResource(t *testing.T) {
	for _, release := range releases() {
		t.Run(string(release), func(t *testing.T) {
			ctx := context.Background()
			wrong := newPatient(otherRelease(release), "female")

			tr := &countingTransport{}
			c, err := rest.NewClient(release, "https://example.org/fhir", rest.WithRoundTripper(tr))
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}

			if _, err := c.Create(ctx, wrong, ""); !errors.Is(err, rest.ErrReleaseMismatch) {
				t.Errorf("Create cross-release: got %v, want ErrReleaseMismatch", err)
			}
			if _, err := c.Update(ctx, "1", wrong, ""); !errors.Is(err, rest.ErrReleaseMismatch) {
				t.Errorf("Update cross-release: got %v, want ErrReleaseMismatch", err)
			}
			crossBundle := transactionBundle(t, otherRelease(release))
			if _, err := c.Transaction(ctx, crossBundle); !errors.Is(err, rest.ErrReleaseMismatch) {
				t.Errorf("Transaction cross-release: got %v, want ErrReleaseMismatch", err)
			}
			// A cross-release Bundle with no resource entries (GET-only) must also be rejected via the
			// Bundle's own release, not slip through an empty entry-resource view.
			crossEmpty := transactionBundleNoResources(t, otherRelease(release))
			if _, err := c.Transaction(ctx, crossEmpty); !errors.Is(err, rest.ErrReleaseMismatch) {
				t.Errorf("Transaction cross-release (no resources): got %v, want ErrReleaseMismatch", err)
			}
			if n := tr.requests(); n != 0 {
				t.Errorf("cross-release writes issued %d HTTP requests, want 0", n)
			}
		})
	}
}

// TestWriteAcceptsMatchingReleaseResource confirms the release check does not break the normal path:
// a resource of the client's own release still creates successfully against a matching-release
// server.
func TestWriteAcceptsMatchingReleaseResource(t *testing.T) {
	for _, release := range releases() {
		t.Run(string(release), func(t *testing.T) {
			c, _ := startClient(t, release)
			if _, err := c.Create(context.Background(), newPatient(release, "female"), ""); err != nil {
				t.Errorf("Create matching-release: %v", err)
			}
		})
	}
}
