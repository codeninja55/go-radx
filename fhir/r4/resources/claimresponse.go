package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeClaimResponse is the FHIR resource type name for ClaimResponse.
const ResourceTypeClaimResponse = "ClaimResponse"

// ClaimResponseItemAdjudication represents a FHIR BackboneElement for ClaimResponse.item.adjudication.
type ClaimResponseItemAdjudication struct {
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
	// Non-monetary value
	Value *float64 `json:"value,omitempty"`
}

// ClaimResponseItemDetailAdjudication represents a FHIR BackboneElement for ClaimResponse.item.detail.adjudication.
type ClaimResponseItemDetailAdjudication struct {
}

// ClaimResponseItemDetailSubDetailAdjudication represents a FHIR BackboneElement for ClaimResponse.item.detail.subDetail.adjudication.
type ClaimResponseItemDetailSubDetailAdjudication struct {
}

// ClaimResponseItemDetailSubDetail represents a FHIR BackboneElement for ClaimResponse.item.detail.subDetail.
type ClaimResponseItemDetailSubDetail struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Claim sub-detail instance identifier
	SubDetailSequence int `json:"subDetailSequence"`
	// Applicable note numbers
	NoteNumber []int `json:"noteNumber,omitempty"`
	// Subdetail level adjudication details
	Adjudication []ClaimResponseItemDetailSubDetailAdjudication `json:"adjudication,omitempty"`
}

// ClaimResponseItemDetail represents a FHIR BackboneElement for ClaimResponse.item.detail.
type ClaimResponseItemDetail struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Claim detail instance identifier
	DetailSequence int `json:"detailSequence"`
	// Applicable note numbers
	NoteNumber []int `json:"noteNumber,omitempty"`
	// Detail level adjudication details
	Adjudication []ClaimResponseItemDetailAdjudication `json:"adjudication,omitempty"`
	// Adjudication for claim sub-details
	SubDetail []ClaimResponseItemDetailSubDetail `json:"subDetail,omitempty"`
}

// ClaimResponseItem represents a FHIR BackboneElement for ClaimResponse.item.
type ClaimResponseItem struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Claim item instance identifier
	ItemSequence int `json:"itemSequence"`
	// Applicable note numbers
	NoteNumber []int `json:"noteNumber,omitempty"`
	// Adjudication details
	Adjudication []ClaimResponseItemAdjudication `json:"adjudication,omitempty"`
	// Adjudication for claim details
	Detail []ClaimResponseItemDetail `json:"detail,omitempty"`
}

// ClaimResponseAddItemAdjudication represents a FHIR BackboneElement for ClaimResponse.addItem.adjudication.
type ClaimResponseAddItemAdjudication struct {
}

// ClaimResponseAddItemDetailAdjudication represents a FHIR BackboneElement for ClaimResponse.addItem.detail.adjudication.
type ClaimResponseAddItemDetailAdjudication struct {
}

// ClaimResponseAddItemDetailSubDetailAdjudication represents a FHIR BackboneElement for ClaimResponse.addItem.detail.subDetail.adjudication.
type ClaimResponseAddItemDetailSubDetailAdjudication struct {
}

// ClaimResponseAddItemDetailSubDetail represents a FHIR BackboneElement for ClaimResponse.addItem.detail.subDetail.
type ClaimResponseAddItemDetailSubDetail struct {
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
	// Added items detail adjudication
	Adjudication []ClaimResponseAddItemDetailSubDetailAdjudication `json:"adjudication,omitempty"`
}

// ClaimResponseAddItemDetail represents a FHIR BackboneElement for ClaimResponse.addItem.detail.
type ClaimResponseAddItemDetail struct {
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
	// Added items detail adjudication
	Adjudication []ClaimResponseAddItemDetailAdjudication `json:"adjudication,omitempty"`
	// Insurer added line items
	SubDetail []ClaimResponseAddItemDetailSubDetail `json:"subDetail,omitempty"`
}

// ClaimResponseAddItem represents a FHIR BackboneElement for ClaimResponse.addItem.
type ClaimResponseAddItem struct {
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
	SubdetailSequence []int `json:"subdetailSequence,omitempty"`
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
	Adjudication []ClaimResponseAddItemAdjudication `json:"adjudication,omitempty"`
	// Insurer added line details
	Detail []ClaimResponseAddItemDetail `json:"detail,omitempty"`
}

// ClaimResponseAdjudication represents a FHIR BackboneElement for ClaimResponse.adjudication.
type ClaimResponseAdjudication struct {
}

// ClaimResponseTotal represents a FHIR BackboneElement for ClaimResponse.total.
type ClaimResponseTotal struct {
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

// ClaimResponsePayment represents a FHIR BackboneElement for ClaimResponse.payment.
type ClaimResponsePayment struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Partial or complete payment
	Type CodeableConcept `json:"type"`
	// Payment adjustment for non-claim issues
	Adjustment *Money `json:"adjustment,omitempty"`
	// Explanation for the adjustment
	AdjustmentReason *CodeableConcept `json:"adjustmentReason,omitempty"`
	// Expected date of payment
	Date *primitives.Date `json:"date,omitempty"`
	// Payable amount after adjustment
	Amount Money `json:"amount"`
	// Business identifier for the payment
	Identifier *Identifier `json:"identifier,omitempty"`
}

// ClaimResponseProcessNote represents a FHIR BackboneElement for ClaimResponse.processNote.
type ClaimResponseProcessNote struct {
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
	Text string `json:"text"`
	// Language of the text
	Language *CodeableConcept `json:"language,omitempty"`
}

// ClaimResponseInsurance represents a FHIR BackboneElement for ClaimResponse.insurance.
type ClaimResponseInsurance struct {
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
	// Insurance information
	Coverage Reference `json:"coverage"`
	// Additional provider contract number
	BusinessArrangement *string `json:"businessArrangement,omitempty"`
	// Adjudication results
	ClaimResponse *Reference `json:"claimResponse,omitempty"`
}

// ClaimResponseError represents a FHIR BackboneElement for ClaimResponse.error.
type ClaimResponseError struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Item sequence number
	ItemSequence *int `json:"itemSequence,omitempty"`
	// Detail sequence number
	DetailSequence *int `json:"detailSequence,omitempty"`
	// Subdetail sequence number
	SubDetailSequence *int `json:"subDetailSequence,omitempty"`
	// Error code detailing processing issues
	Code CodeableConcept `json:"code"`
}

// ClaimResponse represents a FHIR ClaimResponse.
type ClaimResponse struct {
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
	// Business Identifier for a claim response
	Identifier []Identifier `json:"identifier,omitempty"`
	// active | cancelled | draft | entered-in-error
	Status string `json:"status"`
	// More granular claim type
	Type CodeableConcept `json:"type"`
	// More granular claim type
	SubType *CodeableConcept `json:"subType,omitempty"`
	// claim | preauthorization | predetermination
	Use string `json:"use"`
	// The recipient of the products and services
	Patient Reference `json:"patient"`
	// Response creation date
	Created primitives.DateTime `json:"created"`
	// Party responsible for reimbursement
	Insurer Reference `json:"insurer"`
	// Party responsible for the claim
	Requestor *Reference `json:"requestor,omitempty"`
	// Id of resource triggering adjudication
	Request *Reference `json:"request,omitempty"`
	// queued | complete | error | partial
	Outcome string `json:"outcome"`
	// Disposition Message
	Disposition *string `json:"disposition,omitempty"`
	// Preauthorization reference
	PreAuthRef *string `json:"preAuthRef,omitempty"`
	// Preauthorization reference effective period
	PreAuthPeriod *Period `json:"preAuthPeriod,omitempty"`
	// Party to be paid any benefits payable
	PayeeType *CodeableConcept `json:"payeeType,omitempty"`
	// Adjudication for claim line items
	Item []ClaimResponseItem `json:"item,omitempty"`
	// Insurer added line items
	AddItem []ClaimResponseAddItem `json:"addItem,omitempty"`
	// Header-level adjudication
	Adjudication []ClaimResponseAdjudication `json:"adjudication,omitempty"`
	// Adjudication totals
	Total []ClaimResponseTotal `json:"total,omitempty"`
	// Payment Details
	Payment *ClaimResponsePayment `json:"payment,omitempty"`
	// Funds reserved status
	FundsReserve *CodeableConcept `json:"fundsReserve,omitempty"`
	// Printed form identifier
	FormCode *CodeableConcept `json:"formCode,omitempty"`
	// Printed reference or actual form
	Form *Attachment `json:"form,omitempty"`
	// Note concerning adjudication
	ProcessNote []ClaimResponseProcessNote `json:"processNote,omitempty"`
	// Request for additional information
	CommunicationRequest []Reference `json:"communicationRequest,omitempty"`
	// Patient insurance information
	Insurance []ClaimResponseInsurance `json:"insurance,omitempty"`
	// Processing errors
	Error []ClaimResponseError `json:"error,omitempty"`
}
