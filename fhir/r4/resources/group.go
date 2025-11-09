package resources

// ResourceTypeGroup is the FHIR resource type name for Group.
const ResourceTypeGroup = "Group"

// GroupCharacteristic represents a FHIR BackboneElement for Group.characteristic.
type GroupCharacteristic struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Kind of characteristic
	Code CodeableConcept `json:"code"`
	// Value held by characteristic
	Value any `json:"value"`
	// Group includes or excludes
	Exclude bool `json:"exclude"`
	// Period over which characteristic is tested
	Period *Period `json:"period,omitempty"`
}

// GroupMember represents a FHIR BackboneElement for Group.member.
type GroupMember struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Reference to the group member
	Entity Reference `json:"entity"`
	// Period member belonged to the group
	Period *Period `json:"period,omitempty"`
	// If member is no longer in group
	Inactive *bool `json:"inactive,omitempty"`
}

// Group represents a FHIR Group.
type Group struct {
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
	// Unique id
	Identifier []Identifier `json:"identifier,omitempty"`
	// Whether this group's record is in active use
	Active *bool `json:"active,omitempty"`
	// person | animal | practitioner | device | medication | substance
	Type string `json:"type"`
	// Descriptive or actual
	Actual bool `json:"actual"`
	// Kind of Group members
	Code *CodeableConcept `json:"code,omitempty"`
	// Label for Group
	Name *string `json:"name,omitempty"`
	// Number of members
	Quantity *uint `json:"quantity,omitempty"`
	// Entity that is the custodian of the Group's definition
	ManagingEntity *Reference `json:"managingEntity,omitempty"`
	// Include / Exclude group members by Trait
	Characteristic []GroupCharacteristic `json:"characteristic,omitempty"`
	// Who or what is in group
	Member []GroupMember `json:"member,omitempty"`
}
