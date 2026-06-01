package pdu

// PDUError reports a malformed PDU or PDV: a length-limit violation, an underflow
// item length, or a truncated read. It names the violated constraint without PHI.
// The root dimse package translates it into a public *ProtocolError; pdu must not
// import dimse (acyclic layering, dimse.md "Overview of the layers").
type PDUError struct {
	Detail string
}

func (e *PDUError) Error() string { return "dimse/pdu: " + e.Detail }
