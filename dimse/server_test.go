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

// TestServerShutdownClosesConnectionsWhileHandlerBlocked is the named DIMSE-014 regression: Shutdown
// with a deadline returns after closing active association connections even while a handler is
// parked mid-store. The prototype waited for handlers WITHOUT closing connections, so a handler
// blocked in ReadPDU hung Shutdown forever; this confirms Shutdown returns promptly.
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
		release:      make(chan struct{}),
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

	// Shutdown must close the active connection (waking the handler's blocked reads) and return
	// within the deadline, never hang waiting for a handler still blocked in ReadPDU.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()
	done := make(chan error, 1)
	go func() { done <- srv.Shutdown(shutdownCtx) }()

	select {
	case <-done:
		// Returned within the deadline — the close-first ordering worked.
	case <-time.After(4 * time.Second):
		t.Fatal("Shutdown did not return while a handler was parked; connections were not closed first (DIMSE-014)")
	}

	// ListenAndServe returns once the listener is closed.
	select {
	case <-served:
	case <-time.After(3 * time.Second):
		t.Error("ListenAndServe did not return after Shutdown")
	}
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
