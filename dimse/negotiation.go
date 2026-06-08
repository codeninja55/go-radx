package dimse

import (
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/pdu"
)

// UserIdentityType is the identity type of a user-identity negotiation request (PS3.7 D.3.3.7): it
// selects how the primary (and, for the username-and-passcode type, secondary) field is interpreted.
type UserIdentityType uint8

// The five standard user-identity types (PS3.7 D.3.3.7, Table D.3-15).
const (
	UserIdentityUsername         UserIdentityType = UserIdentityType(pdu.UserIdentityUsername)         // username, no passcode
	UserIdentityUsernamePasscode UserIdentityType = UserIdentityType(pdu.UserIdentityUsernamePasscode) // username + passcode
	UserIdentityKerberos         UserIdentityType = UserIdentityType(pdu.UserIdentityKerberos)         // Kerberos service ticket
	UserIdentitySAML             UserIdentityType = UserIdentityType(pdu.UserIdentitySAML)             // SAML assertion
	UserIdentityJWT              UserIdentityType = UserIdentityType(pdu.UserIdentityJWT)              // JSON Web Token
)

// UserIdentity is a user-identity negotiation request (PS3.7 D.3.3.7), the only association-level
// authentication DICOM defines. Type selects the form of the identity; PrimaryField carries the
// username, Kerberos ticket, SAML assertion, or JWT; SecondaryField carries the passcode for the
// username-and-passcode type only. When PositiveResponseRequested is set the acceptor returns a
// server response readable via Association.UserIdentityResponse. The fields hold secrets (passcodes,
// tokens) that are never logged or written to any catalogue (PRD §9.8).
type UserIdentity struct {
	Type                      UserIdentityType
	PrimaryField              []byte
	SecondaryField            []byte
	PositiveResponseRequested bool
}

// AsyncOps is asynchronous-operations-window negotiation (PS3.7 D.3.3.3): the maximum number of
// operations the AE may invoke and perform concurrently (0 means unlimited). go-radx negotiates the
// window honestly but delivers concurrency through goroutines and context cancellation, not the DICOM
// async-ops mechanism; the acceptor echoes the synchronous (1,1) window unless configured otherwise,
// so a negotiated window of (1,1) is the truthful negotiated value, not a stub.
type AsyncOps struct {
	MaxOperationsInvoked   uint16
	MaxOperationsPerformed uint16
}

// ExtendedNegotiation is SOP-class extended negotiation for one abstract syntax (PS3.7 D.3.3.5): an
// opaque service-class application-information blob keyed by SOP Class UID. The blob's layout is
// defined by the SOP Class's service class (PS3.4); go-radx carries it verbatim.
type ExtendedNegotiation struct {
	SOPClassUID         dicom.SOPClassUID
	ServiceClassAppInfo []byte
}

// CommonExtendedNegotiation is SOP-class common-extended negotiation (PS3.7 D.3.3.6): it binds a SOP
// Class to its Service Class and the related general SOP Classes.
type CommonExtendedNegotiation struct {
	SOPClassUID              dicom.SOPClassUID
	ServiceClassUID          dicom.UID
	RelatedGeneralSOPClasses []dicom.SOPClassUID
}

// WithAsyncOps proposes the asynchronous-operations window (PS3.7 D.3.3.3): the maximum operations
// the requestor may invoke and perform concurrently (0 = unlimited). The acceptor echoes a negotiated
// window in its A-ASSOCIATE-AC, read back via Association.NegotiatedAsyncOps. The window is negotiated
// honestly; concurrency itself is delivered through goroutines and context, so the acceptor echoes
// the synchronous (1,1) window by default.
func WithAsyncOps(invoked, performed uint16) AssociateOption {
	return func(c *associateConfig) {
		c.asyncOps = &pdu.AsyncOperations{MaxOperationsInvoked: invoked, MaxOperationsPerformed: performed}
	}
}

// WithUserIdentity presents the requestor's user identity in the A-ASSOCIATE-RQ (PS3.7 D.3.3.7) — the
// only association-level authentication DICOM defines. The acceptor's seam (a WithAuthenticator on
// the Server, or acse Authenticate) accepts or rejects the association by the presented identity, and
// when id sets PositiveResponseRequested returns a server response readable via
// Association.UserIdentityResponse. The identity fields are secrets that are never logged (PRD §9.8).
func WithUserIdentity(id UserIdentity) AssociateOption {
	return func(c *associateConfig) {
		c.userIdentity = &pdu.UserIdentityRQ{
			Type:                      uint8(id.Type),
			PositiveResponseRequested: id.PositiveResponseRequested,
			PrimaryField:              id.PrimaryField,
			SecondaryField:            id.SecondaryField,
		}
	}
}

// WithExtendedNegotiation adds SOP-class extended-negotiation sub-items to the A-ASSOCIATE-RQ (PS3.7
// D.3.3.5), one per item. Repeating the option, or passing several items, accumulates them.
func WithExtendedNegotiation(items ...ExtendedNegotiation) AssociateOption {
	return func(c *associateConfig) {
		for _, it := range items {
			c.extendedNegotiations = append(c.extendedNegotiations, pdu.ExtendedNegotiation{
				SOPClassUID:         string(it.SOPClassUID),
				ServiceClassAppInfo: it.ServiceClassAppInfo,
			})
		}
	}
}

// WithCommonExtendedNegotiation adds SOP-class common-extended-negotiation sub-items to the
// A-ASSOCIATE-RQ (PS3.7 D.3.3.6), one per item.
func WithCommonExtendedNegotiation(items ...CommonExtendedNegotiation) AssociateOption {
	return func(c *associateConfig) {
		for _, it := range items {
			related := make([]string, 0, len(it.RelatedGeneralSOPClasses))
			for _, uid := range it.RelatedGeneralSOPClasses {
				related = append(related, string(uid))
			}
			c.commonExtendedNegotiations = append(c.commonExtendedNegotiations, pdu.CommonExtendedNegotiation{
				SOPClassUID:              string(it.SOPClassUID),
				ServiceClassUID:          string(it.ServiceClassUID),
				RelatedGeneralSOPClasses: related,
			})
		}
	}
}

// NegotiatedAsyncOps returns the asynchronous-operations window the acceptor echoed in its
// A-ASSOCIATE-AC (PS3.7 D.3.3.3), or nil for an unestablished association or one that negotiated no
// async-ops window. A (1,1) result is the honest synchronous window go-radx echoes by default.
func (a *Association) NegotiatedAsyncOps() *AsyncOps {
	if a == nil || a.requestor == nil {
		return nil
	}
	return a.negotiatedAsyncOps
}

// UserIdentityResponse returns the user-identity server response the acceptor returned (PS3.7
// D.3.3.7). It returns nil only when no user-identity AC sub-item was present — the requestor asked
// for no positive response, presented no identity, the acceptor honoured none, or the association is
// unestablished. When the acceptor honoured a positive-response request with an empty body it returns
// a non-nil empty slice, so a caller can distinguish "positive response honoured, empty body" from "no
// positive response at all". The bytes are opaque (a Kerberos server ticket or SAML response) and are
// never logged (PRD §9.8).
func (a *Association) UserIdentityResponse() []byte {
	if a == nil || a.requestor == nil {
		return nil
	}
	return a.userIdentityResponse
}

// NegotiatedExtendedNegotiations returns the SOP-class extended-negotiation response sub-items the
// acceptor returned in its A-ASSOCIATE-AC (PS3.7 D.3.3.5), one per SOP Class the acceptor accepted
// extended negotiation for. It is nil for an unestablished association or one the acceptor returned
// none for. A requestor that proposed extended negotiation via WithExtendedNegotiation reads the
// acceptor's negotiated service-class application information here.
func (a *Association) NegotiatedExtendedNegotiations() []ExtendedNegotiation {
	if a == nil || a.requestor == nil {
		return nil
	}
	return a.extendedNegotiations
}

// NegotiatedCommonExtendedNegotiations returns the SOP-class common-extended-negotiation response
// sub-items the acceptor returned in its A-ASSOCIATE-AC (PS3.7 D.3.3.6). It is nil for an
// unestablished association or one the acceptor returned none for.
func (a *Association) NegotiatedCommonExtendedNegotiations() []CommonExtendedNegotiation {
	if a == nil || a.requestor == nil {
		return nil
	}
	return a.commonExtendedNegotiations
}

// fromPDUExtendedNegotiations translates the acceptor's pdu-level extended-negotiation responses back
// to public ExtendedNegotiation values, returning nil for an empty input so an association that
// negotiated none reports nil rather than a non-nil empty slice.
func fromPDUExtendedNegotiations(items []pdu.ExtendedNegotiation) []ExtendedNegotiation {
	if len(items) == 0 {
		return nil
	}
	out := make([]ExtendedNegotiation, 0, len(items))
	for _, it := range items {
		out = append(out, ExtendedNegotiation{
			SOPClassUID:         dicom.SOPClassUID(it.SOPClassUID),
			ServiceClassAppInfo: it.ServiceClassAppInfo,
		})
	}
	return out
}

// fromPDUCommonExtendedNegotiations translates the acceptor's pdu-level common-extended-negotiation
// responses back to public CommonExtendedNegotiation values, returning nil for an empty input.
func fromPDUCommonExtendedNegotiations(items []pdu.CommonExtendedNegotiation) []CommonExtendedNegotiation {
	if len(items) == 0 {
		return nil
	}
	out := make([]CommonExtendedNegotiation, 0, len(items))
	for _, it := range items {
		related := make([]dicom.SOPClassUID, 0, len(it.RelatedGeneralSOPClasses))
		for _, uid := range it.RelatedGeneralSOPClasses {
			related = append(related, dicom.SOPClassUID(uid))
		}
		out = append(out, CommonExtendedNegotiation{
			SOPClassUID:              dicom.SOPClassUID(it.SOPClassUID),
			ServiceClassUID:          dicom.UID(it.ServiceClassUID),
			RelatedGeneralSOPClasses: related,
		})
	}
	return out
}

// fromPDUAsyncOps translates the pdu-level async-ops echo to the public AsyncOps, returning nil for a
// nil input so an association that negotiated no async-ops window reports nil.
func fromPDUAsyncOps(ao *pdu.AsyncOperations) *AsyncOps {
	if ao == nil {
		return nil
	}
	return &AsyncOps{MaxOperationsInvoked: ao.MaxOperationsInvoked, MaxOperationsPerformed: ao.MaxOperationsPerformed}
}

// fromPDUUserIdentityAC extracts the opaque server-response bytes from the pdu-level user-identity AC
// sub-item, preserving the sub-item's presence. It returns nil only when no user-identity AC sub-item
// was present (the acceptor honoured no positive response); when the sub-item is present its
// server-response bytes are returned, and a non-nil empty slice when those bytes are empty. The empty
// non-nil case must stay distinguishable from the absent case: a positive response honoured with an
// empty body (a username/passcode acceptance) is a different outcome from no positive response at all,
// and a caller reading UserIdentityResponse needs to tell them apart.
func fromPDUUserIdentityAC(ac *pdu.UserIdentityAC) []byte {
	if ac == nil {
		return nil
	}
	if ac.ServerResponse == nil {
		return []byte{}
	}
	return ac.ServerResponse
}
