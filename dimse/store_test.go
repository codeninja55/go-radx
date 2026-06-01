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

// startVerificationOnlySCP listens on loopback and negotiates ONLY the Verification context as
// acceptor, then waits. It is used by the fail-closed regression: a Store of a Storage SOP Class
// finds no accepted context and must transmit nothing, so this SCP never expects a P-DATA-TF. It
// reports on a channel if it ever receives one (which would mean Store sent on a missing context).
func startVerificationOnlySCP(t *testing.T) (string, <-chan error) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	gotData := make(chan error, 1)
	go func() {
		nc, aerr := ln.Accept()
		if aerr != nil {
			return
		}
		conn := dul.NewConn(nc, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		acc, perr := acse.Accept(ctx, conn, acse.AcceptParams{
			CalledAETitle: "SCP",
			MaxPDULength:  16382,
			Supported: []acse.SupportedContext{{
				AbstractSyntax:   string(verificationSOPClass),
				TransferSyntaxes: []string{"1.2.840.10008.1.2.1", "1.2.840.10008.1.2"},
			}},
		})
		if perr != nil {
			_ = nc.Close()
			return
		}
		// Read one PDU: if Store wrongly transmitted, it is a P-DATA-TF and we flag it; otherwise
		// the SCU releases and DriveInbound sees the A-RELEASE-RQ.
		m := acc.Machine()
		p, _, derr := dul.DriveInbound(ctx, acc.Conn(), m)
		if derr == nil && p != nil && p.Type().String() == "P-DATA-TF" {
			gotData <- errors.New("SCP received a P-DATA-TF: Store transmitted on a missing context")
			return
		}
		gotData <- nil
	}()
	return ln.Addr().String(), gotData
}

// TestStoreNoMatchingContextTransmitsNothing is the named PRD §9.2 fail-closed regression: Store
// with no accepted presentation context for the dataset's SOP Class returns a typed error and
// sends no PDU — it never reports success on work it did not do (the rule the prototype's store
// violated).
func TestStoreNoMatchingContextTransmitsNothing(t *testing.T) {
	addr, gotData := startVerificationOnlySCP(t)

	ae, err := NewAE(AETitle("SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Associate proposing ONLY Verification, so no Storage context is accepted.
	assoc, err := ae.Associate(ctx, addr, AETitle("SCP"), VerificationContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}

	// A CT dataset whose SOP Class has no accepted context.
	ds := dicom.NewDataSet()
	ds.SetString(dicom.NewTag(0x0008, 0x0016), "1.2.840.10008.5.1.4.1.1.2") // CT Image Storage
	ds.SetString(dicom.NewTag(0x0008, 0x0018), "1.2.3.4.5")

	_, err = assoc.Store(ctx, ds)
	if err == nil {
		t.Fatal("Store with no matching context returned nil error, want a typed error (fail-closed)")
	}
	var ae2 *AssociationError
	if !errors.As(err, &ae2) {
		t.Errorf("Store error = %T, want *AssociationError", err)
	}

	// Release the association; the SCP then confirms it never received a P-DATA-TF.
	_ = assoc.Release(ctx)
	select {
	case derr := <-gotData:
		if derr != nil {
			t.Error(derr)
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for the SCP to confirm no P-DATA-TF was sent")
	}
}

// TestStoreOnUnestablishedAssociationReturnsTypedError confirms Store guards against an
// unestablished/released association with a typed error, never a panic (DIMSE-017).
func TestStoreOnUnestablishedAssociationReturnsTypedError(t *testing.T) {
	var a *Association
	ds := dicom.NewDataSet()
	ds.SetString(dicom.NewTag(0x0008, 0x0016), "1.2.840.10008.5.1.4.1.1.2")
	_, err := a.Store(context.Background(), ds)
	var ae *AssociationError
	if !errors.As(err, &ae) {
		t.Fatalf("Store on nil association error = %T, want *AssociationError", err)
	}
}

// TestStoreRejectsDatasetWithoutSOPClass confirms Store fails closed when the dataset carries no
// SOP Class UID to select a context with (a validation fault, not a transmitted store).
func TestStoreRejectsDatasetWithoutSOPClass(t *testing.T) {
	addr, _ := startVerificationOnlySCP(t)
	ae, _ := NewAE(AETitle("SCU"))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	assoc, err := ae.Associate(ctx, addr, AETitle("SCP"), VerificationContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	defer func() { _ = assoc.Release(ctx) }()

	ds := dicom.NewDataSet() // no (0008,0016)
	if _, err := assoc.Store(ctx, ds); err == nil {
		t.Error("Store of a dataset with no SOP Class UID returned nil error, want a typed error")
	}
}
