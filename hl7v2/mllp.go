package hl7v2

import (
	"bufio"
	"context"
	"fmt"
	"io"
)

// MLLP framing bytes (Minimal Lower Layer Protocol). A frame is StartBlock,
// the message payload, EndBlock, then CarriageReturn, sent over a single TCP
// stream (HL7 v2 conformance "MLLP transport"). The bytes are control
// characters that never appear unescaped in a conformant message, so they
// delimit the payload without ambiguity.
const (
	StartBlock     byte = 0x0B // VT, marks the start of a frame
	EndBlock       byte = 0x1C // FS, marks the end of the payload
	CarriageReturn byte = 0x0D // CR, the byte that closes a frame after EndBlock
)

// DefaultMaxFrameSize caps the payload ReadFrame will accumulate before an
// EndBlock when the caller passes a non-positive max. It bounds the buffer a
// hostile or runaway peer can force a receiver to allocate (PRD §9.3): an
// 8 MiB message is already far larger than any realistic order or result, so
// the cap protects memory without rejecting legitimate traffic.
const DefaultMaxFrameSize = 8 << 20 // 8 MiB

// FrameError reports a malformed MLLP frame: a missing start block, a payload
// that exceeds the configured maximum before an end block, or a stream that ends
// mid-frame. It names the structural fault and the byte count involved, never any
// payload bytes, since an HL7 message routinely carries PHI (PRD §9.1).
type FrameError struct {
	Reason string // structural description, free of payload bytes
	err    error  // optional wrapped sentinel (e.g. io.ErrUnexpectedEOF on truncation)
}

func (e *FrameError) Error() string { return "hl7v2: mllp frame error: " + e.Reason }

// Unwrap exposes the wrapped sentinel so a caller can match a mid-frame
// truncation with errors.Is(err, io.ErrUnexpectedEOF) while errors.As still
// identifies the error as a *FrameError.
func (e *FrameError) Unwrap() error { return e.err }

// WriteFrame writes payload as a single MLLP frame to w: StartBlock, the
// payload bytes verbatim, EndBlock, then CarriageReturn. The payload is not
// inspected or escaped — it is the already-rendered message bytes — so a
// caller is responsible for passing a conformant message. A short write or a
// transport error is returned wrapped.
func WriteFrame(w io.Writer, payload []byte) error {
	frame := make([]byte, 0, len(payload)+3)
	frame = append(frame, StartBlock)
	frame = append(frame, payload...)
	frame = append(frame, EndBlock, CarriageReturn)
	if _, err := w.Write(frame); err != nil {
		return fmt.Errorf("hl7v2: write mllp frame: %w", err)
	}
	return nil
}

// ReadFrame reads one MLLP frame from br and returns its payload (the bytes
// between the start and end blocks). It is bounded and context-aware: the read
// stops at EndBlock or at max accumulated payload bytes BEFORE the buffer can
// grow without limit, so a peer that never sends an end block cannot drive an
// unbounded allocation (PRD §9.3). A non-positive max uses DefaultMaxFrameSize.
//
// br MUST persist across the frames of a connection. A persistent MLLP stream
// can carry frames back to back, and a bufio.Reader prefetches bytes belonging
// to the NEXT frame while decoding the current one; reusing the same reader
// keeps those prefetched bytes available for the following ReadFrame instead of
// discarding them with a per-call reader. The Client and Server each hold one
// *bufio.Reader per connection and pass it to successive calls for this reason.
//
// The error contract distinguishes the cases a caller must tell apart:
//   - io.EOF when the stream is empty before any byte (a closed connection,
//     which a server read loop treats as a clean stop, not a fault).
//   - io.ErrUnexpectedEOF (wrapped in a *FrameError) when the stream ends
//     mid-frame.
//   - *FrameError when the first byte is not StartBlock, or the payload reaches
//     max before an EndBlock.
//   - ctx.Err() when ctx is cancelled while a read blocks.
//
// ctx cancellation is honoured by scanning the frame on a single goroutine and
// selecting against ctx.Done(); on cancel ReadFrame returns ctx.Err() without
// waiting for the read to unblock. The scan goroutine drains into a buffered
// channel, so it finishes (it does not leak) once the read finally unblocks or
// the connection closes. A caller that needs the read itself interrupted at the
// socket should also set a read deadline on the connection, which the Server
// and Client do via their read timeouts.
func ReadFrame(ctx context.Context, br *bufio.Reader, max int) ([]byte, error) {
	if max <= 0 {
		max = DefaultMaxFrameSize
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	type result struct {
		payload []byte
		err     error
	}
	ch := make(chan result, 1)
	go func() {
		payload, err := scanFrame(br, max)
		ch <- result{payload, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		return res.payload, res.err
	}
}

// scanFrame reads one MLLP frame from br using the two-guard bounded read: it
// stops at the end block, or at max accumulated payload bytes, whichever comes
// first, so the buffer can never exceed max. It is the blocking core ReadFrame
// runs under a context select.
func scanFrame(br *bufio.Reader, max int) ([]byte, error) {
	first, err := br.ReadByte()
	if err != nil {
		// An empty stream before any byte is a clean EOF; ReadByte reports
		// io.EOF there, which the caller distinguishes from mid-frame truncation.
		return nil, err
	}
	if first != StartBlock {
		return nil, &FrameError{Reason: "frame did not begin with the start block"}
	}

	// Grow only while below the cap; the cap is checked BEFORE each append so the
	// buffer can never exceed max bytes.
	payload := make([]byte, 0, 256)
	for {
		b, err := br.ReadByte()
		if err != nil {
			if err == io.EOF {
				return nil, truncatedFrame("frame ended before the end block")
			}
			return nil, err
		}
		if b == EndBlock {
			// The end block must be followed by a carriage return to close the
			// frame; a trailing byte that is not CR is a framing fault.
			cr, err := br.ReadByte()
			if err != nil {
				if err == io.EOF {
					return nil, truncatedFrame("end block not followed by a carriage return")
				}
				return nil, err
			}
			if cr != CarriageReturn {
				return nil, &FrameError{Reason: "end block was not followed by a carriage return"}
			}
			return payload, nil
		}
		if len(payload) >= max {
			return nil, &FrameError{Reason: fmt.Sprintf("frame exceeded the maximum size of %d bytes before an end block", max)}
		}
		payload = append(payload, b)
	}
}

// truncatedFrame builds a *FrameError wrapping io.ErrUnexpectedEOF so a caller
// can match truncation with both errors.As(err, &fe *FrameError) and
// errors.Is(err, io.ErrUnexpectedEOF) while the message names only the
// structural fault.
func truncatedFrame(reason string) *FrameError {
	return &FrameError{Reason: reason, err: io.ErrUnexpectedEOF}
}
