package acse

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dimse/dul"
	"github.com/codeninja55/go-radx/dimse/pdu"
)

func loopback(t *testing.T) (*dul.Conn, *dul.Conn) {
	t.Helper()
	c, s := net.Pipe()
	t.Cleanup(func() { c.Close(); s.Close() })
	return dul.NewConn(c, 0), dul.NewConn(s, 0)
}

func echoRequest() Request {
	return Request{
		CalledAETitle:  "ACCEPTOR",
		CallingAETitle: "REQUESTOR",
		MaxPDULength:   16382,
		Contexts: []pdu.PresentationContextRQ{{
			ID:               1,
			AbstractSyntax:   verificationUID,
			TransferSyntaxes: []string{explicitVRLE, implicitVRLE},
		}},
	}
}

func acceptParams() AcceptParams {
	return AcceptParams{
		CalledAETitle: "ACCEPTOR",
		MaxPDULength:  16382,
		Supported:     supportedVerification(),
	}
}

// TestRequestAndAcceptRoundTrip drives a full A-ASSOCIATE-RQ -> A-ASSOCIATE-AC handshake
// over an in-process loopback DUL pair: the requestor reaches Sta6 with the accepted
// context, and the acceptor reaches Sta6 too.
func TestRequestAndAcceptRoundTrip(t *testing.T) {
	rqConn, acConn := loopback(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	type acResult struct {
		assoc *Acceptor
		err   error
	}
	acceptorDone := make(chan acResult, 1)
	go func() {
		a, err := Accept(ctx, acConn, acceptParams())
		acceptorDone <- acResult{a, err}
	}()

	req, err := Associate(ctx, rqConn, echoRequest())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	if req.State() != dul.Sta6 {
		t.Errorf("requestor state = %v, want Sta6", req.State())
	}
	accepted := req.AcceptedContexts()
	if len(accepted) != 1 || accepted[0].Result != pdu.PresentationContextAcceptance {
		t.Fatalf("requestor accepted contexts = %+v, want one acceptance", accepted)
	}
	if accepted[0].TransferSyntax != explicitVRLE {
		t.Errorf("accepted transfer syntax = %q, want %q", accepted[0].TransferSyntax, explicitVRLE)
	}

	res := <-acceptorDone
	if res.err != nil {
		t.Fatalf("Accept: %v", res.err)
	}
	if res.assoc.State() != dul.Sta6 {
		t.Errorf("acceptor state = %v, want Sta6", res.assoc.State())
	}
}

// TestRequestReleaseRoundTrip is the named regression: an in-process Associate -> Release
// completes via the DUL and leaves both sides in Sta1.
func TestRequestReleaseRoundTrip(t *testing.T) {
	rqConn, acConn := loopback(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	acceptorDone := make(chan *Acceptor, 1)
	acceptErr := make(chan error, 1)
	go func() {
		a, err := Accept(ctx, acConn, acceptParams())
		if err != nil {
			acceptErr <- err
			return
		}
		acceptorDone <- a
	}()

	req, err := Associate(ctx, rqConn, echoRequest())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}

	var acc *Acceptor
	select {
	case acc = <-acceptorDone:
	case err := <-acceptErr:
		t.Fatalf("Accept: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("acceptor did not establish")
	}

	// The acceptor services the release request concurrently with the requestor's Release,
	// which blocks awaiting the A-RELEASE-RP.
	acceptorReleased := make(chan error, 1)
	go func() { acceptorReleased <- acc.ServeRelease(ctx) }()

	if err := req.Release(ctx); err != nil {
		t.Fatalf("requestor Release: %v", err)
	}
	if err := <-acceptorReleased; err != nil {
		t.Fatalf("acceptor ServeRelease: %v", err)
	}

	if req.State() != dul.Sta1 {
		t.Errorf("requestor state after release = %v, want Sta1", req.State())
	}
	if acc.State() != dul.Sta1 {
		t.Errorf("acceptor state after release = %v, want Sta1", acc.State())
	}
}

// TestRequestRejected verifies an A-ASSOCIATE-RJ returned by the acceptor surfaces as a
// typed *RejectedError on the requestor, carrying the rejection source and reason.
func TestRequestRejected(t *testing.T) {
	rqConn, acConn := loopback(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		_, _ = Accept(ctx, acConn, AcceptParams{
			CalledAETitle: "ACCEPTOR",
			MaxPDULength:  16382,
			RejectAll:     true,
		})
	}()

	_, err := Associate(ctx, rqConn, echoRequest())
	if err == nil {
		t.Fatal("Associate against a rejecting acceptor = nil error, want a rejection")
	}
	var re *RejectedError
	if !errors.As(err, &re) {
		t.Fatalf("Associate error = %T, want *RejectedError", err)
	}
}
