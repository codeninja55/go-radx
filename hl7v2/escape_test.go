package hl7v2

import (
	"testing"
)

func TestEscapeDefaultDelimiters(t *testing.T) {
	enc := DefaultEncoding()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no special characters", "Plain text", "Plain text"},
		{"field separator", "a|b", `a\F\b`},
		{"component separator", "a^b", `a\S\b`},
		{"subcomponent separator", "a&b", `a\T\b`},
		{"repetition separator", "a~b", `a\R\b`},
		{"escape character", `a\b`, `a\E\b`},
		{"a literal backslash before a delimiter is not doubled", `5\6`, `5\E\6`},
		{"every delimiter at once", `|^&~\`, `\F\\S\\T\\R\\E\`},
		{"empty string", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Escape(tc.in, enc); got != tc.want {
				t.Fatalf("Escape(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestEscapeDerivedDelimiters(t *testing.T) {
	// The non-standard set from the corpus fixture: field '#', component '@',
	// repetition '+', escape '$', subcomponent '%'. The escape table must be
	// built from these, never from the standard '|^~\&'.
	enc := EncodingCharacters{Field: '#', Component: '@', Repetition: '+', Escape: '$', Subcomponent: '%'}
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"field separator uses derived character", "a#b", "a$F$b"},
		{"component separator uses derived character", "a@b", "a$S$b"},
		{"subcomponent separator uses derived character", "a%b", "a$T$b"},
		{"repetition separator uses derived character", "a+b", "a$R$b"},
		{"escape character uses derived character", "a$b", "a$E$b"},
		{"standard delimiters are literal under a derived table", `|^~\&`, `|^~\&`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Escape(tc.in, enc); got != tc.want {
				t.Fatalf("Escape(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestUnescapeDefaultDelimiters(t *testing.T) {
	enc := DefaultEncoding()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no escape sequences", "Plain text", "Plain text"},
		{"field separator", `a\F\b`, "a|b"},
		{"component separator", `a\S\b`, "a^b"},
		{"subcomponent separator", `a\T\b`, "a&b"},
		{"repetition separator", `a\R\b`, "a~b"},
		{"escape character", `a\E\b`, `a\b`},
		{"hex carriage return", `then \X0D\done`, "then \rdone"},
		{"hex lower-case digits", `\X0d0a\`, "\r\n"},
		{"multiple hex bytes", `\X48656C6C6F\`, "Hello"},
		{"highlight start and end are removed", `\H\warn\N\`, "warn"},
		{"formatting break is removed", `line\.br\next`, "linenext"},
		{"application-defined sequence is removed", `pre\Zlocal\post`, "prepost"},
		{"the OBX note from the escaped fixture", `Reads 5\S\6 mg per 100\F\unit\E\ then \X0D\done`, "Reads 5^6 mg per 100|unit\\ then \rdone"},
		{"empty string", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := Unescape(tc.in, enc)
			if got != tc.want {
				t.Fatalf("Unescape(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestUnescapeDerivedDelimiters(t *testing.T) {
	enc := EncodingCharacters{Field: '#', Component: '@', Repetition: '+', Escape: '$', Subcomponent: '%'}
	got, _ := Unescape("a$F$b$S$c$T$d$R$e$E$f", enc)
	want := "a#b@c%d+e$f"
	if got != want {
		t.Fatalf("Unescape with derived table = %q, want %q", got, want)
	}
}

func TestUnescapeNotesSurfacesDeclinedCharsetEscapes(t *testing.T) {
	enc := DefaultEncoding()
	tests := []struct {
		name      string
		in        string
		want      string
		wantNotes int
	}{
		{
			name:      "single-byte charset switch is surfaced not corrupted",
			in:        `before\C2842\after`,
			want:      `before\C2842\after`,
			wantNotes: 1,
		},
		{
			name:      "multi-byte charset switch is surfaced not corrupted",
			in:        `x\M060000\y`,
			want:      `x\M060000\y`,
			wantNotes: 1,
		},
		{
			name:      "a standard escape alongside a declined one only notes the declined one",
			in:        `a\S\b\C2842\c`,
			want:      `a^b\C2842\c`,
			wantNotes: 1,
		},
		{
			name:      "no declined escapes yields no notes",
			in:        `a\S\b`,
			want:      "a^b",
			wantNotes: 0,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, notes := Unescape(tc.in, enc)
			if got != tc.want {
				t.Fatalf("Unescape(%q) = %q, want %q", tc.in, got, tc.want)
			}
			if len(notes) != tc.wantNotes {
				t.Fatalf("Unescape(%q) notes = %d (%v), want %d", tc.in, len(notes), notes, tc.wantNotes)
			}
		})
	}
}

func TestUnescapeMalformedSequencesArePreserved(t *testing.T) {
	enc := DefaultEncoding()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"unterminated escape", `a\Fb`, `a\Fb`},
		{"trailing lone escape character", `done\`, `done\`},
		{"hex with non-hex digits", `\XZZ\`, `\XZZ\`},
		{"hex with odd digit count", `\X0\`, `\X0\`},
		{"empty escape sequence", `a\\b`, `a\\b`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// A malformed sequence is preserved verbatim rather than dropped, so a
			// value is never silently corrupted.
			got, _ := Unescape(tc.in, enc)
			if got != tc.want {
				t.Fatalf("Unescape(%q) = %q, want %q (malformed must be preserved)", tc.in, got, tc.want)
			}
		})
	}
}

func TestEscapeUnescapeRoundTrip(t *testing.T) {
	enc := DefaultEncoding()
	for _, in := range []string{
		"Plain",
		`a|b^c&d~e\f`,
		"O&BRIEN-SMITH",
		"Reads 5^6 mg per 100|unit\\ done",
		"",
	} {
		escaped := Escape(in, enc)
		got, notes := Unescape(escaped, enc)
		if got != in {
			t.Fatalf("round-trip Escape→Unescape(%q) = %q via %q", in, got, escaped)
		}
		if len(notes) != 0 {
			t.Fatalf("round-trip of %q produced unexpected notes %v", in, notes)
		}
	}
}

// TestUnescapeEscapedFixture proves the corpus escaped.hl7 message decodes to
// the expected unescaped leaf values and re-encodes byte-exact, which is the
// acceptance gate for §2.10 read-side handling layered over the existing
// generic tree.
func TestUnescapeEscapedFixture(t *testing.T) {
	msg := corpusMessage(t, "escaped")
	enc := msg.Encoding()

	pid, ok := msg.Segment("PID")
	if !ok {
		t.Fatal("escaped fixture has no PID segment")
	}
	family := pid.field(5).component(1)
	if got, _ := Unescape(family, enc); got != "O&BRIEN-SMITH" {
		t.Fatalf("PID-5.1 unescaped = %q, want %q", got, "O&BRIEN-SMITH")
	}

	obx, ok := msg.Segment("OBX")
	if !ok {
		t.Fatal("escaped fixture has no OBX segment")
	}
	note := obx.field(5).raw()
	wantNote := "Reads 5^6 mg per 100|unit\\ then \rdone"
	if got, _ := Unescape(note, enc); got != wantNote {
		t.Fatalf("OBX-5 unescaped = %q, want %q", got, wantNote)
	}

	// The fixture round-trips byte-exact through the generic tree: unescaping is
	// a read-side projection and never rewrites the stored escaped bytes.
	out, err := msg.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	if string(out) != string(corpusRaw(t, "escaped")) {
		t.Fatalf("escaped fixture did not round-trip byte-exact")
	}
}
