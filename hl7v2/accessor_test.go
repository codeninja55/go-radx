package hl7v2

import (
	"errors"
	"testing"
)

func TestParseAccessorNumeric(t *testing.T) {
	tests := []struct {
		key  string
		want Accessor
	}{
		{"PID", Accessor{Segment: "PID", SegmentNum: 1}},
		{"PID-5", Accessor{Segment: "PID", SegmentNum: 1, Field: 5}},
		{"PID-5-1-2", Accessor{Segment: "PID", SegmentNum: 1, Field: 5, Repetition: 1, Component: 2}},
		{"PID-5-1-2-3", Accessor{Segment: "PID", SegmentNum: 1, Field: 5, Repetition: 1, Component: 2, Subcomponent: 3}},
		{"PID.5.1.2", Accessor{Segment: "PID", SegmentNum: 1, Field: 5, Repetition: 1, Component: 2}},
		{"MSH-9-1-1", Accessor{Segment: "MSH", SegmentNum: 1, Field: 9, Repetition: 1, Component: 1}},
		{"OBR2-4", Accessor{Segment: "OBR", SegmentNum: 2, Field: 4}},
	}
	for _, tc := range tests {
		got, err := ParseAccessor(tc.key)
		if err != nil {
			t.Errorf("ParseAccessor(%q) error = %v", tc.key, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseAccessor(%q) = %+v, want %+v", tc.key, got, tc.want)
		}
	}
}

func TestParseAccessorPrefixed(t *testing.T) {
	tests := []struct {
		key  string
		want Accessor
	}{
		{"PID.F5", Accessor{Segment: "PID", SegmentNum: 1, Field: 5}},
		{"PID.F5.R1.C2", Accessor{Segment: "PID", SegmentNum: 1, Field: 5, Repetition: 1, Component: 2}},
		{"PID.F5.R1.C2.S3", Accessor{Segment: "PID", SegmentNum: 1, Field: 5, Repetition: 1, Component: 2, Subcomponent: 3}},
		{"OBR2.F4.R1.C1", Accessor{Segment: "OBR", SegmentNum: 2, Field: 4, Repetition: 1, Component: 1}},
	}
	for _, tc := range tests {
		got, err := ParseAccessor(tc.key)
		if err != nil {
			t.Errorf("ParseAccessor(%q) error = %v", tc.key, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseAccessor(%q) = %+v, want %+v", tc.key, got, tc.want)
		}
	}
}

func TestParseAccessorErrors(t *testing.T) {
	for _, key := range []string{
		"",             // empty
		"PI",           // segment ID not three chars
		"PIDD-5",       // segment ID too long before the index
		"PID-0",        // non-positive field index
		"PID--5",       // empty index component
		"PID-x",        // non-numeric index
		"PID.F5.X1",    // unknown prefix
		"PID.R1",       // repetition before field
		"PID0-5",       // non-positive segment instance
		"PID.F5.C2.R1", // prefixes out of order
	} {
		_, err := ParseAccessor(key)
		var ae *AccessorError
		if !errors.As(err, &ae) {
			t.Errorf("ParseAccessor(%q) error = %v, want *AccessorError", key, err)
		}
	}
}

func TestAccessorStringCanonical(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"PID", "PID"},
		{"PID-5", "PID-5"},
		{"PID-5-1-2", "PID-5-1-2"},
		{"PID.F5.R1.C2", "PID-5-1-2"},
		{"PID.F5.R1.C2.S3", "PID-5-1-2-3"},
		{"OBR2-4", "OBR2-4"},
		{"PID2", "PID2"},
	}
	for _, tc := range tests {
		a, err := ParseAccessor(tc.key)
		if err != nil {
			t.Fatalf("ParseAccessor(%q) error = %v", tc.key, err)
		}
		if got := a.String(); got != tc.want {
			t.Errorf("ParseAccessor(%q).String() = %q, want %q", tc.key, got, tc.want)
		}
	}
}

// accessorORM carries a second PID, an escaped value, and an explicit-null
// field so the resolution edge cases can all be exercised against one message.
const accessorORM = "MSH|^~\\&|RADIS|HOSP|PACS|HOSP|202605311230||ORM^O01|MSG00001|P|2.4\r" +
	"PID|||555-44-4444^^^HOSP^MR||SMITH \\T\\ JONES^EVE^E^^^^L||19620320|\"\"\r" +
	"PID|||999-88-7777^^^HOSP^MR||EVERYWOMAN^EVE\r" +
	"ORC|NW|PLACER123|FILLER456||||||202605311230\r" +
	"OBR|1|PLACER123|FILLER456|36643-5^CHEST XRAY^LN|||202605311231\r"

func TestGetUnescapeOnRead(t *testing.T) {
	msg, err := Parse([]byte(accessorORM))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}

	tests := []struct {
		key  string
		want string
	}{
		{"PID-5-1-1", "SMITH & JONES"}, // \T\ decodes to the subcomponent delimiter
		{"PID-5-1-2", "EVE"},
		{"PID-3-1-1", "555-44-4444"},
		{"PID2-3-1-1", "999-88-7777"}, // second PID instance
		{"PID2-5-1-1", "EVERYWOMAN"},
		{"OBR-4-1-2", "CHEST XRAY"},
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

func TestGetMSHDelimitersVerbatim(t *testing.T) {
	msg, err := Parse([]byte(accessorORM))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	if got, _ := msg.Get("MSH-1"); got != "|" {
		t.Errorf("Get(MSH-1) = %q, want %q (verbatim field separator)", got, "|")
	}
	if got, _ := msg.Get("MSH-2"); got != "^~\\&" {
		t.Errorf("Get(MSH-2) = %q, want %q (verbatim encoding characters, never unescaped)", got, "^~\\&")
	}
}

func TestGetExplicitNullVersusAbsence(t *testing.T) {
	msg, err := Parse([]byte(accessorORM))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	// PID-8 is the literal HL7 null ("") in the first PID: present-but-empty.
	if got, err := msg.Get("PID-8"); err != nil || got != `""` {
		t.Errorf("Get(PID-8) = %q, %v, want \"\\\"\\\"\", nil (explicit null preserved)", got, err)
	}
	// PID-99 is simply absent: empty string, no error.
	if got, err := msg.Get("PID-99"); err != nil || got != "" {
		t.Errorf("Get(PID-99) = %q, %v, want \"\", nil (absence)", got, err)
	}
	// An absent segment instance is absence, not an error.
	if got, err := msg.Get("PID3-5"); err != nil || got != "" {
		t.Errorf("Get(PID3-5) = %q, %v, want \"\", nil (absent 3rd PID)", got, err)
	}
}

func TestGetFutureProofedResolution(t *testing.T) {
	msg, err := Parse([]byte(canonicalORM))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	tests := []struct {
		key  string
		want string
	}{
		// Path shallower than the tree descends the first child to a leaf.
		{"PID-5", "EVERYWOMAN"},
		{"PID-5-1", "EVERYWOMAN"},
		{"OBR-4", "36643-5"},
		// Path deeper than the tree where every extra step is index 1 reaches the leaf.
		{"PID-8-1-1-1", "F"},
		{"ORC-1-1-1-1", "NW"},
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

func TestGetPathPastLeaf(t *testing.T) {
	msg, err := Parse([]byte(canonicalORM))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	// PID-8 is the single-valued "F"; asking for its second subcomponent runs
	// past the leaf and is a malformed request, not an absent optional.
	for _, key := range []string{"PID-8-1-1-2", "ORC-1-1-2"} {
		_, err := msg.Get(key)
		var ae *AccessorError
		if !errors.As(err, &ae) {
			t.Errorf("Get(%q) error = %v, want *AccessorError", key, err)
		}
	}
}

func TestSetRoundTripEscapeOnWrite(t *testing.T) {
	msg, err := Parse([]byte(canonicalORM))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}

	// A value carrying a component delimiter must be escaped on write so it does
	// not forge structure, and read back decoded.
	if err := msg.Set("PID-5-1-1", "DOE^SMITH"); err != nil {
		t.Fatalf("Set(PID-5-1-1) error = %v", err)
	}
	if got, _ := msg.Get("PID-5-1-1"); got != "DOE^SMITH" {
		t.Errorf("Get after Set = %q, want %q", got, "DOE^SMITH")
	}
	// The serialized form must carry the escape sequence, not a raw delimiter.
	out := msg.String()
	if !contains(out, `DOE\S\SMITH`) {
		t.Errorf("serialized message does not carry escaped value: %q", out)
	}
}

func TestSetGrowsTree(t *testing.T) {
	msg, err := Parse([]byte(canonicalORM))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	// PID-13 does not exist on the source PID; Set must grow fields to reach it.
	if err := msg.Set("PID-13-1-2", "555-1234"); err != nil {
		t.Fatalf("Set(PID-13-1-2) error = %v", err)
	}
	if got, _ := msg.Get("PID-13-1-2"); got != "555-1234" {
		t.Errorf("Get(PID-13-1-2) after Set = %q, want %q", got, "555-1234")
	}
	// Growing intermediate positions must leave them empty, not corrupted.
	if got, _ := msg.Get("PID-13-1-1"); got != "" {
		t.Errorf("Get(PID-13-1-1) after Set = %q, want empty", got)
	}
}

func TestSetNeverInventsSegment(t *testing.T) {
	msg, err := Parse([]byte(canonicalORM))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	// NTE is absent; Set must refuse rather than synthesize a segment.
	err = msg.Set("NTE-3", "a comment")
	var ae *AccessorError
	if !errors.As(err, &ae) {
		t.Fatalf("Set(NTE-3) error = %v, want *AccessorError", err)
	}
	if _, ok := msg.Segment("NTE"); ok {
		t.Errorf("Set invented an NTE segment")
	}
}

func TestSetSecondInstance(t *testing.T) {
	msg, err := Parse([]byte(accessorORM))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	if err := msg.Set("PID2-8", "M"); err != nil {
		t.Fatalf("Set(PID2-8) error = %v", err)
	}
	if got, _ := msg.Get("PID2-8"); got != "M" {
		t.Errorf("Get(PID2-8) after Set = %q, want %q", got, "M")
	}
	// The first PID instance must be untouched.
	if got, _ := msg.Get("PID-8"); got != `""` {
		t.Errorf("Get(PID-8) = %q, want explicit null untouched", got)
	}
}

func TestSetMSHDelimitersRejected(t *testing.T) {
	msg, err := Parse([]byte(canonicalORM))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	// MSH-1 and MSH-2 define the encoding; assigning them through Set would
	// desynchronize the message from its delimiters.
	for _, key := range []string{"MSH-1", "MSH-2"} {
		err := msg.Set(key, "x")
		var ae *AccessorError
		if !errors.As(err, &ae) {
			t.Errorf("Set(%q) error = %v, want *AccessorError", key, err)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
