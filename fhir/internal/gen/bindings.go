package gen

import (
	"fmt"

	"github.com/codeninja55/go-radx/fhir/internal/gen/loader"
	"github.com/codeninja55/go-radx/fhir/internal/gen/model"
	"github.com/codeninja55/go-radx/fhir/internal/gen/plan"
)

// bindingResolver enumerates a required binding's closed code set from the loaded
// bundle, or classifies why the set cannot be enumerated. It implements
// plan.BindingResolver so the planner can type an enumerable code field as its enum
// while reading no I/O itself.
//
// A value set is enumerable when every compose.include either inlines its concepts or
// names a code system vendored in the bundle with complete content; it is not
// enumerable when an include is defined intensionally by a filter, references another
// value set, or draws from a code system not vendored in the bundle (an external
// terminology such as LOINC, SNOMED CT, UCUM, or an IETF/ISO registry). The generator
// runs no terminology server, so a non-enumerable set becomes a documented open string
// rather than a guessed-at or empty closed enum.
type bindingResolver struct {
	bundle *loader.Bundle
}

// newBindingResolver builds a resolver over the loaded bundle.
func newBindingResolver(bundle *loader.Bundle) *bindingResolver {
	return &bindingResolver{bundle: bundle}
}

// ResolveValueSetName returns the bound value set's FHIR name and whether it is in the
// bundle. A binding whose value set is absent gets no enum.
func (r *bindingResolver) ResolveValueSetName(valueSetURL string) (string, bool) {
	vs, ok := r.bundle.ValueSet(valueSetURL)
	if !ok {
		return "", false
	}
	return vs.Name, vs.Name != ""
}

// ResolveBinding enumerates the value set's codes, or reports why it is not enumerable.
// The codes are returned in value-set order (include order, then concept/code-system
// order); deduplication is left to the planner so the order is preserved. An exclude
// rule removes its inlined concepts from the result, so a value set that includes a
// whole system and excludes a few codes still enumerates correctly.
func (r *bindingResolver) ResolveBinding(valueSetURL string) ([]string, bool, string) {
	vs, ok := r.bundle.ValueSet(valueSetURL)
	if !ok {
		return nil, false, fmt.Sprintf("value set %s is not vendored in the bundle", valueSetURL)
	}
	if vs.Compose == nil || len(vs.Compose.Include) == 0 {
		return nil, false, "value set has no compose.include rules to enumerate"
	}

	var codes []string
	for _, inc := range vs.Compose.Include {
		incCodes, reason := r.enumerateInclude(inc)
		if reason != "" {
			return nil, false, reason
		}
		codes = append(codes, incCodes...)
	}

	codes = r.applyExcludes(codes, vs.Compose.Exclude)
	if len(codes) == 0 {
		return nil, false, "value set enumerates to no codes"
	}
	return codes, true, ""
}

// enumerateInclude resolves one compose.include to its codes, or returns a reason the
// include is not enumerable. An include with a filter or a value-set reference is
// intensional; an include naming only a system relies on that code system being
// vendored with complete content; an include inlining concepts uses them directly.
func (r *bindingResolver) enumerateInclude(inc loader.ValueSetInclude) ([]string, string) {
	if len(inc.Filter) > 0 {
		f := inc.Filter[0]
		return nil, fmt.Sprintf("defined intensionally by a compose.include.filter (%s %s %q on %s)",
			f.Property, f.Op, f.Value, inc.System)
	}
	if len(inc.ValueSet) > 0 {
		return nil, fmt.Sprintf("composed from another value set (%s)", inc.ValueSet[0])
	}
	if len(inc.Concept) > 0 {
		codes := make([]string, 0, len(inc.Concept))
		for _, c := range inc.Concept {
			codes = append(codes, c.Code)
		}
		return codes, ""
	}
	if inc.System == "" {
		return nil, "compose.include names neither a system, concepts, nor a value set"
	}
	cs, ok := r.bundle.CodeSystem(inc.System)
	if !ok {
		return nil, fmt.Sprintf("draws from code system %s, not vendored in the bundle", inc.System)
	}
	codes := codeSystemCodes(cs.Concept)
	if len(codes) == 0 {
		return nil, fmt.Sprintf("code system %s is vendored without enumerable concepts", inc.System)
	}
	return codes, ""
}

// applyExcludes removes any code listed in an exclude rule's inlined concepts from the
// included set, preserving order. A filter-based or system-wide exclude cannot be
// resolved without a terminology server; the value sets in scope use only concept
// excludes, so a non-concept exclude is ignored (the validator gate, not the
// generator, is the authority on exact membership).
func (r *bindingResolver) applyExcludes(codes []string, excludes []loader.ValueSetInclude) []string {
	if len(excludes) == 0 {
		return codes
	}
	excluded := map[string]bool{}
	for _, exc := range excludes {
		for _, c := range exc.Concept {
			excluded[c.Code] = true
		}
	}
	if len(excluded) == 0 {
		return codes
	}
	out := codes[:0]
	for _, c := range codes {
		if !excluded[c] {
			out = append(out, c)
		}
	}
	return out
}

// codeSystemCodes flattens a CodeSystem's concept tree to its codes in pre-order, so a
// hierarchical code system (concepts with child concepts) enumerates every code, not
// just the top level.
func codeSystemCodes(concepts []loader.CodeSystemConcept) []string {
	var codes []string
	for _, c := range concepts {
		codes = append(codes, c.Code)
		codes = append(codes, codeSystemCodes(c.Concept)...)
	}
	return codes
}

// PlannedEnums returns, in stable order, every distinct required-binding enum the
// generated types reference for the bundle. It walks every generated type's model,
// collects each required, code-typed binding's value set, resolves it, and returns one
// PlannedEnum per distinct value set — both the enumerable closed enums and the
// documented not-inlined boundaries, so the emitted enum file is the single authority
// on the release's terminology surface. The enums are sorted by Go name so the file is
// byte-stable.
func PlannedEnums(bundle *loader.Bundle) ([]plan.PlannedEnum, error) {
	resolver := newBindingResolver(bundle)
	seen := map[string]bool{}
	var enums []plan.PlannedEnum

	for _, gt := range GeneratedTypes(bundle) {
		sd, ok := bundle.StructureDefinition(gt.fhirName)
		if !ok {
			continue
		}
		typ, err := model.BuildType(sd)
		if err != nil {
			return nil, fmt.Errorf("fhir/gen: build model for %s: %w", gt.fhirName, err)
		}
		collectEnums(typ.Root, resolver, seen, &enums)
	}

	plan.SortPlannedEnums(enums)
	return enums, nil
}

// collectEnums walks an element subtree and appends one PlannedEnum per distinct
// required, code-typed binding it has not already collected. Both enumerable and
// not-inlined bindings are collected: a not-inlined boundary is still emitted (as a
// documented open string) so the terminology surface is complete and the empty-const
// guard has the full set to check.
func collectEnums(e *model.Element, resolver *bindingResolver, seen map[string]bool, out *[]plan.PlannedEnum) {
	if e.Binding != nil && e.Binding.Required() && e.Binding.ValueSet != "" && isCodeElement(e) {
		if name, ok := resolver.ResolveValueSetName(e.Binding.ValueSet); ok && !seen[name] {
			seen[name] = true
			*out = append(*out, plan.PlanBinding(e.Binding.ValueSet, name, resolver))
		}
	}
	for _, c := range e.Children {
		collectEnums(c, resolver, seen, out)
	}
}

// isCodeElement reports whether an element's value is a bare FHIR "code" primitive, the
// only datatype a required-binding enum types directly (a Coding/CodeableConcept
// binding constrains a nested code but leaves the field the complex datatype).
func isCodeElement(e *model.Element) bool {
	return len(e.Types) == 1 && e.Types[0].Code == "code"
}
