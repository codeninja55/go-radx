package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeClaim is the FHIR resource type name for Claim.
const ResourceTypeClaim = "Claim"

// ClaimRelated represents a FHIR BackboneElement for Claim.related.
type ClaimRelated struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Reference to the related claim
	Claim *Reference `json:"claim,omitempty"`
	// How the reference claim is related
	Relationship *CodeableConcept `json:"relationship,omitempty"`
	// File or case reference
	Reference *Identifier `json:"reference,omitempty"`
}

// ClaimPayee represents a FHIR BackboneElement for Claim.payee.
type ClaimPayee struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Category of recipient
	Type CodeableConcept `json:"type"`
	// Recipient reference
	Party *Reference `json:"party,omitempty"`
}

// ClaimEvent represents a FHIR BackboneElement for Claim.event.
type ClaimEvent struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Specific event
	Type CodeableConcept `json:"type"`
	// Occurance date or period
	When any `json:"when"`
}

// ClaimCareTeam represents a FHIR BackboneElement for Claim.careTeam.
type ClaimCareTeam struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Order of care team
	Sequence int `json:"sequence"`
	// Practitioner or organization
	Provider Reference `json:"provider"`
	// Indicator of the lead practitioner
	Responsible *bool `json:"responsible,omitempty"`
	// Function within the team
	Role *CodeableConcept `json:"role,omitempty"`
	// Practitioner or provider specialization
	Specialty *CodeableConcept `json:"specialty,omitempty"`
}

// ClaimSupportingInfo represents a FHIR BackboneElement for Claim.supportingInfo.
type ClaimSupportingInfo struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Information instance identifier
	Sequence int `json:"sequence"`
	// Classification of the supplied information
	Category CodeableConcept `json:"category"`
	// Type of information
	Code *CodeableConcept `json:"code,omitempty"`
	// When it occurred
	Timing *any `json:"timing,omitempty"`
	// Data to be provided
	Value *any `json:"value,omitempty"`
	// Explanation for the information
	Reason *CodeableConcept `json:"reason,omitempty"`
}

// ClaimDiagnosis represents a FHIR BackboneElement for Claim.diagnosis.
type ClaimDiagnosis struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Diagnosis instance identifier
	Sequence int `json:"sequence"`
	// Nature of illness or problem
	Diagnosis any `json:"diagnosis"`
	// Timing or nature of the diagnosis
	Type []CodeableConcept `json:"type,omitempty"`
	// Present on admission
	OnAdmission *CodeableConcept `json:"onAdmission,omitempty"`
}

// ClaimProcedure represents a FHIR BackboneElement for Claim.procedure.
type ClaimProcedure struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Procedure instance identifier
	Sequence int `json:"sequence"`
	// Category of Procedure
	Type []CodeableConcept `json:"type,omitempty"`
	// When the procedure was performed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Specific clinical procedure
	Procedure any `json:"procedure"`
	// Unique device identifier
	Udi []Reference `json:"udi,omitempty"`
}

// ClaimInsurance represents a FHIR BackboneElement for Claim.insurance.
type ClaimInsurance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Insurance instance identifier
	Sequence int `json:"sequence"`
	// Coverage to be used for adjudication
	Focal bool `json:"focal"`
	// Pre-assigned Claim number
	Identifier *Identifier `json:"identifier,omitempty"`
	// Insurance information
	Coverage Reference `json:"coverage"`
	// Additional provider contract number
	BusinessArrangement *string `json:"businessArrangement,omitempty"`
	// Prior authorization reference number
	PreAuthRef []string `json:"preAuthRef,omitempty"`
	// Adjudication results
	ClaimResponse *Reference `json:"claimResponse,omitempty"`
}

// ClaimAccident represents a FHIR BackboneElement for Claim.accident.
type ClaimAccident struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// When the incident occurred
	Date primitives.Date `json:"date"`
	// The nature of the accident
	Type *CodeableConcept `json:"type,omitempty"`
	// Where the event occurred
	Location *any `json:"location,omitempty"`
}

// ClaimItemBodySite represents a FHIR BackboneElement for Claim.item.bodySite.
type ClaimItemBodySite struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Location
	Site []CodeableReference `json:"site,omitempty"`
	// Sub-location
	SubSite []CodeableConcept `json:"subSite,omitempty"`
}

// ClaimItemDetailSubDetail represents a FHIR BackboneElement for Claim.item.detail.subDetail.
type ClaimItemDetailSubDetail struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Item instance identifier
	Sequence int `json:"sequence"`
	// Number for tracking
	TraceNumber []Identifier `json:"traceNumber,omitempty"`
	// Revenue or cost center code
	Revenue *CodeableConcept `json:"revenue,omitempty"`
	// Benefit classification
	Category *CodeableConcept `json:"category,omitempty"`
	// Billing, service, product, or drug code
	ProductOrService *CodeableConcept `json:"productOrService,omitempty"`
	// End of a range of codes
	ProductOrServiceEnd *CodeableConcept `json:"productOrServiceEnd,omitempty"`
	// Service/Product billing modifiers
	Modifier []CodeableConcept `json:"modifier,omitempty"`
	// Program the product or service is provided under
	ProgramCode []CodeableConcept `json:"programCode,omitempty"`
	// Paid by the patient
	PatientPaid *Money `json:"patientPaid,omitempty"`
	// Count of products or services
	Quantity *Quantity `json:"quantity,omitempty"`
	// Fee, charge or cost per item
	UnitPrice *Money `json:"unitPrice,omitempty"`
	// Price scaling factor
	Factor *float64 `json:"factor,omitempty"`
	// Total tax
	Tax *Money `json:"tax,omitempty"`
	// Total item cost
	Net *Money `json:"net,omitempty"`
	// Unique device identifier
	Udi []Reference `json:"udi,omitempty"`
}

// ClaimItemDetail represents a FHIR BackboneElement for Claim.item.detail.
type ClaimItemDetail struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Item instance identifier
	Sequence int `json:"sequence"`
	// Number for tracking
	TraceNumber []Identifier `json:"traceNumber,omitempty"`
	// Revenue or cost center code
	Revenue *CodeableConcept `json:"revenue,omitempty"`
	// Benefit classification
	Category *CodeableConcept `json:"category,omitempty"`
	// Billing, service, product, or drug code
	ProductOrService *CodeableConcept `json:"productOrService,omitempty"`
	// End of a range of codes
	ProductOrServiceEnd *CodeableConcept `json:"productOrServiceEnd,omitempty"`
	// Service/Product billing modifiers
	Modifier []CodeableConcept `json:"modifier,omitempty"`
	// Program the product or service is provided under
	ProgramCode []CodeableConcept `json:"programCode,omitempty"`
	// Paid by the patient
	PatientPaid *Money `json:"patientPaid,omitempty"`
	// Count of products or services
	Quantity *Quantity `json:"quantity,omitempty"`
	// Fee, charge or cost per item
	UnitPrice *Money `json:"unitPrice,omitempty"`
	// Price scaling factor
	Factor *float64 `json:"factor,omitempty"`
	// Total tax
	Tax *Money `json:"tax,omitempty"`
	// Total item cost
	Net *Money `json:"net,omitempty"`
	// Unique device identifier
	Udi []Reference `json:"udi,omitempty"`
	// Product or service provided
	SubDetail []ClaimItemDetailSubDetail `json:"subDetail,omitempty"`
}

// ClaimItem represents a FHIR BackboneElement for Claim.item.
type ClaimItem struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Item instance identifier
	Sequence int `json:"sequence"`
	// Number for tracking
	TraceNumber []Identifier `json:"traceNumber,omitempty"`
	// Applicable careTeam members
	CareTeamSequence []int `json:"careTeamSequence,omitempty"`
	// Applicable diagnoses
	DiagnosisSequence []int `json:"diagnosisSequence,omitempty"`
	// Applicable procedures
	ProcedureSequence []int `json:"procedureSequence,omitempty"`
	// Applicable exception and supporting information
	InformationSequence []int `json:"informationSequence,omitempty"`
	// Revenue or cost center code
	Revenue *CodeableConcept `json:"revenue,omitempty"`
	// Benefit classification
	Category *CodeableConcept `json:"category,omitempty"`
	// Billing, service, product, or drug code
	ProductOrService *CodeableConcept `json:"productOrService,omitempty"`
	// End of a range of codes
	ProductOrServiceEnd *CodeableConcept `json:"productOrServiceEnd,omitempty"`
	// Request or Referral for Service
	Request []Reference `json:"request,omitempty"`
	// Product or service billing modifiers
	Modifier []CodeableConcept `json:"modifier,omitempty"`
	// Program the product or service is provided under
	ProgramCode []CodeableConcept `json:"programCode,omitempty"`
	// Date or dates of service or product delivery
	Serviced *any `json:"serviced,omitempty"`
	// Place of service or where product was supplied
	Location *any `json:"location,omitempty"`
	// Paid by the patient
	PatientPaid *Money `json:"patientPaid,omitempty"`
	// Count of products or services
	Quantity *Quantity `json:"quantity,omitempty"`
	// Fee, charge or cost per item
	UnitPrice *Money `json:"unitPrice,omitempty"`
	// Price scaling factor
	Factor *float64 `json:"factor,omitempty"`
	// Total tax
	Tax *Money `json:"tax,omitempty"`
	// Total item cost
	Net *Money `json:"net,omitempty"`
	// Unique device identifier
	Udi []Reference `json:"udi,omitempty"`
	// Anatomical location
	BodySite []ClaimItemBodySite `json:"bodySite,omitempty"`
	// Encounters associated with the listed treatments
	Encounter []Reference `json:"encounter,omitempty"`
	// Product or service provided
	Detail []ClaimItemDetail `json:"detail,omitempty"`
}

// Claim represents a FHIR Claim.
type Claim struct {
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
	// Business Identifier for claim
	Identifier []Identifier `json:"identifier,omitempty"`
	// Number for tracking
	TraceNumber []Identifier `json:"traceNumber,omitempty"`
	// active | cancelled | draft | entered-in-error
	Status string `json:"status"`
	// Category or discipline
	Type CodeableConcept `json:"type"`
	// More granular claim type
	SubType *CodeableConcept `json:"subType,omitempty"`
	// claim | preauthorization | predetermination
	Use string `json:"use"`
	// The recipient of the products and services
	Patient Reference `json:"patient"`
	// Relevant time frame for the claim
	BillablePeriod *Period `json:"billablePeriod,omitempty"`
	// Resource creation date
	Created primitives.DateTime `json:"created"`
	// Author of the claim
	Enterer *Reference `json:"enterer,omitempty"`
	// Target
	Insurer *Reference `json:"insurer,omitempty"`
	// Party responsible for the claim
	Provider *Reference `json:"provider,omitempty"`
	// Desired processing urgency
	Priority *CodeableConcept `json:"priority,omitempty"`
	// For whom to reserve funds
	FundsReserve *CodeableConcept `json:"fundsReserve,omitempty"`
	// Prior or corollary claims
	Related []ClaimRelated `json:"related,omitempty"`
	// Prescription authorizing services and products
	Prescription *Reference `json:"prescription,omitempty"`
	// Original prescription if superseded by fulfiller
	OriginalPrescription *Reference `json:"originalPrescription,omitempty"`
	// Recipient of benefits payable
	Payee *ClaimPayee `json:"payee,omitempty"`
	// Treatment referral
	Referral *Reference `json:"referral,omitempty"`
	// Encounters associated with the listed treatments
	Encounter []Reference `json:"encounter,omitempty"`
	// Servicing facility
	Facility *Reference `json:"facility,omitempty"`
	// Package billing code
	DiagnosisRelatedGroup *CodeableConcept `json:"diagnosisRelatedGroup,omitempty"`
	// Event information
	Event []ClaimEvent `json:"event,omitempty"`
	// Members of the care team
	CareTeam []ClaimCareTeam `json:"careTeam,omitempty"`
	// Supporting information
	SupportingInfo []ClaimSupportingInfo `json:"supportingInfo,omitempty"`
	// Pertinent diagnosis information
	Diagnosis []ClaimDiagnosis `json:"diagnosis,omitempty"`
	// Clinical procedures performed
	Procedure []ClaimProcedure `json:"procedure,omitempty"`
	// Patient insurance information
	Insurance []ClaimInsurance `json:"insurance,omitempty"`
	// Details of the event
	Accident *ClaimAccident `json:"accident,omitempty"`
	// Paid by the patient
	PatientPaid *Money `json:"patientPaid,omitempty"`
	// Product or service provided
	Item []ClaimItem `json:"item,omitempty"`
	// Total claim cost
	Total *Money `json:"total,omitempty"`
}
