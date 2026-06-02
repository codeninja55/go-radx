package r5

import "encoding/json"

// diagnosticReportResourceType is the FHIR resourceType discriminator for DiagnosticReport.
const diagnosticReportResourceType = "DiagnosticReport"

// DiagnosticReport is the minimal R5 DiagnosticReport carrying only the elements
// the SR/ORU→DiagnosticReport converters populate in M2's narrative-only slice.
// Code is a CodeableConcept and Category a list of CodeableConcept (both R5 and
// R4 agree here). EffectiveDateTime is the dateTime branch of effective[x] held
// as a validated string; Conclusion is FHIR markdown, also a string. Result is a
// list of Reference to the Observations produced alongside the report. Fields are
// ordered per the R5 StructureDefinition so the JSON keys are stable.
type DiagnosticReport struct {
	Identifier        []Identifier      `json:"identifier,omitempty"`
	Status            string            `json:"status,omitempty"`
	Category          []CodeableConcept `json:"category,omitempty"`
	Code              *CodeableConcept  `json:"code,omitempty"`
	Subject           *Reference        `json:"subject,omitempty"`
	EffectiveDateTime *string           `json:"effectiveDateTime,omitempty"`
	Conclusion        *string           `json:"conclusion,omitempty"`
	Result            []Reference       `json:"result,omitempty"`
}

// ResourceType returns the FHIR discriminator "DiagnosticReport".
func (d *DiagnosticReport) ResourceType() string { return diagnosticReportResourceType }

// MarshalJSON emits the resource with "resourceType" as the first JSON key.
func (d *DiagnosticReport) MarshalJSON() ([]byte, error) {
	type alias DiagnosticReport
	return json.Marshal(struct {
		ResourceType string `json:"resourceType"`
		*alias
	}{
		ResourceType: diagnosticReportResourceType,
		alias:        (*alias)(d),
	})
}
