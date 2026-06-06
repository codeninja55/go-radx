package hl7v2

import (
	"strconv"
	"strings"
)

// Accessor is a parsed 1-based HL7 path of the form SEG[n]-Fn-Rn-Cn-Sn. It
// mirrors the HL7 spec numbering exactly so a path transcribed from segment
// documentation resolves directly, never the 0-based Go slice numbering used by
// the generic tree. SegmentNum selects which instance of a repeated segment
// (1 = first); a zero Field means the path stops at the segment.
type Accessor struct {
	Segment      string // three-character segment ID, e.g. "PID"
	SegmentNum   int    // 1-based segment instance; 1 if omitted
	Field        int    // 1-based HL7 field number; 0 if not in the path
	Repetition   int
	Component    int
	Subcomponent int
}

// ParseAccessor parses both accepted styles into an Accessor: the numeric form
// "PID-5-1-2" (also accepting the dotted "PID.5.1.2"), and the prefixed form
// "PID.F5.R1.C2". A trailing segment-instance index is written inline, so
// "PID2-5" selects field 5 of the second PID. The two styles never mix within
// one key. A malformed key — a segment ID that is not three characters, a
// non-positive or non-numeric index, an out-of-order or unknown prefix —
// returns an *AccessorError.
func ParseAccessor(key string) (Accessor, error) {
	if key == "" {
		return Accessor{}, &AccessorError{Key: key, Reason: "empty accessor key"}
	}

	parts := splitAccessor(key)
	seg, segNum, err := parseSegmentToken(key, parts[0])
	if err != nil {
		return Accessor{}, err
	}
	a := Accessor{Segment: seg, SegmentNum: segNum}

	// Levels descend field -> repetition -> component -> subcomponent. The
	// numeric style fills them positionally; the prefixed style names each one
	// and must still appear in that order, with no level skipped, so a key reads
	// like the spec path.
	levels := []*int{&a.Field, &a.Repetition, &a.Component, &a.Subcomponent}
	prefixes := []byte{'F', 'R', 'C', 'S'}
	pos := 0
	styleSet := false
	stylePrefixed := false

	for _, p := range parts[1:] {
		prefixed := len(p) > 0 && p[0] >= 'A' && p[0] <= 'Z'
		// The numeric and prefixed styles must not mix within one key: the style is
		// fixed by the first level token and every later level must match it.
		if !styleSet {
			stylePrefixed, styleSet = prefixed, true
		} else if prefixed != stylePrefixed {
			return Accessor{}, &AccessorError{Key: key, Reason: "accessor key mixes numeric and prefixed level styles"}
		}
		if prefixed {
			idx := indexOfPrefix(prefixes, p[0])
			if idx < 0 {
				return Accessor{}, &AccessorError{Key: key, Reason: "unknown level prefix in accessor key"}
			}
			if idx != pos {
				return Accessor{}, &AccessorError{Key: key, Reason: "accessor level prefixes are out of order or skip a level"}
			}
			p = p[1:]
		}
		if pos >= len(levels) {
			return Accessor{}, &AccessorError{Key: key, Reason: "accessor key descends past the subcomponent level"}
		}

		n, perr := strconv.Atoi(p)
		if perr != nil || n < 1 {
			return Accessor{}, &AccessorError{Key: key, Reason: "accessor index must be a positive integer"}
		}
		*levels[pos] = n
		pos++
	}

	return a, nil
}

// splitAccessor breaks a key on its '-' or '.' separators without collapsing
// empty tokens, so a doubled separator such as "PID--5" surfaces an empty token
// that parsing rejects rather than being silently absorbed. The two separators
// are interchangeable so both the "PID-5-1" and "PID.5.1" spellings split the
// same way.
func splitAccessor(key string) []string {
	parts := []string{}
	start := 0
	for i := 0; i < len(key); i++ {
		if key[i] == '-' || key[i] == '.' {
			parts = append(parts, key[start:i])
			start = i + 1
		}
	}
	parts = append(parts, key[start:])
	return parts
}

// parseSegmentToken splits a segment token into its three-character ID and its
// optional 1-based instance number. "PID" yields ("PID", 1) and "PID2" yields
// ("PID", 2). A token whose ID is not exactly three characters, or whose
// instance suffix is non-positive or non-numeric, is malformed.
func parseSegmentToken(key, token string) (string, int, error) {
	if len(token) < 3 {
		return "", 0, &AccessorError{Key: key, Reason: "segment ID must be three characters"}
	}
	seg := token[:3]
	suffix := token[3:]
	if suffix == "" {
		return seg, 1, nil
	}
	n, err := strconv.Atoi(suffix)
	if err != nil || n < 1 {
		return "", 0, &AccessorError{Key: key, Reason: "segment instance must be a positive integer"}
	}
	return seg, n, nil
}

// indexOfPrefix returns the position of b in prefixes, or -1 when absent.
func indexOfPrefix(prefixes []byte, b byte) int {
	for i, p := range prefixes {
		if p == b {
			return i
		}
	}
	return -1
}

// String renders the accessor in its canonical numeric form, e.g. "PID-5-1-2".
// A non-default segment instance is written inline as "PID2-5". Trailing levels
// that were never set are omitted; a segment-only path (no field) renders as just
// the segment — "PID" or "PID2" — so the value round-trips through ParseAccessor
// and addresses the segment, not field 1.
func (a Accessor) String() string {
	var b strings.Builder
	b.WriteString(a.Segment)
	if a.SegmentNum > 1 {
		b.WriteString(strconv.Itoa(a.SegmentNum))
	}
	if a.Field == 0 {
		return b.String()
	}
	b.WriteByte('-')
	b.WriteString(strconv.Itoa(a.Field))
	for _, n := range []int{a.Repetition, a.Component, a.Subcomponent} {
		if n == 0 {
			break
		}
		b.WriteByte('-')
		b.WriteString(strconv.Itoa(n))
	}
	return b.String()
}

// Get resolves a 1-based accessor key against the message and returns the leaf
// value. An absent optional position — a missing segment instance, field,
// repetition, or component — returns ("", nil), matching the standard's
// optional-field semantics. A path that descends past a leaf — asking for a
// component or subcomponent of a value that has none beyond index 1 — returns
// an *AccessorError, because that is a malformed request rather than an absent
// optional.
//
// Every leaf is unescaped against the message's EncodingCharacters, except
// MSH-1 and MSH-2: those are the delimiters and the encoding characters
// themselves, so they are returned verbatim (unescaping them would be circular).
func (m *Message) Get(key string) (string, error) {
	a, err := ParseAccessor(key)
	if err != nil {
		return "", err
	}

	seg, ok := m.segmentInstance(a.Segment, a.SegmentNum)
	if !ok {
		return "", nil
	}
	if a.Field == 0 {
		return seg.ID(), nil
	}
	if a.Field >= len(seg.Fields) {
		return "", nil
	}

	raw, ok, err := resolveLeaf(seg.Fields[a.Field], a, key)
	if err != nil || !ok {
		return "", err
	}

	if a.Segment == "MSH" && (a.Field == 1 || a.Field == 2) {
		return raw, nil
	}
	decoded, _ := Unescape(raw, m.Enc)
	return decoded, nil
}

// resolveLeaf descends the repetition/component/subcomponent levels of one field
// following the HL7 "future-proofed" rule python-hl7 implements in extract_field:
// a level the path omits defaults to index 1, a path shallower than the tree
// descends the first child to a leaf, and a path deeper than the tree is the
// leaf itself when every extra index is 1. It returns the raw (still-escaped)
// leaf and whether the position is present.
//
// A repetition index beyond the field's repetitions is treated as an absent
// optional repetition, because a repeated field legitimately carries fewer
// repetitions than the sender might send. A component or subcomponent index
// above 1 that has no corresponding child is a request that descends past a
// leaf — asking for a deeper part of a value that has none — and yields an
// *AccessorError.
func resolveLeaf(f Field, a Accessor, key string) (string, bool, error) {
	rep := defaultIndex(a.Repetition)
	if rep > len(f.Repetitions) {
		return "", false, nil
	}
	r := f.Repetitions[rep-1]

	comp := defaultIndex(a.Component)
	if comp > len(r.Components) {
		if comp == 1 {
			return "", true, nil
		}
		return "", false, &AccessorError{Key: key, Reason: "path descends past a leaf"}
	}
	c := r.Components[comp-1]

	sub := defaultIndex(a.Subcomponent)
	if sub > len(c.Subcomponents) {
		if sub == 1 {
			return "", true, nil
		}
		return "", false, &AccessorError{Key: key, Reason: "path descends past a leaf"}
	}
	return c.Subcomponents[sub-1], true, nil
}

// defaultIndex maps an unset (zero) 1-based level to 1, the HL7 default for an
// omitted repetition/component/subcomponent.
func defaultIndex(n int) int {
	if n == 0 {
		return 1
	}
	return n
}

// segmentInstance returns the n-th 1-based instance of the segment with the
// given ID, and whether it is present.
func (m *Message) segmentInstance(id string, n int) (Segment, bool) {
	count := 0
	for _, seg := range m.Segments {
		if seg.ID() != id {
			continue
		}
		count++
		if count == n {
			return seg, true
		}
	}
	return Segment{}, false
}

// Set assigns value at the accessor key, escaping value against the message's
// EncodingCharacters so a delimiter byte in value cannot forge structure, then
// growing fields, repetitions, components, and subcomponents as needed to reach
// the path. The target segment must already exist: Set never invents a segment,
// so a Set against an absent segment instance returns an *AccessorError. MSH-1
// and MSH-2 are rejected because assigning them would desynchronize the message
// from the delimiters it is parsed and rendered with.
func (m *Message) Set(key, value string) error {
	a, err := ParseAccessor(key)
	if err != nil {
		return err
	}
	if a.Field == 0 {
		return &AccessorError{Key: key, Reason: "Set requires a field; a segment ID alone is not assignable"}
	}
	if a.Segment == "MSH" && (a.Field == 1 || a.Field == 2) {
		return &AccessorError{Key: key, Reason: "MSH-1 and MSH-2 are the encoding characters and are not assignable"}
	}

	idx, ok := m.segmentInstanceIndex(a.Segment, a.SegmentNum)
	if !ok {
		return &AccessorError{Key: key, Reason: "Set targets an absent segment; Set never invents a segment"}
	}

	rep := defaultIndex(a.Repetition)
	comp := defaultIndex(a.Component)
	sub := defaultIndex(a.Subcomponent)

	seg := &m.Segments[idx]
	growFields(seg, a.Field)
	field := &seg.Fields[a.Field]
	growRepetitions(field, rep)
	r := &field.Repetitions[rep-1]
	growComponents(r, comp)
	c := &r.Components[comp-1]
	growSubcomponents(c, sub)
	c.Subcomponents[sub-1] = Escape(value, m.Enc)
	return nil
}

// segmentInstanceIndex returns the index into m.Segments of the n-th 1-based
// instance of the segment with the given ID, and whether it is present.
func (m *Message) segmentInstanceIndex(id string, n int) (int, bool) {
	count := 0
	for i := range m.Segments {
		if m.Segments[i].ID() != id {
			continue
		}
		count++
		if count == n {
			return i, true
		}
	}
	return 0, false
}

// growFields extends a segment's field slice so field index n (1-based) is
// addressable, leaving any new intermediate fields empty.
func growFields(s *Segment, n int) {
	for len(s.Fields) <= n {
		s.Fields = append(s.Fields, emptyField())
	}
}

// growRepetitions extends a field's repetition slice so repetition n (1-based)
// is addressable.
func growRepetitions(f *Field, n int) {
	for len(f.Repetitions) < n {
		f.Repetitions = append(f.Repetitions, emptyRepetition())
	}
}

// growComponents extends a repetition's component slice so component n (1-based)
// is addressable.
func growComponents(r *Repetition, n int) {
	for len(r.Components) < n {
		r.Components = append(r.Components, Component{Subcomponents: []string{""}})
	}
}

// growSubcomponents extends a component's subcomponent slice so subcomponent n
// (1-based) is addressable.
func growSubcomponents(c *Component, n int) {
	for len(c.Subcomponents) < n {
		c.Subcomponents = append(c.Subcomponents, "")
	}
}

// emptyField builds a field holding a single empty leaf, the shape parseField
// produces for an empty field token.
func emptyField() Field {
	return Field{Repetitions: []Repetition{emptyRepetition()}}
}

// emptyRepetition builds a repetition holding a single empty leaf.
func emptyRepetition() Repetition {
	return Repetition{Components: []Component{{Subcomponents: []string{""}}}}
}
