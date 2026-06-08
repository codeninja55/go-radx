package command

import (
	"encoding/json"
	"io"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/convert"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/hl7v2"
)

// releaseR4 and releaseR5 are the --release enum values. R5 is the default (the PRD's R5-first
// sequencing); R4 selects the release-explicit …R4 converter twin.
const (
	releaseR4 = "R4"
	releaseR5 = "R5"
)

// ConvertCmd groups the cross-standard conversions, each driving the convert package's
// release-explicit converters. --release selects FHIR R4 (4.0.1) or R5 (5.0.0); the release is not
// cosmetic — it picks the converter twin (docs/reference/cli.md convert).
type ConvertCmd struct {
	DICOMToFHIR ConvertDICOMToFHIRCmd `cmd:"" name:"dicom-to-fhir" help:"Build a FHIR ImagingStudy from DICOM."`
	SRToFHIR    ConvertSRToFHIRCmd    `cmd:"" name:"sr-to-fhir" help:"Map a DICOM SR to a DiagnosticReport."`
	ORUToFHIR   ConvertORUToFHIRCmd   `cmd:"" name:"oru-to-fhir" help:"Map an HL7 v2 ORU to a DiagnosticReport."`
	ORMToFHIR   ConvertORMToFHIRCmd   `cmd:"" name:"orm-to-fhir" help:"Map an HL7 v2 ORM to a ServiceRequest."`
	ADTToFHIR   ConvertADTToFHIRCmd   `cmd:"" name:"adt-to-fhir" help:"Map an HL7 v2 ADT to a Patient / Encounter."`
}

// emitFHIR serialises a FHIR resource (or a small bundle of resources) as indented JSON to the
// machine sink. csv is a usage error for the conversions (they emit FHIR resources, not tables).
// The conversion report is not part of the resource payload, so a json consumer gets a clean FHIR
// resource on stdout; the report's loss diagnostics are summarised on stderr by the caller.
func emitFHIR(rc *RunContext, resource any) error {
	if rc.Out.Format == cli.FormatCSV {
		return &exitcode.UsageErr{Message: "convert does not support --format csv; it emits FHIR resources"}
	}
	enc := json.NewEncoder(rc.Out.Machine)
	enc.SetIndent("", "  ")
	return enc.Encode(resource)
}

// readDICOMInstances reads one or more DICOM files into datasets for the DICOM-sourced conversions.
// A read or parse failure is fail-closed: a malformed input never produces a lossy resource.
func readDICOMInstances(paths []string, recursive bool) ([]*dicom.DataSet, error) {
	files, err := resolveDICOMPaths(paths, recursive)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, &exitcode.UsageErr{Message: "no DICOM files to convert"}
	}
	datasets := make([]*dicom.DataSet, 0, len(files))
	for _, path := range files {
		f, readErr := dicom.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		datasets = append(datasets, f.DataSet)
	}
	return datasets, nil
}

// readHL7Message reads and parses an HL7 v2 message from a file or stdin for the HL7-sourced
// conversions. A parse failure is a parse error (exit 3).
func readHL7Message(path string, stdin io.Reader) (*hl7v2.Message, error) {
	raw, err := readMessageInput(path, stdin)
	if err != nil {
		return nil, err
	}
	return hl7v2.Parse(raw)
}

// ConvertDICOMToFHIRCmd builds a FHIR ImagingStudy from one or more DICOM instances.
type ConvertDICOMToFHIRCmd struct {
	Paths []string `arg:"" name:"path" help:"DICOM files to convert."`

	Release   string `name:"release" enum:"R4,R5" default:"R5" help:"FHIR release: R4 (4.0.1) or R5 (5.0.0)."`
	Recursive bool   `short:"R" name:"recursive" help:"Descend into directories for *.dcm files."`
}

// Run reads the instances and emits the release-appropriate ImagingStudy. A conversion that cannot
// faithfully map a required element returns an error and exits 3 rather than emitting a lossy
// resource (fail-closed, PRD §9.2).
func (c *ConvertDICOMToFHIRCmd) Run(rc *RunContext) error {
	instances, err := readDICOMInstances(c.Paths, c.Recursive)
	if err != nil {
		return err
	}
	if c.Release == releaseR4 {
		study, _, convErr := convert.DICOMToImagingStudyR4(instances)
		if convErr != nil {
			return convErr
		}
		return emitFHIR(rc, study)
	}
	study, _, convErr := convert.DICOMToImagingStudyR5(instances)
	if convErr != nil {
		return convErr
	}
	return emitFHIR(rc, study)
}

// ConvertSRToFHIRCmd maps a DICOM Structured Report to a DiagnosticReport and its Observations.
type ConvertSRToFHIRCmd struct {
	Path string `arg:"" name:"path" help:"DICOM SR file to convert."`

	Release string `name:"release" enum:"R4,R5" default:"R5" help:"FHIR release: R4 (4.0.1) or R5 (5.0.0)."`
}

// srBundle pairs the DiagnosticReport with its derived Observations so a single conversion emits
// one self-describing JSON document rather than a bare report that drops its observations.
type srBundle struct {
	DiagnosticReport any   `json:"diagnosticReport"`
	Observations     []any `json:"observations"`
}

// Run reads the SR dataset and emits the DiagnosticReport plus Observations for the release.
func (c *ConvertSRToFHIRCmd) Run(rc *RunContext) error {
	f, err := dicom.ReadFile(c.Path)
	if err != nil {
		return err
	}
	if c.Release == releaseR4 {
		report, observations, _, convErr := convert.SRToDiagnosticReportR4(f.DataSet)
		if convErr != nil {
			return convErr
		}
		return emitFHIR(rc, srBundle{DiagnosticReport: report, Observations: toAnySlice(observations)})
	}
	report, observations, _, convErr := convert.SRToDiagnosticReportR5(f.DataSet)
	if convErr != nil {
		return convErr
	}
	return emitFHIR(rc, srBundle{DiagnosticReport: report, Observations: toAnySlice(observations)})
}

// ConvertORUToFHIRCmd maps an HL7 v2 ORU to a DiagnosticReport and its Observations.
type ConvertORUToFHIRCmd struct {
	File string `arg:"" optional:"" name:"file" help:"ORU message file (omit or '-' to read stdin)."`

	Release string `name:"release" enum:"R4,R5" default:"R5" help:"FHIR release: R4 (4.0.1) or R5 (5.0.0)."`
}

// Run reads the ORU message and emits the DiagnosticReport plus Observations for the release.
func (c *ConvertORUToFHIRCmd) Run(rc *RunContext) error {
	msg, err := readHL7Message(c.File, rc.Stdin)
	if err != nil {
		return err
	}
	if c.Release == releaseR4 {
		report, observations, _, convErr := convert.ORUToDiagnosticReportR4(msg)
		if convErr != nil {
			return convErr
		}
		return emitFHIR(rc, srBundle{DiagnosticReport: report, Observations: toAnySlice(observations)})
	}
	report, observations, _, convErr := convert.ORUToDiagnosticReportR5(msg)
	if convErr != nil {
		return convErr
	}
	return emitFHIR(rc, srBundle{DiagnosticReport: report, Observations: toAnySlice(observations)})
}

// ConvertORMToFHIRCmd maps an HL7 v2 ORM to a ServiceRequest.
type ConvertORMToFHIRCmd struct {
	File string `arg:"" optional:"" name:"file" help:"ORM message file (omit or '-' to read stdin)."`

	Release string `name:"release" enum:"R4,R5" default:"R5" help:"FHIR release: R4 (4.0.1) or R5 (5.0.0)."`
}

// Run reads the ORM message and emits the ServiceRequest for the release.
func (c *ConvertORMToFHIRCmd) Run(rc *RunContext) error {
	msg, err := readHL7Message(c.File, rc.Stdin)
	if err != nil {
		return err
	}
	if c.Release == releaseR4 {
		req, _, convErr := convert.ORMToServiceRequestR4(msg)
		if convErr != nil {
			return convErr
		}
		return emitFHIR(rc, req)
	}
	req, _, convErr := convert.ORMToServiceRequestR5(msg)
	if convErr != nil {
		return convErr
	}
	return emitFHIR(rc, req)
}

// ConvertADTToFHIRCmd maps an HL7 v2 ADT to a Patient or an Encounter.
type ConvertADTToFHIRCmd struct {
	File string `arg:"" optional:"" name:"file" help:"ADT message file (omit or '-' to read stdin)."`

	Release string `name:"release" enum:"R4,R5" default:"R5" help:"FHIR release: R4 (4.0.1) or R5 (5.0.0)."`
	As      string `name:"as" enum:"patient,encounter" default:"patient" help:"Map to a Patient or an Encounter."`
}

// Run reads the ADT message and emits a Patient or an Encounter for the release, per --as.
func (c *ConvertADTToFHIRCmd) Run(rc *RunContext) error {
	msg, err := readHL7Message(c.File, rc.Stdin)
	if err != nil {
		return err
	}
	encounter := c.As == "encounter"
	switch {
	case c.Release == releaseR4 && encounter:
		res, _, convErr := convert.ADTToEncounterR4(msg)
		if convErr != nil {
			return convErr
		}
		return emitFHIR(rc, res)
	case c.Release == releaseR4:
		res, _, convErr := convert.ADTToPatientR4(msg)
		if convErr != nil {
			return convErr
		}
		return emitFHIR(rc, res)
	case encounter:
		res, _, convErr := convert.ADTToEncounterR5(msg)
		if convErr != nil {
			return convErr
		}
		return emitFHIR(rc, res)
	default:
		res, _, convErr := convert.ADTToPatientR5(msg)
		if convErr != nil {
			return convErr
		}
		return emitFHIR(rc, res)
	}
}

// toAnySlice widens a typed slice of FHIR observations to []any so the SR/ORU bundle can carry
// either release's Observation type without a generic constraint.
func toAnySlice[T any](in []T) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}
