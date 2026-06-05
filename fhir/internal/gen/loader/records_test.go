package loader

import (
	"encoding/json"
	"testing"
)

const patientStubJSON = `{
  "resourceType": "StructureDefinition",
  "id": "Patient",
  "url": "http://hl7.org/fhir/StructureDefinition/Patient",
  "name": "Patient",
  "kind": "resource",
  "abstract": false,
  "type": "Patient",
  "baseDefinition": "http://hl7.org/fhir/StructureDefinition/DomainResource",
  "snapshot": {
    "element": [
      {
        "path": "Patient.id",
        "min": 0,
        "max": "1",
        "isSummary": true,
        "type": [{"code": "id"}]
      },
      {
        "path": "Patient.gender",
        "min": 0,
        "max": "1",
        "isSummary": true,
        "type": [{"code": "code"}],
        "binding": {
          "strength": "required",
          "valueSet": "http://hl7.org/fhir/ValueSet/administrative-gender|5.0.0"
        }
      },
      {
        "path": "Patient.deceased[x]",
        "min": 0,
        "max": "1",
        "type": [{"code": "boolean"}, {"code": "dateTime"}]
      },
      {
        "path": "Patient.link.other",
        "min": 1,
        "max": "1",
        "contentReference": "#Patient.contact"
      }
    ]
  }
}`

const valueSetStubJSON = `{
  "resourceType": "ValueSet",
  "id": "administrative-gender",
  "url": "http://hl7.org/fhir/ValueSet/administrative-gender",
  "name": "AdministrativeGender",
  "compose": {
    "include": [
      {
        "system": "http://hl7.org/fhir/administrative-gender",
        "concept": [
          {"code": "male", "display": "Male"},
          {"code": "female", "display": "Female"},
          {"code": "other", "display": "Other"},
          {"code": "unknown", "display": "Unknown"}
        ]
      }
    ]
  }
}`

// intensionalValueSetStubJSON is a filter-defined (intensional) ValueSet: its
// membership is the set of LOINC codes whose "parent" property equals "LP43571-6",
// not a literal list of inlined concepts. It mirrors the shape of the R5
// example-intensional value set so the loader's Filter capture is pinned against a
// realistic intensional include.
const intensionalValueSetStubJSON = `{
  "resourceType": "ValueSet",
  "id": "example-intensional",
  "url": "http://hl7.org/fhir/ValueSet/example-intensional",
  "name": "LOINCCodesForCholesterolInSerumPlasma",
  "compose": {
    "include": [
      {
        "system": "http://loinc.org",
        "filter": [
          {"property": "parent", "op": "=", "value": "LP43571-6"}
        ]
      }
    ],
    "exclude": [
      {
        "system": "http://loinc.org",
        "concept": [
          {"code": "5932-9", "display": "Cholesterol [Presence]"}
        ]
      }
    ]
  }
}`

// TestValueSetFilterDecode pins the loader's capture of a filter-based (intensional)
// compose.include. A filter-defined include carries no inlined concepts; its codes
// are resolved by a terminology server applying the property/op/value rule. The
// generator does not enumerate such a set, so the enum stage relies on this captured
// Filter to recognise the value set as non-enumerable and emit a documented
// not-inlined boundary rather than a silently-empty const set. A regression that
// dropped Filter on decode would make a filter-defined required binding silently
// produce an empty enum, which is the failure this test exists to prevent.
func TestValueSetFilterDecode(t *testing.T) {
	t.Parallel()

	var vs ValueSet
	if err := json.Unmarshal([]byte(intensionalValueSetStubJSON), &vs); err != nil {
		t.Fatalf("unmarshal intensional ValueSet: %v", err)
	}

	if vs.Compose == nil {
		t.Fatal("Compose is nil")
	}
	if len(vs.Compose.Include) != 1 {
		t.Fatalf("Compose.Include count = %d, want 1", len(vs.Compose.Include))
	}
	inc := vs.Compose.Include[0]
	if inc.System != "http://loinc.org" {
		t.Errorf("include system = %q, want http://loinc.org", inc.System)
	}
	if len(inc.Concept) != 0 {
		t.Errorf("include concept count = %d, want 0 (filter-defined include inlines no concepts)", len(inc.Concept))
	}
	if len(inc.Filter) != 1 {
		t.Fatalf("include filter count = %d, want 1", len(inc.Filter))
	}
	f := inc.Filter[0]
	if f.Property != "parent" || f.Op != "=" || f.Value != "LP43571-6" {
		t.Errorf("include filter = %+v, want {Property:parent Op:= Value:LP43571-6}", f)
	}

	// The exclude rule inlines a concept; its Filter is empty, so the two shapes are
	// distinguishable on the same value set.
	if len(vs.Compose.Exclude) != 1 {
		t.Fatalf("Compose.Exclude count = %d, want 1", len(vs.Compose.Exclude))
	}
	exc := vs.Compose.Exclude[0]
	if len(exc.Filter) != 0 {
		t.Errorf("exclude filter count = %d, want 0", len(exc.Filter))
	}
	if len(exc.Concept) != 1 || exc.Concept[0].Code != "5932-9" {
		t.Errorf("exclude concept = %+v, want one concept code 5932-9", exc.Concept)
	}
}

func TestStructureDefinitionDecode(t *testing.T) {
	t.Parallel()

	var sd StructureDefinition
	if err := json.Unmarshal([]byte(patientStubJSON), &sd); err != nil {
		t.Fatalf("unmarshal StructureDefinition: %v", err)
	}

	if sd.ResourceType != "StructureDefinition" {
		t.Errorf("ResourceType = %q, want StructureDefinition", sd.ResourceType)
	}
	if sd.Name != "Patient" {
		t.Errorf("Name = %q, want Patient", sd.Name)
	}
	if sd.Kind != "resource" {
		t.Errorf("Kind = %q, want resource", sd.Kind)
	}
	if sd.Abstract {
		t.Error("Abstract = true, want false")
	}
	if sd.Type != "Patient" {
		t.Errorf("Type = %q, want Patient", sd.Type)
	}
	if sd.Snapshot == nil {
		t.Fatal("Snapshot is nil")
	}
	if got, want := len(sd.Snapshot.Element), 4; got != want {
		t.Fatalf("Snapshot.Element count = %d, want %d", got, want)
	}

	gotPaths := []string{
		sd.Snapshot.Element[0].Path,
		sd.Snapshot.Element[1].Path,
		sd.Snapshot.Element[2].Path,
		sd.Snapshot.Element[3].Path,
	}
	wantPaths := []string{"Patient.id", "Patient.gender", "Patient.deceased[x]", "Patient.link.other"}
	for i := range wantPaths {
		if gotPaths[i] != wantPaths[i] {
			t.Errorf("element %d path = %q, want %q", i, gotPaths[i], wantPaths[i])
		}
	}

	id := sd.Snapshot.Element[0]
	if id.Min != 0 || id.Max != "1" {
		t.Errorf("Patient.id cardinality = %d..%q, want 0..1", id.Min, id.Max)
	}
	if !id.IsSummary {
		t.Error("Patient.id IsSummary = false, want true (decoded from isSummary)")
	}
	if len(id.Type) != 1 || id.Type[0].Code != "id" {
		t.Errorf("Patient.id type = %+v, want one type with code id", id.Type)
	}

	gender := sd.Snapshot.Element[1]
	if gender.Binding == nil {
		t.Fatal("Patient.gender Binding is nil")
	}
	if gender.Binding.Strength != "required" {
		t.Errorf("Patient.gender binding strength = %q, want required", gender.Binding.Strength)
	}
	wantVS := "http://hl7.org/fhir/ValueSet/administrative-gender|5.0.0"
	if gender.Binding.ValueSet != wantVS {
		t.Errorf("Patient.gender binding valueSet = %q, want %q", gender.Binding.ValueSet, wantVS)
	}

	deceased := sd.Snapshot.Element[2]
	if len(deceased.Type) != 2 {
		t.Fatalf("Patient.deceased[x] type count = %d, want 2", len(deceased.Type))
	}
	if deceased.Type[0].Code != "boolean" || deceased.Type[1].Code != "dateTime" {
		t.Errorf("Patient.deceased[x] types = %v, want [boolean dateTime]",
			[]string{deceased.Type[0].Code, deceased.Type[1].Code})
	}

	link := sd.Snapshot.Element[3]
	if link.ContentReference != "#Patient.contact" {
		t.Errorf("Patient.link.other ContentReference = %q, want #Patient.contact", link.ContentReference)
	}
}

func TestValueSetDecode(t *testing.T) {
	t.Parallel()

	var vs ValueSet
	if err := json.Unmarshal([]byte(valueSetStubJSON), &vs); err != nil {
		t.Fatalf("unmarshal ValueSet: %v", err)
	}

	if vs.ResourceType != "ValueSet" {
		t.Errorf("ResourceType = %q, want ValueSet", vs.ResourceType)
	}
	if vs.URL != "http://hl7.org/fhir/ValueSet/administrative-gender" {
		t.Errorf("URL = %q", vs.URL)
	}
	if vs.Name != "AdministrativeGender" {
		t.Errorf("Name = %q, want AdministrativeGender", vs.Name)
	}
	if vs.Compose == nil {
		t.Fatal("Compose is nil")
	}
	if len(vs.Compose.Include) != 1 {
		t.Fatalf("Compose.Include count = %d, want 1", len(vs.Compose.Include))
	}
	inc := vs.Compose.Include[0]
	if inc.System != "http://hl7.org/fhir/administrative-gender" {
		t.Errorf("include system = %q", inc.System)
	}
	gotCodes := make([]string, 0, len(inc.Concept))
	for _, c := range inc.Concept {
		gotCodes = append(gotCodes, c.Code)
	}
	wantCodes := []string{"male", "female", "other", "unknown"}
	if len(gotCodes) != len(wantCodes) {
		t.Fatalf("concept codes = %v, want %v", gotCodes, wantCodes)
	}
	for i := range wantCodes {
		if gotCodes[i] != wantCodes[i] {
			t.Errorf("concept %d = %q, want %q", i, gotCodes[i], wantCodes[i])
		}
	}
}
