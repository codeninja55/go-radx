package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeDocumentReference is the FHIR resource type name for DocumentReference.
const ResourceTypeDocumentReference = "DocumentReference"

// DocumentReferenceAttester represents a FHIR BackboneElement for DocumentReference.attester.
type DocumentReferenceAttester struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// personal | professional | legal | official
	Mode CodeableConcept `json:"mode"`
	// When the document was attested
	Time *primitives.DateTime `json:"time,omitempty"`
	// Who attested the document
	Party *Reference `json:"party,omitempty"`
}

// DocumentReferenceRelatesTo represents a FHIR BackboneElement for DocumentReference.relatesTo.
type DocumentReferenceRelatesTo struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The relationship type with another document
	Code CodeableConcept `json:"code"`
	// Target of the relationship
	Target Reference `json:"target"`
}

// DocumentReferenceContentProfile represents a FHIR BackboneElement for DocumentReference.content.profile.
type DocumentReferenceContentProfile struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Code|uri|canonical
	Value any `json:"value"`
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
	// Content profile rules for the document
	Profile []DocumentReferenceContentProfile `json:"profile,omitempty"`
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
	// Business identifiers for the document
	Identifier []Identifier `json:"identifier,omitempty"`
	// An explicitly assigned identifer of a variation of the content in the DocumentReference
	Version *string `json:"version,omitempty"`
	// Procedure that caused this media to be created
	BasedOn []Reference `json:"basedOn,omitempty"`
	// current | superseded | entered-in-error
	Status string `json:"status"`
	// registered | partial | preliminary | final | amended | corrected | appended | cancelled | entered-in-error | deprecated | unknown
	DocStatus *string `json:"docStatus,omitempty"`
	// Imaging modality used
	Modality []CodeableConcept `json:"modality,omitempty"`
	// Kind of document (LOINC if possible)
	Type *CodeableConcept `json:"type,omitempty"`
	// Categorization of document
	Category []CodeableConcept `json:"category,omitempty"`
	// Who/what is the subject of the document
	Subject *Reference `json:"subject,omitempty"`
	// Context of the document content
	Context []Reference `json:"context,omitempty"`
	// Main clinical acts documented
	Event []CodeableReference `json:"event,omitempty"`
	// Body part included
	BodySite []CodeableReference `json:"bodySite,omitempty"`
	// Kind of facility where patient was seen
	FacilityType *CodeableConcept `json:"facilityType,omitempty"`
	// Additional details about where the content was created (e.g. clinical specialty)
	PracticeSetting *CodeableConcept `json:"practiceSetting,omitempty"`
	// Time of service that is being documented
	Period *Period `json:"period,omitempty"`
	// When this document reference was created
	Date *primitives.Instant `json:"date,omitempty"`
	// Who and/or what authored the document
	Author []Reference `json:"author,omitempty"`
	// Attests to accuracy of the document
	Attester []DocumentReferenceAttester `json:"attester,omitempty"`
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
}
