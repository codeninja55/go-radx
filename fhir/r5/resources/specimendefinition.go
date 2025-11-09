package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeSpecimenDefinition is the FHIR resource type name for SpecimenDefinition.
const ResourceTypeSpecimenDefinition = "SpecimenDefinition"

// SpecimenDefinitionTypeTestedContainerAdditive represents a FHIR BackboneElement for SpecimenDefinition.typeTested.container.additive.
type SpecimenDefinitionTypeTestedContainerAdditive struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Additive associated with container
	Additive any `json:"additive"`
}

// SpecimenDefinitionTypeTestedContainer represents a FHIR BackboneElement for SpecimenDefinition.typeTested.container.
type SpecimenDefinitionTypeTestedContainer struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The material type used for the container
	Material *CodeableConcept `json:"material,omitempty"`
	// Kind of container associated with the kind of specimen
	Type *CodeableConcept `json:"type,omitempty"`
	// Color of container cap
	Cap *CodeableConcept `json:"cap,omitempty"`
	// The description of the kind of container
	Description *string `json:"description,omitempty"`
	// The capacity of this kind of container
	Capacity *Quantity `json:"capacity,omitempty"`
	// Minimum volume
	MinimumVolume *any `json:"minimumVolume,omitempty"`
	// Additive associated with container
	Additive []SpecimenDefinitionTypeTestedContainerAdditive `json:"additive,omitempty"`
	// Special processing applied to the container for this specimen type
	Preparation *string `json:"preparation,omitempty"`
}

// SpecimenDefinitionTypeTestedHandling represents a FHIR BackboneElement for SpecimenDefinition.typeTested.handling.
type SpecimenDefinitionTypeTestedHandling struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Qualifies the interval of temperature
	TemperatureQualifier *CodeableConcept `json:"temperatureQualifier,omitempty"`
	// Temperature range for these handling instructions
	TemperatureRange *Range `json:"temperatureRange,omitempty"`
	// Maximum preservation time
	MaxDuration *Duration `json:"maxDuration,omitempty"`
	// Preservation instruction
	Instruction *string `json:"instruction,omitempty"`
}

// SpecimenDefinitionTypeTested represents a FHIR BackboneElement for SpecimenDefinition.typeTested.
type SpecimenDefinitionTypeTested struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Primary or secondary specimen
	IsDerived *bool `json:"isDerived,omitempty"`
	// Type of intended specimen
	Type *CodeableConcept `json:"type,omitempty"`
	// preferred | alternate
	Preference string `json:"preference"`
	// The specimen's container
	Container *SpecimenDefinitionTypeTestedContainer `json:"container,omitempty"`
	// Requirements for specimen delivery and special handling
	Requirement *string `json:"requirement,omitempty"`
	// The usual time for retaining this kind of specimen
	RetentionTime *Duration `json:"retentionTime,omitempty"`
	// Specimen for single use only
	SingleUse *bool `json:"singleUse,omitempty"`
	// Criterion specified for specimen rejection
	RejectionCriterion []CodeableConcept `json:"rejectionCriterion,omitempty"`
	// Specimen handling before testing
	Handling []SpecimenDefinitionTypeTestedHandling `json:"handling,omitempty"`
	// Where the specimen will be tested
	TestingDestination []CodeableConcept `json:"testingDestination,omitempty"`
}

// SpecimenDefinition represents a FHIR SpecimenDefinition.
type SpecimenDefinition struct {
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
	// Logical canonical URL to reference this SpecimenDefinition (globally unique)
	URL *string `json:"url,omitempty"`
	// Business identifier
	Identifier *Identifier `json:"identifier,omitempty"`
	// Business version of the SpecimenDefinition
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this {{title}} (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this SpecimenDefinition (Human friendly)
	Title *string `json:"title,omitempty"`
	// Based on FHIR definition of another SpecimenDefinition
	DerivedFromCanonical []string `json:"derivedFromCanonical,omitempty"`
	// Based on external definition
	DerivedFromUri []string `json:"derivedFromUri,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// If this SpecimenDefinition is not for real usage
	Experimental *bool `json:"experimental,omitempty"`
	// Type of subject for specimen collection
	Subject *any `json:"subject,omitempty"`
	// Date status first applied
	Date *primitives.DateTime `json:"date,omitempty"`
	// The name of the individual or organization that published the SpecimenDefinition
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the SpecimenDefinition
	Description *string `json:"description,omitempty"`
	// Content intends to support these contexts
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for this SpecimenDefinition (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this SpecimenDefinition is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// When SpecimenDefinition was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// The date on which the asset content was last reviewed by the publisher
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// The effective date range for the SpecimenDefinition
	EffectivePeriod *Period `json:"effectivePeriod,omitempty"`
	// Kind of material to collect
	TypeCollected *CodeableConcept `json:"typeCollected,omitempty"`
	// Patient preparation for collection
	PatientPreparation []CodeableConcept `json:"patientPreparation,omitempty"`
	// Time aspect for collection
	TimeAspect *string `json:"timeAspect,omitempty"`
	// Specimen collection procedure
	Collection []CodeableConcept `json:"collection,omitempty"`
	// Specimen in container intended for testing by lab
	TypeTested []SpecimenDefinitionTypeTested `json:"typeTested,omitempty"`
}
