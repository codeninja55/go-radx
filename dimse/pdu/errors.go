package pdu

// PDUError reports a malformed PDU or PDV: a length-limit violation, an underflow
// item length, or a truncated read. It names the violated constraint without PHI.
// The root dimse package translates it into a public *ProtocolError; pdu must not
// import dimse (acyclic layering, dimse.md "Overview of the layers").
type PDUError struct {
	Detail string
}

func (e *PDUError) Error() string { return "dimse/pdu: " + e.Detail }

// EncodeError reports a PDU that cannot be serialised conformantly: a length-prefixed field whose
// byte length exceeds the 2-byte (uint16) length prefix the wire format reserves for it. Truncating
// such a length silently would emit a corrupt PDU whose nested item lengths disagree with the bytes
// that follow, so the encoder refuses rather than producing one. It names the violated field without
// PHI. The root dimse package translates it into a public error; pdu must not import dimse (acyclic
// layering, dimse.md "Overview of the layers").
type EncodeError struct {
	Detail string
}

func (e *EncodeError) Error() string { return "dimse/pdu: " + e.Detail }
