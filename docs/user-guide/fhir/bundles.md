# FHIR Bundle Navigation Utilities

This document explains the Bundle navigation utilities provided by go-radx for working with FHIR Bundles.

## Overview

FHIR Bundles are collections of resources that can represent:
- **Search results** (type: `searchset`)
- **Transaction/batch requests** (type: `transaction` or `batch`)
- **History responses** (type: `history`)
- **Collections** (type: `collection`)
- **Documents** (type: `document`)
- **Messages** (type: `message`)

The `BundleHelper` provides utilities to navigate, search, and manipulate bundles efficiently.

## Quick Start

```go
import "github.com/harrison-ai/go-radx/fhir"

// Parse a bundle from JSON
var bundle fhir.Bundle
json.Unmarshal(data, &bundle)

// Create a helper
helper := fhir.NewBundleHelper(&bundle)

// Find all patients
patients, _ := helper.GetPatients()
fmt.Printf("Found %d patients\n", len(patients))

// Get a specific resource
patient, _ := helper.GetResourceByID("Patient", "example")
```

## Working with R5 Bundles (Type-Safe Approach)

In FHIR R5, Bundle entries use `json.RawMessage` for type-safe lazy deserialization. This approach provides compile-time type safety while maintaining flexibility for polymorphic resources.

### Type-Safe Unmarshaling

Use the generic `fhir.UnmarshalResource[T]()` function to unmarshal Bundle entries with compile-time type checking:

```go
import (
    "github.com/harrison-ai/go-radx/fhir"
    "github.com/harrison-ai/go-radx/fhir/r5/resources"
)

// Unmarshal a Patient resource from a bundle entry
patient, err := fhir.UnmarshalResource[resources.Patient](bundle.Entry[0].Resource)
if err != nil {
    log.Fatal(err)
}

// patient is now a resources.Patient with full type safety
fmt.Printf("Patient ID: %s\n", *patient.ID)
fmt.Printf("Active: %v\n", *patient.Active)
```

### Iterating Over Bundle Entries

Process all resources in a bundle with type checking:

```go
for i, entry := range bundle.Entry {
    // First, determine the resource type
    var typeMap map[string]interface{}
    if err := json.Unmarshal(entry.Resource, &typeMap); err != nil {
        continue
    }

    resourceType, ok := typeMap["resourceType"].(string)
    if !ok {
        continue
    }

    // Unmarshal based on type
    switch resourceType {
    case "Patient":
        patient, err := fhir.UnmarshalResource[resources.Patient](entry.Resource)
        if err != nil {
            log.Printf("Failed to unmarshal patient: %v", err)
            continue
        }
        processPatient(patient)

    case "Observation":
        obs, err := fhir.UnmarshalResource[resources.Observation](entry.Resource)
        if err != nil {
            log.Printf("Failed to unmarshal observation: %v", err)
            continue
        }
        processObservation(obs)

    default:
        log.Printf("Unknown resource type: %s", resourceType)
    }
}
```

### Working with Bundle Responses

Transaction and batch bundles include responses with `OperationOutcome`:

```go
for _, entry := range bundle.Entry {
    if entry.Response != nil && entry.Response.Outcome != nil {
        // Unmarshal the OperationOutcome
        outcome, err := fhir.UnmarshalResource[resources.OperationOutcome](entry.Response.Outcome)
        if err != nil {
            log.Printf("Failed to unmarshal outcome: %v", err)
            continue
        }

        // Check for errors
        for _, issue := range outcome.Issue {
            if issue.Severity == "error" {
                log.Printf("Error: %s", *issue.Diagnostics)
            }
        }
    }
}
```

### Bundle-Level Issues

R5 Bundles can include bundle-level issues:

```go
if bundle.Issues != nil {
    issues, err := fhir.UnmarshalResource[resources.OperationOutcome](bundle.Issues)
    if err != nil {
        log.Printf("Failed to unmarshal bundle issues: %v", err)
    } else {
        for _, issue := range issues.Issue {
            log.Printf("[%s] %s: %s", issue.Severity, issue.Code, *issue.Diagnostics)
        }
    }
}
```

### Creating Bundle Entries

To add resources to a bundle, marshal them to `json.RawMessage`:

```go
// Create a patient
patient := &resources.Patient{
    Active: boolPtr(true),
    Gender: stringPtr("female"),
}
patient.ID = stringPtr("patient-123")
patient.ResourceType = "Patient"

// Marshal to json.RawMessage
patientJSON, err := json.Marshal(patient)
if err != nil {
    log.Fatal(err)
}

// Create bundle with the entry
bundle := &resources.Bundle{
    Type: "transaction",
    Entry: []resources.BundleEntry{
        {
            FullUrl:  stringPtr("Patient/patient-123"),
            Resource: patientJSON,
            Request: &resources.BundleEntryRequest{
                Method: "POST",
                URL:    "Patient",
            },
        },
    },
}
bundle.ResourceType = "Bundle"
```

### Benefits of json.RawMessage in Bundles

1. **Type Safety**: Compile-time type checking when unmarshaling
2. **Lazy Deserialization**: Only unmarshal resources when needed
3. **Memory Efficiency**: Store as bytes until accessed
4. **Flexibility**: Bundle can contain any resource type
5. **Error Handling**: Clear error messages for deserialization failures

## BundleHelper Methods

### Finding Resources

#### FindResourcesByType

Find all resources of a specific type:

```go
helper := fhir.NewBundleHelper(&bundle)

// Find all patients
patients, err := helper.FindResourcesByType("Patient")
if err != nil {
    log.Fatal(err)
}

for _, patientJSON := range patients {
    var patient Patient
    json.Unmarshal(patientJSON, &patient)
    fmt.Printf("Patient: %s\n", *patient.ID)
}
```

#### GetResourceByID

Get a specific resource by type and ID:

```go
// Get Patient with ID "example"
resourceJSON, err := helper.GetResourceByID("Patient", "example")
if err != nil {
    log.Fatal(err)
}

if resourceJSON == nil {
    fmt.Println("Patient not found")
} else {
    var patient Patient
    json.Unmarshal(resourceJSON, &patient)
    // Use patient...
}
```

#### ResolveReference

Resolve a FHIR reference to the actual resource:

```go
// Reference can be:
// - Full URL: "http://example.org/Patient/123"
// - Relative: "Patient/123"
// - UUID: "urn:uuid:..."

resourceJSON, err := helper.ResolveReference("Patient/example")
if err != nil {
    log.Fatal(err)
}

var patient Patient
json.Unmarshal(resourceJSON, &patient)
```

**Use Case - Following References:**

```go
// Get an observation
obsJSON, _ := helper.GetResourceByID("Observation", "obs-1")
var obs Observation
json.Unmarshal(obsJSON, &obs)

// Resolve the subject reference
if obs.Subject != nil && obs.Subject.Reference != nil {
    subjectJSON, _ := helper.ResolveReference(*obs.Subject.Reference)
    var patient Patient
    json.Unmarshal(subjectJSON, &patient)
    fmt.Printf("Observation is for patient: %s\n", *patient.ID)
}
```

### Type-Specific Getters

Convenience methods for common resource types:

```go
helper := fhir.NewBundleHelper(&bundle)

// Get specific resource types
patients, _ := helper.GetPatients()
observations, _ := helper.GetObservations()
practitioners, _ := helper.GetPractitioners()
organizations, _ := helper.GetOrganizations()
medications, _ := helper.GetMedications()
encounters, _ := helper.GetEncounters()
conditions, _ := helper.GetConditions()
procedures, _ := helper.GetProcedures()
diagnosticReports, _ := helper.GetDiagnosticReports()
```

**All Available Getters:**
- `GetPatients()`
- `GetObservations()`
- `GetPractitioners()`
- `GetOrganizations()`
- `GetMedications()`
- `GetEncounters()`
- `GetConditions()`
- `GetProcedures()`
- `GetDiagnosticReports()`

### Adding Resources

#### AddEntry

Add a new resource to the bundle:

```go
helper := fhir.NewBundleHelper(&bundle)

// Create a new patient
patient := &Patient{
    Resource: Resource{
        ResourceType: "Patient",
        ID:           stringPtr("new-patient"),
    },
    Active: boolPtr(true),
}

// Add to bundle with fullUrl
fullURL := "http://example.org/Patient/new-patient"
err := helper.AddEntry(patient, &fullURL)
if err != nil {
    log.Fatal(err)
}

// The bundle's total count is automatically updated
fmt.Printf("Bundle now has %d entries\n", helper.Count())
```

### Information Methods

#### Count and CountByType

```go
helper := fhir.NewBundleHelper(&bundle)

// Total entries
totalCount := helper.Count()
fmt.Printf("Total entries: %d\n", totalCount)

// Count by type
patientCount, _ := helper.CountByType("Patient")
obsCount, _ := helper.CountByType("Observation")
fmt.Printf("Patients: %d, Observations: %d\n", patientCount, obsCount)
```

#### GetAllResources

Get all resources regardless of type:

```go
allResources := helper.GetAllResources()
fmt.Printf("Bundle contains %d resources\n", len(allResources))

for _, resourceJSON := range allResources {
    // Process each resource
    var resource map[string]interface{}
    json.Unmarshal(resourceJSON, &resource)
    fmt.Printf("Type: %s\n", resource["resourceType"])
}
```

#### GetResourceTypes

Get a list of all unique resource types in the bundle:

```go
types, err := helper.GetResourceTypes()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Bundle contains these types: %v\n", types)
// Output: [Patient Observation Practitioner]
```

### Pagination

#### Get Pagination Links

```go
helper := fhir.NewBundleHelper(&bundle)

// Get next page URL
if nextLink := helper.GetNextLink(); nextLink != nil {
    fmt.Printf("Next page: %s\n", *nextLink)
    // Fetch next page...
}

// Get previous page URL
if prevLink := helper.GetPreviousLink(); prevLink != nil {
    fmt.Printf("Previous page: %s\n", *prevLink)
}

// Get self link
if selfLink := helper.GetSelfLink(); selfLink != nil {
    fmt.Printf("Current page: %s\n", *selfLink)
}
```

## Common Use Cases

### 1. Processing Search Results

```go
func processSearchResults(bundleJSON []byte) error {
    var bundle fhir.Bundle
    if err := json.Unmarshal(bundleJSON, &bundle); err != nil {
        return err
    }
    
    helper := fhir.NewBundleHelper(&bundle)
    
    fmt.Printf("Search returned %d results\n", *bundle.Total)
    
    // Process each patient
    patients, err := helper.GetPatients()
    if err != nil {
        return err
    }
    
    for _, patientJSON := range patients {
        var patient fhir.Patient
        json.Unmarshal(patientJSON, &patient)
        processPatient(&patient)
    }
    
    return nil
}
```

### 2. Following References in a Bundle

```go
func getPatientObservations(helper *fhir.BundleHelper, patientID string) ([]fhir.Observation, error) {
    // Get all observations
    obsJSONList, err := helper.GetObservations()
    if err != nil {
        return nil, err
    }
    
    var patientObs []fhir.Observation
    
    for _, obsJSON := range obsJSONList {
        var obs fhir.Observation
        json.Unmarshal(obsJSON, &obs)
        
        // Check if observation is for this patient
        if obs.Subject != nil && obs.Subject.Reference != nil {
            ref := *obs.Subject.Reference
            if strings.Contains(ref, patientID) {
                patientObs = append(patientObs, obs)
            }
        }
    }
    
    return patientObs, nil
}
```

### 3. Resolving All References

```go
func resolveAllReferences(helper *fhir.BundleHelper) map[string]interface{} {
    resolved := make(map[string]interface{})
    
    // Get all resources
    allResources := helper.GetAllResources()
    
    for _, resourceJSON := range allResources {
        var resource map[string]interface{}
        json.Unmarshal(resourceJSON, &resource)
        
        // Store by ID for lookups
        if id, ok := resource["id"].(string); ok {
            resourceType := resource["resourceType"].(string)
            key := fmt.Sprintf("%s/%s", resourceType, id)
            resolved[key] = resource
        }
    }
    
    return resolved
}
```

### 4. Building a Transaction Bundle

```go
func createTransactionBundle() (*fhir.Bundle, error) {
    bundle := &fhir.Bundle{
        DomainResource: fhir.DomainResource{
            Resource: fhir.Resource{
                ResourceType: "Bundle",
            },
        },
        Type: "transaction",
    }
    
    helper := fhir.NewBundleHelper(bundle)
    
    // Add patient creation
    patient := &fhir.Patient{
        Resource: fhir.Resource{
            ResourceType: "Patient",
        },
        Active: boolPtr(true),
    }
    helper.AddEntry(patient, nil)
    
    // Add observation creation
    obs := &fhir.Observation{
        /* ... */
    }
    helper.AddEntry(obs, nil)
    
    return bundle, nil
}
```

### 5. Pagination Handling

```go
func fetchAllPages(baseURL string) ([]fhir.Patient, error) {
    var allPatients []fhir.Patient
    nextURL := baseURL + "/Patient"
    
    for nextURL != "" {
        // Fetch page
        resp, _ := http.Get(nextURL)
        var bundle fhir.Bundle
        json.NewDecoder(resp.Body).Decode(&bundle)
        resp.Body.Close()
        
        helper := fhir.NewBundleHelper(&bundle)
        
        // Extract patients from this page
        patientJSONs, _ := helper.GetPatients()
        for _, pJSON := range patientJSONs {
            var patient fhir.Patient
            json.Unmarshal(pJSON, &patient)
            allPatients = append(allPatients, patient)
        }
        
        // Get next page URL
        if nextLink := helper.GetNextLink(); nextLink != nil {
            nextURL = *nextLink
        } else {
            nextURL = "" // No more pages
        }
    }
    
    return allPatients, nil
}
```

### 6. Filtering and Extracting

```go
func getActivePatients(bundle *fhir.Bundle) ([]*fhir.Patient, error) {
    helper := fhir.NewBundleHelper(bundle)
    
    patientJSONs, err := helper.GetPatients()
    if err != nil {
        return nil, err
    }
    
    var activePatients []*fhir.Patient
    
    for _, pJSON := range patientJSONs {
        var patient fhir.Patient
        if err := json.Unmarshal(pJSON, &patient); err != nil {
            continue
        }
        
        if patient.Active != nil && *patient.Active {
            activePatients = append(activePatients, &patient)
        }
    }
    
    return activePatients, nil
}
```

### 7. Bundle Summary

```go
func printBundleSummary(bundle *fhir.Bundle) {
    helper := fhir.NewBundleHelper(bundle)
    
    fmt.Printf("Bundle Type: %s\n", bundle.Type)
    fmt.Printf("Total Entries: %d\n", helper.Count())
    
    types, _ := helper.GetResourceTypes()
    fmt.Println("Resource Types:")
    
    for _, resourceType := range types {
        count, _ := helper.CountByType(resourceType)
        fmt.Printf("  - %s: %d\n", resourceType, count)
    }
    
    // Pagination info
    if nextLink := helper.GetNextLink(); nextLink != nil {
        fmt.Printf("Has next page: %s\n", *nextLink)
    }
}
```

## Working with RawMessage

Resources in bundles are stored as `json.RawMessage` to avoid type assertions. To use them:

```go
// Get resource as RawMessage
patientJSON, _ := helper.GetResourceByID("Patient", "example")

// Unmarshal to typed struct
var patient fhir.Patient
if err := json.Unmarshal(patientJSON, &patient); err != nil {
    log.Fatal(err)
}

// Now use typed patient
fmt.Printf("Patient name: %s\n", *patient.Name[0].Family)
```

## Performance Considerations

### Caching Lookups

If you need to look up many resources, consider building an index:

```go
type BundleIndex struct {
    byID   map[string]json.RawMessage  // "ResourceType/id" -> resource
    byType map[string][]json.RawMessage // "ResourceType" -> []resource
}

func buildIndex(helper *fhir.BundleHelper) *BundleIndex {
    index := &BundleIndex{
        byID:   make(map[string]json.RawMessage),
        byType: make(map[string][]json.RawMessage),
    }
    
    for _, resource := range helper.GetAllResources() {
        var r map[string]interface{}
        json.Unmarshal(resource, &r)
        
        resourceType := r["resourceType"].(string)
        id := r["id"].(string)
        
        key := fmt.Sprintf("%s/%s", resourceType, id)
        index.byID[key] = resource
        index.byType[resourceType] = append(index.byType[resourceType], resource)
    }
    
    return index
}
```

### Parallel Processing

For large bundles, process resources in parallel:

```go
func processLargeBundle(bundle *fhir.Bundle) error {
    helper := fhir.NewBundleHelper(bundle)
    patients, _ := helper.GetPatients()
    
    // Process in parallel
    results := make(chan error, len(patients))
    
    for _, patientJSON := range patients {
        go func(pJSON json.RawMessage) {
            var patient fhir.Patient
            json.Unmarshal(pJSON, &patient)
            results <- processPatient(&patient)
        }(patientJSON)
    }
    
    // Collect results
    for range patients {
        if err := <-results; err != nil {
            return err
        }
    }
    
    return nil
}
```

## Best Practices

### 1. Check for nil Resources

```go
resourceJSON, err := helper.GetResourceByID("Patient", "example")
if err != nil {
    return err
}

if resourceJSON == nil {
    return fmt.Errorf("patient not found")
}

// Safe to unmarshal
```

### 2. Handle Empty Bundles

```go
helper := fhir.NewBundleHelper(&bundle)

if helper.Count() == 0 {
    fmt.Println("Bundle is empty")
    return
}

// Process resources...
```

### 3. Validate Bundle Type

```go
if bundle.Type != "searchset" {
    return fmt.Errorf("expected searchset bundle, got %s", bundle.Type)
}
```

### 4. Use Type-Specific Getters

Instead of:
```go
resources, _ := helper.FindResourcesByType("Patient")
```

Use:
```go
patients, _ := helper.GetPatients()
```

More readable and type-safe.

### 5. Check Total Count

```go
if bundle.Total != nil {
    fmt.Printf("Total matching resources: %d\n", *bundle.Total)
    fmt.Printf("Returned in this page: %d\n", helper.Count())
}
```

## Error Handling

```go
helper := fhir.NewBundleHelper(&bundle)

// Most methods return errors for malformed resources
resources, err := helper.FindResourcesByType("Patient")
if err != nil {
    log.Printf("Error finding patients: %v", err)
    // Handle error...
}

// Check for not found
resource, err := helper.GetResourceByID("Patient", "example")
if err != nil {
    return err
}
if resource == nil {
    return fmt.Errorf("patient not found")
}
```

## Summary

- **BundleHelper** provides utilities for FHIR bundle navigation
- **Type-safe access** to resources with automatic unmarshaling
- **Reference resolution** for following links between resources
- **Pagination support** with link helpers
- **Counting and filtering** by resource type
- **Easy resource addition** with automatic total updates
- **Performance optimized** for large bundles

Use `NewBundleHelper()` to start working with any FHIR Bundle!
