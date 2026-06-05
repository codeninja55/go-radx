package model

import "github.com/codeninja55/go-radx/fhir/internal/gen/loader"

// Kind classifies a StructureDefinition by the FHIR meta-kind that decides how the
// later stages treat it: a primitive type becomes a Go scalar, a complex type a
// Go struct datatype, and a resource a top-level resource type with a
// resourceType discriminator. The classification is the FHIR "kind" plus the
// abstract flag, mapped to the generator's vocabulary.
type Kind int

const (
	// KindUnknown is the zero value, returned for a StructureDefinition whose FHIR
	// kind the generator does not model (for example "logical").
	KindUnknown Kind = iota

	// KindPrimitive is a FHIR primitive-type (boolean, integer, string, dateTime,
	// decimal, ...): a single-valued type the planner maps to a Go scalar.
	KindPrimitive

	// KindComplexType is a FHIR complex-type (Period, Identifier, CodeableConcept,
	// ...): a reusable datatype the planner maps to a Go struct.
	KindComplexType

	// KindResource is a FHIR resource (Patient, Observation, Bundle, ...): a
	// top-level type carrying a resourceType discriminator.
	KindResource
)

// String returns the generator's name for the kind, matching the FHIR vocabulary
// where it lines up, for readable diagnostics and golden snapshots.
func (k Kind) String() string {
	switch k {
	case KindPrimitive:
		return "primitive-type"
	case KindComplexType:
		return "complex-type"
	case KindResource:
		return "resource"
	default:
		return "unknown"
	}
}

// Classify maps a StructureDefinition's FHIR "kind" to the generator's Kind. It
// reads only sd.Kind, so it is release-agnostic: the same mapping holds for R4 and
// R5. A kind the generator does not model returns KindUnknown rather than guessing.
func Classify(sd *loader.StructureDefinition) Kind {
	if sd == nil {
		return KindUnknown
	}
	switch sd.Kind {
	case "primitive-type":
		return KindPrimitive
	case "complex-type":
		return KindComplexType
	case "resource":
		return KindResource
	default:
		return KindUnknown
	}
}
