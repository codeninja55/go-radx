package loader

import "encoding/json"

// The records in this file are plain structs mirroring the subset of the FHIR
// JSON the generator's model layer (Increment 2) consumes. They are deliberately
// raw: no normalisation, no tree-building, no Go-name mapping happens here. The
// loader's single responsibility is to read, verify, decode, and index; the model
// layer turns these flat records into the element-path tree.

// bundleEntry is a single Bundle.entry. The resource is kept raw so the loader can
// dispatch on its resourceType. The outer Bundle object is consumed by streaming
// decode (see loadFile), not as a whole struct, so the large resource file is
// never fully materialised in memory at once.
type bundleEntry struct {
	FullURL  string          `json:"fullUrl"`
	Resource json.RawMessage `json:"resource"`
}

// resourceHeader is the minimal peek used to route a raw entry resource to its
// record type without decoding the whole thing twice.
type resourceHeader struct {
	ResourceType string `json:"resourceType"`
}

// StructureDefinition is the raw decoded FHIR StructureDefinition. It carries the
// identity and classification fields plus the snapshot element list the model
// layer recurses over.
type StructureDefinition struct {
	ResourceType   string    `json:"resourceType"`
	ID             string    `json:"id"`
	URL            string    `json:"url"`
	Name           string    `json:"name"`
	Kind           string    `json:"kind"` // primitive-type, complex-type, resource, logical
	Abstract       bool      `json:"abstract"`
	Type           string    `json:"type"`
	BaseDefinition string    `json:"baseDefinition"`
	Snapshot       *Snapshot `json:"snapshot"`
}

// Snapshot is the fully-resolved (flattened) element list of a StructureDefinition.
// The generator builds the model from the snapshot, not the differential, because
// the snapshot carries every inherited element with full paths.
type Snapshot struct {
	Element []ElementDefinition `json:"element"`
}

// ElementDefinition is one element in a snapshot, keyed by its dotted FHIR path
// (for example "Observation.component.referenceRange.low"). The model layer splits
// these paths to build the element tree.
type ElementDefinition struct {
	Path       string          `json:"path"`
	Short      string          `json:"short"`
	Definition string          `json:"definition"`
	Min        int             `json:"min"`
	Max        string          `json:"max"` // "0", "1", "*"
	Type       []ElementType   `json:"type"`
	Binding    *ElementBinding `json:"binding"`
	IsModifier bool            `json:"isModifier"`
	IsSummary  bool            `json:"isSummary"`

	// ContentReference points at another element whose child structure this
	// element reuses (a local "#Element.path" anchor), used for recursive or
	// shared shapes such as CodeSystem.concept.concept and Bundle.entry.link. The
	// model layer resolves it; the loader only captures it so a referencing
	// element is not mistaken for an untyped leaf.
	ContentReference string `json:"contentReference"`

	// Fixed and Pattern capture the element's fixed[x]/pattern[x] constraint as
	// raw JSON. The choice suffix (Fixed"Code", Pattern"CodeableConcept", ...) is
	// not split out here; the model layer refines it. Captured via UnmarshalJSON
	// because the suffix is part of the JSON key.
	Fixed   json.RawMessage `json:"-"`
	Pattern json.RawMessage `json:"-"`
}

// elementDefinitionAlias avoids infinite recursion in ElementDefinition's
// UnmarshalJSON: the alias has no custom unmarshaller, so the standard decoder
// handles the named fields while the wrapper picks the fixed[x]/pattern[x] keys
// out of the same object.
type elementDefinitionAlias ElementDefinition

// UnmarshalJSON decodes the named ElementDefinition fields and additionally
// captures any fixed[x] / pattern[x] key (the suffix varies with the element's
// type) as raw JSON, since those keys cannot be expressed as static struct tags.
func (e *ElementDefinition) UnmarshalJSON(data []byte) error {
	var alias elementDefinitionAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*e = ElementDefinition(alias)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for key, val := range raw {
		switch {
		case len(key) > len("fixed") && key[:len("fixed")] == "fixed":
			e.Fixed = val
		case len(key) > len("pattern") && key[:len("pattern")] == "pattern":
			e.Pattern = val
		}
	}
	return nil
}

// ElementType describes one allowed type of an element. An element with more than
// one ElementType is a choice ("[x]") element; the model layer groups the
// branches.
type ElementType struct {
	Code          string   `json:"code"`
	TargetProfile []string `json:"targetProfile"` // for Reference / CodeableReference targets
	Profile       []string `json:"profile"`
}

// ElementBinding describes a coded element's value-set binding. The model layer
// strips the "|<version>" suffix from ValueSet and resolves it against the value
// set index; required strength drives the generated closed enums.
type ElementBinding struct {
	Strength string `json:"strength"` // required, extensible, preferred, example
	ValueSet string `json:"valueSet"`
}

// ValueSet is the raw decoded FHIR ValueSet. The model layer enumerates the codes
// of a required binding from its compose.include (inline concepts) and, where the
// concepts are not inlined, by following the referenced CodeSystem.
type ValueSet struct {
	ResourceType string           `json:"resourceType"`
	ID           string           `json:"id"`
	URL          string           `json:"url"`
	Name         string           `json:"name"`
	Compose      *ValueSetCompose `json:"compose"`
}

// ValueSetCompose is the ValueSet.compose element: the set of include/exclude
// rules that define the value set's membership.
type ValueSetCompose struct {
	Include []ValueSetInclude `json:"include"`
	Exclude []ValueSetInclude `json:"exclude"`
}

// ValueSetInclude is one compose.include (or exclude) rule. System names the code
// system; Concept inlines specific codes; ValueSet references other value sets.
type ValueSetInclude struct {
	System   string            `json:"system"`
	Concept  []ValueSetConcept `json:"concept"`
	ValueSet []string          `json:"valueSet"`
}

// ValueSetConcept is one inline code within a compose.include.
type ValueSetConcept struct {
	Code    string `json:"code"`
	Display string `json:"display"`
}

// CodeSystem is the raw decoded FHIR CodeSystem. When a required binding's value
// set includes a whole system without inlining concepts, the codes come from the
// matching CodeSystem's concept list.
type CodeSystem struct {
	ResourceType string              `json:"resourceType"`
	ID           string              `json:"id"`
	URL          string              `json:"url"`
	Name         string              `json:"name"`
	Concept      []CodeSystemConcept `json:"concept"`
}

// CodeSystemConcept is one code in a CodeSystem. Concepts nest (Concept holds
// child concepts), so the model layer walks the tree to enumerate every code.
type CodeSystemConcept struct {
	Code    string              `json:"code"`
	Display string              `json:"display"`
	Concept []CodeSystemConcept `json:"concept"`
}
