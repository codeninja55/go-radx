package dicom

import "testing"

func TestVRString(t *testing.T) {
	tests := map[VR]string{
		VRAE: "AE", VRPN: "PN", VRSQ: "SQ", VRUI: "UI",
		VROB: "OB", VROV: "OV", VRUV: "UV", VRUN: "UN",
	}
	for vr, want := range tests {
		if got := vr.String(); got != want {
			t.Errorf("VR(%d).String() = %q, want %q", vr, got, want)
		}
	}
}

func TestVRStandardCount(t *testing.T) {
	// 34 standard VRs are enumerated as iota constants VRAE..VRUV.
	if int(VRUV)+1 != 34 {
		t.Errorf("expected 34 standard VRs (VRAE..VRUV), last index = %d", VRUV)
	}
}

func TestVRIs32BitLength(t *testing.T) {
	long := []VR{VROB, VROW, VROD, VROF, VROL, VROV, VRSQ, VRUC, VRUR, VRUT, VRUN}
	for _, vr := range long {
		if !vr.Is32BitLength() {
			t.Errorf("%s should use the 4-byte explicit-VR length form", vr)
		}
	}
	short := []VR{VRAE, VRCS, VRDA, VRPN, VRUS, VRSS, VRUI, VRDS, VRIS}
	for _, vr := range short {
		if vr.Is32BitLength() {
			t.Errorf("%s should use the 2-byte length form", vr)
		}
	}
}

func TestVRPadByte(t *testing.T) {
	// UI pads with NULL; other string VRs pad with SPACE; binary VRs are not padded.
	if b, ok := VRUI.PadByte(); !ok || b != 0x00 {
		t.Errorf("VRUI pad = (%#x,%v), want (0x00,true)", b, ok)
	}
	for _, vr := range []VR{VRAE, VRCS, VRDA, VRDS, VRIS, VRLO, VRPN} {
		if b, ok := vr.PadByte(); !ok || b != 0x20 {
			t.Errorf("%s pad = (%#x,%v), want (0x20,true)", vr, b, ok)
		}
	}
	for _, vr := range []VR{VRUS, VRFL, VROB, VRSQ} {
		if _, ok := vr.PadByte(); ok {
			t.Errorf("%s should not declare a pad byte", vr)
		}
	}
}
