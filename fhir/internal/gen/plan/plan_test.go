package plan

import (
	"testing"

	"github.com/codeninja55/go-radx/fhir/internal/gen/model"
)

func TestGoFieldName(t *testing.T) {
	t.Parallel()
	cases := []struct{ in, want string }{
		{"start", "Start"},
		{"implicitRules", "ImplicitRules"},
		{"value[x]", "Value"},
		{"id", "ID"},
		{"url", "URL"},
		{"userSelected", "UserSelected"},
	}
	for _, c := range cases {
		if got := GoFieldName(c.in); got != c.want {
			t.Errorf("GoFieldName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestGoTypeName(t *testing.T) {
	t.Parallel()
	cases := []struct{ in, want string }{
		{"CodeableConcept", "CodeableConcept"},
		{"Reference", "Reference"},
		{"Period", "Period"},
		{"boolean", "Boolean"},
	}
	for _, c := range cases {
		if got := GoTypeName(c.in); got != c.want {
			t.Errorf("GoTypeName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestGoBackboneTypeName(t *testing.T) {
	t.Parallel()
	got := GoBackboneTypeName("Observation", []string{"component", "referenceRange"})
	want := "ObservationComponentReferenceRange"
	if got != want {
		t.Errorf("GoBackboneTypeName = %q, want %q", got, want)
	}
}

// TestPlanFieldRequiredScalarIsPointer is the structural half of FHIR-007: a required
// scalar (min 1, max 1) must be a pointer so a present false or 0 is distinguishable
// from an absent field. A bare value would make the two indistinguishable.
func TestPlanFieldRequiredScalarIsPointer(t *testing.T) {
	t.Parallel()
	e := &model.Element{
		Name:        "active",
		Path:        "Patient.active",
		Cardinality: model.Cardinality{Min: 1, Max: "1"},
		Types:       []model.TypeRef{{Code: "boolean"}},
	}
	f := PlanField(e)
	if f.GoType != "*bool" {
		t.Errorf("required scalar boolean GoType = %q, want *bool (FHIR-007 structural half)", f.GoType)
	}
}

func TestPlanFieldOptionalScalarIsPointer(t *testing.T) {
	t.Parallel()
	e := &model.Element{
		Name:        "start",
		Path:        "Period.start",
		Cardinality: model.Cardinality{Min: 0, Max: "1"},
		Types:       []model.TypeRef{{Code: "dateTime"}},
	}
	f := PlanField(e)
	if f.GoType != "*string" {
		t.Errorf("optional dateTime GoType = %q, want *string", f.GoType)
	}
}

func TestPlanFieldRepeatingIsSlice(t *testing.T) {
	t.Parallel()
	e := &model.Element{
		Name:        "identifier",
		Path:        "Patient.identifier",
		Cardinality: model.Cardinality{Min: 1, Max: "*"},
		Types:       []model.TypeRef{{Code: "Identifier"}},
	}
	f := PlanField(e)
	if f.GoType != "[]Identifier" {
		t.Errorf("repeating Identifier GoType = %q, want []Identifier", f.GoType)
	}
}

func TestPlanFieldDecimalMapsToDecimal(t *testing.T) {
	t.Parallel()
	e := &model.Element{
		Name:        "value",
		Path:        "Quantity.value",
		Cardinality: model.Cardinality{Min: 0, Max: "1"},
		Types:       []model.TypeRef{{Code: "decimal"}},
	}
	f := PlanField(e)
	if f.GoType != "*fhir.Decimal" {
		t.Errorf("decimal GoType = %q, want *fhir.Decimal (FHIR-009)", f.GoType)
	}
}

// TestPrimitiveGoTypeTable pins the FHIR primitive → Go scalar mapping against
// fhir.md's table: boolean to bool, the integer family to sized integers, the
// string and date/time families to string, and decimal to fhir.Decimal so lexical
// precision survives a round trip rather than collapsing to float64 (Codex
// FHIR-009). The table is the planner's authority for primitive-ness, so a drift
// here changes every generated primitive field.
func TestPrimitiveGoTypeTable(t *testing.T) {
	t.Parallel()
	want := map[string]string{
		"boolean":      "bool",
		"integer":      "int32",
		"positiveInt":  "int32",
		"unsignedInt":  "int32",
		"integer64":    "int64",
		"decimal":      "fhir.Decimal",
		"string":       "string",
		"code":         "string",
		"id":           "string",
		"markdown":     "string",
		"uri":          "string",
		"url":          "string",
		"canonical":    "string",
		"oid":          "string",
		"uuid":         "string",
		"base64Binary": "string",
		"instant":      "string",
		"dateTime":     "string",
		"date":         "string",
		"time":         "string",
		"xhtml":        "string",
	}
	for code, goType := range want {
		if got := primitiveGoTypes[code]; got != goType {
			t.Errorf("primitiveGoTypes[%q] = %q, want %q", code, got, goType)
		}
	}
	if len(primitiveGoTypes) != len(want) {
		t.Errorf("primitiveGoTypes has %d entries, want %d; the table drifted from fhir.md", len(primitiveGoTypes), len(want))
	}
	if got := primitiveGoTypes["decimal"]; got == "float64" {
		t.Error("decimal maps to float64; it must map to fhir.Decimal (FHIR-009)")
	}
}

// TestNoPrimitiveSiblingOnComplexField is the FHIR-005 planner regression: a complex
// field (a CodeableConcept, a Reference, a backbone) is not a primitive and carries
// no "_field" sibling, so the planner emits no companion field for it.
func TestNoPrimitiveSiblingOnComplexField(t *testing.T) {
	t.Parallel()
	complexElem := &model.Element{
		Name:        "code",
		Path:        "Flag.code",
		Cardinality: model.Cardinality{Min: 0, Max: "1"},
		Types:       []model.TypeRef{{Code: "CodeableConcept"}},
	}
	if PlanField(complexElem).Primitive {
		t.Error("a CodeableConcept field is marked primitive; it must not carry a \"_field\" sibling (FHIR-005)")
	}

	backbone := &model.Element{
		Name:        "issue",
		Path:        "OperationOutcome.issue",
		Cardinality: model.Cardinality{Min: 1, Max: "*"},
		Types:       []model.TypeRef{{Code: "BackboneElement"}},
		Children:    []*model.Element{{Name: "code", Path: "OperationOutcome.issue.code", Types: []model.TypeRef{{Code: "code"}}}},
	}
	if PlanField(backbone).Primitive {
		t.Error("a backbone field is marked primitive; it must not carry a \"_field\" sibling (FHIR-005)")
	}

	choice := &model.Element{
		Name:        "value[x]",
		Path:        "Extension.value[x]",
		IsChoice:    true,
		ChoiceBase:  "value",
		Cardinality: model.Cardinality{Min: 0, Max: "1"},
		Types:       []model.TypeRef{{Code: "string"}, {Code: "boolean"}},
	}
	if PlanField(choice).Primitive {
		t.Error("a choice field is marked primitive; it must not carry a \"_field\" sibling (FHIR-005)")
	}
}

// TestPlanCollisionDeterministic asserts two FHIR element names that map to the same
// Go identifier within one struct resolve to a stable, ascending-suffixed sequence, so
// the generated output is byte-stable regardless of map iteration order. The scalar
// string primitive "value" contributes its value field ("Value") and its "_field"
// sibling ("ValueElement"); the choice element "value[x]" must skip both when resolving
// its own stem, so the stem lands on "Value2" and the choice's single string branch
// expands to the suffixed storage field "Value2String" (FHIR-001/002: a choice is
// stored as suffixed branch fields, never a single bare field).
func TestPlanCollisionDeterministic(t *testing.T) {
	t.Parallel()
	root := &model.Element{
		Name: "Thing",
		Path: "Thing",
		Children: []*model.Element{
			{Name: "value", Path: "Thing.value", Cardinality: model.Cardinality{Min: 0, Max: "1"}, Types: []model.TypeRef{{Code: "string"}}},
			{Name: "value[x]", Path: "Thing.value[x]", IsChoice: true, ChoiceBase: "value", Cardinality: model.Cardinality{Min: 0, Max: "1"}, Types: []model.TypeRef{{Code: "string"}}},
		},
	}
	ty := &model.Type{Name: "Thing", Kind: model.KindComplexType, Root: root}
	pt := PlanType(ty, Options{})
	if len(pt.Fields) != 3 {
		t.Fatalf("planned %d fields %+v, want 3 (Value, ValueElement, Value2String)", len(pt.Fields), pt.Fields)
	}
	if pt.Fields[0].GoName != "Value" || pt.Fields[1].GoName != "ValueElement" || pt.Fields[2].GoName != "Value2String" {
		t.Errorf("collision resolution = (%q, %q, %q), want (Value, ValueElement, Value2String)",
			pt.Fields[0].GoName, pt.Fields[1].GoName, pt.Fields[2].GoName)
	}
	if pt.Fields[1].SiblingOf != "Value" {
		t.Errorf("sibling SiblingOf = %q, want Value", pt.Fields[1].SiblingOf)
	}
	// The choice storage field is a plain branch field, not a "_field" primitive
	// sibling: a "[x]" choice gets no "_field" companion (its branches box their own
	// primitives through the wrapper types).
	if pt.Fields[2].IsPrimitiveSibling() {
		t.Error("the choice storage field must not be a primitive sibling (it has no \"_field\" companion)")
	}
	// The choice group is recorded with its sealed interface and getter aligned to the
	// disambiguated stem, and its one branch is the boxed string wrapper.
	if len(pt.Choices) != 1 {
		t.Fatalf("planned %d choices, want 1", len(pt.Choices))
	}
	c := pt.Choices[0]
	if c.Interface != "ThingValue" || c.Getter != "Value2" {
		t.Errorf("choice (iface, getter) = (%q, %q), want (ThingValue, Value2)", c.Interface, c.Getter)
	}
	if len(c.Branches) != 1 || c.Branches[0].GoType != "FHIRString" || c.Branches[0].Field != "Value2String" {
		t.Errorf("choice branch = %+v, want one FHIRString branch stored in Value2String", c.Branches)
	}
	if c.Branches[0].JSONName != "valueString" || c.Branches[0].Setter != "SetValue2String" {
		t.Errorf("choice branch (json, setter) = (%q, %q), want (valueString, SetValue2String)",
			c.Branches[0].JSONName, c.Branches[0].Setter)
	}
}

// TestPlanBackboneShapeDedup asserts two occurrence paths carrying the same backbone
// shape collapse to a single PlannedBackbone — deduplication by shape, not by path.
func TestPlanBackboneShapeDedup(t *testing.T) {
	t.Parallel()
	mkBackbone := func(path string) *model.Element {
		return &model.Element{
			Name:        lastSeg(path),
			Path:        path,
			Cardinality: model.Cardinality{Min: 0, Max: "*"},
			Types:       []model.TypeRef{{Code: "BackboneElement"}},
			Children: []*model.Element{
				{Name: "code", Path: path + ".code", Cardinality: model.Cardinality{Min: 0, Max: "1"}, Types: []model.TypeRef{{Code: "string"}}},
			},
		}
	}
	root := &model.Element{
		Name:     "Thing",
		Path:     "Thing",
		Children: []*model.Element{mkBackbone("Thing.one"), mkBackbone("Thing.two")},
	}
	ty := &model.Type{Name: "Thing", Kind: model.KindComplexType, Root: root}
	pt := PlanType(ty, Options{})
	if len(pt.Backbones) != 1 {
		t.Fatalf("planned %d backbones, want 1 (shape-dedup)", len(pt.Backbones))
	}
	// Both fields must point at the single deduplicated backbone type.
	if pt.Fields[0].GoType != pt.Fields[1].GoType {
		t.Errorf("deduped backbone fields use different types: %q vs %q", pt.Fields[0].GoType, pt.Fields[1].GoType)
	}
}

func TestPlanEmbedsElementBaseForComplexType(t *testing.T) {
	t.Parallel()
	root := &model.Element{
		Name: "Period",
		Path: "Period",
		Children: []*model.Element{
			{Name: "id", Path: "Period.id", Cardinality: model.Cardinality{Min: 0, Max: "1"}, Types: []model.TypeRef{{Code: "string"}}},
			{Name: "extension", Path: "Period.extension", Cardinality: model.Cardinality{Min: 0, Max: "*"}, Types: []model.TypeRef{{Code: "Extension"}}},
			{Name: "start", Path: "Period.start", Cardinality: model.Cardinality{Min: 0, Max: "1"}, Types: []model.TypeRef{{Code: "dateTime"}}},
		},
	}
	ty := &model.Type{Name: "Period", Kind: model.KindComplexType, Root: root}
	pt := PlanType(ty, Options{})
	// A complex datatype with no top-level modifierExtension embeds Element and drops
	// the base members it supplies (id, extension); its one own element (start, a
	// dateTime primitive) survives with its generated "_field" sibling.
	if pt.EmbeddedBase != "Element" {
		t.Errorf("EmbeddedBase = %q, want Element", pt.EmbeddedBase)
	}
	if len(pt.Fields) != 2 {
		t.Fatalf("planned %d fields %+v, want Start and StartElement", len(pt.Fields), pt.Fields)
	}
	if pt.Fields[0].GoName != "Start" || pt.Fields[1].GoName != "StartElement" {
		t.Errorf("fields = (%q, %q), want (Start, StartElement)", pt.Fields[0].GoName, pt.Fields[1].GoName)
	}
}

func TestPlanBaseTypeKeepsMembersAndEmbedsNothing(t *testing.T) {
	t.Parallel()
	root := &model.Element{
		Name: "Element",
		Path: "Element",
		Children: []*model.Element{
			{Name: "id", Path: "Element.id", Cardinality: model.Cardinality{Min: 0, Max: "1"}, Types: []model.TypeRef{{Code: "string"}}},
			{Name: "extension", Path: "Element.extension", Cardinality: model.Cardinality{Min: 0, Max: "*"}, Types: []model.TypeRef{{Code: "Extension"}}},
		},
	}
	ty := &model.Type{Name: "Element", Kind: model.KindComplexType, Abstract: true, Root: root}
	pt := PlanType(ty, Options{IsBaseType: true})
	// A base type embeds nothing, keeps its own members, and suppresses primitive
	// "_field" siblings so it never defines MarshalJSON (which would shadow an
	// embedding type's own MarshalJSON when embedded by value).
	if pt.EmbeddedBase != "" {
		t.Errorf("base type EmbeddedBase = %q, want empty", pt.EmbeddedBase)
	}
	if !pt.IsBaseType {
		t.Error("base type should be flagged IsBaseType")
	}
	if pt.HasPrimitiveSibling() {
		t.Error("base type must not carry primitive _field siblings")
	}
	if len(pt.Fields) != 2 || pt.Fields[0].GoName != "ID" || pt.Fields[1].GoName != "Extension" {
		t.Errorf("base type fields = %+v, want ID and Extension only", pt.Fields)
	}
}

func lastSeg(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i+1:]
		}
	}
	return path
}

// TestPlanRecursionBoundaryUsesNamedType asserts a contentReference recursion
// boundary (a node that kept its marker because its anchor is already expanding, so
// it has no children) is planned as a named reference back to the anchor's backbone
// type, not as an undefined bare type. This honours the model contract that the
// marker is kept for the planner to emit a self-referential named type.
func TestPlanRecursionBoundaryUsesNamedType(t *testing.T) {
	t.Parallel()
	// Model a one-level recursion: Thing.node (backbone) whose child node is the
	// boundary pointing back at #Thing.node.
	boundary := &model.Element{
		Name:             "node",
		Path:             "Thing.node.node",
		Cardinality:      model.Cardinality{Min: 0, Max: "*"},
		ContentReference: "Thing.node",
	}
	node := &model.Element{
		Name:        "node",
		Path:        "Thing.node",
		Cardinality: model.Cardinality{Min: 0, Max: "*"},
		Types:       []model.TypeRef{{Code: "BackboneElement"}},
		Children: []*model.Element{
			{Name: "code", Path: "Thing.node.code", Cardinality: model.Cardinality{Min: 0, Max: "1"}, Types: []model.TypeRef{{Code: "string"}}},
			boundary,
		},
	}
	root := &model.Element{Name: "Thing", Path: "Thing", Children: []*model.Element{node}}
	ty := &model.Type{Name: "Thing", Kind: model.KindComplexType, Root: root}
	pt := PlanType(ty, Options{})

	if len(pt.Backbones) != 1 {
		t.Fatalf("planned %d backbones, want 1", len(pt.Backbones))
	}
	bb := pt.Backbones[0]
	if bb.GoName != "ThingNode" {
		t.Fatalf("backbone name = %q, want ThingNode", bb.GoName)
	}
	// The recursive child field must point back at the named backbone type, decorated
	// as a slice for its repeating cardinality.
	var found bool
	for _, f := range bb.Fields {
		if f.JSONName == "node" {
			found = true
			if f.GoType != "[]ThingNode" {
				t.Errorf("recursive boundary field GoType = %q, want []ThingNode", f.GoType)
			}
		}
	}
	if !found {
		t.Error("recursive backbone is missing its self-referential node field")
	}
}
