package dul

import (
	"strings"
	"testing"
)

// TestEventString guards DIMSE-011: events are named Evt1..Evt19 per PS3.8, never the
// prototype's AE1..AE19 mislabelling. The String() rendering must start with "Evt".
func TestEventString(t *testing.T) {
	tests := map[Evt]string{
		Evt1:  "Evt1",
		Evt2:  "Evt2",
		Evt3:  "Evt3",
		Evt4:  "Evt4",
		Evt5:  "Evt5",
		Evt6:  "Evt6",
		Evt7:  "Evt7",
		Evt8:  "Evt8",
		Evt9:  "Evt9",
		Evt10: "Evt10",
		Evt11: "Evt11",
		Evt12: "Evt12",
		Evt13: "Evt13",
		Evt14: "Evt14",
		Evt15: "Evt15",
		Evt16: "Evt16",
		Evt17: "Evt17",
		Evt18: "Evt18",
		Evt19: "Evt19",
	}
	for e, want := range tests {
		got := e.String()
		if !strings.HasPrefix(got, want) {
			t.Errorf("Evt(%d).String() = %q, want prefix %q", uint8(e), got, want)
		}
		if strings.HasPrefix(got, "AE") {
			t.Errorf("Evt(%d).String() = %q uses the prototype's AE naming (DIMSE-011)", uint8(e), got)
		}
	}
}

func TestEventAllNineteenDistinct(t *testing.T) {
	all := []Evt{
		Evt1, Evt2, Evt3, Evt4, Evt5, Evt6, Evt7, Evt8, Evt9, Evt10,
		Evt11, Evt12, Evt13, Evt14, Evt15, Evt16, Evt17, Evt18, Evt19,
	}
	if len(all) != 19 {
		t.Fatalf("expected 19 events, listed %d", len(all))
	}
	seen := make(map[Evt]bool, len(all))
	for _, e := range all {
		if seen[e] {
			t.Errorf("duplicate event value %d", uint8(e))
		}
		seen[e] = true
	}
}

func TestEventUnknownString(t *testing.T) {
	if got := Evt(0).String(); strings.HasPrefix(got, "Evt1") || got == "" {
		t.Errorf("Evt(0).String() = %q, want an unknown-event rendering", got)
	}
}
