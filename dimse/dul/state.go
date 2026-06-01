package dul

import "fmt"

// State is a DUL state, Sta1..Sta13 (PS3.8 Table 9-10). The rewrite carries all
// thirteen, including the release-collision states Sta9..Sta12 the prototype omitted
// (Codex DIMSE-010).
type State uint8

const (
	Sta1  State = iota + 1 // Idle
	Sta2                   // Transport connection open (awaiting A-ASSOCIATE-RQ)
	Sta3                   // Awaiting local A-ASSOCIATE response
	Sta4                   // Awaiting transport connection open to complete
	Sta5                   // Awaiting A-ASSOCIATE-AC or A-ASSOCIATE-RJ
	Sta6                   // Association established, ready for data transfer
	Sta7                   // Awaiting A-RELEASE-RP
	Sta8                   // Awaiting local A-RELEASE response
	Sta9                   // Release collision, requestor: awaiting local A-RELEASE response
	Sta10                  // Release collision, acceptor: awaiting A-RELEASE-RP
	Sta11                  // Release collision, requestor: awaiting A-RELEASE-RP
	Sta12                  // Release collision, acceptor: awaiting local A-RELEASE response
	Sta13                  // Awaiting transport connection close
)

var stateNames = map[State]string{
	Sta1:  "Sta1",
	Sta2:  "Sta2",
	Sta3:  "Sta3",
	Sta4:  "Sta4",
	Sta5:  "Sta5",
	Sta6:  "Sta6",
	Sta7:  "Sta7",
	Sta8:  "Sta8",
	Sta9:  "Sta9",
	Sta10: "Sta10",
	Sta11: "Sta11",
	Sta12: "Sta12",
	Sta13: "Sta13",
}

// String renders the canonical state name (PHI-free, PRD §8.2).
func (s State) String() string {
	if name, ok := stateNames[s]; ok {
		return name
	}
	return fmt.Sprintf("unknown-state(%d)", uint8(s))
}
