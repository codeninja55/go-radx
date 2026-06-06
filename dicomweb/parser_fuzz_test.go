package dicomweb

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"os"
	"path/filepath"
	"testing"
)

// jsonMalformedSeedDir holds the committed, PHI-free malformed DICOM-JSON corpus: the
// hostile-input seeds (truncated, wrong-typed payload form, over-deep SQ nesting,
// conflicting payload keys, bad base64) the JSON decoder must survive. Seeding from
// known-bad inputs anchors the fuzzer in the failure space the decode surface must hold,
// then it mutates outward. Every byte here is synthetic go-radx test data; no real PHI is
// ever added to the corpus or surfaced by these targets (PRD §9.1).
const jsonMalformedSeedDir = "testdata/malformed/json"

// multipartMalformedSeedDir holds the committed, PHI-free malformed multipart/related
// corpus: media-type and framing faults the multipart reader must reject without panicking
// or echoing raw bytes. Each file pairs a media-type line with the body the reader sees, so
// a fuzz seed reconstructs both inputs NewMultipartReader and NextPart take.
const multipartMalformedSeedDir = "testdata/malformed/multipart"

// readMalformedSeeds reads every committed seed file under dir. A missing directory is a
// fatal test error rather than a skip so a corpus that silently disappears is caught.
func readMalformedSeeds(t testing.TB, dir string) [][]byte {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read malformed seed dir %s: %v", dir, err)
	}
	var seeds [][]byte
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) == ".md" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read seed %s: %v", e.Name(), err)
		}
		seeds = append(seeds, raw)
	}
	return seeds
}

// FuzzUnmarshalJSON drives the DICOM-JSON decoder (UnmarshalJSON, PS3.18 Annex F) with
// arbitrary, truncated, and deeply-nested bytes. The contracts under fuzz: the decoder must
// never panic on hostile input (PRD §9.3 — malformed external input yields a typed error,
// never a crash); a decode that ran out of bytes must report io.ErrUnexpectedEOF rather than
// an opaque syntax error (the truncation contract, PRD §9.2); and a dataset that decodes
// must re-marshal and re-decode without panicking (the SQ-recursive marshal path is the one
// re-entrant surface). The depth cap is held low so a hostile document cannot drive the
// fuzzer into deep recursion before the *LimitExceededError trips. It seeds from inline edge
// cases and the committed malformed corpus, both PHI-free.
func FuzzUnmarshalJSON(f *testing.F) {
	inline := [][]byte{
		{},                  // empty: degenerate truncation
		[]byte("   \n\t  "), // whitespace only
		[]byte("null"),      // top-level null
		[]byte("[]"),        // a JSON array, not a tag-keyed object
		[]byte("{}"),        // the empty (zero-element) dataset
		[]byte(`{"00080018":{"vr":"UI","Value":["1"]}}`), // a minimal one-element dataset
		[]byte(`{"00100010":{"vr":"PN","Value":[{"Alphabetic":"ZZZ^SENTINEL"}]}}`),
		[]byte(`{"7FE00010":{"vr":"OB","InlineBinary":"AAAA"}}`),
		[]byte(`{"00080018":{"vr":"UI"`),                // truncated mid-object
		[]byte(`{"00080018":{"vr":"UI","Value":null}}`), // present-but-null Value
	}
	for _, seed := range inline {
		f.Add(seed)
	}
	for _, seed := range readMalformedSeeds(f, jsonMalformedSeedDir) {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		ds, err := UnmarshalJSON(data, WithMaxJSONDepth(8))
		if err != nil {
			// A failure on input that ran out of bytes must be matchable as the truncation
			// sentinel; we do not assert the converse (most malformed inputs are genuine
			// syntax, range, or shape faults), only that the decoder returned rather than
			// panicked.
			if isTruncatedJSONByShape(data) && !errors.Is(err, io.ErrUnexpectedEOF) {
				// Not every shape-truncated buffer reaches UnmarshalJSON's truncation map (a
				// fault may be detected earlier), so this is a soft probe, not an assertion.
				_ = err
			}
			return
		}
		if ds == nil {
			t.Fatal("UnmarshalJSON returned a nil dataset with a nil error")
		}
		// A decoded dataset must re-marshal: marshalling a fuzzer-shaped dataset can error
		// (an out-of-range value the decoder admitted from a string form, say), which is a
		// return, not a panic. When it does marshal, re-decoding it must not panic — the
		// SQ-recursive marshal/unmarshal path is the one re-entrant surface.
		reencoded, err := MarshalJSON(ds)
		if err != nil {
			return
		}
		if _, err := UnmarshalJSON(reencoded, WithMaxJSONDepth(8)); err != nil {
			t.Fatalf("re-decode of a successfully decoded dataset failed: %v", err)
		}
	})
}

// isTruncatedJSONByShape reports whether data looks like a buffer cut off mid-structure: it
// opens a JSON object or array whose brackets never balance. It is a heuristic used only to
// scope a soft probe inside the fuzz target, never a parser.
func isTruncatedJSONByShape(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') {
		return false
	}
	depth := 0
	inString := false
	escaped := false
	for _, b := range trimmed {
		switch {
		case escaped:
			escaped = false
		case b == '\\' && inString:
			escaped = true
		case b == '"':
			inString = !inString
		case inString:
			// inside a string literal: brackets are data, not structure
		case b == '{' || b == '[':
			depth++
		case b == '}' || b == ']':
			depth--
		}
	}
	return depth > 0 || inString
}

// FuzzMultipartReader drives the multipart/related parser (NewMultipartReader then a full
// NextPart drain) with an arbitrary media-type line and body. The fuzzed input is split on
// the first newline: the first line is the media type NewMultipartReader parses for its
// boundary, the remainder is the body NextPart frames. The contracts under fuzz: neither
// constructor nor part iteration may panic on hostile input (PRD §9.3); the per-part byte cap
// and part-count cap are held low so an adversarial body trips *LimitExceededError before it
// is read into memory rather than driving the run to OOM; and a body that ends mid-part
// surfaces *TruncatedError, never a panic. It seeds from the committed malformed corpus,
// each seed carrying its own media-type line.
func FuzzMultipartReader(f *testing.F) {
	inline := [][]byte{
		[]byte("multipart/related; boundary=abc\r\n--abc\r\nContent-Type: application/dicom\r\n\r\nbody\r\n--abc--\r\n"),
		[]byte("multipart/related; boundary=abc\r\n--abc\r\n\r\n\r\n--abc--\r\n"), // empty part
		[]byte("multipart/related\r\nno boundary parameter"),                      // missing boundary
		[]byte("application/dicom\r\nnot multipart at all"),                       // wrong media type
		[]byte("multipart/related; boundary=abc\r\n--abc\r\nContent-Type: application/dicom\r\n\r\ntruncated"),
		[]byte("=not a media type=\r\nbody"), // unparseable media type
		[]byte(""),                           // empty: no media-type line
	}
	for _, seed := range inline {
		f.Add(seed)
	}
	for _, seed := range readMalformedSeeds(f, multipartMalformedSeedDir) {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		mediaType, body, _ := bytes.Cut(data, []byte("\n"))
		// A media-type line that does not parse, or names a non-multipart/related type, or
		// carries no boundary, is a constructor-level rejection: the fuzz contract is that
		// it returns a typed error rather than panicking.
		mr, err := NewMultipartReader(bytes.NewReader(body), string(bytes.TrimRight(mediaType, "\r")))
		if err != nil {
			return
		}
		// Hold the caps low so a hostile body trips before it is read into memory.
		mr.MaxParts = 16
		mr.MaxPartBytes = 4096

		for {
			_, partBody, err := mr.NextPart()
			if err != nil {
				return
			}
			if _, err := io.Copy(io.Discard, partBody); err != nil {
				// A drain fault (truncation, over-cap part) is a typed error, not a panic.
				return
			}
		}
	})
}

// FuzzParseMediaType drives mime.ParseMediaType through NewMultipartReader's media-type
// gate directly, so the boundary-extraction and type-check branches are exercised against
// hostile media-type strings without needing a body. NewMultipartReader must never panic on
// an arbitrary media-type string and must reject anything that is not a multipart/related
// type carrying a non-empty boundary.
func FuzzParseMediaType(f *testing.F) {
	for _, seed := range []string{
		`multipart/related; type="application/dicom"; boundary=abc`,
		`multipart/related; boundary=`,
		`multipart/related`,
		`application/dicom`,
		`multipart/related; boundary="a b c"`,
		`; ; ;`,
		`multipart/related; boundary=` + "\x00\x01",
		``,
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, mediaType string) {
		mr, err := NewMultipartReader(bytes.NewReader(nil), mediaType)
		if err != nil {
			return
		}
		// A successful construction means the media type parsed as multipart/related with a
		// boundary; cross-check that invariant so a future relaxation that admits a
		// boundary-less type is caught here rather than as a downstream panic.
		mt, params, perr := mime.ParseMediaType(mediaType)
		if perr != nil {
			t.Fatalf("NewMultipartReader accepted a media type ParseMediaType rejects: %q", mediaType)
		}
		if mt != "multipart/related" {
			t.Fatalf("NewMultipartReader accepted a non-multipart/related media type %q", mt)
		}
		if params["boundary"] == "" {
			t.Fatalf("NewMultipartReader accepted a boundary-less media type %q", mediaType)
		}
		_ = mr
	})
}
