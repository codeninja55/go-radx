package dicom

import "testing"

func TestTransferSyntaxPredicates(t *testing.T) {
	tests := []struct {
		ts                                   TransferSyntax
		implicit, bigEndian, deflated, encap bool
	}{
		{ImplicitVRLittleEndian, true, false, false, false},
		{ExplicitVRLittleEndian, false, false, false, false},
		{DeflatedExplicitVRLittleEndian, false, false, true, false},
		{ExplicitVRBigEndian, false, true, false, false},
		{RLELossless, false, false, false, true},
		{JPEGBaseline8Bit, false, false, false, true},
	}
	for _, tc := range tests {
		t.Run(tc.ts.Name(), func(t *testing.T) {
			if got := tc.ts.IsImplicitVR(); got != tc.implicit {
				t.Errorf("IsImplicitVR() = %v, want %v", got, tc.implicit)
			}
			if got := tc.ts.IsBigEndian(); got != tc.bigEndian {
				t.Errorf("IsBigEndian() = %v, want %v", got, tc.bigEndian)
			}
			if got := tc.ts.IsDeflated(); got != tc.deflated {
				t.Errorf("IsDeflated() = %v, want %v", got, tc.deflated)
			}
			if got := tc.ts.IsEncapsulated(); got != tc.encap {
				t.Errorf("IsEncapsulated() = %v, want %v", got, tc.encap)
			}
		})
	}
}

func TestTransferSyntaxName(t *testing.T) {
	if got := ExplicitVRLittleEndian.Name(); got != "Explicit VR Little Endian" {
		t.Errorf("Name() = %q, want Explicit VR Little Endian", got)
	}
	if got := ImplicitVRLittleEndian.Name(); got != "Implicit VR Little Endian" {
		t.Errorf("Name() = %q, want Implicit VR Little Endian", got)
	}
	// An unregistered syntax falls back to the raw UID.
	if got := TransferSyntax("1.2.3.4.5").Name(); got != "1.2.3.4.5" {
		t.Errorf("Name() = %q, want the raw UID", got)
	}
}

func TestTransferSyntaxByteOrder(t *testing.T) {
	if ExplicitVRBigEndian.byteOrder() == nil {
		t.Fatal("byteOrder must not be nil for Big Endian")
	}
	if got := ExplicitVRBigEndian.byteOrder().Uint16([]byte{0x12, 0x34}); got != 0x1234 {
		t.Errorf("Big Endian Uint16 = %#x, want 0x1234", got)
	}
	if got := ExplicitVRLittleEndian.byteOrder().Uint16([]byte{0x34, 0x12}); got != 0x1234 {
		t.Errorf("Little Endian Uint16 = %#x, want 0x1234", got)
	}
}
