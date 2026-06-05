package gen

import (
	"os"
	"path/filepath"
	"testing"
)

// committedR5Dir is the committed generated R5 package, relative to this package
// directory (fhir/internal/gen). The regeneration gate diffs against the files here.
const committedR5Dir = "../../r5"

// TestRegenerationByteForByte regenerates every generated R5 file into a temp dir and
// asserts each is byte-for-byte identical to the committed file under fhir/r5. It is
// the executable form of the "generated, never hand-edited" property: a hand edit to
// a committed generated file, or a generator change that was not regenerated, makes
// this test fail. The gate runs the real pipeline (load, model, plan, emit, write)
// against the pinned bundle, so it also exercises Generate end to end.
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

	// Every file Generate writes — the representative datatypes, the representative
	// resources, and the per-release registry — must match its committed counterpart.
	var fileNames []string
	for _, dt := range representativeDatatypes {
		fileNames = append(fileNames, dt.fileName)
	}
	for _, res := range representativeResources {
		fileNames = append(fileNames, res.fileName)
	}
	fileNames = append(fileNames, "registry.go")

	for _, name := range fileNames {
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
					"or the generator changed without regenerating. Run `mise run gen:fhir-r5` and commit the result.\n"+
					"--- committed ---\n%s\n--- regenerated ---\n%s",
					name, string(committed), string(regenerated))
			}
		})
	}
}
