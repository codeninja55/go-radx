package hl7v2

import "bytes"

// IsHL7 reports whether b looks like a single HL7 v2 message: it begins (after
// leading whitespace) with an MSH segment ID followed by a field separator, and
// carries no second MSH segment. The second-MSH exclusion is what distinguishes a
// lone message from a header-less batch, matching python-hl7's ishl7, whose check
// is `msh[:3] == "MSH" and line.count("\rMSH" + sep) == 0`.
//
// This is a cursory, non-parsing sniff: it does not validate structure, so a body
// IsHL7 accepts can still fail Parse. It exists so a caller can route bytes to
// Parse, ParseBatch, or ParseFile without first parsing and inspecting the result.
func IsHL7(b []byte) bool {
	trimmed := bytes.TrimLeft(b, " \t\r\n")
	if len(trimmed) < 4 {
		return false
	}
	if trimmed[0] != 'M' || trimmed[1] != 'S' || trimmed[2] != 'H' {
		return false
	}
	sep := trimmed[3]
	// A "\rMSH<sep>" run anywhere is a second message header, so the body is a
	// batch rather than a lone message. The leading MSH itself is not preceded by
	// a CR, so it never matches.
	return !bytes.Contains(b, []byte{'\r', 'M', 'S', 'H', sep})
}

// IsBatch reports whether b looks like an HL7 batch: it begins (after leading
// whitespace) with a BHS segment, or it carries more than one MSH and does not
// begin with FHS. This mirrors python-hl7's isbatch, which treats a bare sequence
// of several messages as a header-less batch. A single-message body is not a
// batch, and an FHS-led body is a file, not a batch.
func IsBatch(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	id := leadingTrimmedID(b)
	if id == "BHS" {
		return true
	}
	return countMSHSegments(b) > 1 && id != "FHS"
}

// IsFile reports whether b looks like an HL7 file: it begins (after leading
// whitespace) with an FHS segment, or it is a batch. This mirrors python-hl7's
// isfile, where a file is an FHS/FTS wrapper or, failing that, anything that
// already looks like a batch.
func IsFile(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	return leadingTrimmedID(b) == "FHS" || IsBatch(b)
}

// leadingTrimmedID returns the three-character segment ID at the start of b after
// stripping leading whitespace, or "" when b is too short to carry one. The sniff
// predicates strip leading whitespace the way python-hl7's line.strip() does.
func leadingTrimmedID(b []byte) string {
	trimmed := bytes.TrimLeft(b, " \t\r\n")
	if len(trimmed) < 3 {
		return ""
	}
	return string(trimmed[:3])
}

// SplitFile splits a buffer that may carry a batch or file wrapper into its
// individual messages, throwing away the FHS, BHS, FTS, and BTS framing segments.
// It is the byte-level counterpart of python-hl7's split_file: it splits on the
// segment terminator, drops any line whose ID is FHS, BHS, FTS, or BTS, starts a
// new message at each MSH, and appends every following segment to the message in
// progress. A segment appearing before the first MSH is dropped (python-hl7 logs
// and skips it); SplitFile drops it silently rather than logging, because a
// segment line may carry PHI that must never reach a log.
//
// Each returned message is a self-contained byte slice with every segment
// terminated by a carriage return, so each element parses directly with Parse.
// SplitFile does no structural validation; an element it returns can still fail
// Parse. A buffer with no MSH yields nil.
//
// SplitFile accepts \r, \n, and \r\n segment terminators on input and normalises
// to \r on output, matching python-hl7, which splits on \r and re-joins with \r.
func SplitFile(b []byte) [][]byte {
	var messages [][]byte
	var current []byte
	offset := 0
	for offset < len(b) {
		line, term, next := nextLine(b, offset)
		offset = next
		if len(line) == 0 && term == "" {
			break
		}
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		switch segmentID(trimmed) {
		case "FHS", "BHS", "FTS", "BTS":
			continue
		case "MSH":
			if current != nil {
				messages = append(messages, current)
			}
			current = append([]byte(nil), trimmed...)
			current = append(current, '\r')
		default:
			if current == nil {
				// A segment before the first MSH has no message to attach to.
				continue
			}
			current = append(current, trimmed...)
			current = append(current, '\r')
		}
	}
	if current != nil {
		messages = append(messages, current)
	}
	return messages
}

// segmentID returns the leading three-character ID of a segment line, or the whole
// line when it is shorter than three bytes.
func segmentID(line []byte) string {
	if len(line) < 3 {
		return string(line)
	}
	return string(line[:3])
}
