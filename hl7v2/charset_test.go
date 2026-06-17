package hl7v2

import (
	"testing"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/unicode"
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

func TestParseAnyWithCharsetDecodesBeforeDispatch(t *testing.T) {
	// ParseAny sniffs the container type (leading segment + MSH count) before
	// dispatching. With a charset whose MSH is non-ASCII in the source bytes — here
	// UTF-16LE, where "MSH" is the bytes 4D 00 53 00 48 00 — the leading-segment
	// sniff fails on the RAW bytes, so ParseAny must decode FIRST. Parse already
	// decodes first and works, so ParseAny must route and parse it the same way.
	utf16le := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	src := "MSH|^~\\&|SEND|FAC|RECV|FAC|20240101||ADT^A01|CA1|P|2.5.1\rPID|1||PID001||Ménard^André\r"
	raw, err := utf16le.NewEncoder().Bytes([]byte(src))
	if err != nil {
		t.Fatalf("encode UTF-16LE fixture: %v", err)
	}
	// Guard: the raw bytes must NOT begin with ASCII "MSH", otherwise the test does
	// not exercise the decode-before-sniff path.
	if len(raw) >= 3 && raw[0] == 'M' && raw[1] == 'S' && raw[2] == 'H' {
		t.Fatal("UTF-16LE fixture unexpectedly begins with ASCII MSH")
	}

	c, err := ParseAny(raw, WithCharset(utf16le))
	if err != nil {
		t.Fatalf("ParseAny(WithCharset UTF-16LE) error = %v", err)
	}
	m, ok := c.(*Message)
	if !ok {
		t.Fatalf("ParseAny = %T, want *Message", c)
	}
	if got, _ := m.Get("PID-5-1"); got != "Ménard" {
		t.Errorf("PID-5-1 = %q, want %q", got, "Ménard")
	}
	if got, _ := m.Get("PID-5-1-2"); got != "André" {
		t.Errorf("PID-5-1-2 = %q, want %q", got, "André")
	}
}

func TestParseAnyWithCharsetHeaderlessBatch(t *testing.T) {
	// The decode-before-dispatch rule must also hold for the MSH count that picks
	// *Batch vs *Message. In UTF-16LE the raw bytes contain no ASCII "MSH" run, so
	// the raw-byte count is 0 and the body would be misrouted; after decode the
	// count is 2 and ParseAny must yield a *Batch of two decoded messages.
	utf16le := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	src := "MSH|^~\\&|SEND|FAC|RECV|FAC|20240101||ADT^A01|CB1|P|2.5.1\rPID|1||P1||Ménard\r" +
		"MSH|^~\\&|SEND|FAC|RECV|FAC|20240101||ADT^A02|CB2|P|2.5.1\rPID|1||P2||André\r"
	raw, err := utf16le.NewEncoder().Bytes([]byte(src))
	if err != nil {
		t.Fatalf("encode UTF-16LE fixture: %v", err)
	}

	c, err := ParseAny(raw, WithCharset(utf16le))
	if err != nil {
		t.Fatalf("ParseAny(headerless batch, WithCharset) error = %v", err)
	}
	batch, ok := c.(*Batch)
	if !ok {
		t.Fatalf("ParseAny = %T, want *Batch", c)
	}
	if len(batch.Messages) != 2 {
		t.Fatalf("batch has %d messages, want 2", len(batch.Messages))
	}
	if got, _ := batch.Messages[0].Get("PID-5-1"); got != "Ménard" {
		t.Errorf("message 0 PID-5-1 = %q, want %q", got, "Ménard")
	}
	if got, _ := batch.Messages[1].Get("PID-5-1"); got != "André" {
		t.Errorf("message 1 PID-5-1 = %q, want %q", got, "André")
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
