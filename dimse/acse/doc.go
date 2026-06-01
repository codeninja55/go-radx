// Package acse implements the DICOM association control layer: it negotiates the
// association (A-ASSOCIATE), tracks the accepted presentation contexts, and drives
// the dul state machine with A-ASSOCIATE, A-RELEASE, and A-ABORT primitives
// (dimse.md "Overview of the layers"). It sits above dul and below the DIMSE
// message layer in the root dimse package, depending on dul and pdu but never on
// the root dimse package, keeping the layering acyclic.
package acse
