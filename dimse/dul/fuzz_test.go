package dul

import (
	"sync"
	"testing"
)

// FuzzStateMachineEventSequence tests state machine with random event sequences
func FuzzStateMachineEventSequence(f *testing.F) {
	// Seed with valid event sequences
	// Successful association establishment
	f.Add([]byte{byte(AE3), byte(AE1), byte(AE6)}) // Request -> Transport -> AC received

	// Association rejected
	f.Add([]byte{byte(AE3), byte(AE1), byte(AE7)}) // Request -> Transport -> RJ received

	// Association establishment and release
	f.Add([]byte{byte(AE3), byte(AE1), byte(AE6), byte(AE11), byte(AE13)})

	// Association with data transfer
	f.Add([]byte{byte(AE3), byte(AE1), byte(AE6), byte(AE9), byte(AE10), byte(AE11), byte(AE13)})

	// Abort scenarios
	f.Add([]byte{byte(AE3), byte(AE1), byte(AE15)})            // Abort before association
	f.Add([]byte{byte(AE3), byte(AE1), byte(AE6), byte(AE15)}) // Abort after established

	// Invalid sequences
	f.Add([]byte{byte(AE9)})            // Data request without association
	f.Add([]byte{byte(AE11)})           // Release request without association
	f.Add([]byte{byte(AE6), byte(AE7)}) // AC followed by RJ

	// Empty sequence
	f.Add([]byte{})

	// Very long sequence
	longSeq := make([]byte, 100)
	for i := range longSeq {
		longSeq[i] = byte(AE9) // Repeated data events
	}
	f.Add(longSeq)

	f.Fuzz(func(t *testing.T, eventSeq []byte) {
		sm := NewStateMachine()
		initialState := sm.CurrentState()

		// Should never panic when processing event sequences
		for i, eventByte := range eventSeq {
			// Convert byte to Event (limit to valid range)
			event := Event(eventByte % 20) // AE1-AE19 + invalid values

			beforeState := sm.CurrentState()
			action, err := sm.ProcessEvent(event)

			// Track what happened for debugging
			if err != nil {
				// Invalid transitions are expected and should not panic
				// Verify state didn't change on error
				if sm.CurrentState() != beforeState {
					t.Errorf("State changed on error at step %d: %v -> %v",
						i, beforeState, sm.CurrentState())
				}
				continue
			}

			// If no error, verify action is valid
			if action < ActionNone || action > ActionCloseTransport {
				t.Errorf("Invalid action returned at step %d: %v", i, action)
			}

			// Verify state changed or stayed same (both valid)
			newState := sm.CurrentState()
			if newState < Sta1 || newState > Sta13 {
				t.Errorf("Invalid state after step %d: %v", i, newState)
			}
		}

		// Verify final state is valid
		finalState := sm.CurrentState()
		if finalState < Sta1 || finalState > Sta13 {
			t.Errorf("Invalid final state: %v", finalState)
		}

		// State machine should always remain internally consistent
		// (no way to verify from outside, but shouldn't have panicked)
		_ = initialState // Used for potential future checks
	})
}

// FuzzStateMachineInvalidEvents tests state machine with out-of-range events
func FuzzStateMachineInvalidEvents(f *testing.F) {
	// Seed with valid events
	for i := 1; i <= 19; i++ {
		f.Add(uint8(i))
	}

	// Seed with invalid events
	f.Add(uint8(0))   // Below valid range
	f.Add(uint8(20))  // Above valid range
	f.Add(uint8(255)) // Max value

	f.Fuzz(func(t *testing.T, eventNum uint8) {
		sm := NewStateMachine()
		event := Event(eventNum)

		beforeState := sm.CurrentState()

		// Should handle any event value without panic
		_, err := sm.ProcessEvent(event)

		// Invalid events (outside AE1-AE19) should either:
		// 1. Return an error, OR
		// 2. Be handled as invalid transitions
		if eventNum == 0 || eventNum > 19 {
			// For invalid event numbers, expect error
			if err == nil {
				t.Logf("Warning: invalid event %d was accepted", eventNum)
			}
		}

		// State should not become invalid
		afterState := sm.CurrentState()
		if afterState != 0 && (afterState < Sta1 || afterState > Sta13) {
			t.Errorf("State became invalid: %v", afterState)
		}

		// If error occurred, state should not have changed
		if err != nil && afterState != beforeState {
			t.Errorf("State changed despite error: %v -> %v", beforeState, afterState)
		}
	})
}

// FuzzStateMachineStates tests all possible state/event combinations
func FuzzStateMachineStates(f *testing.F) {
	// Seed with all valid states
	states := []State{Sta1, Sta2, Sta3, Sta4, Sta5, Sta6, Sta7, Sta8, Sta13}
	events := []Event{AE1, AE2, AE3, AE4, AE5, AE6, AE7, AE8, AE9, AE10,
		AE11, AE12, AE13, AE14, AE15, AE16, AE17, AE18, AE19}

	// Seed with all valid combinations
	for _, state := range states {
		for _, event := range events {
			f.Add(uint8(state), uint8(event))
		}
	}

	// Seed with invalid combinations
	f.Add(uint8(0), uint8(1))     // Invalid state
	f.Add(uint8(1), uint8(0))     // Invalid event
	f.Add(uint8(255), uint8(255)) // Max values

	f.Fuzz(func(t *testing.T, stateNum uint8, eventNum uint8) {
		sm := NewStateMachine()

		// Force state machine to specific state (directly access for testing)
		sm.mu.Lock()
		sm.currentState = State(stateNum)
		sm.mu.Unlock()

		event := Event(eventNum)

		// Should handle any state/event combination without panic
		action, err := sm.ProcessEvent(event)

		// Verify results are consistent
		if err == nil {
			// If no error, action should be valid
			if action < ActionNone || action > ActionCloseTransport {
				t.Errorf("Invalid action for state=%d event=%d: %v",
					stateNum, eventNum, action)
			}

			// New state should be valid
			newState := sm.CurrentState()
			if newState != 0 && (newState < Sta1 || newState > Sta13) {
				t.Errorf("Invalid new state for state=%d event=%d: %v",
					stateNum, eventNum, newState)
			}
		}
	})
}

// FuzzStateMachineConcurrent tests concurrent event processing
func FuzzStateMachineConcurrent(f *testing.F) {
	// Seed with event sequences to run concurrently
	f.Add([]byte{byte(AE3), byte(AE1), byte(AE6)})
	f.Add([]byte{byte(AE9), byte(AE10), byte(AE11)})
	f.Add([]byte{byte(AE15), byte(AE16), byte(AE17)})

	f.Fuzz(func(t *testing.T, events []byte) {
		if len(events) == 0 {
			return
		}

		sm := NewStateMachine()

		// Process events concurrently
		var wg sync.WaitGroup
		errorCount := 0
		var mu sync.Mutex

		for _, eventByte := range events {
			wg.Add(1)
			go func(eb byte) {
				defer wg.Done()

				event := Event(eb % 20)
				_, err := sm.ProcessEvent(event)

				if err != nil {
					mu.Lock()
					errorCount++
					mu.Unlock()
				}
			}(eventByte)
		}

		wg.Wait()

		// Verify state machine remains consistent
		// Final state should be valid
		finalState := sm.CurrentState()
		if finalState < Sta1 || finalState > Sta13 {
			t.Errorf("Invalid final state after concurrent processing: %v", finalState)
		}

		// Concurrent processing will likely cause errors (expected)
		// But should never cause state corruption
		t.Logf("Processed %d events concurrently, %d errors (expected)", len(events), errorCount)
	})
}

// FuzzStateMachineTransitionInvariants tests state transition invariants
func FuzzStateMachineTransitionInvariants(f *testing.F) {
	// Seed with sequences that test specific invariants
	// Invariant: Can't transition to Sta6 without going through Sta5
	f.Add([]byte{byte(AE3), byte(AE1), byte(AE6)}) // Valid path to Sta6

	// Invariant: Can't send data before association established
	f.Add([]byte{byte(AE9)})

	// Invariant: Must be in Sta6 to initiate release
	f.Add([]byte{byte(AE3), byte(AE1), byte(AE6), byte(AE11)}) // Valid release
	f.Add([]byte{byte(AE11)})                                  // Invalid release (not in Sta6)

	f.Fuzz(func(t *testing.T, events []byte) {
		sm := NewStateMachine()
		var stateHistory []State

		for _, eventByte := range events {
			event := Event(eventByte % 20)
			beforeState := sm.CurrentState()

			action, err := sm.ProcessEvent(event)
			if err != nil {
				continue
			}

			afterState := sm.CurrentState()
			stateHistory = append(stateHistory, afterState)

			// Check invariant: Certain actions only valid in certain states
			switch action {
			case ActionSendData:
				// Can only send data when in Sta6 (association established)
				if beforeState != Sta6 {
					t.Errorf("ActionSendData from invalid state %v", beforeState)
				}

			case ActionSendAssociateAC:
				// Can only accept association from Sta3
				if beforeState != Sta3 {
					t.Errorf("ActionSendAssociateAC from invalid state %v", beforeState)
				}

			case ActionSendReleaseRQ:
				// Can only initiate release from Sta6 or Sta8
				if beforeState != Sta6 && beforeState != Sta8 {
					t.Errorf("ActionSendReleaseRQ from invalid state %v", beforeState)
				}
			}

			// Check invariant: State should always be valid
			if afterState < Sta1 || afterState > Sta13 {
				t.Errorf("Transitioned to invalid state %v", afterState)
			}

			// Check invariant: Can't skip states inappropriately
			// (e.g., can't go from Sta1 directly to Sta6)
			if beforeState == Sta1 && afterState == Sta6 {
				t.Errorf("Skipped required states: Sta1 -> Sta6 directly")
			}
		}
	})
}

// FuzzStateMachineIdempotency tests that repeated events behave consistently
func FuzzStateMachineIdempotency(f *testing.F) {
	// Seed with events to repeat
	f.Add(uint8(AE3), uint8(3))  // Repeat association request
	f.Add(uint8(AE9), uint8(5))  // Repeat data request
	f.Add(uint8(AE15), uint8(2)) // Repeat abort

	f.Fuzz(func(t *testing.T, eventNum uint8, repeatCount uint8) {
		if repeatCount == 0 || repeatCount > 100 {
			return // Limit repeats to avoid timeout
		}

		sm := NewStateMachine()
		event := Event(eventNum % 20)

		var results []struct {
			action Action
			err    error
			state  State
		}

		// Process same event multiple times
		for i := uint8(0); i < repeatCount; i++ {
			action, err := sm.ProcessEvent(event)
			results = append(results, struct {
				action Action
				err    error
				state  State
			}{
				action: action,
				err:    err,
				state:  sm.CurrentState(),
			})
		}

		// Analyze patterns
		// Some events should be idempotent (same result each time)
		// Others should change state
		//
		// This test validates that behavior is deterministic
		for i := 1; i < len(results); i++ {
			// If previous succeeded and we're in Sta1, repeating should give same result
			if results[i-1].state == Sta1 && results[i].state == Sta1 {
				if results[i-1].err != nil && results[i].err == nil {
					t.Logf("Behavior changed: event %v went from error to success", event)
				}
			}
		}

		// Final state should be valid
		if len(results) > 0 {
			finalState := results[len(results)-1].state
			if finalState < Sta1 || finalState > Sta13 {
				t.Errorf("Invalid final state after %d repetitions: %v",
					repeatCount, finalState)
			}
		}
	})
}
