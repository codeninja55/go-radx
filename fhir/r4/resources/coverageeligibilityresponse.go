package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeCoverageEligibilityResponse is the FHIR resource type name for CoverageEligibilityResponse.
const ResourceTypeCoverageEligibilityResponse = "CoverageEligibilityResponse"

// CoverageEligibilityResponseInsuranceItemBenefit represents a FHIR BackboneElement for CoverageEligibilityResponse.insurance.item.benefit.
type CoverageEligibilityResponseInsuranceItemBenefit struct {
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

// CoverageEligibilityResponseInsuranceItem represents a FHIR BackboneElement for CoverageEligibilityResponse.insurance.item.
type CoverageEligibilityResponseInsuranceItem struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Benefit classification
	Category *CodeableConcept `json:"category,omitempty"`
	// Billing, service, product, or drug code
	ProductOrService *CodeableConcept `json:"productOrService,omitempty"`
	// Product or service billing modifiers
	Modifier []CodeableConcept `json:"modifier,omitempty"`
	// Performing practitioner
	Provider *Reference `json:"provider,omitempty"`
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
	Benefit []CoverageEligibilityResponseInsuranceItemBenefit `json:"benefit,omitempty"`
	// Authorization required flag
	AuthorizationRequired *bool `json:"authorizationRequired,omitempty"`
	// Type of required supporting materials
	AuthorizationSupporting []CodeableConcept `json:"authorizationSupporting,omitempty"`
	// Preauthorization requirements endpoint
	AuthorizationUrl *string `json:"authorizationUrl,omitempty"`
}

// CoverageEligibilityResponseInsurance represents a FHIR BackboneElement for CoverageEligibilityResponse.insurance.
type CoverageEligibilityResponseInsurance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Insurance information
	Coverage Reference `json:"coverage"`
	// Coverage inforce indicator
	Inforce *bool `json:"inforce,omitempty"`
	// When the benefits are applicable
	BenefitPeriod *Period `json:"benefitPeriod,omitempty"`
	// Benefits and authorization details
	Item []CoverageEligibilityResponseInsuranceItem `json:"item,omitempty"`
}

// CoverageEligibilityResponseError represents a FHIR BackboneElement for CoverageEligibilityResponse.error.
type CoverageEligibilityResponseError struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Error code detailing processing issues
	Code CodeableConcept `json:"code"`
}

// CoverageEligibilityResponse represents a FHIR CoverageEligibilityResponse.
type CoverageEligibilityResponse struct {
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
	// auth-requirements | benefits | discovery | validation
	Purpose []string `json:"purpose,omitempty"`
	// Intended recipient of products and services
	Patient Reference `json:"patient"`
	// Estimated date or dates of service
	Serviced *any `json:"serviced,omitempty"`
	// Response creation date
	Created primitives.DateTime `json:"created"`
	// Party responsible for the request
	Requestor *Reference `json:"requestor,omitempty"`
	// Eligibility request reference
	Request Reference `json:"request"`
	// queued | complete | error | partial
	Outcome string `json:"outcome"`
	// Disposition Message
	Disposition *string `json:"disposition,omitempty"`
	// Coverage issuer
	Insurer Reference `json:"insurer"`
	// Patient insurance information
	Insurance []CoverageEligibilityResponseInsurance `json:"insurance,omitempty"`
	// Preauthorization reference
	PreAuthRef *string `json:"preAuthRef,omitempty"`
	// Printed form identifier
	Form *CodeableConcept `json:"form,omitempty"`
	// Processing errors
	Error []CoverageEligibilityResponseError `json:"error,omitempty"`
}
