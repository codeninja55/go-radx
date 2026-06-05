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

// TestEmitHumanNameGolden emits the representative primitive-extension datatype end
// to end and pins the formatted Go against a committed golden. HumanName carries the
// full primitive layer: scalar "_field" siblings (use, family) round-tripped through
// ordinary struct tags, null-aligned repeating siblings (given, prefix, suffix) with
// their generated MarshalJSON/UnmarshalJSON, and the FHIR-005 rule that its complex
// Period field gets no "_field" companion. A regression in any of those is caught
// here before it reaches the generated tree.
func TestEmitHumanNameGolden(t *testing.T) {
	got := emitType(t, "HumanName")

	goldenPath := filepath.Join("testdata", "golden", "human_name.go.golden")
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
		t.Errorf("emitted HumanName drifted from golden %s.\nwant:\n%s\ngot:\n%s", goldenPath, string(want), string(got))
	}
}

// TestEmitFlagGolden emits the representative resource end to end and pins the
// formatted Go against a committed golden. Beyond the struct, the golden carries the
// resource-specific output the resource shape adds: the resourceType constant, the
// ResourceType method, and the always-emit-resourceType MarshalJSON (FHIR-004), so a
// regression in any of those is caught here before it reaches the generated tree.
func TestEmitFlagGolden(t *testing.T) {
	got := emitResource(t, "Flag")

	goldenPath := filepath.Join("testdata", "golden", "flag.go.golden")
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
		t.Errorf("emitted Flag drifted from golden %s.\nwant:\n%s\ngot:\n%s", goldenPath, string(want), string(got))
	}
}

// TestEmitAnnotationGolden emits the representative choice-type datatype end to end and
// pins the formatted Go against a committed golden. Annotation carries the full choice
// layer alongside the primitive layer: a sealed AnnotationAuthor interface closed by an
// unexported marker, that marker implemented on each branch (Reference and the
// FHIRString wrapper), the suffixed storage fields (AuthorReference, AuthorString), an
// Author() getter, and mutually-exclusive setters that clear the siblings (FHIR-001),
// while its time/text primitives keep their "_field" siblings. A regression in the
// choice machinery is caught here before it reaches the generated tree.
func TestEmitAnnotationGolden(t *testing.T) {
	got := emitType(t, "Annotation")

	goldenPath := filepath.Join("testdata", "golden", "annotation.go.golden")
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
		t.Errorf("emitted Annotation drifted from golden %s.\nwant:\n%s\ngot:\n%s", goldenPath, string(want), string(got))
	}
}

// TestEmitEnumsGolden emits a representative pair of required-binding enums — one
// enumerable closed enum (AdministrativeGender) and one documented not-inlined open
// string (UCUMCodes) — and pins the formatted Go against a committed golden. The golden
// carries the full closed-enum shape (defined string type, const set, set-membership
// validator, strict-by-default ParseXxx, and the strict UnmarshalJSON that rejects an
// out-of-set code with fhir.ErrUnknownCode, FHIR-013) alongside the not-inlined
// boundary (a plain string with a documented reason, never an empty const set). A
// regression in the enum template is caught here before it reaches the generated tree.
func TestEmitEnumsGolden(t *testing.T) {
	enums := []plan.PlannedEnum{
		{
			GoName:      "AdministrativeGender",
			FHIRName:    "AdministrativeGender",
			ValueSetURL: "http://hl7.org/fhir/ValueSet/administrative-gender",
			Consts: []plan.EnumConst{
				{GoName: "AdministrativeGenderMale", Value: "male"},
				{GoName: "AdministrativeGenderFemale", Value: "female"},
				{GoName: "AdministrativeGenderOther", Value: "other"},
				{GoName: "AdministrativeGenderUnknown", Value: "unknown"},
			},
		},
		{
			GoName:           "UCUMCodes",
			FHIRName:         "UCUMCodes",
			ValueSetURL:      "http://hl7.org/fhir/ValueSet/ucum-units",
			NotInlined:       true,
			NotInlinedReason: "draws from code system http://unitsofmeasure.org, not vendored in the bundle",
		},
	}
	plan.SortPlannedEnums(enums)
	got, err := EmitEnums(Enums{Package: "r5", Enums: enums})
	if err != nil {
		t.Fatalf("EmitEnums: %v", err)
	}

	goldenPath := filepath.Join("testdata", "golden", "bindings.go.golden")
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
		t.Errorf("emitted enums drifted from golden %s.\nwant:\n%s\ngot:\n%s", goldenPath, string(want), string(got))
	}
}

// TestEmitRegistryGolden emits the per-release registry file and pins it against a
// committed golden, so a drift in the registry init() — the resourceType→factory
// wiring fhir.UnmarshalResource dispatches through — is caught here.
func TestEmitRegistryGolden(t *testing.T) {
	got, err := EmitRegistry(Registry{Package: "r5", Resources: []RegistryEntry{{GoName: "Flag"}}})
	if err != nil {
		t.Fatalf("EmitRegistry: %v", err)
	}

	goldenPath := filepath.Join("testdata", "golden", "registry.go.golden")
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
		t.Errorf("emitted registry drifted from golden %s.\nwant:\n%s\ngot:\n%s", goldenPath, string(want), string(got))
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
	return emitType(t, "Period")
}

// emitResource runs the full back-half pipeline for a resource and returns the
// formatted Go, sharing the datatype path so the resource-specific output is driven
// only by the planner's recorded kind.
func emitResource(t *testing.T, fhirName string) []byte {
	t.Helper()
	return emitType(t, fhirName)
}

// emitType loads the vendored bundle, builds the model, plans, and emits one type,
// returning its formatted Go source.
func emitType(t *testing.T, fhirName string) []byte {
	t.Helper()
	bundle, err := loader.Load(vendoredR5Dir)
	if err != nil {
		t.Fatalf("load vendored R5 bundle: %v", err)
	}
	sd, ok := bundle.StructureDefinition(fhirName)
	if !ok {
		t.Fatalf("%s not in bundle", fhirName)
	}
	typ, err := model.BuildType(sd)
	if err != nil {
		t.Fatalf("BuildType(%s): %v", fhirName, err)
	}
	pt := plan.PlanType(typ, plan.Options{})
	src, err := Emit(File{Package: "r5", Types: []plan.PlannedType{pt}})
	if err != nil {
		t.Fatalf("Emit(%s): %v", fhirName, err)
	}
	return src
}
