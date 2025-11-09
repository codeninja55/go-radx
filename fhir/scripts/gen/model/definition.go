package model

import (
	"fmt"
	"strings"
)

// StructureDefinition represents a FHIR StructureDefinition resource.
type StructureDefinition struct {
	ResourceType   string
	ID             string
	URL            string
	Name           string
	Kind           string // primitive-type, complex-type, resource, logical
	Abstract       bool
	Type           string
	BaseDefinition string
	Snapshot       *Snapshot
}

// Snapshot contains the flattened view of the structure.
type Snapshot struct {
	Elements []ElementDefinition
}

// ElementDefinition describes a single element in a FHIR structure.
type ElementDefinition struct {
	Path         string
	Short        string
	Definition   string
	Min          int
	Max          string // "1", "*", etc.
	Types        []ElementType
	IsModifier   bool
	IsSummary    bool
	Binding      *ElementBinding
	FixedValue   any
	DefaultValue any
}

// ElementType describes the data type(s) allowed for an element.
type ElementType struct {
	Code          string   // e.g., "string", "Patient", "Reference"
	TargetProfile []string // for Reference types
	Profile       []string
}

// ElementBinding describes value set bindings for coded elements.
type ElementBinding struct {
	Strength    string // required, extensible, preferred, example
	ValueSet    string
	ValueSetURI string
}

// Field represents a Go struct field to be generated.
type Field struct {
	Name         string
	GoType       string
	JSONName     string
	Min          int
	Max          string
	Comment      string
	IsPointer    bool
	IsArray      bool
	IsChoice     bool     // for polymorphic fields like deceased[x]
	ChoiceSuffix string   // e.g., "Boolean", "DateTime"
	IsSummary    bool     // FHIR summary element flag
	IsRequired   bool     // Field is required (min >= 1)
	IsEmbedded   bool     // For struct embedding (e.g., DomainResource)
	EnumValues   []string // For coded fields with enum binding
	ChoiceGroup  string   // Choice group name for mutual exclusion (e.g., "deceased")
}

// TypeDefinition represents a Go type to be generated.
type TypeDefinition struct {
	Name          string
	Kind          string // "primitive", "complex", "resource", "backbone"
	Comment       string
	BaseType      string
	Fields        []Field
	IsAbstract    bool
	SourceElement *ElementDefinition
}

// FHIRTag generates a FHIR struct tag for validation metadata.
// Format: fhir:"cardinality=0..1,required,enum=male|female,summary,choice=deceased"
func (f *Field) FHIRTag() string {
	if f == nil {
		return ""
	}

	var parts []string

	// Add cardinality
	if f.Max != "" {
		cardMin := "0"
		if f.Min > 0 {
			cardMin = fmt.Sprintf("%d", f.Min)
		}
		parts = append(parts, fmt.Sprintf("cardinality=%s..%s", cardMin, f.Max))
	}

	// Add required flag
	if f.IsRequired || f.Min >= 1 {
		parts = append(parts, "required")
	}

	// Add enum values
	if len(f.EnumValues) > 0 {
		parts = append(parts, fmt.Sprintf("enum=%s", strings.Join(f.EnumValues, "|")))
	}

	// Add summary flag
	if f.IsSummary {
		parts = append(parts, "summary")
	}

	// Add choice group
	if f.ChoiceGroup != "" {
		parts = append(parts, fmt.Sprintf("choice=%s", f.ChoiceGroup))
	}

	if len(parts) == 0 {
		return ""
	}

	return `fhir:"` + strings.Join(parts, ",") + `"`
}
