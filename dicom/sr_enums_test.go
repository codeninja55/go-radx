package dicom

import "testing"

func TestValueTypeString(t *testing.T) {
	tests := map[ValueType]string{
		ValueTypeContainer: "CONTAINER",
		ValueTypeText:      "TEXT",
		ValueTypeCode:      "CODE",
		ValueTypeNum:       "NUM",
		ValueTypePName:     "PNAME",
		ValueTypeDate:      "DATE",
		ValueTypeTime:      "TIME",
		ValueTypeDateTime:  "DATETIME",
		ValueTypeUIDRef:    "UIDREF",
		ValueTypeComposite: "COMPOSITE",
		ValueTypeImage:     "IMAGE",
		ValueTypeSCoord:    "SCOORD",
		ValueTypeSCoord3D:  "SCOORD3D",
		ValueTypeTCoord:    "TCOORD",
		ValueTypeWaveform:  "WAVEFORM",
	}
	if len(tests) != 15 {
		t.Fatalf("expected 15 ValueType defined terms, listed %d", len(tests))
	}
	for vt, want := range tests {
		if got := vt.String(); got != want {
			t.Errorf("ValueType(%d).String() = %q, want %q", vt, got, want)
		}
	}
}

func TestParseValueType(t *testing.T) {
	for _, want := range []ValueType{
		ValueTypeContainer, ValueTypeText, ValueTypeCode, ValueTypeNum,
		ValueTypePName, ValueTypeDate, ValueTypeTime, ValueTypeDateTime,
		ValueTypeUIDRef, ValueTypeComposite, ValueTypeImage, ValueTypeSCoord,
		ValueTypeSCoord3D, ValueTypeTCoord, ValueTypeWaveform,
	} {
		got, err := parseValueType(want.String())
		if err != nil {
			t.Errorf("parseValueType(%q): %v", want.String(), err)
			continue
		}
		if got != want {
			t.Errorf("parseValueType(%q) = %v, want %v", want.String(), got, want)
		}
	}
}

func TestParseValueTypeInvalid(t *testing.T) {
	for _, bad := range []string{"", "container", "FOO", "NUMBER"} {
		if _, err := parseValueType(bad); err == nil {
			t.Errorf("parseValueType(%q) = nil error, want rejection", bad)
		}
	}
}

func TestRelationshipTypeString(t *testing.T) {
	tests := map[RelationshipType]string{
		RelationshipContains:      "CONTAINS",
		RelationshipHasObsContext: "HAS OBS CONTEXT",
		RelationshipHasConceptMod: "HAS CONCEPT MOD",
		RelationshipHasProperties: "HAS PROPERTIES",
		RelationshipHasAcqContext: "HAS ACQ CONTEXT",
		RelationshipInferredFrom:  "INFERRED FROM",
		RelationshipSelectedFrom:  "SELECTED FROM",
	}
	if len(tests) != 7 {
		t.Fatalf("expected 7 RelationshipType defined terms, listed %d", len(tests))
	}
	for rt, want := range tests {
		if got := rt.String(); got != want {
			t.Errorf("RelationshipType(%d).String() = %q, want %q", rt, got, want)
		}
	}
}

func TestParseRelationshipType(t *testing.T) {
	for _, want := range []RelationshipType{
		RelationshipContains, RelationshipHasObsContext, RelationshipHasConceptMod,
		RelationshipHasProperties, RelationshipHasAcqContext, RelationshipInferredFrom,
		RelationshipSelectedFrom,
	} {
		got, err := parseRelationshipType(want.String())
		if err != nil {
			t.Errorf("parseRelationshipType(%q): %v", want.String(), err)
			continue
		}
		if got != want {
			t.Errorf("parseRelationshipType(%q) = %v, want %v", want.String(), got, want)
		}
	}
}

func TestParseRelationshipTypeInvalid(t *testing.T) {
	for _, bad := range []string{"", "contains", "HASOBSCONTEXT", "FOO"} {
		if _, err := parseRelationshipType(bad); err == nil {
			t.Errorf("parseRelationshipType(%q) = nil error, want rejection", bad)
		}
	}
}
