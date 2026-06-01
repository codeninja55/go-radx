package dul

import "fmt"

// Evt is a DUL state-machine event, Evt1..Evt19 (PS3.8 Table 9-10). These are named
// per the standard; the prototype mislabelled them AE1..AE19 (Codex DIMSE-011).
type Evt uint8

const (
	Evt1  Evt = iota + 1 // A-ASSOCIATE request (local user)
	Evt2                 // Transport connection confirmation (local connect complete)
	Evt3                 // A-ASSOCIATE-AC PDU received
	Evt4                 // A-ASSOCIATE-RJ PDU received
	Evt5                 // Transport connection indication (remote connect accepted)
	Evt6                 // A-ASSOCIATE-RQ PDU received
	Evt7                 // A-ASSOCIATE response primitive (accept)
	Evt8                 // A-ASSOCIATE response primitive (reject)
	Evt9                 // P-DATA request primitive
	Evt10                // P-DATA-TF PDU received
	Evt11                // A-RELEASE request primitive
	Evt12                // A-RELEASE-RQ PDU received
	Evt13                // A-RELEASE-RP PDU received
	Evt14                // A-RELEASE response primitive
	Evt15                // A-ABORT request primitive
	Evt16                // A-ABORT PDU received
	Evt17                // Transport connection closed indication
	Evt18                // ARTIM timer expired
	Evt19                // Unrecognised or invalid PDU received
)

var eventDescriptions = map[Evt]string{
	Evt1:  "A-ASSOCIATE request",
	Evt2:  "transport connect confirmation",
	Evt3:  "A-ASSOCIATE-AC PDU received",
	Evt4:  "A-ASSOCIATE-RJ PDU received",
	Evt5:  "transport connection indication",
	Evt6:  "A-ASSOCIATE-RQ PDU received",
	Evt7:  "A-ASSOCIATE response (accept)",
	Evt8:  "A-ASSOCIATE response (reject)",
	Evt9:  "P-DATA request",
	Evt10: "P-DATA-TF PDU received",
	Evt11: "A-RELEASE request",
	Evt12: "A-RELEASE-RQ PDU received",
	Evt13: "A-RELEASE-RP PDU received",
	Evt14: "A-RELEASE response",
	Evt15: "A-ABORT request",
	Evt16: "A-ABORT PDU received",
	Evt17: "transport connection closed",
	Evt18: "ARTIM timer expired",
	Evt19: "invalid or unrecognised PDU received",
}

// String renders the canonical event name and description (PHI-free, PRD §8.2).
func (e Evt) String() string {
	if desc, ok := eventDescriptions[e]; ok {
		return fmt.Sprintf("Evt%d (%s)", uint8(e), desc)
	}
	return fmt.Sprintf("unknown-event(%d)", uint8(e))
}
