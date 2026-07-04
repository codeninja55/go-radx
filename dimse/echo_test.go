package dimse

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dimse/acse"
	"github.com/codeninja55/go-radx/dimse/dul"
)

// scpResult captures what the SCP-side dispatch observed for a single inbound association.
type scpResult struct {
	status Status
	info   *OpInfo
	err    error
}

// startEchoSCP listens on loopback and, for each inbound association, negotiates the
// Verification context as acceptor, services one C-ECHO via serveEcho dispatching to h, then
// services the graceful release. It returns the listener address and a channel delivering the
// dispatch outcome. The listener closes on test cleanup.
func startEchoSCP(t *testing.T, called AETitle, h EchoHandler) (string, <-chan scpResult) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	results := make(chan scpResult, 1)
	go func() {
		nc, aerr := ln.Accept()
		if aerr != nil {
			return
		}
		conn := dul.NewConn(nc, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		acc, perr := acse.Accept(ctx, conn, acse.AcceptParams{
			CalledAETitle: string(called),
			MaxPDULength:  16382,
			Supported: []acse.SupportedContext{{
				AbstractSyntax:   string(verificationSOPClass),
				TransferSyntaxes: []string{"1.2.840.10008.1.2.1", "1.2.840.10008.1.2"},
			}},
		})
		if perr != nil {
			nc.Close()
			results <- scpResult{err: perr}
			return
		}
		status, serr := serveEcho(ctx, acc, AETitle("SCU"), called, h)
		results <- scpResult{status: status, info: handlerSeen(h), err: serr}
		_ = acc.ServeRelease(ctx)
	}()
	return ln.Addr().String(), results
}

// handlerSeen extracts the OpInfo a recording handler captured, if any.
func handlerSeen(h EchoHandler) *OpInfo {
	if rec, ok := h.(*recordingEchoHandler); ok {
		return rec.opInfo()
	}
	return nil
}

// recordingEchoHandler records the OpInfo it was dispatched with (thread-safe so the test
// goroutine can read what the SCP goroutine captured) and returns a configurable status.
type recordingEchoHandler struct {
	status Status
	mu     sync.Mutex
	seen   *OpInfo
}

func (h *recordingEchoHandler) Echo(_ context.Context, info OpInfo) Status {
	h.mu.Lock()
	h.seen = &info
	h.mu.Unlock()
	return h.status
}

func (h *recordingEchoHandler) opInfo() *OpInfo {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.seen
}

// TestEchoInProcessRoundTrip is the named regression: an in-process SCU C-ECHO over a loopback
// association returns StatusEchoSuccess, and the SCP dispatch returns StatusEchoSuccess too —
// the verification succeeds in both directions.
func TestEchoInProcessRoundTrip(t *testing.T) {
	h := &recordingEchoHandler{status: StatusEchoSuccess}
	addr, results := startEchoSCP(t, AETitle("SCP"), h)

	ae, err := NewAE(AETitle("SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	assoc, err := ae.Associate(ctx, addr, AETitle("SCP"), VerificationContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}

	status, err := assoc.Echo(ctx)
	if err != nil {
		t.Fatalf("Echo transport error: %v", err)
	}
	if !status.IsSuccess() {
		t.Errorf("SCU Echo status = %s, want success", status)
	}
	if status.Code != StatusEchoSuccess.Code {
		t.Errorf("SCU Echo status code = %#04x, want %#04x", status.Code, StatusEchoSuccess.Code)
	}

	if err := assoc.Release(ctx); err != nil {
		t.Fatalf("Release: %v", err)
	}

	res := <-results
	if res.err != nil {
		t.Fatalf("SCP serveEcho: %v", res.err)
	}
	if !res.status.IsSuccess() {
		t.Errorf("SCP dispatch status = %s, want success", res.status)
	}
	if res.info == nil {
		t.Fatal("SCP handler was not dispatched (no OpInfo recorded)")
	}
	if res.info.MessageID != echoMessageID {
		t.Errorf("OpInfo.MessageID = %d, want %d", res.info.MessageID, echoMessageID)
	}
	if res.info.SOPClassUID != verificationSOPClass {
		t.Errorf("OpInfo.SOPClassUID = %q, want Verification", res.info.SOPClassUID)
	}
	if res.info.CallingAETitle != "SCU" || res.info.CalledAETitle != "SCP" {
		t.Errorf("OpInfo AE titles = (%q,%q), want (SCU,SCP)", res.info.CallingAETitle, res.info.CalledAETitle)
	}
}

// TestEchoPropagatesNonSuccessStatus confirms a handler returning a non-success status is
// returned verbatim to the SCU (status is data, not laundered to success): the SCU sees the
// peer's failure code with its category preserved.
func TestEchoPropagatesNonSuccessStatus(t *testing.T) {
	h := &recordingEchoHandler{status: NewStatus(0x0122, ServiceClassVerification)} // SOP Class Not Supported
	addr, results := startEchoSCP(t, AETitle("SCP"), h)

	ae, _ := NewAE(AETitle("SCU"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	assoc, err := ae.Associate(ctx, addr, AETitle("SCP"), VerificationContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}

	status, err := assoc.Echo(ctx)
	if err != nil {
		t.Fatalf("Echo transport error: %v", err)
	}
	if !status.IsFailure() {
		t.Errorf("SCU Echo status = %s, want failure", status)
	}
	if status.Code != 0x0122 {
		t.Errorf("SCU Echo status code = %#04x, want 0x0122", status.Code)
	}

	_ = assoc.Release(ctx)
	<-results
}

// TestEchoOnReleasedAssociationReturnsTypedError is the DIMSE-017 guard for Echo: calling it on
// a released association returns a typed *AssociationError, never a panic. The released-state
// check fails closed before any wire I/O, so the SCP need only negotiate and serve the release.
func TestEchoOnReleasedAssociationReturnsTypedError(t *testing.T) {
	addr := startReleaseOnlyAcceptor(t, AETitle("SCP"))

	ae, _ := NewAE(AETitle("SCU"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	assoc, err := ae.Associate(ctx, addr, AETitle("SCP"), VerificationContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	if err := assoc.Release(ctx); err != nil {
		t.Fatalf("Release: %v", err)
	}

	_, err = assoc.Echo(ctx)
	if err == nil {
		t.Fatal("Echo on a released association = nil error, want a typed error")
	}
	if _, ok := errors.AsType[*AssociationError](err); !ok {
		t.Errorf("Echo-on-released error = %T, want *AssociationError", err)
	}
}

// startReleaseOnlyAcceptor listens on loopback and, for each inbound association, negotiates the
// Verification context and immediately services one graceful release — no DIMSE dispatch. It is
// the minimal acceptor for tests that release without issuing an operation.
func startReleaseOnlyAcceptor(t *testing.T, called AETitle) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		nc, aerr := ln.Accept()
		if aerr != nil {
			return
		}
		conn := dul.NewConn(nc, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		acc, perr := acse.Accept(ctx, conn, verificationAcceptParams(called))
		if perr != nil {
			nc.Close()
			return
		}
		_ = acc.ServeRelease(ctx)
	}()
	return ln.Addr().String()
}

// verificationAcceptParams is the acceptor negotiation config for the Verification SOP Class,
// shared by the test acceptors.
func verificationAcceptParams(called AETitle) acse.AcceptParams {
	return acse.AcceptParams{
		CalledAETitle: string(called),
		MaxPDULength:  16382,
		Supported: []acse.SupportedContext{{
			AbstractSyntax:   string(verificationSOPClass),
			TransferSyntaxes: []string{"1.2.840.10008.1.2.1", "1.2.840.10008.1.2"},
		}},
	}
}

// TestEchoHonoursDIMSETimeout is the regression for the DIMSE-timeout fix: when the peer accepts
// the association but never sends the C-ECHO-RSP, Echo must return bounded by the configured
// DIMSE timeout, not block until the caller's (longer) context expires.
func TestEchoHonoursDIMSETimeout(t *testing.T) {
	addr := startStallingAcceptor(t, AETitle("SCP"))

	ae, err := NewAE(AETitle("SCU"), WithDIMSETimeout(200*time.Millisecond))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	assoc, err := ae.Associate(ctx, addr, AETitle("SCP"), VerificationContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}

	start := time.Now()
	_, err = assoc.Echo(ctx)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("Echo against a non-responding peer = nil error, want a timeout")
	}
	// The 200ms DIMSE timeout must fire well before the caller's 3s context; pre-fix, Echo used
	// the raw context and only returned at ~3s.
	if elapsed > 1500*time.Millisecond {
		t.Errorf("Echo took %v; want it bounded by the 200ms DIMSE timeout (configured timeout ignored?)", elapsed)
	}
}

// startStallingAcceptor negotiates the association then never answers any DIMSE request, holding
// the connection open, so an SCU operation must rely on its own DIMSE timeout.
func startStallingAcceptor(t *testing.T, called AETitle) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		nc, aerr := ln.Accept()
		if aerr != nil {
			return
		}
		defer nc.Close()
		conn := dul.NewConn(nc, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, perr := acse.Accept(ctx, conn, verificationAcceptParams(called)); perr != nil {
			return
		}
		<-ctx.Done() // negotiated, then deliberately silent: never answer the C-ECHO
	}()
	return ln.Addr().String()
}

// TestEchoRejectsResponseWithoutStatus is the regression for the mandatory-Status fix: a
// C-ECHO-RSP missing the Status element is malformed and must surface as a protocol error, never
// be laundered into 0x0000 Success.
func TestEchoRejectsResponseWithoutStatus(t *testing.T) {
	addr := startNoStatusEchoAcceptor(t, AETitle("SCP"))

	ae, _ := NewAE(AETitle("SCU"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	assoc, err := ae.Associate(ctx, addr, AETitle("SCP"), VerificationContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	_, err = assoc.Echo(ctx)
	if err == nil {
		t.Fatal("Echo on a C-ECHO-RSP without Status = nil error, want a protocol error (not success)")
	}
	if _, ok := errors.AsType[*ProtocolError](err); !ok {
		t.Errorf("error = %T, want *ProtocolError", err)
	}
}

// startNoStatusEchoAcceptor negotiates, reads the C-ECHO-RQ, then replies with a C-ECHO-RSP that
// omits the mandatory Status element — a non-conformant response.
func startNoStatusEchoAcceptor(t *testing.T, called AETitle) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		nc, aerr := ln.Accept()
		if aerr != nil {
			return
		}
		defer nc.Close()
		conn := dul.NewConn(nc, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		acc, perr := acse.Accept(ctx, conn, verificationAcceptParams(called))
		if perr != nil {
			return
		}
		cmd, pcID, rerr := receiveCommand(ctx, acc.Conn(), acc.Machine())
		if rerr != nil {
			return
		}
		_ = sendCommand(ctx, acc.Conn(), acc.Machine(), pcID, CommandSet{
			CommandField:              CommandCEchoRSP,
			MessageIDBeingRespondedTo: cmd.MessageID,
			AffectedSOPClassUID:       cmd.AffectedSOPClassUID,
			CommandDataSetType:        CommandDataSetNotPresent,
			HasStatus:                 false, // the malformed bit: no Status element
		})
		_ = acc.ServeRelease(ctx)
	}()
	return ln.Addr().String()
}
