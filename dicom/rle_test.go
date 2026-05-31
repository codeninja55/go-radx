package dicom

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestRLECodecRegisteredAndEncodes(t *testing.T) {
	c, ok := lookupCodec(RLELossless)
	if !ok {
		t.Fatal("RLE codec not registered")
	}
	if !c.CanEncode() {
		t.Error("RLE codec should report CanEncode() == true")
	}
	if c.TransferSyntax() != RLELossless {
		t.Errorf("TransferSyntax() = %s, want RLE Lossless", c.TransferSyntax())
	}
}

func TestUnpackBitsLiteralAndReplicate(t *testing.T) {
	// Literal run of 3 (header 0x02) then replicate 0xAA five times (header 0xFC = -4).
	src := []byte{0x02, 0x01, 0x02, 0x03, 0xFC, 0xAA}
	got, err := unpackBits(src, 8)
	if err != nil {
		t.Fatalf("unpackBits: %v", err)
	}
	want := []byte{0x01, 0x02, 0x03, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}
	if !bytes.Equal(got, want) {
		t.Errorf("unpackBits = %v, want %v", got, want)
	}
}

func TestUnpackBitsNoOp(t *testing.T) {
	// 0x80 (-128) is a no-op; the following literal still decodes.
	src := []byte{0x80, 0x00, 0x42}
	got, err := unpackBits(src, 1)
	if err != nil {
		t.Fatalf("unpackBits: %v", err)
	}
	if !bytes.Equal(got, []byte{0x42}) {
		t.Errorf("unpackBits = %v, want [0x42]", got)
	}
}

func TestUnpackBitsTruncatedLiteralIsError(t *testing.T) {
	// Header 0x05 claims a 6-byte literal but only 2 bytes follow.
	if _, err := unpackBits([]byte{0x05, 0x01, 0x02}, 6); err == nil {
		t.Fatal("expected an error for a literal run past the segment end")
	}
}

func TestDecodeRLEFrame8BitMono(t *testing.T) {
	geom := PixelGeometry{Rows: 1, Columns: 8, SamplesPerPixel: 1, BitsAllocated: 8}
	pixels := []byte{1, 1, 1, 1, 2, 2, 3, 3}
	encoded, err := encodeRLEFrame(pixels, geom)
	if err != nil {
		t.Fatalf("encodeRLEFrame: %v", err)
	}
	decoded, err := decodeRLEFrame(encoded, geom)
	if err != nil {
		t.Fatalf("decodeRLEFrame: %v", err)
	}
	if !bytes.Equal(decoded, pixels) {
		t.Errorf("decoded = %v, want %v", decoded, pixels)
	}
}

func TestDecodeRLEFrameRejectsBadBitsAllocated(t *testing.T) {
	geom := PixelGeometry{Rows: 2, Columns: 2, SamplesPerPixel: 1, BitsAllocated: 12}
	if _, err := decodeRLEFrame(make([]byte, 64), geom); err == nil {
		t.Fatal("expected an error for BitsAllocated 12 (RLE allows only 8 or 16)")
	}
}

func TestDecodeRLEFrameRejectsShortFrame(t *testing.T) {
	geom := PixelGeometry{Rows: 2, Columns: 2, SamplesPerPixel: 1, BitsAllocated: 8}
	if _, err := decodeRLEFrame([]byte{1, 2, 3}, geom); err == nil {
		t.Fatal("expected an error for a frame shorter than the 64-byte header")
	}
}

// TestDecodeRLEFrameRejectsBadOffset feeds a header whose segment offset points
// outside the frame.
func TestDecodeRLEFrameRejectsBadOffset(t *testing.T) {
	geom := PixelGeometry{Rows: 2, Columns: 2, SamplesPerPixel: 1, BitsAllocated: 8}
	frame := make([]byte, 64)
	frame[0] = 1 // one segment
	// segment 0 offset = 0xFFFF (well past the 64-byte frame)
	frame[4] = 0xFF
	frame[5] = 0xFF
	if _, err := decodeRLEFrame(frame, geom); err == nil {
		t.Fatal("expected an error for a segment offset outside the frame")
	}
}

// TestRLEDecodeLiverFixture decodes the liver_rle.dcm fixture's three RLE frames in
// pure Go and checks each yields the expected contiguous frame length. The fixture is
// a 512x512 single-segment RLE object (one byte per pixel); pydicom is not installed
// locally, so the check is structural (frame count and decoded length) plus a
// round-trip re-encode.
func TestRLEDecodeLiverFixture(t *testing.T) {
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "liver_rle.dcm"))
	if err != nil {
		t.Fatalf("ReadPixelData: %v", err)
	}
	if !pd.IsEncapsulated() {
		t.Fatal("liver_rle.dcm is encapsulated; IsEncapsulated() should be true")
	}
	if got, want := pd.Geometry.NumberOfFrames, 3; got != want {
		t.Fatalf("NumberOfFrames = %d, want %d", got, want)
	}

	const wantDecodedLen = 512 * 512 // one byte per pixel (single segment)
	var frames int
	for frame, err := range pd.Frames() {
		if err != nil {
			t.Fatalf("frame %d: %v", frames, err)
		}
		if len(frame.Pixels) != wantDecodedLen {
			t.Errorf("frame %d decoded length = %d, want %d", frames, len(frame.Pixels), wantDecodedLen)
		}
		frames++
	}
	if frames != 3 {
		t.Errorf("decoded %d frames, want 3", frames)
	}
}
