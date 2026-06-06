package hl7v2

import (
	"bytes"
)

// ID returns the segment's three-character identifier (Fields[0] rendered),
// e.g. "PID". An empty segment returns "".
func (s Segment) ID() string {
	if len(s.Fields) == 0 {
		return ""
	}
	return s.Fields[0].raw()
}

// Segment returns the first segment with the given three-character ID and true,
// or the zero Segment and false when none is present. An absent optional segment
// is normal, so absence is reported as false, never an error.
func (m *Message) Segment(id string) (Segment, bool) {
	for _, seg := range m.Segments {
		if seg.ID() == id {
			return seg, true
		}
	}
	return Segment{}, false
}

// AllSegments returns every segment with the given ID in document order, or an
// empty slice when none is present.
func (m *Message) AllSegments(id string) []Segment {
	var out []Segment
	for _, seg := range m.Segments {
		if seg.ID() == id {
			out = append(out, seg)
		}
	}
	return out
}

// Encoding returns the EncodingCharacters in force for the message.
func (m *Message) Encoding() EncodingCharacters { return m.Enc }

// MarshalText renders the message back to bytes, reproducing a conformant input
// byte-for-byte: the original delimiters, repetition structure, and per-segment
// terminators are preserved.
func (m *Message) MarshalText() ([]byte, error) {
	var buf bytes.Buffer
	for i := range m.Segments {
		m.Segments[i].render(&buf, m.Enc)
	}
	return buf.Bytes(), nil
}

// String renders the message as MarshalText would.
func (m *Message) String() string {
	b, _ := m.MarshalText()
	return string(b)
}

// render writes the segment and its terminator to buf using enc.
func (s Segment) render(buf *bytes.Buffer, enc EncodingCharacters) {
	if hasDelimiterFields(s.ID()) {
		s.renderHeader(buf, enc)
	} else {
		for i := range s.Fields {
			if i > 0 {
				buf.WriteByte(enc.Field)
			}
			s.Fields[i].render(buf, enc)
		}
	}
	buf.WriteString(s.term)
}

// renderHeader reproduces the delimiter quirk shared by MSH, BHS, and FHS:
// Fields[1] (the field separator) and Fields[2] (the encoding characters) are
// emitted verbatim, and the field separator that ordinarily joins them is
// implicit in Fields[1] itself.
func (s Segment) renderHeader(buf *bytes.Buffer, enc EncodingCharacters) {
	buf.WriteString(s.Fields[0].raw()) // the segment ID
	if len(s.Fields) > 1 {
		buf.WriteString(s.Fields[1].raw()) // the field separator, verbatim
	}
	if len(s.Fields) > 2 {
		buf.WriteString(s.Fields[2].raw()) // the encoding characters, verbatim
	}
	for i := 3; i < len(s.Fields); i++ {
		buf.WriteByte(enc.Field)
		s.Fields[i].render(buf, enc)
	}
}

func (f Field) render(buf *bytes.Buffer, enc EncodingCharacters) {
	for i := range f.Repetitions {
		if i > 0 {
			buf.WriteByte(enc.Repetition)
		}
		f.Repetitions[i].render(buf, enc)
	}
}

func (r Repetition) render(buf *bytes.Buffer, enc EncodingCharacters) {
	for i := range r.Components {
		if i > 0 {
			buf.WriteByte(enc.Component)
		}
		c := r.Components[i]
		for j := range c.Subcomponents {
			if j > 0 {
				buf.WriteByte(enc.Subcomponent)
			}
			buf.WriteString(c.Subcomponents[j])
		}
	}
}

// raw returns a field's first subcomponent of its first component of its first
// repetition — the value of a flat, single-valued field such as a segment ID or
// a delimiter field.
func (f Field) raw() string {
	if len(f.Repetitions) == 0 {
		return ""
	}
	return f.Repetitions[0].raw()
}

func (r Repetition) raw() string {
	if len(r.Components) == 0 {
		return ""
	}
	c := r.Components[0]
	if len(c.Subcomponents) == 0 {
		return ""
	}
	return c.Subcomponents[0]
}

// component returns the n-th 1-based component's first subcomponent of a field's
// first repetition, or "" when absent. It is the workhorse for the composite
// datatype parsers, which read fixed component positions.
func (f Field) component(n int) string {
	if len(f.Repetitions) == 0 {
		return ""
	}
	return f.Repetitions[0].component(n)
}

func (r Repetition) component(n int) string {
	if n < 1 || n > len(r.Components) {
		return ""
	}
	c := r.Components[n-1]
	if len(c.Subcomponents) == 0 {
		return ""
	}
	return c.Subcomponents[0]
}

// field returns the n-th 1-based HL7 field, or the zero Field when absent.
// Field 1 is Fields[1] for ordinary segments. The caller is responsible for the
// MSH offset quirk via the typed MSH view.
func (s Segment) field(n int) Field {
	if n < 0 || n >= len(s.Fields) {
		return Field{}
	}
	return s.Fields[n]
}
