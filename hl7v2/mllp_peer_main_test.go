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

// mllpPeerHostPort is the host TCP port reserved before the container started, declared as its
// host-access port so mllp_send inside the container can reach a go-radx server bound to it.
var mllpPeerHostPort int

// TestMain provisions the python-hl7 MLLP peer container for the interop gate. When
// RADX_HL7_MLLP_PEER already names an external peer the container is not started, preserving the
// bring-your-own-peer escape hatch. A container start failure fails the run (exit 1) rather than
// skipping: under the interop tag this is a gate, and a silent skip would manufacture green.
func TestMain(m *testing.M) {
	os.Exit(runInteropMain(m))
}

func runInteropMain(m *testing.M) int {
	if os.Getenv("RADX_HL7_MLLP_PEER") != "" {
		return m.Run()
	}

	hostPort, err := reserveLoopbackPort()
	if err != nil {
		fmt.Fprintf(os.Stderr, "interop: reserve host port for the MLLP reverse direction: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	peer, err := hl7peer.Start(ctx, hostPort)
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
	mllpPeerHostPort = hostPort
	// TestMain has no *testing.T, so t.Setenv is unavailable; the variable feeds the existing
	// env-gated TestInteropMLLPPeer in this same process only.
	if err := os.Setenv("RADX_HL7_MLLP_PEER", peer.Addr()); err != nil {
		fmt.Fprintf(os.Stderr, "interop: set RADX_HL7_MLLP_PEER: %v\n", err)
		return 1
	}
	return m.Run()
}

// reserveLoopbackPort binds 127.0.0.1:0, records the assigned port, and releases it. The
// container's host-access tunnel must name the port before the container starts, but the go-radx
// server under test binds it only inside the reverse-direction test; the close-then-rebind window
// is benign here because nothing else on the runner races for ephemeral loopback ports.
func reserveLoopbackPort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		return 0, err
	}
	return port, nil
}

// TestInteropMLLPPeerSender drives python-hl7's mllp_send — the foreign SENDER — against a
// go-radx MLLP Server, the reverse of TestInteropMLLPPeer's direction. It proves a foreign frame
// is read by the go-radx server, and the go-radx acknowledgement frame is read back by the
// foreign client (mllp_send blocks on the reply and prints it). Together the two directions
// cover all four cross-implementation framing pairings.
func TestInteropMLLPPeerSender(t *testing.T) {
	if mllpPeer == nil {
		t.Skip("external peer supplied via RADX_HL7_MLLP_PEER: the reverse direction needs " +
			"the container peer (mllp_send runs inside it)")
	}

	received := make(chan *Message, 1)
	srv := NewServer(HandlerFunc(func(_ context.Context, m *Message) (*Message, error) {
		select {
		case received <- m:
		default:
		}
		return m.BuildACK(AckAccept)
	}))
	go func() {
		_ = srv.ListenAndServe(context.Background(), fmt.Sprintf("127.0.0.1:%d", mllpPeerHostPort))
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown: %v", err)
		}
	}()
	waitForAddr(t, srv)

	raw, err := sampleMessage(t).MarshalText()
	if err != nil {
		t.Fatalf("MarshalText sample: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := mllpPeer.SendToHost(ctx, mllpPeerHostPort, raw)
	if err != nil {
		t.Fatalf("mllp_send from peer container to go-radx server: %v", err)
	}

	select {
	case m := <-received:
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
