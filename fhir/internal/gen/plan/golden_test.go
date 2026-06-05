package plan

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/codeninja55/go-radx/fhir/internal/gen/loader"
	"github.com/codeninja55/go-radx/fhir/internal/gen/model"
)

// updateGolden, set with -update, rewrites the committed planned-model snapshots from
// the current planner output. Run it deliberately when the planner's decisions change
// on purpose; the diff in the regenerated golden is the reviewable record.
var updateGolden = flag.Bool("update", false, "rewrite planned-model golden snapshots")

// vendoredR5Dir is the committed, checksum-pinned R5 bundle relative to this package,
// so the golden pins the plan against real HL7 definitions, not a hand-built stub.
const vendoredR5Dir = "../testdata/definitions/r5"

// goldenTypes are the representative R5 types pinned by the planned-model snapshot.
// Period exercises the optional-scalar-to-pointer decision end to end on a real
// datatype; the set grows as later increments add resources and backbones.
var goldenTypes = []string{"Period"}

func TestPlanGoldenSnapshot(t *testing.T) {
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
			typ, err := model.BuildType(sd)
			if err != nil {
				t.Fatalf("BuildType(%s): %v", name, err)
			}
			got := Snapshot(PlanType(typ, Options{SkipBaseMembers: true}))

			goldenPath := filepath.Join("testdata", "golden", name+".plan.txt")
			if *updateGolden {
				if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
					t.Fatalf("write golden %s: %v", goldenPath, err)
				}
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden %s (run `go test ./fhir/internal/gen/plan -update` to create it): %v", goldenPath, err)
			}
			if got != string(want) {
				t.Errorf("planned-model snapshot for %s drifted from golden %s.\n"+
					"Run `go test ./fhir/internal/gen/plan -update` to inspect and accept the change.\nwant:\n%s\ngot:\n%s",
					name, goldenPath, string(want), got)
			}
		})
	}
}
