package dicom

import (
	"bytes"
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
