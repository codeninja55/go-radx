package pdu

import (
	"bytes"
	"testing"
)

func TestReleaseRQRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	rq := &ReleaseRQ{}
	if err := rq.Encode(&buf); err != nil {
		t.Fatalf("ReleaseRQ.Encode: %v", err)
	}
	// 6-byte header + 4-byte reserved body.
	if buf.Len() != 10 {
		t.Fatalf("A-RELEASE-RQ length = %d, want 10", buf.Len())
	}
	if buf.Bytes()[0] != byte(PDUTypeReleaseRQ) {
		t.Fatalf("first byte = %#02x, want A-RELEASE-RQ", buf.Bytes()[0])
	}
	if _, err := DecodeReleaseRQ(&buf); err != nil {
		t.Fatalf("DecodeReleaseRQ: %v", err)
	}
}

func TestReleaseRPRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	rp := &ReleaseRP{}
	if err := rp.Encode(&buf); err != nil {
		t.Fatalf("ReleaseRP.Encode: %v", err)
	}
	if buf.Len() != 10 {
		t.Fatalf("A-RELEASE-RP length = %d, want 10", buf.Len())
	}
	if buf.Bytes()[0] != byte(PDUTypeReleaseRP) {
		t.Fatalf("first byte = %#02x, want A-RELEASE-RP", buf.Bytes()[0])
	}
	if _, err := DecodeReleaseRP(&buf); err != nil {
		t.Fatalf("DecodeReleaseRP: %v", err)
	}
}

func TestDecodeReleaseRejectsWrongType(t *testing.T) {
	var buf bytes.Buffer
	(&ReleaseRP{}).Encode(&buf)
	if _, err := DecodeReleaseRQ(&buf); err == nil {
		t.Error("DecodeReleaseRQ should reject an A-RELEASE-RP PDU")
	}
}
