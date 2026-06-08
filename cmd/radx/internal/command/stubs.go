package command

import "github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"

// The commands below are committed surface (docs/reference/cli.md "Command tree") that this
// increment does not implement. Each is registered so the command tree is stable and `radx
// --help` lists the full surface, but every one fails closed: its Run returns a typed
// *exitcode.NotImplementedError, which classifies to exit 1 and writes no output. A stub never
// no-ops and reports success — that is the prototype defect the honest-failure rules exist to
// prevent (RADX-001/002). When a command lands it replaces its stub with a real Run.

// notImplemented builds the fail-closed error for capability.
func notImplemented(capability string) error {
	return &exitcode.NotImplementedError{Capability: capability}
}

// ConvertCmd is the cross-standard conversion group. Not implemented in this increment.
type ConvertCmd struct {
	DICOMToFHIR ConvertDICOMToFHIRCmd `cmd:"" name:"dicom-to-fhir" help:"Build a FHIR ImagingStudy from DICOM."`
	SRToFHIR    ConvertSRToFHIRCmd    `cmd:"" name:"sr-to-fhir" help:"Map a DICOM SR to a DiagnosticReport."`
	ORUToFHIR   ConvertORUToFHIRCmd   `cmd:"" name:"oru-to-fhir" help:"Map an HL7 v2 ORU to a DiagnosticReport."`
	ORMToFHIR   ConvertORMToFHIRCmd   `cmd:"" name:"orm-to-fhir" help:"Map an HL7 v2 ORM to a ServiceRequest."`
	ADTToFHIR   ConvertADTToFHIRCmd   `cmd:"" name:"adt-to-fhir" help:"Map an HL7 v2 ADT to a Patient / Encounter."`
}

// ConvertDICOMToFHIRCmd builds a FHIR ImagingStudy. Not implemented in this increment.
type ConvertDICOMToFHIRCmd struct{}

// Run fails closed.
func (c *ConvertDICOMToFHIRCmd) Run(*RunContext) error {
	return notImplemented("convert dicom-to-fhir")
}

// ConvertSRToFHIRCmd maps a DICOM SR. Not implemented in this increment.
type ConvertSRToFHIRCmd struct{}

// Run fails closed.
func (c *ConvertSRToFHIRCmd) Run(*RunContext) error { return notImplemented("convert sr-to-fhir") }

// ConvertORUToFHIRCmd maps an HL7 v2 ORU. Not implemented in this increment.
type ConvertORUToFHIRCmd struct{}

// Run fails closed.
func (c *ConvertORUToFHIRCmd) Run(*RunContext) error { return notImplemented("convert oru-to-fhir") }

// ConvertORMToFHIRCmd maps an HL7 v2 ORM. Not implemented in this increment.
type ConvertORMToFHIRCmd struct{}

// Run fails closed.
func (c *ConvertORMToFHIRCmd) Run(*RunContext) error { return notImplemented("convert orm-to-fhir") }

// ConvertADTToFHIRCmd maps an HL7 v2 ADT. Not implemented in this increment.
type ConvertADTToFHIRCmd struct{}

// Run fails closed.
func (c *ConvertADTToFHIRCmd) Run(*RunContext) error { return notImplemented("convert adt-to-fhir") }

// ServeCmd is the reference-daemon group. Not implemented in this increment.
type ServeCmd struct {
	DICOMweb ServeDICOMwebCmd `cmd:"" name:"dicomweb" help:"Serve WADO-RS / STOW-RS / QIDO-RS."`
	FHIR     ServeFHIRCmd     `cmd:"" name:"fhir" help:"Serve the FHIR REST API."`
}

// ServeDICOMwebCmd serves DICOMweb. Not implemented in this increment.
type ServeDICOMwebCmd struct{}

// Run fails closed.
func (c *ServeDICOMwebCmd) Run(*RunContext) error { return notImplemented("serve dicomweb") }

// ServeFHIRCmd serves the FHIR REST API. Not implemented in this increment.
type ServeFHIRCmd struct{}

// Run fails closed.
func (c *ServeFHIRCmd) Run(*RunContext) error { return notImplemented("serve fhir") }
