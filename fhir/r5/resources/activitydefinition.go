package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeActivityDefinition is the FHIR resource type name for ActivityDefinition.
const ResourceTypeActivityDefinition = "ActivityDefinition"

// ActivityDefinitionParticipant represents a FHIR BackboneElement for ActivityDefinition.participant.
type ActivityDefinitionParticipant struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// careteam | device | group | healthcareservice | location | organization | patient | practitioner | practitionerrole | relatedperson
	Type *string `json:"type,omitempty"`
	// Who or what can participate
	TypeCanonical *string `json:"typeCanonical,omitempty"`
	// Who or what can participate
	TypeReference *Reference `json:"typeReference,omitempty"`
	// E.g. Nurse, Surgeon, Parent, etc
	Role *CodeableConcept `json:"role,omitempty"`
	// E.g. Author, Reviewer, Witness, etc
	Function *CodeableConcept `json:"function,omitempty"`
}

// ActivityDefinitionDynamicValue represents a FHIR BackboneElement for ActivityDefinition.dynamicValue.
type ActivityDefinitionDynamicValue struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The path to the element to be set dynamically
	Path string `json:"path"`
	// An expression that provides the dynamic value for the customization
	Expression Expression `json:"expression"`
}

// ActivityDefinition represents a FHIR ActivityDefinition.
type ActivityDefinition struct {
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
	// Canonical identifier for this activity definition, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the activity definition
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the activity definition
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this activity definition (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this activity definition (human friendly)
	Title *string `json:"title,omitempty"`
	// Subordinate title of the activity definition
	Subtitle *string `json:"subtitle,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// Type of individual the activity definition is intended for
	Subject *any `json:"subject,omitempty"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Name of the publisher/steward (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the activity definition
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for activity definition (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this activity definition is defined
	Purpose *string `json:"purpose,omitempty"`
	// Describes the clinical usage of the activity definition
	Usage *string `json:"usage,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// When the activity definition was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// When the activity definition was last reviewed by the publisher
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// When the activity definition is expected to be used
	EffectivePeriod *Period `json:"effectivePeriod,omitempty"`
	// E.g. Education, Treatment, Assessment, etc
	Topic []CodeableConcept `json:"topic,omitempty"`
	// Who authored the content
	Author []ContactDetail `json:"author,omitempty"`
	// Who edited the content
	Editor []ContactDetail `json:"editor,omitempty"`
	// Who reviewed the content
	Reviewer []ContactDetail `json:"reviewer,omitempty"`
	// Who endorsed the content
	Endorser []ContactDetail `json:"endorser,omitempty"`
	// Additional documentation, citations, etc
	RelatedArtifact []RelatedArtifact `json:"relatedArtifact,omitempty"`
	// Logic used by the activity definition
	Library []string `json:"library,omitempty"`
	// Kind of resource
	Kind *string `json:"kind,omitempty"`
	// What profile the resource needs to conform to
	Profile *string `json:"profile,omitempty"`
	// Detail type of activity
	Code *CodeableConcept `json:"code,omitempty"`
	// proposal | plan | directive | order | original-order | reflex-order | filler-order | instance-order | option
	Intent *string `json:"intent,omitempty"`
	// routine | urgent | asap | stat
	Priority *string `json:"priority,omitempty"`
	// True if the activity should not be performed
	DoNotPerform *bool `json:"doNotPerform,omitempty"`
	// When activity is to occur
	Timing *any `json:"timing,omitempty"`
	// Preconditions for service
	AsNeeded *any `json:"asNeeded,omitempty"`
	// Where it should happen
	Location *CodeableReference `json:"location,omitempty"`
	// Who should participate in the action
	Participant []ActivityDefinitionParticipant `json:"participant,omitempty"`
	// What's administered/supplied
	Product *any `json:"product,omitempty"`
	// How much is administered/consumed/supplied
	Quantity *Quantity `json:"quantity,omitempty"`
	// Detailed dosage instructions
	Dosage []Dosage `json:"dosage,omitempty"`
	// What part of body to perform on
	BodySite []CodeableConcept `json:"bodySite,omitempty"`
	// What specimens are required to perform this action
	SpecimenRequirement []string `json:"specimenRequirement,omitempty"`
	// What observations are required to perform this action
	ObservationRequirement []string `json:"observationRequirement,omitempty"`
	// What observations must be produced by this action
	ObservationResultRequirement []string `json:"observationResultRequirement,omitempty"`
	// Transform to apply the template
	Transform *string `json:"transform,omitempty"`
	// Dynamic aspects of the definition
	DynamicValue []ActivityDefinitionDynamicValue `json:"dynamicValue,omitempty"`
}
