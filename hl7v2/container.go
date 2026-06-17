package hl7v2

import (
	"bytes"
	"encoding"
)

// Container is implemented by Message, Batch, and File so ParseAny can return any
// of them and a caller can render any of them. Every container renders back to
// the bytes it parsed from and exposes the EncodingCharacters in force.
type Container interface {
	encoding.TextMarshaler
	Encoding() EncodingCharacters
}

// Batch is the HL7 batch-protocol container: an optional BHS/BTS header and
// trailer wrapping an ordered list of messages. The header and trailer are
// present together or not at all — a header without its trailer (or a trailer
// without its header) is a malformed batch, matching python-hl7's
// MalformedBatchException boundary. A bare sequence of MSH-led messages with no
// BHS/BTS is a valid header-less batch.
type Batch struct {
	Header   *Segment // BHS, or nil for a header-less batch
	Trailer  *Segment // BTS, or nil for a header-less batch
	Messages []*Message
	Enc      EncodingCharacters
}

// File is the HL7 batch-protocol file container: an optional FHS/FTS header and
// trailer wrapping an ordered list of batches. "File" here is the HL7 container,
// not an OS or .dcm file. The both-or-neither header/trailer rule applies as it
// does for Batch.
type File struct {
	Header  *Segment // FHS, or nil for a header-less file
	Trailer *Segment // FTS, or nil for a header-less file
	Batches []*Batch
	Enc     EncodingCharacters
}

// ParseBatch decodes a BHS/BTS batch (or a bare sequence of messages with no
// BHS/BTS) into a Batch. Encoding characters are derived from the leading
// segment, never a global default, so a sender using non-standard delimiters
// round-trips correctly. A BHS without a BTS, or a BTS without a BHS, is a
// *ParseError (the both-or-neither rule).
func ParseBatch(b []byte, opts ...ParseOption) (*Batch, error) {
	cfg := newParseConfig(opts...)
	b, err := cfg.decodeCharset(b)
	if err != nil {
		return nil, err
	}

	enc, err := deriveContainerEncoding(b)
	if err != nil {
		return nil, err
	}
	segments, err := splitSegments(b, enc)
	if err != nil {
		return nil, err
	}
	return batchFromSegments(segments, enc)
}

// ParseFile decodes an FHS/FTS file (or a bare batch) into a File. The
// both-or-neither rule applies to the FHS/FTS pair, and each inner batch is
// parsed by the same batch rules.
func ParseFile(b []byte, opts ...ParseOption) (*File, error) {
	cfg := newParseConfig(opts...)
	b, err := cfg.decodeCharset(b)
	if err != nil {
		return nil, err
	}

	enc, err := deriveContainerEncoding(b)
	if err != nil {
		return nil, err
	}
	segments, err := splitSegments(b, enc)
	if err != nil {
		return nil, err
	}
	return fileFromSegments(segments, enc)
}

// ParseAny dispatches on the leading segment: a single-MSH body yields a
// *Message, a bare body with more than one MSH and no BHS/FHS yields a *Batch
// (the header-less batch), FHS yields a *File, and BHS yields a *Batch. It
// returns the concrete container in the Container interface so a caller can
// render any of them. A body whose first segment is not MSH, BHS, or FHS is a
// *ParseError.
func ParseAny(b []byte, opts ...ParseOption) (Container, error) {
	switch leadingSegmentID(b) {
	case "MSH":
		// A bare sequence of multiple MSH-led messages is a header-less batch:
		// routing it through Parse would flatten every segment into a single
		// *Message and lose the batch grouping, so callers could not address each
		// message. A single-MSH body is an ordinary *Message.
		if countMSHSegments(b) > 1 {
			return ParseBatch(b, opts...)
		}
		return Parse(b, opts...)
	case "FHS":
		return ParseFile(b, opts...)
	case "BHS":
		return ParseBatch(b, opts...)
	default:
		return nil, &ParseError{Offset: 0, Reason: "container must begin with an MSH, BHS, or FHS segment"}
	}
}

// countMSHSegments counts the MSH-led segments in b so ParseAny can tell a single
// message from a header-less batch of several. It only inspects the start of each
// segment line, so a malformed body is left for ParseBatch or Parse to diagnose
// with a precise *ParseError.
func countMSHSegments(b []byte) int {
	count := 0
	offset := 0
	for offset < len(b) {
		line, term, next := nextLine(b, offset)
		if len(line) == 0 && term == "" {
			break
		}
		if len(line) >= 3 && line[0] == 'M' && line[1] == 'S' && line[2] == 'H' {
			count++
		}
		offset = next
	}
	return count
}

// fileFromSegments groups a flat segment list into a File: an optional FHS/FTS
// pair around one or more batches. The FHS and FTS must both be present or both
// absent.
func fileFromSegments(segments []Segment, enc EncodingCharacters) (*File, error) {
	hasFHS := len(segments) > 0 && segments[0].ID() == "FHS"
	lastIdx := len(segments) - 1
	hasFTS := lastIdx >= 0 && segments[lastIdx].ID() == "FTS"

	if hasFHS != hasFTS {
		return nil, &ParseError{Offset: 0, Reason: "malformed file: FHS header and FTS trailer must both be present or both absent"}
	}

	file := &File{Enc: enc}
	body := segments
	if hasFHS {
		header := segments[0]
		trailer := segments[lastIdx]
		file.Header = &header
		file.Trailer = &trailer
		body = segments[1:lastIdx]
	}

	// A stray FTS or FHS inside the body is a second-trailer/second-header fault.
	if err := rejectNestedFileSegments(body); err != nil {
		return nil, err
	}

	batches, err := splitBatches(body, enc)
	if err != nil {
		return nil, err
	}
	file.Batches = batches
	return file, nil
}

// splitBatches groups the body segments of a file into one or more batches on
// each BHS boundary, applying the batch both-or-neither rule to each group. A
// file with no inner BHS wraps its messages in a single header-less batch.
func splitBatches(body []Segment, enc EncodingCharacters) ([]*Batch, error) {
	if len(body) == 0 {
		return nil, nil
	}

	var batches []*Batch
	var group []Segment
	flush := func() error {
		if len(group) == 0 {
			return nil
		}
		batch, err := batchFromSegments(group, enc)
		if err != nil {
			return err
		}
		batches = append(batches, batch)
		group = nil
		return nil
	}

	for _, seg := range body {
		if seg.ID() == "BHS" && len(group) > 0 {
			if err := flush(); err != nil {
				return nil, err
			}
		}
		group = append(group, seg)
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return batches, nil
}

// batchFromSegments groups a flat segment list into a Batch: an optional BHS/BTS
// pair around the MSH-led messages. The BHS and BTS must both be present or both
// absent. A second BHS inside the batch is a malformed batch.
func batchFromSegments(segments []Segment, enc EncodingCharacters) (*Batch, error) {
	hasBHS := len(segments) > 0 && segments[0].ID() == "BHS"
	lastIdx := len(segments) - 1
	hasBTS := lastIdx >= 0 && segments[lastIdx].ID() == "BTS"

	if hasBHS != hasBTS {
		return nil, &ParseError{Offset: 0, Reason: "malformed batch: BHS header and BTS trailer must both be present or both absent"}
	}

	batch := &Batch{Enc: enc}
	body := segments
	if hasBHS {
		header := segments[0]
		trailer := segments[lastIdx]
		batch.Header = &header
		batch.Trailer = &trailer
		body = segments[1:lastIdx]
	}

	if err := rejectNestedBatchSegments(body); err != nil {
		return nil, err
	}

	messages, err := messagesFromSegments(body, enc)
	if err != nil {
		return nil, err
	}
	batch.Messages = messages
	return batch, nil
}

// messagesFromSegments groups the body segments of a batch into messages on each
// MSH boundary, preserving each segment's terminator so a message round-trips
// byte-exactly. A non-MSH segment before the first MSH is a malformed batch.
func messagesFromSegments(body []Segment, enc EncodingCharacters) ([]*Message, error) {
	if len(body) == 0 {
		return nil, nil
	}

	var messages []*Message
	var current *Message
	for _, seg := range body {
		if seg.ID() == "MSH" {
			current = &Message{Enc: enc}
			messages = append(messages, current)
		}
		if current == nil {
			return nil, &ParseError{Offset: 0, Reason: "malformed batch: a segment precedes the first MSH"}
		}
		current.Segments = append(current.Segments, seg)
	}
	return messages, nil
}

// rejectNestedBatchSegments fails on a BHS or BTS appearing inside a batch body,
// which would be a second header or trailer.
func rejectNestedBatchSegments(body []Segment) error {
	for _, seg := range body {
		switch seg.ID() {
		case "BHS":
			return &ParseError{Offset: 0, Reason: "malformed batch: a second BHS appears inside the batch"}
		case "BTS":
			return &ParseError{Offset: 0, Reason: "malformed batch: a BTS appears inside the batch body"}
		}
	}
	return nil
}

// rejectNestedFileSegments fails on an FHS or FTS appearing inside a file body,
// which would be a second header or trailer.
func rejectNestedFileSegments(body []Segment) error {
	for _, seg := range body {
		switch seg.ID() {
		case "FHS":
			return &ParseError{Offset: 0, Reason: "malformed file: a second FHS appears inside the file"}
		case "FTS":
			return &ParseError{Offset: 0, Reason: "malformed file: an FTS appears inside the file body"}
		}
	}
	return nil
}

// MarshalText renders the batch back to bytes, reproducing a conformant input
// byte-for-byte: the optional BHS/BTS, every contained message, and each
// segment's original terminator are preserved.
func (b *Batch) MarshalText() ([]byte, error) {
	var buf bytes.Buffer
	if b.Header != nil {
		b.Header.render(&buf, b.Enc)
	}
	for _, m := range b.Messages {
		for i := range m.Segments {
			m.Segments[i].render(&buf, m.Enc)
		}
	}
	if b.Trailer != nil {
		b.Trailer.render(&buf, b.Enc)
	}
	return buf.Bytes(), nil
}

// Encoding returns the EncodingCharacters in force for the batch.
func (b *Batch) Encoding() EncodingCharacters { return b.Enc }

// String renders the batch as MarshalText would.
func (b *Batch) String() string {
	out, _ := b.MarshalText()
	return string(out)
}

// MarshalText renders the file back to bytes, reproducing a conformant input
// byte-for-byte: the optional FHS/FTS, every contained batch, and each segment's
// original terminator are preserved.
func (f *File) MarshalText() ([]byte, error) {
	var buf bytes.Buffer
	if f.Header != nil {
		f.Header.render(&buf, f.Enc)
	}
	for _, batch := range f.Batches {
		out, err := batch.MarshalText()
		if err != nil {
			return nil, err
		}
		buf.Write(out)
	}
	if f.Trailer != nil {
		f.Trailer.render(&buf, f.Enc)
	}
	return buf.Bytes(), nil
}

// Encoding returns the EncodingCharacters in force for the file.
func (f *File) Encoding() EncodingCharacters { return f.Enc }

// String renders the file as MarshalText would.
func (f *File) String() string {
	out, _ := f.MarshalText()
	return string(out)
}

// leadingSegmentID returns the three-character segment ID at the start of b, or
// "" when b is too short to carry one.
func leadingSegmentID(b []byte) string {
	if len(b) < 3 {
		return ""
	}
	return string(b[:3])
}

// deriveContainerEncoding reads the encoding characters from a container's
// leading BHS, FHS, or MSH segment. All three share the same delimiter layout —
// the field separator at offset 3 and the encoding characters from offset 4 — so
// derivation is identical regardless of which leads. A body that does not begin
// with one of these segments is a *ParseError.
func deriveContainerEncoding(b []byte) (EncodingCharacters, error) {
	id := leadingSegmentID(b)
	if id != "MSH" && id != "BHS" && id != "FHS" {
		return EncodingCharacters{}, &ParseError{Offset: 0, Reason: "container must begin with an MSH, BHS, or FHS segment"}
	}
	if len(b) < 4 {
		return EncodingCharacters{}, truncatedAt(len(b), "container header truncated before its field separator")
	}
	fieldSep := b[3]
	if indexByte(b[4:], fieldSep) < 0 {
		return EncodingCharacters{}, truncatedAt(len(b), "container header truncated before its encoding characters are terminated")
	}

	enc := DefaultEncoding()
	enc.Field = fieldSep
	end := indexByte(b[4:], fieldSep)
	msh2 := b[4 : 4+end]
	if len(msh2) > 0 {
		enc.Component = msh2[0]
	}
	if len(msh2) > 1 {
		enc.Repetition = msh2[1]
	}
	if len(msh2) > 2 {
		enc.Escape = msh2[2]
	}
	if len(msh2) > 3 {
		enc.Subcomponent = msh2[3]
	}
	return enc, nil
}
