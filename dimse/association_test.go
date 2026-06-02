package dimse

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/acse"
	"github.com/codeninja55/go-radx/dimse/dul"
)

// startEchoAcceptor listens on loopback and, for each inbound connection, runs the acse
// acceptor negotiation for the Verification SOP Class and then services one graceful
// release. It returns the listener address; the listener closes on test cleanup.
func startEchoAcceptor(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			nc, aerr := ln.Accept()
			if aerr != nil {
				return
			}
			go func(c net.Conn) {
				conn := dul.NewConn(c, 0)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				acc, perr := acse.Accept(ctx, conn, acse.AcceptParams{
					CalledAETitle: "ACCEPTOR",
					MaxPDULength:  16382,
					Supported: []acse.SupportedContext{{
						AbstractSyntax:   "1.2.840.10008.1.1",
						TransferSyntaxes: []string{"1.2.840.10008.1.2.1", "1.2.840.10008.1.2"},
					}},
				})
				if perr != nil {
					c.Close()
					return
				}
				_ = acc.ServeRelease(ctx)
			}(nc)
		}
	}()
	return ln.Addr().String()
}

// TestAssociateReleaseRoundTrip drives AE.Associate over a loopback TCP acceptor, verifies
// the public Association reports an accepted Verification context and Sta6, then releases
// gracefully and returns to Sta1.
func TestAssociateReleaseRoundTrip(t *testing.T) {
	addr := startEchoAcceptor(t)

	ae, err := NewAE(AETitle("REQUESTOR"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	assoc, err := ae.Associate(ctx, addr, AETitle("ACCEPTOR"), VerificationContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	if assoc.State() != Sta6 {
		t.Errorf("state after Associate = %v, want Sta6", assoc.State())
	}

	accepted := assoc.AcceptedContexts()
	if len(accepted) != 1 {
		t.Fatalf("accepted contexts = %d, want 1", len(accepted))
	}
	if accepted[0].Result != ContextAccepted {
		t.Errorf("context result = %v, want accepted", accepted[0].Result)
	}
	if accepted[0].AbstractSyntax != dicom.SOPClassUID("1.2.840.10008.1.1") {
		t.Errorf("accepted abstract syntax = %q, want Verification", accepted[0].AbstractSyntax)
	}
	if len(accepted[0].TransferSyntaxes) != 1 || accepted[0].TransferSyntaxes[0] != dicom.ExplicitVRLittleEndian {
		t.Errorf("accepted transfer syntaxes = %v, want [Explicit VR LE]", accepted[0].TransferSyntaxes)
	}

	if err := assoc.Release(ctx); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if assoc.State() != Sta1 {
		t.Errorf("state after Release = %v, want Sta1", assoc.State())
	}
}

// TestOperationOnReleasedAssociationReturnsTypedError is the DIMSE-017 regression: an
// operation on an already-released association returns a typed error and never panics.
// Double-Release is safe.
func TestOperationOnReleasedAssociationReturnsTypedError(t *testing.T) {
	addr := startEchoAcceptor(t)
	ae, _ := NewAE(AETitle("REQUESTOR"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	assoc, err := ae.Associate(ctx, addr, AETitle("ACCEPTOR"), VerificationContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	if err := assoc.Release(ctx); err != nil {
		t.Fatalf("first Release: %v", err)
	}

	// A second Release must be safe (idempotent), never panic.
	if err := assoc.Release(ctx); err != nil {
		t.Errorf("double Release returned %v, want nil (idempotent)", err)
	}

	// Abort on a released association must return a typed *AssociationError, never panic.
	err = assoc.Abort(ctx)
	if err == nil {
		t.Fatal("Abort on a released association = nil error, want a typed error")
	}
	var ae2 *AssociationError
	if !errors.As(err, &ae2) {
		t.Errorf("Abort-on-released error = %T, want *AssociationError", err)
	}
}

// TestOperationOnUnestablishedAssociationReturnsTypedError is the other half of DIMSE-017:
// a zero-value/unestablished Association returns a typed error, never panics.
func TestOperationOnUnestablishedAssociationReturnsTypedError(t *testing.T) {
	var assoc Association // never established
	ctx := context.Background()

	if got := assoc.State(); got != Sta1 {
		t.Errorf("unestablished State() = %v, want Sta1", got)
	}
	if assoc.AcceptedContexts() != nil {
		t.Error("unestablished AcceptedContexts() should be nil")
	}

	err := assoc.Release(ctx)
	if err == nil {
		t.Fatal("Release on an unestablished association = nil error, want a typed error")
	}
	var assocErr *AssociationError
	if !errors.As(err, &assocErr) {
		t.Errorf("Release-on-unestablished error = %T, want *AssociationError", err)
	}
}

// TestAssociationMessageIDAllocator is the DIMSE-016 foundation: the per-association allocator
// hands out distinct, non-zero, monotonically increasing Message IDs for the sub-operation-bearing
// and chained services. A fixed MessageID: 0 miscounted failures and hung against compliant peers.
func TestAssociationMessageIDAllocator(t *testing.T) {
	a := &Association{}
	first := a.nextMessageID()
	second := a.nextMessageID()
	if first == 0 || second == 0 {
		t.Fatalf("message IDs must be non-zero, got %d and %d", first, second)
	}
	if second != first+1 {
		t.Errorf("expected monotonic IDs, got %d then %d", first, second)
	}
}

// TestAssociationMessageIDSkipsZeroOnWrap verifies the 16-bit counter wraps past 0 (which is the
// reserved "no operation" sentinel a sub-operation must never use): from 0xFFFF the next ID is 1,
// not 0.
func TestAssociationMessageIDSkipsZeroOnWrap(t *testing.T) {
	a := &Association{nextMsgID: 0xFFFF}
	got := a.nextMessageID()
	if got != 1 {
		t.Errorf("nextMessageID after 0xFFFF = %d, want 1 (skip the reserved 0)", got)
	}
}

func TestAssociateRejectsInvalidCalledTitle(t *testing.T) {
	ae, _ := NewAE(AETitle("REQUESTOR"))
	ctx := context.Background()
	_, err := ae.Associate(ctx, "127.0.0.1:0", AETitle(""), VerificationContexts())
	if err == nil {
		t.Fatal("Associate with an invalid called AE title = nil error, want rejection")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("error = %T, want *ValidationError (no dial attempted)", err)
	}
}
