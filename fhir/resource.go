package fhir

import "github.com/codeninja55/go-radx/fhir/primitives"

// Resource is the base type for all FHIR resources.
// All FHIR resources inherit these common fields.
type Resource struct {
	// Extension for ID
	IDExt *primitives.PrimitiveExtension `json:"_id,omitempty" fhir:"cardinality=0..1"`

	// Metadata about the resource
	Meta *Meta `json:"meta,omitempty" fhir:"cardinality=0..1,summary"`

	// Extension for ImplicitRules
	ImplicitRulesExt *primitives.PrimitiveExtension `json:"_implicitRules,omitempty" fhir:"cardinality=0..1"`

	// Extension for Language
	LanguageExt *primitives.PrimitiveExtension `json:"_language,omitempty" fhir:"cardinality=0..1"`

	// The type of resource
	ResourceType string `json:"resourceType"`

	// Logical id of this artifact
	ID *string `json:"id,omitempty" fhir:"cardinality=0..1,summary"`

	// A set of rules under which this content was created
	ImplicitRules *string `json:"implicitRules,omitempty" fhir:"cardinality=0..1,summary"`

	// Language of the resource content
	Language *string `json:"language,omitempty" fhir:"cardinality=0..1"`
}

// DomainResource is the base type for all FHIR domain resources.
// DomainResources are resources that include human-readable narrative.
type DomainResource struct {
	Resource

	// Contained, inline Resources
	Contained []interface{} `json:"contained,omitempty" fhir:"cardinality=0..*"`

	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty" fhir:"cardinality=0..*"`

	// Extensions that cannot be ignored
	ModifierExtension []Extension `json:"modifierExtension,omitempty" fhir:"cardinality=0..*"`

	// Text summary of the resource, for human interpretation
	Text *Narrative `json:"text,omitempty" fhir:"cardinality=0..1"`
}

// Meta represents metadata about a resource.
type Meta struct {
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty" fhir:"cardinality=0..*"`

	// Profiles this resource claims to conform to
	Profile []string `json:"profile,omitempty" fhir:"cardinality=0..*,summary"`

	// Extension for Profile
	ProfileExt []*primitives.PrimitiveExtension `json:"_profile,omitempty" fhir:"cardinality=0..*"`

	// Security Labels applied to this resource
	Security []Coding `json:"security,omitempty" fhir:"cardinality=0..*,summary"`

	// Tags applied to this resource
	Tag []Coding `json:"tag,omitempty" fhir:"cardinality=0..*,summary"`

	// Extension for VersionID
	VersionIDExt *primitives.PrimitiveExtension `json:"_versionId,omitempty" fhir:"cardinality=0..1"`

	// When the resource version last changed
	LastUpdated *primitives.Instant `json:"lastUpdated,omitempty" fhir:"cardinality=0..1,summary"`

	// Extension for LastUpdated
	LastUpdatedExt *primitives.PrimitiveExtension `json:"_lastUpdated,omitempty" fhir:"cardinality=0..1"`

	// Extension for Source
	SourceExt *primitives.PrimitiveExtension `json:"_source,omitempty" fhir:"cardinality=0..1"`

	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty" fhir:"cardinality=0..1"`

	// Version specific identifier
	VersionID *string `json:"versionId,omitempty" fhir:"cardinality=0..1,summary"`

	// Identifies where the resource comes from
	Source *string `json:"source,omitempty" fhir:"cardinality=0..1,summary"`
}

// Narrative contains human-readable text for a resource.
type Narrative struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty" fhir:"cardinality=0..1"`

	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty" fhir:"cardinality=0..*"`

	// generated | extensions | additional | empty
	Status string `json:"status" fhir:"cardinality=1..1,required,enum=generated|extensions|additional|empty"`

	// Extension for Status
	StatusExt *primitives.PrimitiveExtension `json:"_status,omitempty" fhir:"cardinality=0..1"`

	// Limited xhtml content
	Div string `json:"div" fhir:"cardinality=1..1,required"`

	// Extension for Div
	DivExt *primitives.PrimitiveExtension `json:"_div,omitempty" fhir:"cardinality=0..1"`
}

// Extension represents a FHIR extension.
type Extension struct {
	// Additional extensions
	Extension []Extension `json:"extension,omitempty" fhir:"cardinality=0..*"`

	// Extension for URL
	URLExt *primitives.PrimitiveExtension `json:"_url,omitempty" fhir:"cardinality=0..1"`

	// Extension for Value fields
	ValueBooleanExt   *primitives.PrimitiveExtension `json:"_valueBoolean,omitempty" fhir:"cardinality=0..1"`
	ValueIntegerExt   *primitives.PrimitiveExtension `json:"_valueInteger,omitempty" fhir:"cardinality=0..1"`
	ValueStringExt    *primitives.PrimitiveExtension `json:"_valueString,omitempty" fhir:"cardinality=0..1"`
	ValueDecimalExt   *primitives.PrimitiveExtension `json:"_valueDecimal,omitempty" fhir:"cardinality=0..1"`
	ValueUriExt       *primitives.PrimitiveExtension `json:"_valueUri,omitempty" fhir:"cardinality=0..1"`
	ValueUrlExt       *primitives.PrimitiveExtension `json:"_valueUrl,omitempty" fhir:"cardinality=0..1"`
	ValueCanonicalExt *primitives.PrimitiveExtension `json:"_valueCanonical,omitempty" fhir:"cardinality=0..1"`
	ValueCodeExt      *primitives.PrimitiveExtension `json:"_valueCode,omitempty" fhir:"cardinality=0..1"`
	ValueDateExt      *primitives.PrimitiveExtension `json:"_valueDate,omitempty" fhir:"cardinality=0..1"`
	ValueDateTimeExt  *primitives.PrimitiveExtension `json:"_valueDateTime,omitempty" fhir:"cardinality=0..1"`
	ValueTimeExt      *primitives.PrimitiveExtension `json:"_valueTime,omitempty" fhir:"cardinality=0..1"`
	ValueInstantExt   *primitives.PrimitiveExtension `json:"_valueInstant,omitempty" fhir:"cardinality=0..1"`

	// Value types
	ValueDate     *primitives.Date     `json:"valueDate,omitempty" fhir:"cardinality=0..1,choice=value"`
	ValueDateTime *primitives.DateTime `json:"valueDateTime,omitempty" fhir:"cardinality=0..1,choice=value"`
	ValueTime     *primitives.Time     `json:"valueTime,omitempty" fhir:"cardinality=0..1,choice=value"`
	ValueInstant  *primitives.Instant  `json:"valueInstant,omitempty" fhir:"cardinality=0..1,choice=value"`

	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty" fhir:"cardinality=0..1"`

	// Identifies the meaning of the extension
	URL string `json:"url" fhir:"cardinality=1..1,required"`

	// Value of extension - primitive types
	ValueBoolean   *bool    `json:"valueBoolean,omitempty" fhir:"cardinality=0..1,choice=value"`
	ValueInteger   *int     `json:"valueInteger,omitempty" fhir:"cardinality=0..1,choice=value"`
	ValueString    *string  `json:"valueString,omitempty" fhir:"cardinality=0..1,choice=value"`
	ValueDecimal   *float64 `json:"valueDecimal,omitempty" fhir:"cardinality=0..1,choice=value"`
	ValueUri       *string  `json:"valueUri,omitempty" fhir:"cardinality=0..1,choice=value"`
	ValueUrl       *string  `json:"valueUrl,omitempty" fhir:"cardinality=0..1,choice=value"`
	ValueCanonical *string  `json:"valueCanonical,omitempty" fhir:"cardinality=0..1,choice=value"`
	ValueCode      *string  `json:"valueCode,omitempty" fhir:"cardinality=0..1,choice=value"`

	// More complex value types can be added as needed
	// ValueCoding, ValueCodeableConcept, ValueReference, etc.
}

// Coding represents a code defined by a terminology system.
type Coding struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty" fhir:"cardinality=0..1"`

	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty" fhir:"cardinality=0..*"`

	// Identity of the terminology system
	System *string `json:"system,omitempty" fhir:"cardinality=0..1,summary"`

	// Extension for System
	SystemExt *primitives.PrimitiveExtension `json:"_system,omitempty" fhir:"cardinality=0..1"`

	// Version of the system - if relevant
	Version *string `json:"version,omitempty" fhir:"cardinality=0..1,summary"`

	// Extension for Version
	VersionExt *primitives.PrimitiveExtension `json:"_version,omitempty" fhir:"cardinality=0..1"`

	// Symbol in syntax defined by the system
	Code *string `json:"code,omitempty" fhir:"cardinality=0..1,summary"`

	// Extension for Code
	CodeExt *primitives.PrimitiveExtension `json:"_code,omitempty" fhir:"cardinality=0..1"`

	// Representation defined by the system
	Display *string `json:"display,omitempty" fhir:"cardinality=0..1,summary"`

	// Extension for Display
	DisplayExt *primitives.PrimitiveExtension `json:"_display,omitempty" fhir:"cardinality=0..1"`

	// If this coding was chosen directly by the user
	UserSelected *bool `json:"userSelected,omitempty" fhir:"cardinality=0..1,summary"`

	// Extension for UserSelected
	UserSelectedExt *primitives.PrimitiveExtension `json:"_userSelected,omitempty" fhir:"cardinality=0..1"`
}
