package model

import (
	"testing"

	"github.com/codeninja55/go-radx/fhir/internal/gen/loader"
)

func TestClassify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sd   *loader.StructureDefinition
		want Kind
	}{
		{"nil", nil, KindUnknown},
		{"primitive boolean", &loader.StructureDefinition{Name: "boolean", Kind: "primitive-type"}, KindPrimitive},
		{"complex Period", &loader.StructureDefinition{Name: "Period", Kind: "complex-type"}, KindComplexType},
		{"resource Patient", &loader.StructureDefinition{Name: "Patient", Kind: "resource"}, KindResource},
		{"logical is unknown", &loader.StructureDefinition{Name: "Definition", Kind: "logical"}, KindUnknown},
		{"empty kind is unknown", &loader.StructureDefinition{Name: "x"}, KindUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Classify(tt.sd); got != tt.want {
				t.Errorf("Classify(%v) = %v, want %v", tt.sd, got, tt.want)
			}
		})
	}
}

func TestKindString(t *testing.T) {
	t.Parallel()

	tests := map[Kind]string{
		KindUnknown:     "unknown",
		KindPrimitive:   "primitive-type",
		KindComplexType: "complex-type",
		KindResource:    "resource",
	}
	for k, want := range tests {
		if got := k.String(); got != want {
			t.Errorf("Kind(%d).String() = %q, want %q", k, got, want)
		}
	}
}

func TestSystemPrimitive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code     string
		wantName string
		wantOK   bool
	}{
		{"http://hl7.org/fhirpath/System.String", "string", true},
		{"http://hl7.org/fhirpath/System.Boolean", "boolean", true},
		{"http://hl7.org/fhirpath/System.Decimal", "decimal", true},
		{"string", "", false},
		{"CodeableConcept", "", false},
		{"http://hl7.org/fhirpath/System.", "", false},
	}
	for _, tt := range tests {
		gotName, gotOK := SystemPrimitive(tt.code)
		if gotName != tt.wantName || gotOK != tt.wantOK {
			t.Errorf("SystemPrimitive(%q) = (%q, %v), want (%q, %v)", tt.code, gotName, gotOK, tt.wantName, tt.wantOK)
		}
	}
}

func TestCardinality(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		card         Cardinality
		wantRequired bool
		wantRepeats  bool
	}{
		{"optional scalar 0..1", Cardinality{Min: 0, Max: "1"}, false, false},
		{"required scalar 1..1", Cardinality{Min: 1, Max: "1"}, true, false},
		{"optional slice 0..*", Cardinality{Min: 0, Max: "*"}, false, true},
		{"required slice 1..*", Cardinality{Min: 1, Max: "*"}, true, true},
		{"forbidden 0..0", Cardinality{Min: 0, Max: "0"}, false, false},
		{"bounded multi 0..5", Cardinality{Min: 0, Max: "5"}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.card.Required(); got != tt.wantRequired {
				t.Errorf("Required() = %v, want %v", got, tt.wantRequired)
			}
			if got := tt.card.Repeats(); got != tt.wantRepeats {
				t.Errorf("Repeats() = %v, want %v", got, tt.wantRepeats)
			}
		})
	}
}
