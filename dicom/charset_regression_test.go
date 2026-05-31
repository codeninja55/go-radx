package dicom

import (
	"strings"
	"testing"
)

// DCM-011 (charset half): a non-default charset value round-trips byte-exact through
// decode and re-encode, and a multi-byte PersonName decodes its component groups.
// These are the named regressions for the character-set decoding the prototype
// ignored. The single-encoding and ISO 2022 cases live in
// specific_character_set_test.go and iso2022_test.go; this file covers the hostile
// and boundary inputs that must degrade to a typed error rather than panic (PRD §9).

func TestCharsetDecodeNeverPanicsOnHostileBytes(t *testing.T) {
	cs, err := NewSpecificCharacterSet("", "ISO 2022 IR 87")
	if err != nil {
		t.Fatalf("NewSpecificCharacterSet: %v", err)
	}
	hostile := [][]byte{
		{0x1b},                         // lone escape
		{0x1b, '$'},                    // truncated multi-byte designation
		{0x1b, '$', 'B', 0x21},         // odd-length double-byte run
		{0x1b, '(', 'Z'},               // unknown single-byte designation
		{0xff, 0xfe, 0xfd},             // junk bytes in double-byte context
		{0x1b, '$', '(', 'D'},          // designation with no following data
		[]byte(strings.Repeat("=", 8)), // all delimiters
	}
	for i, b := range hostile {
		// A failure to decode is acceptable; a panic is not. Recover defends the test
		// from a regression that reintroduces an unchecked index.
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("hostile input %d panicked: %v", i, r)
				}
			}()
			_, _ = cs.Decode(b)
		}()
	}
}

func TestCharsetDecodeUnknownEscapeIsError(t *testing.T) {
	// A value that designates a set the active character set never declared must be a
	// typed error, not a silent switch (single-byte family path).
	cs, err := NewSpecificCharacterSet("", "ISO 2022 IR 100")
	if err != nil {
		t.Fatalf("NewSpecificCharacterSet: %v", err)
	}
	// ESC ) something the configured set does not include.
	_, err = cs.Decode([]byte{'A', 0x1b, ')', 'Z', 0xe4})
	if err == nil {
		t.Fatal("Decode of an undeclared G1 designation = nil error, want error")
	}
}

func TestCharsetDecodeBoundsWithLargeInput(t *testing.T) {
	// A large default-charset value decodes without unbounded auxiliary allocation:
	// the field is already length-bounded by the reader before this point, and decode
	// allocates O(n) once. This asserts behavioural correctness on a sizeable input.
	cs, _ := NewSpecificCharacterSet("ISO_IR 100")
	big := make([]byte, 1<<20)
	for i := range big {
		big[i] = 'A'
	}
	got, err := cs.Decode(big)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(got) != len(big) {
		t.Fatalf("decoded length = %d, want %d", len(got), len(big))
	}
}

func TestGB18030AndGBKAreDistinctTerms(t *testing.T) {
	// Both stand-alone Chinese encodings resolve; GB18030 is a superset that can also
	// encode the four-byte forms GBK cannot. Confirm both decode a shared BMP string.
	for _, term := range []string{"GB18030", "GBK"} {
		cs, err := NewSpecificCharacterSet(term)
		if err != nil {
			t.Fatalf("NewSpecificCharacterSet(%q): %v", term, err)
		}
		enc, err := cs.Encode("中文")
		if err != nil {
			t.Fatalf("%s Encode: %v", term, err)
		}
		got, err := cs.Decode(enc)
		if err != nil {
			t.Fatalf("%s Decode: %v", term, err)
		}
		if got != "中文" {
			t.Errorf("%s round-trip = %q, want 中文", term, got)
		}
	}
}
