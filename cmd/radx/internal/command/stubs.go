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
