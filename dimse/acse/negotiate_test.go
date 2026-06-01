package acse

import (
	"testing"

	"github.com/codeninja55/go-radx/dimse/pdu"
)

const (
	verificationUID = "1.2.840.10008.1.1"
	ctImageUID      = "1.2.840.10008.5.1.4.1.1.2"
	explicitVRLE    = "1.2.840.10008.1.2.1"
	implicitVRLE    = "1.2.840.10008.1.2"
	deflatedVRLE    = "1.2.840.10008.1.2.1.99"
)

// supportedVerification is one acceptor-side supported context: the Verification SOP Class
// accepting the two common uncompressed syntaxes, Explicit VR LE preferred.
func supportedVerification() []SupportedContext {
	return []SupportedContext{{
		AbstractSyntax:   verificationUID,
		TransferSyntaxes: []string{explicitVRLE, implicitVRLE},
	}}
}

// TestNegotiateAcceptsMatchingContext verifies a proposed context whose abstract syntax is
// supported and whose transfer-syntax list overlaps the supported set is accepted, with the
// acceptor's preferred (first supported) transfer syntax selected.
func TestNegotiateAcceptsMatchingContext(t *testing.T) {
	proposed := []pdu.PresentationContextRQ{{
		ID:               1,
		AbstractSyntax:   verificationUID,
		TransferSyntaxes: []string{implicitVRLE, explicitVRLE},
	}}
	got := NegotiateAcceptor(proposed, supportedVerification())
	if len(got) != 1 {
		t.Fatalf("NegotiateAcceptor returned %d results, want 1", len(got))
	}
	r := got[0]
	if r.ID != 1 {
		t.Errorf("result ID = %d, want 1 (the proposed ID is echoed)", r.ID)
	}
	if r.Result != pdu.PresentationContextAcceptance {
		t.Errorf("result = %d, want acceptance (%d)", r.Result, pdu.PresentationContextAcceptance)
	}
	// The acceptor prefers its own first supported transfer syntax (Explicit VR LE) over
	// the requestor's first proposal (Implicit VR LE), since both are offered.
	if r.TransferSyntax != explicitVRLE {
		t.Errorf("accepted transfer syntax = %q, want %q (acceptor preference wins)", r.TransferSyntax, explicitVRLE)
	}
}

// TestNegotiateRejectsUnsupportedAbstractSyntax is the named regression: a proposed
// abstract syntax the acceptor does not support is rejected with
// ContextAbstractSyntaxNotSupported (result 3), and the rejected result still carries
// exactly one (insignificant) transfer-syntax sub-item for the AC PDU (Codex DIMSE-008).
func TestNegotiateRejectsUnsupportedAbstractSyntax(t *testing.T) {
	proposed := []pdu.PresentationContextRQ{{
		ID:               3,
		AbstractSyntax:   ctImageUID, // the acceptor supports only Verification
		TransferSyntaxes: []string{explicitVRLE, implicitVRLE},
	}}
	got := NegotiateAcceptor(proposed, supportedVerification())
	if len(got) != 1 {
		t.Fatalf("NegotiateAcceptor returned %d results, want 1", len(got))
	}
	r := got[0]
	if r.ID != 3 {
		t.Errorf("result ID = %d, want 3", r.ID)
	}
	if r.Result != pdu.PresentationContextAbstractSyntaxNotSupported {
		t.Errorf("result = %d, want abstract-syntax-not-supported (%d)",
			r.Result, pdu.PresentationContextAbstractSyntaxNotSupported)
	}
	if r.TransferSyntax == "" {
		t.Error("rejected context has no transfer syntax; PS3.8 9.3.3.2 requires one insignificant sub-item (DIMSE-008)")
	}
}

// TestNegotiateRejectsUnsupportedTransferSyntaxes verifies a supported abstract syntax with
// no overlapping transfer syntax is rejected with ContextTransferSyntaxesNotSupported.
func TestNegotiateRejectsUnsupportedTransferSyntaxes(t *testing.T) {
	proposed := []pdu.PresentationContextRQ{{
		ID:               1,
		AbstractSyntax:   verificationUID,
		TransferSyntaxes: []string{deflatedVRLE}, // the acceptor offers only explicit/implicit LE
	}}
	got := NegotiateAcceptor(proposed, supportedVerification())
	if got[0].Result != pdu.PresentationContextTransferSyntaxesNotSupported {
		t.Errorf("result = %d, want transfer-syntaxes-not-supported (%d)",
			got[0].Result, pdu.PresentationContextTransferSyntaxesNotSupported)
	}
}

// TestNegotiateMixedProposal verifies each proposed context is negotiated independently and
// the results echo the proposed IDs in order.
func TestNegotiateMixedProposal(t *testing.T) {
	proposed := []pdu.PresentationContextRQ{
		{ID: 1, AbstractSyntax: verificationUID, TransferSyntaxes: []string{explicitVRLE}},
		{ID: 3, AbstractSyntax: ctImageUID, TransferSyntaxes: []string{explicitVRLE}},
	}
	got := NegotiateAcceptor(proposed, supportedVerification())
	if len(got) != 2 {
		t.Fatalf("got %d results, want 2", len(got))
	}
	if got[0].ID != 1 || got[0].Result != pdu.PresentationContextAcceptance {
		t.Errorf("context 1 = (%d, %d), want accepted", got[0].ID, got[0].Result)
	}
	if got[1].ID != 3 || got[1].Result != pdu.PresentationContextAbstractSyntaxNotSupported {
		t.Errorf("context 3 = (%d, %d), want abstract-syntax-not-supported", got[1].ID, got[1].Result)
	}
}

// TestNegotiateMaxPDULengthUnlimited verifies the negotiated effective max for sending is
// resolved with a peer max of 0 (unlimited) yielding the local send cap, never 0 (Codex
// DIMSE-005). The acceptor-side helper resolves it the same way the SCU side does.
func TestNegotiateMaxPDULengthUnlimited(t *testing.T) {
	const localCap = 16382
	if got := EffectiveSendCap(0, localCap); got != localCap {
		t.Errorf("EffectiveSendCap(peer=0 unlimited, local=%d) = %d, want %d (DIMSE-005)", localCap, got, localCap)
	}
	if got := EffectiveSendCap(8192, localCap); got != 8192 {
		t.Errorf("EffectiveSendCap(peer=8192, local=%d) = %d, want 8192", localCap, got)
	}
	if got := EffectiveSendCap(99999, localCap); got != localCap {
		t.Errorf("EffectiveSendCap(peer=99999, local=%d) = %d, want %d (local cap wins)", localCap, got, localCap)
	}
}
