package gen

import (
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir/internal/gen/loader"
)

// TestNoEmptyRequiredBindingEnum is the empty-const-set guard: it fails if any
// required-binding enum the generator would emit for the real R5 bundle is both
// enumerable (a closed enum) and empty. An empty closed enum would reject every code
// at the boundary — strictly worse than not enforcing the binding at all — so the
// generator must instead emit a documented not-inlined open string. This guard makes a
// silently-empty const set impossible to ship.
func TestNoEmptyRequiredBindingEnum(t *testing.T) {
	bundle := loadR5(t)
	enums, err := PlannedEnums(bundle)
	if err != nil {
		t.Fatalf("PlannedEnums: %v", err)
	}
	if len(enums) == 0 {
		t.Fatal("PlannedEnums returned no enums; the R5 bundle has required bindings, so this is a regression")
	}
	for _, e := range enums {
		if e.IsEmpty() {
			t.Errorf("required-binding enum %s (%s) is an enumerable closed enum with an empty const set; "+
				"it must be emitted as a documented not-inlined open string instead", e.GoName, e.ValueSetURL)
		}
		if !e.NotInlined && len(e.Consts) == 0 {
			t.Errorf("required-binding enum %s would ship a closed type with no constants", e.GoName)
		}
		if e.NotInlined && e.NotInlinedReason == "" {
			t.Errorf("not-inlined binding %s has no documented reason", e.GoName)
		}
	}
}

// TestEnumerableEnumsResolve pins that the workflow and well-known required bindings
// resolve to closed enums with the expected codes, so the enumeration path (extensional
// concepts and complete code systems) is proven against the real bundle, not only the
// fake-resolver unit tests.
func TestEnumerableEnumsResolve(t *testing.T) {
	bundle := loadR5(t)
	enums, err := PlannedEnums(bundle)
	if err != nil {
		t.Fatalf("PlannedEnums: %v", err)
	}
	byName := map[string]bool{}
	codes := map[string][]string{}
	for _, e := range enums {
		if e.NotInlined {
			continue
		}
		byName[e.GoName] = true
		for _, c := range e.Consts {
			codes[e.GoName] = append(codes[e.GoName], c.Value)
		}
	}
	for _, want := range []string{"AdministrativeGender", "PublicationStatus", "NarrativeStatus", "FlagStatus"} {
		if !byName[want] {
			t.Errorf("required-binding enum %s not generated as a closed enum", want)
		}
	}
	if !contains(codes["AdministrativeGender"], "female") {
		t.Errorf("AdministrativeGender codes = %v, want to contain female", codes["AdministrativeGender"])
	}
}

// TestFilterAndExternalBindingsAreNotInlined pins the not-inlined boundary against the
// real bundle: a required binding whose value set draws from an un-vendored external
// terminology (UCUM, an IETF/ISO registry) is emitted as a documented open string, not
// a closed enum, and its reason names why. This is the terminology-scope boundary the
// loader's Filter capture and the resolver's external-system detection produce.
func TestFilterAndExternalBindingsAreNotInlined(t *testing.T) {
	bundle := loadR5(t)
	enums, err := PlannedEnums(bundle)
	if err != nil {
		t.Fatalf("PlannedEnums: %v", err)
	}
	notInlined := map[string]string{}
	for _, e := range enums {
		if e.NotInlined {
			notInlined[e.GoName] = e.NotInlinedReason
		}
	}
	if len(notInlined) == 0 {
		t.Fatal("no not-inlined bindings found; R5 required bindings include external-terminology sets, so this is a regression")
	}
	// UCUMCodes draws from http://unitsofmeasure.org, which is not vendored, so it must
	// be a documented open string with a reason naming the external system.
	reason, ok := notInlined["UCUMCodes"]
	if !ok {
		t.Errorf("UCUMCodes not marked not-inlined; want a documented open string (external terminology)")
	} else if !strings.Contains(reason, "unitsofmeasure.org") {
		t.Errorf("UCUMCodes reason = %q, want it to name the external system unitsofmeasure.org", reason)
	}
}

// TestResolverClassifiesIntensionalInclude exercises the resolver's filter handling
// directly with a hand-built intensional value set, so the path the loader's captured
// Filter feeds is covered even though no R5 required binding is filter-defined today
// (a future release or R4 may differ). A filter-based include must classify as not
// enumerable with a reason naming the filter, never silently yield zero codes that a
// caller could mistake for a complete empty set.
func TestResolverClassifiesIntensionalInclude(t *testing.T) {
	inc := loader.ValueSetInclude{
		System: "http://loinc.org",
		Filter: []loader.ValueSetFilter{{Property: "parent", Op: "=", Value: "LP43571-6"}},
	}
	r := &bindingResolver{}
	codes, reason := r.enumerateInclude(inc)
	if reason == "" {
		t.Fatal("filter-defined include classified as enumerable; want a not-inlined reason")
	}
	if len(codes) != 0 {
		t.Errorf("filter-defined include yielded %d codes, want 0", len(codes))
	}
	if !strings.Contains(reason, "filter") || !strings.Contains(reason, "parent") {
		t.Errorf("reason = %q, want it to name the intensional filter (parent)", reason)
	}
}

func loadR5(t *testing.T) *loader.Bundle {
	t.Helper()
	bundle, err := loader.Load(vendoredR5Dir)
	if err != nil {
		t.Fatalf("load vendored R5 bundle: %v", err)
	}
	return bundle
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
