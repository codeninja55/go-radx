package pdu

import (
	"bytes"
	"testing"
)

func TestAbortRoundTrip(t *testing.T) {
	cases := []Abort{
		{Source: AbortSourceServiceUser, Reason: AbortReasonNotSpecified},
		{Source: AbortSourceServiceProvider, Reason: AbortReasonInvalidPDUParameter},
		{Source: AbortSourceServiceProvider, Reason: AbortReasonUnrecognizedPDU},
	}
	for _, a := range cases {
		var buf bytes.Buffer
		if err := a.Encode(&buf); err != nil {
			t.Fatalf("Abort.Encode: %v", err)
		}
		// 6-byte header + 4-byte body (reserved, reserved, source, reason).
		if buf.Len() != 10 {
			t.Fatalf("A-ABORT length = %d, want 10", buf.Len())
		}
		if buf.Bytes()[0] != byte(PDUTypeAbort) {
			t.Fatalf("first byte = %#02x, want A-ABORT", buf.Bytes()[0])
		}
		got, err := DecodeAbort(&buf)
		if err != nil {
			t.Fatalf("DecodeAbort: %v", err)
		}
		if got.Source != a.Source || got.Reason != a.Reason {
			t.Errorf("Abort round-trip = %+v, want %+v", got, a)
		}
	}
}

// TestProviderAbortConstants documents the provider-source/reason values the AA-8 path
// uses for an invalid PDU (Codex DIMSE-011), so the DUL can populate them by name.
func TestProviderAbortConstants(t *testing.T) {
	if AbortSourceServiceProvider != 2 {
		t.Errorf("AbortSourceServiceProvider = %d, want 2", AbortSourceServiceProvider)
	}
	if AbortReasonUnrecognizedPDU != 1 {
		t.Errorf("AbortReasonUnrecognizedPDU = %d, want 1", AbortReasonUnrecognizedPDU)
	}
}
