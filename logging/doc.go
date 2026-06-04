// Package logging is go-radx's structured-logging composition point. It wraps
// go.uber.org/zap and is built around two non-negotiable rules from the PRD
// (§9.10 Observability, §9.1 PHI safety):
//
//   - The logger is constructed once at the application's composition root and
//     flows through call chains via context.Context. There is no package-global
//     logger; WithContext attaches one and FromContext retrieves it, returning a
//     safe no-op logger when none is present so library code never panics on a
//     bare context.
//   - No Protected Health Information (PHI) is logged through the field helpers
//     this package provides. They render DICOM, HL7 v2, and FHIR concepts *by
//     name* — a DICOM tag keyword, an HL7 segment-and-field locator, a FHIR
//     element path — never by value. They take structural identifiers, not
//     patient data, so the API refuses raw patient values by construction: there
//     is deliberately no helper that logs an element's value. FromContext returns
//     a raw *zap.Logger, so the no-PHI rule still binds the caller — log structure
//     through the helpers and never pass a patient value to zap.String or a
//     sibling field.
//
// PHI governance beyond these safe defaults — encryption at rest, retention,
// access control, audit — belongs to the consumer that integrates go-radx, per
// PRD §9.1. This package provides the safe default and the structural field
// vocabulary, not a compliance regime.
//
// See docs/conformance/cli-server.md for the operator-facing logging contract.
package logging
