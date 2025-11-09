package parser

import (
	"testing"

	"github.com/codeninja55/go-radx/fhir/scripts/gen/model"
)

// TestMapType tests basic type mapping from FHIR to Go types.
func TestMapType(t *testing.T) {
	tm := NewTypeMapper()

	tests := []struct {
		name     string
		fhirType string
		want     string
	}{
		// Primitive types
		{"boolean", "boolean", "bool"},
		{"string", "string", "string"},
		{"integer", "integer", "int"},
		{"integer64", "integer64", "int64"},
		{"decimal", "decimal", "float64"},
		{"unsignedInt", "unsignedInt", "uint"},
		{"positiveInt", "positiveInt", "int"},

		// String-based primitives
		{"code", "code", "string"},
		{"id", "id", "string"},
		{"uri", "uri", "string"},
		{"url", "url", "string"},
		{"canonical", "canonical", "string"},
		{"uuid", "uuid", "string"},
		{"oid", "oid", "string"},
		{"markdown", "markdown", "string"},
		{"base64Binary", "base64Binary", "string"},
		{"xhtml", "xhtml", "string"},

		// Date/time primitives
		{"date", "date", "primitives.Date"},
		{"dateTime", "dateTime", "primitives.DateTime"},
		{"instant", "instant", "primitives.Instant"},
		{"time", "time", "primitives.Time"},

		// Special types
		{"Resource", "Resource", "json.RawMessage"},

		// Complex types (passthrough)
		{"HumanName", "HumanName", "HumanName"},
		{"Reference", "Reference", "Reference"},
		{"CodeableConcept", "CodeableConcept", "CodeableConcept"},
		{"Patient", "Patient", "Patient"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tm.MapType(tt.fhirType)
			if got != tt.want {
				t.Errorf("MapType(%q) = %q, want %q", tt.fhirType, got, tt.want)
			}
		})
	}
}

// TestIsPrimitiveType tests primitive type detection.
func TestIsPrimitiveType(t *testing.T) {
	tm := NewTypeMapper()

	tests := []struct {
		name     string
		fhirType string
		want     bool
	}{
		{"boolean is primitive", "boolean", true},
		{"string is primitive", "string", true},
		{"integer is primitive", "integer", true},
		{"date is primitive", "date", true},
		{"Resource is primitive (special)", "Resource", true},
		{"HumanName is not primitive", "HumanName", false},
		{"Patient is not primitive", "Patient", false},
		{"CodeableConcept is not primitive", "CodeableConcept", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tm.IsPrimitiveType(tt.fhirType)
			if got != tt.want {
				t.Errorf("IsPrimitiveType(%q) = %v, want %v", tt.fhirType, got, tt.want)
			}
		})
	}
}

// TestMapElementToField_SingleType tests mapping a single-type element.
func TestMapElementToField_SingleType(t *testing.T) {
	tm := NewTypeMapper()

	elem := model.ElementDefinition{
		Path:  "Patient.active",
		Short: "Whether this patient record is in active use",
		Min:   0,
		Max:   "1",
		Types: []model.ElementType{
			{Code: "boolean"},
		},
		IsSummary: true,
	}

	field, err := tm.MapElementToField(elem, "Patient")
	if err != nil {
		t.Fatalf("MapElementToField() error = %v", err)
	}

	if field.Name != "Active" {
		t.Errorf("field.Name = %q, want %q", field.Name, "Active")
	}
	if field.JSONName != "active" {
		t.Errorf("field.JSONName = %q, want %q", field.JSONName, "active")
	}
	if field.GoType != "bool" {
		t.Errorf("field.GoType = %q, want %q", field.GoType, "bool")
	}
	if !field.IsPointer {
		t.Error("field.IsPointer = false, want true (min=0, max=1)")
	}
	if field.IsArray {
		t.Error("field.IsArray = true, want false")
	}
	if !field.IsSummary {
		t.Error("field.IsSummary = false, want true")
	}
}

// TestMapElementToField_ArrayType tests mapping an array element.
func TestMapElementToField_ArrayType(t *testing.T) {
	tm := NewTypeMapper()

	elem := model.ElementDefinition{
		Path:  "Patient.name",
		Short: "A name associated with the patient",
		Min:   0,
		Max:   "*",
		Types: []model.ElementType{
			{Code: "HumanName"},
		},
		IsSummary: true,
	}

	field, err := tm.MapElementToField(elem, "Patient")
	if err != nil {
		t.Fatalf("MapElementToField() error = %v", err)
	}

	if field.Name != "Name" {
		t.Errorf("field.Name = %q, want %q", field.Name, "Name")
	}
	if field.GoType != "HumanName" {
		t.Errorf("field.GoType = %q, want %q", field.GoType, "HumanName")
	}
	if !field.IsArray {
		t.Error("field.IsArray = false, want true (max=*)")
	}
	if field.IsPointer {
		t.Error("field.IsPointer = true, want false (arrays don't need pointers)")
	}
}

// TestMapElementToField_RequiredField tests mapping a required element.
func TestMapElementToField_RequiredField(t *testing.T) {
	tm := NewTypeMapper()

	elem := model.ElementDefinition{
		Path:  "Observation.status",
		Short: "Status of the observation",
		Min:   1,
		Max:   "1",
		Types: []model.ElementType{
			{Code: "code"},
		},
		IsSummary: true,
	}

	field, err := tm.MapElementToField(elem, "Observation")
	if err != nil {
		t.Fatalf("MapElementToField() error = %v", err)
	}

	if field.Name != "Status" {
		t.Errorf("field.Name = %q, want %q", field.Name, "Status")
	}
	if !field.IsRequired {
		t.Error("field.IsRequired = false, want true (min=1)")
	}
	if field.IsPointer {
		t.Error("field.IsPointer = true, want false (required fields shouldn't be pointers)")
	}
}

// TestMapElementToField_ResourceType tests json.RawMessage for Resource type.
func TestMapElementToField_ResourceType(t *testing.T) {
	tm := NewTypeMapper()

	elem := model.ElementDefinition{
		Path:  "Bundle.entry.resource",
		Short: "A resource in the bundle",
		Min:   0,
		Max:   "1",
		Types: []model.ElementType{
			{Code: "Resource"},
		},
	}

	field, err := tm.MapElementToField(elem, "Bundle.entry")
	if err != nil {
		t.Fatalf("MapElementToField() error = %v", err)
	}

	if field.GoType != "json.RawMessage" {
		t.Errorf("field.GoType = %q, want %q", field.GoType, "json.RawMessage")
	}
	// json.RawMessage should not use pointer (it's already a reference type)
	if field.IsPointer {
		t.Error("field.IsPointer = true, want false (json.RawMessage is a reference type)")
	}
}

// TestMapElementToField_NonChoicePolymorphic tests non-choice polymorphic types.
func TestMapElementToField_NonChoicePolymorphic(t *testing.T) {
	tm := NewTypeMapper()

	// Hypothetical element with multiple types but no [x] suffix
	elem := model.ElementDefinition{
		Path:  "Example.polymorphic",
		Short: "A polymorphic field",
		Min:   0,
		Max:   "1",
		Types: []model.ElementType{
			{Code: "string"},
			{Code: "integer"},
			{Code: "boolean"},
		},
	}

	field, err := tm.MapElementToField(elem, "Example")
	if err != nil {
		t.Fatalf("MapElementToField() error = %v", err)
	}

	if field.GoType != "json.RawMessage" {
		t.Errorf("field.GoType = %q, want %q for non-choice polymorphic", field.GoType, "json.RawMessage")
	}
	if field.IsPointer {
		t.Error("field.IsPointer = true, want false (json.RawMessage is a reference type)")
	}
}

// TestMapElementToField_BackboneElement tests BackboneElement handling.
func TestMapElementToField_BackboneElement(t *testing.T) {
	tm := NewTypeMapper()

	elem := model.ElementDefinition{
		Path:  "Patient.contact",
		Short: "A contact party for the patient",
		Min:   0,
		Max:   "*",
		Types: []model.ElementType{}, // Empty types means BackboneElement
	}

	field, err := tm.MapElementToField(elem, "Patient")
	if err != nil {
		t.Fatalf("MapElementToField() error = %v", err)
	}

	if field.Name != "Contact" {
		t.Errorf("field.Name = %q, want %q", field.Name, "Contact")
	}
	if field.GoType != "Contact" {
		t.Errorf("field.GoType = %q, want %q (uses field name for BackboneElement)", field.GoType, "Contact")
	}
	if !field.IsArray {
		t.Error("field.IsArray = false, want true")
	}
}

// TestMapElementToChoiceFields tests choice type expansion.
func TestMapElementToChoiceFields(t *testing.T) {
	tm := NewTypeMapper()

	elem := model.ElementDefinition{
		Path:  "Patient.deceased[x]",
		Short: "Indicates if the patient is deceased",
		Min:   0,
		Max:   "1",
		Types: []model.ElementType{
			{Code: "boolean"},
			{Code: "dateTime"},
		},
		IsSummary: true,
	}

	fields, err := tm.MapElementToChoiceFields(elem, "Patient")
	if err != nil {
		t.Fatalf("MapElementToChoiceFields() error = %v", err)
	}

	if len(fields) != 2 {
		t.Fatalf("len(fields) = %d, want 2", len(fields))
	}

	// Check boolean variant
	if fields[0].Name != "DeceasedBoolean" {
		t.Errorf("fields[0].Name = %q, want %q", fields[0].Name, "DeceasedBoolean")
	}
	if fields[0].JSONName != "deceasedBoolean" {
		t.Errorf("fields[0].JSONName = %q, want %q", fields[0].JSONName, "deceasedBoolean")
	}
	if fields[0].GoType != "bool" {
		t.Errorf("fields[0].GoType = %q, want %q", fields[0].GoType, "bool")
	}
	if fields[0].ChoiceGroup != "deceased" {
		t.Errorf("fields[0].ChoiceGroup = %q, want %q", fields[0].ChoiceGroup, "deceased")
	}
	if !fields[0].IsChoice {
		t.Error("fields[0].IsChoice = false, want true")
	}

	// Check dateTime variant
	if fields[1].Name != "DeceasedDateTime" {
		t.Errorf("fields[1].Name = %q, want %q", fields[1].Name, "DeceasedDateTime")
	}
	if fields[1].JSONName != "deceasedDateTime" {
		t.Errorf("fields[1].JSONName = %q, want %q", fields[1].JSONName, "deceasedDateTime")
	}
	if fields[1].GoType != "primitives.DateTime" {
		t.Errorf("fields[1].GoType = %q, want %q", fields[1].GoType, "primitives.DateTime")
	}
	if fields[1].ChoiceGroup != "deceased" {
		t.Errorf("fields[1].ChoiceGroup = %q, want %q", fields[1].ChoiceGroup, "deceased")
	}
}

// TestToPascalCase tests PascalCase conversion.
func TestToPascalCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "name", "Name"},
		{"camelCase", "firstName", "FirstName"},
		{"snake_case", "first_name", "FirstName"},
		{"kebab-case", "first-name", "FirstName"},
		{"dots", "meta.profile", "MetaProfile"},
		{"id special case", "id", "ID"},
		{"url special case", "url", "URL"},
		{"uri special case", "uri", "URI"},
		{"uuid special case", "uuid", "UUID"},
		{"empty", "", ""},
		{"multiple underscores", "foo__bar", "FooBar"},
		{"trailing separator", "name_", "Name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToPascalCase(tt.input)
			if got != tt.want {
				t.Errorf("ToPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestExtractFieldName tests field name extraction from paths.
func TestExtractFieldName(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		parentPath string
		want       string
	}{
		{"simple field", "Patient.active", "Patient", "active"},
		{"nested field", "Patient.contact.name", "Patient", "contact"},
		{"choice type", "Patient.deceased[x]", "Patient", "deceased"},
		{"root element", "Patient", "Patient", ""},
		{"unrelated path", "Observation.status", "Patient", ""},
		{"deeply nested", "Bundle.entry.resource.id", "Bundle.entry", "resource"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFieldName(tt.path, tt.parentPath)
			if got != tt.want {
				t.Errorf("extractFieldName(%q, %q) = %q, want %q", tt.path, tt.parentPath, got, tt.want)
			}
		})
	}
}

// TestIsBackboneElement tests BackboneElement detection.
func TestIsBackboneElement(t *testing.T) {
	tests := []struct {
		name string
		elem model.ElementDefinition
		want bool
	}{
		{
			name: "explicit BackboneElement",
			elem: model.ElementDefinition{
				Types: []model.ElementType{{Code: "BackboneElement"}},
			},
			want: true,
		},
		{
			name: "explicit Element",
			elem: model.ElementDefinition{
				Types: []model.ElementType{{Code: "Element"}},
			},
			want: true,
		},
		{
			name: "no types, no choice",
			elem: model.ElementDefinition{
				Path:  "Patient.contact",
				Types: []model.ElementType{},
			},
			want: true,
		},
		{
			name: "choice type is not backbone",
			elem: model.ElementDefinition{
				Path:  "Patient.deceased[x]",
				Types: []model.ElementType{},
			},
			want: false,
		},
		{
			name: "regular type",
			elem: model.ElementDefinition{
				Types: []model.ElementType{{Code: "string"}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsBackboneElement(tt.elem)
			if got != tt.want {
				t.Errorf("IsBackboneElement() = %v, want %v", got, tt.want)
			}
		})
	}
}
