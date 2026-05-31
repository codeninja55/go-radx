package dicom

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

func TestDecodeStringsSingleValue(t *testing.T) {
	br := newBoundedReader(bytes.NewReader([]byte("ISO_IR 192")), defaultMaxElementLen)
	v, err := decodeValue(br, elementHeader{tag: NewTag(0x0008, 0x0005), vr: VRCS, length: 10}, encodingFor(ExplicitVRLittleEndian))
	if err != nil {
		t.Fatalf("decodeValue: %v", err)
	}
	sv, ok := v.(*Strings)
	if !ok {
		t.Fatalf("value is %T, want *Strings", v)
	}
	got := sv.Strings()
	if len(got) != 1 || got[0] != "ISO_IR 192" {
		t.Errorf("Strings = %q, want [ISO_IR 192]", got)
	}
}

func TestDecodeStringsMultiValueTrimsPad(t *testing.T) {
	// "DERIVED\SECONDARY\OTHER " is 23 chars + 1 SPACE pad = 24 bytes.
	raw := []byte("DERIVED\\SECONDARY\\OTHER ")
	br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
	v, err := decodeValue(br, elementHeader{tag: NewTag(0x0008, 0x0008), vr: VRCS, length: uint32(len(raw))}, encodingFor(ExplicitVRLittleEndian))
	if err != nil {
		t.Fatalf("decodeValue: %v", err)
	}
	got := v.(*Strings).Strings()
	want := []string{"DERIVED", "SECONDARY", "OTHER"}
	if len(got) != 3 {
		t.Fatalf("Strings = %q, want 3 values", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Strings[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestDecodeUITrimsNullPad(t *testing.T) {
	// "1.2.840.10008.1.2.2" is 19 chars + 1 NULL pad = 20 bytes.
	raw := append([]byte("1.2.840.10008.1.2.2"), 0x00)
	br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
	v, err := decodeValue(br, elementHeader{tag: NewTag(0x0002, 0x0010), vr: VRUI, length: uint32(len(raw))}, encodingFor(ExplicitVRLittleEndian))
	if err != nil {
		t.Fatalf("decodeValue: %v", err)
	}
	got := v.(*Strings).Strings()
	if len(got) != 1 || got[0] != "1.2.840.10008.1.2.2" {
		t.Errorf("UI = %q, want [1.2.840.10008.1.2.2] (NULL trimmed)", got)
	}
}

func TestDecodeIntsLittleEndian(t *testing.T) {
	raw := []byte{0x01, 0x00, 0x02, 0x00} // US 1, US 2
	br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
	v, err := decodeValue(br, elementHeader{tag: NewTag(0x0028, 0x0010), vr: VRUS, length: 4}, encodingFor(ExplicitVRLittleEndian))
	if err != nil {
		t.Fatalf("decodeValue: %v", err)
	}
	got := v.(*Ints).Ints()
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("Ints = %v, want [1 2]", got)
	}
}

func TestDecodeIntsBigEndian(t *testing.T) {
	raw := []byte{0x00, 0x01, 0x00, 0x02} // US 1, US 2 big endian
	br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
	v, err := decodeValue(br, elementHeader{tag: NewTag(0x0028, 0x0010), vr: VRUS, length: 4}, encodingFor(ExplicitVRBigEndian))
	if err != nil {
		t.Fatalf("decodeValue: %v", err)
	}
	got := v.(*Ints).Ints()
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("Ints = %v, want [1 2]", got)
	}
}

func TestDecodeSignedInts(t *testing.T) {
	raw := []byte{0xFF, 0xFF} // SS -1
	br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
	v, err := decodeValue(br, elementHeader{tag: NewTag(0x0028, 0x0106), vr: VRSS, length: 2}, encodingFor(ExplicitVRLittleEndian))
	if err != nil {
		t.Fatalf("decodeValue: %v", err)
	}
	if got := v.(*Ints).Ints(); len(got) != 1 || got[0] != -1 {
		t.Errorf("SS = %v, want [-1]", got)
	}
}

func TestDecodeFloatsLittleEndian(t *testing.T) {
	raw := make([]byte, 8)
	binary.LittleEndian.PutUint64(raw, 0x3FF0000000000000) // 1.0 as float64
	br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
	v, err := decodeValue(br, elementHeader{tag: NewTag(0x0028, 0x1052), vr: VRFD, length: 8}, encodingFor(ExplicitVRLittleEndian))
	if err != nil {
		t.Fatalf("decodeValue: %v", err)
	}
	if got := v.(*Floats).Floats(); len(got) != 1 || got[0] != 1.0 {
		t.Errorf("FD = %v, want [1]", got)
	}
}

func TestDecodeDecimalsDS(t *testing.T) {
	raw := []byte("1.5\\2.5 ") // 7 chars + SPACE pad = 8
	br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
	v, err := decodeValue(br, elementHeader{tag: NewTag(0x0028, 0x1050), vr: VRDS, length: uint32(len(raw))}, encodingFor(ExplicitVRLittleEndian))
	if err != nil {
		t.Fatalf("decodeValue: %v", err)
	}
	got := v.(*Decimals).Decimals()
	if len(got) != 2 || got[0].String() != "1.5" || got[1].String() != "2.5" {
		t.Errorf("DS = %v, want [1.5 2.5]", got)
	}
}

func TestDecodeTagsAT(t *testing.T) {
	raw := []byte{0x10, 0x00, 0x20, 0x00} // (0010,0020) little endian
	br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
	v, err := decodeValue(br, elementHeader{tag: NewTag(0x0020, 0x9165), vr: VRAT, length: 4}, encodingFor(ExplicitVRLittleEndian))
	if err != nil {
		t.Fatalf("decodeValue: %v", err)
	}
	if got := v.(*Tags).Tags(); len(got) != 1 || got[0] != NewTag(0x0010, 0x0020) {
		t.Errorf("AT = %v, want [(0010,0020)]", got)
	}
}

func TestDecodeBytesOB(t *testing.T) {
	raw := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
	v, err := decodeValue(br, elementHeader{tag: NewTag(0x7FE0, 0x0010), vr: VROB, length: 4}, encodingFor(ExplicitVRLittleEndian))
	if err != nil {
		t.Fatalf("decodeValue: %v", err)
	}
	if got := v.(*Bytes).Bytes(); !bytes.Equal(got, raw) {
		t.Errorf("OB = % x, want % x", got, raw)
	}
}

// DCM-005: an SQ value is parsed structurally into nested datasets, never dropped.
// A defined-length sequence with one defined-length item carrying a PN element
// navigates to that element and re-encodes byte-identically.
func TestDecodeSQStructured(t *testing.T) {
	ts := ExplicitVRLittleEndian
	var inner bytes.Buffer
	inner.Write(buildElement(t, ts, NewTag(0x0010, 0x0010), VRPN, NewStrings(VRPN, "Doe^Jane")))

	var sq bytes.Buffer
	sq.Write(leTag(0xFFFE, 0xE000))     // item header
	sq.Write(le32(uint32(inner.Len()))) // defined-length item
	sq.Write(inner.Bytes())             // item content

	br := newBoundedReader(bytes.NewReader(sq.Bytes()), defaultMaxElementLen)
	seq, err := decodeSequence(br, elementHeader{tag: NewTag(0x0040, 0xA730), vr: VRSQ, length: uint32(sq.Len())}, ts, newReadConfig(), 1)
	if err != nil {
		t.Fatalf("decodeSequence: %v", err)
	}
	if seq.Len() != 1 {
		t.Fatalf("Len = %d, want 1", seq.Len())
	}
	if seq.undefinedLength {
		t.Error("defined-length sequence parsed as undefined-length")
	}
	for it := range seq.Items() {
		if it.undefinedLength {
			t.Error("defined-length item parsed as undefined-length")
		}
		if v, ok := it.DataSet.GetString(NewTag(0x0010, 0x0010)); !ok || v != "Doe^Jane" {
			t.Errorf("item PatientName = %q,%v, want Doe^Jane", v, ok)
		}
	}

	// Re-encode the value field; it reproduces the source bytes exactly.
	var out bytes.Buffer
	if _, err := encodeSequenceValue(&out, seq, ts); err != nil {
		t.Fatalf("encodeSequenceValue: %v", err)
	}
	if !bytes.Equal(out.Bytes(), sq.Bytes()) {
		t.Errorf("re-encoded SQ value not byte-identical:\n got % x\nwant % x", out.Bytes(), sq.Bytes())
	}
}

func TestDecodeTruncatedValueIsUnexpectedEOF(t *testing.T) {
	// A value field declaring 8 bytes over a 4-byte stream is a truncation.
	br := newBoundedReader(bytes.NewReader([]byte("abcd")), defaultMaxElementLen)
	_, err := decodeValue(br, elementHeader{tag: NewTag(0x0010, 0x0010), vr: VRPN, length: 8}, encodingFor(ExplicitVRLittleEndian))
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("truncated value = %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestEncodeValueRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		vr   VR
		val  Value
		ts   TransferSyntax
	}{
		{"strings-le", VRCS, NewStrings(VRCS, "DERIVED", "SECONDARY"), ExplicitVRLittleEndian},
		{"ui-le", VRUI, NewStrings(VRUI, "1.2.840.10008.1.2.1"), ExplicitVRLittleEndian},
		{"ints-le", VRUS, NewInts(VRUS, 1, 2, 3), ExplicitVRLittleEndian},
		{"ints-be", VRUS, NewInts(VRUS, 256, 257), ExplicitVRBigEndian},
		{"floats-le", VRFD, NewFloats(VRFD, 1.5, 2.5), ExplicitVRLittleEndian},
		{"decimals-le", VRDS, mustDecimals(t, "1.5", "2.5"), ExplicitVRLittleEndian},
		{"tags-le", VRAT, NewTags(NewTag(0x0010, 0x0020)), ExplicitVRLittleEndian},
		{"bytes-le", VROB, NewBytes(VROB, []byte{1, 2, 3, 4}), ExplicitVRLittleEndian},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			enc := encodingFor(tc.ts)
			var buf bytes.Buffer
			n, err := encodeValue(&buf, tc.val, enc)
			if err != nil {
				t.Fatalf("encodeValue: %v", err)
			}
			if n != tc.val.EncodedLen(enc.byteOrder) {
				t.Errorf("encodeValue wrote %d, EncodedLen = %d", n, tc.val.EncodedLen(enc.byteOrder))
			}
			if n%2 != 0 {
				t.Errorf("encoded value length %d is odd; must be even", n)
			}
			br := newBoundedReader(bytes.NewReader(buf.Bytes()), defaultMaxElementLen)
			h := elementHeader{tag: NewTag(0x0010, 0x0010), vr: tc.vr, length: n}
			got, err := decodeValue(br, h, enc)
			if err != nil {
				t.Fatalf("decodeValue: %v", err)
			}
			if got.EncodedLen(enc.byteOrder) != n {
				t.Errorf("round-trip EncodedLen = %d, want %d", got.EncodedLen(enc.byteOrder), n)
			}
		})
	}
}

// Clone must deep-copy a sequence so a mutation of the clone's nested item dataset
// never reaches the source (Codex DCM-016).
func TestCloneDeepCopiesSequence(t *testing.T) {
	src := NewDataSet()
	item := NewDataSet()
	item.SetString(NewTag(0x0010, 0x0010), "Doe^Jane")
	src.Set(Element{Tag: NewTag(0x0040, 0xA730), VR: VRSQ, Value: NewSequenceValue(NewSequence(item))})

	clone := src.Clone()
	clonedSeq, ok := clone.GetSequence(NewTag(0x0040, 0xA730))
	if !ok {
		t.Fatal("clone lost the sequence")
	}
	for it := range clonedSeq.Items() {
		it.DataSet.SetString(NewTag(0x0010, 0x0010), "Mutated^Clone")
	}

	srcSeq, _ := src.GetSequence(NewTag(0x0040, 0xA730))
	for it := range srcSeq.Items() {
		if v, _ := it.DataSet.GetString(NewTag(0x0010, 0x0010)); v != "Doe^Jane" {
			t.Errorf("mutating the clone's nested item reached the source: %q (Codex DCM-016)", v)
		}
	}
}

func mustDecimals(t *testing.T, ss ...string) Value {
	t.Helper()
	ds := make([]Decimal, len(ss))
	for i, s := range ss {
		d, err := ParseDecimal(s)
		if err != nil {
			t.Fatalf("ParseDecimal(%q): %v", s, err)
		}
		ds[i] = d
	}
	return NewDecimals(VRDS, ds...)
}
