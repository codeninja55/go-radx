package dimse

import "fmt"

// AssociationErrorKind classifies a failed or refused association.
type AssociationErrorKind uint8

const (
	// AssociationRejected is an A-ASSOCIATE-RJ: the peer refused the association.
	AssociationRejected AssociationErrorKind = iota
	// AssociationAborted is an A-ABORT during establishment (service-user source).
	AssociationAborted
	// AssociationProviderAborted is an A-P-ABORT during establishment (provider source).
	AssociationProviderAborted
	// AssociationTimeout is a negotiation that exceeded its deadline.
	AssociationTimeout
	// AssociationNotEstablished is an operation attempted on an association that was never
	// established or has already been released/aborted (Codex DIMSE-017).
	AssociationNotEstablished
)

var associationErrorKindNames = map[AssociationErrorKind]string{
	AssociationRejected:        "rejected",
	AssociationAborted:         "aborted",
	AssociationProviderAborted: "provider-aborted",
	AssociationTimeout:         "timeout",
	AssociationNotEstablished:  "not established",
}

// String renders the association-error kind by name (PRD §8.2).
func (k AssociationErrorKind) String() string {
	if name, ok := associationErrorKindNames[k]; ok {
		return name
	}
	return "unknown"
}

// AssociationError reports a failed or refused association: an A-ASSOCIATE-RJ (with source
// and reason), an A-ABORT during establishment, a negotiation timeout, or an operation on an
// unestablished/released association (Codex DIMSE-017). It names the kind, source, and reason
// without PHI; it never carries a patient value.
type AssociationError struct {
	Kind   AssociationErrorKind
	Source uint8 // PS3.8 rejection/abort source (0 when not applicable)
	Reason uint8 // PS3.8 reason (0 when not applicable)
	// Detail names the violated constraint for kinds without a wire source/reason.
	Detail string
}

func (e *AssociationError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("dimse: association %s: %s", e.Kind, e.Detail)
	}
	return fmt.Sprintf("dimse: association %s (source %d, reason %d)", e.Kind, e.Source, e.Reason)
}

// AbortError reports an A-ABORT (user) or A-P-ABORT (provider) on an established association.
type AbortError struct {
	Provider bool // true => A-P-ABORT (provider-initiated)
	Source   uint8
	Reason   uint8
}

func (e *AbortError) Error() string {
	kind := "A-ABORT"
	if e.Provider {
		kind = "A-P-ABORT"
	}
	return fmt.Sprintf("dimse: %s (source %d, reason %d)", kind, e.Source, e.Reason)
}

// ProtocolError reports a malformed PDU, a length-limit violation, or an unexpected PDU for
// the current DUL state. It names the state and the violated constraint without PHI;
// truncated or short reads wrap io.ErrUnexpectedEOF.
type ProtocolError struct {
	State State
	// Detail names the violated constraint without any patient value.
	Detail string
}

func (e *ProtocolError) Error() string {
	return fmt.Sprintf("dimse: protocol error in %s: %s", e.State, e.Detail)
}

// ValidationError reports an invalid value-type construction at the API boundary: an
// AE Title outside the 1..16 length or default character repertoire, a max-PDU length
// too small to carry a PDV, or a malformed presentation context. It is a fault the
// caller fixes (PRD §8.2, §9.2), distinct from the protocol/association faults that
// arise once a conversation is under way. It names the violated constraint without PHI.
type ValidationError struct {
	// Detail names the violated constraint without any patient value.
	Detail string
}

func (e *ValidationError) Error() string { return "dimse: " + e.Detail }
