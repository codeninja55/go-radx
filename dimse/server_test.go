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

// startServer builds and serves a Server on loopback (OS-assigned port), returning it and a
// function that blocks until ListenAndServe has returned. The caller is responsible for Shutdown.
func startServer(t *testing.T, h Handler, opts ...ServerOption) (*Server, *AE) {
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
	// return PROMPTLY — not wait out its (generous) deadline. A 5s deadline with a sub-1s assertion
	// makes the deadline path unmistakable: only the cancel/wake path returns this fast.
	const shutdownDeadline = 5 * time.Second
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
		if elapsed >= time.Second {
			t.Fatalf("Shutdown took %s (>= 1s); it returned via the deadline, not by cancelling the handler context (DIMSE-014)", elapsed)
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
