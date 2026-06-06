package hl7v2

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
)

func TestWriteFrameWrapsPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		want    []byte
	}{
		{
			name:    "empty payload",
			payload: nil,
			want:    []byte{StartBlock, EndBlock, CarriageReturn},
		},
		{
			name:    "short payload",
			payload: []byte("MSH|^~\\&"),
			want:    append([]byte{StartBlock}, append([]byte("MSH|^~\\&"), EndBlock, CarriageReturn)...),
		},
		{
			name:    "payload with embedded carriage returns",
			payload: []byte("MSH|^~\\&\rPID|1"),
			want:    append([]byte{StartBlock}, append([]byte("MSH|^~\\&\rPID|1"), EndBlock, CarriageReturn)...),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteFrame(&buf, tt.payload); err != nil {
				t.Fatalf("WriteFrame: %v", err)
			}
			if got := buf.Bytes(); !bytes.Equal(got, tt.want) {
				t.Fatalf("frame = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadFrameRoundTrip(t *testing.T) {
	payload := []byte("MSH|^~\\&|SEND|FAC|RECV|FAC|20240101010101||ADT^A01|MSGID|P|2.5.1\rPID|1")
	var buf bytes.Buffer
	if err := WriteFrame(&buf, payload); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}

	got, err := ReadFrame(context.Background(), &buf, DefaultMaxFrameSize)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload = %q, want %q", got, payload)
	}
}

func TestReadFrameRejectsMissingStartBlock(t *testing.T) {
	// A frame whose first byte is not the start block is a framing error, not
	// silently tolerated, because a peer that drops the start block has lost sync.
	src := bytes.NewReader([]byte{0x42, EndBlock, CarriageReturn})
	_, err := ReadFrame(context.Background(), src, DefaultMaxFrameSize)
	var fe *FrameError
	if !errors.As(err, &fe) {
		t.Fatalf("err = %v, want *FrameError", err)
	}
}

func TestReadFrameBoundsBeforeAllocation(t *testing.T) {
	// A payload that grows past the cap WITHOUT ever sending an end block must
	// fail at the cap rather than buffer without limit: the read stops the moment
	// the accumulated size would exceed max, before any unbounded growth.
	const max = 64
	hostile := make([]byte, 0, max+8)
	hostile = append(hostile, StartBlock)
	for len(hostile) < max+8 {
		hostile = append(hostile, 'A') // never an EndBlock
	}
	_, err := ReadFrame(context.Background(), bytes.NewReader(hostile), max)
	var fe *FrameError
	if !errors.As(err, &fe) {
		t.Fatalf("err = %v, want *FrameError for oversize frame", err)
	}
}

func TestReadFrameTruncatedIsUnexpectedEOF(t *testing.T) {
	// A start block followed by payload but no end block, then EOF, is truncation.
	src := bytes.NewReader(append([]byte{StartBlock}, []byte("MSH|^~\\&")...))
	_, err := ReadFrame(context.Background(), src, DefaultMaxFrameSize)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestReadFrameEOFBeforeStartBlock(t *testing.T) {
	// An empty stream is a clean io.EOF so a server read loop can stop without
	// treating a closed connection as a fault.
	_, err := ReadFrame(context.Background(), bytes.NewReader(nil), DefaultMaxFrameSize)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("err = %v, want io.EOF", err)
	}
}

func TestReadFrameContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// A blocking reader plus a cancelled context must return ctx.Err(), not hang.
	src := &blockingReader{}
	_, err := ReadFrame(ctx, src, DefaultMaxFrameSize)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestReadFrameZeroMaxUsesDefault(t *testing.T) {
	payload := []byte("MSH|^~\\&")
	var buf bytes.Buffer
	if err := WriteFrame(&buf, payload); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	got, err := ReadFrame(context.Background(), &buf, 0)
	if err != nil {
		t.Fatalf("ReadFrame with zero max: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload = %q, want %q", got, payload)
	}
}

// blockingReader blocks forever on Read, modelling a peer that has connected but
// sent nothing, so a context-cancellation test exercises the cancel path.
type blockingReader struct{}

func (blockingReader) Read(p []byte) (int, error) {
	select {} //nolint:staticcheck // intentional block to exercise ctx cancellation
}

// FuzzReadFrame drives the MLLP frame decoder with arbitrary bytes. Hostile
// input — a missing start block, a payload that never ends, embedded control
// bytes, premature EOF — must surface a typed error, never panic or allocate
// past the cap (PRD §9.3). The fuzz max is small so the run is fast and a
// would-be unbounded payload is rejected at the guard rather than buffered.
func FuzzReadFrame(f *testing.F) {
	wellFormed := func(payload []byte) []byte {
		var buf bytes.Buffer
		_ = WriteFrame(&buf, payload)
		return buf.Bytes()
	}
	f.Add(wellFormed([]byte("MSH|^~\\&")))
	f.Add(wellFormed(nil))
	f.Add([]byte{})                                 // empty stream: clean EOF
	f.Add([]byte{StartBlock})                       // start block then EOF: truncation
	f.Add([]byte{StartBlock, EndBlock})             // missing trailing CR
	f.Add([]byte{0x42, EndBlock, CarriageReturn})   // missing start block
	f.Add([]byte{StartBlock, StartBlock, EndBlock}) // nested control bytes

	const fuzzMax = 4096
	f.Fuzz(func(t *testing.T, data []byte) {
		// Must never panic; an error is the acceptable outcome for malformed input,
		// and a decoded payload must never exceed the cap.
		got, err := ReadFrame(context.Background(), bytes.NewReader(data), fuzzMax)
		if err == nil && len(got) > fuzzMax {
			t.Fatalf("decoded payload %d bytes exceeds cap %d", len(got), fuzzMax)
		}
	})
}
