package dicom

import "strings"

// Private data elements (PS3.5 §7.8.1).
//
// A private data element lives in an odd-numbered group. Within a group, a
// "private creator" reserves a block of 256 element slots by writing its
// identifier string into a private-creator element (gggg,0010..00FF). The low
// byte of that creator element (0x10..0xFF) is the block number. Every private
// data element the creator owns then has the tag (gggg, bb00..bbFF), where bb is
// the block number and the low byte is the offset (0x00..0xFF) within the block.
//
// This mirrors pydicom's Dataset.private_block / private_creators /
// get_private_item and the PrivateBlock type, and resolves block ownership
// against the elements actually present in a parsed dataset.

const (
	// privateCreatorMin and privateCreatorMax bound the private-creator element
	// numbers (gggg,0010..00FF) per PS3.5 §7.8.1.
	privateCreatorMin uint16 = 0x0010
	privateCreatorMax uint16 = 0x00FF

	// blockShift positions the block number in the high byte of the element
	// half of a private data-element tag (gggg, bb00..bbFF).
	blockShift = 8
)

// PrivateBlock is a typed handle to one private creator's reserved block of data
// elements within a group. It is resolved from a dataset by PrivateBlock and is a
// read-through view: the data elements it exposes live in the underlying DataSet.
//
// The zero value is not usable; obtain a PrivateBlock via DataSet.PrivateBlock.
type PrivateBlock struct {
	ds      *DataSet
	group   uint16
	creator string
	block   uint16 // block number, 0x10..0xFF (low byte of the creator element)
}

// Group returns the (odd) group this block belongs to.
func (b *PrivateBlock) Group() uint16 { return b.group }

// Creator returns the private-creator identifier string for this block.
func (b *PrivateBlock) Creator() string { return b.creator }

// BlockStart returns the element number of the first slot the block owns,
// (block << 8). A data element at offset n has element number BlockStart()+n.
func (b *PrivateBlock) BlockStart() uint16 { return b.block << blockShift }

// Tag returns the full tag of the data element at offset within the block. offset
// is the low byte (0x00..0xFF) of the private data element; it must be <= 0xFF.
func (b *PrivateBlock) Tag(offset uint8) Tag {
	return NewTag(b.group, b.BlockStart()|uint16(offset))
}

// Get returns the element at offset within the block. ok is false if absent.
func (b *PrivateBlock) Get(offset uint8) (Element, bool) {
	return b.ds.Get(b.Tag(offset))
}

// GetString returns the first text value of the element at offset.
func (b *PrivateBlock) GetString(offset uint8) (string, bool) {
	return b.ds.GetString(b.Tag(offset))
}

// Set inserts or replaces a data element at offset within the block under vr.
// The creator element is left untouched; PrivateBlock was resolved with create or
// from a parsed dataset, so the reservation already exists.
func (b *PrivateBlock) Set(offset uint8, vr VR, value Value) {
	b.ds.Set(Element{Tag: b.Tag(offset), VR: vr, Value: value})
}

// SetString inserts or replaces a text element at offset under vr.
func (b *PrivateBlock) SetString(offset uint8, vr VR, vals ...string) {
	b.Set(offset, vr, NewStrings(vr, vals...))
}

// Delete removes the data element at offset within the block; it is not an error
// if absent. The creator element itself is not removed.
func (b *PrivateBlock) Delete(offset uint8) {
	b.ds.Delete(b.Tag(offset))
}

// Lookup resolves the private-dictionary entry for the data element at offset,
// using this block's creator. ok is false when the creator or offset is not seeded
// in the private dictionary.
func (b *PrivateBlock) Lookup(offset uint8) (PrivateTagInfo, bool) {
	return LookupPrivate(b.creator, b.group, offset)
}

// PrivateBlock resolves the private creator in group whose value equals creator and
// returns a handle to its reserved block of data elements.
//
// With create == false the creator must already be present as a private-creator
// element (gggg,0010..00FF); an absent creator returns ok == false. With
// create == true a missing creator is added: the lowest free block (0x10..0xFF) is
// reserved and the creator element written, matching pydicom's
// Dataset.private_block(..., create=True).
//
// group must be odd (private); an even group returns ok == false.
func (ds *DataSet) PrivateBlock(group uint16, creator string, create bool) (*PrivateBlock, bool) {
	if group%2 == 0 || creator == "" {
		return nil, false
	}
	if block, ok := ds.findPrivateCreatorBlock(group, creator); ok {
		return &PrivateBlock{ds: ds, group: group, creator: creator, block: block}, true
	}
	if !create {
		return nil, false
	}
	block, ok := ds.reservePrivateBlock(group)
	if !ok {
		return nil, false
	}
	ds.Set(Element{
		Tag:   NewTag(group, block),
		VR:    VRLO,
		Value: NewStrings(VRLO, creator),
	})
	return &PrivateBlock{ds: ds, group: group, creator: creator, block: block}, true
}

// findPrivateCreatorBlock returns the block number whose private-creator element in
// group carries creator. The creator string is matched after trimming the trailing
// space/NUL pad a stored LO value may carry.
func (ds *DataSet) findPrivateCreatorBlock(group uint16, creator string) (uint16, bool) {
	want := strings.TrimRight(creator, " \x00")
	for _, t := range ds.order {
		if t.Group() != group {
			continue
		}
		el := t.Element()
		if el < privateCreatorMin || el > privateCreatorMax {
			continue
		}
		got, ok := ds.privateCreatorValue(t)
		if !ok {
			continue
		}
		if got == want {
			return el, true
		}
	}
	return 0, false
}

// privateCreatorValue reads a private-creator element's identifier, accepting both
// the decoded text form (a stored LO/SH/UN-as-string value) and the raw-byte form a
// creator read under Implicit VR LE or as UN carries. Trailing pad is trimmed.
func (ds *DataSet) privateCreatorValue(t Tag) (string, bool) {
	if s, ok := ds.GetString(t); ok {
		return strings.TrimRight(s, " \x00"), true
	}
	e, ok := ds.Get(t)
	if !ok {
		return "", false
	}
	v, ok := materialise(e.Value)
	if !ok {
		return "", false
	}
	if bv, ok := v.(*Bytes); ok {
		return strings.TrimRight(string(bv.Bytes()), " \x00"), true
	}
	return "", false
}

// reservePrivateBlock returns the lowest unused block number (0x10..0xFF) in group,
// i.e. the lowest private-creator element slot not already populated.
func (ds *DataSet) reservePrivateBlock(group uint16) (uint16, bool) {
	for block := privateCreatorMin; block <= privateCreatorMax; block++ {
		if _, taken := ds.Get(NewTag(group, block)); !taken {
			return block, true
		}
	}
	return 0, false
}

// PrivateCreators lists the private-creator identifiers present in group, in
// ascending block order. Pad characters are trimmed. Mirrors pydicom's
// Dataset.private_creators(group).
func (ds *DataSet) PrivateCreators(group uint16) []string {
	if group%2 == 0 {
		return nil
	}
	var creators []string
	for _, t := range ds.order {
		if t.Group() != group {
			continue
		}
		el := t.Element()
		if el < privateCreatorMin || el > privateCreatorMax {
			continue
		}
		if s, ok := ds.privateCreatorValue(t); ok {
			creators = append(creators, s)
		}
	}
	return creators
}

// GetPrivateItem returns the data element at elementOffset within the block reserved
// by creator in group. elementOffset is the low byte (0x00..0xFF) of the private data
// element. ok is false if the creator is absent or the element is not present.
// Mirrors pydicom's Dataset.get_private_item(group, element_offset, private_creator).
func (ds *DataSet) GetPrivateItem(group uint16, elementOffset uint8, creator string) (Element, bool) {
	block, ok := ds.PrivateBlock(group, creator, false)
	if !ok {
		return Element{}, false
	}
	return block.Get(elementOffset)
}
