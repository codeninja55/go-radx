package model

import (
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir/internal/gen/loader"
)

// elem is a terse ElementDefinition builder for the synthetic snapshots below.
func elem(path string, min int, max string, types ...string) loader.ElementDefinition {
	ed := loader.ElementDefinition{Path: path, Min: min, Max: max}
	for _, code := range types {
		ed.Type = append(ed.Type, loader.ElementType{Code: code})
	}
	return ed
}

// sd wraps a snapshot in a resource StructureDefinition for BuildType.
func sd(name string, elements ...loader.ElementDefinition) *loader.StructureDefinition {
	return &loader.StructureDefinition{
		Name:     name,
		URL:      "http://hl7.org/fhir/StructureDefinition/" + name,
		Kind:     "resource",
		Snapshot: &loader.Snapshot{Element: elements},
	}
}

// childPath walks a chain of final-segment names from the root and returns the
// reached element, failing the test if any link is missing.
func childPath(t *testing.T, root *Element, names ...string) *Element {
	t.Helper()
	node := root
	for _, name := range names {
		c, ok := node.Child(name)
		if !ok {
			t.Fatalf("element %q has no child %q", node.Path, name)
		}
		node = c
	}
	return node
}

// TestBackboneTreeFullyRecursed builds an inline-nested backbone and asserts the
// tree carries the deep child, the structural guard against empty backbones.
func TestBackboneTreeFullyRecursed(t *testing.T) {
	t.Parallel()

	def := sd("Observation",
		elem("Observation", 0, "*", "BackboneElement"),
		elem("Observation.component", 0, "*", "BackboneElement"),
		elem("Observation.component.code", 1, "1", "CodeableConcept"),
		elem("Observation.component.referenceRange", 0, "*", "BackboneElement"),
		elem("Observation.component.referenceRange.low", 0, "1", "Quantity"),
		elem("Observation.component.referenceRange.high", 0, "1", "Quantity"),
	)

	typ, err := BuildType(def)
	if err != nil {
		t.Fatalf("BuildType: %v", err)
	}

	low := childPath(t, typ.Root, "component", "referenceRange", "low")
	if low.Path != "Observation.component.referenceRange.low" {
		t.Errorf("deep child path = %q, want full dotted path", low.Path)
	}
	if len(low.Types) != 1 || low.Types[0].Code != "Quantity" {
		t.Errorf("low types = %v, want [Quantity]", low.Types)
	}

	refRange := childPath(t, typ.Root, "component", "referenceRange")
	if len(refRange.Children) != 2 {
		t.Errorf("referenceRange has %d children, want 2 (low, high)", len(refRange.Children))
	}
	if !refRange.IsBackbone() {
		t.Errorf("referenceRange should be classified as a backbone element")
	}
}

// TestContentReferenceGraft is the FHIR-006 regression: a backbone whose children
// come from a contentReference (rather than inline) must be populated, not empty.
// This mirrors the real R5 Observation, where component.referenceRange reuses
// #Observation.referenceRange.
func TestContentReferenceGraft(t *testing.T) {
	t.Parallel()

	def := sd("Observation",
		elem("Observation", 0, "*", "BackboneElement"),
		elem("Observation.referenceRange", 0, "*", "BackboneElement"),
		elem("Observation.referenceRange.low", 0, "1", "Quantity"),
		elem("Observation.referenceRange.high", 0, "1", "Quantity"),
		elem("Observation.component", 0, "*", "BackboneElement"),
		// referenceRange under component carries no inline children; its shape comes
		// from the contentReference anchor.
		func() loader.ElementDefinition {
			ed := elem("Observation.component.referenceRange", 0, "*")
			ed.ContentReference = "#Observation.referenceRange"
			return ed
		}(),
	)

	typ, err := BuildType(def)
	if err != nil {
		t.Fatalf("BuildType: %v", err)
	}

	refRange := childPath(t, typ.Root, "component", "referenceRange")
	if len(refRange.Children) != 2 {
		t.Fatalf("component.referenceRange has %d children, want 2 grafted from the contentReference", len(refRange.Children))
	}
	low := childPath(t, refRange, "low")
	if len(low.Types) != 1 || low.Types[0].Code != "Quantity" {
		t.Errorf("grafted low types = %v, want [Quantity]", low.Types)
	}
	if refRange.ContentReference != "Observation.referenceRange" {
		t.Errorf("contentReference = %q, want %q for traceability", refRange.ContentReference, "Observation.referenceRange")
	}

	// The graft must be a deep copy: mutating the component subtree must not perturb
	// the source subtree.
	low.Name = "MUTATED"
	source := childPath(t, typ.Root, "referenceRange", "low")
	if source.Name == "MUTATED" {
		t.Errorf("grafted subtree aliases the source; expected an independent deep copy")
	}
}

// TestContentReferenceRebasesChildPaths asserts grafted children are rebased onto
// their occurrence path, not left under the donor path the snapshot defined them
// under, so the IR is a true occurrence-path tree.
func TestContentReferenceRebasesChildPaths(t *testing.T) {
	t.Parallel()

	def := sd("Observation",
		elem("Observation", 0, "*", "BackboneElement"),
		elem("Observation.referenceRange", 0, "*", "BackboneElement"),
		elem("Observation.referenceRange.low", 0, "1", "Quantity"),
		elem("Observation.component", 0, "*", "BackboneElement"),
		func() loader.ElementDefinition {
			ed := elem("Observation.component.referenceRange", 0, "*")
			ed.ContentReference = "#Observation.referenceRange"
			return ed
		}(),
	)

	typ, err := BuildType(def)
	if err != nil {
		t.Fatalf("BuildType: %v", err)
	}

	low := childPath(t, typ.Root, "component", "referenceRange", "low")
	if low.Path != "Observation.component.referenceRange.low" {
		t.Errorf("grafted low path = %q, want the occurrence path Observation.component.referenceRange.low", low.Path)
	}
}

// TestRecursiveContentReferenceBounded is the unbounded-recursion regression: a
// self-recursive contentReference (concept.concept reuses #concept) must be grafted
// once and then bounded at the recursion boundary, which keeps its contentReference
// marker and carries no further expansion. Without the bound, BuildType would
// expand forever.
func TestRecursiveContentReferenceBounded(t *testing.T) {
	t.Parallel()

	def := sd("CodeSystem",
		elem("CodeSystem", 0, "1", "DomainResource"),
		elem("CodeSystem.concept", 0, "*", "BackboneElement"),
		elem("CodeSystem.concept.code", 1, "1", "code"),
		func() loader.ElementDefinition {
			ed := elem("CodeSystem.concept.concept", 0, "*")
			ed.ContentReference = "#CodeSystem.concept"
			return ed
		}(),
	)

	typ, err := BuildType(def)
	if err != nil {
		t.Fatalf("BuildType: %v", err)
	}

	// concept -> concept is grafted once with the donor's children (code, concept).
	cc := childPath(t, typ.Root, "concept", "concept")
	if _, ok := cc.Child("code"); !ok {
		t.Error("first concept.concept should be grafted with the donor's code child")
	}

	// concept -> concept -> concept is the boundary: no further expansion, marker kept.
	boundary := childPath(t, cc, "concept")
	if len(boundary.Children) != 0 {
		t.Errorf("recursion boundary has %d children; want it bounded at 0", len(boundary.Children))
	}
	if boundary.ContentReference != "CodeSystem.concept" {
		t.Errorf("recursion boundary contentReference = %q, want the marker kept", boundary.ContentReference)
	}
	if boundary.Path != "CodeSystem.concept.concept.concept" {
		t.Errorf("recursion boundary path = %q, want the occurrence path", boundary.Path)
	}
}

// TestMissingParentIsError asserts the model fails closed when a child path's
// parent is absent from the snapshot, rather than silently dropping the element
// (the failure mode that produces empty backbones).
func TestMissingParentIsError(t *testing.T) {
	t.Parallel()

	def := sd("Observation",
		elem("Observation", 0, "*", "BackboneElement"),
		// component is omitted, so this child has no parent in the tree.
		elem("Observation.component.code", 1, "1", "CodeableConcept"),
	)

	_, err := BuildType(def)
	if err == nil {
		t.Fatal("BuildType should reject an element whose parent path is missing")
	}
	if !strings.Contains(err.Error(), "Observation.component.code") {
		t.Errorf("error %q should name the offending element", err.Error())
	}
}

// TestDanglingContentReferenceIsError asserts a contentReference whose anchor is
// not in the snapshot is a hard error, not an empty backbone.
func TestDanglingContentReferenceIsError(t *testing.T) {
	t.Parallel()

	def := sd("Observation",
		elem("Observation", 0, "*", "BackboneElement"),
		func() loader.ElementDefinition {
			ed := elem("Observation.component", 0, "*")
			ed.ContentReference = "#Observation.nonexistent"
			return ed
		}(),
	)

	_, err := BuildType(def)
	if err == nil {
		t.Fatal("BuildType should reject a dangling contentReference")
	}
	if !strings.Contains(err.Error(), "Observation.nonexistent") {
		t.Errorf("error %q should name the unresolved anchor", err.Error())
	}
}

// TestChoiceDetection asserts a "[x]" element is flagged with its base name and
// carries every branch type.
func TestChoiceDetection(t *testing.T) {
	t.Parallel()

	def := sd("Observation",
		elem("Observation", 0, "*", "BackboneElement"),
		elem("Observation.value[x]", 0, "1", "Quantity", "CodeableConcept", "string", "boolean"),
		elem("Observation.status", 1, "1", "code"),
	)

	typ, err := BuildType(def)
	if err != nil {
		t.Fatalf("BuildType: %v", err)
	}

	value := childPath(t, typ.Root, "value[x]")
	if !value.IsChoice {
		t.Fatal("value[x] should be flagged IsChoice")
	}
	if value.ChoiceBase != "value" {
		t.Errorf("ChoiceBase = %q, want %q", value.ChoiceBase, "value")
	}
	wantBranches := []string{"Quantity", "CodeableConcept", "string", "boolean"}
	if len(value.Types) != len(wantBranches) {
		t.Fatalf("value[x] has %d branches, want %d", len(value.Types), len(wantBranches))
	}
	for i, want := range wantBranches {
		if value.Types[i].Code != want {
			t.Errorf("branch %d = %q, want %q", i, value.Types[i].Code, want)
		}
	}

	status := childPath(t, typ.Root, "status")
	if status.IsChoice {
		t.Error("status should not be flagged IsChoice")
	}
}

// TestBindingNormalised asserts a coded element's binding strength is captured and
// the value-set version suffix is stripped.
func TestBindingNormalised(t *testing.T) {
	t.Parallel()

	statusEl := elem("Observation.status", 1, "1", "code")
	statusEl.Binding = &loader.ElementBinding{
		Strength: "required",
		ValueSet: "http://hl7.org/fhir/ValueSet/observation-status|5.0.0",
	}
	def := sd("Observation",
		elem("Observation", 0, "*", "BackboneElement"),
		statusEl,
	)

	typ, err := BuildType(def)
	if err != nil {
		t.Fatalf("BuildType: %v", err)
	}

	status := childPath(t, typ.Root, "status")
	if status.Binding == nil {
		t.Fatal("status should carry a binding")
	}
	if !status.Binding.Required() {
		t.Errorf("binding strength = %q, want required", status.Binding.Strength)
	}
	if status.Binding.ValueSet != "http://hl7.org/fhir/ValueSet/observation-status" {
		t.Errorf("binding value set = %q, want version suffix stripped", status.Binding.ValueSet)
	}
}

// TestTypeProfilesCaptured asserts a profiled datatype (Range.low is a Quantity
// profiled as SimpleQuantity) carries its profile in the IR, so the planner can
// apply datatype profiling.
func TestTypeProfilesCaptured(t *testing.T) {
	t.Parallel()

	lowEl := elem("Range.low", 0, "1")
	lowEl.Type = []loader.ElementType{{
		Code:    "Quantity",
		Profile: []string{"http://hl7.org/fhir/StructureDefinition/SimpleQuantity"},
	}}
	def := &loader.StructureDefinition{
		Name:     "Range",
		URL:      "http://hl7.org/fhir/StructureDefinition/Range",
		Kind:     "complex-type",
		Snapshot: &loader.Snapshot{Element: []loader.ElementDefinition{elem("Range", 0, "1"), lowEl}},
	}

	typ, err := BuildType(def)
	if err != nil {
		t.Fatalf("BuildType: %v", err)
	}
	low := childPath(t, typ.Root, "low")
	if len(low.Types) != 1 {
		t.Fatalf("low has %d types, want 1", len(low.Types))
	}
	if got := low.Types[0].Profiles; len(got) != 1 || got[0] != "http://hl7.org/fhir/StructureDefinition/SimpleQuantity" {
		t.Errorf("low type profiles = %v, want [SimpleQuantity URL]", got)
	}
}

// TestSystemPrimitiveNormalisedInTree asserts an Element.id-style FHIRPath System
// type code is normalised to the FHIR primitive name in the built tree.
func TestSystemPrimitiveNormalisedInTree(t *testing.T) {
	t.Parallel()

	def := sd("Observation",
		elem("Observation", 0, "*", "BackboneElement"),
		elem("Observation.id", 0, "1", "http://hl7.org/fhirpath/System.String"),
	)

	typ, err := BuildType(def)
	if err != nil {
		t.Fatalf("BuildType: %v", err)
	}

	id := childPath(t, typ.Root, "id")
	if len(id.Types) != 1 || id.Types[0].Code != "string" {
		t.Errorf("id types = %v, want [string] (System.String normalised)", id.Types)
	}
}

// TestNilAndEmptySnapshotRejected asserts the empty/nil guards fail closed.
func TestNilAndEmptySnapshotRejected(t *testing.T) {
	t.Parallel()

	if _, err := BuildType(nil); err == nil {
		t.Error("BuildType(nil) should error")
	}
	if _, err := BuildType(&loader.StructureDefinition{Name: "X"}); err == nil {
		t.Error("BuildType with no snapshot should error")
	}
}
