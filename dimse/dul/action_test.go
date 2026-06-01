package dul

import "testing"

// TestActionString checks the 28 PS3.8 actions render their canonical identifiers.
func TestActionString(t *testing.T) {
	tests := map[Action]string{
		AE1: "AE-1", AE2: "AE-2", AE3: "AE-3", AE4: "AE-4",
		AE5: "AE-5", AE6: "AE-6", AE7: "AE-7", AE8: "AE-8",
		DT1: "DT-1", DT2: "DT-2",
		AR1: "AR-1", AR2: "AR-2", AR3: "AR-3", AR4: "AR-4", AR5: "AR-5",
		AR6: "AR-6", AR7: "AR-7", AR8: "AR-8", AR9: "AR-9", AR10: "AR-10",
		AA1: "AA-1", AA2: "AA-2", AA3: "AA-3", AA4: "AA-4",
		AA5: "AA-5", AA6: "AA-6", AA7: "AA-7", AA8: "AA-8",
	}
	if len(tests) != 28 {
		t.Fatalf("expected 28 actions, listed %d", len(tests))
	}
	seen := make(map[Action]bool, len(tests))
	for a, want := range tests {
		if seen[a] {
			t.Errorf("duplicate action value %d", uint8(a))
		}
		seen[a] = true
		if got := a.String(); got != want {
			t.Errorf("Action(%d).String() = %q, want %q", uint8(a), got, want)
		}
	}
}

func TestActionUnknownString(t *testing.T) {
	if got := Action(0).String(); got == "AE-1" || got == "" {
		t.Errorf("Action(0).String() = %q, want an unknown-action rendering", got)
	}
}
