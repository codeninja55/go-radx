package plan

import (
	"sort"
	"strings"

	"github.com/codeninja55/go-radx/fhir/internal/gen/model"
)

// BindingResolver resolves a value-set canonical URL to the closed code set a
// required binding enumerates. It is implemented by the generator stage that holds
// the loaded bundle (the planner itself reads no I/O), so the planner stays a pure
// function of the model plus the resolved codes.
//
// ResolveBinding returns the value set's enumerable codes and whether the set is
// inlineable from the bundle alone. inlined is false when the value set is defined
// intensionally (a compose.include.filter), references another value set, or draws
// from a code system not vendored in the bundle (an external terminology such as
// LOINC, SNOMED CT, UCUM, or an IETF/ISO registry): the generator cannot enumerate
// such a set without a terminology server, so the enum stage emits a documented
// not-inlined boundary rather than a silently-empty const set. When inlined is true,
// codes is the closed set in value-set order, deduplicated.
//
// ResolveValueSetName returns the bound value set's FHIR name (from which the enum's
// Go type name is derived) and whether the value set is present in the bundle; a
// binding whose value set is absent gets no enum and the field stays a plain code
// string.
type BindingResolver interface {
	ResolveBinding(valueSetURL string) (codes []string, inlined bool, reason string)
	ResolveValueSetName(valueSetURL string) (name string, ok bool)
}

// PlannedEnum is the emitter-ready plan for one required-strength value-set binding.
// An inlineable binding becomes a closed Go enum (a defined string type, a const set,
// a validating ParseXxx, and a strict/lenient UnmarshalJSON); a non-inlineable one
// becomes a documented open string type (NotInlined), so a field bound to it stays a
// plain code string and the terminology boundary is explicit in the generated source
// rather than hidden behind an empty const set.
type PlannedEnum struct {
	// GoName is the enum's Go type identifier, the value set's FHIR name lifted to an
	// exported identifier (for example "AdministrativeGender").
	GoName string

	// FHIRName is the source value-set name, kept for the godoc summary.
	FHIRName string

	// ValueSetURL is the bound value set's canonical URL (version stripped), kept for
	// the godoc summary and traceability.
	ValueSetURL string

	// Consts are the enumerated codes, each a Go const identifier and its wire value,
	// in value-set order. Empty when NotInlined is true.
	Consts []EnumConst

	// NotInlined reports that the value set could not be enumerated from the bundle
	// (intensional, external-system, or value-set-reference). The emitter renders the
	// type as a documented open string with no const set and no validating parser,
	// never an empty closed enum.
	NotInlined bool

	// NotInlinedReason names why the set is not enumerable (for example "defined by a
	// compose.include.filter" or "draws from code system http://loinc.org, not
	// vendored in the bundle"), surfaced in the generated type's godoc so the boundary
	// is documented in the source.
	NotInlinedReason string
}

// IsEmpty reports whether an inlineable enum would ship with no constants, the exact
// failure the not-inlined boundary exists to prevent. A PlannedEnum should never be
// both enumerable (NotInlined false) and empty; the guard test asserts this invariant
// across the whole generated set.
func (e PlannedEnum) IsEmpty() bool { return !e.NotInlined && len(e.Consts) == 0 }

// EnumConst is one code in a closed enum: the Go const identifier and the wire value.
type EnumConst struct {
	// GoName is the const identifier, the enum type name concatenated with the code
	// token lifted to an exported identifier (for example "AdministrativeGenderMale"
	// for code "male"). Prefixing with the type name keeps the const set collision-free
	// across the package's many enums without a hand-maintained prefix table.
	GoName string

	// Value is the wire code exactly as it appears in the value set (for example
	// "male"), the string the defined type compares against on decode.
	Value string
}

// PlanBinding turns one required-strength binding into a PlannedEnum, resolving its
// codes through the resolver. A binding the resolver cannot enumerate becomes a
// documented not-inlined open string; an enumerable one becomes a closed enum with a
// collision-free const set in value-set order. PlanBinding is a pure function of the
// binding and the resolved codes; it reads no I/O.
//
// PlanBinding never returns an enumerable PlannedEnum with an empty const set: if the
// resolver reports the set inlineable but yields no codes, the binding is downgraded
// to not-inlined with a reason, so an empty closed enum is unrepresentable in the
// output (the guard test pins this).
func PlanBinding(valueSetURL, valueSetName string, resolver BindingResolver) PlannedEnum {
	goName := GoTypeName(valueSetName)
	pe := PlannedEnum{
		GoName:      goName,
		FHIRName:    valueSetName,
		ValueSetURL: valueSetURL,
	}

	codes, inlined, reason := resolver.ResolveBinding(valueSetURL)
	if !inlined || len(codes) == 0 {
		pe.NotInlined = true
		pe.NotInlinedReason = reason
		if reason == "" {
			pe.NotInlinedReason = "value set is not enumerable from the vendored bundle"
		}
		return pe
	}

	used := map[string]bool{goName: true} // reserve the type name so no const shadows it
	for _, code := range dedupeStable(codes) {
		pe.Consts = append(pe.Consts, EnumConst{
			GoName: resolveCollision(goName+enumCodeIdentifier(code), used),
			Value:  code,
		})
	}
	return pe
}

// bindingEnumName returns the Go enum type name for a required binding, or empty when
// the element carries no required binding. It is the single authority for "does this
// element get an enum type", read by the field planner to decide a code field's Go
// type and by the type planner to collect the distinct enums a file references.
func bindingEnumName(e *model.Element, resolver BindingResolver) (string, bool) {
	if resolver == nil || e.Binding == nil || !e.Binding.Required() || e.Binding.ValueSet == "" {
		return "", false
	}
	if !isCodeTyped(e) {
		return "", false
	}
	if _, ok := resolver.ResolveValueSetName(e.Binding.ValueSet); !ok {
		return "", false
	}
	codes, inlined, _ := resolver.ResolveBinding(e.Binding.ValueSet)
	if !inlined || len(codes) == 0 {
		// A not-inlined binding keeps the plain code string: there is no closed enum to
		// type the field with, and inventing an empty one is exactly the silently-empty
		// const set the boundary forbids.
		return "", false
	}
	name, _ := resolver.ResolveValueSetName(e.Binding.ValueSet)
	return GoTypeName(name), true
}

// isCodeTyped reports whether an element's value is a bare FHIR "code" primitive, the
// only datatype a required-binding enum types directly. A Coding or CodeableConcept
// binding constrains a nested code but the field itself stays the complex datatype, so
// only a single-typed "code" element is lifted to the enum.
func isCodeTyped(e *model.Element) bool {
	return len(e.Types) == 1 && e.Types[0].Code == "code"
}

// enumCodeIdentifier lifts a value-set code token to an exported Go identifier
// fragment. Codes use characters Go identifiers cannot ("entered-in-error", "<",
// "_LinkType"), so the token is split on non-alphanumeric runs, each run title-cased,
// and a leading digit prefixed with an underscore-free "N" so the result is always a
// valid exported identifier. The mapping is deterministic, which is what keeps the
// const set byte-stable across regenerations.
func enumCodeIdentifier(code string) string {
	var words []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			words = append(words, cur.String())
			cur.Reset()
		}
	}
	for _, r := range code {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			cur.WriteRune(r)
		default:
			flush()
		}
	}
	flush()

	var b strings.Builder
	for _, w := range words {
		if up, ok := commonInitialisms[strings.ToLower(w)]; ok {
			b.WriteString(up)
			continue
		}
		b.WriteString(upperFirst(w))
	}
	out := b.String()
	if out == "" {
		// A code consisting only of punctuation (the comparator codes "<", "<=", ">",
		// ">=", "=") yields no identifier fragment; name it from its FHIR meaning so the
		// const is still a valid, stable identifier.
		return comparatorIdentifier(code)
	}
	if c := out[0]; c >= '0' && c <= '9' {
		return "N" + out
	}
	return out
}

// comparatorIdentifier names the FHIR quantity-comparator codes, which are pure
// punctuation and so have no alphanumeric identifier fragment. The mapping is fixed so
// the generated const for, say, the "<" code is stably "LessThan" rather than an
// empty or numeric-indexed name.
func comparatorIdentifier(code string) string {
	switch code {
	case "<":
		return "LessThan"
	case "<=":
		return "LessOrEqual"
	case ">":
		return "GreaterThan"
	case ">=":
		return "GreaterOrEqual"
	case "=":
		return "Equal"
	case "!=":
		return "NotEqual"
	default:
		return "Code"
	}
}

// dedupeStable removes duplicate codes while preserving first-seen order, so a value
// set that includes the same code from two systems emits the const once and the order
// stays value-set order (not sorted), which the golden pins.
func dedupeStable(codes []string) []string {
	seen := make(map[string]bool, len(codes))
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		if seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	return out
}

// SortPlannedEnums orders a set of planned enums by Go name, so the emitter writes the
// release's enum file in a stable order independent of discovery order.
func SortPlannedEnums(enums []PlannedEnum) {
	sort.Slice(enums, func(i, j int) bool { return enums[i].GoName < enums[j].GoName })
}
