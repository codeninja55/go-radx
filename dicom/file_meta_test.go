package dicom

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// buildFileMetaBytes assembles a valid preamble + DICM + group-0002 with a correct
// (0002,0000) group length, for the file-meta read tests.
func buildFileMetaBytes(t *testing.T, elems *DataSet) []byte {
	t.Helper()
	// Serialise group-0002 elements (Explicit VR LE).
	var group bytes.Buffer
	if err := writeDataSet(&group, elems, ExplicitVRLittleEndian); err != nil {
		t.Fatalf("writeDataSet(group2): %v", err)
	}
	// Group-length element: (0002,0000) UL 4 = len(group).
	gl := NewDataSet()
	gl.Set(Element{Tag: NewTag(0x0002, 0x0000), VR: VRUL, Value: NewInts(VRUL, int64(group.Len()))})
	var glBuf bytes.Buffer
	if err := writeDataSet(&glBuf, gl, ExplicitVRLittleEndian); err != nil {
		t.Fatalf("writeDataSet(grouplen): %v", err)
	}

	var out bytes.Buffer
	out.Write(make([]byte, 128)) // preamble
	out.WriteString("DICM")
	out.Write(glBuf.Bytes())
	out.Write(group.Bytes())
	return out.Bytes()
}

func metaElems() *DataSet {
	ds := NewDataSet()
	ds.Set(Element{Tag: NewTag(0x0002, 0x0001), VR: VROB, Value: NewBytes(VROB, []byte{0x00, 0x01})})
	ds.Set(Element{Tag: NewTag(0x0002, 0x0002), VR: VRUI, Value: NewStrings(VRUI, "1.2.840.10008.5.1.4.1.1.7")})
	ds.Set(Element{Tag: NewTag(0x0002, 0x0003), VR: VRUI, Value: NewStrings(VRUI, "1.2.3.4.5.6.7.8.9")})
	ds.Set(Element{Tag: NewTag(0x0002, 0x0010), VR: VRUI, Value: NewStrings(VRUI, string(ExplicitVRLittleEndian))})
	ds.Set(Element{Tag: NewTag(0x0002, 0x0012), VR: VRUI, Value: NewStrings(VRUI, "1.2.3.4.5")})
	return ds
}

func TestReadFileMetaTypedFields(t *testing.T) {
	raw := buildFileMetaBytes(t, metaElems())
	br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)

	preamble, err := readPreamble(br)
	if err != nil {
		t.Fatalf("readPreamble: %v", err)
	}
	if len(preamble) != 128 {
		t.Fatalf("preamble = %d bytes, want 128", len(preamble))
	}

	meta, err := readFileMeta(br)
	if err != nil {
		t.Fatalf("readFileMeta: %v", err)
	}
	if meta.TransferSyntaxUID != ExplicitVRLittleEndian {
		t.Errorf("TransferSyntaxUID = %q, want %q", meta.TransferSyntaxUID, ExplicitVRLittleEndian)
	}
	if meta.MediaStorageSOPClassUID != "1.2.840.10008.5.1.4.1.1.7" {
		t.Errorf("MediaStorageSOPClassUID = %q", meta.MediaStorageSOPClassUID)
	}
	if meta.MediaStorageSOPInstanceUID != "1.2.3.4.5.6.7.8.9" {
		t.Errorf("MediaStorageSOPInstanceUID = %q", meta.MediaStorageSOPInstanceUID)
	}
	if meta.ImplementationClassUID != "1.2.3.4.5" {
		t.Errorf("ImplementationClassUID = %q", meta.ImplementationClassUID)
	}
	if meta.Elements == nil || meta.Elements.Len() == 0 {
		t.Error("Elements should carry the full group-0002 dataset")
	}
}

func TestReadPreambleRejectsMissingDICM(t *testing.T) {
	bad := make([]byte, 132)
	copy(bad[128:], []byte("NOPE"))
	br := newBoundedReader(bytes.NewReader(bad), defaultMaxElementLen)
	if _, err := readPreamble(br); err == nil {
		t.Error("readPreamble should reject a stream without the DICM magic")
	}
}

// The group length bounds the file meta exactly: only the declared bytes are read,
// and the main dataset is not consumed as group-0002.
func TestReadFileMetaStopsAtGroupBoundary(t *testing.T) {
	metaRaw := buildFileMetaBytes(t, metaElems())
	// Append a main-dataset element after the file meta.
	main := buildElement(t, ExplicitVRLittleEndian, NewTag(0x0010, 0x0010), VRPN, NewStrings(VRPN, "Doe^Jane"))
	full := append(append([]byte{}, metaRaw...), main...)

	br := newBoundedReader(bytes.NewReader(full), defaultMaxElementLen)
	if _, err := readPreamble(br); err != nil {
		t.Fatalf("readPreamble: %v", err)
	}
	meta, err := readFileMeta(br)
	if err != nil {
		t.Fatalf("readFileMeta: %v", err)
	}
	// The main element must NOT have been read into the file-meta group.
	if _, ok := meta.Elements.Get(NewTag(0x0010, 0x0010)); ok {
		t.Error("file meta read past its group-length boundary into the main dataset")
	}
	// And the reader should be positioned exactly at the main element.
	ds, err := readDataSet(br, meta.TransferSyntaxUID, newReadConfig())
	if err != nil {
		t.Fatalf("readDataSet after meta: %v", err)
	}
	if v, ok := ds.GetString(NewTag(0x0010, 0x0010)); !ok || v != "Doe^Jane" {
		t.Errorf("main dataset after meta = %q,%v, want Doe^Jane", v, ok)
	}
}

func TestReadFileMetaMissingGroupLengthFails(t *testing.T) {
	// File meta that starts at (0002,0001) without (0002,0000) must be rejected.
	var group bytes.Buffer
	if err := writeDataSet(&group, metaElems(), ExplicitVRLittleEndian); err != nil {
		t.Fatalf("writeDataSet: %v", err)
	}
	var raw bytes.Buffer
	raw.Write(make([]byte, 128))
	raw.WriteString("DICM")
	raw.Write(group.Bytes()) // no (0002,0000) first

	br := newBoundedReader(bytes.NewReader(raw.Bytes()), defaultMaxElementLen)
	if _, err := readPreamble(br); err != nil {
		t.Fatalf("readPreamble: %v", err)
	}
	if _, err := readFileMeta(br); err == nil {
		t.Error("readFileMeta should reject a group without (0002,0000) group length")
	}
}

func TestReadFileMetaTruncatedGroupFails(t *testing.T) {
	raw := buildFileMetaBytes(t, metaElems())
	truncated := raw[:len(raw)-4] // chop the tail of the group
	br := newBoundedReader(bytes.NewReader(truncated), defaultMaxElementLen)
	if _, err := readPreamble(br); err != nil {
		t.Fatalf("readPreamble: %v", err)
	}
	_, err := readFileMeta(br)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("truncated file meta = %v, want io.ErrUnexpectedEOF", err)
	}
}
