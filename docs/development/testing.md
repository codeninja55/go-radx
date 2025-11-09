# Testing Guide

This guide covers testing practices and guidelines for go-radx.

## Overview

go-radx uses Go's built-in testing framework with the following test types:

- **Unit Tests** - Test individual functions and methods
- **Integration Tests** - Test component interactions
- **Validation Tests** - Test FHIR resource validation
- **Example Tests** - Executable examples in documentation

## Running Tests

### Quick Start

```bash
# Run all tests
mise test

# Run with coverage
mise test:coverage

# Run verbosely
mise test:verbose

# Run specific package
go test ./fhir/validation/...

# Run specific test
go test -run TestValidatePatient ./fhir/validation/...
```

### Test Commands

```bash
# All tests
go test ./...

# Verbose output
go test -v ./...

# With coverage
go test -cover ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Race detector
go test -race ./...

# Short mode (skip long tests)
go test -short ./...

# Parallel execution
go test -parallel 4 ./...
```

## Writing Tests

### Unit Tests

#### Basic Test

```go
package primitives

import "testing"

func TestDateParsing(t *testing.T) {
    // Test valid date
    date, err := NewDate("2024-01-15")
    if err != nil {
        t.Errorf("NewDate() error = %v, want nil", err)
    }

    if date.String() != "2024-01-15" {
        t.Errorf("Date.String() = %s, want 2024-01-15", date.String())
    }

    // Test invalid date
    _, err = NewDate("invalid")
    if err == nil {
        t.Error("NewDate() error = nil, want error")
    }
}
```

#### Table-Driven Tests

```go
func TestDateParsing(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "full date",
            input:   "2024-01-15",
            want:    "2024-01-15",
            wantErr: false,
        },
        {
            name:    "year only",
            input:   "2024",
            want:    "2024",
            wantErr: false,
        },
        {
            name:    "year and month",
            input:   "2024-01",
            want:    "2024-01",
            wantErr: false,
        },
        {
            name:    "invalid format",
            input:   "01/15/2024",
            want:    "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            date, err := NewDate(tt.input)

            if (err != nil) != tt.wantErr {
                t.Errorf("NewDate() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if !tt.wantErr && date.String() != tt.want {
                t.Errorf("Date.String() = %s, want %s", date.String(), tt.want)
            }
        })
    }
}
```

### Validation Tests

```go
func TestValidatePatient(t *testing.T) {
    validator := validation.NewFHIRValidator()

    t.Run("valid patient", func(t *testing.T) {
        patient := &resources.Patient{
            Name: []resources.HumanName{
                {Family: stringPtr("Doe")},
            },
        }

        if err := validator.Validate(patient); err != nil {
            t.Errorf("Validate() error = %v, want nil", err)
        }
    })

    t.Run("missing required name", func(t *testing.T) {
        patient := &resources.Patient{}

        err := validator.Validate(patient)
        if err == nil {
            t.Error("Validate() error = nil, want error")
        }

        if !strings.Contains(err.Error(), "Name") {
            t.Errorf("Error should mention 'Name', got: %v", err)
        }
    })
}

func stringPtr(s string) *string { return &s }
```

### Integration Tests

```go
func TestBundleWorkflow(t *testing.T) {
    // Create patient
    birthDate := primitives.MustDate("1990-05-15")
    patient := &resources.Patient{
        ID:        stringPtr("patient-1"),
        BirthDate: &birthDate,
        Name: []resources.HumanName{
            {Family: stringPtr("Doe")},
        },
    }

    // Create observation
    effectiveDateTime := primitives.MustDateTime("2024-01-15T10:30:00Z")
    observation := &resources.Observation{
        ID:     stringPtr("obs-1"),
        Status: "final",
        Code: resources.CodeableConcept{
            Coding: []resources.Coding{
                {Code: stringPtr("85354-9")},
            },
        },
        Subject: resources.Reference{
            Reference: stringPtr("Patient/patient-1"),
        },
        EffectiveDateTime: &effectiveDateTime,
    }

    // Create bundle
    bundle := &resources.Bundle{
        Type: "collection",
        Entry: []resources.BundleEntry{
            {
                FullUrl:  stringPtr("Patient/patient-1"),
                Resource: patient,
            },
            {
                FullUrl:  stringPtr("Observation/obs-1"),
                Resource: observation,
            },
        },
    }

    // Validate bundle
    validator := validation.NewFHIRValidator()
    for i, entry := range bundle.Entry {
        if err := validator.Validate(entry.Resource); err != nil {
            t.Errorf("Entry %d validation failed: %v", i, err)
        }
    }

    // Test serialization
    data, err := json.Marshal(bundle)
    if err != nil {
        t.Fatalf("Marshal() error = %v", err)
    }

    // Test deserialization
    var loaded resources.Bundle
    if err := json.Unmarshal(data, &loaded); err != nil {
        t.Fatalf("Unmarshal() error = %v", err)
    }

    if len(loaded.Entry) != 2 {
        t.Errorf("Bundle entries = %d, want 2", len(loaded.Entry))
    }
}
```

### Benchmark Tests

```go
func BenchmarkValidatePatient(b *testing.B) {
    patient := &resources.Patient{
        Name: []resources.HumanName{
            {Family: stringPtr("Doe")},
        },
    }

    validator := validation.NewFHIRValidator()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = validator.Validate(patient)
    }
}

func BenchmarkDateParsing(b *testing.B) {
    input := "2024-01-15"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = primitives.NewDate(input)
    }
}
```

**Running Benchmarks**:

```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkValidatePatient ./fhir/validation/...

# With memory stats
go test -bench=. -benchmem ./...

# Multiple iterations
go test -bench=. -count=10 ./...
```

### Example Tests

```go
func ExampleNewDate() {
    // Create a date from string
    date := primitives.MustDate("2024-01-15")
    fmt.Println(date.String())
    // Output: 2024-01-15
}

func ExampleDate_Precision() {
    yearOnly := primitives.MustDate("2024")
    fmt.Println(yearOnly.Precision())

    fullDate := primitives.MustDate("2024-01-15")
    fmt.Println(fullDate.Precision())

    // Output:
    // year
    // day
}
```

## Test Organization

### File Naming

```
package/
  ├── code.go           # Production code
  ├── code_test.go      # Unit tests
  ├── integration_test.go  # Integration tests (optional)
  └── examples_test.go     # Example tests (optional)
```

### Package Naming

```go
// Same package for testing private functions
package primitives

func TestInternalFunction(t *testing.T) { }

// Separate package for testing public API only
package primitives_test

import "github.com/codeninja55/go-radx/fhir/primitives"

func TestPublicAPI(t *testing.T) { }
```

## Test Helpers

### Common Helper Functions

```go
// testdata/helpers.go
package testdata

// Helper functions used across test files
func stringPtr(s string) *string       { return &s }
func boolPtr(b bool) *bool             { return &b }
func intPtr(i int) *int                { return &i }
func float64Ptr(f float64) *float64    { return &f }

// CreateTestPatient creates a valid test patient
func CreateTestPatient() *resources.Patient {
    birthDate := primitives.MustDate("1990-05-15")
    return &resources.Patient{
        ID:        stringPtr("test-patient"),
        BirthDate: &birthDate,
        Name: []resources.HumanName{
            {
                Family: stringPtr("Doe"),
                Given:  []string{"John"},
            },
        },
    }
}
```

### Test Fixtures

```go
// testdata/fixtures.go
package testdata

const (
    ValidPatientJSON = `{
        "resourceType": "Patient",
        "id": "example",
        "name": [{"family": "Doe", "given": ["John"]}]
    }`

    InvalidPatientJSON = `{
        "resourceType": "Patient",
        "id": "example"
    }`
)
```

## Coverage

### Measuring Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View in browser
go tool cover -html=coverage.out

# View in terminal
go tool cover -func=coverage.out

# Coverage by package
go test -cover ./...
```

### Coverage Goals

- **Overall** - 80%+ coverage
- **Critical paths** - 90%+ coverage
- **New code** - 80%+ coverage
- **Bug fixes** - Include regression test

### Improving Coverage

```go
// Before (50% coverage)
func Divide(a, b int) int {
    if b == 0 {
        return 0  // Not tested
    }
    return a / b
}

// Test
func TestDivide(t *testing.T) {
    result := Divide(10, 2)
    if result != 5 {
        t.Errorf("got %d, want 5", result)
    }
}

// After (100% coverage)
func TestDivide(t *testing.T) {
    tests := []struct {
        name string
        a, b int
        want int
    }{
        {"normal", 10, 2, 5},
        {"divide by zero", 10, 0, 0},  // Now testing error case
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Divide(tt.a, tt.b)
            if got != tt.want {
                t.Errorf("got %d, want %d", got, tt.want)
            }
        })
    }
}
```

## Continuous Integration

### GitHub Actions

Tests run automatically on:

- **Pull requests** - All tests must pass
- **Push to main** - Regression check
- **Scheduled** - Daily test runs

### CI Workflow

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25.4'

      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v4
```

## Best Practices

### DO

✅ **Write table-driven tests** - Easier to add test cases

✅ **Test error cases** - Don't just test happy paths

✅ **Use descriptive names** - `TestValidatePatientWithMissingName`

✅ **Keep tests simple** - Easy to understand and maintain

✅ **Test behavior, not implementation** - Focus on what, not how

✅ **Use subtests** - `t.Run()` for better organization

✅ **Clean up resources** - Use `defer` or `t.Cleanup()`

### DON'T

❌ **Don't test external services** - Use mocks/stubs

❌ **Don't use global state** - Tests should be isolated

❌ **Don't ignore errors** - Always check error returns

❌ **Don't write flaky tests** - Tests should be deterministic

❌ **Don't test private implementation** - Test public API

❌ **Don't skip tests** - Fix or remove, don't skip

## Debugging Tests

### Verbose Output

```bash
# Print all test output
go test -v ./...

# Print for specific package
go test -v ./fhir/validation/...

# Print only failures
go test ./... | grep FAIL
```

### Running Single Tests

```bash
# Run specific test
go test -run TestValidatePatient ./fhir/validation/...

# Run tests matching pattern
go test -run "Patient" ./...

# Run specific subtest
go test -run TestValidate/valid_patient ./...
```

### Debugging with Delve

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug test
dlv test ./fhir/validation -- -test.run TestValidatePatient

# Set breakpoint
(dlv) break TestValidatePatient
(dlv) continue
```

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Table Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Advanced Testing](https://talks.golang.org/2014/testing.slide)
- [Testing in Go Guide](https://quii.gitbook.io/learn-go-with-tests/)

## Next Steps

- [Contributing Guide](contributing.md)
- [Code Style Guidelines](contributing.md#code-style)
- [Community Support](../community/support.md)
