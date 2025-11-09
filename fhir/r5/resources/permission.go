package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypePermission is the FHIR resource type name for Permission.
const ResourceTypePermission = "Permission"

// PermissionJustification represents a FHIR BackboneElement for Permission.justification.
type PermissionJustification struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The regulatory grounds upon which this Permission builds
	Basis []CodeableConcept `json:"basis,omitempty"`
	// Justifing rational
	Evidence []Reference `json:"evidence,omitempty"`
}

// PermissionRuleDataResource represents a FHIR BackboneElement for Permission.rule.data.resource.
type PermissionRuleDataResource struct {
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

// PermissionRuleData represents a FHIR BackboneElement for Permission.rule.data.
type PermissionRuleData struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Explicit FHIR Resource references
	Resource []PermissionRuleDataResource `json:"resource,omitempty"`
	// Security tag code on .meta.security
	Security []Coding `json:"security,omitempty"`
	// Timeframe encompasing data create/update
	Period []Period `json:"period,omitempty"`
	// Expression identifying the data
	Expression *Expression `json:"expression,omitempty"`
}

// PermissionRuleActivity represents a FHIR BackboneElement for Permission.rule.activity.
type PermissionRuleActivity struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Authorized actor(s)
	Actor []Reference `json:"actor,omitempty"`
	// Actions controlled by this rule
	Action []CodeableConcept `json:"action,omitempty"`
	// The purpose for which the permission is given
	Purpose []CodeableConcept `json:"purpose,omitempty"`
}

// PermissionRule represents a FHIR BackboneElement for Permission.rule.
type PermissionRule struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// deny | permit
	Type *string `json:"type,omitempty"`
	// The selection criteria to identify data that is within scope of this provision
	Data []PermissionRuleData `json:"data,omitempty"`
	// A description or definition of which activities are allowed to be done on the data
	Activity []PermissionRuleActivity `json:"activity,omitempty"`
	// What limits apply to the use of the data
	Limit []CodeableConcept `json:"limit,omitempty"`
}

// Permission represents a FHIR Permission.
type Permission struct {
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
	// active | entered-in-error | draft | rejected
	Status string `json:"status"`
	// The person or entity that asserts the permission
	Asserter *Reference `json:"asserter,omitempty"`
	// The date that permission was asserted
	Date []primitives.DateTime `json:"date,omitempty"`
	// The period in which the permission is active
	Validity *Period `json:"validity,omitempty"`
	// The asserted justification for using the data
	Justification *PermissionJustification `json:"justification,omitempty"`
	// deny-overrides | permit-overrides | ordered-deny-overrides | ordered-permit-overrides | deny-unless-permit | permit-unless-deny
	Combining string `json:"combining"`
	// Constraints to the Permission
	Rule []PermissionRule `json:"rule,omitempty"`
}
