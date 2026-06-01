package dul

import "testing"

func TestStateString(t *testing.T) {
	tests := map[State]string{
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
	for s, want := range tests {
		if got := s.String(); got != want {
			t.Errorf("State(%d).String() = %q, want %q", uint8(s), got, want)
		}
	}
}

// TestStateAllThirteenDistinct guards DIMSE-010: the rewrite must carry all thirteen
// states including the release-collision states Sta9..Sta12 the prototype omitted.
func TestStateAllThirteenDistinct(t *testing.T) {
	all := []State{Sta1, Sta2, Sta3, Sta4, Sta5, Sta6, Sta7, Sta8, Sta9, Sta10, Sta11, Sta12, Sta13}
	if len(all) != 13 {
		t.Fatalf("expected 13 states, listed %d", len(all))
	}
	seen := make(map[State]bool, len(all))
	for _, s := range all {
		if seen[s] {
			t.Errorf("duplicate state value %d", uint8(s))
		}
		seen[s] = true
	}
	// The collision states must be present and distinct from the established/release states.
	for _, s := range []State{Sta9, Sta10, Sta11, Sta12} {
		if !seen[s] {
			t.Errorf("release-collision state %v missing (DIMSE-010)", s)
		}
	}
}

func TestStateUnknownString(t *testing.T) {
	if got := State(0).String(); got == "Sta1" || got == "" {
		t.Errorf("State(0).String() = %q, want an unknown-state rendering", got)
	}
}
