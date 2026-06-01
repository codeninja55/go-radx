package dimse

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
