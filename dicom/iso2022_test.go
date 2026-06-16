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

// canonicalIR149Name are the value-field bytes for the PS3.5 Annex I.2 Korean person
// name under defined terms ["", "ISO 2022 IR 149"]: an ASCII alphabetic group, then
// Hanja ideographic and Hangul phonetic groups, each component designating KS X 1001
// into G1 with ESC $ ) C and decoding through EUC-KR.
var canonicalIR149Name = []byte{
	'H', 'o', 'n', 'g', '^', 'G', 'i', 'l', 'd', 'o', 'n', 'g', '=',
	0x1b, '$', ')', 'C', 0xFB, 0xF3, '^',
	0x1b, '$', ')', 'C', 0xD1, 0xCE, 0xD4, 0xD7, '=',
	0x1b, '$', ')', 'C', 0xC8, 0xAB, '^',
	0x1b, '$', ')', 'C', 0xB1, 0xE6, 0xB5, 0xBF,
}

// canonicalIR58Name are the value-field bytes for the PS3.5 Annex K.2 Simplified
// Chinese person name under defined terms ["", "ISO 2022 IR 58"]: an ASCII alphabetic
// group then GB2312 ideographic group, designated into G1 with ESC $ ) A.
var canonicalIR58Name = []byte{
	'Z', 'h', 'a', 'n', 'g', '^', 'X', 'i', 'a', 'o', 'D', 'o', 'n', 'g', '=',
	0x1b, '$', ')', 'A', 0xD5, 0xC5, '^',
	0x1b, '$', ')', 'A', 0xD0, 0xA1, 0xB6, 0xAB,
}

func TestISO2022KoreanPersonNameDecode(t *testing.T) {
	cs, err := NewSpecificCharacterSet("", "ISO 2022 IR 149")
	if err != nil {
		t.Fatalf("NewSpecificCharacterSet: %v", err)
	}
	got, err := cs.Decode(canonicalIR149Name)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	want := "Hong^Gildong=洪^吉洞=홍^길동"
	if got != want {
		t.Fatalf("Decode = %q, want %q", got, want)
	}
}

func TestISO2022KoreanComponentsDecodePerGroup(t *testing.T) {
	cs, _ := NewSpecificCharacterSet("", "ISO 2022 IR 149")
	got, err := cs.Decode(canonicalIR149Name)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	pn, err := ParsePersonName(got)
	if err != nil {
		t.Fatalf("ParsePersonName: %v", err)
	}
	if pn.Alphabetic.FamilyName != "Hong" || pn.Alphabetic.GivenName != "Gildong" {
		t.Errorf("alphabetic = %+v, want Hong/Gildong", pn.Alphabetic)
	}
	if pn.Ideographic.FamilyName != "洪" || pn.Ideographic.GivenName != "吉洞" {
		t.Errorf("ideographic = %+v, want 洪/吉洞", pn.Ideographic)
	}
	if pn.Phonetic.FamilyName != "홍" || pn.Phonetic.GivenName != "길동" {
		t.Errorf("phonetic = %+v, want 홍/길동", pn.Phonetic)
	}
}

func TestISO2022KoreanRoundTrip(t *testing.T) {
	cs, _ := NewSpecificCharacterSet("", "ISO 2022 IR 149")
	decoded, err := cs.Decode(canonicalIR149Name)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	reencoded, err := cs.Encode(decoded)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if got, err := cs.Decode(reencoded); err != nil || got != decoded {
		t.Fatalf("re-decode = %q (err %v), want %q", got, err, decoded)
	}
}

func TestISO2022SimplifiedChinesePersonNameDecode(t *testing.T) {
	cs, err := NewSpecificCharacterSet("", "ISO 2022 IR 58")
	if err != nil {
		t.Fatalf("NewSpecificCharacterSet: %v", err)
	}
	got, err := cs.Decode(canonicalIR58Name)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	want := "Zhang^XiaoDong=张^小东"
	if got != want {
		t.Fatalf("Decode = %q, want %q", got, want)
	}
	pn, err := ParsePersonName(got)
	if err != nil {
		t.Fatalf("ParsePersonName: %v", err)
	}
	if pn.Ideographic.FamilyName != "张" || pn.Ideographic.GivenName != "小东" {
		t.Errorf("ideographic = %+v, want 张/小东", pn.Ideographic)
	}
}

func TestISO2022SimplifiedChineseRoundTrip(t *testing.T) {
	cs, _ := NewSpecificCharacterSet("", "ISO 2022 IR 58")
	decoded, err := cs.Decode(canonicalIR58Name)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	reencoded, err := cs.Encode(decoded)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if got, err := cs.Decode(reencoded); err != nil || got != decoded {
		t.Fatalf("re-decode = %q (err %v), want %q", got, err, decoded)
	}
}

func TestISO2022ThaiDesignationDecode(t *testing.T) {
	// ISO 2022 IR 166 (TIS 620) designated into G1 with ESC - T. The Thai run
	// "นามสกุล" is the pydicom test_charset.py ENCODED_NAMES vector.
	cs, err := NewSpecificCharacterSet("", "ISO 2022 IR 166")
	if err != nil {
		t.Fatalf("NewSpecificCharacterSet: %v", err)
	}
	raw := []byte{0x1b, '-', 'T', 0xB9, 0xD2, 0xC1, 0xCA, 0xA1, 0xD8, 0xC5}
	got, err := cs.Decode(raw)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if want := "นามสกุล"; got != want {
		t.Fatalf("Decode = %q (% x), want %q", got, []byte(got), want)
	}
}

func TestBareThaiDecode(t *testing.T) {
	// Bare ISO_IR 166: TIS 620 invoked directly into GR, no escape sequence.
	cs, err := NewSpecificCharacterSet("ISO_IR 166")
	if err != nil {
		t.Fatalf("NewSpecificCharacterSet: %v", err)
	}
	raw := []byte{0xB9, 0xD2, 0xC1, 0xCA, 0xA1, 0xD8, 0xC5}
	got, err := cs.Decode(raw)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if want := "นามสกุล"; got != want {
		t.Fatalf("Decode = %q (% x), want %q", got, []byte(got), want)
	}
}

func TestBareKatakanaDecode(t *testing.T) {
	// Bare ISO_IR 13: JIS X 0201 half-width katakana with no escapes, bytes in GR
	// (0xA1-0xDF -> U+FF61-U+FF9F). "ﾔﾏﾀﾞ" then "^" then "ﾀﾛｳ" from the pydicom
	// test_charset.py ENCODED_NAMES vector (escape-stripped body).
	cs, err := NewSpecificCharacterSet("ISO_IR 13")
	if err != nil {
		t.Fatalf("NewSpecificCharacterSet: %v", err)
	}
	raw := []byte{0xD4, 0xCF, 0xC0, 0xDE, '^', 0xC0, 0xDB, 0xB3}
	got, err := cs.Decode(raw)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if want := "ﾔﾏﾀﾞ^ﾀﾛｳ"; got != want {
		t.Fatalf("Decode = %q (% x), want %q", got, []byte(got), want)
	}
}

func TestBareKatakanaRoundTrip(t *testing.T) {
	cs, _ := NewSpecificCharacterSet("ISO_IR 13")
	raw := []byte{0xD4, 0xCF, 0xC0, 0xDE, '^', 0xC0, 0xDB, 0xB3}
	decoded, err := cs.Decode(raw)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	reencoded, err := cs.Encode(decoded)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if string(reencoded) != string(raw) {
		t.Fatalf("round-trip bytes differ:\n got % x\nwant % x", reencoded, raw)
	}
}
