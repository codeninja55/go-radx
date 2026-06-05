package emit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codeninja55/go-radx/fhir/internal/gen/plan"
)

// descriptorFixture is a representative pair of validation descriptors: one resource with
// a required scalar, a choice group, and a required-binding code, and one (Bundle) flagged
// for the hand-written extra checks. It exercises every branch of the descriptor template.
func descriptorFixture() []plan.ValidationDescriptor {
	return []plan.ValidationDescriptor{
		{
			GoName:   "Sample",
			FHIRName: "Sample",
			Required: []plan.RequiredField{
				{GoName: "Status", Path: "Sample.status"},
				{GoName: "Members", Path: "Sample.members", Repeats: true},
			},
			Choices: []plan.ChoiceCheck{
				{Fields: []string{"ValueQuantity", "ValueString"}, Path: "Sample.value[x]"},
			},
			Bindings: []plan.BindingCheck{
				{GoName: "Gender", Validator: "validAdministrativeGender", EnumName: "AdministrativeGender", Path: "Sample.gender"},
				{GoName: "Categories", Validator: "validSampleCategory", EnumName: "SampleCategory", Path: "Sample.category", Repeats: true},
			},
			Summary: []plan.SummaryFlag{
				{JSONName: "text", IsText: true},
				{JSONName: "status", IsSummary: true, IsMandatory: true, IsModifier: true},
				{JSONName: "valueQuantity", IsSummary: true},
				{JSONName: "valueString", IsSummary: true},
				{JSONName: "note"},
			},
		},
		{
			GoName:   "Bundle",
			FHIRName: "Bundle",
			Required: []plan.RequiredField{{GoName: "Type", Path: "Bundle.type"}},
			HasExtra: true,
			Summary: []plan.SummaryFlag{
				{JSONName: "type", IsSummary: true, IsMandatory: true},
				{JSONName: "total", IsSummary: true, IsCount: true},
				{JSONName: "entry", IsSummary: true},
			},
		},
	}
}

// TestEmitDescriptorsGolden emits the validation-descriptor file and pins it against a
// committed golden, so a drift in the generated registration init() — the required,
// choice, binding, and extra-hook wiring fhir.Validate consumes — is caught here.
func TestEmitDescriptorsGolden(t *testing.T) {
	got, err := EmitDescriptors(Descriptors{Package: "r5", Descriptors: descriptorFixture()})
	if err != nil {
		t.Fatalf("EmitDescriptors: %v", err)
	}

	goldenPath := filepath.Join("testdata", "golden", "validation_descriptors.go.golden")
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
		t.Errorf("emitted descriptors drifted from golden %s.\nwant:\n%s\ngot:\n%s", goldenPath, string(want), string(got))
	}
}

// TestEmitDescriptorsIsDeterministic asserts two emits of the same descriptors produce
// identical bytes, the property that makes regeneration reproducible.
func TestEmitDescriptorsIsDeterministic(t *testing.T) {
	a, err := EmitDescriptors(Descriptors{Package: "r5", Descriptors: descriptorFixture()})
	if err != nil {
		t.Fatalf("EmitDescriptors: %v", err)
	}
	b, err := EmitDescriptors(Descriptors{Package: "r5", Descriptors: descriptorFixture()})
	if err != nil {
		t.Fatalf("EmitDescriptors: %v", err)
	}
	if string(a) != string(b) {
		t.Error("two emits of the same descriptors differ; output must be deterministic")
	}
}
