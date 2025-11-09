package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeChargeItem is the FHIR resource type name for ChargeItem.
const ResourceTypeChargeItem = "ChargeItem"

// ChargeItemPerformer represents a FHIR BackboneElement for ChargeItem.performer.
type ChargeItemPerformer struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// What type of performance was done
	Function *CodeableConcept `json:"function,omitempty"`
	// Individual who was performing
	Actor Reference `json:"actor"`
}

// ChargeItem represents a FHIR ChargeItem.
type ChargeItem struct {
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
	// Business Identifier for item
	Identifier []Identifier `json:"identifier,omitempty"`
	// Defining information about the code of this charge item
	DefinitionUri []string `json:"definitionUri,omitempty"`
	// Resource defining the code of this ChargeItem
	DefinitionCanonical []string `json:"definitionCanonical,omitempty"`
	// planned | billable | not-billable | aborted | billed | entered-in-error | unknown
	Status string `json:"status"`
	// Part of referenced ChargeItem
	PartOf []Reference `json:"partOf,omitempty"`
	// A code that identifies the charge, like a billing code
	Code CodeableConcept `json:"code"`
	// Individual service was done for/to
	Subject Reference `json:"subject"`
	// Encounter associated with this ChargeItem
	Encounter *Reference `json:"encounter,omitempty"`
	// When the charged service was applied
	Occurrence *any `json:"occurrence,omitempty"`
	// Who performed charged service
	Performer []ChargeItemPerformer `json:"performer,omitempty"`
	// Organization providing the charged service
	PerformingOrganization *Reference `json:"performingOrganization,omitempty"`
	// Organization requesting the charged service
	RequestingOrganization *Reference `json:"requestingOrganization,omitempty"`
	// Organization that has ownership of the (potential, future) revenue
	CostCenter *Reference `json:"costCenter,omitempty"`
	// Quantity of which the charge item has been serviced
	Quantity *Quantity `json:"quantity,omitempty"`
	// Anatomical location, if relevant
	Bodysite []CodeableConcept `json:"bodysite,omitempty"`
	// Unit price overriding the associated rules
	UnitPriceComponent *MonetaryComponent `json:"unitPriceComponent,omitempty"`
	// Total price overriding the associated rules
	TotalPriceComponent *MonetaryComponent `json:"totalPriceComponent,omitempty"`
	// Reason for overriding the list price/factor
	OverrideReason *CodeableConcept `json:"overrideReason,omitempty"`
	// Individual who was entering
	Enterer *Reference `json:"enterer,omitempty"`
	// Date the charge item was entered
	EnteredDate *primitives.DateTime `json:"enteredDate,omitempty"`
	// Why was the charged  service rendered?
	Reason []CodeableConcept `json:"reason,omitempty"`
	// Which rendered service is being charged?
	Service []CodeableReference `json:"service,omitempty"`
	// Product charged
	Product []CodeableReference `json:"product,omitempty"`
	// Account to place this charge
	Account []Reference `json:"account,omitempty"`
	// Comments made about the ChargeItem
	Note []Annotation `json:"note,omitempty"`
	// Further information supporting this charge
	SupportingInformation []Reference `json:"supportingInformation,omitempty"`
}
