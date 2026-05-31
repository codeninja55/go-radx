package dicom

import "fmt"

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

// Lookup resolves a tag through the standard dictionary, resolving 60xx/50xx/5xxx
// repeating groups by mask. ok is false for genuinely unknown tags.
func Lookup(t Tag) (TagInfo, bool) {
	if info, ok := tagDict[t]; ok {
		return info, true
	}
	for _, re := range repeatingDict {
		if uint32(t)&re.mask == re.value {
			return re.info, true
		}
	}
	return TagInfo{}, false
}

// keywordTag is the reverse index built once from tagDict at init.
var keywordTag = func() map[string]Tag {
	m := make(map[string]Tag, len(tagDict))
	for tag, info := range tagDict {
		if info.Keyword != "" {
			m[info.Keyword] = tag
		}
	}
	return m
}()

// LookupKeyword resolves a keyword to its canonical tag for dynamic input,
// returning ok == false for an unknown keyword.
func LookupKeyword(keyword string) (Tag, bool) {
	t, ok := keywordTag[keyword]
	return t, ok
}

// LookupKeywordTag resolves a compile-time-literal keyword to its tag. It panics
// only on a keyword not in the standard dictionary — a programmer error, never
// reachable from external input.
func LookupKeywordTag(keyword string) Tag {
	t, ok := keywordTag[keyword]
	if !ok {
		panic(fmt.Sprintf("dicom: LookupKeywordTag: unknown keyword %q", keyword))
	}
	return t
}
