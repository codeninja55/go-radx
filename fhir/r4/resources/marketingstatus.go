package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// MarketingStatus represents a FHIR MarketingStatus.
type MarketingStatus struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The country in which the marketing authorisation has been granted shall be specified It should be specified using the ISO 3166 ‑ 1 alpha-2 code elements
	Country CodeableConcept `json:"country"`
	// Where a Medicines Regulatory Agency has granted a marketing authorisation for which specific provisions within a jurisdiction apply, the jurisdiction can be specified using an appropriate controlled terminology The controlled term and the controlled term identifier shall be specified
	Jurisdiction *CodeableConcept `json:"jurisdiction,omitempty"`
	// This attribute provides information on the status of the marketing of the medicinal product See ISO/TS 20443 for more information and examples
	Status CodeableConcept `json:"status"`
	// The date when the Medicinal Product is placed on the market by the Marketing Authorisation Holder (or where applicable, the manufacturer/distributor) in a country and/or jurisdiction shall be provided A complete date consisting of day, month and year shall be specified using the ISO 8601 date format NOTE “Placed on the market” refers to the release of the Medicinal Product into the distribution chain
	DateRange Period `json:"dateRange"`
	// The date when the Medicinal Product is placed on the market by the Marketing Authorisation Holder (or where applicable, the manufacturer/distributor) in a country and/or jurisdiction shall be provided A complete date consisting of day, month and year shall be specified using the ISO 8601 date format NOTE “Placed on the market” refers to the release of the Medicinal Product into the distribution chain
	RestoreDate *primitives.DateTime `json:"restoreDate,omitempty"`
}
