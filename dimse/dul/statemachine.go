package dul

import "sync"

// transition keys the state-machine table by the current state and the incoming event.
type transition struct {
	state State
	event Evt
}

// outcome is the table value: the action to perform and the resulting state.
type outcome struct {
	action Action
	next   State
}

// table is the data model of PS3.8 Table 9-10. Modelling the machine as data rather
// than a deep switch makes completeness auditable (iterate states × events) and keeps
// it faithful to the standard. Verified against the pynetdicom fsm.py TRANSITION_TABLE,
// the project's named reference implementation.
//
// AR-8 collision-branch note: PS3.8 routes AR-8 by who originated the release. The only
// path into Sta7 is requestor-initiated (we sent A-RELEASE-RQ via AR-1), so Sta7 + Evt12
// resolves to Sta9. The acceptor counterpart Sta10 is entered acceptor-side and leads to
// Sta12 via AR-10; both collision halves are present and reachable (Codex DIMSE-010).
var table = map[transition]outcome{
	// Sta1 — Idle.
	{Sta1, Evt1}: {AE1, Sta4},
	{Sta1, Evt5}: {AE5, Sta2},

	// Sta2 — transport connection open, awaiting A-ASSOCIATE-RQ.
	{Sta2, Evt3}:  {AA1, Sta13},
	{Sta2, Evt4}:  {AA1, Sta13},
	{Sta2, Evt6}:  {AE6, Sta3},
	{Sta2, Evt10}: {AA1, Sta13},
	{Sta2, Evt12}: {AA1, Sta13},
	{Sta2, Evt13}: {AA1, Sta13},
	{Sta2, Evt16}: {AA2, Sta1},
	{Sta2, Evt17}: {AA5, Sta1},
	{Sta2, Evt18}: {AA2, Sta1},
	{Sta2, Evt19}: {AA1, Sta13},

	// Sta3 — awaiting local A-ASSOCIATE response.
	{Sta3, Evt3}:  {AA8, Sta13},
	{Sta3, Evt4}:  {AA8, Sta13},
	{Sta3, Evt6}:  {AA8, Sta13},
	{Sta3, Evt7}:  {AE7, Sta6},
	{Sta3, Evt8}:  {AE8, Sta13},
	{Sta3, Evt10}: {AA8, Sta13},
	{Sta3, Evt12}: {AA8, Sta13},
	{Sta3, Evt13}: {AA8, Sta13},
	{Sta3, Evt15}: {AA1, Sta13},
	{Sta3, Evt17}: {AA4, Sta1},
	{Sta3, Evt19}: {AA8, Sta13},

	// Sta4 — awaiting transport connection open to complete.
	{Sta4, Evt2}:  {AE2, Sta5},
	{Sta4, Evt15}: {AA2, Sta1},

	// Sta5 — awaiting A-ASSOCIATE-AC or A-ASSOCIATE-RJ.
	{Sta5, Evt3}:  {AE3, Sta6},
	{Sta5, Evt4}:  {AE4, Sta1},
	{Sta5, Evt6}:  {AA8, Sta13},
	{Sta5, Evt10}: {AA8, Sta13},
	{Sta5, Evt12}: {AA8, Sta13},
	{Sta5, Evt13}: {AA8, Sta13},
	{Sta5, Evt15}: {AA1, Sta13},
	{Sta5, Evt16}: {AA3, Sta1},
	{Sta5, Evt17}: {AA4, Sta1},
	{Sta5, Evt19}: {AA8, Sta13},

	// Sta6 — association established, ready for data transfer.
	{Sta6, Evt6}:  {AA8, Sta13},
	{Sta6, Evt9}:  {DT1, Sta6},
	{Sta6, Evt10}: {DT2, Sta6},
	{Sta6, Evt11}: {AR1, Sta7},
	{Sta6, Evt12}: {AR2, Sta8},
	{Sta6, Evt13}: {AA8, Sta13},
	{Sta6, Evt15}: {AA1, Sta13},
	{Sta6, Evt16}: {AA3, Sta1},
	{Sta6, Evt17}: {AA4, Sta1},
	{Sta6, Evt19}: {AA8, Sta13},

	// Sta7 — awaiting A-RELEASE-RP.
	{Sta7, Evt3}:  {AA8, Sta13},
	{Sta7, Evt6}:  {AA8, Sta13},
	{Sta7, Evt10}: {AR6, Sta7},
	{Sta7, Evt12}: {AR8, Sta9},
	{Sta7, Evt13}: {AR3, Sta1},
	{Sta7, Evt15}: {AA1, Sta13},
	{Sta7, Evt16}: {AA3, Sta1},
	{Sta7, Evt17}: {AA4, Sta1},
	{Sta7, Evt19}: {AA8, Sta13},

	// Sta8 — awaiting local A-RELEASE response.
	{Sta8, Evt3}:  {AA8, Sta13},
	{Sta8, Evt4}:  {AA8, Sta13},
	{Sta8, Evt6}:  {AA8, Sta13},
	{Sta8, Evt9}:  {AR7, Sta8},
	{Sta8, Evt10}: {AA8, Sta13},
	{Sta8, Evt12}: {AA8, Sta13},
	{Sta8, Evt14}: {AR4, Sta13},
	{Sta8, Evt15}: {AA1, Sta13},
	{Sta8, Evt16}: {AA3, Sta1},
	{Sta8, Evt17}: {AA4, Sta1},
	{Sta8, Evt19}: {AA8, Sta13},

	// Sta9 — release collision, requestor awaiting local A-RELEASE response.
	{Sta9, Evt3}:  {AA8, Sta13},
	{Sta9, Evt14}: {AR9, Sta11},
	{Sta9, Evt15}: {AA1, Sta13},
	{Sta9, Evt16}: {AA3, Sta1},
	{Sta9, Evt17}: {AA4, Sta1},
	{Sta9, Evt19}: {AA8, Sta13},

	// Sta10 — release collision, acceptor awaiting A-RELEASE-RP.
	{Sta10, Evt3}:  {AA8, Sta13},
	{Sta10, Evt13}: {AR10, Sta12},
	{Sta10, Evt15}: {AA1, Sta13},
	{Sta10, Evt16}: {AA3, Sta1},
	{Sta10, Evt17}: {AA4, Sta1},
	{Sta10, Evt19}: {AA8, Sta13},

	// Sta11 — release collision, requestor awaiting A-RELEASE-RP.
	{Sta11, Evt3}:  {AA8, Sta13},
	{Sta11, Evt13}: {AR3, Sta1},
	{Sta11, Evt15}: {AA1, Sta13},
	{Sta11, Evt16}: {AA3, Sta1},
	{Sta11, Evt17}: {AA4, Sta1},
	{Sta11, Evt19}: {AA8, Sta13},

	// Sta12 — release collision, acceptor awaiting local A-RELEASE response.
	{Sta12, Evt3}:  {AA8, Sta13},
	{Sta12, Evt14}: {AR4, Sta13},
	{Sta12, Evt15}: {AA1, Sta13},
	{Sta12, Evt16}: {AA3, Sta1},
	{Sta12, Evt17}: {AA4, Sta1},
	{Sta12, Evt19}: {AA8, Sta13},

	// Sta13 — awaiting transport connection close.
	{Sta13, Evt3}:  {AA6, Sta13},
	{Sta13, Evt4}:  {AA6, Sta13},
	{Sta13, Evt6}:  {AA7, Sta13},
	{Sta13, Evt10}: {AA6, Sta13},
	{Sta13, Evt12}: {AA6, Sta13},
	{Sta13, Evt13}: {AA6, Sta13},
	{Sta13, Evt16}: {AA2, Sta1},
	{Sta13, Evt17}: {AR5, Sta1},
	{Sta13, Evt18}: {AA2, Sta1},
	{Sta13, Evt19}: {AA7, Sta13},
}

// StateMachine is the DUL finite state machine. It holds only the current state; it
// performs no I/O (the connection drives it). Methods are safe for concurrent use so an
// observer can read the state while the owning goroutine advances it.
type StateMachine struct {
	mu    sync.RWMutex
	state State
}

// NewStateMachine returns a state machine in Sta1 (Idle).
func NewStateMachine() *StateMachine {
	return &StateMachine{state: Sta1}
}

// CurrentState returns the current DUL state.
func (sm *StateMachine) CurrentState() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state
}

// Apply looks up (current state, event) in PS3.8 Table 9-10, advances to the next
// state, and returns the action the caller must perform. An event with no defined
// transition is a protocol violation: Apply advances to Sta13, returns the AA-8 action,
// and a typed *StateError so the caller sends an A-ABORT instead of closing silently
// (Codex DIMSE-011). It never panics.
func (sm *StateMachine) Apply(event Evt) (Action, State, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	out, ok := table[transition{sm.state, event}]
	if !ok {
		from := sm.state
		sm.state = Sta13
		return AA8, Sta13, &StateError{State: from, Event: event}
	}
	sm.state = out.next
	return out.action, out.next, nil
}
