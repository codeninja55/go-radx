package gen

import (
	"errors"
	"fmt"
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

// ErrNotImplemented is returned by Generate while the pipeline stages are still
// being built out increment by increment. It is a sentinel so the CLI can report
// the scaffold state distinctly from a real failure.
var ErrNotImplemented = errors.New("fhir/gen: generator pipeline not implemented")

// Generate runs the full generator pipeline for one release: load and verify the
// bundle, build the model, plan, and emit the generated Go. It is the single
// library entry point the CLI and the go:generate directive call.
//
// The pipeline stages (loader, model, planner, emitter) are filled in by later
// increments; until then Generate returns ErrNotImplemented so the package
// compiles and the wiring is exercisable end to end.
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
	return ErrNotImplemented
}
