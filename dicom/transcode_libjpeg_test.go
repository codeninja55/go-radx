//go:build cgo && dicom_libjpeg

package dicom

import (
	"bytes"
	"path/filepath"
	"testing"
)

// TestSetPixelDataColorJPEGDecompressReconcilesAttributes is the colour acceptance
// for the decompress seam: pulling a lossy colour JPEG through NewPixelData ->
// Transcode -> SetPixelData must leave the Image Pixel attributes consistent with
// the decoded bytes (interleaved RGB) and the lossy bookkeeping in place, and the
// written native object must decode to exactly the frames the original decodes to.
func TestSetPixelDataColorJPEGDecompressReconcilesAttributes(t *testing.T) {
	path := filepath.Join("..", "testdata", "dicom", "SC_jpeg_no_color_transform.dcm")
	orig, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	origPD, err := NewPixelData(orig.DataSet, orig.Meta.TransferSyntaxUID)
	if err != nil {
		t.Fatalf("NewPixelData(original): %v", err)
	}
	wantFrames := collectFrames(t, origPD)

	f, err := ReadFile(path)
	if err != nil {
		t.Fatalf("re-ReadFile: %v", err)
	}
	// Drop the stored lossy flag so the test proves SetPixelData sets it, not merely
	// that the fixture already carried it.
	f.DataSet.Delete(TagLossyImageCompression)
	pd, err := NewPixelData(f.DataSet, f.Meta.TransferSyntaxUID)
	if err != nil {
		t.Fatalf("NewPixelData: %v", err)
	}
	native, err := Transcode(pd, ExplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("Transcode: %v", err)
	}
	if err := f.SetPixelData(native); err != nil {
		t.Fatalf("SetPixelData: %v", err)
	}

	if pi, _ := f.DataSet.GetString(TagPhotometricInterpretation); pi != "RGB" {
		t.Errorf("PhotometricInterpretation = %q, want RGB", pi)
	}
	if pc, ok := f.DataSet.GetInt(TagPlanarConfiguration); !ok || pc != 0 {
		t.Errorf("PlanarConfiguration = %d,%v, want 0 (decoded output is interleaved)", pc, ok)
	}
	if lic, _ := f.DataSet.GetString(TagLossyImageCompression); lic != "01" {
		t.Errorf("LossyImageCompression = %q, want %q (JPEG Baseline source is lossy)", lic, "01")
	}

	// The written object must decode to the same frames as decoding the original.
	out := filepath.Join(t.TempDir(), "native.dcm")
	if err := WriteFile(out, f); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := ReadFile(out)
	if err != nil {
		t.Fatalf("re-ReadFile(native): %v", err)
	}
	gotPD, err := NewPixelData(got.DataSet, got.Meta.TransferSyntaxUID)
	if err != nil {
		t.Fatalf("NewPixelData(native): %v", err)
	}
	gotFrames := collectFrames(t, gotPD)
	if len(gotFrames) != len(wantFrames) {
		t.Fatalf("frame count %d != %d", len(gotFrames), len(wantFrames))
	}
	for i := range wantFrames {
		if !bytes.Equal(gotFrames[i], wantFrames[i]) {
			t.Errorf("frame %d differs from decoding the original", i)
		}
	}
}
