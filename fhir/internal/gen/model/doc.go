// Package model is the FHIR generator's model / intermediate representation: it
// turns the loader's flat list of dotted-path ElementDefinitions into the explicit
// element-path tree the later pipeline stages recurse over.
//
// The loader hands the model a StructureDefinition whose snapshot is a flat slice
// of elements keyed by dotted FHIR paths (for example "Observation",
// "Observation.component", "Observation.component.referenceRange",
// "Observation.referenceRange.low"). BuildType nests those paths into a tree so a
// backbone element carries its real child elements rather than an empty stub. It
// resolves the contentReference indirection FHIR uses for recursive or shared
// backbone shapes, so a node that reuses another element's structure
// (Observation.component.referenceRange points at #Observation.referenceRange) is
// populated with that structure's children instead of being left a leaf. This
// recursion is the structural fix for empty backbone structs.
//
// The model is the release-agnostic IR: it records only what the loaded bundle
// describes (cardinality, type set, binding strength and value-set reference,
// summary and modifier flags, choice [x] grouping, and the resolved
// contentReference) and makes no assumption about R4 versus R5. It maps no FHIR
// type to a Go type and decides no Go name; those are the planner's job. The model
// produces no Go source. Its only output is the in-memory tree, validated by a
// golden snapshot.
package model
