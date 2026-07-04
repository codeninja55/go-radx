package hl7v2

import (
	"errors"
	"testing"
)

// A minimal two-message batch with BHS/BTS, canonical \r terminators.
const canonicalBatch = "BHS|^~\\&|REGADT|HOSP|EMR|HOSP|202605310900||||BATCH0001\r" +
	"MSH|^~\\&|REGADT|HOSP|EMR|HOSP|202605310901||ADT^A04^ADT_A01|MSGBAT0001|P|2.5.1\r" +
	"EVN|A04|202605310901\r" +
	"PID|1||MRN0005005^^^HOSP^MR||ALPHA^ERIN^E^^^^L||19700707|F\r" +
	"PV1|1|O\r" +
	"MSH|^~\\&|REGADT|HOSP|EMR|HOSP|202605310902||ADT^A04^ADT_A01|MSGBAT0002|P|2.5.1\r" +
	"EVN|A04|202605310902\r" +
	"PID|1||MRN0006006^^^HOSP^MR||BETA^FINN^F^^^^L||19651212|M\r" +
	"PV1|1|O\r" +
	"BTS|2\r"

// A file with FHS/FTS wrapping one batch of one message.
const canonicalFile = "FHS|^~\\&|REGADT|HOSP|EMR|HOSP|202605310930||||FILE0001.hl7\r" +
	"BHS|^~\\&|REGADT|HOSP|EMR|HOSP|202605310930||||BATCH0002\r" +
	"MSH|^~\\&|REGADT|HOSP|EMR|HOSP|202605310931||ADT^A04^ADT_A01|MSGFIL0001|P|2.5.1\r" +
	"EVN|A04|202605310931\r" +
	"PID|1||MRN0007007^^^HOSP^MR||GAMMA^GAIL^G^^^^L||19880228|F\r" +
	"PV1|1|O\r" +
	"BTS|1\r" +
	"FTS|1\r"

func TestParseBatch(t *testing.T) {
	batch, err := ParseBatch([]byte(canonicalBatch))
	if err != nil {
		t.Fatalf("ParseBatch error = %v", err)
	}
	if batch.Header == nil || batch.Header.ID() != "BHS" {
		t.Fatalf("Header = %+v, want a BHS segment", batch.Header)
	}
	if batch.Trailer == nil || batch.Trailer.ID() != "BTS" {
		t.Fatalf("Trailer = %+v, want a BTS segment", batch.Trailer)
	}
	if len(batch.Messages) != 2 {
		t.Fatalf("Messages = %d, want 2", len(batch.Messages))
	}
	// Each contained message is a parseable, typed ADT.
	for i, m := range batch.Messages {
		adt, ok := AsADT(m)
		if !ok || adt.Event() != "A04" {
			t.Errorf("message %d not an ADT^A04 (ok=%v)", i, ok)
		}
	}
}

func TestParseBatchRoundTripByteExact(t *testing.T) {
	batch, err := ParseBatch([]byte(canonicalBatch))
	if err != nil {
		t.Fatalf("ParseBatch error = %v", err)
	}
	out, err := batch.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText error = %v", err)
	}
	if string(out) != canonicalBatch {
		t.Errorf("batch round-trip mismatch:\n got = %q\nwant = %q", out, canonicalBatch)
	}
}

func TestParseFile(t *testing.T) {
	file, err := ParseFile([]byte(canonicalFile))
	if err != nil {
		t.Fatalf("ParseFile error = %v", err)
	}
	if file.Header == nil || file.Header.ID() != "FHS" {
		t.Fatalf("Header = %+v, want an FHS segment", file.Header)
	}
	if file.Trailer == nil || file.Trailer.ID() != "FTS" {
		t.Fatalf("Trailer = %+v, want an FTS segment", file.Trailer)
	}
	if len(file.Batches) != 1 {
		t.Fatalf("Batches = %d, want 1", len(file.Batches))
	}
	if got := len(file.Batches[0].Messages); got != 1 {
		t.Fatalf("inner batch Messages = %d, want 1", got)
	}
}

func TestParseFileRoundTripByteExact(t *testing.T) {
	file, err := ParseFile([]byte(canonicalFile))
	if err != nil {
		t.Fatalf("ParseFile error = %v", err)
	}
	out, err := file.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText error = %v", err)
	}
	if string(out) != canonicalFile {
		t.Errorf("file round-trip mismatch:\n got = %q\nwant = %q", out, canonicalFile)
	}
}

func TestParseBatchBareMessages(t *testing.T) {
	// A bare sequence of messages with no BHS/BTS parses as a header-less batch
	// (python-hl7's "implied" batch).
	bare := "MSH|^~\\&|A|B|C|D|202605311230||ADT^A04^ADT_A01|M1|P|2.5.1\r" +
		"PID|1\r" +
		"MSH|^~\\&|A|B|C|D|202605311231||ADT^A04^ADT_A01|M2|P|2.5.1\r" +
		"PID|1\r"
	batch, err := ParseBatch([]byte(bare))
	if err != nil {
		t.Fatalf("ParseBatch(bare) error = %v", err)
	}
	if batch.Header != nil || batch.Trailer != nil {
		t.Errorf("bare batch should have no header/trailer, got %+v / %+v", batch.Header, batch.Trailer)
	}
	if len(batch.Messages) != 2 {
		t.Fatalf("bare batch Messages = %d, want 2", len(batch.Messages))
	}
	out, err := batch.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText error = %v", err)
	}
	if string(out) != bare {
		t.Errorf("bare batch round-trip mismatch:\n got = %q\nwant = %q", out, bare)
	}
}

func TestBatchBothOrNeitherRule(t *testing.T) {
	// A BHS header with no BTS trailer is a malformed batch, and vice versa,
	// matching python-hl7's MalformedBatchException boundary.
	headerOnly := "BHS|^~\\&|A|B|C|D|202605310900||||BATCH0001\r" +
		"MSH|^~\\&|A|B|C|D|202605311230||ADT^A04^ADT_A01|M1|P|2.5.1\r" +
		"PID|1\r"
	if _, err := ParseBatch([]byte(headerOnly)); err == nil {
		t.Error("ParseBatch(header-only) = nil error, want a malformed-batch error")
	} else if _, ok := errors.AsType[*ParseError](err); !ok {
		t.Errorf("ParseBatch(header-only) error = %T, want *ParseError", err)
	}

	trailerOnly := "MSH|^~\\&|A|B|C|D|202605311230||ADT^A04^ADT_A01|M1|P|2.5.1\r" +
		"PID|1\r" +
		"BTS|1\r"
	if _, err := ParseBatch([]byte(trailerOnly)); err == nil {
		t.Error("ParseBatch(trailer-only) = nil error, want a malformed-batch error")
	}
}

func TestFileBothOrNeitherRule(t *testing.T) {
	// An FHS header with no FTS trailer is a malformed file.
	headerOnly := "FHS|^~\\&|A|B|C|D|202605310930||||FILE0001.hl7\r" +
		"BHS|^~\\&|A|B|C|D|202605310930||||BATCH0002\r" +
		"MSH|^~\\&|A|B|C|D|202605310931||ADT^A04^ADT_A01|M1|P|2.5.1\r" +
		"PID|1\r" +
		"BTS|1\r"
	if _, err := ParseFile([]byte(headerOnly)); err == nil {
		t.Error("ParseFile(header-only) = nil error, want a malformed-file error")
	} else if _, ok := errors.AsType[*ParseError](err); !ok {
		t.Errorf("ParseFile(header-only) error = %T, want *ParseError", err)
	}
}

func TestParseAnyDispatch(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string // the concrete container type tag
	}{
		{"single-message", canonicalADT, "*hl7v2.Message"},
		{"batch", canonicalBatch, "*hl7v2.Batch"},
		{"file", canonicalFile, "*hl7v2.File"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, err := ParseAny([]byte(tc.raw))
			if err != nil {
				t.Fatalf("ParseAny error = %v", err)
			}
			switch v := c.(type) {
			case *Message:
				if tc.want != "*hl7v2.Message" {
					t.Errorf("ParseAny = *Message, want %s", tc.want)
				}
			case *Batch:
				if tc.want != "*hl7v2.Batch" {
					t.Errorf("ParseAny = *Batch, want %s", tc.want)
				}
			case *File:
				if tc.want != "*hl7v2.File" {
					t.Errorf("ParseAny = *File, want %s", tc.want)
				}
			default:
				t.Errorf("ParseAny returned unexpected type %T", v)
			}
			// Every container round-trips byte-exactly through MarshalText.
			out, err := c.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText error = %v", err)
			}
			if string(out) != tc.raw {
				t.Errorf("ParseAny round-trip mismatch:\n got = %q\nwant = %q", out, tc.raw)
			}
		})
	}
}

func TestParseAnyBareMultiMessageYieldsBatch(t *testing.T) {
	// A bare body with more than one MSH and no BHS/FHS is a header-less batch, so
	// ParseAny must return a type-assertable *Batch carrying every message —
	// routing it through Parse would flatten the segments into one *Message and
	// lose the grouping.
	bare := "MSH|^~\\&|A|B|C|D|202605311230||ADT^A04^ADT_A01|M1|P|2.5.1\r" +
		"PID|1\r" +
		"MSH|^~\\&|A|B|C|D|202605311231||ADT^A04^ADT_A01|M2|P|2.5.1\r" +
		"PID|1\r"
	c, err := ParseAny([]byte(bare))
	if err != nil {
		t.Fatalf("ParseAny(bare multi-message) error = %v", err)
	}
	batch, ok := c.(*Batch)
	if !ok {
		t.Fatalf("ParseAny(bare multi-message) = %T, want *Batch", c)
	}
	if batch.Header != nil || batch.Trailer != nil {
		t.Errorf("header-less batch should have no BHS/BTS, got %+v / %+v", batch.Header, batch.Trailer)
	}
	if len(batch.Messages) != 2 {
		t.Fatalf("batch Messages = %d, want 2", len(batch.Messages))
	}
	out, err := c.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText error = %v", err)
	}
	if string(out) != bare {
		t.Errorf("bare batch round-trip mismatch:\n got = %q\nwant = %q", out, bare)
	}
}

func TestParseAnySingleMessageYieldsMessage(t *testing.T) {
	// A single-MSH body stays a *Message; only multi-MSH bodies become batches.
	single := "MSH|^~\\&|A|B|C|D|202605311230||ADT^A04^ADT_A01|M1|P|2.5.1\r" +
		"PID|1\r"
	c, err := ParseAny([]byte(single))
	if err != nil {
		t.Fatalf("ParseAny(single message) error = %v", err)
	}
	if _, ok := c.(*Message); !ok {
		t.Fatalf("ParseAny(single message) = %T, want *Message", c)
	}
}

func TestParseAnyRejectsUnknownLeadingSegment(t *testing.T) {
	if _, err := ParseAny([]byte("PID|1\r")); err == nil {
		t.Error("ParseAny on a non-MSH/BHS/FHS body = nil error, want *ParseError")
	}
}

func TestContainerInterfaceSatisfied(t *testing.T) {
	// Message, Batch, and File all satisfy Container so ParseAny can return any.
	var _ Container = (*Message)(nil)
	var _ Container = (*Batch)(nil)
	var _ Container = (*File)(nil)
}

func TestBatchEncodingDerivedFromHeader(t *testing.T) {
	// A batch whose BHS uses non-standard delimiters round-trips and exposes the
	// derived encoding, never the static defaults.
	nonstd := "BHS#@+$%#REGADT#HOSP#EMR#HOSP#202605310900#####BATCH0001\r" +
		"MSH#@+$%#REGADT#HOSP#EMR#HOSP#202605310901##ADT@A04@ADT_A01#M1#P#2.5.1\r" +
		"PID#1\r" +
		"BTS#1\r"
	batch, err := ParseBatch([]byte(nonstd))
	if err != nil {
		t.Fatalf("ParseBatch(non-standard) error = %v", err)
	}
	if batch.Enc.Field != '#' || batch.Enc.Component != '@' {
		t.Errorf("derived encoding = %+v, want field '#' component '@'", batch.Enc)
	}
	out, err := batch.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText error = %v", err)
	}
	if string(out) != nonstd {
		t.Errorf("non-standard batch round-trip mismatch:\n got = %q\nwant = %q", out, nonstd)
	}
}
