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
	if got, want := len(sd.Snapshot.Element), 3; got != want {
		t.Fatalf("Snapshot.Element count = %d, want %d", got, want)
	}

	gotPaths := []string{
		sd.Snapshot.Element[0].Path,
		sd.Snapshot.Element[1].Path,
		sd.Snapshot.Element[2].Path,
	}
	wantPaths := []string{"Patient.id", "Patient.gender", "Patient.deceased[x]"}
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
