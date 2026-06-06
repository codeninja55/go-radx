package acse

import "github.com/codeninja55/go-radx/dimse/pdu"

// insignificantTransferSyntax is the placeholder transfer syntax a rejected presentation
// context still carries in an A-ASSOCIATE-AC (PS3.8 9.3.3.2 requires exactly one, even when
// the context is not accepted). Implicit VR Little Endian is the conventional placeholder
// (Codex DIMSE-008). The pdu encoder substitutes it for an empty transfer syntax; setting it
// here keeps the negotiation result self-describing.
const insignificantTransferSyntax = "1.2.840.10008.1.2"

// SupportedContext is one abstract syntax the acceptor supports, with the transfer syntaxes
// it accepts for that syntax in preference order (the acceptor's first match wins). The
// acse layer works in plain UID strings at the pdu level so it never imports the root dimse
// package; the root package translates its public PresentationContext to and from these
// (acyclic layering, dimse.md "Overview of the layers").
type SupportedContext struct {
	AbstractSyntax   string
	TransferSyntaxes []string
}

// NegotiateAcceptor matches each proposed presentation context against the acceptor's
// supported set and returns one A-ASSOCIATE-AC result per proposal, echoing the proposed
// context ID and preserving order (PS3.8 9.3.3.2). A proposed abstract syntax the acceptor
// does not support is rejected with PresentationContextAbstractSyntaxNotSupported; a
// supported abstract syntax with no overlapping transfer syntax is rejected with
// PresentationContextTransferSyntaxesNotSupported; an accepted context selects the
// acceptor's first supported transfer syntax that the requestor also proposed. Every
// result carries a transfer syntax — the selected one when accepted, an insignificant
// placeholder when rejected — so the AC PDU always encodes exactly one transfer-syntax
// sub-item per context (Codex DIMSE-008).
func NegotiateAcceptor(proposed []pdu.PresentationContextRQ, supported []SupportedContext) []pdu.PresentationContextAC {
	results := make([]pdu.PresentationContextAC, 0, len(proposed))
	for _, rq := range proposed {
		results = append(results, negotiateOne(rq, supported))
	}
	return results
}

// negotiateOne negotiates a single proposed context against the supported set.
func negotiateOne(rq pdu.PresentationContextRQ, supported []SupportedContext) pdu.PresentationContextAC {
	sc, ok := findSupported(supported, rq.AbstractSyntax)
	if !ok {
		return pdu.PresentationContextAC{
			ID:             rq.ID,
			Result:         pdu.PresentationContextAbstractSyntaxNotSupported,
			TransferSyntax: insignificantTransferSyntax,
		}
	}
	if ts, ok := selectTransferSyntax(sc.TransferSyntaxes, rq.TransferSyntaxes); ok {
		return pdu.PresentationContextAC{
			ID:             rq.ID,
			Result:         pdu.PresentationContextAcceptance,
			TransferSyntax: ts,
		}
	}
	return pdu.PresentationContextAC{
		ID:             rq.ID,
		Result:         pdu.PresentationContextTransferSyntaxesNotSupported,
		TransferSyntax: insignificantTransferSyntax,
	}
}

// findSupported returns the supported context for abstract, if present.
func findSupported(supported []SupportedContext, abstract string) (SupportedContext, bool) {
	for _, sc := range supported {
		if sc.AbstractSyntax == abstract {
			return sc, true
		}
	}
	return SupportedContext{}, false
}

// selectTransferSyntax returns the acceptor's first supported transfer syntax that the
// requestor also proposed, so the acceptor's preference order wins (PS3.8 9.3.3.2).
func selectTransferSyntax(acceptorPreferred, proposed []string) (string, bool) {
	for _, ts := range acceptorPreferred {
		for _, p := range proposed {
			if ts == p {
				return ts, true
			}
		}
	}
	return "", false
}

// SupportedRole declares, for one SOP Class, which roles the acceptor will grant the
// association requestor (PS3.7 D.3.3.4). SCURole permits the requestor to act as the SCU and
// SCPRole permits it to act as the SCP; the acceptor itself takes the complementary role (the
// requestor acting as SCP means the acceptor acts as the SCU). It bounds what NegotiateRoles
// grants: a role not permitted here is denied. For same-association C-GET the acceptor permits
// the requestor the Storage SCP role by setting SCPRole. The acse layer works in plain UID
// strings so it never imports the root dimse package (acyclic layering).
type SupportedRole struct {
	SOPClassUID string
	SCURole     bool
	SCPRole     bool
}

// NegotiateRoles matches each requested SCP/SCU role-selection sub-item against the roles the
// acceptor permits and returns one A-ASSOCIATE-AC role-selection response per request, in the
// requestor's order (PS3.7 D.3.3.4). A request for a SOP Class the acceptor declared no role
// for yields no response, so the DICOM default roles apply rather than an explicit denial: an
// acceptor with no role configuration must not regress a peer that proposed the default SCU
// role into an explicit refusal. Otherwise each requested role is granted only when the
// acceptor also permits it for that SOP Class. A SOP Class the acceptor permits but the
// requestor did not propose a role for yields no response, because role selection is
// requestor-initiated.
func NegotiateRoles(requested []pdu.RoleSelection, supported []SupportedRole) []pdu.RoleSelection {
	results := make([]pdu.RoleSelection, 0, len(requested))
	for _, rq := range requested {
		sr, ok := findSupportedRole(supported, rq.SOPClassUID)
		if !ok {
			// No declared role for this SOP Class: omit the response so the default roles
			// apply (PS3.7 D.3.3.4). Emitting a both-roles-denied response here would turn
			// the default SCU role a peer proposed into an explicit denial.
			continue
		}
		results = append(results, pdu.RoleSelection{
			SOPClassUID: rq.SOPClassUID,
			SCURole:     rq.SCURole && sr.SCURole,
			SCPRole:     rq.SCPRole && sr.SCPRole,
		})
	}
	return results
}

// findSupportedRole returns the acceptor's supported role for sopClass, if declared.
func findSupportedRole(supported []SupportedRole, sopClass string) (SupportedRole, bool) {
	for _, sr := range supported {
		if sr.SOPClassUID == sopClass {
			return sr, true
		}
	}
	return SupportedRole{}, false
}

// EffectiveSendCap resolves the number of P-DATA-TF body bytes that may be sent to a peer
// whose advertised maximum PDU length is peerMax, given the local send cap localCap. A peer
// max of 0 means "no maximum specified" (unlimited) and resolves to localCap, never 0
// (Codex DIMSE-005); otherwise the smaller of the peer max and the local cap wins. Both
// the SCU (after reading the A-ASSOCIATE-AC) and the SCP use this so a sent PDU never
// exceeds either party's limit.
func EffectiveSendCap(peerMax, localCap uint32) uint32 {
	if peerMax == 0 {
		return localCap
	}
	if localCap != 0 && localCap < peerMax {
		return localCap
	}
	return peerMax
}
