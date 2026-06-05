package gen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codeninja55/go-radx/fhir/internal/gen/loader"
)

// committedR5Dir is the committed generated R5 package, relative to this package
// directory (fhir/internal/gen). The regeneration gate diffs against the files here.
const committedR5Dir = "../../r5"

// TestRegenerationByteForByte regenerates the full R5 generated tree into a temp dir
// and asserts each generated file is byte-for-byte identical to the committed file
// under fhir/r5. It is the executable form of the "generated, never hand-edited"
// property over the whole tree (every resource, every datatype, the base types, and
// the registry): a hand edit to a committed generated file, or a generator change
// that was not regenerated, makes this test fail. The gate runs the real pipeline
// (load, model, plan, emit, write) against the pinned bundle, so it also exercises
// Generate end to end. The committed hand-written skeleton files are not in the
// generated set, so they are not diffed here.
func TestRegenerationByteForByte(t *testing.T) {
	tmp := t.TempDir()
	cfg := Config{
		Release:        "r5",
		DefinitionsDir: vendoredR5Dir,
		OutputDir:      tmp,
	}
	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	bundle, err := loader.Load(vendoredR5Dir)
	if err != nil {
		t.Fatalf("load vendored R5 bundle: %v", err)
	}

	for _, name := range GeneratedFileNames(bundle) {
		t.Run(name, func(t *testing.T) {
			regenerated, err := os.ReadFile(filepath.Join(tmp, name))
			if err != nil {
				t.Fatalf("read regenerated %s: %v", name, err)
			}
			committed, err := os.ReadFile(filepath.Join(committedR5Dir, name))
			if err != nil {
				t.Fatalf("read committed %s (run `mise run gen:fhir-r5`): %v", name, err)
			}
			if string(regenerated) != string(committed) {
				t.Errorf("committed %s differs from a fresh regeneration; the file was hand-edited "+
					"or the generator changed without regenerating. Run `mise run gen:fhir-r5` and commit the result.",
					name)
			}
		})
	}
}
