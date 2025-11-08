# FHIR Library for Go

A comprehensive Go library for working with FHIR (Fast Healthcare Interoperability Resources) R4 and R5 specifications.

## Features

- **Complete FHIR Coverage**: All R4 (146 resources, 41 complex types) and R5 (158 resources, 44 complex types)
- **Type-Safe Primitives**: Custom Date, DateTime, Time, and Instant types with validation
- **Standards Compliant**: Generated directly from official FHIR StructureDefinitions
- **JSON Support**: Full marshaling/unmarshaling with `encoding/json`
- **Partial Precision**: Supports FHIR's partial date/time precision (YYYY, YYYY-MM, YYYY-MM-DD)
- **Validation**: Built-in validation for primitive types with proper error handling

## Installation

```bash
go get github.com/harrison-ai/go-radx/fhir
```

## Quick Start

### Creating a FHIR Resource

```go
package main

import (
    "encoding/json"
    "fmt"

    "github.com/harrison-ai/go-radx/fhir/r4/resources"
    "github.com/harrison-ai/go-radx/fhir/primitives"
)

func main() {
    // Create a Patient resource
    active := true
    birthDate := primitives.MustDate("1974-12-25")

    patient := resources.Patient{
        ID:     stringPtr("example"),
        Active: &active,
        Name: []resources.HumanName{
            {
                Use:    stringPtr("official"),
                Family: stringPtr("Doe"),
                Given:  []string{"John"},
            },
        },
        Gender:    stringPtr("male"),
        BirthDate: &birthDate,
    }

    // Marshal to JSON
    data, err := json.MarshalIndent(patient, "", "  ")
    if err != nil {
        panic(err)
    }

    fmt.Println(string(data))
}

func stringPtr(s string) *string {
    return &s
}
```

### Reading FHIR JSON

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/harrison-ai/go-radx/fhir/r4/resources"
)

func main() {
    // Read FHIR JSON file
    data, err := os.ReadFile("patient.json")
    if err != nil {
        panic(err)
    }

    // Unmarshal into Patient struct
    var patient resources.Patient
    if err := json.Unmarshal(data, &patient); err != nil {
        panic(err)
    }

    // Access fields
    fmt.Printf("Patient ID: %s\n", *patient.ID)
    fmt.Printf("Birth Date: %s\n", patient.BirthDate.String())
    fmt.Printf("Date Precision: %s\n", patient.BirthDate.Precision())

    // Convert to time.Time if needed
    if t, err := patient.BirthDate.Time(); err == nil {
        fmt.Printf("As time.Time: %v\n", t)
    }
}
```

## Working with Primitive Types

### Date Type

FHIR dates support partial precision:

```go
import "github.com/harrison-ai/go-radx/fhir/primitives"

// Year only
yearDate := primitives.MustDate("2024")
fmt.Println(yearDate.Precision()) // "year"

// Year and month
monthDate := primitives.MustDate("2024-01")
fmt.Println(monthDate.Precision()) // "month"

// Full date
fullDate := primitives.MustDate("2024-01-15")
fmt.Println(fullDate.Precision()) // "day"

// Convert to time.Time (uses first day for partial dates)
t, _ := fullDate.Time()
fmt.Println(t) // 2024-01-15 00:00:00 +0000 UTC

// Create from time.Time
now := time.Now()
date := primitives.FromTime(now)              // Full precision
yearOnly := primitives.FromTimeYear(now)      // Year only
monthOnly := primitives.FromTimeMonth(now)    // Year-month
```

### DateTime Type

Supports partial precision and optional timezones:

```go
// Various precisions
dt1 := primitives.MustDateTime("2024")                        // Year
dt2 := primitives.MustDateTime("2024-01")                     // Year-month
dt3 := primitives.MustDateTime("2024-01-15")                  // Date
dt4 := primitives.MustDateTime("2024-01-15T10:30:00Z")        // Full with timezone
dt5 := primitives.MustDateTime("2024-01-15T10:30:00+10:00")   // With offset
dt6 := primitives.MustDateTime("2024-01-15T10:30:00.123Z")    // With milliseconds

// Create from time.Time
dt := primitives.FromTimeDateTime(time.Now())
```

### Time Type

24-hour format with optional fractional seconds:

```go
// Create time
t1 := primitives.MustTime("10:30:00")
t2 := primitives.MustTime("10:30:00.123")

// Get as duration from midnight
duration, _ := t1.Duration()
fmt.Println(duration) // 10h30m0s

// Apply to a date
date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
datetime, _ := t1.TimeOfDay(date)
fmt.Println(datetime) // 2024-01-15 10:30:00 +0000 UTC
```

### Instant Type

Always includes timezone:

```go
// Create instant (requires timezone)
instant := primitives.MustInstant("2024-01-15T10:30:00Z")

// Convert to time.Time
t, _ := instant.Time()

// Create from time.Time
instant2 := primitives.FromTimeInstant(time.Now())
instant3 := primitives.FromTimeInstantNano(time.Now()) // With nanosecond precision
```

## Working with Resources

### Creating an Observation

```go
// Create a blood pressure observation
obs := resources.Observation{
    ID:     stringPtr("blood-pressure"),
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
    Subject: resources.Reference{
        Reference: stringPtr("Patient/example"),
    },
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
```

### Creating a Bundle

```go
// Create a searchset bundle
bundle := resources.Bundle{
    ID:   stringPtr("example-bundle"),
    Type: "searchset",
    Total: uintPtr(2),
    Entry: []resources.BundleEntry{
        {
            FullUrl: stringPtr("https://example.com/Patient/1"),
            Resource: patientResource1,
        },
        {
            FullUrl: stringPtr("https://example.com/Patient/2"),
            Resource: patientResource2,
        },
    },
}
```

## FHIR R4 vs R5

The library supports both FHIR R4 and R5:

```go
// R4 (most widely adopted)
import "github.com/harrison-ai/go-radx/fhir/r4/resources"

// R5 (latest specification)
import "github.com/harrison-ai/go-radx/fhir/r5/resources"
```

**Key Differences:**
- R5 has 158 resources vs R4's 146
- R5 introduces `integer64` type
- Some resources have different fields between versions
- Use R4 for maximum compatibility, R5 for latest features

## Validation

Primitive types perform automatic validation:

```go
// Valid dates
date1, err := primitives.NewDate("2024-01-15")       // OK
date2, err := primitives.NewDate("2024-01")          // OK
date3, err := primitives.NewDate("2024")             // OK

// Invalid dates return errors
date4, err := primitives.NewDate("2024-1-15")        // Error: invalid format
date5, err := primitives.NewDate("invalid")          // Error: invalid format

// MustDate panics on invalid input (use in tests or when input is trusted)
date := primitives.MustDate("2024-01-15")  // OK
date := primitives.MustDate("invalid")     // Panics
```

## BackboneElements

FHIR BackboneElements are nested structures within resources:

```go
// PatientContact is a BackboneElement
contact := resources.PatientContact{
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
        Family: stringPtr("Emergency"),
        Given:  []string{"Contact"},
    },
    Telecom: []resources.ContactPoint{
        {
            System: stringPtr("phone"),
            Value:  stringPtr("555-1234"),
        },
    },
}

patient.Contact = []resources.PatientContact{contact}
```

## Choice Types

Some FHIR fields accept multiple types (e.g., `deceased[x]` can be boolean or dateTime):

```go
// Currently represented as *any
patient.Deceased = boolPtr(false)                  // deceasedBoolean
// OR
patient.Deceased = primitives.MustDateTime("2015-02-14T13:42:00+10:00") // deceasedDateTime

// When unmarshaling, check the type:
switch v := patient.Deceased.(type) {
case bool:
    fmt.Printf("Deceased: %v\n", v)
case primitives.DateTime:
    fmt.Printf("Deceased at: %s\n", v.String())
}
```

## Error Handling

Always check errors when working with FHIR data:

```go
// Creating primitives
date, err := primitives.NewDate("2024-01-15")
if err != nil {
    return fmt.Errorf("invalid date: %w", err)
}

// Unmarshaling JSON
var patient resources.Patient
if err := json.Unmarshal(data, &patient); err != nil {
    return fmt.Errorf("unmarshal patient: %w", err)
}

// Converting to time.Time
t, err := date.Time()
if err != nil {
    return fmt.Errorf("convert to time: %w", err)
}
```

## Performance Considerations

- **Zero Allocations**: Primitives use value types where possible
- **Lazy Parsing**: Time conversions are only done when needed
- **Efficient JSON**: Uses standard `encoding/json` with no reflection overhead
- **Memory**: Pointer fields allow nil for optional values, saving memory

## Testing

The library includes comprehensive test coverage:

```bash
# Test primitives
cd fhir/primitives && go test -v

# Test R4 resources
cd fhir/r4/resources && go test -v

# Test R5 resources
cd fhir/r5/resources && go test -v
```

## Code Generation

The library is generated from official FHIR StructureDefinitions. To regenerate:

```bash
# Generate R4 resources
./fhir/bin/fhir-gen -version r4 \
    -input fhir_schemas/profiles-resources.json \
    -output fhir/r4/resources

# Generate R4 complex types
./fhir/bin/fhir-gen -version r4 \
    -input fhir_schemas/profiles-types.json \
    -output fhir/r4/resources

# Generate R5 resources
./fhir/bin/fhir-gen -version r5 \
    -input fhir_schemas/r5/profiles-resources.json \
    -output fhir/r5/resources

# Generate R5 complex types
./fhir/bin/fhir-gen -version r5 \
    -input fhir_schemas/r5/profiles-types.json \
    -output fhir/r5/resources
```

## Comparison with Other Libraries

### vs google/fhir

- **go-radx/fhir**: Simple, idiomatic Go with no dependencies
- **google/fhir**: Protocol buffer based, more complex

### vs samply/golang-fhir-models

- **go-radx/fhir**: Custom primitive types with validation, R5 support
- **samply/golang-fhir-models**: String-based primitives, R4 only

### vs friendly-fhir/go-fhir

- **go-radx/fhir**: Generated from official StructureDefinitions
- **friendly-fhir/go-fhir**: Handwritten, may lag behind spec updates

## Contributing

Contributions are welcome! Please:

1. Run tests: `go test ./...`
2. Format code: `go fmt ./...`
3. Update documentation for new features

## License

See LICENSE file for details.

## Resources

- [FHIR R4 Specification](https://hl7.org/fhir/R4/)
- [FHIR R5 Specification](https://hl7.org/fhir/R5/)
- [go-radx GitHub](https://github.com/harrison-ai/go-radx)
