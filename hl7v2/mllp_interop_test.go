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
// The peer is normally the pinned python-hl7 container TestMain (in
// mllp_peer_main_test.go) provisions for the interop gate, which exports
// RADX_HL7_MLLP_PEER before the run; setting the variable yourself substitutes
// an external listener for the container. The go-radx <-> go-radx round-trip in
// TestClientServerRoundTrip remains a standing correctness gate that needs no
// peer at all.
func TestInteropMLLPPeer(t *testing.T) {
	addr := os.Getenv("RADX_HL7_MLLP_PEER")
	if addr == "" {
		// Unreachable under the normal gate: TestMain either exports the variable or
		// fails the run before any test executes.
		t.Skip("RADX_HL7_MLLP_PEER not set and no peer container was provisioned")
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
