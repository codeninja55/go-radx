package fhir_test

import (
	"encoding/json"
	"runtime"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/internal/testutil"
	"github.com/codeninja55/go-radx/fhir/primitives"
	"github.com/codeninja55/go-radx/fhir/r5/resources"
	"github.com/codeninja55/go-radx/fhir/validation"
)

// Helper function aliases for benchmarks (shared with example_test.go via package fhir_test)
var (
	stringPtr = testutil.StringPtr
	boolPtr   = testutil.BoolPtr
)

// Benchmark JSON marshaling of a typical Patient resource
func BenchmarkPatient_Marshal(b *testing.B) {
	patient := createTestPatient()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(patient)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark JSON unmarshaling of a typical Patient resource
func BenchmarkPatient_Unmarshal(b *testing.B) {
	patient := createTestPatient()
	data, _ := json.Marshal(patient)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var p resources.Patient
		if err := json.Unmarshal(data, &p); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark round-trip (marshal + unmarshal)
func BenchmarkPatient_RoundTrip(b *testing.B) {
	patient := createTestPatient()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := json.Marshal(patient)
		if err != nil {
			b.Fatal(err)
		}

		var p resources.Patient
		if err := json.Unmarshal(data, &p); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark validation
func BenchmarkPatient_Validation(b *testing.B) {
	patient := createTestPatient()
	validator := validation.NewFHIRValidator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(patient)
	}
}

// Benchmark summary mode serialization
func BenchmarkPatient_SummaryMode(b *testing.B) {
	patient := createTestPatient()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := fhir.MarshalSummaryJSON(patient)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark Bundle creation and navigation
func BenchmarkBundle_Create(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bundle := &fhir.Bundle{
			Type: "searchset",
		}

		helper := fhir.NewBundleHelper(bundle)
		for j := 0; j < 10; j++ {
			patient := createTestPatient()
			_ = helper.AddEntry(patient, stringPtr("Patient/"+string(rune('A'+j))))
		}
	}
}

// Benchmark Bundle resource lookup
func BenchmarkBundle_GetResourceByID(b *testing.B) {
	bundle := createTestBundle(100)
	helper := fhir.NewBundleHelper(bundle)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = helper.GetResourceByID("Patient", "patient-50")
	}
}

// Benchmark Bundle reference resolution
func BenchmarkBundle_ResolveReference(b *testing.B) {
	bundle := createTestBundle(100)
	helper := fhir.NewBundleHelper(bundle)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = helper.ResolveReference("Patient/patient-50")
	}
}

// Benchmark Bundle type filtering
func BenchmarkBundle_GetPatients(b *testing.B) {
	bundle := createTestBundle(100)
	helper := fhir.NewBundleHelper(bundle)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = helper.GetPatients()
	}
}

// Benchmark Observation creation
func BenchmarkObservation_Create(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		obs := &resources.Observation{
			ID:     stringPtr("obs-001"),
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
		}
		runtime.KeepAlive(obs)
	}
}

// Benchmark complex Observation with components
func BenchmarkObservation_WithComponents(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		obs := &resources.Observation{
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
				},
			},
		}
		runtime.KeepAlive(obs)
	}
}

// Helper functions
func createTestPatient() *resources.Patient {
	birthDate := primitives.MustDate("1974-12-25")
	return &resources.Patient{
		ID:     stringPtr("example"),
		Active: boolPtr(true),
		Name: []resources.HumanName{
			{
				Use:    stringPtr("official"),
				Family: stringPtr("Doe"),
				Given:  []string{"John"},
			},
		},
		Gender:    stringPtr("male"),
		BirthDate: &birthDate,
		Telecom: []resources.ContactPoint{
			{
				System: stringPtr("phone"),
				Value:  stringPtr("555-1234"),
			},
		},
		Address: []resources.Address{
			{
				Line:       []string{"123 Main St"},
				City:       stringPtr("Springfield"),
				State:      stringPtr("IL"),
				PostalCode: stringPtr("62701"),
			},
		},
	}
}

func createTestBundle(size int) *fhir.Bundle {
	bundle := &fhir.Bundle{
		Type: "searchset",
	}

	helper := fhir.NewBundleHelper(bundle)

	for i := 0; i < size; i++ {
		patient := createTestPatient()
		patient.ID = stringPtr("patient-" + string(rune('0'+i%10)))
		_ = helper.AddEntry(patient, stringPtr("Patient/patient-"+string(rune('0'+i%10))))
	}

	return bundle
}
