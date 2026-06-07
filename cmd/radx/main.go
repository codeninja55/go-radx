// Command radx is the go-radx command-line interface for DICOM, HL7 v2, and FHIR.
//
// radx lives in its own Go module so that consumers importing the library packages do
// not inherit the CLI's Kong parser and terminal-UI dependency graph. The shared output
// contract (clean machine stdout, diagnostics to stderr), the 12-factor RADX_* environment
// configuration, the exit-code taxonomy, and the honest-failure rules are described in
// docs/reference/cli.md and implemented under internal/.
package main

import (
	"os"

	"github.com/codeninja55/go-radx/cmd/radx/internal/command"
)

func main() {
	os.Exit(command.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
