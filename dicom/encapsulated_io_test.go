package dicom

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// compressedFixtures names the vendored encapsulated Part 10 fixtures and their
// transfer syntaxes (see testdata/dicom/README.md for provenance).
var compressedFixtures = []struct {
	name string
	ts   TransferSyntax
}{
	{"liver_rle.dcm", RLELossless},
	{"liver_j2k.dcm", JPEG2000Lossless},
	{"MR_small_jpeg_ls_lossless.dcm", JPEGLSLossless},
	{"SC_jpeg_no_color_transform.dcm", JPEGBaseline8Bit},
	{"HTJ2KLossless_08_RGB.dcm", HTJ2KLossless},
}

// TestReadEncapsulatedFixtureRetainsDataSet is the acceptance for compressed
// Part 10 read: Read/ReadFile must parse the full dataset of an encapsulated file
// (metadata reachable like pydicom's dcmread) and retain the (7FE0,0010) fragment
// stream on the dataset without decoding it.
func TestReadEncapsulatedFixtureRetainsDataSet(t *testing.T) {
	for _, fx := range compressedFixtures {
		t.Run(fx.name, func(t *testing.T) {
			f, err := ReadFile(filepath.Join("..", "testdata", "dicom", fx.name))
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			if f.Meta.TransferSyntaxUID != fx.ts {
				t.Errorf("TransferSyntaxUID = %q, want %q", f.Meta.TransferSyntaxUID, fx.ts)
			}
			if _, ok := f.DataSet.GetInt(TagRows); !ok {
				t.Error("Rows is unreachable: the main dataset was not retained")
			}
			e, ok := f.DataSet.Get(TagPixelData)
			if !ok {
				t.Fatal("PixelData element was not retained on the dataset")
			}
			ev, ok := e.Value.(*encapsulatedValue)
			if !ok {
				t.Fatalf("PixelData value is %T, want the retained encapsulated fragment stream", e.Value)
			}
			if len(ev.stream) == 0 {
				t.Error("retained fragment stream is empty")
			}

			// The retained stream must feed the decode pipeline exactly like the
			// dedicated pixel reader.
			pd, err := NewPixelData(f.DataSet, f.Meta.TransferSyntaxUID)
			if err != nil {
				t.Fatalf("NewPixelData from the retained dataset: %v", err)
			}
			if !pd.IsEncapsulated() {
				t.Error("PixelData built from the retained dataset should be encapsulated")
			}
		})
	}
}

// TestReadEncapsulatedMatchesReadPixelData checks the unified read path: the
// PixelData built from a Read-retained dataset must decode the same frames as
// ReadPixelData. RLE keeps this build-tag independent (pure-Go codec).
func TestReadEncapsulatedMatchesReadPixelData(t *testing.T) {
	path := filepath.Join("..", "testdata", "dicom", "liver_rle.dcm")
	want, err := ReadPixelData(path)
	if err != nil {
		t.Fatalf("ReadPixelData: %v", err)
	}
	f, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got, err := NewPixelData(f.DataSet, f.Meta.TransferSyntaxUID)
	if err != nil {
		t.Fatalf("NewPixelData: %v", err)
	}
	wantFrames := collectFrames(t, want)
	gotFrames := collectFrames(t, got)
	if len(wantFrames) != len(gotFrames) {
		t.Fatalf("frame count %d != %d", len(gotFrames), len(wantFrames))
	}
	for i := range wantFrames {
		if !bytes.Equal(wantFrames[i], gotFrames[i]) {
			t.Errorf("frame %d differs between Read-retained and ReadPixelData paths", i)
		}
	}
}

func collectFrames(t *testing.T, pd *PixelData) [][]byte {
	t.Helper()
	var out [][]byte
	for frame, err := range pd.Frames() {
		if err != nil {
			t.Fatalf("frame %d: %v", len(out), err)
		}
		out = append(out, frame.Pixels)
	}
	return out
}

// TestEncapsulatedFixtureMainDataSetByteIdentical is the byte-exact round-trip
// acceptance for compressed Part 10 write: read each encapsulated fixture and
// re-encode its main dataset; the bytes after the file-meta group — including the
// undefined-length (7FE0,0010) element, its Basic Offset Table, and every fragment —
// must reproduce exactly.
func TestEncapsulatedFixtureMainDataSetByteIdentical(t *testing.T) {
	for _, fx := range compressedFixtures {
		t.Run(fx.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "testdata", "dicom", fx.name))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			f, err := Read(bytes.NewReader(raw))
			if err != nil {
				t.Fatalf("Read: %v", err)
			}

			br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
			if _, err := readPreamble(br); err != nil {
				t.Fatalf("readPreamble: %v", err)
			}
			h, _ := readElementHeader(br, ExplicitVRLittleEndian)
			gv, _ := decodeValue(br, h, encodingFor(ExplicitVRLittleEndian), nil)
			groupLen := gv.(*Ints).Ints()[0]
			mainStart := br.offset() + groupLen
			originalMain := raw[mainStart:]

			var out bytes.Buffer
			if err := writeDataSet(&out, f.DataSet, f.Meta.TransferSyntaxUID); err != nil {
				t.Fatalf("writeDataSet: %v", err)
			}
			if !bytes.Equal(out.Bytes(), originalMain) {
				t.Errorf("%s main dataset re-encode not byte-identical: got %d bytes, want %d bytes",
					fx.name, out.Len(), len(originalMain))
			}
		})
	}
}

// TestWriteFileEncapsulatedRoundTrip exercises the public WriteFile/ReadFile pair
// for a compressed transfer syntax: the re-read dataset must carry a pixel stream
// byte-identical to the source's.
func TestWriteFileEncapsulatedRoundTrip(t *testing.T) {
	src, err := ReadFile(filepath.Join("..", "testdata", "dicom", "liver_rle.dcm"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	path := filepath.Join(t.TempDir(), "out.dcm")
	if err := WriteFile(path, src); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := ReadFile(path)
	if err != nil {
		t.Fatalf("re-ReadFile: %v", err)
	}
	we, _ := src.DataSet.Get(TagPixelData)
	ge, ok := got.DataSet.Get(TagPixelData)
	if !ok {
		t.Fatal("re-read file lost its PixelData element")
	}
	wev := we.Value.(*encapsulatedValue)
	gev, ok := ge.Value.(*encapsulatedValue)
	if !ok {
		t.Fatalf("re-read PixelData value is %T, want encapsulated", ge.Value)
	}
	if !bytes.Equal(wev.stream, gev.stream) {
		t.Error("pixel fragment stream not byte-identical across WriteFile/ReadFile")
	}
}

// TestWriteEncapsulatedValueUnderUncompressedSyntaxFails pins the fail-closed rule:
// a dataset still holding compressed fragments cannot be emitted under an
// uncompressed transfer syntax (PS3.5 A.4 requires native pixel data there); the
// caller must transcode first. No bytes may reach the writer.
func TestWriteEncapsulatedValueUnderUncompressedSyntaxFails(t *testing.T) {
	f, err := ReadFile(filepath.Join("..", "testdata", "dicom", "liver_rle.dcm"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	f.Meta.TransferSyntaxUID = ExplicitVRLittleEndian

	var buf bytes.Buffer
	err = Write(&buf, f)
	if _, ok := errors.AsType[*ValueError](err); !ok {
		t.Fatalf("Write = %v, want *ValueError for encapsulated pixel data under an uncompressed syntax", err)
	}
	if buf.Len() != 0 {
		t.Errorf("Write emitted %d bytes before failing; must reject before writing", buf.Len())
	}
}

// TestReadRejectsUnknownTransferSyntax pins the fail-closed rule for a transfer
// syntax go-radx does not recognise: the main-dataset encoding of an unknown or
// private syntax cannot be assumed, so the read is rejected with a typed error
// rather than guessed.
func TestReadRejectsUnknownTransferSyntax(t *testing.T) {
	const privateTS TransferSyntax = "1.2.840.113619.5.2" // GE private; not Explicit VR LE
	stream := seedEncapsulatedWithTS(privateTS)
	if _, err := Read(bytes.NewReader(stream)); err == nil {
		t.Error("Read should reject an unrecognised transfer syntax")
	}
}

// seedEncapsulatedWithTS builds a minimal encapsulated Part 10 stream under ts: the
// geometry elements, then an empty Basic Offset Table, one 4-byte fragment, and the
// Sequence Delimitation Item.
func seedEncapsulatedWithTS(ts TransferSyntax) []byte {
	return seedEncapsulatedPart10(ts)
}

// TestReadEncapsulatedHostileInputs drives the new Read path with malformed
// encapsulated streams. Every case must surface a typed error — never a panic, and
// never a silent success (PRD 9.3; Codex DCM-006).
func TestReadEncapsulatedHostileInputs(t *testing.T) {
	base := seedEncapsulatedPart10(RLELossless)

	// pixelHeaderOff locates the (7FE0,0010) element header in the seed so the cases
	// can tamper with the fragment stream that follows it.
	pixelHeader := encapsulatedPixelHeader()
	pixelOff := bytes.Index(base, pixelHeader)
	if pixelOff < 0 {
		t.Fatal("seed does not contain the encapsulated pixel header")
	}
	streamOff := pixelOff + len(pixelHeader)

	cases := []struct {
		name    string
		mutate  func([]byte) []byte
		wantEOF bool
	}{
		{
			name: "truncated mid-fragment",
			mutate: func(b []byte) []byte {
				return b[:len(b)-10] // cuts into the fragment value and loses the delimiter
			},
			wantEOF: true,
		},
		{
			name: "fragment length past EOF",
			mutate: func(b []byte) []byte {
				out := bytes.Clone(b)
				// First item after the BOT item (8-byte header + 0-byte value): set its
				// declared length far past the bytes present. The length stays even so
				// the structural even-length check does not fire before the EOF guard.
				fragHeader := streamOff + 8
				out[fragHeader+4] = 0xFE
				out[fragHeader+5] = 0xFF
				out[fragHeader+6] = 0x00
				out[fragHeader+7] = 0x00
				return out
			},
			wantEOF: true,
		},
		{
			name: "Basic Offset Table with undefined length",
			mutate: func(b []byte) []byte {
				out := bytes.Clone(b)
				out[streamOff+4] = 0xFF
				out[streamOff+5] = 0xFF
				out[streamOff+6] = 0xFF
				out[streamOff+7] = 0xFF
				return out
			},
		},
		{
			name: "missing sequence delimiter",
			mutate: func(b []byte) []byte {
				return b[:len(b)-8]
			},
			wantEOF: true,
		},
		{
			name: "foreign tag inside the fragment stream",
			mutate: func(b []byte) []byte {
				out := bytes.Clone(b)
				// Overwrite the Sequence Delimitation Item tag with a dataset tag.
				delim := len(out) - 8
				out[delim] = 0x08
				out[delim+1] = 0x00
				out[delim+2] = 0x16
				out[delim+3] = 0x00
				return out
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := tc.mutate(bytes.Clone(base))
			_, err := Read(bytes.NewReader(data))
			if err == nil {
				t.Fatal("Read of a malformed encapsulated stream must fail")
			}
			if tc.wantEOF {
				if !errors.Is(err, io.ErrUnexpectedEOF) {
					t.Errorf("error = %v, want io.ErrUnexpectedEOF", err)
				}
				return
			}
			_, isValueError := errors.AsType[*ValueError](err)
			_, isLimitError := errors.AsType[*LimitExceededError](err)
			if !isValueError && !isLimitError && !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Errorf("error = %v (%T), want a typed dicom error", err, err)
			}
		})
	}
}

// TestStopAtPixelDataOnEncapsulatedFile checks the partial-read option against a
// compressed file: the dataset is returned without the pixel element and without
// consuming the fragment stream.
func TestStopAtPixelDataOnEncapsulatedFile(t *testing.T) {
	f, err := ReadFile(filepath.Join("..", "testdata", "dicom", "liver_rle.dcm"), WithStopAtPixelData())
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if _, ok := f.DataSet.Get(TagPixelData); ok {
		t.Error("pixel data should be skipped with WithStopAtPixelData")
	}
	if _, ok := f.DataSet.GetInt(TagRows); !ok {
		t.Error("metadata before the pixel element should be readable")
	}
}

// TestSetPixelDataTranscodeRoundTrip is the dataset-level transcode acceptance:
// decode -> re-encode -> write must work end to end through the public seam
// (NewPixelData -> Transcode -> File.SetPixelData -> Write), in both directions.
// MR2_UNCI.dcm is a conformant uncompressed 16-bit MR image; the chain compresses
// it to RLE Lossless, writes and re-reads the compressed file, decompresses it back
// to Explicit VR LE, writes and re-reads again, and requires pixel-exact frames.
func TestSetPixelDataTranscodeRoundTrip(t *testing.T) {
	f, err := ReadFile(filepath.Join("..", "testdata", "dicom", "MR2_UNCI.dcm"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	pd, err := NewPixelData(f.DataSet, f.Meta.TransferSyntaxUID)
	if err != nil {
		t.Fatalf("NewPixelData: %v", err)
	}
	wantFrames := collectFrames(t, pd)

	// Compress: native -> RLE Lossless -> Part 10 file.
	rle, err := Transcode(pd, RLELossless)
	if err != nil {
		t.Fatalf("Transcode to RLE: %v", err)
	}
	if err := f.SetPixelData(rle); err != nil {
		t.Fatalf("SetPixelData(RLE): %v", err)
	}
	if f.Meta.TransferSyntaxUID != RLELossless {
		t.Errorf("SetPixelData left TransferSyntaxUID = %q, want RLE Lossless", f.Meta.TransferSyntaxUID)
	}
	dir := t.TempDir()
	rlePath := filepath.Join(dir, "rle.dcm")
	if err := WriteFile(rlePath, f); err != nil {
		t.Fatalf("WriteFile(RLE): %v", err)
	}

	// Decompress: re-read the compressed file, transcode back, write again.
	compressed, err := ReadFile(rlePath)
	if err != nil {
		t.Fatalf("re-ReadFile(RLE): %v", err)
	}
	if compressed.Meta.TransferSyntaxUID != RLELossless {
		t.Fatalf("re-read TransferSyntaxUID = %q, want RLE Lossless", compressed.Meta.TransferSyntaxUID)
	}
	cpd, err := NewPixelData(compressed.DataSet, compressed.Meta.TransferSyntaxUID)
	if err != nil {
		t.Fatalf("NewPixelData(RLE): %v", err)
	}
	native, err := Transcode(cpd, ExplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("Transcode to native: %v", err)
	}
	if err := compressed.SetPixelData(native); err != nil {
		t.Fatalf("SetPixelData(native): %v", err)
	}
	nativePath := filepath.Join(dir, "native.dcm")
	if err := WriteFile(nativePath, compressed); err != nil {
		t.Fatalf("WriteFile(native): %v", err)
	}

	got, err := ReadFile(nativePath)
	if err != nil {
		t.Fatalf("re-ReadFile(native): %v", err)
	}
	if got.Meta.TransferSyntaxUID != ExplicitVRLittleEndian {
		t.Errorf("final TransferSyntaxUID = %q, want Explicit VR LE", got.Meta.TransferSyntaxUID)
	}
	gotPD, err := NewPixelData(got.DataSet, got.Meta.TransferSyntaxUID)
	if err != nil {
		t.Fatalf("NewPixelData after round-trip: %v", err)
	}
	gotFrames := collectFrames(t, gotPD)
	if len(gotFrames) != len(wantFrames) {
		t.Fatalf("frame count %d != %d", len(gotFrames), len(wantFrames))
	}
	for i := range wantFrames {
		if !bytes.Equal(gotFrames[i], wantFrames[i]) {
			t.Errorf("frame %d pixels differ after compress/decompress round-trip", i)
		}
	}
}

// TestSetPixelDataRemovesStaleExtendedOffsets pins the staleness rule: replacing the
// pixel stream drops the Extended Offset Table elements, which describe the previous
// stream's byte layout.
func TestSetPixelDataRemovesStaleExtendedOffsets(t *testing.T) {
	f, err := ReadFile(filepath.Join("..", "testdata", "dicom", "liver_rle.dcm"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	pd, err := NewPixelData(f.DataSet, f.Meta.TransferSyntaxUID)
	if err != nil {
		t.Fatalf("NewPixelData: %v", err)
	}
	native, err := Transcode(pd, ExplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("Transcode: %v", err)
	}
	f.DataSet.Set(Element{Tag: TagExtendedOffsetTable, VR: VROV, Value: NewBytes(VROV, make([]byte, 8))})
	f.DataSet.Set(Element{Tag: TagExtendedOffsetTableLengths, VR: VROV, Value: NewBytes(VROV, make([]byte, 8))})
	if err := f.SetPixelData(native); err != nil {
		t.Fatalf("SetPixelData: %v", err)
	}
	if _, ok := f.DataSet.Get(TagExtendedOffsetTable); ok {
		t.Error("stale ExtendedOffsetTable survived SetPixelData")
	}
	if _, ok := f.DataSet.Get(TagExtendedOffsetTableLengths); ok {
		t.Error("stale ExtendedOffsetTableLengths survived SetPixelData")
	}
}

// TestSetPixelDataGuards pins the typed-error guards on the transcode seam.
func TestSetPixelDataGuards(t *testing.T) {
	f, err := ReadFile(filepath.Join("..", "testdata", "dicom", "liver_rle.dcm"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if err := f.SetPixelData(nil); err == nil {
		t.Error("SetPixelData(nil) should fail")
	}
	var nilFile *File
	pd, err := NewPixelData(f.DataSet, f.Meta.TransferSyntaxUID)
	if err != nil {
		t.Fatalf("NewPixelData: %v", err)
	}
	if err := nilFile.SetPixelData(pd); err == nil {
		t.Error("SetPixelData on a nil File should fail")
	}
}

// TestDataSetCloneCopiesEncapsulatedStream checks the retained fragment stream is
// deep-copied by Clone, never aliased.
func TestDataSetCloneCopiesEncapsulatedStream(t *testing.T) {
	f, err := ReadFile(filepath.Join("..", "testdata", "dicom", "liver_rle.dcm"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	cl := f.DataSet.Clone()
	orig, _ := f.DataSet.Get(TagPixelData)
	cp, _ := cl.Get(TagPixelData)
	ov := orig.Value.(*encapsulatedValue)
	cv, ok := cp.Value.(*encapsulatedValue)
	if !ok {
		t.Fatalf("cloned PixelData value is %T, want encapsulated", cp.Value)
	}
	if !bytes.Equal(ov.stream, cv.stream) {
		t.Fatal("cloned stream differs")
	}
	if len(cv.stream) > 0 {
		cv.stream[0] ^= 0xFF
		if ov.stream[0] == cv.stream[0] {
			t.Error("clone aliases the source stream")
		}
	}
}
