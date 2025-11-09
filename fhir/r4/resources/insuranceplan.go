package resources

// ResourceTypeInsurancePlan is the FHIR resource type name for InsurancePlan.
const ResourceTypeInsurancePlan = "InsurancePlan"

// InsurancePlanContact represents a FHIR BackboneElement for InsurancePlan.contact.
type InsurancePlanContact struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The type of contact
	Purpose *CodeableConcept `json:"purpose,omitempty"`
	// A name associated with the contact
	Name *HumanName `json:"name,omitempty"`
	// Contact details (telephone, email, etc.)  for a contact
	Telecom []ContactPoint `json:"telecom,omitempty"`
	// Visiting or postal addresses for the contact
	Address *Address `json:"address,omitempty"`
}

// InsurancePlanCoverageBenefitLimit represents a FHIR BackboneElement for InsurancePlan.coverage.benefit.limit.
type InsurancePlanCoverageBenefitLimit struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Maximum value allowed
	Value *Quantity `json:"value,omitempty"`
	// Benefit limit details
	Code *CodeableConcept `json:"code,omitempty"`
}

// InsurancePlanCoverageBenefit represents a FHIR BackboneElement for InsurancePlan.coverage.benefit.
type InsurancePlanCoverageBenefit struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of benefit
	Type CodeableConcept `json:"type"`
	// Referral requirements
	Requirement *string `json:"requirement,omitempty"`
	// Benefit limits
	Limit []InsurancePlanCoverageBenefitLimit `json:"limit,omitempty"`
}

// InsurancePlanCoverage represents a FHIR BackboneElement for InsurancePlan.coverage.
type InsurancePlanCoverage struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of coverage
	Type CodeableConcept `json:"type"`
	// What networks provide coverage
	Network []Reference `json:"network,omitempty"`
	// List of benefits
	Benefit []InsurancePlanCoverageBenefit `json:"benefit,omitempty"`
}

// InsurancePlanPlanGeneralCost represents a FHIR BackboneElement for InsurancePlan.plan.generalCost.
type InsurancePlanPlanGeneralCost struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of cost
	Type *CodeableConcept `json:"type,omitempty"`
	// Number of enrollees
	GroupSize *int `json:"groupSize,omitempty"`
	// Cost value
	Cost *Money `json:"cost,omitempty"`
	// Additional cost information
	Comment *string `json:"comment,omitempty"`
}

// InsurancePlanPlanSpecificCostBenefitCost represents a FHIR BackboneElement for InsurancePlan.plan.specificCost.benefit.cost.
type InsurancePlanPlanSpecificCostBenefitCost struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of cost
	Type CodeableConcept `json:"type"`
	// in-network | out-of-network | other
	Applicability *CodeableConcept `json:"applicability,omitempty"`
	// Additional information about the cost
	Qualifiers []CodeableConcept `json:"qualifiers,omitempty"`
	// The actual cost value
	Value *Quantity `json:"value,omitempty"`
}

// InsurancePlanPlanSpecificCostBenefit represents a FHIR BackboneElement for InsurancePlan.plan.specificCost.benefit.
type InsurancePlanPlanSpecificCostBenefit struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of specific benefit
	Type CodeableConcept `json:"type"`
	// List of the costs
	Cost []InsurancePlanPlanSpecificCostBenefitCost `json:"cost,omitempty"`
}

// InsurancePlanPlanSpecificCost represents a FHIR BackboneElement for InsurancePlan.plan.specificCost.
type InsurancePlanPlanSpecificCost struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// General category of benefit
	Category CodeableConcept `json:"category"`
	// Benefits list
	Benefit []InsurancePlanPlanSpecificCostBenefit `json:"benefit,omitempty"`
}

// InsurancePlanPlan represents a FHIR BackboneElement for InsurancePlan.plan.
type InsurancePlanPlan struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Business Identifier for Product
	Identifier []Identifier `json:"identifier,omitempty"`
	// Type of plan
	Type *CodeableConcept `json:"type,omitempty"`
	// Where product applies
	CoverageArea []Reference `json:"coverageArea,omitempty"`
	// What networks provide coverage
	Network []Reference `json:"network,omitempty"`
	// Overall costs
	GeneralCost []InsurancePlanPlanGeneralCost `json:"generalCost,omitempty"`
	// Specific costs
	SpecificCost []InsurancePlanPlanSpecificCost `json:"specificCost,omitempty"`
}

// InsurancePlan represents a FHIR InsurancePlan.
type InsurancePlan struct {
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
	// Business Identifier for Product
	Identifier []Identifier `json:"identifier,omitempty"`
	// draft | active | retired | unknown
	Status *string `json:"status,omitempty"`
	// Kind of product
	Type []CodeableConcept `json:"type,omitempty"`
	// Official name
	Name *string `json:"name,omitempty"`
	// Alternate names
	Alias []string `json:"alias,omitempty"`
	// When the product is available
	Period *Period `json:"period,omitempty"`
	// Plan issuer
	OwnedBy *Reference `json:"ownedBy,omitempty"`
	// Product administrator
	AdministeredBy *Reference `json:"administeredBy,omitempty"`
	// Where product applies
	CoverageArea []Reference `json:"coverageArea,omitempty"`
	// Contact for the product
	Contact []InsurancePlanContact `json:"contact,omitempty"`
	// Technical endpoint
	Endpoint []Reference `json:"endpoint,omitempty"`
	// What networks are Included
	Network []Reference `json:"network,omitempty"`
	// Coverage details
	Coverage []InsurancePlanCoverage `json:"coverage,omitempty"`
	// Plan details
	Plan []InsurancePlanPlan `json:"plan,omitempty"`
}
