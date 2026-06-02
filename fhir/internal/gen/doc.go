// Package gen is the build-time FHIR resource code generator. It reads the
// pinned, checksum-verified HL7 FHIR StructureDefinition bundles vendored under
// testdata/definitions/<release> and emits type-safe Go into the fhir/r4 and
// fhir/r5 release packages. It is invoked by `go generate` (see fhir/gen.go) and
// is never part of the runtime dependency graph: consumers import the generated
// packages, never this generator. The generated output is reproducible
// byte-for-byte from the pinned input and is never hand-edited.
//
// The pipeline is staged with single-responsibility boundaries: a loader reads
// and checksum-verifies the bundle, a model builds the element-path tree, a
// planner decides Go names and types, and an emitter renders gofmt-stable Go.
// Later increments fill these stages in; this scaffold defines the entry point
// and the configuration the CLI passes.
package gen
