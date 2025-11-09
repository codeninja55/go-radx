package resources

// ResourceTypeEvidenceReport is the FHIR resource type name for EvidenceReport.
const ResourceTypeEvidenceReport = "EvidenceReport"

// EvidenceReportSubjectCharacteristic represents a FHIR BackboneElement for EvidenceReport.subject.characteristic.
type EvidenceReportSubjectCharacteristic struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Characteristic code
	Code CodeableConcept `json:"code"`
	// Characteristic value
	Value any `json:"value"`
	// Is used to express not the characteristic
	Exclude *bool `json:"exclude,omitempty"`
	// Timeframe for the characteristic
	Period *Period `json:"period,omitempty"`
}

// EvidenceReportSubject represents a FHIR BackboneElement for EvidenceReport.subject.
type EvidenceReportSubject struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Characteristic
	Characteristic []EvidenceReportSubjectCharacteristic `json:"characteristic,omitempty"`
	// Footnotes and/or explanatory notes
	Note []Annotation `json:"note,omitempty"`
}

// EvidenceReportRelatesToTarget represents a FHIR BackboneElement for EvidenceReport.relatesTo.target.
type EvidenceReportRelatesToTarget struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Target of the relationship URL
	URL *string `json:"url,omitempty"`
	// Target of the relationship Identifier
	Identifier *Identifier `json:"identifier,omitempty"`
	// Target of the relationship Display
	Display *string `json:"display,omitempty"`
	// Target of the relationship Resource reference
	Resource *Reference `json:"resource,omitempty"`
}

// EvidenceReportRelatesTo represents a FHIR BackboneElement for EvidenceReport.relatesTo.
type EvidenceReportRelatesTo struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// replaces | amends | appends | transforms | replacedWith | amendedWith | appendedWith | transformedWith
	Code string `json:"code"`
	// Target of the relationship
	Target EvidenceReportRelatesToTarget `json:"target"`
}

// EvidenceReportSectionSection represents a FHIR BackboneElement for EvidenceReport.section.section.
type EvidenceReportSectionSection struct {
}

// EvidenceReportSection represents a FHIR BackboneElement for EvidenceReport.section.
type EvidenceReportSection struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Label for section (e.g. for ToC)
	Title *string `json:"title,omitempty"`
	// Classification of section (recommended)
	Focus *CodeableConcept `json:"focus,omitempty"`
	// Classification of section by Resource
	FocusReference *Reference `json:"focusReference,omitempty"`
	// Who and/or what authored the section
	Author []Reference `json:"author,omitempty"`
	// Text summary of the section, for human interpretation
	Text *Narrative `json:"text,omitempty"`
	// working | snapshot | changes
	Mode *string `json:"mode,omitempty"`
	// Order of section entries
	OrderedBy *CodeableConcept `json:"orderedBy,omitempty"`
	// Extensible classifiers as content
	EntryClassifier []CodeableConcept `json:"entryClassifier,omitempty"`
	// Reference to resources as content
	EntryReference []Reference `json:"entryReference,omitempty"`
	// Quantity as content
	EntryQuantity []Quantity `json:"entryQuantity,omitempty"`
	// Why the section is empty
	EmptyReason *CodeableConcept `json:"emptyReason,omitempty"`
	// Nested Section
	Section []EvidenceReportSectionSection `json:"section,omitempty"`
}

// EvidenceReport represents a FHIR EvidenceReport.
type EvidenceReport struct {
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
	// Canonical identifier for this EvidenceReport, represented as a globally unique URI
	URL *string `json:"url,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Unique identifier for the evidence report
	Identifier []Identifier `json:"identifier,omitempty"`
	// Identifiers for articles that may relate to more than one evidence report
	RelatedIdentifier []Identifier `json:"relatedIdentifier,omitempty"`
	// Citation for this report
	CiteAs *any `json:"citeAs,omitempty"`
	// Kind of report
	Type *CodeableConcept `json:"type,omitempty"`
	// Used for footnotes and annotations
	Note []Annotation `json:"note,omitempty"`
	// Link, description or reference to artifact associated with the report
	RelatedArtifact []RelatedArtifact `json:"relatedArtifact,omitempty"`
	// Focus of the report
	Subject EvidenceReportSubject `json:"subject"`
	// Name of the publisher/steward (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Who authored the content
	Author []ContactDetail `json:"author,omitempty"`
	// Who edited the content
	Editor []ContactDetail `json:"editor,omitempty"`
	// Who reviewed the content
	Reviewer []ContactDetail `json:"reviewer,omitempty"`
	// Who endorsed the content
	Endorser []ContactDetail `json:"endorser,omitempty"`
	// Relationships to other compositions/documents
	RelatesTo []EvidenceReportRelatesTo `json:"relatesTo,omitempty"`
	// Composition is broken into sections
	Section []EvidenceReportSection `json:"section,omitempty"`
}
