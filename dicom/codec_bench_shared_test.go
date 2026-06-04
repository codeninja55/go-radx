//go:build cgo && (dicom_openjpeg || dicom_libjpeg || dicom_charls)

package dicom

import (
	"path/filepath"
	"testing"
)

// readPixelDataBench reads the named fixture's pixel data for a benchmark. Every CGo codec
// benchmark uses it, so it is compiled when at least one codec build tag is set.
func readPixelDataBench(b *testing.B, name string) *PixelData {
	b.Helper()
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", name))
	if err != nil {
		b.Fatalf("ReadPixelData %s: %v", name, err)
	}
	return pd
}
