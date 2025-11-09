package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeDocumentReference is the FHIR resource type name for DocumentReference.
const ResourceTypeDocumentReference = "DocumentReference"

// DocumentReferenceRelatesTo represents a FHIR BackboneElement for DocumentReference.relatesTo.
type DocumentReferenceRelatesTo struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// replaces | transforms | signs | appends
	Code string `json:"code"`
	// Target of the relationship
	Target Reference `json:"target"`
}

// DocumentReferenceContent represents a FHIR BackboneElement for DocumentReference.content.
type DocumentReferenceContent struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Where to access the document
	Attachment Attachment `json:"attachment"`
	// Format/content rules for the document
	Format *Coding `json:"format,omitempty"`
}

// DocumentReferenceContext represents a FHIR BackboneElement for DocumentReference.context.
type DocumentReferenceContext struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Context of the document  content
	Encounter []Reference `json:"encounter,omitempty"`
	// Main clinical acts documented
	Event []CodeableConcept `json:"event,omitempty"`
	// Time of service that is being documented
	Period *Period `json:"period,omitempty"`
	// Kind of facility where patient was seen
	FacilityType *CodeableConcept `json:"facilityType,omitempty"`
	// Additional details about where the content was created (e.g. clinical specialty)
	PracticeSetting *CodeableConcept `json:"practiceSetting,omitempty"`
	// Patient demographics from source
	SourcePatientInfo *Reference `json:"sourcePatientInfo,omitempty"`
	// Related identifiers or resources
	Related []Reference `json:"related,omitempty"`
}

// DocumentReference represents a FHIR DocumentReference.
type DocumentReference struct {
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
	// Master Version Specific Identifier
	MasterIdentifier *Identifier `json:"masterIdentifier,omitempty"`
	// Other identifiers for the document
	Identifier []Identifier `json:"identifier,omitempty"`
	// current | superseded | entered-in-error
	Status string `json:"status"`
	// preliminary | final | amended | entered-in-error
	DocStatus *string `json:"docStatus,omitempty"`
	// Kind of document (LOINC if possible)
	Type *CodeableConcept `json:"type,omitempty"`
	// Categorization of document
	Category []CodeableConcept `json:"category,omitempty"`
	// Who/what is the subject of the document
	Subject *Reference `json:"subject,omitempty"`
	// When this document reference was created
	Date *primitives.Instant `json:"date,omitempty"`
	// Who and/or what authored the document
	Author []Reference `json:"author,omitempty"`
	// Who/what authenticated the document
	Authenticator *Reference `json:"authenticator,omitempty"`
	// Organization which maintains the document
	Custodian *Reference `json:"custodian,omitempty"`
	// Relationships to other documents
	RelatesTo []DocumentReferenceRelatesTo `json:"relatesTo,omitempty"`
	// Human-readable description
	Description *string `json:"description,omitempty"`
	// Document security-tags
	SecurityLabel []CodeableConcept `json:"securityLabel,omitempty"`
	// Document referenced
	Content []DocumentReferenceContent `json:"content,omitempty"`
	// Clinical context of document
	Context *DocumentReferenceContext `json:"context,omitempty"`
}
