package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeVerificationResult is the FHIR resource type name for VerificationResult.
const ResourceTypeVerificationResult = "VerificationResult"

// VerificationResultPrimarySource represents a FHIR BackboneElement for VerificationResult.primarySource.
type VerificationResultPrimarySource struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Reference to the primary source
	Who *Reference `json:"who,omitempty"`
	// Type of primary source (License Board; Primary Education; Continuing Education; Postal Service; Relationship owner; Registration Authority; legal source; issuing source; authoritative source)
	Type []CodeableConcept `json:"type,omitempty"`
	// Method for exchanging information with the primary source
	CommunicationMethod []CodeableConcept `json:"communicationMethod,omitempty"`
	// successful | failed | unknown
	ValidationStatus *CodeableConcept `json:"validationStatus,omitempty"`
	// When the target was validated against the primary source
	ValidationDate *primitives.DateTime `json:"validationDate,omitempty"`
	// yes | no | undetermined
	CanPushUpdates *CodeableConcept `json:"canPushUpdates,omitempty"`
	// specific | any | source
	PushTypeAvailable []CodeableConcept `json:"pushTypeAvailable,omitempty"`
}

// VerificationResultAttestation represents a FHIR BackboneElement for VerificationResult.attestation.
type VerificationResultAttestation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The individual or organization attesting to information
	Who *Reference `json:"who,omitempty"`
	// When the who is asserting on behalf of another (organization or individual)
	OnBehalfOf *Reference `json:"onBehalfOf,omitempty"`
	// The method by which attested information was submitted/retrieved
	CommunicationMethod *CodeableConcept `json:"communicationMethod,omitempty"`
	// The date the information was attested to
	Date *primitives.Date `json:"date,omitempty"`
	// A digital identity certificate associated with the attestation source
	SourceIdentityCertificate *string `json:"sourceIdentityCertificate,omitempty"`
	// A digital identity certificate associated with the proxy entity submitting attested information on behalf of the attestation source
	ProxyIdentityCertificate *string `json:"proxyIdentityCertificate,omitempty"`
	// Proxy signature
	ProxySignature *Signature `json:"proxySignature,omitempty"`
	// Attester signature
	SourceSignature *Signature `json:"sourceSignature,omitempty"`
}

// VerificationResultValidator represents a FHIR BackboneElement for VerificationResult.validator.
type VerificationResultValidator struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Reference to the organization validating information
	Organization Reference `json:"organization"`
	// A digital identity certificate associated with the validator
	IdentityCertificate *string `json:"identityCertificate,omitempty"`
	// Validator signature
	AttestationSignature *Signature `json:"attestationSignature,omitempty"`
}

// VerificationResult represents a FHIR VerificationResult.
type VerificationResult struct {
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
	// A resource that was validated
	Target []Reference `json:"target,omitempty"`
	// The fhirpath location(s) within the resource that was validated
	TargetLocation []string `json:"targetLocation,omitempty"`
	// none | initial | periodic
	Need *CodeableConcept `json:"need,omitempty"`
	// attested | validated | in-process | req-revalid | val-fail | reval-fail
	Status string `json:"status"`
	// When the validation status was updated
	StatusDate *primitives.DateTime `json:"statusDate,omitempty"`
	// nothing | primary | multiple
	ValidationType *CodeableConcept `json:"validationType,omitempty"`
	// The primary process by which the target is validated (edit check; value set; primary source; multiple sources; standalone; in context)
	ValidationProcess []CodeableConcept `json:"validationProcess,omitempty"`
	// Frequency of revalidation
	Frequency *Timing `json:"frequency,omitempty"`
	// The date/time validation was last completed (including failed validations)
	LastPerformed *primitives.DateTime `json:"lastPerformed,omitempty"`
	// The date when target is next validated, if appropriate
	NextScheduled *primitives.Date `json:"nextScheduled,omitempty"`
	// fatal | warn | rec-only | none
	FailureAction *CodeableConcept `json:"failureAction,omitempty"`
	// Information about the primary source(s) involved in validation
	PrimarySource []VerificationResultPrimarySource `json:"primarySource,omitempty"`
	// Information about the entity attesting to information
	Attestation *VerificationResultAttestation `json:"attestation,omitempty"`
	// Information about the entity validating information
	Validator []VerificationResultValidator `json:"validator,omitempty"`
}
