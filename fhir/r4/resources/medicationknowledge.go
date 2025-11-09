package resources

// ResourceTypeMedicationKnowledge is the FHIR resource type name for MedicationKnowledge.
const ResourceTypeMedicationKnowledge = "MedicationKnowledge"

// MedicationKnowledgeRelatedMedicationKnowledge represents a FHIR BackboneElement for MedicationKnowledge.relatedMedicationKnowledge.
type MedicationKnowledgeRelatedMedicationKnowledge struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Category of medicationKnowledge
	Type CodeableConcept `json:"type"`
	// Associated documentation about the associated medication knowledge
	Reference []Reference `json:"reference,omitempty"`
}

// MedicationKnowledgeMonograph represents a FHIR BackboneElement for MedicationKnowledge.monograph.
type MedicationKnowledgeMonograph struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The category of medication document
	Type *CodeableConcept `json:"type,omitempty"`
	// Associated documentation about the medication
	Source *Reference `json:"source,omitempty"`
}

// MedicationKnowledgeIngredient represents a FHIR BackboneElement for MedicationKnowledge.ingredient.
type MedicationKnowledgeIngredient struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Medication(s) or substance(s) contained in the medication
	Item any `json:"item"`
	// Active ingredient indicator
	IsActive *bool `json:"isActive,omitempty"`
	// Quantity of ingredient present
	Strength *Ratio `json:"strength,omitempty"`
}

// MedicationKnowledgeCost represents a FHIR BackboneElement for MedicationKnowledge.cost.
type MedicationKnowledgeCost struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The category of the cost information
	Type CodeableConcept `json:"type"`
	// The source or owner for the price information
	Source *string `json:"source,omitempty"`
	// The price of the medication
	Cost Money `json:"cost"`
}

// MedicationKnowledgeMonitoringProgram represents a FHIR BackboneElement for MedicationKnowledge.monitoringProgram.
type MedicationKnowledgeMonitoringProgram struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of program under which the medication is monitored
	Type *CodeableConcept `json:"type,omitempty"`
	// Name of the reviewing program
	Name *string `json:"name,omitempty"`
}

// MedicationKnowledgeAdministrationGuidelinesDosage represents a FHIR BackboneElement for MedicationKnowledge.administrationGuidelines.dosage.
type MedicationKnowledgeAdministrationGuidelinesDosage struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of dosage
	Type CodeableConcept `json:"type"`
	// Dosage for the medication for the specific guidelines
	Dosage []Dosage `json:"dosage,omitempty"`
}

// MedicationKnowledgeAdministrationGuidelinesPatientCharacteristics represents a FHIR BackboneElement for MedicationKnowledge.administrationGuidelines.patientCharacteristics.
type MedicationKnowledgeAdministrationGuidelinesPatientCharacteristics struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Specific characteristic that is relevant to the administration guideline
	Characteristic any `json:"characteristic"`
	// The specific characteristic
	Value []string `json:"value,omitempty"`
}

// MedicationKnowledgeAdministrationGuidelines represents a FHIR BackboneElement for MedicationKnowledge.administrationGuidelines.
type MedicationKnowledgeAdministrationGuidelines struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Dosage for the medication for the specific guidelines
	Dosage []MedicationKnowledgeAdministrationGuidelinesDosage `json:"dosage,omitempty"`
	// Indication for use that apply to the specific administration guidelines
	Indication *any `json:"indication,omitempty"`
	// Characteristics of the patient that are relevant to the administration guidelines
	PatientCharacteristics []MedicationKnowledgeAdministrationGuidelinesPatientCharacteristics `json:"patientCharacteristics,omitempty"`
}

// MedicationKnowledgeMedicineClassification represents a FHIR BackboneElement for MedicationKnowledge.medicineClassification.
type MedicationKnowledgeMedicineClassification struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The type of category for the medication (for example, therapeutic classification, therapeutic sub-classification)
	Type CodeableConcept `json:"type"`
	// Specific category assigned to the medication
	Classification []CodeableConcept `json:"classification,omitempty"`
}

// MedicationKnowledgePackaging represents a FHIR BackboneElement for MedicationKnowledge.packaging.
type MedicationKnowledgePackaging struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A code that defines the specific type of packaging that the medication can be found in
	Type *CodeableConcept `json:"type,omitempty"`
	// The number of product units the package would contain if fully loaded
	Quantity *Quantity `json:"quantity,omitempty"`
}

// MedicationKnowledgeDrugCharacteristic represents a FHIR BackboneElement for MedicationKnowledge.drugCharacteristic.
type MedicationKnowledgeDrugCharacteristic struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Code specifying the type of characteristic of medication
	Type *CodeableConcept `json:"type,omitempty"`
	// Description of the characteristic
	Value *any `json:"value,omitempty"`
}

// MedicationKnowledgeRegulatorySubstitution represents a FHIR BackboneElement for MedicationKnowledge.regulatory.substitution.
type MedicationKnowledgeRegulatorySubstitution struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Specifies the type of substitution allowed
	Type CodeableConcept `json:"type"`
	// Specifies if regulation allows for changes in the medication when dispensing
	Allowed bool `json:"allowed"`
}

// MedicationKnowledgeRegulatorySchedule represents a FHIR BackboneElement for MedicationKnowledge.regulatory.schedule.
type MedicationKnowledgeRegulatorySchedule struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Specifies the specific drug schedule
	Schedule CodeableConcept `json:"schedule"`
}

// MedicationKnowledgeRegulatoryMaxDispense represents a FHIR BackboneElement for MedicationKnowledge.regulatory.maxDispense.
type MedicationKnowledgeRegulatoryMaxDispense struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The maximum number of units of the medication that can be dispensed
	Quantity Quantity `json:"quantity"`
	// The period that applies to the maximum number of units
	Period *Duration `json:"period,omitempty"`
}

// MedicationKnowledgeRegulatory represents a FHIR BackboneElement for MedicationKnowledge.regulatory.
type MedicationKnowledgeRegulatory struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Specifies the authority of the regulation
	RegulatoryAuthority Reference `json:"regulatoryAuthority"`
	// Specifies if changes are allowed when dispensing a medication from a regulatory perspective
	Substitution []MedicationKnowledgeRegulatorySubstitution `json:"substitution,omitempty"`
	// Specifies the schedule of a medication in jurisdiction
	Schedule []MedicationKnowledgeRegulatorySchedule `json:"schedule,omitempty"`
	// The maximum number of units of the medication that can be dispensed in a period
	MaxDispense *MedicationKnowledgeRegulatoryMaxDispense `json:"maxDispense,omitempty"`
}

// MedicationKnowledgeKinetics represents a FHIR BackboneElement for MedicationKnowledge.kinetics.
type MedicationKnowledgeKinetics struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The drug concentration measured at certain discrete points in time
	AreaUnderCurve []Quantity `json:"areaUnderCurve,omitempty"`
	// The median lethal dose of a drug
	LethalDose50 []Quantity `json:"lethalDose50,omitempty"`
	// Time required for concentration in the body to decrease by half
	HalfLifePeriod *Duration `json:"halfLifePeriod,omitempty"`
}

// MedicationKnowledge represents a FHIR MedicationKnowledge.
type MedicationKnowledge struct {
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
	// Code that identifies this medication
	Code *CodeableConcept `json:"code,omitempty"`
	// active | inactive | entered-in-error
	Status *string `json:"status,omitempty"`
	// Manufacturer of the item
	Manufacturer *Reference `json:"manufacturer,omitempty"`
	// powder | tablets | capsule +
	DoseForm *CodeableConcept `json:"doseForm,omitempty"`
	// Amount of drug in package
	Amount *Quantity `json:"amount,omitempty"`
	// Additional names for a medication
	Synonym []string `json:"synonym,omitempty"`
	// Associated or related medication information
	RelatedMedicationKnowledge []MedicationKnowledgeRelatedMedicationKnowledge `json:"relatedMedicationKnowledge,omitempty"`
	// A medication resource that is associated with this medication
	AssociatedMedication []Reference `json:"associatedMedication,omitempty"`
	// Category of the medication or product
	ProductType []CodeableConcept `json:"productType,omitempty"`
	// Associated documentation about the medication
	Monograph []MedicationKnowledgeMonograph `json:"monograph,omitempty"`
	// Active or inactive ingredient
	Ingredient []MedicationKnowledgeIngredient `json:"ingredient,omitempty"`
	// The instructions for preparing the medication
	PreparationInstruction *string `json:"preparationInstruction,omitempty"`
	// The intended or approved route of administration
	IntendedRoute []CodeableConcept `json:"intendedRoute,omitempty"`
	// The pricing of the medication
	Cost []MedicationKnowledgeCost `json:"cost,omitempty"`
	// Program under which a medication is reviewed
	MonitoringProgram []MedicationKnowledgeMonitoringProgram `json:"monitoringProgram,omitempty"`
	// Guidelines for administration of the medication
	AdministrationGuidelines []MedicationKnowledgeAdministrationGuidelines `json:"administrationGuidelines,omitempty"`
	// Categorization of the medication within a formulary or classification system
	MedicineClassification []MedicationKnowledgeMedicineClassification `json:"medicineClassification,omitempty"`
	// Details about packaged medications
	Packaging *MedicationKnowledgePackaging `json:"packaging,omitempty"`
	// Specifies descriptive properties of the medicine
	DrugCharacteristic []MedicationKnowledgeDrugCharacteristic `json:"drugCharacteristic,omitempty"`
	// Potential clinical issue with or between medication(s)
	Contraindication []Reference `json:"contraindication,omitempty"`
	// Regulatory information about a medication
	Regulatory []MedicationKnowledgeRegulatory `json:"regulatory,omitempty"`
	// The time course of drug absorption, distribution, metabolism and excretion of a medication from the body
	Kinetics []MedicationKnowledgeKinetics `json:"kinetics,omitempty"`
}
