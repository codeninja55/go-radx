package fhir_test

import (
	"encoding/json"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
)

func strptr(s string) *string { return &s }

// TestPrimitiveElementRoundTrip asserts a PrimitiveElement carrying an id and a raw
// extension survives a marshal/unmarshal cycle with both intact.
func TestPrimitiveElementRoundTrip(t *testing.T) {
	original := &fhir.PrimitiveElement{
		ID:        strptr("x"),
		Extension: json.RawMessage(`[{"url":"http://example.org/ext","valueString":"v"}]`),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal PrimitiveElement: %v", err)
	}

	var decoded fhir.PrimitiveElement
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal PrimitiveElement: %v", err)
	}
	if decoded.ID == nil || *decoded.ID != "x" {
		t.Errorf("decoded id = %v, want x", decoded.ID)
	}
	if string(decoded.Extension) == "" {
		t.Error("decoded extension is empty; the raw extension was lost")
	}
}

// TestPrimitiveElementIsZero asserts an all-empty element (and a nil one) report as
// zero, the condition under which the "_field" sibling is omitted entirely.
func TestPrimitiveElementIsZero(t *testing.T) {
	var nilElem *fhir.PrimitiveElement
	if !nilElem.IsZero() {
		t.Error("nil PrimitiveElement should be zero")
	}
	if !(&fhir.PrimitiveElement{}).IsZero() {
		t.Error("empty PrimitiveElement should be zero")
	}
	if (&fhir.PrimitiveElement{ID: strptr("a")}).IsZero() {
		t.Error("PrimitiveElement with an id should not be zero")
	}
}

// TestMarshalPrimitiveExtensionsNullAlignment asserts the helper pads the "_field"
// array with JSON nulls so it lines up with the value array: a value array of length
// two whose only extended position is index one yields [null,{...}].
func TestMarshalPrimitiveExtensionsNullAlignment(t *testing.T) {
	elements := []*fhir.PrimitiveElement{nil, {ID: strptr("x")}}
	raw, err := fhir.MarshalPrimitiveExtensions(2, elements)
	if err != nil {
		t.Fatalf("MarshalPrimitiveExtensions: %v", err)
	}
	if string(raw) != `[null,{"id":"x"}]` {
		t.Errorf("aligned sibling = %s, want [null,{\"id\":\"x\"}]", raw)
	}
}

// TestMarshalPrimitiveExtensionsPadsToValueCount asserts the sibling array is padded
// to the value array's length even when the extended positions are all earlier, so a
// trailing un-extended value still has a null placeholder.
func TestMarshalPrimitiveExtensionsPadsToValueCount(t *testing.T) {
	elements := []*fhir.PrimitiveElement{{ID: strptr("x")}}
	raw, err := fhir.MarshalPrimitiveExtensions(3, elements)
	if err != nil {
		t.Fatalf("MarshalPrimitiveExtensions: %v", err)
	}
	if string(raw) != `[{"id":"x"},null,null]` {
		t.Errorf("padded sibling = %s, want [{\"id\":\"x\"},null,null]", raw)
	}
}

// TestMarshalPrimitiveExtensionsOmitsWhenEmpty asserts an all-empty element slice
// produces no "_field" array, so a repeating primitive with no extensions carries no
// companion key.
func TestMarshalPrimitiveExtensionsOmitsWhenEmpty(t *testing.T) {
	raw, err := fhir.MarshalPrimitiveExtensions(2, []*fhir.PrimitiveElement{nil, {}})
	if err != nil {
		t.Fatalf("MarshalPrimitiveExtensions: %v", err)
	}
	if raw != nil {
		t.Errorf("expected no sibling array for all-empty elements, got %s", raw)
	}
}

// TestUnmarshalPrimitiveExtensionsRestoresNulls asserts a JSON null in the "_field"
// array decodes to a nil entry, preserving the positional alignment with the value
// array.
func TestUnmarshalPrimitiveExtensionsRestoresNulls(t *testing.T) {
	elements, err := fhir.UnmarshalPrimitiveExtensions([]byte(`[null,{"id":"x"}]`))
	if err != nil {
		t.Fatalf("UnmarshalPrimitiveExtensions: %v", err)
	}
	if len(elements) != 2 {
		t.Fatalf("decoded %d elements, want 2", len(elements))
	}
	if elements[0] != nil {
		t.Errorf("element[0] = %v, want nil (the null placeholder)", elements[0])
	}
	if elements[1] == nil || elements[1].ID == nil || *elements[1].ID != "x" {
		t.Errorf("element[1] = %v, want id x", elements[1])
	}
}
