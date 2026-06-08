package dimse

import (
	"bytes"
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/pdu"
)

// associateForNegotiation opens an association from a fresh SCU to srv with the given options and the
// echo+storage contexts, returning the established association (registered for release). It is the
// shared driver for the root-level negotiation tests.
func associateForNegotiation(t *testing.T, srv *Server, opts ...AssociateOption) *Association {
	t.Helper()
	scu, err := NewAE(AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	assoc, err := scu.Associate(ctx, srv.Addr().String(), AETitle("RADX-SCP"), echoAndStorageContexts(), opts...)
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	t.Cleanup(func() {
		relCtx, relCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer relCancel()
		_ = assoc.Release(relCtx)
	})
	return assoc
}

// TestAssociateNegotiatesAsyncOpsWindow verifies WithAsyncOps proposes the window and the acceptor
// echoes the synchronous (1,1) default, observable on the established association (PS3.7 D.3.3.3).
func TestAssociateNegotiatesAsyncOpsWindow(t *testing.T) {
	h := &serverTestHandler{echoStatus: StatusEchoSuccess, storeStatus: StatusStoreSuccess}
	srv, _ := startServer(t, h)

	assoc := associateForNegotiation(t, srv, WithAsyncOps(8, 8))

	got := assoc.NegotiatedAsyncOps()
	if got == nil {
		t.Fatal("association reports no negotiated async-ops window")
	}
	if got.MaxOperationsInvoked != 1 || got.MaxOperationsPerformed != 1 {
		t.Errorf("negotiated window = %+v, want (1,1) synchronous default", *got)
	}
}

// TestAssociateOmitsAsyncOpsWhenNotRequested verifies an association that proposes no async-ops
// window reports nil, so the negotiated-or-not distinction is observable.
func TestAssociateOmitsAsyncOpsWhenNotRequested(t *testing.T) {
	h := &serverTestHandler{echoStatus: StatusEchoSuccess}
	srv, _ := startServer(t, h)

	assoc := associateForNegotiation(t, srv)
	if assoc.NegotiatedAsyncOps() != nil {
		t.Error("association reports an async-ops window it did not request")
	}
}

// TestAuthenticatorAcceptsUserIdentity drives the user-identity seam end to end: the SCU presents a
// username/passcode with a positive-response request, the Server's authenticator accepts it and
// returns a server response, and the association carries the echoed response (PS3.7 D.3.3.7).
func TestAuthenticatorAcceptsUserIdentity(t *testing.T) {
	var (
		seenType    UserIdentityType
		seenPrimary []byte
		seenPeer    net.Addr
	)
	authenticate := func(id *UserIdentity, peer net.Addr) ([]byte, error) {
		if id == nil {
			return nil, errors.New("anonymous association refused")
		}
		seenType = id.Type
		seenPrimary = id.PrimaryField
		seenPeer = peer
		if !bytes.Equal(id.SecondaryField, []byte("s3cret")) {
			return nil, errors.New("bad passcode")
		}
		return []byte("session-token"), nil
	}
	h := &serverTestHandler{echoStatus: StatusEchoSuccess}
	srv, _ := startServer(t, h, WithAuthenticator(authenticate))

	assoc := associateForNegotiation(t, srv, WithUserIdentity(UserIdentity{
		Type:                      UserIdentityUsernamePasscode,
		PrimaryField:              []byte("alice"),
		SecondaryField:            []byte("s3cret"),
		PositiveResponseRequested: true,
	}))

	if seenType != UserIdentityUsernamePasscode {
		t.Errorf("authenticator saw type %d, want %d", seenType, UserIdentityUsernamePasscode)
	}
	if !bytes.Equal(seenPrimary, []byte("alice")) {
		t.Errorf("authenticator saw primary %q, want alice", seenPrimary)
	}
	if seenPeer == nil {
		t.Error("authenticator saw no peer address")
	}
	if resp := assoc.UserIdentityResponse(); !bytes.Equal(resp, []byte("session-token")) {
		t.Errorf("server response = %q, want session-token", resp)
	}
}

// TestAuthenticatorRejectsUserIdentity verifies a failed authentication surfaces to the SCU as an
// AssociationRejected error before any service runs (PS3.7 D.3.3.7).
func TestAuthenticatorRejectsUserIdentity(t *testing.T) {
	authenticate := func(id *UserIdentity, _ net.Addr) ([]byte, error) {
		return nil, errors.New("invalid credentials")
	}
	h := &serverTestHandler{echoStatus: StatusEchoSuccess}
	srv, _ := startServer(t, h, WithAuthenticator(authenticate))

	scu, err := NewAE(AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = scu.Associate(ctx, srv.Addr().String(), AETitle("RADX-SCP"), echoAndStorageContexts(),
		WithUserIdentity(UserIdentity{Type: UserIdentityUsername, PrimaryField: []byte("mallory")}))
	if err == nil {
		t.Fatal("Associate succeeded; want a rejection for failed authentication")
	}
	var ae *AssociationError
	if !errors.As(err, &ae) {
		t.Fatalf("error = %v, want *AssociationError", err)
	}
	if ae.Kind != AssociationRejected {
		t.Errorf("rejection kind = %v, want AssociationRejected", ae.Kind)
	}
}

// TestAuthenticatorRefusesAnonymous verifies the authenticator receives a nil identity for an
// association presenting none, so a policy can refuse anonymous associations.
func TestAuthenticatorRefusesAnonymous(t *testing.T) {
	authenticate := func(id *UserIdentity, _ net.Addr) ([]byte, error) {
		if id == nil {
			return nil, errors.New("identity required")
		}
		return nil, nil
	}
	h := &serverTestHandler{echoStatus: StatusEchoSuccess}
	srv, _ := startServer(t, h, WithAuthenticator(authenticate))

	scu, err := NewAE(AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := scu.Associate(ctx, srv.Addr().String(), AETitle("RADX-SCP"), echoAndStorageContexts()); err == nil {
		t.Fatal("anonymous Associate succeeded; want a rejection")
	}
}

// TestAssociateCarriesExtendedNegotiation verifies WithExtendedNegotiation and
// WithCommonExtendedNegotiation reach the acceptor through the established association without
// breaking negotiation (PS3.7 D.3.3.5, D.3.3.6). The acceptor accepts the association, proving the
// sub-items round-trip the wire.
func TestAssociateCarriesExtendedNegotiation(t *testing.T) {
	h := &serverTestHandler{echoStatus: StatusEchoSuccess}
	srv, _ := startServer(t, h)

	assoc := associateForNegotiation(t, srv,
		WithExtendedNegotiation(ExtendedNegotiation{
			SOPClassUID:         dicom.SOPClassUID("1.2.840.10008.1.1"),
			ServiceClassAppInfo: []byte{0x01, 0x01},
		}),
		WithCommonExtendedNegotiation(CommonExtendedNegotiation{
			SOPClassUID:              dicom.SOPClassUID("1.2.840.10008.5.1.4.1.1.2"),
			ServiceClassUID:          dicom.UID("1.2.840.10008.4.2"),
			RelatedGeneralSOPClasses: []dicom.SOPClassUID{"1.2.840.10008.5.1.4.1.1.4"},
		}),
	)
	if len(assoc.AcceptedContexts()) == 0 {
		t.Error("association accepted no contexts; the extended-negotiation sub-items may have broken negotiation")
	}
}

// TestExtendedNegotiationResponseConverters verifies the pdu-to-public converters that surface an
// acceptor's extended- and common-extended-negotiation AC responses on the established association
// (PS3.7 D.3.3.5, D.3.3.6). The stock acceptor echoes no extended-negotiation response, so the
// requestor-side observability of an AC that does carry one is covered end to end by the acse package;
// this exercises the conversion boundary the root accessors delegate to, including the nil-for-empty
// contract and UID-typing.
func TestExtendedNegotiationResponseConverters(t *testing.T) {
	if got := fromPDUExtendedNegotiations(nil); got != nil {
		t.Errorf("fromPDUExtendedNegotiations(nil) = %v, want nil", got)
	}
	if got := fromPDUCommonExtendedNegotiations(nil); got != nil {
		t.Errorf("fromPDUCommonExtendedNegotiations(nil) = %v, want nil", got)
	}

	en := fromPDUExtendedNegotiations([]pdu.ExtendedNegotiation{
		{SOPClassUID: "1.2.840.10008.1.1", ServiceClassAppInfo: []byte{0x01, 0x02}},
	})
	if len(en) != 1 || en[0].SOPClassUID != dicom.SOPClassUID("1.2.840.10008.1.1") {
		t.Fatalf("extended negotiation = %+v", en)
	}
	if !bytes.Equal(en[0].ServiceClassAppInfo, []byte{0x01, 0x02}) {
		t.Errorf("service-class app info = %v, want [1 2]", en[0].ServiceClassAppInfo)
	}

	cen := fromPDUCommonExtendedNegotiations([]pdu.CommonExtendedNegotiation{
		{
			SOPClassUID:              "1.2.840.10008.5.1.4.1.1.2",
			ServiceClassUID:          "1.2.840.10008.4.2",
			RelatedGeneralSOPClasses: []string{"1.2.840.10008.5.1.4.1.1.4"},
		},
	})
	if len(cen) != 1 || cen[0].SOPClassUID != dicom.SOPClassUID("1.2.840.10008.5.1.4.1.1.2") {
		t.Fatalf("common extended negotiation = %+v", cen)
	}
	if cen[0].ServiceClassUID != dicom.UID("1.2.840.10008.4.2") {
		t.Errorf("service class UID = %q, want 1.2.840.10008.4.2", cen[0].ServiceClassUID)
	}
	if len(cen[0].RelatedGeneralSOPClasses) != 1 || cen[0].RelatedGeneralSOPClasses[0] != dicom.SOPClassUID("1.2.840.10008.5.1.4.1.1.4") {
		t.Errorf("related general SOP classes = %v", cen[0].RelatedGeneralSOPClasses)
	}
}

// TestAssociateOmitsExtendedNegotiationResponseWhenNoneEchoed verifies an established association
// whose acceptor echoed no extended-negotiation response reports nil from both accessors, so the
// negotiated-or-not distinction is observable (PS3.7 D.3.3.5, D.3.3.6).
func TestAssociateOmitsExtendedNegotiationResponseWhenNoneEchoed(t *testing.T) {
	h := &serverTestHandler{echoStatus: StatusEchoSuccess}
	srv, _ := startServer(t, h)

	assoc := associateForNegotiation(t, srv,
		WithExtendedNegotiation(ExtendedNegotiation{
			SOPClassUID:         dicom.SOPClassUID("1.2.840.10008.1.1"),
			ServiceClassAppInfo: []byte{0x01},
		}),
	)
	if got := assoc.NegotiatedExtendedNegotiations(); got != nil {
		t.Errorf("NegotiatedExtendedNegotiations = %+v, want nil (stock acceptor echoes none)", got)
	}
	if got := assoc.NegotiatedCommonExtendedNegotiations(); got != nil {
		t.Errorf("NegotiatedCommonExtendedNegotiations = %+v, want nil", got)
	}
}
