package dicom

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func sampleFile(ts TransferSyntax) *File {
	ds := NewDataSet()
	ds.Set(Element{Tag: NewTag(0x0008, 0x0016), VR: VRUI, Value: NewStrings(VRUI, "1.2.840.10008.5.1.4.1.1.7")})
	ds.Set(Element{Tag: NewTag(0x0008, 0x0018), VR: VRUI, Value: NewStrings(VRUI, "1.2.3.4.5.6.7.8.9")})
	ds.Set(Element{Tag: NewTag(0x0008, 0x0060), VR: VRCS, Value: NewStrings(VRCS, "OT")})
	ds.Set(Element{Tag: NewTag(0x0010, 0x0010), VR: VRPN, Value: NewStrings(VRPN, "Doe^Jane")})
	ds.Set(Element{Tag: NewTag(0x0028, 0x0010), VR: VRUS, Value: NewInts(VRUS, 256)})
	return &File{
		Meta: &FileMeta{
			MediaStorageSOPClassUID:    "1.2.840.10008.5.1.4.1.1.7",
			MediaStorageSOPInstanceUID: "1.2.3.4.5.6.7.8.9",
			TransferSyntaxUID:          ts,
		},
		DataSet: ds,
	}
}

func TestWriteReadRoundTripAllSyntaxes(t *testing.T) {
	for _, ts := range []TransferSyntax{
		ExplicitVRLittleEndian,
		ImplicitVRLittleEndian,
		ExplicitVRBigEndian,
		DeflatedExplicitVRLittleEndian,
	} {
		t.Run(ts.Name(), func(t *testing.T) {
			f := sampleFile(ts)
			var buf bytes.Buffer
			if err := Write(&buf, f); err != nil {
				t.Fatalf("Write: %v", err)
			}
			got, err := Read(bytes.NewReader(buf.Bytes()))
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if got.Meta.TransferSyntaxUID != ts {
				t.Errorf("round-trip TransferSyntaxUID = %q, want %q", got.Meta.TransferSyntaxUID, ts)
			}
			if v, ok := got.DataSet.GetString(NewTag(0x0010, 0x0010)); !ok || v != "Doe^Jane" {
				t.Errorf("round-trip PatientName = %q,%v", v, ok)
			}
			if n, ok := got.DataSet.GetInt(NewTag(0x0028, 0x0010)); !ok || n != 256 {
				t.Errorf("round-trip Rows = %d,%v", n, ok)
			}
		})
	}
}

func TestWriteRejectsUnsupportedTransferSyntax(t *testing.T) {
	f := sampleFile(JPEGBaseline8Bit) // encapsulated; not writable as a main dataset
	var buf bytes.Buffer
	if err := Write(&buf, f); err == nil {
		t.Error("Write should reject an unsupported (encapsulated) transfer syntax before writing")
	}
	if buf.Len() != 0 {
		t.Errorf("Write emitted %d bytes for an unsupported syntax; must reject before writing", buf.Len())
	}
}

func TestReadFileAndWriteFile(t *testing.T) {
	f := sampleFile(ExplicitVRLittleEndian)
	dir := t.TempDir()
	path := filepath.Join(dir, "out.dcm")
	if err := WriteFile(path, f); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if v, ok := got.DataSet.GetString(NewTag(0x0010, 0x0010)); !ok || v != "Doe^Jane" {
		t.Errorf("ReadFile PatientName = %q,%v", v, ok)
	}
}

func TestReadFixturesRoundTrip(t *testing.T) {
	// liver.dcm and MR2_UNCI.dcm are Explicit VR LE; SC_rgb_expb.dcm is Explicit VR BE.
	fixtures := []string{"liver.dcm", "MR2_UNCI.dcm", "SC_rgb_expb.dcm"}
	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "testdata", "dicom", name)
			f, err := ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile(%s): %v", name, err)
			}
			if f.Meta.TransferSyntaxUID == "" {
				t.Fatal("missing transfer syntax")
			}
			if f.DataSet.Len() == 0 {
				t.Error("main dataset is empty")
			}
		})
	}
}

// Parse -> write -> parse equality on an uncompressed fixture (SQ preserved
// opaquely). Round-trip the typed values that must survive.
func TestFixtureParseWriteParseEquality(t *testing.T) {
	path := filepath.Join("..", "testdata", "dicom", "liver.dcm")
	first, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var buf bytes.Buffer
	if err := Write(&buf, first); err != nil {
		t.Fatalf("Write: %v", err)
	}
	second, err := Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("re-Read: %v", err)
	}
	if first.DataSet.Len() != second.DataSet.Len() {
		t.Errorf("element count changed across round-trip: %d -> %d", first.DataSet.Len(), second.DataSet.Len())
	}
	for e := range first.DataSet.All() {
		if _, ok := second.DataSet.Get(e.Tag); !ok {
			t.Errorf("element %s lost across round-trip", e.Tag)
		}
	}
}

// Byte-identical re-encode of the uncompressed fixtures: the bytes after the
// file-meta group must reproduce exactly (SQ preserved opaquely). This is the
// strongest round-trip guarantee for a lossless transfer syntax with no re-ordering
// ambiguity in the source. The file-meta group is excluded because the writer
// recomputes its group length and normalises required elements; the main dataset
// is the byte-stability target.
func TestFixtureMainDataSetByteIdentical(t *testing.T) {
	// liver.dcm (Explicit VR LE) and SC_rgb_expb.dcm (Explicit VR BE) have a clean
	// ascending-tag main dataset.
	for _, name := range []string{"liver.dcm", "SC_rgb_expb.dcm"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "testdata", "dicom", name))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			f, err := Read(bytes.NewReader(raw))
			if err != nil {
				t.Fatalf("Read: %v", err)
			}

			// Locate the original main dataset bytes: skip preamble+DICM, read the
			// group-length element, skip its declared group bytes.
			br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
			if _, err := readPreamble(br); err != nil {
				t.Fatalf("readPreamble: %v", err)
			}
			h, _ := readElementHeader(br, ExplicitVRLittleEndian)
			gv, _ := decodeValue(br, h, encodingFor(ExplicitVRLittleEndian))
			groupLen := gv.(*Ints).Ints()[0]
			mainStart := br.offset() + groupLen
			originalMain := raw[mainStart:]

			var out bytes.Buffer
			if err := writeDataSet(&out, f.DataSet, f.Meta.TransferSyntaxUID); err != nil {
				t.Fatalf("writeDataSet: %v", err)
			}
			if !bytes.Equal(out.Bytes(), originalMain) {
				t.Errorf("%s main dataset re-encode not byte-identical: got %d bytes, want %d bytes",
					name, out.Len(), len(originalMain))
			}
		})
	}
}

func TestReadTruncatedStreamFails(t *testing.T) {
	f := sampleFile(ExplicitVRLittleEndian)
	var buf bytes.Buffer
	if err := Write(&buf, f); err != nil {
		t.Fatalf("Write: %v", err)
	}
	truncated := buf.Bytes()[:buf.Len()-3]
	_, err := Read(bytes.NewReader(truncated))
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("truncated stream = %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestReadFileMissingPath(t *testing.T) {
	if _, err := ReadFile(filepath.Join(os.TempDir(), "does-not-exist-12345.dcm")); err == nil {
		t.Error("ReadFile of a missing path should error")
	}
}
