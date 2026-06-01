package dul

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/codeninja55/go-radx/dimse/pdu"
)

// Conn is the socket owner at the DUL layer: it wraps a net.Conn, frames PDUs through
// the pdu codec, and advances the PS3.8 state machine. It owns the ARTIM timer. It knows
// only PDUs, never DICOM messages, and imports only the pdu package and the standard
// library, keeping the layering acyclic (dimse.md "Overview of the layers").
//
// A single goroutine drives the machine; reads and writes are serialised by separate
// mutexes so a reader and a writer may run concurrently (full-duplex) but two writers
// (or two readers) cannot interleave a half-framed PDU. context.Context cancels a blocked
// read or write by setting a past deadline on the connection.
type Conn struct {
	nc      net.Conn
	machine *StateMachine
	artim   *artim

	writeMu sync.Mutex
	readMu  sync.Mutex
}

// NewConn wraps a net.Conn. artimTimeout configures the ARTIM timer; zero disables it.
// The machine starts in Sta1; the association layer drives it to the appropriate state.
func NewConn(nc net.Conn, artimTimeout time.Duration) *Conn {
	return &Conn{
		nc:      nc,
		machine: NewStateMachine(),
		artim:   newARTIM(artimTimeout),
	}
}

// State reports the current DUL state (for observability).
func (c *Conn) State() State { return c.machine.CurrentState() }

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
	stop := c.watchContext(ctx)
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
	stop := c.watchContext(ctx)
	defer stop()

	p, err := pdu.ReadPDU(c.nc)
	if err != nil {
		return nil, c.translateDeadline(ctx, err)
	}
	return p, nil
}

// driveOnce reads one PDU and advances the state machine by the event it represents. On
// a successful read it maps the PDU to its received-event and applies it. On a decode
// error it raises Evt19 (invalid PDU received), which the FSM turns into the AA-8 path; it
// then sends the provider-source A-ABORT the standard requires, so an invalid PDU never
// causes a silent socket close (Codex DIMSE-011). It returns the action the FSM selected,
// the decoded PDU (nil on a decode error), and any error.
func (c *Conn) driveOnce(ctx context.Context) (Action, pdu.PDU, error) {
	p, readErr := c.ReadPDU(ctx)
	if readErr != nil {
		// A context or transport error is not a protocol violation; surface it directly.
		if errors.Is(readErr, context.Canceled) || errors.Is(readErr, context.DeadlineExceeded) {
			return ActionNone, nil, readErr
		}
		// Any other read error is treated as an invalid/unrecognised PDU: Evt19.
		action, _, smErr := c.machine.Apply(Evt19)
		c.performAbortAction(ctx, action)
		// Prefer the protocol error from the machine; fall back to the read error.
		if smErr != nil {
			return action, nil, smErr
		}
		return action, nil, readErr
	}

	event := pduToEvent(p)
	action, _, smErr := c.machine.Apply(event)
	if smErr != nil {
		// An unexpected (but well-formed) PDU also drives AA-8; send the A-ABORT.
		c.performAbortAction(ctx, action)
	}
	return action, p, smErr
}

// performAbortAction sends the A-ABORT a fault action requires. AA-8 and AA-1 send a
// provider-source / user-source A-ABORT respectively; AA-7 sends an A-ABORT during the
// closing state. Other actions send nothing here (the association layer performs the
// non-abort sends, which need association state the DUL does not hold).
func (c *Conn) performAbortAction(ctx context.Context, action Action) {
	switch action {
	case AA8, AA7:
		_ = c.WritePDU(ctx, &pdu.Abort{
			Source: pdu.AbortSourceServiceProvider,
			Reason: pdu.AbortReasonUnrecognizedPDU,
		})
		c.artim.start()
	case AA1:
		_ = c.WritePDU(ctx, &pdu.Abort{
			Source: pdu.AbortSourceServiceUser,
			Reason: pdu.AbortReasonNotSpecified,
		})
		c.artim.start()
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
// connection so a blocked Read/Write returns immediately. The returned stop function
// tears the watcher down and clears the deadline.
func (c *Conn) watchContext(ctx context.Context) func() {
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			// A past deadline interrupts an in-flight Read or Write on the net.Conn.
			_ = c.nc.SetDeadline(time.Unix(1, 0))
		case <-done:
		}
	}()
	return func() {
		close(done)
		_ = c.nc.SetDeadline(time.Time{})
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
