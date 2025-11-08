# FHIR Examples

This guide provides comprehensive examples of working with FHIR resources in go-radx.

## Basic Examples

### Creating a Patient

This example demonstrates creating a complete Patient resource with all common fields:

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/codeninja55/go-radx/fhir/primitives"
    "github.com/codeninja55/go-radx/fhir/r5/resources"
)

func main() {
    patient := createPatient()

    data, err := json.MarshalIndent(patient, "", "  ")
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error marshaling patient: %v\n", err)
        os.Exit(1)
    }

    fmt.Println(string(data))
}

func createPatient() resources.Patient {
    birthDate := primitives.MustDate("1974-12-25")

    return resources.Patient{
        ID:     stringPtr("example"),
        Active: boolPtr(true),
        Name: []resources.HumanName{
            {
                Use:    stringPtr("official"),
                Family: stringPtr("Chalmers"),
                Given:  []string{"Peter", "James"},
            },
            {
                Use:   stringPtr("usual"),
                Given: []string{"Jim"},
            },
        },
        Gender:    stringPtr("male"),
        BirthDate: &birthDate,
        Telecom: []resources.ContactPoint{
            {
                System: stringPtr("phone"),
                Value:  stringPtr("(03) 5555 6473"),
                Use:    stringPtr("work"),
                Rank:   intPtr(1),
            },
            {
                System: stringPtr("phone"),
                Value:  stringPtr("(03) 3410 5613"),
                Use:    stringPtr("mobile"),
                Rank:   intPtr(2),
            },
        },
        Address: []resources.Address{
            {
                Use:        stringPtr("home"),
                Type:       stringPtr("both"),
                Line:       []string{"534 Erewhon St"},
                City:       stringPtr("PleasantVille"),
                State:      stringPtr("Vic"),
                PostalCode: stringPtr("3999"),
                Period: &resources.Period{
                    Start: datetimePtr("1974-12-25"),
                },
            },
        },
        MaritalStatus: &resources.CodeableConcept{
            Coding: []resources.Coding{
                {
                    System:  stringPtr("http://terminology.hl7.org/CodeSystem/v3-MaritalStatus"),
                    Code:    stringPtr("M"),
                    Display: stringPtr("Married"),
                },
            },
        },
        Contact: []resources.PatientContact{
            {
                Relationship: []resources.CodeableConcept{
                    {
                        Coding: []resources.Coding{
                            {
                                System: stringPtr("http://terminology.hl7.org/CodeSystem/v2-0131"),
                                Code:   stringPtr("N"),
                            },
                        },
                    },
                },
                Name: &resources.HumanName{
                    Family: stringPtr("du Marché"),
                    Given:  []string{"Bénédicte"},
                },
                Telecom: []resources.ContactPoint{
                    {
                        System: stringPtr("phone"),
                        Value:  stringPtr("+33 (237) 998327"),
                    },
                },
                Gender: stringPtr("female"),
            },
        },
    }
}

// Helper functions
func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool       { return &b }
func intPtr(i int) *int          { return &i }

func datetimePtr(s string) *primitives.DateTime {
    dt := primitives.MustDateTime(s)
    return &dt
}
```

### Creating an Observation

This example shows how to create a blood pressure observation with components:

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/codeninja55/go-radx/fhir/primitives"
    "github.com/codeninja55/go-radx/fhir/r5/resources"
)

func main() {
    observation := createBloodPressureObservation()

    data, err := json.MarshalIndent(observation, "", "  ")
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error marshaling observation: %v\n", err)
        os.Exit(1)
    }

    fmt.Println(string(data))
}

func createBloodPressureObservation() resources.Observation {
    effectiveDateTime := primitives.MustDateTime("2024-01-15T10:30:00Z")

    return resources.Observation{
        ID:     stringPtr("blood-pressure"),
        Status: "final",
        Category: []resources.CodeableConcept{
            {
                Coding: []resources.Coding{
                    {
                        System:  stringPtr("http://terminology.hl7.org/CodeSystem/observation-category"),
                        Code:    stringPtr("vital-signs"),
                        Display: stringPtr("Vital Signs"),
                    },
                },
            },
        },
        Code: resources.CodeableConcept{
            Coding: []resources.Coding{
                {
                    System:  stringPtr("http://loinc.org"),
                    Code:    stringPtr("85354-9"),
                    Display: stringPtr("Blood pressure panel"),
                },
            },
            Text: stringPtr("Blood pressure systolic & diastolic"),
        },
        Subject: resources.Reference{
            Reference: stringPtr("Patient/example"),
        },
        EffectiveDateTime: &effectiveDateTime,
        Component: []resources.ObservationComponent{
            {
                Code: resources.CodeableConcept{
                    Coding: []resources.Coding{
                        {
                            System:  stringPtr("http://loinc.org"),
                            Code:    stringPtr("8480-6"),
                            Display: stringPtr("Systolic blood pressure"),
                        },
                    },
                },
                ValueQuantity: &resources.Quantity{
                    Value:  float64Ptr(120),
                    Unit:   stringPtr("mmHg"),
                    System: stringPtr("http://unitsofmeasure.org"),
                    Code:   stringPtr("mm[Hg]"),
                },
            },
            {
                Code: resources.CodeableConcept{
                    Coding: []resources.Coding{
                        {
                            System:  stringPtr("http://loinc.org"),
                            Code:    stringPtr("8462-4"),
                            Display: stringPtr("Diastolic blood pressure"),
                        },
                    },
                },
                ValueQuantity: &resources.Quantity{
                    Value:  float64Ptr(80),
                    Unit:   stringPtr("mmHg"),
                    System: stringPtr("http://unitsofmeasure.org"),
                    Code:   stringPtr("mm[Hg]"),
                },
            },
        },
        Interpretation: []resources.CodeableConcept{
            {
                Coding: []resources.Coding{
                    {
                        System:  stringPtr("http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation"),
                        Code:    stringPtr("N"),
                        Display: stringPtr("Normal"),
                    },
                },
            },
        },
    }
}

func stringPtr(s string) *string    { return &s }
func float64Ptr(f float64) *float64 { return &f }
```

## Reading FHIR Resources

### Reading from JSON File

This example demonstrates parsing a FHIR Patient resource from a JSON file and accessing its fields:

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/codeninja55/go-radx/fhir/r5/resources"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Fprintf(os.Stderr, "Usage: %s <fhir-json-file>\n", os.Args[0])
        os.Exit(1)
    }

    filename := os.Args[1]

    // Read the FHIR JSON file
    data, err := os.ReadFile(filename)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
        os.Exit(1)
    }

    // Unmarshal into Patient struct
    var patient resources.Patient
    if err := json.Unmarshal(data, &patient); err != nil {
        fmt.Fprintf(os.Stderr, "Error unmarshaling JSON: %v\n", err)
        os.Exit(1)
    }

    // Access and print patient information
    fmt.Println("=== Patient Information ===")

    if patient.ID != nil {
        fmt.Printf("ID: %s\n", *patient.ID)
    }

    if patient.Active != nil {
        fmt.Printf("Active: %v\n", *patient.Active)
    }

    // Print names
    if len(patient.Name) > 0 {
        fmt.Println("\nNames:")
        for i, name := range patient.Name {
            fmt.Printf("  %d. ", i+1)
            if name.Use != nil {
                fmt.Printf("[%s] ", *name.Use)
            }
            if len(name.Given) > 0 {
                for _, given := range name.Given {
                    fmt.Printf("%s ", given)
                }
            }
            if name.Family != nil {
                fmt.Printf("%s", *name.Family)
            }
            fmt.Println()
        }
    }

    // Print birth date with precision info
    if patient.BirthDate != nil {
        fmt.Printf("\nBirth Date: %s\n", patient.BirthDate.String())
        fmt.Printf("Precision: %s\n", patient.BirthDate.Precision())

        // Convert to time.Time
        if t, err := patient.BirthDate.Time(); err == nil {
            fmt.Printf("As time.Time: %v\n", t)
        }
    }

    // Print contact information
    if len(patient.Telecom) > 0 {
        fmt.Println("\nContact Information:")
        for i, telecom := range patient.Telecom {
            fmt.Printf("  %d. ", i+1)
            if telecom.System != nil {
                fmt.Printf("%s: ", *telecom.System)
            }
            if telecom.Value != nil {
                fmt.Printf("%s", *telecom.Value)
            }
            if telecom.Use != nil {
                fmt.Printf(" (%s)", *telecom.Use)
            }
            fmt.Println()
        }
    }
}
```

**Usage**:

```bash
go run read_patient.go patient.json
```

**Output**:

```
=== Patient Information ===
ID: example
Active: true

Names:
  1. [official] Peter James Chalmers
  2. [usual] Jim

Birth Date: 1974-12-25
Precision: day
As time.Time: 1974-12-25 00:00:00 +0000 UTC

Contact Information:
  1. phone: (03) 5555 6473 (work)
  2. phone: (03) 3410 5613 (mobile)
```

## Working with Bundles

### Searching and Processing Results

This example shows how to work with search results returned in a Bundle:

```go
package main

import (
    "encoding/json"
    "fmt"

    "github.com/codeninja55/go-radx/fhir/r5/resources"
)

func main() {
    // Simulate a FHIR search result bundle
    bundleJSON := `{
        "resourceType": "Bundle",
        "type": "searchset",
        "total": 2,
        "entry": [
            {
                "fullUrl": "http://example.org/Patient/1",
                "resource": {
                    "resourceType": "Patient",
                    "id": "1",
                    "active": true,
                    "name": [{"family": "Doe", "given": ["John"]}],
                    "gender": "male"
                }
            },
            {
                "fullUrl": "http://example.org/Patient/2",
                "resource": {
                    "resourceType": "Patient",
                    "id": "2",
                    "active": true,
                    "name": [{"family": "Smith", "given": ["Jane"]}],
                    "gender": "female"
                }
            }
        ],
        "link": [
            {"relation": "self", "url": "http://example.org/Patient?name=doe"},
            {"relation": "next", "url": "http://example.org/Patient?name=doe&page=2"}
        ]
    }`

    // Parse bundle
    var bundle resources.Bundle
    if err := json.Unmarshal([]byte(bundleJSON), &bundle); err != nil {
        panic(err)
    }

    // Process results
    fmt.Printf("Search returned %d total results\n", *bundle.Total)
    fmt.Printf("This page contains %d results\n", len(bundle.Entry))

    // Iterate through entries
    for i, entry := range bundle.Entry {
        var patient resources.Patient
        if err := json.Unmarshal(entry.Resource, &patient); err != nil {
            continue
        }

        fmt.Printf("\n%d. Patient ID: %s\n", i+1, *patient.ID)
        if len(patient.Name) > 0 {
            name := patient.Name[0]
            fmt.Printf("   Name: %s", name.Given[0])
            if name.Family != nil {
                fmt.Printf(" %s", *name.Family)
            }
            fmt.Println()
        }
        if patient.Gender != nil {
            fmt.Printf("   Gender: %s\n", *patient.Gender)
        }
    }

    // Check for next page
    for _, link := range bundle.Link {
        if link.Relation == "next" {
            fmt.Printf("\nNext page available at: %s\n", link.Url)
        }
    }
}
```

### Paginated Search Results

This example demonstrates handling pagination when fetching multiple pages of search results:

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/codeninja55/go-radx/fhir/r5/resources"
)

func fetchAllPatients(baseURL string) error {
    var allPatients []resources.Patient
    nextURL := baseURL + "/Patient?name=smith"

    for nextURL != "" {
        // Fetch page
        resp, err := http.Get(nextURL)
        if err != nil {
            return err
        }
        defer resp.Body.Close()

        // Parse bundle
        var bundle resources.Bundle
        if err := json.NewDecoder(resp.Body).Decode(&bundle); err != nil {
            return err
        }

        // Extract patients from this page
        for _, entry := range bundle.Entry {
            var patient resources.Patient
            if err := json.Unmarshal(entry.Resource, &patient); err != nil {
                continue
            }
            allPatients = append(allPatients, patient)
        }

        fmt.Printf("Fetched page with %d patients (total so far: %d)\n",
            len(bundle.Entry), len(allPatients))

        // Get next page URL
        nextURL = ""
        for _, link := range bundle.Link {
            if link.Relation == "next" {
                nextURL = link.Url
                break
            }
        }
    }

    fmt.Printf("\nFetched %d total patients across all pages\n", len(allPatients))
    return nil
}
```

## Complete Healthcare Workflow

This example demonstrates a complete healthcare workflow creating a patient encounter with observations and medication:

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "time"

    "github.com/codeninja55/go-radx/fhir/primitives"
    "github.com/codeninja55/go-radx/fhir/r5/resources"
    "github.com/codeninja55/go-radx/fhir/validation"
)

func main() {
    // Create patient
    patient := createPatient()

    // Create encounter
    encounter := createEncounter(patient.ID)

    // Create vital signs
    observations := createVitalSigns(patient.ID, encounter.ID)

    // Create medication request
    medication := createMedicationRequest(patient.ID, encounter.ID)

    // Create bundle with all resources
    bundle := createBundle(patient, encounter, observations, medication)

    // Validate all resources
    if err := validateBundle(bundle); err != nil {
        log.Fatalf("Validation failed: %v", err)
    }

    // Serialize to JSON
    data, err := json.MarshalIndent(bundle, "", "  ")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Healthcare workflow bundle created successfully!")
    fmt.Printf("Bundle contains %d resources\n", len(bundle.Entry))
}

func createPatient() *resources.Patient {
    birthDate := primitives.MustDate("1970-05-15")
    return &resources.Patient{
        ID:     stringPtr("patient-001"),
        Active: boolPtr(true),
        Name: []resources.HumanName{
            {
                Use:    stringPtr("official"),
                Family: stringPtr("Smith"),
                Given:  []string{"Jane", "Marie"},
            },
        },
        Gender:    stringPtr("female"),
        BirthDate: &birthDate,
        Telecom: []resources.ContactPoint{
            {
                System: stringPtr("phone"),
                Value:  stringPtr("+1-555-0123"),
                Use:    stringPtr("mobile"),
            },
        },
    }
}

func createEncounter(patientID *string) *resources.Encounter {
    now := primitives.MustDateTime(time.Now().Format(time.RFC3339))
    return &resources.Encounter{
        ID:     stringPtr("encounter-001"),
        Status: "finished",
        Class: []resources.CodeableConcept{
            {
                Coding: []resources.Coding{
                    {
                        System:  stringPtr("http://terminology.hl7.org/CodeSystem/v3-ActCode"),
                        Code:    stringPtr("AMB"),
                        Display: stringPtr("ambulatory"),
                    },
                },
            },
        },
        Subject: &resources.Reference{
            Reference: stringPtr(fmt.Sprintf("Patient/%s", *patientID)),
            Display:   stringPtr("Jane Smith"),
        },
        Period: &resources.Period{
            Start: &now,
            End:   &now,
        },
    }
}

func createVitalSigns(patientID, encounterID *string) []*resources.Observation {
    effectiveDateTime := primitives.MustDateTime(time.Now().Format(time.RFC3339))

    // Blood Pressure
    bp := &resources.Observation{
        ID:     stringPtr("obs-bp-001"),
        Status: "final",
        Code: resources.CodeableConcept{
            Coding: []resources.Coding{
                {
                    System:  stringPtr("http://loinc.org"),
                    Code:    stringPtr("85354-9"),
                    Display: stringPtr("Blood pressure panel"),
                },
            },
        },
        Subject: &resources.Reference{
            Reference: stringPtr(fmt.Sprintf("Patient/%s", *patientID)),
        },
        EffectiveDateTime: &effectiveDateTime,
        Component: []resources.ObservationComponent{
            {
                Code: resources.CodeableConcept{
                    Coding: []resources.Coding{
                        {
                            System:  stringPtr("http://loinc.org"),
                            Code:    stringPtr("8480-6"),
                            Display: stringPtr("Systolic blood pressure"),
                        },
                    },
                },
                ValueQuantity: &resources.Quantity{
                    Value:  float64Ptr(120),
                    Unit:   stringPtr("mmHg"),
                    System: stringPtr("http://unitsofmeasure.org"),
                    Code:   stringPtr("mm[Hg]"),
                },
            },
            {
                Code: resources.CodeableConcept{
                    Coding: []resources.Coding{
                        {
                            System:  stringPtr("http://loinc.org"),
                            Code:    stringPtr("8462-4"),
                            Display: stringPtr("Diastolic blood pressure"),
                        },
                    },
                },
                ValueQuantity: &resources.Quantity{
                    Value:  float64Ptr(80),
                    Unit:   stringPtr("mmHg"),
                    System: stringPtr("http://unitsofmeasure.org"),
                    Code:   stringPtr("mm[Hg]"),
                },
            },
        },
    }

    // Heart Rate
    hr := &resources.Observation{
        ID:     stringPtr("obs-hr-001"),
        Status: "final",
        Code: resources.CodeableConcept{
            Coding: []resources.Coding{
                {
                    System:  stringPtr("http://loinc.org"),
                    Code:    stringPtr("8867-4"),
                    Display: stringPtr("Heart rate"),
                },
            },
        },
        Subject: &resources.Reference{
            Reference: stringPtr(fmt.Sprintf("Patient/%s", *patientID)),
        },
        EffectiveDateTime: &effectiveDateTime,
        ValueQuantity: &resources.Quantity{
            Value:  float64Ptr(72),
            Unit:   stringPtr("beats/minute"),
            System: stringPtr("http://unitsofmeasure.org"),
            Code:   stringPtr("/min"),
        },
    }

    return []*resources.Observation{bp, hr}
}

func createMedicationRequest(patientID, encounterID *string) *resources.MedicationRequest {
    authoredOn := primitives.MustDateTime(time.Now().Format(time.RFC3339))

    return &resources.MedicationRequest{
        ID:     stringPtr("medreq-001"),
        Status: "active",
        Intent: "order",
        Medication: resources.CodeableReference{
            Concept: &resources.CodeableConcept{
                Coding: []resources.Coding{
                    {
                        System:  stringPtr("http://www.nlm.nih.gov/research/umls/rxnorm"),
                        Code:    stringPtr("197361"),
                        Display: stringPtr("Lisinopril 10 MG Oral Tablet"),
                    },
                },
                Text: stringPtr("Lisinopril 10mg tablet"),
            },
        },
        Subject: &resources.Reference{
            Reference: stringPtr(fmt.Sprintf("Patient/%s", *patientID)),
            Display:   stringPtr("Jane Smith"),
        },
        AuthoredOn: &authoredOn,
        DosageInstruction: []resources.Dosage{
            {
                Text: stringPtr("Take one tablet by mouth once daily"),
            },
        },
    }
}

func createBundle(patient *resources.Patient, encounter *resources.Encounter,
    observations []*resources.Observation, medication *resources.MedicationRequest) *resources.Bundle {

    entries := []resources.BundleEntry{
        {
            FullUrl:  stringPtr(fmt.Sprintf("Patient/%s", *patient.ID)),
            Resource: patient,
        },
        {
            FullUrl:  stringPtr(fmt.Sprintf("Encounter/%s", *encounter.ID)),
            Resource: encounter,
        },
    }

    for _, obs := range observations {
        entries = append(entries, resources.BundleEntry{
            FullUrl:  stringPtr(fmt.Sprintf("Observation/%s", *obs.ID)),
            Resource: obs,
        })
    }

    entries = append(entries, resources.BundleEntry{
        FullUrl:  stringPtr(fmt.Sprintf("MedicationRequest/%s", *medication.ID)),
        Resource: medication,
    })

    return &resources.Bundle{
        Type:  "collection",
        Entry: entries,
    }
}

func validateBundle(bundle *resources.Bundle) error {
    validator := validation.NewFHIRValidator()

    for i, entry := range bundle.Entry {
        if err := validator.Validate(entry.Resource); err != nil {
            return fmt.Errorf("entry %d validation failed: %w", i, err)
        }
    }

    fmt.Println("All resources validated successfully!")
    return nil
}

// Helper functions
func stringPtr(s string) *string       { return &s }
func boolPtr(b bool) *bool             { return &b }
func float64Ptr(f float64) *float64    { return &f }
```

## Running the Examples

To run any of these examples:

1. Save the example code to a file (e.g., `create_patient.go`)
2. Run with Go:

```bash
go run create_patient.go
```

Or build and run:

```bash
go build -o example create_patient.go
./example
```

## Next Steps

- Read the [FHIR User Guide](../user-guide/fhir/index.md) for detailed documentation
- See [Validation](../user-guide/fhir/validation.md) for validation techniques
- Explore [Bundles](../user-guide/fhir/bundles.md) for bundle utilities
- Learn about [Primitives](../user-guide/fhir/primitives.md) for date/time handling

## Additional Resources

- [FHIR R5 Specification](http://hl7.org/fhir/R5/)
- [LOINC Codes](https://loinc.org/)
- [RxNorm Drug Codes](https://www.nlm.nih.gov/research/umls/rxnorm/)
- [FHIR Terminology](http://hl7.org/fhir/R5/terminologies.html)
