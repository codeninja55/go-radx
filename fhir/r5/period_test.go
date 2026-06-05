package r5_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// TestGeneratedPeriodRoundTrips exercises the first generated datatype end to end:
// the emitted r5.Period must marshal to the expected JSON and decode back unchanged,
// proving the generator produces usable Go, not just compiling Go.
func TestGeneratedPeriodRoundTrips(t *testing.T) {
	p := r5.Period{Start: strptr("2024-01-01T00:00:00Z"), End: strptr("2024-12-31T23:59:59Z")}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"start":"2024-01-01T00:00:00Z","end":"2024-12-31T23:59:59Z"}`
	if string(b) != want {
		t.Errorf("Period JSON = %s, want %s", b, want)
	}

	var back r5.Period
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Start == nil || *back.Start != *p.Start || back.End == nil || *back.End != *p.End {
		t.Errorf("round-tripped Period = %+v, want %+v", back, p)
	}
}

// TestGeneratedPeriodOmitsEmpty asserts an absent optional scalar is omitted from the
// wire form, the omitempty contract the planner records for optional fields.
func TestGeneratedPeriodOmitsEmpty(t *testing.T) {
	b, err := json.Marshal(r5.Period{Start: strptr("2024-01-01")})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"start":"2024-01-01"}` {
		t.Errorf("Period with only Start = %s, want {\"start\":\"2024-01-01\"}", b)
	}
}

// TestGeneratedPeriodSiblingPreservesCanonicalOrder asserts the "_field" sibling fold
// keeps the value fields in their canonical (struct-declared) order and appends the
// sibling after them, rather than re-sorting every key as a map round-trip would.
func TestGeneratedPeriodSiblingPreservesCanonicalOrder(t *testing.T) {
	p := r5.Period{
		Start:        strptr("2024-01-01"),
		End:          strptr("2024-12-31"),
		StartElement: &fhir.PrimitiveElement{ID: strptr("s1")},
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"start":"2024-01-01","end":"2024-12-31","_start":{"id":"s1"}}`
	if string(b) != want {
		t.Errorf("Period with a start sibling = %s, want %s", b, want)
	}
}

// TestGeneratedPeriodOmitsEmptyScalarSibling asserts a non-nil but empty scalar
// sibling (no id, no extension) is not emitted, so the wire form carries no noise
// "_field":{} key — Go's omitempty cannot drop a non-nil pointer, so the sibling fold
// applies the emptiness rule explicitly.
func TestGeneratedPeriodOmitsEmptyScalarSibling(t *testing.T) {
	b, err := json.Marshal(r5.Period{Start: strptr("2024"), EndElement: &fhir.PrimitiveElement{}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "_end") {
		t.Errorf("Period = %s, want no _end key for an empty scalar sibling", b)
	}
}

// TestGeneratedPeriodScalarSiblingRoundTrip asserts a scalar primitive's id and
// extension survive a marshal/unmarshal cycle through the generated custom JSON
// methods.
func TestGeneratedPeriodScalarSiblingRoundTrip(t *testing.T) {
	original := r5.Period{
		Start: strptr("2024-01-01"),
		StartElement: &fhir.PrimitiveElement{
			ID:        strptr("s1"),
			Extension: json.RawMessage(`[{"url":"http://example.org/ext","valueString":"v"}]`),
		},
	}
	b, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var back r5.Period
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Start == nil || *back.Start != "2024-01-01" {
		t.Errorf("round-tripped start = %v, want 2024-01-01", back.Start)
	}
	if back.StartElement == nil || back.StartElement.ID == nil || *back.StartElement.ID != "s1" {
		t.Errorf("round-tripped _start = %+v, want id s1", back.StartElement)
	}
	if len(back.StartElement.Extension) == 0 {
		t.Error("round-tripped _start extension is empty; the extension was lost")
	}
}
