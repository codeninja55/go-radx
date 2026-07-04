package dimse

import (
	"bytes"
	"context"
	"errors"
	"net"
	"path/filepath"
	"sync"
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
	if _, ok := errors.AsType[*AssociationError](err); !ok {
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
	if _, ok := errors.AsType[*AssociationError](err); !ok {
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

// TestStoreNoSOPInstanceUIDTransmitsNothing confirms Store fails closed when the dataset carries a
// SOP Class UID but no SOP Instance UID (0008,0018). SOP Instance UID is Type 1 in C-STORE-RQ;
// absent, it would build a malformed RQ whose Affected SOP Instance UID element is silently
// omitted. Store must return a *ValidationError before any wire I/O and transmit no PDU (PRD §9.2).
func TestStoreNoSOPInstanceUIDTransmitsNothing(t *testing.T) {
	addr, gotData := startVerificationOnlySCP(t)
	ae, err := NewAE(AETitle("SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	assoc, err := ae.Associate(ctx, addr, AETitle("SCP"), VerificationContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}

	// A dataset with a SOP Class UID but NO SOP Instance UID (0008,0018).
	ds := dicom.NewDataSet()
	ds.SetString(dicom.NewTag(0x0008, 0x0016), "1.2.840.10008.5.1.4.1.1.2") // CT Image Storage

	_, err = assoc.Store(ctx, ds)
	if err == nil {
		t.Fatal("Store of a dataset with no SOP Instance UID returned nil error, want a *ValidationError")
	}
	if _, ok := errors.AsType[*ValidationError](err); !ok {
		t.Errorf("Store error = %T, want *ValidationError", err)
	}

	// Release; the SCP then confirms it never received a P-DATA-TF (nothing was transmitted).
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

// segmentationStorageSOPClass is the SOP Class of the liver.dcm fixture — Segmentation Storage,
// which is in StorageContexts() (the validated radiology Storage set), encoded uncompressed
// Explicit VR LE, so the four-syntax DIMSE skeleton can negotiate and exercise it end-to-end.
const segmentationStorageSOPClass = "1.2.840.10008.5.1.4.1.1.66.4"

// storeSCPResult captures what the storage SCP dispatch observed.
type storeSCPResult struct {
	status Status
	ds     *dicom.DataSet
	info   *OpInfo
	err    error
}

// recordingStoreHandler persists (records) the dataset it receives and returns a configurable
// status — a StoreHandler that satisfies the fail-closed rule by holding the dataset before
// reporting success.
type recordingStoreHandler struct {
	status Status
	mu     sync.Mutex
	ds     *dicom.DataSet
	info   *OpInfo
}

func (h *recordingStoreHandler) Store(_ context.Context, ds *dicom.DataSet, info OpInfo) Status {
	h.mu.Lock()
	h.ds = ds
	h.info = &info
	h.mu.Unlock()
	return h.status
}

// startStoreSCP listens on loopback and, for each inbound association, negotiates the Segmentation
// Storage context (Explicit VR LE preferred) as acceptor, services one C-STORE via serveStore
// dispatching to h, then services the graceful release.
func startStoreSCP(t *testing.T, called AETitle, h StoreHandler) (string, <-chan storeSCPResult) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	results := make(chan storeSCPResult, 1)
	go func() {
		nc, aerr := ln.Accept()
		if aerr != nil {
			return
		}
		conn := dul.NewConn(nc, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		acc, perr := acse.Accept(ctx, conn, acse.AcceptParams{
			CalledAETitle: string(called),
			MaxPDULength:  16382,
			Supported: []acse.SupportedContext{{
				AbstractSyntax:   segmentationStorageSOPClass,
				TransferSyntaxes: []string{"1.2.840.10008.1.2.1", "1.2.840.10008.1.2"},
			}},
		})
		if perr != nil {
			_ = nc.Close()
			results <- storeSCPResult{err: perr}
			return
		}
		status, serr := serveStore(ctx, acc, AETitle("SCU"), called, h)
		var ds *dicom.DataSet
		var info *OpInfo
		if rec, ok := h.(*recordingStoreHandler); ok {
			rec.mu.Lock()
			ds, info = rec.ds, rec.info
			rec.mu.Unlock()
		}
		results <- storeSCPResult{status: status, ds: ds, info: info, err: serr}
		_ = acc.ServeRelease(ctx)
	}()
	return ln.Addr().String(), results
}

// TestStoreInProcessRoundTrip is the load-bearing C-STORE end-to-end proof: an in-process SCU
// stores a real uncompressed fixture (liver.dcm, Segmentation Storage, Explicit VR LE) to an
// in-process SCP, which persists it and returns StatusStoreSuccess. The SCP-received dataset must
// match the sent one on the SOP Instance UID and a pixel-data sample — proof the command-last-bit
// fix and the negotiated-transfer-syntax dataset decode work end-to-end (DIMSE-001/002/003).
func TestStoreInProcessRoundTrip(t *testing.T) {
	src, err := dicom.ReadFile(filepath.Join("..", "testdata", "dicom", "liver.dcm"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	sentDS := src.DataSet
	sentInstance, _ := sentDS.GetString(tagSOPInstanceUID)
	sentClass, _ := sentDS.GetString(tagSOPClassUID)
	if sentClass != segmentationStorageSOPClass {
		t.Fatalf("fixture SOP Class = %q, want %q", sentClass, segmentationStorageSOPClass)
	}
	sentPixels := pixelSample(t, sentDS)

	h := &recordingStoreHandler{status: StatusStoreSuccess}
	addr, results := startStoreSCP(t, AETitle("SCP"), h)

	ae, err := NewAE(AETitle("SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Negotiate a context for the fixture's SOP Class (Segmentation Storage is in StorageContexts).
	assoc, err := ae.Associate(ctx, addr, AETitle("SCP"), StorageContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}

	status, err := assoc.Store(ctx, sentDS)
	if err != nil {
		t.Fatalf("Store transport error: %v", err)
	}
	if status.Code != StatusStoreSuccess.Code || !status.IsSuccess() {
		t.Errorf("SCU Store status = %s, want store success", status)
	}

	if err := assoc.Release(ctx); err != nil {
		t.Fatalf("Release: %v", err)
	}

	res := <-results
	if res.err != nil {
		t.Fatalf("SCP serveStore: %v", res.err)
	}
	if !res.status.IsSuccess() {
		t.Errorf("SCP dispatch status = %s, want success", res.status)
	}
	if res.ds == nil {
		t.Fatal("SCP persisted no dataset")
	}
	if res.info == nil {
		t.Fatal("SCP recorded no OpInfo")
	}
	if res.info.SOPClassUID != dicom.SOPClassUID(segmentationStorageSOPClass) {
		t.Errorf("OpInfo.SOPClassUID = %q, want %q", res.info.SOPClassUID, segmentationStorageSOPClass)
	}
	if res.info.TransferSyntax != dicom.ExplicitVRLittleEndian {
		t.Errorf("OpInfo.TransferSyntax = %q, want Explicit VR LE (negotiated)", res.info.TransferSyntax)
	}

	// The received dataset must match the sent one on the SOP Instance UID and a pixel sample.
	gotInstance, _ := res.ds.GetString(tagSOPInstanceUID)
	if gotInstance != sentInstance {
		t.Errorf("received SOP Instance UID = %q, want %q", gotInstance, sentInstance)
	}
	gotPixels := pixelSample(t, res.ds)
	if !bytes.Equal(gotPixels, sentPixels) {
		t.Errorf("received pixel sample (%d bytes) does not match the sent sample (%d bytes)",
			len(gotPixels), len(sentPixels))
	}
}

// pixelSample returns the first 64 bytes (or fewer) of the Pixel Data element (7FE0,0010), the
// sample compared across the C-STORE round-trip.
func pixelSample(t *testing.T, ds *dicom.DataSet) []byte {
	t.Helper()
	e, ok := ds.Get(dicom.NewTag(0x7FE0, 0x0010))
	if !ok {
		t.Fatal("dataset has no Pixel Data (7FE0,0010)")
	}
	b, ok := e.Value.(*dicom.Bytes)
	if !ok {
		t.Fatalf("Pixel Data value type = %T, want *dicom.Bytes", e.Value)
	}
	data := b.Bytes()
	if len(data) > 64 {
		data = data[:64]
	}
	return append([]byte(nil), data...)
}
