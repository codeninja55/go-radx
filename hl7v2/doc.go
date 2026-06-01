// Package hl7v2 parses, constructs, and exchanges HL7 v2.x messages. It provides a
// generic six-level parse tree — message, segment, field, repetition, component,
// subcomponent — that round-trips any conformant message byte-for-byte, and a
// primary typed layer of segments (MSH, EVN, PID, PV1, OBR, OBX, ORC, MSA, ERR)
// and message types (ADT, ORM, ORU, ACK) backed by typed composite datatypes, so
// callers read msg.PID().PatientName.Family rather than seg[5][0][0]. It also
// covers encoding-character derivation, Chapter 2 escape handling, batch and file
// framing, and a Minimal Lower Layer Protocol (MLLP) client and server.
//
// See docs/reference/hl7v2.md for the public API and docs/conformance/hl7v2.md for
// the supported message types.
package hl7v2
