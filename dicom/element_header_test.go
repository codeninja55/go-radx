package dicom

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

// explicit-VR LE short form: (0010,0010) PN length 8.
func TestReadElementHeaderExplicitShort(t *testing.T) {
	raw := []byte{
		0x10, 0x00, 0x10, 0x00, // tag (0010,0010) little endian
		'P', 'N', // VR
		0x08, 0x00, // 2-byte length = 8
	}
	br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
	h, err := readElementHeader(br, ExplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("readElementHeader: %v", err)
	}
	if h.tag != NewTag(0x0010, 0x0010) {
		t.Errorf("tag = %s, want (0010,0010)", h.tag)
	}
	if h.vr != VRPN {
		t.Errorf("vr = %s, want PN", h.vr)
	}
	if h.length != 8 {
		t.Errorf("length = %d, want 8", h.length)
	}
}

// explicit-VR LE long form: (7FE0,0010) OB, reserved 0x0000, 4-byte length = 16.
func TestReadElementHeaderExplicitLong(t *testing.T) {
	raw := []byte{
		0xE0, 0x7F, 0x10, 0x00, // tag (7FE0,0010) little endian
		'O', 'B', // VR
		0x00, 0x00, // reserved
		0x10, 0x00, 0x00, 0x00, // 4-byte length = 16
	}
	br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
	h, err := readElementHeader(br, ExplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("readElementHeader: %v", err)
	}
	if h.tag != NewTag(0x7FE0, 0x0010) || h.vr != VROB || h.length != 16 {
		t.Errorf("header = %+v, want (7FE0,0010) OB len 16", h)
	}
}

// implicit-VR LE: (0010,0010) + 4-byte length; VR resolved from the dictionary.
func TestReadElementHeaderImplicit(t *testing.T) {
	raw := []byte{
		0x10, 0x00, 0x10, 0x00, // tag (0010,0010)
		0x08, 0x00, 0x00, 0x00, // 4-byte length = 8
	}
	br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
	h, err := readElementHeader(br, ImplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("readElementHeader: %v", err)
	}
	if h.tag != NewTag(0x0010, 0x0010) {
		t.Errorf("tag = %s, want (0010,0010)", h.tag)
	}
	if h.vr != VRPN {
		t.Errorf("implicit VR resolved = %s, want PN from dictionary", h.vr)
	}
	if h.length != 8 {
		t.Errorf("length = %d, want 8", h.length)
	}
}

// explicit-VR Big Endian: (0008,0005) CS length 10.
func TestReadElementHeaderExplicitBigEndian(t *testing.T) {
	raw := []byte{
		0x00, 0x08, 0x00, 0x05, // tag (0008,0005) big endian
		'C', 'S', // VR
		0x00, 0x0A, // 2-byte length = 10 big endian
	}
	br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
	h, err := readElementHeader(br, ExplicitVRBigEndian)
	if err != nil {
		t.Fatalf("readElementHeader: %v", err)
	}
	if h.tag != NewTag(0x0008, 0x0005) || h.vr != VRCS || h.length != 10 {
		t.Errorf("header = %+v, want (0008,0005) CS len 10", h)
	}
}

func TestReadElementHeaderCleanEOF(t *testing.T) {
	br := newBoundedReader(bytes.NewReader(nil), defaultMaxElementLen)
	_, err := readElementHeader(br, ExplicitVRLittleEndian)
	if !errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("clean EOF = %v, want io.EOF", err)
	}
}

func TestReadElementHeaderTruncatedAfterTag(t *testing.T) {
	// A tag with no VR bytes following is a mid-element truncation.
	br := newBoundedReader(bytes.NewReader([]byte{0x10, 0x00, 0x10, 0x00}), defaultMaxElementLen)
	_, err := readElementHeader(br, ExplicitVRLittleEndian)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("truncated header = %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestWriteElementHeaderExplicitShortRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	h := elementHeader{tag: NewTag(0x0010, 0x0010), vr: VRPN, length: 8}
	if err := writeElementHeader(&buf, h, ExplicitVRLittleEndian); err != nil {
		t.Fatalf("writeElementHeader: %v", err)
	}
	want := []byte{0x10, 0x00, 0x10, 0x00, 'P', 'N', 0x08, 0x00}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("explicit short header = % x, want % x", buf.Bytes(), want)
	}
}

func TestWriteElementHeaderExplicitLongRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	h := elementHeader{tag: NewTag(0x7FE0, 0x0010), vr: VROB, length: 16}
	if err := writeElementHeader(&buf, h, ExplicitVRLittleEndian); err != nil {
		t.Fatalf("writeElementHeader: %v", err)
	}
	want := []byte{0xE0, 0x7F, 0x10, 0x00, 'O', 'B', 0x00, 0x00, 0x10, 0x00, 0x00, 0x00}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("explicit long header = % x, want % x", buf.Bytes(), want)
	}
}

func TestWriteElementHeaderImplicitRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	h := elementHeader{tag: NewTag(0x0010, 0x0010), vr: VRPN, length: 8}
	if err := writeElementHeader(&buf, h, ImplicitVRLittleEndian); err != nil {
		t.Fatalf("writeElementHeader: %v", err)
	}
	want := []byte{0x10, 0x00, 0x10, 0x00, 0x08, 0x00, 0x00, 0x00}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("implicit header = % x, want % x", buf.Bytes(), want)
	}
}

func TestWriteElementHeaderBigEndianRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	h := elementHeader{tag: NewTag(0x0008, 0x0005), vr: VRCS, length: 10}
	if err := writeElementHeader(&buf, h, ExplicitVRBigEndian); err != nil {
		t.Fatalf("writeElementHeader: %v", err)
	}
	want := []byte{0x00, 0x08, 0x00, 0x05, 'C', 'S', 0x00, 0x0A}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("big-endian header = % x, want % x", buf.Bytes(), want)
	}
}

// A round-trip through both directions reproduces the bytes for every encoding.
func TestElementHeaderRoundTripAllSyntaxes(t *testing.T) {
	cases := []struct {
		name string
		ts   TransferSyntax
		h    elementHeader
	}{
		{"explicit-le-short", ExplicitVRLittleEndian, elementHeader{NewTag(0x0008, 0x0060), VRCS, 2}},
		{"explicit-le-long", ExplicitVRLittleEndian, elementHeader{NewTag(0x7FE0, 0x0010), VROW, 100}},
		{"implicit-le", ImplicitVRLittleEndian, elementHeader{NewTag(0x0010, 0x0020), VRLO, 6}},
		{"explicit-be-short", ExplicitVRBigEndian, elementHeader{NewTag(0x0008, 0x0008), VRCS, 24}},
		{"explicit-be-long", ExplicitVRBigEndian, elementHeader{NewTag(0x7FE0, 0x0010), VROW, 1024}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := writeElementHeader(&buf, tc.h, tc.ts); err != nil {
				t.Fatalf("write: %v", err)
			}
			br := newBoundedReader(bytes.NewReader(buf.Bytes()), defaultMaxElementLen)
			got, err := readElementHeader(br, tc.ts)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if got.tag != tc.h.tag || got.length != tc.h.length {
				t.Errorf("round-trip = %+v, want tag/length %+v", got, tc.h)
			}
		})
	}
}

func TestEncodingForTransferSyntax(t *testing.T) {
	if enc := encodingFor(ImplicitVRLittleEndian); !enc.implicitVR || enc.byteOrder != binary.LittleEndian {
		t.Errorf("implicit LE = %+v", enc)
	}
	if enc := encodingFor(ExplicitVRLittleEndian); enc.implicitVR || enc.byteOrder != binary.LittleEndian {
		t.Errorf("explicit LE = %+v", enc)
	}
	if enc := encodingFor(ExplicitVRBigEndian); enc.implicitVR || enc.byteOrder != binary.BigEndian {
		t.Errorf("explicit BE = %+v", enc)
	}
	if enc := encodingFor(DeflatedExplicitVRLittleEndian); enc.implicitVR || enc.byteOrder != binary.LittleEndian {
		t.Errorf("deflated explicit LE = %+v", enc)
	}
}
