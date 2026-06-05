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

// TestPlanCollisionDeterministic asserts two FHIR element names that map to the same
// Go identifier within one struct resolve to a stable, ascending-suffixed pair, so
// the generated output is byte-stable regardless of map iteration order.
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
	if len(pt.Fields) != 2 {
		t.Fatalf("planned %d fields, want 2", len(pt.Fields))
	}
	if pt.Fields[0].GoName != "Value" || pt.Fields[1].GoName != "Value2" {
		t.Errorf("collision resolution = (%q, %q), want (Value, Value2)", pt.Fields[0].GoName, pt.Fields[1].GoName)
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

func TestPlanSkipsBaseMembers(t *testing.T) {
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
	pt := PlanType(ty, Options{SkipBaseMembers: true})
	if len(pt.Fields) != 1 || pt.Fields[0].GoName != "Start" {
		t.Fatalf("SkipBaseMembers planned %d fields %+v, want only Start", len(pt.Fields), pt.Fields)
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
