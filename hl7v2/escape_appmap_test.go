package hl7v2

import "testing"

func TestUnescapeWithAppMapZSequence(t *testing.T) {
	enc := DefaultEncoding()
	// A site agrees that \Zb\ means "bold-on" text. Without an app map the
	// sequence is preserved verbatim and surfaced via a note; with one it decodes
	// to the mapped literal.
	appMap := map[string]string{"Zb": "[bold]"}

	plain, notes := Unescape(`pre\Zb\post`, enc)
	if plain != `pre\Zb\post` {
		t.Fatalf("default Unescape changed the \\Zb\\ sequence: %q", plain)
	}
	if len(notes) != 1 {
		t.Fatalf("default Unescape produced %d notes, want 1 declined-sequence note", len(notes))
	}

	mapped, notes := Unescape(`pre\Zb\post`, enc, WithAppMap(appMap))
	if mapped != "pre[bold]post" {
		t.Errorf("Unescape with app map = %q, want %q", mapped, "pre[bold]post")
	}
	if len(notes) != 0 {
		t.Errorf("Unescape with app map produced %d notes, want 0", len(notes))
	}
}

func TestUnescapeWithAppMapOverridesBuiltin(t *testing.T) {
	enc := DefaultEncoding()
	// \N\ is normal highlight (decodes to nothing) by default; a site can map it
	// to a literal marker, and the app map takes precedence.
	if got, _ := Unescape(`a\N\b`, enc); got != "ab" {
		t.Fatalf("default Unescape(\\N\\) = %q, want %q", got, "ab")
	}
	got, _ := Unescape(`a\N\b`, enc, WithAppMap(map[string]string{"N": "<nl>"}))
	if got != "a<nl>b" {
		t.Errorf("Unescape with mapped \\N\\ = %q, want %q", got, "a<nl>b")
	}
}

func TestUnescapeAppMapCannotRedefineReservedEscapes(t *testing.T) {
	enc := DefaultEncoding()
	// An app map keyed on a reserved structural escape (\F\ = field separator) must
	// NOT change how \F\ decodes: the built-in §2.10 handling wins and the map is
	// never consulted for F/S/R/T/E. Otherwise a site could turn a standard field
	// escape into arbitrary text, corrupting every value carrying a delimiter.
	appMap := map[string]string{"F": "SHOULD_NOT_APPEAR"}
	got, _ := Unescape(`a\F\b`, enc, WithAppMap(appMap))
	if got != "a|b" {
		t.Errorf("Unescape(\\F\\) with reserved-key app map = %q, want %q (reserved escape redefined)", got, "a|b")
	}

	// All five reserved escapes resist redefinition.
	reserved := map[string]string{"E": "\\", "F": "|", "S": "^", "T": "&", "R": "~"}
	for body, want := range reserved {
		in := `x\` + body + `\y`
		evil := map[string]string{body: "EVIL"}
		if got, _ := Unescape(in, enc, WithAppMap(evil)); got != "x"+want+"y" {
			t.Errorf("Unescape(%q) with app map {%q} = %q, want %q", in, body, got, "x"+want+"y")
		}
	}

	// A non-reserved key (\Zx\) still maps, so the reservation is narrow.
	if got, _ := Unescape(`p\Zx\q`, enc, WithAppMap(map[string]string{"Zx": "[z]"})); got != "p[z]q" {
		t.Errorf("Unescape(\\Zx\\) with app map = %q, want %q (non-reserved key should still map)", got, "p[z]q")
	}
}

func TestEscapeWithAppMapRoundTrip(t *testing.T) {
	enc := DefaultEncoding()
	appMap := map[string]string{"Zb": "[bold]"}

	// A literal run matching a mapped value is encoded to its \body\ sequence.
	escaped := Escape("pre[bold]post", enc, WithAppMap(appMap))
	if escaped != `pre\Zb\post` {
		t.Fatalf("Escape with app map = %q, want %q", escaped, `pre\Zb\post`)
	}

	// And the pair round-trips under the same map.
	back, _ := Unescape(escaped, enc, WithAppMap(appMap))
	if back != "pre[bold]post" {
		t.Errorf("round-trip = %q, want %q", back, "pre[bold]post")
	}
}

func TestEscapeWithAppMapAndDelimiters(t *testing.T) {
	enc := DefaultEncoding()
	appMap := map[string]string{"Zb": "[bold]"}
	// A value carrying both a delimiter and a mapped run: the delimiter takes its
	// §2.10 escape and the mapped run takes its app-map sequence.
	escaped := Escape("a|b[bold]c", enc, WithAppMap(appMap))
	if escaped != `a\F\b\Zb\c` {
		t.Fatalf("Escape = %q, want %q", escaped, `a\F\b\Zb\c`)
	}
	back, _ := Unescape(escaped, enc, WithAppMap(appMap))
	if back != "a|b[bold]c" {
		t.Errorf("round-trip = %q, want %q", back, "a|b[bold]c")
	}
}

func TestEscapeWithAppMapLongestMatchWins(t *testing.T) {
	enc := DefaultEncoding()
	// Two mapped values where one is a prefix of the other; the longer must win
	// so the shorter never shadows it.
	appMap := map[string]string{"Z1": "AB", "Z2": "ABC"}
	if got := Escape("xABCy", enc, WithAppMap(appMap)); got != `x\Z2\y` {
		t.Errorf("Escape longest-match = %q, want %q", got, `x\Z2\y`)
	}
}
