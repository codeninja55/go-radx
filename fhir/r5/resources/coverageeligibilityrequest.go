package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeCoverageEligibilityRequest is the FHIR resource type name for CoverageEligibilityRequest.
const ResourceTypeCoverageEligibilityRequest = "CoverageEligibilityRequest"

// CoverageEligibilityRequestEvent represents a FHIR BackboneElement for CoverageEligibilityRequest.event.
type CoverageEligibilityRequestEvent struct {
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

// CoverageEligibilityRequestSupportingInfo represents a FHIR BackboneElement for CoverageEligibilityRequest.supportingInfo.
type CoverageEligibilityRequestSupportingInfo struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Information instance identifier
	Sequence int `json:"sequence"`
	// Data to be provided
	Information Reference `json:"information"`
	// Applies to all items
	AppliesToAll *bool `json:"appliesToAll,omitempty"`
}

// CoverageEligibilityRequestInsurance represents a FHIR BackboneElement for CoverageEligibilityRequest.insurance.
type CoverageEligibilityRequestInsurance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Applicable coverage
	Focal *bool `json:"focal,omitempty"`
	// Insurance information
	Coverage Reference `json:"coverage"`
	// Additional provider contract number
	BusinessArrangement *string `json:"businessArrangement,omitempty"`
}

// CoverageEligibilityRequestItemDiagnosis represents a FHIR BackboneElement for CoverageEligibilityRequest.item.diagnosis.
type CoverageEligibilityRequestItemDiagnosis struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Nature of illness or problem
	Diagnosis *any `json:"diagnosis,omitempty"`
}

// CoverageEligibilityRequestItem represents a FHIR BackboneElement for CoverageEligibilityRequest.item.
type CoverageEligibilityRequestItem struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Applicable exception or supporting information
	SupportingInfoSequence []int `json:"supportingInfoSequence,omitempty"`
	// Benefit classification
	Category *CodeableConcept `json:"category,omitempty"`
	// Billing, service, product, or drug code
	ProductOrService *CodeableConcept `json:"productOrService,omitempty"`
	// Product or service billing modifiers
	Modifier []CodeableConcept `json:"modifier,omitempty"`
	// Perfoming practitioner
	Provider *Reference `json:"provider,omitempty"`
	// Count of products or services
	Quantity *Quantity `json:"quantity,omitempty"`
	// Fee, charge or cost per item
	UnitPrice *Money `json:"unitPrice,omitempty"`
	// Servicing facility
	Facility *Reference `json:"facility,omitempty"`
	// Applicable diagnosis
	Diagnosis []CoverageEligibilityRequestItemDiagnosis `json:"diagnosis,omitempty"`
	// Product or service details
	Detail []Reference `json:"detail,omitempty"`
}

// CoverageEligibilityRequest represents a FHIR CoverageEligibilityRequest.
type CoverageEligibilityRequest struct {
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
	// Business Identifier for coverage eligiblity request
	Identifier []Identifier `json:"identifier,omitempty"`
	// active | cancelled | draft | entered-in-error
	Status string `json:"status"`
	// Desired processing priority
	Priority *CodeableConcept `json:"priority,omitempty"`
	// auth-requirements | benefits | discovery | validation
	Purpose []string `json:"purpose,omitempty"`
	// Intended recipient of products and services
	Patient Reference `json:"patient"`
	// Event information
	Event []CoverageEligibilityRequestEvent `json:"event,omitempty"`
	// Estimated date or dates of service
	Serviced *any `json:"serviced,omitempty"`
	// Creation date
	Created primitives.DateTime `json:"created"`
	// Author
	Enterer *Reference `json:"enterer,omitempty"`
	// Party responsible for the request
	Provider *Reference `json:"provider,omitempty"`
	// Coverage issuer
	Insurer Reference `json:"insurer"`
	// Servicing facility
	Facility *Reference `json:"facility,omitempty"`
	// Supporting information
	SupportingInfo []CoverageEligibilityRequestSupportingInfo `json:"supportingInfo,omitempty"`
	// Patient insurance information
	Insurance []CoverageEligibilityRequestInsurance `json:"insurance,omitempty"`
	// Item to be evaluated for eligibiity
	Item []CoverageEligibilityRequestItem `json:"item,omitempty"`
}
