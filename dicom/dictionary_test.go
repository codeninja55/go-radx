package dicom

import "testing"

func TestLookupExact(t *testing.T) {
	info, ok := Lookup(TagPatientName)
	if !ok {
		t.Fatal("PatientName should be in the dictionary")
	}
	if info.Keyword != "PatientName" || info.VR != VRPN {
		t.Errorf("Lookup(PatientName) = %+v, want keyword PatientName VR PN", info)
	}
}

func TestLookupRepeatingGroupMask(t *testing.T) {
	// (6002,3000) is Overlay Data in the repeating 60xx group; it must resolve, not
	// degrade to unknown (Codex DCM-012).
	info, ok := Lookup(NewTag(0x6002, 0x3000))
	if !ok {
		t.Fatal("(6002,3000) should resolve via the 60xx mask")
	}
	if info.Keyword != "OverlayData" {
		t.Errorf("Lookup((6002,3000)).Keyword = %q, want OverlayData", info.Keyword)
	}
}

func TestLookupUnknown(t *testing.T) {
	if _, ok := Lookup(NewTag(0x0011, 0x1234)); ok {
		t.Error("a genuinely unknown private tag should return ok == false")
	}
}

func TestLookupKeyword(t *testing.T) {
	got, ok := LookupKeyword("StudyInstanceUID")
	if !ok || got != TagStudyInstanceUID {
		t.Errorf("LookupKeyword(StudyInstanceUID) = (%s,%v), want (%s,true)", got, ok, TagStudyInstanceUID)
	}
	if _, ok := LookupKeyword("NotARealKeyword"); ok {
		t.Error("unknown keyword should return ok == false, not panic")
	}
}

func TestLookupKeywordTagPanicsOnTypo(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("LookupKeywordTag should panic on a keyword not in the dictionary")
		}
	}()
	_ = LookupKeywordTag("DefinitelyNotAKeyword")
}
