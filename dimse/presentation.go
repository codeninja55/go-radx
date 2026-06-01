package dimse

import "github.com/codeninja55/go-radx/dicom"

// MaxPDULength is the negotiated largest acceptable P-DATA-TF PDU (PS3.7 Annex D.1). A
// value of 0 means "no maximum specified" — unlimited — and is represented as unlimited,
// never as a literal allocation size (Codex DIMSE-005). Callers resolve the size to send
// with SendCap, which treats 0 as the configured local send cap rather than a zero or
// negative slice bound.
type MaxPDULength uint32

// IsUnlimited reports whether the peer specified no maximum PDU length (the value 0).
func (m MaxPDULength) IsUnlimited() bool { return m == 0 }

// SendCap resolves the number of P-DATA-TF body bytes that may be sent to a peer whose
// advertised maximum is m, given the local send cap localCap. An unlimited peer max
// (m == 0) resolves to localCap, never 0 (Codex DIMSE-005); otherwise the smaller of the
// peer max and the local cap wins, so a PDU never exceeds either party's limit.
func (m MaxPDULength) SendCap(localCap MaxPDULength) MaxPDULength {
	if m.IsUnlimited() {
		return localCap
	}
	if localCap != 0 && localCap < m {
		return localCap
	}
	return m
}

// ContextResult is the outcome of negotiating one presentation context (PS3.8 9.3.3.2).
// Its values match the wire result/reason codes so translation to and from the pdu layer
// is a plain conversion.
type ContextResult uint8

const (
	ContextAccepted                     ContextResult = iota // 0: acceptance
	ContextUserRejected                                      // 1: user rejection
	ContextNoReason                                          // 2: no reason (provider rejection)
	ContextAbstractSyntaxNotSupported                        // 3: abstract syntax not supported
	ContextTransferSyntaxesNotSupported                      // 4: transfer syntaxes not supported
)

var contextResultNames = map[ContextResult]string{
	ContextAccepted:                     "accepted",
	ContextUserRejected:                 "user rejection",
	ContextNoReason:                     "no reason (provider rejection)",
	ContextAbstractSyntaxNotSupported:   "abstract syntax not supported",
	ContextTransferSyntaxesNotSupported: "transfer syntaxes not supported",
}

// String renders the negotiation outcome by name, never a bare code (PRD §8.2).
func (r ContextResult) String() string {
	if name, ok := contextResultNames[r]; ok {
		return name
	}
	return "unknown context result"
}

// PresentationContext is a proposed or accepted presentation context: one Abstract Syntax
// (a SOP Class) paired with one or more Transfer Syntaxes, keyed by an odd context ID
// (PS3.8 9.3.2.2). On an accepted context TransferSyntaxes holds the single accepted
// syntax and Result reports the negotiation outcome.
type PresentationContext struct {
	ID               uint8                  // odd, 1..255
	AbstractSyntax   dicom.SOPClassUID      // the SOP Class proposed
	TransferSyntaxes []dicom.TransferSyntax // proposed list (RQ) or single accepted (AC)
	Result           ContextResult          // meaningful on an accepted context
}

// DefaultTransferSyntaxes is the proposal-preference order of the four uncompressed and
// deflated syntaxes the core library reads and writes (PRD §6.2 DICOM floor). Explicit VR
// Little Endian leads because the acceptor may pick the first acceptable syntax. This
// literal is identical to the one in docs/conformance/dicom.md.
var DefaultTransferSyntaxes = []dicom.TransferSyntax{
	dicom.ExplicitVRLittleEndian,         // 1.2.840.10008.1.2.1
	dicom.ImplicitVRLittleEndian,         // 1.2.840.10008.1.2
	dicom.DeflatedExplicitVRLittleEndian, // 1.2.840.10008.1.2.1.99
	dicom.ExplicitVRBigEndian,            // 1.2.840.10008.1.2.2 (retired)
}

// NewPresentationContext builds a proposal for one SOP Class with the given transfer
// syntaxes, keyed by id. When ts is empty the proposed set is DefaultTransferSyntaxes. The
// transfer-syntax slice is copied so the context never shares the caller's (or the
// package-level default's) backing array.
func NewPresentationContext(id uint8, abstract dicom.SOPClassUID, ts ...dicom.TransferSyntax) PresentationContext {
	source := ts
	if len(source) == 0 {
		source = DefaultTransferSyntaxes
	}
	syntaxes := make([]dicom.TransferSyntax, len(source))
	copy(syntaxes, source)
	return PresentationContext{
		ID:               id,
		AbstractSyntax:   abstract,
		TransferSyntaxes: syntaxes,
	}
}
