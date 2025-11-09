package resources

// ResourceTypeSubstanceNucleicAcid is the FHIR resource type name for SubstanceNucleicAcid.
const ResourceTypeSubstanceNucleicAcid = "SubstanceNucleicAcid"

// SubstanceNucleicAcidSubunitLinkage represents a FHIR BackboneElement for SubstanceNucleicAcid.subunit.linkage.
type SubstanceNucleicAcidSubunitLinkage struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The entity that links the sugar residues together should also be captured for nearly all naturally occurring nucleic acid the linkage is a phosphate group. For many synthetic oligonucleotides phosphorothioate linkages are often seen. Linkage connectivity is assumed to be 3’-5’. If the linkage is either 3’-3’ or 5’-5’ this should be specified
	Connectivity *string `json:"connectivity,omitempty"`
	// Each linkage will be registered as a fragment and have an ID
	Identifier *Identifier `json:"identifier,omitempty"`
	// Each linkage will be registered as a fragment and have at least one name. A single name shall be assigned to each linkage
	Name *string `json:"name,omitempty"`
	// Residues shall be captured as described in 5.3.6.8.3
	ResidueSite *string `json:"residueSite,omitempty"`
}

// SubstanceNucleicAcidSubunitSugar represents a FHIR BackboneElement for SubstanceNucleicAcid.subunit.sugar.
type SubstanceNucleicAcidSubunitSugar struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The Substance ID of the sugar or sugar-like component that make up the nucleotide
	Identifier *Identifier `json:"identifier,omitempty"`
	// The name of the sugar or sugar-like component that make up the nucleotide
	Name *string `json:"name,omitempty"`
	// The residues that contain a given sugar will be captured. The order of given residues will be captured in the 5‘-3‘direction consistent with the base sequences listed above
	ResidueSite *string `json:"residueSite,omitempty"`
}

// SubstanceNucleicAcidSubunit represents a FHIR BackboneElement for SubstanceNucleicAcid.subunit.
type SubstanceNucleicAcidSubunit struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Index of linear sequences of nucleic acids in order of decreasing length. Sequences of the same length will be ordered by molecular weight. Subunits that have identical sequences will be repeated and have sequential subscripts
	Subunit *int `json:"subunit,omitempty"`
	// Actual nucleotide sequence notation from 5' to 3' end using standard single letter codes. In addition to the base sequence, sugar and type of phosphate or non-phosphate linkage should also be captured
	Sequence *string `json:"sequence,omitempty"`
	// The length of the sequence shall be captured
	Length *int `json:"length,omitempty"`
	// (TBC)
	SequenceAttachment *Attachment `json:"sequenceAttachment,omitempty"`
	// The nucleotide present at the 5’ terminal shall be specified based on a controlled vocabulary. Since the sequence is represented from the 5' to the 3' end, the 5’ prime nucleotide is the letter at the first position in the sequence. A separate representation would be redundant
	FivePrime *CodeableConcept `json:"fivePrime,omitempty"`
	// The nucleotide present at the 3’ terminal shall be specified based on a controlled vocabulary. Since the sequence is represented from the 5' to the 3' end, the 5’ prime nucleotide is the letter at the last position in the sequence. A separate representation would be redundant
	ThreePrime *CodeableConcept `json:"threePrime,omitempty"`
	// The linkages between sugar residues will also be captured
	Linkage []SubstanceNucleicAcidSubunitLinkage `json:"linkage,omitempty"`
	// 5.3.6.8.1 Sugar ID (Mandatory)
	Sugar []SubstanceNucleicAcidSubunitSugar `json:"sugar,omitempty"`
}

// SubstanceNucleicAcid represents a FHIR SubstanceNucleicAcid.
type SubstanceNucleicAcid struct {
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
	// The type of the sequence shall be specified based on a controlled vocabulary
	SequenceType *CodeableConcept `json:"sequenceType,omitempty"`
	// The number of linear sequences of nucleotides linked through phosphodiester bonds shall be described. Subunits would be strands of nucleic acids that are tightly associated typically through Watson-Crick base pairing. NOTE: If not specified in the reference source, the assumption is that there is 1 subunit
	NumberOfSubunits *int `json:"numberOfSubunits,omitempty"`
	// The area of hybridisation shall be described if applicable for double stranded RNA or DNA. The number associated with the subunit followed by the number associated to the residue shall be specified in increasing order. The underscore “” shall be used as separator as follows: “Subunitnumber Residue”
	AreaOfHybridisation *string `json:"areaOfHybridisation,omitempty"`
	// (TBC)
	OligoNucleotideType *CodeableConcept `json:"oligoNucleotideType,omitempty"`
	// Subunits are listed in order of decreasing length; sequences of the same length will be ordered by molecular weight; subunits that have identical sequences will be repeated multiple times
	Subunit []SubstanceNucleicAcidSubunit `json:"subunit,omitempty"`
}
