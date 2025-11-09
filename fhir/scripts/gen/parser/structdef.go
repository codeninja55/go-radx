package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/codeninja55/go-radx/fhir/scripts/gen/model"
)

// StructDefBundle represents the FHIR Bundle containing StructureDefinitions.
type StructDefBundle struct {
	ResourceType string        `json:"resourceType"`
	Type         string        `json:"type"`
	Entry        []BundleEntry `json:"entry"`
}

// BundleEntry represents a single entry in the bundle.
type BundleEntry struct {
	FullURL  string          `json:"fullUrl"`
	Resource json.RawMessage `json:"resource"`
}

// RawStructDef is the raw JSON structure from FHIR specification.
type RawStructDef struct {
	ResourceType   string       `json:"resourceType"`
	ID             string       `json:"id"`
	URL            string       `json:"url"`
	Name           string       `json:"name"`
	Kind           string       `json:"kind"`
	Abstract       bool         `json:"abstract"`
	Type           string       `json:"type"`
	BaseDefinition string       `json:"baseDefinition"`
	Snapshot       *RawSnapshot `json:"snapshot"`
}

// RawSnapshot contains element definitions.
type RawSnapshot struct {
	Element []RawElement `json:"element"`
}

// RawElement is a single element definition.
type RawElement struct {
	Path         string          `json:"path"`
	Short        string          `json:"short"`
	Definition   string          `json:"definition"`
	Min          int             `json:"min"`
	Max          string          `json:"max"`
	Type         []RawType       `json:"type"`
	IsModifier   bool            `json:"isModifier"`
	IsSummary    bool            `json:"isSummary"`
	Binding      *RawBinding     `json:"binding"`
	FixedValue   json.RawMessage `json:"fixedValue"`
	DefaultValue json.RawMessage `json:"defaultValue"`
}

// RawType describes element types.
type RawType struct {
	Code          string   `json:"code"`
	TargetProfile []string `json:"targetProfile"`
	Profile       []string `json:"profile"`
}

// RawBinding describes value set binding.
type RawBinding struct {
	Strength string `json:"strength"`
	ValueSet string `json:"valueSet"`
}

// Parser parses FHIR StructureDefinitions.
type Parser struct {
	definitions map[string]*model.StructureDefinition
}

// New creates a new parser.
func New() *Parser {
	return &Parser{
		definitions: make(map[string]*model.StructureDefinition),
	}
}

// ParseFile parses a FHIR profiles-resources.json file.
func (p *Parser) ParseFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	var bundle StructDefBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return fmt.Errorf("unmarshal bundle: %w", err)
	}

	for _, entry := range bundle.Entry {
		var raw RawStructDef
		if err := json.Unmarshal(entry.Resource, &raw); err != nil {
			// Skip non-StructureDefinition resources
			continue
		}

		if raw.ResourceType != "StructureDefinition" {
			continue
		}

		def, err := p.parseStructDef(&raw)
		if err != nil {
			return fmt.Errorf("parse %s: %w", raw.ID, err)
		}

		p.definitions[raw.ID] = def
	}

	return nil
}

// parseStructDef converts raw structure to internal model.
func (p *Parser) parseStructDef(raw *RawStructDef) (*model.StructureDefinition, error) {
	def := &model.StructureDefinition{
		ResourceType:   raw.ResourceType,
		ID:             raw.ID,
		URL:            raw.URL,
		Name:           raw.Name,
		Kind:           raw.Kind,
		Abstract:       raw.Abstract,
		Type:           raw.Type,
		BaseDefinition: raw.BaseDefinition,
	}

	if raw.Snapshot != nil {
		def.Snapshot = &model.Snapshot{
			Elements: make([]model.ElementDefinition, 0, len(raw.Snapshot.Element)),
		}

		for _, elem := range raw.Snapshot.Element {
			elemDef := model.ElementDefinition{
				Path:       elem.Path,
				Short:      elem.Short,
				Definition: elem.Definition,
				Min:        elem.Min,
				Max:        elem.Max,
				IsModifier: elem.IsModifier,
				IsSummary:  elem.IsSummary,
			}

			// Parse types
			for _, t := range elem.Type {
				elemDef.Types = append(elemDef.Types, model.ElementType{
					Code:          t.Code,
					TargetProfile: t.TargetProfile,
					Profile:       t.Profile,
				})
			}

			// Parse binding
			if elem.Binding != nil {
				elemDef.Binding = &model.ElementBinding{
					Strength: elem.Binding.Strength,
					ValueSet: elem.Binding.ValueSet,
				}
			}

			def.Snapshot.Elements = append(def.Snapshot.Elements, elemDef)
		}
	}

	return def, nil
}

// GetDefinition returns a parsed structure definition by ID.
func (p *Parser) GetDefinition(id string) (*model.StructureDefinition, bool) {
	def, ok := p.definitions[id]
	return def, ok
}

// GetAllDefinitions returns all parsed definitions.
func (p *Parser) GetAllDefinitions() map[string]*model.StructureDefinition {
	return p.definitions
}

// GetResources returns only resource-type definitions.
func (p *Parser) GetResources() []*model.StructureDefinition {
	var resources []*model.StructureDefinition
	for _, def := range p.definitions {
		if def.Kind == "resource" {
			resources = append(resources, def)
		}
	}
	return resources
}

// GetComplexTypes returns complex data type definitions.
func (p *Parser) GetComplexTypes() []*model.StructureDefinition {
	var types []*model.StructureDefinition
	for _, def := range p.definitions {
		if def.Kind == "complex-type" {
			types = append(types, def)
		}
	}
	return types
}

// GetPrimitiveTypes returns primitive type definitions.
func (p *Parser) GetPrimitiveTypes() []*model.StructureDefinition {
	var types []*model.StructureDefinition
	for _, def := range p.definitions {
		if def.Kind == "primitive-type" {
			types = append(types, def)
		}
	}
	return types
}

// IsChoiceType checks if an element path represents a choice type (e.g., "deceased[x]").
func IsChoiceType(path string) bool {
	return strings.HasSuffix(path, "[x]")
}

// GetChoiceBaseName extracts the base name from a choice type path.
// e.g., "Patient.deceased[x]" -> "deceased"
func GetChoiceBaseName(path string) string {
	if !IsChoiceType(path) {
		return path
	}
	parts := strings.Split(path, ".")
	lastName := parts[len(parts)-1]
	return strings.TrimSuffix(lastName, "[x]")
}
