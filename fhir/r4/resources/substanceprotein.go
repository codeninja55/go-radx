package resources

// ResourceTypeSubstanceProtein is the FHIR resource type name for SubstanceProtein.
const ResourceTypeSubstanceProtein = "SubstanceProtein"

// SubstanceProteinSubunit represents a FHIR BackboneElement for SubstanceProtein.subunit.
type SubstanceProteinSubunit struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Index of primary sequences of amino acids linked through peptide bonds in order of decreasing length. Sequences of the same length will be ordered by molecular weight. Subunits that have identical sequences will be repeated and have sequential subscripts
	Subunit *int `json:"subunit,omitempty"`
	// The sequence information shall be provided enumerating the amino acids from N- to C-terminal end using standard single-letter amino acid codes. Uppercase shall be used for L-amino acids and lowercase for D-amino acids. Transcribed SubstanceProteins will always be described using the translated sequence; for synthetic peptide containing amino acids that are not represented with a single letter code an X should be used within the sequence. The modified amino acids will be distinguished by their position in the sequence
	Sequence *string `json:"sequence,omitempty"`
	// Length of linear sequences of amino acids contained in the subunit
	Length *int `json:"length,omitempty"`
	// The sequence information shall be provided enumerating the amino acids from N- to C-terminal end using standard single-letter amino acid codes. Uppercase shall be used for L-amino acids and lowercase for D-amino acids. Transcribed SubstanceProteins will always be described using the translated sequence; for synthetic peptide containing amino acids that are not represented with a single letter code an X should be used within the sequence. The modified amino acids will be distinguished by their position in the sequence
	SequenceAttachment *Attachment `json:"sequenceAttachment,omitempty"`
	// Unique identifier for molecular fragment modification based on the ISO 11238 Substance ID
	NTerminalModificationId *Identifier `json:"nTerminalModificationId,omitempty"`
	// The name of the fragment modified at the N-terminal of the SubstanceProtein shall be specified
	NTerminalModification *string `json:"nTerminalModification,omitempty"`
	// Unique identifier for molecular fragment modification based on the ISO 11238 Substance ID
	CTerminalModificationId *Identifier `json:"cTerminalModificationId,omitempty"`
	// The modification at the C-terminal shall be specified
	CTerminalModification *string `json:"cTerminalModification,omitempty"`
}

// SubstanceProtein represents a FHIR SubstanceProtein.
type SubstanceProtein struct {
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
	// The SubstanceProtein descriptive elements will only be used when a complete or partial amino acid sequence is available or derivable from a nucleic acid sequence
	SequenceType *CodeableConcept `json:"sequenceType,omitempty"`
	// Number of linear sequences of amino acids linked through peptide bonds. The number of subunits constituting the SubstanceProtein shall be described. It is possible that the number of subunits can be variable
	NumberOfSubunits *int `json:"numberOfSubunits,omitempty"`
	// The disulphide bond between two cysteine residues either on the same subunit or on two different subunits shall be described. The position of the disulfide bonds in the SubstanceProtein shall be listed in increasing order of subunit number and position within subunit followed by the abbreviation of the amino acids involved. The disulfide linkage positions shall actually contain the amino acid Cysteine at the respective positions
	DisulfideLinkage []string `json:"disulfideLinkage,omitempty"`
	// This subclause refers to the description of each subunit constituting the SubstanceProtein. A subunit is a linear sequence of amino acids linked through peptide bonds. The Subunit information shall be provided when the finished SubstanceProtein is a complex of multiple sequences; subunits are not used to delineate domains within a single sequence. Subunits are listed in order of decreasing length; sequences of the same length will be ordered by decreasing molecular weight; subunits that have identical sequences will be repeated multiple times
	Subunit []SubstanceProteinSubunit `json:"subunit,omitempty"`
}
