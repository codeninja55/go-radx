package fhir

// Validate runs go-radx's structural and binding-level validation over a resource of
// any release and returns the issues it finds as an *OperationOutcome. It is the fast
// in-process gate (the HL7 FHIR validator is the authoritative external check): it
// verifies resourceType integrity, required-element presence by presence rather than
// truthiness (a present required false or 0 is valid, never reported missing —
// Codex FHIR-007), choice-group mutual exclusion (at most one [x] branch set, caught
// even when a caller wrote two suffixed storage fields directly rather than through
// the setters — Codex FHIR-001), required value-set binding codes, primitive lexical
// validity, the Bundle bdl-* invariants, and intra-Bundle/contained reference
// integrity. It reports every issue it finds rather than stopping at the first.
//
// Validate never panics on malformed or partial input (PRD §9.3): a nil resource, an
// unregistered resourceType, and a structurally broken value each yield an
// OperationOutcome with an issue, never a crash. It never leaks PHI: every issue names
// an element, a path, a resource type, or a code, never a patient value (PRD §9.1).
//
// Validation is data-driven by a per-resource descriptor the generator emits and each
// release package registers at init time, so there is no per-call StructureDefinition
// lookup and no metadata reflection on the validation path: the descriptor carries the
// required paths, the choice groups, and the required bindings as typed closures over
// the concrete resource. A resource whose type has no registered descriptor (a
// hand-written type not yet covered) is reported as a single unvalidated-type issue
// rather than silently passing, so the gap is visible instead of a false "valid".
func (reg *Registry) Validate(r Resource) *OperationOutcome {
	outcome := &OperationOutcome{}
	if r == nil || isNilResource(r) {
		outcome.add(OutcomeIssue{
			Severity:    SeverityError,
			Code:        IssueTypeStructure,
			Diagnostics: "resource is nil",
		})
		return outcome
	}

	resourceType := r.ResourceType()
	if resourceType == "" {
		outcome.add(OutcomeIssue{
			Severity:    SeverityError,
			Code:        IssueTypeStructure,
			Diagnostics: "resource has no resourceType",
		})
		return outcome
	}

	descriptor, ok := reg.lookupValidationDescriptor(resourceType)
	if !ok {
		outcome.add(OutcomeIssue{
			Severity:    SeverityWarning,
			Code:        IssueTypeStructure,
			Diagnostics: "no validation descriptor registered for resourceType " + resourceType,
			Expression:  resourceType,
		})
		return outcome
	}

	descriptor.validate(r, resourceType, outcome)
	return outcome
}

// Validate runs go-radx's structural and binding-level validation through the root
// package's default registry. It is the release-agnostic counterpart to
// (*Registry).Validate; a consumer validates a specific release's resource through
// that release's r4.Validate or r5.Validate so the descriptor lookup is unambiguous.
func Validate(r Resource) *OperationOutcome {
	return defaultRegistry.Validate(r)
}

// ValidationDescriptor is the generated, per-resource validation metadata the engine
// consumes. Each release package emits one descriptor per concrete resource and
// registers it at init time, keyed by the resource's resourceType. Every check is a
// typed closure over the concrete resource rather than a reflective field walk, so the
// validation path takes no metadata reflection and no spec lookup: the generator has
// already resolved which elements are required, which form a choice group, and which
// carry a required binding, and emits a closure that reads exactly those fields.
//
// A closure is handed the resource as the Resource interface and asserts it to the
// concrete type once; the engine never inspects field shapes itself. The closures are
// generated, so they always match the resource they are registered for.
type ValidationDescriptor struct {
	// Required reports the required elements that are absent on the resource. Each
	// returned path names a missing required element (for example "Patient.name");
	// presence is tested by the concrete field being non-nil or a non-empty slice,
	// never by the value being non-zero, so a present required false or 0 is not
	// reported (Codex FHIR-007). The engine emits one required-issue per returned path.
	Required func(r Resource) []string

	// Choices reports the choice ([x]) groups that have more than one branch set. Each
	// returned path names a violated group (for example "Patient.deceased[x]"); the
	// closure counts the non-nil suffixed storage fields of each group and returns the
	// group's path when the count exceeds one, catching a direct two-field write the
	// mutually-exclusive setters would have prevented (Codex FHIR-001). The engine
	// emits one structure-issue per returned path.
	Choices func(r Resource) []string

	// Bindings reports the required-binding code fields whose value is outside the
	// bound value set. Each returned BindingIssue names the element path and the kind
	// of violation; the closure validates each required-binding code against its
	// generated closed enum, so an out-of-set code retained under lenient decode is
	// surfaced (Codex FHIR-013). The engine emits one value-issue per returned issue.
	Bindings func(r Resource) []BindingIssue

	// Primitives reports the date/time-family primitive fields (date, dateTime, time,
	// instant) whose present value violates the release's lexical rules — the official
	// FHIR primitive regexes plus the offset-with-time prose rule (Codex FHIR-008). The
	// closure checks only values that are present; absence is a cardinality concern,
	// not a lexical one. Each issue names the element path and the primitive type,
	// never the offending value, because a date can itself be PHI (a birth date). The
	// engine emits one value-issue per returned issue.
	Primitives func(r Resource) []PrimitiveIssue

	// Extra runs any resource-specific structural checks that are not expressible as
	// required/choice/binding metadata, appending their issues directly. The Bundle
	// descriptor uses it to compose the bdl-* invariants and the reference-integrity
	// walk (both hand-written per release because they encode FHIR prose rules the
	// StructureDefinition does not). It is nil for a resource with no extra checks.
	Extra func(r Resource, outcome *OperationOutcome)
}

// BindingIssue is one required-binding violation reported by a descriptor's Bindings
// closure: the element path and the diagnostic naming the binding and the offending
// code. The diagnostic is built from the binding name and the code token, both of
// which are coded values, not patient data, so it carries no PHI.
type BindingIssue struct {
	// Expression is the element path of the offending code field.
	Expression string

	// Diagnostics names the binding and why the code is invalid.
	Diagnostics string
}

// PrimitiveIssue is one primitive lexical violation reported by a descriptor's
// Primitives closure: the element path and the diagnostic naming the primitive type
// whose lexical rules the value breaks. The offending value is deliberately absent —
// a date/dateTime value can itself be PHI (a birth date, a death time), so the issue
// carries only the path and the type name (PRD §9.1).
type PrimitiveIssue struct {
	// Expression is the element path of the offending primitive field.
	Expression string

	// Diagnostics names the primitive type and that its lexical form is invalid.
	Diagnostics string
}

// validate runs the descriptor's checks against r and records every issue. It runs the
// required, choice, binding, and primitive-lexical checks in a fixed order so the issue
// order is deterministic for a given resource, then the resource-specific Extra checks.
// A nil closure (a resource with no required elements, no choices, no required
// bindings, or no date/time-family primitives) is skipped.
func (d ValidationDescriptor) validate(r Resource, resourceType string, outcome *OperationOutcome) {
	if d.Required != nil {
		for _, path := range d.Required(r) {
			outcome.add(OutcomeIssue{
				Severity:    SeverityError,
				Code:        IssueTypeRequired,
				Diagnostics: "required element " + path + " is missing",
				Expression:  path,
			})
		}
	}
	if d.Choices != nil {
		for _, path := range d.Choices(r) {
			outcome.add(OutcomeIssue{
				Severity:    SeverityError,
				Code:        IssueTypeStructure,
				Diagnostics: "choice element " + path + " has more than one value set; at most one branch is allowed",
				Expression:  path,
			})
		}
	}
	if d.Bindings != nil {
		for _, issue := range d.Bindings(r) {
			outcome.add(OutcomeIssue{
				Severity:    SeverityError,
				Code:        IssueTypeValue,
				Diagnostics: issue.Diagnostics,
				Expression:  issue.Expression,
			})
		}
	}
	if d.Primitives != nil {
		for _, issue := range d.Primitives(r) {
			outcome.add(OutcomeIssue{
				Severity:    SeverityError,
				Code:        IssueTypeValue,
				Diagnostics: issue.Diagnostics,
				Expression:  issue.Expression,
			})
		}
	}
	if d.Extra != nil {
		d.Extra(r, outcome)
	}
}

// RegisterValidationDescriptor records the validation descriptor for a resourceType in
// this registry. It exists for the generated per-release validation init() to call; a
// consumer never calls it directly. It is a method on Registry because the generated
// release package and this root package are distinct packages, so the registration
// hook must cross the package boundary.
//
// It panics on an empty resourceType or a duplicate registration: a duplicate within
// one release means the generator emitted conflicting descriptors, a build-time defect
// that must fail loudly rather than let one descriptor silently shadow the other. Two
// releases no longer collide here because each owns its own Registry.
func (reg *Registry) RegisterValidationDescriptor(resourceType string, d ValidationDescriptor) {
	if resourceType == "" {
		panic("fhir: RegisterValidationDescriptor: empty resourceType")
	}
	reg.validationMu.Lock()
	defer reg.validationMu.Unlock()
	if _, exists := reg.validation[resourceType]; exists {
		panic("fhir: RegisterValidationDescriptor: duplicate descriptor for resourceType " + resourceType)
	}
	reg.validation[resourceType] = d
}

// lookupValidationDescriptor returns the descriptor for a resourceType and whether one
// is registered, under the registry read lock so it never races a registration.
func (reg *Registry) lookupValidationDescriptor(resourceType string) (ValidationDescriptor, bool) {
	reg.validationMu.RLock()
	defer reg.validationMu.RUnlock()
	d, ok := reg.validation[resourceType]
	return d, ok
}

// RegisterValidationDescriptor records a validation descriptor in the root package's
// default registry. It is the release-agnostic counterpart to
// (*Registry).RegisterValidationDescriptor; a release package registers into its own
// Registry instead.
func RegisterValidationDescriptor(resourceType string, d ValidationDescriptor) {
	defaultRegistry.RegisterValidationDescriptor(resourceType, d)
}

// CountSet returns how many of the given presence flags are true. The generated
// choice-group check passes one flag per suffixed storage field (each "field != nil"),
// so a result above one means more than one branch of a [x] group is set — the
// mutual-exclusion violation a direct field write can introduce that the typed setters
// prevent (Codex FHIR-001). Keeping the count here means the generated descriptor emits
// a single readable call rather than an open-coded chain.
func CountSet(flags ...bool) int {
	n := 0
	for _, f := range flags {
		if f {
			n++
		}
	}
	return n
}

// AddIssue appends a structural issue to an outcome from a descriptor's Extra closure.
// It is the boundary helper the generated/hand-written Extra checks call so a
// resource-specific check (the Bundle bdl-* invariants, the reference walk) records an
// issue through the same path as the engine's own checks, keeping the OperationOutcome
// shape uniform. The caller supplies an element name, a path, a type, or a code in
// diagnostics, never a patient value.
func AddIssue(outcome *OperationOutcome, severity IssueSeverity, code IssueType, expression, diagnostics string) {
	outcome.add(OutcomeIssue{
		Severity:    severity,
		Code:        code,
		Diagnostics: diagnostics,
		Expression:  expression,
	})
}
