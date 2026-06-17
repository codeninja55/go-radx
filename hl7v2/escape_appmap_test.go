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
