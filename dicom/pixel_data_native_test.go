package dicom

import (
	"path/filepath"
	"testing"
)

// TestNativePixelDataFromSCRGB decodes the uncompressed RGB fixture SC_rgb_expb.dcm
// (100x100, 8-bit, 3 samples/pixel, Explicit VR Big Endian) and checks the single
// decoded frame is the expected contiguous length.
func TestNativePixelDataFromSCRGB(t *testing.T) {
	f, err := ReadFile(filepath.Join("..", "testdata", "dicom", "SC_rgb_expb.dcm"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	pd, err := NewPixelData(f.DataSet, f.Meta.TransferSyntaxUID)
	if err != nil {
		t.Fatalf("NewPixelData: %v", err)
	}
	if pd.IsEncapsulated() {
		t.Fatal("SC_rgb_expb.dcm is uncompressed; IsEncapsulated() should be false")
	}

	const wantFrameBytes = 100 * 100 * 3
	var frames int
	for frame, err := range pd.Frames() {
		if err != nil {
			t.Fatalf("frame %d: %v", frames, err)
		}
		if frame.Index != frames {
			t.Errorf("frame.Index = %d, want %d", frame.Index, frames)
		}
		if len(frame.Pixels) != wantFrameBytes {
			t.Errorf("frame %d length = %d, want %d", frames, len(frame.Pixels), wantFrameBytes)
		}
		frames++
	}
	if frames != 1 {
		t.Errorf("decoded %d frames, want 1", frames)
	}
}

// TestNativePixelDataMultiFrameSlicing checks per-frame slicing of a synthetic
// multi-frame native buffer: three 2x2 8-bit mono frames, each a distinct fill byte.
func TestNativePixelDataMultiFrameSlicing(t *testing.T) {
	geom := PixelGeometry{
		Rows: 2, Columns: 2, SamplesPerPixel: 1, BitsAllocated: 8,
		NumberOfFrames: 3, TransferSyntax: ExplicitVRLittleEndian,
	}
	frameLen := geom.FrameLength()
	if frameLen != 4 {
		t.Fatalf("frame length = %d, want 4", frameLen)
	}
	buf := make([]byte, 0, frameLen*3)
	for f := 0; f < 3; f++ {
		for i := 0; i < frameLen; i++ {
			buf = append(buf, byte(0x10*(f+1)))
		}
	}

	pd := newNativePixelData(geom, buf)
	var got [][]byte
	for frame, err := range pd.Frames() {
		if err != nil {
			t.Fatalf("frame %d: %v", frame.Index, err)
		}
		got = append(got, frame.Pixels)
	}
	if len(got) != 3 {
		t.Fatalf("decoded %d frames, want 3", len(got))
	}
	for f := 0; f < 3; f++ {
		want := byte(0x10 * (f + 1))
		for _, b := range got[f] {
			if b != want {
				t.Fatalf("frame %d byte = 0x%02X, want 0x%02X", f, b, want)
			}
		}
	}
}

// TestNativePixelDataRejectsShortBuffer fails closed when the native buffer is too
// short for the declared geometry, rather than yielding a truncated final frame.
func TestNativePixelDataRejectsShortBuffer(t *testing.T) {
	geom := PixelGeometry{
		Rows: 4, Columns: 4, SamplesPerPixel: 1, BitsAllocated: 8,
		NumberOfFrames: 2, TransferSyntax: ExplicitVRLittleEndian,
	}
	// Only one frame's worth of bytes for a two-frame geometry.
	pd := newNativePixelData(geom, make([]byte, geom.FrameLength()))

	var sawErr bool
	for _, err := range pd.Frames() {
		if err != nil {
			sawErr = true
			break
		}
	}
	if !sawErr {
		t.Fatal("expected an error for a buffer shorter than the declared frame count")
	}
}

// TestNativePixelDataBitPacked accesses a 1-bit packed segmentation frame: 3 frames
// of 4x4 bits = 2 bytes each.
func TestNativePixelDataBitPacked(t *testing.T) {
	geom := PixelGeometry{
		Rows: 4, Columns: 4, SamplesPerPixel: 1, BitsAllocated: 1,
		NumberOfFrames: 3, TransferSyntax: ExplicitVRLittleEndian,
	}
	if geom.FrameLength() != 2 {
		t.Fatalf("frame length = %d, want 2", geom.FrameLength())
	}
	pd := newNativePixelData(geom, make([]byte, geom.FrameLength()*3))

	var frames int
	for frame, err := range pd.Frames() {
		if err != nil {
			t.Fatalf("frame %d: %v", frame.Index, err)
		}
		if len(frame.Pixels) != 2 {
			t.Errorf("frame %d length = %d, want 2", frame.Index, len(frame.Pixels))
		}
		frames++
	}
	if frames != 3 {
		t.Errorf("decoded %d frames, want 3", frames)
	}
}
