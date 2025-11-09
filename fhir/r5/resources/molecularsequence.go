package resources

// ResourceTypeMolecularSequence is the FHIR resource type name for MolecularSequence.
const ResourceTypeMolecularSequence = "MolecularSequence"

// MolecularSequenceRelativeStartingSequence represents a FHIR BackboneElement for MolecularSequence.relative.startingSequence.
type MolecularSequenceRelativeStartingSequence struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The genome assembly used for starting sequence, e.g. GRCh38
	GenomeAssembly *CodeableConcept `json:"genomeAssembly,omitempty"`
	// Chromosome Identifier
	Chromosome *CodeableConcept `json:"chromosome,omitempty"`
	// The reference sequence that represents the starting sequence
	Sequence *any `json:"sequence,omitempty"`
	// Start position of the window on the starting sequence
	WindowStart *int `json:"windowStart,omitempty"`
	// End position of the window on the starting sequence
	WindowEnd *int `json:"windowEnd,omitempty"`
	// sense | antisense
	Orientation *string `json:"orientation,omitempty"`
	// watson | crick
	Strand *string `json:"strand,omitempty"`
}

// MolecularSequenceRelativeEdit represents a FHIR BackboneElement for MolecularSequence.relative.edit.
type MolecularSequenceRelativeEdit struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Start position of the edit on the starting sequence
	Start *int `json:"start,omitempty"`
	// End position of the edit on the starting sequence
	End *int `json:"end,omitempty"`
	// Allele that was observed
	ReplacementSequence *string `json:"replacementSequence,omitempty"`
	// Allele in the starting sequence
	ReplacedSequence *string `json:"replacedSequence,omitempty"`
}

// MolecularSequenceRelative represents a FHIR BackboneElement for MolecularSequence.relative.
type MolecularSequenceRelative struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Ways of identifying nucleotides or amino acids within a sequence
	CoordinateSystem CodeableConcept `json:"coordinateSystem"`
	// Indicates the order in which the sequence should be considered when putting multiple 'relative' elements together
	OrdinalPosition *int `json:"ordinalPosition,omitempty"`
	// Indicates the nucleotide range in the composed sequence when multiple 'relative' elements are used together
	SequenceRange *Range `json:"sequenceRange,omitempty"`
	// A sequence used as starting sequence
	StartingSequence *MolecularSequenceRelativeStartingSequence `json:"startingSequence,omitempty"`
	// Changes in sequence from the starting sequence
	Edit []MolecularSequenceRelativeEdit `json:"edit,omitempty"`
}

// MolecularSequence represents a FHIR MolecularSequence.
type MolecularSequence struct {
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
	// Unique ID for this particular sequence
	Identifier []Identifier `json:"identifier,omitempty"`
	// aa | dna | rna
	Type *string `json:"type,omitempty"`
	// Subject this sequence is associated too
	Subject *Reference `json:"subject,omitempty"`
	// What the molecular sequence is about, when it is not about the subject of record
	Focus []Reference `json:"focus,omitempty"`
	// Specimen used for sequencing
	Specimen *Reference `json:"specimen,omitempty"`
	// The method for sequencing
	Device *Reference `json:"device,omitempty"`
	// Who should be responsible for test result
	Performer *Reference `json:"performer,omitempty"`
	// Sequence that was observed
	Literal *string `json:"literal,omitempty"`
	// Embedded file or a link (URL) which contains content to represent the sequence
	Formatted []Attachment `json:"formatted,omitempty"`
	// A sequence defined relative to another sequence
	Relative []MolecularSequenceRelative `json:"relative,omitempty"`
}
