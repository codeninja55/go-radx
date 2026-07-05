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

// buildNestedDescriptorType hand-builds a resource with the nested shapes the depth
// planner reads: a top-level date field, a date-family choice branch, a repeating
// backbone with a required child and a dateTime, a pointer backbone whose only content
// is a deeper backbone with a required child (the transitive-reachability case), and a
// backbone with no required or date-family content (pruned entirely).
func buildNestedDescriptorType() *model.Type {
	required := model.Cardinality{Min: 1, Max: "1"}
	optional := model.Cardinality{Min: 0, Max: "1"}
	repeating := model.Cardinality{Min: 0, Max: "*"}

	root := &model.Element{
		Name: "Sample",
		Path: "Sample",
		Children: []*model.Element{
			{Name: "text", Path: "Sample.text", Cardinality: optional, Types: []model.TypeRef{{Code: "Narrative"}}},
			{Name: "birthDate", Path: "Sample.birthDate", Cardinality: optional, Types: []model.TypeRef{{Code: "date"}}},
			{
				Name: "effective[x]", Path: "Sample.effective[x]", Cardinality: optional, IsChoice: true, ChoiceBase: "effective",
				Types: []model.TypeRef{{Code: "dateTime"}, {Code: "Quantity"}},
			},
			{
				Name: "series", Path: "Sample.series", Cardinality: repeating, Types: []model.TypeRef{{Code: "BackboneElement"}},
				Children: []*model.Element{
					{Name: "uid", Path: "Sample.series.uid", Cardinality: required, Types: []model.TypeRef{{Code: "id"}}},
					{Name: "started", Path: "Sample.series.started", Cardinality: optional, Types: []model.TypeRef{{Code: "dateTime"}}},
				},
			},
			{
				Name: "outer", Path: "Sample.outer", Cardinality: optional, Types: []model.TypeRef{{Code: "BackboneElement"}},
				Children: []*model.Element{
					{
						Name: "inner", Path: "Sample.outer.inner", Cardinality: optional, Types: []model.TypeRef{{Code: "BackboneElement"}},
						Children: []*model.Element{
							{Name: "system", Path: "Sample.outer.inner.system", Cardinality: required, Types: []model.TypeRef{{Code: "uri"}}},
						},
					},
				},
			},
			{
				Name: "plain", Path: "Sample.plain", Cardinality: optional, Types: []model.TypeRef{{Code: "BackboneElement"}},
				Children: []*model.Element{
					{Name: "note", Path: "Sample.plain.note", Cardinality: optional, Types: []model.TypeRef{{Code: "string"}}},
				},
			},
		},
	}
	return &model.Type{Name: "Sample", Kind: model.KindResource, Root: root}
}

// TestPlanValidationDescriptorDepth pins the depth surface: the top-level date and the
// boxed choice branch become PrimitiveChecks, the backbone with a required child and
// the backbone whose depth holds one both get required walk calls (transitive
// reachability included), the date-carrying backbone gets a lexical walk call, the
// content-free backbone is pruned from calls and helpers, and the DomainResource
// embedder is flagged for the extension-url walk.
func TestPlanValidationDescriptorDepth(t *testing.T) {
	typ := buildNestedDescriptorType()
	resolver := descriptorResolver()
	pt := PlanType(typ, Options{Bindings: resolver})

	vd, ok := PlanValidationDescriptor(typ, pt, resolver)
	if !ok {
		t.Fatal("PlanValidationDescriptor returned ok=false for a resource")
	}

	if !vd.CheckExtensions {
		t.Error("a DomainResource embedder should be flagged for the extension-url walk")
	}

	if len(vd.Primitives) != 2 {
		t.Fatalf("top-level primitive checks = %+v, want birthDate + effectiveDateTime", vd.Primitives)
	}
	byPath := map[string]PrimitiveCheck{}
	for _, pc := range vd.Primitives {
		byPath[pc.Path] = pc
	}
	if pc := byPath["Sample.birthDate"]; pc.Validator != "validDateLexical" || pc.Boxed {
		t.Errorf("birthDate check = %+v, want plain validDateLexical", pc)
	}
	if pc := byPath["Sample.effectiveDateTime"]; pc.Validator != "validDateTimeLexical" || !pc.Boxed {
		t.Errorf("effectiveDateTime check = %+v, want boxed validDateTimeLexical", pc)
	}

	requiredTargets := map[string]bool{}
	for _, c := range vd.RequiredCalls {
		requiredTargets[c.TypeGoName] = true
	}
	if !requiredTargets["SampleSeries"] || !requiredTargets["SampleOuter"] {
		t.Errorf("required walk calls = %+v, want SampleSeries and SampleOuter (transitive)", vd.RequiredCalls)
	}
	if requiredTargets["SamplePlain"] {
		t.Errorf("the content-free backbone should be pruned from required calls: %+v", vd.RequiredCalls)
	}

	lexicalTargets := map[string]bool{}
	for _, c := range vd.LexicalCalls {
		lexicalTargets[c.TypeGoName] = true
	}
	if !lexicalTargets["SampleSeries"] || lexicalTargets["SampleOuter"] || lexicalTargets["SamplePlain"] {
		t.Errorf("lexical walk calls = %+v, want SampleSeries only", vd.LexicalCalls)
	}

	helpers := map[string]BackboneHelper{}
	for _, h := range vd.Helpers {
		helpers[h.GoName] = h
	}
	if _, pruned := helpers["SamplePlain"]; pruned {
		t.Error("the content-free backbone should emit no helper")
	}
	series, ok := helpers["SampleSeries"]
	if !ok || !series.EmitRequired || !series.EmitLexical {
		t.Fatalf("SampleSeries helper = %+v, want required+lexical", series)
	}
	if len(series.Required) != 1 || series.Required[0].Path != ".uid" {
		t.Errorf("SampleSeries required = %+v, want .uid", series.Required)
	}
	if len(series.Primitives) != 1 || series.Primitives[0].Path != ".started" {
		t.Errorf("SampleSeries primitives = %+v, want .started", series.Primitives)
	}
	outer, ok := helpers["SampleOuter"]
	if !ok || !outer.EmitRequired || outer.EmitLexical {
		t.Fatalf("SampleOuter helper = %+v, want required-only (transitively)", outer)
	}
	if len(outer.Required) != 0 || len(outer.RequiredCalls) != 1 || outer.RequiredCalls[0].TypeGoName != "SampleOuterInner" {
		t.Errorf("SampleOuter should only forward into SampleOuterInner: %+v", outer)
	}
	inner, ok := helpers["SampleOuterInner"]
	if !ok || len(inner.Required) != 1 || inner.Required[0].Path != ".system" {
		t.Errorf("SampleOuterInner required = %+v, want .system", inner.Required)
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

// TestPlanValidationDescriptorSummaryFlags confirms the planner records per-element summary
// metadata MarshalSummary filters on: the isSummary, mandatory, modifier, narrative, and
// count flags, with a choice ([x]) group contributing one entry per suffixed branch key.
func TestPlanValidationDescriptorSummaryFlags(t *testing.T) {
	required := model.Cardinality{Min: 1, Max: "1"}
	optional := model.Cardinality{Min: 0, Max: "1"}
	root := &model.Element{
		Name: "Sample", Path: "Sample",
		Children: []*model.Element{
			{Name: "text", Path: "Sample.text", Cardinality: optional, Types: []model.TypeRef{{Code: "Narrative"}}},
			{Name: "status", Path: "Sample.status", Cardinality: required, IsSummary: true, IsModifier: true, Types: []model.TypeRef{{Code: "code"}}},
			{
				Name: "value[x]", Path: "Sample.value[x]", Cardinality: optional, IsChoice: true, ChoiceBase: "value",
				IsSummary: true, Types: []model.TypeRef{{Code: "Quantity"}, {Code: "string"}},
			},
			{Name: "total", Path: "Sample.total", Cardinality: optional, IsSummary: true, Types: []model.TypeRef{{Code: "integer"}}},
			{Name: "note", Path: "Sample.note", Cardinality: optional, Types: []model.TypeRef{{Code: "string"}}},
		},
	}
	typ := &model.Type{Name: "Sample", Kind: model.KindResource, Root: root}
	resolver := descriptorResolver()
	pt := PlanType(typ, Options{Bindings: resolver})

	vd, ok := PlanValidationDescriptor(typ, pt, resolver)
	if !ok {
		t.Fatal("PlanValidationDescriptor returned ok=false for a resource")
	}

	byKey := make(map[string]SummaryFlag, len(vd.Summary))
	for _, sf := range vd.Summary {
		byKey[sf.JSONName] = sf
	}

	if sf := byKey["text"]; !sf.IsText {
		t.Errorf("text element not marked IsText: %+v", sf)
	}
	if sf := byKey["status"]; !sf.IsSummary || !sf.IsMandatory || !sf.IsModifier {
		t.Errorf("status flags = %+v, want summary+mandatory+modifier", sf)
	}
	if sf := byKey["total"]; !sf.IsSummary || !sf.IsCount {
		t.Errorf("total flags = %+v, want summary+count", sf)
	}
	if sf := byKey["note"]; sf.IsSummary || sf.IsMandatory || sf.IsModifier || sf.IsText || sf.IsCount {
		t.Errorf("note should carry no summary flags: %+v", sf)
	}
	// The choice group contributes one entry per suffixed branch, both summary-flagged.
	for _, branch := range []string{"valueQuantity", "valueString"} {
		sf, ok := byKey[branch]
		if !ok {
			t.Errorf("choice branch %q has no summary flag; got %v", branch, vd.Summary)
			continue
		}
		if !sf.IsSummary {
			t.Errorf("choice branch %q not marked IsSummary: %+v", branch, sf)
		}
	}
}
