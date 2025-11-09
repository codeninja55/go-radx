package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeTerminologyCapabilities is the FHIR resource type name for TerminologyCapabilities.
const ResourceTypeTerminologyCapabilities = "TerminologyCapabilities"

// TerminologyCapabilitiesSoftware represents a FHIR BackboneElement for TerminologyCapabilities.software.
type TerminologyCapabilitiesSoftware struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A name the software is known by
	Name string `json:"name"`
	// Version covered by this statement
	Version *string `json:"version,omitempty"`
}

// TerminologyCapabilitiesImplementation represents a FHIR BackboneElement for TerminologyCapabilities.implementation.
type TerminologyCapabilitiesImplementation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Describes this specific instance
	Description string `json:"description"`
	// Base URL for the implementation
	URL *string `json:"url,omitempty"`
}

// TerminologyCapabilitiesCodeSystemVersionFilter represents a FHIR BackboneElement for TerminologyCapabilities.codeSystem.version.filter.
type TerminologyCapabilitiesCodeSystemVersionFilter struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Code of the property supported
	Code string `json:"code"`
	// Operations supported for the property
	Op []string `json:"op,omitempty"`
}

// TerminologyCapabilitiesCodeSystemVersion represents a FHIR BackboneElement for TerminologyCapabilities.codeSystem.version.
type TerminologyCapabilitiesCodeSystemVersion struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Version identifier for this version
	Code *string `json:"code,omitempty"`
	// If this is the default version for this code system
	IsDefault *bool `json:"isDefault,omitempty"`
	// If compositional grammar is supported
	Compositional *bool `json:"compositional,omitempty"`
	// Language Displays supported
	Language []string `json:"language,omitempty"`
	// Filter Properties supported
	Filter []TerminologyCapabilitiesCodeSystemVersionFilter `json:"filter,omitempty"`
	// Properties supported for $lookup
	Property []string `json:"property,omitempty"`
}

// TerminologyCapabilitiesCodeSystem represents a FHIR BackboneElement for TerminologyCapabilities.codeSystem.
type TerminologyCapabilitiesCodeSystem struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Canonical identifier for the code system, represented as a URI
	URI *string `json:"uri,omitempty"`
	// Version of Code System supported
	Version []TerminologyCapabilitiesCodeSystemVersion `json:"version,omitempty"`
	// not-present | example | fragment | complete | supplement
	Content string `json:"content"`
	// Whether subsumption is supported
	Subsumption *bool `json:"subsumption,omitempty"`
}

// TerminologyCapabilitiesExpansionParameter represents a FHIR BackboneElement for TerminologyCapabilities.expansion.parameter.
type TerminologyCapabilitiesExpansionParameter struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Name of the supported expansion parameter
	Name string `json:"name"`
	// Description of support for parameter
	Documentation *string `json:"documentation,omitempty"`
}

// TerminologyCapabilitiesExpansion represents a FHIR BackboneElement for TerminologyCapabilities.expansion.
type TerminologyCapabilitiesExpansion struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Whether the server can return nested value sets
	Hierarchical *bool `json:"hierarchical,omitempty"`
	// Whether the server supports paging on expansion
	Paging *bool `json:"paging,omitempty"`
	// Allow request for incomplete expansions?
	Incomplete *bool `json:"incomplete,omitempty"`
	// Supported expansion parameter
	Parameter []TerminologyCapabilitiesExpansionParameter `json:"parameter,omitempty"`
	// Documentation about text searching works
	TextFilter *string `json:"textFilter,omitempty"`
}

// TerminologyCapabilitiesValidateCode represents a FHIR BackboneElement for TerminologyCapabilities.validateCode.
type TerminologyCapabilitiesValidateCode struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Whether translations are validated
	Translations bool `json:"translations"`
}

// TerminologyCapabilitiesTranslation represents a FHIR BackboneElement for TerminologyCapabilities.translation.
type TerminologyCapabilitiesTranslation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Whether the client must identify the map
	NeedsMap bool `json:"needsMap"`
}

// TerminologyCapabilitiesClosure represents a FHIR BackboneElement for TerminologyCapabilities.closure.
type TerminologyCapabilitiesClosure struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// If cross-system closure is supported
	Translation *bool `json:"translation,omitempty"`
}

// TerminologyCapabilities represents a FHIR TerminologyCapabilities.
type TerminologyCapabilities struct {
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
	// Canonical identifier for this terminology capabilities, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the terminology capabilities
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the terminology capabilities
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this terminology capabilities (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this terminology capabilities (human friendly)
	Title *string `json:"title,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// Date last changed
	Date primitives.DateTime `json:"date"`
	// Name of the publisher/steward (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the terminology capabilities
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for terminology capabilities (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this terminology capabilities is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// instance | capability | requirements
	Kind string `json:"kind"`
	// Software that is covered by this terminology capability statement
	Software *TerminologyCapabilitiesSoftware `json:"software,omitempty"`
	// If this describes a specific instance
	Implementation *TerminologyCapabilitiesImplementation `json:"implementation,omitempty"`
	// Whether lockedDate is supported
	LockedDate *bool `json:"lockedDate,omitempty"`
	// A code system supported by the server
	CodeSystem []TerminologyCapabilitiesCodeSystem `json:"codeSystem,omitempty"`
	// Information about the [ValueSet/$expand](valueset-operation-expand.html) operation
	Expansion *TerminologyCapabilitiesExpansion `json:"expansion,omitempty"`
	// in-compose | in-expansion | in-compose-or-expansion
	CodeSearch *string `json:"codeSearch,omitempty"`
	// Information about the [ValueSet/$validate-code](valueset-operation-validate-code.html) operation
	ValidateCode *TerminologyCapabilitiesValidateCode `json:"validateCode,omitempty"`
	// Information about the [ConceptMap/$translate](conceptmap-operation-translate.html) operation
	Translation *TerminologyCapabilitiesTranslation `json:"translation,omitempty"`
	// Information about the [ConceptMap/$closure](conceptmap-operation-closure.html) operation
	Closure *TerminologyCapabilitiesClosure `json:"closure,omitempty"`
}
