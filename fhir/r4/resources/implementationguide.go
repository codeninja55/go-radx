package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeImplementationGuide is the FHIR resource type name for ImplementationGuide.
const ResourceTypeImplementationGuide = "ImplementationGuide"

// ImplementationGuideDependsOn represents a FHIR BackboneElement for ImplementationGuide.dependsOn.
type ImplementationGuideDependsOn struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Identity of the IG that this depends on
	URI string `json:"uri"`
	// NPM Package name for IG this depends on
	PackageId *string `json:"packageId,omitempty"`
	// Version of the IG
	Version *string `json:"version,omitempty"`
}

// ImplementationGuideGlobal represents a FHIR BackboneElement for ImplementationGuide.global.
type ImplementationGuideGlobal struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type this profile applies to
	Type string `json:"type"`
	// Profile that all resources must conform to
	Profile string `json:"profile"`
}

// ImplementationGuideDefinitionGrouping represents a FHIR BackboneElement for ImplementationGuide.definition.grouping.
type ImplementationGuideDefinitionGrouping struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Descriptive name for the package
	Name string `json:"name"`
	// Human readable text describing the package
	Description *string `json:"description,omitempty"`
}

// ImplementationGuideDefinitionResource represents a FHIR BackboneElement for ImplementationGuide.definition.resource.
type ImplementationGuideDefinitionResource struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Location of the resource
	Reference Reference `json:"reference"`
	// Versions this applies to (if different to IG)
	FhirVersion []string `json:"fhirVersion,omitempty"`
	// Human Name for the resource
	Name *string `json:"name,omitempty"`
	// Reason why included in guide
	Description *string `json:"description,omitempty"`
	// Is an example/What is this an example of?
	Example *any `json:"example,omitempty"`
	// Grouping this is part of
	GroupingId *string `json:"groupingId,omitempty"`
}

// ImplementationGuideDefinitionPagePage represents a FHIR BackboneElement for ImplementationGuide.definition.page.page.
type ImplementationGuideDefinitionPagePage struct {
}

// ImplementationGuideDefinitionPage represents a FHIR BackboneElement for ImplementationGuide.definition.page.
type ImplementationGuideDefinitionPage struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Where to find that page
	Name any `json:"name"`
	// Short title shown for navigational assistance
	Title string `json:"title"`
	// html | markdown | xml | generated
	Generation string `json:"generation"`
	// Nested Pages / Sections
	Page []ImplementationGuideDefinitionPagePage `json:"page,omitempty"`
}

// ImplementationGuideDefinitionParameter represents a FHIR BackboneElement for ImplementationGuide.definition.parameter.
type ImplementationGuideDefinitionParameter struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// apply | path-resource | path-pages | path-tx-cache | expansion-parameter | rule-broken-links | generate-xml | generate-json | generate-turtle | html-template
	Code string `json:"code"`
	// Value for named type
	Value string `json:"value"`
}

// ImplementationGuideDefinitionTemplate represents a FHIR BackboneElement for ImplementationGuide.definition.template.
type ImplementationGuideDefinitionTemplate struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of template specified
	Code string `json:"code"`
	// The source location for the template
	Source string `json:"source"`
	// The scope in which the template applies
	Scope *string `json:"scope,omitempty"`
}

// ImplementationGuideDefinition represents a FHIR BackboneElement for ImplementationGuide.definition.
type ImplementationGuideDefinition struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Grouping used to present related resources in the IG
	Grouping []ImplementationGuideDefinitionGrouping `json:"grouping,omitempty"`
	// Resource in the implementation guide
	Resource []ImplementationGuideDefinitionResource `json:"resource,omitempty"`
	// Page/Section in the Guide
	Page *ImplementationGuideDefinitionPage `json:"page,omitempty"`
	// Defines how IG is built by tools
	Parameter []ImplementationGuideDefinitionParameter `json:"parameter,omitempty"`
	// A template for building resources
	Template []ImplementationGuideDefinitionTemplate `json:"template,omitempty"`
}

// ImplementationGuideManifestResource represents a FHIR BackboneElement for ImplementationGuide.manifest.resource.
type ImplementationGuideManifestResource struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Location of the resource
	Reference Reference `json:"reference"`
	// Is an example/What is this an example of?
	Example *any `json:"example,omitempty"`
	// Relative path for page in IG
	RelativePath *string `json:"relativePath,omitempty"`
}

// ImplementationGuideManifestPage represents a FHIR BackboneElement for ImplementationGuide.manifest.page.
type ImplementationGuideManifestPage struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// HTML page name
	Name string `json:"name"`
	// Title of the page, for references
	Title *string `json:"title,omitempty"`
	// Anchor available on the page
	Anchor []string `json:"anchor,omitempty"`
}

// ImplementationGuideManifest represents a FHIR BackboneElement for ImplementationGuide.manifest.
type ImplementationGuideManifest struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Location of rendered implementation guide
	Rendering *string `json:"rendering,omitempty"`
	// Resource in the implementation guide
	Resource []ImplementationGuideManifestResource `json:"resource,omitempty"`
	// HTML page within the parent IG
	Page []ImplementationGuideManifestPage `json:"page,omitempty"`
	// Image within the IG
	Image []string `json:"image,omitempty"`
	// Additional linkable file in IG
	Other []string `json:"other,omitempty"`
}

// ImplementationGuide represents a FHIR ImplementationGuide.
type ImplementationGuide struct {
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
	// Canonical identifier for this implementation guide, represented as a URI (globally unique)
	URL string `json:"url"`
	// Business version of the implementation guide
	Version *string `json:"version,omitempty"`
	// Name for this implementation guide (computer friendly)
	Name string `json:"name"`
	// Name for this implementation guide (human friendly)
	Title *string `json:"title,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Name of the publisher (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the implementation guide
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for implementation guide (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// NPM Package name for IG
	PackageId string `json:"packageId"`
	// SPDX license code for this IG (or not-open-source)
	License *string `json:"license,omitempty"`
	// FHIR Version(s) this Implementation Guide targets
	FhirVersion []string `json:"fhirVersion,omitempty"`
	// Another Implementation guide this depends on
	DependsOn []ImplementationGuideDependsOn `json:"dependsOn,omitempty"`
	// Profiles that apply globally
	Global []ImplementationGuideGlobal `json:"global,omitempty"`
	// Information needed to build the IG
	Definition *ImplementationGuideDefinition `json:"definition,omitempty"`
	// Information about an assembled IG
	Manifest *ImplementationGuideManifest `json:"manifest,omitempty"`
}
