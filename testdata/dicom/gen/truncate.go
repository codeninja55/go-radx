// Command truncate writes the first N bytes of a valid DICOM fixture to a
// sibling <name>.truncated.dcm file. It supplies the deliberately truncated
// input for the DCM-003 "truncation is failure" regression (Increment 2)
// without committing a corrupt binary to the corpus.
//
// Usage:
//
//	go run ./testdata/dicom/gen -in liver.dcm -bytes 512
//
// -in is resolved relative to the testdata/dicom directory. The output is
// written next to the input as <stem>.truncated.dcm.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const fixtureDir = "testdata/dicom"

func main() {
	in := flag.String("in", "", "input fixture filename, relative to "+fixtureDir)
	n := flag.Int("bytes", 0, "number of leading bytes to keep (must be > 0)")
	flag.Parse()

	if err := run(*in, *n); err != nil {
		fmt.Fprintln(os.Stderr, "truncate:", err)
		os.Exit(1)
	}
}

func run(in string, n int) error {
	if in == "" {
		return fmt.Errorf("-in is required")
	}
	if n <= 0 {
		return fmt.Errorf("-bytes must be > 0, got %d", n)
	}
	// Reject path traversal: the input must be a plain filename within fixtureDir.
	if in != filepath.Base(in) {
		return fmt.Errorf("-in must be a bare filename, not a path: %q", in)
	}

	inPath := filepath.Join(fixtureDir, in)
	info, err := os.Stat(inPath)
	if err != nil {
		return fmt.Errorf("stat input: %w", err)
	}
	if int64(n) > info.Size() {
		return fmt.Errorf("-bytes %d exceeds input size %d", n, info.Size())
	}

	src, err := os.Open(inPath)
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer src.Close()

	stem := strings.TrimSuffix(in, filepath.Ext(in))
	outPath := filepath.Join(fixtureDir, stem+".truncated.dcm")

	dst, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer dst.Close()

	if _, err := io.CopyN(dst, src, int64(n)); err != nil {
		return fmt.Errorf("copy %d bytes: %w", n, err)
	}

	fmt.Printf("wrote %s (%d bytes)\n", outPath, n)
	return nil
}
