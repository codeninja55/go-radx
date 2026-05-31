package dicom

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// buildElement is a tiny test helper: encode one element (header + value) in ts.
func buildElement(t *testing.T, ts TransferSyntax, tag Tag, vr VR, v Value) []byte {
	t.Helper()
	enc := encodingFor(ts)
	var vbuf bytes.Buffer
	n, err := encodeValue(&vbuf, v, enc)
	if err != nil {
		t.Fatalf("encodeValue: %v", err)
	}
	var hbuf bytes.Buffer
	if err := writeElementHeader(&hbuf, elementHeader{tag: tag, vr: vr, length: n}, ts); err != nil {
		t.Fatalf("writeElementHeader: %v", err)
	}
	return append(hbuf.Bytes(), vbuf.Bytes()...)
}

func TestReadDataSetExplicitLE(t *testing.T) {
	ts := ExplicitVRLittleEndian
	var raw bytes.Buffer
	raw.Write(buildElement(t, ts, NewTag(0x0008, 0x0060), VRCS, NewStrings(VRCS, "OT")))
	raw.Write(buildElement(t, ts, NewTag(0x0010, 0x0010), VRPN, NewStrings(VRPN, "Doe^Jane")))

	br := newBoundedReader(bytes.NewReader(raw.Bytes()), defaultMaxElementLen)
	ds, err := readDataSet(br, ts, newReadConfig())
	if err != nil {
		t.Fatalf("readDataSet: %v", err)
	}
	if ds.Len() != 2 {
		t.Fatalf("Len = %d, want 2", ds.Len())
	}
	if v, ok := ds.GetString(NewTag(0x0010, 0x0010)); !ok || v != "Doe^Jane" {
		t.Errorf("PatientName = %q,%v, want Doe^Jane", v, ok)
	}
	if v, ok := ds.GetString(NewTag(0x0008, 0x0060)); !ok || v != "OT" {
		t.Errorf("Modality = %q,%v, want OT", v, ok)
	}
}

func TestReadDataSetImplicitLE(t *testing.T) {
	ts := ImplicitVRLittleEndian
	var raw bytes.Buffer
	raw.Write(buildElement(t, ts, NewTag(0x0010, 0x0010), VRPN, NewStrings(VRPN, "Doe^Jane")))

	br := newBoundedReader(bytes.NewReader(raw.Bytes()), defaultMaxElementLen)
	ds, err := readDataSet(br, ts, newReadConfig())
	if err != nil {
		t.Fatalf("readDataSet: %v", err)
	}
	if v, ok := ds.GetString(NewTag(0x0010, 0x0010)); !ok || v != "Doe^Jane" {
		t.Errorf("PatientName = %q,%v, want Doe^Jane", v, ok)
	}
}

func TestReadDataSetBigEndian(t *testing.T) {
	ts := ExplicitVRBigEndian
	var raw bytes.Buffer
	raw.Write(buildElement(t, ts, NewTag(0x0028, 0x0010), VRUS, NewInts(VRUS, 256)))

	br := newBoundedReader(bytes.NewReader(raw.Bytes()), defaultMaxElementLen)
	ds, err := readDataSet(br, ts, newReadConfig())
	if err != nil {
		t.Fatalf("readDataSet: %v", err)
	}
	if n, ok := ds.GetInt(NewTag(0x0028, 0x0010)); !ok || n != 256 {
		t.Errorf("Rows = %d,%v, want 256", n, ok)
	}
}

// DCM-003: a stream ending inside a value field is a truncation, never a complete
// dataset.
func TestReadDataSetTruncatedValueFails(t *testing.T) {
	ts := ExplicitVRLittleEndian
	good := buildElement(t, ts, NewTag(0x0010, 0x0010), VRPN, NewStrings(VRPN, "Doe^Jane"))
	truncated := good[:len(good)-2] // chop two value bytes

	br := newBoundedReader(bytes.NewReader(truncated), defaultMaxElementLen)
	_, err := readDataSet(br, ts, newReadConfig())
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("truncated dataset = %v, want io.ErrUnexpectedEOF", err)
	}
}

// DCM-003: a clean EOF at a top-level tag boundary is a complete dataset, not an
// error.
func TestReadDataSetCleanEOFIsComplete(t *testing.T) {
	ts := ExplicitVRLittleEndian
	raw := buildElement(t, ts, NewTag(0x0010, 0x0010), VRPN, NewStrings(VRPN, "Doe^Jane"))
	br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
	ds, err := readDataSet(br, ts, newReadConfig())
	if err != nil {
		t.Fatalf("clean EOF should be a complete read, got %v", err)
	}
	if ds.Len() != 1 {
		t.Errorf("Len = %d, want 1", ds.Len())
	}
}

func TestReadDataSetStopAtPixelData(t *testing.T) {
	ts := ExplicitVRLittleEndian
	var raw bytes.Buffer
	raw.Write(buildElement(t, ts, NewTag(0x0010, 0x0010), VRPN, NewStrings(VRPN, "Doe^Jane")))
	raw.Write(buildElement(t, ts, NewTag(0x7FE0, 0x0010), VROB, NewBytes(VROB, []byte{1, 2, 3, 4})))

	br := newBoundedReader(bytes.NewReader(raw.Bytes()), defaultMaxElementLen)
	ds, err := readDataSet(br, ts, newReadConfig(WithStopAtPixelData()))
	if err != nil {
		t.Fatalf("readDataSet: %v", err)
	}
	if _, ok := ds.Get(NewTag(0x7FE0, 0x0010)); ok {
		t.Error("pixel data should be skipped with WithStopAtPixelData")
	}
	if _, ok := ds.Get(NewTag(0x0010, 0x0010)); !ok {
		t.Error("elements before pixel data should still be read")
	}
}

func TestWriteDataSetRoundTrip(t *testing.T) {
	for _, ts := range []TransferSyntax{ExplicitVRLittleEndian, ImplicitVRLittleEndian, ExplicitVRBigEndian} {
		t.Run(ts.Name(), func(t *testing.T) {
			ds := NewDataSet()
			ds.Set(Element{Tag: NewTag(0x0008, 0x0060), VR: VRCS, Value: NewStrings(VRCS, "OT")})
			ds.Set(Element{Tag: NewTag(0x0010, 0x0010), VR: VRPN, Value: NewStrings(VRPN, "Doe^Jane")})
			ds.Set(Element{Tag: NewTag(0x0028, 0x0010), VR: VRUS, Value: NewInts(VRUS, 256)})

			var buf bytes.Buffer
			if err := writeDataSet(&buf, ds, ts); err != nil {
				t.Fatalf("writeDataSet: %v", err)
			}
			br := newBoundedReader(bytes.NewReader(buf.Bytes()), defaultMaxElementLen)
			got, err := readDataSet(br, ts, newReadConfig())
			if err != nil {
				t.Fatalf("readDataSet: %v", err)
			}
			if got.Len() != ds.Len() {
				t.Fatalf("round-trip Len = %d, want %d", got.Len(), ds.Len())
			}
			if v, ok := got.GetString(NewTag(0x0010, 0x0010)); !ok || v != "Doe^Jane" {
				t.Errorf("round-trip PatientName = %q,%v", v, ok)
			}
			if n, ok := got.GetInt(NewTag(0x0028, 0x0010)); !ok || n != 256 {
				t.Errorf("round-trip Rows = %d,%v", n, ok)
			}
		})
	}
}

// A dataset re-encodes to exactly the bytes it was parsed from (lossless syntax).
func TestWriteDataSetByteIdentical(t *testing.T) {
	ts := ExplicitVRLittleEndian
	var raw bytes.Buffer
	raw.Write(buildElement(t, ts, NewTag(0x0008, 0x0060), VRCS, NewStrings(VRCS, "OT")))
	raw.Write(buildElement(t, ts, NewTag(0x0010, 0x0010), VRPN, NewStrings(VRPN, "Doe^Jane")))
	original := raw.Bytes()

	br := newBoundedReader(bytes.NewReader(original), defaultMaxElementLen)
	ds, err := readDataSet(br, ts, newReadConfig())
	if err != nil {
		t.Fatalf("readDataSet: %v", err)
	}
	var out bytes.Buffer
	if err := writeDataSet(&out, ds, ts); err != nil {
		t.Fatalf("writeDataSet: %v", err)
	}
	if !bytes.Equal(out.Bytes(), original) {
		t.Errorf("re-encode not byte-identical:\n got % x\nwant % x", out.Bytes(), original)
	}
}
