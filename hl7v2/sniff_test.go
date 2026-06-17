package hl7v2

import (
	"bytes"
	"testing"
)

// A non-PHI fixture: synthetic identifiers only, no real patient data.
const (
	sniffMsg1 = "MSH|^~\\&|SEND|FAC|RECV|FAC|20240101000000||ADT^A01|MSG00001|P|2.5.1\rEVN|A01|20240101000000\rPID|1||PID001\r"
	sniffMsg2 = "MSH|^~\\&|SEND|FAC|RECV|FAC|20240101000001||ADT^A02|MSG00002|P|2.5.1\rPID|1||PID002\r"
	sniffMsg3 = "MSH|^~\\&|SEND|FAC|RECV|FAC|20240101000002||ADT^A03|MSG00003|P|2.5.1\rPID|1||PID003\r"
)

func TestIsHL7(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"single message", sniffMsg1, true},
		{"single message with leading spaces", "   " + sniffMsg1, true},
		{"two messages is a batch not a lone message", sniffMsg1 + sniffMsg2, false},
		{"BHS-wrapped is not a lone message", "BHS|^~\\&\r" + sniffMsg1 + "BTS|1\r", false},
		{"not MSH-led", "PID|1||PID001\r", false},
		{"too short", "MSH", false},
		{"empty", "", false},
		{"non-standard field separator", "MSH#^~\\&#SEND\r", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsHL7([]byte(tc.in)); got != tc.want {
				t.Fatalf("IsHL7(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsBatch(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"BHS-wrapped", "BHS|^~\\&\r" + sniffMsg1 + "BTS|1\r", true},
		{"two bare messages is a header-less batch", sniffMsg1 + sniffMsg2, true},
		{"single message is not a batch", sniffMsg1, false},
		{"FHS-led is a file not a batch", "FHS|^~\\&\r" + sniffMsg1 + sniffMsg2 + "FTS|1\r", false},
		{"empty", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsBatch([]byte(tc.in)); got != tc.want {
				t.Fatalf("IsBatch(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsFile(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"FHS-led", "FHS|^~\\&\r" + sniffMsg1 + "FTS|1\r", true},
		{"a batch is also a file", sniffMsg1 + sniffMsg2, true},
		{"BHS-wrapped batch is a file", "BHS|^~\\&\r" + sniffMsg1 + "BTS|1\r", true},
		{"single message is not a file", sniffMsg1, false},
		{"empty", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsFile([]byte(tc.in)); got != tc.want {
				t.Fatalf("IsFile(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestSplitFileBareBatch(t *testing.T) {
	in := sniffMsg1 + sniffMsg2 + sniffMsg3
	got := SplitFile([]byte(in))
	if len(got) != 3 {
		t.Fatalf("SplitFile bare batch yielded %d messages, want 3", len(got))
	}
	// Each element must parse and carry the right control ID, proving the MSH
	// boundary split kept segments with their own message.
	wantIDs := []string{"MSG00001", "MSG00002", "MSG00003"}
	for i, raw := range got {
		if !bytes.HasSuffix(raw, []byte("\r")) {
			t.Errorf("message %d does not end with \\r: %q", i, raw)
		}
		m, err := Parse(raw)
		if err != nil {
			t.Fatalf("Parse(message %d) error = %v", i, err)
		}
		id, _ := m.Get("MSH-10")
		if id != wantIDs[i] {
			t.Errorf("message %d MSH-10 = %q, want %q", i, id, wantIDs[i])
		}
	}
}

func TestSplitFileWrappedFileStripsFraming(t *testing.T) {
	// A full FHS/BHS ... BTS/FTS wrapper around three messages. SplitFile must
	// throw away all four framing segments and return exactly the three messages.
	in := "FHS|^~\\&|SEND|FAC\r" +
		"BHS|^~\\&|SEND|FAC\r" +
		sniffMsg1 + sniffMsg2 + sniffMsg3 +
		"BTS|3\r" +
		"FTS|1\r"
	got := SplitFile([]byte(in))
	if len(got) != 3 {
		t.Fatalf("SplitFile wrapped file yielded %d messages, want 3", len(got))
	}
	for i, raw := range got {
		for _, framing := range []string{"FHS", "BHS", "BTS", "FTS"} {
			if bytes.Contains(raw, []byte(framing)) {
				t.Errorf("message %d still contains framing segment %s: %q", i, framing, raw)
			}
		}
		if _, err := Parse(raw); err != nil {
			t.Fatalf("Parse(message %d) error = %v", i, err)
		}
	}
}

func TestSplitFileSingleMessage(t *testing.T) {
	got := SplitFile([]byte(sniffMsg1))
	if len(got) != 1 {
		t.Fatalf("SplitFile single message yielded %d, want 1", len(got))
	}
}

func TestSplitFileNoMSH(t *testing.T) {
	// A body with framing but no MSH yields no messages, and a stray pre-MSH
	// segment is dropped rather than attached to a phantom message.
	if got := SplitFile([]byte("BHS|^~\\&\rBTS|0\r")); got != nil {
		t.Fatalf("SplitFile with no MSH = %v, want nil", got)
	}
	if got := SplitFile([]byte("PID|1||orphan\r")); got != nil {
		t.Fatalf("SplitFile with a pre-MSH segment = %v, want nil", got)
	}
}

func TestSplitFilePreservesTrailingWhitespace(t *testing.T) {
	// HL7 leaf values are whitespace-significant: an OBX value with intentional
	// trailing spaces must survive SplitFile byte-for-byte. SplitFile must only
	// drop blank lines and framing, never trim the content of a real segment.
	const obxValue = "value with trailing spaces  "
	in := "MSH|^~\\&|SEND|FAC|RECV|FAC|20240101||ORU^R01|WS1|P|2.5.1\r" +
		"OBX|1|ST|CODE^Name||" + obxValue + "\r"
	got := SplitFile([]byte(in))
	if len(got) != 1 {
		t.Fatalf("SplitFile yielded %d, want 1", len(got))
	}
	// The message must round-trip byte-for-byte through SplitFile (the input is a
	// single \r-terminated message, so the one returned element equals the input).
	if !bytes.Equal(got[0], []byte(in)) {
		t.Fatalf("SplitFile mutated segment bytes:\n got = %q\nwant = %q", got[0], in)
	}
	// And the trailing spaces survive parsing into the OBX-5 value.
	m, err := Parse(got[0])
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	if v, _ := m.Get("OBX-5"); v != obxValue {
		t.Errorf("OBX-5 = %q, want %q (trailing whitespace lost)", v, obxValue)
	}
}

func TestSniffHeaderlessBatchLFAndCRLF(t *testing.T) {
	// A header-less batch terminated only with \n (and one with \r\n) must be
	// recognised as a batch, not misreported as a single message: Parse/SplitFile
	// accept \n and \r\n, so IsHL7's second-MSH check must use the same terminator
	// handling. Otherwise a caller routing on IsHL7 would call Parse and flatten
	// multiple messages into one.
	lf := "MSH|^~\\&|SEND|FAC|RECV|FAC|20240101||ADT^A01|LF1|P|2.5.1\n" +
		"PID|1||PIDLF1\n" +
		"MSH|^~\\&|SEND|FAC|RECV|FAC|20240101||ADT^A02|LF2|P|2.5.1\n" +
		"PID|1||PIDLF2\n"
	crlf := "MSH|^~\\&|SEND|FAC|RECV|FAC|20240101||ADT^A01|CRLF1|P|2.5.1\r\n" +
		"PID|1||PIDCRLF1\r\n" +
		"MSH|^~\\&|SEND|FAC|RECV|FAC|20240101||ADT^A02|CRLF2|P|2.5.1\r\n" +
		"PID|1||PIDCRLF2\r\n"

	for _, tc := range []struct {
		name string
		in   string
	}{{"lf", lf}, {"crlf", crlf}} {
		t.Run(tc.name, func(t *testing.T) {
			if IsHL7([]byte(tc.in)) {
				t.Errorf("IsHL7 = true for a 2-message %s batch, want false", tc.name)
			}
			if !IsBatch([]byte(tc.in)) {
				t.Errorf("IsBatch = false for a 2-message %s batch, want true", tc.name)
			}
			if got := SplitFile([]byte(tc.in)); len(got) != 2 {
				t.Errorf("SplitFile yielded %d, want 2", len(got))
			}
		})
	}
}

func TestSplitFileNormalisesTerminators(t *testing.T) {
	// Input uses \n and \r\n; output must normalise to \r, matching python-hl7.
	in := "MSH|^~\\&|SEND|FAC|RECV|FAC|20240101||ADT^A01|MSGN1|P|2.5.1\nPID|1||PIDN1\r\n"
	got := SplitFile([]byte(in))
	if len(got) != 1 {
		t.Fatalf("SplitFile yielded %d, want 1", len(got))
	}
	if bytes.ContainsAny(got[0], "\n") {
		t.Errorf("output retained a \\n terminator: %q", got[0])
	}
	if !bytes.HasSuffix(got[0], []byte("\r")) {
		t.Errorf("output not \\r-terminated: %q", got[0])
	}
}
