package dul_test

import (
	"testing"

	"github.com/codeninja55/go-radx/dimse/dul"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStateMachine_InitialState tests that state machine starts in Sta1
func TestStateMachine_InitialState(t *testing.T) {
	sm := dul.NewStateMachine()
	assert.Equal(t, dul.Sta1, sm.CurrentState())
}

// TestStateMachine_AssociationEstablishment tests SCU association establishment
func TestStateMachine_AssociationEstablishment(t *testing.T) {
	sm := dul.NewStateMachine()

	// Start in Sta1 (Idle)
	assert.Equal(t, dul.Sta1, sm.CurrentState())

	// AE-1: Transport connection confirmation
	_, err := sm.ProcessEvent(dul.AE1)
	require.NoError(t, err)
	assert.Equal(t, dul.Sta4, sm.CurrentState())

	// Note: In real implementation, Sta4 would transition based on local A-ASSOCIATE request
	// For this test, we'll simulate the sequence

	// AE-3: A-ASSOCIATE request
	action, err := sm.ProcessEvent(dul.AE3)
	require.NoError(t, err)
	assert.Equal(t, dul.ActionSendAssociateRQ, action)
	assert.Equal(t, dul.Sta5, sm.CurrentState())

	// AE-6: A-ASSOCIATE-AC received
	_, err = sm.ProcessEvent(dul.AE6)
	require.NoError(t, err)
	assert.Equal(t, dul.Sta6, sm.CurrentState()) // Association established
}

// TestStateMachine_AssociationAcceptance tests SCP association acceptance
func TestStateMachine_AssociationAcceptance(t *testing.T) {
	sm := dul.NewStateMachine()

	// Start in Sta1 (Idle)
	assert.Equal(t, dul.Sta1, sm.CurrentState())

	// AE-2: Transport connection indication
	_, err := sm.ProcessEvent(dul.AE2)
	require.NoError(t, err)
	assert.Equal(t, dul.Sta2, sm.CurrentState())

	// AE-8: A-ASSOCIATE indication (received A-ASSOCIATE-RQ)
	_, err = sm.ProcessEvent(dul.AE8)
	require.NoError(t, err)
	assert.Equal(t, dul.Sta3, sm.CurrentState())

	// AE-4: A-ASSOCIATE response (accept)
	action, err := sm.ProcessEvent(dul.AE4)
	require.NoError(t, err)
	assert.Equal(t, dul.ActionSendAssociateAC, action)
	assert.Equal(t, dul.Sta6, sm.CurrentState()) // Association established
}

// TestStateMachine_AssociationRejection tests association rejection
func TestStateMachine_AssociationRejection(t *testing.T) {
	sm := dul.NewStateMachine()

	// Get to Sta3 (awaiting local response)
	sm.ProcessEvent(dul.AE2) // Sta2
	sm.ProcessEvent(dul.AE8) // Sta3

	// AE-5: A-ASSOCIATE response (reject)
	action, err := sm.ProcessEvent(dul.AE5)
	require.NoError(t, err)
	assert.Equal(t, dul.ActionSendAssociateRJ, action)
	assert.Equal(t, dul.Sta13, sm.CurrentState()) // Awaiting transport close
}

// TestStateMachine_AssociationRelease tests graceful association release
func TestStateMachine_AssociationRelease(t *testing.T) {
	sm := dul.NewStateMachine()

	// Establish association first
	sm.ProcessEvent(dul.AE1) // Sta4
	sm.ProcessEvent(dul.AE3) // Sta5
	sm.ProcessEvent(dul.AE6) // Sta6

	assert.Equal(t, dul.Sta6, sm.CurrentState())

	// AE-11: A-RELEASE request
	action, err := sm.ProcessEvent(dul.AE11)
	require.NoError(t, err)
	assert.Equal(t, dul.ActionSendReleaseRQ, action)
	assert.Equal(t, dul.Sta7, sm.CurrentState())

	// AE-13: A-RELEASE-RP received
	_, err = sm.ProcessEvent(dul.AE13)
	require.NoError(t, err)
	assert.Equal(t, dul.Sta1, sm.CurrentState()) // Back to idle
}

// TestStateMachine_AssociationAbort tests association abort
func TestStateMachine_AssociationAbort(t *testing.T) {
	sm := dul.NewStateMachine()

	// Establish association
	sm.ProcessEvent(dul.AE1)
	sm.ProcessEvent(dul.AE3)
	sm.ProcessEvent(dul.AE6)

	assert.Equal(t, dul.Sta6, sm.CurrentState())

	// AE-15: A-ABORT request
	action, err := sm.ProcessEvent(dul.AE15)
	require.NoError(t, err)
	assert.Equal(t, dul.ActionSendAbort, action)
	assert.Equal(t, dul.Sta13, sm.CurrentState())
}

// TestStateMachine_DataTransfer tests P-DATA transfer in established association
func TestStateMachine_DataTransfer(t *testing.T) {
	sm := dul.NewStateMachine()

	// Establish association
	sm.ProcessEvent(dul.AE1)
	sm.ProcessEvent(dul.AE3)
	sm.ProcessEvent(dul.AE6)

	assert.Equal(t, dul.Sta6, sm.CurrentState())

	// AE-9: P-DATA request
	action, err := sm.ProcessEvent(dul.AE9)
	require.NoError(t, err)
	assert.Equal(t, dul.ActionSendData, action)
	assert.Equal(t, dul.Sta6, sm.CurrentState()) // Remain in Sta6

	// AE-10: P-DATA indication (received data)
	_, err = sm.ProcessEvent(dul.AE10)
	require.NoError(t, err)
	assert.Equal(t, dul.Sta6, sm.CurrentState()) // Remain in Sta6
}

// TestStateMachine_InvalidTransition tests invalid state transitions
func TestStateMachine_InvalidTransition(t *testing.T) {
	sm := dul.NewStateMachine()

	// Try to send data in Sta1 (not associated)
	_, err := sm.ProcessEvent(dul.AE9)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transition")

	// Try to release in Sta1 (not associated)
	_, err = sm.ProcessEvent(dul.AE11)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transition")
}

// TestStateMachine_UnexpectedPDU tests handling of unexpected PDUs
func TestStateMachine_UnexpectedPDU(t *testing.T) {
	sm := dul.NewStateMachine()

	// Establish association
	sm.ProcessEvent(dul.AE1)
	sm.ProcessEvent(dul.AE3)
	sm.ProcessEvent(dul.AE6)

	// AE-19: Unrecognized/unexpected PDU
	action, err := sm.ProcessEvent(dul.AE19)
	require.NoError(t, err)
	assert.Equal(t, dul.ActionSendAbort, action)
	assert.Equal(t, dul.Sta13, sm.CurrentState())
}

// TestStateMachine_TransportConnectionClosed tests transport connection closure
func TestStateMachine_TransportConnectionClosed(t *testing.T) {
	sm := dul.NewStateMachine()

	// Establish association
	sm.ProcessEvent(dul.AE1)
	sm.ProcessEvent(dul.AE3)
	sm.ProcessEvent(dul.AE6)

	assert.Equal(t, dul.Sta6, sm.CurrentState())

	// AE-17: Transport connection closed
	_, err := sm.ProcessEvent(dul.AE17)
	require.NoError(t, err)
	assert.Equal(t, dul.Sta1, sm.CurrentState()) // Back to idle
}

// TestStateMachine_Concurrency tests thread-safe state transitions
func TestStateMachine_Concurrency(t *testing.T) {
	sm := dul.NewStateMachine()

	// Establish association
	sm.ProcessEvent(dul.AE1)
	sm.ProcessEvent(dul.AE3)
	sm.ProcessEvent(dul.AE6)

	// Concurrent data transfers
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := sm.ProcessEvent(dul.AE9)
			assert.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should still be in Sta6
	assert.Equal(t, dul.Sta6, sm.CurrentState())
}

// TestStateMachine_FullLifecycle tests complete association lifecycle
func TestStateMachine_FullLifecycle(t *testing.T) {
	sm := dul.NewStateMachine()

	// 1. Start idle
	assert.Equal(t, dul.Sta1, sm.CurrentState())

	// 2. Connect
	sm.ProcessEvent(dul.AE1)
	assert.Equal(t, dul.Sta4, sm.CurrentState())

	// 3. Request association
	sm.ProcessEvent(dul.AE3)
	assert.Equal(t, dul.Sta5, sm.CurrentState())

	// 4. Association accepted
	sm.ProcessEvent(dul.AE6)
	assert.Equal(t, dul.Sta6, sm.CurrentState())

	// 5. Transfer data
	sm.ProcessEvent(dul.AE9)
	assert.Equal(t, dul.Sta6, sm.CurrentState())

	sm.ProcessEvent(dul.AE10)
	assert.Equal(t, dul.Sta6, sm.CurrentState())

	// 6. Request release
	sm.ProcessEvent(dul.AE11)
	assert.Equal(t, dul.Sta7, sm.CurrentState())

	// 7. Release confirmed
	sm.ProcessEvent(dul.AE13)
	assert.Equal(t, dul.Sta1, sm.CurrentState())
}

// TestStateMachine_SCPLifecycle tests SCP (server) lifecycle
func TestStateMachine_SCPLifecycle(t *testing.T) {
	sm := dul.NewStateMachine()

	// 1. Start idle
	assert.Equal(t, dul.Sta1, sm.CurrentState())

	// 2. Transport connection indication
	sm.ProcessEvent(dul.AE2)
	assert.Equal(t, dul.Sta2, sm.CurrentState())

	// 3. Receive A-ASSOCIATE-RQ
	sm.ProcessEvent(dul.AE8)
	assert.Equal(t, dul.Sta3, sm.CurrentState())

	// 4. Accept association
	sm.ProcessEvent(dul.AE4)
	assert.Equal(t, dul.Sta6, sm.CurrentState())

	// 5. Receive data
	sm.ProcessEvent(dul.AE10)
	assert.Equal(t, dul.Sta6, sm.CurrentState())

	// 6. Send response data
	sm.ProcessEvent(dul.AE9)
	assert.Equal(t, dul.Sta6, sm.CurrentState())

	// 7. Receive release request
	sm.ProcessEvent(dul.AE12)
	assert.Equal(t, dul.Sta8, sm.CurrentState())

	// 8. Send release response
	sm.ProcessEvent(dul.AE14)
	assert.Equal(t, dul.Sta13, sm.CurrentState())

	// 9. Transport close
	sm.ProcessEvent(dul.AE17)
	assert.Equal(t, dul.Sta1, sm.CurrentState())
}
