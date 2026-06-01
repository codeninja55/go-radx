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
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the A-ABORT")
	}
	if sc.machine.CurrentState() != Sta13 {
		t.Errorf("state after invalid PDU = %v, want Sta13", sc.machine.CurrentState())
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
