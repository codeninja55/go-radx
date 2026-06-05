package gen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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

// generatedType pairs a FHIR type name with the generated file's base name. The
// generator currently emits a representative slice of the full type set; until the
// bulk-generation increment lands, these slices are the single source of truth for
// what the generator produces, so regeneration is reproducible from them alone.
type generatedType struct {
	fhirName string
	fileName string
}

// representativeDatatypes are the complex datatypes the generator currently emits —
// one simple datatype that proves the pipeline end to end (load, model, plan, emit,
// write).
var representativeDatatypes = []generatedType{
	{fhirName: "Period", fileName: "period.go"},
}

// representativeResources are the resources the generator currently emits — one
// small resource that exercises the resource shape end to end: the resourceType
// constant, the ResourceType method, the always-emit-resourceType MarshalJSON
// (FHIR-004), and a factory registration in the generated registry so
// fhir.UnmarshalResource can dispatch to it. The full resource set (about 158 R5
// resources) is generated in a later increment; this one resource proves the
// identity API and registry machinery without bulk-generating.
//
// Flag is chosen because its own elements reference only already-available types
// (Identifier, CodeableConcept, Reference, and the generated Period); its inherited
// Resource/DomainResource base members are planned away (Options.SkipBaseMembers)
// until the base machinery lands, so the representative resource compiles today.
var representativeResources = []generatedType{
	{fhirName: "Flag", fileName: "flag.go"},
}

// Generate runs the full generator pipeline for one release: load and verify the
// bundle, build the model, plan each type, emit gofmt-stable Go, and write the
// generated files into the release package. It is the single library entry point the
// CLI and the go:generate directive call. Re-running it reproduces the committed
// output byte-for-byte, which is the property the diff gate verifies.
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

	for _, dt := range representativeDatatypes {
		if err := emitTypeFile(bundle, cfg, dt); err != nil {
			return err
		}
	}
	for _, res := range representativeResources {
		if err := emitTypeFile(bundle, cfg, res); err != nil {
			return err
		}
	}
	if err := emitRegistryFile(bundle, cfg); err != nil {
		return err
	}
	return nil
}

// emitTypeFile runs the back-half pipeline for one type and writes its generated Go
// file into the output directory. Datatypes and resources share the pipeline; the
// planner's recorded kind drives the resource-specific output (resourceType,
// ResourceType, MarshalJSON), so the same emit path serves both.
func emitTypeFile(bundle *loader.Bundle, cfg Config, gt generatedType) error {
	pt, err := planType(bundle, gt.fhirName)
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
// representativeResources order, which is the canonical, stable order the
// byte-for-byte regeneration gate depends on.
func emitRegistryFile(bundle *loader.Bundle, cfg Config) error {
	entries := make([]emit.RegistryEntry, 0, len(representativeResources))
	for _, res := range representativeResources {
		pt, err := planType(bundle, res.fhirName)
		if err != nil {
			return err
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
// its emitter-ready PlannedType. SkipBaseMembers is set because the shared base
// machinery (Element id/extension, the resource and DomainResource bases, and the
// primitive-extension siblings) is not yet generated; a later increment embeds the
// base types and retires the option.
func planType(bundle *loader.Bundle, fhirName string) (plan.PlannedType, error) {
	sd, ok := bundle.StructureDefinition(fhirName)
	if !ok {
		return plan.PlannedType{}, fmt.Errorf("fhir/gen: StructureDefinition %q not in bundle", fhirName)
	}
	typ, err := model.BuildType(sd)
	if err != nil {
		return plan.PlannedType{}, fmt.Errorf("fhir/gen: build model for %s: %w", fhirName, err)
	}
	return plan.PlanType(typ, plan.Options{SkipBaseMembers: true}), nil
}
