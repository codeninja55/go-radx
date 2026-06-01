// Package dul implements the DICOM Upper Layer (PS3.8) finite state machine: the
// Table 9-10 lifecycle of 13 states (including release-collision Sta9–Sta12), 19
// events (Evt1–Evt19), and 28 actions (AE/DT/AR/AA). It owns the socket and the
// ARTIM timer, frames the association, release, and abort PDUs, and knows only
// PDUs — never DICOM messages (dimse.md "Overview of the layers"). It depends on
// the pdu package for the wire codec and never on acse or the root dimse package,
// keeping the layering acyclic.
package dul
