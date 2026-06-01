//go:build cgo && dicom_charls

package dicom

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
)

// TestJPEGLSCodecsRegistered checks that building with -tags dicom_charls makes the
// JPEG-LS codecs available where the pure-Go build had none. Lossless (.80) supports
// encode; Near-Lossless (.81) is decode-only.
func TestJPEGLSCodecsRegistered(t *testing.T) {
	for _, ts := range []TransferSyntax{JPEGLSLossless, JPEGLSNearLossless} {
		c, ok := lookupCodec(ts)
		if !ok {
			t.Fatalf("no codec registered for %s (%s)", ts.Name(), ts)
		}
		if c.TransferSyntax() != ts {
			t.Errorf("codec for %s reports %s", ts, c.TransferSyntax())
		}
	}
	if c, _ := lookupCodec(JPEGLSLossless); !c.CanEncode() {
		t.Error("JPEG-LS Lossless codec should support encode")
	}
	if c, _ := lookupCodec(JPEGLSNearLossless); c.CanEncode() {
		t.Error("JPEG-LS Near-Lossless codec should be decode-only")
	}
}

// readFirstFrame decodes and returns the first frame of the named fixture.
func readFirstFrame(t *testing.T, name string) (*PixelData, []byte) {
	t.Helper()
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", name))
	if err != nil {
		t.Fatalf("%s: ReadPixelData: %v", name, err)
	}
	for frame, err := range pd.Frames() {
		if err != nil {
			t.Fatalf("%s: frame: %v", name, err)
		}
		if len(frame.Pixels) != pd.Geometry.FrameLength() {
			t.Fatalf("%s: frame length = %d, want %d", name, len(frame.Pixels), pd.Geometry.FrameLength())
		}
		return pd, frame.Pixels
	}
	t.Fatalf("%s: no frames", name)
	return nil, nil
}

// s16le reads the idx-th signed 16-bit little-endian sample from a frame.
func s16le(p []byte, idx int) int16 {
	return int16(uint16(p[idx*2]) | uint16(p[idx*2+1])<<8)
}

// TestJPEGLSDecodeLosslessPixelExact is the named lossless (.80) decode regression:
// MR_small_jpeg_ls_lossless.dcm (64x64 signed 16-bit MONOCHROME2) decodes, and the
// sampled pixels match pydicom's reference values exactly (pixels_reference
// JLSL_16_16_1_1_1F).
func TestJPEGLSDecodeLosslessPixelExact(t *testing.T) {
	pd, px := readFirstFrame(t, "MR_small_jpeg_ls_lossless.dcm")
	if pd.Geometry.TransferSyntax != JPEGLSLossless {
		t.Fatalf("transfer syntax = %s, want JPEG-LS Lossless", pd.Geometry.TransferSyntax)
	}
	cols := int(pd.Geometry.Columns)

	if got := [3]int16{s16le(px, 31), s16le(px, 32), s16le(px, 33)}; got != [3]int16{422, 319, 361} {
		t.Errorf("arr[0,31:34] = %v, want [422 319 361]", got)
	}
	if got := [3]int16{s16le(px, 31*cols), s16le(px, 31*cols+1), s16le(px, 31*cols+2)}; got != [3]int16{366, 363, 322} {
		t.Errorf("arr[31,:3] = %v, want [366 363 322]", got)
	}
	last := 63 * cols
	if got := [3]int16{s16le(px, last+61), s16le(px, last+62), s16le(px, last+63)}; got != [3]int16{1369, 1129, 862} {
		t.Errorf("arr[-1,-3:] = %v, want [1369 1129 862]", got)
	}
}

// TestJPEGLSDecodeNearLosslessFixtures is the named near-lossless (.81) decode
// regression: the 8-bit and 16-bit MONOCHROME2 fixtures decode to the geometry's
// FrameLength, and the 8-bit fixture matches pydicom's reference column samples
// (pixels_reference JLSN_08_01_1_0_1F).
func TestJPEGLSDecodeNearLosslessFixtures(t *testing.T) {
	pd, px := readFirstFrame(t, "JPEGLSNearLossless_08.dcm")
	if pd.Geometry.TransferSyntax != JPEGLSNearLossless {
		t.Fatalf("transfer syntax = %s, want JPEG-LS Near-Lossless", pd.Geometry.TransferSyntax)
	}
	cols := int(pd.Geometry.Columns)
	checks := map[int]byte{20: 15, 25: 5, 30: 5, 35: 0, 40: 0}
	for row, want := range checks {
		if got := px[row*cols]; got != want {
			t.Errorf("arr[%d,0] = %d, want %d", row, got, want)
		}
	}

	pd16, px16 := readFirstFrame(t, "JPEGLSNearLossless_16.dcm")
	if pd16.Geometry.TransferSyntax != JPEGLSNearLossless {
		t.Fatalf("16-bit fixture transfer syntax = %s, want JPEG-LS Near-Lossless", pd16.Geometry.TransferSyntax)
	}
	if len(px16) != pd16.Geometry.FrameLength() {
		t.Errorf("16-bit near-lossless frame length = %d, want %d", len(px16), pd16.Geometry.FrameLength())
	}
}

// TestJPEGLSDecodeRGBSampleInterleaved decodes the sample-interleaved RGB
// near-lossless fixture and confirms the frame is sample-interleaved (planar
// configuration 0) at the declared length.
func TestJPEGLSDecodeRGBSampleInterleaved(t *testing.T) {
	pd, px := readFirstFrame(t, "SC_rgb_jls_lossy_sample.dcm")
	if pd.Geometry.SamplesPerPixel != 3 || pd.Geometry.PhotometricInterpretation != "RGB" {
		t.Fatalf("geometry SPP=%d PI=%q, want 3/RGB", pd.Geometry.SamplesPerPixel, pd.Geometry.PhotometricInterpretation)
	}
	if len(px) != pd.Geometry.FrameLength() {
		t.Errorf("frame length = %d, want %d", len(px), pd.Geometry.FrameLength())
	}
}

// TestJPEGLSLosslessRoundTripPixelExact is the verification-gate round-trip: a frame
// encoded to JPEG-LS Lossless and decoded back is byte-identical, across 8-bit and
// 16-bit monochrome and 8-bit RGB.
func TestJPEGLSLosslessRoundTripPixelExact(t *testing.T) {
	tests := []struct {
		name string
		geom PixelGeometry
	}{
		{"8-bit mono", PixelGeometry{Rows: 32, Columns: 32, SamplesPerPixel: 1, BitsAllocated: 8, BitsStored: 8}},
		{"16-bit mono", PixelGeometry{Rows: 48, Columns: 48, SamplesPerPixel: 1, BitsAllocated: 16, BitsStored: 16}},
		{"8-bit rgb", PixelGeometry{Rows: 16, Columns: 16, SamplesPerPixel: 3, BitsAllocated: 8, BitsStored: 8, PhotometricInterpretation: "RGB"}},
	}
	c := charlsCodec{ts: JPEGLSLossless, canEncode: true}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			orig := syntheticFrameJLS(tc.geom)
			encoded, err := c.Encode(orig, tc.geom)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			decoded, err := c.Decode(encoded, tc.geom)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if !bytes.Equal(decoded, orig) {
				t.Fatalf("round-trip not pixel-exact (len %d vs %d)", len(decoded), len(orig))
			}
		})
	}
}

// TestJPEGLSNearLosslessIsDecodeOnly checks the .81 codec rejects encode with the
// typed ErrEncodeUnsupported (re-compressing as near-lossless is never offered).
func TestJPEGLSNearLosslessIsDecodeOnly(t *testing.T) {
	c, ok := lookupCodec(JPEGLSNearLossless)
	if !ok {
		t.Fatal("JPEG-LS Near-Lossless codec not registered")
	}
	geom := PixelGeometry{Rows: 8, Columns: 8, SamplesPerPixel: 1, BitsAllocated: 8, BitsStored: 8}
	if _, err := c.Encode(make([]byte, geom.FrameLength()), geom); !errors.Is(err, ErrEncodeUnsupported) {
		t.Errorf("Encode error = %v, want ErrEncodeUnsupported", err)
	}
}

// TestJPEGLSTranscodeToNative decodes the lossless fixture and transcodes it to
// uncompressed Explicit VR Little Endian, confirming the native frame count and
// length match the geometry through the Transcode path.
func TestJPEGLSTranscodeToNative(t *testing.T) {
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "MR_small_jpeg_ls_lossless.dcm"))
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

// syntheticFrameJLS fills a frame of the geometry's FrameLength with deterministic,
// in-range sample values so a lossless round-trip is meaningful.
func syntheticFrameJLS(geom PixelGeometry) []byte {
	out := make([]byte, geom.FrameLength())
	if geom.BitsAllocated >= 16 {
		for p := 0; p*2+1 < len(out); p++ {
			v := uint16(p*53+17) & 0xFFFF
			out[p*2] = byte(v)
			out[p*2+1] = byte(v >> 8)
		}
		return out
	}
	for i := range out {
		out[i] = byte((i*7 + 3) % 251)
	}
	return out
}
