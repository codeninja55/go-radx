package codegen

import (
	"fmt"
	"log"
	"strings"

	"github.com/codeninja55/go-radx/fhir/scripts/gen/model"
	"github.com/codeninja55/go-radx/fhir/scripts/gen/parser"
)

// Builder builds Go types from FHIR StructureDefinitions.
type Builder struct {
	parser         *parser.Parser
	typeMapper     *parser.TypeMapper
	generator      *Generator
	verbose        bool
	resourceFilter map[string]bool // Set of resource names to generate (nil = all)
}

// NewBuilder creates a new type builder.
func NewBuilder(p *parser.Parser, packageName string, verbose bool) *Builder {
	return &Builder{
		parser:     p,
		typeMapper: parser.NewTypeMapper(),
		generator:  New(packageName),
		verbose:    verbose,
	}
}

// SetResourceFilter sets a filter for which resources to generate.
// If the filter is empty or nil, all resources will be generated.
func (b *Builder) SetResourceFilter(resources []string) {
	if len(resources) == 0 {
		b.resourceFilter = nil
		return
	}

	b.resourceFilter = make(map[string]bool)
	for _, name := range resources {
		b.resourceFilter[name] = true
	}
}

// shouldGenerateResource returns true if the resource should be generated based on the filter.
func (b *Builder) shouldGenerateResource(name string) bool {
	if b.resourceFilter == nil {
		return true
	}
	return b.resourceFilter[name]
}

// logf logs a formatted message if verbose mode is enabled.
func (b *Builder) logf(format string, args ...any) {
	if b.verbose {
		log.Printf(format, args...)
	}
}

// BuildResource generates Go code for a FHIR resource.
func (b *Builder) BuildResource(def *model.StructureDefinition) (string, error) {
	if def.Kind != "resource" {
		return "", fmt.Errorf("definition %s is not a resource (kind=%s)", def.ID, def.Kind)
	}

	if def.Snapshot == nil || len(def.Snapshot.Elements) == 0 {
		return "", fmt.Errorf("definition %s has no snapshot elements", def.ID)
	}

	// Extract fields and nested types
	// Use resource name as prefix for BackboneElements to avoid naming conflicts
	fields, nestedTypes, err := b.extractFieldsAndTypes(def, def.Type, def.Name)
	if err != nil {
		return "", fmt.Errorf("extract fields: %w", err)
	}

	// Check if this resource should use embedding
	embeddedType := b.getEmbeddedBaseType(def)
	if embeddedType != "" {
		// Filter out base resource fields (they're in the embedded type)
		fields = b.filterBaseFields(fields, embeddedType)

		// Add embedding field at the beginning
		embeddingField := model.Field{
			Name:       embeddedType,
			GoType:     "fhir." + embeddedType,
			IsEmbedded: true,
		}
		fields = append([]model.Field{embeddingField}, fields...)
	}

	// Build main type
	mainType := model.TypeDefinition{
		Name:       def.Name,
		Kind:       def.Kind,
		Comment:    def.Type,
		Fields:     fields,
		IsAbstract: def.Abstract,
	}

	// Append main type after nested types (nested types need to be defined first)
	nestedTypes = append(nestedTypes, mainType)

	// Generate Go code with all types
	return b.generator.GenerateFile(nestedTypes)
}

// getEmbeddedBaseType returns the base type to embed, or empty string if no embedding.
func (b *Builder) getEmbeddedBaseType(def *model.StructureDefinition) string {
	if def.BaseDefinition == "" {
		return ""
	}

	// Check if base is DomainResource or Resource
	if strings.HasSuffix(def.BaseDefinition, "/DomainResource") {
		return "DomainResource"
	}
	if strings.HasSuffix(def.BaseDefinition, "/Resource") {
		return "Resource"
	}

	return ""
}

// filterBaseFields removes fields that belong to the base type.
func (b *Builder) filterBaseFields(fields []model.Field, baseType string) []model.Field {
	// Define which fields belong to each base type
	resourceFields := map[string]bool{
		"ID": true, "IDExt": true,
		"Meta":          true,
		"ImplicitRules": true, "ImplicitRulesExt": true,
		"Language": true, "LanguageExt": true,
	}

	domainResourceFields := map[string]bool{
		"Text":              true,
		"Contained":         true,
		"Extension":         true,
		"ModifierExtension": true,
	}

	// Determine which fields to filter
	fieldsToRemove := make(map[string]bool)
	if baseType == "Resource" {
		for k, v := range resourceFields {
			fieldsToRemove[k] = v
		}
	} else if baseType == "DomainResource" {
		// DomainResource includes Resource fields too
		for k, v := range resourceFields {
			fieldsToRemove[k] = v
		}
		for k, v := range domainResourceFields {
			fieldsToRemove[k] = v
		}
	}

	// Filter out base fields
	var filtered []model.Field
	for _, field := range fields {
		if !fieldsToRemove[field.Name] {
			filtered = append(filtered, field)
		}
	}

	return filtered
}

// BuildComplexType generates Go code for a FHIR complex type.
func (b *Builder) BuildComplexType(def *model.StructureDefinition) (string, error) {
	if def.Kind != "complex-type" {
		return "", fmt.Errorf("definition %s is not a complex type (kind=%s)", def.ID, def.Kind)
	}

	if def.Snapshot == nil || len(def.Snapshot.Elements) == 0 {
		return "", fmt.Errorf("definition %s has no snapshot elements", def.ID)
	}

	// Extract fields and nested types
	// Use type name as prefix for BackboneElements to avoid naming conflicts
	fields, nestedTypes, err := b.extractFieldsAndTypes(def, def.Type, def.Name)
	if err != nil {
		return "", fmt.Errorf("extract fields: %w", err)
	}

	// Build main type
	mainType := model.TypeDefinition{
		Name:       def.Name,
		Kind:       "complex",
		Comment:    def.Type,
		Fields:     fields,
		IsAbstract: def.Abstract,
	}

	// Append main type after nested types (nested types need to be defined first)
	nestedTypes = append(nestedTypes, mainType)

	// Generate Go code with all types
	return b.generator.GenerateFile(nestedTypes)
}

// extractFieldsAndTypes extracts struct fields and nested type definitions from element definitions.
func (b *Builder) extractFieldsAndTypes(def *model.StructureDefinition, parentPath, prefix string) ([]model.Field, []model.TypeDefinition, error) {
	var fields []model.Field
	var nestedTypes []model.TypeDefinition

	// Get direct children of the parent path
	children := parser.GetDirectChildren(def.Snapshot.Elements, parentPath)

	for _, elem := range children {
		// Handle choice types - expand to multiple fields
		if parser.IsChoiceType(elem.Path) {
			choiceFields, err := b.typeMapper.MapElementToChoiceFields(elem, parentPath)
			if err != nil {
				return nil, nil, fmt.Errorf("map choice element %s: %w", elem.Path, err)
			}

			// Add all choice fields
			for _, choiceField := range choiceFields {
				fields = append(fields, *choiceField)

				// Add primitive extension field if needed
				if len(elem.Types) > 0 {
					for _, typeInfo := range elem.Types {
						if b.typeMapper.IsPrimitiveType(typeInfo.Code) {
							// Find the matching field name
							if choiceField.ChoiceSuffix == parser.ToPascalCase(typeInfo.Code) {
								extField := model.Field{
									Name:      choiceField.Name + "Ext",
									GoType:    "primitives.PrimitiveExtension",
									JSONName:  "_" + choiceField.JSONName,
									Min:       0,
									Max:       "1",
									Comment:   "Extension for " + choiceField.Name,
									IsPointer: true,
									IsArray:   false,
								}
								fields = append(fields, extField)
							}
						}
					}
				}
			}
			continue
		}

		// Map element to field
		field, err := b.typeMapper.MapElementToField(elem, parentPath)
		if err != nil {
			return nil, nil, fmt.Errorf("map element %s: %w", elem.Path, err)
		}

		if field == nil {
			continue // Skip root element
		}

		// Handle BackboneElements (nested structs)
		if parser.IsBackboneElement(elem) {
			// Generate nested struct type
			nestedTypeName := prefix + field.Name
			nestedFields, deeperTypes, err := b.extractFieldsAndTypes(def, elem.Path, nestedTypeName)
			if err != nil {
				return nil, nil, fmt.Errorf("extract nested fields for %s: %w", elem.Path, err)
			}

			// Create type definition for this BackboneElement
			nestedType := model.TypeDefinition{
				Name:    nestedTypeName,
				Kind:    "backbone",
				Comment: "BackboneElement for " + elem.Path,
				Fields:  nestedFields,
			}

			// Add deeper nested types first, then this one
			nestedTypes = append(nestedTypes, deeperTypes...)
			nestedTypes = append(nestedTypes, nestedType)

			// Update field to use the nested type name
			field.GoType = nestedTypeName
		}

		// Add the main field
		fields = append(fields, *field)

		// For primitive types, add parallel extension field
		if len(elem.Types) == 1 && b.typeMapper.IsPrimitiveType(elem.Types[0].Code) && !parser.IsBackboneElement(elem) {
			extField := model.Field{
				Name:      field.Name + "Ext",
				GoType:    "primitives.PrimitiveExtension",
				JSONName:  "_" + field.JSONName,
				Min:       0,
				Max:       "1",
				Comment:   "Extension for " + field.Name,
				IsPointer: true,
				IsArray:   false,
			}
			fields = append(fields, extField)
		}
	}

	return fields, nestedTypes, nil
}

// BuildAll generates Go code for all resources and complex types.
func (b *Builder) BuildAll() (map[string]string, error) {
	result := make(map[string]string)

	// Generate resources
	resources := b.parser.GetResources()
	b.logf("Generating %d resources...", len(resources))

	resourceCount := 0
	skippedResources := 0
	filteredOut := 0
	for _, def := range resources {
		if def.Abstract {
			// Skip abstract resources (can't be instantiated)
			b.logf("  Skipping abstract resource: %s", def.Name)
			skippedResources++
			continue
		}

		// Apply resource filter
		if !b.shouldGenerateResource(def.Name) {
			b.logf("  Filtered out resource: %s", def.Name)
			filteredOut++
			continue
		}

		b.logf("  Building resource: %s", def.Name)
		code, err := b.BuildResource(def)
		if err != nil {
			return nil, fmt.Errorf("build resource %s: %w", def.Name, err)
		}

		filename := strings.ToLower(def.Name) + ".go"
		result[filename] = code
		resourceCount++
	}
	b.logf("Generated %d resources (%d skipped, %d filtered)", resourceCount, skippedResources, filteredOut)

	// Generate complex types
	complexTypes := b.parser.GetComplexTypes()
	b.logf("Generating %d complex types...", len(complexTypes))

	complexCount := 0
	skippedComplex := 0
	for _, def := range complexTypes {
		if def.Abstract {
			// Skip abstract types
			b.logf("  Skipping abstract type: %s", def.Name)
			skippedComplex++
			continue
		}

		b.logf("  Building complex type: %s", def.Name)
		code, err := b.BuildComplexType(def)
		if err != nil {
			return nil, fmt.Errorf("build complex type %s: %w", def.Name, err)
		}

		filename := strings.ToLower(def.Name) + ".go"
		result[filename] = code
		complexCount++
	}
	b.logf("Generated %d complex types (%d skipped)", complexCount, skippedComplex)
	b.logf("Total files generated: %d", len(result))

	return result, nil
}

// GeneratePackage generates a complete package with all types.
func (b *Builder) GeneratePackage(outputDir string) error {
	files, err := b.BuildAll()
	if err != nil {
		return fmt.Errorf("build all types: %w", err)
	}

	// TODO: Write files to outputDir
	_ = files

	return nil
}
