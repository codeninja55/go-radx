//go:build interop

package hl7v2

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestInteropMLLPPeer drives a go-radx Client against an EXTERNAL MLLP peer (a
// python-hl7 / hl7apy listener) and asserts the peer returns a parseable
// acknowledgement. It is the cross-implementation framing check: it proves a
// go-radx frame is read, and a foreign ACK frame is parsed, by an independent
// stack.
//
// There is no python-hl7 / hl7apy MLLP peer container in this harness, so the
// test SKIPs unless RADX_HL7_MLLP_PEER names a reachable "host:port" of such a
// listener. This is a deliberate skip, not a silent pass: the go-radx <->
// go-radx round-trip in TestClientServerRoundTrip remains the hard correctness
// gate (it needs no external peer), and this test only adds confidence against a
// foreign implementation when one is supplied. The skip avoids inventing a
// fragile container the CI cannot reproduce.
func TestInteropMLLPPeer(t *testing.T) {
	addr := os.Getenv("RADX_HL7_MLLP_PEER")
	if addr == "" {
		t.Skip("RADX_HL7_MLLP_PEER not set: no external python-hl7/hl7apy MLLP peer " +
			"is provisioned in this harness; the go-radx<->go-radx round-trip " +
			"(TestClientServerRoundTrip) is the hard interop gate")
	}

	client, err := NewClient(addr, WithClientReadTimeout(10*time.Second))
	if err != nil {
		t.Fatalf("NewClient(%q): %v", addr, err)
	}
	defer client.Close()

	ack, err := client.Send(context.Background(), sampleMessage(t))
	if err != nil {
		t.Fatalf("Send to external peer: %v", err)
	}
	view, ok := AsACK(ack)
	if !ok {
		t.Fatal("external peer reply is not a typed ACK")
	}
	if _, ok := view.MSA(); !ok {
		t.Fatal("external peer ACK has no MSA segment")
	}
}
