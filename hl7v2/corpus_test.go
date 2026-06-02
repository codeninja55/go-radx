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
// knows which entry point will eventually parse it. The single-message
// fixtures parse with Parse today; the batch/file containers are parsed by
// ParseBatch/ParseFile in a later increment, so the harness only loads their
// raw bytes for now.
type corpusKind int

const (
	kindMessage corpusKind = iota // a single MSH-led message (Parse)
	kindBatch                     // a BHS/BTS batch (ParseBatch, later increment)
	kindFile                      // an FHS/FTS file (ParseFile, later increment)
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
// serves every fixture kind including the batch/file containers whose parsers
// land in a later increment.
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

// TestCorpusHarness is the Increment 0 gate: every fixture is present and
// loadable, every single-message fixture parses and round-trips byte-exact
// through the existing Parse/MarshalText, and every container fixture is
// present with a header/trailer the later batch/file parser will consume. It
// proves the corpus is wired before any increment depends on it.
func TestCorpusHarness(t *testing.T) {
	for _, f := range corpus {
		t.Run(f.Name, func(t *testing.T) {
			raw := corpusRaw(t, f.Name)
			if len(raw) == 0 {
				t.Fatalf("fixture %q is empty", f.File)
			}

			switch f.Kind {
			case kindMessage:
				msg := corpusMessage(t, f.Name)
				out, err := msg.MarshalText()
				if err != nil {
					t.Fatalf("MarshalText(%q): %v", f.File, err)
				}
				if string(out) != string(raw) {
					t.Fatalf("round-trip mismatch for %q:\n got = %q\nwant = %q", f.File, out, raw)
				}
			case kindBatch:
				assertLeadingSegment(t, raw, "BHS")
				assertContainsSegment(t, raw, "BTS")
			case kindFile:
				assertLeadingSegment(t, raw, "FHS")
				assertContainsSegment(t, raw, "FTS")
			}
		})
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
