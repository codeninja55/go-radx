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
	"github.com/codeninja55/go-radx/dicomweb"
)

// composeSourceDataSet is the synthetic dataset the compose fixtures marshal to PS3.18 JSON:
// structural identifiers plus one string, one numeric, and one binary attribute, never PHI.
func composeSourceDataSet() *dicom.DataSet {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.7") // Secondary Capture
	ds.SetString(dicom.TagSOPInstanceUID, "1.2.3.4.600.1")
	ds.SetString(dicom.TagStudyInstanceUID, "1.2.3.4.600.2")
	ds.SetString(dicom.TagSeriesInstanceUID, "1.2.3.4.600.3")
	ds.SetString(dicom.TagModality, "OT")
	ds.Set(dicom.Element{Tag: dicom.TagPatientName, VR: dicom.VRPN, Value: dicom.NewStrings(dicom.VRPN, "TEST^FIXTURE")})
	ds.Set(dicom.Element{Tag: dicom.TagRows, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 4)})
	ds.Set(dicom.Element{Tag: dicom.TagPixelData, VR: dicom.VROB, Value: dicom.NewBytes(dicom.VROB, []byte{0, 7, 14, 21})})
	return ds
}

// writePS318JSON marshals ds through the dicomweb PS3.18 codec (Annex F) to a file, so the
// compose input is the real DICOM-JSON shape, not radx dump's tag-keyed shape.
func writePS318JSON(t *testing.T, ds *dicom.DataSet) string {
	t.Helper()
	data, err := dicomweb.MarshalJSON(ds)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	path := filepath.Join(t.TempDir(), "in.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write json fixture: %v", err)
	}
	return path
}

// TestComposeRoundTripsPS318JSONToPart10 is the golden round trip: PS3.18 JSON composes to a
// Part 10 file whose meta honours the present SOP/Study/Series UIDs, whose dataset matches the
// source element for element, and whose bytes equal the library's own Part 10 encoding of the
// same dataset (the byte-check).
func TestComposeRoundTripsPS318JSONToPart10(t *testing.T) {
	src := composeSourceDataSet()
	in := writePS318JSON(t, src)
	out := filepath.Join(t.TempDir(), "out.dcm")

	stdout, stderr, code := runRadx(t, "compose", "--format", "json", in, out)
	if code != exitcode.Success {
		t.Fatalf("compose exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}
	var r composeResult
	if err := json.Unmarshal([]byte(stdout), &r); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	if r.Status != "success" || r.Output != out {
		t.Errorf("result = %+v, want success at %q", r, out)
	}
	if len(r.MintedUIDs) != 0 {
		t.Errorf("minted UIDs = %v, want none (every UID was present and must be honoured)", r.MintedUIDs)
	}

	got, err := dicom.ReadFile(out)
	if err != nil {
		t.Fatalf("read composed file: %v", err)
	}
	if got.Meta.TransferSyntaxUID != dicom.ExplicitVRLittleEndian {
		t.Errorf("TransferSyntaxUID = %q, want Explicit VR LE (the default)", got.Meta.TransferSyntaxUID)
	}
	if got.Meta.MediaStorageSOPInstanceUID != "1.2.3.4.600.1" {
		t.Errorf("MediaStorageSOPInstanceUID = %q, want the honoured source UID", got.Meta.MediaStorageSOPInstanceUID)
	}
	for _, tag := range []dicom.Tag{dicom.TagSOPClassUID, dicom.TagSOPInstanceUID, dicom.TagStudyInstanceUID,
		dicom.TagSeriesInstanceUID, dicom.TagModality, dicom.TagPatientName} {
		want, _ := src.GetString(tag)
		if v, ok := got.DataSet.GetString(tag); !ok || v != want {
			t.Errorf("element %s = %q ok=%v, want %q", tag, v, ok, want)
		}
	}

	// Byte-check: the composed file must equal the library's own Part 10 encoding of the same
	// dataset under the same transfer syntax.
	expectedPath := filepath.Join(t.TempDir(), "expected.dcm")
	if err := src.WriteFile(expectedPath, dicom.ExplicitVRLittleEndian); err != nil {
		t.Fatalf("write expected file: %v", err)
	}
	expected, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read expected file: %v", err)
	}
	composed, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read composed file bytes: %v", err)
	}
	if !bytes.Equal(composed, expected) {
		t.Errorf("composed Part 10 bytes differ from the library encoding (len %d vs %d)", len(composed), len(expected))
	}
}

// TestComposeMintsMissingUIDs confirms the meta glue mints fresh conformant SOP/Study/Series
// Instance UIDs ONLY where the JSON carries none, reporting which were minted.
func TestComposeMintsMissingUIDs(t *testing.T) {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.7")
	ds.SetString(dicom.TagModality, "OT")
	in := writePS318JSON(t, ds)
	out := filepath.Join(t.TempDir(), "out.dcm")

	stdout, stderr, code := runRadx(t, "compose", "--format", "json", in, out)
	if code != exitcode.Success {
		t.Fatalf("compose exit = %d, want %d\nstderr=%q", code, exitcode.Success, stderr)
	}
	var r composeResult
	if err := json.Unmarshal([]byte(stdout), &r); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	if len(r.MintedUIDs) != 3 {
		t.Errorf("minted UIDs = %v, want the three instance UIDs", r.MintedUIDs)
	}

	got, err := dicom.ReadFile(out)
	if err != nil {
		t.Fatalf("read composed file: %v", err)
	}
	for _, tag := range []dicom.Tag{dicom.TagSOPInstanceUID, dicom.TagStudyInstanceUID, dicom.TagSeriesInstanceUID} {
		v, ok := got.DataSet.GetString(tag)
		if !ok || v == "" {
			t.Errorf("element %s absent from the composed file; a fresh UID must be minted", tag)
			continue
		}
		if err := dicom.UID(v).Validate(); err != nil {
			t.Errorf("minted %s = %q is not a conformant UID: %v", tag, v, err)
		}
	}
}

// TestComposeReadsStdin confirms the "-" input convention shared with the other stdin-reading
// commands.
func TestComposeReadsStdin(t *testing.T) {
	data, err := dicomweb.MarshalJSON(composeSourceDataSet())
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	out := filepath.Join(t.TempDir(), "out.dcm")

	_, stderr, code := runRadxStdin(t, string(data), "compose", "--format", "json", "-", out)
	if code != exitcode.Success {
		t.Fatalf("compose from stdin exit = %d, want %d\nstderr=%q", code, exitcode.Success, stderr)
	}
	if _, err := dicom.ReadFile(out); err != nil {
		t.Errorf("composed file from stdin does not read back: %v", err)
	}
}

// TestComposeMissingSOPClassFailsClosed pins the required-meta rule: a dataset with no SOP Class
// UID cannot derive File Meta, so compose fails closed (exit 3) and writes nothing.
func TestComposeMissingSOPClassFailsClosed(t *testing.T) {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagModality, "OT")
	in := writePS318JSON(t, ds)
	out := filepath.Join(t.TempDir(), "out.dcm")

	_, _, code := runRadx(t, "compose", in, out)
	if code != exitcode.ParseError {
		t.Fatalf("compose without SOPClassUID exit = %d, want %d", code, exitcode.ParseError)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("a failed compose must write no output file (stat err = %v)", err)
	}
}

// TestComposeUnknownVRFailsClosed pins the hostile-input contract: an attribute carrying a VR
// outside the PS3.5 registry is refused by the codec's typed error (exit 3), never guessed.
func TestComposeUnknownVRFailsClosed(t *testing.T) {
	in := filepath.Join(t.TempDir(), "bad-vr.json")
	if err := os.WriteFile(in, []byte(`{"00100010":{"vr":"XX","Value":["A"]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "out.dcm")

	_, _, code := runRadx(t, "compose", in, out)
	if code != exitcode.ParseError {
		t.Fatalf("compose with an unknown VR exit = %d, want %d", code, exitcode.ParseError)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("a failed compose must write no output file (stat err = %v)", err)
	}
}

// TestComposeBadInlineBinaryFailsClosed pins the second hostile input: InlineBinary that is not
// valid base64 is refused fail-closed (exit 3) with no output.
func TestComposeBadInlineBinaryFailsClosed(t *testing.T) {
	in := filepath.Join(t.TempDir(), "bad-b64.json")
	if err := os.WriteFile(in, []byte(`{"7FE00010":{"vr":"OB","InlineBinary":"!!!not-base64!!!"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "out.dcm")

	_, _, code := runRadx(t, "compose", in, out)
	if code != exitcode.ParseError {
		t.Fatalf("compose with bad InlineBinary exit = %d, want %d", code, exitcode.ParseError)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("a failed compose must write no output file (stat err = %v)", err)
	}
}

// TestComposeEncapsulatedTargetIsUsageError pins the transfer-syntax constraint: PS3.18 JSON
// carries native binary values, so an encapsulated --transfer-syntax cannot be honoured and is
// refused before any read.
func TestComposeEncapsulatedTargetIsUsageError(t *testing.T) {
	in := writePS318JSON(t, composeSourceDataSet())
	_, _, code := runRadx(t, "compose", "--transfer-syntax", "RLELossless", in,
		filepath.Join(t.TempDir(), "out.dcm"))
	if code != exitcode.UsageError {
		t.Fatalf("compose --transfer-syntax RLELossless exit = %d, want %d", code, exitcode.UsageError)
	}
}

// TestComposeMissingInputExits5 confirms an unreadable JSON input keeps its file-I/O class.
func TestComposeMissingInputExits5(t *testing.T) {
	_, _, code := runRadx(t, "compose", filepath.Join(t.TempDir(), "absent.json"),
		filepath.Join(t.TempDir(), "out.dcm"))
	if code != exitcode.FileIOError {
		t.Fatalf("compose with a missing input exit = %d, want %d", code, exitcode.FileIOError)
	}
}

// TestComposeRejectsCSVFormat confirms compose treats --format csv as a usage error.
func TestComposeRejectsCSVFormat(t *testing.T) {
	in := writePS318JSON(t, composeSourceDataSet())
	_, _, code := runRadx(t, "compose", "--format", "csv", in, filepath.Join(t.TempDir(), "out.dcm"))
	if code != exitcode.UsageError {
		t.Fatalf("compose --format csv exit = %d, want %d", code, exitcode.UsageError)
	}
}

// TestComposeRefusesExistingOutputWithoutOverwrite pins the clobber guard: a second compose to an
// existing output path is refused unless --overwrite is passed, so an existing Part 10 file is
// never silently replaced.
func TestComposeRefusesExistingOutputWithoutOverwrite(t *testing.T) {
	in := writePS318JSON(t, composeSourceDataSet())
	out := filepath.Join(t.TempDir(), "out.dcm")

	if _, _, code := runRadx(t, "compose", in, out); code != exitcode.Success {
		t.Fatalf("first compose exit = %d, want %d", code, exitcode.Success)
	}
	before := mustRead(t, out)

	_, _, code := runRadx(t, "compose", in, out)
	if code != exitcode.UsageError {
		t.Fatalf("second compose without --overwrite exit = %d, want %d", code, exitcode.UsageError)
	}
	if !bytes.Equal(before, mustRead(t, out)) {
		t.Error("the existing output was modified without --overwrite")
	}

	if _, _, code := runRadx(t, "compose", "--overwrite", in, out); code != exitcode.Success {
		t.Fatalf("compose --overwrite exit = %d, want %d", code, exitcode.Success)
	}
}

// TestComposeWarnsOnNonconformantPresentUID confirms a present-but-malformed dataset UID (here a
// StudyInstanceUID, which is not promoted into File Meta) is honoured verbatim (not silently minted
// over) but warned about on stderr, so the operator is told the file carries a non-conformant
// identifier rather than it passing unnoticed. (A non-conformant SOP Class/Instance UID, which the
// Part 10 File Meta requires, is instead rejected by the writer's own meta validation - fail-closed.)
func TestComposeWarnsOnNonconformantPresentUID(t *testing.T) {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.7")
	ds.SetString(dicom.TagSOPInstanceUID, "1.2.3.4.700.1")
	ds.SetString(dicom.TagStudyInstanceUID, "1.2.3.4..5") // empty component -> non-conformant
	ds.SetString(dicom.TagSeriesInstanceUID, "1.2.3.4.700.3")
	in := writePS318JSON(t, ds)
	out := filepath.Join(t.TempDir(), "out.dcm")

	_, stderr, code := runRadx(t, "compose", "--format", "json", in, out)
	if code != exitcode.Success {
		t.Fatalf("compose with a nonconformant present UID exit = %d, want %d\nstderr=%q", code, exitcode.Success, stderr)
	}
	if !strings.Contains(strings.ToLower(stderr), "conformant") && !strings.Contains(strings.ToLower(stderr), "studyinstanceuid") {
		t.Errorf("expected a stderr warning about the non-conformant UID, got:\n%s", stderr)
	}
	got, err := dicom.ReadFile(out)
	if err != nil {
		t.Fatalf("read composed file: %v", err)
	}
	if v, _ := got.DataSet.GetString(dicom.TagStudyInstanceUID); v != "1.2.3.4..5" {
		t.Errorf("present UID = %q, want it honoured verbatim (%q)", v, "1.2.3.4..5")
	}
}
