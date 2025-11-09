package resources

// ResourceTypeCoverage is the FHIR resource type name for Coverage.
const ResourceTypeCoverage = "Coverage"

// CoverageClass represents a FHIR BackboneElement for Coverage.class.
type CoverageClass struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of class such as 'group' or 'plan'
	Type CodeableConcept `json:"type"`
	// Value associated with the type
	Value string `json:"value"`
	// Human readable description of the type and value
	Name *string `json:"name,omitempty"`
}

// CoverageCostToBeneficiaryException represents a FHIR BackboneElement for Coverage.costToBeneficiary.exception.
type CoverageCostToBeneficiaryException struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Exception category
	Type CodeableConcept `json:"type"`
	// The effective period of the exception
	Period *Period `json:"period,omitempty"`
}

// CoverageCostToBeneficiary represents a FHIR BackboneElement for Coverage.costToBeneficiary.
type CoverageCostToBeneficiary struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Cost category
	Type *CodeableConcept `json:"type,omitempty"`
	// The amount or percentage due from the beneficiary
	Value any `json:"value"`
	// Exceptions for patient payments
	Exception []CoverageCostToBeneficiaryException `json:"exception,omitempty"`
}

// Coverage represents a FHIR Coverage.
type Coverage struct {
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
	// Business Identifier for the coverage
	Identifier []Identifier `json:"identifier,omitempty"`
	// active | cancelled | draft | entered-in-error
	Status string `json:"status"`
	// Coverage category such as medical or accident
	Type *CodeableConcept `json:"type,omitempty"`
	// Owner of the policy
	PolicyHolder *Reference `json:"policyHolder,omitempty"`
	// Subscriber to the policy
	Subscriber *Reference `json:"subscriber,omitempty"`
	// ID assigned to the subscriber
	SubscriberId *string `json:"subscriberId,omitempty"`
	// Plan beneficiary
	Beneficiary Reference `json:"beneficiary"`
	// Dependent number
	Dependent *string `json:"dependent,omitempty"`
	// Beneficiary relationship to the subscriber
	Relationship *CodeableConcept `json:"relationship,omitempty"`
	// Coverage start and end dates
	Period *Period `json:"period,omitempty"`
	// Issuer of the policy
	Payor []Reference `json:"payor,omitempty"`
	// Additional coverage classifications
	Class []CoverageClass `json:"class,omitempty"`
	// Relative order of the coverage
	Order *int `json:"order,omitempty"`
	// Insurer network
	Network *string `json:"network,omitempty"`
	// Patient payments for services/products
	CostToBeneficiary []CoverageCostToBeneficiary `json:"costToBeneficiary,omitempty"`
	// Reimbursement to insurer
	Subrogation *bool `json:"subrogation,omitempty"`
	// Contract details
	Contract []Reference `json:"contract,omitempty"`
}
