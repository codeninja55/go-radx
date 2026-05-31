package dicom

import (
	"testing"
)

// canonicalIR87Name are the value-field bytes for the PS3.5 Annex H.3 Japanese
// person name "Yamada^Tarou=山田^太郎=やまだ^たろう" under defined terms
// ["", "ISO 2022 IR 87"]: an ASCII alphabetic group, then ideographic and phonetic
// groups in JIS X 0208 (ESC $ B), each subcomponent returned to ASCII (ESC ( B)
// before the next delimiter.
var canonicalIR87Name = []byte{
	'Y', 'a', 'm', 'a', 'd', 'a', '^', 'T', 'a', 'r', 'o', 'u', '=',
	0x1b, '$', 'B', 0x3b, 0x33, 0x45, 0x44, 0x1b, '(', 'B', '^',
	0x1b, '$', 'B', 0x42, 0x40, 0x4f, 0x3a, 0x1b, '(', 'B', '=',
	0x1b, '$', 'B', 0x24, 0x64, 0x24, 0x5e, 0x24, 0x40, 0x1b, '(', 'B', '^',
	0x1b, '$', 'B', 0x24, 0x3f, 0x24, 0x6d, 0x24, 0x26, 0x1b, '(', 'B',
}

func TestISO2022JapanesePersonNameDecode(t *testing.T) {
	cs, err := NewSpecificCharacterSet("", "ISO 2022 IR 87")
	if err != nil {
		t.Fatalf("NewSpecificCharacterSet: %v", err)
	}
	got, err := cs.Decode(canonicalIR87Name)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	want := "Yamada^Tarou=山田^太郎=やまだ^たろう"
	if got != want {
		t.Fatalf("Decode = %q, want %q", got, want)
	}
}

func TestISO2022JapanesePersonNameRoundTrip(t *testing.T) {
	cs, err := NewSpecificCharacterSet("", "ISO 2022 IR 87")
	if err != nil {
		t.Fatalf("NewSpecificCharacterSet: %v", err)
	}
	decoded, err := cs.Decode(canonicalIR87Name)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	reencoded, err := cs.Encode(decoded)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if string(reencoded) != string(canonicalIR87Name) {
		t.Fatalf("round-trip bytes differ:\n got % x\nwant % x", reencoded, canonicalIR87Name)
	}
}

func TestISO2022JapaneseComponentsDecodePerGroup(t *testing.T) {
	cs, _ := NewSpecificCharacterSet("", "ISO 2022 IR 87")
	got, err := cs.Decode(canonicalIR87Name)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	pn, err := ParsePersonName(got)
	if err != nil {
		t.Fatalf("ParsePersonName: %v", err)
	}
	if pn.Alphabetic.FamilyName != "Yamada" || pn.Alphabetic.GivenName != "Tarou" {
		t.Errorf("alphabetic = %+v, want Yamada/Tarou", pn.Alphabetic)
	}
	if pn.Ideographic.FamilyName != "山田" || pn.Ideographic.GivenName != "太郎" {
		t.Errorf("ideographic = %+v, want 山田/太郎", pn.Ideographic)
	}
	if pn.Phonetic.FamilyName != "やまだ" || pn.Phonetic.GivenName != "たろう" {
		t.Errorf("phonetic = %+v, want やまだ/たろう", pn.Phonetic)
	}
}

func TestISO2022Latin1G1DesignationDecode(t *testing.T) {
	// A multi-valued set with code extensions: default G0 plus ISO 2022 IR 100
	// designated into G1. "Bär" with ä as the Latin-1 G1 byte 0xe4.
	cs, err := NewSpecificCharacterSet("", "ISO 2022 IR 100")
	if err != nil {
		t.Fatalf("NewSpecificCharacterSet: %v", err)
	}
	raw := []byte{'B', 0xe4, 'r'}
	got, err := cs.Decode(raw)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got != "Bär" {
		t.Fatalf("Decode = %q, want Bär", got)
	}
}

func TestISO2022KatakanaDecode(t *testing.T) {
	// ISO 2022 IR 13 (JIS X 0201 katakana) designated via ESC ( I. The katakana
	// occupies the 7-bit GL range after the designation: ﾔ is 0x54, ﾏ is 0x4f.
	cs, err := NewSpecificCharacterSet("ISO 2022 IR 13", "ISO 2022 IR 87")
	if err != nil {
		t.Fatalf("NewSpecificCharacterSet: %v", err)
	}
	raw := []byte{0x1b, '(', 'I', 0x54, 0x4f, 0x1b, '(', 'B'}
	got, err := cs.Decode(raw)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got != "ﾔﾏ" {
		t.Fatalf("Decode = %q (% x), want ﾔﾏ", got, []byte(got))
	}
}
