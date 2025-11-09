package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/codeninja55/go-radx/fhir/scripts/gen/codegen"
	"github.com/codeninja55/go-radx/fhir/scripts/gen/parser"
)

const (
	versionR4 = "r4"
	versionR5 = "r5"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Parse command line flags
	var (
		version   = flag.String("version", versionR4, "FHIR version (r4 or r5)")
		outputDir = flag.String("output", "", "Output directory for generated code")
		inputFile = flag.String("input", "", "Input StructureDefinitions file (profiles-resources.json)")
		resources = flag.String("resources", "", "Comma-separated list of specific resources to generate (e.g., 'Patient,Observation'). If empty, generates all resources.")
		verbose   = flag.Bool("verbose", false, "Enable verbose output")
	)
	flag.Parse()

	// Validate flags
	if *version != versionR4 && *version != versionR5 {
		return fmt.Errorf("invalid version: %s (must be r4 or r5)", *version)
	}

	if *outputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	// Determine input file path
	inputPath := *inputFile
	if inputPath == "" {
		// Default to profiles-resources.json in project root
		inputPath = filepath.Join("fhir_schemas", "profiles-resources.json")
	}

	if *verbose {
		fmt.Printf("FHIR version: %s\n", *version)
		fmt.Printf("Input file: %s\n", inputPath)
		fmt.Printf("Output directory: %s\n", *outputDir)
	}

	// Create parser and load definitions
	p := parser.New()
	if err := p.ParseFile(inputPath); err != nil {
		return fmt.Errorf("parse file: %w", err)
	}

	if *verbose {
		resources := p.GetResources()
		complexTypes := p.GetComplexTypes()
		fmt.Printf("Loaded %d resources and %d complex types\n", len(resources), len(complexTypes))
	}

	// Parse resource filter
	var resourceFilter []string
	if *resources != "" {
		resourceFilter = strings.Split(*resources, ",")
		// Trim whitespace from each resource name
		for i := range resourceFilter {
			resourceFilter[i] = strings.TrimSpace(resourceFilter[i])
		}
		if *verbose {
			fmt.Printf("Filtering to specific resources: %v\n", resourceFilter)
		}
	}

	// Create builder with appropriate package name
	packageName := "resources"
	builder := codegen.NewBuilder(p, packageName, *verbose)

	// Set resource filter if provided
	if len(resourceFilter) > 0 {
		builder.SetResourceFilter(resourceFilter)
	}

	// Generate all types
	files, err := builder.BuildAll()
	if err != nil {
		return fmt.Errorf("build all types: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	// Write generated files
	for filename, content := range files {
		outputPath := filepath.Join(*outputDir, filename)
		if *verbose {
			fmt.Printf("Writing %s\n", outputPath)
		}

		if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write file %s: %w", outputPath, err)
		}
	}

	fmt.Printf("Successfully generated %d files in %s\n", len(files), *outputDir)
	return nil
}
