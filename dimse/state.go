package dimse

import "github.com/codeninja55/go-radx/dimse/dul"

// State is a DUL state, Sta1..Sta13 (PS3.8 Table 9-10). It is an alias of the dul layer's
// State so there is a single source of truth for the lifecycle vocabulary; the root package
// re-exports the constants so callers need not import dul for observability.
type State = dul.State

// The thirteen DUL states, re-exported from dul (PS3.8 Table 9-10).
const (
	Sta1  = dul.Sta1  // Idle
	Sta2  = dul.Sta2  // Transport connection open (awaiting A-ASSOCIATE-RQ)
	Sta3  = dul.Sta3  // Awaiting local A-ASSOCIATE response
	Sta4  = dul.Sta4  // Awaiting transport connection open to complete
	Sta5  = dul.Sta5  // Awaiting A-ASSOCIATE-AC or A-ASSOCIATE-RJ
	Sta6  = dul.Sta6  // Association established, ready for data transfer
	Sta7  = dul.Sta7  // Awaiting A-RELEASE-RP
	Sta8  = dul.Sta8  // Awaiting local A-RELEASE response
	Sta9  = dul.Sta9  // Release collision, requestor: awaiting local A-RELEASE response
	Sta10 = dul.Sta10 // Release collision, acceptor: awaiting A-RELEASE-RP
	Sta11 = dul.Sta11 // Release collision, requestor: awaiting A-RELEASE-RP
	Sta12 = dul.Sta12 // Release collision, acceptor: awaiting local A-RELEASE response
	Sta13 = dul.Sta13 // Awaiting transport connection close
)
