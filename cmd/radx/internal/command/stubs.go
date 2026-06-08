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

// ServeFHIRCmd serves the FHIR REST API. The FHIR server role is a separate concurrent increment,
// so this stays fail-closed: it returns a typed not-implemented error (exit 1) rather than a stub
// that reports success (docs/reference/cli.md serve, the serve-fhir deferral). The serve dicomweb
// daemon and the ServeCmd group live in serve.go.
type ServeFHIRCmd struct{}

// Run fails closed.
func (c *ServeFHIRCmd) Run(*RunContext) error { return notImplemented("serve fhir") }
