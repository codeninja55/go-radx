package resources

// ResourceTypeSubstancePolymer is the FHIR resource type name for SubstancePolymer.
const ResourceTypeSubstancePolymer = "SubstancePolymer"

// SubstancePolymerMonomerSetStartingMaterial represents a FHIR BackboneElement for SubstancePolymer.monomerSet.startingMaterial.
type SubstancePolymerMonomerSetStartingMaterial struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Todo
	Material *CodeableConcept `json:"material,omitempty"`
	// Todo
	Type *CodeableConcept `json:"type,omitempty"`
	// Todo
	IsDefining *bool `json:"isDefining,omitempty"`
	// Todo
	Amount *SubstanceAmount `json:"amount,omitempty"`
}

// SubstancePolymerMonomerSet represents a FHIR BackboneElement for SubstancePolymer.monomerSet.
type SubstancePolymerMonomerSet struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Todo
	RatioType *CodeableConcept `json:"ratioType,omitempty"`
	// Todo
	StartingMaterial []SubstancePolymerMonomerSetStartingMaterial `json:"startingMaterial,omitempty"`
}

// SubstancePolymerRepeatRepeatUnitDegreeOfPolymerisation represents a FHIR BackboneElement for SubstancePolymer.repeat.repeatUnit.degreeOfPolymerisation.
type SubstancePolymerRepeatRepeatUnitDegreeOfPolymerisation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Todo
	Degree *CodeableConcept `json:"degree,omitempty"`
	// Todo
	Amount *SubstanceAmount `json:"amount,omitempty"`
}

// SubstancePolymerRepeatRepeatUnitStructuralRepresentation represents a FHIR BackboneElement for SubstancePolymer.repeat.repeatUnit.structuralRepresentation.
type SubstancePolymerRepeatRepeatUnitStructuralRepresentation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Todo
	Type *CodeableConcept `json:"type,omitempty"`
	// Todo
	Representation *string `json:"representation,omitempty"`
	// Todo
	Attachment *Attachment `json:"attachment,omitempty"`
}

// SubstancePolymerRepeatRepeatUnit represents a FHIR BackboneElement for SubstancePolymer.repeat.repeatUnit.
type SubstancePolymerRepeatRepeatUnit struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Todo
	OrientationOfPolymerisation *CodeableConcept `json:"orientationOfPolymerisation,omitempty"`
	// Todo
	RepeatUnit *string `json:"repeatUnit,omitempty"`
	// Todo
	Amount *SubstanceAmount `json:"amount,omitempty"`
	// Todo
	DegreeOfPolymerisation []SubstancePolymerRepeatRepeatUnitDegreeOfPolymerisation `json:"degreeOfPolymerisation,omitempty"`
	// Todo
	StructuralRepresentation []SubstancePolymerRepeatRepeatUnitStructuralRepresentation `json:"structuralRepresentation,omitempty"`
}

// SubstancePolymerRepeat represents a FHIR BackboneElement for SubstancePolymer.repeat.
type SubstancePolymerRepeat struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Todo
	NumberOfUnits *int `json:"numberOfUnits,omitempty"`
	// Todo
	AverageMolecularFormula *string `json:"averageMolecularFormula,omitempty"`
	// Todo
	RepeatUnitAmountType *CodeableConcept `json:"repeatUnitAmountType,omitempty"`
	// Todo
	RepeatUnit []SubstancePolymerRepeatRepeatUnit `json:"repeatUnit,omitempty"`
}

// SubstancePolymer represents a FHIR SubstancePolymer.
type SubstancePolymer struct {
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
	// Todo
	Class *CodeableConcept `json:"class,omitempty"`
	// Todo
	Geometry *CodeableConcept `json:"geometry,omitempty"`
	// Todo
	CopolymerConnectivity []CodeableConcept `json:"copolymerConnectivity,omitempty"`
	// Todo
	Modification []string `json:"modification,omitempty"`
	// Todo
	MonomerSet []SubstancePolymerMonomerSet `json:"monomerSet,omitempty"`
	// Todo
	Repeat []SubstancePolymerRepeat `json:"repeat,omitempty"`
}
