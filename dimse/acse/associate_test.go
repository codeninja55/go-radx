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

// tcpLoopback returns a connected pair over the loopback interface. Unlike net.Pipe, the
// kernel socket buffers let a side write a PDU the peer has not yet read, which a single
// goroutine driving both a write and a follow-up read needs to avoid deadlock.
func tcpLoopback(t *testing.T) (*dul.Conn, *dul.Conn) {
	t.Helper()
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
	return dul.NewConn(dialed, 0), dul.NewConn(peer, 0)
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

// TestServeReleaseSendsAbortOnProtocolViolation proves the consolidation closed the
// correctness gap: the live acse path now SENDS the provider-source A-ABORT that PS3.8 AA-8
// requires when a peer commits a protocol violation, routing inbound reads through the shared
// dul.DriveInbound. Here an established association is asked to release, but the peer sends an
// unexpected (recognised) A-ASSOCIATE-RQ instead of an A-RELEASE-RQ; in Sta6 that drives AA-8,
// and the acceptor must write an A-ABORT back rather than closing silently (the pre-refactor
// behaviour).
func TestServeReleaseSendsAbortOnProtocolViolation(t *testing.T) {
	// A real TCP loopback (not net.Pipe) so the kernel socket buffers a write the peer has
	// not yet read; net.Pipe is fully synchronous and would deadlock the abort send against
	// the violating write on a single goroutine.
	rqConn, acConn := tcpLoopback(t)
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

	// The acceptor services a release; the requestor instead sends a recognised-but-unexpected
	// PDU, a protocol violation that must drive AA-8 and a provider A-ABORT back on the wire.
	serveDone := make(chan error, 1)
	go func() { serveDone <- acc.ServeRelease(ctx) }()

	violation := &pdu.AssociateRQ{
		ProtocolVersion:      1,
		ApplicationContext:   "1.2.840.10008.3.1.1.1",
		PresentationContexts: []pdu.PresentationContextRQ{{ID: 1, AbstractSyntax: verificationUID, TransferSyntaxes: []string{implicitVRLE}}},
		UserInfo:             pdu.UserInformation{MaxPDULength: 16382},
	}
	if werr := req.Conn().WritePDU(ctx, violation); werr != nil {
		t.Fatalf("writing the violating PDU: %v", werr)
	}

	// The pre-refactor acse closed silently here; the consolidated path sends the A-ABORT.
	resp, rerr := req.Conn().ReadPDU(ctx)
	if rerr != nil {
		t.Fatalf("reading the A-ABORT the acceptor must send on a violation: %v", rerr)
	}
	ab, ok := resp.(*pdu.Abort)
	if !ok {
		t.Fatalf("peer received %s, want an A-ABORT after a protocol violation", resp.Type())
	}
	if ab.Source != pdu.AbortSourceServiceProvider {
		t.Errorf("A-ABORT source = %d, want provider (%d)", ab.Source, pdu.AbortSourceServiceProvider)
	}
	if ab.Reason != pdu.AbortReasonUnexpectedPDU {
		t.Errorf("A-ABORT reason = %d, want UnexpectedPDU (%d)", ab.Reason, pdu.AbortReasonUnexpectedPDU)
	}

	// ServeRelease surfaces the violation as a typed *ProtocolError, not a silent success.
	select {
	case serr := <-serveDone:
		if serr == nil {
			t.Fatal("ServeRelease on a protocol violation = nil error, want a *ProtocolError")
		}
		var pe *ProtocolError
		if !errors.As(serr, &pe) {
			t.Fatalf("ServeRelease error = %T, want *ProtocolError", serr)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("ServeRelease did not return after the violation")
	}
}

// TestAcceptExposesAETitles confirms the established acceptor reports the Calling and Called AE
// titles from the inbound A-ASSOCIATE-RQ (trimmed of the fixed-field padding), so the SCP can
// build the no-PHI OpInfo for each operation.
func TestAcceptExposesAETitles(t *testing.T) {
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

	if _, err := Associate(ctx, rqConn, echoRequest()); err != nil {
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

	if acc.CallingAETitle() != "REQUESTOR" {
		t.Errorf("CallingAETitle() = %q, want REQUESTOR", acc.CallingAETitle())
	}
	if acc.CalledAETitle() != "ACCEPTOR" {
		t.Errorf("CalledAETitle() = %q, want ACCEPTOR", acc.CalledAETitle())
	}
}

// TestAcceptRejectsWrongCalledAETitle is the named SCP regression: when the acceptor requires a
// specific Called AE Title, an A-ASSOCIATE-RQ naming a different one is rejected at negotiation
// with an A-ASSOCIATE-RJ (service-user source, called-AE-title-not-recognized), not accepted.
func TestAcceptRejectsWrongCalledAETitle(t *testing.T) {
	rqConn, acConn := loopback(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	acceptErr := make(chan error, 1)
	go func() {
		params := acceptParams()
		params.RequireCalledAETitle = "OTHER-SCP" // the RQ names "ACCEPTOR", a mismatch
		_, err := Accept(ctx, acConn, params)
		acceptErr <- err
	}()

	_, err := Associate(ctx, rqConn, echoRequest())
	if err == nil {
		t.Fatal("Associate with a wrong Called AE Title = nil error, want a rejection")
	}
	var re *RejectedError
	if !errors.As(err, &re) {
		t.Fatalf("Associate error = %T, want *RejectedError", err)
	}
	if re.Source != pdu.AssociateRJSourceServiceUser {
		t.Errorf("rejection source = %d, want service-user (%d)", re.Source, pdu.AssociateRJSourceServiceUser)
	}
	if re.Reason != reasonCalledAETitleNotRecognized {
		t.Errorf("rejection reason = %d, want called-AE-title-not-recognized (%d)", re.Reason, reasonCalledAETitleNotRecognized)
	}

	if serr := <-acceptErr; serr == nil {
		t.Fatal("Accept on a wrong Called AE Title = nil error, want a *RejectedError")
	} else {
		var sre *RejectedError
		if !errors.As(serr, &sre) {
			t.Fatalf("Accept error = %T, want *RejectedError", serr)
		}
	}
}

// TestAcceptRejectsUnlistedCallingAETitle confirms that when the acceptor restricts the Calling
// AE Titles it serves, an RQ from a title outside the list is rejected at negotiation with the
// calling-AE-title-not-recognized reason; a listed title is accepted.
func TestAcceptRejectsUnlistedCallingAETitle(t *testing.T) {
	t.Run("unlisted is rejected", func(t *testing.T) {
		rqConn, acConn := loopback(t)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		go func() {
			params := acceptParams()
			params.RequireCallingAETitles = []string{"TRUSTED"} // the RQ is from "REQUESTOR"
			_, _ = Accept(ctx, acConn, params)
		}()

		_, err := Associate(ctx, rqConn, echoRequest())
		if err == nil {
			t.Fatal("Associate from an unlisted Calling AE Title = nil error, want a rejection")
		}
		var re *RejectedError
		if !errors.As(err, &re) {
			t.Fatalf("Associate error = %T, want *RejectedError", err)
		}
		if re.Reason != reasonCallingAETitleNotRecognized {
			t.Errorf("rejection reason = %d, want calling-AE-title-not-recognized (%d)", re.Reason, reasonCallingAETitleNotRecognized)
		}
	})

	t.Run("listed is accepted", func(t *testing.T) {
		rqConn, acConn := loopback(t)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		acceptorDone := make(chan *Acceptor, 1)
		acceptErr := make(chan error, 1)
		go func() {
			params := acceptParams()
			params.RequireCallingAETitles = []string{"REQUESTOR", "TRUSTED"}
			a, err := Accept(ctx, acConn, params)
			if err != nil {
				acceptErr <- err
				return
			}
			acceptorDone <- a
		}()

		if _, err := Associate(ctx, rqConn, echoRequest()); err != nil {
			t.Fatalf("Associate from a listed Calling AE Title: %v", err)
		}
		select {
		case <-acceptorDone:
		case err := <-acceptErr:
			t.Fatalf("Accept of a listed Calling AE Title = %v, want acceptance", err)
		case <-time.After(3 * time.Second):
			t.Fatal("acceptor did not establish")
		}
	})
}
