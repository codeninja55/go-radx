package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeCapabilityStatement is the FHIR resource type name for CapabilityStatement.
const ResourceTypeCapabilityStatement = "CapabilityStatement"

// CapabilityStatementSoftware represents a FHIR BackboneElement for CapabilityStatement.software.
type CapabilityStatementSoftware struct {
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
	// Date this version was released
	ReleaseDate *primitives.DateTime `json:"releaseDate,omitempty"`
}

// CapabilityStatementImplementation represents a FHIR BackboneElement for CapabilityStatement.implementation.
type CapabilityStatementImplementation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Describes this specific instance
	Description string `json:"description"`
	// Base URL for the installation
	URL *string `json:"url,omitempty"`
	// Organization that manages the data
	Custodian *Reference `json:"custodian,omitempty"`
}

// CapabilityStatementRestSecurity represents a FHIR BackboneElement for CapabilityStatement.rest.security.
type CapabilityStatementRestSecurity struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Adds CORS Headers (http://enable-cors.org/)
	Cors *bool `json:"cors,omitempty"`
	// OAuth | SMART-on-FHIR | NTLM | Basic | Kerberos | Certificates
	Service []CodeableConcept `json:"service,omitempty"`
	// General description of how security works
	Description *string `json:"description,omitempty"`
}

// CapabilityStatementRestResourceInteraction represents a FHIR BackboneElement for CapabilityStatement.rest.resource.interaction.
type CapabilityStatementRestResourceInteraction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// read | vread | update | patch | delete | history-instance | history-type | create | search-type
	Code string `json:"code"`
	// Anything special about operation behavior
	Documentation *string `json:"documentation,omitempty"`
}

// CapabilityStatementRestResourceSearchParam represents a FHIR BackboneElement for CapabilityStatement.rest.resource.searchParam.
type CapabilityStatementRestResourceSearchParam struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Name of search parameter
	Name string `json:"name"`
	// Source of definition for parameter
	Definition *string `json:"definition,omitempty"`
	// number | date | string | token | reference | composite | quantity | uri | special
	Type string `json:"type"`
	// Server-specific usage
	Documentation *string `json:"documentation,omitempty"`
}

// CapabilityStatementRestResourceOperation represents a FHIR BackboneElement for CapabilityStatement.rest.resource.operation.
type CapabilityStatementRestResourceOperation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Name by which the operation/query is invoked
	Name string `json:"name"`
	// The defined operation/query
	Definition string `json:"definition"`
	// Specific details about operation behavior
	Documentation *string `json:"documentation,omitempty"`
}

// CapabilityStatementRestResource represents a FHIR BackboneElement for CapabilityStatement.rest.resource.
type CapabilityStatementRestResource struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A resource type that is supported
	Type string `json:"type"`
	// Base System profile for all uses of resource
	Profile *string `json:"profile,omitempty"`
	// Profiles for use cases supported
	SupportedProfile []string `json:"supportedProfile,omitempty"`
	// Additional information about the use of the resource type
	Documentation *string `json:"documentation,omitempty"`
	// What operations are supported?
	Interaction []CapabilityStatementRestResourceInteraction `json:"interaction,omitempty"`
	// no-version | versioned | versioned-update
	Versioning *string `json:"versioning,omitempty"`
	// Whether vRead can return past versions
	ReadHistory *bool `json:"readHistory,omitempty"`
	// If update can commit to a new identity
	UpdateCreate *bool `json:"updateCreate,omitempty"`
	// If allows/uses conditional create
	ConditionalCreate *bool `json:"conditionalCreate,omitempty"`
	// not-supported | modified-since | not-match | full-support
	ConditionalRead *string `json:"conditionalRead,omitempty"`
	// If allows/uses conditional update
	ConditionalUpdate *bool `json:"conditionalUpdate,omitempty"`
	// not-supported | single | multiple - how conditional delete is supported
	ConditionalDelete *string `json:"conditionalDelete,omitempty"`
	// literal | logical | resolves | enforced | local
	ReferencePolicy []string `json:"referencePolicy,omitempty"`
	// _include values supported by the server
	SearchInclude []string `json:"searchInclude,omitempty"`
	// _revinclude values supported by the server
	SearchRevInclude []string `json:"searchRevInclude,omitempty"`
	// Search parameters supported by implementation
	SearchParam []CapabilityStatementRestResourceSearchParam `json:"searchParam,omitempty"`
	// Definition of a resource operation
	Operation []CapabilityStatementRestResourceOperation `json:"operation,omitempty"`
}

// CapabilityStatementRestInteraction represents a FHIR BackboneElement for CapabilityStatement.rest.interaction.
type CapabilityStatementRestInteraction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// transaction | batch | search-system | history-system
	Code string `json:"code"`
	// Anything special about operation behavior
	Documentation *string `json:"documentation,omitempty"`
}

// CapabilityStatementRestSearchParam represents a FHIR BackboneElement for CapabilityStatement.rest.searchParam.
type CapabilityStatementRestSearchParam struct {
}

// CapabilityStatementRestOperation represents a FHIR BackboneElement for CapabilityStatement.rest.operation.
type CapabilityStatementRestOperation struct {
}

// CapabilityStatementRest represents a FHIR BackboneElement for CapabilityStatement.rest.
type CapabilityStatementRest struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// client | server
	Mode string `json:"mode"`
	// General description of implementation
	Documentation *string `json:"documentation,omitempty"`
	// Information about security of implementation
	Security *CapabilityStatementRestSecurity `json:"security,omitempty"`
	// Resource served on the REST interface
	Resource []CapabilityStatementRestResource `json:"resource,omitempty"`
	// What operations are supported?
	Interaction []CapabilityStatementRestInteraction `json:"interaction,omitempty"`
	// Search parameters for searching all resources
	SearchParam []CapabilityStatementRestSearchParam `json:"searchParam,omitempty"`
	// Definition of a system level operation
	Operation []CapabilityStatementRestOperation `json:"operation,omitempty"`
	// Compartments served/used by system
	Compartment []string `json:"compartment,omitempty"`
}

// CapabilityStatementMessagingEndpoint represents a FHIR BackboneElement for CapabilityStatement.messaging.endpoint.
type CapabilityStatementMessagingEndpoint struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// http | ftp | mllp +
	Protocol Coding `json:"protocol"`
	// Network address or identifier of the end-point
	Address string `json:"address"`
}

// CapabilityStatementMessagingSupportedMessage represents a FHIR BackboneElement for CapabilityStatement.messaging.supportedMessage.
type CapabilityStatementMessagingSupportedMessage struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// sender | receiver
	Mode string `json:"mode"`
	// Message supported by this system
	Definition string `json:"definition"`
}

// CapabilityStatementMessaging represents a FHIR BackboneElement for CapabilityStatement.messaging.
type CapabilityStatementMessaging struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Where messages should be sent
	Endpoint []CapabilityStatementMessagingEndpoint `json:"endpoint,omitempty"`
	// Reliable Message Cache Length (min)
	ReliableCache *uint `json:"reliableCache,omitempty"`
	// Messaging interface behavior details
	Documentation *string `json:"documentation,omitempty"`
	// Messages supported by this system
	SupportedMessage []CapabilityStatementMessagingSupportedMessage `json:"supportedMessage,omitempty"`
}

// CapabilityStatementDocument represents a FHIR BackboneElement for CapabilityStatement.document.
type CapabilityStatementDocument struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// producer | consumer
	Mode string `json:"mode"`
	// Description of document support
	Documentation *string `json:"documentation,omitempty"`
	// Constraint on the resources used in the document
	Profile string `json:"profile"`
}

// CapabilityStatement represents a FHIR CapabilityStatement.
type CapabilityStatement struct {
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
	// Canonical identifier for this capability statement, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Business version of the capability statement
	Version *string `json:"version,omitempty"`
	// Name for this capability statement (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this capability statement (human friendly)
	Title *string `json:"title,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// Date last changed
	Date primitives.DateTime `json:"date"`
	// Name of the publisher (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the capability statement
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for capability statement (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this capability statement is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// instance | capability | requirements
	Kind string `json:"kind"`
	// Canonical URL of another capability statement this implements
	Instantiates []string `json:"instantiates,omitempty"`
	// Canonical URL of another capability statement this adds to
	Imports []string `json:"imports,omitempty"`
	// Software that is covered by this capability statement
	Software *CapabilityStatementSoftware `json:"software,omitempty"`
	// If this describes a specific instance
	Implementation *CapabilityStatementImplementation `json:"implementation,omitempty"`
	// FHIR Version the system supports
	FhirVersion string `json:"fhirVersion"`
	// formats supported (xml | json | ttl | mime type)
	Format []string `json:"format,omitempty"`
	// Patch formats supported
	PatchFormat []string `json:"patchFormat,omitempty"`
	// Implementation guides supported
	ImplementationGuide []string `json:"implementationGuide,omitempty"`
	// If the endpoint is a RESTful one
	Rest []CapabilityStatementRest `json:"rest,omitempty"`
	// If messaging is supported
	Messaging []CapabilityStatementMessaging `json:"messaging,omitempty"`
	// Document definition
	Document []CapabilityStatementDocument `json:"document,omitempty"`
}
