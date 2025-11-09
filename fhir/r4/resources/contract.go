package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeContract is the FHIR resource type name for Contract.
const ResourceTypeContract = "Contract"

// ContractContentDefinition represents a FHIR BackboneElement for Contract.contentDefinition.
type ContractContentDefinition struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Content structure and use
	Type CodeableConcept `json:"type"`
	// Detailed Content Type Definition
	SubType *CodeableConcept `json:"subType,omitempty"`
	// Publisher Entity
	Publisher *Reference `json:"publisher,omitempty"`
	// When published
	PublicationDate *primitives.DateTime `json:"publicationDate,omitempty"`
	// amended | appended | cancelled | disputed | entered-in-error | executable | executed | negotiable | offered | policy | rejected | renewed | revoked | resolved | terminated
	PublicationStatus string `json:"publicationStatus"`
	// Publication Ownership
	Copyright *string `json:"copyright,omitempty"`
}

// ContractTermSecurityLabel represents a FHIR BackboneElement for Contract.term.securityLabel.
type ContractTermSecurityLabel struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Link to Security Labels
	Number []uint `json:"number,omitempty"`
	// Confidentiality Protection
	Classification Coding `json:"classification"`
	// Applicable Policy
	Category []Coding `json:"category,omitempty"`
	// Handling Instructions
	Control []Coding `json:"control,omitempty"`
}

// ContractTermOfferParty represents a FHIR BackboneElement for Contract.term.offer.party.
type ContractTermOfferParty struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Referenced entity
	Reference []Reference `json:"reference,omitempty"`
	// Participant engagement type
	Role CodeableConcept `json:"role"`
}

// ContractTermOfferAnswer represents a FHIR BackboneElement for Contract.term.offer.answer.
type ContractTermOfferAnswer struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The actual answer response
	Value any `json:"value"`
}

// ContractTermOffer represents a FHIR BackboneElement for Contract.term.offer.
type ContractTermOffer struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Offer business ID
	Identifier []Identifier `json:"identifier,omitempty"`
	// Offer Recipient
	Party []ContractTermOfferParty `json:"party,omitempty"`
	// Negotiable offer asset
	Topic *Reference `json:"topic,omitempty"`
	// Contract Offer Type or Form
	Type *CodeableConcept `json:"type,omitempty"`
	// Accepting party choice
	Decision *CodeableConcept `json:"decision,omitempty"`
	// How decision is conveyed
	DecisionMode []CodeableConcept `json:"decisionMode,omitempty"`
	// Response to offer text
	Answer []ContractTermOfferAnswer `json:"answer,omitempty"`
	// Human readable offer text
	Text *string `json:"text,omitempty"`
	// Pointer to text
	LinkId []string `json:"linkId,omitempty"`
	// Offer restriction numbers
	SecurityLabelNumber []uint `json:"securityLabelNumber,omitempty"`
}

// ContractTermAssetContext represents a FHIR BackboneElement for Contract.term.asset.context.
type ContractTermAssetContext struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Creator,custodian or owner
	Reference *Reference `json:"reference,omitempty"`
	// Codeable asset context
	Code []CodeableConcept `json:"code,omitempty"`
	// Context description
	Text *string `json:"text,omitempty"`
}

// ContractTermAssetAnswer represents a FHIR BackboneElement for Contract.term.asset.answer.
type ContractTermAssetAnswer struct {
}

// ContractTermAssetValuedItem represents a FHIR BackboneElement for Contract.term.asset.valuedItem.
type ContractTermAssetValuedItem struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Contract Valued Item Type
	Entity *any `json:"entity,omitempty"`
	// Contract Valued Item Number
	Identifier *Identifier `json:"identifier,omitempty"`
	// Contract Valued Item Effective Tiem
	EffectiveTime *primitives.DateTime `json:"effectiveTime,omitempty"`
	// Count of Contract Valued Items
	Quantity *Quantity `json:"quantity,omitempty"`
	// Contract Valued Item fee, charge, or cost
	UnitPrice *Money `json:"unitPrice,omitempty"`
	// Contract Valued Item Price Scaling Factor
	Factor *float64 `json:"factor,omitempty"`
	// Contract Valued Item Difficulty Scaling Factor
	Points *float64 `json:"points,omitempty"`
	// Total Contract Valued Item Value
	Net *Money `json:"net,omitempty"`
	// Terms of valuation
	Payment *string `json:"payment,omitempty"`
	// When payment is due
	PaymentDate *primitives.DateTime `json:"paymentDate,omitempty"`
	// Who will make payment
	Responsible *Reference `json:"responsible,omitempty"`
	// Who will receive payment
	Recipient *Reference `json:"recipient,omitempty"`
	// Pointer to specific item
	LinkId []string `json:"linkId,omitempty"`
	// Security Labels that define affected terms
	SecurityLabelNumber []uint `json:"securityLabelNumber,omitempty"`
}

// ContractTermAsset represents a FHIR BackboneElement for Contract.term.asset.
type ContractTermAsset struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Range of asset
	Scope *CodeableConcept `json:"scope,omitempty"`
	// Asset category
	Type []CodeableConcept `json:"type,omitempty"`
	// Associated entities
	TypeReference []Reference `json:"typeReference,omitempty"`
	// Asset sub-category
	Subtype []CodeableConcept `json:"subtype,omitempty"`
	// Kinship of the asset
	Relationship *Coding `json:"relationship,omitempty"`
	// Circumstance of the asset
	Context []ContractTermAssetContext `json:"context,omitempty"`
	// Quality desctiption of asset
	Condition *string `json:"condition,omitempty"`
	// Asset availability types
	PeriodType []CodeableConcept `json:"periodType,omitempty"`
	// Time period of the asset
	Period []Period `json:"period,omitempty"`
	// Time period
	UsePeriod []Period `json:"usePeriod,omitempty"`
	// Asset clause or question text
	Text *string `json:"text,omitempty"`
	// Pointer to asset text
	LinkId []string `json:"linkId,omitempty"`
	// Response to assets
	Answer []ContractTermAssetAnswer `json:"answer,omitempty"`
	// Asset restriction numbers
	SecurityLabelNumber []uint `json:"securityLabelNumber,omitempty"`
	// Contract Valued Item List
	ValuedItem []ContractTermAssetValuedItem `json:"valuedItem,omitempty"`
}

// ContractTermActionSubject represents a FHIR BackboneElement for Contract.term.action.subject.
type ContractTermActionSubject struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Entity of the action
	Reference []Reference `json:"reference,omitempty"`
	// Role type of the agent
	Role *CodeableConcept `json:"role,omitempty"`
}

// ContractTermAction represents a FHIR BackboneElement for Contract.term.action.
type ContractTermAction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// True if the term prohibits the  action
	DoNotPerform *bool `json:"doNotPerform,omitempty"`
	// Type or form of the action
	Type CodeableConcept `json:"type"`
	// Entity of the action
	Subject []ContractTermActionSubject `json:"subject,omitempty"`
	// Purpose for the Contract Term Action
	Intent CodeableConcept `json:"intent"`
	// Pointer to specific item
	LinkId []string `json:"linkId,omitempty"`
	// State of the action
	Status CodeableConcept `json:"status"`
	// Episode associated with action
	Context *Reference `json:"context,omitempty"`
	// Pointer to specific item
	ContextLinkId []string `json:"contextLinkId,omitempty"`
	// When action happens
	Occurrence *any `json:"occurrence,omitempty"`
	// Who asked for action
	Requester []Reference `json:"requester,omitempty"`
	// Pointer to specific item
	RequesterLinkId []string `json:"requesterLinkId,omitempty"`
	// Kind of service performer
	PerformerType []CodeableConcept `json:"performerType,omitempty"`
	// Competency of the performer
	PerformerRole *CodeableConcept `json:"performerRole,omitempty"`
	// Actor that wil execute (or not) the action
	Performer *Reference `json:"performer,omitempty"`
	// Pointer to specific item
	PerformerLinkId []string `json:"performerLinkId,omitempty"`
	// Why is action (not) needed?
	ReasonCode []CodeableConcept `json:"reasonCode,omitempty"`
	// Why is action (not) needed?
	ReasonReference []Reference `json:"reasonReference,omitempty"`
	// Why action is to be performed
	Reason []string `json:"reason,omitempty"`
	// Pointer to specific item
	ReasonLinkId []string `json:"reasonLinkId,omitempty"`
	// Comments about the action
	Note []Annotation `json:"note,omitempty"`
	// Action restriction numbers
	SecurityLabelNumber []uint `json:"securityLabelNumber,omitempty"`
}

// ContractTermGroup represents a FHIR BackboneElement for Contract.term.group.
type ContractTermGroup struct {
}

// ContractTerm represents a FHIR BackboneElement for Contract.term.
type ContractTerm struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Contract Term Number
	Identifier *Identifier `json:"identifier,omitempty"`
	// Contract Term Issue Date Time
	Issued *primitives.DateTime `json:"issued,omitempty"`
	// Contract Term Effective Time
	Applies *Period `json:"applies,omitempty"`
	// Term Concern
	Topic *any `json:"topic,omitempty"`
	// Contract Term Type or Form
	Type *CodeableConcept `json:"type,omitempty"`
	// Contract Term Type specific classification
	SubType *CodeableConcept `json:"subType,omitempty"`
	// Term Statement
	Text *string `json:"text,omitempty"`
	// Protection for the Term
	SecurityLabel []ContractTermSecurityLabel `json:"securityLabel,omitempty"`
	// Context of the Contract term
	Offer ContractTermOffer `json:"offer"`
	// Contract Term Asset List
	Asset []ContractTermAsset `json:"asset,omitempty"`
	// Entity being ascribed responsibility
	Action []ContractTermAction `json:"action,omitempty"`
	// Nested Contract Term Group
	Group []ContractTermGroup `json:"group,omitempty"`
}

// ContractSigner represents a FHIR BackboneElement for Contract.signer.
type ContractSigner struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Contract Signatory Role
	Type Coding `json:"type"`
	// Contract Signatory Party
	Party Reference `json:"party"`
	// Contract Documentation Signature
	Signature []Signature `json:"signature,omitempty"`
}

// ContractFriendly represents a FHIR BackboneElement for Contract.friendly.
type ContractFriendly struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Easily comprehended representation of this Contract
	Content any `json:"content"`
}

// ContractLegal represents a FHIR BackboneElement for Contract.legal.
type ContractLegal struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Contract Legal Text
	Content any `json:"content"`
}

// ContractRule represents a FHIR BackboneElement for Contract.rule.
type ContractRule struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Computable Contract Rules
	Content any `json:"content"`
}

// Contract represents a FHIR Contract.
type Contract struct {
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
	// Contract number
	Identifier []Identifier `json:"identifier,omitempty"`
	// Basal definition
	URL *string `json:"url,omitempty"`
	// Business edition
	Version *string `json:"version,omitempty"`
	// amended | appended | cancelled | disputed | entered-in-error | executable | executed | negotiable | offered | policy | rejected | renewed | revoked | resolved | terminated
	Status *string `json:"status,omitempty"`
	// Negotiation status
	LegalState *CodeableConcept `json:"legalState,omitempty"`
	// Source Contract Definition
	InstantiatesCanonical *Reference `json:"instantiatesCanonical,omitempty"`
	// External Contract Definition
	InstantiatesUri *string `json:"instantiatesUri,omitempty"`
	// Content derived from the basal information
	ContentDerivative *CodeableConcept `json:"contentDerivative,omitempty"`
	// When this Contract was issued
	Issued *primitives.DateTime `json:"issued,omitempty"`
	// Effective time
	Applies *Period `json:"applies,omitempty"`
	// Contract cessation cause
	ExpirationType *CodeableConcept `json:"expirationType,omitempty"`
	// Contract Target Entity
	Subject []Reference `json:"subject,omitempty"`
	// Authority under which this Contract has standing
	Authority []Reference `json:"authority,omitempty"`
	// A sphere of control governed by an authoritative jurisdiction, organization, or person
	Domain []Reference `json:"domain,omitempty"`
	// Specific Location
	Site []Reference `json:"site,omitempty"`
	// Computer friendly designation
	Name *string `json:"name,omitempty"`
	// Human Friendly name
	Title *string `json:"title,omitempty"`
	// Subordinate Friendly name
	Subtitle *string `json:"subtitle,omitempty"`
	// Acronym or short name
	Alias []string `json:"alias,omitempty"`
	// Source of Contract
	Author *Reference `json:"author,omitempty"`
	// Range of Legal Concerns
	Scope *CodeableConcept `json:"scope,omitempty"`
	// Focus of contract interest
	Topic *any `json:"topic,omitempty"`
	// Legal instrument category
	Type *CodeableConcept `json:"type,omitempty"`
	// Subtype within the context of type
	SubType []CodeableConcept `json:"subType,omitempty"`
	// Contract precursor content
	ContentDefinition *ContractContentDefinition `json:"contentDefinition,omitempty"`
	// Contract Term List
	Term []ContractTerm `json:"term,omitempty"`
	// Extra Information
	SupportingInfo []Reference `json:"supportingInfo,omitempty"`
	// Key event in Contract History
	RelevantHistory []Reference `json:"relevantHistory,omitempty"`
	// Contract Signatory
	Signer []ContractSigner `json:"signer,omitempty"`
	// Contract Friendly Language
	Friendly []ContractFriendly `json:"friendly,omitempty"`
	// Contract Legal Language
	Legal []ContractLegal `json:"legal,omitempty"`
	// Computable Contract Language
	Rule []ContractRule `json:"rule,omitempty"`
	// Binding Contract
	LegallyBinding *any `json:"legallyBinding,omitempty"`
}
