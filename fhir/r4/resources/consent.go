package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeConsent is the FHIR resource type name for Consent.
const ResourceTypeConsent = "Consent"

// ConsentPolicy represents a FHIR BackboneElement for Consent.policy.
type ConsentPolicy struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Enforcement source for policy
	Authority *string `json:"authority,omitempty"`
	// Specific policy covered by this consent
	URI *string `json:"uri,omitempty"`
}

// ConsentVerification represents a FHIR BackboneElement for Consent.verification.
type ConsentVerification struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Has been verified
	Verified bool `json:"verified"`
	// Person who verified
	VerifiedWith *Reference `json:"verifiedWith,omitempty"`
	// When consent verified
	VerificationDate *primitives.DateTime `json:"verificationDate,omitempty"`
}

// ConsentProvisionActor represents a FHIR BackboneElement for Consent.provision.actor.
type ConsentProvisionActor struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// How the actor is involved
	Role CodeableConcept `json:"role"`
	// Resource for the actor (or group, by role)
	Reference Reference `json:"reference"`
}

// ConsentProvisionData represents a FHIR BackboneElement for Consent.provision.data.
type ConsentProvisionData struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// instance | related | dependents | authoredby
	Meaning string `json:"meaning"`
	// The actual data reference
	Reference Reference `json:"reference"`
}

// ConsentProvisionProvision represents a FHIR BackboneElement for Consent.provision.provision.
type ConsentProvisionProvision struct {
}

// ConsentProvision represents a FHIR BackboneElement for Consent.provision.
type ConsentProvision struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// deny | permit
	Type *string `json:"type,omitempty"`
	// Timeframe for this rule
	Period *Period `json:"period,omitempty"`
	// Who|what controlled by this rule (or group, by role)
	Actor []ConsentProvisionActor `json:"actor,omitempty"`
	// Actions controlled by this rule
	Action []CodeableConcept `json:"action,omitempty"`
	// Security Labels that define affected resources
	SecurityLabel []Coding `json:"securityLabel,omitempty"`
	// Context of activities covered by this rule
	Purpose []Coding `json:"purpose,omitempty"`
	// e.g. Resource Type, Profile, CDA, etc.
	Class []Coding `json:"class,omitempty"`
	// e.g. LOINC or SNOMED CT code, etc. in the content
	Code []CodeableConcept `json:"code,omitempty"`
	// Timeframe for data controlled by this rule
	DataPeriod *Period `json:"dataPeriod,omitempty"`
	// Data controlled by this rule
	Data []ConsentProvisionData `json:"data,omitempty"`
	// Nested Exception Rules
	Provision []ConsentProvisionProvision `json:"provision,omitempty"`
}

// Consent represents a FHIR Consent.
type Consent struct {
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
	// Identifier for this record (external references)
	Identifier []Identifier `json:"identifier,omitempty"`
	// draft | proposed | active | rejected | inactive | entered-in-error
	Status string `json:"status"`
	// Which of the four areas this resource covers (extensible)
	Scope CodeableConcept `json:"scope"`
	// Classification of the consent statement - for indexing/retrieval
	Category []CodeableConcept `json:"category,omitempty"`
	// Who the consent applies to
	Patient *Reference `json:"patient,omitempty"`
	// When this Consent was created or indexed
	DateTime *primitives.DateTime `json:"dateTime,omitempty"`
	// Who is agreeing to the policy and rules
	Performer []Reference `json:"performer,omitempty"`
	// Custodian of the consent
	Organization []Reference `json:"organization,omitempty"`
	// Source from which this consent is taken
	Source *any `json:"source,omitempty"`
	// Policies covered by this consent
	Policy []ConsentPolicy `json:"policy,omitempty"`
	// Regulation that this consents to
	PolicyRule *CodeableConcept `json:"policyRule,omitempty"`
	// Consent Verified by patient or family
	Verification []ConsentVerification `json:"verification,omitempty"`
	// Constraints to the base Consent.policyRule
	Provision *ConsentProvision `json:"provision,omitempty"`
}
