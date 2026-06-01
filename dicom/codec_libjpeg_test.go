//go:build cgo && dicom_libjpeg

package dicom

import (
	"errors"
	"path/filepath"
	"testing"
)

// TestJPEGCodecsRegistered checks that building with -tags dicom_libjpeg makes the
// baseline and extended JPEG codecs available where the pure-Go build had none. Both
// are decode-only (re-encoding to a lossy syntax is never a default).
func TestJPEGCodecsRegistered(t *testing.T) {
	for _, ts := range []TransferSyntax{JPEGBaseline8Bit, JPEGExtended12Bit} {
		c, ok := lookupCodec(ts)
		if !ok {
			t.Fatalf("no codec registered for %s (%s)", ts.Name(), ts)
		}
		if c.TransferSyntax() != ts {
			t.Errorf("codec for %s reports %s", ts, c.TransferSyntax())
		}
		if c.CanEncode() {
			t.Errorf("JPEG codec for %s should be decode-only", ts)
		}
	}
}

// TestJPEGDecodeBaselineRGBFixture is the named baseline (.50) decode regression:
// SC_jpeg_no_color_transform.dcm (256x256 RGB 8-bit, no colour transform) decodes,
// and the frame is exactly the FrameLength the resolved geometry declares.
func TestJPEGDecodeBaselineRGBFixture(t *testing.T) {
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "SC_jpeg_no_color_transform.dcm"))
	if err != nil {
		t.Fatalf("ReadPixelData: %v", err)
	}
	if pd.Geometry.TransferSyntax != JPEGBaseline8Bit {
		t.Fatalf("transfer syntax = %s, want JPEG Baseline", pd.Geometry.TransferSyntax)
	}

	wantLen := pd.Geometry.FrameLength()
	var frames int
	for frame, err := range pd.Frames() {
		if err != nil {
			t.Fatalf("frame %d: %v", frames, err)
		}
		if len(frame.Pixels) != wantLen {
			t.Errorf("frame %d length = %d, want %d", frames, len(frame.Pixels), wantLen)
		}
		frames++
	}
	if frames != pd.Geometry.NumberOfFrames {
		t.Errorf("decoded %d frames, want %d", frames, pd.Geometry.NumberOfFrames)
	}
}

// TestJPEGDecodeBaselineYBRPixelExact decodes the small YBR_FULL baseline fixture
// (3x3, JFIF/YCbCr) and asserts the first three pixels match pydicom's reference
// decode exactly. Because the dataset declares YBR_FULL, the decoded samples must
// stay YCbCr (no RGB conversion); these reference values come from pydicom's
// pixels_reference (JPGB_08_08_3_0_1F_YBR_FULL: arr[0,0]=(138,78,147),
// arr[1,0]=(90,178,108), arr[2,0]=(158,126,129)).
func TestJPEGDecodeBaselineYBRPixelExact(t *testing.T) {
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "SC_rgb_small_odd_jpeg.dcm"))
	if err != nil {
		t.Fatalf("ReadPixelData: %v", err)
	}
	if pd.Geometry.PhotometricInterpretation != "YBR_FULL" {
		t.Fatalf("photometric interpretation = %q, want YBR_FULL", pd.Geometry.PhotometricInterpretation)
	}

	var pixels []byte
	for frame, err := range pd.Frames() {
		if err != nil {
			t.Fatalf("frame: %v", err)
		}
		pixels = frame.Pixels
		break
	}
	if len(pixels) != pd.Geometry.FrameLength() {
		t.Fatalf("frame length = %d, want %d", len(pixels), pd.Geometry.FrameLength())
	}

	// The frame is 3x3x3 interleaved YBR samples in raster order. The reference
	// indexes [row, col]; for a 3-column image, pixel (row r, col 0) is at byte
	// r*3*3 + 0.
	cols := int(pd.Geometry.Columns)
	wantRowCol0 := [3][3]byte{
		{138, 78, 147},  // (0,0)
		{90, 178, 108},  // (1,0)
		{158, 126, 129}, // (2,0)
	}
	for r := 0; r < 3; r++ {
		base := (r * cols) * 3
		got := [3]byte{pixels[base], pixels[base+1], pixels[base+2]}
		if got != wantRowCol0[r] {
			t.Errorf("pixel (%d,0) = %v, want %v", r, got, wantRowCol0[r])
		}
	}
}

// TestJPEGDecodeExtended12BitFixture is the named extended (.51) decode regression:
// JPGExtended.dcm (1024x256 MONOCHROME2, 16-bit allocated / 12-bit stored) decodes,
// and the frame is exactly the FrameLength the resolved geometry declares.
func TestJPEGDecodeExtended12BitFixture(t *testing.T) {
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "JPGExtended.dcm"))
	if err != nil {
		t.Fatalf("ReadPixelData: %v", err)
	}
	if pd.Geometry.TransferSyntax != JPEGExtended12Bit {
		t.Fatalf("transfer syntax = %s, want JPEG Extended", pd.Geometry.TransferSyntax)
	}
	if pd.Geometry.BitsStored != 12 || pd.Geometry.BitsAllocated != 16 {
		t.Fatalf("geometry BitsStored/BitsAllocated = %d/%d, want 12/16",
			pd.Geometry.BitsStored, pd.Geometry.BitsAllocated)
	}

	wantLen := pd.Geometry.FrameLength()
	var frames int
	for frame, err := range pd.Frames() {
		if err != nil {
			t.Fatalf("frame %d: %v", frames, err)
		}
		if len(frame.Pixels) != wantLen {
			t.Errorf("frame %d length = %d, want %d", frames, len(frame.Pixels), wantLen)
		}
		// Every 12-bit sample must fit in 12 bits once unpacked from the 16-bit
		// little-endian frame, confirming the precision was preserved.
		for i := 0; i+1 < len(frame.Pixels); i += 2 {
			v := uint16(frame.Pixels[i]) | uint16(frame.Pixels[i+1])<<8
			if v > 0x0FFF {
				t.Fatalf("12-bit sample at byte %d exceeds 12 bits: %d", i, v)
			}
		}
		frames++
	}
	if frames != pd.Geometry.NumberOfFrames {
		t.Errorf("decoded %d frames, want %d", frames, pd.Geometry.NumberOfFrames)
	}
}

// TestJPEGTranscodeToNative decodes the baseline RGB fixture and transcodes it to
// uncompressed Explicit VR Little Endian, confirming the native frame count and
// length match the geometry through the Transcode path.
func TestJPEGTranscodeToNative(t *testing.T) {
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "SC_jpeg_no_color_transform.dcm"))
	if err != nil {
		t.Fatalf("ReadPixelData: %v", err)
	}
	native, err := Transcode(pd, ExplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("Transcode: %v", err)
	}
	if native.IsEncapsulated() {
		t.Fatal("transcoded-to-native PixelData should not be encapsulated")
	}
	wantLen := pd.Geometry.FrameLength()
	var frames int
	for frame, err := range native.Frames() {
		if err != nil {
			t.Fatalf("native frame %d: %v", frames, err)
		}
		if len(frame.Pixels) != wantLen {
			t.Errorf("native frame %d length = %d, want %d", frames, len(frame.Pixels), wantLen)
		}
		frames++
	}
	if frames != pd.Geometry.NumberOfFrames {
		t.Errorf("transcoded %d frames, want %d", frames, pd.Geometry.NumberOfFrames)
	}
}

// TestJPEGIsDecodeOnly checks both JPEG codecs reject encode with the typed
// ErrEncodeUnsupported (re-encoding to a lossy syntax is never offered).
func TestJPEGIsDecodeOnly(t *testing.T) {
	geom := PixelGeometry{Rows: 8, Columns: 8, SamplesPerPixel: 1, BitsAllocated: 8, BitsStored: 8}
	for _, ts := range []TransferSyntax{JPEGBaseline8Bit, JPEGExtended12Bit} {
		c, ok := lookupCodec(ts)
		if !ok {
			t.Fatalf("no codec registered for %s", ts)
		}
		if _, err := c.Encode(make([]byte, geom.FrameLength()), geom); !errors.Is(err, ErrEncodeUnsupported) {
			t.Errorf("%s Encode error = %v, want ErrEncodeUnsupported", ts, err)
		}
	}
}

// TestJPEGLosslessSyntaxesUnavailable documents the scope boundary: libjpeg-turbo
// does not implement the lossless-JPEG processes DICOM .57/.70 use, so those
// syntaxes register no codec even with the dicom_libjpeg tag, and an instance
// degrades to the typed ErrCodecUnavailable.
func TestJPEGLosslessSyntaxesUnavailable(t *testing.T) {
	for _, ts := range []TransferSyntax{
		TransferSyntax("1.2.840.10008.1.2.4.57"), // JPEG Lossless, Non-Hierarchical (Process 14)
		TransferSyntax("1.2.840.10008.1.2.4.70"), // JPEG Lossless, Non-Hierarchical, SV1
	} {
		if c, ok := lookupCodec(ts); ok {
			t.Errorf("did not expect a codec for lossless JPEG %s, got %T", ts, c)
		}
	}
}
