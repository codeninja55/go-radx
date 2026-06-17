package hl7v2

import (
	"testing"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
)

func TestParseWithCharsetLatin1(t *testing.T) {
	// A PID family name containing é, encoded as ISO-8859-1 where é is the single
	// byte 0xE9. Synthetic name, no real patient data.
	raw := []byte("MSH|^~\\&|SEND|FAC|RECV|FAC|20240101||ADT^A01|C1|P|2.5.1\rPID|1||PID001||M\xe9nard^Andr\xe9\r")

	// Without the charset option the raw byte parses but the field is not UTF-8 é.
	plain, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse(raw) error = %v", err)
	}
	if got, _ := plain.Get("PID-5-1"); got == "Ménard" {
		t.Fatalf("raw parse unexpectedly produced UTF-8 %q without a charset option", got)
	}

	m, err := Parse(raw, WithCharset(charmap.ISO8859_1))
	if err != nil {
		t.Fatalf("Parse(WithCharset ISO-8859-1) error = %v", err)
	}
	if got, _ := m.Get("PID-5-1"); got != "Ménard" {
		t.Errorf("PID-5-1 = %q, want %q", got, "Ménard")
	}
	if got, _ := m.Get("PID-5-1-2"); got != "André" {
		t.Errorf("PID-5-1-2 = %q, want %q", got, "André")
	}
}

func TestParseWithCharsetShiftJIS(t *testing.T) {
	// A field carrying the Shift-JIS bytes for ヤマダ (Yamada in katakana).
	// Shift-JIS half-width katakana: ヤ=0xD4, マ=0xCF, ダ=0xC0 0xDE is two bytes;
	// use full-width via the encoder to stay correct regardless of mapping.
	enc := japanese.ShiftJIS.NewEncoder()
	name, err := enc.String("山田")
	if err != nil {
		t.Fatalf("encode Shift-JIS fixture: %v", err)
	}
	raw := []byte("MSH|^~\\&|SEND|FAC|RECV|FAC|20240101||ADT^A01|C2|P|2.5.1\rPID|1||PID002||" + name + "\r")

	m, err := Parse(raw, WithCharset(japanese.ShiftJIS))
	if err != nil {
		t.Fatalf("Parse(WithCharset Shift-JIS) error = %v", err)
	}
	if got, _ := m.Get("PID-5-1"); got != "山田" {
		t.Errorf("PID-5-1 = %q, want %q", got, "山田")
	}
}

func TestParseWithCharsetDelimitersStayAscii(t *testing.T) {
	// The structural delimiters are ASCII in every HL7 charset, so a charset
	// decode must not disturb field/component splitting. A Latin-1 body with a
	// high byte in a later field still splits on '|' and '^' correctly.
	raw := []byte("MSH|^~\\&|SEND|FAC|RECV|FAC|20240101||ADT^A01|C3|P|2.5.1\rPID|1||PID003||Cafe\xe9^Test\r")
	m, err := Parse(raw, WithCharset(charmap.ISO8859_1))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	if got, _ := m.Get("PID-5-1"); got != "Cafeé" {
		t.Errorf("PID-5-1 = %q, want %q", got, "Cafeé")
	}
	if got, _ := m.Get("PID-5-1-2"); got != "Test" {
		t.Errorf("PID-5-1-2 = %q, want %q", got, "Test")
	}
}
