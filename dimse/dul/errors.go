package dul

import "fmt"

// StateError reports an unexpected event in the current DUL state: a protocol
// violation that drives the AA-8 path (send A-ABORT with provider source, issue
// A-P-ABORT, start ARTIM) rather than a panic or a silent socket close (Codex
// DIMSE-011). It names the state and event without PHI (PRD §8.2, §9.1). The upper
// layers wrap it into a public protocol error.
type StateError struct {
	State State
	Event Evt
}

func (e *StateError) Error() string {
	return fmt.Sprintf("dimse/dul: unexpected %s in %s (protocol error -> A-ABORT)", e.Event, e.State)
}
