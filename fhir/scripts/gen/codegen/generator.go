package codegen

import (
	"bytes"
	"fmt"
	"go/format"
	"sort"
	"strings"
	"text/template"

	"github.com/codeninja55/go-radx/fhir/scripts/gen/model"
)

// Generator generates Go code from FHIR StructureDefinitions.
type Generator struct {
	packageName string
}

// New creates a new code generator.
func New(packageName string) *Generator {
	return &Generator{
		packageName: packageName,
	}
}

// GenerateResource generates Go code for a FHIR resource.
func (g *Generator) GenerateResource(def *model.StructureDefinition, fields []model.Field) (string, error) {
	// Build type definition
	typeDef := model.TypeDefinition{
		Name:       def.Name,
		Kind:       def.Kind,
		Comment:    def.Type,
		Fields:     fields,
		IsAbstract: def.Abstract,
	}

	return g.generateType(typeDef)
}

// GenerateComplexType generates Go code for a FHIR complex type.
func (g *Generator) GenerateComplexType(def *model.StructureDefinition, fields []model.Field) (string, error) {
	typeDef := model.TypeDefinition{
		Name:       def.Name,
		Kind:       "complex",
		Comment:    def.Type,
		Fields:     fields,
		IsAbstract: def.Abstract,
	}

	return g.generateType(typeDef)
}

// generateType generates Go code for a type definition.
func (g *Generator) generateType(typeDef model.TypeDefinition) (string, error) {
	var buf bytes.Buffer

	// Write package declaration
	fmt.Fprintf(&buf, "package %s\n\n", g.packageName)

	// Write imports if needed
	if needsPrimitivesImport(typeDef.Fields) {
		fmt.Fprintf(&buf, "import \"github.com/codeninja55/go-radx/fhir/primitives\"\n\n")
	}

	// Write type comment
	if typeDef.Comment != "" {
		fmt.Fprintf(&buf, "// %s represents a FHIR %s.\n", typeDef.Name, typeDef.Comment)
	}

	// Write type declaration
	fmt.Fprintf(&buf, "type %s struct {\n", typeDef.Name)

	// Sort fields by name for consistent output
	sortedFields := make([]model.Field, len(typeDef.Fields))
	copy(sortedFields, typeDef.Fields)
	sort.Slice(sortedFields, func(i, j int) bool {
		return sortedFields[i].Name < sortedFields[j].Name
	})

	// Write fields
	for _, field := range sortedFields {
		g.writeField(&buf, field)
	}

	fmt.Fprintf(&buf, "}\n")

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Return unformatted code with error for debugging
		return buf.String(), fmt.Errorf("format code: %w", err)
	}

	return string(formatted), nil
}

// writeField writes a single struct field.
func (g *Generator) writeField(buf *bytes.Buffer, field model.Field) {
	// Handle embedded fields (no JSON tag, no field name)
	if field.IsEmbedded {
		if field.Comment != "" {
			comment := formatComment(field.Comment)
			fmt.Fprintf(buf, "\t// %s\n", comment)
		}
		fmt.Fprintf(buf, "\t%s\n", field.GoType)
		return
	}

	// Write field comment
	if field.Comment != "" {
		comment := formatComment(field.Comment)
		fmt.Fprintf(buf, "\t// %s\n", comment)
	}

	// Build field type string
	fieldType := buildFieldType(field)

	// Write field declaration with JSON tag
	jsonTag := buildJSONTag(field)
	fmt.Fprintf(buf, "\t%s %s `json:\"%s\"`\n", field.Name, fieldType, jsonTag)
}

// buildFieldType constructs the Go type string for a field.
func buildFieldType(field model.Field) string {
	baseType := field.GoType

	// Handle arrays
	if field.IsArray {
		baseType = "[]" + baseType
	}

	// Handle pointers (optional fields)
	if field.IsPointer && !field.IsArray {
		baseType = "*" + baseType
	}

	return baseType
}

// buildJSONTag constructs the JSON struct tag for a field.
func buildJSONTag(field model.Field) string {
	tag := field.JSONName

	// Add omitempty for optional fields
	if field.IsPointer || field.IsArray {
		tag += ",omitempty"
	}

	return tag
}

// formatComment formats a comment string for Go code.
func formatComment(comment string) string {
	comment = strings.TrimSpace(comment)
	if !strings.HasSuffix(comment, ".") {
		comment += "."
	}
	return comment
}

// FieldWithTags wraps a Field with pre-computed struct tags.
type FieldWithTags struct {
	model.Field
	FHIRTag string
}

// GenerateFile generates a complete Go source file with multiple types.
func (g *Generator) GenerateFile(types []model.TypeDefinition) (string, error) {
	// Check if any type needs imports
	needsPrimitives := false
	needsJSON := false
	needsFHIR := false
	for _, t := range types {
		if needsPrimitivesImport(t.Fields) {
			needsPrimitives = true
		}
		if needsJSONImport(t.Fields) {
			needsJSON = true
		}
		if needsFHIRImport(t.Fields) {
			needsFHIR = true
		}
		if needsPrimitives && needsJSON && needsFHIR {
			break
		}
	}

	// Check if this is a resource file (generate ResourceType constant)
	var resourceType string
	for _, t := range types {
		if t.Kind == "resource" {
			resourceType = t.Name
			break
		}
	}

	// Convert types to include pre-computed FHIR tags
	type TypeWithTags struct {
		Name    string
		Comment string
		Kind    string
		Fields  []FieldWithTags
	}

	typesWithTags := make([]TypeWithTags, len(types))
	for i, t := range types {
		fieldsWithTags := make([]FieldWithTags, len(t.Fields))
		for j, f := range t.Fields {
			fieldsWithTags[j] = FieldWithTags{
				Field:   f,
				FHIRTag: f.FHIRTag(),
			}
		}
		typesWithTags[i] = TypeWithTags{
			Name:    t.Name,
			Comment: t.Comment,
			Kind:    t.Kind,
			Fields:  fieldsWithTags,
		}
	}

	tmpl := template.Must(template.New("file").Parse(fileTemplate))

	data := struct {
		Package          string
		Types            []TypeWithTags
		NeedsPrimitives  bool
		NeedsJSON        bool
		NeedsFHIR        bool
		ResourceType     string
		FHIRVersion      string
	}{
		Package:          g.packageName,
		Types:            typesWithTags,
		NeedsPrimitives:  needsPrimitives,
		NeedsJSON:        needsJSON,
		NeedsFHIR:        needsFHIR,
		ResourceType:     resourceType,
		FHIRVersion:      "R5",
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return buf.String(), fmt.Errorf("format code: %w", err)
	}

	return string(formatted), nil
}

// needsPrimitivesImport checks if any field uses primitives types.
func needsPrimitivesImport(fields []model.Field) bool {
	for _, field := range fields {
		if strings.HasPrefix(field.GoType, "primitives.") {
			return true
		}
	}
	return false
}

// needsJSONImport checks if any field uses json.RawMessage.
func needsJSONImport(fields []model.Field) bool {
	for _, field := range fields {
		if field.GoType == "json.RawMessage" {
			return true
		}
	}
	return false
}

// needsFHIRImport checks if any field uses fhir package types (e.g., fhir.DomainResource).
func needsFHIRImport(fields []model.Field) bool {
	for _, field := range fields {
		if strings.HasPrefix(field.GoType, "fhir.") {
			return true
		}
	}
	return false
}

const fileTemplate = `// Code generated by fhirgen. DO NOT EDIT.
// FHIR Version: {{.FHIRVersion}}
// Source: FHIR StructureDefinitions from https://hl7.org/fhir/{{.FHIRVersion}}/

package {{.Package}}
{{if or .NeedsPrimitives .NeedsJSON .NeedsFHIR}}
import (
{{- if .NeedsJSON}}
	"encoding/json"
{{- end}}
{{- if .NeedsFHIR}}

	"github.com/codeninja55/go-radx/fhir"
{{- end}}
{{- if .NeedsPrimitives}}
	"github.com/codeninja55/go-radx/fhir/primitives"
{{- end}}
)
{{end}}
{{if .ResourceType}}
// ResourceType{{.ResourceType}} is the FHIR resource type name for {{.ResourceType}}.
const ResourceType{{.ResourceType}} = "{{.ResourceType}}"
{{end}}
{{range .Types}}
// {{.Name}} represents a FHIR {{.Comment}}.
type {{.Name}} struct {
{{- range .Fields}}
{{- if .IsEmbedded}}
	{{if .Comment}}// {{.Comment}}{{end}}
	{{.GoType}}
{{- else}}
	{{if .Comment}}// {{.Comment}}{{end}}
	{{.Name}} {{if .IsArray}}[]{{end}}{{if .IsPointer}}*{{end}}{{.GoType}} ` + "`json:\"{{.JSONName}}{{if or .IsPointer .IsArray}},omitempty{{end}}\"{{if .FHIRTag}} {{.FHIRTag}}{{end}}`" + `
{{- end}}
{{- end}}
}
{{end}}
`
