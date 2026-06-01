// Package pdu implements the DICOM Upper Layer Protocol Data Units (PS3.8 §9.3):
// the 6-byte PDU header framing, the P-DATA-TF PDU and its Presentation Data
// Values, and the association/release/abort PDUs. It owns no socket and knows
// nothing about DICOM messages — only PDU bytes (dimse.md "Overview of the
// layers"). All length math is bounds-checked before allocation (PRD §9.3).
package pdu
