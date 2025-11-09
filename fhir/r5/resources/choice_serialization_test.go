package resources

import (
	"encoding/json"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/internal/testutil"
	"github.com/codeninja55/go-radx/fhir/primitives"
)

// TestPatientDeceasedChoice_Serialization tests JSON marshaling and unmarshaling
// of Patient.deceased[x] choice type
func TestPatientDeceasedChoice_Serialization(t *testing.T) {
	t.Run("marshal Patient with DeceasedBoolean", func(t *testing.T) {
		patient := &Patient{
			DomainResource: fhir.DomainResource{
				Resource: fhir.Resource{
					ResourceType: "Patient",
					ID:           testutil.StringPtr("example"),
				},
			},
			DeceasedBoolean: testutil.BoolPtr(true),
		}

		// Marshal to JSON
		data, err := json.Marshal(patient)
		if err != nil {
			t.Fatalf("Failed to marshal patient: %v", err)
		}

		// Verify JSON contains deceasedBoolean field
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Failed to unmarshal to map: %v", err)
		}

		if _, exists := raw["deceasedBoolean"]; !exists {
			t.Error("Expected deceasedBoolean field in JSON")
		}

		if deceasedBoolean, ok := raw["deceasedBoolean"].(bool); !ok || !deceasedBoolean {
			t.Errorf("Expected deceasedBoolean=true, got %v", raw["deceasedBoolean"])
		}

		// Verify deceasedDateTime is NOT present
		if _, exists := raw["deceasedDateTime"]; exists {
			t.Error("deceasedDateTime should not be present when DeceasedBoolean is set")
		}
	})

	t.Run("marshal Patient with DeceasedDateTime", func(t *testing.T) {
		deceasedDate := primitives.MustDateTime("2024-01-15T10:30:00Z")

		patient := &Patient{
			DomainResource: fhir.DomainResource{
				Resource: fhir.Resource{
					ResourceType: "Patient",
					ID:           testutil.StringPtr("example"),
				},
			},
			DeceasedDateTime: &deceasedDate,
		}

		// Marshal to JSON
		data, err := json.Marshal(patient)
		if err != nil {
			t.Fatalf("Failed to marshal patient: %v", err)
		}

		// Verify JSON contains deceasedDateTime field
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Failed to unmarshal to map: %v", err)
		}

		if _, exists := raw["deceasedDateTime"]; !exists {
			t.Error("Expected deceasedDateTime field in JSON")
		}

		// Verify deceasedBoolean is NOT present
		if _, exists := raw["deceasedBoolean"]; exists {
			t.Error("deceasedBoolean should not be present when DeceasedDateTime is set")
		}
	})

	t.Run("unmarshal JSON with deceasedBoolean", func(t *testing.T) {
		jsonData := `{
			"resourceType": "Patient",
			"id": "example",
			"active": true,
			"deceasedBoolean": true
		}`

		var patient Patient
		if err := json.Unmarshal([]byte(jsonData), &patient); err != nil {
			t.Fatalf("Failed to unmarshal patient: %v", err)
		}

		// Verify DeceasedBoolean is set
		if patient.DeceasedBoolean == nil {
			t.Fatal("Expected DeceasedBoolean to be set")
		}

		if !*patient.DeceasedBoolean {
			t.Errorf("Expected DeceasedBoolean=true, got false")
		}

		// Verify DeceasedDateTime is NOT set
		if patient.DeceasedDateTime != nil {
			t.Error("Expected DeceasedDateTime to be nil")
		}
	})

	t.Run("unmarshal JSON with deceasedDateTime", func(t *testing.T) {
		jsonData := `{
			"resourceType": "Patient",
			"id": "example",
			"active": true,
			"deceasedDateTime": "2024-01-15T10:30:00Z"
		}`

		var patient Patient
		if err := json.Unmarshal([]byte(jsonData), &patient); err != nil {
			t.Fatalf("Failed to unmarshal patient: %v", err)
		}

		// Verify DeceasedDateTime is set
		if patient.DeceasedDateTime == nil {
			t.Fatal("Expected DeceasedDateTime to be set")
		}

		if patient.DeceasedDateTime.String() != "2024-01-15T10:30:00Z" {
			t.Errorf("Expected DeceasedDateTime=2024-01-15T10:30:00Z, got %s", patient.DeceasedDateTime.String())
		}

		// Verify DeceasedBoolean is NOT set
		if patient.DeceasedBoolean != nil {
			t.Error("Expected DeceasedBoolean to be nil")
		}
	})

	t.Run("unmarshal JSON with neither choice field", func(t *testing.T) {
		jsonData := `{
			"resourceType": "Patient",
			"id": "example",
			"active": true
		}`

		var patient Patient
		if err := json.Unmarshal([]byte(jsonData), &patient); err != nil {
			t.Fatalf("Failed to unmarshal patient: %v", err)
		}

		// Verify both choice fields are nil
		if patient.DeceasedBoolean != nil {
			t.Error("Expected DeceasedBoolean to be nil")
		}

		if patient.DeceasedDateTime != nil {
			t.Error("Expected DeceasedDateTime to be nil")
		}
	})

	t.Run("roundtrip with DeceasedBoolean", func(t *testing.T) {
		original := &Patient{
			DomainResource: fhir.DomainResource{
				Resource: fhir.Resource{
					ResourceType: "Patient",
					ID:           testutil.StringPtr("roundtrip"),
				},
			},
			Active:          testutil.BoolPtr(true),
			DeceasedBoolean: testutil.BoolPtr(false),
		}

		// Marshal
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		// Unmarshal
		var retrieved Patient
		if err := json.Unmarshal(data, &retrieved); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		// Verify DeceasedBoolean preserved
		if retrieved.DeceasedBoolean == nil {
			t.Fatal("DeceasedBoolean should be preserved")
		}

		if *retrieved.DeceasedBoolean != false {
			t.Error("DeceasedBoolean value should be preserved")
		}
	})

	t.Run("roundtrip with DeceasedDateTime", func(t *testing.T) {
		deceasedDate := primitives.MustDateTime("2020-05-10T12:00:00Z")

		original := &Patient{
			DomainResource: fhir.DomainResource{
				Resource: fhir.Resource{
					ResourceType: "Patient",
					ID:           testutil.StringPtr("roundtrip"),
				},
			},
			Active:           testutil.BoolPtr(true),
			DeceasedDateTime: &deceasedDate,
		}

		// Marshal
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		// Unmarshal
		var retrieved Patient
		if err := json.Unmarshal(data, &retrieved); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		// Verify DeceasedDateTime preserved
		if retrieved.DeceasedDateTime == nil {
			t.Fatal("DeceasedDateTime should be preserved")
		}

		if retrieved.DeceasedDateTime.String() != "2020-05-10T12:00:00Z" {
			t.Errorf("DeceasedDateTime should be preserved, got %s", retrieved.DeceasedDateTime.String())
		}
	})
}

// TestPatientMultipleBirthChoice_Serialization tests Patient.multipleBirth[x] choice type
func TestPatientMultipleBirthChoice_Serialization(t *testing.T) {
	t.Run("marshal with MultipleBirthBoolean", func(t *testing.T) {
		patient := &Patient{
			DomainResource: fhir.DomainResource{
				Resource: fhir.Resource{
					ResourceType: "Patient",
					ID:           testutil.StringPtr("twin"),
				},
			},
			MultipleBirthBoolean: testutil.BoolPtr(true),
		}

		data, err := json.Marshal(patient)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var raw map[string]interface{}
		json.Unmarshal(data, &raw)

		if _, exists := raw["multipleBirthBoolean"]; !exists {
			t.Error("Expected multipleBirthBoolean in JSON")
		}

		if _, exists := raw["multipleBirthInteger"]; exists {
			t.Error("multipleBirthInteger should not be present")
		}
	})

	t.Run("marshal with MultipleBirthInteger", func(t *testing.T) {
		birthOrder := 2

		patient := &Patient{
			DomainResource: fhir.DomainResource{
				Resource: fhir.Resource{
					ResourceType: "Patient",
					ID:           testutil.StringPtr("twin"),
				},
			},
			MultipleBirthInteger: &birthOrder,
		}

		data, err := json.Marshal(patient)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var raw map[string]interface{}
		json.Unmarshal(data, &raw)

		if _, exists := raw["multipleBirthInteger"]; !exists {
			t.Error("Expected multipleBirthInteger in JSON")
		}

		if _, exists := raw["multipleBirthBoolean"]; exists {
			t.Error("multipleBirthBoolean should not be present")
		}
	})

	t.Run("unmarshal JSON with multipleBirthBoolean", func(t *testing.T) {
		jsonData := `{
			"resourceType": "Patient",
			"id": "twin",
			"multipleBirthBoolean": true
		}`

		var patient Patient
		if err := json.Unmarshal([]byte(jsonData), &patient); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if patient.MultipleBirthBoolean == nil || !*patient.MultipleBirthBoolean {
			t.Error("MultipleBirthBoolean should be true")
		}

		if patient.MultipleBirthInteger != nil {
			t.Error("MultipleBirthInteger should be nil")
		}
	})

	t.Run("unmarshal JSON with multipleBirthInteger", func(t *testing.T) {
		jsonData := `{
			"resourceType": "Patient",
			"id": "twin",
			"multipleBirthInteger": 3
		}`

		var patient Patient
		if err := json.Unmarshal([]byte(jsonData), &patient); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if patient.MultipleBirthInteger == nil || *patient.MultipleBirthInteger != 3 {
			t.Error("MultipleBirthInteger should be 3")
		}

		if patient.MultipleBirthBoolean != nil {
			t.Error("MultipleBirthBoolean should be nil")
		}
	})
}

// TestObservationValueChoice_Serialization tests Observation.value[x] choice type
func TestObservationValueChoice_Serialization(t *testing.T) {
	t.Run("marshal with ValueString", func(t *testing.T) {
		obs := &Observation{
			DomainResource: fhir.DomainResource{
				Resource: fhir.Resource{
					ResourceType: "Observation",
					ID:           testutil.StringPtr("example"),
				},
			},
			Status: "final",
			Code: CodeableConcept{
				Text: testutil.StringPtr("Test observation"),
			},
			ValueString: testutil.StringPtr("Normal"),
		}

		data, err := json.Marshal(obs)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var raw map[string]interface{}
		json.Unmarshal(data, &raw)

		if valueString, ok := raw["valueString"].(string); !ok || valueString != "Normal" {
			t.Error("valueString should be 'Normal'")
		}

		// Verify other value[x] fields are not present
		if _, exists := raw["valueInteger"]; exists {
			t.Error("valueInteger should not be present")
		}
		if _, exists := raw["valueBoolean"]; exists {
			t.Error("valueBoolean should not be present")
		}
	})

	t.Run("unmarshal JSON with valueQuantity", func(t *testing.T) {
		jsonData := `{
			"resourceType": "Observation",
			"id": "example",
			"status": "final",
			"code": {
				"text": "Heart Rate"
			},
			"valueQuantity": {
				"value": 72,
				"unit": "beats/minute",
				"system": "http://unitsofmeasure.org",
				"code": "/min"
			}
		}`

		var obs Observation
		if err := json.Unmarshal([]byte(jsonData), &obs); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if obs.ValueQuantity == nil {
			t.Fatal("ValueQuantity should be set")
		}

		if obs.ValueQuantity.Value == nil || *obs.ValueQuantity.Value != 72 {
			t.Error("ValueQuantity.Value should be 72")
		}

		// Verify other value[x] fields are nil
		if obs.ValueString != nil {
			t.Error("ValueString should be nil")
		}
		if obs.ValueBoolean != nil {
			t.Error("ValueBoolean should be nil")
		}
	})

	t.Run("roundtrip with ValueCodeableConcept", func(t *testing.T) {
		original := &Observation{
			DomainResource: fhir.DomainResource{
				Resource: fhir.Resource{
					ResourceType: "Observation",
					ID:           testutil.StringPtr("coded"),
				},
			},
			Status: "final",
			Code: CodeableConcept{
				Text: testutil.StringPtr("Finding"),
			},
			ValueCodeableConcept: &CodeableConcept{
				Coding: []Coding{
					{
						System:  testutil.StringPtr("http://snomed.info/sct"),
						Code:    testutil.StringPtr("123456"),
						Display: testutil.StringPtr("Example finding"),
					},
				},
			},
		}

		// Marshal
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		// Unmarshal
		var retrieved Observation
		if err := json.Unmarshal(data, &retrieved); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		// Verify ValueCodeableConcept preserved
		if retrieved.ValueCodeableConcept == nil {
			t.Fatal("ValueCodeableConcept should be preserved")
		}

		if len(retrieved.ValueCodeableConcept.Coding) != 1 {
			t.Error("Coding should be preserved")
		}

		if *retrieved.ValueCodeableConcept.Coding[0].Code != "123456" {
			t.Error("Code should be preserved")
		}
	})
}
