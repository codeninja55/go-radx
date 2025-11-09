package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeConsent is the FHIR resource type name for Consent.
const ResourceTypeConsent = "Consent"

// ConsentPolicyBasis represents a FHIR BackboneElement for Consent.policyBasis.
type ConsentPolicyBasis struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Reference backing policy resource
	Reference *Reference `json:"reference,omitempty"`
	// URL to a computable backing policy
	URL *string `json:"url,omitempty"`
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
	// Business case of verification
	VerificationType *CodeableConcept `json:"verificationType,omitempty"`
	// Person conducting verification
	VerifiedBy *Reference `json:"verifiedBy,omitempty"`
	// Person who verified
	VerifiedWith *Reference `json:"verifiedWith,omitempty"`
	// When consent verified
	VerificationDate []primitives.DateTime `json:"verificationDate,omitempty"`
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
	Role *CodeableConcept `json:"role,omitempty"`
	// Resource for the actor (or group, by role)
	Reference *Reference `json:"reference,omitempty"`
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
	// Timeframe for this provision
	Period *Period `json:"period,omitempty"`
	// Who|what controlled by this provision (or group, by role)
	Actor []ConsentProvisionActor `json:"actor,omitempty"`
	// Actions controlled by this provision
	Action []CodeableConcept `json:"action,omitempty"`
	// Security Labels that define affected resources
	SecurityLabel []Coding `json:"securityLabel,omitempty"`
	// Context of activities covered by this provision
	Purpose []Coding `json:"purpose,omitempty"`
	// e.g. Resource Type, Profile, CDA, etc
	DocumentType []Coding `json:"documentType,omitempty"`
	// e.g. Resource Type, Profile, etc
	ResourceType []Coding `json:"resourceType,omitempty"`
	// e.g. LOINC or SNOMED CT code, etc. in the content
	Code []CodeableConcept `json:"code,omitempty"`
	// Timeframe for data controlled by this provision
	DataPeriod *Period `json:"dataPeriod,omitempty"`
	// Data controlled by this provision
	Data []ConsentProvisionData `json:"data,omitempty"`
	// A computable expression of the consent
	Expression *Expression `json:"expression,omitempty"`
	// Nested Exception Provisions
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
	// draft | active | inactive | not-done | entered-in-error | unknown
	Status string `json:"status"`
	// Classification of the consent statement - for indexing/retrieval
	Category []CodeableConcept `json:"category,omitempty"`
	// Who the consent applies to
	Subject *Reference `json:"subject,omitempty"`
	// Fully executed date of the consent
	Date *primitives.Date `json:"date,omitempty"`
	// Effective period for this Consent
	Period *Period `json:"period,omitempty"`
	// Who is granting rights according to the policy and rules
	Grantor []Reference `json:"grantor,omitempty"`
	// Who is agreeing to the policy and rules
	Grantee []Reference `json:"grantee,omitempty"`
	// Consent workflow management
	Manager []Reference `json:"manager,omitempty"`
	// Consent Enforcer
	Controller []Reference `json:"controller,omitempty"`
	// Source from which this consent is taken
	SourceAttachment []Attachment `json:"sourceAttachment,omitempty"`
	// Source from which this consent is taken
	SourceReference []Reference `json:"sourceReference,omitempty"`
	// Regulations establishing base Consent
	RegulatoryBasis []CodeableConcept `json:"regulatoryBasis,omitempty"`
	// Computable version of the backing policy
	PolicyBasis *ConsentPolicyBasis `json:"policyBasis,omitempty"`
	// Human Readable Policy
	PolicyText []Reference `json:"policyText,omitempty"`
	// Consent Verified by patient or family
	Verification []ConsentVerification `json:"verification,omitempty"`
	// deny | permit
	Decision *string `json:"decision,omitempty"`
	// Constraints to the base Consent.policyRule/Consent.policy
	Provision []ConsentProvision `json:"provision,omitempty"`
}
