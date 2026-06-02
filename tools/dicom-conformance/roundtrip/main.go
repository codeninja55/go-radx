// Command roundtrip reads a DICOM Part 10 file with go-radx and writes it back out, so the
// conformance gate can validate go-radx's WRITER output rather than the borrowed input corpus. It
// exits 0 on a successful round-trip, 2 when go-radx cannot read the input, and 3 when go-radx
// cannot write it (e.g. an encapsulated transfer syntax the writer does not support). The gate
// skips the inputs that exit 2 or 3.
package main

import (
	"fmt"
	"os"

	"github.com/codeninja55/go-radx/dicom"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: roundtrip <input.dcm> <output.dcm>")
		os.Exit(64)
	}
	in, out := os.Args[1], os.Args[2]

	f, err := dicom.ReadFile(in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", in, err)
		os.Exit(2)
	}
	if err := dicom.WriteFile(out, f); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", out, err)
		os.Exit(3)
	}
}
