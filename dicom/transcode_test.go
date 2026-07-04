package dicom

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
)

func twoFrameNativePixelData(t *testing.T) *PixelData {
	t.Helper()
	geom := PixelGeometry{
		Rows: 4, Columns: 4, SamplesPerPixel: 1, BitsAllocated: 8,
		NumberOfFrames: 2, TransferSyntax: ExplicitVRLittleEndian,
	}
	frameLen := geom.FrameLength()
	buf := make([]byte, frameLen*2)
	for i := range buf {
		buf[i] = byte(i % 13)
	}
	return newNativePixelData(geom, buf)
}

// TestTranscodeNativeToRLEAndBack round-trips a native frame through RLE and back to
// a native syntax, requiring pixel-exact frames at every step.
func TestTranscodeNativeToRLEAndBack(t *testing.T) {
	src := twoFrameNativePixelData(t)
	var srcFrames [][]byte
	for frame, err := range src.Frames() {
		if err != nil {
			t.Fatalf("source frame: %v", err)
		}
		srcFrames = append(srcFrames, frame.Pixels)
	}

	rle, err := Transcode(src, RLELossless)
	if err != nil {
		t.Fatalf("Transcode to RLE: %v", err)
	}
	if !rle.IsEncapsulated() {
		t.Fatal("RLE target should be encapsulated")
	}
	if rle.Geometry.TransferSyntax != RLELossless {
		t.Errorf("target syntax = %s, want RLE", rle.Geometry.TransferSyntax)
	}

	// Decoding the RLE result must reproduce the source frames exactly.
	var rleFrames [][]byte
	for frame, err := range rle.Frames() {
		if err != nil {
			t.Fatalf("RLE frame: %v", err)
		}
		rleFrames = append(rleFrames, frame.Pixels)
	}
	if len(rleFrames) != len(srcFrames) {
		t.Fatalf("RLE produced %d frames, want %d", len(rleFrames), len(srcFrames))
	}
	for i := range srcFrames {
		if !bytes.Equal(rleFrames[i], srcFrames[i]) {
			t.Errorf("RLE frame %d not pixel-exact", i)
		}
	}

	back, err := Transcode(rle, ExplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("Transcode back to native: %v", err)
	}
	if back.IsEncapsulated() {
		t.Fatal("native target should not be encapsulated")
	}
	var backFrames [][]byte
	for frame, err := range back.Frames() {
		if err != nil {
			t.Fatalf("native frame: %v", err)
		}
		backFrames = append(backFrames, frame.Pixels)
	}
	for i := range srcFrames {
		if !bytes.Equal(backFrames[i], srcFrames[i]) {
			t.Errorf("native frame %d not pixel-exact after round-trip", i)
		}
	}
}

// TestTranscodeLiverRLEToNative decodes the RLE liver fixture and re-encodes its
// frames into a native buffer, checking the decoded frame count and length survive.
func TestTranscodeLiverRLEToNative(t *testing.T) {
	src, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "liver_rle.dcm"))
	if err != nil {
		t.Fatalf("ReadPixelData: %v", err)
	}
	native, err := Transcode(src, ExplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("Transcode to native: %v", err)
	}
	var frames int
	for frame, err := range native.Frames() {
		if err != nil {
			t.Fatalf("frame %d: %v", frames, err)
		}
		if len(frame.Pixels) != 512*512 {
			t.Errorf("frame %d length = %d, want %d", frames, len(frame.Pixels), 512*512)
		}
		frames++
	}
	if frames != 3 {
		t.Errorf("transcoded %d frames, want 3", frames)
	}
}

// TestSetPixelDataMonochromeRLEAttributesUnchanged pins the reconciliation floor for
// a monochrome decompress: nothing but the transfer syntax (and the pixel element
// itself) may change. No PlanarConfiguration appears for single-sample data, the
// colour model stays MONOCHROME2, the lossless source leaves LossyImageCompression
// exactly as stored ("00" in the fixture), and NumberOfFrames keeps the value the
// three-frame fixture already declares.
func TestSetPixelDataMonochromeRLEAttributesUnchanged(t *testing.T) {
	f, err := ReadFile(filepath.Join("..", "testdata", "dicom", "liver_rle.dcm"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
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

	if pi, _ := f.DataSet.GetString(TagPhotometricInterpretation); pi != "MONOCHROME2" {
		t.Errorf("PhotometricInterpretation = %q, want MONOCHROME2", pi)
	}
	if _, ok := f.DataSet.Get(TagPlanarConfiguration); ok {
		t.Error("PlanarConfiguration appeared on single-sample data")
	}
	if lic, _ := f.DataSet.GetString(TagLossyImageCompression); lic != "00" {
		t.Errorf("LossyImageCompression = %q, want the stored %q (RLE is lossless)", lic, "00")
	}
	if nf, ok := f.DataSet.GetInt(TagNumberOfFrames); !ok || nf != 3 {
		t.Errorf("NumberOfFrames = %d,%v, want the fixture's 3", nf, ok)
	}
	if f.Meta.TransferSyntaxUID != ExplicitVRLittleEndian {
		t.Errorf("TransferSyntaxUID = %q, want Explicit VR LE", f.Meta.TransferSyntaxUID)
	}
}

// TestSetPixelDataLossySourceBookkeeping pins PS3.3 C.7.6.1.1.5: pixels that enter
// SetPixelData from a lossy source syntax must leave with LossyImageCompression
// (0028,2110) = "01" on the output dataset, and the seam must not invent (or drop)
// the ratio/method attributes it cannot know.
func TestSetPixelDataLossySourceBookkeeping(t *testing.T) {
	f, err := ReadFile(filepath.Join("..", "testdata", "dicom", "MR2_UNCI.dcm"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	pd, err := NewPixelData(f.DataSet, f.Meta.TransferSyntaxUID)
	if err != nil {
		t.Fatalf("NewPixelData: %v", err)
	}

	// Simulate the decompress seam for a lossy source: SetPixelData reads the source
	// syntax from f.Meta before overwriting it.
	f.Meta.TransferSyntaxUID = JPEG2000
	f.DataSet.Delete(TagLossyImageCompression)
	f.DataSet.SetString(TagLossyImageCompressionRatio, "10.5")
	f.DataSet.SetString(TagLossyImageCompressionMethod, "ISO_15444_1")

	if err := f.SetPixelData(pd); err != nil {
		t.Fatalf("SetPixelData: %v", err)
	}
	if lic, _ := f.DataSet.GetString(TagLossyImageCompression); lic != "01" {
		t.Errorf("LossyImageCompression = %q, want %q after a lossy source", lic, "01")
	}
	if ratio, _ := f.DataSet.GetString(TagLossyImageCompressionRatio); ratio != "10.5" {
		t.Errorf("LossyImageCompressionRatio = %q, want the preserved %q", ratio, "10.5")
	}
	if method, _ := f.DataSet.GetString(TagLossyImageCompressionMethod); method != "ISO_15444_1" {
		t.Errorf("LossyImageCompressionMethod = %q, want the preserved %q", method, "ISO_15444_1")
	}
}

// TestDecodedPhotometricInterpretationMapping pins the colour-model reconciliation
// table: the inverse multiple-component transform OpenJPEG applies on decode turns
// the JPEG 2000 transform terms into RGB, the JPEG path upsamples (or rejects)
// subsampled chroma so YBR_FULL_422 decodes to YBR_FULL, and a term whose decoded
// layout cannot be determined fails closed.
func TestDecodedPhotometricInterpretationMapping(t *testing.T) {
	cases := []struct {
		src     TransferSyntax
		pi      string
		want    string
		wantErr bool
	}{
		{JPEG2000, "YBR_ICT", "RGB", false},
		{JPEG2000Lossless, "YBR_RCT", "RGB", false},
		{HTJ2K, "YBR_ICT", "RGB", false},
		{JPEGBaseline8Bit, "YBR_FULL_422", "YBR_FULL", false},
		{JPEGBaseline8Bit, "YBR_FULL", "YBR_FULL", false},
		{JPEGBaseline8Bit, "RGB", "RGB", false},
		{RLELossless, "MONOCHROME2", "MONOCHROME2", false},
		{RLELossless, "YBR_FULL", "YBR_FULL", false},
		{RLELossless, "YBR_ICT", "", true},      // no inverse transform outside J2K
		{RLELossless, "YBR_FULL_422", "", true}, // subsampled layout unknowable here
		{JPEGBaseline8Bit, "YBR_PARTIAL_422", "", true},
		{JPEGBaseline8Bit, "YBR_PARTIAL_420", "", true},
	}
	for _, tc := range cases {
		got, err := decodedPhotometricInterpretation(tc.src, tc.pi)
		if tc.wantErr {
			if _, ok := errors.AsType[*ValueError](err); !ok {
				t.Errorf("(%s, %s): err = %v, want *ValueError (fail closed)", tc.src.Name(), tc.pi, err)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("(%s, %s) = %q, %v; want %q", tc.src.Name(), tc.pi, got, err, tc.want)
		}
	}
}

// TestTranscodeFailsClosedOnUndeterminedColourModel pins the end-to-end fail-closed
// behaviour: transcoding an encapsulated colour source whose decoded colour model
// cannot be determined is a typed error before any frame is decoded, never a dataset
// whose attributes mismatch its pixels.
func TestTranscodeFailsClosedOnUndeterminedColourModel(t *testing.T) {
	src := &PixelData{
		Geometry: PixelGeometry{
			Rows: 2, Columns: 2, SamplesPerPixel: 3, BitsAllocated: 8,
			NumberOfFrames:            1,
			PhotometricInterpretation: "YBR_PARTIAL_422",
			TransferSyntax:            RLELossless,
		},
		encaps: &encapsulated{fragments: []fragment{{data: []byte{0, 0}}}},
	}
	_, err := Transcode(src, ExplicitVRLittleEndian)
	ve, ok := errors.AsType[*ValueError](err)
	if !ok {
		t.Fatalf("Transcode = %v, want *ValueError failing closed on the colour model", err)
	}
	if ve.Tag != TagPhotometricInterpretation {
		t.Errorf("ValueError.Tag = %s, want %s", ve.Tag, TagPhotometricInterpretation)
	}
}
