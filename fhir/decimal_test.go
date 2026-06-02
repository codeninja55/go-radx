package fhir_test

import (
	"testing"

	"github.com/codeninja55/go-radx/fhir"
)

func TestDecimalMarshalJSONLexicalFidelity(t *testing.T) {
	d, err := fhir.ParseDecimal("1.500")
	if err != nil {
		t.Fatalf("ParseDecimal(%q): %v", "1.500", err)
	}
	b, err := d.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "1.500" {
		t.Errorf("MarshalJSON() = %q, want unquoted 1.500 (FHIR decimal lexical fidelity)", b)
	}
	if d.String() != "1.500" {
		t.Errorf("String() = %q, want preserved 1.500", d.String())
	}
}

func TestDecimalIsDICOMTwin(t *testing.T) {
	// fhir.Decimal is the FHIR-side twin of dicom.Decimal: trailing zeros and
	// the source lexical form survive a round-trip.
	d, err := fhir.ParseDecimal("3.14159265358979")
	if err != nil {
		t.Fatal(err)
	}
	if f, ok := d.Float64(); !ok || f == 0 {
		t.Errorf("Float64() = (%v,%v), want a finite non-zero value", f, ok)
	}
}
