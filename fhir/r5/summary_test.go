package r5_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// objectKeys decodes a JSON object's top-level keys in their wire order, so a test can
// assert both which keys survive a summary filter and that their canonical order is
// preserved.
func objectKeys(t *testing.T, data []byte) []string {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(string(data)))
	if _, err := dec.Token(); err != nil {
		t.Fatalf("read object start: %v", err)
	}
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			t.Fatalf("read key: %v", err)
		}
		keys = append(keys, tok.(string))
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			t.Fatalf("read value: %v", err)
		}
	}
	return keys
}

func contains(keys []string, want string) bool {
	for _, k := range keys {
		if k == want {
			return true
		}
	}
	return false
}

// summaryPatient builds a Patient carrying both summary-flagged elements (name, gender,
// active) and non-summary elements (maritalStatus, photo), plus the narrative, so the
// modes have something to keep and something to drop.
func summaryPatient() *r5.Patient {
	gender := r5.AdministrativeGenderFemale
	narrativeStatus := r5.NarrativeStatusGenerated
	return &r5.Patient{
		DomainResource: r5.DomainResource{
			ID:   strPtr("pat-1"),
			Text: &r5.Narrative{Status: &narrativeStatus, Div: strPtr("<div>summary</div>")},
		},
		Active:        boolPtr(true),
		Name:          []r5.HumanName{{Family: strPtr("Synthetic")}},
		Gender:        &gender,
		MaritalStatus: &r5.CodeableConcept{Text: strPtr("unknown")},
	}
}

// TestMarshalSummaryPatientTrue confirms _summary=true over a real generated Patient keeps
// the isSummary-flagged elements, drops the non-summary ones, tags SUBSETTED, and preserves
// the canonical element order the resource's own MarshalJSON produces.
func TestMarshalSummaryPatientTrue(t *testing.T) {
	got, err := fhir.MarshalSummary(summaryPatient(), fhir.SummaryTrue)
	if err != nil {
		t.Fatalf("MarshalSummary: %v", err)
	}
	keys := objectKeys(t, got)

	for _, want := range []string{"resourceType", "id", "active", "name", "gender"} {
		if !contains(keys, want) {
			t.Errorf("_summary=true dropped %q it should keep; keys = %v", want, keys)
		}
	}
	for _, drop := range []string{"text", "maritalStatus"} {
		if contains(keys, drop) {
			t.Errorf("_summary=true kept %q it should drop; keys = %v", drop, keys)
		}
	}
	if !strings.Contains(string(got), "SUBSETTED") {
		t.Errorf("_summary=true did not set the SUBSETTED tag: %s", got)
	}

	// name precedes gender in the StructureDefinition snapshot order, so the filtered
	// output must keep that order.
	if i, j := indexOf(keys, "name"), indexOf(keys, "gender"); i < 0 || j < 0 || i > j {
		t.Errorf("_summary=true did not preserve canonical order; keys = %v", keys)
	}
}

// TestMarshalSummaryPatientText confirms _summary=text keeps the narrative and the
// mandatory elements only (Patient has no mandatory element beyond the base, so only the
// narrative plus infrastructure survive) and drops the data elements.
func TestMarshalSummaryPatientText(t *testing.T) {
	got, err := fhir.MarshalSummary(summaryPatient(), fhir.SummaryText)
	if err != nil {
		t.Fatalf("MarshalSummary: %v", err)
	}
	keys := objectKeys(t, got)
	if !contains(keys, "text") {
		t.Errorf("_summary=text dropped the narrative; keys = %v", keys)
	}
	for _, drop := range []string{"active", "name", "gender", "maritalStatus"} {
		if contains(keys, drop) {
			t.Errorf("_summary=text kept data element %q; keys = %v", drop, keys)
		}
	}
}

// TestMarshalSummaryPatientData confirms _summary=data keeps everything except the
// narrative.
func TestMarshalSummaryPatientData(t *testing.T) {
	got, err := fhir.MarshalSummary(summaryPatient(), fhir.SummaryData)
	if err != nil {
		t.Fatalf("MarshalSummary: %v", err)
	}
	keys := objectKeys(t, got)
	if contains(keys, "text") {
		t.Errorf("_summary=data kept the narrative; keys = %v", keys)
	}
	for _, want := range []string{"active", "name", "gender", "maritalStatus"} {
		if !contains(keys, want) {
			t.Errorf("_summary=data dropped %q it should keep; keys = %v", want, keys)
		}
	}
}

// TestMarshalSummaryBundleCount is the count-mode acceptance: a searchset Bundle summarised
// with _summary=count emits the total and drops the entries.
func TestMarshalSummaryBundleCount(t *testing.T) {
	bundle, err := r5.NewSearchSet(2,
		r5.SearchEntry{FullURL: "urn:uuid:1", Resource: summaryPatient()},
		r5.SearchEntry{FullURL: "urn:uuid:2", Resource: summaryPatient()},
	)
	if err != nil {
		t.Fatalf("NewSearchSet: %v", err)
	}
	got, err := fhir.MarshalSummary(bundle, fhir.SummaryCount)
	if err != nil {
		t.Fatalf("MarshalSummary: %v", err)
	}
	keys := objectKeys(t, got)
	if !contains(keys, "total") {
		t.Errorf("_summary=count dropped total; keys = %v", keys)
	}
	if contains(keys, "entry") {
		t.Errorf("_summary=count kept entry; keys = %v", keys)
	}
	// The mandatory Bundle.type must survive so the count view stays a valid Bundle: a
	// re-decoded count summary must not report a missing required element.
	if !contains(keys, "type") {
		t.Errorf("_summary=count dropped the mandatory type; keys = %v", keys)
	}
	round, err := fhir.Unmarshal[*r5.Bundle](got)
	if err != nil {
		t.Fatalf("Unmarshal count summary: %v", err)
	}
	if outcome := fhir.Validate(round); outcome.HasErrors() {
		t.Errorf("count summary failed validation: %s", outcome.Error())
	}
}

// TestMarshalSummaryFullIsResourceJSON confirms _summary=false returns the resource's own
// MarshalJSON byte-for-byte, with no filtering and no SUBSETTED tag.
func TestMarshalSummaryFullIsResourceJSON(t *testing.T) {
	patient := summaryPatient()
	full, err := json.Marshal(patient)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	got, err := fhir.MarshalSummary(patient, fhir.SummaryFull)
	if err != nil {
		t.Fatalf("MarshalSummary: %v", err)
	}
	if string(got) != string(full) {
		t.Fatalf("_summary=false = %s, want resource JSON %s", got, full)
	}
}

// TestCanonicalRoundTripByteStable confirms canonical FHIR JSON survives a decode/encode
// cycle byte-for-byte: the resource's UnmarshalJSON lifts primitive "_field" siblings and
// the residual keys back into the struct, and its MarshalJSON re-emits them in the same
// canonical element order, so a payload authored in go-radx's canonical order round-trips
// unchanged. go-radx's canonical form emits the value fields in StructureDefinition
// snapshot order and folds the scalar primitive "_field" siblings in after the value
// object's own keys (AppendSiblings appends rather than interleaves); a nested object
// folds its own siblings at the end of that object. This exercises a value/sibling pair
// (gender with a "_gender" extension), a choice branch (deceasedBoolean), and a repeating
// primitive's null-aligned sibling array (the "given" name parts, "_given" trailing the
// name object).
func TestCanonicalRoundTripByteStable(t *testing.T) {
	canonical := `{"resourceType":"Patient",` +
		`"id":"rt-1",` +
		`"active":true,` +
		`"name":[{"family":"Synthetic","given":["Ada","Q"],"_given":[null,{"id":"g2"}]}],` +
		`"gender":"female",` +
		`"deceasedBoolean":false,` +
		`"_gender":{"id":"gx"}}`

	patient, err := fhir.Unmarshal[*r5.Patient]([]byte(canonical))
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	got, err := json.Marshal(patient)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(got) != canonical {
		t.Fatalf("round-trip not byte-stable:\n got  %s\n want %s", got, canonical)
	}
}

// TestMarshalSummaryPreservesMetaOrder is the regression for the SUBSETTED tag splice: a
// resource that already carries an ordered meta (versionId, then an existing tag) keeps
// meta's canonical key order when the summary appends the SUBSETTED coding, so the
// summarised resource still round-trips byte-stably. A map round-trip on meta would
// re-sort its keys and break this.
func TestMarshalSummaryPreservesMetaOrder(t *testing.T) {
	patient := summaryPatient()
	existingTag := r5.Coding{Code: strPtr("existing")}
	patient.Meta = &r5.Meta{
		VersionId: strPtr("3"),
		Tag:       []r5.Coding{existingTag},
	}

	summary, err := fhir.MarshalSummary(patient, fhir.SummaryTrue)
	if err != nil {
		t.Fatalf("MarshalSummary: %v", err)
	}

	// The summary must decode and re-encode byte-for-byte: meta keeps its order and the
	// existing tag is preserved ahead of the appended SUBSETTED coding.
	round, err := fhir.Unmarshal[*r5.Patient](summary)
	if err != nil {
		t.Fatalf("Unmarshal summary: %v", err)
	}
	reencoded, err := json.Marshal(round)
	if err != nil {
		t.Fatalf("Marshal round-trip: %v", err)
	}
	if string(reencoded) != string(summary) {
		t.Fatalf("summary not byte-stable on round-trip:\n got  %s\n want %s", reencoded, summary)
	}
	if !strings.Contains(string(summary), `"versionId":"3"`) {
		t.Errorf("summary dropped meta.versionId: %s", summary)
	}
	if !strings.Contains(string(summary), `"code":"existing"`) || !strings.Contains(string(summary), "SUBSETTED") {
		t.Errorf("summary did not keep the existing tag alongside SUBSETTED: %s", summary)
	}
}

// TestMarshalSummaryMetaSiblingNoTag covers the byte-stable edge case where meta carries a
// primitive "_field" sibling (a versionId with an id extension) but no existing tag: the
// inserted SUBSETTED tag must land among meta's value fields, ahead of the trailing
// sibling, so a fresh re-marshal of the decoded Meta (which orders tag before the sibling)
// reproduces the summary byte-for-byte.
func TestMarshalSummaryMetaSiblingNoTag(t *testing.T) {
	patient := summaryPatient()
	patient.Meta = &r5.Meta{
		VersionId:        strPtr("7"),
		VersionIdElement: &fhir.PrimitiveElement{ID: strPtr("vid")},
	}

	summary, err := fhir.MarshalSummary(patient, fhir.SummaryTrue)
	if err != nil {
		t.Fatalf("MarshalSummary: %v", err)
	}
	round, err := fhir.Unmarshal[*r5.Patient](summary)
	if err != nil {
		t.Fatalf("Unmarshal summary: %v", err)
	}
	reencoded, err := json.Marshal(round)
	if err != nil {
		t.Fatalf("Marshal round-trip: %v", err)
	}
	if string(reencoded) != string(summary) {
		t.Fatalf("summary with a meta sibling not byte-stable:\n got  %s\n want %s", reencoded, summary)
	}
}

func indexOf(keys []string, want string) int {
	for i, k := range keys {
		if k == want {
			return i
		}
	}
	return -1
}
