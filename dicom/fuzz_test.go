package dicom

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// seedPart10 returns a minimal, structurally-valid Part 10 stream in ts for the fuzz
// corpus. It carries a tiny native dataset so the fuzzer starts from a parseable object
// and mutates outward into the malformed space. ts must be one of the four uncompressed
// syntaxes that Write accepts. Identifiers are synthetic sentinels, never real PHI.
func seedPart10(ts TransferSyntax) []byte {
	ds := NewDataSet()
	ds.Set(Element{Tag: NewTag(0x0008, 0x0016), VR: VRUI, Value: NewStrings(VRUI, "1.2.840.10008.5.1.4.1.1.7")})
	ds.Set(Element{Tag: NewTag(0x0008, 0x0018), VR: VRUI, Value: NewStrings(VRUI, "1.2.3.4.5.6.7.8.9")})
	ds.Set(Element{Tag: NewTag(0x0010, 0x0010), VR: VRPN, Value: NewStrings(VRPN, "ZZZTEST^PHI^SENTINEL")})
	ds.Set(Element{Tag: NewTag(0x0028, 0x0010), VR: VRUS, Value: NewInts(VRUS, 2)})
	ds.Set(Element{Tag: NewTag(0x0028, 0x0011), VR: VRUS, Value: NewInts(VRUS, 2)})
	ds.Set(Element{Tag: NewTag(0x0028, 0x0100), VR: VRUS, Value: NewInts(VRUS, 8)})
	ds.Set(Element{Tag: NewTag(0x0028, 0x0002), VR: VRUS, Value: NewInts(VRUS, 1)})
	ds.Set(Element{Tag: NewTag(0x7FE0, 0x0010), VR: VROW, Value: NewBytes(VROW, make([]byte, 4))})
	f := &File{
		Meta: &FileMeta{
			MediaStorageSOPClassUID:    "1.2.840.10008.5.1.4.1.1.7",
			MediaStorageSOPInstanceUID: "1.2.3.4.5.6.7.8.9",
			TransferSyntaxUID:          ts,
		},
		DataSet: ds,
	}
	var buf bytes.Buffer
	_ = Write(&buf, f)
	return buf.Bytes()
}

// seedEncapsulatedPart10 returns a minimal Part 10 stream whose pixel data is an
// encapsulated fragment stream (empty Basic Offset Table, one fragment, Sequence
// Delimitation Item) under ts, an encapsulated transfer syntax. Write only emits the
// four uncompressed syntaxes, so the file-meta group is built through the library's own
// writeFileMeta (so the seed's group-0002 matches a Write-produced object), and the
// geometry elements pixel resolution needs plus the undefined-length (7FE0,0010) header
// are appended so the seed reaches ReadPixelDataFrom's encapsulated fragment-parsing
// path.
func seedEncapsulatedPart10(ts TransferSyntax) []byte {
	meta := &FileMeta{
		MediaStorageSOPClassUID:    "1.2.840.10008.5.1.4.1.1.7",
		MediaStorageSOPInstanceUID: "1.2.3.4.5.6.7.8.9",
		TransferSyntaxUID:          ts,
	}
	var out bytes.Buffer
	_ = writeFileMeta(&out, [128]byte{}, meta)

	// The main dataset must carry the geometry that ResolvePixelGeometry requires
	// (Rows, Columns, BitsAllocated) before the encapsulated fragment stream is parsed.
	geom := NewDataSet()
	geom.Set(Element{Tag: NewTag(0x0028, 0x0002), VR: VRUS, Value: NewInts(VRUS, 1)})
	geom.Set(Element{Tag: NewTag(0x0028, 0x0010), VR: VRUS, Value: NewInts(VRUS, 2)})
	geom.Set(Element{Tag: NewTag(0x0028, 0x0011), VR: VRUS, Value: NewInts(VRUS, 2)})
	geom.Set(Element{Tag: NewTag(0x0028, 0x0100), VR: VRUS, Value: NewInts(VRUS, 8)})
	_ = writeDataSet(&out, geom, ExplicitVRLittleEndian)

	out.Write(encapsulatedPixelHeader())
	out.Write(seedItemHeader(0)) // empty Basic Offset Table
	frag := make([]byte, 4)
	out.Write(seedItemHeader(uint32(len(frag))))
	out.Write(frag)
	out.Write(seedSequenceDelimiter())
	return out.Bytes()
}

// encapsulatedPixelHeader writes the (7FE0,0010) OB undefined-length element header
// that introduces an encapsulated fragment stream (Explicit VR LE: tag, "OB", 2-byte
// reserved, then the 4-byte undefined-length sentinel).
func encapsulatedPixelHeader() []byte {
	b := make([]byte, 12)
	binary.LittleEndian.PutUint16(b[0:2], TagPixelData.Group())
	binary.LittleEndian.PutUint16(b[2:4], TagPixelData.Element())
	copy(b[4:6], "OB")
	binary.LittleEndian.PutUint32(b[8:12], undefinedLength)
	return b
}

// seedItemHeader writes an (FFFE,E000) item header with length in little endian.
func seedItemHeader(length uint32) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint16(b[0:2], tagItem.Group())
	binary.LittleEndian.PutUint16(b[2:4], tagItem.Element())
	binary.LittleEndian.PutUint32(b[4:8], length)
	return b
}

// seedSequenceDelimiter writes the (FFFE,E0DD) Sequence Delimitation Item.
func seedSequenceDelimiter() []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint16(b[0:2], tagSequenceDelimit.Group())
	binary.LittleEndian.PutUint16(b[2:4], tagSequenceDelimit.Element())
	binary.LittleEndian.PutUint32(b[4:8], 0)
	return b
}

// FuzzRead drives the Part 10 / dataset reader with arbitrary bytes. A malformed or
// truncated object must surface a typed error, never panic or over-allocate: every
// value length is bounds-checked against the bytes remaining before any make([]byte, n)
// (PRD §9.3).
func FuzzRead(f *testing.F) {
	f.Add(seedPart10(ExplicitVRLittleEndian))
	f.Add(seedPart10(ImplicitVRLittleEndian))
	f.Add(seedPart10(ExplicitVRBigEndian))
	f.Add(seedPart10(DeflatedExplicitVRLittleEndian))
	f.Add([]byte{})                             // empty stream: short preamble
	f.Add(append(make([]byte, 128), "DICM"...)) // preamble + magic, no file-meta group
	f.Add([]byte("DICM"))                       // magic without the 128-byte preamble
	f.Fuzz(func(t *testing.T, data []byte) {
		// Must never panic; an error is the acceptable outcome for malformed input.
		_, _ = Read(bytes.NewReader(data))
	})
}

// FuzzReadPixelDataFrom drives the pixel reader, whose encapsulated path parses a
// bounded fragment-item stream after the (7FE0,0010) header. A hostile item length must
// fail with a typed error rather than drive an allocation past the source (Codex DCM-006).
func FuzzReadPixelDataFrom(f *testing.F) {
	f.Add(seedPart10(ExplicitVRLittleEndian))
	f.Add(seedPart10(ImplicitVRLittleEndian))
	f.Add(seedEncapsulatedPart10(RLELossless))
	f.Add(seedEncapsulatedPart10(JPEGBaseline8Bit))
	f.Add([]byte{})
	f.Add(append(make([]byte, 128), "DICM"...))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ReadPixelDataFrom(bytes.NewReader(data))
	})
}
