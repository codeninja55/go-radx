//go:build unix

package phisweep

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
	"github.com/codeninja55/go-radx/hl7v2"
	"github.com/codeninja55/go-radx/logging"
)

// The sentinels below are deliberately synthetic and never represent a real
// patient. Each is shaped like the value it stands in for so a leak through a
// value-formatting path is caught exactly as a real value would be, yet each is
// distinctive enough that an incidental substring match is implausible.
const (
	sentinelPatientName = "SENTINEL^PHI^DONOTLOG"
	sentinelPatientID   = "ZZZTEST-MRN-PHI-SENTINEL"
	sentinelAccession   = "ZZZTEST-ACC-PHI-SENTINEL"
	sentinelBirthDate   = "ZZZTEST^PHI^SENTINEL-DOB"
)

// builtSentinels are the tokens the harness plants into the datasets and messages
// it constructs itself.
var builtSentinels = []string{
	sentinelPatientName,
	sentinelPatientID,
	sentinelAccession,
	sentinelBirthDate,
}

// corpusSentinels are distinctive value tokens lifted from the shipped testdata
// fixtures. They are synthetic, fictitious values (see each testdata directory's
// README and LICENSE files for provenance), so treating them as PHI sentinels lets
// the sweep over the real corpus actually catch a value leak rather than scanning
// for tokens that are not present.
var corpusSentinels = []string{
	"MRN0001001",                    // PID-3 identifier in adt-a01.hl7
	"TESTPATIENT",                   // PID-5 family-name token across the HL7 corpus
	"100 FICTION ST",                // PID-11 street address in adt-a01.hl7
	"COMPREHENSIVE METABOLIC PANEL", // OBR-4 study text in oru-r01.hl7
}

// allSentinels is every token the main sweep scans for.
func allSentinels() []string {
	return append(append([]string{}, builtSentinels...), corpusSentinels...)
}

// testdataDir locates the repository test corpora relative to this package, which
// lives two levels below the module root.
const testdataDir = "../../testdata"

// buildSentinelDICOM constructs a minimal but valid Part 10 file whose dataset
// carries the synthetic PHI sentinels in the canonical PHI tags, writes it to a
// temp file, and returns the path. Exercising dicom.ReadFile against this file
// drives the real decode and dataset-access paths over known PHI.
func buildSentinelDICOM(t *testing.T) string {
	t.Helper()

	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagPatientName, sentinelPatientName)
	ds.SetString(dicom.TagPatientID, sentinelPatientID)
	ds.SetString(dicom.TagAccessionNumber, sentinelAccession)
	ds.SetString(dicom.TagPatientBirthDate, sentinelBirthDate)
	ds.SetString(dicom.TagStudyInstanceUID, "1.2.840.10008.3.1.2.3.4.99999")
	ds.SetString(dicom.TagSOPInstanceUID, "1.2.840.10008.3.1.2.3.4.99998")
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.7")
	ds.SetString(dicom.TagModality, "OT")

	file := &dicom.File{
		Meta: &dicom.FileMeta{
			MediaStorageSOPClassUID:    "1.2.840.10008.5.1.4.1.1.7",
			MediaStorageSOPInstanceUID: "1.2.840.10008.3.1.2.3.4.99998",
			TransferSyntaxUID:          dicom.ExplicitVRLittleEndian,
		},
		DataSet: ds,
	}

	path := filepath.Join(t.TempDir(), "sentinel.dcm")
	if err := dicom.WriteFile(path, file); err != nil {
		t.Fatalf("write sentinel dicom: %v", err)
	}
	return path
}

// sentinelHL7 is a synthetic ADT^A01 message carrying the PHI sentinels in their
// canonical segments and fields. It is a string literal, not a fixture file, so the
// tokens it contains are unmistakably test sentinels.
func sentinelHL7() []byte {
	const msg = "MSH|^~\\&|REGADT|HOSP|EMR|HOSP|20260101000000||ADT^A01^ADT_A01|MSGZZZ0001|P|2.5.1\r" +
		"EVN|A01|20260101000000\r" +
		"PID|1||" + sentinelPatientID + "^^^HOSP^MR||" + sentinelPatientName + "||" + sentinelBirthDate + "|M\r" +
		"PV1|1|I\r"
	return []byte(msg)
}

// exerciseDICOM drives representative DICOM entry points over sentinel-bearing and
// shipped fixtures: the Part 10 decoder, dataset value access, and a decode of a
// real vendored file. It returns every error it produced so their strings are
// scanned. It never prints; a leak can only arise from a returned error string or
// from a library call logging through the injected context.
func exerciseDICOM(ctx context.Context, sentinelPath string) []error {
	log := logging.FromContext(ctx)
	var errs []error

	f, err := dicom.ReadFile(sentinelPath)
	if err != nil {
		errs = append(errs, fmt.Errorf("read sentinel dicom: %w", err))
		return errs
	}
	log.Info("decoded part10 file", logging.DICOMTag(0x0010, 0x0010, "PatientName"))

	for _, tag := range []dicom.Tag{
		dicom.TagPatientName, dicom.TagPatientID,
		dicom.TagAccessionNumber, dicom.TagPatientBirthDate,
	} {
		if _, ok := f.DataSet.GetString(tag); !ok {
			errs = append(errs, fmt.Errorf("sentinel tag %v missing after round-trip", tag))
		}
	}
	for range f.DataSet.All() {
		// Iterating every element drives the dataset's element-access path; the
		// loop body intentionally reads nothing into a printable sink.
	}

	for _, name := range []string{"liver.dcm", "basic-text-sr.dcm"} {
		path := filepath.Join(testdataDir, "dicom", name)
		if _, statErr := os.Stat(path); statErr != nil {
			continue
		}
		if _, err := dicom.ReadFile(path); err != nil {
			errs = append(errs, fmt.Errorf("read corpus dicom %s: %w", name, err))
		}
	}
	return errs
}

// exerciseHL7 drives representative HL7 v2 entry points over a sentinel-bearing
// message and the shipped corpus: the message parser, segment and field accessors,
// and the round-trip marshaller. It returns every error it produced.
func exerciseHL7(ctx context.Context) []error {
	log := logging.FromContext(ctx)
	var errs []error

	msg, err := hl7v2.Parse(sentinelHL7())
	if err != nil {
		errs = append(errs, fmt.Errorf("parse sentinel hl7: %w", err))
		return errs
	}
	log.Info("parsed hl7 message", logging.HL7Field("PID", 5))

	for _, key := range []string{"PID-3", "PID-5", "PID-7", "MSH-9"} {
		if _, err := msg.Get(key); err != nil {
			errs = append(errs, fmt.Errorf("hl7 accessor %s: %w", key, err))
		}
	}
	if _, err := msg.MarshalText(); err != nil {
		errs = append(errs, fmt.Errorf("hl7 marshal: %w", err))
	}

	for _, name := range []string{"adt-a01.hl7", "oru-r01.hl7", "orm-o01.hl7"} {
		path := filepath.Join(testdataDir, "hl7v2", name)
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		corpusMsg, parseErr := hl7v2.Parse(raw)
		if parseErr != nil {
			errs = append(errs, fmt.Errorf("parse corpus hl7 %s: %w", name, parseErr))
			continue
		}
		// Drive the accessors over the real corpus so a value-leaking accessor
		// path would surface a corpus sentinel.
		_, _ = corpusMsg.Get("PID-5")
		_, _ = corpusMsg.MarshalText()
	}
	return errs
}

// exerciseFHIR drives the FHIR structural validator over resources seeded with PHI
// sentinels in their value fields, then surfaces the resulting OperationOutcome as both a
// returned error string and a logged message. r5.Validate builds its issue diagnostics
// and element paths from element names and codes, never field values, so a sentinel
// planted in a patient's name, identifier, or a free-text field must not appear in the
// outcome error string or the log. This is the FHIR half of the no-PHI guarantee, swept
// through the same machinery as the DICOM and HL7 paths.
func exerciseFHIR(ctx context.Context) []error {
	log := logging.FromContext(ctx)
	var errs []error

	// A Patient whose human-readable fields carry sentinels, plus an out-of-set gender
	// (a binding violation) and a direct two-branch choice write (a mutual-exclusion
	// violation), so the outcome reports issues over a resource full of PHI.
	bad := r5.AdministrativeGender(sentinelPatientName)
	deceasedBool := r5.FHIRBoolean(true)
	deceasedTime := r5.FHIRDateTime(sentinelBirthDate)
	patient := &r5.Patient{
		Gender:           &bad,
		DeceasedBoolean:  &deceasedBool,
		DeceasedDateTime: &deceasedTime,
	}

	oo := r5.Validate(patient)
	if err := oo.Error(); err != nil {
		// The outcome error string is scanned for any sentinel; a leak would surface here.
		errs = append(errs, fmt.Errorf("validate patient: %w", err))
	}
	log.Info("validated fhir patient", zap.Int("issue_count", len(oo.Issue)))

	// A Bundle with a dangling reference and a non-Composition first entry drives the
	// bdl-* and reference-integrity extra checks. The reference value is a neutral local
	// fragment id (an opaque element id, not patient data, per the documented rule that a
	// reference string is not PHI); the PHI sentinel stays in the free-text value field,
	// where a leak would actually be a leak.
	bt := r5.BundleTypeDocument
	obs := &r5.Observation{
		Code:    &r5.CodeableConcept{Text: stringPtr(sentinelPatientID)},
		Subject: &r5.Reference{Reference: stringPtr("#dangling-target")},
	}
	bundle := &r5.Bundle{
		Type:  &bt,
		Entry: []r5.BundleEntry{{Resource: resourceRef(obs)}},
	}
	if err := r5.Validate(bundle).Error(); err != nil {
		errs = append(errs, fmt.Errorf("validate bundle: %w", err))
	}
	return errs
}

// stringPtr boxes a string for an optional FHIR field in the sweep fixtures.
func stringPtr(s string) *string { return &s }

// resourceRef boxes a resource into the *fhir.Resource a BundleEntry carries.
func resourceRef(r fhir.Resource) *fhir.Resource { return &r }

// TestPHISanitySweep is the authoritative library-wide PHI-default sanity sweep
// (PRD §11.2). It exercises representative DICOM and HL7 v2 entry points at default
// verbosity over fixtures carrying known PHI sentinel tokens and fails if any token
// surfaces in stdout, stderr, a returned error string, or the structured log.
//
// It does not call t.Parallel: Run redirects the process-global standard streams,
// which must not overlap with another redirecting test.
func TestPHISanitySweep(t *testing.T) {
	sentinelPath := buildSentinelDICOM(t)

	cases := []struct {
		name     string
		exercise func(ctx context.Context) []error
	}{
		{"dicom", func(ctx context.Context) []error { return exerciseDICOM(ctx, sentinelPath) }},
		{"hl7v2", exerciseHL7},
		{"fhir", exerciseFHIR},
	}

	sentinels := allSentinels()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			capture, err := Run(tc.exercise)
			if err != nil {
				t.Fatalf("run sweep: %v", err)
			}
			if leaks := Scan(capture, sentinels); len(leaks) > 0 {
				for _, leak := range leaks {
					t.Errorf("PHI leak: %s", leak)
				}
			}
		})
	}
}

// TestSweepDetectsPlantedLeak proves the sweep bites: it runs an exercise closure
// rigged to surface a sentinel through each sink in turn and asserts the sweep
// detects the leak in exactly that sink. A gate that cannot fail is worthless, so
// this is the harness's own regression net. The leak is synthetic and contained to
// the closure.
func TestSweepDetectsPlantedLeak(t *testing.T) {
	const planted = sentinelPatientName

	cases := []struct {
		name     string
		want     Sink
		exercise func(ctx context.Context) []error
	}{
		{
			name: "stdout",
			want: SinkStdout,
			exercise: func(context.Context) []error {
				fmt.Fprintln(os.Stdout, "patient:", planted)
				return nil
			},
		},
		{
			name: "stderr",
			want: SinkStderr,
			exercise: func(context.Context) []error {
				fmt.Fprintln(os.Stderr, "patient:", planted)
				return nil
			},
		},
		{
			name: "error",
			want: SinkError,
			exercise: func(context.Context) []error {
				return []error{fmt.Errorf("failed for patient %s", planted)}
			},
		},
		{
			name: "log",
			want: SinkLog,
			exercise: func(ctx context.Context) []error {
				// Deliberately misroute a raw patient value into a zap value field,
				// the exact mistake the sweep exists to catch. FromContext hands
				// back a raw logger, so the no-PHI rule binds the caller; here the
				// caller breaks it on purpose so the sweep can prove it bites.
				logging.FromContext(ctx).Info("decode failed", zap.String("patient", planted))
				return nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			capture, err := Run(tc.exercise)
			if err != nil {
				t.Fatalf("run sweep: %v", err)
			}
			leaks := Scan(capture, builtSentinels)
			if len(leaks) == 0 {
				t.Fatalf("sweep failed to detect planted leak in %s", tc.want)
			}
			found := false
			for _, leak := range leaks {
				if leak.Sink == tc.want && leak.Sentinel == planted {
					found = true
				}
			}
			if !found {
				t.Fatalf("planted leak not reported in %s; got %v", tc.want, leaks)
			}
		})
	}
}

func TestScanReportsNothingWhenClean(t *testing.T) {
	clean := Capture{
		Stdout: "decoded (0010,0010)/PatientName",
		Stderr: "warning: PID-5 missing",
		Errors: []string{"hl7v2: parse error at offset 3: bad segment"},
		Logs:   `{"level":"info","msg":"parsed","hl7_field":{"locator":"PID-5"}}`,
	}
	if leaks := Scan(clean, allSentinels()); len(leaks) > 0 {
		t.Fatalf("clean capture reported leaks: %v", leaks)
	}
}

// TestSweepCapturesRawDescriptorWrite proves the redirection works at the file
// descriptor level, not merely by swapping the os.Stdout variable: a write straight
// to descriptor 1 — the path a cgo codec or a writer bound to the descriptor before
// the run would take — is still captured.
func TestSweepCapturesRawDescriptorWrite(t *testing.T) {
	capture, err := Run(func(context.Context) []error {
		_, _ = syscall.Write(syscall.Stdout, []byte("raw fd write: "+sentinelPatientID+"\n"))
		return nil
	})
	if err != nil {
		t.Fatalf("run sweep: %v", err)
	}
	leaks := Scan(capture, builtSentinels)
	if len(leaks) != 1 || leaks[0].Sink != SinkStdout || leaks[0].Sentinel != sentinelPatientID {
		t.Fatalf("raw descriptor write not captured on stdout; got %v", leaks)
	}
}

func TestScanMatchesEverySink(t *testing.T) {
	for _, id := range Sinks {
		t.Run(string(id), func(t *testing.T) {
			var c Capture
			switch id {
			case SinkStdout:
				c.Stdout = sentinelPatientID
			case SinkStderr:
				c.Stderr = sentinelPatientID
			case SinkError:
				c.Errors = []string{"boom: " + sentinelPatientID}
			case SinkLog:
				c.Logs = `{"v":"` + sentinelPatientID + `"}`
			}
			leaks := Scan(c, builtSentinels)
			if len(leaks) != 1 || leaks[0].Sink != id || leaks[0].Sentinel != sentinelPatientID {
				t.Fatalf("sink %s: got %v, want one leak of %q", id, leaks, sentinelPatientID)
			}
		})
	}
}
