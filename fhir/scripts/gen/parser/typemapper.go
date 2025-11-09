package parser

import (
	"fmt"
	"strings"

	"github.com/codeninja55/go-radx/fhir/scripts/gen/model"
)

// TypeMapper maps FHIR types to Go types.
type TypeMapper struct {
	// Map of FHIR primitive type codes to Go types
	primitiveMap map[string]string
}

// NewTypeMapper creates a new type mapper.
func NewTypeMapper() *TypeMapper {
	return &TypeMapper{
		primitiveMap: map[string]string{
			// FHIR primitive types
			"base64Binary": "string",
			"boolean":      "bool",
			"canonical":    "string",
			"code":         "string",
			"date":         "primitives.Date",
			"dateTime":     "primitives.DateTime",
			"decimal":      "float64",
			"id":           "string",
			"instant":      "primitives.Instant",
			"integer":      "int",
			"integer64":    "int64", // R5 new type
			"markdown":     "string",
			"oid":          "string",
			"positiveInt":  "int",
			"string":       "string",
			"time":         "primitives.Time",
			"unsignedInt":  "uint",
			"uri":          "string",
			"url":          "string",
			"uuid":         "string",
			"xhtml":        "string",
			// Special FHIR types
			"http://hl7.org/fhirpath/System.String": "string",
			"Resource":                              "json.RawMessage", // polymorphic - use lazy deserialization
		},
	}
}

// MapType converts a FHIR type to a Go type.
func (tm *TypeMapper) MapType(fhirType string) string {
	// Check if it's a primitive type
	if goType, ok := tm.primitiveMap[fhirType]; ok {
		return goType
	}

	// Complex types and resources use their name as-is
	// e.g., "HumanName", "Patient", "Reference"
	return fhirType
}

// IsPrimitiveType returns true if the type is a FHIR primitive.
func (tm *TypeMapper) IsPrimitiveType(fhirType string) bool {
	_, ok := tm.primitiveMap[fhirType]
	return ok
}

// MapElementToField converts an ElementDefinition to a Field for code generation.
func (tm *TypeMapper) MapElementToField(elem model.ElementDefinition, parentPath string) (*model.Field, error) {
	// Extract field name from path
	fieldName := extractFieldName(elem.Path, parentPath)
	if fieldName == "" {
		return nil, nil // Skip root element
	}

	field := &model.Field{
		JSONName:   fieldName,
		Min:        elem.Min,
		Max:        elem.Max,
		Comment:    elem.Short,
		IsSummary:  elem.IsSummary,
		IsRequired: elem.Min >= 1,
	}

	// Convert to PascalCase for Go field name
	field.Name = ToPascalCase(fieldName)

	// Check if it's a choice type
	if IsChoiceType(elem.Path) {
		field.IsChoice = true
		baseName := GetChoiceBaseName(elem.Path)
		field.Name = ToPascalCase(baseName)
		field.ChoiceGroup = baseName // Set choice group for validation
	}

	// Determine if field is an array
	field.IsArray = elem.Max == "*"

	// Map type(s) first to determine the Go type
	var goType string
	if len(elem.Types) == 0 {
		// BackboneElement - will be generated as nested struct
		goType = field.Name
	} else if len(elem.Types) == 1 {
		// Single type
		goType = tm.MapType(elem.Types[0].Code)
	} else if IsChoiceType(elem.Path) {
		// Choice type - will be expanded to multiple fields in builder
		goType = tm.MapType(elem.Types[0].Code)
	} else {
		// Multiple types - use interface{}
		goType = "any"
	}

	// Determine if field is optional (pointer)
	// json.RawMessage should not use pointers since it's already a reference type ([]byte)
	field.IsPointer = elem.Min == 0 && elem.Max == "1" && goType != "json.RawMessage"

	// Extract enum values from binding if present
	if elem.Binding != nil && elem.Binding.Strength == "required" {
		// For required bindings, we could extract enum values
		// This would require parsing the ValueSet, which is complex
		// For now, we'll mark the field but not populate EnumValues
		// TODO: Implement ValueSet parsing for enum validation
	}

	// Set the Go type that we already determined above
	field.GoType = goType

	// For choice types, set the choice suffix
	if field.IsChoice {
		field.ChoiceSuffix = ToPascalCase(elem.Types[0].Code)
	}

	return field, nil
}

// MapElementToChoiceFields converts a choice type element to multiple fields, one per type.
func (tm *TypeMapper) MapElementToChoiceFields(elem model.ElementDefinition, parentPath string) ([]*model.Field, error) {
	if !IsChoiceType(elem.Path) {
		return nil, fmt.Errorf("element %s is not a choice type", elem.Path)
	}

	baseName := GetChoiceBaseName(elem.Path)
	choiceGroup := baseName

	var fields []*model.Field

	for _, typeInfo := range elem.Types {
		field := &model.Field{
			JSONName:     baseName + ToPascalCase(typeInfo.Code),
			Min:          elem.Min,
			Max:          elem.Max,
			Comment:      elem.Short + " - " + typeInfo.Code + " option",
			IsSummary:    elem.IsSummary,
			IsRequired:   elem.Min >= 1,
			IsChoice:     true,
			ChoiceGroup:  choiceGroup,
			ChoiceSuffix: ToPascalCase(typeInfo.Code),
			Name:         ToPascalCase(baseName) + ToPascalCase(typeInfo.Code),
			GoType:       tm.MapType(typeInfo.Code),
			IsPointer:    elem.Min == 0,
			IsArray:      elem.Max == "*",
		}

		fields = append(fields, field)
	}

	return fields, nil
}

// extractFieldName extracts the field name from a path.
// e.g., "Patient.name" with parentPath "Patient" returns "name"
func extractFieldName(path, parentPath string) string {
	if path == parentPath {
		return "" // Root element
	}

	// Remove parent path prefix
	if !strings.HasPrefix(path, parentPath+".") {
		return ""
	}

	remainder := strings.TrimPrefix(path, parentPath+".")

	// Get first segment (in case of nested paths like "Patient.contact.name")
	parts := strings.Split(remainder, ".")
	if len(parts) == 0 {
		return ""
	}

	fieldName := parts[0]

	// Handle choice types
	if strings.HasSuffix(fieldName, "[x]") {
		return strings.TrimSuffix(fieldName, "[x]")
	}

	return fieldName
}

// ToPascalCase converts a string to PascalCase for Go identifiers.
func ToPascalCase(s string) string {
	if s == "" {
		return ""
	}

	// Handle special cases
	switch s {
	case "id":
		return "ID"
	case "url":
		return "URL"
	case "uri":
		return "URI"
	case "uuid":
		return "UUID"
	}

	// Split on common delimiters
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == '.'
	})

	var result strings.Builder
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		// Capitalize first letter, keep rest as-is
		result.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			result.WriteString(part[1:])
		}
	}

	return result.String()
}

// IsBackboneElement checks if an element is a BackboneElement.
func IsBackboneElement(elem model.ElementDefinition) bool {
	for _, t := range elem.Types {
		if t.Code == "BackboneElement" || t.Code == "Element" {
			return true
		}
	}
	return len(elem.Types) == 0 && !strings.HasSuffix(elem.Path, "[x]")
}

// GetDirectChildren returns elements that are direct children of the given path.
func GetDirectChildren(elements []model.ElementDefinition, parentPath string) []model.ElementDefinition {
	var children []model.ElementDefinition

	for _, elem := range elements {
		if elem.Path == parentPath {
			continue // Skip parent itself
		}

		if !strings.HasPrefix(elem.Path, parentPath+".") {
			continue // Not a descendant
		}

		// Check if it's a direct child (not a grandchild)
		remainder := strings.TrimPrefix(elem.Path, parentPath+".")
		if !strings.Contains(remainder, ".") {
			children = append(children, elem)
		}
	}

	return children
}

// FormatComment formats a comment string for Go code.
func FormatComment(comment string) string {
	if comment == "" {
		return ""
	}

	// Trim and ensure it ends with a period
	comment = strings.TrimSpace(comment)
	if !strings.HasSuffix(comment, ".") {
		comment += "."
	}

	return comment
}
