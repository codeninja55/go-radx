package gen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/codeninja55/go-radx/fhir/internal/gen/emit"
	"github.com/codeninja55/go-radx/fhir/internal/gen/loader"
	"github.com/codeninja55/go-radx/fhir/internal/gen/model"
	"github.com/codeninja55/go-radx/fhir/internal/gen/plan"
)

// Config controls a single generator run: which release to generate, where the
// vendored definition bundle lives, and where to write the generated Go. The CLI
// (cmd/fhir-gen) builds this from flags; the go:generate directive in fhir/gen.go
// drives it with the committed paths.
type Config struct {
	// Release is the FHIR release to generate (for example "r5"). It selects both
	// the vendored bundle subdirectory and the output release package.
	Release string

	// DefinitionsDir is the directory holding the vendored, checksum-pinned bundle
	// (profiles-types.json, profiles-resources.json, valuesets.json, SHA256SUMS).
	DefinitionsDir string

	// OutputDir is the release package directory the generated Go is written to
	// (for example "r5"). The generator overwrites generated files in place.
	OutputDir string
}

// baseTypes are the shared abstract base types every concrete resource and datatype
// inherits from. They are generated once into base.go and embedded by the concrete
// types so a resource is faithful (it carries id, meta, text, contained, extension,
// and the rest) without restating the base members inline. A base type is generated
// flat (it embeds nothing) and without primitive "_field" siblings: a base type must
// not define MarshalJSON, because a value-embedded type whose MarshalJSON is promoted
// would shadow the embedding type's own MarshalJSON and drop every non-base field on
// the wire.
var baseTypes = []string{"Element", "BackboneElement", "Resource", "DomainResource"}

// handWrittenTypes are the R5 types the M2 walking skeleton still hand-writes
// (fhir/r5/service_request.go, diagnostic_report.go, imaging_study.go, and
// datatypes.go). The bulk generator does not emit these now, because a generated
// definition would collide with the hand-written one at the same Go name in the same
// package and break compilation. The migration increment (F1-O) deletes the
// hand-written files, re-points convert and the skeleton at the generated shapes, and
// drops this exclusion so the generator owns the full set.
var handWrittenTypes = map[string]bool{
	// Resources (fhir/r5/{service_request,diagnostic_report,imaging_study}.go and
	// the ImagingStudy backbones the hand-written file defines).
	"ServiceRequest":   true,
	"DiagnosticReport": true,
	"ImagingStudy":     true,
	// Datatypes (fhir/r5/datatypes.go).
	"Reference":         true,
	"Identifier":        true,
	"Coding":            true,
	"CodeableConcept":   true,
	"CodeableReference": true,
}

// generatedType pairs a FHIR type name with the generated file's base name and the
// flag set when planning it as a base type. The generator derives this list from the
// loaded bundle on every run (deterministically, by sorted name), so a single
// authority decides what is generated and the byte-for-byte regeneration gate
// reproduces it from the bundle alone.
type generatedType struct {
	fhirName   string
	fileName   string
	isBaseType bool
}

// Generate runs the full generator pipeline for one release: load and verify the
// bundle, build the model, plan each type, emit gofmt-stable Go, and write the
// generated files into the release package. It generates every concrete resource and
// complex datatype (and the shared base types), excluding only the names the M2
// walking skeleton still hand-writes. Re-running it reproduces the committed output
// byte-for-byte, which is the property the diff gate verifies.
func Generate(cfg Config) error {
	if cfg.Release == "" {
		return fmt.Errorf("fhir/gen: %w: release is required", errors.ErrUnsupported)
	}
	if cfg.DefinitionsDir == "" {
		return fmt.Errorf("fhir/gen: %w: definitions directory is required", errors.ErrUnsupported)
	}
	if cfg.OutputDir == "" {
		return fmt.Errorf("fhir/gen: %w: output directory is required", errors.ErrUnsupported)
	}

	bundle, err := loader.Load(cfg.DefinitionsDir)
	if err != nil {
		return fmt.Errorf("fhir/gen: load definitions: %w", err)
	}

	types := GeneratedTypes(bundle)
	if err := emitBaseFile(bundle, cfg); err != nil {
		return err
	}
	if err := emitPrimitivesFile(cfg); err != nil {
		return err
	}
	if err := emitEnumsFile(bundle, cfg); err != nil {
		return err
	}
	for _, gt := range types {
		if gt.isBaseType {
			continue
		}
		if err := emitTypeFile(bundle, cfg, gt); err != nil {
			return err
		}
	}
	if err := emitRegistryFile(bundle, cfg, types); err != nil {
		return err
	}
	return nil
}

// GeneratedTypes returns, in stable order, every type the generator emits for the
// bundle: the shared base types first (all written into base.go), then every concrete
// resource and complex datatype the bundle defines, in sorted name order, excluding
// the hand-written skeleton types and the abstract types the generator does not
// model. Both Generate and the byte-for-byte regeneration gate read this single
// authority, so the committed file set is exactly what a fresh run reproduces.
func GeneratedTypes(bundle *loader.Bundle) []generatedType {
	types := make([]generatedType, 0, len(baseTypes)+200)
	for _, name := range baseTypes {
		types = append(types, generatedType{fhirName: name, fileName: "base.go", isBaseType: true})
	}

	for _, name := range bundle.StructureDefinitionNames() {
		if handWrittenTypes[name] || isBaseTypeName(name) {
			continue
		}
		sd, ok := bundle.StructureDefinition(name)
		if !ok || sd.Abstract {
			continue
		}
		switch model.Classify(sd) {
		case model.KindResource, model.KindComplexType:
			types = append(types, generatedType{fhirName: name, fileName: goFileName(name)})
		default:
			// A primitive type (mapped to a Go scalar, not a struct) or a kind the
			// generator does not model: skip it.
		}
	}
	return types
}

// GeneratedFileNames returns the sorted set of file names the generator writes for
// the bundle (base.go, every per-type file, and registry.go), so the regeneration
// gate can diff exactly that set against the committed tree without re-deriving the
// naming rules.
func GeneratedFileNames(bundle *loader.Bundle) []string {
	seen := map[string]bool{}
	var names []string
	for _, gt := range GeneratedTypes(bundle) {
		if !seen[gt.fileName] {
			seen[gt.fileName] = true
			names = append(names, gt.fileName)
		}
	}
	names = append(names, "primitives.go", "bindings.go", "registry.go")
	return names
}

// isBaseTypeName reports whether a FHIR type name is one of the shared base types
// emitted into base.go, so the per-type loop does not emit it a second time.
func isBaseTypeName(name string) bool {
	for _, b := range baseTypes {
		if b == name {
			return true
		}
	}
	return false
}

// emitBaseFile renders the shared base types into base.go in their fixed order. The
// base types share one file and one package import block, so they are emitted
// together rather than one file per base; the order is the fixed baseTypes order so
// the file is byte-stable.
func emitBaseFile(bundle *loader.Bundle, cfg Config) error {
	planned := make([]plan.PlannedType, 0, len(baseTypes))
	for _, name := range baseTypes {
		pt, err := planType(bundle, name, true)
		if err != nil {
			return err
		}
		planned = append(planned, pt)
	}
	src, err := emit.Emit(emit.File{Package: cfg.Release, Types: planned})
	if err != nil {
		return fmt.Errorf("fhir/gen: emit base types: %w", err)
	}
	outPath := filepath.Join(cfg.OutputDir, "base.go")
	if err := os.WriteFile(outPath, src, 0o644); err != nil {
		return fmt.Errorf("fhir/gen: write %s: %w", outPath, err)
	}
	return nil
}

// emitPrimitivesFile renders the release's primitive wrapper types into primitives.go.
// The wrappers box a primitive value so it can satisfy a choice group's sealed value
// interface, which a built-in scalar cannot carry; they are emitted once per release
// from the fixed wrapper set, so the file is byte-stable and does not depend on the
// loaded bundle.
func emitPrimitivesFile(cfg Config) error {
	src, err := emit.EmitPrimitives(emit.Primitives{Package: cfg.Release, Wrappers: plan.PrimitiveWrappers()})
	if err != nil {
		return fmt.Errorf("fhir/gen: emit primitive wrappers: %w", err)
	}
	outPath := filepath.Join(cfg.OutputDir, "primitives.go")
	if err := os.WriteFile(outPath, src, 0o644); err != nil {
		return fmt.Errorf("fhir/gen: write %s: %w", outPath, err)
	}
	return nil
}

// emitEnumsFile renders the release's required-binding enums into bindings.go: every
// enumerable binding as a closed enum (defined string type, const set, validating
// ParseXxx, strict UnmarshalJSON) and every non-enumerable binding as a documented
// not-inlined plain string type, never a silently-empty closed enum (FHIR-013). The
// enum set is derived from the bundle's required, code-typed bindings and sorted by
// Go name, so the file is byte-stable.
func emitEnumsFile(bundle *loader.Bundle, cfg Config) error {
	enums, err := PlannedEnums(bundle)
	if err != nil {
		return err
	}
	src, err := emit.EmitEnums(emit.Enums{Package: cfg.Release, Enums: enums})
	if err != nil {
		return fmt.Errorf("fhir/gen: emit required-binding enums: %w", err)
	}
	outPath := filepath.Join(cfg.OutputDir, "bindings.go")
	if err := os.WriteFile(outPath, src, 0o644); err != nil {
		return fmt.Errorf("fhir/gen: write %s: %w", outPath, err)
	}
	return nil
}

// emitTypeFile runs the back-half pipeline for one concrete type and writes its
// generated Go file. Datatypes and resources share the pipeline; the planner's
// recorded kind drives the resource-specific output (resourceType, ResourceType,
// MarshalJSON), so the same emit path serves both. The binding resolver types a code
// field with an enumerable required binding as its generated enum.
func emitTypeFile(bundle *loader.Bundle, cfg Config, gt generatedType) error {
	pt, err := planType(bundle, gt.fhirName, gt.isBaseType)
	if err != nil {
		return err
	}
	src, err := emit.Emit(emit.File{Package: cfg.Release, Types: []plan.PlannedType{pt}})
	if err != nil {
		return fmt.Errorf("fhir/gen: emit %s: %w", gt.fhirName, err)
	}
	outPath := filepath.Join(cfg.OutputDir, gt.fileName)
	if err := os.WriteFile(outPath, src, 0o644); err != nil {
		return fmt.Errorf("fhir/gen: write %s: %w", outPath, err)
	}
	return nil
}

// emitRegistryFile renders the per-release resourceType→factory registry from the
// generated resources and writes registry.go. The registry's generated init()
// registers each resource's factory with the root fhir package so
// fhir.UnmarshalResource can dispatch by resourceType. The resources are listed in
// the same stable order GeneratedTypes produces, which is the canonical order the
// byte-for-byte regeneration gate depends on.
func emitRegistryFile(bundle *loader.Bundle, cfg Config, types []generatedType) error {
	var entries []emit.RegistryEntry
	for _, gt := range types {
		if gt.isBaseType {
			continue
		}
		pt, err := planType(bundle, gt.fhirName, false)
		if err != nil {
			return err
		}
		if !pt.IsResource() {
			continue
		}
		entries = append(entries, emit.RegistryEntry{GoName: pt.GoName})
	}
	src, err := emit.EmitRegistry(emit.Registry{Package: cfg.Release, Resources: entries})
	if err != nil {
		return fmt.Errorf("fhir/gen: emit registry: %w", err)
	}
	outPath := filepath.Join(cfg.OutputDir, "registry.go")
	if err := os.WriteFile(outPath, src, 0o644); err != nil {
		return fmt.Errorf("fhir/gen: write %s: %w", outPath, err)
	}
	return nil
}

// planType runs the front-half pipeline (model, plan) for one FHIR type and returns
// its emitter-ready PlannedType. A concrete type embeds the shared base it inherits
// from and drops the members that base supplies; a base type (isBaseType) keeps every
// member, embeds nothing, and carries no primitive "_field" siblings. A concrete type
// is planned with the binding resolver so a code field with an enumerable required
// binding is typed as its generated enum; a base type is planned with no resolver
// because its members carry no required-binding code field.
func planType(bundle *loader.Bundle, fhirName string, isBaseType bool) (plan.PlannedType, error) {
	sd, ok := bundle.StructureDefinition(fhirName)
	if !ok {
		return plan.PlannedType{}, fmt.Errorf("fhir/gen: StructureDefinition %q not in bundle", fhirName)
	}
	typ, err := model.BuildType(sd)
	if err != nil {
		return plan.PlannedType{}, fmt.Errorf("fhir/gen: build model for %s: %w", fhirName, err)
	}
	opts := plan.Options{IsBaseType: isBaseType}
	if !isBaseType {
		opts.Bindings = newBindingResolver(bundle)
	}
	return plan.PlanType(typ, opts), nil
}

// goFileName maps a FHIR type name to its generated file's snake_case base name
// ("CodeSystem" -> "code_system.go", "OperationOutcome" -> "operation_outcome.go"),
// matching the existing hand-written file naming so the generated tree reads
// consistently. The mapping is a pure function of the name, so it is stable across
// runs.
func goFileName(fhirName string) string {
	var b strings.Builder
	for i, r := range fhirName {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String() + ".go"
}
