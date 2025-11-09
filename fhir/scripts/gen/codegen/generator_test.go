package codegen

import (
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir/scripts/gen/model"
)

func TestGenerator_GenerateFile(t *testing.T) {
	gen := New("resources")

	types := []model.TypeDefinition{
		{
			Name: "PatientContact",
			Kind: "backbone",
			Fields: []model.Field{
				{
					Name:      "Name",
					GoType:    "string",
					JSONName:  "name",
					Min:       0,
					Max:       "1",
					IsPointer: true,
				},
			},
		},
		{
			Name: "Patient",
			Kind: "resource",
			Fields: []model.Field{
				{
					Name:      "ID",
					GoType:    "string",
					JSONName:  "id",
					Min:       0,
					Max:       "1",
					IsPointer: true,
				},
				{
					Name:     "Contact",
					GoType:   "PatientContact",
					JSONName: "contact",
					Min:      0,
					Max:      "*",
					IsArray:  true,
				},
			},
		},
	}

	code, err := gen.GenerateFile(types)
	if err != nil {
		t.Fatalf("GenerateFile() error = %v", err)
	}

	// Check package declaration
	if !strings.Contains(code, "package resources") {
		t.Error("Generated file should have correct package")
	}

	// Check resource type constant
	if !strings.Contains(code, "const ResourceTypePatient") {
		t.Error("Generated file should have resource type constant")
	}

	// Check both types are generated
	if !strings.Contains(code, "type PatientContact struct") {
		t.Error("Generated file should contain BackboneElement type")
	}
	if !strings.Contains(code, "type Patient struct") {
		t.Error("Generated file should contain main resource type")
	}

	// BackboneElement should come before main type
	contactIdx := strings.Index(code, "type PatientContact")
	patientIdx := strings.Index(code, "type Patient struct")
	if contactIdx > patientIdx {
		t.Error("BackboneElement should be defined before main type")
	}
}

func TestGenerator_NeedsPrimitivesImport(t *testing.T) {
	tests := []struct {
		name   string
		fields []model.Field
		want   bool
	}{
		{
			name: "with primitives import",
			fields: []model.Field{
				{GoType: "primitives.Date"},
			},
			want: true,
		},
		{
			name: "without primitives import",
			fields: []model.Field{
				{GoType: "string"},
				{GoType: "int"},
			},
			want: false,
		},
		{
			name: "with primitive extension",
			fields: []model.Field{
				{GoType: "primitives.PrimitiveExtension"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsPrimitivesImport(tt.fields)
			if got != tt.want {
				t.Errorf("needsPrimitivesImport() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestField_FHIRTag(t *testing.T) {
	tests := []struct {
		name  string
		field model.Field
		want  string
	}{
		{
			name: "simple cardinality",
			field: model.Field{
				Min: 0,
				Max: "1",
			},
			want: `fhir:"cardinality=0..1"`,
		},
		{
			name: "required field",
			field: model.Field{
				Min:        1,
				Max:        "1",
				IsRequired: true,
			},
			want: `fhir:"cardinality=1..1,required"`,
		},
		{
			name: "with enum",
			field: model.Field{
				Min:        0,
				Max:        "1",
				EnumValues: []string{"male", "female"},
			},
			want: `fhir:"cardinality=0..1,enum=male|female"`,
		},
		{
			name: "with summary",
			field: model.Field{
				Min:       0,
				Max:       "1",
				IsSummary: true,
			},
			want: `fhir:"cardinality=0..1,summary"`,
		},
		{
			name: "with choice",
			field: model.Field{
				Min:         0,
				Max:         "1",
				ChoiceGroup: "deceased",
			},
			want: `fhir:"cardinality=0..1,choice=deceased"`,
		},
		{
			name: "all features",
			field: model.Field{
				Min:         1,
				Max:         "*",
				IsRequired:  true,
				EnumValues:  []string{"a", "b"},
				IsSummary:   true,
				ChoiceGroup: "value",
			},
			want: `fhir:"cardinality=1..*,required,enum=a|b,summary,choice=value"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.field.FHIRTag()
			if got != tt.want {
				t.Errorf("FHIRTag() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerator_NeedsJSONImport(t *testing.T) {
	tests := []struct {
		name   string
		fields []model.Field
		want   bool
	}{
		{
			name: "with json.RawMessage",
			fields: []model.Field{
				{GoType: "json.RawMessage"},
			},
			want: true,
		},
		{
			name: "without json.RawMessage",
			fields: []model.Field{
				{GoType: "string"},
				{GoType: "primitives.Date"},
			},
			want: false,
		},
		{
			name: "multiple fields with json.RawMessage",
			fields: []model.Field{
				{GoType: "string"},
				{GoType: "json.RawMessage"},
				{GoType: "int"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsJSONImport(tt.fields)
			if got != tt.want {
				t.Errorf("needsJSONImport() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerator_NeedsFHIRImport(t *testing.T) {
	tests := []struct {
		name   string
		fields []model.Field
		want   bool
	}{
		{
			name: "with fhir.DomainResource",
			fields: []model.Field{
				{GoType: "fhir.DomainResource", IsEmbedded: true},
			},
			want: true,
		},
		{
			name: "without fhir import",
			fields: []model.Field{
				{GoType: "string"},
				{GoType: "primitives.Date"},
			},
			want: false,
		},
		{
			name: "with fhir.Resource",
			fields: []model.Field{
				{GoType: "fhir.Resource", IsEmbedded: true},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsFHIRImport(tt.fields)
			if got != tt.want {
				t.Errorf("needsFHIRImport() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerator_GeneratedCodeHeaders(t *testing.T) {
	gen := New("resources")

	types := []model.TypeDefinition{
		{
			Name: "Patient",
			Kind: "resource",
			Fields: []model.Field{
				{
					Name:      "ID",
					GoType:    "string",
					JSONName:  "id",
					Min:       0,
					Max:       "1",
					IsPointer: true,
				},
			},
		},
	}

	code, err := gen.GenerateFile(types)
	if err != nil {
		t.Fatalf("GenerateFile() error = %v", err)
	}

	// Check for generated code warning
	if !strings.Contains(code, "Code generated by fhirgen") {
		t.Error("Generated file should have 'Code generated by fhirgen' warning")
	}

	if !strings.Contains(code, "DO NOT EDIT") {
		t.Error("Generated file should have 'DO NOT EDIT' warning")
	}

	// Check for FHIR version
	if !strings.Contains(code, "FHIR Version: R5") {
		t.Error("Generated file should have FHIR version")
	}

	// Check for source URL
	if !strings.Contains(code, "Source: FHIR StructureDefinitions from https://hl7.org/fhir/R5/") {
		t.Error("Generated file should have source URL")
	}

	// Check for timestamp (Generated at: format)
	if !strings.Contains(code, "Generated at:") {
		t.Error("Generated file should have timestamp")
	}

	// Check for generator version
	if !strings.Contains(code, "fhirgen v") {
		t.Error("Generated file should have generator version")
	}
}

func TestGenerator_ChoiceTypeGeneration(t *testing.T) {
	gen := New("resources")

	types := []model.TypeDefinition{
		{
			Name: "Patient",
			Kind: "resource",
			Fields: []model.Field{
				{
					Name:         "DeceasedBoolean",
					GoType:       "bool",
					JSONName:     "deceasedBoolean",
					Min:          0,
					Max:          "1",
					IsPointer:    true,
					IsChoice:     true,
					ChoiceGroup:  "deceased",
					ChoiceSuffix: "Boolean",
				},
				{
					Name:         "DeceasedDateTime",
					GoType:       "primitives.DateTime",
					JSONName:     "deceasedDateTime",
					Min:          0,
					Max:          "1",
					IsPointer:    true,
					IsChoice:     true,
					ChoiceGroup:  "deceased",
					ChoiceSuffix: "DateTime",
				},
			},
		},
	}

	code, err := gen.GenerateFile(types)
	if err != nil {
		t.Fatalf("GenerateFile() error = %v", err)
	}

	// Check both choice fields are generated
	if !strings.Contains(code, "DeceasedBoolean") {
		t.Error("Generated file should contain DeceasedBoolean field")
	}

	if !strings.Contains(code, "DeceasedDateTime") {
		t.Error("Generated file should contain DeceasedDateTime field")
	}

	// Check choice group tags
	if !strings.Contains(code, "choice=deceased") {
		t.Error("Generated file should have choice=deceased tag")
	}

	// Both fields should have same choice group
	deceasedCount := strings.Count(code, "choice=deceased")
	if deceasedCount != 2 {
		t.Errorf("Expected 2 fields with choice=deceased, got %d", deceasedCount)
	}
}

func TestGenerator_JSONRawMessageField(t *testing.T) {
	gen := New("resources")

	types := []model.TypeDefinition{
		{
			Name: "Bundle",
			Kind: "resource",
			Fields: []model.Field{
				{
					Name:     "Resource",
					GoType:   "json.RawMessage",
					JSONName: "resource",
					Min:      0,
					Max:      "1",
					// json.RawMessage should not be pointer
					IsPointer: false,
				},
			},
		},
	}

	code, err := gen.GenerateFile(types)
	if err != nil {
		t.Fatalf("GenerateFile() error = %v", err)
	}

	// Check json.RawMessage field is generated correctly
	if !strings.Contains(code, "Resource json.RawMessage") {
		t.Error("Generated file should contain json.RawMessage field without pointer")
	}

	// Check encoding/json import is present
	if !strings.Contains(code, `"encoding/json"`) {
		t.Error("Generated file should import encoding/json for json.RawMessage")
	}
}

func TestGenerator_StructEmbedding(t *testing.T) {
	gen := New("resources")

	types := []model.TypeDefinition{
		{
			Name: "Patient",
			Kind: "resource",
			Fields: []model.Field{
				{
					GoType:     "fhir.DomainResource",
					IsEmbedded: true,
					Comment:    "Base resource fields",
				},
				{
					Name:      "Active",
					GoType:    "bool",
					JSONName:  "active",
					Min:       0,
					Max:       "1",
					IsPointer: true,
				},
			},
		},
	}

	code, err := gen.GenerateFile(types)
	if err != nil {
		t.Fatalf("GenerateFile() error = %v", err)
	}

	// Check embedded field is generated (no field name, no JSON tag)
	if !strings.Contains(code, "fhir.DomainResource") {
		t.Error("Generated file should contain embedded fhir.DomainResource")
	}

	// Check fhir import is present
	if !strings.Contains(code, `"github.com/codeninja55/go-radx/fhir"`) {
		t.Error("Generated file should import fhir package for struct embedding")
	}

	// Embedded field should not have JSON tag
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if strings.Contains(line, "fhir.DomainResource") {
			// Embedded fields don't have JSON tags
			if strings.Contains(line, "json:") {
				t.Error("Embedded field should not have JSON tag")
			}
			// But should be followed by regular fields with JSON tags
			if i+1 < len(lines) && strings.Contains(lines[i+1], "Active") {
				if !strings.Contains(lines[i+1], "json:") {
					t.Error("Regular fields should have JSON tags")
				}
			}
		}
	}
}
