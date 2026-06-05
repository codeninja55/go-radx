package plan

import (
	"testing"

	"github.com/codeninja55/go-radx/fhir/internal/gen/model"
)

// buildDescriptorType hand-builds a resource model.Type with the four element shapes the
// descriptor planner reads: a required scalar (status), a required boolean (active), a
// choice ([x]) group (value[x] with two branches), and a required-binding code (gender),
// so the descriptor plan is exercised without loading the real bundle.
func buildDescriptorType() *model.Type {
	required := model.Cardinality{Min: 1, Max: "1"}
	optional := model.Cardinality{Min: 0, Max: "1"}
	const genderVS = "http://hl7.org/fhir/ValueSet/administrative-gender"

	root := &model.Element{
		Name: "Sample",
		Path: "Sample",
		Children: []*model.Element{
			{Name: "status", Path: "Sample.status", Cardinality: required, Types: []model.TypeRef{{Code: "code"}}},
			{Name: "active", Path: "Sample.active", Cardinality: required, Types: []model.TypeRef{{Code: "boolean"}}},
			{
				Name: "value[x]", Path: "Sample.value[x]", Cardinality: optional, IsChoice: true, ChoiceBase: "value",
				Types: []model.TypeRef{{Code: "Quantity"}, {Code: "string"}},
			},
			{
				Name: "gender", Path: "Sample.gender", Cardinality: optional, Types: []model.TypeRef{{Code: "code"}},
				Binding: &model.Binding{Strength: "required", ValueSet: genderVS},
			},
		},
	}
	return &model.Type{Name: "Sample", Kind: model.KindResource, Root: root}
}

// descriptorResolver enumerates the gender binding so the code field is typed as the
// AdministrativeGender enum and gets a binding check.
func descriptorResolver() fakeResolver {
	const genderVS = "http://hl7.org/fhir/ValueSet/administrative-gender"
	return fakeResolver{
		names:   map[string]string{genderVS: "AdministrativeGender"},
		codes:   map[string][]string{genderVS: {"male", "female", "other", "unknown"}},
		inlined: map[string]bool{genderVS: true},
	}
}

// TestPlanValidationDescriptor pins the descriptor planned for a resource: the required
// scalars by presence, the choice group's suffixed storage fields, and the
// required-binding code's validator and enum. The status field is also a required-binding
// code in real resources, but here only gender carries a binding, so the binding count is
// one.
func TestPlanValidationDescriptor(t *testing.T) {
	typ := buildDescriptorType()
	resolver := descriptorResolver()
	pt := PlanType(typ, Options{Bindings: resolver})

	vd, ok := PlanValidationDescriptor(typ, pt, resolver)
	if !ok {
		t.Fatal("PlanValidationDescriptor returned ok=false for a resource")
	}
	if vd.GoName != "Sample" || vd.FHIRName != "Sample" {
		t.Errorf("descriptor names = {%s %s}, want {Sample Sample}", vd.GoName, vd.FHIRName)
	}

	if len(vd.Required) != 2 {
		t.Fatalf("required count = %d, want 2 (status, active): %+v", len(vd.Required), vd.Required)
	}
	wantRequired := map[string]bool{"Sample.status": true, "Sample.active": true}
	for _, rf := range vd.Required {
		if !wantRequired[rf.Path] {
			t.Errorf("unexpected required path %q", rf.Path)
		}
		if rf.Repeats {
			t.Errorf("required %q is single-valued, should not be marked repeating", rf.Path)
		}
	}

	if len(vd.Choices) != 1 {
		t.Fatalf("choice count = %d, want 1: %+v", len(vd.Choices), vd.Choices)
	}
	choice := vd.Choices[0]
	if choice.Path != "Sample.value[x]" {
		t.Errorf("choice path = %q, want Sample.value[x]", choice.Path)
	}
	if len(choice.Fields) != 2 {
		t.Errorf("choice fields = %v, want two suffixed storage fields", choice.Fields)
	}

	if len(vd.Bindings) != 1 {
		t.Fatalf("binding count = %d, want 1 (gender): %+v", len(vd.Bindings), vd.Bindings)
	}
	binding := vd.Bindings[0]
	if binding.Path != "Sample.gender" {
		t.Errorf("binding path = %q, want Sample.gender", binding.Path)
	}
	if binding.Validator != "validAdministrativeGender" {
		t.Errorf("binding validator = %q, want validAdministrativeGender", binding.Validator)
	}
	if binding.EnumName != "AdministrativeGender" {
		t.Errorf("binding enum = %q, want AdministrativeGender", binding.EnumName)
	}
}

// TestPlanValidationDescriptorSkipsNonResource confirms a complex datatype yields ok=false
// (no resourceType to register under), so only concrete resources get a descriptor.
func TestPlanValidationDescriptorSkipsNonResource(t *testing.T) {
	typ := buildDescriptorType()
	typ.Kind = model.KindComplexType
	pt := PlanType(typ, Options{Bindings: descriptorResolver()})
	if _, ok := PlanValidationDescriptor(typ, pt, descriptorResolver()); ok {
		t.Error("a complex datatype should not get a validation descriptor")
	}
}

// TestPlanValidationDescriptorBundleHasExtra confirms the Bundle descriptor is flagged for
// the hand-written extra checks (bdl-* invariants and reference integrity).
func TestPlanValidationDescriptorBundleHasExtra(t *testing.T) {
	typ := buildDescriptorType()
	typ.Name = "Bundle"
	typ.Root.Name = "Bundle"
	typ.Root.Path = "Bundle"
	pt := PlanType(typ, Options{Bindings: descriptorResolver()})
	vd, ok := PlanValidationDescriptor(typ, pt, descriptorResolver())
	if !ok {
		t.Fatal("Bundle should get a descriptor")
	}
	if !vd.HasExtra {
		t.Error("the Bundle descriptor should be flagged HasExtra for the bdl-* and reference checks")
	}
}
