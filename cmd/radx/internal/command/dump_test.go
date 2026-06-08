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

// writeSyntheticDICOM writes a small, non-PHI Part-10 file to a temp path and returns it. The
// values are deliberately structural and fictional (a Secondary Capture SOP class, a test
// instance UID, a synthetic patient name token), so no real patient data ever enters the test
// corpus (PRD §9.1).
func writeSyntheticDICOM(t *testing.T) string {
	t.Helper()
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0008, 0x0016), VR: dicom.VRUI, Value: dicom.NewStrings(dicom.VRUI, "1.2.840.10008.5.1.4.1.1.7")})
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0008, 0x0018), VR: dicom.VRUI, Value: dicom.NewStrings(dicom.VRUI, "1.2.3.4.5.6.7.8.9")})
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0008, 0x0060), VR: dicom.VRCS, Value: dicom.NewStrings(dicom.VRCS, "OT")})
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0010, 0x0010), VR: dicom.VRPN, Value: dicom.NewStrings(dicom.VRPN, "TEST^FIXTURE")})
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0028, 0x0010), VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 256)})

	f := &dicom.File{
		Meta: &dicom.FileMeta{
			MediaStorageSOPClassUID:    "1.2.840.10008.5.1.4.1.1.7",
			MediaStorageSOPInstanceUID: "1.2.3.4.5.6.7.8.9",
			TransferSyntaxUID:          dicom.ExplicitVRLittleEndian,
		},
		DataSet: ds,
	}

	path := filepath.Join(t.TempDir(), "fixture.dcm")
	if err := dicom.WriteFile(path, f); err != nil {
		t.Fatalf("write synthetic DICOM: %v", err)
	}
	return path
}

// syntheticPatientName is the synthetic PatientName token written into every fixture. It is a
// deliberately fictional sentinel, never a real patient value (PRD §9.1), so a test can assert
// that a default dump shows it and a --redact dump masks it.
const syntheticPatientName = "TEST^FIXTURE"

// writeSyntheticDICOMWithPixelData writes a synthetic Part-10 file that also carries a small
// PixelData (7FE0,0010) element, so a test can assert that pixel-data bytes are summarised
// structurally and never rendered unless --process-pixel-data is set.
func writeSyntheticDICOMWithPixelData(t *testing.T, dir string) string {
	t.Helper()
	ds := dicom.NewDataSet()
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0008, 0x0016), VR: dicom.VRUI, Value: dicom.NewStrings(dicom.VRUI, "1.2.840.10008.5.1.4.1.1.7")})
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0008, 0x0018), VR: dicom.VRUI, Value: dicom.NewStrings(dicom.VRUI, "1.2.3.4.5.6.7.8.9")})
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0010, 0x0010), VR: dicom.VRPN, Value: dicom.NewStrings(dicom.VRPN, syntheticPatientName)})
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0028, 0x0010), VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 2)})
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x0028, 0x0011), VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 2)})
	ds.Set(dicom.Element{Tag: dicom.NewTag(0x7FE0, 0x0010), VR: dicom.VROW, Value: dicom.NewBytes(dicom.VROW, []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07})})

	f := &dicom.File{
		Meta: &dicom.FileMeta{
			MediaStorageSOPClassUID:    "1.2.840.10008.5.1.4.1.1.7",
			MediaStorageSOPInstanceUID: "1.2.3.4.5.6.7.8.9",
			TransferSyntaxUID:          dicom.ExplicitVRLittleEndian,
		},
		DataSet: ds,
	}

	path := filepath.Join(dir, "pixels.dcm")
	if err := dicom.WriteFile(path, f); err != nil {
		t.Fatalf("write synthetic DICOM with pixel data: %v", err)
	}
	return path
}

// runRadx runs the radx entry point in-process with the given args, capturing stdout and
// stderr separately so a test can assert the clean-stdout contract.
func runRadx(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	return runRadxStdin(t, "", args...)
}

// runRadxStdin runs the radx entry point in-process with the given stdin payload and args, for the
// commands that read a message from stdin (hl7 send, the convert HL7 mappers).
func runRadxStdin(t *testing.T, stdin string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errBuf bytes.Buffer
	code = Main(args, strings.NewReader(stdin), &out, &errBuf)
	return out.String(), errBuf.String(), code
}

// TestDumpJSONGolden is the json golden: a synthetic file dumps to a clean, tag-keyed JSON
// object with the expected structure, and exits 0.
func TestDumpJSONGolden(t *testing.T) {
	path := writeSyntheticDICOM(t)
	stdout, _, code := runRadx(t, "dump", "--format", "json", path)
	if code != exitcode.Success {
		t.Fatalf("dump exit = %d, want %d; stdout=%q", code, exitcode.Success, stdout)
	}

	var got dumpFile
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	if got.Status != "success" {
		t.Errorf("status = %q, want success", got.Status)
	}
	// Spot-check a structural element by its tag key (SOP Class UID, Modality, Rows).
	if e, ok := got.Elements["0008,0016"]; !ok || e.Keyword != "SOPClassUID" {
		t.Errorf("SOPClassUID element = %+v, ok=%v", e, ok)
	}
	if e, ok := got.Elements["0008,0060"]; !ok || e.Value != "OT" {
		t.Errorf("Modality element = %+v, ok=%v", e, ok)
	}
	if e, ok := got.Elements["0028,0010"]; !ok || e.Value != "256" {
		t.Errorf("Rows element = %+v, ok=%v", e, ok)
	}
}

// TestDumpCSVGolden is the csv golden: a synthetic file dumps to RFC 4180 CSV with a header
// row and one row per element, and exits 0.
func TestDumpCSVGolden(t *testing.T) {
	path := writeSyntheticDICOM(t)
	stdout, _, code := runRadx(t, "dump", "--format", "csv", path)
	if code != exitcode.Success {
		t.Fatalf("dump exit = %d, want %d", code, exitcode.Success)
	}
	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	if lines[0] != "file,status,tag,keyword,vr,value" {
		t.Errorf("csv header = %q, want the documented column set", lines[0])
	}
	// One header + five elements.
	if len(lines) != 6 {
		t.Errorf("csv line count = %d, want 6 (header + 5 elements)\n%s", len(lines), stdout)
	}
	for _, want := range []string{"SOPClassUID", "Modality", "Rows"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("csv missing expected structural keyword %q:\n%s", want, stdout)
		}
	}
}

// TestDumpHumanGolden is the human golden: a synthetic file dumps to an indented listing
// headed by the file path, naming the structural elements, and exits 0.
func TestDumpHumanGolden(t *testing.T) {
	path := writeSyntheticDICOM(t)
	stdout, _, code := runRadx(t, "dump", path)
	if code != exitcode.Success {
		t.Fatalf("dump exit = %d, want %d", code, exitcode.Success)
	}
	for _, want := range []string{path, "SOPClassUID", "Modality", "Rows"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("human dump missing %q:\n%s", want, stdout)
		}
	}
}

// TestDumpTruncatedExits3 is the truncation regression: a file cut mid-value is a parse
// failure (exit 3), never a clean dump. Truncation is failure (docs/reference/cli.md
// "Honest-failure rules").
func TestDumpTruncatedExits3(t *testing.T) {
	path := writeSyntheticDICOM(t)
	full, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	// Cut the file well inside the main dataset (past the 132-byte preamble+magic and the
	// file-meta group) so the reader fails mid-element, not at a clean record boundary.
	truncated := full[:len(full)-8]
	truncPath := filepath.Join(t.TempDir(), "truncated.dcm")
	if err := os.WriteFile(truncPath, truncated, 0o600); err != nil {
		t.Fatalf("write truncated fixture: %v", err)
	}

	stdout, _, code := runRadx(t, "dump", "--format", "json", truncPath)
	if code != exitcode.ParseError {
		t.Fatalf("truncated dump exit = %d, want %d (truncation is a parse failure)\nstdout=%q", code, exitcode.ParseError, stdout)
	}
	// The per-file machine output flags the failure.
	if !strings.Contains(stdout, "\"status\": \"failure\"") {
		t.Errorf("truncated dump did not flag the file as failed:\n%s", stdout)
	}
}

// TestDumpMissingFileExits5 is the file-I/O regression: a missing file is a file error (exit
// 5), not a parse error.
func TestDumpMissingFileExits5(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.dcm")
	_, _, code := runRadx(t, "dump", "--format", "json", missing)
	if code != exitcode.FileIOError {
		t.Fatalf("missing-file dump exit = %d, want %d", code, exitcode.FileIOError)
	}
}

// TestDumpIgnoreErrorsExits0 confirms --ignore-errors opts a failing batch into a zero exit
// for exploratory use, while still recording the per-file failure.
func TestDumpIgnoreErrorsExits0(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.dcm")
	stdout, _, code := runRadx(t, "dump", "--format", "json", "--ignore-errors", missing)
	if code != exitcode.Success {
		t.Fatalf("dump --ignore-errors exit = %d, want %d", code, exitcode.Success)
	}
	if !strings.Contains(stdout, "\"status\": \"failure\"") {
		t.Errorf("dump --ignore-errors did not record the per-file failure:\n%s", stdout)
	}
}

// TestDumpShowsValuesByDefault confirms a default dump is an authorized local inspection that
// shows element values, including PHI-sensitive ones: the synthetic PatientName sentinel is
// present. A dump the user explicitly ran on a file they hold is exempt from the ambient-logging
// no-PHI rule (docs/reference/cli.md dump).
func TestDumpShowsValuesByDefault(t *testing.T) {
	path := writeSyntheticDICOM(t)
	stdout, _, code := runRadx(t, "dump", "--format", "json", path)
	if code != exitcode.Success {
		t.Fatalf("dump exit = %d, want %d", code, exitcode.Success)
	}
	if !strings.Contains(stdout, syntheticPatientName) {
		t.Errorf("default dump did not show the PatientName value %q:\n%s", syntheticPatientName, stdout)
	}
}

// TestDumpRedactMasksPHI confirms --redact masks PHI-sensitive element values to the fixed
// marker: the synthetic PatientName sentinel is replaced by [redacted] and never appears.
func TestDumpRedactMasksPHI(t *testing.T) {
	path := writeSyntheticDICOM(t)
	stdout, _, code := runRadx(t, "dump", "--format", "json", "--redact", path)
	if code != exitcode.Success {
		t.Fatalf("dump --redact exit = %d, want %d", code, exitcode.Success)
	}
	if strings.Contains(stdout, syntheticPatientName) {
		t.Errorf("dump --redact leaked the PatientName value %q:\n%s", syntheticPatientName, stdout)
	}
	if !strings.Contains(stdout, redactedMarker) {
		t.Errorf("dump --redact did not mask the value to %q:\n%s", redactedMarker, stdout)
	}
}

// TestDumpMultiFileJSONIsJSONLines confirms multiple files dumped as json are emitted as a JSON
// Lines stream — one parseable JSON object per line — rather than concatenated indented
// documents a consumer cannot parse.
func TestDumpMultiFileJSONIsJSONLines(t *testing.T) {
	a := writeSyntheticDICOM(t)
	b := writeSyntheticDICOM(t)
	stdout, _, code := runRadx(t, "dump", "--format", "json", a, b)
	if code != exitcode.Success {
		t.Fatalf("multi-file dump exit = %d, want %d\nstdout=%q", code, exitcode.Success, stdout)
	}
	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("multi-file json line count = %d, want 2 (one per file)\n%s", len(lines), stdout)
	}
	for i, line := range lines {
		var df dumpFile
		if err := json.Unmarshal([]byte(line), &df); err != nil {
			t.Fatalf("line %d is not valid JSON: %v\nline=%q", i, err, line)
		}
		if df.Status != "success" {
			t.Errorf("line %d status = %q, want success", i, df.Status)
		}
	}
}

// TestDumpRecursiveDirectory confirms -R descends a directory for *.dcm files and that pixel
// data is summarised structurally — the raw bytes never appear — unless --process-pixel-data is
// set.
func TestDumpRecursiveDirectory(t *testing.T) {
	dir := t.TempDir()
	writeSyntheticDICOMWithPixelData(t, dir)

	stdout, _, code := runRadx(t, "dump", "--format", "json", "-R", dir)
	if code != exitcode.Success {
		t.Fatalf("dump -R exit = %d, want %d\nstdout=%q", code, exitcode.Success, stdout)
	}

	var df dumpFile
	if err := json.Unmarshal([]byte(strings.TrimRight(stdout, "\n")), &df); err != nil {
		t.Fatalf("recursive dump line is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	pix, ok := df.Elements["7FE0,0010"]
	if !ok {
		t.Fatalf("recursive dump did not include the PixelData element:\n%s", stdout)
	}
	if !strings.Contains(pix.Value, "not processed") {
		t.Errorf("PixelData value = %q, want a structural summary (pixel bytes must be omitted by default)", pix.Value)
	}
}

// TestDumpDirectoryWithoutRecursiveIsUsageError confirms a directory input without -R is a usage
// error (exit 2): a directory is not a DICOM file, and the fault is in the invocation.
func TestDumpDirectoryWithoutRecursiveIsUsageError(t *testing.T) {
	dir := t.TempDir()
	writeSyntheticDICOMWithPixelData(t, dir)
	_, _, code := runRadx(t, "dump", "--format", "json", dir)
	if code != exitcode.UsageError {
		t.Fatalf("dump on a directory without -R exit = %d, want %d (usage error)", code, exitcode.UsageError)
	}
}
