package acse

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dimse/dul"
	"github.com/codeninja55/go-radx/dimse/pdu"
)

// negotiateHandshake drives a full in-process A-ASSOCIATE handshake: it runs Accept on one end of a
// loopback pair with the given params and Associate on the other with req, returning the established
// Requestor and Acceptor (or failing the test on either error). It is the shared driver for the
// negotiation round-trip and authentication tests.
func negotiateHandshake(t *testing.T, req Request, params AcceptParams) (*Requestor, *Acceptor) {
	t.Helper()
	rqConn, acConn := loopback(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	type acResult struct {
		acc *Acceptor
		err error
	}
	done := make(chan acResult, 1)
	go func() {
		a, err := Accept(ctx, acConn, params)
		done <- acResult{a, err}
	}()

	requestor, err := Associate(ctx, rqConn, req)
	res := <-done
	if err != nil {
		t.Fatalf("Associate: %v (accept err: %v)", err, res.err)
	}
	if res.err != nil {
		t.Fatalf("Accept: %v", res.err)
	}
	return requestor, res.acc
}

// rejectingHandshake drives Accept/Associate where the acceptor is expected to reject. It returns the
// requestor's error (which must be a *RejectedError) and the acceptor's error.
func rejectingHandshake(t *testing.T, req Request, params AcceptParams) (reqErr, accErr error) {
	t.Helper()
	rqConn, acConn := loopback(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	done := make(chan error, 1)
	go func() {
		_, err := Accept(ctx, acConn, params)
		done <- err
	}()
	_, reqErr = Associate(ctx, rqConn, req)
	accErr = <-done
	return reqErr, accErr
}

// negotiateRaw sends a caller-built A-ASSOCIATE-RQ on conn and reads the acceptor's response PDU,
// bypassing Associate so a test can present a malformed or non-conformant RQ (an unsupported protocol
// version, an over-limit context count) the conformant requestor would never send. It returns the
// decoded response PDU and any transport error.
func negotiateRaw(ctx context.Context, conn *dul.Conn, rq *pdu.AssociateRQ) (pdu.PDU, *dul.StateMachine, error) {
	m := dul.NewStateMachine()
	if _, _, err := m.Apply(dul.Evt1); err != nil {
		return nil, m, err
	}
	if _, _, err := m.Apply(dul.Evt2); err != nil {
		return nil, m, err
	}
	if err := conn.WritePDU(ctx, rq); err != nil {
		return nil, m, err
	}
	resp, _, err := dul.DriveInbound(ctx, conn, m)
	return resp, m, err
}

// TestAcceptEchoesAsyncOpsWindow verifies the acceptor echoes a synchronous (1,1) async-ops window
// when the requestor proposes one and the acceptor declares none (PS3.7 D.3.3.3). Both sides observe
// the negotiated window.
func TestAcceptEchoesAsyncOpsWindow(t *testing.T) {
	req := echoRequest()
	req.AsyncOps = &pdu.AsyncOperations{MaxOperationsInvoked: 5, MaxOperationsPerformed: 5}
	requestor, acc := negotiateHandshake(t, req, acceptParams())

	got := requestor.NegotiatedAsyncOps()
	if got == nil {
		t.Fatal("requestor observed no negotiated async-ops window")
	}
	if got.MaxOperationsInvoked != 1 || got.MaxOperationsPerformed != 1 {
		t.Errorf("negotiated window = %+v, want (1,1) synchronous default", got)
	}
	if acc.NegotiatedAsyncOps() == nil {
		t.Error("acceptor reports no echoed async-ops window")
	}
}

// TestAcceptHonoursConfiguredAsyncOpsWindow verifies the acceptor echoes its own configured window
// when set, rather than the (1,1) default.
func TestAcceptHonoursConfiguredAsyncOpsWindow(t *testing.T) {
	req := echoRequest()
	req.AsyncOps = &pdu.AsyncOperations{MaxOperationsInvoked: 5, MaxOperationsPerformed: 5}
	params := acceptParams()
	params.AsyncOps = &pdu.AsyncOperations{MaxOperationsInvoked: 2, MaxOperationsPerformed: 3}
	requestor, _ := negotiateHandshake(t, req, params)

	got := requestor.NegotiatedAsyncOps()
	if got == nil || got.MaxOperationsInvoked != 2 || got.MaxOperationsPerformed != 3 {
		t.Errorf("negotiated window = %+v, want (2,3)", got)
	}
}

// TestAcceptOmitsAsyncOpsWhenNotRequested verifies the acceptor does not send an async-ops echo when
// the requestor proposed none: the sub-item is requestor-initiated (PS3.7 D.3.3.3).
func TestAcceptOmitsAsyncOpsWhenNotRequested(t *testing.T) {
	requestor, acc := negotiateHandshake(t, echoRequest(), acceptParams())
	if requestor.NegotiatedAsyncOps() != nil {
		t.Error("requestor observed an async-ops window it did not request")
	}
	if acc.NegotiatedAsyncOps() != nil {
		t.Error("acceptor echoed an async-ops window the requestor did not request")
	}
}

// TestAcceptCarriesExtendedNegotiation verifies the requestor's extended and common-extended
// negotiation sub-items reach the acceptor through the handshake (PS3.7 D.3.3.5, D.3.3.6).
func TestAcceptCarriesExtendedNegotiation(t *testing.T) {
	req := echoRequest()
	req.ExtendedNegotiations = []pdu.ExtendedNegotiation{{SOPClassUID: verificationUID, ServiceClassAppInfo: []byte{0x01}}}
	req.CommonExtendedNegotiations = []pdu.CommonExtendedNegotiation{{SOPClassUID: ctImageUID, ServiceClassUID: "1.2.840.10008.4.2"}}
	_, acc := negotiateHandshake(t, req, acceptParams())

	if acc.request == nil {
		t.Fatal("acceptor recorded no request")
	}
	en := acc.request.UserInfo.ExtendedNegotiations
	if len(en) != 1 || en[0].SOPClassUID != verificationUID {
		t.Errorf("acceptor extended negotiations = %+v", en)
	}
	cen := acc.request.UserInfo.CommonExtendedNegotiations
	if len(cen) != 1 || cen[0].SOPClassUID != ctImageUID {
		t.Errorf("acceptor common extended negotiations = %+v", cen)
	}
}

// TestRequestorObservesExtendedNegotiationResponse verifies the requestor reads back the SOP-class
// extended- and common-extended-negotiation response sub-items the acceptor returns in its
// A-ASSOCIATE-AC (PS3.7 D.3.3.5, D.3.3.6). The stock Accept does not echo these, so the acceptor side
// is hand-driven to write an AC that carries them; without the requestor-side accessors the AC
// response would be decoded and dropped.
func TestRequestorObservesExtendedNegotiationResponse(t *testing.T) {
	rqConn, acConn := loopback(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	req := echoRequest()
	req.ExtendedNegotiations = []pdu.ExtendedNegotiation{{SOPClassUID: verificationUID, ServiceClassAppInfo: []byte{0x01}}}

	done := make(chan error, 1)
	go func() {
		done <- acceptWithExtendedNegotiationAC(ctx, acConn)
	}()

	requestor, err := Associate(ctx, rqConn, req)
	if accErr := <-done; accErr != nil {
		t.Fatalf("acceptor: %v", accErr)
	}
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}

	en := requestor.NegotiatedExtendedNegotiations()
	if len(en) != 1 || en[0].SOPClassUID != verificationUID {
		t.Fatalf("negotiated extended negotiations = %+v, want one for %s", en, verificationUID)
	}
	if want := []byte{0x02}; len(en[0].ServiceClassAppInfo) != 1 || en[0].ServiceClassAppInfo[0] != want[0] {
		t.Errorf("service-class app info = %v, want %v", en[0].ServiceClassAppInfo, want)
	}

	cen := requestor.NegotiatedCommonExtendedNegotiations()
	if len(cen) != 1 || cen[0].SOPClassUID != ctImageUID {
		t.Errorf("negotiated common extended negotiations = %+v, want one for %s", cen, ctImageUID)
	}
}

// acceptWithExtendedNegotiationAC hand-drives the acceptor side of a handshake: it reads the
// A-ASSOCIATE-RQ and replies with an A-ASSOCIATE-AC that carries SOP-class extended- and
// common-extended-negotiation response sub-items, which the stock Accept never echoes. It mirrors the
// state-machine transitions Accept uses (Evt5 -> Sta2, read RQ -> Sta3, Evt7 -> send AC -> Sta6).
func acceptWithExtendedNegotiationAC(ctx context.Context, conn *dul.Conn) error {
	m := dul.NewStateMachine()
	if _, _, err := m.Apply(dul.Evt5); err != nil {
		return err
	}
	resp, _, err := dul.DriveInbound(ctx, conn, m)
	if err != nil {
		return err
	}
	rq, ok := resp.(*pdu.AssociateRQ)
	if !ok {
		return errors.New("acceptor expected an A-ASSOCIATE-RQ")
	}
	ac := &pdu.AssociateAC{
		ProtocolVersion:      protocolVersion,
		CalledAETitle:        rq.CalledAETitle,
		CallingAETitle:       rq.CallingAETitle,
		ApplicationContext:   applicationContextUID,
		PresentationContexts: []pdu.PresentationContextAC{{ID: 1, Result: pdu.PresentationContextAcceptance, TransferSyntax: explicitVRLE}},
		UserInfo: pdu.UserInformation{
			MaxPDULength:               16382,
			ExtendedNegotiations:       []pdu.ExtendedNegotiation{{SOPClassUID: verificationUID, ServiceClassAppInfo: []byte{0x02}}},
			CommonExtendedNegotiations: []pdu.CommonExtendedNegotiation{{SOPClassUID: ctImageUID, ServiceClassUID: "1.2.840.10008.4.2"}},
		},
	}
	if _, _, serr := m.Apply(dul.Evt7); serr != nil {
		return serr
	}
	return conn.WritePDU(ctx, ac)
}

// TestAuthenticateAcceptsAndEchoesResponse drives the user-identity authentication seam: the acceptor
// accepts the presented identity and, since a positive response was requested, echoes the server
// response back to the requestor (PS3.7 D.3.3.7).
func TestAuthenticateAcceptsAndEchoesResponse(t *testing.T) {
	req := echoRequest()
	req.UserIdentity = &pdu.UserIdentityRQ{
		Type:                      pdu.UserIdentityUsernamePasscode,
		PrimaryField:              []byte("alice"),
		SecondaryField:            []byte("pw"),
		PositiveResponseRequested: true,
	}
	params := acceptParams()
	var seen *pdu.UserIdentityRQ
	params.Authenticate = func(id *pdu.UserIdentityRQ) ([]byte, error) {
		seen = id
		return []byte("welcome"), nil
	}
	requestor, _ := negotiateHandshake(t, req, params)

	if seen == nil || string(seen.PrimaryField) != "alice" {
		t.Fatalf("authenticator saw identity %+v, want primary=alice", seen)
	}
	resp := requestor.UserIdentityResponse()
	if resp == nil || string(resp.ServerResponse) != "welcome" {
		t.Errorf("server response = %+v, want welcome", resp)
	}
}

// TestAuthenticateRejectsAssociation verifies a non-nil authenticator error rejects the association
// with a service-user A-ASSOCIATE-RJ before any service runs (PS3.7 D.3.3.7).
func TestAuthenticateRejectsAssociation(t *testing.T) {
	req := echoRequest()
	req.UserIdentity = &pdu.UserIdentityRQ{Type: pdu.UserIdentityUsername, PrimaryField: []byte("mallory")}
	params := acceptParams()
	params.Authenticate = func(id *pdu.UserIdentityRQ) ([]byte, error) {
		return nil, errors.New("bad credentials")
	}
	reqErr, _ := rejectingHandshake(t, req, params)

	var rej *RejectedError
	if !errors.As(reqErr, &rej) {
		t.Fatalf("requestor error = %v, want *RejectedError", reqErr)
	}
	if rej.Source != pdu.AssociateRJSourceServiceUser {
		t.Errorf("reject source = %d, want service-user (%d)", rej.Source, pdu.AssociateRJSourceServiceUser)
	}
}

// TestAuthenticateSeesNilForAnonymous verifies the authenticator runs with a nil identity for an
// association that presents none, so a policy can refuse anonymous associations.
func TestAuthenticateSeesNilForAnonymous(t *testing.T) {
	params := acceptParams()
	params.Authenticate = func(id *pdu.UserIdentityRQ) ([]byte, error) {
		if id != nil {
			t.Errorf("authenticator saw identity %+v, want nil for anonymous", id)
		}
		return nil, nil
	}
	negotiateHandshake(t, echoRequest(), params)
}

// TestAcceptNoPositiveResponseOmitsAC verifies the acceptor returns no user-identity server response
// when the requestor did not request a positive response (PS3.7 D.3.3.7).
func TestAcceptNoPositiveResponseOmitsAC(t *testing.T) {
	req := echoRequest()
	req.UserIdentity = &pdu.UserIdentityRQ{Type: pdu.UserIdentityUsername, PrimaryField: []byte("alice")}
	params := acceptParams()
	params.Authenticate = func(id *pdu.UserIdentityRQ) ([]byte, error) { return []byte("ignored"), nil }
	requestor, _ := negotiateHandshake(t, req, params)

	if requestor.UserIdentityResponse() != nil {
		t.Error("acceptor returned a server response without a positive-response request")
	}
}

// TestAcceptRejectsUnsupportedProtocolVersion verifies the acceptor rejects an A-ASSOCIATE-RQ whose
// protocol version offers no supported bit, with a service-provider-ACSE A-ASSOCIATE-RJ carrying the
// protocol-version-not-supported reason (PS3.8 §7.1.1.7, Table 9-22).
func TestAcceptRejectsUnsupportedProtocolVersion(t *testing.T) {
	rqConn, acConn := loopback(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := Accept(ctx, acConn, acceptParams())
		done <- err
	}()

	// Build the RQ directly so the unsupported protocol version reaches the acceptor (Associate always
	// sends version 1).
	rq := buildAssociateRQ(echoRequest())
	rq.ProtocolVersion = 0 // no supported version bit set
	if _, _, err := negotiateRaw(ctx, rqConn, rq); err != nil {
		t.Fatalf("send RQ: %v", err)
	}
	accErr := <-done

	var rej *RejectedError
	if !errors.As(accErr, &rej) {
		t.Fatalf("acceptor error = %v, want *RejectedError", accErr)
	}
	if rej.Source != pdu.AssociateRJSourceServiceProviderACSE {
		t.Errorf("reject source = %d, want service-provider-ACSE (%d)", rej.Source, pdu.AssociateRJSourceServiceProviderACSE)
	}
	if rej.Reason != reasonProtocolVersionNotSupported {
		t.Errorf("reject reason = %d, want protocol-version-not-supported (%d)", rej.Reason, reasonProtocolVersionNotSupported)
	}
}

// TestAcceptRejectsOverContextLimit verifies the acceptor rejects an A-ASSOCIATE-RQ proposing more
// than the 128-presentation-context limit, with a service-provider-ACSE A-ASSOCIATE-RJ (PS3.8
// §7.1.1.13).
func TestAcceptRejectsOverContextLimit(t *testing.T) {
	rqConn, acConn := loopback(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := Accept(ctx, acConn, acceptParams())
		done <- err
	}()

	rq := buildAssociateRQ(echoRequest())
	rq.PresentationContexts = make([]pdu.PresentationContextRQ, maxPresentationContexts+1)
	for i := range rq.PresentationContexts {
		rq.PresentationContexts[i] = pdu.PresentationContextRQ{
			ID:               uint8(2*i + 1),
			AbstractSyntax:   verificationUID,
			TransferSyntaxes: []string{implicitVRLE},
		}
	}
	if _, _, err := negotiateRaw(ctx, rqConn, rq); err != nil {
		t.Fatalf("send RQ: %v", err)
	}
	accErr := <-done

	var rej *RejectedError
	if !errors.As(accErr, &rej) {
		t.Fatalf("acceptor error = %v, want *RejectedError", accErr)
	}
	if rej.Source != pdu.AssociateRJSourceServiceProviderACSE {
		t.Errorf("reject source = %d, want service-provider-ACSE (%d)", rej.Source, pdu.AssociateRJSourceServiceProviderACSE)
	}
}

// TestAcceptRejectsMalformedAETitle verifies the acceptor rejects an A-ASSOCIATE-RQ whose AE-title
// fields carry a non-conformant character (a control byte) with a service-user A-ASSOCIATE-RJ naming
// the bad title (PS3.5 VR AE, PS3.8 Table 9-21).
func TestAcceptRejectsMalformedAETitle(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(rq *pdu.AssociateRQ)
		wantReason uint8
	}{
		{
			name:       "malformed called",
			mutate:     func(rq *pdu.AssociateRQ) { rq.CalledAETitle = [16]byte{'B', 'A', 'D', 0x01} },
			wantReason: reasonCalledAETitleNotRecognized,
		},
		{
			name:       "malformed calling",
			mutate:     func(rq *pdu.AssociateRQ) { rq.CallingAETitle = [16]byte{'B', 'A', 'D', '\\'} },
			wantReason: reasonCallingAETitleNotRecognized,
		},
		{
			name:       "empty called",
			mutate:     func(rq *pdu.AssociateRQ) { rq.CalledAETitle = [16]byte{' ', ' ', ' '} },
			wantReason: reasonCalledAETitleNotRecognized,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rqConn, acConn := loopback(t)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			done := make(chan error, 1)
			go func() {
				_, err := Accept(ctx, acConn, acceptParams())
				done <- err
			}()

			rq := buildAssociateRQ(echoRequest())
			tt.mutate(rq)
			if _, _, err := negotiateRaw(ctx, rqConn, rq); err != nil {
				t.Fatalf("send RQ: %v", err)
			}
			accErr := <-done

			var rej *RejectedError
			if !errors.As(accErr, &rej) {
				t.Fatalf("acceptor error = %v, want *RejectedError", accErr)
			}
			if rej.Source != pdu.AssociateRJSourceServiceUser {
				t.Errorf("reject source = %d, want service-user (%d)", rej.Source, pdu.AssociateRJSourceServiceUser)
			}
			if rej.Reason != tt.wantReason {
				t.Errorf("reject reason = %d, want %d", rej.Reason, tt.wantReason)
			}
		})
	}
}
