package dicomweb

import (
	"net/url"
	"testing"
)

// FuzzParseQueryRequest drives the QIDO-RS query parser with arbitrary path segments and
// query strings. The parser is the trust boundary for a hostile QIDO-RS request: it must
// never panic, and a successful parse must keep its bounds (non-negative limit/offset, no
// zero-tag match key). The PHI-redaction contract is verified separately and precisely by
// FuzzSafeAttributeName, since scanning a composed free-form message for input echoes is
// inherently false-positive-prone against the message's own fixed prose.
func FuzzParseQueryRequest(f *testing.F) {
	seeds := []struct {
		path  string
		query string
	}{
		{"studies", "PatientID=12345"},
		{"studies", "StudyDate=20200101-20201231&limit=50&offset=10"},
		{"studies/1.2.3/series", "Modality=CT"},
		{"series", "includefield=all"},
		{"instances", "includefield=BodyPartExamined,00080060&fuzzymatching=true"},
		{"studies", "00100020=*"},
		{"studies", "PatientName=Doe*&limit=-1"},
		{"frames", "x=y"},
		{"studies/not-a-uid/series", ""},
		{"studies", "Doe^Jane=smuggled"},
		{"studies", "limit=abc"},
		{"studies", "%ZZ=broken"},
		{"studies", "includefield=0000 "},
		{"studies", "00000000=x"},
	}
	for _, s := range seeds {
		f.Add(s.path, s.query)
	}

	f.Fuzz(func(t *testing.T, path, rawQuery string) {
		segs := splitPath(path)
		query, err := url.ParseQuery(rawQuery)
		if err != nil {
			// An unparseable query string is the HTTP layer's concern, not the parser's; the
			// fuzz target only exercises parseQueryRequest's handling of parsed values.
			return
		}

		q, perr := parseQueryRequest(segs, query)
		if perr != nil {
			return
		}

		if q.Limit < 0 || q.Offset < 0 {
			t.Fatalf("parsed a negative bound: limit=%d offset=%d", q.Limit, q.Offset)
		}
		for _, mk := range q.Match {
			if mk.Tag == 0 || mk.Tag.IsGroupLength() {
				t.Fatalf("parsed an unqueryable match-key tag: %+v", mk)
			}
		}
		for _, tag := range q.IncludeFields {
			if tag == 0 || tag.IsGroupLength() {
				t.Fatalf("parsed an unqueryable includefield tag: %v", tag)
			}
		}
	})
}

// FuzzSafeAttributeName proves the PHI-redaction contract (PRD §9.1) at its source: the
// only function through which an attribute reference reaches an error message. The
// contract is exact — safeAttributeName returns the input verbatim only when it is a
// structural token (a plain alphanumeric keyword or hex tag), and otherwise returns the
// fixed "(redacted)" placeholder. Any other output would mean an attacker-controlled,
// non-structural reference reached the message.
func FuzzSafeAttributeName(f *testing.F) {
	for _, seed := range []string{"PatientID", "00080018", "", " ", "Doe^Jane", "a ", "x,y", "../etc", "0000 "} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, ref string) {
		got := safeAttributeName(ref)
		if isStructuralAttributeName(ref) {
			if got != ref {
				t.Fatalf("safeAttributeName(%q) = %q, want the structural input verbatim", ref, got)
			}
			return
		}
		if got != "(redacted)" {
			t.Fatalf("safeAttributeName(%q) = %q, want \"(redacted)\" for a non-structural reference", ref, got)
		}
	})
}
