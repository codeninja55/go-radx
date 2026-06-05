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

// representativeDatatypes are the FHIR types the generator currently emits — one
// simple complex datatype that proves the pipeline end to end (load, model, plan,
// emit, write). Each entry pairs a FHIR type name with the generated file's base
// name. The full type set is generated in a later increment; until then this set is
// the single source of truth for what the generator produces, so regeneration is
// reproducible from it alone.
var representativeDatatypes = []struct {
	fhirName string
	fileName string
}{
	{fhirName: "Period", fileName: "period.go"},
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
		src, err := emitDatatype(bundle, cfg.Release, dt.fhirName)
		if err != nil {
			return err
		}
		outPath := filepath.Join(cfg.OutputDir, dt.fileName)
		if err := os.WriteFile(outPath, src, 0o644); err != nil {
			return fmt.Errorf("fhir/gen: write %s: %w", outPath, err)
		}
	}
	return nil
}

// emitDatatype runs the back-half pipeline (model, plan, emit) for one datatype and
// returns its formatted Go source. SkipBaseMembers is set because the shared Element
// base machinery (id/extension and the primitive-extension siblings) is not yet
// generated; a later increment embeds the base and retires the option.
func emitDatatype(bundle *loader.Bundle, release, fhirName string) ([]byte, error) {
	sd, ok := bundle.StructureDefinition(fhirName)
	if !ok {
		return nil, fmt.Errorf("fhir/gen: StructureDefinition %q not in bundle", fhirName)
	}
	typ, err := model.BuildType(sd)
	if err != nil {
		return nil, fmt.Errorf("fhir/gen: build model for %s: %w", fhirName, err)
	}
	pt := plan.PlanType(typ, plan.Options{SkipBaseMembers: true})
	src, err := emit.Emit(emit.File{Package: release, Types: []plan.PlannedType{pt}})
	if err != nil {
		return nil, fmt.Errorf("fhir/gen: emit %s: %w", fhirName, err)
	}
	return src, nil
}
