package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeRegulatedAuthorization is the FHIR resource type name for RegulatedAuthorization.
const ResourceTypeRegulatedAuthorization = "RegulatedAuthorization"

// RegulatedAuthorizationCaseApplication represents a FHIR BackboneElement for RegulatedAuthorization.case.application.
type RegulatedAuthorizationCaseApplication struct {
}

// RegulatedAuthorizationCase represents a FHIR BackboneElement for RegulatedAuthorization.case.
type RegulatedAuthorizationCase struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Identifier by which this case can be referenced
	Identifier *Identifier `json:"identifier,omitempty"`
	// The defining type of case
	Type *CodeableConcept `json:"type,omitempty"`
	// The status associated with the case
	Status *CodeableConcept `json:"status,omitempty"`
	// Relevant date for this case
	Date *any `json:"date,omitempty"`
	// Applications submitted to obtain a regulated authorization. Steps within the longer running case or procedure
	Application []RegulatedAuthorizationCaseApplication `json:"application,omitempty"`
}

// RegulatedAuthorization represents a FHIR RegulatedAuthorization.
type RegulatedAuthorization struct {
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
	// Business identifier for the authorization, typically assigned by the authorizing body
	Identifier []Identifier `json:"identifier,omitempty"`
	// The product type, treatment, facility or activity that is being authorized
	Subject []Reference `json:"subject,omitempty"`
	// Overall type of this authorization, for example drug marketing approval, orphan drug designation
	Type *CodeableConcept `json:"type,omitempty"`
	// General textual supporting information
	Description *string `json:"description,omitempty"`
	// The territory in which the authorization has been granted
	Region []CodeableConcept `json:"region,omitempty"`
	// The status that is authorised e.g. approved. Intermediate states can be tracked with cases and applications
	Status *CodeableConcept `json:"status,omitempty"`
	// The date at which the current status was assigned
	StatusDate *primitives.DateTime `json:"statusDate,omitempty"`
	// The time period in which the regulatory approval etc. is in effect, e.g. a Marketing Authorization includes the date of authorization and/or expiration date
	ValidityPeriod *Period `json:"validityPeriod,omitempty"`
	// Condition for which the use of the regulated product applies
	Indication []CodeableReference `json:"indication,omitempty"`
	// The intended use of the product, e.g. prevention, treatment
	IntendedUse *CodeableConcept `json:"intendedUse,omitempty"`
	// The legal/regulatory framework or reasons under which this authorization is granted
	Basis []CodeableConcept `json:"basis,omitempty"`
	// The organization that has been granted this authorization, by the regulator
	Holder *Reference `json:"holder,omitempty"`
	// The regulatory authority or authorizing body granting the authorization
	Regulator *Reference `json:"regulator,omitempty"`
	// Additional information or supporting documentation about the authorization
	AttachedDocument []Reference `json:"attachedDocument,omitempty"`
	// The case or regulatory procedure for granting or amending a regulated authorization. Note: This area is subject to ongoing review and the workgroup is seeking implementer feedback on its use (see link at bottom of page)
	Case *RegulatedAuthorizationCase `json:"case,omitempty"`
}
