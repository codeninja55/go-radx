package resources

// ResourceTypeAccount is the FHIR resource type name for Account.
const ResourceTypeAccount = "Account"

// AccountCoverage represents a FHIR BackboneElement for Account.coverage.
type AccountCoverage struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The party(s), such as insurances, that may contribute to the payment of this account
	Coverage Reference `json:"coverage"`
	// The priority of the coverage in the context of this account
	Priority *int `json:"priority,omitempty"`
}

// AccountGuarantor represents a FHIR BackboneElement for Account.guarantor.
type AccountGuarantor struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Responsible entity
	Party Reference `json:"party"`
	// Credit or other hold applied
	OnHold *bool `json:"onHold,omitempty"`
	// Guarantee account during
	Period *Period `json:"period,omitempty"`
}

// Account represents a FHIR Account.
type Account struct {
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
	// Account number
	Identifier []Identifier `json:"identifier,omitempty"`
	// active | inactive | entered-in-error | on-hold | unknown
	Status string `json:"status"`
	// E.g. patient, expense, depreciation
	Type *CodeableConcept `json:"type,omitempty"`
	// Human-readable label
	Name *string `json:"name,omitempty"`
	// The entity that caused the expenses
	Subject []Reference `json:"subject,omitempty"`
	// Transaction window
	ServicePeriod *Period `json:"servicePeriod,omitempty"`
	// The party(s) that are responsible for covering the payment of this account, and what order should they be applied to the account
	Coverage []AccountCoverage `json:"coverage,omitempty"`
	// Entity managing the Account
	Owner *Reference `json:"owner,omitempty"`
	// Explanation of purpose/use
	Description *string `json:"description,omitempty"`
	// The parties ultimately responsible for balancing the Account
	Guarantor []AccountGuarantor `json:"guarantor,omitempty"`
	// Reference to a parent Account
	PartOf *Reference `json:"partOf,omitempty"`
}
