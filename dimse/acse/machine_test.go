package acse

import (
	"context"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dimse/dul"
)

// TestRequestorMachineExposesStateMachine verifies the message layer can reach the
// established association's StateMachine to drive inbound P-DATA-TF reads through
// dul.DriveInbound. The accessor must return the same machine the requestor advances, so
// the inbound transition stays consistent with the association lifecycle.
func TestRequestorMachineExposesStateMachine(t *testing.T) {
	rqConn, acConn := loopback(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	acceptorDone := make(chan *Acceptor, 1)
	acceptErr := make(chan error, 1)
	go func() {
		a, err := Accept(ctx, acConn, acceptParams())
		if err != nil {
			acceptErr <- err
			return
		}
		acceptorDone <- a
	}()

	req, err := Associate(ctx, rqConn, echoRequest())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	select {
	case err := <-acceptErr:
		t.Fatalf("Accept: %v", err)
	case acc := <-acceptorDone:
		if acc.Machine() == nil {
			t.Error("Acceptor.Machine() = nil, want the acceptor's StateMachine")
		}
		if acc.Machine().CurrentState() != dul.Sta6 {
			t.Errorf("acceptor machine state = %v, want Sta6", acc.Machine().CurrentState())
		}
	}

	m := req.Machine()
	if m == nil {
		t.Fatal("Requestor.Machine() = nil, want the requestor's StateMachine")
	}
	if m.CurrentState() != dul.Sta6 {
		t.Errorf("requestor machine state = %v, want Sta6", m.CurrentState())
	}
	if m.CurrentState() != req.State() {
		t.Errorf("Machine().CurrentState() = %v, but State() = %v — must be the same machine",
			m.CurrentState(), req.State())
	}
}
