package command

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
)

// writeUncompressedPixelDICOM writes a synthetic CT-class Part 10 file with native
// (Explicit VR LE) pixel data, so a test can drive the encode side of transcode.
// Identifiers are synthetic, never PHI.
func writeUncompressedPixelDICOM(t *testing.T, dir, sopInstanceUID string) string {
	t.Helper()
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.2") // CT Image Storage
	ds.SetString(dicom.TagSOPInstanceUID, sopInstanceUID)
	ds.SetString(dicom.TagStudyInstanceUID, "1.2.3.4.5.1")
	ds.SetString(dicom.TagSeriesInstanceUID, "1.2.3.4.5.2")
	ds.SetString(dicom.TagModality, "CT")
	ds.Set(dicom.Element{Tag: dicom.TagRows, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 4)})
	ds.Set(dicom.Element{Tag: dicom.TagColumns, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 4)})
	ds.Set(dicom.Element{Tag: dicom.TagBitsAllocated, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 8)})
	ds.Set(dicom.Element{Tag: dicom.TagSamplesPerPixel, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 1)})
	ds.Set(dicom.Element{Tag: dicom.TagPixelData, VR: dicom.VROB, Value: dicom.NewBytes(dicom.VROB, transcodeFixturePixels())})

	path := filepath.Join(dir, strings.ReplaceAll(sopInstanceUID, ".", "_")+".dcm")
	if err := ds.WriteFile(path, dicom.ExplicitVRLittleEndian); err != nil {
		t.Fatalf("write uncompressed pixel DICOM: %v", err)
	}
	return path
}

// transcodeFixturePixels is the deterministic 4x4x8-bit pixel pattern shared by the
// transcode fixtures, so a decode-after-transcode comparison has a known expectation.
func transcodeFixturePixels() []byte {
	pixels := make([]byte, 16)
	for i := range pixels {
		pixels[i] = byte(i * 7)
	}
	return pixels
}

// decodedPixels reads a Part 10 file and decodes its single frame to native bytes.
func decodedPixels(t *testing.T, path string) []byte {
	t.Helper()
	f, err := dicom.ReadFile(path)
	if err != nil {
		t.Fatalf("read transcoded file: %v", err)
	}
	pd, err := dicom.NewPixelData(f.DataSet, f.Meta.TransferSyntaxUID)
	if err != nil {
		t.Fatalf("NewPixelData: %v", err)
	}
	var pixels []byte
	for frame, err := range pd.Frames() {
		if err != nil {
			t.Fatalf("frame decode: %v", err)
		}
		pixels = append(pixels, frame.Pixels...)
	}
	return pixels
}

// TestTranscodeDecompressesRLEToExplicitVRLE is the decode-side golden: an RLE Lossless
// object transcoded to Explicit VR LE round-trips its pixels, carries the new transfer
// syntax in the meta, and reports a success JSON Line.
func TestTranscodeDecompressesRLEToExplicitVRLE(t *testing.T) {
	src := writeCompressedStorableDICOM(t, t.TempDir(), "1.2.3.4.5.60")
	outDir := filepath.Join(t.TempDir(), "out")

	stdout, stderr, code := runRadx(t, "transcode", "--format", "json",
		"--to", string(dicom.ExplicitVRLittleEndian), "--output-dir", outDir, src)
	if code != exitcode.Success {
		t.Fatalf("transcode exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}

	var r transcodeResult
	if err := json.Unmarshal([]byte(stdout), &r); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	if r.Status != "success" {
		t.Errorf("status = %q, want success (error=%q)", r.Status, r.Error)
	}
	if r.From != string(dicom.RLELossless) || r.To != string(dicom.ExplicitVRLittleEndian) {
		t.Errorf("from/to = %q/%q, want RLE Lossless -> Explicit VR LE", r.From, r.To)
	}

	out := filepath.Join(outDir, filepath.Base(src))
	got, err := dicom.ReadFile(out)
	if err != nil {
		t.Fatalf("read transcoded file: %v", err)
	}
	if got.Meta.TransferSyntaxUID != dicom.ExplicitVRLittleEndian {
		t.Errorf("TransferSyntaxUID = %q, want Explicit VR LE", got.Meta.TransferSyntaxUID)
	}
	if pixels := decodedPixels(t, out); string(pixels) != string(transcodeFixturePixels()) {
		t.Errorf("decoded pixels differ from the fixture pattern: got %v", pixels)
	}
}

// TestTranscodeEncodesToRLEByKeyword is the encode-side golden, driven with the dicom
// package's keyword form for the target: an uncompressed object transcoded to
// RLELossless (pure-Go encode) decodes back to the same pixels.
func TestTranscodeEncodesToRLEByKeyword(t *testing.T) {
	src := writeUncompressedPixelDICOM(t, t.TempDir(), "1.2.3.4.5.61")
	outDir := filepath.Join(t.TempDir(), "out")

	stdout, stderr, code := runRadx(t, "transcode", "--format", "json",
		"--to", "RLELossless", "--output-dir", outDir, src)
	if code != exitcode.Success {
		t.Fatalf("transcode exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}

	out := filepath.Join(outDir, filepath.Base(src))
	got, err := dicom.ReadFile(out)
	if err != nil {
		t.Fatalf("read transcoded file: %v", err)
	}
	if got.Meta.TransferSyntaxUID != dicom.RLELossless {
		t.Errorf("TransferSyntaxUID = %q, want RLE Lossless", got.Meta.TransferSyntaxUID)
	}
	if pixels := decodedPixels(t, out); string(pixels) != string(transcodeFixturePixels()) {
		t.Errorf("decoded pixels differ from the fixture pattern: got %v", pixels)
	}
}

// TestTranscodePixelLessObjectRewritesMetaOnly confirms an object with no (7FE0,0010)
// element passes through with a meta rewrite: the dataset is re-encoded under the target
// syntax, nothing else changes, and the run succeeds.
func TestTranscodePixelLessObjectRewritesMetaOnly(t *testing.T) {
	src := writeStorableDICOM(t, t.TempDir(), "1.2.3.4.5.62")
	outDir := filepath.Join(t.TempDir(), "out")

	_, stderr, code := runRadx(t, "transcode", "--format", "json",
		"--to", string(dicom.ImplicitVRLittleEndian), "--output-dir", outDir, src)
	if code != exitcode.Success {
		t.Fatalf("transcode exit = %d, want %d\nstderr=%q", code, exitcode.Success, stderr)
	}

	out := filepath.Join(outDir, filepath.Base(src))
	got, err := dicom.ReadFile(out)
	if err != nil {
		t.Fatalf("read transcoded file: %v", err)
	}
	if got.Meta.TransferSyntaxUID != dicom.ImplicitVRLittleEndian {
		t.Errorf("TransferSyntaxUID = %q, want Implicit VR LE", got.Meta.TransferSyntaxUID)
	}
	if modality, ok := got.DataSet.GetString(dicom.TagModality); !ok || modality != "CT" {
		t.Errorf("Modality = %q ok=%v, want CT", modality, ok)
	}
}

// TestTranscodeUnsupportedEncodeTargetFailsClosed pins the fail-closed contract for an
// encode target this build has no encoder for: the pure-Go build cannot encode JPEG
// Baseline, so the file fails with the library's typed error (exit 3, the
// unsupported-feature parse class), the per-file line reports the failure, and no
// output file is written.
func TestTranscodeUnsupportedEncodeTargetFailsClosed(t *testing.T) {
	src := writeUncompressedPixelDICOM(t, t.TempDir(), "1.2.3.4.5.63")
	outDir := filepath.Join(t.TempDir(), "out")

	stdout, _, code := runRadx(t, "transcode", "--format", "json",
		"--to", string(dicom.JPEGBaseline8Bit), "--output-dir", outDir, src)
	if code != exitcode.ParseError {
		t.Fatalf("transcode to an unsupported encode target exit = %d, want %d\nstdout=%q",
			code, exitcode.ParseError, stdout)
	}
	var r transcodeResult
	if err := json.Unmarshal([]byte(stdout), &r); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	if r.Status != "failure" || r.Error == "" {
		t.Errorf("result = %+v, want a failure with a structural error", r)
	}
	if _, err := os.Stat(filepath.Join(outDir, filepath.Base(src))); !os.IsNotExist(err) {
		t.Errorf("a failed transcode must write no output file (stat err = %v)", err)
	}
}

// TestTranscodeInPlaceRewritesOriginal confirms --in-place replaces the source file
// atomically with the transcoded object.
func TestTranscodeInPlaceRewritesOriginal(t *testing.T) {
	src := writeCompressedStorableDICOM(t, t.TempDir(), "1.2.3.4.5.64")

	_, stderr, code := runRadx(t, "transcode", "--format", "json",
		"--to", string(dicom.ExplicitVRLittleEndian), "--in-place", src)
	if code != exitcode.Success {
		t.Fatalf("transcode --in-place exit = %d, want %d\nstderr=%q", code, exitcode.Success, stderr)
	}
	got, err := dicom.ReadFile(src)
	if err != nil {
		t.Fatalf("read in-place transcoded file: %v", err)
	}
	if got.Meta.TransferSyntaxUID != dicom.ExplicitVRLittleEndian {
		t.Errorf("TransferSyntaxUID = %q, want Explicit VR LE", got.Meta.TransferSyntaxUID)
	}
}

// TestTranscodeInvalidTargetIsUsageError pins flag validation: a --to value that is
// neither a valid UID nor a known transfer syntax keyword is a usage error before any
// file is touched.
func TestTranscodeInvalidTargetIsUsageError(t *testing.T) {
	src := writeStorableDICOM(t, t.TempDir(), "1.2.3.4.5.65")
	_, _, code := runRadx(t, "transcode", "--to", "not-a-syntax", "--output-dir", t.TempDir(), src)
	if code != exitcode.UsageError {
		t.Fatalf("transcode --to not-a-syntax exit = %d, want %d", code, exitcode.UsageError)
	}
}

// TestTranscodeRequiresExactlyOneDestination pins the modify-style destination rule:
// exactly one of --output-dir or --in-place.
func TestTranscodeRequiresExactlyOneDestination(t *testing.T) {
	src := writeStorableDICOM(t, t.TempDir(), "1.2.3.4.5.66")
	if _, _, code := runRadx(t, "transcode", "--to", "RLELossless", src); code != exitcode.UsageError {
		t.Fatalf("transcode with no destination exit = %d, want %d", code, exitcode.UsageError)
	}
	if _, _, code := runRadx(t, "transcode", "--to", "RLELossless",
		"--output-dir", t.TempDir(), "--in-place", src); code != exitcode.UsageError {
		t.Fatalf("transcode with both destinations exit = %d, want %d", code, exitcode.UsageError)
	}
}

// TestTranscodeRejectsCSVFormat confirms transcode treats --format csv as a usage error:
// it is not a tabular command.
func TestTranscodeRejectsCSVFormat(t *testing.T) {
	src := writeStorableDICOM(t, t.TempDir(), "1.2.3.4.5.67")
	_, _, code := runRadx(t, "transcode", "--format", "csv",
		"--to", "RLELossless", "--output-dir", t.TempDir(), src)
	if code != exitcode.UsageError {
		t.Fatalf("transcode --format csv exit = %d, want %d", code, exitcode.UsageError)
	}
}

// TestTranscodeSameSyntaxIsByteIdentical pins the passthrough fix: transcoding a file to its own
// transfer syntax must copy the bytes unchanged (no re-encode that rebuilds File Meta and would
// break checksums or signatures).
func TestTranscodeSameSyntaxIsByteIdentical(t *testing.T) {
	src := writeCompressedStorableDICOM(t, t.TempDir(), "1.2.3.4.5.70") // RLE Lossless
	outDir := filepath.Join(t.TempDir(), "out")

	_, stderr, code := runRadx(t, "transcode", "--format", "json",
		"--to", string(dicom.RLELossless), "--output-dir", outDir, src)
	if code != exitcode.Success {
		t.Fatalf("same-syntax transcode exit = %d, want %d\nstderr=%q", code, exitcode.Success, stderr)
	}
	in, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	out, err := os.ReadFile(filepath.Join(outDir, filepath.Base(src)))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !bytes.Equal(in, out) {
		t.Errorf("same-syntax transcode changed the bytes (in %d, out %d); passthrough must be byte-identical", len(in), len(out))
	}
}

// TestTranscodeCollisionPreflightPreservesFirstInput pins the preflight fix: two inputs sharing a
// basename under one --output-dir must be rejected BEFORE any write, so the first input's would-be
// output (which is itself the second input) is never destroyed.
func TestTranscodeCollisionPreflightPreservesFirstInput(t *testing.T) {
	outDir := t.TempDir()
	sub := filepath.Join(t.TempDir(), "a")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}
	first := writeCompressedStorableDICOM(t, sub, "1.2.3.4.5.71") // a/<name>.dcm
	base := filepath.Base(first)                                  //
	second := filepath.Join(outDir, base)                         // out/<name>.dcm (an input AND a dest)
	if err := os.WriteFile(second, mustRead(t, first), 0o600); err != nil {
		t.Fatal(err)
	}
	before := mustRead(t, second)

	_, _, code := runRadx(t, "transcode", "--to", string(dicom.ExplicitVRLittleEndian),
		"--output-dir", outDir, first, second)
	if code != exitcode.UsageError {
		t.Fatalf("colliding transcode exit = %d, want %d (usage error, preflight)", code, exitcode.UsageError)
	}
	if !bytes.Equal(before, mustRead(t, second)) {
		t.Error("the colliding input was modified; the preflight must reject before any write")
	}
}

// mustRead reads a file or fails the test.
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return b
}
