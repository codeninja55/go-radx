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
	// The type of substance for this starting material
	Code *CodeableConcept `json:"code,omitempty"`
	// Substance high level category, e.g. chemical substance
	Category *CodeableConcept `json:"category,omitempty"`
	// Used to specify whether the attribute described is a defining element for the unique identification of the polymer
	IsDefining *bool `json:"isDefining,omitempty"`
	// A percentage
	Amount *Quantity `json:"amount,omitempty"`
}

// SubstancePolymerMonomerSet represents a FHIR BackboneElement for SubstancePolymer.monomerSet.
type SubstancePolymerMonomerSet struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Captures the type of ratio to the entire polymer, e.g. Monomer/Polymer ratio, SRU/Polymer Ratio
	RatioType *CodeableConcept `json:"ratioType,omitempty"`
	// The starting materials - monomer(s) used in the synthesis of the polymer
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
	// The type of the degree of polymerisation shall be described, e.g. SRU/Polymer Ratio
	Type *CodeableConcept `json:"type,omitempty"`
	// An average amount of polymerisation
	Average *int `json:"average,omitempty"`
	// A low expected limit of the amount
	Low *int `json:"low,omitempty"`
	// A high expected limit of the amount
	High *int `json:"high,omitempty"`
}

// SubstancePolymerRepeatRepeatUnitStructuralRepresentation represents a FHIR BackboneElement for SubstancePolymer.repeat.repeatUnit.structuralRepresentation.
type SubstancePolymerRepeatRepeatUnitStructuralRepresentation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The type of structure (e.g. Full, Partial, Representative)
	Type *CodeableConcept `json:"type,omitempty"`
	// The structural representation as text string in a standard format e.g. InChI, SMILES, MOLFILE, CDX, SDF, PDB, mmCIF
	Representation *string `json:"representation,omitempty"`
	// The format of the representation e.g. InChI, SMILES, MOLFILE, CDX, SDF, PDB, mmCIF
	Format *CodeableConcept `json:"format,omitempty"`
	// An attached file with the structural representation
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
	// Structural repeat units are essential elements for defining polymers
	Unit *string `json:"unit,omitempty"`
	// The orientation of the polymerisation, e.g. head-tail, head-head, random
	Orientation *CodeableConcept `json:"orientation,omitempty"`
	// Number of repeats of this unit
	Amount *int `json:"amount,omitempty"`
	// Applies to homopolymer and block co-polymers where the degree of polymerisation within a block can be described
	DegreeOfPolymerisation []SubstancePolymerRepeatRepeatUnitDegreeOfPolymerisation `json:"degreeOfPolymerisation,omitempty"`
	// A graphical structure for this SRU
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
	// A representation of an (average) molecular formula from a polymer
	AverageMolecularFormula *string `json:"averageMolecularFormula,omitempty"`
	// How the quantitative amount of Structural Repeat Units is captured (e.g. Exact, Numeric, Average)
	RepeatUnitAmountType *CodeableConcept `json:"repeatUnitAmountType,omitempty"`
	// An SRU - Structural Repeat Unit
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
	// A business idenfier for this polymer, but typically this is handled by a SubstanceDefinition identifier
	Identifier *Identifier `json:"identifier,omitempty"`
	// Overall type of the polymer
	Class *CodeableConcept `json:"class,omitempty"`
	// Polymer geometry, e.g. linear, branched, cross-linked, network or dendritic
	Geometry *CodeableConcept `json:"geometry,omitempty"`
	// Descrtibes the copolymer sequence type (polymer connectivity)
	CopolymerConnectivity []CodeableConcept `json:"copolymerConnectivity,omitempty"`
	// Todo - this is intended to connect to a repeating full modification structure, also used by Protein and Nucleic Acid . String is just a placeholder
	Modification *string `json:"modification,omitempty"`
	// Todo
	MonomerSet []SubstancePolymerMonomerSet `json:"monomerSet,omitempty"`
	// Specifies and quantifies the repeated units and their configuration
	Repeat []SubstancePolymerRepeat `json:"repeat,omitempty"`
}
