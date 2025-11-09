package fhir

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Bundle represents a FHIR Bundle resource.
type Bundle struct {
	DomainResource
	Type      string        `json:"type" fhir:"cardinality=1..1,required"`
	Total     *int          `json:"total,omitempty" fhir:"cardinality=0..1,summary"`
	Link      []BundleLink  `json:"link,omitempty" fhir:"cardinality=0..*,summary"`
	Entry     []BundleEntry `json:"entry,omitempty" fhir:"cardinality=0..*,summary"`
	Signature *string       `json:"signature,omitempty" fhir:"cardinality=0..1"`
}

// BundleLink represents a link in a bundle.
type BundleLink struct {
	Relation string `json:"relation" fhir:"cardinality=1..1,required"`
	URL      string `json:"url" fhir:"cardinality=1..1,required"`
}

// BundleEntry represents an entry in a bundle.
type BundleEntry struct {
	Link     []BundleLink         `json:"link,omitempty" fhir:"cardinality=0..*"`
	Resource json.RawMessage      `json:"resource,omitempty" fhir:"cardinality=0..1,summary"`
	FullURL  *string              `json:"fullUrl,omitempty" fhir:"cardinality=0..1,summary"`
	Search   *BundleEntrySearch   `json:"search,omitempty" fhir:"cardinality=0..1,summary"`
	Request  *BundleEntryRequest  `json:"request,omitempty" fhir:"cardinality=0..1,summary"`
	Response *BundleEntryResponse `json:"response,omitempty" fhir:"cardinality=0..1,summary"`
}

// BundleEntrySearch represents search metadata in a bundle entry.
type BundleEntrySearch struct {
	Mode  *string  `json:"mode,omitempty" fhir:"cardinality=0..1,summary"`
	Score *float64 `json:"score,omitempty" fhir:"cardinality=0..1,summary"`
}

// BundleEntryRequest represents request information in a bundle entry.
type BundleEntryRequest struct {
	IfNoneMatch     *string `json:"ifNoneMatch,omitempty" fhir:"cardinality=0..1,summary"`
	IfModifiedSince *string `json:"ifModifiedSince,omitempty" fhir:"cardinality=0..1,summary"`
	IfMatch         *string `json:"ifMatch,omitempty" fhir:"cardinality=0..1,summary"`
	IfNoneExist     *string `json:"ifNoneExist,omitempty" fhir:"cardinality=0..1,summary"`
	Method          string  `json:"method" fhir:"cardinality=1..1,required,summary"`
	URL             string  `json:"url" fhir:"cardinality=1..1,required,summary"`
}

// BundleEntryResponse represents response information in a bundle entry.
type BundleEntryResponse struct {
	Status       string          `json:"status" fhir:"cardinality=1..1,required,summary"`
	Location     *string         `json:"location,omitempty" fhir:"cardinality=0..1,summary"`
	Etag         *string         `json:"etag,omitempty" fhir:"cardinality=0..1,summary"`
	LastModified *string         `json:"lastModified,omitempty" fhir:"cardinality=0..1,summary"`
	Outcome      json.RawMessage `json:"outcome,omitempty" fhir:"cardinality=0..1,summary"`
}

// BundleHelper provides utilities for working with FHIR Bundles.
type BundleHelper struct {
	bundle *Bundle
}

// NewBundleHelper creates a new BundleHelper for the given bundle.
func NewBundleHelper(bundle *Bundle) *BundleHelper {
	return &BundleHelper{bundle: bundle}
}

// FindResourcesByType returns all resources of the specified type from the bundle.
// resourceType should be the FHIR resource type name (e.g., "Patient", "Observation").
func (h *BundleHelper) FindResourcesByType(resourceType string) ([]json.RawMessage, error) {
	var resources []json.RawMessage

	for _, entry := range h.bundle.Entry {
		if entry.Resource == nil {
			continue
		}

		// Parse resource to check type
		var resource map[string]interface{}
		if err := json.Unmarshal(entry.Resource, &resource); err != nil {
			return nil, fmt.Errorf("failed to parse resource: %w", err)
		}

		if resType, ok := resource["resourceType"].(string); ok && resType == resourceType {
			resources = append(resources, entry.Resource)
		}
	}

	return resources, nil
}

// GetResourceByID finds a resource by its ID and type.
// Returns the resource as RawMessage, or nil if not found.
func (h *BundleHelper) GetResourceByID(resourceType, id string) (json.RawMessage, error) {
	for _, entry := range h.bundle.Entry {
		if entry.Resource == nil {
			continue
		}

		var resource map[string]interface{}
		if err := json.Unmarshal(entry.Resource, &resource); err != nil {
			return nil, fmt.Errorf("failed to parse resource: %w", err)
		}

		resType, typeOk := resource["resourceType"].(string)
		resID, idOk := resource["id"].(string)

		if typeOk && idOk && resType == resourceType && resID == id {
			return entry.Resource, nil
		}
	}

	return nil, nil // Not found
}

// ResolveReference resolves a FHIR reference to the actual resource in the bundle.
// reference should be in the format "ResourceType/id" or a fullUrl.
// Returns the resource as RawMessage, or nil if not found.
func (h *BundleHelper) ResolveReference(reference string) (json.RawMessage, error) {
	if reference == "" {
		return nil, fmt.Errorf("empty reference")
	}

	// Try to resolve by fullUrl first
	for _, entry := range h.bundle.Entry {
		if entry.FullURL != nil && *entry.FullURL == reference {
			return entry.Resource, nil
		}
	}

	// Try to resolve by relative reference (ResourceType/id)
	parts := strings.Split(reference, "/")
	if len(parts) >= 2 {
		resourceType := parts[len(parts)-2]
		id := parts[len(parts)-1]
		return h.GetResourceByID(resourceType, id)
	}

	return nil, fmt.Errorf("reference not found: %s", reference)
}

// AddEntry adds a new entry to the bundle with the given resource.
// The resource will be marshaled to JSON.
func (h *BundleHelper) AddEntry(resource interface{}, fullURL *string) error {
	data, err := json.Marshal(resource)
	if err != nil {
		return fmt.Errorf("failed to marshal resource: %w", err)
	}

	entry := BundleEntry{
		FullURL:  fullURL,
		Resource: data,
	}

	h.bundle.Entry = append(h.bundle.Entry, entry)

	// Update total if set
	if h.bundle.Total != nil {
		*h.bundle.Total++
	} else {
		total := len(h.bundle.Entry)
		h.bundle.Total = &total
	}

	return nil
}

// GetPatients returns all Patient resources from the bundle.
func (h *BundleHelper) GetPatients() ([]json.RawMessage, error) {
	return h.FindResourcesByType("Patient")
}

// GetObservations returns all Observation resources from the bundle.
func (h *BundleHelper) GetObservations() ([]json.RawMessage, error) {
	return h.FindResourcesByType("Observation")
}

// GetPractitioners returns all Practitioner resources from the bundle.
func (h *BundleHelper) GetPractitioners() ([]json.RawMessage, error) {
	return h.FindResourcesByType("Practitioner")
}

// GetOrganizations returns all Organization resources from the bundle.
func (h *BundleHelper) GetOrganizations() ([]json.RawMessage, error) {
	return h.FindResourcesByType("Organization")
}

// GetMedications returns all Medication resources from the bundle.
func (h *BundleHelper) GetMedications() ([]json.RawMessage, error) {
	return h.FindResourcesByType("Medication")
}

// GetEncounters returns all Encounter resources from the bundle.
func (h *BundleHelper) GetEncounters() ([]json.RawMessage, error) {
	return h.FindResourcesByType("Encounter")
}

// GetConditions returns all Condition resources from the bundle.
func (h *BundleHelper) GetConditions() ([]json.RawMessage, error) {
	return h.FindResourcesByType("Condition")
}

// GetProcedures returns all Procedure resources from the bundle.
func (h *BundleHelper) GetProcedures() ([]json.RawMessage, error) {
	return h.FindResourcesByType("Procedure")
}

// GetDiagnosticReports returns all DiagnosticReport resources from the bundle.
func (h *BundleHelper) GetDiagnosticReports() ([]json.RawMessage, error) {
	return h.FindResourcesByType("DiagnosticReport")
}

// GetAllResources returns all resources from the bundle, regardless of type.
func (h *BundleHelper) GetAllResources() []json.RawMessage {
	var resources []json.RawMessage
	for _, entry := range h.bundle.Entry {
		if entry.Resource != nil {
			resources = append(resources, entry.Resource)
		}
	}
	return resources
}

// GetResourceTypes returns a list of unique resource types in the bundle.
func (h *BundleHelper) GetResourceTypes() ([]string, error) {
	typeMap := make(map[string]bool)

	for _, entry := range h.bundle.Entry {
		if entry.Resource == nil {
			continue
		}

		var resource map[string]interface{}
		if err := json.Unmarshal(entry.Resource, &resource); err != nil {
			return nil, fmt.Errorf("failed to parse resource: %w", err)
		}

		if resType, ok := resource["resourceType"].(string); ok {
			typeMap[resType] = true
		}
	}

	types := make([]string, 0, len(typeMap))
	for t := range typeMap {
		types = append(types, t)
	}

	return types, nil
}

// Count returns the number of entries in the bundle.
func (h *BundleHelper) Count() int {
	return len(h.bundle.Entry)
}

// CountByType returns the number of resources of the specified type.
func (h *BundleHelper) CountByType(resourceType string) (int, error) {
	resources, err := h.FindResourcesByType(resourceType)
	if err != nil {
		return 0, err
	}
	return len(resources), nil
}

// GetNextLink returns the URL for the next page of results, if available.
func (h *BundleHelper) GetNextLink() *string {
	for _, link := range h.bundle.Link {
		if link.Relation == "next" {
			return &link.URL
		}
	}
	return nil
}

// GetPreviousLink returns the URL for the previous page of results, if available.
func (h *BundleHelper) GetPreviousLink() *string {
	for _, link := range h.bundle.Link {
		if link.Relation == "previous" || link.Relation == "prev" {
			return &link.URL
		}
	}
	return nil
}

// GetSelfLink returns the self link URL, if available.
func (h *BundleHelper) GetSelfLink() *string {
	for _, link := range h.bundle.Link {
		if link.Relation == "self" {
			return &link.URL
		}
	}
	return nil
}
