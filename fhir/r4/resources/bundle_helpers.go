package resources

import (
	"encoding/json"
	"fmt"
)

// NewSearchSetBundle creates a new Bundle with type "searchset".
func NewSearchSetBundle() *Bundle {
	return &Bundle{
		Type: "searchset",
	}
}

// NewTransactionBundle creates a new Bundle with type "transaction".
func NewTransactionBundle() *Bundle {
	return &Bundle{
		Type: "transaction",
	}
}

// NewBatchBundle creates a new Bundle with type "batch".
func NewBatchBundle() *Bundle {
	return &Bundle{
		Type: "batch",
	}
}

// NewCollectionBundle creates a new Bundle with type "collection".
func NewCollectionBundle() *Bundle {
	return &Bundle{
		Type: "collection",
	}
}

// AddEntry adds a resource entry to the Bundle.
// The resource is stored as a map with resourceType included.
func (b *Bundle) AddEntry(resource any, fullURL string) error {
	// Marshal the resource to JSON
	resourceJSON, err := json.Marshal(resource)
	if err != nil {
		return fmt.Errorf("failed to marshal resource: %w", err)
	}

	// Unmarshal into a map and add resourceType
	var resourceMap map[string]any
	if err := json.Unmarshal(resourceJSON, &resourceMap); err != nil {
		return fmt.Errorf("failed to unmarshal to map: %w", err)
	}

	// Add resourceType if not already present
	if _, ok := resourceMap["resourceType"]; !ok {
		// Infer resource type from the struct type
		resourceType, err := getResourceTypeName(resource)
		if err != nil {
			return err
		}
		resourceMap["resourceType"] = resourceType
	}

	// Create the bundle entry
	var anyResource any = resourceMap
	entry := BundleEntry{
		FullUrl:  stringPtr(fullURL),
		Resource: &anyResource,
	}

	b.Entry = append(b.Entry, entry)

	// Update total if it's set
	if b.Total != nil {
		*b.Total = *b.Total + 1
	} else {
		total := uint(1)
		b.Total = &total
	}

	return nil
}

// getResourceTypeName infers the FHIR resource type name from a Go struct.
func getResourceTypeName(resource any) (string, error) {
	switch resource.(type) {
	case *Patient, Patient:
		return "Patient", nil
	case *Observation, Observation:
		return "Observation", nil
	case *Practitioner, Practitioner:
		return "Practitioner", nil
	case *Organization, Organization:
		return "Organization", nil
	case *Encounter, Encounter:
		return "Encounter", nil
	case *Procedure, Procedure:
		return "Procedure", nil
	case *Condition, Condition:
		return "Condition", nil
	case *DiagnosticReport, DiagnosticReport:
		return "DiagnosticReport", nil
	case *MedicationRequest, MedicationRequest:
		return "MedicationRequest", nil
	case *AllergyIntolerance, AllergyIntolerance:
		return "AllergyIntolerance", nil
	// Add more as needed
	default:
		return "", fmt.Errorf("unknown resource type: %T", resource)
	}
}

// GetEntry retrieves a resource from a Bundle entry by index.
func (b *Bundle) GetEntry(index int) (any, error) {
	if index < 0 || index >= len(b.Entry) {
		return nil, fmt.Errorf("index %d out of range (bundle has %d entries)", index, len(b.Entry))
	}

	entry := b.Entry[index]
	if entry.Resource == nil {
		return nil, fmt.Errorf("entry %d has no resource", index)
	}

	// The resource is already stored as a pointer to any
	// We need to marshal it back to JSON and then unmarshal with the factory
	// to get the properly typed resource
	resourceJSON, err := json.Marshal(*entry.Resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resource: %w", err)
	}

	resource, err := UnmarshalResource(resourceJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal entry %d: %w", index, err)
	}

	return resource, nil
}

// GetAllEntries retrieves and unmarshals all resources from the Bundle.
func (b *Bundle) GetAllEntries() ([]any, error) {
	resources := make([]any, 0, len(b.Entry))

	for i := range b.Entry {
		resource, err := b.GetEntry(i)
		if err != nil {
			return nil, fmt.Errorf("failed to get entry %d: %w", i, err)
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// FindResourceByID searches the Bundle for a resource with the given ID.
// Returns the resource and its index, or an error if not found.
func (b *Bundle) FindResourceByID(id string) (resource any, index int, err error) {
	for i := range b.Entry {
		res, err := b.GetEntry(i)
		if err != nil {
			continue
		}

		// Try to extract ID using type assertion for common types
		var resourceID *string

		switch r := res.(type) {
		case *Patient:
			resourceID = r.ID
		case *Observation:
			resourceID = r.ID
		case *Practitioner:
			resourceID = r.ID
		case *Organization:
			resourceID = r.ID
		case *Encounter:
			resourceID = r.ID
		case *Procedure:
			resourceID = r.ID
		case *Condition:
			resourceID = r.ID
		case *DiagnosticReport:
			resourceID = r.ID
		case *MedicationRequest:
			resourceID = r.ID
		case *AllergyIntolerance:
			resourceID = r.ID
		// Add more types as needed
		default:
			continue
		}

		if resourceID != nil && *resourceID == id {
			return res, i, nil
		}
	}

	return nil, -1, fmt.Errorf("resource with ID %s not found in bundle", id)
}

// FilterByResourceType returns all resources of a specific type from the Bundle.
func (b *Bundle) FilterByResourceType(resourceType string) ([]any, error) {
	var resources []any

	for i := range b.Entry {
		entry := b.Entry[i]
		if entry.Resource == nil {
			continue
		}

		// Marshal to JSON to peek at the resourceType field
		resourceJSON, err := json.Marshal(*entry.Resource)
		if err != nil {
			continue
		}

		// Peek at the resourceType field
		var typeField resourceTypeField
		if err := json.Unmarshal(resourceJSON, &typeField); err != nil {
			continue
		}

		if typeField.ResourceType == resourceType {
			resource, err := b.GetEntry(i)
			if err != nil {
				return nil, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// stringPtr is a helper function to get a pointer to a string.
func stringPtr(s string) *string {
	return &s
}
