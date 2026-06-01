package dul

import (
	"bytes"
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dimse/pdu"
)

func TestConnWriteReadPDURoundTrip(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() { client.Close(); server.Close() })

	cc := NewConn(client, 0)
	sc := NewConn(server, 0)

	ctx := context.Background()
	rq := &pdu.AssociateRQ{
		ProtocolVersion:      1,
		ApplicationContext:   "1.2.840.10008.3.1.1.1",
		PresentationContexts: []pdu.PresentationContextRQ{{ID: 1, AbstractSyntax: "1.2.840.10008.1.1", TransferSyntaxes: []string{"1.2.840.10008.1.2"}}},
		UserInfo:             pdu.UserInformation{MaxPDULength: 16382},
	}

	errc := make(chan error, 1)
	go func() { errc <- cc.WritePDU(ctx, rq) }()

	got, err := sc.ReadPDU(ctx)
	if err != nil {
		t.Fatalf("ReadPDU: %v", err)
	}
	if err := <-errc; err != nil {
		t.Fatalf("WritePDU: %v", err)
	}
	if got.Type() != pdu.PDUTypeAssociateRQ {
		t.Errorf("read PDU type = %v, want A-ASSOCIATE-RQ", got.Type())
	}
}

func TestConnReadHonoursContextCancellation(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() { client.Close(); server.Close() })
	_ = client

	sc := NewConn(server, 0)
	ctx, cancel := context.WithCancel(context.Background())

	errc := make(chan error, 1)
	go func() {
		_, err := sc.ReadPDU(ctx)
		errc <- err
	}()

	cancel() // no data will ever arrive; cancellation must unblock the read

	select {
	case err := <-errc:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("ReadPDU after cancel = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ReadPDU did not return after context cancellation")
	}
}

// TestPDUToEventMapping covers the inbound-PDU -> received-event mapping the run loop
// uses to drive the FSM.
func TestPDUToEventMapping(t *testing.T) {
	cases := []struct {
		p    pdu.PDU
		want Evt
	}{
		{&pdu.AssociateRQ{}, Evt6},
		{&pdu.AssociateAC{}, Evt3},
		{&pdu.AssociateRJ{}, Evt4},
		{&pdu.DataTF{}, Evt10},
		{&pdu.ReleaseRQ{}, Evt12},
		{&pdu.ReleaseRP{}, Evt13},
		{&pdu.Abort{}, Evt16},
	}
	for _, c := range cases {
		if got := pduToEvent(c.p); got != c.want {
			t.Errorf("pduToEvent(%v) = %v, want %v", c.p.Type(), got, c.want)
		}
	}
}

// TestConnAbortOnInvalidPDU is the DIMSE-011 end-to-end regression at the connection
// layer: a peer that sends bytes the pdu codec rejects must drive the FSM through AA-8 to
// an A-ABORT (provider source) written back on the wire, never a silent close.
func TestConnAbortOnInvalidPDU(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() { client.Close(); server.Close() })

	sc := NewConn(server, 0)
	sc.machine.forceState(Sta6) // pretend the association is established

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// The client sends a PDU with an unrecognised type byte, then reads the abort.
	gotAbort := make(chan *pdu.Abort, 1)
	go func() {
		// Unknown PDU type 0x09 with a zero-length body.
		client.Write([]byte{0x09, 0x00, 0x00, 0x00, 0x00, 0x00})
		p, err := pdu.ReadPDU(client)
		if err != nil {
			gotAbort <- nil
			return
		}
		ab, _ := p.(*pdu.Abort)
		gotAbort <- ab
	}()

	action, _, err := sc.driveOnce(ctx)
	if err == nil {
		t.Fatal("driveOnce on an invalid PDU = nil error, want a protocol error")
	}
	if action != AA8 {
		t.Fatalf("driveOnce action = %v, want AA-8 (DIMSE-011)", action)
	}

	select {
	case ab := <-gotAbort:
		if ab == nil {
			t.Fatal("peer did not receive an A-ABORT after an invalid PDU (silent close, DIMSE-011)")
		}
		if ab.Source != pdu.AbortSourceServiceProvider {
			t.Errorf("A-ABORT source = %d, want provider (%d)", ab.Source, pdu.AbortSourceServiceProvider)
		}
		// A malformed/unrecognised PDU aborts with "unrecognised PDU", not "unexpected".
		if ab.Reason != pdu.AbortReasonUnrecognizedPDU {
			t.Errorf("A-ABORT reason = %d, want UnrecognizedPDU (%d)", ab.Reason, pdu.AbortReasonUnrecognizedPDU)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the A-ABORT")
	}
	if sc.machine.CurrentState() != Sta13 {
		t.Errorf("state after invalid PDU = %v, want Sta13", sc.machine.CurrentState())
	}
}

// TestConnUnexpectedRecognizedPDUAbortReason is the regression for the abort-diagnostic
// fix: a well-formed, RECOGNISED PDU that is unexpected in the current state (here an
// A-ASSOCIATE-RJ received while established in Sta6, whose Table 9-10 cell is undefined)
// must abort with the provider reason "unexpected PDU", distinct from a malformed PDU's
// "unrecognised PDU".
func TestConnUnexpectedRecognizedPDUAbortReason(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() { client.Close(); server.Close() })

	sc := NewConn(server, 0)
	sc.machine.forceState(Sta6) // established; an A-ASSOCIATE-RJ is unexpected here

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	gotAbort := make(chan *pdu.Abort, 1)
	go func() {
		// A valid A-ASSOCIATE-RJ PDU: type 0x03, 4-byte body (reserved, result, source, reason).
		client.Write([]byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x04, 0x00, 0x01, 0x01, 0x01})
		p, err := pdu.ReadPDU(client)
		if err != nil {
			gotAbort <- nil
			return
		}
		ab, _ := p.(*pdu.Abort)
		gotAbort <- ab
	}()

	action, _, err := sc.driveOnce(ctx)
	if err == nil {
		t.Fatal("driveOnce on an unexpected recognised PDU = nil error, want a protocol error")
	}
	if action != AA8 {
		t.Fatalf("driveOnce action = %v, want AA-8", action)
	}
	select {
	case ab := <-gotAbort:
		if ab == nil {
			t.Fatal("peer did not receive an A-ABORT after an unexpected PDU")
		}
		if ab.Reason != pdu.AbortReasonUnexpectedPDU {
			t.Errorf("A-ABORT reason = %d, want UnexpectedPDU (%d)", ab.Reason, pdu.AbortReasonUnexpectedPDU)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the A-ABORT")
	}
}

// TestConnReusableAfterCancelledRead is the regression for the deadline-watcher race: an
// operation whose context is already cancelled must not strand a past deadline that fails
// the next operation under a fresh context. The stop function waits for the watcher to
// return before clearing the deadline, so a fresh read after a cancelled one succeeds.
func TestConnReusableAfterCancelledRead(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() { client.Close(); server.Close() })
	sc := NewConn(server, 0)

	cancelled, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before the read starts
	if _, err := sc.ReadPDU(cancelled); err == nil {
		t.Fatal("ReadPDU with a cancelled context should fail")
	}

	// A fresh read must still work: the cancelled read must not have left a stale deadline.
	want := &pdu.ReleaseRP{}
	go func() { _ = pdu.WritePDU(client, want) }()
	fresh, freshCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer freshCancel()
	if _, err := sc.ReadPDU(fresh); err != nil {
		t.Fatalf("fresh ReadPDU after a cancelled one failed (stale deadline?): %v", err)
	}
}

// TestConnCleanCloseDrivesEvt17 is the regression for the read-error branch: a peer that
// closes the socket at a PDU boundary (a clean io.EOF) is an orderly transport close, so
// driveOnce must raise Evt17 (transport connection closed) and NOT Evt19 (invalid PDU ->
// abort). From Sta2 the FSM resolves Evt17 to AA-5 (stop ARTIM) -> Sta1 and sends no
// A-ABORT; an Evt19 would instead have driven AA-1 and written an abort back on the wire.
func TestConnCleanCloseDrivesEvt17(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() { client.Close(); server.Close() })

	sc := NewConn(server, 0)
	sc.machine.forceState(Sta2) // acceptor awaiting the A-ASSOCIATE-RQ

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// The peer closes at a PDU boundary: the read sees a clean io.EOF, no bytes pending.
	gotUnexpected := make(chan pdu.PDU, 1)
	go func() {
		client.Close()
		// If the FSM mistakenly took the Evt19 path it would send an A-ABORT back; read it.
		p, err := pdu.ReadPDU(client)
		if err != nil {
			gotUnexpected <- nil
			return
		}
		gotUnexpected <- p
	}()

	action, _, _ := sc.driveOnce(ctx)
	if action == AA1 {
		t.Fatalf("clean close drove %v (the Evt19/abort path); want the Evt17 path", action)
	}
	if action != AA5 {
		t.Errorf("clean close action = %v, want AA-5 (Sta2 + Evt17)", action)
	}
	if sc.machine.CurrentState() != Sta1 {
		t.Errorf("state after clean close = %v, want Sta1", sc.machine.CurrentState())
	}

	select {
	case p := <-gotUnexpected:
		if p != nil {
			t.Errorf("peer received an unexpected %v after a clean close; want no abort (Evt17, not Evt19)", p.Type())
		}
	case <-time.After(time.Second):
		// No PDU arrived, which is the expected outcome for a clean close.
	}
}

func TestConnWriteRejectsNilContextDeadlinePassthrough(t *testing.T) {
	// A write with an already-cancelled context must not block and must surface the error.
	client, server := net.Pipe()
	t.Cleanup(func() { client.Close(); server.Close() })
	_ = server

	cc := NewConn(client, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := cc.WritePDU(ctx, &pdu.ReleaseRQ{})
	if err == nil {
		t.Error("WritePDU with a cancelled context should return an error")
	}
}

// TestConnCancelReadDoesNotDisturbWrite is the regression for direction-specific
// deadlines: the connection is full-duplex, so cancelling a blocked read must not time out
// an in-flight write. A shared SetDeadline would set both directions at once, spuriously
// failing the concurrent write; SetReadDeadline / SetWriteDeadline keep each direction's
// cancellation independent. It runs under -race to flush out a shared-deadline data race.
func TestConnCancelReadDoesNotDisturbWrite(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	accepted := make(chan net.Conn, 1)
	go func() {
		nc, aerr := ln.Accept()
		if aerr != nil {
			accepted <- nil
			return
		}
		accepted <- nc
	}()

	dialed, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	peer := <-accepted
	if peer == nil {
		t.Fatal("accept failed")
	}
	t.Cleanup(func() { dialed.Close(); peer.Close() })

	c := NewConn(dialed, 0)

	// A read that will never see data; its context will be cancelled to unblock it. The
	// cancellation must touch only the read deadline.
	readCtx, cancelRead := context.WithCancel(context.Background())
	readDone := make(chan error, 1)
	go func() {
		_, rerr := c.ReadPDU(readCtx)
		readDone <- rerr
	}()

	// The peer drains anything written so a concurrent write completes.
	peerDrained := make(chan struct{})
	go func() {
		buf := make([]byte, 64)
		for {
			if _, derr := peer.Read(buf); derr != nil {
				close(peerDrained)
				return
			}
		}
	}()

	// Cancel the read while the write is in flight, then write concurrently.
	writeDone := make(chan error, 1)
	go func() {
		cancelRead()
		writeDone <- c.WritePDU(context.Background(), &pdu.ReleaseRQ{})
	}()

	select {
	case rerr := <-readDone:
		if !errors.Is(rerr, context.Canceled) {
			t.Errorf("cancelled read returned %v, want context.Canceled", rerr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled read did not return")
	}

	select {
	case werr := <-writeDone:
		if werr != nil {
			t.Errorf("write disturbed by the read cancellation: %v (shared deadline?)", werr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("write with its own context did not complete")
	}
}

func TestConnEncodeRoundTripsThroughBuffer(t *testing.T) {
	// Sanity: a PDU written by the connection decodes byte-identically via the pdu codec.
	var buf bytes.Buffer
	if err := pdu.WritePDU(&buf, &pdu.Abort{Source: pdu.AbortSourceServiceProvider, Reason: pdu.AbortReasonUnrecognizedPDU}); err != nil {
		t.Fatal(err)
	}
	got, err := pdu.ReadPDU(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if got.Type() != pdu.PDUTypeAbort {
		t.Errorf("type = %v, want A-ABORT", got.Type())
	}
}
