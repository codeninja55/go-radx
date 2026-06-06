package dimse

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/acse"
	"github.com/codeninja55/go-radx/dimse/dul"
)

const ctImageStorageUID = dicom.SOPClassUID("1.2.840.10008.5.1.4.1.1.2")

// startStorageRoleAcceptor listens on loopback and runs the acse acceptor for the CT Image
// Storage SOP Class, granting the SCP role so a requestor proposing same-association C-GET
// roles sees its grant. It services one graceful release per connection.
func startStorageRoleAcceptor(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			nc, aerr := ln.Accept()
			if aerr != nil {
				return
			}
			go func(c net.Conn) {
				conn := dul.NewConn(c, 0)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				acc, perr := acse.Accept(ctx, conn, acse.AcceptParams{
					CalledAETitle: "ACCEPTOR",
					MaxPDULength:  16382,
					Supported: []acse.SupportedContext{{
						AbstractSyntax:   string(ctImageStorageUID),
						TransferSyntaxes: []string{"1.2.840.10008.1.2.1", "1.2.840.10008.1.2"},
					}},
					SupportedRoles: []acse.SupportedRole{{
						SOPClassUID: string(ctImageStorageUID),
						SCURole:     true,
						SCPRole:     true,
					}},
				})
				if perr != nil {
					c.Close()
					return
				}
				_ = acc.ServeRelease(ctx)
			}(nc)
		}
	}()
	return ln.Addr().String()
}

// TestAssociateWithRoleSelectionObservesGrantedSCPRole drives AE.Associate with the
// WithRoleSelection option proposing the C-GET roles (requestor as SCU, acceptor as SCP) and
// verifies the granted SCP role is observable on the public Association.
func TestAssociateWithRoleSelectionObservesGrantedSCPRole(t *testing.T) {
	addr := startStorageRoleAcceptor(t)

	ae, err := NewAE(AETitle("REQUESTOR"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	contexts := []PresentationContext{NewPresentationContext(1, ctImageStorageUID)}
	assoc, err := ae.Associate(ctx, addr, AETitle("ACCEPTOR"), contexts,
		WithRoleSelection(RoleSelection{SOPClassUID: ctImageStorageUID, SCURole: true, SCPRole: true}))
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}

	roles := assoc.NegotiatedRoles()
	if len(roles) != 1 {
		t.Fatalf("negotiated roles = %d, want 1", len(roles))
	}
	if roles[0].SOPClassUID != ctImageStorageUID {
		t.Errorf("negotiated role SOP Class = %q, want %q", roles[0].SOPClassUID, ctImageStorageUID)
	}
	if !roles[0].SCPRole {
		t.Error("acceptor did not grant the SCP role; same-association C-GET cannot proceed")
	}

	if err := assoc.Release(ctx); err != nil {
		t.Fatalf("Release: %v", err)
	}
}

// TestWithRoleSelectionAccumulatesPerSOPClass verifies repeated WithRoleSelection options
// accumulate one role-selection request per SOP Class on the outbound A-ASSOCIATE-RQ.
func TestWithRoleSelectionAccumulatesPerSOPClass(t *testing.T) {
	var cfg associateConfig
	opts := []AssociateOption{
		WithRoleSelection(RoleSelection{SOPClassUID: ctImageStorageUID, SCURole: true, SCPRole: true}),
		WithRoleSelection(RoleSelection{SOPClassUID: dicom.SOPClassUID("1.2.840.10008.1.1"), SCURole: true}),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if len(cfg.roleSelections) != 2 {
		t.Fatalf("accumulated role selections = %d, want 2", len(cfg.roleSelections))
	}
	if cfg.roleSelections[0].SOPClassUID != string(ctImageStorageUID) {
		t.Errorf("first role SOP Class = %q, want %q", cfg.roleSelections[0].SOPClassUID, ctImageStorageUID)
	}
	if !cfg.roleSelections[0].SCPRole {
		t.Error("first role SCP flag not carried into the request config")
	}
	if cfg.roleSelections[1].SCPRole {
		t.Error("second role granted an SCP it did not request")
	}
}
