package fhir

// The FHIR release packages under fhir/r4 and fhir/r5 are produced by the
// build-time generator in fhir/internal/gen from the pinned, checksum-verified
// StructureDefinition bundles vendored under internal/gen/testdata/definitions.
// Running `go generate ./fhir/...` reproduces the committed generated output
// byte-for-byte; a committed generated file is never hand-edited, and the
// regeneration gate (TestRegenerationByteForByte / the gen:verify task) fails if
// one is. The generator is never part of the runtime dependency graph.

//go:generate go run ./internal/gen/cmd/fhir-gen -release r4
//go:generate go run ./internal/gen/cmd/fhir-gen -release r5
