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
