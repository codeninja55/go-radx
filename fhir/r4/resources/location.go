package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeLocation is the FHIR resource type name for Location.
const ResourceTypeLocation = "Location"

// LocationPosition represents a FHIR BackboneElement for Location.position.
type LocationPosition struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Longitude with WGS84 datum
	Longitude float64 `json:"longitude"`
	// Latitude with WGS84 datum
	Latitude float64 `json:"latitude"`
	// Altitude with WGS84 datum
	Altitude *float64 `json:"altitude,omitempty"`
}

// LocationHoursOfOperation represents a FHIR BackboneElement for Location.hoursOfOperation.
type LocationHoursOfOperation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// mon | tue | wed | thu | fri | sat | sun
	DaysOfWeek []string `json:"daysOfWeek,omitempty"`
	// The Location is open all day
	AllDay *bool `json:"allDay,omitempty"`
	// Time that the Location opens
	OpeningTime *primitives.Time `json:"openingTime,omitempty"`
	// Time that the Location closes
	ClosingTime *primitives.Time `json:"closingTime,omitempty"`
}

// Location represents a FHIR Location.
type Location struct {
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
	// Unique code or number identifying the location to its users
	Identifier []Identifier `json:"identifier,omitempty"`
	// active | suspended | inactive
	Status *string `json:"status,omitempty"`
	// The operational status of the location (typically only for a bed/room)
	OperationalStatus *Coding `json:"operationalStatus,omitempty"`
	// Name of the location as used by humans
	Name *string `json:"name,omitempty"`
	// A list of alternate names that the location is known as, or was known as, in the past
	Alias []string `json:"alias,omitempty"`
	// Additional details about the location that could be displayed as further information to identify the location beyond its name
	Description *string `json:"description,omitempty"`
	// instance | kind
	Mode *string `json:"mode,omitempty"`
	// Type of function performed
	Type []CodeableConcept `json:"type,omitempty"`
	// Contact details of the location
	Telecom []ContactPoint `json:"telecom,omitempty"`
	// Physical location
	Address *Address `json:"address,omitempty"`
	// Physical form of the location
	PhysicalType *CodeableConcept `json:"physicalType,omitempty"`
	// The absolute geographic location
	Position *LocationPosition `json:"position,omitempty"`
	// Organization responsible for provisioning and upkeep
	ManagingOrganization *Reference `json:"managingOrganization,omitempty"`
	// Another Location this one is physically a part of
	PartOf *Reference `json:"partOf,omitempty"`
	// What days/times during a week is this location usually open
	HoursOfOperation []LocationHoursOfOperation `json:"hoursOfOperation,omitempty"`
	// Description of availability exceptions
	AvailabilityExceptions *string `json:"availabilityExceptions,omitempty"`
	// Technical endpoints providing access to services operated for the location
	Endpoint []Reference `json:"endpoint,omitempty"`
}
