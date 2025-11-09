// Package tag defines DICOM element tags and tag-related operations.
//
// A Tag represents a DICOM data element identifier as defined in the DICOM standard.
// See https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.1
// and https://dicom.nema.org/medical/dicom/current/output/html/part06.html#chapter_6
package tag

import (
	"fmt"
	"strings"

	"github.com/codeninja55/go-radx/dicom/vr"
)

const (
	// MetadataGroup is the group number for DICOM file meta information elements.
	// See https://dicom.nema.org/medical/dicom/current/output/html/part10.html#sect_7.1
	MetadataGroup = 0x0002
)

// Tag represents a DICOM element tag as a (group, element) pair.
// Tags are used to uniquely identify elements within a DICOM dataset.
//
// According to the DICOM standard Part 5, Section 7.1:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.1
//   - Group numbers with an odd value are used for private elements
//     (see https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.8.1)
//   - Group 0x0002 is reserved for file meta information
//   - Tags are ordered first by group, then by element
//
// The complete data dictionary of standard tags is defined in Part 6:
// https://dicom.nema.org/medical/dicom/current/output/html/part06.html#chapter_6
type Tag struct {
	Group   uint16
	Element uint16
}

// New creates a new Tag with the specified group and element numbers.
func New(group, element uint16) Tag {
	return Tag{
		Group:   group,
		Element: element,
	}
}

// Equals returns true if this tag equals the provided tag.
func (t Tag) Equals(other Tag) bool {
	return t.Group == other.Group && t.Element == other.Element
}

// Compare returns -1, 0, or 1 if t < other, t == other, or t > other, respectively.
// Tags are ordered first by group, then by element as specified in the DICOM standard.
func (t Tag) Compare(other Tag) int {
	if t.Equals(other) {
		return 0
	}
	if t.Uint32() < other.Uint32() {
		return -1
	}
	return 1
}

// String returns a string representation of the tag in the format "(GGGG,EEEE)",
// where GGGG is the group number and EEEE is the element number, both in uppercase hexadecimal.
// This format follows the standard DICOM tag notation.
func (t Tag) String() string {
	return fmt.Sprintf("(%04X,%04X)", t.Group, t.Element)
}

// Uint32 returns the tag as an uint32 value.
// The group number occupies the upper 16 bits, and the element number occupies the lower 16 bits.
// This representation is useful for tag comparison and sorting.
func (t Tag) Uint32() uint32 {
	return (uint32(t.Group) << 16) | uint32(t.Element)
}

// IsPrivate returns true if this tag represents a private element.
// Private elements have an odd group number, according to DICOM Part 5, Section 7.8.1:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.8.1
func (t Tag) IsPrivate() bool {
	return t.Group%2 == 1
}

// IsMetaElement returns true if this tag is part of the file meta-information group (0x0002).
// File meta information is defined in DICOM Part 10, Section 7:
// https://dicom.nema.org/medical/dicom/current/output/html/part10.html#sect_7.1
func (t Tag) IsMetaElement() bool {
	return t.Group == MetadataGroup
}

// Parse parses a tag string in the format "(GGGG,EEEE)" or "GGGG,EEEE"
// and returns the corresponding Tag.
// This supports both the standard DICOM notation with parentheses and without.
func Parse(s string) (Tag, error) {
	// Remove parentheses if present
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "(")
	s = strings.TrimSuffix(s, ")")

	// Split by comma
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return Tag{}, fmt.Errorf("invalid tag format: %q, expected (GGGG,EEEE)", s)
	}

	// Parse group
	var group, element uint16
	_, err := fmt.Sscanf(strings.TrimSpace(parts[0]), "%x", &group)
	if err != nil {
		return Tag{}, fmt.Errorf("invalid group number: %w", err)
	}

	// Parse element
	_, err = fmt.Sscanf(strings.TrimSpace(parts[1]), "%x", &element)
	if err != nil {
		return Tag{}, fmt.Errorf("invalid element number: %w", err)
	}

	return New(group, element), nil
}

// Info stores detailed information about a Tag defined in the DICOM
// standard.
type Info struct {
	Tag Tag
	// List of all possible data encodings for this tag, e.g., "UL", "CS", etc.
	// At least one entry is present.
	VRs []vr.VR
	// Human-readable name of the tag appropriately formatted for printing, e.g., "Pixel Data"
	Name string
	// Human-readable identifier of the tag, e.g., "PixelData"
	Keyword string
	// Cardinality (# of values expected in the element)
	VM string
	// Whether the tag is retired.
	Retired bool
}

// Find returns information about the given tag from the DICOM standard dictionary.
// Returns an error if the tag is not found in the standard.
//
// Special case: For even-numbered groups with element 0x0000, returns a GenericGroupLength entry.
// This follows the DICOM standard where (gggg,0000) represents the group length for group gggg.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part06.html#chapter_6
func Find(t Tag) (Info, error) {
	info, ok := TagDict[t]
	if !ok {
		// Special case: GenericGroupLength for even groups with element 0x0000
		// (0000-u-ffff,0000) UL GenericGroupLength 1
		if t.Group%2 == 0 && t.Element == 0x0000 {
			return Info{
				Tag:     t,
				VRs:     []vr.VR{vr.UnsignedLong},
				Name:    "Generic Group Length",
				Keyword: "GenericGroupLength",
				VM:      "1",
				Retired: false,
			}, nil
		}
		return Info{}, fmt.Errorf("tag %s not found in dictionary", t.String())
	}
	return info, nil
}

// FindByKeyword searches for a tag by its keyword or name field.
// Returns an error if no tag with the given keyword or name is found.
//
// Note: This performs a linear search through all tags, so it's less efficient than Find.
// The search first checks keywords, then falls back to checking names.
//
// Example: FindByKeyword("SOPClassUID") or FindByKeyword("SOP Class UID")
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part06.html#chapter_6
func FindByKeyword(keyword string) (Info, error) {
	if keyword == "" {
		return Info{}, fmt.Errorf("keyword cannot be empty")
	}
	for _, info := range TagDict {
		if info.Keyword == keyword || info.Name == keyword {
			return info, nil
		}
	}
	return Info{}, fmt.Errorf("tag with keyword %q not found in dictionary", keyword)
}

// FindByName searches for a tag by its human-readable name.
// This is a convenience wrapper around FindByKeyword.
//
// Example: FindByName("Specific Character Set")
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part06.html#chapter_6
func FindByName(name string) (Info, error) {
	return FindByKeyword(name)
}

// MustFind is like Find, but panics if the tag is not found.
// This should only be used for well-known tags that are guaranteed to exist.
func MustFind(t Tag) Info {
	info, err := Find(t)
	if err != nil {
		panic(fmt.Sprintf("tag %s not found: %v", t.String(), err))
	}
	return info
}
