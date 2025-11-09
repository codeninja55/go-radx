// Package dicom provides DICOM file parsing and manipulation.
package dicom

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
)

// ElementParser reads individual DICOM data elements from a binary stream.
//
// It handles both Explicit VR and Implicit VR encoding based on the Transfer Syntax.
// Element structure varies by VR:
//   - Explicit VR (most VRs): Tag(4) + VR(2) + Length(2) + Value(n)
//   - Explicit VR (OB/OW/SQ/etc): Tag(4) + VR(2) + Reserved(2) + Length(4) + Value(n)
//   - Implicit VR: Tag(4) + Length(4) + Value(n), VR looked up in dictionary
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.1.2
type ElementParser struct {
	reader *Reader
	ts     *TransferSyntax
}

// NewElementParser creates a new element parser with the specified reader and transfer syntax.
func NewElementParser(reader *Reader, ts *TransferSyntax) *ElementParser {
	return &ElementParser{
		reader: reader,
		ts:     ts,
	}
}

// ReadElement reads the next data element from the stream.
//
// Returns an error if the element cannot be parsed or if the stream ends unexpectedly.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.1
func (p *ElementParser) ReadElement() (*element.Element, error) {
	// Read tag (4 bytes: group + element)
	t, err := p.readTag()
	if err != nil {
		return nil, fmt.Errorf("failed to read tag: %w", err)
	}

	// Read VR based on transfer syntax
	var v vr.VR
	var length uint32

	if p.ts.ExplicitVR {
		// Explicit VR: VR is in the file
		v, err = p.readVRExplicit()
		if err != nil {
			return nil, fmt.Errorf("failed to read VR for tag %s: %w", t, err)
		}

		// Read length (2 or 4 bytes depending on VR)
		length, err = p.readLength(v)
		if err != nil {
			return nil, fmt.Errorf("failed to read length for tag %s: %w", t, err)
		}
	} else {
		// Implicit VR: VR must be looked up from tag dictionary
		v, err = p.readVRImplicit(t)
		if err != nil {
			return nil, fmt.Errorf("failed to look up VR for tag %s: %w", t, err)
		}

		// For Implicit VR, length is always 4 bytes
		length, err = p.reader.ReadUint32()
		if err != nil {
			return nil, fmt.Errorf("failed to read length for tag %s: %w", t, err)
		}
	}

	// Read value based on VR type
	val, err := p.readValue(t, v, length)
	if err != nil {
		return nil, fmt.Errorf("failed to read value for tag %s: %w", t, err)
	}

	// Create and return element
	elem, err := element.NewElement(t, v, val)
	if err != nil {
		return nil, fmt.Errorf("failed to create element for tag %s: %w", t, err)
	}

	return elem, nil
}

// readTag reads a DICOM tag (group and element).
func (p *ElementParser) readTag() (tag.Tag, error) {
	// Read group (2 bytes)
	group, err := p.reader.ReadUint16()
	if err != nil {
		return tag.Tag{}, fmt.Errorf("failed to read tag group: %w", err)
	}

	// Read element (2 bytes)
	elem, err := p.reader.ReadUint16()
	if err != nil {
		return tag.Tag{}, fmt.Errorf("failed to read tag element: %w", err)
	}

	return tag.New(group, elem), nil
}

// readVRExplicit reads a 2-byte VR in Explicit VR encoding.
func (p *ElementParser) readVRExplicit() (vr.VR, error) {
	// Read 2-byte VR string
	vrStr, err := p.reader.ReadString(2)
	if err != nil {
		return 0, fmt.Errorf("failed to read VR: %w", err)
	}

	// Parse VR string
	v, err := vr.Parse(vrStr)
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrInvalidVR, vrStr)
	}

	return v, nil
}

// readVRImplicit looks up the VR for a tag from the DICOM data dictionary.
// This is used for Implicit VR transfer syntaxes where VR is not encoded in the file.
//
// For tags with multiple possible VRs (e.g., PixelData can be "OB or OW"),
// this returns the first VR in the list as the default.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.1.2
func (p *ElementParser) readVRImplicit(t tag.Tag) (vr.VR, error) {
	// Look up tag in dictionary
	info, err := tag.Find(t)
	if err != nil {
		// Tag not in dictionary - use UN (Unknown) as fallback
		return vr.Unknown, nil
	}

	// Return first VR (for tags with multiple VRs like "OB or OW", use the first one)
	if len(info.VRs) == 0 {
		return vr.Unknown, nil
	}

	return info.VRs[0], nil
}

// readLength reads the value length field.
//
// Length encoding depends on VR:
//   - Most VRs: 2-byte uint16
//   - OB, OD, OF, OL, OV, OW, SQ, UC, UN, UR, UT: 2-byte reserved (0x0000) + 4-byte uint32
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.1.2
func (p *ElementParser) readLength(v vr.VR) (uint32, error) {
	// Check if this VR uses 32-bit length field
	if v.UsesExplicitLength32() {
		// Read 2-byte reserved field (must be 0x0000)
		reserved, err := p.reader.ReadUint16()
		if err != nil {
			return 0, fmt.Errorf("failed to read reserved field: %w", err)
		}
		if reserved != 0x0000 {
			// Not strictly an error per standard, but log for debugging
			// Standard says it "should" be 0x0000 but implementations may vary
		}

		// Read 4-byte length
		length, err := p.reader.ReadUint32()
		if err != nil {
			return 0, fmt.Errorf("failed to read 32-bit length: %w", err)
		}

		return length, nil
	}

	// Read 2-byte length for standard VRs
	length16, err := p.reader.ReadUint16()
	if err != nil {
		return 0, fmt.Errorf("failed to read 16-bit length: %w", err)
	}

	return uint32(length16), nil
}

// readValue reads and parses the value field based on VR type.
func (p *ElementParser) readValue(t tag.Tag, v vr.VR, length uint32) (value.Value, error) {
	// Handle empty values
	if length == 0 {
		return p.createEmptyValue(v)
	}

	// Handle undefined length (0xFFFFFFFF)
	if length == 0xFFFFFFFF {
		// For sequences with undefined length, skip the sequence content
		// Sequences are delimited by Sequence Delimitation Item (FFFE,E0DD)
		if v == vr.SequenceOfItems {
			return p.skipUndefinedLengthSequence(t)
		}

		// Handle encapsulated pixel data (OB/OW with undefined length)
		// This is used for compressed transfer syntaxes (JPEG, JPEG 2000, RLE, etc.)
		// Per DICOM Part 5, Section A.4: encapsulated data uses undefined length
		// with fragments: Item (FFFE,E000) + data, terminated by Sequence Delimitation (FFFE,E0DD)
		//
		// DICOM Standard Reference:
		// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_A.4
		if (v == vr.OtherByte || v == vr.OtherWord) && t.Group == 0x7FE0 && t.Element == 0x0010 {
			return p.skipEncapsulatedPixelData(t, v)
		}

		return nil, fmt.Errorf("%w: undefined length for non-sequence VR %s", ErrUndefinedLength, v.String())
	}

	// Dispatch to VR-specific reader
	// Check sequences first, then float types before numeric types (floats are also numeric)
	switch {
	case v == vr.SequenceOfItems:
		// Sequence with defined length - skip it
		return p.skipDefinedLengthSequence(t, length)
	case v.IsStringType():
		return p.readStringValue(v, length)
	case v == vr.FloatingPointSingle || v == vr.FloatingPointDouble:
		return p.readFloatValue(v, length)
	case v.IsNumericType():
		return p.readIntValue(v, length)
	case v.IsBinaryType():
		return p.readBytesValue(v, length)
	default:
		// Unknown VR, read as bytes
		return p.readBytesValue(vr.Unknown, length)
	}
}

// createEmptyValue creates an empty value for the given VR.
func (p *ElementParser) createEmptyValue(v vr.VR) (value.Value, error) {
	switch {
	case v == vr.SequenceOfItems:
		return value.NewBytesValue(vr.SequenceOfItems, []byte{})
	case v.IsStringType():
		return value.NewStringValue(v, []string{})
	case v.IsNumericType():
		return value.NewIntValue(v, []int64{})
	case v == vr.FloatingPointSingle || v == vr.FloatingPointDouble:
		return value.NewFloatValue(v, []float64{})
	case v.IsBinaryType():
		return value.NewBytesValue(v, []byte{})
	default:
		return value.NewBytesValue(vr.Unknown, []byte{})
	}
}

// readStringValue reads a string-based VR value.
//
// DICOM strings may contain multiple values separated by backslash (\).
// String values are space-padded for even length and may have trailing nulls for UI.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func (p *ElementParser) readStringValue(v vr.VR, length uint32) (*value.StringValue, error) {
	// Read raw bytes
	data, err := p.reader.ReadBytes(int(length))
	if err != nil {
		return nil, fmt.Errorf("failed to read string data: %w", err)
	}

	// Convert to string
	str := string(data)

	// Trim trailing null and space padding
	str = strings.TrimRight(str, "\x00 ")

	// Split by backslash for multi-valued elements
	var values []string
	if str == "" {
		values = []string{}
	} else {
		values = strings.Split(str, "\\")
	}

	// Create string value
	val, err := value.NewStringValue(v, values)
	if err != nil {
		return nil, fmt.Errorf("failed to create string value: %w", err)
	}

	return val, nil
}

// readIntValue reads an integer VR value.
//
// Handles: SS (int16), US (uint16), SL (int32), UL (uint32), SV (int64), UV (uint64), AT (tag)
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func (p *ElementParser) readIntValue(v vr.VR, length uint32) (*value.IntValue, error) {
	var values []int64

	// Determine bytes per value
	var bytesPerValue int
	switch v {
	case vr.SignedShort, vr.UnsignedShort:
		bytesPerValue = 2
	case vr.SignedLong, vr.UnsignedLong, vr.AttributeTag:
		bytesPerValue = 4
	case vr.SignedVeryLong, vr.UnsignedVeryLong:
		bytesPerValue = 8
	default:
		return nil, fmt.Errorf("unsupported integer VR: %s", v.String())
	}

	// Calculate number of values
	numValues := int(length) / bytesPerValue
	if int(length)%bytesPerValue != 0 {
		return nil, fmt.Errorf("invalid length %d for VR %s (not multiple of %d)", length, v.String(), bytesPerValue)
	}

	// Read each value
	for i := 0; i < numValues; i++ {
		var val int64

		switch v {
		case vr.SignedShort:
			u16, err := p.reader.ReadUint16()
			if err != nil {
				return nil, err
			}
			val = int64(int16(u16))

		case vr.UnsignedShort:
			u16, err := p.reader.ReadUint16()
			if err != nil {
				return nil, err
			}
			val = int64(u16)

		case vr.SignedLong:
			u32, err := p.reader.ReadUint32()
			if err != nil {
				return nil, err
			}
			val = int64(int32(u32))

		case vr.UnsignedLong:
			u32, err := p.reader.ReadUint32()
			if err != nil {
				return nil, err
			}
			val = int64(u32)

		case vr.AttributeTag:
			u32, err := p.reader.ReadUint32()
			if err != nil {
				return nil, err
			}
			val = int64(u32)

		case vr.SignedVeryLong:
			data, err := p.reader.ReadBytes(8)
			if err != nil {
				return nil, err
			}
			val = int64(p.ts.ByteOrder.Uint64(data))

		case vr.UnsignedVeryLong:
			data, err := p.reader.ReadBytes(8)
			if err != nil {
				return nil, err
			}
			val = int64(p.ts.ByteOrder.Uint64(data))
		}

		values = append(values, val)
	}

	// Create int value
	intVal, err := value.NewIntValue(v, values)
	if err != nil {
		return nil, fmt.Errorf("failed to create int value: %w", err)
	}

	return intVal, nil
}

// readFloatValue reads a floating-point VR value.
//
// Handles: FL (float32), FD (float64)
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func (p *ElementParser) readFloatValue(v vr.VR, length uint32) (*value.FloatValue, error) {
	var values []float64

	// Determine bytes per value
	var bytesPerValue int
	switch v {
	case vr.FloatingPointSingle:
		bytesPerValue = 4
	case vr.FloatingPointDouble:
		bytesPerValue = 8
	default:
		return nil, fmt.Errorf("unsupported float VR: %s", v.String())
	}

	// Calculate number of values
	numValues := int(length) / bytesPerValue
	if int(length)%bytesPerValue != 0 {
		return nil, fmt.Errorf("invalid length %d for VR %s (not multiple of %d)", length, v.String(), bytesPerValue)
	}

	// Read each value
	for i := 0; i < numValues; i++ {
		if v == vr.FloatingPointSingle {
			// Read float32
			data, err := p.reader.ReadBytes(4)
			if err != nil {
				return nil, err
			}
			bits := p.ts.ByteOrder.Uint32(data)
			f32 := math.Float32frombits(bits)
			values = append(values, float64(f32))
		} else {
			// Read float64
			data, err := p.reader.ReadBytes(8)
			if err != nil {
				return nil, err
			}
			bits := p.ts.ByteOrder.Uint64(data)
			f64 := math.Float64frombits(bits)
			values = append(values, f64)
		}
	}

	// Create float value
	floatVal, err := value.NewFloatValue(v, values)
	if err != nil {
		return nil, fmt.Errorf("failed to create float value: %w", err)
	}

	return floatVal, nil
}

// readBytesValue reads a binary VR value.
//
// Handles: OB, OD, OF, OL, OV, OW, UN
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func (p *ElementParser) readBytesValue(v vr.VR, length uint32) (*value.BytesValue, error) {
	// Read raw bytes
	data, err := p.reader.ReadBytes(int(length))
	if err != nil {
		return nil, fmt.Errorf("failed to read binary data: %w", err)
	}

	// Create bytes value
	bytesVal, err := value.NewBytesValue(v, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create bytes value: %w", err)
	}

	return bytesVal, nil
}

// skipDefinedLengthSequence skips over a sequence with defined length.
//
// For sequences with known length, we simply skip the specified number of bytes.
// The sequence content is not parsed.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.5
func (p *ElementParser) skipDefinedLengthSequence(sequenceTag tag.Tag, length uint32) (value.Value, error) {
	// Skip the sequence content
	if length > 0 {
		_, err := p.reader.ReadBytes(int(length))
		if err != nil {
			return nil, fmt.Errorf("failed to skip sequence %s content (%d bytes): %w", sequenceTag, length, err)
		}
	}

	// Return empty bytes value as placeholder for skipped sequence
	return value.NewBytesValue(vr.SequenceOfItems, []byte{})
}

// skipUndefinedLengthSequence skips over a sequence with undefined length.
//
// Sequences with undefined length are terminated by a Sequence Delimitation Item (FFFE,E0DD).
// Items within the sequence start with Item tag (FFFE,E000) and may have undefined length too,
// in which case they're terminated by Item Delimitation Item (FFFE,E00D).
//
// For now, we skip the entire sequence content and return a placeholder value.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.5
func (p *ElementParser) skipUndefinedLengthSequence(sequenceTag tag.Tag) (value.Value, error) {
	// Delimiter tags
	const (
		itemTag                 = uint32(0xFFFEE000) // Item
		itemDelimitationTag     = uint32(0xFFFEE00D) // Item Delimitation Item
		sequenceDelimitationTag = uint32(0xFFFEE0DD) // Sequence Delimitation Item
	)

	for {
		// Read next tag
		t, err := p.readTag()
		if err != nil {
			return nil, fmt.Errorf("unexpected EOF while skipping sequence %s: %w", sequenceTag, err)
		}

		tagValue := t.Uint32()

		// Check for sequence delimitation first
		if tagValue == sequenceDelimitationTag {
			// Read and discard length (should be 0)
			_, err = p.reader.ReadUint32()
			if err != nil {
				return nil, fmt.Errorf("failed to read sequence delimitation length: %w", err)
			}
			// Return empty bytes value as placeholder for skipped sequence
			return value.NewBytesValue(vr.SequenceOfItems, []byte{})
		}

		// Read length based on tag type
		var elemLength uint32
		if tagValue == itemTag || tagValue == itemDelimitationTag {
			// Delimiter items: read 4-byte length directly (no VR)
			elemLength, err = p.reader.ReadUint32()
			if err != nil {
				return nil, fmt.Errorf("failed to read delimiter item length: %w", err)
			}
		} else {
			// Regular data element within item
			// Read VR and length according to transfer syntax
			var v vr.VR
			if p.ts.ExplicitVR {
				v, err = p.readVRExplicit()
				if err != nil {
					return nil, fmt.Errorf("failed to read VR while skipping sequence: %w", err)
				}

				elemLength, err = p.readLength(v)
				if err != nil {
					return nil, fmt.Errorf("failed to read length while skipping sequence: %w", err)
				}
			} else {
				// Implicit VR: look up VR and read 4-byte length
				v, err = p.readVRImplicit(t)
				if err != nil {
					return nil, fmt.Errorf("failed to look up VR while skipping sequence: %w", err)
				}

				elemLength, err = p.reader.ReadUint32()
				if err != nil {
					return nil, fmt.Errorf("failed to read length while skipping sequence: %w", err)
				}
			}

			// If this is a nested sequence with undefined length, recurse
			if v == vr.SequenceOfItems && elemLength == 0xFFFFFFFF {
				_, err = p.skipUndefinedLengthSequence(t)
				if err != nil {
					return nil, fmt.Errorf("failed to skip nested sequence %s: %w", t, err)
				}
				continue
			}
		}

		// Handle different tag types
		switch tagValue {
		case itemTag:
			// Start of an item
			if elemLength == 0xFFFFFFFF {
				// Item with undefined length - skip until item delimitation
				err = p.skipUndefinedLengthItem()
				if err != nil {
					return nil, fmt.Errorf("failed to skip undefined length item: %w", err)
				}
			} else {
				// Item with defined length - skip the content
				if elemLength > 0 {
					_, err := p.reader.ReadBytes(int(elemLength))
					if err != nil {
						return nil, fmt.Errorf("failed to skip item content: %w", err)
					}
				}
			}

		case itemDelimitationTag:
			// This should not occur at sequence level
			return nil, fmt.Errorf("unexpected item delimitation tag while skipping sequence %s", sequenceTag)

		default:
			// Regular data element - skip it
			if elemLength > 0 && elemLength != 0xFFFFFFFF {
				_, err := p.reader.ReadBytes(int(elemLength))
				if err != nil {
					return nil, fmt.Errorf("failed to skip element value: %w", err)
				}
			}
		}
	}
}

// skipUndefinedLengthItem skips over an item with undefined length.
//
// Items with undefined length are terminated by an Item Delimitation Item (FFFE,E00D).
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.5
func (p *ElementParser) skipUndefinedLengthItem() error {
	// Delimiter tags
	const (
		itemDelimitationTag     = uint32(0xFFFEE00D) // Item Delimitation Item
		sequenceDelimitationTag = uint32(0xFFFEE0DD) // Sequence Delimitation Item (not expected here, but check anyway)
	)

	for {
		// Read next tag
		t, err := p.readTag()
		if err != nil {
			if err == io.EOF {
				// EOF while reading item - this might mean the sequence/item delimiter is missing
				// Return EOF so caller can handle it
				return io.EOF
			}
			return fmt.Errorf("failed to read tag while skipping item: %w", err)
		}

		tagValue := t.Uint32()

		// Check for item delimitation
		if tagValue == itemDelimitationTag {
			// Read and discard length (should be 0)
			_, err = p.reader.ReadUint32()
			if err != nil {
				return fmt.Errorf("failed to read item delimitation length: %w", err)
			}
			return nil
		}

		// Check if we accidentally hit sequence delimitation (shouldn't happen, but be defensive)
		if tagValue == sequenceDelimitationTag {
			// This is actually the end of the parent sequence, not the item
			// We need to "un-read" this tag by putting it back somehow
			// For now, just return an error - the sequence parser will handle this
			return fmt.Errorf("found sequence delimitation while expecting item delimitation")
		}

		// Regular data element - read VR and length according to transfer syntax
		var v vr.VR
		var elemLength uint32
		if p.ts.ExplicitVR {
			v, err = p.readVRExplicit()
			if err != nil {
				return fmt.Errorf("failed to read VR while skipping item: %w", err)
			}

			elemLength, err = p.readLength(v)
			if err != nil {
				return fmt.Errorf("failed to read length while skipping item: %w", err)
			}
		} else {
			// Implicit VR: look up VR and read 4-byte length
			v, err = p.readVRImplicit(t)
			if err != nil {
				return fmt.Errorf("failed to look up VR while skipping item: %w", err)
			}

			elemLength, err = p.reader.ReadUint32()
			if err != nil {
				return fmt.Errorf("failed to read length while skipping item: %w", err)
			}
		}

		// If this is a nested sequence with undefined length, recurse
		if v == vr.SequenceOfItems && elemLength == 0xFFFFFFFF {
			_, err = p.skipUndefinedLengthSequence(t)
			if err != nil {
				if err == io.EOF {
					return io.EOF
				}
				return fmt.Errorf("failed to skip nested sequence %s: %w", t, err)
			}
			continue
		}

		// Skip defined-length values
		if elemLength > 0 && elemLength != 0xFFFFFFFF {
			_, err := p.reader.ReadBytes(int(elemLength))
			if err != nil {
				if err == io.EOF {
					return io.EOF
				}
				return fmt.Errorf("failed to skip element value: %w", err)
			}
		}
	}
}

// skipEncapsulatedPixelData reads encapsulated pixel data with undefined length.
//
// Encapsulated pixel data is used for compressed transfer syntaxes (JPEG, JPEG 2000, RLE, etc.)
// and uses a structure similar to sequences:
//   - Basic Offset Table: Item (FFFE,E000) + length + data (may be empty)
//   - Pixel Data Fragments: One or more Item (FFFE,E000) + length + compressed data
//   - Sequence Delimitation Item (FFFE,E0DD) with length 0
//
// This function reads all fragments and stores them as raw bytes in encapsulated format.
// The pixel module will later parse and decompress this data.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_A.4
func (p *ElementParser) skipEncapsulatedPixelData(pixelDataTag tag.Tag, pixelVR vr.VR) (value.Value, error) {
	// Delimiter tags
	const (
		itemTag                 = uint32(0xFFFEE000) // Item
		sequenceDelimitationTag = uint32(0xFFFEE0DD) // Sequence Delimitation Item
	)

	// Buffer to collect all encapsulated data (including item tags and lengths)
	var encapsulatedData []byte

	for {
		// Read next tag
		t, err := p.readTag()
		if err != nil {
			return nil, fmt.Errorf("unexpected EOF while reading encapsulated pixel data %s: %w", pixelDataTag, err)
		}

		tagValue := t.Uint32()

		// Check for sequence delimitation (end of encapsulated data)
		if tagValue == sequenceDelimitationTag {
			// Read and discard length (should be 0)
			_, err = p.reader.ReadUint32()
			if err != nil {
				return nil, fmt.Errorf("failed to read sequence delimitation length: %w", err)
			}

			// Add sequence delimitation tag to encapsulated data
			encapsulatedData = append(encapsulatedData, 0xFE, 0xFF, 0xDD, 0xE0)
			encapsulatedData = append(encapsulatedData, 0x00, 0x00, 0x00, 0x00) // length = 0

			// Return bytes value with all encapsulated data
			return value.NewBytesValue(pixelVR, encapsulatedData)
		}

		// Should only encounter Item tags in encapsulated pixel data
		if tagValue != itemTag {
			return nil, fmt.Errorf("unexpected tag %s while reading encapsulated pixel data (expected Item or Sequence Delimitation)", t)
		}

		// Read item length (4 bytes)
		itemLength, err := p.reader.ReadUint32()
		if err != nil {
			return nil, fmt.Errorf("failed to read item length: %w", err)
		}

		// Add item tag to encapsulated data
		encapsulatedData = append(encapsulatedData, 0xFE, 0xFF, 0x00, 0xE0)

		// Add item length (little-endian uint32)
		encapsulatedData = append(encapsulatedData,
			byte(itemLength&0xFF),
			byte((itemLength>>8)&0xFF),
			byte((itemLength>>16)&0xFF),
			byte((itemLength>>24)&0xFF))

		// Read and append the item data
		if itemLength > 0 {
			itemData, err := p.reader.ReadBytes(int(itemLength))
			if err != nil {
				return nil, fmt.Errorf("failed to read item data (%d bytes): %w", itemLength, err)
			}
			encapsulatedData = append(encapsulatedData, itemData...)
		}
	}
}
