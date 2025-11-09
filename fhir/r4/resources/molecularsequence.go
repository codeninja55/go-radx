package resources

// ResourceTypeMolecularSequence is the FHIR resource type name for MolecularSequence.
const ResourceTypeMolecularSequence = "MolecularSequence"

// MolecularSequenceReferenceSeq represents a FHIR BackboneElement for MolecularSequence.referenceSeq.
type MolecularSequenceReferenceSeq struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Chromosome containing genetic finding
	Chromosome *CodeableConcept `json:"chromosome,omitempty"`
	// The Genome Build used for reference, following GRCh build versions e.g. 'GRCh 37'
	GenomeBuild *string `json:"genomeBuild,omitempty"`
	// sense | antisense
	Orientation *string `json:"orientation,omitempty"`
	// Reference identifier
	ReferenceSeqId *CodeableConcept `json:"referenceSeqId,omitempty"`
	// A pointer to another MolecularSequence entity as reference sequence
	ReferenceSeqPointer *Reference `json:"referenceSeqPointer,omitempty"`
	// A string to represent reference sequence
	ReferenceSeqString *string `json:"referenceSeqString,omitempty"`
	// watson | crick
	Strand *string `json:"strand,omitempty"`
	// Start position of the window on the  reference sequence
	WindowStart *int `json:"windowStart,omitempty"`
	// End position of the window on the reference sequence
	WindowEnd *int `json:"windowEnd,omitempty"`
}

// MolecularSequenceVariant represents a FHIR BackboneElement for MolecularSequence.variant.
type MolecularSequenceVariant struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Start position of the variant on the  reference sequence
	Start *int `json:"start,omitempty"`
	// End position of the variant on the reference sequence
	End *int `json:"end,omitempty"`
	// Allele that was observed
	ObservedAllele *string `json:"observedAllele,omitempty"`
	// Allele in the reference sequence
	ReferenceAllele *string `json:"referenceAllele,omitempty"`
	// Extended CIGAR string for aligning the sequence with reference bases
	Cigar *string `json:"cigar,omitempty"`
	// Pointer to observed variant information
	VariantPointer *Reference `json:"variantPointer,omitempty"`
}

// MolecularSequenceQualityRoc represents a FHIR BackboneElement for MolecularSequence.quality.roc.
type MolecularSequenceQualityRoc struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Genotype quality score
	Score []int `json:"score,omitempty"`
	// Roc score true positive numbers
	NumTP []int `json:"numTP,omitempty"`
	// Roc score false positive numbers
	NumFP []int `json:"numFP,omitempty"`
	// Roc score false negative numbers
	NumFN []int `json:"numFN,omitempty"`
	// Precision of the GQ score
	Precision []float64 `json:"precision,omitempty"`
	// Sensitivity of the GQ score
	Sensitivity []float64 `json:"sensitivity,omitempty"`
	// FScore of the GQ score
	FMeasure []float64 `json:"fMeasure,omitempty"`
}

// MolecularSequenceQuality represents a FHIR BackboneElement for MolecularSequence.quality.
type MolecularSequenceQuality struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// indel | snp | unknown
	Type string `json:"type"`
	// Standard sequence for comparison
	StandardSequence *CodeableConcept `json:"standardSequence,omitempty"`
	// Start position of the sequence
	Start *int `json:"start,omitempty"`
	// End position of the sequence
	End *int `json:"end,omitempty"`
	// Quality score for the comparison
	Score *Quantity `json:"score,omitempty"`
	// Method to get quality
	Method *CodeableConcept `json:"method,omitempty"`
	// True positives from the perspective of the truth data
	TruthTP *float64 `json:"truthTP,omitempty"`
	// True positives from the perspective of the query data
	QueryTP *float64 `json:"queryTP,omitempty"`
	// False negatives
	TruthFN *float64 `json:"truthFN,omitempty"`
	// False positives
	QueryFP *float64 `json:"queryFP,omitempty"`
	// False positives where the non-REF alleles in the Truth and Query Call Sets match
	GtFP *float64 `json:"gtFP,omitempty"`
	// Precision of comparison
	Precision *float64 `json:"precision,omitempty"`
	// Recall of comparison
	Recall *float64 `json:"recall,omitempty"`
	// F-score
	FScore *float64 `json:"fScore,omitempty"`
	// Receiver Operator Characteristic (ROC) Curve
	Roc *MolecularSequenceQualityRoc `json:"roc,omitempty"`
}

// MolecularSequenceRepository represents a FHIR BackboneElement for MolecularSequence.repository.
type MolecularSequenceRepository struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// directlink | openapi | login | oauth | other
	Type string `json:"type"`
	// URI of the repository
	URL *string `json:"url,omitempty"`
	// Repository's name
	Name *string `json:"name,omitempty"`
	// Id of the dataset that used to call for dataset in repository
	DatasetId *string `json:"datasetId,omitempty"`
	// Id of the variantset that used to call for variantset in repository
	VariantsetId *string `json:"variantsetId,omitempty"`
	// Id of the read
	ReadsetId *string `json:"readsetId,omitempty"`
}

// MolecularSequenceStructureVariantOuter represents a FHIR BackboneElement for MolecularSequence.structureVariant.outer.
type MolecularSequenceStructureVariantOuter struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Structural variant outer start
	Start *int `json:"start,omitempty"`
	// Structural variant outer end
	End *int `json:"end,omitempty"`
}

// MolecularSequenceStructureVariantInner represents a FHIR BackboneElement for MolecularSequence.structureVariant.inner.
type MolecularSequenceStructureVariantInner struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Structural variant inner start
	Start *int `json:"start,omitempty"`
	// Structural variant inner end
	End *int `json:"end,omitempty"`
}

// MolecularSequenceStructureVariant represents a FHIR BackboneElement for MolecularSequence.structureVariant.
type MolecularSequenceStructureVariant struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Structural variant change type
	VariantType *CodeableConcept `json:"variantType,omitempty"`
	// Does the structural variant have base pair resolution breakpoints?
	Exact *bool `json:"exact,omitempty"`
	// Structural variant length
	Length *int `json:"length,omitempty"`
	// Structural variant outer
	Outer *MolecularSequenceStructureVariantOuter `json:"outer,omitempty"`
	// Structural variant inner
	Inner *MolecularSequenceStructureVariantInner `json:"inner,omitempty"`
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
	// Unique ID for this particular sequence. This is a FHIR-defined id
	Identifier []Identifier `json:"identifier,omitempty"`
	// aa | dna | rna
	Type *string `json:"type,omitempty"`
	// Base number of coordinate system (0 for 0-based numbering or coordinates, inclusive start, exclusive end, 1 for 1-based numbering, inclusive start, inclusive end)
	CoordinateSystem int `json:"coordinateSystem"`
	// Who and/or what this is about
	Patient *Reference `json:"patient,omitempty"`
	// Specimen used for sequencing
	Specimen *Reference `json:"specimen,omitempty"`
	// The method for sequencing
	Device *Reference `json:"device,omitempty"`
	// Who should be responsible for test result
	Performer *Reference `json:"performer,omitempty"`
	// The number of copies of the sequence of interest.  (RNASeq)
	Quantity *Quantity `json:"quantity,omitempty"`
	// A sequence used as reference
	ReferenceSeq *MolecularSequenceReferenceSeq `json:"referenceSeq,omitempty"`
	// Variant in sequence
	Variant []MolecularSequenceVariant `json:"variant,omitempty"`
	// Sequence that was observed
	ObservedSeq *string `json:"observedSeq,omitempty"`
	// An set of value as quality of sequence
	Quality []MolecularSequenceQuality `json:"quality,omitempty"`
	// Average number of reads representing a given nucleotide in the reconstructed sequence
	ReadCoverage *int `json:"readCoverage,omitempty"`
	// External repository which contains detailed report related with observedSeq in this resource
	Repository []MolecularSequenceRepository `json:"repository,omitempty"`
	// Pointer to next atomic sequence
	Pointer []Reference `json:"pointer,omitempty"`
	// Structural variant
	StructureVariant []MolecularSequenceStructureVariant `json:"structureVariant,omitempty"`
}
