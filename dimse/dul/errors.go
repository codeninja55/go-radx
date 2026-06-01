package dul

import "fmt"

// StateError reports an unexpected event in the current DUL state: a protocol
// violation that drives the AA-8 path (send A-ABORT with provider source, issue
// A-P-ABORT, start ARTIM) rather than a panic or a silent socket close (Codex
// DIMSE-011). It names the state and event without PHI (PRD §8.2, §9.1). The upper
// layers wrap it into a public protocol error.
//
// State is the state that RECEIVED the offending event (not the post-abort Sta13),
// so callers report where the violation occurred. Err carries the underlying cause
// for a malformed-PDU violation (the pdu codec error); it is nil for a well-formed
// but unexpected PDU.
type StateError struct {
	State State
	Event Evt
	Err   error
}

func (e *StateError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("dimse/dul: unexpected %s in %s (protocol error -> A-ABORT): %v", e.Event, e.State, e.Err)
	}
	return fmt.Sprintf("dimse/dul: unexpected %s in %s (protocol error -> A-ABORT)", e.Event, e.State)
}

func (e *StateError) Unwrap() error { return e.Err }
