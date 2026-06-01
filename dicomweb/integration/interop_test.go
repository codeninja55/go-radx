//go:build interop

// Package integration holds the DICOMweb interop regression net: a STOW-RS store
// followed by a WADO-RS retrieve against a real Orthanc container, proving the
// round-trip end-to-end.
//
// The committed dicomweb client/server API does not exist until the DICOMweb leg
// (Increments 8-9), so every test here is a skipped stub that references no
// unbuilt symbol and compiles standalone under the interop tag; the real port
// against the committed API lands in Increment 9.
package integration

import "testing"

// TestInteropStowThenWadoOrthanc will STOW a vendored .dcm to an Orthanc container
// and then WADO-RS retrieve it, asserting the dataset round-trips.
func TestInteropStowThenWadoOrthanc(t *testing.T) {
	t.Skip("DICOMweb API lands in Increments 8-9; interop test activated in Increment 9")
}
