package dicom

import "fmt"

// Tag is the 32-bit (group, element) identifier of a data element, written
// (gggg,eeee). Odd groups are private. It is a named type, never a bare uint32.
type Tag uint32

// NewTag composes a Tag from its group and element halves.
func NewTag(group, element uint16) Tag {
	return Tag(uint32(group)<<16 | uint32(element))
}

// Group returns the high 16 bits.
func (t Tag) Group() uint16 { return uint16(t >> 16) }

// Element returns the low 16 bits.
func (t Tag) Element() uint16 { return uint16(t) } // #nosec G115 -- intentional low-16-bit extraction of the 32-bit tag

// String renders the canonical (gggg,eeee) form.
func (t Tag) String() string {
	return fmt.Sprintf("(%04X,%04X)", t.Group(), t.Element())
}

// IsPrivate reports whether the group is odd (private data).
func (t Tag) IsPrivate() bool { return t.Group()%2 == 1 }

// IsPrivateCreator reports a private-creator tag: odd group, element in 0x0010..0x00FF.
func (t Tag) IsPrivateCreator() bool {
	return t.IsPrivate() && t.Element() >= 0x0010 && t.Element() <= 0x00FF
}

// IsGroupLength reports element == 0x0000.
func (t Tag) IsGroupLength() bool { return t.Element() == 0x0000 }
