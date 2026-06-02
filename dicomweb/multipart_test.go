package dicomweb

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"strings"
	"testing"
)

const mediaTypeDICOM = "application/dicom"

func TestMultipartRoundTripTwoParts(t *testing.T) {
	part1 := []byte("\x00first dicom payload\xff")
	part2 := []byte("\x00second dicom payload\xff")

	var buf bytes.Buffer
	mw := NewMultipartWriter(&buf, mediaTypeDICOM)
	if err := mw.AddPart(mediaTypeDICOM, bytes.NewReader(part1)); err != nil {
		t.Fatalf("AddPart 1: %v", err)
	}
	if err := mw.AddPart(mediaTypeDICOM, bytes.NewReader(part2)); err != nil {
		t.Fatalf("AddPart 2: %v", err)
	}
	boundary, err := mw.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if boundary == "" {
		t.Fatal("Close returned an empty boundary")
	}

	mediaType := fmt.Sprintf("multipart/related; boundary=%q; type=%q", boundary, mediaTypeDICOM)
	mr, err := NewMultipartReader(&buf, mediaType)
	if err != nil {
		t.Fatalf("NewMultipartReader: %v", err)
	}

	wantBodies := [][]byte{part1, part2}
	for i, want := range wantBodies {
		ct, body, err := mr.NextPart()
		if err != nil {
			t.Fatalf("NextPart %d: %v", i, err)
		}
		mt, _, perr := mime.ParseMediaType(ct)
		if perr != nil || mt != mediaTypeDICOM {
			t.Errorf("part %d content type = %q, want %s", i, ct, mediaTypeDICOM)
		}
		got, err := io.ReadAll(body)
		if err != nil {
			t.Fatalf("read part %d: %v", i, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("part %d body = %v, want %v", i, got, want)
		}
	}

	if _, _, err := mr.NextPart(); !errors.Is(err, io.EOF) {
		t.Errorf("after last part, NextPart err = %v, want io.EOF", err)
	}
}

func TestMultipartWriterContentType(t *testing.T) {
	var buf bytes.Buffer
	mw := NewMultipartWriter(&buf, mediaTypeDICOM)
	if err := mw.AddPart(mediaTypeDICOM, strings.NewReader("payload")); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	ct := mw.ContentType()

	// The outer Content-Type must carry the root type and a boundary that round-trips
	// through the reader.
	mt, params, err := mime.ParseMediaType(ct)
	if err != nil {
		t.Fatalf("ParseMediaType(%q): %v", ct, err)
	}
	if mt != "multipart/related" {
		t.Errorf("media type = %q, want multipart/related", mt)
	}
	if params["type"] != mediaTypeDICOM {
		t.Errorf("type param = %q, want %s", params["type"], mediaTypeDICOM)
	}
	boundary, err := mw.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if params["boundary"] != boundary {
		t.Errorf("boundary param = %q, want %q", params["boundary"], boundary)
	}
	if _, err := NewMultipartReader(&buf, ct); err != nil {
		t.Errorf("reader rejects the writer's own ContentType %q: %v", ct, err)
	}
}

func TestMultipartPreservesRawBytesIgnoringTransferEncoding(t *testing.T) {
	// A part declaring Content-Transfer-Encoding: quoted-printable must not have its body
	// transparently decoded: DICOMweb payloads are raw binary octets. The reader uses
	// NextRawPart, so the "=3D" sequence below reaches the caller verbatim rather than
	// being decoded to "=".
	boundary := "testboundary"
	raw := "--" + boundary + "\r\n" +
		"Content-Type: application/octet-stream\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n\r\n" +
		"A=3DB\r\n" +
		"--" + boundary + "--\r\n"

	mediaType := fmt.Sprintf("multipart/related; boundary=%q", boundary)
	mr, err := NewMultipartReader(strings.NewReader(raw), mediaType)
	if err != nil {
		t.Fatalf("NewMultipartReader: %v", err)
	}
	_, body, err := mr.NextPart()
	if err != nil {
		t.Fatalf("NextPart: %v", err)
	}
	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(got) != "A=3DB" {
		t.Errorf("body = %q, want the raw, undecoded %q", got, "A=3DB")
	}
}

func TestMultipartPartCountCapBeforeReadingAll(t *testing.T) {
	// Assemble a body with more parts than the reader's cap allows.
	var buf bytes.Buffer
	mw := NewMultipartWriter(&buf, mediaTypeDICOM)
	const parts = 5
	for i := 0; i < parts; i++ {
		if err := mw.AddPart(mediaTypeDICOM, strings.NewReader(fmt.Sprintf("payload-%d", i))); err != nil {
			t.Fatalf("AddPart %d: %v", i, err)
		}
	}
	boundary, err := mw.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	mediaType := fmt.Sprintf("multipart/related; boundary=%q", boundary)
	mr, err := NewMultipartReader(&buf, mediaType)
	if err != nil {
		t.Fatalf("NewMultipartReader: %v", err)
	}
	mr.MaxParts = 2

	// Reading up to the cap must succeed; exceeding it returns ErrLimitExceeded
	// at the part boundary, not after buffering the whole body.
	if _, _, err := mr.NextPart(); err != nil {
		t.Fatalf("part 0: %v", err)
	}
	if _, _, err := mr.NextPart(); err != nil {
		t.Fatalf("part 1: %v", err)
	}
	_, _, err = mr.NextPart()
	if !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("part 2 err = %v, want ErrLimitExceeded", err)
	}
}

func TestMultipartExactlyMaxPartsAccepted(t *testing.T) {
	// A body with exactly MaxParts parts must be accepted: the cap trips only on the
	// part beyond the cap, and the call after the last part returns io.EOF.
	var buf bytes.Buffer
	mw := NewMultipartWriter(&buf, mediaTypeDICOM)
	const parts = 3
	for i := 0; i < parts; i++ {
		if err := mw.AddPart(mediaTypeDICOM, strings.NewReader(fmt.Sprintf("p-%d", i))); err != nil {
			t.Fatalf("AddPart %d: %v", i, err)
		}
	}
	boundary, err := mw.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	mediaType := fmt.Sprintf("multipart/related; boundary=%q", boundary)
	mr, err := NewMultipartReader(&buf, mediaType)
	if err != nil {
		t.Fatalf("NewMultipartReader: %v", err)
	}
	mr.MaxParts = parts

	for i := 0; i < parts; i++ {
		_, body, err := mr.NextPart()
		if err != nil {
			t.Fatalf("part %d at exactly the cap should be accepted, got %v", i, err)
		}
		if _, err := io.ReadAll(body); err != nil {
			t.Fatalf("read part %d: %v", i, err)
		}
	}
	if _, _, err := mr.NextPart(); !errors.Is(err, io.EOF) {
		t.Errorf("after exactly MaxParts parts, err = %v, want io.EOF", err)
	}
}

func TestMultipartPartSizeCapEnforcedOnSkippedBody(t *testing.T) {
	// A caller that skips a part body must still trip the per-part cap on an oversized
	// part: advancing drains the unread remainder through the bounded reader rather than
	// letting mime/multipart consume it uncapped.
	var buf bytes.Buffer
	mw := NewMultipartWriter(&buf, mediaTypeDICOM)
	if err := mw.AddPart(mediaTypeDICOM, strings.NewReader(strings.Repeat("X", 8192))); err != nil {
		t.Fatalf("AddPart 0: %v", err)
	}
	if err := mw.AddPart(mediaTypeDICOM, strings.NewReader("small")); err != nil {
		t.Fatalf("AddPart 1: %v", err)
	}
	boundary, err := mw.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	mediaType := fmt.Sprintf("multipart/related; boundary=%q", boundary)
	mr, err := NewMultipartReader(&buf, mediaType)
	if err != nil {
		t.Fatalf("NewMultipartReader: %v", err)
	}
	mr.MaxParts = 2
	mr.MaxPartBytes = 64 // far below the 8192-byte first part

	// Fetch the first part but skip reading its body, then advance.
	if _, _, err := mr.NextPart(); err != nil {
		t.Fatalf("part 0: %v", err)
	}
	_, _, err = mr.NextPart()
	if !errors.Is(err, ErrLimitExceeded) {
		t.Errorf("advancing past a skipped oversized body err = %v, want ErrLimitExceeded", err)
	}
}

func TestMultipartPartSizeCapBeforeAllocation(t *testing.T) {
	body := strings.Repeat("A", 4096)
	var buf bytes.Buffer
	mw := NewMultipartWriter(&buf, mediaTypeDICOM)
	if err := mw.AddPart(mediaTypeDICOM, strings.NewReader(body)); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	boundary, err := mw.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	mediaType := fmt.Sprintf("multipart/related; boundary=%q", boundary)
	mr, err := NewMultipartReader(&buf, mediaType)
	if err != nil {
		t.Fatalf("NewMultipartReader: %v", err)
	}
	mr.MaxPartBytes = 64

	_, partBody, err := mr.NextPart()
	if err != nil {
		t.Fatalf("NextPart: %v", err)
	}
	// The cap is enforced as the body is read, before the full part lands in memory.
	_, err = io.ReadAll(partBody)
	if !errors.Is(err, ErrLimitExceeded) {
		t.Errorf("reading an over-cap part err = %v, want ErrLimitExceeded", err)
	}
}

func TestMultipartTruncatedPartIsUnexpectedEOF(t *testing.T) {
	var buf bytes.Buffer
	mw := NewMultipartWriter(&buf, mediaTypeDICOM)
	if err := mw.AddPart(mediaTypeDICOM, strings.NewReader("a complete enough payload to truncate")); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	boundary, err := mw.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Truncate the body mid-part: drop the trailing closing boundary so the part
	// never terminates cleanly.
	full := buf.Bytes()
	truncated := full[:len(full)-10]

	mediaType := fmt.Sprintf("multipart/related; boundary=%q", boundary)
	mr, err := NewMultipartReader(bytes.NewReader(truncated), mediaType)
	if err != nil {
		t.Fatalf("NewMultipartReader: %v", err)
	}

	_, body, err := mr.NextPart()
	if err != nil {
		// Some truncations surface at NextPart; that is acceptable as long as it is EOF-typed.
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("NextPart err = %v, want io.ErrUnexpectedEOF", err)
		}
		return
	}
	_, err = io.ReadAll(body)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("reading a truncated part err = %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestNewMultipartReaderRejectsNonMultipart(t *testing.T) {
	_, err := NewMultipartReader(strings.NewReader(""), "application/dicom+json")
	if err == nil {
		t.Fatal("expected an error for a non-multipart media type")
	}
}

func TestNewMultipartReaderRejectsMissingBoundary(t *testing.T) {
	_, err := NewMultipartReader(strings.NewReader(""), "multipart/related")
	if err == nil {
		t.Fatal("expected an error for a multipart media type without a boundary")
	}
}

func TestMalformedMultipartErrorIsStructural(t *testing.T) {
	// A malformed part header must surface a structural error that does not echo the raw
	// (possibly PHI-bearing) input bytes (PRD §9.1).
	secret := "PATIENTSECRET999"
	boundary := "b"
	body := "--" + boundary + "\r\n" +
		"Content-Type" + secret + "\r\n\r\n" + // missing colon: a malformed header line
		"data\r\n--" + boundary + "--\r\n"

	mediaType := fmt.Sprintf("multipart/related; boundary=%q", boundary)
	mr, err := NewMultipartReader(strings.NewReader(body), mediaType)
	if err != nil {
		t.Fatalf("NewMultipartReader: %v", err)
	}
	_, _, err = mr.NextPart()
	if err == nil {
		t.Fatal("expected an error for a malformed part header")
	}
	var mpErr *MalformedPartError
	if !errors.As(err, &mpErr) {
		t.Errorf("error = %v, want *MalformedPartError", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("error echoes raw input bytes: %q", err.Error())
	}
}

func TestNewMultipartReaderRejectsNonRelatedSubtype(t *testing.T) {
	// The reader serves multipart/related only; multipart/mixed or form-data must fail.
	for _, mt := range []string{
		`multipart/mixed; boundary="abc"`,
		`multipart/form-data; boundary="abc"`,
	} {
		if _, err := NewMultipartReader(strings.NewReader(""), mt); err == nil {
			t.Errorf("media type %q should be rejected", mt)
		}
	}
}

func TestMultipartDefaultCaps(t *testing.T) {
	var buf bytes.Buffer
	mw := NewMultipartWriter(&buf, mediaTypeDICOM)
	boundary, err := mw.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	mediaType := fmt.Sprintf("multipart/related; boundary=%q", boundary)
	mr, err := NewMultipartReader(&buf, mediaType)
	if err != nil {
		t.Fatalf("NewMultipartReader: %v", err)
	}
	if mr.MaxParts != defaultMaxParts {
		t.Errorf("MaxParts default = %d, want %d", mr.MaxParts, defaultMaxParts)
	}
	if mr.MaxPartBytes != defaultMaxPartBytes {
		t.Errorf("MaxPartBytes default = %d, want %d", mr.MaxPartBytes, defaultMaxPartBytes)
	}
}
