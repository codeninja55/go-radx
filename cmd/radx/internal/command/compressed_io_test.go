package command

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
)

// writeCompressedStorableDICOM writes a synthetic CT-class Part 10 file whose pixel
// data is RLE Lossless encapsulated, built through the library's own transcode seam
// so the fixture is conformant. Identifiers are synthetic, never PHI.
func writeCompressedStorableDICOM(t *testing.T, dir, sopInstanceUID string) string {
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
	pixels := make([]byte, 16)
	for i := range pixels {
		pixels[i] = byte(i * 7)
	}
	ds.Set(dicom.Element{Tag: dicom.TagPixelData, VR: dicom.VROB, Value: dicom.NewBytes(dicom.VROB, pixels)})

	f := &dicom.File{
		Meta: &dicom.FileMeta{
			MediaStorageSOPClassUID:    "1.2.840.10008.5.1.4.1.1.2",
			MediaStorageSOPInstanceUID: dicom.SOPInstanceUID(sopInstanceUID),
			TransferSyntaxUID:          dicom.ExplicitVRLittleEndian,
		},
		DataSet: ds,
	}
	pd, err := dicom.NewPixelData(ds, dicom.ExplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("NewPixelData: %v", err)
	}
	rle, err := dicom.Transcode(pd, dicom.RLELossless)
	if err != nil {
		t.Fatalf("Transcode to RLE: %v", err)
	}
	if err := f.SetPixelData(rle); err != nil {
		t.Fatalf("SetPixelData: %v", err)
	}
	path := filepath.Join(dir, strings.ReplaceAll(sopInstanceUID, ".", "_")+".dcm")
	if err := dicom.WriteFile(path, f); err != nil {
		t.Fatalf("write compressed DICOM: %v", err)
	}
	return path
}

// TestStoreTranscodeToUncompressedSendsCompressedObject is the --transcode-to
// acceptance: a compressed (RLE) object decompressed to Explicit VR LE on send must
// reach the SCP and exit 0.
func TestStoreTranscodeToUncompressedSendsCompressedObject(t *testing.T) {
	host, port := startStorageServer(t, "")
	f := writeCompressedStorableDICOM(t, t.TempDir(), "1.2.3.4.5.40")

	stdout, stderr, code := runRadx(t, "store", "--format", "json",
		"--host", host, "--port", strconv.Itoa(port), "--called-ae", "RADX-SCP",
		"--transcode-to", string(dicom.ExplicitVRLittleEndian), f)
	if code != exitcode.Success {
		t.Fatalf("store --transcode-to exit = %d, want %d\nstdout=%q\nstderr=%q",
			code, exitcode.Success, stdout, stderr)
	}
	lines := nonEmptyLines(stdout)
	var summary storeSummary
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &summary); err != nil {
		t.Fatalf("summary line is not valid JSON: %v", err)
	}
	if summary.Status != "success" || summary.Succeeded != 1 {
		t.Errorf("summary = %+v, want success 1/1", summary)
	}
}

// TestStoreCompressedWithoutTranscodeFailsPerFile pins the honest failure for a
// compressed object sent without --transcode-to: the store transport carries the
// negotiated uncompressed syntaxes only, so the object fails per file with a clear
// error instead of being silently decompressed (RADX-011).
func TestStoreCompressedWithoutTranscodeFailsPerFile(t *testing.T) {
	host, port := startStorageServer(t, "")
	f := writeCompressedStorableDICOM(t, t.TempDir(), "1.2.3.4.5.41")

	stdout, _, code := runRadx(t, "store", "--format", "json",
		"--host", host, "--port", strconv.Itoa(port), "--called-ae", "RADX-SCP", f)
	if code == exitcode.Success {
		t.Fatalf("store of a compressed object without --transcode-to must fail\nstdout=%q", stdout)
	}
	if !strings.Contains(stdout, "transcode") {
		t.Errorf("per-file error should point at --transcode-to; stdout=%q", stdout)
	}
}

// TestStoreTranscodeToPixelLessObjectSendsAsStored confirms an object with no
// (7FE0,0010) element (an SR-like instance) passes through --transcode-to
// unchanged: there is nothing to transcode and nothing is silently altered.
func TestStoreTranscodeToPixelLessObjectSendsAsStored(t *testing.T) {
	host, port := startStorageServer(t, "")
	f := writeStorableDICOM(t, t.TempDir(), "1.2.3.4.5.42")

	_, stderr, code := runRadx(t, "store", "--format", "json",
		"--host", host, "--port", strconv.Itoa(port), "--called-ae", "RADX-SCP",
		"--transcode-to", string(dicom.ExplicitVRLittleEndian), f)
	if code != exitcode.Success {
		t.Fatalf("store --transcode-to of a pixel-less object exit = %d, want %d\nstderr=%q",
			code, exitcode.Success, stderr)
	}
}

// TestStoreTranscodeToInvalidUIDIsUsageError pins flag validation: a malformed
// transfer syntax UID is a usage error before any file is touched.
func TestStoreTranscodeToInvalidUIDIsUsageError(t *testing.T) {
	f := writeStorableDICOM(t, t.TempDir(), "1.2.3.4.5.43")
	_, _, code := runRadx(t, "store", "--host", "127.0.0.1", "--port", "11112",
		"--transcode-to", "not-a-uid", f)
	if code != exitcode.UsageError {
		t.Fatalf("store --transcode-to not-a-uid exit = %d, want %d", code, exitcode.UsageError)
	}
}

// TestDumpCompressedFile confirms radx dump inspects a compressed (RLE) file: the
// metadata elements render and the pixel element is summarised structurally.
func TestDumpCompressedFile(t *testing.T) {
	f := writeCompressedStorableDICOM(t, t.TempDir(), "1.2.3.4.5.44")
	stdout, stderr, code := runRadx(t, "dump", "--format", "json", f)
	if code != exitcode.Success {
		t.Fatalf("dump exit = %d, want %d\nstderr=%q", code, exitcode.Success, stderr)
	}
	var df dumpFile
	if err := json.Unmarshal([]byte(stdout), &df); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	if df.Status != "success" {
		t.Fatalf("dump status = %q, want success (error=%q)", df.Status, df.Error)
	}
	if e, ok := df.Elements["0008,0060"]; !ok || e.Value != "CT" {
		t.Errorf("Modality = %+v, want CT", e)
	}
	if _, ok := df.Elements["7FE0,0010"]; !ok {
		t.Error("pixel data element missing from the dump")
	}
}

// TestModifyCompressedFile confirms radx modify edits the metadata of a compressed
// file and writes it back under its original (compressed) transfer syntax with the
// pixel stream intact.
func TestModifyCompressedFile(t *testing.T) {
	src := writeCompressedStorableDICOM(t, t.TempDir(), "1.2.3.4.5.45")
	outDir := filepath.Join(t.TempDir(), "modified")

	_, stderr, code := runRadx(t, "modify", "--format", "json",
		"--output-dir", outDir,
		"--insert", "PatientID=ANON-002",
		src)
	if code != exitcode.Success {
		t.Fatalf("modify exit = %d, want %d\nstderr=%q", code, exitcode.Success, stderr)
	}

	out := filepath.Join(outDir, filepath.Base(src))
	got, err := dicom.ReadFile(out)
	if err != nil {
		t.Fatalf("read modified file: %v", err)
	}
	if got.Meta.TransferSyntaxUID != dicom.RLELossless {
		t.Errorf("modified TransferSyntaxUID = %q, want RLE Lossless", got.Meta.TransferSyntaxUID)
	}
	if pid, ok := got.DataSet.GetString(dicom.TagPatientID); !ok || pid != "ANON-002" {
		t.Errorf("PatientID = %q ok=%v, want ANON-002", pid, ok)
	}
	pd, err := dicom.NewPixelData(got.DataSet, got.Meta.TransferSyntaxUID)
	if err != nil {
		t.Fatalf("NewPixelData on the modified file: %v", err)
	}
	var frames int
	for _, err := range pd.Frames() {
		if err != nil {
			t.Fatalf("frame decode after modify: %v", err)
		}
		frames++
	}
	if frames != 1 {
		t.Errorf("decoded %d frames, want 1", frames)
	}
}
