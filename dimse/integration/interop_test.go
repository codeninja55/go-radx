//go:build interop

// Package integration holds the DIMSE interop regression net: C-ECHO and C-STORE
// driven as SCU and SCP against a real Orthanc (and dcm4chee-arc) container, the
// gate that proves the prototype's last-fragment-bit defect is fixed end-to-end.
//
// It is ported first, before the DIMSE rewrite, so the interop gate exists ahead
// of the work it guards (PRD §12). The committed dimse.AE/Associate/Echo/Store API
// does not exist until Increments 1–6, so every test here is a skipped stub that
// compiles standalone under the interop tag; the real port against the committed
// API lands in Increment 7.
package integration

import "testing"

// TestInteropCEchoOrthanc will drive an SCU C-ECHO against an Orthanc container and
// assert the returned Status is the success-class verification status.
func TestInteropCEchoOrthanc(t *testing.T) {
	t.Skip("DIMSE API lands in Increments 1-6; interop test activated in Increment 7")
}

// TestInteropCStoreOrthanc will store a vendored .dcm to an Orthanc container and
// verify it via the REST API; it is the named regression test for the
// last-fragment-bit fix (Codex DIMSE-001).
func TestInteropCStoreOrthanc(t *testing.T) {
	t.Skip("DIMSE API lands in Increments 1-6; interop test activated in Increment 7")
}

// TestInteropSCPReceivesCStore will have a reference SCU store to the go-radx
// dimse.Server, proving the SCP receive path.
func TestInteropSCPReceivesCStore(t *testing.T) {
	t.Skip("DIMSE API lands in Increments 1-6; interop test activated in Increment 7")
}
