package gen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codeninja55/go-radx/fhir/internal/gen/loader"
)

// regenerationCases pairs each release with its committed generated package and the
// vendored bundle it is produced from, both relative to this package directory
// (fhir/internal/gen). The byte-for-byte gate runs over every case, so a hand edit
// or an un-regenerated generator change fails for whichever release it touched.
var regenerationCases = []struct {
	release      string
	committedDir string
	vendoredDir  string
	genTask      string
}{
	{release: "r4", committedDir: "../../r4", vendoredDir: vendoredR4Dir, genTask: "gen:fhir-r4"},
	{release: "r5", committedDir: "../../r5", vendoredDir: vendoredR5Dir, genTask: "gen:fhir-r5"},
}

// TestRegenerationByteForByte regenerates each release's full generated tree into a
// temp dir and asserts every generated file is byte-for-byte identical to the
// committed file under fhir/<release>. It is the executable form of the "generated,
// never hand-edited" property over the whole tree (every resource, every datatype,
// the base types, the enums, the registry, and the validation descriptors): a hand
// edit to a committed generated file, or a generator change that was not regenerated,
// makes this test fail. The gate runs the real pipeline (load, model, plan, emit,
// write) against each pinned bundle, so it also exercises Generate end to end. The
// committed hand-written companion files (the Bundle builders, reference-integrity
// helpers, and Bundle validation hook) are not in the generated set, so they are not
// diffed here.
func TestRegenerationByteForByte(t *testing.T) {
	for _, tc := range regenerationCases {
		t.Run(tc.release, func(t *testing.T) {
			tmp := t.TempDir()
			cfg := Config{
				Release:        tc.release,
				DefinitionsDir: tc.vendoredDir,
				OutputDir:      tmp,
			}
			if err := Generate(cfg); err != nil {
				t.Fatalf("Generate: %v", err)
			}

			bundle, err := loader.Load(tc.vendoredDir)
			if err != nil {
				t.Fatalf("load vendored %s bundle: %v", tc.release, err)
			}

			for _, name := range GeneratedFileNames(bundle) {
				t.Run(name, func(t *testing.T) {
					regenerated, err := os.ReadFile(filepath.Join(tmp, name))
					if err != nil {
						t.Fatalf("read regenerated %s: %v", name, err)
					}
					committed, err := os.ReadFile(filepath.Join(tc.committedDir, name))
					if err != nil {
						t.Fatalf("read committed %s (run `mise run %s`): %v", name, tc.genTask, err)
					}
					if string(regenerated) != string(committed) {
						t.Errorf("committed %s differs from a fresh regeneration; the file was hand-edited "+
							"or the generator changed without regenerating. Run `mise run %s` and commit the result.",
							name, tc.genTask)
					}
				})
			}
		})
	}
}
