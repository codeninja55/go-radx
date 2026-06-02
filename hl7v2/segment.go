package hl7v2

import (
	"bytes"
	"strconv"
	"strings"
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
	if s.ID() == "MSH" {
		s.renderMSH(buf, enc)
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

// renderMSH reproduces the MSH header's quirk: Fields[1] (the field separator)
// and Fields[2] (the encoding characters) are emitted verbatim, and the field
// separator that ordinarily joins them is implicit in Fields[1] itself.
func (s Segment) renderMSH(buf *bytes.Buffer, enc EncodingCharacters) {
	buf.WriteString(s.Fields[0].raw()) // "MSH"
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

// Get resolves a 1-based HL7 accessor key of the form "SEG-F-R-C-S" (or the
// dotted "SEG.F.R.C.S" form) against the message and returns the leaf value, or
// "" with no error when the position is simply absent. This is the minimal M2
// accessor: it descends field/repetition/component/subcomponent in 1-based HL7
// spec numbering and is sufficient for the ORM-feeding slice. The full
// future-proofed resolution and the prefixed key form arrive in M5.
func (m *Message) Get(key string) (string, error) {
	parts := strings.FieldsFunc(key, func(r rune) bool { return r == '-' || r == '.' })
	if len(parts) == 0 {
		return "", &ParseError{Offset: 0, Reason: "empty accessor key"}
	}

	segID := parts[0]
	seg, ok := m.Segment(segID)
	if !ok {
		return "", nil
	}
	if len(parts) == 1 {
		return seg.ID(), nil
	}

	nums := make([]int, len(parts)-1)
	for i, p := range parts[1:] {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 {
			return "", &ParseError{Offset: 0, Reason: "accessor key has a non-positive or non-numeric index"}
		}
		nums[i] = n
	}

	return resolveSegment(seg, nums), nil
}

// resolveSegment resolves field/repetition/component/subcomponent indices.
// HL7 field N lives at Fields[N] for every segment: ordinary segments store the
// ID at Fields[0], and MSH additionally stores MSH-1/MSH-2 (the field separator
// and encoding characters) at Fields[1]/Fields[2], so the field-N-at-Fields[N]
// mapping holds uniformly and MSH-1/MSH-2 read back verbatim.
func resolveSegment(seg Segment, nums []int) string {
	fieldNum := nums[0]
	if fieldNum >= len(seg.Fields) {
		return ""
	}
	return resolveField(seg.Fields[fieldNum], nums[1:])
}

// resolveField descends the remaining 1-based repetition/component/subcomponent
// indices into f.
func resolveField(f Field, nums []int) string {
	if len(nums) == 0 {
		return f.raw()
	}
	repNum := nums[0]
	if repNum < 1 || repNum > len(f.Repetitions) {
		return ""
	}
	rep := f.Repetitions[repNum-1]

	if len(nums) == 1 {
		return rep.raw()
	}
	compNum := nums[1]
	if compNum < 1 || compNum > len(rep.Components) {
		return ""
	}
	comp := rep.Components[compNum-1]

	if len(nums) == 2 {
		if len(comp.Subcomponents) == 0 {
			return ""
		}
		return comp.Subcomponents[0]
	}
	subNum := nums[2]
	if subNum < 1 || subNum > len(comp.Subcomponents) {
		return ""
	}
	return comp.Subcomponents[subNum-1]
}
