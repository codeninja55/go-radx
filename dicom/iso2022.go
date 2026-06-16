package dicom

import (
	"fmt"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// iso2022Reset are the bytes that PS3.5 §6.1.2.5.3 requires the value field to be in
// the initial designation state before: at each component delimiter the active code
// elements revert to the initial designation. The decoder splits on these and
// decodes each run independently so a designation in one component never leaks into
// the next. The delimiter byte itself is ASCII and re-emitted verbatim.
var iso2022Reset = [256]bool{
	'\\': true, '^': true, '=': true,
	'\r': true, '\n': true, '\f': true,
}

// isISO2022Reset reports whether b is a delimiter that resets ISO 2022 designation.
func isISO2022Reset(b byte) bool { return iso2022Reset[b] }

// hasJapaneseFamily reports whether any configured defined term is a Japanese code
// extension. The DICOM Japanese escapes match the ISO-2022-JP vocabulary, so when
// present a component run is decoded by that shared codec, which natively switches
// between ASCII, JIS-Roman, JIS X 0201 katakana, JIS X 0208, and JIS X 0212.
func (c *SpecificCharacterSet) hasJapaneseFamily() bool {
	for _, e := range c.entries {
		if e.family == familyJapanese {
			return true
		}
	}
	return false
}

// decodeISO2022 decodes a code-extension value field. Each component is bounded by
// the reset delimiters; designation reverts to the initial state at every boundary.
// A delimiter byte is only a delimiter in a single-byte designation state: the same
// byte value (e.g. 0x5e '^') can appear as the low byte of a JIS X 0208 character, so
// the split tracks whether a multi-byte set is currently invoked (PS3.5 §6.1.2.5.3).
func (c *SpecificCharacterSet) decodeISO2022(b []byte) (string, error) {
	japaneseFamily := c.hasJapaneseFamily()
	var out []byte
	for _, comp := range splitISO2022Components(b) {
		if comp.delimiter {
			out = append(out, comp.bytes...) // a single ASCII delimiter byte
			continue
		}
		seg, err := c.decodeISO2022Segment(comp.bytes, japaneseFamily)
		if err != nil {
			return "", err
		}
		out = append(out, seg...)
	}
	return string(out), nil
}

// iso2022Component is one piece of a state-aware split: either a decodable run or a
// single delimiter byte that resets designation.
type iso2022Component struct {
	bytes     []byte
	delimiter bool
}

// splitISO2022Components splits b into decodable runs separated by reset delimiters,
// honouring a delimiter only while in a single-byte designation state. Escape
// sequences that designate a multi-byte set (ESC $ B, ESC $ @, ESC $ ( D) enter the
// double-byte state; the single-byte designations (ESC ( B/J/I) leave it.
func splitISO2022Components(b []byte) []iso2022Component {
	var comps []iso2022Component
	start := 0
	doubleByte := false
	i := 0
	for i < len(b) {
		if b[i] == 0x1b {
			n, dbl, ok := classifyEscape(b[i:])
			if ok {
				doubleByte = dbl
			} else {
				n = 1 // skip the lone ESC; the decoder reports the malformed run
			}
			i += n
			continue
		}
		if !doubleByte && isISO2022Reset(b[i]) {
			if i > start {
				comps = append(comps, iso2022Component{bytes: b[start:i]})
			}
			comps = append(comps, iso2022Component{bytes: b[i : i+1], delimiter: true})
			i++
			start = i
			continue
		}
		i++
	}
	if start < len(b) {
		comps = append(comps, iso2022Component{bytes: b[start:]})
	}
	return comps
}

// classifyEscape reports the length of an ISO 2022 designation escape at the start of
// b and whether it invokes a multi-byte (double-byte) set. The recognised escapes are
// the Japanese designations that can carry delimiter byte values inside their runs.
func classifyEscape(b []byte) (n int, doubleByte bool, ok bool) {
	switch {
	case hasBytePrefix(b, "\x1b$("): // ESC $ ( F : multi-byte (e.g. JIS X 0212 = D)
		return 4, true, true
	case hasBytePrefix(b, "\x1b$)"): // ESC $ ) F : multi-byte G1 (Korean = C, Chinese = A)
		return 4, true, true
	case hasBytePrefix(b, "\x1b$"): // ESC $ F : multi-byte (JIS X 0208 = B/@)
		return 3, true, true
	case hasBytePrefix(b, "\x1b("): // ESC ( F : single-byte G0 (ASCII/JIS-Roman/katakana)
		return 3, false, true
	case hasBytePrefix(b, "\x1b)"), hasBytePrefix(b, "\x1b-"): // single-byte G1 designation
		return 3, false, true
	default:
		return 0, false, false
	}
}

// decodeISO2022Segment decodes one delimiter-bounded run. A Japanese-family set
// delegates the whole run (ASCII plus JIS escapes) to japanese.ISO2022JP; a
// single-byte ISO 8859 mix is decoded by the G0/G1 byte dispatcher.
func (c *SpecificCharacterSet) decodeISO2022Segment(seg []byte, japaneseFamily bool) ([]byte, error) {
	if japaneseFamily {
		out, _, err := transform.Bytes(japanese.ISO2022JP.NewDecoder(), seg)
		if err != nil {
			return nil, fmt.Errorf("dicom: decode ISO 2022 (Japanese) component: %w", err)
		}
		return out, nil
	}
	return c.decodeSingleByteSegment(seg)
}

// decodeSingleByteSegment decodes a run that mixes ASCII (G0) with one or more G1
// supplements selected by ESC designation sequences. Bytes with the high bit clear
// come from G0 (ASCII); bytes with the high bit set come from the active G1 set. A
// single-byte G1 (ISO 8859, TIS 620) decodes one byte at a time; a double-byte G1
// (Korean EUC-KR, Chinese GB2312) consumes a contiguous run of high bytes as pairs
// (PS3.5 Annex I.2, Annex K.2). The initial G1 is the first configured G1 supplement.
func (c *SpecificCharacterSet) decodeSingleByteSegment(seg []byte) ([]byte, error) {
	g1 := c.initialG1()
	var out []byte
	for i := 0; i < len(seg); {
		if seg[i] == 0x1b { // ESC: a G1 designation sequence
			entry, n, ok := c.matchEscape(seg[i:])
			if !ok {
				return nil, fmt.Errorf("dicom: unrecognised ISO 2022 escape sequence in value field")
			}
			if entry.element == codeElementG1 {
				g1 = entry
			}
			i += n
			continue
		}
		if seg[i] < 0x80 {
			out = append(out, seg[i]) // ASCII G0 byte
			i++
			continue
		}
		if g1.enc == nil {
			return nil, fmt.Errorf("dicom: G1 byte 0x%02x with no designated G1 set", seg[i])
		}
		// A double-byte G1: decode the maximal contiguous run of high bytes through
		// the multi-byte codec so two-byte characters are not split.
		if g1.family == familyDoubleByteG1 {
			j := i
			for j < len(seg) && seg[j] >= 0x80 {
				j++
			}
			dec, _, err := transform.Bytes(g1.enc.NewDecoder(), seg[i:j])
			if err != nil {
				return nil, fmt.Errorf("dicom: decode ISO 2022 double-byte G1 run: %w", err)
			}
			out = append(out, dec...)
			i = j
			continue
		}
		// A single-byte G1: decode through the supplement's charmap, which indexes
		// the full 0x00-0xFF range.
		dec, _, err := transform.Bytes(g1.enc.NewDecoder(), seg[i:i+1])
		if err != nil {
			return nil, fmt.Errorf("dicom: decode ISO 2022 G1 byte: %w", err)
		}
		out = append(out, dec...)
		i++
	}
	return out, nil
}

// initialG1 returns the first configured G1 supplement (single- or double-byte), or
// the default (no G1) when none is configured.
func (c *SpecificCharacterSet) initialG1() charsetEntry {
	for _, e := range c.entries {
		if e.element != codeElementG1 {
			continue
		}
		if e.family == familySingleByte || e.family == familyDoubleByteG1 {
			return e
		}
	}
	return asciiOnly
}

// matchEscape resolves an ESC designation sequence at the start of b against the
// configured entries' escapes, returning the entry and the sequence length. Only the
// configured sets are honoured so an unexpected designation is a typed error rather
// than a silent switch (PS3.5 §6.1.2.3).
func (c *SpecificCharacterSet) matchEscape(b []byte) (charsetEntry, int, bool) {
	for _, e := range c.entries {
		if e.escape == "" {
			continue
		}
		if hasBytePrefix(b, e.escape) {
			return e, len(e.escape), true
		}
		// Also honour the code-extension form of a bare single-byte term so a value
		// that designates explicitly still resolves.
		if iso := definedTermTable[isoTermForEntry(termFor(e))]; iso.escape != "" && hasBytePrefix(b, iso.escape) {
			return iso, len(iso.escape), true
		}
	}
	return charsetEntry{}, 0, false
}

// hasBytePrefix reports whether b begins with the bytes of prefix.
func hasBytePrefix(b []byte, prefix string) bool {
	if len(b) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if b[i] != prefix[i] {
			return false
		}
	}
	return true
}

// termFor returns the defined term whose entry matches e by encoding identity, used
// to recover the code-extension escape for a bare single-byte term.
func termFor(e charsetEntry) string {
	for term, entry := range definedTermTable {
		if entry.enc == e.enc && entry.element == e.element && entry.escape == "" {
			return term
		}
	}
	return ""
}

// isoTermForEntry maps a bare single-byte term to its "ISO 2022 IR n" code-extension
// form so the encoder and escape matcher can find the designation sequence.
func isoTermForEntry(term string) string {
	switch term {
	case "ISO_IR 100":
		return "ISO 2022 IR 100"
	case "ISO_IR 101":
		return "ISO 2022 IR 101"
	case "ISO_IR 109":
		return "ISO 2022 IR 109"
	case "ISO_IR 110":
		return "ISO 2022 IR 110"
	case "ISO_IR 144":
		return "ISO 2022 IR 144"
	case "ISO_IR 127":
		return "ISO 2022 IR 127"
	case "ISO_IR 126":
		return "ISO 2022 IR 126"
	case "ISO_IR 138":
		return "ISO 2022 IR 138"
	case "ISO_IR 148":
		return "ISO 2022 IR 148"
	default:
		return term
	}
}

// encodeISO2022 encodes s under the code-extension set, component by component so a
// designation never crosses a delimiter (the inverse of decodeISO2022).
func (c *SpecificCharacterSet) encodeISO2022(s string) ([]byte, error) {
	japaneseFamily := c.hasJapaneseFamily()
	bs := []byte(s)
	var out []byte
	start := 0
	for i := 0; i <= len(bs); i++ {
		if i == len(bs) || isISO2022Reset(bs[i]) {
			seg, err := c.encodeISO2022Segment(bs[start:i], japaneseFamily)
			if err != nil {
				return nil, err
			}
			out = append(out, seg...)
			if i < len(bs) {
				out = append(out, bs[i])
			}
			start = i + 1
		}
	}
	return out, nil
}

// encodeISO2022Segment encodes one delimiter-bounded component. The Japanese family
// delegates to japanese.ISO2022JP, which emits the JIS escapes and returns to ASCII
// before the run ends. The single-byte family emits ASCII directly and one G1
// designation escape before the first non-ASCII rune of each supplement.
func (c *SpecificCharacterSet) encodeISO2022Segment(seg []byte, japaneseFamily bool) ([]byte, error) {
	if len(seg) == 0 {
		return nil, nil
	}
	if japaneseFamily {
		out, _, err := transform.Bytes(japanese.ISO2022JP.NewEncoder(), seg)
		if err != nil {
			return nil, fmt.Errorf("dicom: encode ISO 2022 (Japanese) component: %w", err)
		}
		return out, nil
	}
	return c.encodeSingleByteSegment(seg)
}

// encodeSingleByteSegment encodes a component as ASCII plus ISO 8859 G1 runs. ASCII
// runes pass through; a non-ASCII rune is encoded by the first configured G1
// supplement that can represent it, emitting that supplement's G1 designation escape
// on first use.
func (c *SpecificCharacterSet) encodeSingleByteSegment(seg []byte) ([]byte, error) {
	var out []byte
	designated := false
	var g1 charsetEntry
	for _, r := range string(seg) {
		if r < 0x80 {
			out = append(out, byte(r)) // #nosec G115 -- r < 0x80 guarded immediately above
			continue
		}
		entry, ok := c.g1ForRune(r)
		if !ok {
			return nil, fmt.Errorf("dicom: no configured ISO 8859 supplement can encode rune %q", r)
		}
		if !designated || entry.enc != g1.enc {
			out = append(out, []byte(c.escapeFor(entry))...)
			designated = true
			g1 = entry
		}
		b, _, err := transform.Bytes(entry.enc.NewEncoder(), []byte(string(r)))
		if err != nil {
			return nil, fmt.Errorf("dicom: encode ISO 8859 rune: %w", err)
		}
		out = append(out, b...)
	}
	return out, nil
}

// g1ForRune returns the first configured G1 supplement (single- or double-byte) that
// can encode r.
func (c *SpecificCharacterSet) g1ForRune(r rune) (charsetEntry, bool) {
	for _, e := range c.entries {
		if e.enc == nil || (e.family != familySingleByte && e.family != familyDoubleByteG1) {
			continue
		}
		if _, _, err := transform.Bytes(e.enc.NewEncoder(), []byte(string(r))); err == nil {
			return e, true
		}
	}
	return charsetEntry{}, false
}

// escapeFor returns the G1 designation escape for a supplement, resolving the
// code-extension form when the configured term was the bare single-byte one.
func (c *SpecificCharacterSet) escapeFor(e charsetEntry) string {
	if e.escape != "" {
		return e.escape
	}
	return definedTermTable[isoTermForEntry(termFor(e))].escape
}
