package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
)

// TestDaemonDIMSERoundTrip wires the default filesystem object store, an in-memory SQLite catalogue,
// and the DIMSE SCP role into a runnable daemon, starts it on loopback, drives an in-process C-ECHO
// and C-STORE from a real DIMSE SCU, asserts the stored object round-trips through the backends, and
// shuts the daemon down cleanly. It is the end-to-end "a daemon is runnable out of the box" check.
func TestDaemonDIMSERoundTrip(t *testing.T) {
	t.Parallel()
	store, cat := newTestBackends(t)

	aet, err := dimse.ParseAETitle("RADX-SCP")
	if err != nil {
		t.Fatalf("ParseAETitle: %v", err)
	}
	dimseRole, err := NewDIMSERole(aet, store, cat, WithDIMSEPort(0)) // ":0" => OS-assigned port
	if err != nil {
		t.Fatalf("NewDIMSERole: %v", err)
	}
	d, err := New(WithDIMSE(dimseRole))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	waitForAddrs(t, d, "dimse")

	addr := d.Addrs()["dimse"]
	if addr == nil {
		t.Fatal("daemon reported no DIMSE address after start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	scu, err := dimse.NewAE(dimse.AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE (scu): %v", err)
	}
	contexts := echoStorageContexts()
	assoc, err := scu.Associate(ctx, addr.String(), aet, contexts)
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}

	echoStatus, err := assoc.Echo(ctx)
	if err != nil {
		t.Fatalf("Echo: %v", err)
	}
	if !echoStatus.IsSuccess() {
		t.Errorf("C-ECHO status = %v, want success", echoStatus)
	}

	const study = "1.2.840.113619.2.55.3.1"
	const series = "1.2.840.113619.2.55.3.2"
	const instance = "1.2.840.113619.2.55.3.3"
	sentDS := newTestObject(study, series, instance)

	storeStatus, err := assoc.Store(ctx, sentDS)
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	if !storeStatus.IsSuccess() {
		t.Errorf("C-STORE status = %v, want success", storeStatus)
	}
	_ = assoc.Release(ctx)

	// The object the SCP received is durably in the object store and indexed in the catalogue.
	got, err := store.Get(ctx, dicom.SOPInstanceUID(instance))
	if err != nil {
		t.Fatalf("object store Get after C-STORE: %v", err)
	}
	if v, _ := got.GetString(dicom.TagStudyInstanceUID); v != study {
		t.Errorf("stored StudyInstanceUID = %q, want %q", v, study)
	}
	rows := collect(t, cat.Query(ctx, CatalogueQuery{
		Level: dimse.QueryLevelStudy,
		Match: map[dicom.Tag]string{dicom.TagStudyInstanceUID: study},
	}))
	if len(rows) != 1 {
		t.Fatalf("catalogue query after C-STORE returned %d rows, want 1", len(rows))
	}

	// Cancelling Run drains every role cleanly under the race detector.
	cancelRun()
	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("daemon Run returned %v on clean shutdown, want nil", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("daemon did not shut down within 10s")
	}
}

// TestDaemonStartsLoopbackByDefault asserts the §11.2 bind-default sanity check: with default
// options, every role listens on loopback.
func TestDaemonStartsLoopbackByDefault(t *testing.T) {
	t.Parallel()
	store, cat := newTestBackends(t)
	aet, _ := dimse.ParseAETitle("RADX-SCP")
	dimseRole, _ := NewDIMSERole(aet, store, cat, WithDIMSEPort(0))
	webRole, _ := NewDICOMwebRole(store, cat, WithDICOMwebPort(0))

	d, err := New(WithDIMSE(dimseRole), WithDICOMweb(webRole))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	waitForAddrs(t, d, "dimse")
	waitForAddrs(t, d, "dicomweb")

	for role, addr := range d.Addrs() {
		host, _, err := net.SplitHostPort(addr.String())
		if err != nil {
			t.Fatalf("role %q addr %q not host:port: %v", role, addr, err)
		}
		if !isLoopbackHost(host) {
			t.Errorf("role %q bound non-loopback host %q by default", role, host)
		}
	}

	cancelRun()
	<-runErr
}

// rejectingAuth is an Authenticator that admits HTTP and admits exactly one Calling AE Title over
// DIMSE, rejecting every other association. It is the test seam proving the daemon actually runs the
// authenticator at the DIMSE association-accept layer rather than holding an unused reference.
type rejectingAuth struct {
	allowAE string
}

func (a rejectingAuth) AuthenticateHTTP(_ context.Context, _ *http.Request) (Principal, error) {
	return Principal{ID: "anonymous"}, nil
}

func (a rejectingAuth) AuthenticateDIMSE(_ context.Context, calling dimse.AETitle) (Principal, error) {
	if string(calling) != a.allowAE {
		return Principal{}, errors.New("calling AE not authorized")
	}
	return Principal{ID: string(calling)}, nil
}

// TestDaemonDIMSEAuthRejectsUnauthorizedCallingAE asserts the DIMSE authorization fix: a daemon on a
// non-loopback bind with an Authenticator that rejects a given Calling AE Title refuses that SCU's
// association, and accepts an authorized one. Without enforcement at the accept layer a remote SCU
// could C-ECHO/C-STORE/C-FIND with no credentials (Finding 1). The daemon binds 0.0.0.0 (a
// non-loopback host, which the bind policy permits only with an explicit Authenticator) and the SCU
// dials the loopback interface at the OS-assigned port.
func TestDaemonDIMSEAuthRejectsUnauthorizedCallingAE(t *testing.T) {
	t.Parallel()
	store, cat := newTestBackends(t)
	aet, err := dimse.ParseAETitle("RADX-SCP")
	if err != nil {
		t.Fatalf("ParseAETitle: %v", err)
	}
	dimseRole, err := NewDIMSERole(aet, store, cat, WithDIMSEPort(0))
	if err != nil {
		t.Fatalf("NewDIMSERole: %v", err)
	}
	// Non-loopback bind with an explicit Authenticator that admits only TRUSTED-SCU.
	d, err := New(WithDIMSE(dimseRole), WithBind("0.0.0.0"), WithAuthenticator(rejectingAuth{allowAE: "TRUSTED-SCU"}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	waitForAddrs(t, d, "dimse")
	defer func() {
		cancelRun()
		<-runErr
	}()

	bound := d.Addrs()["dimse"]
	if bound == nil {
		t.Fatal("daemon reported no DIMSE address after start")
	}
	_, port, err := net.SplitHostPort(bound.String())
	if err != nil {
		t.Fatalf("bound addr %q not host:port: %v", bound, err)
	}
	dialAddr := net.JoinHostPort("127.0.0.1", port)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// An unauthorized Calling AE Title is refused at negotiation (A-ASSOCIATE-RJ), before any service.
	rejected, err := dimse.NewAE(dimse.AETitle("EVIL-SCU"))
	if err != nil {
		t.Fatalf("NewAE (rejected): %v", err)
	}
	if _, err := rejected.Associate(ctx, dialAddr, aet, dimse.VerificationContexts()); err == nil {
		t.Fatal("Associate from an unauthorized Calling AE = nil error, want an association rejection")
	}

	// The authorized Calling AE Title establishes and can C-ECHO.
	trusted, err := dimse.NewAE(dimse.AETitle("TRUSTED-SCU"))
	if err != nil {
		t.Fatalf("NewAE (trusted): %v", err)
	}
	assoc, err := trusted.Associate(ctx, dialAddr, aet, dimse.VerificationContexts())
	if err != nil {
		t.Fatalf("Associate from the authorized Calling AE: %v", err)
	}
	echoStatus, err := assoc.Echo(ctx)
	if err != nil {
		t.Fatalf("Echo: %v", err)
	}
	if !echoStatus.IsSuccess() {
		t.Errorf("C-ECHO status = %v, want success", echoStatus)
	}
	_ = assoc.Release(ctx)
}

// echoStorageContexts builds an SCU proposal combining Verification and the validated Storage
// contexts with unique odd IDs, so the SCU can both C-ECHO and C-STORE on one association.
func echoStorageContexts() []dimse.PresentationContext {
	merged := append(dimse.VerificationContexts(), dimse.StorageContexts()...)
	id := uint8(1)
	for i := range merged {
		merged[i].ID = id
		id += 2
	}
	return merged
}
