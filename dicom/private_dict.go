package dicom

// Private-creator dictionary (PS3.5 §7.8.1).
//
// Private data-element meanings are vendor-defined; the DICOM standard reserves
// the odd groups for them but does NOT specify their VR, keyword, or name. This
// dictionary therefore maps a (private-creator, element-offset) pair to a typed
// entry, analogous to pydicom's pydicom.datadict.private_dictionaries.
//
// Scope and provenance. The lookup MECHANISM is complete: any number of creators
// and offsets can be registered, and PrivateBlock.Lookup resolves a private tag's
// VR/keyword/description through it. The SEED below is deliberately small. Only
// entries that can be attributed to a published, non-proprietary source are
// included; vendor tag meanings are never invented. The seed currently covers:
//
//   - "ACME 3.1": the illustrative private creator from the pydicom documentation
//     and test suite (https://pydicom.github.io, examples of private_block). Used
//     for examples and tests, not a real device.
//
// Larger, vendor-specific dictionaries (Siemens CSA, GE, Philips, etc.) are
// published in GDCM's gdcmPrivateDict.xml. They are intentionally NOT vendored or
// transcribed here because no such source is available in this tree to attribute
// against; transcribing them from memory would risk fabricating tag meanings.
// Adding them is a generator task (port gdcmPrivateDict.xml the way gen/gentags
// ports the Innolitics standard dictionary), tracked as a TODO below.
//
// TODO: vendor a private-dictionary generator from GDCM's gdcmPrivateDict.xml
// (Apache-2.0) to seed the full Siemens/GE/Philips/etc. catalogue, mirroring
// pydicom's _private_dict.py. Until then the seed stays minimal and attributed.

// PrivateTagInfo is a private-dictionary entry: the typed meaning of a private data
// element resolved through its creator. It parallels TagInfo for standard tags.
type PrivateTagInfo struct {
	VR          VR     // dictionary VR for the private element
	VM          string // value multiplicity, e.g. "1", "1-n"
	Keyword     string // creator-defined keyword, may be empty
	Description string // human-readable description
}

// privateDictEntry keys a seed entry by creator and the in-block element offset.
type privateDictEntry struct {
	creator string
	offset  uint8
	info    PrivateTagInfo
}

// privateDict is the seed table. It is keyed at init into privateDictByCreator.
//
// Each entry's offset is the low byte (0x00..0xFF) of the private data element
// within the creator's reserved block; the block (high byte) is assigned at runtime
// and is not part of the dictionary key, matching pydicom, where private_dictionaries
// is keyed by creator then by the "ggggxxee" pattern with xx == block wildcard.
var privateDict = []privateDictEntry{
	// pydicom documentation/test creator "ACME 3.1". Illustrative only.
	{creator: "ACME 3.1", offset: 0x01, info: PrivateTagInfo{VR: VRSH, VM: "1", Keyword: "ACMEPrivateData01", Description: "ACME private data 01"}},
	{creator: "ACME 3.1", offset: 0x02, info: PrivateTagInfo{VR: VRLO, VM: "1", Keyword: "ACMEPrivateData02", Description: "ACME private data 02"}},
	{creator: "ACME 3.1", offset: 0x03, info: PrivateTagInfo{VR: VRDS, VM: "1-n", Keyword: "ACMEPrivateData03", Description: "ACME private data 03"}},
}

// privateDictByCreator indexes the seed by creator then by offset, built once at
// init. Lookups are O(1) and never mutate, so it is safe for concurrent reads.
var privateDictByCreator = func() map[string]map[uint8]PrivateTagInfo {
	m := make(map[string]map[uint8]PrivateTagInfo)
	for _, e := range privateDict {
		byOffset, ok := m[e.creator]
		if !ok {
			byOffset = make(map[uint8]PrivateTagInfo)
			m[e.creator] = byOffset
		}
		byOffset[e.offset] = e.info
	}
	return m
}()

// LookupPrivate resolves the private-dictionary entry for a private data element.
// creator is the private-creator identifier, group is the element's (odd) group,
// and offset is the low byte (0x00..0xFF) of the data element within the creator's
// block. ok is false when the creator/offset pair is not seeded.
//
// group is accepted for parity with pydicom's keying (and to reject even groups)
// even though the current seed is group-independent; a future vendored dictionary
// may key by group.
func LookupPrivate(creator string, group uint16, offset uint8) (PrivateTagInfo, bool) {
	if group%2 == 0 {
		return PrivateTagInfo{}, false
	}
	byOffset, ok := privateDictByCreator[creator]
	if !ok {
		return PrivateTagInfo{}, false
	}
	info, ok := byOffset[offset]
	return info, ok
}

// PrivateCreatorsKnown reports the private creators the dictionary has seeded. It is
// a diagnostic aid for callers that want to know the dictionary's breadth.
func PrivateCreatorsKnown() []string {
	creators := make([]string, 0, len(privateDictByCreator))
	for c := range privateDictByCreator {
		creators = append(creators, c)
	}
	return creators
}
