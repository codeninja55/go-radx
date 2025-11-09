package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeGenomicStudy is the FHIR resource type name for GenomicStudy.
const ResourceTypeGenomicStudy = "GenomicStudy"

// GenomicStudyAnalysisInput represents a FHIR BackboneElement for GenomicStudy.analysis.input.
type GenomicStudyAnalysisInput struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// File containing input data
	File *Reference `json:"file,omitempty"`
	// Type of input data (e.g., BAM, CRAM, or FASTA)
	Type *CodeableConcept `json:"type,omitempty"`
	// The analysis event or other GenomicStudy that generated this input file
	GeneratedBy *any `json:"generatedBy,omitempty"`
}

// GenomicStudyAnalysisOutput represents a FHIR BackboneElement for GenomicStudy.analysis.output.
type GenomicStudyAnalysisOutput struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// File containing output data
	File *Reference `json:"file,omitempty"`
	// Type of output data (e.g., VCF, MAF, or BAM)
	Type *CodeableConcept `json:"type,omitempty"`
}

// GenomicStudyAnalysisPerformer represents a FHIR BackboneElement for GenomicStudy.analysis.performer.
type GenomicStudyAnalysisPerformer struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The organization, healthcare professional, or others who participated in performing this analysis
	Actor *Reference `json:"actor,omitempty"`
	// Role of the actor for this analysis
	Role *CodeableConcept `json:"role,omitempty"`
}

// GenomicStudyAnalysisDevice represents a FHIR BackboneElement for GenomicStudy.analysis.device.
type GenomicStudyAnalysisDevice struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Device used for the analysis
	Device *Reference `json:"device,omitempty"`
	// Specific function for the device used for the analysis
	Function *CodeableConcept `json:"function,omitempty"`
}

// GenomicStudyAnalysis represents a FHIR BackboneElement for GenomicStudy.analysis.
type GenomicStudyAnalysis struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Identifiers for the analysis event
	Identifier []Identifier `json:"identifier,omitempty"`
	// Type of the methods used in the analysis (e.g., FISH, Karyotyping, MSI)
	MethodType []CodeableConcept `json:"methodType,omitempty"`
	// Type of the genomic changes studied in the analysis (e.g., DNA, RNA, or AA change)
	ChangeType []CodeableConcept `json:"changeType,omitempty"`
	// Genome build that is used in this analysis
	GenomeBuild *CodeableConcept `json:"genomeBuild,omitempty"`
	// The defined protocol that describes the analysis
	InstantiatesCanonical *string `json:"instantiatesCanonical,omitempty"`
	// The URL pointing to an externally maintained protocol that describes the analysis
	InstantiatesUri *string `json:"instantiatesUri,omitempty"`
	// Name of the analysis event (human friendly)
	Title *string `json:"title,omitempty"`
	// What the genomic analysis is about, when it is not about the subject of record
	Focus []Reference `json:"focus,omitempty"`
	// The specimen used in the analysis event
	Specimen []Reference `json:"specimen,omitempty"`
	// The date of the analysis event
	Date *primitives.DateTime `json:"date,omitempty"`
	// Any notes capture with the analysis event
	Note []Annotation `json:"note,omitempty"`
	// The protocol that was performed for the analysis event
	ProtocolPerformed *Reference `json:"protocolPerformed,omitempty"`
	// The genomic regions to be studied in the analysis (BED file)
	RegionsStudied []Reference `json:"regionsStudied,omitempty"`
	// Genomic regions actually called in the analysis event (BED file)
	RegionsCalled []Reference `json:"regionsCalled,omitempty"`
	// Inputs for the analysis event
	Input []GenomicStudyAnalysisInput `json:"input,omitempty"`
	// Outputs for the analysis event
	Output []GenomicStudyAnalysisOutput `json:"output,omitempty"`
	// Performer for the analysis event
	Performer []GenomicStudyAnalysisPerformer `json:"performer,omitempty"`
	// Devices used for the analysis (e.g., instruments, software), with settings and parameters
	Device []GenomicStudyAnalysisDevice `json:"device,omitempty"`
}

// GenomicStudy represents a FHIR GenomicStudy.
type GenomicStudy struct {
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
	// Identifiers for this genomic study
	Identifier []Identifier `json:"identifier,omitempty"`
	// registered | available | cancelled | entered-in-error | unknown
	Status string `json:"status"`
	// The type of the study (e.g., Familial variant segregation, Functional variation detection, or Gene expression profiling)
	Type []CodeableConcept `json:"type,omitempty"`
	// The primary subject of the genomic study
	Subject Reference `json:"subject"`
	// The healthcare event with which this genomics study is associated
	Encounter *Reference `json:"encounter,omitempty"`
	// When the genomic study was started
	StartDate *primitives.DateTime `json:"startDate,omitempty"`
	// Event resources that the genomic study is based on
	BasedOn []Reference `json:"basedOn,omitempty"`
	// Healthcare professional who requested or referred the genomic study
	Referrer *Reference `json:"referrer,omitempty"`
	// Healthcare professionals who interpreted the genomic study
	Interpreter []Reference `json:"interpreter,omitempty"`
	// Why the genomic study was performed
	Reason []CodeableReference `json:"reason,omitempty"`
	// The defined protocol that describes the study
	InstantiatesCanonical *string `json:"instantiatesCanonical,omitempty"`
	// The URL pointing to an externally maintained protocol that describes the study
	InstantiatesUri *string `json:"instantiatesUri,omitempty"`
	// Comments related to the genomic study
	Note []Annotation `json:"note,omitempty"`
	// Description of the genomic study
	Description *string `json:"description,omitempty"`
	// Genomic Analysis Event
	Analysis []GenomicStudyAnalysis `json:"analysis,omitempty"`
}
