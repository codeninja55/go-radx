package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeDeviceMetric is the FHIR resource type name for DeviceMetric.
const ResourceTypeDeviceMetric = "DeviceMetric"

// DeviceMetricCalibration represents a FHIR BackboneElement for DeviceMetric.calibration.
type DeviceMetricCalibration struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// unspecified | offset | gain | two-point
	Type *string `json:"type,omitempty"`
	// not-calibrated | calibration-required | calibrated | unspecified
	State *string `json:"state,omitempty"`
	// Describes the time last calibration has been performed
	Time *primitives.Instant `json:"time,omitempty"`
}

// DeviceMetric represents a FHIR DeviceMetric.
type DeviceMetric struct {
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
	// Instance identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// Identity of metric, for example Heart Rate or PEEP Setting
	Type CodeableConcept `json:"type"`
	// Unit of Measure for the Metric
	Unit *CodeableConcept `json:"unit,omitempty"`
	// Describes the link to the source Device
	Source *Reference `json:"source,omitempty"`
	// Describes the link to the parent Device
	Parent *Reference `json:"parent,omitempty"`
	// on | off | standby | entered-in-error
	OperationalStatus *string `json:"operationalStatus,omitempty"`
	// black | red | green | yellow | blue | magenta | cyan | white
	Color *string `json:"color,omitempty"`
	// measurement | setting | calculation | unspecified
	Category string `json:"category"`
	// Describes the measurement repetition time
	MeasurementPeriod *Timing `json:"measurementPeriod,omitempty"`
	// Describes the calibrations that have been performed or that are required to be performed
	Calibration []DeviceMetricCalibration `json:"calibration,omitempty"`
}
