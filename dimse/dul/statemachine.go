package dul

import (
	"fmt"
	"sync"
)

// State represents a state in the DICOM Upper Layer state machine
type State int

const (
	Sta1  State = iota + 1 // Idle
	Sta2                   // Transport connection open
	Sta3                   // Awaiting local A-ASSOCIATE response primitive
	Sta4                   // Awaiting transport connection opening
	Sta5                   // Awaiting A-ASSOCIATE-AC or A-ASSOCIATE-RJ PDU
	Sta6                   // Association established and ready for data transfer
	Sta7                   // Awaiting A-RELEASE-RP PDU
	Sta8                   // Awaiting local A-RELEASE response primitive
	Sta13                  // Awaiting transport connection close
)

// Event represents an event in the DICOM Upper Layer state machine
type Event int

const (
	AE1  Event = iota + 1 // Transport connect confirmation
	AE2                   // Transport connection indication
	AE3                   // A-ASSOCIATE request (local user)
	AE4                   // A-ASSOCIATE response primitive (accept)
	AE5                   // A-ASSOCIATE response primitive (reject)
	AE6                   // A-ASSOCIATE-AC PDU (received)
	AE7                   // A-ASSOCIATE-RJ PDU (received)
	AE8                   // A-ASSOCIATE-RQ PDU (received)
	AE9                   // P-DATA request primitive
	AE10                  // P-DATA-TF PDU (received)
	AE11                  // A-RELEASE request primitive
	AE12                  // A-RELEASE-RQ PDU (received)
	AE13                  // A-RELEASE-RP PDU (received)
	AE14                  // A-RELEASE response primitive
	AE15                  // A-ABORT request primitive
	AE16                  // A-ABORT PDU (received)
	AE17                  // Transport connection closed
	AE18                  // ARTIM timer expired
	AE19                  // Invalid PDU received
)

// Action represents an action to perform in response to an event
type Action int

const (
	ActionNone Action = iota
	ActionSendAssociateRQ
	ActionSendAssociateAC
	ActionSendAssociateRJ
	ActionSendData
	ActionSendReleaseRQ
	ActionSendReleaseRP
	ActionSendAbort
	ActionIssueAssociateConfirmation
	ActionIssueAssociateIndication
	ActionIssueDataIndication
	ActionIssueReleaseConfirmation
	ActionIssueReleaseIndication
	ActionCloseTransport
)

// StateMachine represents the DICOM Upper Layer state machine
type StateMachine struct {
	mu           sync.RWMutex
	currentState State
}

// NewStateMachine creates a new state machine in the idle state
func NewStateMachine() *StateMachine {
	return &StateMachine{
		currentState: Sta1,
	}
}

// CurrentState returns the current state
func (sm *StateMachine) CurrentState() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

// ProcessEvent processes an event and returns the action to perform
func (sm *StateMachine) ProcessEvent(event Event) (Action, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	nextState, action := sm.transition(sm.currentState, event)
	if nextState == 0 {
		return ActionNone, fmt.Errorf("invalid transition: state=%v event=%v", sm.currentState, event)
	}

	sm.currentState = nextState
	return action, nil
}

// transition defines the state machine transition table
func (sm *StateMachine) transition(state State, event Event) (State, Action) {
	switch state {
	case Sta1: // Idle
		switch event {
		case AE1: // Transport connect confirmation
			return Sta4, ActionNone
		case AE2: // Transport connection indication
			return Sta2, ActionNone
		case AE3: // A-ASSOCIATE request
			return Sta4, ActionSendAssociateRQ
		case AE5: // A-ASSOCIATE response (reject)
			return Sta1, ActionSendAssociateRJ
		}

	case Sta2: // Transport connection open (waiting for A-ASSOCIATE-RQ)
		switch event {
		case AE6: // A-ASSOCIATE-AC received (unexpected)
			return Sta13, ActionSendAbort
		case AE7: // A-ASSOCIATE-RJ received (unexpected)
			return Sta13, ActionSendAbort
		case AE8: // A-ASSOCIATE-RQ received
			return Sta3, ActionIssueAssociateIndication
		case AE15: // A-ABORT request
			return Sta13, ActionSendAbort
		case AE16: // A-ABORT received
			return Sta1, ActionCloseTransport
		case AE17: // Transport closed
			return Sta1, ActionNone
		case AE19: // Invalid PDU
			return Sta13, ActionSendAbort
		}

	case Sta3: // Awaiting local A-ASSOCIATE response
		switch event {
		case AE4: // A-ASSOCIATE response (accept)
			return Sta6, ActionSendAssociateAC
		case AE5: // A-ASSOCIATE response (reject)
			return Sta13, ActionSendAssociateRJ
		case AE15: // A-ABORT request
			return Sta13, ActionSendAbort
		case AE16: // A-ABORT received
			return Sta1, ActionCloseTransport
		case AE17: // Transport closed
			return Sta1, ActionNone
		}

	case Sta4: // Awaiting transport connection opening
		switch event {
		case AE1: // Transport connect confirmation
			return Sta5, ActionSendAssociateRQ
		case AE3: // A-ASSOCIATE request (when transport already open)
			return Sta5, ActionSendAssociateRQ
		case AE15: // A-ABORT request
			return Sta1, ActionCloseTransport
		case AE17: // Transport closed
			return Sta1, ActionNone
		}

	case Sta5: // Awaiting A-ASSOCIATE-AC or A-ASSOCIATE-RJ
		switch event {
		case AE6: // A-ASSOCIATE-AC received
			return Sta6, ActionIssueAssociateConfirmation
		case AE7: // A-ASSOCIATE-RJ received
			return Sta1, ActionCloseTransport
		case AE15: // A-ABORT request
			return Sta13, ActionSendAbort
		case AE16: // A-ABORT received
			return Sta1, ActionCloseTransport
		case AE17: // Transport closed
			return Sta1, ActionNone
		case AE19: // Invalid PDU
			return Sta13, ActionSendAbort
		}

	case Sta6: // Association established
		switch event {
		case AE9: // P-DATA request
			return Sta6, ActionSendData
		case AE10: // P-DATA-TF received
			return Sta6, ActionIssueDataIndication
		case AE11: // A-RELEASE request
			return Sta7, ActionSendReleaseRQ
		case AE12: // A-RELEASE-RQ received
			return Sta8, ActionIssueReleaseIndication
		case AE15: // A-ABORT request
			return Sta13, ActionSendAbort
		case AE16: // A-ABORT received
			return Sta1, ActionCloseTransport
		case AE17: // Transport closed
			return Sta1, ActionNone
		case AE19: // Invalid PDU
			return Sta13, ActionSendAbort
		}

	case Sta7: // Awaiting A-RELEASE-RP
		switch event {
		case AE10: // P-DATA-TF received (ignore during release)
			return Sta7, ActionNone
		case AE12: // A-RELEASE-RQ received (collision)
			return Sta7, ActionSendReleaseRP
		case AE13: // A-RELEASE-RP received
			return Sta1, ActionCloseTransport
		case AE15: // A-ABORT request
			return Sta13, ActionSendAbort
		case AE16: // A-ABORT received
			return Sta1, ActionCloseTransport
		case AE17: // Transport closed
			return Sta1, ActionNone
		case AE18: // ARTIM timer expired (timeout waiting for release)
			return Sta13, ActionSendAbort
		case AE19: // Invalid PDU
			return Sta13, ActionSendAbort
		}

	case Sta8: // Awaiting local A-RELEASE response
		switch event {
		case AE10: // P-DATA-TF received (ignore)
			return Sta8, ActionNone
		case AE11: // A-RELEASE request (collision)
			return Sta13, ActionSendReleaseRQ
		case AE14: // A-RELEASE response
			return Sta13, ActionSendReleaseRP
		case AE15: // A-ABORT request
			return Sta13, ActionSendAbort
		case AE16: // A-ABORT received
			return Sta1, ActionCloseTransport
		case AE17: // Transport closed
			return Sta1, ActionNone
		case AE19: // Invalid PDU
			return Sta13, ActionSendAbort
		}

	case Sta13: // Awaiting transport connection close
		if event == AE17 { // Transport closed
			return Sta1, ActionNone
		}
	}

	// Invalid transition
	return 0, ActionNone
}

// String representations
func (s State) String() string {
	switch s {
	case Sta1:
		return "Sta1 (Idle)"
	case Sta2:
		return "Sta2 (Transport Open)"
	case Sta3:
		return "Sta3 (Awaiting Local Associate Response)"
	case Sta4:
		return "Sta4 (Awaiting Transport Opening)"
	case Sta5:
		return "Sta5 (Awaiting Associate AC/RJ)"
	case Sta6:
		return "Sta6 (Association Established)"
	case Sta7:
		return "Sta7 (Awaiting Release RP)"
	case Sta8:
		return "Sta8 (Awaiting Local Release Response)"
	case Sta13:
		return "Sta13 (Awaiting Transport Close)"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

func (e Event) String() string {
	switch e {
	case AE1:
		return "AE-1 (Transport Connect Confirmation)"
	case AE2:
		return "AE-2 (Transport Connection Indication)"
	case AE3:
		return "AE-3 (A-ASSOCIATE Request)"
	case AE4:
		return "AE-4 (A-ASSOCIATE Response Accept)"
	case AE5:
		return "AE-5 (A-ASSOCIATE Response Reject)"
	case AE6:
		return "AE-6 (A-ASSOCIATE-AC PDU)"
	case AE7:
		return "AE-7 (A-ASSOCIATE-RJ PDU)"
	case AE8:
		return "AE-8 (A-ASSOCIATE-RQ PDU)"
	case AE9:
		return "AE-9 (P-DATA Request)"
	case AE10:
		return "AE-10 (P-DATA-TF PDU)"
	case AE11:
		return "AE-11 (A-RELEASE Request)"
	case AE12:
		return "AE-12 (A-RELEASE-RQ PDU)"
	case AE13:
		return "AE-13 (A-RELEASE-RP PDU)"
	case AE14:
		return "AE-14 (A-RELEASE Response)"
	case AE15:
		return "AE-15 (A-ABORT Request)"
	case AE16:
		return "AE-16 (A-ABORT PDU)"
	case AE17:
		return "AE-17 (Transport Closed)"
	case AE18:
		return "AE-18 (ARTIM Timer Expired)"
	case AE19:
		return "AE-19 (Invalid PDU)"
	default:
		return fmt.Sprintf("Unknown(%d)", e)
	}
}

func (a Action) String() string {
	switch a {
	case ActionNone:
		return "None"
	case ActionSendAssociateRQ:
		return "Send A-ASSOCIATE-RQ"
	case ActionSendAssociateAC:
		return "Send A-ASSOCIATE-AC"
	case ActionSendAssociateRJ:
		return "Send A-ASSOCIATE-RJ"
	case ActionSendData:
		return "Send P-DATA-TF"
	case ActionSendReleaseRQ:
		return "Send A-RELEASE-RQ"
	case ActionSendReleaseRP:
		return "Send A-RELEASE-RP"
	case ActionSendAbort:
		return "Send A-ABORT"
	case ActionIssueAssociateConfirmation:
		return "Issue A-ASSOCIATE Confirmation"
	case ActionIssueAssociateIndication:
		return "Issue A-ASSOCIATE Indication"
	case ActionIssueDataIndication:
		return "Issue P-DATA Indication"
	case ActionIssueReleaseConfirmation:
		return "Issue A-RELEASE Confirmation"
	case ActionIssueReleaseIndication:
		return "Issue A-RELEASE Indication"
	case ActionCloseTransport:
		return "Close Transport"
	default:
		return fmt.Sprintf("Unknown(%d)", a)
	}
}
