package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeExplanationOfBenefit is the FHIR resource type name for ExplanationOfBenefit.
const ResourceTypeExplanationOfBenefit = "ExplanationOfBenefit"

// ExplanationOfBenefitRelated represents a FHIR BackboneElement for ExplanationOfBenefit.related.
type ExplanationOfBenefitRelated struct {
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

// ExplanationOfBenefitPayee represents a FHIR BackboneElement for ExplanationOfBenefit.payee.
type ExplanationOfBenefitPayee struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Category of recipient
	Type *CodeableConcept `json:"type,omitempty"`
	// Recipient reference
	Party *Reference `json:"party,omitempty"`
}

// ExplanationOfBenefitCareTeam represents a FHIR BackboneElement for ExplanationOfBenefit.careTeam.
type ExplanationOfBenefitCareTeam struct {
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
	// Practitioner credential or specialization
	Qualification *CodeableConcept `json:"qualification,omitempty"`
}

// ExplanationOfBenefitSupportingInfo represents a FHIR BackboneElement for ExplanationOfBenefit.supportingInfo.
type ExplanationOfBenefitSupportingInfo struct {
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
	Reason *Coding `json:"reason,omitempty"`
}

// ExplanationOfBenefitDiagnosis represents a FHIR BackboneElement for ExplanationOfBenefit.diagnosis.
type ExplanationOfBenefitDiagnosis struct {
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
	// Package billing code
	PackageCode *CodeableConcept `json:"packageCode,omitempty"`
}

// ExplanationOfBenefitProcedure represents a FHIR BackboneElement for ExplanationOfBenefit.procedure.
type ExplanationOfBenefitProcedure struct {
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

// ExplanationOfBenefitInsurance represents a FHIR BackboneElement for ExplanationOfBenefit.insurance.
type ExplanationOfBenefitInsurance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Coverage to be used for adjudication
	Focal bool `json:"focal"`
	// Insurance information
	Coverage Reference `json:"coverage"`
	// Prior authorization reference number
	PreAuthRef []string `json:"preAuthRef,omitempty"`
}

// ExplanationOfBenefitAccident represents a FHIR BackboneElement for ExplanationOfBenefit.accident.
type ExplanationOfBenefitAccident struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// When the incident occurred
	Date *primitives.Date `json:"date,omitempty"`
	// The nature of the accident
	Type *CodeableConcept `json:"type,omitempty"`
	// Where the event occurred
	Location *any `json:"location,omitempty"`
}

// ExplanationOfBenefitItemAdjudication represents a FHIR BackboneElement for ExplanationOfBenefit.item.adjudication.
type ExplanationOfBenefitItemAdjudication struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of adjudication information
	Category CodeableConcept `json:"category"`
	// Explanation of adjudication outcome
	Reason *CodeableConcept `json:"reason,omitempty"`
	// Monetary amount
	Amount *Money `json:"amount,omitempty"`
	// Non-monitary value
	Value *float64 `json:"value,omitempty"`
}

// ExplanationOfBenefitItemDetailAdjudication represents a FHIR BackboneElement for ExplanationOfBenefit.item.detail.adjudication.
type ExplanationOfBenefitItemDetailAdjudication struct {
}

// ExplanationOfBenefitItemDetailSubDetailAdjudication represents a FHIR BackboneElement for ExplanationOfBenefit.item.detail.subDetail.adjudication.
type ExplanationOfBenefitItemDetailSubDetailAdjudication struct {
}

// ExplanationOfBenefitItemDetailSubDetail represents a FHIR BackboneElement for ExplanationOfBenefit.item.detail.subDetail.
type ExplanationOfBenefitItemDetailSubDetail struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Product or service provided
	Sequence int `json:"sequence"`
	// Revenue or cost center code
	Revenue *CodeableConcept `json:"revenue,omitempty"`
	// Benefit classification
	Category *CodeableConcept `json:"category,omitempty"`
	// Billing, service, product, or drug code
	ProductOrService CodeableConcept `json:"productOrService"`
	// Service/Product billing modifiers
	Modifier []CodeableConcept `json:"modifier,omitempty"`
	// Program the product or service is provided under
	ProgramCode []CodeableConcept `json:"programCode,omitempty"`
	// Count of products or services
	Quantity *Quantity `json:"quantity,omitempty"`
	// Fee, charge or cost per item
	UnitPrice *Money `json:"unitPrice,omitempty"`
	// Price scaling factor
	Factor *float64 `json:"factor,omitempty"`
	// Total item cost
	Net *Money `json:"net,omitempty"`
	// Unique device identifier
	Udi []Reference `json:"udi,omitempty"`
	// Applicable note numbers
	NoteNumber []int `json:"noteNumber,omitempty"`
	// Subdetail level adjudication details
	Adjudication []ExplanationOfBenefitItemDetailSubDetailAdjudication `json:"adjudication,omitempty"`
}

// ExplanationOfBenefitItemDetail represents a FHIR BackboneElement for ExplanationOfBenefit.item.detail.
type ExplanationOfBenefitItemDetail struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Product or service provided
	Sequence int `json:"sequence"`
	// Revenue or cost center code
	Revenue *CodeableConcept `json:"revenue,omitempty"`
	// Benefit classification
	Category *CodeableConcept `json:"category,omitempty"`
	// Billing, service, product, or drug code
	ProductOrService CodeableConcept `json:"productOrService"`
	// Service/Product billing modifiers
	Modifier []CodeableConcept `json:"modifier,omitempty"`
	// Program the product or service is provided under
	ProgramCode []CodeableConcept `json:"programCode,omitempty"`
	// Count of products or services
	Quantity *Quantity `json:"quantity,omitempty"`
	// Fee, charge or cost per item
	UnitPrice *Money `json:"unitPrice,omitempty"`
	// Price scaling factor
	Factor *float64 `json:"factor,omitempty"`
	// Total item cost
	Net *Money `json:"net,omitempty"`
	// Unique device identifier
	Udi []Reference `json:"udi,omitempty"`
	// Applicable note numbers
	NoteNumber []int `json:"noteNumber,omitempty"`
	// Detail level adjudication details
	Adjudication []ExplanationOfBenefitItemDetailAdjudication `json:"adjudication,omitempty"`
	// Additional items
	SubDetail []ExplanationOfBenefitItemDetailSubDetail `json:"subDetail,omitempty"`
}

// ExplanationOfBenefitItem represents a FHIR BackboneElement for ExplanationOfBenefit.item.
type ExplanationOfBenefitItem struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Item instance identifier
	Sequence int `json:"sequence"`
	// Applicable care team members
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
	ProductOrService CodeableConcept `json:"productOrService"`
	// Product or service billing modifiers
	Modifier []CodeableConcept `json:"modifier,omitempty"`
	// Program the product or service is provided under
	ProgramCode []CodeableConcept `json:"programCode,omitempty"`
	// Date or dates of service or product delivery
	Serviced *any `json:"serviced,omitempty"`
	// Place of service or where product was supplied
	Location *any `json:"location,omitempty"`
	// Count of products or services
	Quantity *Quantity `json:"quantity,omitempty"`
	// Fee, charge or cost per item
	UnitPrice *Money `json:"unitPrice,omitempty"`
	// Price scaling factor
	Factor *float64 `json:"factor,omitempty"`
	// Total item cost
	Net *Money `json:"net,omitempty"`
	// Unique device identifier
	Udi []Reference `json:"udi,omitempty"`
	// Anatomical location
	BodySite *CodeableConcept `json:"bodySite,omitempty"`
	// Anatomical sub-location
	SubSite []CodeableConcept `json:"subSite,omitempty"`
	// Encounters related to this billed item
	Encounter []Reference `json:"encounter,omitempty"`
	// Applicable note numbers
	NoteNumber []int `json:"noteNumber,omitempty"`
	// Adjudication details
	Adjudication []ExplanationOfBenefitItemAdjudication `json:"adjudication,omitempty"`
	// Additional items
	Detail []ExplanationOfBenefitItemDetail `json:"detail,omitempty"`
}

// ExplanationOfBenefitAddItemAdjudication represents a FHIR BackboneElement for ExplanationOfBenefit.addItem.adjudication.
type ExplanationOfBenefitAddItemAdjudication struct {
}

// ExplanationOfBenefitAddItemDetailAdjudication represents a FHIR BackboneElement for ExplanationOfBenefit.addItem.detail.adjudication.
type ExplanationOfBenefitAddItemDetailAdjudication struct {
}

// ExplanationOfBenefitAddItemDetailSubDetailAdjudication represents a FHIR BackboneElement for ExplanationOfBenefit.addItem.detail.subDetail.adjudication.
type ExplanationOfBenefitAddItemDetailSubDetailAdjudication struct {
}

// ExplanationOfBenefitAddItemDetailSubDetail represents a FHIR BackboneElement for ExplanationOfBenefit.addItem.detail.subDetail.
type ExplanationOfBenefitAddItemDetailSubDetail struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Billing, service, product, or drug code
	ProductOrService CodeableConcept `json:"productOrService"`
	// Service/Product billing modifiers
	Modifier []CodeableConcept `json:"modifier,omitempty"`
	// Count of products or services
	Quantity *Quantity `json:"quantity,omitempty"`
	// Fee, charge or cost per item
	UnitPrice *Money `json:"unitPrice,omitempty"`
	// Price scaling factor
	Factor *float64 `json:"factor,omitempty"`
	// Total item cost
	Net *Money `json:"net,omitempty"`
	// Applicable note numbers
	NoteNumber []int `json:"noteNumber,omitempty"`
	// Added items adjudication
	Adjudication []ExplanationOfBenefitAddItemDetailSubDetailAdjudication `json:"adjudication,omitempty"`
}

// ExplanationOfBenefitAddItemDetail represents a FHIR BackboneElement for ExplanationOfBenefit.addItem.detail.
type ExplanationOfBenefitAddItemDetail struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Billing, service, product, or drug code
	ProductOrService CodeableConcept `json:"productOrService"`
	// Service/Product billing modifiers
	Modifier []CodeableConcept `json:"modifier,omitempty"`
	// Count of products or services
	Quantity *Quantity `json:"quantity,omitempty"`
	// Fee, charge or cost per item
	UnitPrice *Money `json:"unitPrice,omitempty"`
	// Price scaling factor
	Factor *float64 `json:"factor,omitempty"`
	// Total item cost
	Net *Money `json:"net,omitempty"`
	// Applicable note numbers
	NoteNumber []int `json:"noteNumber,omitempty"`
	// Added items adjudication
	Adjudication []ExplanationOfBenefitAddItemDetailAdjudication `json:"adjudication,omitempty"`
	// Insurer added line items
	SubDetail []ExplanationOfBenefitAddItemDetailSubDetail `json:"subDetail,omitempty"`
}

// ExplanationOfBenefitAddItem represents a FHIR BackboneElement for ExplanationOfBenefit.addItem.
type ExplanationOfBenefitAddItem struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Item sequence number
	ItemSequence []int `json:"itemSequence,omitempty"`
	// Detail sequence number
	DetailSequence []int `json:"detailSequence,omitempty"`
	// Subdetail sequence number
	SubDetailSequence []int `json:"subDetailSequence,omitempty"`
	// Authorized providers
	Provider []Reference `json:"provider,omitempty"`
	// Billing, service, product, or drug code
	ProductOrService CodeableConcept `json:"productOrService"`
	// Service/Product billing modifiers
	Modifier []CodeableConcept `json:"modifier,omitempty"`
	// Program the product or service is provided under
	ProgramCode []CodeableConcept `json:"programCode,omitempty"`
	// Date or dates of service or product delivery
	Serviced *any `json:"serviced,omitempty"`
	// Place of service or where product was supplied
	Location *any `json:"location,omitempty"`
	// Count of products or services
	Quantity *Quantity `json:"quantity,omitempty"`
	// Fee, charge or cost per item
	UnitPrice *Money `json:"unitPrice,omitempty"`
	// Price scaling factor
	Factor *float64 `json:"factor,omitempty"`
	// Total item cost
	Net *Money `json:"net,omitempty"`
	// Anatomical location
	BodySite *CodeableConcept `json:"bodySite,omitempty"`
	// Anatomical sub-location
	SubSite []CodeableConcept `json:"subSite,omitempty"`
	// Applicable note numbers
	NoteNumber []int `json:"noteNumber,omitempty"`
	// Added items adjudication
	Adjudication []ExplanationOfBenefitAddItemAdjudication `json:"adjudication,omitempty"`
	// Insurer added line items
	Detail []ExplanationOfBenefitAddItemDetail `json:"detail,omitempty"`
}

// ExplanationOfBenefitAdjudication represents a FHIR BackboneElement for ExplanationOfBenefit.adjudication.
type ExplanationOfBenefitAdjudication struct {
}

// ExplanationOfBenefitTotal represents a FHIR BackboneElement for ExplanationOfBenefit.total.
type ExplanationOfBenefitTotal struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of adjudication information
	Category CodeableConcept `json:"category"`
	// Financial total for the category
	Amount Money `json:"amount"`
}

// ExplanationOfBenefitPayment represents a FHIR BackboneElement for ExplanationOfBenefit.payment.
type ExplanationOfBenefitPayment struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Partial or complete payment
	Type *CodeableConcept `json:"type,omitempty"`
	// Payment adjustment for non-claim issues
	Adjustment *Money `json:"adjustment,omitempty"`
	// Explanation for the variance
	AdjustmentReason *CodeableConcept `json:"adjustmentReason,omitempty"`
	// Expected date of payment
	Date *primitives.Date `json:"date,omitempty"`
	// Payable amount after adjustment
	Amount *Money `json:"amount,omitempty"`
	// Business identifier for the payment
	Identifier *Identifier `json:"identifier,omitempty"`
}

// ExplanationOfBenefitProcessNote represents a FHIR BackboneElement for ExplanationOfBenefit.processNote.
type ExplanationOfBenefitProcessNote struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Note instance identifier
	Number *int `json:"number,omitempty"`
	// display | print | printoper
	Type *string `json:"type,omitempty"`
	// Note explanatory text
	Text *string `json:"text,omitempty"`
	// Language of the text
	Language *CodeableConcept `json:"language,omitempty"`
}

// ExplanationOfBenefitBenefitBalanceFinancial represents a FHIR BackboneElement for ExplanationOfBenefit.benefitBalance.financial.
type ExplanationOfBenefitBenefitBalanceFinancial struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Benefit classification
	Type CodeableConcept `json:"type"`
	// Benefits allowed
	Allowed *any `json:"allowed,omitempty"`
	// Benefits used
	Used *any `json:"used,omitempty"`
}

// ExplanationOfBenefitBenefitBalance represents a FHIR BackboneElement for ExplanationOfBenefit.benefitBalance.
type ExplanationOfBenefitBenefitBalance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Benefit classification
	Category CodeableConcept `json:"category"`
	// Excluded from the plan
	Excluded *bool `json:"excluded,omitempty"`
	// Short name for the benefit
	Name *string `json:"name,omitempty"`
	// Description of the benefit or services covered
	Description *string `json:"description,omitempty"`
	// In or out of network
	Network *CodeableConcept `json:"network,omitempty"`
	// Individual or family
	Unit *CodeableConcept `json:"unit,omitempty"`
	// Annual or lifetime
	Term *CodeableConcept `json:"term,omitempty"`
	// Benefit Summary
	Financial []ExplanationOfBenefitBenefitBalanceFinancial `json:"financial,omitempty"`
}

// ExplanationOfBenefit represents a FHIR ExplanationOfBenefit.
type ExplanationOfBenefit struct {
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
	// Business Identifier for the resource
	Identifier []Identifier `json:"identifier,omitempty"`
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
	// Response creation date
	Created primitives.DateTime `json:"created"`
	// Author of the claim
	Enterer *Reference `json:"enterer,omitempty"`
	// Party responsible for reimbursement
	Insurer Reference `json:"insurer"`
	// Party responsible for the claim
	Provider Reference `json:"provider"`
	// Desired processing urgency
	Priority *CodeableConcept `json:"priority,omitempty"`
	// For whom to reserve funds
	FundsReserveRequested *CodeableConcept `json:"fundsReserveRequested,omitempty"`
	// Funds reserved status
	FundsReserve *CodeableConcept `json:"fundsReserve,omitempty"`
	// Prior or corollary claims
	Related []ExplanationOfBenefitRelated `json:"related,omitempty"`
	// Prescription authorizing services or products
	Prescription *Reference `json:"prescription,omitempty"`
	// Original prescription if superceded by fulfiller
	OriginalPrescription *Reference `json:"originalPrescription,omitempty"`
	// Recipient of benefits payable
	Payee *ExplanationOfBenefitPayee `json:"payee,omitempty"`
	// Treatment Referral
	Referral *Reference `json:"referral,omitempty"`
	// Servicing Facility
	Facility *Reference `json:"facility,omitempty"`
	// Claim reference
	Claim *Reference `json:"claim,omitempty"`
	// Claim response reference
	ClaimResponse *Reference `json:"claimResponse,omitempty"`
	// queued | complete | error | partial
	Outcome string `json:"outcome"`
	// Disposition Message
	Disposition *string `json:"disposition,omitempty"`
	// Preauthorization reference
	PreAuthRef []string `json:"preAuthRef,omitempty"`
	// Preauthorization in-effect period
	PreAuthRefPeriod []Period `json:"preAuthRefPeriod,omitempty"`
	// Care Team members
	CareTeam []ExplanationOfBenefitCareTeam `json:"careTeam,omitempty"`
	// Supporting information
	SupportingInfo []ExplanationOfBenefitSupportingInfo `json:"supportingInfo,omitempty"`
	// Pertinent diagnosis information
	Diagnosis []ExplanationOfBenefitDiagnosis `json:"diagnosis,omitempty"`
	// Clinical procedures performed
	Procedure []ExplanationOfBenefitProcedure `json:"procedure,omitempty"`
	// Precedence (primary, secondary, etc.)
	Precedence *int `json:"precedence,omitempty"`
	// Patient insurance information
	Insurance []ExplanationOfBenefitInsurance `json:"insurance,omitempty"`
	// Details of the event
	Accident *ExplanationOfBenefitAccident `json:"accident,omitempty"`
	// Product or service provided
	Item []ExplanationOfBenefitItem `json:"item,omitempty"`
	// Insurer added line items
	AddItem []ExplanationOfBenefitAddItem `json:"addItem,omitempty"`
	// Header-level adjudication
	Adjudication []ExplanationOfBenefitAdjudication `json:"adjudication,omitempty"`
	// Adjudication totals
	Total []ExplanationOfBenefitTotal `json:"total,omitempty"`
	// Payment Details
	Payment *ExplanationOfBenefitPayment `json:"payment,omitempty"`
	// Printed form identifier
	FormCode *CodeableConcept `json:"formCode,omitempty"`
	// Printed reference or actual form
	Form *Attachment `json:"form,omitempty"`
	// Note concerning adjudication
	ProcessNote []ExplanationOfBenefitProcessNote `json:"processNote,omitempty"`
	// When the benefits are applicable
	BenefitPeriod *Period `json:"benefitPeriod,omitempty"`
	// Balance by Benefit Category
	BenefitBalance []ExplanationOfBenefitBenefitBalance `json:"benefitBalance,omitempty"`
}
