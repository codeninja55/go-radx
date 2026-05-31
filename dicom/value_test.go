package dicom

import (
	"encoding/binary"
	"testing"
)

func TestStringsValueEncodedLenIsEven(t *testing.T) {
	// "Doe^Jane" is 8 bytes (even). "ABC" is 3 -> padded to 4 with SPACE.
	tests := []struct {
		vr   VR
		vals []string
		want uint32
	}{
		{VRPN, []string{"Doe^Jane"}, 8},
		{VRCS, []string{"ABC"}, 4},                // SPACE-padded
		{VRUI, []string{"1.2.840.10008.1.2"}, 18}, // 17 chars -> NULL-padded to 18
		{VRLO, []string{"a", "bb"}, 4},            // "a\bb" = 4 bytes, even
	}
	for _, tc := range tests {
		v := NewStrings(tc.vr, tc.vals...)
		if got := v.EncodedLen(binary.LittleEndian); got != tc.want {
			t.Errorf("NewStrings(%s,%v).EncodedLen = %d, want %d", tc.vr, tc.vals, got, tc.want)
		}
		if got := v.EncodedLen(binary.LittleEndian); got%2 != 0 {
			t.Errorf("%s encoded length %d is odd (Codex DCM-007)", tc.vr, got)
		}
	}
}

func TestIntsValueEncodedLen(t *testing.T) {
	if got := NewInts(VRUS, 1, 2, 3).EncodedLen(binary.LittleEndian); got != 6 {
		t.Errorf("US x3 EncodedLen = %d, want 6", got)
	}
	if got := NewInts(VRUL, 1).EncodedLen(binary.LittleEndian); got != 4 {
		t.Errorf("UL EncodedLen = %d, want 4", got)
	}
	if got := NewInts(VRSV, 1).EncodedLen(binary.LittleEndian); got != 8 {
		t.Errorf("SV EncodedLen = %d, want 8", got)
	}
}

func TestValueVR(t *testing.T) {
	if NewStrings(VRPN, "x").VR() != VRPN {
		t.Error("VR() should report PN")
	}
}
