package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeComposition is the FHIR resource type name for Composition.
const ResourceTypeComposition = "Composition"

// CompositionAttester represents a FHIR BackboneElement for Composition.attester.
type CompositionAttester struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// personal | professional | legal | official
	Mode string `json:"mode"`
	// When the composition was attested
	Time *primitives.DateTime `json:"time,omitempty"`
	// Who attested the composition
	Party *Reference `json:"party,omitempty"`
}

// CompositionRelatesTo represents a FHIR BackboneElement for Composition.relatesTo.
type CompositionRelatesTo struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// replaces | transforms | signs | appends
	Code string `json:"code"`
	// Target of the relationship
	Target any `json:"target"`
}

// CompositionEvent represents a FHIR BackboneElement for Composition.event.
type CompositionEvent struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Code(s) that apply to the event being documented
	Code []CodeableConcept `json:"code,omitempty"`
	// The period covered by the documentation
	Period *Period `json:"period,omitempty"`
	// The event(s) being documented
	Detail []Reference `json:"detail,omitempty"`
}

// CompositionSectionSection represents a FHIR BackboneElement for Composition.section.section.
type CompositionSectionSection struct {
}

// CompositionSection represents a FHIR BackboneElement for Composition.section.
type CompositionSection struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Label for section (e.g. for ToC)
	Title *string `json:"title,omitempty"`
	// Classification of section (recommended)
	Code *CodeableConcept `json:"code,omitempty"`
	// Who and/or what authored the section
	Author []Reference `json:"author,omitempty"`
	// Who/what the section is about, when it is not about the subject of composition
	Focus *Reference `json:"focus,omitempty"`
	// Text summary of the section, for human interpretation
	Text *Narrative `json:"text,omitempty"`
	// working | snapshot | changes
	Mode *string `json:"mode,omitempty"`
	// Order of section entries
	OrderedBy *CodeableConcept `json:"orderedBy,omitempty"`
	// A reference to data that supports this section
	Entry []Reference `json:"entry,omitempty"`
	// Why the section is empty
	EmptyReason *CodeableConcept `json:"emptyReason,omitempty"`
	// Nested Section
	Section []CompositionSectionSection `json:"section,omitempty"`
}

// Composition represents a FHIR Composition.
type Composition struct {
	// Logical id of this artifact
	ID *string `json:"id,omitempty"`
	// Metadata about the resource
	Meta *Meta `json:"meta,omitempty"`
	// A set of rules under which this content was created
	ImplicitRules *string `json:"implicitRules,omitempty"`
	// Language of the resource content
	Language *string `json:"language,omitempty"`
	// Text summary of the resource, for human interpretation
	Text *Narrative `json:"text,omitempty"`
	// Contained, inline Resources
	Contained []any `json:"contained,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Version-independent identifier for the Composition
	Identifier *Identifier `json:"identifier,omitempty"`
	// preliminary | final | amended | entered-in-error
	Status string `json:"status"`
	// Kind of composition (LOINC if possible)
	Type CodeableConcept `json:"type"`
	// Categorization of Composition
	Category []CodeableConcept `json:"category,omitempty"`
	// Who and/or what the composition is about
	Subject *Reference `json:"subject,omitempty"`
	// Context of the Composition
	Encounter *Reference `json:"encounter,omitempty"`
	// Composition editing time
	Date primitives.DateTime `json:"date"`
	// Who and/or what authored the composition
	Author []Reference `json:"author,omitempty"`
	// Human Readable name/title
	Title string `json:"title"`
	// As defined by affinity domain
	Confidentiality *string `json:"confidentiality,omitempty"`
	// Attests to accuracy of composition
	Attester []CompositionAttester `json:"attester,omitempty"`
	// Organization which maintains the composition
	Custodian *Reference `json:"custodian,omitempty"`
	// Relationships to other compositions/documents
	RelatesTo []CompositionRelatesTo `json:"relatesTo,omitempty"`
	// The clinical service(s) being documented
	Event []CompositionEvent `json:"event,omitempty"`
	// Composition is broken into sections
	Section []CompositionSection `json:"section,omitempty"`
}
