package dul

import "fmt"

// Action is one of the 28 PS3.8 §9.2.3 state-machine actions, grouped into four
// families: association establishment (AE-1..AE-8), data transfer (DT-1, DT-2),
// association release (AR-1..AR-10, carrying the release-collision transitions), and
// abort (AA-1..AA-8). AA-8 is the "send A-ABORT (provider source), issue A-P-ABORT,
// start ARTIM" action an unexpected or invalid PDU drives (Codex DIMSE-011).
type Action uint8

const (
	ActionNone Action = iota

	// Association establishment.
	AE1 // Issue TRANSPORT CONNECT request to the transport service
	AE2 // Send A-ASSOCIATE-RQ PDU
	AE3 // Issue A-ASSOCIATE confirmation (accept) primitive
	AE4 // Issue A-ASSOCIATE confirmation (reject) and close transport connection
	AE5 // Issue TRANSPORT RESPONSE primitive and start ARTIM timer
	AE6 // Stop ARTIM; if A-ASSOCIATE-RQ acceptable issue indication, else reject
	AE7 // Send A-ASSOCIATE-AC PDU
	AE8 // Send A-ASSOCIATE-RJ PDU and start ARTIM timer

	// Data transfer.
	DT1 // Send P-DATA-TF PDU
	DT2 // Issue P-DATA indication primitive

	// Association release.
	AR1  // Send A-RELEASE-RQ PDU
	AR2  // Issue A-RELEASE indication primitive
	AR3  // Issue A-RELEASE confirmation primitive and close transport connection
	AR4  // Send A-RELEASE-RP PDU and start ARTIM timer
	AR5  // Stop ARTIM timer
	AR6  // Issue P-DATA indication primitive (release collision)
	AR7  // Send P-DATA-TF PDU (release collision)
	AR8  // Issue A-RELEASE indication (release collision)
	AR9  // Send A-RELEASE-RP PDU (release collision)
	AR10 // Issue A-RELEASE confirmation primitive (release collision)

	// Abort.
	AA1 // Send A-ABORT PDU (service-user source) and start ARTIM timer
	AA2 // Stop ARTIM timer if running and close transport connection
	AA3 // Issue abort indication and close transport connection
	AA4 // Issue A-P-ABORT indication primitive
	AA5 // Stop ARTIM timer
	AA6 // Ignore PDU
	AA7 // Send A-ABORT PDU
	AA8 // Send A-ABORT PDU (provider source), issue A-P-ABORT, start ARTIM timer
)

var actionNames = map[Action]string{
	ActionNone: "none",
	AE1:        "AE-1", AE2: "AE-2", AE3: "AE-3", AE4: "AE-4",
	AE5: "AE-5", AE6: "AE-6", AE7: "AE-7", AE8: "AE-8",
	DT1: "DT-1", DT2: "DT-2",
	AR1: "AR-1", AR2: "AR-2", AR3: "AR-3", AR4: "AR-4", AR5: "AR-5",
	AR6: "AR-6", AR7: "AR-7", AR8: "AR-8", AR9: "AR-9", AR10: "AR-10",
	AA1: "AA-1", AA2: "AA-2", AA3: "AA-3", AA4: "AA-4",
	AA5: "AA-5", AA6: "AA-6", AA7: "AA-7", AA8: "AA-8",
}

// String renders the canonical action identifier (PHI-free, PRD §8.2).
func (a Action) String() string {
	if name, ok := actionNames[a]; ok {
		return name
	}
	return fmt.Sprintf("unknown-action(%d)", uint8(a))
}
