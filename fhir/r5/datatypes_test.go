package r5_test

import (
	"encoding/json"
	"testing"

	"github.com/codeninja55/go-radx/fhir/r5"
)

func strptr(s string) *string { return &s }

func TestReferenceMarshalOmitsEmpty(t *testing.T) {
	ref := r5.Reference{Type: strptr("Patient"), Display: strptr("Jane Doe")}
	b, err := json.Marshal(ref)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	want := `{"type":"Patient","display":"Jane Doe"}`
	if got != want {
		t.Errorf("Reference JSON = %s, want %s", got, want)
	}
}

func TestIdentifierMarshalsDICOMUID(t *testing.T) {
	// A DICOM UID maps to an Identifier under the urn:dicom:uid system (glossary).
	id := r5.Identifier{System: strptr("urn:dicom:uid"), Value: strptr("urn:oid:1.2.3")}
	b, err := json.Marshal(id)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"system":"urn:dicom:uid","value":"urn:oid:1.2.3"}`
	if string(b) != want {
		t.Errorf("Identifier JSON = %s, want %s", b, want)
	}
}

func TestCodeableConceptAndCoding(t *testing.T) {
	cc := r5.CodeableConcept{
		Coding: []r5.Coding{{System: strptr("http://loinc.org"), Code: strptr("24558-9"), Display: strptr("US Abdomen")}},
		Text:   strptr("US Abdomen"),
	}
	b, err := json.Marshal(cc)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"coding":[{"system":"http://loinc.org","code":"24558-9","display":"US Abdomen"}],"text":"US Abdomen"}`
	if string(b) != want {
		t.Errorf("CodeableConcept JSON = %s, want %s", b, want)
	}
}

func TestCodeableReference(t *testing.T) {
	// R5 CodeableReference has concept (CodeableConcept) and reference (Reference).
	cr := r5.CodeableReference{
		Concept: &r5.CodeableConcept{Text: strptr("CT Chest")},
	}
	b, err := json.Marshal(cr)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"concept":{"text":"CT Chest"}}`
	if string(b) != want {
		t.Errorf("CodeableReference JSON = %s, want %s", b, want)
	}
}
