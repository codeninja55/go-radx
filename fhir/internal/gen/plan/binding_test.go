package plan

import "testing"

// fakeResolver is a hand-built BindingResolver for the planning unit tests, so the
// planner's binding decisions are exercised without loading the real bundle.
type fakeResolver struct {
	names   map[string]string // valueSetURL -> FHIR name
	codes   map[string][]string
	inlined map[string]bool
	reasons map[string]string
}

func (f fakeResolver) ResolveValueSetName(url string) (string, bool) {
	n, ok := f.names[url]
	return n, ok
}

func (f fakeResolver) ResolveBinding(url string) ([]string, bool, string) {
	return f.codes[url], f.inlined[url], f.reasons[url]
}

// TestPlanBindingEnumerable pins the closed-enum plan for an extensional required
// binding: the Go type name from the value-set name, the const set in value-set order
// with the type-name prefix, and codes with punctuation lifted to valid identifiers.
func TestPlanBindingEnumerable(t *testing.T) {
	const url = "http://hl7.org/fhir/ValueSet/administrative-gender"
	r := fakeResolver{
		names:   map[string]string{url: "AdministrativeGender"},
		codes:   map[string][]string{url: {"male", "female", "other", "unknown"}},
		inlined: map[string]bool{url: true},
	}

	pe := PlanBinding(url, "AdministrativeGender", r)
	if pe.NotInlined {
		t.Fatalf("AdministrativeGender planned not-inlined, want closed enum: reason=%q", pe.NotInlinedReason)
	}
	if pe.GoName != "AdministrativeGender" {
		t.Errorf("GoName = %q, want AdministrativeGender", pe.GoName)
	}
	if pe.IsEmpty() {
		t.Fatal("enumerable enum has an empty const set")
	}
	wantConsts := []struct{ name, value string }{
		{"AdministrativeGenderMale", "male"},
		{"AdministrativeGenderFemale", "female"},
		{"AdministrativeGenderOther", "other"},
		{"AdministrativeGenderUnknown", "unknown"},
	}
	if len(pe.Consts) != len(wantConsts) {
		t.Fatalf("const count = %d, want %d", len(pe.Consts), len(wantConsts))
	}
	for i, w := range wantConsts {
		if pe.Consts[i].GoName != w.name || pe.Consts[i].Value != w.value {
			t.Errorf("const %d = {%s %q}, want {%s %q}", i, pe.Consts[i].GoName, pe.Consts[i].Value, w.name, w.value)
		}
	}
}

// TestPlanBindingHyphenatedCodes pins the identifier lifting for codes carrying
// hyphens and a leading-digit token, the shapes the FHIR code set actually uses.
func TestPlanBindingHyphenatedCodes(t *testing.T) {
	const url = "vs://account-status"
	r := fakeResolver{
		names:   map[string]string{url: "AccountStatus"},
		codes:   map[string][]string{url: {"active", "entered-in-error", "on-hold"}},
		inlined: map[string]bool{url: true},
	}
	pe := PlanBinding(url, "AccountStatus", r)
	got := map[string]string{}
	for _, c := range pe.Consts {
		got[c.Value] = c.GoName
	}
	if got["entered-in-error"] != "AccountStatusEnteredInError" {
		t.Errorf("entered-in-error -> %q, want AccountStatusEnteredInError", got["entered-in-error"])
	}
	if got["on-hold"] != "AccountStatusOnHold" {
		t.Errorf("on-hold -> %q, want AccountStatusOnHold", got["on-hold"])
	}
}

// TestPlanBindingComparatorCodes pins the punctuation-only comparator codes (the
// Quantity comparator value set), which have no alphanumeric identifier fragment and
// must still produce stable, valid const names rather than empty or numeric ones.
func TestPlanBindingComparatorCodes(t *testing.T) {
	const url = "vs://quantity-comparator"
	r := fakeResolver{
		names:   map[string]string{url: "QuantityComparator"},
		codes:   map[string][]string{url: {"<", "<=", ">=", ">"}},
		inlined: map[string]bool{url: true},
	}
	pe := PlanBinding(url, "QuantityComparator", r)
	want := []string{
		"QuantityComparatorLessThan",
		"QuantityComparatorLessOrEqual",
		"QuantityComparatorGreaterOrEqual",
		"QuantityComparatorGreaterThan",
	}
	if len(pe.Consts) != len(want) {
		t.Fatalf("const count = %d, want %d", len(pe.Consts), len(want))
	}
	for i, w := range want {
		if pe.Consts[i].GoName != w {
			t.Errorf("const %d = %q, want %q", i, pe.Consts[i].GoName, w)
		}
	}
}

// TestPlanBindingFilterIsNotInlined pins that a filter-defined (intensional) binding
// the resolver cannot enumerate becomes a documented not-inlined boundary, never a
// closed enum and never an empty const set. This is the filter-based-binding case the
// loader's Filter capture (F1-B) feeds.
func TestPlanBindingFilterIsNotInlined(t *testing.T) {
	const url = "http://hl7.org/fhir/ValueSet/example-intensional"
	const reason = "defined intensionally by a compose.include.filter (parent = \"LP43571-6\" on http://loinc.org)"
	r := fakeResolver{
		names:   map[string]string{url: "LOINCCodesForCholesterolInSerumPlasma"},
		inlined: map[string]bool{url: false},
		reasons: map[string]string{url: reason},
	}
	pe := PlanBinding(url, "LOINCCodesForCholesterolInSerumPlasma", r)
	if !pe.NotInlined {
		t.Fatal("filter-defined binding planned as a closed enum, want not-inlined")
	}
	if len(pe.Consts) != 0 {
		t.Errorf("not-inlined binding has %d consts, want 0", len(pe.Consts))
	}
	if pe.NotInlinedReason != reason {
		t.Errorf("reason = %q, want %q", pe.NotInlinedReason, reason)
	}
}

// TestPlanBindingEmptyDowngradesToNotInlined pins the empty-const-set invariant at the
// planner level: a resolver that claims a set is inlineable but yields no codes is
// downgraded to a documented not-inlined boundary rather than an empty closed enum, so
// an enumerable-but-empty enum is unrepresentable in the output.
func TestPlanBindingEmptyDowngradesToNotInlined(t *testing.T) {
	const url = "vs://empty"
	r := fakeResolver{
		names:   map[string]string{url: "EmptyBinding"},
		codes:   map[string][]string{url: {}},
		inlined: map[string]bool{url: true},
	}
	pe := PlanBinding(url, "EmptyBinding", r)
	if !pe.NotInlined {
		t.Fatal("an inlineable-but-empty binding became a closed enum; want not-inlined downgrade")
	}
	if pe.IsEmpty() {
		t.Fatal("downgraded binding still reports IsEmpty (NotInlined must suppress it)")
	}
}
