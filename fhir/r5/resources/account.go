package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

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

// AccountDiagnosis represents a FHIR BackboneElement for Account.diagnosis.
type AccountDiagnosis struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Ranking of the diagnosis (for each type)
	Sequence *int `json:"sequence,omitempty"`
	// The diagnosis relevant to the account
	Condition CodeableReference `json:"condition"`
	// Date of the diagnosis (when coded diagnosis)
	DateOfDiagnosis *primitives.DateTime `json:"dateOfDiagnosis,omitempty"`
	// Type that this diagnosis has relevant to the account (e.g. admission, billing, discharge …)
	Type []CodeableConcept `json:"type,omitempty"`
	// Diagnosis present on Admission
	OnAdmission *bool `json:"onAdmission,omitempty"`
	// Package Code specific for billing
	PackageCode []CodeableConcept `json:"packageCode,omitempty"`
}

// AccountProcedure represents a FHIR BackboneElement for Account.procedure.
type AccountProcedure struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Ranking of the procedure (for each type)
	Sequence *int `json:"sequence,omitempty"`
	// The procedure relevant to the account
	Code CodeableReference `json:"code"`
	// Date of the procedure (when coded procedure)
	DateOfService *primitives.DateTime `json:"dateOfService,omitempty"`
	// How this procedure value should be used in charging the account
	Type []CodeableConcept `json:"type,omitempty"`
	// Package Code specific for billing
	PackageCode []CodeableConcept `json:"packageCode,omitempty"`
	// Any devices that were associated with the procedure
	Device []Reference `json:"device,omitempty"`
}

// AccountRelatedAccount represents a FHIR BackboneElement for Account.relatedAccount.
type AccountRelatedAccount struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Relationship of the associated Account
	Relationship *CodeableConcept `json:"relationship,omitempty"`
	// Reference to an associated Account
	Account Reference `json:"account"`
}

// AccountBalance represents a FHIR BackboneElement for Account.balance.
type AccountBalance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Who is expected to pay this part of the balance
	Aggregate *CodeableConcept `json:"aggregate,omitempty"`
	// current | 30 | 60 | 90 | 120
	Term *CodeableConcept `json:"term,omitempty"`
	// Estimated balance
	Estimate *bool `json:"estimate,omitempty"`
	// Calculated amount
	Amount Money `json:"amount"`
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
	// Tracks the lifecycle of the account through the billing process
	BillingStatus *CodeableConcept `json:"billingStatus,omitempty"`
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
	// The list of diagnoses relevant to this account
	Diagnosis []AccountDiagnosis `json:"diagnosis,omitempty"`
	// The list of procedures relevant to this account
	Procedure []AccountProcedure `json:"procedure,omitempty"`
	// Other associated accounts related to this account
	RelatedAccount []AccountRelatedAccount `json:"relatedAccount,omitempty"`
	// The base or default currency
	Currency *CodeableConcept `json:"currency,omitempty"`
	// Calculated account balance(s)
	Balance []AccountBalance `json:"balance,omitempty"`
	// Time the balance amount was calculated
	CalculatedAt *primitives.Instant `json:"calculatedAt,omitempty"`
}
