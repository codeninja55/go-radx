package hl7v2

import (
	"bytes"
)

// parseConfig holds the resolved options for a single Parse call. It is never a
// package global: encoding characters and limits are per-message so a
// non-standard sender round-trips correctly (PRD §9.4).
type parseConfig struct{}

// ParseOption configures a parse. Pass none for the standard behaviour. The
// option set is intentionally minimal for the M2 ORM slice; size caps, an
// application-defined escape map, and strictness toggles arrive in M5.
type ParseOption func(*parseConfig)

func newParseConfig(opts ...ParseOption) parseConfig {
	var cfg parseConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Message is the root of the six-level tree: an ordered list of Segments, the
// first of which is MSH. Enc carries the EncodingCharacters in force so String
// and MarshalText re-render with the original delimiters.
type Message struct {
	Segments []Segment
	Enc      EncodingCharacters
}

// Segment is one delimiter-separated line. Fields[0] is the segment ID, e.g.
// "PID". term preserves the exact terminator bytes that followed the segment in
// the source (one of "\r", "\n", "\r\n", or "" for a final segment with no
// trailing terminator) so MarshalText reproduces the input byte-for-byte.
type Segment struct {
	Fields []Field
	term   string
}

// Field is one '|'-separated field, holding one or more Repetitions.
type Field struct {
	Repetitions []Repetition
}

// Repetition is one '~'-separated repetition of a field, holding Components.
type Repetition struct {
	Components []Component
}

// Component is one '^'-separated component, holding the leaf subcomponents.
type Component struct {
	Subcomponents []string // leaf level
}

// Parse decodes a single HL7 v2 message from b. b may use \r, \n, or \r\n
// segment terminators; the canonical form uses \r. Encoding characters are
// derived from MSH-1/MSH-2 (see EncodingCharacters), never from a global
// default, so a sender using non-standard delimiters round-trips correctly.
//
// Parse is strict about structure but lenient about line endings, matching how
// real interfaces emit messages: a body whose first segment is not MSH is a
// *ParseError, and a body that ends inside a segment is truncation and returns
// io.ErrUnexpectedEOF wrapped in a *ParseError (PRD §9.2).
func Parse(b []byte, opts ...ParseOption) (*Message, error) {
	_ = newParseConfig(opts...)

	if len(b) < 3 || b[0] != 'M' || b[1] != 'S' || b[2] != 'H' {
		return nil, &ParseError{Offset: 0, Reason: "message must begin with an MSH segment"}
	}

	// The MSH header must carry at least MSH-1 (the field separator at offset 3)
	// and a MSH-2 terminated by the next field separator. A body that begins
	// "MSH" but stops before MSH-2 is closed is mid-segment truncation, not a
	// well-formed but tiny message.
	if len(b) < 4 {
		return nil, truncatedAt(len(b), "MSH header truncated before MSH-1 field separator")
	}
	fieldSep := b[3]
	if indexByte(b[4:], fieldSep) < 0 {
		return nil, truncatedAt(len(b), "MSH header truncated before MSH-2 is terminated")
	}

	enc, err := DeriveEncoding(b)
	if err != nil {
		return nil, err
	}

	segments, err := splitSegments(b, enc)
	if err != nil {
		return nil, err
	}

	return &Message{Segments: segments, Enc: enc}, nil
}

// splitSegments breaks b into segments on \r, \n, or \r\n terminators,
// preserving the exact terminator bytes so the message round-trips. A trailing
// empty fragment after a terminator is not a segment.
func splitSegments(b []byte, enc EncodingCharacters) ([]Segment, error) {
	var segments []Segment
	offset := 0

	for offset < len(b) {
		line, term, next := nextLine(b, offset)
		if len(line) == 0 && term == "" {
			break
		}
		// A bare terminator with no content is skipped rather than yielding an
		// empty segment; real interfaces sometimes emit a trailing blank line.
		if len(line) > 0 {
			seg, err := parseSegment(line, term, enc, offset)
			if err != nil {
				return nil, err
			}
			segments = append(segments, seg)
		}
		offset = next
	}

	if len(segments) == 0 {
		return nil, &ParseError{Offset: 0, Reason: "message has no segments"}
	}
	return segments, nil
}

// nextLine returns the segment content starting at offset, the terminator bytes
// that followed it (one of "\r", "\n", "\r\n", or "" at end of input), and the
// offset of the next segment.
func nextLine(b []byte, offset int) (line []byte, term string, next int) {
	rest := b[offset:]
	idx := bytes.IndexAny(rest, "\r\n")
	if idx < 0 {
		return rest, "", len(b)
	}
	line = rest[:idx]
	switch rest[idx] {
	case '\r':
		if idx+1 < len(rest) && rest[idx+1] == '\n' {
			return line, "\r\n", offset + idx + 2
		}
		return line, "\r", offset + idx + 1
	default: // '\n'
		return line, "\n", offset + idx + 1
	}
}

// parseSegment splits one segment line into the field/repetition/component/
// subcomponent tree. The MSH segment is offset-quirky: the field separator is
// itself MSH-1 and the encoding characters are MSH-2, so they are spliced in
// verbatim before the remaining fields are split (matching python-hl7's _split).
func parseSegment(line []byte, term string, enc EncodingCharacters, baseOffset int) (Segment, error) {
	id := line
	if len(line) >= 3 {
		id = line[:3]
	}

	if bytes.Equal(id, []byte("MSH")) {
		return parseMSHSegment(line, term, enc, baseOffset)
	}

	fields := splitFields(line, enc)
	return Segment{Fields: fields, term: term}, nil
}

// parseMSHSegment handles the MSH header's indexing quirk: Fields[0] is "MSH",
// Fields[1] is the field separator (MSH-1, a single byte), Fields[2] is the
// encoding characters (MSH-2), and the remaining fields split normally from
// after MSH-2's terminating field separator.
func parseMSHSegment(line []byte, term string, enc EncodingCharacters, baseOffset int) (Segment, error) {
	// line[3] is the field separator; MSH-2 runs to the next field separator.
	if len(line) < 4 {
		return Segment{}, truncatedAt(baseOffset+len(line), "MSH segment truncated before MSH-1 field separator")
	}
	fieldSep := line[3]
	rel := indexByte(line[4:], fieldSep)
	if rel < 0 {
		return Segment{}, truncatedAt(baseOffset+len(line), "MSH segment truncated before MSH-2 is terminated")
	}
	end := 4 + rel

	fields := []Field{
		leafField(line[:3]),    // MSH-0: "MSH"
		leafField(line[3:4]),   // MSH-1: the field separator, verbatim
		leafField(line[4:end]), // MSH-2: the encoding characters, verbatim
	}
	// line[end] is the field separator that terminates MSH-2; fields from MSH-3
	// onward are everything after it, split normally.
	rest := line[end+1:]
	fields = append(fields, splitFields(rest, enc)...)

	return Segment{Fields: fields, term: term}, nil
}

// splitFields splits a segment line into fields on the field separator, then
// each field into repetitions, components, and subcomponents.
func splitFields(line []byte, enc EncodingCharacters) []Field {
	rawFields := bytes.Split(line, []byte{enc.Field})
	fields := make([]Field, len(rawFields))
	for i, rf := range rawFields {
		fields[i] = parseField(rf, enc)
	}
	return fields
}

// parseField splits one field's bytes into repetitions/components/subcomponents.
func parseField(b []byte, enc EncodingCharacters) Field {
	rawReps := bytes.Split(b, []byte{enc.Repetition})
	reps := make([]Repetition, len(rawReps))
	for i, rr := range rawReps {
		reps[i] = parseRepetition(rr, enc)
	}
	return Field{Repetitions: reps}
}

// parseRepetition splits one repetition into components and subcomponents.
func parseRepetition(b []byte, enc EncodingCharacters) Repetition {
	rawComps := bytes.Split(b, []byte{enc.Component})
	comps := make([]Component, len(rawComps))
	for i, rc := range rawComps {
		rawSubs := bytes.Split(rc, []byte{enc.Subcomponent})
		subs := make([]string, len(rawSubs))
		for j, rs := range rawSubs {
			subs[j] = string(rs)
		}
		comps[i] = Component{Subcomponents: subs}
	}
	return Repetition{Components: comps}
}

// leafField builds a single-subcomponent field carrying b verbatim, used for the
// MSH delimiter fields that must not be further split.
func leafField(b []byte) Field {
	return Field{Repetitions: []Repetition{{Components: []Component{{Subcomponents: []string{string(b)}}}}}}
}
