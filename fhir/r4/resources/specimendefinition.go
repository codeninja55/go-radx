package resources

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
	// Container material
	Material *CodeableConcept `json:"material,omitempty"`
	// Kind of container associated with the kind of specimen
	Type *CodeableConcept `json:"type,omitempty"`
	// Color of container cap
	Cap *CodeableConcept `json:"cap,omitempty"`
	// Container description
	Description *string `json:"description,omitempty"`
	// Container capacity
	Capacity *Quantity `json:"capacity,omitempty"`
	// Minimum volume
	MinimumVolume *any `json:"minimumVolume,omitempty"`
	// Additive associated with container
	Additive []SpecimenDefinitionTypeTestedContainerAdditive `json:"additive,omitempty"`
	// Specimen container preparation
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
	// Temperature qualifier
	TemperatureQualifier *CodeableConcept `json:"temperatureQualifier,omitempty"`
	// Temperature range
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
	// Specimen requirements
	Requirement *string `json:"requirement,omitempty"`
	// Specimen retention time
	RetentionTime *Duration `json:"retentionTime,omitempty"`
	// Rejection criterion
	RejectionCriterion []CodeableConcept `json:"rejectionCriterion,omitempty"`
	// Specimen handling before testing
	Handling []SpecimenDefinitionTypeTestedHandling `json:"handling,omitempty"`
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
	// Business identifier of a kind of specimen
	Identifier *Identifier `json:"identifier,omitempty"`
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
