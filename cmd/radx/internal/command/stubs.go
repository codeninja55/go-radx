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

// ModifyCmd edits DICOM tags and regenerates UIDs (dcmodify). Not implemented in this
// increment.
type ModifyCmd struct {
	Paths []string `arg:"" optional:"" name:"path" help:"DICOM files to edit."`
}

// Run fails closed.
func (c *ModifyCmd) Run(*RunContext) error { return notImplemented("modify") }

// OrganizeCmd reorganises files by UID structure. Not implemented in this increment.
type OrganizeCmd struct {
	Dir string `arg:"" optional:"" name:"dir" help:"Source directory."`
}

// Run fails closed.
func (c *OrganizeCmd) Run(*RunContext) error { return notImplemented("organize") }

// LookupCmd resolves DICOM dictionary information. Not implemented in this increment.
type LookupCmd struct {
	Query []string `arg:"" optional:"" name:"query" help:"Tag, keyword, or search fragment."`
}

// Run fails closed.
func (c *LookupCmd) Run(*RunContext) error { return notImplemented("lookup") }

// CatalogueCmd indexes and queries a local catalogue. Not implemented in this increment.
type CatalogueCmd struct {
	Dir string `arg:"" optional:"" name:"dir" help:"Directory to index."`
}

// Run fails closed.
func (c *CatalogueCmd) Run(*RunContext) error { return notImplemented("catalogue") }

// HL7Cmd is the HL7 v2 over MLLP group. Not implemented in this increment.
type HL7Cmd struct {
	Send   HL7SendCmd   `cmd:"" help:"Send a message and read the ACK."`
	Listen HL7ListenCmd `cmd:"" help:"Receive messages and reply with ACK/NAK."`
}

// HL7SendCmd sends an HL7 v2 message. Not implemented in this increment.
type HL7SendCmd struct{}

// Run fails closed.
func (c *HL7SendCmd) Run(*RunContext) error { return notImplemented("hl7 send") }

// HL7ListenCmd receives HL7 v2 messages. Not implemented in this increment.
type HL7ListenCmd struct{}

// Run fails closed.
func (c *HL7ListenCmd) Run(*RunContext) error { return notImplemented("hl7 listen") }

// DICOMwebCmd is the DICOMweb client group. Not implemented in this increment.
type DICOMwebCmd struct {
	Wado DICOMwebWadoCmd `cmd:"" help:"Retrieve via WADO-RS."`
	Stow DICOMwebStowCmd `cmd:"" help:"Store via STOW-RS."`
	Qido DICOMwebQidoCmd `cmd:"" help:"Search via QIDO-RS."`
}

// DICOMwebWadoCmd retrieves via WADO-RS. Not implemented in this increment.
type DICOMwebWadoCmd struct{}

// Run fails closed.
func (c *DICOMwebWadoCmd) Run(*RunContext) error { return notImplemented("dicomweb wado") }

// DICOMwebStowCmd stores via STOW-RS. Not implemented in this increment.
type DICOMwebStowCmd struct{}

// Run fails closed.
func (c *DICOMwebStowCmd) Run(*RunContext) error { return notImplemented("dicomweb stow") }

// DICOMwebQidoCmd searches via QIDO-RS. Not implemented in this increment.
type DICOMwebQidoCmd struct{}

// Run fails closed.
func (c *DICOMwebQidoCmd) Run(*RunContext) error { return notImplemented("dicomweb qido") }

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
