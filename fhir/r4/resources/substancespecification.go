package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeSubstanceSpecification is the FHIR resource type name for SubstanceSpecification.
const ResourceTypeSubstanceSpecification = "SubstanceSpecification"

// SubstanceSpecificationMoiety represents a FHIR BackboneElement for SubstanceSpecification.moiety.
type SubstanceSpecificationMoiety struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Role that the moiety is playing
	Role *CodeableConcept `json:"role,omitempty"`
	// Identifier by which this moiety substance is known
	Identifier *Identifier `json:"identifier,omitempty"`
	// Textual name for this moiety substance
	Name *string `json:"name,omitempty"`
	// Stereochemistry type
	Stereochemistry *CodeableConcept `json:"stereochemistry,omitempty"`
	// Optical activity type
	OpticalActivity *CodeableConcept `json:"opticalActivity,omitempty"`
	// Molecular formula
	MolecularFormula *string `json:"molecularFormula,omitempty"`
	// Quantitative value for this moiety
	Amount *any `json:"amount,omitempty"`
}

// SubstanceSpecificationProperty represents a FHIR BackboneElement for SubstanceSpecification.property.
type SubstanceSpecificationProperty struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A category for this property, e.g. Physical, Chemical, Enzymatic
	Category *CodeableConcept `json:"category,omitempty"`
	// Property type e.g. viscosity, pH, isoelectric point
	Code *CodeableConcept `json:"code,omitempty"`
	// Parameters that were used in the measurement of a property (e.g. for viscosity: measured at 20C with a pH of 7.1)
	Parameters *string `json:"parameters,omitempty"`
	// A substance upon which a defining property depends (e.g. for solubility: in water, in alcohol)
	DefiningSubstance *any `json:"definingSubstance,omitempty"`
	// Quantitative value for this property
	Amount *any `json:"amount,omitempty"`
}

// SubstanceSpecificationStructureIsotopeMolecularWeight represents a FHIR BackboneElement for SubstanceSpecification.structure.isotope.molecularWeight.
type SubstanceSpecificationStructureIsotopeMolecularWeight struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The method by which the molecular weight was determined
	Method *CodeableConcept `json:"method,omitempty"`
	// Type of molecular weight such as exact, average (also known as. number average), weight average
	Type *CodeableConcept `json:"type,omitempty"`
	// Used to capture quantitative values for a variety of elements. If only limits are given, the arithmetic mean would be the average. If only a single definite value for a given element is given, it would be captured in this field
	Amount *Quantity `json:"amount,omitempty"`
}

// SubstanceSpecificationStructureIsotope represents a FHIR BackboneElement for SubstanceSpecification.structure.isotope.
type SubstanceSpecificationStructureIsotope struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Substance identifier for each non-natural or radioisotope
	Identifier *Identifier `json:"identifier,omitempty"`
	// Substance name for each non-natural or radioisotope
	Name *CodeableConcept `json:"name,omitempty"`
	// The type of isotopic substitution present in a single substance
	Substitution *CodeableConcept `json:"substitution,omitempty"`
	// Half life - for a non-natural nuclide
	HalfLife *Quantity `json:"halfLife,omitempty"`
	// The molecular weight or weight range (for proteins, polymers or nucleic acids)
	MolecularWeight *SubstanceSpecificationStructureIsotopeMolecularWeight `json:"molecularWeight,omitempty"`
}

// SubstanceSpecificationStructureMolecularWeight represents a FHIR BackboneElement for SubstanceSpecification.structure.molecularWeight.
type SubstanceSpecificationStructureMolecularWeight struct {
}

// SubstanceSpecificationStructureRepresentation represents a FHIR BackboneElement for SubstanceSpecification.structure.representation.
type SubstanceSpecificationStructureRepresentation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The type of structure (e.g. Full, Partial, Representative)
	Type *CodeableConcept `json:"type,omitempty"`
	// The structural representation as text string in a format e.g. InChI, SMILES, MOLFILE, CDX
	Representation *string `json:"representation,omitempty"`
	// An attached file with the structural representation
	Attachment *Attachment `json:"attachment,omitempty"`
}

// SubstanceSpecificationStructure represents a FHIR BackboneElement for SubstanceSpecification.structure.
type SubstanceSpecificationStructure struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Stereochemistry type
	Stereochemistry *CodeableConcept `json:"stereochemistry,omitempty"`
	// Optical activity type
	OpticalActivity *CodeableConcept `json:"opticalActivity,omitempty"`
	// Molecular formula
	MolecularFormula *string `json:"molecularFormula,omitempty"`
	// Specified per moiety according to the Hill system, i.e. first C, then H, then alphabetical, each moiety separated by a dot
	MolecularFormulaByMoiety *string `json:"molecularFormulaByMoiety,omitempty"`
	// Applicable for single substances that contain a radionuclide or a non-natural isotopic ratio
	Isotope []SubstanceSpecificationStructureIsotope `json:"isotope,omitempty"`
	// The molecular weight or weight range (for proteins, polymers or nucleic acids)
	MolecularWeight *SubstanceSpecificationStructureMolecularWeight `json:"molecularWeight,omitempty"`
	// Supporting literature
	Source []Reference `json:"source,omitempty"`
	// Molecular structural representation
	Representation []SubstanceSpecificationStructureRepresentation `json:"representation,omitempty"`
}

// SubstanceSpecificationCode represents a FHIR BackboneElement for SubstanceSpecification.code.
type SubstanceSpecificationCode struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The specific code
	Code *CodeableConcept `json:"code,omitempty"`
	// Status of the code assignment
	Status *CodeableConcept `json:"status,omitempty"`
	// The date at which the code status is changed as part of the terminology maintenance
	StatusDate *primitives.DateTime `json:"statusDate,omitempty"`
	// Any comment can be provided in this field, if necessary
	Comment *string `json:"comment,omitempty"`
	// Supporting literature
	Source []Reference `json:"source,omitempty"`
}

// SubstanceSpecificationNameSynonym represents a FHIR BackboneElement for SubstanceSpecification.name.synonym.
type SubstanceSpecificationNameSynonym struct {
}

// SubstanceSpecificationNameTranslation represents a FHIR BackboneElement for SubstanceSpecification.name.translation.
type SubstanceSpecificationNameTranslation struct {
}

// SubstanceSpecificationNameOfficial represents a FHIR BackboneElement for SubstanceSpecification.name.official.
type SubstanceSpecificationNameOfficial struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Which authority uses this official name
	Authority *CodeableConcept `json:"authority,omitempty"`
	// The status of the official name
	Status *CodeableConcept `json:"status,omitempty"`
	// Date of official name change
	Date *primitives.DateTime `json:"date,omitempty"`
}

// SubstanceSpecificationName represents a FHIR BackboneElement for SubstanceSpecification.name.
type SubstanceSpecificationName struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The actual name
	Name string `json:"name"`
	// Name type
	Type *CodeableConcept `json:"type,omitempty"`
	// The status of the name
	Status *CodeableConcept `json:"status,omitempty"`
	// If this is the preferred name for this substance
	Preferred *bool `json:"preferred,omitempty"`
	// Language of the name
	Language []CodeableConcept `json:"language,omitempty"`
	// The use context of this name for example if there is a different name a drug active ingredient as opposed to a food colour additive
	Domain []CodeableConcept `json:"domain,omitempty"`
	// The jurisdiction where this name applies
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// A synonym of this name
	Synonym []SubstanceSpecificationNameSynonym `json:"synonym,omitempty"`
	// A translation for this name
	Translation []SubstanceSpecificationNameTranslation `json:"translation,omitempty"`
	// Details of the official nature of this name
	Official []SubstanceSpecificationNameOfficial `json:"official,omitempty"`
	// Supporting literature
	Source []Reference `json:"source,omitempty"`
}

// SubstanceSpecificationMolecularWeight represents a FHIR BackboneElement for SubstanceSpecification.molecularWeight.
type SubstanceSpecificationMolecularWeight struct {
}

// SubstanceSpecificationRelationship represents a FHIR BackboneElement for SubstanceSpecification.relationship.
type SubstanceSpecificationRelationship struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A pointer to another substance, as a resource or just a representational code
	Substance *any `json:"substance,omitempty"`
	// For example "salt to parent", "active moiety", "starting material"
	Relationship *CodeableConcept `json:"relationship,omitempty"`
	// For example where an enzyme strongly bonds with a particular substance, this is a defining relationship for that enzyme, out of several possible substance relationships
	IsDefining *bool `json:"isDefining,omitempty"`
	// A numeric factor for the relationship, for instance to express that the salt of a substance has some percentage of the active substance in relation to some other
	Amount *any `json:"amount,omitempty"`
	// For use when the numeric
	AmountRatioLowLimit *Ratio `json:"amountRatioLowLimit,omitempty"`
	// An operator for the amount, for example "average", "approximately", "less than"
	AmountType *CodeableConcept `json:"amountType,omitempty"`
	// Supporting literature
	Source []Reference `json:"source,omitempty"`
}

// SubstanceSpecification represents a FHIR SubstanceSpecification.
type SubstanceSpecification struct {
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
	// Identifier by which this substance is known
	Identifier *Identifier `json:"identifier,omitempty"`
	// High level categorization, e.g. polymer or nucleic acid
	Type *CodeableConcept `json:"type,omitempty"`
	// Status of substance within the catalogue e.g. approved
	Status *CodeableConcept `json:"status,omitempty"`
	// If the substance applies to only human or veterinary use
	Domain *CodeableConcept `json:"domain,omitempty"`
	// Textual description of the substance
	Description *string `json:"description,omitempty"`
	// Supporting literature
	Source []Reference `json:"source,omitempty"`
	// Textual comment about this record of a substance
	Comment *string `json:"comment,omitempty"`
	// Moiety, for structural modifications
	Moiety []SubstanceSpecificationMoiety `json:"moiety,omitempty"`
	// General specifications for this substance, including how it is related to other substances
	Property []SubstanceSpecificationProperty `json:"property,omitempty"`
	// General information detailing this substance
	ReferenceInformation *Reference `json:"referenceInformation,omitempty"`
	// Structural information
	Structure *SubstanceSpecificationStructure `json:"structure,omitempty"`
	// Codes associated with the substance
	Code []SubstanceSpecificationCode `json:"code,omitempty"`
	// Names applicable to this substance
	Name []SubstanceSpecificationName `json:"name,omitempty"`
	// The molecular weight or weight range (for proteins, polymers or nucleic acids)
	MolecularWeight []SubstanceSpecificationMolecularWeight `json:"molecularWeight,omitempty"`
	// A link between this substance and another, with details of the relationship
	Relationship []SubstanceSpecificationRelationship `json:"relationship,omitempty"`
	// Data items specific to nucleic acids
	NucleicAcid *Reference `json:"nucleicAcid,omitempty"`
	// Data items specific to polymers
	Polymer *Reference `json:"polymer,omitempty"`
	// Data items specific to proteins
	Protein *Reference `json:"protein,omitempty"`
	// Material or taxonomic/anatomical source for the substance
	SourceMaterial *Reference `json:"sourceMaterial,omitempty"`
}
