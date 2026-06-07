package r5_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// TestParseRequiredBindingRejectsUnknown is the FHIR-013 regression against a real
// generated enum: a known code parses to the typed value, and an out-of-set code
// returns fhir.ErrUnknownCode wrapped with the binding name — never silently coerced and
// never echoing the offending token, which on a required-binding field could be a patient
// value (PRD §9.1).
func TestParseRequiredBindingRejectsUnknown(t *testing.T) {
	g, err := r5.ParseAdministrativeGender("female")
	if err != nil {
		t.Fatalf("ParseAdministrativeGender(\"female\"): %v", err)
	}
	if g != r5.AdministrativeGenderFemale {
		t.Errorf("parsed gender = %q, want female", g)
	}

	// A distinctive token stands in for a value a hostile upstream could stuff into a code
	// field; the error must name the binding but must not echo the token.
	const offending = "ZZZ-PHI-SENTINEL-banana"
	_, err = r5.ParseAdministrativeGender(offending)
	if !errors.Is(err, fhir.ErrUnknownCode) {
		t.Fatalf("ParseAdministrativeGender(%q): err = %v, want ErrUnknownCode", offending, err)
	}
	if !strings.Contains(err.Error(), "AdministrativeGender") {
		t.Errorf("error %q should name the binding", err.Error())
	}
	if strings.Contains(err.Error(), offending) {
		t.Errorf("error %q must not echo the offending token (potential PHI)", err.Error())
	}
}

// TestStrictDecodeRejectsUnknownCode is the FHIR-013 boundary regression: strict decode
// (the default) of a payload carrying an out-of-set required code fails with
// ErrUnknownCode rather than silently populating the field.
func TestStrictDecodeRejectsUnknownCode(t *testing.T) {
	var p r5.Patient
	err := json.Unmarshal([]byte(`{"resourceType":"Patient","gender":"banana"}`), &p)
	if !errors.Is(err, fhir.ErrUnknownCode) {
		t.Fatalf("strict decode of an unknown gender: err = %v, want ErrUnknownCode", err)
	}

	// A known code decodes to the typed enum value.
	var ok r5.Patient
	if err := json.Unmarshal([]byte(`{"resourceType":"Patient","gender":"female"}`), &ok); err != nil {
		t.Fatalf("strict decode of a known gender: %v", err)
	}
	if ok.Gender == nil || *ok.Gender != r5.AdministrativeGenderFemale {
		t.Errorf("decoded gender = %v, want female", ok.Gender)
	}
}

// TestEnumRoundTrip pins that a typed enum field round-trips its wire code unchanged.
func TestEnumRoundTrip(t *testing.T) {
	g := r5.AdministrativeGenderOther
	p := &r5.Patient{Gender: &g}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal Patient: %v", err)
	}
	if !strings.Contains(string(data), `"gender":"other"`) {
		t.Errorf("marshalled Patient = %s, want gender other", data)
	}
	decoded, err := fhir.Unmarshal[*r5.Patient](data)
	if err != nil {
		t.Fatalf("round-trip Unmarshal[*r5.Patient]: %v", err)
	}
	if decoded.Gender == nil || *decoded.Gender != r5.AdministrativeGenderOther {
		t.Errorf("round-tripped gender = %v, want other", decoded.Gender)
	}
}

// TestLenientDecodeRetainsUnknownCode pins the opt-in lenient policy at the boundary
// helper: with fhir.DecodeLenient an out-of-set code is retained verbatim (ok=false,
// nil error) for Validate to surface, rather than rejected. ParseXxx and the generated
// UnmarshalJSON stay strict regardless; lenient is the explicit, threaded alternative.
func TestLenientDecodeRetainsUnknownCode(t *testing.T) {
	value, ok, err := fhir.DecodeCode([]byte(`"banana"`), func(s string) bool { return s == "female" }, "AdministrativeGender", fhir.DecodeLenient)
	if err != nil {
		t.Fatalf("lenient DecodeCode: unexpected error %v", err)
	}
	if ok {
		t.Error("lenient DecodeCode reported an out-of-set code as valid")
	}
	if value != "banana" {
		t.Errorf("lenient DecodeCode retained %q, want banana", value)
	}
}

// TestNotInlinedBindingStaysString pins the terminology-scope boundary in the generated
// code: a required binding whose value set is not enumerable from the bundle (an
// external terminology) is a plain code string, so its field accepts any code and decode
// never rejects. UCUMCodes draws from the un-vendored UCUM system.
func TestNotInlinedBindingStaysString(t *testing.T) {
	// The not-inlined type is a string alias, so a string value assigns directly; if it
	// had been (wrongly) emitted as a closed enum this would not compile.
	var u r5.UCUMCodes = "mg/dL"
	if u != "mg/dL" {
		t.Errorf("UCUMCodes value = %q, want mg/dL", u)
	}
}
