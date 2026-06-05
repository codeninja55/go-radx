package model

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/codeninja55/go-radx/fhir/internal/gen/loader"
)

// updateGolden, set with -update, rewrites the committed golden snapshots from the
// current model output. Run it deliberately when the IR shape changes on purpose;
// the diff in the regenerated golden is the reviewable record of the change.
var updateGolden = flag.Bool("update", false, "rewrite model golden snapshots")

// vendoredR5Dir is the committed, checksum-pinned R5 bundle, relative to this
// package directory. The golden test loads the real bundle so the snapshot pins
// the model against actual HL7 definitions, not a hand-built stub.
const vendoredR5Dir = "../testdata/definitions/r5"

// goldenTypes are the representative R5 types pinned by the golden snapshot, chosen
// to exercise every IR feature against real definitions:
//   - Observation: deep backbones, a contentReference-grafted backbone
//     (component.referenceRange reuses #Observation.referenceRange), and choice
//     elements (value[x]) — the structural fix for empty backbones.
//   - Period: a small complex datatype.
//   - boolean: a primitive type, proving classification across all three kinds.
var goldenTypes = []string{"Observation", "Period", "boolean"}

func TestModelGoldenSnapshot(t *testing.T) {
	bundle, err := loader.Load(vendoredR5Dir)
	if err != nil {
		t.Fatalf("load vendored R5 bundle: %v", err)
	}

	for _, name := range goldenTypes {
		t.Run(name, func(t *testing.T) {
			sd, ok := bundle.StructureDefinition(name)
			if !ok {
				t.Fatalf("StructureDefinition %q not in bundle", name)
			}
			typ, err := BuildType(sd)
			if err != nil {
				t.Fatalf("BuildType(%s): %v", name, err)
			}
			got := Snapshot(typ)

			goldenPath := filepath.Join("testdata", "golden", name+".snapshot.txt")
			if *updateGolden {
				if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
					t.Fatalf("write golden %s: %v", goldenPath, err)
				}
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden %s (run `go test ./fhir/internal/gen/model -update` to create it): %v", goldenPath, err)
			}
			if got != string(want) {
				t.Errorf("model snapshot for %s drifted from golden %s.\n"+
					"Run `go test ./fhir/internal/gen/model -update` to inspect and accept the change.\n%s",
					name, goldenPath, firstDiff(string(want), got))
			}
		})
	}
}

// TestModelGoldenObservationProperties cross-checks the golden-pinned Observation
// against the load-bearing structural invariants in code, so an accidental
// -update that hides a regression is still caught. The golden pins the exact
// shape; this pins the property that matters most (the contentReference graft is
// non-empty — the empty-backbone fix on real definitions).
func TestModelGoldenObservationProperties(t *testing.T) {
	t.Parallel()

	bundle, err := loader.Load(vendoredR5Dir)
	if err != nil {
		t.Fatalf("load vendored R5 bundle: %v", err)
	}
	sd, ok := bundle.StructureDefinition("Observation")
	if !ok {
		t.Fatal("Observation not in bundle")
	}
	typ, err := BuildType(sd)
	if err != nil {
		t.Fatalf("BuildType(Observation): %v", err)
	}

	refRange, ok := walk(typ.Root, "component", "referenceRange")
	if !ok {
		t.Fatal("Observation.component.referenceRange not in tree")
	}
	if len(refRange.Children) == 0 {
		t.Fatal("Observation.component.referenceRange is an empty backbone (the FHIR-006 defect on real definitions)")
	}
	if _, ok := refRange.Child("low"); !ok {
		t.Error("grafted component.referenceRange should carry a low child")
	}

	value, ok := walk(typ.Root, "value[x]")
	if !ok || !value.IsChoice {
		t.Fatal("Observation.value[x] should be a detected choice element")
	}
	if len(value.Types) < 2 {
		t.Errorf("value[x] should carry multiple branch types, got %d", len(value.Types))
	}
}

// walk follows a chain of child names from the root.
func walk(root *Element, names ...string) (*Element, bool) {
	node := root
	for _, name := range names {
		c, ok := node.Child(name)
		if !ok {
			return nil, false
		}
		node = c
	}
	return node, true
}

// firstDiff returns a short pointer to the first differing line between want and
// got, so a golden mismatch reports where it drifted rather than dumping both
// full trees.
func firstDiff(want, got string) string {
	wl := splitLines(want)
	gl := splitLines(got)
	for i := 0; i < len(wl) && i < len(gl); i++ {
		if wl[i] != gl[i] {
			return "first diff at line " + itoa(i+1) + ":\n  want: " + wl[i] + "\n   got: " + gl[i]
		}
	}
	if len(wl) != len(gl) {
		return "line count differs: want " + itoa(len(wl)) + ", got " + itoa(len(gl))
	}
	return "(no line difference; trailing whitespace?)"
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
