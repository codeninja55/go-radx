package pdu

import (
	"bytes"
	"testing"
)

func TestPDUTypeString(t *testing.T) {
	tests := map[PDUType]string{
		PDUTypeAssociateRQ: "A-ASSOCIATE-RQ",
		PDUTypeAssociateAC: "A-ASSOCIATE-AC",
		PDUTypeAssociateRJ: "A-ASSOCIATE-RJ",
		PDUTypeData:        "P-DATA-TF",
		PDUTypeReleaseRQ:   "A-RELEASE-RQ",
		PDUTypeReleaseRP:   "A-RELEASE-RP",
		PDUTypeAbort:       "A-ABORT",
	}
	for pt, want := range tests {
		if got := pt.String(); got != want {
			t.Errorf("PDUType(%#02x).String() = %q, want %q", byte(pt), got, want)
		}
	}
}

func TestWriteReadHeaderRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := writeHeader(&buf, PDUTypeData, 0x12345678); err != nil {
		t.Fatalf("writeHeader: %v", err)
	}
	// PDU header is 6 bytes: type(1) reserved(1) length(4 big-endian).
	if got := buf.Len(); got != 6 {
		t.Fatalf("header length = %d, want 6", got)
	}
	want := []byte{0x04, 0x00, 0x12, 0x34, 0x56, 0x78}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Fatalf("header bytes = % x, want % x", buf.Bytes(), want)
	}
	pt, length, err := readHeader(&buf)
	if err != nil {
		t.Fatalf("readHeader: %v", err)
	}
	if pt != PDUTypeData || length != 0x12345678 {
		t.Errorf("readHeader = (%s, %#x), want (P-DATA-TF, 0x12345678)", pt, length)
	}
}

func TestReadHeaderRejectsUnknownType(t *testing.T) {
	r := bytes.NewReader([]byte{0x09, 0x00, 0x00, 0x00, 0x00, 0x00})
	if _, _, err := readHeader(r); err == nil {
		t.Error("readHeader should reject an unknown PDU type 0x09")
	}
}

func TestReadHeaderRejectsTruncated(t *testing.T) {
	// A header that ends mid-length must be io.ErrUnexpectedEOF, never a clean read.
	r := bytes.NewReader([]byte{0x04, 0x00, 0x12})
	if _, _, err := readHeader(r); err == nil {
		t.Error("readHeader should reject a truncated header")
	}
}
