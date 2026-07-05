package fhir

import (
	"strings"
	"testing"
)

// validatable is a minimal in-package Resource used to drive the engine without
// importing a release package (which would create an import cycle). Its ResourceType
// is fixed and its fields stand in for a required element, a choice group, and a
// required-binding code so the descriptor closures have something to read.
type validatable struct {
	resourceType string
	status       *string // a required scalar; presence is != nil, not truthiness
	flag         *bool   // a required boolean; a present false is still present
	branchA      *string // choice branch A
	branchB      *string // choice branch B
	code         *string // a required-binding code
	when         *string // a date/time-family primitive with lexical rules
}

func (v *validatable) ResourceType() string { return v.resourceType }

// registerValidatable registers a descriptor for a unique resourceType so each test is
// isolated from the package-global registry (a duplicate registration panics by design).
func registerValidatable(t *testing.T, resourceType string) {
	t.Helper()
	RegisterValidationDescriptor(resourceType, ValidationDescriptor{
		Required: func(r Resource) []string {
			v, ok := r.(*validatable)
			if !ok {
				return nil
			}
			var missing []string
			if v.status == nil {
				missing = append(missing, resourceType+".status")
			}
			if v.flag == nil {
				missing = append(missing, resourceType+".flag")
			}
			return missing
		},
		Choices: func(r Resource) []string {
			v, ok := r.(*validatable)
			if !ok {
				return nil
			}
			var violations []string
			if CountSet(v.branchA != nil, v.branchB != nil) > 1 {
				violations = append(violations, resourceType+".value[x]")
			}
			return violations
		},
		Bindings: func(r Resource) []BindingIssue {
			v, ok := r.(*validatable)
			if !ok {
				return nil
			}
			var issues []BindingIssue
			if v.code != nil && *v.code != "ok" {
				issues = append(issues, BindingIssue{
					Expression:  resourceType + ".code",
					Diagnostics: "code is not in the required Example value set",
				})
			}
			return issues
		},
		Primitives: func(r Resource) []PrimitiveIssue {
			v, ok := r.(*validatable)
			if !ok {
				return nil
			}
			var issues []PrimitiveIssue
			// The stand-in lexical rule: any present value other than "2026-01-01" is
			// invalid, so the engine path is exercised without the release regexes.
			if v.when != nil && *v.when != "2026-01-01" {
				issues = append(issues, PrimitiveIssue{
					Expression:  resourceType + ".when",
					Diagnostics: "value is not a valid FHIR date",
				})
			}
			return issues
		},
	})
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

// TestValidateNilResource reports a nil resource as a structural issue rather than
// panicking, honouring the never-panic-on-malformed-input contract (PRD §9.3).
func TestValidateNilResource(t *testing.T) {
	oo := Validate(nil)
	if oo == nil || len(oo.Issue) != 1 {
		t.Fatalf("expected one issue for a nil resource, got %+v", oo)
	}
	if !oo.HasErrors() {
		t.Error("a nil resource should be an error-severity issue")
	}
}

// TestValidateTypedNilResource handles a typed-nil pointer (a nil *validatable wrapped
// in the interface) without dereferencing it.
func TestValidateTypedNilResource(t *testing.T) {
	var v *validatable
	oo := Validate(v)
	if !oo.HasErrors() {
		t.Fatalf("a typed-nil resource should fail closed, got %+v", oo)
	}
}

// TestValidateUnregisteredType reports a resource whose type has no descriptor as a
// visible warning rather than silently passing, so the coverage gap is surfaced.
func TestValidateUnregisteredType(t *testing.T) {
	oo := Validate(&validatable{resourceType: "NotRegisteredAnywhere"})
	if len(oo.Issue) != 1 {
		t.Fatalf("expected one issue for an unregistered type, got %+v", oo)
	}
	if oo.HasErrors() {
		t.Error("an unregistered type should be a warning, not an error")
	}
}

// TestValidateRequiredAbsent reports every absent required element, not just the first.
func TestValidateRequiredAbsent(t *testing.T) {
	const rt = "ValidatableRequired"
	registerValidatable(t, rt)
	oo := Validate(&validatable{resourceType: rt})
	if len(oo.Issue) != 2 {
		t.Fatalf("expected two required issues (status, flag), got %+v", oo.Issue)
	}
	for _, issue := range oo.Issue {
		if issue.Code != IssueTypeRequired {
			t.Errorf("expected required code, got %q", issue.Code)
		}
		if !strings.HasPrefix(issue.Expression, rt+".") {
			t.Errorf("issue should name an element path, got %q", issue.Expression)
		}
	}
}

// TestValidateRequiredFalseIsPresent is the FHIR-007 behavioural regression: a required
// boolean that is validly false is *present* (a non-nil pointer) and must not be reported
// missing. Presence is tracked by the pointer, never by the value's truthiness.
func TestValidateRequiredFalseIsPresent(t *testing.T) {
	const rt = "ValidatableFalse"
	registerValidatable(t, rt)
	oo := Validate(&validatable{
		resourceType: rt,
		status:       strPtr("active"),
		flag:         boolPtr(false), // present and false — must NOT be reported missing
	})
	for _, issue := range oo.Issue {
		if strings.Contains(issue.Expression, ".flag") {
			t.Errorf("a present required false was reported missing (FHIR-007 regression): %+v", issue)
		}
	}
}

// TestValidateChoiceMutualExclusion is the FHIR-001 regression caught at validation: two
// choice branches set directly (bypassing the mutually-exclusive setters) is one issue.
func TestValidateChoiceMutualExclusion(t *testing.T) {
	const rt = "ValidatableChoice"
	registerValidatable(t, rt)
	oo := Validate(&validatable{
		resourceType: rt,
		status:       strPtr("active"),
		flag:         boolPtr(true),
		branchA:      strPtr("a"),
		branchB:      strPtr("b"), // a second branch set directly
	})
	choiceIssues := 0
	for _, issue := range oo.Issue {
		if strings.Contains(issue.Expression, "value[x]") {
			choiceIssues++
			if issue.Code != IssueTypeStructure {
				t.Errorf("a choice violation should be a structure issue, got %q", issue.Code)
			}
		}
	}
	if choiceIssues != 1 {
		t.Fatalf("expected exactly one choice mutual-exclusion issue, got %d: %+v", choiceIssues, oo.Issue)
	}
}

// TestValidateBindingCode reports an out-of-set required-binding code as a value issue.
func TestValidateBindingCode(t *testing.T) {
	const rt = "ValidatableBinding"
	registerValidatable(t, rt)
	oo := Validate(&validatable{
		resourceType: rt,
		status:       strPtr("active"),
		flag:         boolPtr(true),
		code:         strPtr("banana"), // not "ok"
	})
	found := false
	for _, issue := range oo.Issue {
		if strings.HasSuffix(issue.Expression, ".code") {
			found = true
			if issue.Code != IssueTypeValue {
				t.Errorf("a binding violation should be a value issue, got %q", issue.Code)
			}
		}
	}
	if !found {
		t.Fatalf("expected a binding issue for an out-of-set code, got %+v", oo.Issue)
	}
}

// TestValidateReportsEveryIssue confirms the engine accumulates issues across categories
// rather than stopping at the first failure.
func TestValidateReportsEveryIssue(t *testing.T) {
	const rt = "ValidatableEvery"
	registerValidatable(t, rt)
	oo := Validate(&validatable{
		resourceType: rt,
		// status missing (1 required), flag missing (1 required),
		branchA: strPtr("a"),
		branchB: strPtr("b"),    // choice violation (1)
		code:    strPtr("nope"), // binding violation (1)
	})
	if len(oo.Issue) != 4 {
		t.Fatalf("expected 4 issues across categories, got %d: %+v", len(oo.Issue), oo.Issue)
	}
}

// TestRegisterValidationDescriptorRejectsDuplicate confirms a duplicate registration
// panics, so a generator defect fails loudly rather than silently shadowing.
func TestRegisterValidationDescriptorRejectsDuplicate(t *testing.T) {
	const rt = "ValidatableDuplicate"
	registerValidatable(t, rt)
	defer func() {
		if recover() == nil {
			t.Error("expected a panic on a duplicate descriptor registration")
		}
	}()
	registerValidatable(t, rt)
}

// TestRegisterValidationDescriptorRejectsEmpty confirms an empty resourceType panics.
func TestRegisterValidationDescriptorRejectsEmpty(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected a panic on an empty resourceType")
		}
	}()
	RegisterValidationDescriptor("", ValidationDescriptor{})
}

// TestValidatePrimitiveLexical drives the engine's Primitives path: a present
// date-family value that violates the descriptor's lexical rule is one value-coded
// error naming the path and the primitive type — never the offending value, which can
// itself be PHI — and a valid or absent value reports nothing.
func TestValidatePrimitiveLexical(t *testing.T) {
	const rt = "ValidatablePrimitive"
	registerValidatable(t, rt)
	valid := &validatable{resourceType: rt, status: strPtr("active"), flag: boolPtr(true)}
	if oo := Validate(valid); oo.HasErrors() {
		t.Fatalf("an absent primitive should report nothing, got %+v", oo.Issue)
	}
	valid.when = strPtr("2026-01-01")
	if oo := Validate(valid); oo.HasErrors() {
		t.Fatalf("a valid primitive should report nothing, got %+v", oo.Issue)
	}

	invalid := &validatable{resourceType: rt, status: strPtr("active"), flag: boolPtr(true), when: strPtr("2026-13-01")}
	oo := Validate(invalid)
	if len(oo.Issue) != 1 {
		t.Fatalf("expected one primitive lexical issue, got %+v", oo.Issue)
	}
	issue := oo.Issue[0]
	if issue.Code != IssueTypeValue {
		t.Errorf("a lexical violation should be a value issue, got %q", issue.Code)
	}
	if issue.Expression != rt+".when" {
		t.Errorf("issue should name the element path, got %q", issue.Expression)
	}
	if strings.Contains(issue.Diagnostics, "2026-13-01") {
		t.Errorf("the diagnostic must never echo the value (it can be PHI): %q", issue.Diagnostics)
	}
}

// TestOperationOutcomeErrorIsNilWhenClean confirms a clean outcome maps to a nil Go error
// and an outcome with errors maps to a non-nil one whose message carries no value.
func TestOperationOutcomeErrorIsNilWhenClean(t *testing.T) {
	clean := &OperationOutcome{}
	if clean.Error() != nil {
		t.Error("a clean outcome should map to a nil error")
	}
	dirty := &OperationOutcome{}
	dirty.add(OutcomeIssue{Severity: SeverityError, Code: IssueTypeRequired, Expression: "X.y", Diagnostics: "required element X.y is missing"})
	if dirty.Error() == nil {
		t.Error("an outcome with an error issue should map to a non-nil error")
	}
}
