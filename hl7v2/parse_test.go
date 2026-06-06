package hl7v2

import (
	"errors"
	"io"
	"strings"
	"testing"
)

// canonicalORM is a representative ORM^O01 order message with \r segment
// terminators: header, patient, common order, and one observation request. It
// is shaped after the worked examples in docs/reference/hl7v2.md.
const canonicalORM = "MSH|^~\\&|RADIS|HOSP|PACS|HOSP|202605311230||ORM^O01|MSG00001|P|2.4\r" +
	"PID|||555-44-4444^^^HOSP^MR||EVERYWOMAN^EVE^E^^^^L||19620320|F\r" +
	"ORC|NW|PLACER123|FILLER456||||||202605311230\r" +
	"OBR|1|PLACER123|FILLER456|36643-5^CHEST XRAY^LN|||202605311231\r"

func TestParseCanonicalORM(t *testing.T) {
	msg, err := Parse([]byte(canonicalORM))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}

	if got := len(msg.Segments); got != 4 {
		t.Fatalf("len(Segments) = %d, want 4", got)
	}

	wantIDs := []string{"MSH", "PID", "ORC", "OBR"}
	for i, want := range wantIDs {
		if got := msg.Segments[i].ID(); got != want {
			t.Errorf("Segments[%d].ID() = %q, want %q", i, got, want)
		}
	}

	if msg.Enc != DefaultEncoding() {
		t.Errorf("Enc = %+v, want default", msg.Enc)
	}
}

func TestParseRoundTripByteExact(t *testing.T) {
	msg, err := Parse([]byte(canonicalORM))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	out, err := msg.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText error = %v", err)
	}
	if string(out) != canonicalORM {
		t.Fatalf("round-trip mismatch:\n got = %q\nwant = %q", string(out), canonicalORM)
	}
}

func TestParseRoundTripLineEndings(t *testing.T) {
	// Lenient on \n and \r\n terminators; each round-trips byte-exact because
	// the original terminator is preserved per segment.
	for _, term := range []string{"\n", "\r\n"} {
		src := strings.ReplaceAll(canonicalORM, "\r", term)
		msg, err := Parse([]byte(src))
		if err != nil {
			t.Fatalf("Parse(%q terminators) error = %v", term, err)
		}
		out, _ := msg.MarshalText()
		if string(out) != src {
			t.Fatalf("round-trip with %q mismatch:\n got = %q\nwant = %q", term, string(out), src)
		}
	}
}

func TestParseNotMSHFirst(t *testing.T) {
	body := "PID|||555-44-4444\rMSH|^~\\&|\r"
	_, err := Parse([]byte(body))
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("Parse(non-MSH-first) error = %v, want *ParseError", err)
	}
}

func TestParseTruncatedMidSegment(t *testing.T) {
	// A body that begins MSH but ends before MSH-2 is terminated is mid-segment
	// truncation and must surface io.ErrUnexpectedEOF wrapped in a *ParseError.
	for _, body := range []string{"MSH", "MSH|", "MSH|^~\\"} {
		_, err := Parse([]byte(body))
		var pe *ParseError
		if !errors.As(err, &pe) {
			t.Fatalf("Parse(%q) error = %v, want *ParseError", body, err)
		}
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("Parse(%q) error = %v, want wrapped io.ErrUnexpectedEOF", body, err)
		}
	}
}

func TestParseHeaderSegmentDelimiterFields(t *testing.T) {
	// MSH, BHS, and FHS share the delimiter quirk: field 1 is the field separator
	// itself and field 2 is the encoding characters, so every later field must
	// resolve at its HL7 position rather than being shifted by one.
	cases := []struct {
		name string
		raw  string
		id   string
	}{
		{"MSH", canonicalORM, "MSH"},
		{"BHS", canonicalBatch, "BHS"},
		{"FHS", canonicalFile, "FHS"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			segs, err := splitSegments([]byte(tc.raw), DefaultEncoding())
			if err != nil {
				t.Fatalf("splitSegments error = %v", err)
			}
			seg := segs[0]
			if seg.ID() != tc.id {
				t.Fatalf("leading segment ID = %q, want %q", seg.ID(), tc.id)
			}
			// Field 1 is the field separator verbatim; field 2 is the encoding
			// characters verbatim — identical to the MSH quirk.
			if got := seg.field(1).raw(); got != "|" {
				t.Errorf("%s-1 = %q, want %q (the field separator)", tc.id, got, "|")
			}
			if got := seg.field(2).raw(); got != "^~\\&" {
				t.Errorf("%s-2 = %q, want %q (the encoding characters)", tc.id, got, "^~\\&")
			}
			// Field 3 onward resolve at their HL7 positions, not shifted by one.
			if got := seg.field(3).raw(); got != "REGADT" && got != "RADIS" {
				t.Errorf("%s-3 = %q, want the sending application at its HL7 position", tc.id, got)
			}
			if got := seg.field(4).raw(); got != "HOSP" {
				t.Errorf("%s-4 = %q, want %q", tc.id, got, "HOSP")
			}
		})
	}
}

func TestParseBHSHeaderFieldsAtHL7Positions(t *testing.T) {
	// A BHS parsed as a batch header must address its fields by HL7 position:
	// BHS-3 is the sending application, never the encoding characters at Fields[2].
	batch, err := ParseBatch([]byte(canonicalBatch))
	if err != nil {
		t.Fatalf("ParseBatch error = %v", err)
	}
	if batch.Header == nil {
		t.Fatal("batch has no BHS header")
	}
	h := *batch.Header
	wantByPos := map[int]string{
		1:  "|",
		2:  "^~\\&",
		3:  "REGADT",
		4:  "HOSP",
		5:  "EMR",
		6:  "HOSP",
		7:  "202605310900",
		11: "BATCH0001",
	}
	for pos, want := range wantByPos {
		if got := h.field(pos).raw(); got != want {
			t.Errorf("BHS-%d = %q, want %q", pos, got, want)
		}
	}
}

func TestParseFHSHeaderFieldsAtHL7Positions(t *testing.T) {
	// The same HL7-positional addressing must hold for an FHS file header.
	file, err := ParseFile([]byte(canonicalFile))
	if err != nil {
		t.Fatalf("ParseFile error = %v", err)
	}
	if file.Header == nil {
		t.Fatal("file has no FHS header")
	}
	h := *file.Header
	wantByPos := map[int]string{
		1:  "|",
		2:  "^~\\&",
		3:  "REGADT",
		4:  "HOSP",
		5:  "EMR",
		6:  "HOSP",
		7:  "202605310930",
		11: "FILE0001.hl7",
	}
	for pos, want := range wantByPos {
		if got := h.field(pos).raw(); got != want {
			t.Errorf("FHS-%d = %q, want %q", pos, got, want)
		}
	}
}

func TestGetAccessor1Based(t *testing.T) {
	msg, err := Parse([]byte(canonicalORM))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}

	tests := []struct {
		key  string
		want string
	}{
		{"PID-5-1-1", "EVERYWOMAN"},
		{"PID-5-1-2", "EVE"},
		{"PID-3-1-1", "555-44-4444"},
		{"PID-8", "F"},
		{"ORC-1", "NW"},
		{"OBR-4-1-1", "36643-5"},
		{"OBR-4-1-2", "CHEST XRAY"},
		{"MSH-9-1-1", "ORM"},
		{"MSH-9-1-2", "O01"},
		{"MSH-10", "MSG00001"},
		{"PID.5.1.1", "EVERYWOMAN"}, // dotted form
		{"PID-99", ""},              // absent field -> "" no error
		{"NTE-1", ""},               // absent segment -> "" no error
	}
	for _, tc := range tests {
		got, err := msg.Get(tc.key)
		if err != nil {
			t.Errorf("Get(%q) error = %v", tc.key, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Get(%q) = %q, want %q", tc.key, got, tc.want)
		}
	}
}
