package dul

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/codeninja55/go-radx/dimse/pdu"
)

// Conn is the socket owner at the DUL layer: it wraps a net.Conn and frames PDUs through
// the pdu codec. It owns the ARTIM timer. It performs no state-machine logic of its own:
// the association layer owns the StateMachine and drives inbound reads through DriveInbound,
// which shares the PS3.8 hardening across every consumer. Conn knows only PDUs, never DICOM
// messages, and imports only the pdu package and the standard library, keeping the layering
// acyclic (dimse.md "Overview of the layers").
//
// Reads and writes are serialised by separate mutexes so a reader and a writer may run
// concurrently (full-duplex) but two writers (or two readers) cannot interleave a half-framed
// PDU. context.Context cancels a blocked read or write by setting a past deadline on the
// connection.
type Conn struct {
	nc    net.Conn
	artim *artim

	writeMu sync.Mutex
	readMu  sync.Mutex
}

// NewConn wraps a net.Conn. artimTimeout configures the ARTIM timer; zero disables it.
func NewConn(nc net.Conn, artimTimeout time.Duration) *Conn {
	return &Conn{
		nc:    nc,
		artim: newARTIM(artimTimeout),
	}
}

// Close closes the underlying transport and stops the ARTIM timer.
func (c *Conn) Close() error {
	c.artim.stop()
	return c.nc.Close()
}

// WritePDU encodes one PDU onto the connection, bounded by ctx: if ctx is cancelled the
// write is interrupted and ctx's error is returned. Writes are serialised so a PDU is
// never interleaved with another on the wire.
func (c *Conn) WritePDU(ctx context.Context, p pdu.PDU) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	stop := c.watchContext(ctx, c.nc.SetWriteDeadline)
	defer stop()

	if err := pdu.WritePDU(c.nc, p); err != nil {
		return c.translateDeadline(ctx, fmt.Errorf("dimse/dul: write %s: %w", p.Type(), err))
	}
	return nil
}

// ReadPDU reads one complete PDU from the connection, bounded by ctx. A clean transport
// close surfaces as io.EOF; a malformed PDU surfaces as the pdu codec's error (the caller
// turns that into Evt19). Reads are serialised.
func (c *Conn) ReadPDU(ctx context.Context) (pdu.PDU, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	stop := c.watchContext(ctx, c.nc.SetReadDeadline)
	defer stop()

	p, err := pdu.ReadPDU(c.nc)
	if err != nil {
		return nil, c.translateDeadline(ctx, err)
	}
	return p, nil
}

// DriveInbound reads one PDU on conn and advances the caller's state machine m by the event
// the read represents, sharing the PS3.8 inbound hardening across every consumer (acse drives
// its own StateMachine through this one path). On a successful read it maps the PDU to its
// received-event and applies it. A clean io.EOF at a PDU boundary (the peer closed the socket
// in an orderly way) raises Evt17 (transport connection closed), an orderly close that sends
// no A-ABORT; a genuinely malformed or unrecognised PDU raises Evt19 (invalid PDU received),
// which the FSM turns into the AA-8 path that sends the provider-source A-ABORT the standard
// requires, so an invalid PDU never causes a silent socket close (Codex DIMSE-011).
//
// It returns the decoded PDU (nil on a read error), the action the FSM selected, and the
// error: a context error or io.EOF verbatim, or a typed *StateError when the event was a
// protocol violation (else nil). A recognised PDU that is unexpected in the current state is a
// violation whether the Table 9-10 cell is undefined (Apply returns the *StateError) or an
// explicit fault-abort cell (e.g. Sta6 + Evt6 -> AA-8, which Apply returns with a nil error);
// in both cases the provider A-ABORT is sent and a *StateError is returned so the caller never
// mistakes the aborting PDU for a valid response. m must be the StateMachine the caller
// advances for this connection so the inbound transition stays consistent with the caller's
// local-primitive transitions.
func DriveInbound(ctx context.Context, conn *Conn, m *StateMachine) (pdu.PDU, Action, error) {
	p, readErr := conn.ReadPDU(ctx)
	if readErr != nil {
		// A context error is not a protocol violation; surface it directly, no FSM change.
		if errors.Is(readErr, context.Canceled) || errors.Is(readErr, context.DeadlineExceeded) {
			return nil, ActionNone, readErr
		}
		// A clean io.EOF at a PDU boundary is an orderly transport close: Evt17, not Evt19,
		// and never an A-ABORT.
		if errors.Is(readErr, io.EOF) {
			action, _, smErr := m.Apply(Evt17)
			if smErr != nil {
				return nil, action, smErr
			}
			return nil, action, readErr
		}
		// Any other read error is treated as an invalid/unrecognised PDU: Evt19.
		action, _, smErr := m.Apply(Evt19)
		sendFaultAbort(ctx, conn, action, pdu.AbortReasonUnrecognizedPDU)
		// Prefer the protocol error from the machine; fall back to the read error.
		if smErr != nil {
			return nil, action, smErr
		}
		return nil, action, readErr
	}

	event := pduToEvent(p)
	from := m.CurrentState()
	action, _, smErr := m.Apply(event)
	if isFaultAbort(action) {
		// A recognised-but-unexpected PDU drives AA-8 (or AA-7/AA-1); the provider abort reason
		// is "unexpected PDU", distinct from a malformed PDU's "unrecognised". Apply may return
		// this with a nil error (an explicit fault-abort cell); synthesise the *StateError so
		// the violation surfaces consistently.
		sendFaultAbort(ctx, conn, action, pdu.AbortReasonUnexpectedPDU)
		if smErr == nil {
			smErr = &StateError{State: from, Event: event}
		}
	}
	return p, action, smErr
}

// isFaultAbort reports whether an action is one of the abort actions sendFaultAbort writes an
// A-ABORT for: AA-8 and AA-7 (provider source) and AA-1 (user source).
func isFaultAbort(action Action) bool {
	switch action {
	case AA8, AA7, AA1:
		return true
	default:
		return false
	}
}

// sendFaultAbort sends the A-ABORT a fault action requires. AA-8 and AA-7 send a
// provider-source A-ABORT carrying providerReason (which distinguishes a malformed PDU
// from a well-formed-but-unexpected one); AA-1 sends a user-source A-ABORT. Other actions
// send nothing here (the association layer performs the non-abort sends, which need
// association state the DUL does not hold). Conn still owns the ARTIM, so the abort arms it.
func sendFaultAbort(ctx context.Context, conn *Conn, action Action, providerReason uint8) {
	switch action {
	case AA8, AA7:
		_ = conn.WritePDU(ctx, &pdu.Abort{
			Source: pdu.AbortSourceServiceProvider,
			Reason: providerReason,
		})
		conn.artim.start()
	case AA1:
		_ = conn.WritePDU(ctx, &pdu.Abort{
			Source: pdu.AbortSourceServiceUser,
			Reason: pdu.AbortReasonNotSpecified,
		})
		conn.artim.start()
	}
}

// pduToEvent maps a received PDU to the PS3.8 event it raises in the state machine.
func pduToEvent(p pdu.PDU) Evt {
	switch p.Type() {
	case pdu.PDUTypeAssociateRQ:
		return Evt6
	case pdu.PDUTypeAssociateAC:
		return Evt3
	case pdu.PDUTypeAssociateRJ:
		return Evt4
	case pdu.PDUTypeData:
		return Evt10
	case pdu.PDUTypeReleaseRQ:
		return Evt12
	case pdu.PDUTypeReleaseRP:
		return Evt13
	case pdu.PDUTypeAbort:
		return Evt16
	default:
		return Evt19
	}
}

// watchContext arms a goroutine that, on ctx cancellation, sets a past deadline on the
// connection so a blocked Read/Write returns immediately. setDeadline is the
// direction-specific setter (SetReadDeadline on the read path, SetWriteDeadline on the
// write path) so cancelling a read cannot disturb an in-flight write and vice versa: the
// connection is full-duplex and each direction's cancellation is independent. The returned
// stop function tears the watcher down and clears that direction's deadline.
func (c *Conn) watchContext(ctx context.Context, setDeadline func(time.Time) error) func() {
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		select {
		case <-ctx.Done():
			// A past deadline interrupts an in-flight Read or Write on the net.Conn.
			_ = setDeadline(time.Unix(1, 0))
		case <-done:
		}
	}()
	return func() {
		close(done)
		// Wait for the watcher to return before clearing the deadline. Otherwise a watcher
		// that observed ctx.Done() at about the same time the operation completed could set
		// a past deadline AFTER this clear, stranding a stale deadline that fails the next
		// operation under a fresh context with a spurious timeout.
		<-finished
		_ = setDeadline(time.Time{})
	}
}

// translateDeadline converts the deadline-driven interruption from watchContext back into
// the originating context error, so callers see context.Canceled / DeadlineExceeded
// rather than an opaque timeout.
func (c *Conn) translateDeadline(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			return ctxErr
		}
		if errors.Is(err, os.ErrDeadlineExceeded) {
			return ctxErr
		}
	}
	return err
}
