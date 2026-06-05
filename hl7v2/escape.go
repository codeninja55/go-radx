package hl7v2

import "strings"

// escapeTable maps each delimiter byte to the single-letter escape sequence that
// HL7 Chapter 2 §2.10 defines for it, and the reverse. It is derived from the
// EncodingCharacters in force for the message — never from the package defaults —
// so a sender using non-standard delimiters escapes and unescapes against its own
// header. The five sequences are \F\ (field), \S\ (component), \T\ (subcomponent),
// \R\ (repetition), and \E\ (the escape character itself).
type escapeTable struct {
	enc EncodingCharacters
}

// escapeCode returns the §2.10 escape letter for a delimiter byte, and whether b
// is one of the five delimiters that must be escaped. The escape character is
// reported first so a literal escape byte is never mistaken for one of the other
// delimiters when a header reuses a byte across roles.
func (t escapeTable) escapeCode(b byte) (byte, bool) {
	switch b {
	case t.enc.Escape:
		return 'E', true
	case t.enc.Field:
		return 'F', true
	case t.enc.Component:
		return 'S', true
	case t.enc.Subcomponent:
		return 'T', true
	case t.enc.Repetition:
		return 'R', true
	default:
		return 0, false
	}
}

// delimiterFor returns the delimiter byte a §2.10 escape letter decodes to, and
// whether the letter names a delimiter. It is the inverse of escapeCode.
func (t escapeTable) delimiterFor(code byte) (byte, bool) {
	switch code {
	case 'E':
		return t.enc.Escape, true
	case 'F':
		return t.enc.Field, true
	case 'S':
		return t.enc.Component, true
	case 'T':
		return t.enc.Subcomponent, true
	case 'R':
		return t.enc.Repetition, true
	default:
		return 0, false
	}
}

// Escape encodes a leaf value for transmission, replacing each delimiter byte
// with its §2.10 escape sequence so the value survives the message structure
// intact. The escape table is derived from enc, so a non-standard sender's
// delimiters are escaped against its own header rather than the defaults. The
// escape character maps to \E\, which keeps a literal escape byte from being read
// back as the start of a sequence on unescape.
//
// Escape is the inverse of Unescape for values that contain no hex, highlight,
// formatting, or application-defined sequences; those are read-side only and
// Escape never emits them.
func Escape(value string, enc EncodingCharacters) string {
	table := escapeTable{enc: enc}

	// Most leaf values carry no delimiter byte, so scan first and return the
	// input unchanged when there is nothing to escape, avoiding an allocation.
	needsEscape := false
	for i := 0; i < len(value); i++ {
		if _, ok := table.escapeCode(value[i]); ok {
			needsEscape = true
			break
		}
	}
	if !needsEscape {
		return value
	}

	var b strings.Builder
	b.Grow(len(value) + 8)
	for i := 0; i < len(value); i++ {
		if code, ok := table.escapeCode(value[i]); ok {
			b.WriteByte(enc.Escape)
			b.WriteByte(code)
			b.WriteByte(enc.Escape)
			continue
		}
		b.WriteByte(value[i])
	}
	return b.String()
}

// UnescapeNote records a §2.10 escape sequence that Unescape recognised but
// declined to apply, so the caller can surface it rather than discover a value
// silently changed. The only declined sequences are inline character-set
// switches (\Cxxyy\ and \Mxxyyzz\), which are out of scope for v1; their raw
// bytes are preserved verbatim in the returned value. Sequence holds the exact
// source bytes including both escape delimiters, never any surrounding field
// value, so a note carries no PHI.
type UnescapeNote struct {
	Sequence string // the verbatim escape sequence, e.g. `\C2842\`
	Reason   string // why it was declined, free of field values
}

// Unescape decodes a leaf value read from a message, reversing the §2.10 escape
// sequences against the table derived from enc. It handles the delimiter escapes
// (\F\ \S\ \T\ \R\ \E\), hex data (\Xdd...\), highlight start/end (\H\ \N\), the
// formatting and rich-text commands (\.br\ and the other \.xx\ sequences), and
// application-defined sequences (\Zxxx\). Highlight, formatting, and
// application-defined sequences carry no character data, so they decode to the
// empty string.
//
// Inline character-set switches (\Cxxyy\, \Mxxyyzz\) are out of scope for v1: they
// are preserved verbatim in the result and reported through the returned notes
// rather than corrupted. A malformed sequence — an unterminated escape, a
// non-hex \X\ body, or an empty \\ — is also preserved verbatim so a value is
// never silently lost.
//
// Unescape never mutates the message; it is a read-side projection, so a value
// round-trips byte-exact through Parse and MarshalText regardless of how it is
// read.
func Unescape(value string, enc EncodingCharacters) (string, []UnescapeNote) {
	if !strings.ContainsRune(value, rune(enc.Escape)) {
		return value, nil
	}

	table := escapeTable{enc: enc}
	var b strings.Builder
	b.Grow(len(value))
	var notes []UnescapeNote

	for i := 0; i < len(value); {
		if value[i] != enc.Escape {
			b.WriteByte(value[i])
			i++
			continue
		}

		// A sequence runs from this escape byte to the next one. With no closing
		// escape the run is malformed; preserve it verbatim and stop.
		end := strings.IndexByte(value[i+1:], enc.Escape)
		if end < 0 {
			b.WriteString(value[i:])
			break
		}
		end += i + 1 // absolute index of the closing escape byte

		body := value[i+1 : end]
		seq := value[i : end+1] // the full sequence including both delimiters

		decoded, decodedNote, ok := decodeEscape(body, table)
		if !ok {
			// Unrecognised or malformed: preserve verbatim rather than drop it.
			b.WriteString(seq)
			i = end + 1
			continue
		}
		if decodedNote != nil {
			decodedNote.Sequence = seq
			notes = append(notes, *decodedNote)
			b.WriteString(seq) // declined sequences keep their raw bytes
			i = end + 1
			continue
		}
		b.WriteString(decoded)
		i = end + 1
	}

	return b.String(), notes
}

// decodeEscape interprets the body of one escape sequence (the bytes between the
// two escape delimiters). It returns the decoded text, an optional note for a
// recognised-but-declined sequence, and whether the body was a well-formed §2.10
// sequence at all. An empty body is malformed (ok == false), so a stray \\ is
// preserved rather than collapsed.
func decodeEscape(body string, table escapeTable) (string, *UnescapeNote, bool) {
	if body == "" {
		return "", nil, false
	}

	// Single-letter delimiter escapes: \F\ \S\ \T\ \R\ \E\.
	if len(body) == 1 {
		if d, ok := table.delimiterFor(body[0]); ok {
			return string(d), nil, true
		}
	}

	switch body[0] {
	case 'X':
		return decodeHex(body[1:])
	case 'H', 'N':
		// Highlight start/end carry no character data; they decode to nothing.
		if len(body) == 1 {
			return "", nil, true
		}
	case '.':
		// Formatting / rich-text commands (\.br\, \.sp\, \.fi\, ...). They carry
		// layout intent, not character data, so they decode to nothing.
		return "", nil, true
	case 'Z':
		// Application-defined sequence (\Zxxx\): locally agreed, no portable
		// meaning, so it decodes to nothing.
		return "", nil, true
	case 'C', 'M':
		// Inline character-set switch: declined for v1 and surfaced via a note.
		return "", &UnescapeNote{Reason: "inline character-set switch escape is not applied (out of scope)"}, true
	}

	return "", nil, false
}

// decodeHex decodes the digits of an \Xdd...\ sequence into raw bytes. The digit
// count must be even and every digit hexadecimal; otherwise the sequence is
// malformed and preserved verbatim by the caller.
func decodeHex(digits string) (string, *UnescapeNote, bool) {
	if len(digits) == 0 || len(digits)%2 != 0 {
		return "", nil, false
	}
	out := make([]byte, len(digits)/2)
	for i := 0; i < len(digits); i += 2 {
		hi, ok := hexNibble(digits[i])
		if !ok {
			return "", nil, false
		}
		lo, ok := hexNibble(digits[i+1])
		if !ok {
			return "", nil, false
		}
		out[i/2] = hi<<4 | lo
	}
	return string(out), nil, true
}

// hexNibble returns the value of a single hexadecimal digit and whether c is one.
func hexNibble(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	default:
		return 0, false
	}
}
