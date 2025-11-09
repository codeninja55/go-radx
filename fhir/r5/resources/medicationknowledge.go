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

// MedicationKnowledgeCost represents a FHIR BackboneElement for MedicationKnowledge.cost.
type MedicationKnowledgeCost struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The date range for which the cost is effective
	EffectiveDate []Period `json:"effectiveDate,omitempty"`
	// The category of the cost information
	Type CodeableConcept `json:"type"`
	// The source or owner for the price information
	Source *string `json:"source,omitempty"`
	// The price or category of the cost of the medication
	Cost any `json:"cost"`
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

// MedicationKnowledgeIndicationGuidelineDosingGuidelineDosage represents a FHIR BackboneElement for MedicationKnowledge.indicationGuideline.dosingGuideline.dosage.
type MedicationKnowledgeIndicationGuidelineDosingGuidelineDosage struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Category of dosage for a medication
	Type CodeableConcept `json:"type"`
	// Dosage for the medication for the specific guidelines
	Dosage []Dosage `json:"dosage,omitempty"`
}

// MedicationKnowledgeIndicationGuidelineDosingGuidelinePatientCharacteristic represents a FHIR BackboneElement for MedicationKnowledge.indicationGuideline.dosingGuideline.patientCharacteristic.
type MedicationKnowledgeIndicationGuidelineDosingGuidelinePatientCharacteristic struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Categorization of specific characteristic that is relevant to the administration guideline
	Type CodeableConcept `json:"type"`
	// The specific characteristic
	Value *any `json:"value,omitempty"`
}

// MedicationKnowledgeIndicationGuidelineDosingGuideline represents a FHIR BackboneElement for MedicationKnowledge.indicationGuideline.dosingGuideline.
type MedicationKnowledgeIndicationGuidelineDosingGuideline struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Intention of the treatment
	TreatmentIntent *CodeableConcept `json:"treatmentIntent,omitempty"`
	// Dosage for the medication for the specific guidelines
	Dosage []MedicationKnowledgeIndicationGuidelineDosingGuidelineDosage `json:"dosage,omitempty"`
	// Type of treatment the guideline applies to
	AdministrationTreatment *CodeableConcept `json:"administrationTreatment,omitempty"`
	// Characteristics of the patient that are relevant to the administration guidelines
	PatientCharacteristic []MedicationKnowledgeIndicationGuidelineDosingGuidelinePatientCharacteristic `json:"patientCharacteristic,omitempty"`
}

// MedicationKnowledgeIndicationGuideline represents a FHIR BackboneElement for MedicationKnowledge.indicationGuideline.
type MedicationKnowledgeIndicationGuideline struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Indication for use that applies to the specific administration guideline
	Indication []CodeableReference `json:"indication,omitempty"`
	// Guidelines for dosage of the medication
	DosingGuideline []MedicationKnowledgeIndicationGuidelineDosingGuideline `json:"dosingGuideline,omitempty"`
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
	// The source of the classification
	Source *any `json:"source,omitempty"`
	// Specific category assigned to the medication
	Classification []CodeableConcept `json:"classification,omitempty"`
}

// MedicationKnowledgePackagingCost represents a FHIR BackboneElement for MedicationKnowledge.packaging.cost.
type MedicationKnowledgePackagingCost struct {
}

// MedicationKnowledgePackaging represents a FHIR BackboneElement for MedicationKnowledge.packaging.
type MedicationKnowledgePackaging struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Cost of the packaged medication
	Cost []MedicationKnowledgePackagingCost `json:"cost,omitempty"`
	// The packaged medication that is being priced
	PackagedProduct *Reference `json:"packagedProduct,omitempty"`
}

// MedicationKnowledgeStorageGuidelineEnvironmentalSetting represents a FHIR BackboneElement for MedicationKnowledge.storageGuideline.environmentalSetting.
type MedicationKnowledgeStorageGuidelineEnvironmentalSetting struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Categorization of the setting
	Type CodeableConcept `json:"type"`
	// Value of the setting
	Value any `json:"value"`
}

// MedicationKnowledgeStorageGuideline represents a FHIR BackboneElement for MedicationKnowledge.storageGuideline.
type MedicationKnowledgeStorageGuideline struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Reference to additional information
	Reference *string `json:"reference,omitempty"`
	// Additional storage notes
	Note []Annotation `json:"note,omitempty"`
	// Duration remains stable
	StabilityDuration *Duration `json:"stabilityDuration,omitempty"`
	// Setting or value of environment for adequate storage
	EnvironmentalSetting []MedicationKnowledgeStorageGuidelineEnvironmentalSetting `json:"environmentalSetting,omitempty"`
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
	Schedule []CodeableConcept `json:"schedule,omitempty"`
	// The maximum number of units of the medication that can be dispensed in a period
	MaxDispense *MedicationKnowledgeRegulatoryMaxDispense `json:"maxDispense,omitempty"`
}

// MedicationKnowledgeDefinitionalIngredient represents a FHIR BackboneElement for MedicationKnowledge.definitional.ingredient.
type MedicationKnowledgeDefinitionalIngredient struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Substances contained in the medication
	Item CodeableReference `json:"item"`
	// A code that defines the type of ingredient, active, base, etc
	Type *CodeableConcept `json:"type,omitempty"`
	// Quantity of ingredient present
	Strength *any `json:"strength,omitempty"`
}

// MedicationKnowledgeDefinitionalDrugCharacteristic represents a FHIR BackboneElement for MedicationKnowledge.definitional.drugCharacteristic.
type MedicationKnowledgeDefinitionalDrugCharacteristic struct {
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

// MedicationKnowledgeDefinitional represents a FHIR BackboneElement for MedicationKnowledge.definitional.
type MedicationKnowledgeDefinitional struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Definitional resources that provide more information about this medication
	Definition []Reference `json:"definition,omitempty"`
	// powder | tablets | capsule +
	DoseForm *CodeableConcept `json:"doseForm,omitempty"`
	// The intended or approved route of administration
	IntendedRoute []CodeableConcept `json:"intendedRoute,omitempty"`
	// Active or inactive ingredient
	Ingredient []MedicationKnowledgeDefinitionalIngredient `json:"ingredient,omitempty"`
	// Specifies descriptive properties of the medicine
	DrugCharacteristic []MedicationKnowledgeDefinitionalDrugCharacteristic `json:"drugCharacteristic,omitempty"`
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
	// Business identifier for this medication
	Identifier []Identifier `json:"identifier,omitempty"`
	// Code that identifies this medication
	Code *CodeableConcept `json:"code,omitempty"`
	// active | entered-in-error | inactive
	Status *string `json:"status,omitempty"`
	// Creator or owner of the knowledge or information about the medication
	Author *Reference `json:"author,omitempty"`
	// Codes that identify the different jurisdictions for which the information of this resource was created
	IntendedJurisdiction []CodeableConcept `json:"intendedJurisdiction,omitempty"`
	// A name associated with the medication being described
	Name []string `json:"name,omitempty"`
	// Associated or related medication information
	RelatedMedicationKnowledge []MedicationKnowledgeRelatedMedicationKnowledge `json:"relatedMedicationKnowledge,omitempty"`
	// The set of medication resources that are associated with this medication
	AssociatedMedication []Reference `json:"associatedMedication,omitempty"`
	// Category of the medication or product
	ProductType []CodeableConcept `json:"productType,omitempty"`
	// Associated documentation about the medication
	Monograph []MedicationKnowledgeMonograph `json:"monograph,omitempty"`
	// The instructions for preparing the medication
	PreparationInstruction *string `json:"preparationInstruction,omitempty"`
	// The pricing of the medication
	Cost []MedicationKnowledgeCost `json:"cost,omitempty"`
	// Program under which a medication is reviewed
	MonitoringProgram []MedicationKnowledgeMonitoringProgram `json:"monitoringProgram,omitempty"`
	// Guidelines or protocols for administration of the medication for an indication
	IndicationGuideline []MedicationKnowledgeIndicationGuideline `json:"indicationGuideline,omitempty"`
	// Categorization of the medication within a formulary or classification system
	MedicineClassification []MedicationKnowledgeMedicineClassification `json:"medicineClassification,omitempty"`
	// Details about packaged medications
	Packaging []MedicationKnowledgePackaging `json:"packaging,omitempty"`
	// Potential clinical issue with or between medication(s)
	ClinicalUseIssue []Reference `json:"clinicalUseIssue,omitempty"`
	// How the medication should be stored
	StorageGuideline []MedicationKnowledgeStorageGuideline `json:"storageGuideline,omitempty"`
	// Regulatory information about a medication
	Regulatory []MedicationKnowledgeRegulatory `json:"regulatory,omitempty"`
	// Minimal definition information about the medication
	Definitional *MedicationKnowledgeDefinitional `json:"definitional,omitempty"`
}
