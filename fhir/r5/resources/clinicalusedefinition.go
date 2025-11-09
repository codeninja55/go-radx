package resources

// ResourceTypeClinicalUseDefinition is the FHIR resource type name for ClinicalUseDefinition.
const ResourceTypeClinicalUseDefinition = "ClinicalUseDefinition"

// ClinicalUseDefinitionContraindicationOtherTherapy represents a FHIR BackboneElement for ClinicalUseDefinition.contraindication.otherTherapy.
type ClinicalUseDefinitionContraindicationOtherTherapy struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The type of relationship between the product indication/contraindication and another therapy
	RelationshipType CodeableConcept `json:"relationshipType"`
	// Reference to a specific medication, substance etc. as part of an indication or contraindication
	Treatment CodeableReference `json:"treatment"`
}

// ClinicalUseDefinitionContraindication represents a FHIR BackboneElement for ClinicalUseDefinition.contraindication.
type ClinicalUseDefinitionContraindication struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The situation that is being documented as contraindicating against this item
	DiseaseSymptomProcedure *CodeableReference `json:"diseaseSymptomProcedure,omitempty"`
	// The status of the disease or symptom for the contraindication
	DiseaseStatus *CodeableReference `json:"diseaseStatus,omitempty"`
	// A comorbidity (concurrent condition) or coinfection
	Comorbidity []CodeableReference `json:"comorbidity,omitempty"`
	// The indication which this is a contraidication for
	Indication []Reference `json:"indication,omitempty"`
	// An expression that returns true or false, indicating whether the indication is applicable or not, after having applied its other elements
	Applicability *Expression `json:"applicability,omitempty"`
	// Information about use of the product in relation to other therapies described as part of the contraindication
	OtherTherapy []ClinicalUseDefinitionContraindicationOtherTherapy `json:"otherTherapy,omitempty"`
}

// ClinicalUseDefinitionIndicationOtherTherapy represents a FHIR BackboneElement for ClinicalUseDefinition.indication.otherTherapy.
type ClinicalUseDefinitionIndicationOtherTherapy struct {
}

// ClinicalUseDefinitionIndication represents a FHIR BackboneElement for ClinicalUseDefinition.indication.
type ClinicalUseDefinitionIndication struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The situation that is being documented as an indicaton for this item
	DiseaseSymptomProcedure *CodeableReference `json:"diseaseSymptomProcedure,omitempty"`
	// The status of the disease or symptom for the indication
	DiseaseStatus *CodeableReference `json:"diseaseStatus,omitempty"`
	// A comorbidity or coinfection as part of the indication
	Comorbidity []CodeableReference `json:"comorbidity,omitempty"`
	// The intended effect, aim or strategy to be achieved
	IntendedEffect *CodeableReference `json:"intendedEffect,omitempty"`
	// Timing or duration information
	Duration *any `json:"duration,omitempty"`
	// An unwanted side effect or negative outcome of the subject of this resource when being used for this indication
	UndesirableEffect []Reference `json:"undesirableEffect,omitempty"`
	// An expression that returns true or false, indicating whether the indication is applicable or not, after having applied its other elements
	Applicability *Expression `json:"applicability,omitempty"`
	// The use of the medicinal product in relation to other therapies described as part of the indication
	OtherTherapy []ClinicalUseDefinitionIndicationOtherTherapy `json:"otherTherapy,omitempty"`
}

// ClinicalUseDefinitionInteractionInteractant represents a FHIR BackboneElement for ClinicalUseDefinition.interaction.interactant.
type ClinicalUseDefinitionInteractionInteractant struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The specific medication, product, food etc. or laboratory test that interacts
	Item any `json:"item"`
}

// ClinicalUseDefinitionInteraction represents a FHIR BackboneElement for ClinicalUseDefinition.interaction.
type ClinicalUseDefinitionInteraction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The specific medication, product, food etc. or laboratory test that interacts
	Interactant []ClinicalUseDefinitionInteractionInteractant `json:"interactant,omitempty"`
	// The type of the interaction e.g. drug-drug interaction, drug-lab test interaction
	Type *CodeableConcept `json:"type,omitempty"`
	// The effect of the interaction, for example "reduced gastric absorption of primary medication"
	Effect *CodeableReference `json:"effect,omitempty"`
	// The incidence of the interaction, e.g. theoretical, observed
	Incidence *CodeableConcept `json:"incidence,omitempty"`
	// Actions for managing the interaction
	Management []CodeableConcept `json:"management,omitempty"`
}

// ClinicalUseDefinitionUndesirableEffect represents a FHIR BackboneElement for ClinicalUseDefinition.undesirableEffect.
type ClinicalUseDefinitionUndesirableEffect struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The situation in which the undesirable effect may manifest
	SymptomConditionEffect *CodeableReference `json:"symptomConditionEffect,omitempty"`
	// High level classification of the effect
	Classification *CodeableConcept `json:"classification,omitempty"`
	// How often the effect is seen
	FrequencyOfOccurrence *CodeableConcept `json:"frequencyOfOccurrence,omitempty"`
}

// ClinicalUseDefinitionWarning represents a FHIR BackboneElement for ClinicalUseDefinition.warning.
type ClinicalUseDefinitionWarning struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A textual definition of this warning, with formatting
	Description *string `json:"description,omitempty"`
	// A coded or unformatted textual definition of this warning
	Code *CodeableConcept `json:"code,omitempty"`
}

// ClinicalUseDefinition represents a FHIR ClinicalUseDefinition.
type ClinicalUseDefinition struct {
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
	// Business identifier for this issue
	Identifier []Identifier `json:"identifier,omitempty"`
	// indication | contraindication | interaction | undesirable-effect | warning
	Type string `json:"type"`
	// A categorisation of the issue, primarily for dividing warnings into subject heading areas such as "Pregnancy", "Overdose"
	Category []CodeableConcept `json:"category,omitempty"`
	// The medication, product, substance, device, procedure etc. for which this is an indication
	Subject []Reference `json:"subject,omitempty"`
	// Whether this is a current issue or one that has been retired etc
	Status *CodeableConcept `json:"status,omitempty"`
	// Specifics for when this is a contraindication
	Contraindication *ClinicalUseDefinitionContraindication `json:"contraindication,omitempty"`
	// Specifics for when this is an indication
	Indication *ClinicalUseDefinitionIndication `json:"indication,omitempty"`
	// Specifics for when this is an interaction
	Interaction *ClinicalUseDefinitionInteraction `json:"interaction,omitempty"`
	// The population group to which this applies
	Population []Reference `json:"population,omitempty"`
	// Logic used by the clinical use definition
	Library []string `json:"library,omitempty"`
	// A possible negative outcome from the use of this treatment
	UndesirableEffect *ClinicalUseDefinitionUndesirableEffect `json:"undesirableEffect,omitempty"`
	// Critical environmental, health or physical risks or hazards. For example 'Do not operate heavy machinery', 'May cause drowsiness'
	Warning *ClinicalUseDefinitionWarning `json:"warning,omitempty"`
}
