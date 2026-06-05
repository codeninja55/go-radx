package r5_test

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// valueKeyPattern matches any value[x] suffixed wire key, used to assert a marshalled
// choice authors exactly one of them.
var valueKeyPattern = regexp.MustCompile(`"value[A-Z][A-Za-z0-9]*":`)

// TestChoiceSetterClearsSiblings is the FHIR-001 regression: setting one branch of a
// choice group clears every other branch, so a two-branches-set state is never reachable
// through the typed setters. Observation.value[x] holds Quantity, CodeableConcept,
// string, and boolean branches; setting the string branch after the Quantity branch must
// leave only the string populated.
func TestChoiceSetterClearsSiblings(t *testing.T) {
	t.Parallel()
	value, err := fhir.ParseDecimal("7.40")
	if err != nil {
		t.Fatalf("ParseDecimal: %v", err)
	}
	unit := "mmol/L"
	obs := &r5.Observation{}
	obs.SetValueQuantity(r5.Quantity{Value: &value, Unit: &unit})
	if obs.ValueQuantity == nil {
		t.Fatal("ValueQuantity not set after SetValueQuantity")
	}

	// Setting another branch must clear the first.
	obs.SetValueString(r5.FHIRString("see attached"))
	if obs.ValueQuantity != nil {
		t.Error("ValueQuantity still set after SetValueString; the setter must clear siblings (FHIR-001)")
	}
	if obs.ValueCodeableConcept != nil || obs.ValueBoolean != nil {
		t.Error("a sibling branch remained set after SetValueString")
	}
	if obs.ValueString == nil || *obs.ValueString != "see attached" {
		t.Errorf("ValueString = %v, want \"see attached\"", obs.ValueString)
	}
}

// TestChoiceValueGetterReturnsSetBranch asserts Value() returns the currently-set branch
// via the sealed interface, recoverable by a type switch, and (nil, false) when no branch
// is set.
func TestChoiceValueGetterReturnsSetBranch(t *testing.T) {
	t.Parallel()
	obs := &r5.Observation{}
	if _, ok := obs.Value(); ok {
		t.Error("Value() reported a value on an empty Observation")
	}

	obs.SetValueString(r5.FHIRString("normal"))
	v, ok := obs.Value()
	if !ok {
		t.Fatal("Value() reported no value after SetValueString")
	}
	s, ok := v.(r5.FHIRString)
	if !ok {
		t.Fatalf("Value() returned %T, want r5.FHIRString", v)
	}
	if string(s) != "normal" {
		t.Errorf("recovered value = %q, want \"normal\"", string(s))
	}
}

// TestChoicePrimitiveBranchSingleSuffixedKey is the FHIR-002 regression: marshalling an
// Observation whose value[x] is a string boxes the value through the FHIRString wrapper
// and authors exactly one suffixed key ("valueString":"normal") — never the prototype's
// unsuffixed "value" key, and never a second value* key.
func TestChoicePrimitiveBranchSingleSuffixedKey(t *testing.T) {
	t.Parallel()
	obs := &r5.Observation{}
	obs.SetValueString(r5.FHIRString("normal"))

	data, err := json.Marshal(obs)
	if err != nil {
		t.Fatalf("marshal Observation: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, `"valueString":"normal"`) {
		t.Errorf("marshalled Observation = %s, want a bare valueString key", got)
	}
	if keys := valueKeyPattern.FindAllString(got, -1); len(keys) != 1 {
		t.Errorf("marshalled Observation has value keys %v, want exactly one suffixed key", keys)
	}
}

// TestChoiceComplexBranchRoundTrip asserts a complex branch marshals under its suffixed
// key and round-trips back to the same branch with its value preserved.
func TestChoiceComplexBranchRoundTrip(t *testing.T) {
	t.Parallel()
	value, err := fhir.ParseDecimal("7.40")
	if err != nil {
		t.Fatalf("ParseDecimal: %v", err)
	}
	unit := "mmol/L"
	obs := &r5.Observation{}
	obs.SetValueQuantity(r5.Quantity{Value: &value, Unit: &unit})

	data, err := json.Marshal(obs)
	if err != nil {
		t.Fatalf("marshal Observation: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, `"valueQuantity":{`) {
		t.Errorf("marshalled Observation = %s, want a valueQuantity object", got)
	}
	if keys := valueKeyPattern.FindAllString(got, -1); len(keys) != 1 {
		t.Errorf("marshalled Observation has value keys %v, want exactly one suffixed key", keys)
	}

	var decoded r5.Observation
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal Observation: %v", err)
	}
	v, ok := decoded.Value()
	if !ok {
		t.Fatal("decoded Observation has no value")
	}
	q, ok := v.(r5.Quantity)
	if !ok {
		t.Fatalf("decoded value = %T, want r5.Quantity", v)
	}
	if q.Value == nil || q.Value.String() != "7.40" {
		t.Errorf("decoded Quantity.Value = %v, want lexical 7.40", q.Value)
	}
}

// TestChoiceDecimalBranchLexicalRoundTrip asserts a decimal-valued choice branch boxes
// through FHIRDecimal and preserves the lexical form (trailing zero) across a round trip,
// authoring an unquoted JSON number under the suffixed key.
func TestChoiceDecimalBranchLexicalRoundTrip(t *testing.T) {
	t.Parallel()
	d, err := fhir.ParseDecimal("1.20")
	if err != nil {
		t.Fatalf("ParseDecimal: %v", err)
	}
	// Exercise the FHIRDecimal wrapper directly: it is the branch type a decimal-valued
	// choice (for example a quantity/decimal "[x]" group) boxes through, and the lexical
	// round trip is the property the wrapper exists to preserve.
	wrapped := r5.FHIRDecimal(d)
	data, err := json.Marshal(wrapped)
	if err != nil {
		t.Fatalf("marshal FHIRDecimal: %v", err)
	}
	if string(data) != "1.20" {
		t.Errorf("marshalled FHIRDecimal = %s, want unquoted lexical 1.20", string(data))
	}
	var rt r5.FHIRDecimal
	if err := json.Unmarshal(data, &rt); err != nil {
		t.Fatalf("unmarshal FHIRDecimal: %v", err)
	}
	if fhir.Decimal(rt).String() != "1.20" {
		t.Errorf("round-tripped FHIRDecimal = %q, want 1.20", fhir.Decimal(rt).String())
	}
}

// TestChoiceInteger64BranchStringRoundTrip asserts the FHIRInteger64 wrapper marshals
// as a quoted JSON string (the FHIR R5 integer64 wire form, which preserves 64-bit
// precision past JSON parsers that decode numbers as float64) and round-trips a value
// beyond the float64 safe-integer range.
func TestChoiceInteger64BranchStringRoundTrip(t *testing.T) {
	t.Parallel()
	const big = int64(9007199254740993) // 2^53 + 1, not exactly representable as float64
	data, err := json.Marshal(r5.FHIRInteger64(big))
	if err != nil {
		t.Fatalf("marshal FHIRInteger64: %v", err)
	}
	if string(data) != `"9007199254740993"` {
		t.Errorf("marshalled FHIRInteger64 = %s, want a quoted decimal string", string(data))
	}
	var rt r5.FHIRInteger64
	if err := json.Unmarshal(data, &rt); err != nil {
		t.Fatalf("unmarshal FHIRInteger64: %v", err)
	}
	if int64(rt) != big {
		t.Errorf("round-tripped FHIRInteger64 = %d, want %d", int64(rt), big)
	}
}

// TestChoiceInterfaceIsSealed documents the sealed-interface guarantee: the value
// interface's marker is unexported, so only this package's branch types satisfy it and a
// built-in scalar cannot. This is a compile-time property; the test pins that the
// wrapper types — not the built-ins — implement the interface.
func TestChoiceInterfaceIsSealed(t *testing.T) {
	t.Parallel()
	var _ r5.ObservationValue = r5.FHIRString("x")
	var _ r5.ObservationValue = r5.FHIRBoolean(true)
	var _ r5.ObservationValue = r5.Quantity{}
	// The following would not compile, which is the sealing guarantee:
	//   var _ r5.ObservationValue = "x"   // built-in string lacks isObservationValue
	//   var _ r5.ObservationValue = true  // built-in bool lacks isObservationValue
}
