package dul

import (
	"errors"
	"testing"
)

// forceState sets the machine to a state directly so a single Table 9-10 cell can be
// exercised without replaying the path that reaches it. Test-only.
func (sm *StateMachine) forceState(s State) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state = s
}

// transitionCase is one authoritative PS3.8 Table 9-10 cell (verified against the
// pynetdicom fsm.py TRANSITION_TABLE, the project's named reference implementation).
type transitionCase struct {
	state  State
	event  Evt
	action Action
	next   State
}

// table910 is the full set of documented transitions. Apply must reproduce each one.
var table910 = []transitionCase{
	// Sta1
	{Sta1, Evt1, AE1, Sta4},
	{Sta1, Evt5, AE5, Sta2},
	// Sta2
	{Sta2, Evt3, AA1, Sta13},
	{Sta2, Evt4, AA1, Sta13},
	{Sta2, Evt6, AE6, Sta3},
	{Sta2, Evt10, AA1, Sta13},
	{Sta2, Evt12, AA1, Sta13},
	{Sta2, Evt13, AA1, Sta13},
	{Sta2, Evt16, AA2, Sta1},
	{Sta2, Evt17, AA5, Sta1},
	{Sta2, Evt18, AA2, Sta1},
	{Sta2, Evt19, AA1, Sta13},
	// Sta3
	{Sta3, Evt3, AA8, Sta13},
	{Sta3, Evt4, AA8, Sta13},
	{Sta3, Evt6, AA8, Sta13},
	{Sta3, Evt7, AE7, Sta6},
	{Sta3, Evt8, AE8, Sta13},
	{Sta3, Evt10, AA8, Sta13},
	{Sta3, Evt12, AA8, Sta13},
	{Sta3, Evt13, AA8, Sta13},
	{Sta3, Evt15, AA1, Sta13},
	{Sta3, Evt17, AA4, Sta1},
	{Sta3, Evt19, AA8, Sta13},
	// Sta4
	{Sta4, Evt2, AE2, Sta5},
	{Sta4, Evt15, AA2, Sta1},
	// Sta5
	{Sta5, Evt3, AE3, Sta6},
	{Sta5, Evt4, AE4, Sta1},
	{Sta5, Evt6, AA8, Sta13},
	{Sta5, Evt10, AA8, Sta13},
	{Sta5, Evt12, AA8, Sta13},
	{Sta5, Evt13, AA8, Sta13},
	{Sta5, Evt15, AA1, Sta13},
	{Sta5, Evt16, AA3, Sta1},
	{Sta5, Evt17, AA4, Sta1},
	{Sta5, Evt19, AA8, Sta13},
	// Sta6
	{Sta6, Evt6, AA8, Sta13},
	{Sta6, Evt9, DT1, Sta6},
	{Sta6, Evt10, DT2, Sta6},
	{Sta6, Evt11, AR1, Sta7},
	{Sta6, Evt12, AR2, Sta8},
	{Sta6, Evt13, AA8, Sta13},
	{Sta6, Evt15, AA1, Sta13},
	{Sta6, Evt16, AA3, Sta1},
	{Sta6, Evt17, AA4, Sta1},
	{Sta6, Evt19, AA8, Sta13},
	// Sta7
	{Sta7, Evt3, AA8, Sta13},
	{Sta7, Evt6, AA8, Sta13},
	{Sta7, Evt10, AR6, Sta7},
	{Sta7, Evt12, AR8, Sta9}, // requestor-side collision (see resolution note)
	{Sta7, Evt13, AR3, Sta1},
	{Sta7, Evt15, AA1, Sta13},
	{Sta7, Evt16, AA3, Sta1},
	{Sta7, Evt17, AA4, Sta1},
	{Sta7, Evt19, AA8, Sta13},
	// Sta8
	{Sta8, Evt3, AA8, Sta13},
	{Sta8, Evt4, AA8, Sta13},
	{Sta8, Evt6, AA8, Sta13},
	{Sta8, Evt9, AR7, Sta8},
	{Sta8, Evt10, AA8, Sta13},
	{Sta8, Evt12, AA8, Sta13},
	{Sta8, Evt14, AR4, Sta13},
	{Sta8, Evt15, AA1, Sta13},
	{Sta8, Evt16, AA3, Sta1},
	{Sta8, Evt17, AA4, Sta1},
	{Sta8, Evt19, AA8, Sta13},
	// Sta9 (release collision, requestor)
	{Sta9, Evt3, AA8, Sta13},
	{Sta9, Evt14, AR9, Sta11},
	{Sta9, Evt15, AA1, Sta13},
	{Sta9, Evt16, AA3, Sta1},
	{Sta9, Evt17, AA4, Sta1},
	{Sta9, Evt19, AA8, Sta13},
	// Sta10 (release collision, acceptor)
	{Sta10, Evt3, AA8, Sta13},
	{Sta10, Evt13, AR10, Sta12},
	{Sta10, Evt15, AA1, Sta13},
	{Sta10, Evt16, AA3, Sta1},
	{Sta10, Evt17, AA4, Sta1},
	{Sta10, Evt19, AA8, Sta13},
	// Sta11 (release collision, requestor)
	{Sta11, Evt3, AA8, Sta13},
	{Sta11, Evt13, AR3, Sta1},
	{Sta11, Evt15, AA1, Sta13},
	{Sta11, Evt16, AA3, Sta1},
	{Sta11, Evt17, AA4, Sta1},
	{Sta11, Evt19, AA8, Sta13},
	// Sta12 (release collision, acceptor)
	{Sta12, Evt3, AA8, Sta13},
	{Sta12, Evt14, AR4, Sta13},
	{Sta12, Evt15, AA1, Sta13},
	{Sta12, Evt16, AA3, Sta1},
	{Sta12, Evt17, AA4, Sta1},
	{Sta12, Evt19, AA8, Sta13},
	// Sta13
	{Sta13, Evt3, AA6, Sta13},
	{Sta13, Evt4, AA6, Sta13},
	{Sta13, Evt6, AA7, Sta13},
	{Sta13, Evt10, AA6, Sta13},
	{Sta13, Evt12, AA6, Sta13},
	{Sta13, Evt13, AA6, Sta13},
	{Sta13, Evt16, AA2, Sta1},
	{Sta13, Evt17, AR5, Sta1},
	{Sta13, Evt18, AA2, Sta1},
	{Sta13, Evt19, AA7, Sta13},
}

func TestApplyEveryDocumentedTransition(t *testing.T) {
	for _, c := range table910 {
		sm := NewStateMachine() // starts in Sta1
		sm.forceState(c.state)
		action, next, err := sm.Apply(c.event)
		if err != nil {
			t.Errorf("Apply(%v, %v) unexpected error: %v", c.state, c.event, err)
			continue
		}
		if action != c.action || next != c.next {
			t.Errorf("Apply(%v, %v) = (%v -> %v), want (%v -> %v)",
				c.state, c.event, action, next, c.action, c.next)
		}
		if sm.CurrentState() != c.next {
			t.Errorf("after Apply(%v, %v) state = %v, want %v",
				c.state, c.event, sm.CurrentState(), c.next)
		}
	}
}

// TestApplyUnexpectedEventDrivesAA8 is the DIMSE-011 regression: an unexpected event in
// a state where it is not in Table 9-10 must drive AA-8 (protocol error -> A-ABORT) and a
// typed error, never a panic and never a silent close.
func TestApplyUnexpectedEventDrivesAA8(t *testing.T) {
	cases := []struct {
		state State
		event Evt
	}{
		{Sta6, Evt7}, // an A-ASSOCIATE accept response is meaningless once established
		{Sta6, Evt4}, // an A-ASSOCIATE-RJ once established
		{Sta7, Evt7}, // accept response while awaiting release-rp
		{Sta3, Evt9}, // a P-DATA request before the association is established
	}
	for _, c := range cases {
		sm := NewStateMachine()
		sm.forceState(c.state)
		action, next, err := sm.Apply(c.event)
		if err == nil {
			t.Errorf("Apply(%v, %v) = nil error, want a protocol-error", c.state, c.event)
		}
		var se *StateError
		if !errors.As(err, &se) {
			t.Errorf("Apply(%v, %v) error = %T, want *StateError", c.state, c.event, err)
		}
		if action != AA8 {
			t.Errorf("Apply(%v, %v) action = %v, want AA-8 (DIMSE-011)", c.state, c.event, action)
		}
		if next != Sta13 {
			t.Errorf("Apply(%v, %v) next = %v, want Sta13", c.state, c.event, next)
		}
	}
}

// TestApplyEvt19AlwaysAA8InActiveStates is the DIMSE-011 regression for the explicit
// Evt19 (invalid PDU) cells: every association/data-transfer/awaiting state turns an
// invalid PDU into AA-8, not a silent socket close (the prototype's defect).
func TestApplyEvt19AlwaysAA8InActiveStates(t *testing.T) {
	for _, s := range []State{Sta3, Sta5, Sta6, Sta7, Sta8, Sta9, Sta10, Sta11, Sta12} {
		sm := NewStateMachine()
		sm.forceState(s)
		action, next, err := sm.Apply(Evt19)
		if err != nil {
			t.Errorf("Apply(%v, Evt19) unexpected error: %v", s, err)
		}
		if action != AA8 || next != Sta13 {
			t.Errorf("Apply(%v, Evt19) = (%v -> %v), want (AA-8 -> Sta13)", s, action, next)
		}
	}
}

// TestReleaseCollisionReachesSta9AndSta12 is the DIMSE-010 regression: the
// release-collision states the prototype lacked are reachable. The requestor lands in
// Sta9 (Sta7 + Evt12 / AR-8), then Sta11; the acceptor collision lands in Sta12 via
// Sta10 + Evt13 / AR-10. (The plan's "Sta8->Sta12" shorthand is reconciled here to the
// authoritative PS3.8/pynetdicom table — see PLAN.md deviation note.)
func TestReleaseCollisionReachesSta9AndSta12(t *testing.T) {
	// Requestor path: Sta6 -> AR-1 -> Sta7, peer's A-RELEASE-RQ collides -> AR-8 -> Sta9.
	sm := NewStateMachine()
	sm.forceState(Sta6)
	if _, next, _ := sm.Apply(Evt11); next != Sta7 {
		t.Fatalf("Sta6 + Evt11 -> %v, want Sta7", next)
	}
	action, next, err := sm.Apply(Evt12)
	if err != nil || action != AR8 || next != Sta9 {
		t.Fatalf("Sta7 + Evt12 = (%v -> %v, err=%v), want (AR-8 -> Sta9)", action, next, err)
	}
	// Sta9 -> AR-9 -> Sta11 -> AR-3 -> Sta1.
	if _, next, _ = sm.Apply(Evt14); next != Sta11 {
		t.Fatalf("Sta9 + Evt14 -> %v, want Sta11", next)
	}
	if _, next, _ = sm.Apply(Evt13); next != Sta1 {
		t.Fatalf("Sta11 + Evt13 -> %v, want Sta1", next)
	}

	// Acceptor path: Sta10 -> AR-10 -> Sta12 -> AR-4 -> Sta13.
	sm2 := NewStateMachine()
	sm2.forceState(Sta10)
	if action, next, err = sm2.Apply(Evt13); err != nil || action != AR10 || next != Sta12 {
		t.Fatalf("Sta10 + Evt13 = (%v -> %v, err=%v), want (AR-10 -> Sta12)", action, next, err)
	}
	if action, next, _ = sm2.Apply(Evt14); action != AR4 || next != Sta13 {
		t.Fatalf("Sta12 + Evt14 = (%v -> %v), want (AR-4 -> Sta13)", action, next)
	}
}

// TestAcceptInboundTransportConnection guards the acceptor's entry transition that the
// original Increment 2 work omitted on the false belief that pynetdicom lacks it. PS3.8
// Table 9-10 and pynetdicom's fsm.py both define Sta1 + Evt5 (an accepted inbound TCP
// connection) -> AE-5 (issue transport response, start ARTIM timer) -> Sta2 (await the
// A-ASSOCIATE-RQ). Without it the SCP can never accept an association.
func TestAcceptInboundTransportConnection(t *testing.T) {
	sm := NewStateMachine() // starts in Sta1
	action, next, err := sm.Apply(Evt5)
	if err != nil {
		t.Fatalf("Apply(Sta1, Evt5) unexpected error: %v", err)
	}
	if action != AE5 {
		t.Errorf("Apply(Sta1, Evt5) action = %v, want AE-5", action)
	}
	if next != Sta2 {
		t.Errorf("Apply(Sta1, Evt5) next = %v, want Sta2", next)
	}
	if sm.CurrentState() != Sta2 {
		t.Errorf("after Apply(Sta1, Evt5) state = %v, want Sta2", sm.CurrentState())
	}
}

func TestNewStateMachineStartsIdle(t *testing.T) {
	if got := NewStateMachine().CurrentState(); got != Sta1 {
		t.Errorf("NewStateMachine() state = %v, want Sta1", got)
	}
}

// TestTransitionTableMatchesDocumentedCells audits the data model against the
// documented PS3.8 cells: every implemented transition must be one this test asserts,
// and every asserted cell must be implemented. This keeps the table auditable so a
// stray or missing cell is caught (the point of modelling the FSM as data).
func TestTransitionTableMatchesDocumentedCells(t *testing.T) {
	documented := make(map[transition]outcome, len(table910))
	for _, c := range table910 {
		documented[transition{c.state, c.event}] = outcome{c.action, c.next}
	}
	if len(documented) != len(table) {
		t.Errorf("documented cells = %d, implemented table cells = %d", len(documented), len(table))
	}
	for key, want := range documented {
		got, ok := table[key]
		if !ok {
			t.Errorf("table missing documented cell (%v, %v)", key.state, key.event)
			continue
		}
		if got != want {
			t.Errorf("table[%v,%v] = (%v -> %v), documented (%v -> %v)",
				key.state, key.event, got.action, got.next, want.action, want.next)
		}
	}
	for key := range table {
		if _, ok := documented[key]; !ok {
			t.Errorf("table has undocumented cell (%v, %v)", key.state, key.event)
		}
	}
}
