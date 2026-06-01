package pdu

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestBoundedReaderEnforcesRemaining(t *testing.T) {
	br := newBoundedReader(bytes.NewReader([]byte("abcdef")), 4)
	got := make([]byte, 4)
	if _, err := io.ReadFull(br, got); err != nil {
		t.Fatalf("ReadFull within bound: %v", err)
	}
	if string(got) != "abcd" {
		t.Errorf("read %q, want abcd", got)
	}
	// A further read past the declared bound is EOF, not the underlying "ef".
	if _, err := br.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Errorf("read past bound = %v, want io.EOF", err)
	}
}

func TestBoundedReaderRejectsOverLongAllocation(t *testing.T) {
	br := newBoundedReader(bytes.NewReader([]byte("ab")), 2)
	// A caller asking whether it may allocate n must consult Remaining first.
	if br.Remaining() != 2 {
		t.Fatalf("Remaining() = %d, want 2", br.Remaining())
	}
	if br.CanRead(3) {
		t.Error("CanRead(3) should be false when only 2 bytes remain")
	}
	if !br.CanRead(2) {
		t.Error("CanRead(2) should be true when 2 bytes remain")
	}
}
