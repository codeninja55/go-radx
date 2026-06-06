package hl7v2

import (
	"os"
	"path/filepath"
	"testing"
)

// The M5 HL7 v2 test corpus lives under ../testdata/hl7v2. Every fixture is a
// synthetic message with fictitious patient data (see that directory's
// README.md and LICENSE-python-hl7.txt for provenance and attribution). This
// file is the corpus harness the later M5 increments load their inputs from; it
// holds no production code.

// corpusDir is the corpus location relative to the hl7v2 package directory,
// following the same convention as the dicom/dimse testdata loaders.
const corpusDir = "../testdata/hl7v2"

// corpusKind classifies a fixture by its top-level container so the harness
// knows which entry point parses it: single-message fixtures parse with Parse,
// batch containers with ParseBatch, file containers with ParseFile. Every kind
// round-trips byte-exactly through MarshalText.
type corpusKind int

const (
	kindMessage corpusKind = iota // a single MSH-led message (Parse)
	kindBatch                     // a BHS/BTS batch (ParseBatch)
	kindFile                      // an FHS/FTS file (ParseFile)
)

// corpusFixture describes one vendored fixture: its filename, container kind,
// and what message shape or edge case it exercises for the increments that
// consume it.
type corpusFixture struct {
	Name      string // logical key, e.g. "adt-a01"
	File      string // filename under corpusDir
	Kind      corpusKind
	Exercises string // what this fixture is for
}

// corpus is the registry of every fixture in the corpus. The later increments
// look fixtures up by name (corpusMessage("oru-r01"), corpusRaw("batch")); the
// harness self-test below iterates the whole registry.
var corpus = []corpusFixture{
	{"adt-a01", "adt-a01.hl7", kindMessage, "ADT^A01 admission: MSH/EVN/PID/PV1, attending doctor, address (Inc 2,3)"},
	{"adt-a03", "adt-a03.hl7", kindMessage, "ADT^A03 discharge: trigger-event scope, discharge datetime (Inc 3)"},
	{"oru-r01", "oru-r01.hl7", kindMessage, "ORU^R01 results: two OBR+OBX groups, NM/TX values, repeated OBX-5 (Inc 2,4)"},
	{"orm-o01", "orm-o01.hl7", kindMessage, "ORM^O01 order: ORC+OBR group (Inc 4 grouping template; existing ORM)"},
	{"omg-o19", "omg-o19.hl7", kindMessage, "OMG^O19 imaging order: ORM lens accepts both ORM and OMG (open question 5)"},
	{"ack", "ack.hl7", kindMessage, "ACK with MSA AA acknowledging the ORU control ID (Inc 7)"},
	{"nonstandard-delim", "nonstandard-delim.hl7", kindMessage, "Non-standard delimiters (#@+$%): encoding derivation round-trip (Inc 1,6)"},
	{"escaped", "escaped.hl7", kindMessage, "Chapter 2 §2.10 escape sequences in field values (Inc 5,6)"},
	{"batch", "batch.hl7", kindBatch, "BHS/BTS batch of two ADT^A04 messages (Inc 11)"},
	{"file", "file.hl7", kindFile, "FHS/FTS file containing one batch (Inc 11)"},
}

// corpusByName looks a fixture up in the registry, failing the test if the name
// is unknown so a typo in a consuming test surfaces immediately.
func corpusByName(t *testing.T, name string) corpusFixture {
	t.Helper()
	for _, f := range corpus {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("corpus: no fixture named %q", name)
	return corpusFixture{}
}

// corpusRaw loads a fixture's exact bytes from disk. It does not parse, so it
// serves every fixture kind including the batch/file containers, which the
// harness then parses through ParseAny.
func corpusRaw(t *testing.T, name string) []byte {
	t.Helper()
	f := corpusByName(t, name)
	b, err := os.ReadFile(filepath.Join(corpusDir, f.File))
	if err != nil {
		t.Fatalf("corpus: read %q: %v", f.File, err)
	}
	return b
}

// corpusMessage loads and parses a single-message fixture, failing the test if
// the fixture is a multi-message container (which Parse does not accept) or if
// parsing fails.
func corpusMessage(t *testing.T, name string) *Message {
	t.Helper()
	f := corpusByName(t, name)
	if f.Kind != kindMessage {
		t.Fatalf("corpus: fixture %q is a container, not a single message; use corpusRaw", name)
	}
	msg, err := Parse(corpusRaw(t, name))
	if err != nil {
		t.Fatalf("corpus: parse %q: %v", f.File, err)
	}
	return msg
}

// TestCorpusHarness is the corpus gate: every fixture is present, loadable, and
// round-trips byte-exact through the parser for its kind. A single-message
// fixture parses with Parse; a batch with ParseBatch; a file with ParseFile.
// ParseAny must also dispatch to the same container and round-trip it.
func TestCorpusHarness(t *testing.T) {
	for _, f := range corpus {
		t.Run(f.Name, func(t *testing.T) {
			raw := corpusRaw(t, f.Name)
			if len(raw) == 0 {
				t.Fatalf("fixture %q is empty", f.File)
			}

			switch f.Kind {
			case kindMessage:
				assertLeadingSegment(t, raw, "MSH")
				msg := corpusMessage(t, f.Name)
				assertContainerRoundTrip(t, f.File, msg, raw)
			case kindBatch:
				assertLeadingSegment(t, raw, "BHS")
				assertContainsSegment(t, raw, "BTS")
				batch, err := ParseBatch(raw)
				if err != nil {
					t.Fatalf("ParseBatch(%q): %v", f.File, err)
				}
				assertContainerRoundTrip(t, f.File, batch, raw)
			case kindFile:
				assertLeadingSegment(t, raw, "FHS")
				assertContainsSegment(t, raw, "FTS")
				file, err := ParseFile(raw)
				if err != nil {
					t.Fatalf("ParseFile(%q): %v", f.File, err)
				}
				assertContainerRoundTrip(t, f.File, file, raw)
			}

			// ParseAny dispatches on the leading segment and the result round-trips
			// byte-exactly regardless of kind.
			c, err := ParseAny(raw)
			if err != nil {
				t.Fatalf("ParseAny(%q): %v", f.File, err)
			}
			assertContainerRoundTrip(t, f.File, c, raw)
		})
	}
}

// assertContainerRoundTrip fails unless c renders back to raw byte-for-byte.
func assertContainerRoundTrip(t *testing.T, file string, c Container, raw []byte) {
	t.Helper()
	out, err := c.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText(%q): %v", file, err)
	}
	if string(out) != string(raw) {
		t.Fatalf("round-trip mismatch for %q:\n got = %q\nwant = %q", file, out, raw)
	}
}

// TestCorpusEscapeDecode asserts the escaped fixture decodes its Chapter 2 §2.10
// escape sequences on read while still round-tripping byte-exact when rendered.
// The escaped value carries separator escapes (\S\ \F\ \E\ \T\) and a hex CR
// (\X0D\); Get unescapes them against the message's derived encoding.
func TestCorpusEscapeDecode(t *testing.T) {
	msg := corpusMessage(t, "escaped")

	// The escaped fixture round-trips byte-exact even though Get unescapes on read
	// (Unescape is a read-side projection that never mutates the tree).
	out, err := msg.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	if string(out) != string(corpusRaw(t, "escaped")) {
		t.Fatalf("escaped fixture did not round-trip byte-exact")
	}

	// PID-5.1 family name carries \T\ (the subcomponent separator '&').
	family, err := msg.Get("PID-5-1-1")
	if err != nil {
		t.Fatalf("Get(PID-5-1-1): %v", err)
	}
	if family != "O&BRIEN-SMITH" {
		t.Errorf("Get(PID-5-1-1) = %q, want %q (\\T\\ decoded to '&')", family, "O&BRIEN-SMITH")
	}

	// OBX-5 carries \S\ (component '^'), \F\ (field '|'), \E\ (escape '\'), and a
	// hex CR \X0D\. Every sequence must decode to its literal byte.
	value, err := msg.Get("OBX-5")
	if err != nil {
		t.Fatalf("Get(OBX-5): %v", err)
	}
	want := "Reads 5^6 mg per 100|unit\\ then \rdone"
	if value != want {
		t.Errorf("Get(OBX-5) = %q, want %q", value, want)
	}
}

// TestCorpusRoundTripByteExact is the package-level round-trip invariant
// (PRD §11.1) applied to every parseable single-message fixture, including the
// non-standard-delimiter and escaped messages that are the most likely to
// regress a renderer.
func TestCorpusRoundTripByteExact(t *testing.T) {
	for _, f := range corpus {
		if f.Kind != kindMessage {
			continue
		}
		t.Run(f.Name, func(t *testing.T) {
			raw := corpusRaw(t, f.Name)
			msg, err := Parse(raw)
			if err != nil {
				t.Fatalf("Parse(%q): %v", f.File, err)
			}
			out, err := msg.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText(%q): %v", f.File, err)
			}
			if string(out) != string(raw) {
				t.Fatalf("round-trip mismatch for %q:\n got = %q\nwant = %q", f.File, out, raw)
			}
		})
	}
}

// assertLeadingSegment fails unless raw's first segment ID is want.
func assertLeadingSegment(t *testing.T, raw []byte, want string) {
	t.Helper()
	if len(raw) < len(want) || string(raw[:len(want)]) != want {
		t.Fatalf("leading segment = %.10q..., want %q", raw, want)
	}
}

// assertContainsSegment fails unless raw contains a segment whose ID is want
// (the ID at the start of a CR-delimited line).
func assertContainsSegment(t *testing.T, raw []byte, want string) {
	t.Helper()
	needle := "\r" + want
	if string(raw[:min(len(raw), len(want))]) == want {
		return
	}
	for i := 0; i+len(needle) <= len(raw); i++ {
		if string(raw[i:i+len(needle)]) == needle {
			return
		}
	}
	t.Fatalf("no %q segment found in container", want)
}
