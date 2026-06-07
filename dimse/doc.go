// Package dimse implements the DICOM Message Service Element and its transport
// layer: the DICOM Upper Layer protocol (DUL, PS3.8) and the DIMSE-C / DIMSE-N
// services (PS3.7). It is the network plane of go-radx, carrying DICOM datasets
// defined by the dicom package between Application Entities over plaintext TCP
// (transport-layer TLS is not yet implemented). The package is built in three
// stacked layers — pdu (the PDU/PDV
// wire codec), dul (the PS3.8 state machine and socket owner), and acse
// (association negotiation) — over which this root package builds the DIMSE
// message layer (command-set build/parse and PDV fragmentation) and the typed
// C/N service operations as both Service Class User and Service Class Provider.
// It reuses dicom.TransferSyntax, dicom.UID, dicom.DataSet, and the SOP UID types
// rather than redeclaring them.
//
// See docs/reference/dimse.md for the public API and docs/conformance/dicom.md for
// the supported SOP classes and negotiation features.
//
// Stability: experimental. Pre-1.0; the API may change between v0.x releases.
package dimse
