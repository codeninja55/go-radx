package hl7v2

// AckCode is an HL7 Table 0008 acknowledgement code, the value of MSA-1. It is a
// closed enum: a negative acknowledgement is not a separate message type but an
// ACK whose AckCode rejects (reference doc "Acknowledgement codes"). The original
// acknowledgement mode uses AA/AE/AR; the enhanced mode adds the commit-level
// CA/CE/CR. The full BuildACK construction lives in the construction layer; the
// enum is defined here so MSA.AckCode is typed from the start.
type AckCode string

const (
	AckAccept AckCode = "AA" // Application Accept
	AckError  AckCode = "AE" // Application Error
	AckReject AckCode = "AR" // Application Reject

	AckCommitAccept AckCode = "CA" // Enhanced mode: Commit Accept
	AckCommitError  AckCode = "CE" // Enhanced mode: Commit Error
	AckCommitReject AckCode = "CR" // Enhanced mode: Commit Reject
)

// ParseAckCode validates s against HL7 Table 0008 and returns the typed code. An
// unrecognised code is a *ParseError naming the position, never the value, since
// MSA-1 is a control field rather than PHI.
func ParseAckCode(s string) (AckCode, error) {
	switch AckCode(s) {
	case AckAccept, AckError, AckReject, AckCommitAccept, AckCommitError, AckCommitReject:
		return AckCode(s), nil
	default:
		return "", &ParseError{Offset: 0, Reason: "MSA-1 is not a recognised acknowledgement code"}
	}
}

// IsPositive reports whether the code accepts (AA or CA).
func (c AckCode) IsPositive() bool { return c == AckAccept || c == AckCommitAccept }

// IsError reports whether the code signals an application/commit error (AE or CE).
func (c AckCode) IsError() bool { return c == AckError || c == AckCommitError }

// IsReject reports whether the code rejects (AR or CR).
func (c AckCode) IsReject() bool { return c == AckReject || c == AckCommitReject }
