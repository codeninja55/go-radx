package dicom

//go:generate go run ./gen/gentags

// TagInfo is a dictionary entry resolved from a Tag.
type TagInfo struct {
	Keyword string // e.g. "PatientName"
	VR      VR     // dictionary VR; ambiguous VRs use the ambiguous placeholders
	VM      string // value multiplicity, e.g. "1", "1-n", "2"
	Name    string // human-readable name from PS3.6
}

// repeatingEntry holds a masked dictionary entry for a repeating group.
type repeatingEntry struct {
	mask, value uint32 // (tag & mask) == value matches
	info        TagInfo
}

// dictLen reports the exact-entry count, used by the dictionary coverage test.
func dictLen() int { return len(tagDict) }
