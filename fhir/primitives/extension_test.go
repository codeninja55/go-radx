package primitives

import (
	"encoding/json"
	"testing"
)

func TestPrimitiveExtension_Marshal(t *testing.T) {
	// Test struct simulating FHIR resource with extension
	type TestResource struct {
		Active    *bool               `json:"active,omitempty"`
		ActiveExt *PrimitiveExtension `json:"_active,omitempty"`
	}

	tests := []struct {
		name     string
		resource TestResource
		want     string
	}{
		{
			name: "primitive without extension",
			resource: TestResource{
				Active: boolPtr(true),
			},
			want: `{"active":true}`,
		},
		{
			name: "primitive with extension",
			resource: TestResource{
				Active: boolPtr(true),
				ActiveExt: &PrimitiveExtension{
					Extension: []Extension{
						{
							URL:         "http://example.org/ext",
							ValueString: stringPtr("test"),
						},
					},
				},
			},
			want: `{"active":true,"_active":{"extension":[{"url":"http://example.org/ext","valueString":"test"}]}}`,
		},
		{
			name: "extension without value (allowed by FHIR)",
			resource: TestResource{
				ActiveExt: &PrimitiveExtension{
					Extension: []Extension{
						{
							URL:         "http://example.org/ext",
							ValueString: stringPtr("test"),
						},
					},
				},
			},
			want: `{"_active":{"extension":[{"url":"http://example.org/ext","valueString":"test"}]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.resource)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("Marshal() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestPrimitiveExtension_Unmarshal(t *testing.T) {
	type TestResource struct {
		Active    *bool               `json:"active,omitempty"`
		ActiveExt *PrimitiveExtension `json:"_active,omitempty"`
	}

	tests := []struct {
		name    string
		json    string
		want    TestResource
		wantErr bool
	}{
		{
			name: "primitive without extension",
			json: `{"active":true}`,
			want: TestResource{
				Active: boolPtr(true),
			},
		},
		{
			name: "primitive with extension",
			json: `{"active":true,"_active":{"extension":[{"url":"http://example.org/ext","valueString":"test"}]}}`,
			want: TestResource{
				Active: boolPtr(true),
				ActiveExt: &PrimitiveExtension{
					Extension: []Extension{
						{
							URL:         "http://example.org/ext",
							ValueString: stringPtr("test"),
						},
					},
				},
			},
		},
		{
			name: "extension without value",
			json: `{"_active":{"extension":[{"url":"http://example.org/ext","valueString":"test"}]}}`,
			want: TestResource{
				ActiveExt: &PrimitiveExtension{
					Extension: []Extension{
						{
							URL:         "http://example.org/ext",
							ValueString: stringPtr("test"),
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got TestResource
			err := json.Unmarshal([]byte(tt.json), &got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}

			// Compare Active
			if (got.Active == nil) != (tt.want.Active == nil) {
				t.Errorf("Active nil mismatch: got %v, want %v", got.Active, tt.want.Active)
			} else if got.Active != nil && *got.Active != *tt.want.Active {
				t.Errorf("Active = %v, want %v", *got.Active, *tt.want.Active)
			}

			// Compare ActiveExt
			if (got.ActiveExt == nil) != (tt.want.ActiveExt == nil) {
				t.Errorf("ActiveExt nil mismatch")
			} else if got.ActiveExt != nil {
				if len(got.ActiveExt.Extension) != len(tt.want.ActiveExt.Extension) {
					t.Errorf("Extension count = %v, want %v", len(got.ActiveExt.Extension), len(tt.want.ActiveExt.Extension))
				} else if len(got.ActiveExt.Extension) > 0 {
					if got.ActiveExt.Extension[0].URL != tt.want.ActiveExt.Extension[0].URL {
						t.Errorf("Extension URL = %v, want %v", got.ActiveExt.Extension[0].URL, tt.want.ActiveExt.Extension[0].URL)
					}
				}
			}
		})
	}
}

func TestPrimitiveExtension_RoundTrip(t *testing.T) {
	type TestResource struct {
		ID     *string             `json:"id,omitempty"`
		IDExt  *PrimitiveExtension `json:"_id,omitempty"`
		Active *bool               `json:"active,omitempty"`
	}

	original := TestResource{
		ID: stringPtr("123"),
		IDExt: &PrimitiveExtension{
			Extension: []Extension{
				{
					URL:         "http://example.org/ext",
					ValueString: stringPtr("id-extension"),
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

	// Verify round-trip
	if *result.ID != *original.ID {
		t.Errorf("ID = %v, want %v", *result.ID, *original.ID)
	}
	if *result.Active != *original.Active {
		t.Errorf("Active = %v, want %v", *result.Active, *original.Active)
	}
	if result.IDExt == nil {
		t.Fatal("IDExt is nil after round-trip")
	}
	if len(result.IDExt.Extension) != 1 {
		t.Fatalf("Extension count = %v, want 1", len(result.IDExt.Extension))
	}
	if result.IDExt.Extension[0].URL != "http://example.org/ext" {
		t.Errorf("Extension URL = %v, want http://example.org/ext", result.IDExt.Extension[0].URL)
	}
}

func TestPrimitiveExtension_Helpers(t *testing.T) {
	ext := &PrimitiveExtension{
		Extension: []Extension{
			{URL: "http://example.org/ext1", ValueString: stringPtr("value1")},
			{URL: "http://example.org/ext2", ValueString: stringPtr("value2")},
		},
	}

	// Test HasExtension
	if !ext.HasExtension() {
		t.Error("HasExtension() = false, want true")
	}

	emptyExt := &PrimitiveExtension{}
	if emptyExt.HasExtension() {
		t.Error("HasExtension() = true for empty, want false")
	}

	// Test GetExtensionByURL
	found := ext.GetExtensionByURL("http://example.org/ext1")
	if found == nil {
		t.Fatal("GetExtensionByURL() returned nil")
	}
	if found.URL != "http://example.org/ext1" {
		t.Errorf("GetExtensionByURL() URL = %v, want http://example.org/ext1", found.URL)
	}

	notFound := ext.GetExtensionByURL("http://example.org/not-exists")
	if notFound != nil {
		t.Error("GetExtensionByURL() should return nil for non-existent URL")
	}

	// Test AddExtension
	ext.AddExtension(Extension{
		URL:         "http://example.org/ext3",
		ValueString: stringPtr("value3"),
	})
	if len(ext.Extension) != 3 {
		t.Errorf("After AddExtension, length = %v, want 3", len(ext.Extension))
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}
