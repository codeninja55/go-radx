package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeMedicinalProductAuthorization is the FHIR resource type name for MedicinalProductAuthorization.
const ResourceTypeMedicinalProductAuthorization = "MedicinalProductAuthorization"

// MedicinalProductAuthorizationJurisdictionalAuthorization represents a FHIR BackboneElement for MedicinalProductAuthorization.jurisdictionalAuthorization.
type MedicinalProductAuthorizationJurisdictionalAuthorization struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The assigned number for the marketing authorization
	Identifier []Identifier `json:"identifier,omitempty"`
	// Country of authorization
	Country *CodeableConcept `json:"country,omitempty"`
	// Jurisdiction within a country
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// The legal status of supply in a jurisdiction or region
	LegalStatusOfSupply *CodeableConcept `json:"legalStatusOfSupply,omitempty"`
	// The start and expected end date of the authorization
	ValidityPeriod *Period `json:"validityPeriod,omitempty"`
}

// MedicinalProductAuthorizationProcedureApplication represents a FHIR BackboneElement for MedicinalProductAuthorization.procedure.application.
type MedicinalProductAuthorizationProcedureApplication struct {
}

// MedicinalProductAuthorizationProcedure represents a FHIR BackboneElement for MedicinalProductAuthorization.procedure.
type MedicinalProductAuthorizationProcedure struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Identifier for this procedure
	Identifier *Identifier `json:"identifier,omitempty"`
	// Type of procedure
	Type CodeableConcept `json:"type"`
	// Date of procedure
	Date *any `json:"date,omitempty"`
	// Applcations submitted to obtain a marketing authorization
	Application []MedicinalProductAuthorizationProcedureApplication `json:"application,omitempty"`
}

// MedicinalProductAuthorization represents a FHIR MedicinalProductAuthorization.
type MedicinalProductAuthorization struct {
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
	// Business identifier for the marketing authorization, as assigned by a regulator
	Identifier []Identifier `json:"identifier,omitempty"`
	// The medicinal product that is being authorized
	Subject *Reference `json:"subject,omitempty"`
	// The country in which the marketing authorization has been granted
	Country []CodeableConcept `json:"country,omitempty"`
	// Jurisdiction within a country
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// The status of the marketing authorization
	Status *CodeableConcept `json:"status,omitempty"`
	// The date at which the given status has become applicable
	StatusDate *primitives.DateTime `json:"statusDate,omitempty"`
	// The date when a suspended the marketing or the marketing authorization of the product is anticipated to be restored
	RestoreDate *primitives.DateTime `json:"restoreDate,omitempty"`
	// The beginning of the time period in which the marketing authorization is in the specific status shall be specified A complete date consisting of day, month and year shall be specified using the ISO 8601 date format
	ValidityPeriod *Period `json:"validityPeriod,omitempty"`
	// A period of time after authorization before generic product applicatiosn can be submitted
	DataExclusivityPeriod *Period `json:"dataExclusivityPeriod,omitempty"`
	// The date when the first authorization was granted by a Medicines Regulatory Agency
	DateOfFirstAuthorization *primitives.DateTime `json:"dateOfFirstAuthorization,omitempty"`
	// Date of first marketing authorization for a company's new medicinal product in any country in the World
	InternationalBirthDate *primitives.DateTime `json:"internationalBirthDate,omitempty"`
	// The legal framework against which this authorization is granted
	LegalBasis *CodeableConcept `json:"legalBasis,omitempty"`
	// Authorization in areas within a country
	JurisdictionalAuthorization []MedicinalProductAuthorizationJurisdictionalAuthorization `json:"jurisdictionalAuthorization,omitempty"`
	// Marketing Authorization Holder
	Holder *Reference `json:"holder,omitempty"`
	// Medicines Regulatory Agency
	Regulator *Reference `json:"regulator,omitempty"`
	// The regulatory procedure for granting or amending a marketing authorization
	Procedure *MedicinalProductAuthorizationProcedure `json:"procedure,omitempty"`
}
