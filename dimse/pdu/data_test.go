package pdu

import "testing"

func TestPDVControlBits(t *testing.T) {
	cases := []struct {
		header            byte
		isCommand, isLast bool
	}{
		{0x00, false, false}, // dataset, more fragments
		{0x01, true, false},  // command, more fragments
		{0x02, false, true},  // dataset, last
		{0x03, true, true},   // command, last (the DIMSE-001 case)
	}
	for _, c := range cases {
		pdv := PresentationDataValue{MessageControlHeader: c.header}
		if pdv.IsCommand() != c.isCommand {
			t.Errorf("header %#02x IsCommand() = %v, want %v", c.header, pdv.IsCommand(), c.isCommand)
		}
		if pdv.IsLastFragment() != c.isLast {
			t.Errorf("header %#02x IsLastFragment() = %v, want %v", c.header, pdv.IsLastFragment(), c.isLast)
		}
	}
}

func TestMakeControlHeader(t *testing.T) {
	// A final command fragment is 0x03 regardless of whether a dataset follows.
	if got := MakeControlHeader(true, true); got != 0x03 {
		t.Errorf("MakeControlHeader(command=true, last=true) = %#02x, want 0x03", got)
	}
	if got := MakeControlHeader(false, true); got != 0x02 {
		t.Errorf("MakeControlHeader(command=false, last=true) = %#02x, want 0x02", got)
	}
	if got := MakeControlHeader(true, false); got != 0x01 {
		t.Errorf("MakeControlHeader(command=true, last=false) = %#02x, want 0x01", got)
	}
}
