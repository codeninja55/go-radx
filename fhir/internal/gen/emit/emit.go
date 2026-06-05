// Package emit is the generator's emitter stage. It turns the planner's
// emitter-ready PlannedTypes into byte-stable, gofmt-clean Go source through
// text/template followed by a go/format pass, so the committed generated files are
// reproduced byte-for-byte on every run. The emitter makes no Go-shape decision —
// every name, type, and tag is already fixed by the planner; the emitter only
// renders. The output carries the standard "DO NOT EDIT" banner so the
// generated-never-hand-edited property is signalled in the file itself.
package emit

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"sort"
	"strings"
	"text/template"

	"github.com/codeninja55/go-radx/fhir/internal/gen/plan"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// File is the input to a single emitted Go file: the target package, the imports the
// rendered types require, and the planned types to render in the file. The emitter
// renders the types in the given order, so the caller fixes the order (canonical,
// stable) before calling Emit.
type File struct {
	// Package is the Go package clause of the emitted file (for example "r5").
	Package string

	// Types are the planned types rendered into the file, in the order given.
	Types []plan.PlannedType
}

// Emit renders a File to formatted Go source. It executes the datatype template, runs
// the result through go/format (so the output is gofmt-stable and any template
// whitespace slop is normalised), and returns the formatted bytes. A template or
// format error is returned with the rendered source attached so a failure is
// diagnosable. The returned bytes are deterministic for a given File: the same plan
// always yields the same source, which is what makes regeneration byte-for-byte
// reproducible.
func Emit(f File) ([]byte, error) {
	tmpl, err := template.New("datatype.go.tmpl").ParseFS(templatesFS, "templates/datatype.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("emit: parse template: %w", err)
	}

	data := struct {
		Package string
		Imports []string
		Types   []plan.PlannedType
	}{
		Package: f.Package,
		Imports: requiredImports(f.Types),
		Types:   f.Types,
	}

	var raw bytes.Buffer
	if err := tmpl.Execute(&raw, data); err != nil {
		return nil, fmt.Errorf("emit: execute template: %w", err)
	}

	formatted, err := format.Source(raw.Bytes())
	if err != nil {
		return nil, fmt.Errorf("emit: gofmt the rendered source: %w\n--- rendered ---\n%s", err, raw.String())
	}
	return formatted, nil
}

// Registry is the input to the per-release registry file: the target package and the
// resources whose factories the generated init() registers. The emitter renders the
// resources in the order given, so the caller fixes a stable (canonical) order before
// calling EmitRegistry.
type Registry struct {
	// Package is the Go package clause of the emitted registry file (for example "r5").
	Package string

	// Resources are the release's resources, each contributing one factory
	// registration. Only the Go name is needed; the resourceType constant the
	// registration references is the generated <GoName>ResourceType.
	Resources []RegistryEntry
}

// RegistryEntry names one resource for the registry: its Go type name, from which the
// generated <GoName>ResourceType constant and the &<GoName>{} factory are derived.
type RegistryEntry struct {
	GoName string
}

// Primitives is the input to the per-release primitive-wrapper file: the target
// package and the wrapper descriptors to render. The wrappers box a primitive value so
// it can satisfy a choice group's sealed value interface, which a built-in scalar
// cannot. The caller fixes a stable (sorted) order before calling EmitPrimitives.
type Primitives struct {
	// Package is the Go package clause of the emitted file (for example "r5").
	Package string

	// Wrappers are the release primitive wrapper descriptors, in stable order.
	Wrappers []plan.PrimitiveWrapper
}

// EmitPrimitives renders a release's primitive wrapper types to formatted Go source.
// Each wrapper is a distinct named type per FHIR primitive code so a choice Value()
// type switch can recover which branch was set; the decimal wrapper delegates its JSON
// round-trip to fhir.Decimal so the lexical form is preserved. Like Emit, the output is
// deterministic for a given Primitives, keeping regeneration byte-for-byte reproducible.
func EmitPrimitives(p Primitives) ([]byte, error) {
	tmpl, err := template.New("primitives.go.tmpl").ParseFS(templatesFS, "templates/primitives.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("emit: parse primitives template: %w", err)
	}

	data := struct {
		Package  string
		Imports  []string
		Wrappers []plan.PrimitiveWrapper
	}{
		Package:  p.Package,
		Imports:  primitiveImports(p.Wrappers),
		Wrappers: p.Wrappers,
	}

	var raw bytes.Buffer
	if err := tmpl.Execute(&raw, data); err != nil {
		return nil, fmt.Errorf("emit: execute primitives template: %w", err)
	}

	formatted, err := format.Source(raw.Bytes())
	if err != nil {
		return nil, fmt.Errorf("emit: gofmt the rendered primitives: %w\n--- rendered ---\n%s", err, raw.String())
	}
	return formatted, nil
}

// primitiveImports returns the import paths the wrapper file needs: the root fhir
// package when any wrapper delegates to fhir.Decimal (the decimal wrapper always
// does), and nothing otherwise.
func primitiveImports(wrappers []plan.PrimitiveWrapper) []string {
	for _, w := range wrappers {
		if w.Kind == plan.WrapperDecimal {
			return []string{"github.com/codeninja55/go-radx/fhir"}
		}
	}
	return nil
}

// EmitRegistry renders a release's resourceType→factory registry file to formatted Go
// source. The generated init() registers each resource's factory with the root fhir
// package, so fhir.UnmarshalResource can dispatch by resourceType. Like Emit, the
// output is deterministic for a given Registry, which keeps regeneration
// byte-for-byte reproducible.
func EmitRegistry(r Registry) ([]byte, error) {
	tmpl, err := template.New("registry.go.tmpl").ParseFS(templatesFS, "templates/registry.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("emit: parse registry template: %w", err)
	}

	var raw bytes.Buffer
	if err := tmpl.Execute(&raw, r); err != nil {
		return nil, fmt.Errorf("emit: execute registry template: %w", err)
	}

	formatted, err := format.Source(raw.Bytes())
	if err != nil {
		return nil, fmt.Errorf("emit: gofmt the rendered registry: %w\n--- rendered ---\n%s", err, raw.String())
	}
	return formatted, nil
}

// requiredImports computes the deduplicated, sorted import paths the planned types
// reference. It recognises the dependencies the generated code in this stage can
// carry: the root fhir import (fhir.Decimal, fhir.PrimitiveElement, and the
// "_field" sibling helpers) and encoding/json (a resource's always-resourceType
// MarshalJSON and the "_field" sibling marshal/unmarshal methods all call into
// encoding/json). Sorting keeps the import block stable across runs.
func requiredImports(types []plan.PlannedType) []string {
	const fhirImport = "github.com/codeninja55/go-radx/fhir"
	const jsonImport = "encoding/json"
	set := map[string]bool{}
	for _, t := range types {
		if t.IsResource() {
			set[jsonImport] = true
		}
		if hasPrimitiveSibling(t) {
			set[jsonImport] = true
			set[fhirImport] = true
		}
		for _, f := range allFields(t) {
			if usesFHIR(f.GoType) {
				set[fhirImport] = true
			}
			if f.IsPrimitiveSibling() {
				set[fhirImport] = true
			}
		}
	}
	out := make([]string, 0, len(set))
	for imp := range set {
		out = append(out, imp)
	}
	sort.Strings(out)
	return out
}

// hasPrimitiveSibling reports whether a planned type or any of its backbones owns a
// primitive "_field" sibling, so the import set includes the dependencies the
// generated sibling marshal/unmarshal methods need.
func hasPrimitiveSibling(t plan.PlannedType) bool {
	if t.HasPrimitiveSibling() {
		return true
	}
	for _, bb := range t.Backbones {
		if bb.HasPrimitiveSibling() {
			return true
		}
	}
	return false
}

// allFields flattens a planned type's top-level fields and its backbones' fields, so
// import detection sees every field a file will write.
func allFields(t plan.PlannedType) []plan.Field {
	fields := append([]plan.Field(nil), t.Fields...)
	for _, bb := range t.Backbones {
		fields = append(fields, bb.Fields...)
	}
	return fields
}

// usesFHIR reports whether a Go field type references the root fhir package — the
// shared fhir.Decimal, fhir.PrimitiveElement, or the fhir.Resource interface a
// generated field can carry — so the import set includes the root package when any
// such type appears.
func usesFHIR(goType string) bool {
	return strings.Contains(goType, "fhir.")
}
