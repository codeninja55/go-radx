package dimse

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/dul"
	"github.com/codeninja55/go-radx/dimse/pdu"
)

// echoAndStorageContexts builds a single proposal list combining the Verification context and the
// validated Storage contexts with UNIQUE odd context IDs. The presets each number their contexts
// from ID 1 independently, so appending them directly would collide IDs and break negotiation;
// this renumbers the merged set 1, 3, 5, … (PS3.8 §9.3.2.2).
func echoAndStorageContexts() []PresentationContext {
	merged := append(VerificationContexts(), StorageContexts()...)
	id := uint8(1)
	for i := range merged {
		merged[i].ID = id
		id += 2
	}
	return merged
}

// serverTestHandler is a Handler whose Echo and Store responses are configurable, and whose Store
// can optionally block until released — the lever the Shutdown regression uses to park a handler
// mid-operation while connections are closed underneath it.
type serverTestHandler struct {
	echoStatus  Status
	storeStatus Status

	mu       sync.Mutex
	storedDS *dicom.DataSet
	storeOp  *OpInfo

	storeEntered chan struct{} // closed once when Store is first entered
	release      chan struct{} // Store blocks until this is closed (nil = never block)
	enteredOnce  sync.Once
}

func (h *serverTestHandler) Echo(_ context.Context, _ OpInfo) Status { return h.echoStatus }

func (h *serverTestHandler) Store(ctx context.Context, ds *dicom.DataSet, info OpInfo) Status {
	if h.storeEntered != nil {
		h.enteredOnce.Do(func() { close(h.storeEntered) })
	}
	if h.release != nil {
		select {
		case <-h.release:
		case <-ctx.Done():
		}
	}
	h.mu.Lock()
	h.storedDS = ds
	op := info
	h.storeOp = &op
	h.mu.Unlock()
	return h.storeStatus
}

// storeOnlyHandler implements ONLY StoreHandler — the interface-segregation case a store-only SCP
// writes (no dummy Echo method). NewServer must accept it; a C-ECHO routed to it must be refused
// with StatusSOPClassNotSupported, not a panic.
type storeOnlyHandler struct {
	status   Status
	mu       sync.Mutex
	storedDS *dicom.DataSet
}

func (h *storeOnlyHandler) Store(_ context.Context, ds *dicom.DataSet, _ OpInfo) Status {
	h.mu.Lock()
	h.storedDS = ds
	h.mu.Unlock()
	return h.status
}

// echoOnlyHandler implements ONLY EchoHandler — the symmetric interface-segregation case.
type echoOnlyHandler struct {
	status Status
}

func (h *echoOnlyHandler) Echo(_ context.Context, _ OpInfo) Status { return h.status }

// funcStoreHandler adapts a function to StoreHandler, letting a test install arbitrary Store
// behaviour (e.g. a handler that deliberately ignores its context to outlive a Shutdown deadline).
type funcStoreHandler struct {
	fn func(ctx context.Context, ds *dicom.DataSet, info OpInfo) Status
}

func (h *funcStoreHandler) Store(ctx context.Context, ds *dicom.DataSet, info OpInfo) Status {
	return h.fn(ctx, ds, info)
}

// startServer builds and serves a Server on loopback (OS-assigned port), returning it and a
// function that blocks until ListenAndServe has returned. The caller is responsible for Shutdown.
func startServer(t *testing.T, h any, opts ...ServerOption) (*Server, *AE) {
	t.Helper()
	ae, err := NewAE(AETitle("RADX-SCP"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	contexts := echoAndStorageContexts()
	srv := NewServer(ae, contexts, h, opts...)

	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()

	waitForAddr(t, srv)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		select {
		case <-served:
		case <-time.After(5 * time.Second):
			t.Error("ListenAndServe did not return after Shutdown")
		}
	})
	return srv, ae
}

// waitForAddr blocks until the server has bound and Addr reports a non-nil address.
func waitForAddr(t *testing.T, srv *Server) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if srv.Addr() != nil {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("server did not bind within the deadline")
}

// TestServerBindsLoopbackByDefault is the named bind-default regression (PRD §9.1/§11.2): a server
// given an address with no host (a bare ":port") binds 127.0.0.1, not all interfaces.
func TestServerBindsLoopbackByDefault(t *testing.T) {
	ae, err := NewAE(AETitle("RADX-SCP"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	srv := NewServer(ae, VerificationContexts(), &serverTestHandler{echoStatus: StatusEchoSuccess})

	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), ":0") }()
	waitForAddr(t, srv)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		<-served
	})

	tcpAddr, ok := srv.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("Addr() = %T, want *net.TCPAddr", srv.Addr())
	}
	if !tcpAddr.IP.IsLoopback() {
		t.Errorf("server bound %s, want a loopback address", tcpAddr.IP)
	}
}

// TestServerEchoAndStoreInProcess proves an in-process SCU runs C-ECHO and C-STORE against the
// Server and the registered Handler answers each, end to end.
func TestServerEchoAndStoreInProcess(t *testing.T) {
	src, err := dicom.ReadFile(filepath.Join("..", "testdata", "dicom", "liver.dcm"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	sentDS := src.DataSet
	sentInstance, _ := sentDS.GetString(tagSOPInstanceUID)

	h := &serverTestHandler{echoStatus: StatusEchoSuccess, storeStatus: StatusStoreSuccess}
	srv, _ := startServer(t, h)

	scu, err := NewAE(AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	contexts := echoAndStorageContexts()
	assoc, err := scu.Associate(ctx, srv.Addr().String(), AETitle("RADX-SCP"), contexts)
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}

	echoStatus, err := assoc.Echo(ctx)
	if err != nil {
		t.Fatalf("Echo transport error: %v", err)
	}
	if !echoStatus.IsSuccess() {
		t.Errorf("Echo status = %s, want success", echoStatus)
	}

	storeStatus, err := assoc.Store(ctx, sentDS)
	if err != nil {
		t.Fatalf("Store transport error: %v", err)
	}
	if !storeStatus.IsSuccess() {
		t.Errorf("Store status = %s, want success", storeStatus)
	}

	if err := assoc.Release(ctx); err != nil {
		t.Fatalf("Release: %v", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.storedDS == nil {
		t.Fatal("handler stored no dataset")
	}
	gotInstance, _ := h.storedDS.GetString(tagSOPInstanceUID)
	if gotInstance != sentInstance {
		t.Errorf("stored SOP Instance UID = %q, want %q", gotInstance, sentInstance)
	}
	if h.storeOp == nil || h.storeOp.CallingAETitle != "RADX-SCU" || h.storeOp.CalledAETitle != "RADX-SCP" {
		t.Errorf("store OpInfo AE titles = %+v, want calling RADX-SCU / called RADX-SCP", h.storeOp)
	}
}

// TestServerRejectsWrongCalledAETitle confirms an association naming a Called AE Title the server
// does not answer is rejected at negotiation.
func TestServerRejectsWrongCalledAETitle(t *testing.T) {
	h := &serverTestHandler{echoStatus: StatusEchoSuccess}
	srv, _ := startServer(t, h, WithRequireCalledAETitle(AETitle("RADX-SCP")))

	scu, err := NewAE(AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = scu.Associate(ctx, srv.Addr().String(), AETitle("WRONG-SCP"), VerificationContexts())
	if err == nil {
		t.Fatal("Associate with a wrong Called AE Title = nil error, want an association rejection")
	}
	var ae *AssociationError
	if !errors.As(err, &ae) {
		t.Fatalf("error = %T, want *AssociationError", err)
	}
	if ae.Kind != AssociationRejected {
		t.Errorf("association error kind = %s, want rejected", ae.Kind)
	}
}

// TestServerMaxAssociationsRefusesBeforeSpawn is the named DIMSE-013 regression: with a capacity of
// one, a second concurrent association is refused before a handler goroutine is spawned for it.
// The first association is held open (its handler parked in Store) while the second is attempted;
// the second must fail to establish, and the handler must never have been entered for it.
func TestServerMaxAssociationsRefusesBeforeSpawn(t *testing.T) {
	src, err := dicom.ReadFile(filepath.Join("..", "testdata", "dicom", "liver.dcm"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	sentDS := src.DataSet

	h := &serverTestHandler{
		echoStatus:   StatusEchoSuccess,
		storeStatus:  StatusStoreSuccess,
		storeEntered: make(chan struct{}),
		release:      make(chan struct{}),
	}
	srv, _ := startServer(t, h, WithMaxAssociations(1))
	var releaseOnce sync.Once
	releaseHandler := func() { releaseOnce.Do(func() { close(h.release) }) }
	defer releaseHandler()

	scu, err := NewAE(AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	contexts := echoAndStorageContexts()

	// First association: establish and start a C-STORE whose handler parks, occupying the one slot.
	assoc1, err := scu.Associate(ctx, srv.Addr().String(), AETitle("RADX-SCP"), contexts)
	if err != nil {
		t.Fatalf("first Associate: %v", err)
	}
	storeDone := make(chan struct{})
	go func() {
		_, _ = assoc1.Store(ctx, sentDS)
		close(storeDone)
	}()
	select {
	case <-h.storeEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("first handler never entered Store; cannot prove the slot is occupied")
	}

	// Second association: the slot is taken, so the server must refuse it before spawning a handler.
	dialCtx, dialCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer dialCancel()
	assoc2, err := scu.Associate(dialCtx, srv.Addr().String(), AETitle("RADX-SCP"), contexts)
	if err == nil {
		_ = assoc2.Abort(dialCtx)
		t.Fatal("second Associate succeeded despite WithMaxAssociations(1); the capacity was not enforced before spawn")
	}

	// Release the parked handler and let the first association complete.
	releaseHandler()
	select {
	case <-storeDone:
	case <-time.After(5 * time.Second):
		t.Fatal("first Store did not complete after release")
	}
	_ = assoc1.Release(ctx)
}

// TestServerShutdownClosesConnectionsWhileHandlerBlocked is the named DIMSE-014 regression for
// cooperative shutdown. A handler doing application work (here: a Store parked observing its
// context, the realistic C-STORE-persisting-to-disk case) must be woken by Shutdown CANCELLING the
// handler context — not left to wait out the Shutdown deadline. The handler here blocks ONLY on its
// context (h.release is never closed during the assertion), so if Shutdown returned via its
// deadline rather than the cancel/wake path the elapsed time would approach the 5s deadline; the
// test asserts it returns well under 1s, and so FAILS if the context is not cancelled.
//
// Earlier this test was hollow: its handler blocked on a test channel and Shutdown passed via the
// close-first deadline branch (~the deadline), never exercising the wake path it claimed to prove.
func TestServerShutdownClosesConnectionsWhileHandlerBlocked(t *testing.T) {
	src, err := dicom.ReadFile(filepath.Join("..", "testdata", "dicom", "liver.dcm"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	sentDS := src.DataSet

	h := &serverTestHandler{
		echoStatus:   StatusEchoSuccess,
		storeStatus:  StatusStoreSuccess,
		storeEntered: make(chan struct{}),
		release:      make(chan struct{}), // closed only on cleanup, never during the assertion
	}

	ae, err := NewAE(AETitle("RADX-SCP"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	contexts := echoAndStorageContexts()
	srv := NewServer(ae, contexts, h)
	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	waitForAddr(t, srv)
	defer close(h.release)

	scu, err := NewAE(AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	assoc, err := scu.Associate(ctx, srv.Addr().String(), AETitle("RADX-SCP"), contexts)
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	go func() { _, _ = assoc.Store(ctx, sentDS) }()

	select {
	case <-h.storeEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("handler never entered Store; cannot prove it is parked mid-operation")
	}

	// Shutdown cancels the handler context, waking the parked handler cooperatively, and must
	// return PROMPTLY — not wait out its (generous) deadline. The assertion bound is sized for the
	// race detector: a cooperative cancel/wake returns in milliseconds, so a 3s bound (well under the
	// 8s deadline) still makes the deadline path unmistakable while leaving headroom for scheduler
	// jitter on a cold -race run, where a tight sub-1s bound trips spuriously.
	const (
		shutdownDeadline = 8 * time.Second
		promptReturn     = 3 * time.Second // cooperative wake returns in ms; bound padded for -race jitter
	)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownDeadline)
	defer shutdownCancel()
	done := make(chan error, 1)
	start := time.Now()
	go func() { done <- srv.Shutdown(shutdownCtx) }()

	select {
	case serr := <-done:
		elapsed := time.Since(start)
		if serr != nil {
			t.Fatalf("Shutdown returned %v (deadline path); want a prompt cooperative return", serr)
		}
		if elapsed >= promptReturn {
			t.Fatalf("Shutdown took %s (>= %s); it returned via the deadline, not by cancelling the handler context (DIMSE-014)", elapsed, promptReturn)
		}
		t.Logf("Shutdown returned promptly in %s (cooperative cancel, deadline %s)", elapsed, shutdownDeadline)
	case <-time.After(shutdownDeadline + time.Second):
		t.Fatal("Shutdown did not return; the handler context was not cancelled (DIMSE-014)")
	}

	// ListenAndServe returns once the listener is closed.
	select {
	case <-served:
	case <-time.After(3 * time.Second):
		t.Error("ListenAndServe did not return after Shutdown")
	}
}

// TestServerIdleAssociationTimesOut is the named regression for the idle-association DoS: a peer
// that completes negotiation then sends nothing must not hold a capacity slot forever. With a
// short injected network timeout and a single capacity slot, the silent peer's association is
// closed within ~the network timeout, releasing the slot so a subsequent C-ECHO association is
// served (Codex/concurrency review). Without the per-read timeout this test would hang on the
// second association until the outer deadline.
func TestServerIdleAssociationTimesOut(t *testing.T) {
	const networkTimeout = 300 * time.Millisecond

	ae, err := NewAE(AETitle("RADX-SCP"), WithNetworkTimeout(networkTimeout))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	contexts := echoAndStorageContexts()
	h := &serverTestHandler{echoStatus: StatusEchoSuccess, storeStatus: StatusStoreSuccess}
	srv := NewServer(ae, contexts, h, WithMaxAssociations(1))

	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	waitForAddr(t, srv)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		select {
		case <-served:
		case <-time.After(5 * time.Second):
			t.Error("ListenAndServe did not return after Shutdown")
		}
	})

	scu, err := NewAE(AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}

	// First association: negotiate, then send NOTHING. It holds the single slot until the
	// server's per-read network timeout closes it.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	idle, err := scu.Associate(ctx, srv.Addr().String(), AETitle("RADX-SCP"), contexts)
	if err != nil {
		t.Fatalf("first (idle) Associate: %v", err)
	}
	t.Cleanup(func() { _ = idle.Abort(context.Background()) })

	// Second association: it can only establish AND complete a C-ECHO once the idle peer's slot is
	// released by the timeout. Bound it generously relative to the injected network timeout; if the
	// slot were never released this would fail at the deadline.
	start := time.Now()
	deadline := time.Now().Add(5 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		dialCtx, dialCancel := context.WithTimeout(context.Background(), 2*time.Second)
		assoc, aerr := scu.Associate(dialCtx, srv.Addr().String(), AETitle("RADX-SCP"), contexts)
		if aerr != nil {
			dialCancel()
			lastErr = aerr
			time.Sleep(20 * time.Millisecond)
			continue
		}
		status, eerr := assoc.Echo(dialCtx)
		if eerr == nil && status.IsSuccess() {
			_ = assoc.Release(dialCtx)
			dialCancel()
			t.Logf("second association served after %s (network timeout %s)", time.Since(start), networkTimeout)
			return
		}
		_ = assoc.Abort(dialCtx)
		dialCancel()
		lastErr = eerr
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("second association was never served within the deadline; the idle slot was not released (last error: %v)", lastErr)
}

// TestServerNegotiationTimesOut is the named regression for the negotiation-phase slot-exhaustion
// DoS (P1 adversarial review): a peer that opens TCP, acquires a capacity slot, but NEVER sends the
// A-ASSOCIATE-RQ must not hold the slot (and its goroutine) forever. With a single capacity slot and
// a short injected ACSE timeout, the silent dialer's negotiation read times out, releasing the slot
// so a subsequent real association can establish and complete a C-ECHO. Without bounding the
// acse.Accept read by the ACSE timeout this test would hang on the second association.
func TestServerNegotiationTimesOut(t *testing.T) {
	const acseTimeout = 300 * time.Millisecond

	ae, err := NewAE(AETitle("RADX-SCP"), WithACSETimeout(acseTimeout))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	contexts := echoAndStorageContexts()
	h := &serverTestHandler{echoStatus: StatusEchoSuccess, storeStatus: StatusStoreSuccess}
	srv := NewServer(ae, contexts, h, WithMaxAssociations(1))

	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	waitForAddr(t, srv)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		select {
		case <-served:
		case <-time.After(5 * time.Second):
			t.Error("ListenAndServe did not return after Shutdown")
		}
	})

	// Silent peer: connect TCP (acquiring the one slot) and send NOTHING. The server's negotiation
	// read (acse.Accept) must time out under the ACSE timeout and release the slot.
	raw, err := net.Dial("tcp", srv.Addr().String())
	if err != nil {
		t.Fatalf("raw Dial: %v", err)
	}
	t.Cleanup(func() { _ = raw.Close() })

	scu, err := NewAE(AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}

	// Second association: it can only establish AND complete a C-ECHO once the silent peer's slot is
	// released by the negotiation timeout. Poll generously relative to the injected ACSE timeout; if
	// the slot were never released this would fail at the deadline.
	start := time.Now()
	deadline := time.Now().Add(5 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		dialCtx, dialCancel := context.WithTimeout(context.Background(), 2*time.Second)
		assoc, aerr := scu.Associate(dialCtx, srv.Addr().String(), AETitle("RADX-SCP"), contexts)
		if aerr != nil {
			dialCancel()
			lastErr = aerr
			time.Sleep(20 * time.Millisecond)
			continue
		}
		status, eerr := assoc.Echo(dialCtx)
		if eerr == nil && status.IsSuccess() {
			_ = assoc.Release(dialCtx)
			dialCancel()
			t.Logf("second association served after %s (ACSE timeout %s)", time.Since(start), acseTimeout)
			return
		}
		_ = assoc.Abort(dialCtx)
		dialCancel()
		lastErr = eerr
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("second association was never served within the deadline; the negotiation slot was not released (last error: %v)", lastErr)
}

// TestServerLargeStoreNotCappedByIdleTimeout is the named regression for the idle-vs-total timeout
// defect (P2 adversarial review): WithNetworkTimeout is an IDLE-association bound — it must bound
// each individual inbound PDU read, NOT the whole multi-PDU message. A legitimate large/slow C-STORE
// delivered as many P-DATA-TF PDUs, each arriving within the network timeout of the previous but
// whose TOTAL duration exceeds it, must succeed: continuous progress resets the deadline.
//
// The slow-drip SCU is built with the PRODUCTION encoders: it negotiates normally, then uses
// fragmentMessage (the same fragmenter Store uses) to encode the C-STORE-RQ command and dataset, and
// writes the resulting PDUs one at a time through the requestor's own conn/state-machine, advancing
// the machine by Evt9 per write exactly as sendMessage does. A small advertised maximum PDU length
// forces the dataset to fragment into many PDVs. The first PDUs are dripped with a gap that is well
// under the injected network timeout but whose CUMULATIVE duration exceeds it, then the remainder are
// sent back to back; the slow phase alone crosses the total bound while no single inter-PDU gap does,
// so the only way the store succeeds is if the timeout is reset per read. Before the per-read fix the
// server derived ONE timeout for the whole reassembly loop, so this store was aborted mid-transfer
// once the total exceeded the timeout.
//
// Margins are sized for the race detector: an 80ms gap sits 120ms under the 200ms bound, so
// scheduler jitter cannot spuriously trip the per-read deadline, while four such gaps (320ms) clear
// the 200ms total comfortably.
func TestServerLargeStoreNotCappedByIdleTimeout(t *testing.T) {
	const (
		networkTimeout  = 200 * time.Millisecond
		interPDUGap     = 80 * time.Millisecond // < networkTimeout: each idle gap stays well under the bound
		slowDrips       = 4                     // 4 * 80ms = 320ms cumulative >> 200ms network timeout
		scuMaxPDULength = MaxPDULength(64)      // tiny cap => the dataset fragments into many PDVs
	)

	src, err := dicom.ReadFile(filepath.Join("..", "testdata", "dicom", "liver.dcm"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	sentDS := src.DataSet
	sentInstance, _ := sentDS.GetString(tagSOPInstanceUID)

	ae, err := NewAE(AETitle("RADX-SCP"), WithNetworkTimeout(networkTimeout))
	if err != nil {
		t.Fatalf("NewAE (SCP): %v", err)
	}
	contexts := echoAndStorageContexts()
	h := &serverTestHandler{echoStatus: StatusEchoSuccess, storeStatus: StatusStoreSuccess}
	srv := NewServer(ae, contexts, h)

	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	waitForAddr(t, srv)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		select {
		case <-served:
		case <-time.After(5 * time.Second):
			t.Error("ListenAndServe did not return after Shutdown")
		}
	})

	scu, err := NewAE(AETitle("RADX-SCU"), WithMaxPDULength(scuMaxPDULength))
	if err != nil {
		t.Fatalf("NewAE (SCU): %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	assoc, err := scu.Associate(ctx, srv.Addr().String(), AETitle("RADX-SCP"), contexts)
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}

	// Select the accepted Storage context and its transfer syntax, then build the C-STORE-RQ exactly
	// as Store does, fragmenting with the production fragmenter under the SCU's send cap.
	sopClass, _ := sentDS.GetString(tagSOPClassUID)
	pcID, ts, ok := assoc.contextForStorage(dicom.SOPClassUID(sopClass))
	if !ok {
		t.Fatalf("no accepted presentation context for SOP Class %s", sopClass)
	}
	rq := CommandSet{
		CommandField:           CommandCStoreRQ,
		MessageID:              storeMessageID,
		AffectedSOPClassUID:    dicom.UID(sopClass),
		AffectedSOPInstanceUID: dicom.UID(sentInstance),
		HasPriority:            true,
		Priority:               PriorityMedium,
		CommandDataSetType:     CommandDataSetPresent,
	}
	pdus, err := fragmentMessage(rq, sentDS, ts, pcID, assoc.sendCap())
	if err != nil {
		t.Fatalf("fragmentMessage: %v", err)
	}
	if len(pdus) <= slowDrips {
		t.Fatalf("fragmentMessage produced %d PDUs, need more than %d to span the network timeout", len(pdus), slowDrips)
	}

	conn := assoc.requestor.Conn()
	m := assoc.requestor.Machine()

	// Drip the first slowDrips PDUs with a gap < networkTimeout between each (cumulatively crossing
	// the timeout), then stream the rest back to back. Each write advances the state machine by Evt9
	// just as sendMessage does. The fix keeps the association alive because each individual read
	// completes within the bound even though the slow phase alone outlasts the whole timeout.
	for i, p := range pdus {
		if i > 0 && i <= slowDrips {
			time.Sleep(interPDUGap)
		}
		if _, _, serr := m.Apply(dul.Evt9); serr != nil {
			t.Fatalf("Apply(Evt9) before PDU %d: %v", i, serr)
		}
		if werr := conn.WritePDU(ctx, p); werr != nil {
			t.Fatalf("WritePDU %d/%d: %v", i+1, len(pdus), werr)
		}
	}

	// The server reassembled the slow-dripped message and the handler answered: read the C-STORE-RSP.
	rsp, _, _, err := receiveMessage(ctx, conn, m, newMessageReassembler(ts))
	if err != nil {
		t.Fatalf("receive C-STORE-RSP: %v (the slow store was aborted; the idle timeout capped the whole message)", err)
	}
	if rsp.CommandField != CommandCStoreRSP {
		t.Fatalf("response command field = %v, want C-STORE-RSP", rsp.CommandField)
	}
	status := NewStatus(rsp.Status, ServiceClassStorage)
	if !status.IsSuccess() {
		t.Errorf("C-STORE status = %s, want success", status)
	}
	_ = assoc.Release(ctx)

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.storedDS == nil {
		t.Fatal("handler stored no dataset despite a successful response")
	}
	gotInstance, _ := h.storedDS.GetString(tagSOPInstanceUID)
	if gotInstance != sentInstance {
		t.Errorf("stored SOP Instance UID = %q, want %q", gotInstance, sentInstance)
	}
}

// TestServerReleaseCompletionTimesOut is the named regression for the release-completion DoS (P1
// adversarial review): the acceptor's CompleteRelease awaits the peer's orderly transport close (a
// final inbound read). A peer that sends A-RELEASE-RQ then HOLDS the TCP connection open — never
// closing — must not block that read forever, holding the capacity slot and goroutine until
// Shutdown. With a single capacity slot and a short network timeout, the await-close read must time
// out and release the slot so a subsequent association can be served.
//
// The misbehaving peer is built with the production association machinery: it negotiates normally,
// then writes an A-RELEASE-RQ directly through its conn/state-machine (advancing by Evt11 exactly as
// Requestor.Release does) but, unlike Release, never reads the A-RELEASE-RP and never closes — it
// just holds the socket open. Before the fix the server's CompleteRelease read used the long-lived
// server context and parked forever on that silent peer, never releasing the slot.
func TestServerReleaseCompletionTimesOut(t *testing.T) {
	const networkTimeout = 300 * time.Millisecond

	ae, err := NewAE(AETitle("RADX-SCP"), WithNetworkTimeout(networkTimeout))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	contexts := echoAndStorageContexts()
	h := &serverTestHandler{echoStatus: StatusEchoSuccess, storeStatus: StatusStoreSuccess}
	srv := NewServer(ae, contexts, h, WithMaxAssociations(1))

	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	waitForAddr(t, srv)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		select {
		case <-served:
		case <-time.After(5 * time.Second):
			t.Error("ListenAndServe did not return after Shutdown")
		}
	})

	scu, err := NewAE(AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE (SCU): %v", err)
	}

	// Misbehaving peer: negotiate, send A-RELEASE-RQ, then hold the connection open (never reading
	// the A-RELEASE-RP, never closing). It occupies the single slot until the await-close read times
	// out.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stuck, err := scu.Associate(ctx, srv.Addr().String(), AETitle("RADX-SCP"), contexts)
	if err != nil {
		t.Fatalf("misbehaving peer Associate: %v", err)
	}
	t.Cleanup(func() { _ = stuck.requestor.Conn().Close() })

	stuckConn := stuck.requestor.Conn()
	stuckMachine := stuck.requestor.Machine()
	if _, _, serr := stuckMachine.Apply(dul.Evt11); serr != nil { // A-RELEASE request -> AR-1 -> Sta7
		t.Fatalf("Apply(Evt11): %v", serr)
	}
	if werr := stuckConn.WritePDU(ctx, &pdu.ReleaseRQ{}); werr != nil {
		t.Fatalf("write A-RELEASE-RQ: %v", werr)
	}
	// Deliberately do NOT read the A-RELEASE-RP and do NOT close: the peer holds the socket open.

	// Second association: it can only establish AND complete a C-ECHO once the stuck peer's slot is
	// released by the await-close timeout. Poll generously relative to the injected network timeout;
	// if the slot were never released this would fail at the deadline.
	start := time.Now()
	deadline := time.Now().Add(5 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		dialCtx, dialCancel := context.WithTimeout(context.Background(), 2*time.Second)
		assoc, aerr := scu.Associate(dialCtx, srv.Addr().String(), AETitle("RADX-SCP"), contexts)
		if aerr != nil {
			dialCancel()
			lastErr = aerr
			time.Sleep(20 * time.Millisecond)
			continue
		}
		status, eerr := assoc.Echo(dialCtx)
		if eerr == nil && status.IsSuccess() {
			_ = assoc.Release(dialCtx)
			dialCancel()
			t.Logf("second association served after %s (network timeout %s)", time.Since(start), networkTimeout)
			return
		}
		_ = assoc.Abort(dialCtx)
		dialCancel()
		lastErr = eerr
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("second association was never served within the deadline; the release-completion slot was not released (last error: %v)", lastErr)
}

// TestServerShutdownIsIdempotent confirms a second Shutdown after the first is a safe no-op.
func TestServerShutdownIsIdempotent(t *testing.T) {
	srv, _ := startServer(t, &serverTestHandler{echoStatus: StatusEchoSuccess})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("first Shutdown: %v", err)
	}
	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("second Shutdown = %v, want nil (idempotent)", err)
	}
}

// TestServerListenAndServeStopsOnContextCancel confirms ListenAndServe returns when its context is
// cancelled (the documented "serves until ctx is cancelled" contract), not only on Shutdown.
func TestServerListenAndServeStopsOnContextCancel(t *testing.T) {
	ae, err := NewAE(AETitle("RADX-SCP"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	srv := NewServer(ae, VerificationContexts(), &serverTestHandler{echoStatus: StatusEchoSuccess})

	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(ctx, "127.0.0.1:0") }()
	waitForAddr(t, srv)

	cancel()
	select {
	case err := <-served:
		if err != nil {
			t.Errorf("ListenAndServe after ctx cancel = %v, want nil (clean stop)", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("ListenAndServe did not return after its context was cancelled")
	}
}

// TestNewServerAcceptsStoreOnlyHandler is the named interface-segregation regression (P2
// adversarial review): a handler implementing ONLY StoreHandler (no Echo method) must compile, be
// accepted by NewServer, and serve a C-STORE end to end. A C-ECHO routed to the same store-only
// handler must be refused gracefully with StatusSOPClassNotSupported (0x0122) — the dispatcher
// type-asserts the EchoHandler capability per operation — never a panic or an accepted echo.
func TestNewServerAcceptsStoreOnlyHandler(t *testing.T) {
	src, err := dicom.ReadFile(filepath.Join("..", "testdata", "dicom", "liver.dcm"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	sentDS := src.DataSet
	sentInstance, _ := sentDS.GetString(tagSOPInstanceUID)

	h := &storeOnlyHandler{status: StatusStoreSuccess}
	srv, _ := startServer(t, h)

	scu, err := NewAE(AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	contexts := echoAndStorageContexts()
	assoc, err := scu.Associate(ctx, srv.Addr().String(), AETitle("RADX-SCP"), contexts)
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}

	// A C-STORE is the handler's capability: it must be served and stored.
	storeStatus, err := assoc.Store(ctx, sentDS)
	if err != nil {
		t.Fatalf("Store transport error: %v", err)
	}
	if !storeStatus.IsSuccess() {
		t.Errorf("Store status = %s, want success", storeStatus)
	}

	// A C-ECHO is a capability this handler does NOT implement: the dispatcher must refuse it with
	// StatusSOPClassNotSupported (a peer-visible RSP), not panic and not accept it.
	echoStatus, err := assoc.Echo(ctx)
	if err != nil {
		t.Fatalf("Echo transport error: %v (want a graceful unsupported-status RSP, not a fault)", err)
	}
	if echoStatus.Code != StatusSOPClassNotSupported.Code {
		t.Errorf("Echo status = %s, want 0x%04X (Refused: SOP Class Not Supported)", echoStatus, StatusSOPClassNotSupported.Code)
	}

	if err := assoc.Release(ctx); err != nil {
		t.Fatalf("Release: %v", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.storedDS == nil {
		t.Fatal("store-only handler stored no dataset")
	}
	gotInstance, _ := h.storedDS.GetString(tagSOPInstanceUID)
	if gotInstance != sentInstance {
		t.Errorf("stored SOP Instance UID = %q, want %q", gotInstance, sentInstance)
	}
}

// TestNewServerAcceptsEchoOnlyHandler is the symmetric interface-segregation regression: a handler
// implementing ONLY EchoHandler must compile, be accepted by NewServer, and serve a C-ECHO. A
// C-STORE routed to it must be refused with StatusSOPClassNotSupported, not a panic.
func TestNewServerAcceptsEchoOnlyHandler(t *testing.T) {
	src, err := dicom.ReadFile(filepath.Join("..", "testdata", "dicom", "liver.dcm"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	sentDS := src.DataSet

	h := &echoOnlyHandler{status: StatusEchoSuccess}
	srv, _ := startServer(t, h)

	scu, err := NewAE(AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	contexts := echoAndStorageContexts()
	assoc, err := scu.Associate(ctx, srv.Addr().String(), AETitle("RADX-SCP"), contexts)
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}

	// A C-ECHO is the handler's capability: it must be served.
	echoStatus, err := assoc.Echo(ctx)
	if err != nil {
		t.Fatalf("Echo transport error: %v", err)
	}
	if !echoStatus.IsSuccess() {
		t.Errorf("Echo status = %s, want success", echoStatus)
	}

	// A C-STORE is a capability this handler does NOT implement: refuse with the unsupported status.
	storeStatus, err := assoc.Store(ctx, sentDS)
	if err != nil {
		t.Fatalf("Store transport error: %v (want a graceful unsupported-status RSP, not a fault)", err)
	}
	if storeStatus.Code != StatusSOPClassNotSupported.Code {
		t.Errorf("Store status = %s, want 0x%04X (Refused: SOP Class Not Supported)", storeStatus, StatusSOPClassNotSupported.Code)
	}

	if err := assoc.Release(ctx); err != nil {
		t.Fatalf("Release: %v", err)
	}
}

// TestServerAdvertisesImplementationIdentity is the named regression (P2 adversarial review): the
// Server must propagate the AE's configured Implementation Class UID and Implementation Version
// Name into the A-ASSOCIATE-AC it sends on an inbound association, mirroring the outbound SCU side.
// An SCU associating to such a Server must see those values in the accepted association's user
// information.
func TestServerAdvertisesImplementationIdentity(t *testing.T) {
	const (
		implClassUID = dicom.UID("1.2.840.99999.1")
		implVersion  = "GO-RADX-TEST"
	)

	ae, err := NewAE(
		AETitle("RADX-SCP"),
		WithImplementationClassUID(implClassUID),
		WithImplementationVersionName(implVersion),
	)
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	contexts := echoAndStorageContexts()
	srv := NewServer(ae, contexts, &serverTestHandler{echoStatus: StatusEchoSuccess, storeStatus: StatusStoreSuccess})

	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	waitForAddr(t, srv)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		select {
		case <-served:
		case <-time.After(5 * time.Second):
			t.Error("ListenAndServe did not return after Shutdown")
		}
	})

	scu, err := NewAE(AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE (SCU): %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	assoc, err := scu.Associate(ctx, srv.Addr().String(), AETitle("RADX-SCP"), contexts)
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	defer func() { _ = assoc.Release(ctx) }()

	if got := assoc.PeerImplementationClassUID(); got != implClassUID {
		t.Errorf("peer Implementation Class UID = %q, want %q", got, implClassUID)
	}
	if got := assoc.PeerImplementationVersionName(); got != implVersion {
		t.Errorf("peer Implementation Version Name = %q, want %q", got, implVersion)
	}
}

// TestServerShutdownRetryAfterDeadline is the named regression (P2 adversarial review): a second
// Shutdown after the first hit its deadline must report the REAL outcome, not rubber-stamp success.
// The once-only teardown (close listener, cancel handler ctx, close conns) happens once and is
// idempotent, but the bounded wg.Wait() is re-runnable: each Shutdown re-attempts the join against
// its own ctx.
//
// A handler is parked observing neither its context wake nor a release until the test allows it
// (it blocks on a test channel that ignores ctx for this assertion). The first Shutdown(shortCtx)
// must return a deadline error (handlers still running). A second Shutdown while the handler is
// STILL parked must ALSO return a non-nil deadline error — a false nil here is the defect. Then the
// handler is released and a final Shutdown(freshCtx) must return nil, proving the second call truly
// waited rather than short-circuiting through a sync.Once.
func TestServerShutdownRetryAfterDeadline(t *testing.T) {
	src, err := dicom.ReadFile(filepath.Join("..", "testdata", "dicom", "liver.dcm"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	sentDS := src.DataSet

	// hardParked blocks the handler ONLY on the test release channel (ignoring ctx), so neither the
	// listener close nor the handler-ctx cancel can wake it — the handler genuinely outlives the
	// first Shutdown's deadline, the situation that exposed the false-nil second Shutdown.
	hardParked := make(chan struct{})
	parkedEntered := make(chan struct{})
	var enteredOnce sync.Once
	h := &funcStoreHandler{
		fn: func(_ context.Context, ds *dicom.DataSet, _ OpInfo) Status {
			enteredOnce.Do(func() { close(parkedEntered) })
			<-hardParked // intentionally NOT selecting on ctx: this handler ignores its context
			return StatusStoreSuccess
		},
	}
	_ = sentDS

	ae, err := NewAE(AETitle("RADX-SCP"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	contexts := echoAndStorageContexts()
	srv := NewServer(ae, contexts, h)
	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	waitForAddr(t, srv)
	// Guarantee the parked handler is released no matter how the test exits, so ListenAndServe and
	// the handler goroutine can finish.
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(hardParked) }) }
	defer release()

	scu, err := NewAE(AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE (SCU): %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	assoc, err := scu.Associate(ctx, srv.Addr().String(), AETitle("RADX-SCP"), contexts)
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	go func() { _, _ = assoc.Store(ctx, sentDS) }()

	select {
	case <-parkedEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("handler never entered Store; cannot park it for the retry assertion")
	}

	// First Shutdown: the handler is parked and ignores its context, so the bounded wait must hit
	// the short deadline and return a non-nil error.
	firstCtx, firstCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer firstCancel()
	if err := srv.Shutdown(firstCtx); err == nil {
		t.Fatal("first Shutdown returned nil while a handler was still parked; want a deadline error")
	}

	// Second Shutdown while STILL parked: it must RE-RUN the bounded wait against its own ctx and
	// return the real (still-running) outcome — a non-nil deadline error. A nil here is the
	// rubber-stamp defect a sync.Once-gated wait produces.
	secondCtx, secondCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer secondCancel()
	if err := srv.Shutdown(secondCtx); err == nil {
		t.Fatal("second Shutdown while the handler is still parked returned nil; it falsely reported a clean shutdown (the bounded wait was gated behind a sync.Once)")
	}

	// Release the handler so it actually finishes, then a fresh Shutdown must observe the real
	// completion and return nil — proving the retry genuinely waited rather than short-circuiting.
	release()
	finalCtx, finalCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer finalCancel()
	if err := srv.Shutdown(finalCtx); err != nil {
		t.Fatalf("final Shutdown after releasing the handler = %v, want nil (handlers finished)", err)
	}

	select {
	case <-served:
	case <-time.After(5 * time.Second):
		t.Error("ListenAndServe did not return after the handlers finished")
	}
}

// TestServerListenRejectsBadBind confirms a malformed bind address surfaces as an error from
// ListenAndServe rather than a hang.
func TestServerListenRejectsBadBind(t *testing.T) {
	ae, err := NewAE(AETitle("RADX-SCP"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	srv := NewServer(ae, VerificationContexts(), &serverTestHandler{echoStatus: StatusEchoSuccess})
	err = srv.ListenAndServe(context.Background(), "127.0.0.1:not-a-port")
	if err == nil {
		t.Fatal("ListenAndServe on a bad bind address = nil error, want a bind error")
	}
	if !strings.Contains(err.Error(), "dimse") {
		t.Errorf("error = %q, want a dimse-prefixed bind error", err)
	}
}
