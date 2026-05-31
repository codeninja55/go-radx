package dicom

import "testing"

func TestGeneratedTagConstants(t *testing.T) {
	tests := map[string]struct {
		got  Tag
		want Tag
	}{
		"PatientID":         {TagPatientID, 0x00100020},
		"PatientName":       {TagPatientName, 0x00100010},
		"StudyInstanceUID":  {TagStudyInstanceUID, 0x0020000D},
		"SeriesInstanceUID": {TagSeriesInstanceUID, 0x0020000E},
		"SOPClassUID":       {TagSOPClassUID, 0x00080016},
		"SOPInstanceUID":    {TagSOPInstanceUID, 0x00080018},
		"PixelData":         {TagPixelData, 0x7FE00010},
	}
	for name, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("Tag%s = %#08x, want %#08x", name, uint32(tc.got), uint32(tc.want))
		}
	}
}

func TestDictionaryEntryCount(t *testing.T) {
	// The standard data dictionary is ~5,189 entries (reference doc, conformance).
	if n := dictLen(); n < 5000 {
		t.Errorf("dictionary has %d entries, expected ~5,189", n)
	}
}

func TestGeneratedUIDNames(t *testing.T) {
	if got := UID("1.2.840.10008.1.2").Name(); got != "Implicit VR Little Endian" {
		t.Errorf("Name() = %q, want Implicit VR Little Endian", got)
	}
}
