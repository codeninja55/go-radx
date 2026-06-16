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

// privateDictKey identifies a private-dictionary entry by its three addressing
// dimensions: the private creator, the (odd) group it lives in, and the low byte
// (0x00..0xFF) of the data element within the creator's reserved block.
//
// This mirrors pydicom, where private_dictionaries is keyed by creator then by the
// "ggggxxee" tag pattern: the group (gggg) is fixed, the block byte (xx) is a
// wildcard, and the element low byte (ee) is fixed. group and offset together are
// the part of that pattern that identifies the entry; block is assigned at runtime
// and is deliberately not part of the key.
type privateDictKey struct {
	creator string
	group   uint16
	offset  uint8
}

// privateDictEntry is a seed entry: its addressing key and the typed meaning.
type privateDictEntry struct {
	privateDictKey
	info PrivateTagInfo
}

// privateDict is the seed table. It is keyed at init into privateDictByKey.
//
// Each entry's offset is the low byte (0x00..0xFF) of the private data element
// within the creator's reserved block; the block (high byte) is assigned at runtime
// and is not part of the dictionary key. The group is part of the key because, per
// PS3.5 §7.8.1, private element addressing is group-scoped: the same creator and
// offset in two different odd groups are distinct entries.
var privateDict = []privateDictEntry{
	// pydicom documentation/test creator "ACME 3.1". Illustrative only. The seed
	// group (0x0009) matches the group the private-block tests reserve the creator in.
	{privateDictKey{creator: "ACME 3.1", group: 0x0009, offset: 0x01}, PrivateTagInfo{VR: VRSH, VM: "1", Keyword: "ACMEPrivateData01", Description: "ACME private data 01"}},
	{privateDictKey{creator: "ACME 3.1", group: 0x0009, offset: 0x02}, PrivateTagInfo{VR: VRLO, VM: "1", Keyword: "ACMEPrivateData02", Description: "ACME private data 02"}},
	{privateDictKey{creator: "ACME 3.1", group: 0x0009, offset: 0x03}, PrivateTagInfo{VR: VRDS, VM: "1-n", Keyword: "ACMEPrivateData03", Description: "ACME private data 03"}},
}

// privateDictByKey indexes the seed by (creator, group, offset), built once at init.
// Lookups are O(1) and never mutate, so it is safe for concurrent reads.
var privateDictByKey = func() map[privateDictKey]PrivateTagInfo {
	m := make(map[privateDictKey]PrivateTagInfo, len(privateDict))
	for _, e := range privateDict {
		m[e.privateDictKey] = e.info
	}
	return m
}()

// LookupPrivate resolves the private-dictionary entry for a private data element.
// creator is the private-creator identifier, group is the element's (odd) group,
// and offset is the low byte (0x00..0xFF) of the data element within the creator's
// block. ok is false when the (creator, group, offset) triple is not seeded.
//
// Per PS3.5 §7.8.1 private element addressing is group-scoped, so group is part of
// the dictionary key: the same creator and offset in a different odd group is a
// distinct entry and resolves independently.
func LookupPrivate(creator string, group uint16, offset uint8) (PrivateTagInfo, bool) {
	if group%2 == 0 {
		return PrivateTagInfo{}, false
	}
	info, ok := privateDictByKey[privateDictKey{creator: creator, group: group, offset: offset}]
	return info, ok
}

// PrivateCreatorsKnown reports the private creators the dictionary has seeded. It is
// a diagnostic aid for callers that want to know the dictionary's breadth.
func PrivateCreatorsKnown() []string {
	seen := make(map[string]struct{})
	creators := make([]string, 0, len(privateDictByKey))
	for k := range privateDictByKey {
		if _, dup := seen[k.creator]; dup {
			continue
		}
		seen[k.creator] = struct{}{}
		creators = append(creators, k.creator)
	}
	return creators
}
