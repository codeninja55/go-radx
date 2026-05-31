package dicom

import "fmt"

// ValueError reports a value that does not conform to its VR (bad UID, bad date,
// odd binary length, over-long PN component). It names the tag and VR; it never
// carries the offending PHI value (PRD §9.1).
type ValueError struct {
	Tag Tag
	VR  VR
	Msg string
}

func (e *ValueError) Error() string {
	if e.Tag != 0 {
		return fmt.Sprintf("dicom: invalid %s at %s %s: %s", e.VR, keywordFor(e.Tag), e.Tag, e.Msg)
	}
	return fmt.Sprintf("dicom: invalid %s: %s", e.VR, e.Msg)
}

// keywordFor renders a tag's keyword for diagnostics; falls back to "" if unknown.
// Task 1.5 enriches this to resolve the keyword through Lookup once the dictionary
// exists; until then it is deliberately empty so this file compiles standalone.
func keywordFor(Tag) string { return "" }
