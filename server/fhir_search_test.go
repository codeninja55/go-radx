package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r4"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// searchBundle is the release-neutral view of a searchset Bundle the search-depth tests read: the
// total, the link set (relation -> url), and each entry's resource type/id and search.mode. It is
// decoded from the wire JSON so the tests assert on the served bytes, not on a release concrete type.
type searchBundle struct {
	ResourceType string `json:"resourceType"`
	Type         string `json:"type"`
	Total        int    `json:"total"`
	Link         []struct {
		Relation string `json:"relation"`
		URL      string `json:"url"`
	} `json:"link"`
	Entry []struct {
		Resource struct {
			ResourceType string `json:"resourceType"`
			ID           string `json:"id"`
		} `json:"resource"`
		Search struct {
			Mode string `json:"mode"`
		} `json:"search"`
	} `json:"entry"`
}

func (b searchBundle) linkURL(relation string) (string, bool) {
	for _, l := range b.Link {
		if l.Relation == relation {
			return l.URL, true
		}
	}
	return "", false
}

// matchIDs returns the ids of the entries with search.mode "match".
func (b searchBundle) matchIDs() []string {
	var out []string
	for _, e := range b.Entry {
		if e.Search.Mode == "match" {
			out = append(out, e.Resource.ID)
		}
	}
	return out
}

// includeKeys returns the "Type/id" keys of the entries with search.mode "include".
func (b searchBundle) includeKeys() []string {
	var out []string
	for _, e := range b.Entry {
		if e.Search.Mode == "include" {
			out = append(out, e.Resource.ResourceType+"/"+e.Resource.ID)
		}
	}
	return out
}

// getSearchBundle issues a search GET against an absolute URL and decodes the searchset Bundle,
// failing the test on a non-200 or a non-searchset body.
func getSearchBundle(t *testing.T, rawURL string) searchBundle {
	t.Helper()
	status, body, _ := httpDo(t, http.MethodGet, rawURL, "", nil)
	if status != http.StatusOK {
		t.Fatalf("search %s status = %d, want 200; body=%s", rawURL, status, body)
	}
	var b searchBundle
	if err := json.Unmarshal(body, &b); err != nil {
		t.Fatalf("decode searchset: %v; body=%s", err, body)
	}
	if b.ResourceType != "Bundle" || b.Type != "searchset" {
		t.Fatalf("search body is %s/%s, want Bundle/searchset; body=%s", b.ResourceType, b.Type, body)
	}
	return b
}

// createResource POSTs a resource of the release and returns its server-assigned id.
func createResource(t *testing.T, base, resourceType string, resource fhir.Resource) string {
	t.Helper()
	payload, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("marshal %s: %v", resourceType, err)
	}
	status, body, _ := httpDo(t, http.MethodPost, base+"/"+resourceType, "application/fhir+json", payload)
	if status != http.StatusCreated {
		t.Fatalf("create %s status = %d, want 201; body=%s", resourceType, status, body)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode created %s: %v", resourceType, err)
	}
	if created.ID == "" {
		t.Fatalf("created %s has no id: %s", resourceType, body)
	}
	return created.ID
}

func strp(s string) *string { return &s }

// patientWithName builds a valid Patient of the release carrying a family name.
func patientWithName(release fhir.Release, family string) fhir.Resource {
	switch release {
	case fhir.R4:
		return &r4.Patient{Name: []r4.HumanName{{Family: strp(family)}}}
	default:
		return &r5.Patient{Name: []r5.HumanName{{Family: strp(family)}}}
	}
}

// observationForSubject builds a valid Observation of the release referencing the given patient id.
func observationForSubject(release fhir.Release, patientID string) fhir.Resource {
	ref := "Patient/" + patientID
	switch release {
	case fhir.R4:
		st := r4.ObservationStatus("final")
		return &r4.Observation{Status: &st, Code: &r4.CodeableConcept{Text: strp("vital")}, Subject: &r4.Reference{Reference: strp(ref)}}
	default:
		st := r5.ObservationStatus("final")
		return &r5.Observation{Status: &st, Code: &r5.CodeableConcept{Text: strp("vital")}, Subject: &r5.Reference{Reference: strp(ref)}}
	}
}

// imagingStudyForSubject builds a valid ImagingStudy of the release referencing the given patient id
// (status and subject are the two required elements, ImagingStudy validation descriptor).
func imagingStudyForSubject(release fhir.Release, patientID string) fhir.Resource {
	ref := "Patient/" + patientID
	switch release {
	case fhir.R4:
		st := r4.ImagingStudyStatus("available")
		return &r4.ImagingStudy{Status: &st, Subject: &r4.Reference{Reference: strp(ref)}}
	default:
		st := r5.ImagingStudyStatus("available")
		return &r5.ImagingStudy{Status: &st, Subject: &r5.Reference{Reference: strp(ref)}}
	}
}

// observationDerivedFrom builds a valid Observation of the release whose derivedFrom references both an
// Observation and an ImagingStudy — the multi-target reference an _include target-type modifier must
// narrow.
func observationDerivedFrom(release fhir.Release, obsID, studyID string) fhir.Resource {
	obsRef := "Observation/" + obsID
	studyRef := "ImagingStudy/" + studyID
	switch release {
	case fhir.R4:
		st := r4.ObservationStatus("final")
		return &r4.Observation{
			Status: &st, Code: &r4.CodeableConcept{Text: strp("derived")},
			DerivedFrom: []r4.Reference{{Reference: strp(obsRef)}, {Reference: strp(studyRef)}},
		}
	default:
		st := r5.ObservationStatus("final")
		return &r5.Observation{
			Status: &st, Code: &r5.CodeableConcept{Text: strp("derived")},
			DerivedFrom: []r5.Reference{{Reference: strp(obsRef)}, {Reference: strp(studyRef)}},
		}
	}
}

// TestFHIRRoleSearchPaging proves the searchset carries Bundle.link self/next/prev paging honouring
// _count, and that following the next link round-trips to the next page until the last page (which
// carries no next link), per FHIR R5 http.html#paging. total reports the full match count on every
// page, not the page size.
func TestFHIRRoleSearchPaging(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			const n = 5
			created := map[string]bool{}
			for i := 0; i < n; i++ {
				id := createResource(t, base, "Patient", patientWithName(release, "Page"))
				created[id] = true
			}

			// Page 1 of 2 (_count=2): total is the full count, the page has 2 matches, a next link is
			// present, and no prev link (first page).
			page1 := getSearchBundle(t, base+"/Patient?_count=2")
			if page1.Total != n {
				t.Errorf("page1 total = %d, want %d", page1.Total, n)
			}
			if got := len(page1.matchIDs()); got != 2 {
				t.Errorf("page1 match entries = %d, want 2", got)
			}
			if _, ok := page1.linkURL("self"); !ok {
				t.Error("page1 missing self link")
			}
			if _, ok := page1.linkURL("prev"); ok {
				t.Error("page1 has a prev link on the first page")
			}
			next1, ok := page1.linkURL("next")
			if !ok {
				t.Fatal("page1 missing next link")
			}

			// Follow next: page 2 has 2 more matches, a next and a prev link.
			page2 := getSearchBundle(t, next1)
			if got := len(page2.matchIDs()); got != 2 {
				t.Errorf("page2 match entries = %d, want 2", got)
			}
			if _, ok := page2.linkURL("prev"); !ok {
				t.Error("page2 missing prev link")
			}
			next2, ok := page2.linkURL("next")
			if !ok {
				t.Fatal("page2 missing next link")
			}

			// Follow next: page 3 (the last) has the final 1 match and NO next link.
			page3 := getSearchBundle(t, next2)
			if got := len(page3.matchIDs()); got != 1 {
				t.Errorf("page3 match entries = %d, want 1", got)
			}
			if url, ok := page3.linkURL("next"); ok {
				t.Errorf("page3 (last page) has a next link %q, want none", url)
			}

			// Every match across the three pages is distinct and accounts for all created resources.
			seen := map[string]bool{}
			for _, ids := range [][]string{page1.matchIDs(), page2.matchIDs(), page3.matchIDs()} {
				for _, id := range ids {
					if seen[id] {
						t.Errorf("id %s appeared on more than one page", id)
					}
					seen[id] = true
				}
			}
			if len(seen) != n {
				t.Errorf("paged through %d distinct ids, want %d", len(seen), n)
			}
			for id := range created {
				if !seen[id] {
					t.Errorf("created id %s never appeared in a page", id)
				}
			}
		})
	}
}

// TestFHIRRoleSearchCountClampedAndZero proves _count is clamped to the server max and that _count=0
// returns an empty page with the honest total and NO next link (the FHIR count-only request,
// search.html#count): a next link whose _offset did not advance would loop the client forever.
func TestFHIRRoleSearchCountClampedAndZero(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			for i := 0; i < 3; i++ {
				createResource(t, base, "Patient", patientWithName(release, "Zero"))
			}

			// _count=0: no match entries, total is the full count, and NO next link (the cursor cannot
			// advance off a zero-size page, so a next link would loop the client on the same empty page).
			zero := getSearchBundle(t, base+"/Patient?_count=0")
			if got := len(zero.matchIDs()); got != 0 {
				t.Errorf("_count=0 match entries = %d, want 0", got)
			}
			if zero.Total != 3 {
				t.Errorf("_count=0 total = %d, want 3", zero.Total)
			}
			if url, ok := zero.linkURL("next"); ok {
				t.Errorf("_count=0 has a next link %q, want none (count-only request, non-advancing cursor)", url)
			}

			// _count above the cap is clamped: the self link names the clamped count, not the requested
			// one, so a client cannot ask for an unbounded page.
			big := getSearchBundle(t, base+"/Patient?_count=100000")
			self, ok := big.linkURL("self")
			if !ok {
				t.Fatal("clamped search missing self link")
			}
			u, err := url.Parse(self)
			if err != nil {
				t.Fatalf("parse self link: %v", err)
			}
			if got := u.Query().Get("_count"); got != "200" {
				t.Errorf("clamped self _count = %q, want 200 (maxSearchCount)", got)
			}
		})
	}
}

// TestFHIRRoleSearchInclude proves _include pulls the referenced resource into the searchset as an
// include-mode entry (FHIR R5 search.html#include): an Observation search with
// _include=Observation:subject returns the matched Observations (mode match) plus their referenced
// Patients (mode include), with the Patient never double-counted in total.
func TestFHIRRoleSearchInclude(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			patientID := createResource(t, base, "Patient", patientWithName(release, "Subject"))
			obsID := createResource(t, base, "Observation", observationForSubject(release, patientID))

			b := getSearchBundle(t, base+"/Observation?_include=Observation:subject")
			if b.Total != 1 {
				t.Errorf("include total = %d, want 1 (the match count, includes excluded)", b.Total)
			}
			if matches := b.matchIDs(); len(matches) != 1 || matches[0] != obsID {
				t.Errorf("include matches = %v, want [%s]", matches, obsID)
			}
			includes := b.includeKeys()
			wantInclude := "Patient/" + patientID
			if len(includes) != 1 || includes[0] != wantInclude {
				t.Errorf("include entries = %v, want [%s]", includes, wantInclude)
			}
		})
	}
}

// TestFHIRRoleSearchRevInclude proves _revinclude pulls the resources that reference a match into the
// searchset as include-mode entries (FHIR R5 search.html#revinclude): a Patient search with
// _revinclude=Observation:subject returns the matched Patient (mode match) plus the Observations that
// reference it (mode include).
func TestFHIRRoleSearchRevInclude(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			patientID := createResource(t, base, "Patient", patientWithName(release, "RevSubject"))
			obsID := createResource(t, base, "Observation", observationForSubject(release, patientID))
			// A second patient with no observations to prove the revinclude is scoped to the match.
			createResource(t, base, "Patient", patientWithName(release, "Lonely"))

			b := getSearchBundle(t, base+"/Patient?_id="+patientID+"&_revinclude=Observation:subject")
			if matches := b.matchIDs(); len(matches) != 1 || matches[0] != patientID {
				t.Errorf("revinclude matches = %v, want [%s]", matches, patientID)
			}
			includes := b.includeKeys()
			wantInclude := "Observation/" + obsID
			if len(includes) != 1 || includes[0] != wantInclude {
				t.Errorf("revinclude entries = %v, want [%s]", includes, wantInclude)
			}
		})
	}
}

// TestFHIRRoleSearchChained proves a one-hop chained parameter resolves against the repository (FHIR
// R5 search.html#chaining): Observation?subject:Patient.name=Chained returns only the Observations
// whose subject Patient has that name, and the typeless form (subject.name) resolves the same way.
func TestFHIRRoleSearchChained(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			// A patient named "Chained" with an observation, and a patient named "Other" with one.
			chainedID := createResource(t, base, "Patient", patientWithName(release, "Chained"))
			wantObs := createResource(t, base, "Observation", observationForSubject(release, chainedID))
			otherID := createResource(t, base, "Patient", patientWithName(release, "Other"))
			createResource(t, base, "Observation", observationForSubject(release, otherID))

			// Typed chain: subject:Patient.name.
			typed := getSearchBundle(t, base+"/Observation?subject:Patient.name=Chained")
			if matches := typed.matchIDs(); len(matches) != 1 || matches[0] != wantObs {
				t.Errorf("typed chain matches = %v, want [%s]", matches, wantObs)
			}
			if typed.Total != 1 {
				t.Errorf("typed chain total = %d, want 1", typed.Total)
			}

			// Typeless chain: subject.name resolves to the same observation.
			typeless := getSearchBundle(t, base+"/Observation?subject.name=Chained")
			if matches := typeless.matchIDs(); len(matches) != 1 || matches[0] != wantObs {
				t.Errorf("typeless chain matches = %v, want [%s]", matches, wantObs)
			}

			// A chain that matches no target yields an empty searchset, not all observations.
			none := getSearchBundle(t, base+"/Observation?subject:Patient.name=Nobody")
			if matches := none.matchIDs(); len(matches) != 0 {
				t.Errorf("no-match chain matches = %v, want []", matches)
			}
			if none.Total != 0 {
				t.Errorf("no-match chain total = %d, want 0", none.Total)
			}
		})
	}
}

// TestFHIRRoleSearchDirectReference proves the dev repository matches a direct reference search
// parameter (Observation?subject=Patient/id), the base match the chained-parameter resolution and a
// production Repository build on.
func TestFHIRRoleSearchDirectReference(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			p1 := createResource(t, base, "Patient", patientWithName(release, "Ref1"))
			p2 := createResource(t, base, "Patient", patientWithName(release, "Ref2"))
			wantObs := createResource(t, base, "Observation", observationForSubject(release, p1))
			createResource(t, base, "Observation", observationForSubject(release, p2))

			b := getSearchBundle(t, base+"/Observation?subject="+url.QueryEscape("Patient/"+p1))
			if matches := b.matchIDs(); len(matches) != 1 || matches[0] != wantObs {
				t.Errorf("direct reference matches = %v, want [%s]", matches, wantObs)
			}
		})
	}
}

// TestFHIRRoleSearchIterateNotApplied proves an _include with the :iterate modifier is not applied
// (recursive include is out of scope): the spec parses but the iterate hop is ignored, so the search
// still succeeds and returns the plain match without erroring.
func TestFHIRRoleSearchIterateNotApplied(t *testing.T) {
	base, cleanup := startFHIRDaemon(t, fhir.R5)
	defer cleanup()

	patientID := createResource(t, base, "Patient", patientWithName(fhir.R5, "Iter"))
	obsID := createResource(t, base, "Observation", observationForSubject(fhir.R5, patientID))

	b := getSearchBundle(t, base+"/Observation?_include=Observation:subject:iterate")
	if matches := b.matchIDs(); len(matches) != 1 || matches[0] != obsID {
		t.Errorf("iterate-include matches = %v, want [%s]", matches, obsID)
	}
	// The :iterate spec is rejected by the parser, so no include entry is produced.
	if includes := b.includeKeys(); len(includes) != 0 {
		t.Errorf("iterate-include produced include entries %v, want none (iterate out of scope)", includes)
	}
}

// TestSplitChainedParams unit-tests the chained-parameter parser: a dotted name splits into a chain
// (with an optional :Type modifier), the result/control parameters are dropped from the plain set,
// and a plain parameter passes through.
func TestSplitChainedParams(t *testing.T) {
	q := url.Values{}
	q.Set("subject:Patient.name", "Smith")
	q.Set("subject.name", "Jones")
	q.Set("status", "final")
	q.Set("_count", "10")
	q.Set("_include", "Observation:subject")

	plain, chains := splitChainedParams(q)
	if plain.Get("status") != "final" {
		t.Errorf("plain status = %q, want final", plain.Get("status"))
	}
	if plain.Has("_count") || plain.Has("_include") || plain.Has("subject:Patient.name") {
		t.Errorf("plain params leaked a control or chained parameter: %v", plain)
	}
	if len(chains) != 2 {
		t.Fatalf("parsed %d chains, want 2", len(chains))
	}
	var typed, typeless bool
	for _, c := range chains {
		switch {
		case c.refParam == "subject" && c.targetType == "Patient" && c.targetParam == "name" && c.value == "Smith":
			typed = true
		case c.refParam == "subject" && c.targetType == "" && c.targetParam == "name" && c.value == "Jones":
			typeless = true
		}
	}
	if !typed || !typeless {
		t.Errorf("chains = %+v, want a typed and a typeless subject.name chain", chains)
	}
}

// TestSplitReference unit-tests the reference splitter: a relative, an absolute, and a versioned
// reference all reduce to the same Type/id.
func TestSplitReference(t *testing.T) {
	cases := map[string][2]string{
		"Patient/123":                           {"Patient", "123"},
		"http://example.org/fhir/Patient/123":   {"Patient", "123"},
		"Patient/123/_history/4":                {"Patient", "123"},
		"https://h/fhir/Patient/123/_history/2": {"Patient", "123"},
		"notareference":                         {"", ""},
	}
	for ref, want := range cases {
		rt, id := splitReference(ref)
		if rt != want[0] || id != want[1] {
			t.Errorf("splitReference(%q) = (%q,%q), want (%q,%q)", ref, rt, id, want[0], want[1])
		}
	}
}

// TestSearchLinkURLRoundTrips proves the next link the pager emits is a self-consistent absolute URL:
// following it (parsing back the query) yields the offset the page advanced to. The check guards the
// cursor contract — a client follows the link, never builds the offset.
func TestSearchLinkURLRoundTrips(t *testing.T) {
	base, cleanup := startFHIRDaemon(t, fhir.R5)
	defer cleanup()

	for i := 0; i < 4; i++ {
		createResource(t, base, "Patient", patientWithName(fhir.R5, "Link"))
	}
	page1 := getSearchBundle(t, base+"/Patient?_count=2")
	next, ok := page1.linkURL("next")
	if !ok {
		t.Fatal("page1 missing next link")
	}
	u, err := url.Parse(next)
	if err != nil {
		t.Fatalf("parse next link: %v", err)
	}
	if got := u.Query().Get("_offset"); got != "2" {
		t.Errorf("next link _offset = %q, want 2", got)
	}
	if !strings.Contains(u.Path, "/Patient") {
		t.Errorf("next link path = %q, want it to name the Patient type", u.Path)
	}
}

// TestFHIRRoleSearchSortPreservesRepoOrder proves the search-depth layer preserves the Repository's
// returned order when the request carries _sort (FHIR R5 search.html#sort: the server, here the
// Repository, owns the sort), and only imposes its own id sort when no _sort is given. The stub
// Repository returns its matches in a fixed descending-id order; with _sort the served pages keep that
// order, while without _sort the layer's stable id sort reorders them ascending.
func TestFHIRRoleSearchSortPreservesRepoOrder(t *testing.T) {
	repo := &sortStubRepo{release: fhir.R5, ids: []string{"3", "2", "1"}}
	base, cleanup := startFHIRDaemonRepo(t, fhir.R5, repo)
	defer cleanup()

	// With _sort the Repository's order is authoritative and must survive paging end to end.
	sorted := getSearchBundle(t, base+"/Patient?_sort=name")
	if got := sorted.matchIDs(); len(got) != 3 || got[0] != "3" || got[1] != "2" || got[2] != "1" {
		t.Errorf("_sort search match order = %v, want [3 2 1] (repo order preserved)", got)
	}

	// The repo order is preserved across pages too: page 1 of a _count=1 _sort search is the repo's
	// first element, not the id-smallest.
	page1 := getSearchBundle(t, base+"/Patient?_sort=name&_count=1")
	if got := page1.matchIDs(); len(got) != 1 || got[0] != "3" {
		t.Errorf("_sort page1 = %v, want [3] (first repo element, not id-smallest)", got)
	}
	next, ok := page1.linkURL("next")
	if !ok {
		t.Fatal("_sort page1 missing next link")
	}
	page2 := getSearchBundle(t, next)
	if got := page2.matchIDs(); len(got) != 1 || got[0] != "2" {
		t.Errorf("_sort page2 = %v, want [2] (second repo element)", got)
	}

	// Without _sort the layer imposes its stable id sort so paging is deterministic over the map-backed
	// store: the same repo order is reordered ascending by id.
	unsorted := getSearchBundle(t, base+"/Patient")
	if got := unsorted.matchIDs(); len(got) != 3 || got[0] != "1" || got[1] != "2" || got[2] != "3" {
		t.Errorf("no-_sort search match order = %v, want [1 2 3] (layer id sort)", got)
	}
}

// TestFHIRRoleSearchUnsupportedChainIgnored proves an unsupported or unparseable dotted parameter is
// ignored (the documented out-of-scope stance, FHIR R5 search.html#chaining) rather than applied as a
// filter that excludes every match: an unknown chain head and a multi-hop chain both return the
// unfiltered base matches, while a SUPPORTED chain that resolves to no target still returns empty.
func TestFHIRRoleSearchUnsupportedChainIgnored(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			patientID := createResource(t, base, "Patient", patientWithName(release, "ChainBase"))
			obsA := createResource(t, base, "Observation", observationForSubject(release, patientID))
			obsB := createResource(t, base, "Observation", observationForSubject(release, patientID))
			all := map[string]bool{obsA: true, obsB: true}

			assertAll := func(label, rawURL string) {
				b := getSearchBundle(t, rawURL)
				ids := b.matchIDs()
				if len(ids) != len(all) {
					t.Errorf("%s matches = %v, want all %d observations (chain ignored)", label, ids, len(all))
				}
				for _, id := range ids {
					if !all[id] {
						t.Errorf("%s returned unexpected id %s", label, id)
					}
				}
			}

			// Unknown chain head (foo is not a reference parameter on Observation): ignored, not a filter.
			assertAll("unknown-head", base+"/Observation?foo.bar=x")

			// Multi-hop chain (subject.organization.name): the head subject is a real reference but the
			// target parameter "organization.name" is not a supported single hop, so the whole chain is
			// out of scope and ignored rather than excluding every match.
			assertAll("multi-hop", base+"/Observation?subject.organization.name=Acme")

			// A SUPPORTED chain that resolves to no target still returns empty — the distinction the fix
			// preserves: this is a real constraint that matched nothing, not an ignored parameter.
			none := getSearchBundle(t, base+"/Observation?subject:Patient.name=NoSuchName")
			if ids := none.matchIDs(); len(ids) != 0 {
				t.Errorf("supported no-match chain matches = %v, want [] (real empty constraint)", ids)
			}
		})
	}
}

// TestFHIRRoleSearchIncludeTargetType proves an _include target-type modifier narrows a multi-target
// reference to the named type (FHIR R5 search.html#include): an Observation whose derivedFrom points at
// both another Observation and an ImagingStudy, searched with _include=Observation:derived-from:
// ImagingStudy, includes only the ImagingStudy target — not the Observation target.
func TestFHIRRoleSearchIncludeTargetType(t *testing.T) {
	for _, release := range fhirReleases() {
		t.Run(string(release), func(t *testing.T) {
			base, cleanup := startFHIRDaemon(t, release)
			defer cleanup()

			patientID := createResource(t, base, "Patient", patientWithName(release, "DerivedSubject"))
			sourceObs := createResource(t, base, "Observation", observationForSubject(release, patientID))
			studyID := createResource(t, base, "ImagingStudy", imagingStudyForSubject(release, patientID))
			derivedID := createResource(t, base, "Observation", observationDerivedFrom(release, sourceObs, studyID))

			// Target-type modifier ImagingStudy: only the ImagingStudy target is included.
			b := getSearchBundle(t, base+"/Observation?_id="+derivedID+"&_include="+
				url.QueryEscape("Observation:derived-from:ImagingStudy"))
			if matches := b.matchIDs(); len(matches) != 1 || matches[0] != derivedID {
				t.Errorf("target-type include matches = %v, want [%s]", matches, derivedID)
			}
			includes := b.includeKeys()
			wantInclude := "ImagingStudy/" + studyID
			if len(includes) != 1 || includes[0] != wantInclude {
				t.Errorf("target-type include entries = %v, want [%s] only (Observation target excluded)",
					includes, wantInclude)
			}

			// Without the modifier, both targets are included — proving the modifier is what narrows.
			bAll := getSearchBundle(t, base+"/Observation?_id="+derivedID+"&_include="+
				url.QueryEscape("Observation:derived-from"))
			gotAll := map[string]bool{}
			for _, k := range bAll.includeKeys() {
				gotAll[k] = true
			}
			if !gotAll["ImagingStudy/"+studyID] || !gotAll["Observation/"+sourceObs] || len(gotAll) != 2 {
				t.Errorf("unmodified include entries = %v, want both ImagingStudy/%s and Observation/%s",
					bAll.includeKeys(), studyID, sourceObs)
			}
		})
	}
}

// sortStubRepo is a Repository whose Search returns a fixed list of minimal Patients in the order its
// ids slice gives, so a test can prove the search-depth layer preserves the Repository's order under
// _sort (and only imposes its own id sort when _sort is absent). Only Search and Read are exercised;
// the write and version methods are unused.
type sortStubRepo struct {
	release fhir.Release
	ids     []string
}

func (s *sortStubRepo) patient(id string) fhir.Resource {
	pid := id
	return &r5.Patient{DomainResource: r5.DomainResource{ID: &pid}, Name: []r5.HumanName{{Family: strp("Sort")}}}
}

func (s *sortStubRepo) Search(_ context.Context, resourceType string, _ url.Values) (fhir.Resource, error) {
	if resourceType != "Patient" {
		return r5.NewSearchSet(0)
	}
	mode := r5.SearchEntryModeMatch
	entries := make([]r5.SearchEntry, 0, len(s.ids))
	for _, id := range s.ids {
		entries = append(entries, r5.SearchEntry{Resource: s.patient(id), Mode: &mode})
	}
	return r5.NewSearchSet(int32(len(entries)), entries...) // #nosec G115 -- test fixture, small count
}

func (s *sortStubRepo) Read(_ context.Context, resourceType, id string) (fhir.Resource, error) {
	if resourceType != "Patient" {
		return nil, fmt.Errorf("%w: %s/%s", ErrNotFound, resourceType, id)
	}
	for _, have := range s.ids {
		if have == id {
			return s.patient(id), nil
		}
	}
	return nil, fmt.Errorf("%w: %s/%s", ErrNotFound, resourceType, id)
}

func (s *sortStubRepo) VRead(context.Context, string, string, string) (fhir.Resource, error) {
	return nil, fmt.Errorf("sortStubRepo: vread is not supported")
}

func (s *sortStubRepo) History(context.Context, string, string) ([]ResourceVersion, error) {
	return nil, fmt.Errorf("sortStubRepo: history is not supported")
}

func (s *sortStubRepo) Create(context.Context, fhir.Resource) (fhir.Resource, error) {
	return nil, fmt.Errorf("sortStubRepo: create is not supported")
}

func (s *sortStubRepo) Update(context.Context, string, string, fhir.Resource, string) (fhir.Resource, bool, error) {
	return nil, false, fmt.Errorf("sortStubRepo: update is not supported")
}

func (s *sortStubRepo) Delete(context.Context, string, string, string) (bool, error) {
	return false, fmt.Errorf("sortStubRepo: delete is not supported")
}

func (s *sortStubRepo) Transaction(context.Context, fhir.Resource) (fhir.Resource, error) {
	return nil, fmt.Errorf("sortStubRepo: transactions are not supported")
}
