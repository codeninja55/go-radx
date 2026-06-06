package acse

import (
	"context"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dimse/pdu"
)

// TestNegotiateRolesGrantsRequestedRoles verifies the acceptor grants the SCP and SCU roles
// the requestor proposes for a supported SOP Class, bounded by the roles the acceptor itself
// is willing to play (PS3.7 D.3.3.4).
func TestNegotiateRolesGrantsRequestedRoles(t *testing.T) {
	requested := []pdu.RoleSelection{
		{SOPClassUID: ctImageUID, SCURole: true, SCPRole: true},
	}
	supported := []SupportedRole{
		{SOPClassUID: ctImageUID, SCURole: true, SCPRole: true},
	}
	got := NegotiateRoles(requested, supported)
	if len(got) != 1 {
		t.Fatalf("NegotiateRoles returned %d results, want 1", len(got))
	}
	if got[0].SOPClassUID != ctImageUID {
		t.Errorf("SOP Class UID = %q, want %q", got[0].SOPClassUID, ctImageUID)
	}
	if !got[0].SCURole || !got[0].SCPRole {
		t.Errorf("granted roles = (scu=%v, scp=%v), want both", got[0].SCURole, got[0].SCPRole)
	}
}

// TestNegotiateRolesClampsToAcceptorCapability verifies the acceptor grants only the roles it
// permits the requestor: when the acceptor permits the requestor the SCU role but not the SCP
// role for a SOP Class, a requestor proposing both is granted SCU and denied SCP (PS3.7
// D.3.3.4).
func TestNegotiateRolesClampsToAcceptorCapability(t *testing.T) {
	requested := []pdu.RoleSelection{
		{SOPClassUID: ctImageUID, SCURole: true, SCPRole: true},
	}
	supported := []SupportedRole{
		{SOPClassUID: ctImageUID, SCURole: true, SCPRole: false},
	}
	got := NegotiateRoles(requested, supported)
	if len(got) != 1 {
		t.Fatalf("NegotiateRoles returned %d results, want 1", len(got))
	}
	if !got[0].SCURole {
		t.Error("SCU role denied; the acceptor permits it and the requestor proposed it")
	}
	if got[0].SCPRole {
		t.Error("SCP role granted; the acceptor does not permit the requestor the SCP role for this SOP Class")
	}
}

// TestNegotiateRolesOmitsResponseWhenUnconfigured verifies that an acceptor with no declared
// role for a requested SOP Class returns no role-selection response for it, so the DICOM
// default roles apply rather than an explicit both-roles-denied refusal that would regress a
// peer proposing the default SCU role (PS3.7 D.3.3.4).
func TestNegotiateRolesOmitsResponseWhenUnconfigured(t *testing.T) {
	requested := []pdu.RoleSelection{
		{SOPClassUID: ctImageUID, SCURole: true, SCPRole: false},
	}
	if got := NegotiateRoles(requested, nil); len(got) != 0 {
		t.Errorf("NegotiateRoles with no supported roles returned %d responses, want 0 (defaults apply)", len(got))
	}
	supported := []SupportedRole{{SOPClassUID: verificationUID, SCURole: true, SCPRole: true}}
	if got := NegotiateRoles(requested, supported); len(got) != 0 {
		t.Errorf("NegotiateRoles for an unconfigured SOP Class returned %d responses, want 0", len(got))
	}
}

// TestNegotiateRolesGrantsRequestorSCPForCGet documents the same-association C-GET arrangement
// under the grant-the-requestor semantics: the acceptor permits the requestor the Storage SCP
// role (SCPRole), and a requestor proposing the SCP role is granted it so it can receive the
// C-STORE sub-operations (PS3.7 D.3.3.4).
func TestNegotiateRolesGrantsRequestorSCPForCGet(t *testing.T) {
	requested := []pdu.RoleSelection{
		{SOPClassUID: ctImageUID, SCURole: false, SCPRole: true},
	}
	supported := []SupportedRole{
		{SOPClassUID: ctImageUID, SCURole: false, SCPRole: true},
	}
	got := NegotiateRoles(requested, supported)
	if len(got) != 1 {
		t.Fatalf("NegotiateRoles returned %d results, want 1", len(got))
	}
	if !got[0].SCPRole {
		t.Error("requestor SCP role denied; same-association C-GET cannot proceed")
	}
}

// TestNegotiateRolesOmitsUnrequestedSOPClasses verifies the acceptor responds only to the SOP
// Classes the requestor proposed a role for: a SOP Class the acceptor supports but the
// requestor did not name yields no role-selection response.
func TestNegotiateRolesOmitsUnrequestedSOPClasses(t *testing.T) {
	requested := []pdu.RoleSelection{
		{SOPClassUID: ctImageUID, SCURole: true, SCPRole: true},
	}
	supported := []SupportedRole{
		{SOPClassUID: ctImageUID, SCURole: true, SCPRole: true},
		{SOPClassUID: verificationUID, SCURole: true, SCPRole: true},
	}
	got := NegotiateRoles(requested, supported)
	if len(got) != 1 {
		t.Fatalf("NegotiateRoles returned %d results, want 1 (only the requested SOP Class)", len(got))
	}
}

// TestAcceptObservesNegotiatedSCPRole drives a full A-ASSOCIATE handshake where the requestor
// proposes the C-GET roles (requestor as SCU, acceptor as SCP) for a storage SOP Class and the
// acceptor grants them. The negotiated SCP role is observable on both sides: the acceptor
// reports the role it granted, and the requestor reads the role the acceptor granted back. This
// is the hard prerequisite for same-association C-GET.
func TestAcceptObservesNegotiatedSCPRole(t *testing.T) {
	rqConn, acConn := loopback(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	acceptorDone := make(chan *Acceptor, 1)
	acceptErr := make(chan error, 1)
	go func() {
		params := acceptParams()
		params.Supported = []SupportedContext{{
			AbstractSyntax:   ctImageUID,
			TransferSyntaxes: []string{explicitVRLE, implicitVRLE},
		}}
		params.SupportedRoles = []SupportedRole{{SOPClassUID: ctImageUID, SCURole: true, SCPRole: true}}
		a, err := Accept(ctx, acConn, params)
		if err != nil {
			acceptErr <- err
			return
		}
		acceptorDone <- a
	}()

	req := Request{
		CalledAETitle:  "ACCEPTOR",
		CallingAETitle: "REQUESTOR",
		MaxPDULength:   16382,
		Contexts: []pdu.PresentationContextRQ{{
			ID:               1,
			AbstractSyntax:   ctImageUID,
			TransferSyntaxes: []string{explicitVRLE, implicitVRLE},
		}},
		RoleSelections: []pdu.RoleSelection{{SOPClassUID: ctImageUID, SCURole: true, SCPRole: true}},
	}
	requestor, err := Associate(ctx, rqConn, req)
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}

	var acc *Acceptor
	select {
	case acc = <-acceptorDone:
	case err := <-acceptErr:
		t.Fatalf("Accept: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("acceptor did not establish")
	}

	// The acceptor reports the role it granted for the SOP Class.
	accRoles := acc.NegotiatedRoles()
	if len(accRoles) != 1 {
		t.Fatalf("acceptor negotiated roles = %d, want 1", len(accRoles))
	}
	if !accRoles[0].SCPRole {
		t.Error("acceptor did not grant itself the SCP role; same-association C-GET cannot proceed")
	}

	// The requestor reads the role the acceptor granted back from the A-ASSOCIATE-AC.
	rqRoles := requestor.NegotiatedRoles()
	if len(rqRoles) != 1 {
		t.Fatalf("requestor negotiated roles = %d, want 1", len(rqRoles))
	}
	if rqRoles[0].SOPClassUID != ctImageUID {
		t.Errorf("negotiated role SOP Class = %q, want %q", rqRoles[0].SOPClassUID, ctImageUID)
	}
	if !rqRoles[0].SCPRole {
		t.Error("requestor observed no granted SCP role; the acceptor's grant did not round-trip")
	}
}
