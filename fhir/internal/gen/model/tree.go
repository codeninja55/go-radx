package model

// Type is one classified StructureDefinition with its element-path tree fully
// built and recursed. It is the unit the planner consumes: a primitive, a complex
// datatype, or a resource, plus the root Element whose Children form the tree.
type Type struct {
	// Name is the FHIR type name (for example "Observation" or "Period").
	Name string

	// URL is the StructureDefinition's canonical URL, kept so later stages can
	// resolve cross-type references without re-indexing.
	URL string

	// Kind is the meta-kind decided by Classify.
	Kind Kind

	// Abstract reports the StructureDefinition's abstract flag (for example
	// Resource and DomainResource are abstract bases).
	Abstract bool

	// Base is the canonical URL of the StructureDefinition this one specialises
	// (its baseDefinition), or empty for a root such as Element. Carried verbatim
	// from the bundle; the model does not resolve inheritance.
	Base string

	// Root is the tree's root element (the element whose path is the type Name
	// itself). Its Children are the type's top-level elements; their Children are
	// nested backbones recursed all the way down.
	Root *Element
}

// Cardinality is an element's min/max occurrence constraint, carried verbatim from
// the StructureDefinition so the planner can decide pointer-versus-slice without
// re-reading the raw record. Max is the raw FHIR token ("0", "1", or "*") rather
// than a number, because "*" has no integer value and the planner distinguishes
// the unbounded case explicitly.
type Cardinality struct {
	Min int
	Max string
}

// Required reports whether at least one occurrence is mandated (min >= 1).
func (c Cardinality) Required() bool { return c.Min >= 1 }

// Repeats reports whether more than one occurrence is allowed (max "*" or a
// numeric maximum above one), which the planner maps to a Go slice.
func (c Cardinality) Repeats() bool {
	switch c.Max {
	case "", "0", "1":
		return false
	default:
		return true
	}
}

// TypeRef is one allowed type of an element, referencing a FHIR type by its code
// (for example "string", "CodeableConcept", "Reference"). Profiles and target
// profiles are kept verbatim so a later stage can apply datatype profiling and
// constrain Reference targets; the model does not resolve them.
type TypeRef struct {
	// Code is the FHIR type code. For a primitive-valued system type FHIR encodes
	// the code as a URL ("http://hl7.org/fhirpath/System.String"); the model
	// normalises it to the FHIR primitive name (see SystemPrimitive).
	Code string

	// Profiles are the canonical URLs of profiles applied to this type itself (for
	// example Range.low is a Quantity profiled as SimpleQuantity). Empty for an
	// unprofiled type.
	Profiles []string

	// TargetProfiles are the allowed referents of a Reference or CodeableReference,
	// as canonical URLs. Empty for non-reference types.
	TargetProfiles []string
}

// Binding is a coded element's value-set binding: the strength and the value set's
// canonical URL with any "|<version>" suffix stripped, so a later stage resolves
// it against the loader's value-set index. The required strength is what drives the
// generated closed enums; weaker strengths stay open strings.
type Binding struct {
	Strength string
	ValueSet string
}

// Required reports whether the binding strength is "required", the only strength
// that mandates a closed enum.
func (b Binding) Required() bool { return b.Strength == "required" }

// Element is one node in the element-path tree. A leaf carries its types and
// metadata; a backbone element additionally carries Children recursed from the
// snapshot (or grafted from a resolved contentReference). The root Element of a
// Type has the type's own name as its Name and Path.
type Element struct {
	// Name is the final path segment (for example "low" for
	// "Observation.referenceRange.low"). For a choice element the "[x]" suffix is
	// retained in Name and Path so the planner can detect and strip it; the
	// stripped base is available via ChoiceBase.
	Name string

	// Path is the full dotted occurrence path of the element in this tree (for
	// example "Observation.component.referenceRange.low"). After a contentReference
	// graft the grafted children are rebased onto the occurrence path so Path always
	// reflects where the element sits in this tree, not the donor path the
	// StructureDefinition defined it under.
	Path string

	// Cardinality is the element's occurrence constraint.
	Cardinality Cardinality

	// Types is the element's allowed type set. A single entry is a plain element; a
	// "BackboneElement" entry marks a nested backbone whose shape is its Children;
	// more than one entry on a "[x]" element makes it a choice (see IsChoice).
	Types []TypeRef

	// Binding is the value-set binding for a coded element, or nil if unbound.
	Binding *Binding

	// IsSummary records the StructureDefinition's isSummary flag, consumed by the
	// summary-mode serialiser.
	IsSummary bool

	// IsModifier records the StructureDefinition's isModifier flag.
	IsModifier bool

	// IsChoice reports whether this is a polymorphic "[x]" element. When true,
	// Types holds the branch types and ChoiceBase holds the name without the "[x]".
	IsChoice bool

	// ChoiceBase is the element name with the "[x]" suffix removed (for example
	// "value" for "value[x]"), set only when IsChoice is true.
	ChoiceBase string

	// ContentReference is the resolved local anchor an element reuses for its child
	// structure (for example "#Observation.referenceRange"), kept for traceability
	// after BuildType grafts the referenced children onto Children. Empty when the
	// element defines its own children inline or is a leaf.
	ContentReference string

	// Children are the nested elements one level below this one, recursed all the
	// way down. Empty for a leaf element.
	Children []*Element
}

// IsBackbone reports whether the element is a nested anonymous structure whose
// shape is its Children, identified by a "BackboneElement" or "Element" type code.
// A backbone with no Children after building is the empty-backbone defect the
// model exists to prevent.
func (e *Element) IsBackbone() bool {
	for _, t := range e.Types {
		if t.Code == "BackboneElement" || t.Code == "Element" {
			return true
		}
	}
	// An element with no declared type but resolved children (a contentReference
	// target before its own type is restated) is also structural.
	return len(e.Types) == 0 && len(e.Children) > 0
}

// Child returns the direct child with the given final-segment name and whether it
// exists, a convenience for tests and later stages walking the tree.
func (e *Element) Child(name string) (*Element, bool) {
	for _, c := range e.Children {
		if c.Name == name {
			return c, true
		}
	}
	return nil, false
}
