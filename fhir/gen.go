package fhir

// The FHIR release packages under fhir/r4 and fhir/r5 are produced by the
// build-time generator in fhir/internal/gen from the pinned, checksum-verified
// StructureDefinition bundles vendored under internal/gen/testdata/definitions.
// Once the generator pipeline lands (M6 Increments 1+), `go generate ./fhir/...`
// reproduces the committed generated output byte-for-byte; the scaffold entry
// point reports that the pipeline is not yet implemented. The generator is never
// part of the runtime dependency graph.

//go:generate go run ./internal/gen/cmd/fhir-gen -release r5
