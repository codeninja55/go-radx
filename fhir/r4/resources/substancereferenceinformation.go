package resources

// ResourceTypeSubstanceReferenceInformation is the FHIR resource type name for SubstanceReferenceInformation.
const ResourceTypeSubstanceReferenceInformation = "SubstanceReferenceInformation"

// SubstanceReferenceInformationGene represents a FHIR BackboneElement for SubstanceReferenceInformation.gene.
type SubstanceReferenceInformationGene struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Todo
	GeneSequenceOrigin *CodeableConcept `json:"geneSequenceOrigin,omitempty"`
	// Todo
	Gene *CodeableConcept `json:"gene,omitempty"`
	// Todo
	Source []Reference `json:"source,omitempty"`
}

// SubstanceReferenceInformationGeneElement represents a FHIR BackboneElement for SubstanceReferenceInformation.geneElement.
type SubstanceReferenceInformationGeneElement struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Todo
	Type *CodeableConcept `json:"type,omitempty"`
	// Todo
	Element *Identifier `json:"element,omitempty"`
	// Todo
	Source []Reference `json:"source,omitempty"`
}

// SubstanceReferenceInformationClassification represents a FHIR BackboneElement for SubstanceReferenceInformation.classification.
type SubstanceReferenceInformationClassification struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Todo
	Domain *CodeableConcept `json:"domain,omitempty"`
	// Todo
	Classification *CodeableConcept `json:"classification,omitempty"`
	// Todo
	Subtype []CodeableConcept `json:"subtype,omitempty"`
	// Todo
	Source []Reference `json:"source,omitempty"`
}

// SubstanceReferenceInformationTarget represents a FHIR BackboneElement for SubstanceReferenceInformation.target.
type SubstanceReferenceInformationTarget struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Todo
	Target *Identifier `json:"target,omitempty"`
	// Todo
	Type *CodeableConcept `json:"type,omitempty"`
	// Todo
	Interaction *CodeableConcept `json:"interaction,omitempty"`
	// Todo
	Organism *CodeableConcept `json:"organism,omitempty"`
	// Todo
	OrganismType *CodeableConcept `json:"organismType,omitempty"`
	// Todo
	Amount *any `json:"amount,omitempty"`
	// Todo
	AmountType *CodeableConcept `json:"amountType,omitempty"`
	// Todo
	Source []Reference `json:"source,omitempty"`
}

// SubstanceReferenceInformation represents a FHIR SubstanceReferenceInformation.
type SubstanceReferenceInformation struct {
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
	Comment *string `json:"comment,omitempty"`
	// Todo
	Gene []SubstanceReferenceInformationGene `json:"gene,omitempty"`
	// Todo
	GeneElement []SubstanceReferenceInformationGeneElement `json:"geneElement,omitempty"`
	// Todo
	Classification []SubstanceReferenceInformationClassification `json:"classification,omitempty"`
	// Todo
	Target []SubstanceReferenceInformationTarget `json:"target,omitempty"`
}
