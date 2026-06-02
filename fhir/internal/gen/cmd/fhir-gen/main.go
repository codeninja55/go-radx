// Command fhir-gen is the thin CLI wrapper around the build-time FHIR generator.
// It parses flags, builds a gen.Config, and calls gen.Generate. It is invoked by
// the go:generate directive in fhir/gen.go and is never part of the runtime
// dependency graph. Being a build tool, it logs plainly to stderr.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/codeninja55/go-radx/fhir/internal/gen"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "fhir-gen: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("fhir-gen", flag.ContinueOnError)
	release := fs.String("release", "", "FHIR release to generate (r4 or r5)")
	definitions := fs.String("definitions", "", "directory holding the vendored, checksum-pinned definition bundle")
	output := fs.String("output", "", "release package directory to write generated Go into")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg := gen.Config{
		Release:        *release,
		DefinitionsDir: defaultDefinitionsDir(*definitions, *release),
		OutputDir:      defaultOutputDir(*output, *release),
	}
	return gen.Generate(cfg)
}

// defaultDefinitionsDir resolves the vendored bundle path from the release when
// the caller does not pass an explicit -definitions flag, so the go:generate
// directive stays terse.
func defaultDefinitionsDir(dir, release string) string {
	if dir != "" {
		return dir
	}
	if release == "" {
		return ""
	}
	return "internal/gen/testdata/definitions/" + release
}

// defaultOutputDir resolves the output package from the release when the caller
// does not pass an explicit -output flag.
func defaultOutputDir(dir, release string) string {
	if dir != "" {
		return dir
	}
	return release
}
