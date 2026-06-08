// Package hostilecorpus replays the committed malformed-input corpus through the parsers that sit on
// go-radx's trust boundary, under an enforced memory ceiling and a wall-clock timeout, asserting no
// parser panics, hangs, or exhausts memory on hostile input.
//
// This is the harness half of the hostile-input memory-capped corpus gate. The memory ceiling and the
// hang-is-a-failure timeout are ENFORCED by the process that runs it, not by this code: CI (and the
// `mise run hostile:corpus` task) invoke it as
//
//	GOMEMLIMIT=<cap> timeout <wall> go test ./internal/hostilecorpus/...
//
// so a parser that drives allocation past the cap takes the Go runtime past its soft limit and into a
// hard out-of-memory abort (a non-zero exit, a FAILURE), and a parser that wedges is killed by
// `timeout` (also a FAILURE). This test asserts the in-process contract — every corpus file decodes
// to a typed error or a value WITHOUT panicking — and records the peak heap each file drives so a
// regression toward the cap is visible in the log before it becomes an OOM.
//
// Scope. This harness replays the RAW, on-disk malformed corpus under dicomweb/testdata/malformed —
// the DICOM-JSON and multipart/related files that are byte-for-byte the parser inputs — through the
// exported dicomweb parsers, mirroring the consumption convention the package fuzz targets use
// (SOURCE.md: JSON files to UnmarshalJSON; multipart files are a media-type line, a newline, then the
// body, fed to NewMultipartReader + a NextPart drain). The Go-fuzz seed corpora for the dicom,
// dimse/pdu, and fhir/r5 parsers are stored in Go's fuzz-encoded format and are auto-replayed as
// subtests by a plain `go test` of those packages; the CI job runs that replay under the SAME
// GOMEMLIMIT + timeout so every committed seed crosses its parser under the cap too. Keeping this
// harness on the raw corpus avoids re-decoding Go's fuzz wire format here and keeps each parser's
// seeds exercised in the package that owns them.
package hostilecorpus

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/codeninja55/go-radx/dicomweb"
)

// malformedRoot is the committed raw malformed corpus, relative to this test's package directory.
const malformedRoot = "../../dicomweb/testdata/malformed"

// jsonMaxDepth and the multipart caps mirror the tightened limits the dicomweb fuzz targets impose
// (parser_fuzz_test.go), so a cap-tripping corpus file surfaces a typed *LimitExceededError here the
// same way it does under fuzzing — the point of the gate is that a hostile input trips a bounded
// error before it can drive the run to OOM, never that it decodes.
const (
	jsonMaxDepth     = 8
	multipartMaxN    = 16
	multipartMaxByte = 4096
)

// TestHostileCorpusUnderMemoryCap feeds every raw malformed corpus file to its parser and asserts the
// parser returns (panicking is the only failure; a typed error or a value are both acceptable safe
// outcomes). The enclosing process enforces the memory ceiling and the timeout; this test proves the
// no-panic contract and logs the peak heap each file drove so a creep toward the cap is visible.
func TestHostileCorpusUnderMemoryCap(t *testing.T) {
	jsonFiles := corpusFiles(t, filepath.Join(malformedRoot, "json"), ".json")
	multipartFiles := corpusFiles(t, filepath.Join(malformedRoot, "multipart"), ".txt")
	if len(jsonFiles) == 0 || len(multipartFiles) == 0 {
		t.Fatalf("hostile corpus is empty (json=%d multipart=%d); the corpus path is wrong or the corpus was removed",
			len(jsonFiles), len(multipartFiles))
	}

	for _, path := range jsonFiles {
		path := path
		t.Run("json/"+filepath.Base(path), func(t *testing.T) {
			data := readCorpus(t, path)
			peak := withHeapPeak(func() {
				// UnmarshalJSON is the DICOM-JSON decode trust boundary (PS3.18 Annex F). A
				// malformed document must return a typed error, never panic. recoverPanic below
				// converts a panic into a test failure with the offending file named.
				defer recoverPanic(t, path)
				_, _ = dicomweb.UnmarshalJSON(data, dicomweb.WithMaxJSONDepth(jsonMaxDepth))
			})
			t.Logf("decoded %d bytes, peak heap delta %d bytes", len(data), peak)
		})
	}

	for _, path := range multipartFiles {
		path := path
		t.Run("multipart/"+filepath.Base(path), func(t *testing.T) {
			data := readCorpus(t, path)
			peak := withHeapPeak(func() {
				defer recoverPanic(t, path)
				replayMultipart(data)
			})
			t.Logf("read %d bytes, peak heap delta %d bytes", len(data), peak)
		})
	}
}

// replayMultipart mirrors FuzzMultipartReader's consumption of a corpus file: the first line is the
// media type NewMultipartReader parses, the remainder is the body NextPart frames. The caps are held
// low so a hostile body trips its *LimitExceededError before it is read into memory. Every error path
// is a safe outcome; only a panic (caught by the deferred recover in the caller) fails the gate.
func replayMultipart(data []byte) {
	mediaType, body, _ := bytes.Cut(data, []byte("\n"))
	mr, err := dicomweb.NewMultipartReader(bytes.NewReader(body), string(bytes.TrimRight(mediaType, "\r")))
	if err != nil {
		return
	}
	mr.MaxParts = multipartMaxN
	mr.MaxPartBytes = multipartMaxByte
	for {
		_, partBody, err := mr.NextPart()
		if err != nil {
			return
		}
		if _, err := io.Copy(io.Discard, partBody); err != nil {
			return
		}
	}
}

// withHeapPeak runs fn and returns the increase in cumulative heap allocation (HeapAlloc delta) it
// drove, a coarse signal of how much memory a corpus file pushed the parser to allocate. It is a
// log-only diagnostic; the hard ceiling is the process GOMEMLIMIT the runner sets.
func withHeapPeak(fn func()) uint64 {
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	fn()
	runtime.ReadMemStats(&after)
	if after.HeapAlloc < before.HeapAlloc {
		return 0
	}
	return after.HeapAlloc - before.HeapAlloc
}

// recoverPanic converts a parser panic into a test failure that names the corpus file, so a crash on
// hostile input is attributed to the exact input rather than surfacing as an opaque process abort.
func recoverPanic(t *testing.T, path string) {
	t.Helper()
	if r := recover(); r != nil {
		t.Fatalf("parser panicked on hostile corpus file %s: %v", path, r)
	}
}

// corpusFiles returns the sorted set of files with the given extension directly under dir. The order
// is deterministic (ReadDir sorts by name) so the gate runs the corpus in a stable order.
func corpusFiles(t *testing.T, dir, ext string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read corpus dir %s: %v", dir, err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ext {
			continue
		}
		files = append(files, filepath.Join(dir, e.Name()))
	}
	return files
}

// readCorpus reads a corpus file, failing the test if it cannot be read.
func readCorpus(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read corpus file %s: %v", path, err)
	}
	return data
}
