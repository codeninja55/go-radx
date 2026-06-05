package emit

import (
	"flag"
	"go/format"
	"os"
	"path/filepath"
	"testing"

	"github.com/codeninja55/go-radx/fhir/internal/gen/loader"
	"github.com/codeninja55/go-radx/fhir/internal/gen/model"
	"github.com/codeninja55/go-radx/fhir/internal/gen/plan"
)

var updateGolden = flag.Bool("update", false, "rewrite emitter golden files")

const vendoredR5Dir = "../testdata/definitions/r5"

// TestEmitPeriodGolden emits the representative datatype end to end — load, model,
// plan, emit — and pins the formatted Go against a committed golden. The golden is
// the exact bytes the committed fhir/r5/period.go must contain, so a template drift
// is caught here before it reaches the generated tree.
func TestEmitPeriodGolden(t *testing.T) {
	got := emitPeriod(t)

	goldenPath := filepath.Join("testdata", "golden", "period.go.golden")
	if *updateGolden {
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden %s: %v", goldenPath, err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s (run `go test ./fhir/internal/gen/emit -update` to create it): %v", goldenPath, err)
	}
	if string(got) != string(want) {
		t.Errorf("emitted Period drifted from golden %s.\nwant:\n%s\ngot:\n%s", goldenPath, string(want), string(got))
	}
}

// TestEmitIsGofmtStable asserts the emitter's own output is already gofmt-clean:
// running gofmt over it is a no-op. If the emitter produced un-formatted source, the
// committed file and a regenerated one could differ only by formatting, breaking the
// byte-for-byte property.
func TestEmitIsGofmtStable(t *testing.T) {
	got := emitPeriod(t)
	reformatted, err := format.Source(got)
	if err != nil {
		t.Fatalf("gofmt the emitted source: %v", err)
	}
	if string(reformatted) != string(got) {
		t.Error("emitted source is not gofmt-stable; the emitter must format its output")
	}
}

// TestEmitIsDeterministic asserts two emits of the same plan produce identical bytes,
// the property that makes regeneration reproducible.
func TestEmitIsDeterministic(t *testing.T) {
	a := emitPeriod(t)
	b := emitPeriod(t)
	if string(a) != string(b) {
		t.Error("two emits of the same plan differ; output must be deterministic")
	}
}

// emitPeriod runs the full back-half pipeline for Period and returns the formatted Go.
func emitPeriod(t *testing.T) []byte {
	t.Helper()
	bundle, err := loader.Load(vendoredR5Dir)
	if err != nil {
		t.Fatalf("load vendored R5 bundle: %v", err)
	}
	sd, ok := bundle.StructureDefinition("Period")
	if !ok {
		t.Fatal("Period not in bundle")
	}
	typ, err := model.BuildType(sd)
	if err != nil {
		t.Fatalf("BuildType(Period): %v", err)
	}
	pt := plan.PlanType(typ, plan.Options{SkipBaseMembers: true})
	src, err := Emit(File{Package: "r5", Types: []plan.PlannedType{pt}})
	if err != nil {
		t.Fatalf("Emit(Period): %v", err)
	}
	return src
}
