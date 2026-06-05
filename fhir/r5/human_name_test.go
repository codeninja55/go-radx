package r5_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// TestHumanNameScalarPrimitiveExtensionRoundTrip exercises the scalar "_field"
// sibling against the generated HumanName: a family name carrying an id and
// extension serialises both the value and its "_family" companion, and a decode
// reconstructs the pair.
func TestHumanNameScalarPrimitiveExtensionRoundTrip(t *testing.T) {
	original := &r5.HumanName{
		Family: strptr("Doe"),
		FamilyElement: &fhir.PrimitiveElement{
			ID:        strptr("f1"),
			Extension: json.RawMessage(`[{"url":"http://example.org/qualifier","valueCode":"BR"}]`),
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal HumanName: %v", err)
	}
	if got := string(data); !strings.Contains(got, `"family":"Doe"`) || !strings.Contains(got, `"_family":{`) {
		t.Fatalf("marshalled HumanName = %s, want both family and _family keys", got)
	}

	var decoded r5.HumanName
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal HumanName: %v", err)
	}
	if decoded.Family == nil || *decoded.Family != "Doe" {
		t.Errorf("decoded family = %v, want Doe", decoded.Family)
	}
	if decoded.FamilyElement == nil || decoded.FamilyElement.ID == nil || *decoded.FamilyElement.ID != "f1" {
		t.Errorf("decoded _family = %+v, want id f1", decoded.FamilyElement)
	}
}

// TestHumanNameRepeatingPrimitiveNullAlignment is the central acceptance test: a
// repeating primitive (given) with a partially-extended array round-trips
// "given":["Jane","Q"] / "_given":[null,{"id":"x"}] with the null placeholder
// preserved on BOTH marshal and unmarshal, so the value array and the "_field"
// array stay index-aligned (Codex FHIR-005 alignment rule).
func TestHumanNameRepeatingPrimitiveNullAlignment(t *testing.T) {
	original := &r5.HumanName{
		Given: []string{"Jane", "Q"},
		GivenElement: []*fhir.PrimitiveElement{
			nil,
			{ID: strptr("x")},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal HumanName: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, `"given":["Jane","Q"]`) {
		t.Errorf("marshalled value array = %s, want \"given\":[\"Jane\",\"Q\"]", got)
	}
	if !strings.Contains(got, `"_given":[null,{"id":"x"}]`) {
		t.Errorf("marshalled sibling array = %s, want \"_given\":[null,{\"id\":\"x\"}]", got)
	}

	var decoded r5.HumanName
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal HumanName: %v", err)
	}
	if !reflect.DeepEqual(decoded.Given, []string{"Jane", "Q"}) {
		t.Errorf("decoded given = %v, want [Jane Q]", decoded.Given)
	}
	if len(decoded.GivenElement) != 2 {
		t.Fatalf("decoded _given length = %d, want 2", len(decoded.GivenElement))
	}
	if decoded.GivenElement[0] != nil {
		t.Errorf("decoded _given[0] = %v, want nil (the null placeholder)", decoded.GivenElement[0])
	}
	if decoded.GivenElement[1] == nil || decoded.GivenElement[1].ID == nil || *decoded.GivenElement[1].ID != "x" {
		t.Errorf("decoded _given[1] = %+v, want id x", decoded.GivenElement[1])
	}
}

// TestHumanNameRepeatingPrimitiveNoExtensionsOmitsSibling asserts a repeating
// primitive with values but no extensions carries no "_given" key, so a plain
// repeating primitive serialises as a bare value array.
func TestHumanNameRepeatingPrimitiveNoExtensionsOmitsSibling(t *testing.T) {
	data, err := json.Marshal(&r5.HumanName{Given: []string{"Jane", "Q"}})
	if err != nil {
		t.Fatalf("marshal HumanName: %v", err)
	}
	if strings.Contains(string(data), `"_given"`) {
		t.Errorf("marshalled HumanName = %s, want no _given key for an un-extended repeating primitive", data)
	}
}

// TestHumanNameNoSiblingOnComplexField is the FHIR-005 regression against generated
// code: the complex Period field gets no "_field" companion. The generated struct
// must not declare a PeriodElement, and a marshalled HumanName must not carry a
// "_period" key.
func TestHumanNameNoSiblingOnComplexField(t *testing.T) {
	if _, ok := reflect.TypeOf(r5.HumanName{}).FieldByName("PeriodElement"); ok {
		t.Error("HumanName declares a PeriodElement; a complex field must have no \"_field\" sibling (FHIR-005)")
	}

	data, err := json.Marshal(&r5.HumanName{Period: &r5.Period{Start: strptr("2024-01-01")}})
	if err != nil {
		t.Fatalf("marshal HumanName: %v", err)
	}
	if strings.Contains(string(data), `"_period"`) {
		t.Errorf("marshalled HumanName = %s, want no _period key (FHIR-005)", data)
	}
}

// TestHumanNameMultipleRepeatingPrimitivesAlign asserts every repeating primitive
// (given, prefix, suffix) null-aligns independently, so partial extensions on one
// do not perturb another.
func TestHumanNameMultipleRepeatingPrimitivesAlign(t *testing.T) {
	original := &r5.HumanName{
		Given:         []string{"A", "B", "C"},
		GivenElement:  []*fhir.PrimitiveElement{nil, {ID: strptr("g")}},
		Prefix:        []string{"Dr"},
		PrefixElement: []*fhir.PrimitiveElement{{ID: strptr("p")}},
		Suffix:        []string{"Jr"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal HumanName: %v", err)
	}

	var decoded r5.HumanName
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal HumanName: %v", err)
	}
	if len(decoded.GivenElement) != 3 || decoded.GivenElement[1] == nil || *decoded.GivenElement[1].ID != "g" {
		t.Errorf("given sibling misaligned: %+v", decoded.GivenElement)
	}
	if len(decoded.PrefixElement) != 1 || decoded.PrefixElement[0] == nil || *decoded.PrefixElement[0].ID != "p" {
		t.Errorf("prefix sibling misaligned: %+v", decoded.PrefixElement)
	}
	if decoded.SuffixElement != nil {
		t.Errorf("suffix sibling = %+v, want nil (no extensions)", decoded.SuffixElement)
	}
}
