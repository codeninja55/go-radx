package fhir

import (
	"encoding/json"
	"testing"

	"github.com/codeninja55/go-radx/fhir/primitives"
)

// TestResource is a sample resource for testing embedding
type TestResource struct {
	DomainResource
	Active *bool `json:"active,omitempty"`
}

func TestResourceEmbedding_JSONSerialization(t *testing.T) {
	// Create a resource with embedded fields
	resource := TestResource{
		DomainResource: DomainResource{
			Resource: Resource{
				ResourceType: "TestResource",
				ID:           stringPtr("123"),
				Meta: &Meta{
					VersionID: stringPtr("1"),
				},
				Language: stringPtr("en"),
			},
			Text: &Narrative{
				Status: "generated",
				Div:    "<div>Test</div>",
			},
			Extension: []Extension{
				{
					URL:         "http://example.org/ext",
					ValueString: stringPtr("test-value"),
				},
			},
		},
		Active: boolPtr(true),
	}

	// Marshal to JSON
	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Verify JSON structure
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal to map error = %v", err)
	}

	// Check that embedded fields are at the top level
	if result["resourceType"] != "TestResource" {
		t.Errorf("resourceType = %v, want TestResource", result["resourceType"])
	}
	if result["id"] != "123" {
		t.Errorf("id = %v, want 123", result["id"])
	}
	if result["language"] != "en" {
		t.Errorf("language = %v, want en", result["language"])
	}
	if result["active"] != true {
		t.Errorf("active = %v, want true", result["active"])
	}

	// Check nested embedded fields
	if _, ok := result["text"]; !ok {
		t.Error("text field missing")
	}
	if _, ok := result["extension"]; !ok {
		t.Error("extension field missing")
	}
	if _, ok := result["meta"]; !ok {
		t.Error("meta field missing")
	}

	t.Logf("JSON: %s", string(data))
}

func TestResourceEmbedding_JSONDeserialization(t *testing.T) {
	jsonData := `{
		"resourceType": "TestResource",
		"id": "456",
		"meta": {
			"versionId": "2"
		},
		"language": "es",
		"text": {
			"status": "generated",
			"div": "<div>Prueba</div>"
		},
		"extension": [{
			"url": "http://example.org/ext",
			"valueString": "test"
		}],
		"active": false
	}`

	var resource TestResource
	err := json.Unmarshal([]byte(jsonData), &resource)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Check Resource fields
	if resource.ResourceType != "TestResource" {
		t.Errorf("ResourceType = %v, want TestResource", resource.ResourceType)
	}
	if resource.ID == nil || *resource.ID != "456" {
		t.Errorf("ID = %v, want 456", resource.ID)
	}
	if resource.Language == nil || *resource.Language != "es" {
		t.Errorf("Language = %v, want es", resource.Language)
	}

	// Check DomainResource fields
	if resource.Text == nil {
		t.Fatal("Text is nil")
	}
	if resource.Text.Status != "generated" {
		t.Errorf("Text.Status = %v, want generated", resource.Text.Status)
	}
	if len(resource.Extension) != 1 {
		t.Fatalf("Extension count = %v, want 1", len(resource.Extension))
	}
	if resource.Extension[0].URL != "http://example.org/ext" {
		t.Errorf("Extension[0].URL = %v, want http://example.org/ext", resource.Extension[0].URL)
	}

	// Check resource-specific field
	if resource.Active == nil || *resource.Active != false {
		t.Errorf("Active = %v, want false", resource.Active)
	}
}

func TestResourceEmbedding_FieldAccess(t *testing.T) {
	// Test that embedded fields are directly accessible
	resource := TestResource{
		DomainResource: DomainResource{
			Resource: Resource{
				ResourceType: "TestResource",
				ID:           stringPtr("789"),
				Meta: &Meta{
					VersionID: stringPtr("1"),
				},
			},
		},
		Active: boolPtr(true),
	}

	// Direct access to Resource fields through embedding
	if resource.ResourceType != "TestResource" {
		t.Errorf("ResourceType access failed: got %v, want TestResource", resource.ResourceType)
	}
	if resource.ID == nil || *resource.ID != "789" {
		t.Errorf("ID access failed: got %v, want 789", resource.ID)
	}
	if resource.Meta == nil || resource.Meta.VersionID == nil || *resource.Meta.VersionID != "1" {
		t.Error("Meta.VersionID access failed")
	}

	// Modify embedded fields
	resource.ResourceType = "Modified"
	resource.ID = stringPtr("updated")

	if resource.ResourceType != "Modified" {
		t.Error("ResourceType modification failed")
	}
	if *resource.ID != "updated" {
		t.Error("ID modification failed")
	}

	// Access DomainResource fields
	resource.Extension = []Extension{
		{URL: "http://example.org/ext", ValueString: stringPtr("value")},
	}
	if len(resource.Extension) != 1 {
		t.Error("Extension modification failed")
	}
}

func TestResourceEmbedding_RoundTrip(t *testing.T) {
	original := TestResource{
		DomainResource: DomainResource{
			Resource: Resource{
				ResourceType:  "TestResource",
				ID:            stringPtr("round-trip"),
				Language:      stringPtr("en-US"),
				ImplicitRules: stringPtr("http://example.org/rules"),
			},
			Text: &Narrative{
				Status: "generated",
				Div:    "<div>Round trip test</div>",
			},
			Extension: []Extension{
				{
					URL:         "http://example.org/ext1",
					ValueString: stringPtr("value1"),
				},
			},
			ModifierExtension: []Extension{
				{
					URL:          "http://example.org/modifier",
					ValueBoolean: boolPtr(true),
				},
			},
		},
		Active: boolPtr(true),
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Unmarshal
	var result TestResource
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Verify all fields preserved
	if result.ResourceType != original.ResourceType {
		t.Errorf("ResourceType mismatch: got %v, want %v", result.ResourceType, original.ResourceType)
	}
	if *result.ID != *original.ID {
		t.Errorf("ID mismatch: got %v, want %v", *result.ID, *original.ID)
	}
	if *result.Language != *original.Language {
		t.Errorf("Language mismatch: got %v, want %v", *result.Language, *original.Language)
	}
	if *result.ImplicitRules != *original.ImplicitRules {
		t.Errorf("ImplicitRules mismatch: got %v, want %v", *result.ImplicitRules, *original.ImplicitRules)
	}
	if result.Text.Status != original.Text.Status {
		t.Errorf("Text.Status mismatch: got %v, want %v", result.Text.Status, original.Text.Status)
	}
	if len(result.Extension) != len(original.Extension) {
		t.Errorf("Extension count mismatch: got %v, want %v", len(result.Extension), len(original.Extension))
	}
	if len(result.ModifierExtension) != len(original.ModifierExtension) {
		t.Errorf("ModifierExtension count mismatch: got %v, want %v", len(result.ModifierExtension), len(original.ModifierExtension))
	}
	if *result.Active != *original.Active {
		t.Errorf("Active mismatch: got %v, want %v", *result.Active, *original.Active)
	}
}

func TestPrimitiveExtensions_OnBaseTypes(t *testing.T) {
	// Test that primitive extensions work on base type fields
	resource := TestResource{
		DomainResource: DomainResource{
			Resource: Resource{
				ResourceType: "TestResource",
				ID:           stringPtr("ext-test"),
				IDExt: &primitives.PrimitiveExtension{
					Extension: []primitives.Extension{
						{
							URL:         "http://example.org/id-ext",
							ValueString: stringPtr("id-extension-value"),
						},
					},
				},
				Language: stringPtr("en"),
				LanguageExt: &primitives.PrimitiveExtension{
					Extension: []primitives.Extension{
						{
							URL:         "http://example.org/lang-ext",
							ValueString: stringPtr("language-extension-value"),
						},
					},
				},
			},
		},
	}

	// Marshal
	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Verify extensions in JSON
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal to map error = %v", err)
	}

	if _, ok := result["_id"]; !ok {
		t.Error("_id extension field missing")
	}
	if _, ok := result["_language"]; !ok {
		t.Error("_language extension field missing")
	}

	// Round trip
	var parsed TestResource
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if parsed.IDExt == nil {
		t.Error("IDExt is nil after unmarshal")
	}
	if parsed.LanguageExt == nil {
		t.Error("LanguageExt is nil after unmarshal")
	}

	t.Logf("JSON with extensions: %s", string(data))
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
