package acse

import (
	"fmt"

	"github.com/codeninja55/go-radx/dimse/dul"
)

// RejectedError reports an A-ASSOCIATE-RJ: the peer refused the association with a result
// (permanent/transient), a source, and a reason (PS3.8 9.3.4). It names the codes without
// PHI. The root dimse package translates it into a public *AssociationError.
type RejectedError struct {
	Result uint8
	Source uint8
	Reason uint8
}

func (e *RejectedError) Error() string {
	return fmt.Sprintf("dimse/acse: association rejected (result %d, source %d, reason %d)",
		e.Result, e.Source, e.Reason)
}

// AbortedError reports an A-ABORT (or A-P-ABORT) received during establishment or on the
// association. Provider is true for a provider-initiated A-P-ABORT (PS3.8 9.3.8).
type AbortedError struct {
	Provider bool
	Source   uint8
	Reason   uint8
}

func (e *AbortedError) Error() string {
	kind := "A-ABORT"
	if e.Provider {
		kind = "A-P-ABORT"
	}
	return fmt.Sprintf("dimse/acse: %s (source %d, reason %d)", kind, e.Source, e.Reason)
}

// ProtocolError reports a malformed or unexpected PDU for the current DUL state, or a
// transport fault during negotiation. It names the state and the violated constraint
// without PHI (PRD §8.2, §9.1). The root dimse package wraps it into a public
// *ProtocolError carrying the same state.
type ProtocolError struct {
	State  dul.State
	Detail string
}

func (e *ProtocolError) Error() string {
	return fmt.Sprintf("dimse/acse: %s in %s", e.Detail, e.State)
}
