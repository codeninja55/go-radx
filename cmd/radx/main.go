// Command radx is the go-radx command-line interface for DICOM, HL7 v2, and FHIR.
//
// radx lives in its own Go module so that consumers importing the library packages do
// not inherit the CLI's dependency graph. The command surface (echo, store, scp, dump,
// modify, organize, lookup, catalogue, plus the hl7, dicomweb, and convert groups) is
// implemented in a later milestone; see docs/reference/cli.md for the planned design.
package main

import "fmt"

func main() {
	fmt.Println("radx: command-line interface — not yet implemented (see docs/reference/cli.md)")
}
