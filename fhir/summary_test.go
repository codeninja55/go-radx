package fhir

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestPatient is a simplified patient structure for testing
type TestPatient struct {
	Resource
	Active              *bool   `json:"active,omitempty" fhir:"cardinality=0..1,summary"`
	ActiveExt           *string `json:"_active,omitempty" fhir:"cardinality=0..1"`
	BirthDate           *string `json:"birthDate,omitempty" fhir:"cardinality=0..1,summary"`
	BirthDateExt        *string `json:"_birthDate,omitempty" fhir:"cardinality=0..1"`
	Photo               *string `json:"photo,omitempty" fhir:"cardinality=0..1"`               // Not summary
	GeneralPractitioner *string `json:"generalPractitioner,omitempty" fhir:"cardinality=0..1"` // Not summary
}

func TestMarshalSummaryJSON(t *testing.T) {
	patient := &TestPatient{
		Resource: Resource{
			ResourceType: "Patient",
			ID:           stringPtr("example"),
			Meta: &Meta{
				VersionID: stringPtr("1"),
			},
		},
		Active:              boolPtr(true),
		BirthDate:           stringPtr("1990-01-01"),
		Photo:               stringPtr("photo-url"),
		GeneralPractitioner: stringPtr("Practitioner/123"),
	}

	// Marshal with summary mode
	data, err := MarshalSummaryJSON(patient)
	if err != nil {
		t.Fatalf("MarshalSummaryJSON() error = %v", err)
	}

	// Unmarshal to check what was included
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	// Should include summary fields
	if result["resourceType"] != "Patient" {
		t.Error("resourceType should be included")
	}
	if result["id"] != "example" {
		t.Error("id should be included")
	}
	if _, ok := result["meta"]; !ok {
		t.Error("meta should be included")
	}
	if result["active"] != true {
		t.Error("active (summary field) should be included")
	}
	if result["birthDate"] != "1990-01-01" {
		t.Error("birthDate (summary field) should be included")
	}

	// Should NOT include non-summary fields
	if _, ok := result["photo"]; ok {
		t.Error("photo (not summary) should not be included")
	}
	if _, ok := result["generalPractitioner"]; ok {
		t.Error("generalPractitioner (not summary) should not be included")
	}

	t.Logf("Summary JSON: %s", string(data))
}

func TestMarshalWithSummaryMode_All(t *testing.T) {
	patient := &TestPatient{
		Resource: Resource{
			ResourceType: "Patient",
			ID:           stringPtr("example"),
		},
		Active:    boolPtr(true),
		BirthDate: stringPtr("1990-01-01"),
		Photo:     stringPtr("photo-url"),
	}

	data, err := MarshalWithSummaryMode(patient, SummaryModeAll)
	if err != nil {
		t.Fatalf("MarshalWithSummaryMode() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	// Should include everything
	if result["active"] != true {
		t.Error("active should be included")
	}
	if result["photo"] != "photo-url" {
		t.Error("photo should be included")
	}
}

func TestMarshalWithSummaryMode_False(t *testing.T) {
	patient := &TestPatient{
		Resource: Resource{
			ResourceType: "Patient",
			ID:           stringPtr("example"),
		},
		Active:              boolPtr(true),
		BirthDate:           stringPtr("1990-01-01"),
		Photo:               stringPtr("photo-url"),
		GeneralPractitioner: stringPtr("Practitioner/123"),
	}

	data, err := MarshalWithSummaryMode(patient, SummaryModeFalse)
	if err != nil {
		t.Fatalf("MarshalWithSummaryMode() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	// Should include non-summary fields
	if result["photo"] != "photo-url" {
		t.Error("photo (not summary) should be included")
	}
	if result["generalPractitioner"] != "Practitioner/123" {
		t.Error("generalPractitioner (not summary) should be included")
	}

	// Should NOT include summary fields
	if _, ok := result["active"]; ok {
		t.Error("active (summary) should not be included")
	}
	if _, ok := result["birthDate"]; ok {
		t.Error("birthDate (summary) should not be included")
	}
}

func TestMarshalWithSummaryMode_Text(t *testing.T) {
	type TestResourceWithText struct {
		Resource
		Text      *Narrative `json:"text,omitempty" fhir:"cardinality=0..1"`
		Active    *bool      `json:"active,omitempty" fhir:"cardinality=0..1,summary"`
		BirthDate *string    `json:"birthDate,omitempty" fhir:"cardinality=0..1,summary"`
	}

	patient := &TestResourceWithText{
		Resource: Resource{
			ResourceType: "Patient",
			ID:           stringPtr("example"),
		},
		Text: &Narrative{
			Status: "generated",
			Div:    "<div>Test</div>",
		},
		Active:    boolPtr(true),
		BirthDate: stringPtr("1990-01-01"),
	}

	data, err := MarshalWithSummaryMode(patient, SummaryModeText)
	if err != nil {
		t.Fatalf("MarshalWithSummaryMode() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	// Should include text + metadata
	if _, ok := result["text"]; !ok {
		t.Error("text should be included")
	}
	if result["resourceType"] != "Patient" {
		t.Error("resourceType should be included")
	}
	if result["id"] != "example" {
		t.Error("id should be included")
	}

	// Should NOT include other fields
	if _, ok := result["active"]; ok {
		t.Error("active should not be included in text mode")
	}
	if _, ok := result["birthDate"]; ok {
		t.Error("birthDate should not be included in text mode")
	}
}

func TestMarshalWithSummaryMode_Data(t *testing.T) {
	type TestResourceWithText struct {
		Resource
		Text      *Narrative `json:"text,omitempty" fhir:"cardinality=0..1"`
		Active    *bool      `json:"active,omitempty" fhir:"cardinality=0..1,summary"`
		BirthDate *string    `json:"birthDate,omitempty" fhir:"cardinality=0..1,summary"`
	}

	patient := &TestResourceWithText{
		Resource: Resource{
			ResourceType: "Patient",
			ID:           stringPtr("example"),
		},
		Text: &Narrative{
			Status: "generated",
			Div:    "<div>Test</div>",
		},
		Active:    boolPtr(true),
		BirthDate: stringPtr("1990-01-01"),
	}

	data, err := MarshalWithSummaryMode(patient, SummaryModeData)
	if err != nil {
		t.Fatalf("MarshalWithSummaryMode() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	// Should include data fields
	if result["active"] != true {
		t.Error("active should be included")
	}
	if result["birthDate"] != "1990-01-01" {
		t.Error("birthDate should be included")
	}

	// Should NOT include text
	if _, ok := result["text"]; ok {
		t.Error("text should not be included in data mode")
	}
}

func TestGetSummaryFields(t *testing.T) {
	patient := &TestPatient{}

	fields := GetSummaryFields(patient)

	// Should include summary field names
	expectedFields := []string{"active", "birthDate"}

	if len(fields) < 2 {
		t.Errorf("Expected at least 2 summary fields, got %d: %v", len(fields), fields)
	}

	// Check that expected fields are in the result
	for _, expected := range expectedFields {
		found := false
		for _, field := range fields {
			if field == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected summary field %q not found in: %v", expected, fields)
		}
	}

	// Should NOT include non-summary fields
	for _, field := range fields {
		if field == "photo" || field == "generalPractitioner" {
			t.Errorf("Non-summary field %q should not be in summary fields list", field)
		}
	}
}

func TestSummary_WithEmbeddedStruct(t *testing.T) {
	type TestEmbedded struct {
		DomainResource
		Active    *bool   `json:"active,omitempty" fhir:"summary"`
		BirthDate *string `json:"birthDate,omitempty" fhir:"summary"`
		Photo     *string `json:"photo,omitempty"`
	}

	patient := &TestEmbedded{
		DomainResource: DomainResource{
			Resource: Resource{
				ResourceType: "Patient",
				ID:           stringPtr("example"),
			},
			Text: &Narrative{
				Status: "generated",
				Div:    "<div>Test</div>",
			},
		},
		Active:    boolPtr(true),
		BirthDate: stringPtr("1990-01-01"),
		Photo:     stringPtr("photo-url"),
	}

	data, err := MarshalSummaryJSON(patient)
	if err != nil {
		t.Fatalf("MarshalSummaryJSON() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	// Should include embedded Resource fields
	if result["resourceType"] != "Patient" {
		t.Error("resourceType from embedded Resource should be included")
	}
	if result["id"] != "example" {
		t.Error("id from embedded Resource should be included")
	}

	// Should include summary fields
	if result["active"] != true {
		t.Error("active (summary field) should be included")
	}

	// Should NOT include non-summary fields
	if _, ok := result["photo"]; ok {
		t.Error("photo (not summary) should not be included")
	}
}

func TestSummary_NilValues(t *testing.T) {
	patient := &TestPatient{
		Resource: Resource{
			ResourceType: "Patient",
			ID:           stringPtr("example"),
		},
		// All optional fields are nil
	}

	data, err := MarshalSummaryJSON(patient)
	if err != nil {
		t.Fatalf("MarshalSummaryJSON() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	// Should include required fields
	if result["resourceType"] != "Patient" {
		t.Error("resourceType should be included")
	}

	// Nil optional fields should not be included (omitempty)
	if _, ok := result["active"]; ok {
		t.Error("nil active field should be omitted")
	}
	if _, ok := result["birthDate"]; ok {
		t.Error("nil birthDate field should be omitted")
	}
}

func TestSummary_Comparison(t *testing.T) {
	patient := &TestPatient{
		Resource: Resource{
			ResourceType: "Patient",
			ID:           stringPtr("example"),
		},
		Active:              boolPtr(true),
		BirthDate:           stringPtr("1990-01-01"),
		Photo:               stringPtr("photo-url"),
		GeneralPractitioner: stringPtr("Practitioner/123"),
	}

	// Full JSON
	fullData, _ := json.Marshal(patient)

	// Summary JSON
	summaryData, _ := MarshalSummaryJSON(patient)

	// Summary should be smaller
	if len(summaryData) >= len(fullData) {
		t.Errorf("Summary JSON (%d bytes) should be smaller than full JSON (%d bytes)",
			len(summaryData), len(fullData))
	}

	// Summary should contain summary fields
	summaryStr := string(summaryData)
	if !strings.Contains(summaryStr, "active") {
		t.Error("Summary should contain 'active' field")
	}
	if !strings.Contains(summaryStr, "birthDate") {
		t.Error("Summary should contain 'birthDate' field")
	}

	// Summary should NOT contain non-summary fields
	if strings.Contains(summaryStr, "photo") {
		t.Error("Summary should not contain 'photo' field")
	}
	if strings.Contains(summaryStr, "generalPractitioner") {
		t.Error("Summary should not contain 'generalPractitioner' field")
	}

	t.Logf("Full JSON size: %d bytes", len(fullData))
	t.Logf("Summary JSON size: %d bytes", len(summaryData))
	t.Logf("Reduction: %.1f%%", float64(len(fullData)-len(summaryData))/float64(len(fullData))*100)
}
