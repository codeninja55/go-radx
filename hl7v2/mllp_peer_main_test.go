//go:build interop

package hl7v2

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/hl7v2/integration/hl7peer"
)

// mllpPeer is the python-hl7 peer container TestMain provisions for the interop gate. It stays
// nil when RADX_HL7_MLLP_PEER already names an external peer — in that case the forward test
// runs against the supplied listener and the reverse-direction test (which must exec mllp_send
// inside the container) skips.
var mllpPeer *hl7peer.Container

// mllpReverseServerPort is the port of the go-radx Server TestMain binds for the reverse
// direction, declared as the peer container's host-access port.
var mllpReverseServerPort int

// mllpReverseInbox receives every message the reverse-direction go-radx Server handles, so the
// reverse test can assert the foreign frame actually reached the handler.
var mllpReverseInbox = make(chan *Message, 4)

// TestMain provisions the python-hl7 MLLP peer container for the interop gate. When
// RADX_HL7_MLLP_PEER already names an external peer the container is not started, preserving the
// bring-your-own-peer escape hatch. A container start failure fails the run (exit 1) rather than
// skipping: under the interop tag this is a gate, and a silent skip would manufacture green.
//
// Ordering matters: the go-radx Server for the reverse direction binds BEFORE the peer container
// is created, because testcontainers' HostAccessPorts contract requires the host service to be
// live when the container (and its sshd tunnel sidecar) starts.
func TestMain(m *testing.M) {
	os.Exit(runInteropMain(m))
}

func runInteropMain(m *testing.M) int {
	if os.Getenv("RADX_HL7_MLLP_PEER") != "" {
		return m.Run()
	}

	srv := NewServer(HandlerFunc(func(_ context.Context, msg *Message) (*Message, error) {
		select {
		case mllpReverseInbox <- msg:
		default:
		}
		return msg.BuildACK(AckAccept)
	}))
	go func() {
		_ = srv.ListenAndServe(context.Background(), "127.0.0.1:0")
	}()
	defer func() {
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			fmt.Fprintf(os.Stderr, "interop: shut down reverse-direction MLLP server: %v\n", err)
		}
	}()
	port, err := waitForServerPort(srv, 5*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "interop: bind reverse-direction MLLP server: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	peer, err := hl7peer.Start(ctx, port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "interop: start python-hl7 MLLP peer container: %v\n", err)
		return 1
	}
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Minute)
		defer stopCancel()
		if err := peer.Stop(stopCtx); err != nil {
			fmt.Fprintf(os.Stderr, "interop: stop python-hl7 MLLP peer container: %v\n", err)
		}
	}()

	mllpPeer = peer
	mllpReverseServerPort = port
	// TestMain has no *testing.T, so t.Setenv is unavailable; the variable feeds the existing
	// env-gated TestInteropMLLPPeer in this same process only.
	if err := os.Setenv("RADX_HL7_MLLP_PEER", peer.Addr()); err != nil {
		fmt.Fprintf(os.Stderr, "interop: set RADX_HL7_MLLP_PEER: %v\n", err)
		return 1
	}
	return m.Run()
}

// waitForServerPort blocks until srv has bound its listener and returns the assigned port. It is
// the TestMain-safe analogue of waitForAddr, which needs a *testing.T.
func waitForServerPort(srv *Server, timeout time.Duration) (int, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if addr, ok := srv.Addr().(*net.TCPAddr); ok {
			return addr.Port, nil
		}
		time.Sleep(time.Millisecond)
	}
	return 0, fmt.Errorf("server did not bind within %v", timeout)
}

// TestInteropMLLPPeerSender drives python-hl7's mllp_send — the foreign SENDER — against the
// go-radx MLLP Server TestMain bound before the peer container started, the reverse of
// TestInteropMLLPPeer's direction. It proves a foreign frame is read by the go-radx server, and
// the go-radx acknowledgement frame is read back by the foreign client (mllp_send blocks on the
// reply and prints it). Together the two directions cover all four cross-implementation framing
// pairings.
func TestInteropMLLPPeerSender(t *testing.T) {
	if mllpPeer == nil {
		t.Skip("external peer supplied via RADX_HL7_MLLP_PEER: the reverse direction needs " +
			"the container peer (mllp_send runs inside it)")
	}

	raw, err := sampleMessage(t).MarshalText()
	if err != nil {
		t.Fatalf("MarshalText sample: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := mllpPeer.SendToHost(ctx, mllpReverseServerPort, raw)
	if err != nil {
		t.Fatalf("mllp_send from peer container to go-radx server: %v", err)
	}

	select {
	case m := <-mllpReverseInbox:
		if _, ok := m.MSH(); !ok {
			t.Error("server-received message has no MSH segment")
		}
	default:
		t.Error("go-radx server handler never received the peer-sent message")
	}
	// mllp_send prints the raw reply frame; the go-radx acceptance ACK carries MSA|AA.
	if !strings.Contains(out, "MSA|AA") {
		t.Errorf("mllp_send output does not show the go-radx ACK (want MSA|AA): %q", out)
	}
}
