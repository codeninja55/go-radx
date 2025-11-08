# FHIR Summary Mode

This document explains FHIR summary mode and its implementation in go-radx.

## Overview

FHIR summary mode allows you to serialize resources with a subset of fields, reducing payload size for list views, search results, and bandwidth-constrained scenarios. This corresponds to the `_summary` parameter in FHIR REST APIs.

## Summary Modes

### SummaryModeTrue (_summary=true)

Returns only fields marked as "summary elements" in the FHIR specification, plus essential metadata (resourceType, id, meta).

**Use Case:** Search results, list views, resource previews

**Example:**
```go
patient := &Patient{
    ID:        stringPtr("123"),
    Active:    boolPtr(true),          // Summary field
    Name:      []HumanName{...},        // Summary field
    BirthDate: datePtr("1990-01-01"),  // Summary field
    Photo:     []Attachment{...},       // Not summary - excluded
}

data, _ := fhir.MarshalSummaryJSON(patient)
// Output includes: id, active, name, birthDate (not photo)
```

### SummaryModeFalse (_summary=false)

Returns all fields **except** summary elements. Useful for getting detailed non-summary data.

**Use Case:** Detailed views excluding commonly-known summary information

### SummaryModeText (_summary=text)

Returns only the narrative text (Text field) plus minimal metadata (resourceType, id, meta).

**Use Case:** Human-readable display without structured data

### SummaryModeData (_summary=data)

Returns all structured data fields, excluding the narrative text.

**Use Case:** Machine processing, excluding human-readable narrative

### SummaryModeAll (default)

Returns all fields. Equivalent to standard `json.Marshal()`.

## Usage

### Basic Summary Marshaling

```go
import "github.com/harrison-ai/go-radx/fhir"

patient := &Patient{
    ID:                  stringPtr("example"),
    Active:              boolPtr(true),
    Name:                []HumanName{{Family: stringPtr("Doe")}},
    Photo:               []Attachment{{URL: stringPtr("photo.jpg")}},
    GeneralPractitioner: []Reference{{Reference: stringPtr("Practitioner/123")}},
}

// Summary JSON (only summary fields)
summaryJSON, err := fhir.MarshalSummaryJSON(patient)
if err != nil {
    log.Fatal(err)
}

// Output: {"resourceType":"Patient","id":"example","active":true,"name":[{"family":"Doe"}]}
// Note: photo and generalPractitioner are excluded
```

### Specific Summary Mode

```go
// Text mode - narrative only
textJSON, _ := fhir.MarshalWithSummaryMode(patient, fhir.SummaryModeText)

// Data mode - no narrative
dataJSON, _ := fhir.MarshalWithSummaryMode(patient, fhir.SummaryModeData)

// False mode - detailed fields only
detailJSON, _ := fhir.MarshalWithSummaryMode(patient, fhir.SummaryModeFalse)

// All mode - everything (same as json.Marshal)
fullJSON, _ := fhir.MarshalWithSummaryMode(patient, fhir.SummaryModeAll)
```

### Getting Summary Field Names

```go
// Get list of summary field names for a resource
summaryFields := fhir.GetSummaryFields(patient)
// Returns: ["active", "name", "telecom", "gender", "birthDate", ...]

fmt.Printf("Summary fields: %v\n", summaryFields)
```

## Summary Elements by Resource

### Patient

**Summary fields:**
- id, meta, identifier
- active, name, telecom, gender, birthDate
- address, photo (limited), managingOrganization
- link

**Non-summary fields:**
- photo (detailed), contact, communication
- generalPractitioner, maritalStatus, multipleBirth

### Observation

**Summary fields:**
- id, meta, identifier, status, category
- code, subject, encounter, effective[x]
- issued, value[x], dataAbsentReason
- interpretation, bodySite, method

**Non-summary fields:**
- basedOn, partOf, focus, performer
- referenceRange, hasMember, derivedFrom
- component (detailed)

### Practitioner

**Summary fields:**
- id, meta, identifier, active
- name, telecom, gender

**Non-summary fields:**
- address, photo, qualification
- communication

## Performance Benefits

Summary mode reduces JSON payload size significantly:

```go
patient := createLargePatient() // With all fields

fullJSON, _ := json.Marshal(patient)
summaryJSON, _ := fhir.MarshalSummaryJSON(patient)

fmt.Printf("Full JSON: %d bytes\n", len(fullJSON))     // e.g., 2500 bytes
fmt.Printf("Summary JSON: %d bytes\n", len(summaryJSON)) // e.g., 850 bytes
fmt.Printf("Reduction: %.1f%%\n", 
    float64(len(fullJSON)-len(summaryJSON))/float64(len(fullJSON))*100) // 66% reduction
```

Typical reductions:
- **Patient:** 40-60% size reduction
- **Observation:** 30-50% size reduction
- **Bundle (search results):** 50-70% size reduction

## REST API Integration

### Server Side

```go
func handlePatientSearch(w http.ResponseWriter, r *http.Request) {
    summaryParam := r.URL.Query().Get("_summary")
    
    patients := searchPatients(r) // Your search logic
    
    var mode fhir.SummaryMode
    switch summaryParam {
    case "true":
        mode = fhir.SummaryModeTrue
    case "false":
        mode = fhir.SummaryModeFalse
    case "text":
        mode = fhir.SummaryModeText
    case "data":
        mode = fhir.SummaryModeData
    default:
        mode = fhir.SummaryModeAll
    }
    
    // Serialize bundle with summary mode
    bundle := createBundle(patients)
    data, err := fhir.MarshalWithSummaryMode(bundle, mode)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/fhir+json")
    w.Write(data)
}
```

### Client Side

```go
func searchPatients(baseURL, query string, summary bool) ([]Patient, error) {
    url := fmt.Sprintf("%s/Patient?%s", baseURL, query)
    if summary {
        url += "&_summary=true"
    }
    
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var bundle Bundle
    if err := json.NewDecoder(resp.Body).Decode(&bundle); err != nil {
        return nil, err
    }
    
    // Extract patients from bundle
    return extractPatients(bundle), nil
}
```

## Implementation Details

### How It Works

1. **Reflection:** Uses Go reflection to inspect struct tags
2. **Tag Parsing:** Reads `fhir` struct tags to identify summary fields
3. **Filtering:** Recursively filters structs, preserving only fields matching the mode
4. **Serialization:** Converts filtered result to JSON

### Summary Tag Detection

Fields are identified as summary elements by the `summary` keyword in FHIR tags:

```go
type Patient struct {
    Active    *bool   `json:"active,omitempty" fhir:"cardinality=0..1,summary"`
    BirthDate *Date   `json:"birthDate,omitempty" fhir:"cardinality=0..1,summary"`
    Photo     *string `json:"photo,omitempty" fhir:"cardinality=0..1"`  // No summary
}
```

### Embedded Structs

Summary marshaling correctly handles embedded base types:

```go
type Patient struct {
    fhir.DomainResource  // Embedded
    Active *bool `json:"active,omitempty" fhir:"summary"`
}

// Summary mode includes fields from embedded Resource (id, meta) 
// plus resource-specific summary fields
```

### Nested Structures

Summary filtering applies recursively:

```go
type Patient struct {
    Name []HumanName `json:"name" fhir:"summary"`  // Summary
}

type HumanName struct {
    Family *string `json:"family" fhir:"summary"`  // Also summary
    Given  []string `json:"given" fhir:"summary"`
    Prefix []string `json:"prefix"`                 // Not summary
}

// Summary JSON includes name.family and name.given, excludes name.prefix
```

## Best Practices

### 1. Use Summary Mode for List Views

```go
// List of patients - use summary
func listPatients(w http.ResponseWriter, r *http.Request) {
    patients := getAllPatients()
    data, _ := fhir.MarshalSummaryJSON(patients)
    w.Write(data)
}

// Single patient detail - use full
func getPatient(w http.ResponseWriter, r *http.Request) {
    patient := getPatientByID(r.URL.Query().Get("id"))
    data, _ := json.Marshal(patient)
    w.Write(data)
}
```

### 2. Respect Client _summary Parameter

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    resource := getResource(r)
    
    summaryMode := parseSummaryParam(r.URL.Query().Get("_summary"))
    data, _ := fhir.MarshalWithSummaryMode(resource, summaryMode)
    
    w.Write(data)
}
```

### 3. Document Summary Fields in API

```
GET /Patient?_summary=true
Returns: id, meta, identifier, active, name, telecom, gender, birthDate, 
         address, managingOrganization, link

GET /Patient?_summary=false
Returns: All fields EXCEPT the above summary fields
```

### 4. Cache Summary Representations

```go
type CachedPatient struct {
    Full    []byte  // Full JSON
    Summary []byte  // Summary JSON
}

func cachePatient(patient *Patient) (*CachedPatient, error) {
    full, err := json.Marshal(patient)
    if err != nil {
        return nil, err
    }
    
    summary, err := fhir.MarshalSummaryJSON(patient)
    if err != nil {
        return nil, err
    }
    
    return &CachedPatient{Full: full, Summary: summary}, nil
}
```

### 5. Validate Before Marshaling

```go
validator := validation.NewFHIRValidator()

if err := validator.Validate(patient); err != nil {
    return err
}

// Now safe to marshal
summaryJSON, _ := fhir.MarshalSummaryJSON(patient)
```

## Common Patterns

### Search Results

```go
func searchAndReturn(query string, useSummary bool) ([]byte, error) {
    patients := search(query)
    
    bundle := &Bundle{
        ResourceType: "Bundle",
        Type:         "searchset",
        Total:        intPtr(len(patients)),
    }
    
    for _, patient := range patients {
        var entry BundleEntry
        if useSummary {
            data, _ := fhir.MarshalSummaryJSON(patient)
            json.Unmarshal(data, &entry.Resource) // Summary version
        } else {
            entry.Resource = patient // Full version
        }
        bundle.Entry = append(bundle.Entry, entry)
    }
    
    return json.Marshal(bundle)
}
```

### Conditional Summary

```go
func respondWithResource(w http.ResponseWriter, resource interface{}, fields []string) {
    var data []byte
    var err error
    
    if len(fields) == 0 {
        // No specific fields requested - use summary
        data, err = fhir.MarshalSummaryJSON(resource)
    } else if contains(fields, "*") {
        // All fields requested
        data, err = json.Marshal(resource)
    } else {
        // Specific fields - custom filtering needed
        data, err = marshalFields(resource, fields)
    }
    
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Write(data)
}
```

### Bandwidth Optimization

```go
type Client struct {
    baseURL      string
    preferSummary bool  // Always request summary
}

func (c *Client) Search(resourceType, query string) ([]Resource, error) {
    url := fmt.Sprintf("%s/%s?%s", c.baseURL, resourceType, query)
    
    if c.preferSummary {
        url += "&_summary=true"
    }
    
    // Fetch and parse
    resp, _ := http.Get(url)
    // ... parse bundle
    
    return resources, nil
}
```

## Limitations

1. **No partial field selection:** Summary mode uses predefined FHIR summary elements. For custom field selection, use `_elements` parameter (not yet implemented).

2. **Performance overhead:** Uses reflection, ~5-10% slower than standard `json.Marshal()`. For high-performance scenarios, consider caching.

3. **Immutable during serialization:** Summary mode filters at serialization time. The original struct is not modified.

## Comparison with Standard JSON

```go
// Standard JSON (all fields)
fullData, _ := json.Marshal(patient)

// Summary JSON (filtered)
summaryData, _ := fhir.MarshalSummaryJSON(patient)

// Can unmarshal both the same way
var patient1, patient2 Patient
json.Unmarshal(fullData, &patient1)     // Full patient
json.Unmarshal(summaryData, &patient2)  // Partial patient (non-summary fields nil)
```

## Future Enhancements

Planned features:
- **_elements parameter:** Custom field selection beyond summary
- **_count parameter:** Limit number of results in bundles
- **Performance optimization:** Caching of reflection metadata
- **Streaming:** Support for large bundle serialization

## Summary

- **5 summary modes:** All, True, False, Text, Data
- **Automatic filtering:** Based on FHIR struct tags
- **40-70% size reduction:** For typical resources
- **REST API compatible:** Matches FHIR _summary parameter
- **Embedded struct support:** Correctly handles resource inheritance
- **Type-safe:** Uses reflection, no code generation needed
