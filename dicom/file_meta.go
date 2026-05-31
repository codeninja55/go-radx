package dicom

import (
	"bytes"
	"fmt"
)

// fileMetaTransferSyntax is the fixed encoding of the File Meta Information group:
// it is always Explicit VR Little Endian regardless of the main dataset's syntax
// (PS3.10 §7.1).
const fileMetaTransferSyntax = ExplicitVRLittleEndian

// dicmMagic is the 4-byte prefix following the 128-byte preamble.
const dicmMagic = "DICM"

// File Meta required-element tags.
var (
	tagFileMetaGroupLength   = NewTag(0x0002, 0x0000)
	tagMediaStorageSOPClass  = NewTag(0x0002, 0x0002)
	tagMediaStorageSOPInst   = NewTag(0x0002, 0x0003)
	tagTransferSyntax        = NewTag(0x0002, 0x0010)
	tagImplementationClassID = NewTag(0x0002, 0x0012)
)

// readPreamble reads the 128-byte preamble and validates the DICM magic. A stream
// missing the magic is not a Part 10 file.
func readPreamble(br *boundedReader) ([]byte, error) {
	preamble, err := br.readExact(128)
	if err != nil {
		return nil, midElementEOF(err)
	}
	magic, err := br.readExact(4)
	if err != nil {
		return nil, midElementEOF(err)
	}
	if string(magic) != dicmMagic {
		return nil, fmt.Errorf("dicom: not a Part 10 file: missing %q magic after preamble", dicmMagic)
	}
	return preamble, nil
}

// readFileMeta reads the File Meta Information group. (0002,0000) Group Length is
// required (Type 1) and read first; it bounds the group exactly, so the reader
// consumes precisely the declared bytes and stops at the boundary before the main
// dataset (PS3.10 Table 7.1-1).
func readFileMeta(br *boundedReader) (*FileMeta, error) {
	h, err := readElementHeader(br, fileMetaTransferSyntax)
	if err != nil {
		return nil, midElementEOF(err)
	}
	if h.tag != tagFileMetaGroupLength {
		return nil, fmt.Errorf("dicom: file meta must start with %s File Meta Information Group Length, got %s",
			tagFileMetaGroupLength, h.tag)
	}
	glVal, err := decodeValue(br, h, encodingFor(fileMetaTransferSyntax), nil)
	if err != nil {
		return nil, err
	}
	ints, ok := glVal.(*Ints)
	if !ok || len(ints.Ints()) != 1 {
		return nil, fmt.Errorf("dicom: %s File Meta Information Group Length is not a single UL value", tagFileMetaGroupLength)
	}
	groupLen := ints.Ints()[0]
	if groupLen < 0 {
		return nil, fmt.Errorf("dicom: negative file meta group length")
	}

	// Read exactly groupLen bytes; that hard boundary is what keeps file-meta
	// parsing from spilling into the main dataset, and a short read here is a
	// truncated file (Codex DCM-003).
	groupBytes, err := br.readN(uint32(groupLen))
	if err != nil {
		return nil, err
	}

	inner := newBoundedReader(bytes.NewReader(groupBytes), defaultMaxElementLen)
	elems, err := readDataSet(inner, fileMetaTransferSyntax, newReadConfig())
	if err != nil {
		return nil, err
	}
	if rem, _ := inner.remaining(); rem != 0 {
		return nil, fmt.Errorf("dicom: file meta group length does not align with its elements: %d trailing bytes", rem)
	}

	return fileMetaFromDataSet(elems)
}

// fileMetaFromDataSet projects the typed required fields out of the parsed
// group-0002 dataset, keeping the full dataset for optional elements.
func fileMetaFromDataSet(elems *DataSet) (*FileMeta, error) {
	meta := &FileMeta{Elements: elems}
	if v, ok := elems.GetString(tagMediaStorageSOPClass); ok {
		meta.MediaStorageSOPClassUID = SOPClassUID(v)
	}
	if v, ok := elems.GetString(tagMediaStorageSOPInst); ok {
		meta.MediaStorageSOPInstanceUID = SOPInstanceUID(v)
	}
	if v, ok := elems.GetString(tagTransferSyntax); ok {
		meta.TransferSyntaxUID = TransferSyntax(v)
	}
	if v, ok := elems.GetString(tagImplementationClassID); ok {
		meta.ImplementationClassUID = UID(v)
	}
	if meta.TransferSyntaxUID == "" {
		return nil, fmt.Errorf("dicom: file meta is missing %s Transfer Syntax UID", tagTransferSyntax)
	}
	return meta, nil
}
