package r5

import "encoding/json"

// serviceRequestResourceType is the FHIR resourceType discriminator for ServiceRequest.
const serviceRequestResourceType = "ServiceRequest"

// ServiceRequest is the minimal R5 ServiceRequest carrying only the elements the
// ORM→ServiceRequest converter populates. Code is a CodeableReference in R5 (it
// was a CodeableConcept in R4). Status and Intent are required `code` primitives
// that the converter always sets, so they are plain strings here; the optional
// scalars are pointers so an absent element is distinguishable from an empty
// value and is omitted from the JSON.
type ServiceRequest struct {
	Identifier []Identifier       `json:"identifier,omitempty"`
	Status     string             `json:"status,omitempty"`
	Intent     string             `json:"intent,omitempty"`
	Code       *CodeableReference `json:"code,omitempty"`
	Subject    *Reference         `json:"subject,omitempty"`
	AuthoredOn *string            `json:"authoredOn,omitempty"`
	Requester  *Reference         `json:"requester,omitempty"`
}

// ResourceType returns the FHIR discriminator "ServiceRequest".
func (s *ServiceRequest) ResourceType() string { return serviceRequestResourceType }

// MarshalJSON emits the resource with "resourceType" as the first JSON key.
func (s *ServiceRequest) MarshalJSON() ([]byte, error) {
	type alias ServiceRequest
	return json.Marshal(struct {
		ResourceType string `json:"resourceType"`
		*alias
	}{
		ResourceType: serviceRequestResourceType,
		alias:        (*alias)(s),
	})
}
