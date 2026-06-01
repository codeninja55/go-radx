// Package server is go-radx's composition layer for the receiving side of the
// radiology workflow. The four server roles — the DIMSE SCP (including a Modality
// Worklist SCP), the DICOMweb server, the FHIR REST server, and the HL7 v2 MLLP
// server — expose their own embeddable handlers in the dimse, dicomweb, fhir, and
// hl7v2 packages; this package adds the cross-cutting glue every deployment needs
// and no single protocol package should own: small pluggable backends (object
// store, catalogue, authenticator), shared observability wiring (structured zap
// logging and OpenTelemetry, never carrying PHI), shared listener policy (loopback
// bind by default with an explicit opt-in), and shared graceful-shutdown
// lifecycle. On top of those primitives it ships thin reference daemons that wire
// sane defaults so the radx CLI and a first-time user get something runnable
// immediately.
//
// See docs/reference/servers.md for the public API.
package server
