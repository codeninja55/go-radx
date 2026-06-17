package dicom

import (
	"fmt"
	"strings"

	xencoding "golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// UnsupportedCharacterSetError reports a defined term in (0008,0005) that go-radx
// does not map to an encoding. It names the offending term so callers can degrade
// deliberately rather than silently mojibake the value (PS3.5 §6.1; Codex DCM-011).
type UnsupportedCharacterSetError struct {
	DefinedTerm string
}

func (e *UnsupportedCharacterSetError) Error() string {
	return fmt.Sprintf("dicom: unsupported Specific Character Set defined term %q", e.DefinedTerm)
}

// codeElement names the ISO 2022 invocation slot a designated character set occupies.
type codeElement uint8

const (
	codeElementG0 codeElement = iota // invoked into GL (0x20-0x7F)
	codeElementG1                    // invoked into GR (0xA0-0xFF)
)

// charsetFamily distinguishes how a code-extension defined term is decoded. The
// single-byte ISO 8859 supplements are decoded one byte at a time through their
// G1 charmap; the Japanese sets share the ISO-2022-JP escape vocabulary and are
// decoded by feeding the whole component run to that codec.
type charsetFamily uint8

const (
	familyDefault      charsetFamily = iota // ISO 646 pass-through
	familySingleByte                        // ISO 8859 (and TIS 620) G1 supplement
	familyJapanese                          // ISO-2022-JP escape family (IR 13/14/87/159)
	familyDoubleByteG1                      // two-byte G1 set: Korean IR 149 (EUC-KR), Chinese IR 58 (GB2312)
)

// charsetEntry describes one mapped defined term. enc is the byte<->rune codec for
// the single-byte families; escape is the ISO 2022 designation sequence (empty for
// the stand-alone, non-extensible encodings). element records whether a single-byte
// supplement is invoked into G0 or G1.
type charsetEntry struct {
	enc     xencoding.Encoding
	family  charsetFamily
	element codeElement
	escape  string
}

// asciiOnly marks the default repertoire (ISO 646 / ISO_IR 6): bytes are taken as
// 7-bit ASCII with no transformation, so the byte-identical fixture round-trips of
// the prior increments are untouched by the charset path.
var asciiOnly = charsetEntry{enc: nil, family: familyDefault, element: codeElementG0}

// definedTermTable maps DICOM defined terms to their codec, mirroring pydicom's
// charset.py for the in-scope repertoires. Both the bare single-byte terms (e.g.
// "ISO_IR 100") and the code-extension forms (e.g. "ISO 2022 IR 100") resolve here;
// the ISO 2022 forms additionally carry the escape sequence the encoder emits and
// the decoder honours.
var definedTermTable = map[string]charsetEntry{
	// Default repertoire.
	"":              asciiOnly,
	"ISO_IR 6":      asciiOnly,
	"ISO 2022 IR 6": {family: familyDefault, element: codeElementG0, escape: "\x1b(B"},

	// Single-byte ISO 8859 G1 supplements and their code-extension forms. The
	// escapes are the PS3.3 designation sequences for each supplementary set.
	"ISO_IR 100":      {enc: charmap.ISO8859_1, family: familySingleByte, element: codeElementG1},
	"ISO 2022 IR 100": {enc: charmap.ISO8859_1, family: familySingleByte, element: codeElementG1, escape: "\x1b-A"},
	"ISO_IR 101":      {enc: charmap.ISO8859_2, family: familySingleByte, element: codeElementG1},
	"ISO 2022 IR 101": {enc: charmap.ISO8859_2, family: familySingleByte, element: codeElementG1, escape: "\x1b-B"},
	"ISO_IR 109":      {enc: charmap.ISO8859_3, family: familySingleByte, element: codeElementG1},
	"ISO 2022 IR 109": {enc: charmap.ISO8859_3, family: familySingleByte, element: codeElementG1, escape: "\x1b-C"},
	"ISO_IR 110":      {enc: charmap.ISO8859_4, family: familySingleByte, element: codeElementG1},
	"ISO 2022 IR 110": {enc: charmap.ISO8859_4, family: familySingleByte, element: codeElementG1, escape: "\x1b-D"},
	"ISO_IR 144":      {enc: charmap.ISO8859_5, family: familySingleByte, element: codeElementG1},
	"ISO 2022 IR 144": {enc: charmap.ISO8859_5, family: familySingleByte, element: codeElementG1, escape: "\x1b-L"},
	"ISO_IR 127":      {enc: charmap.ISO8859_6, family: familySingleByte, element: codeElementG1},
	"ISO 2022 IR 127": {enc: charmap.ISO8859_6, family: familySingleByte, element: codeElementG1, escape: "\x1b-G"},
	"ISO_IR 126":      {enc: charmap.ISO8859_7, family: familySingleByte, element: codeElementG1},
	"ISO 2022 IR 126": {enc: charmap.ISO8859_7, family: familySingleByte, element: codeElementG1, escape: "\x1b-F"},
	"ISO_IR 138":      {enc: charmap.ISO8859_8, family: familySingleByte, element: codeElementG1},
	"ISO 2022 IR 138": {enc: charmap.ISO8859_8, family: familySingleByte, element: codeElementG1, escape: "\x1b-H"},
	"ISO_IR 148":      {enc: charmap.ISO8859_9, family: familySingleByte, element: codeElementG1},
	"ISO 2022 IR 148": {enc: charmap.ISO8859_9, family: familySingleByte, element: codeElementG1, escape: "\x1b-M"},

	// Thai (TIS 620). The bare form invokes TIS 620 directly into GR; the
	// code-extension form designates it into G1 with ESC - T (PS3.3 C.12.1.1.2,
	// PS3.5 §6.1.2.5.4). charmap.Windows874 is a TIS-620 superset whose 0xA1-0xFB
	// Thai range maps identically to ISO-8859-11 / TIS-620 (pydicom: iso_ir_166).
	"ISO_IR 166":      {enc: charmap.Windows874, family: familySingleByte, element: codeElementG1},
	"ISO 2022 IR 166": {enc: charmap.Windows874, family: familySingleByte, element: codeElementG1, escape: "\x1b-T"},

	// Korean (KS X 1001) and Simplified Chinese (GB2312) two-byte sets. DICOM uses
	// these only as G1 code extensions (PS3.5 Annex I.2, Annex K.2). Korean
	// designates with ESC $ ) C and decodes through EUC-KR; Chinese designates with
	// ESC $ ) A and decodes through GB2312, which is the 8-bit subset of GBK
	// (pydicom: euc_kr, iso_ir_58).
	"ISO 2022 IR 149": {enc: korean.EUCKR, family: familyDoubleByteG1, element: codeElementG1, escape: "\x1b$)C"},
	"ISO 2022 IR 58":  {enc: simplifiedchinese.GBK, family: familyDoubleByteG1, element: codeElementG1, escape: "\x1b$)A"},

	// Bare Japanese half-width katakana (JIS X 0201). The single-valued ISO_IR 13
	// form carries no escapes: katakana bytes 0xA1-0xDF sit directly in GR and map
	// to U+FF61-U+FF9F. Shift-JIS decodes that range natively (pydicom: shift_jis).
	"ISO_IR 13": {enc: japanese.ShiftJIS, family: familyDefault},

	// Japanese code extensions. The DICOM escapes match the ISO-2022-JP vocabulary
	// exactly, so a component run is decoded by the shared japanese.ISO2022JP codec:
	// JIS-Roman (IR 14, ESC ( J), JIS X 0201 katakana (IR 13, ESC ( I), JIS X 0208
	// (IR 87, ESC $ B), and JIS X 0212 (IR 159, ESC $ ( D).
	"ISO 2022 IR 13":  {enc: japanese.ISO2022JP, family: familyJapanese, element: codeElementG1, escape: "\x1b(I"},
	"ISO 2022 IR 14":  {enc: japanese.ISO2022JP, family: familyJapanese, element: codeElementG0, escape: "\x1b(J"},
	"ISO 2022 IR 87":  {enc: japanese.ISO2022JP, family: familyJapanese, element: codeElementG0, escape: "\x1b$B"},
	"ISO 2022 IR 159": {enc: japanese.ISO2022JP, family: familyJapanese, element: codeElementG0, escape: "\x1b$(D"},

	// Stand-alone multi-byte encodings without code extensions.
	"ISO_IR 192": {enc: unicode.UTF8, family: familyDefault},
	"GB18030":    {enc: simplifiedchinese.GB18030, family: familyDefault},
	"GBK":        {enc: simplifiedchinese.GBK, family: familyDefault},
}

// SpecificCharacterSet is the resolved (0008,0005) decode/encode pipeline. An empty
// value set means the default repertoire (ISO 646). The struct is immutable after
// construction so it may be shared read-only across a parse without locking
// (PRD §9.4); charset state for ISO 2022 switching is local to each Decode call.
type SpecificCharacterSet struct {
	terms   []string
	entries []charsetEntry
	// extended is true when any defined term carries an ISO 2022 escape, so the
	// value field may switch code elements mid-string.
	extended bool
}

// NewSpecificCharacterSet resolves the value-multiplicity-N defined terms of
// (0008,0005) into a decode/encode pipeline. An empty argument list (or a single
// empty term) means the default repertoire (ISO_IR 6 / ASCII). An unknown defined
// term returns an *UnsupportedCharacterSetError rather than silently mojibake-ing.
func NewSpecificCharacterSet(definedTerms ...string) (*SpecificCharacterSet, error) {
	if len(definedTerms) == 0 {
		return &SpecificCharacterSet{terms: []string{""}, entries: []charsetEntry{asciiOnly}}, nil
	}

	cs := &SpecificCharacterSet{
		terms:   make([]string, len(definedTerms)),
		entries: make([]charsetEntry, len(definedTerms)),
	}
	for i, raw := range definedTerms {
		term := normaliseDefinedTerm(raw)
		entry, ok := definedTermTable[term]
		if !ok {
			return nil, &UnsupportedCharacterSetError{DefinedTerm: raw}
		}
		cs.terms[i] = term
		cs.entries[i] = entry
		if entry.escape != "" {
			cs.extended = true
		}
	}
	// PS3.5 §6.1.2.5.3: a multi-valued set with more than one term invokes ISO 2022
	// code extensions even if the first term is the bare default form.
	if len(cs.entries) > 1 {
		cs.extended = true
	}
	return cs, nil
}

// normaliseDefinedTerm trims surrounding whitespace and collapses the single
// interior space DICOM uses in "ISO 2022 IR n" so a padded or sloppy term still
// resolves. It does not alter case; defined terms are case-sensitive per PS3.5.
func normaliseDefinedTerm(s string) string {
	return strings.TrimSpace(s)
}

// IsDefaultRepertoire reports whether the set is the single default repertoire, in
// which case text decoding is a pure ASCII pass-through and the byte-identical
// round-trips of the prior increments are unaffected.
func (c *SpecificCharacterSet) IsDefaultRepertoire() bool {
	return c == nil || (len(c.entries) == 1 && c.entries[0].enc == nil && !c.extended)
}

// Decode converts raw value-field bytes to a Go string under the active set,
// applying ISO 2022 G0/G1 code-element switching for code-extension sets.
func (c *SpecificCharacterSet) Decode(b []byte) (string, error) {
	if c == nil || c.IsDefaultRepertoire() {
		return decodeDefaultRepertoire(b), nil
	}
	if c.extended {
		return c.decodeISO2022(b)
	}
	return decodeSingle(c.entries[0], b)
}

// Encode converts a Go string back to value-field bytes under the active set.
func (c *SpecificCharacterSet) Encode(s string) ([]byte, error) {
	if c == nil || c.IsDefaultRepertoire() {
		return []byte(encodeDefaultRepertoire(s)), nil
	}
	if c.extended {
		return c.encodeISO2022(s)
	}
	return encodeSingle(c.entries[0], s)
}

// decodeDefaultRepertoire takes bytes as ISO 646 (7-bit ASCII), the byte-stable
// pass-through used when no Specific Character Set is in force.
func decodeDefaultRepertoire(b []byte) string { return string(b) }

// encodeDefaultRepertoire is the inverse pass-through for the default repertoire.
func encodeDefaultRepertoire(s string) string { return s }

// decodeSingle decodes b through a single non-extensible encoding. A nil enc is the
// default repertoire pass-through.
func decodeSingle(e charsetEntry, b []byte) (string, error) {
	if e.enc == nil {
		return decodeDefaultRepertoire(b), nil
	}
	out, _, err := transform.Bytes(e.enc.NewDecoder(), b)
	if err != nil {
		return "", fmt.Errorf("dicom: decode value field: %w", err)
	}
	return string(out), nil
}

// encodeSingle encodes s through a single non-extensible encoding.
func encodeSingle(e charsetEntry, s string) ([]byte, error) {
	if e.enc == nil {
		return []byte(s), nil
	}
	out, _, err := transform.Bytes(e.enc.NewEncoder(), []byte(s))
	if err != nil {
		return nil, fmt.Errorf("dicom: encode value: %w", err)
	}
	return out, nil
}
