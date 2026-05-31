package dicom

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// DCM-002 (byte order): a dataset declared Explicit VR Big Endian is actually
// encoded big-endian on the wire, not silently Little Endian. Read the raw value
// bytes back and assert the byte order.
func TestRegressionDCM002BigEndianEncoding(t *testing.T) {
	ds := NewDataSet()
	ds.Set(Element{Tag: NewTag(0x0008, 0x0016), VR: VRUI, Value: NewStrings(VRUI, "1.2.840.10008.5.1.4.1.1.7")})
	ds.Set(Element{Tag: NewTag(0x0008, 0x0018), VR: VRUI, Value: NewStrings(VRUI, "1.2.3.4.5")})
	ds.Set(Element{Tag: NewTag(0x0028, 0x0010), VR: VRUS, Value: NewInts(VRUS, 0x0102)}) // Rows

	f := &File{Meta: &FileMeta{
		MediaStorageSOPClassUID:    "1.2.840.10008.5.1.4.1.1.7",
		MediaStorageSOPInstanceUID: "1.2.3.4.5",
		TransferSyntaxUID:          ExplicitVRBigEndian,
	}, DataSet: ds}

	var buf bytes.Buffer
	if err := Write(&buf, f); err != nil {
		t.Fatalf("Write: %v", err)
	}
	raw := buf.Bytes()

	// Locate the (0028,0010) Rows US value in the raw bytes. Under big endian the
	// tag bytes are 00 28 00 10 and the value 0x0102 is stored as 01 02; under little
	// endian they would be 28 00 10 00 and 02 01. Assert big-endian layout.
	beTag := []byte{0x00, 0x28, 0x00, 0x10, 'U', 'S', 0x00, 0x02, 0x01, 0x02}
	if !bytes.Contains(raw, beTag) {
		t.Error("Explicit VR BE output does not contain the big-endian (0028,0010) US encoding")
	}
	leTag := []byte{0x28, 0x00, 0x10, 0x00, 'U', 'S', 0x02, 0x00, 0x02, 0x01}
	if bytes.Contains(raw, leTag) {
		t.Error("Explicit VR BE output contains a little-endian encoding; transfer syntax was not honoured (DCM-002)")
	}

	// And it re-reads to the same value, proving the reader honours BE too.
	got, err := Read(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if n, ok := got.DataSet.GetInt(NewTag(0x0028, 0x0010)); !ok || n != 0x0102 {
		t.Errorf("round-trip Rows = %d, want %d", n, 0x0102)
	}
}

// DCM-002 (deflate): a dataset declared Deflated Explicit VR LE is actually
// DEFLATE-compressed after the file-meta group, not stored uncompressed. Inflate
// the trailing bytes and parse them.
func TestRegressionDCM002Deflated(t *testing.T) {
	ds := NewDataSet()
	ds.Set(Element{Tag: NewTag(0x0008, 0x0016), VR: VRUI, Value: NewStrings(VRUI, "1.2.840.10008.5.1.4.1.1.7")})
	ds.Set(Element{Tag: NewTag(0x0008, 0x0018), VR: VRUI, Value: NewStrings(VRUI, "1.2.3.4.5")})
	ds.Set(Element{Tag: NewTag(0x0010, 0x0010), VR: VRPN, Value: NewStrings(VRPN, "Doe^Jane")})

	f := &File{Meta: &FileMeta{
		MediaStorageSOPClassUID:    "1.2.840.10008.5.1.4.1.1.7",
		MediaStorageSOPInstanceUID: "1.2.3.4.5",
		TransferSyntaxUID:          DeflatedExplicitVRLittleEndian,
	}, DataSet: ds}

	var buf bytes.Buffer
	if err := Write(&buf, f); err != nil {
		t.Fatalf("Write: %v", err)
	}
	raw := buf.Bytes()

	// The plaintext "Doe^Jane" must NOT appear uncompressed in the output.
	if bytes.Contains(raw, []byte("Doe^Jane")) {
		t.Error("Deflated output contains uncompressed plaintext; the dataset was not deflated (DCM-002)")
	}

	// Skip preamble+DICM+file-meta group, then inflate the remainder and confirm it
	// decodes to the dataset.
	br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
	if _, err := readPreamble(br); err != nil {
		t.Fatalf("readPreamble: %v", err)
	}
	h, _ := readElementHeader(br, ExplicitVRLittleEndian)
	gv, _ := decodeValue(br, h, encodingFor(ExplicitVRLittleEndian))
	groupLen := gv.(*Ints).Ints()[0]
	mainStart := br.offset() + groupLen

	fr := flate.NewReader(bytes.NewReader(raw[mainStart:]))
	defer fr.Close()
	inflated, err := io.ReadAll(fr)
	if err != nil {
		t.Fatalf("inflate: %v", err)
	}
	if !bytes.Contains(inflated, []byte("Doe^Jane")) {
		t.Error("inflated deflate stream does not contain the dataset; not a valid DEFLATE main dataset (DCM-002)")
	}
}

// DCM-003: reading a truncated fixture produced by the testdata gen/truncate
// generator returns an io.ErrUnexpectedEOF-class error, not a short complete
// dataset.
func TestRegressionDCM003TruncatedFixture(t *testing.T) {
	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	// Generate liver.truncated.dcm = first 600 bytes of liver.dcm (well into the
	// main dataset, mid-element).
	cmd := exec.Command("go", "run", "./testdata/dicom/gen", "-in", "liver.dcm", "-bytes", "600")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("truncate generator: %v\n%s", err, out)
	}
	truncatedPath := filepath.Join(repoRoot, "testdata", "dicom", "liver.truncated.dcm")
	defer os.Remove(truncatedPath)

	_, err = ReadFile(truncatedPath)
	if err == nil {
		t.Fatal("reading a truncated file returned no error: truncation accepted as complete (DCM-003)")
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("truncated file error = %v, want io.ErrUnexpectedEOF class (DCM-003)", err)
	}
}

// DCM-007 (emitted bytes): an odd-length character-VR value is written with an even
// value field ending in the correct pad byte (SPACE for AE/CS/DA/DS/IS/LO/PN, NULL
// for UI). Read the raw value field back and assert.
func TestRegressionDCM007EmittedPadBytes(t *testing.T) {
	cases := []struct {
		vr      VR
		value   string // odd-length so padding is exercised
		wantPad byte
	}{
		{VRAE, "AET", 0x20},
		{VRCS, "ODD", 0x20},
		{VRDA, "2017010", 0x20}, // 7 chars (odd), shape is irrelevant to the byte assertion
		{VRDS, "1.5", 0x20},
		{VRIS, "123", 0x20},
		{VRLO, "ID1", 0x20},
		{VRPN, "Doe", 0x20},
		{VRUI, "1.2.3", 0x00},
	}
	for _, tc := range cases {
		t.Run(tc.vr.String(), func(t *testing.T) {
			assertEmittedPad(t, tc.vr, tc.value, tc.wantPad)
		})
	}
}

func assertEmittedPad(t *testing.T, vr VR, value string, wantPad byte) {
	t.Helper()
	if len(value)%2 == 0 {
		t.Fatalf("test value %q must be odd-length to exercise padding", value)
	}
	var val Value
	if vr == VRDS || vr == VRIS {
		d, err := ParseDecimal(value)
		if err != nil {
			t.Fatalf("ParseDecimal(%q): %v", value, err)
		}
		val = NewDecimals(vr, d)
	} else {
		val = NewStrings(vr, value)
	}

	enc := encodingFor(ExplicitVRLittleEndian)
	var buf bytes.Buffer
	n, err := encodeValue(&buf, val, enc)
	if err != nil {
		t.Fatalf("encodeValue: %v", err)
	}
	if n%2 != 0 {
		t.Errorf("%s emitted value length %d is odd; must be padded to even (DCM-007)", vr, n)
	}
	field := buf.Bytes()
	if len(field) == 0 || field[len(field)-1] != wantPad {
		t.Errorf("%s emitted value field % x does not end in pad byte %#x (DCM-007)", vr, field, wantPad)
	}
}

// DCM-009: the writer validates UIDs through the single ParseUID path, rejecting
// the invalid forms a weaker local validator accepted ("1..2", "1.02").
func TestRegressionDCM009WriterUsesParseUID(t *testing.T) {
	invalid := []string{"1..2", "1.02", "1.2.", ".1.2"}
	for _, bad := range invalid {
		t.Run(bad, func(t *testing.T) {
			// Confirm the package validator rejects it (the single source of truth).
			if _, err := ParseUID(bad); err == nil {
				t.Fatalf("ParseUID(%q) accepted an invalid UID; validator is the contract", bad)
			}
			// And the writer rejects it through that same path.
			f := &File{Meta: &FileMeta{
				MediaStorageSOPClassUID:    SOPClassUID(bad),
				MediaStorageSOPInstanceUID: "1.2.3.4.5",
				TransferSyntaxUID:          ExplicitVRLittleEndian,
			}, DataSet: NewDataSet()}
			var buf bytes.Buffer
			if err := Write(&buf, f); err == nil {
				t.Errorf("Write accepted invalid file-meta UID %q; writer must reject through ParseUID (DCM-009)", bad)
			}
			if buf.Len() != 0 {
				t.Errorf("Write emitted %d bytes before rejecting invalid UID %q", buf.Len(), bad)
			}
		})
	}
}

// Cross-check: the binary layout of the (0028,0010) value uses the declared byte
// order, asserted directly with binary.Read for the LE path.
func TestRegressionByteOrderLittleEndian(t *testing.T) {
	var vbuf bytes.Buffer
	if _, err := encodeValue(&vbuf, NewInts(VRUS, 0x0102), encodingFor(ExplicitVRLittleEndian)); err != nil {
		t.Fatalf("encodeValue: %v", err)
	}
	if got := binary.LittleEndian.Uint16(vbuf.Bytes()); got != 0x0102 {
		t.Errorf("LE US value = %#x, want 0x0102", got)
	}
}
