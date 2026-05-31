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

func TestNewBytesCopiesInput(t *testing.T) {
	src := []byte{1, 2, 3, 4}
	v := NewBytes(VROB, src)
	src[0] = 0xFF // mutate the caller's slice after construction
	bv, ok := v.(*Bytes)
	if !ok {
		t.Fatal("NewBytes should return *Bytes")
	}
	if got := bv.Bytes(); got[0] != 1 {
		t.Errorf("value aliased the caller's slice: got[0] = %#x, want 1 (Codex DCM-016)", got[0])
	}
}

func TestBytesAccessorReturnsCopy(t *testing.T) {
	v := NewBytes(VROB, []byte{1, 2, 3, 4})
	bv := v.(*Bytes)
	out := bv.Bytes()
	out[0] = 0xFF // mutate the returned slice
	if bv.Bytes()[0] != 1 {
		t.Error("Bytes() returned an internal slice that callers can mutate (Codex DCM-016)")
	}
}

func TestBytesEncodedLenEvenPadded(t *testing.T) {
	// Odd OB length pads to even with a trailing NULL.
	if got := NewBytes(VROB, []byte{1, 2, 3}).EncodedLen(binary.LittleEndian); got != 4 {
		t.Errorf("odd OB EncodedLen = %d, want 4 (even-padded)", got)
	}
}
