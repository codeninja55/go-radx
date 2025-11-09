package resources

// ResourceTypeBinary is the FHIR resource type name for Binary.
const ResourceTypeBinary = "Binary"

// Binary represents a FHIR Binary.
type Binary struct {
	// Logical id of this artifact
	ID *string `json:"id,omitempty"`
	// Metadata about the resource
	Meta *Meta `json:"meta,omitempty"`
	// A set of rules under which this content was created
	ImplicitRules *string `json:"implicitRules,omitempty"`
	// Language of the resource content
	Language *string `json:"language,omitempty"`
	// MimeType of the binary content
	ContentType string `json:"contentType"`
	// Identifies another resource to use as proxy when enforcing access control
	SecurityContext *Reference `json:"securityContext,omitempty"`
	// The actual content
	Data *string `json:"data,omitempty"`
}
