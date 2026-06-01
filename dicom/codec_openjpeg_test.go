//go:build cgo && dicom_openjpeg

package dicom

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
)

// TestJPEG2000CodecRegistered checks that building with -tags dicom_openjpeg makes
// the JPEG 2000 codecs available where the pure-Go build had none.
func TestJPEG2000CodecRegistered(t *testing.T) {
	for _, ts := range []TransferSyntax{JPEG2000Lossless, JPEG2000} {
		c, ok := lookupCodec(ts)
		if !ok {
			t.Fatalf("no codec registered for %s (%s)", ts.Name(), ts)
		}
		if c.TransferSyntax() != ts {
			t.Errorf("codec for %s reports %s", ts, c.TransferSyntax())
		}
	}
	if c, _ := lookupCodec(JPEG2000Lossless); !c.CanEncode() {
		t.Error("JPEG 2000 Lossless codec should support encode")
	}
	if c, _ := lookupCodec(JPEG2000); c.CanEncode() {
		t.Error("JPEG 2000 (lossy) codec should be decode-only")
	}
}

// TestJPEG2000DecodeLiverFixture is the named decode regression: with the codec
// built in, liver_j2k.dcm (JPEG 2000 Lossless, three 512x512 1-bit frames) decodes,
// and every frame is exactly the FrameLength the resolved PixelGeometry declares.
func TestJPEG2000DecodeLiverFixture(t *testing.T) {
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "liver_j2k.dcm"))
	if err != nil {
		t.Fatalf("ReadPixelData: %v", err)
	}
	if pd.Geometry.TransferSyntax != JPEG2000Lossless {
		t.Fatalf("transfer syntax = %s, want JPEG 2000 Lossless", pd.Geometry.TransferSyntax)
	}

	wantLen := pd.Geometry.FrameLength()
	wantFrames := pd.Geometry.NumberOfFrames
	var frames int
	for frame, err := range pd.Frames() {
		if err != nil {
			t.Fatalf("frame %d: %v", frames, err)
		}
		if len(frame.Pixels) != wantLen {
			t.Errorf("frame %d length = %d, want %d", frames, len(frame.Pixels), wantLen)
		}
		if frame.Index != frames {
			t.Errorf("frame index = %d, want %d", frame.Index, frames)
		}
		frames++
	}
	if frames != wantFrames {
		t.Errorf("decoded %d frames, want %d", frames, wantFrames)
	}
}

// TestJPEG2000LosslessRoundTripPixelExact is the verification-gate round-trip: a
// frame encoded to JPEG 2000 Lossless and decoded back is byte-identical. It covers
// 16-bit and 8-bit monochrome, 8-bit RGB, and a 1-bit segmentation frame, plus a
// small frame that exercises the resolution-level clamp.
func TestJPEG2000LosslessRoundTripPixelExact(t *testing.T) {
	tests := []struct {
		name string
		geom PixelGeometry
	}{
		{"16-bit mono", PixelGeometry{Rows: 64, Columns: 64, SamplesPerPixel: 1, BitsAllocated: 16, BitsStored: 16, PixelRepresentation: 0}},
		{"16-bit signed mono", PixelGeometry{Rows: 48, Columns: 48, SamplesPerPixel: 1, BitsAllocated: 16, BitsStored: 16, PixelRepresentation: 1}},
		{"8-bit mono", PixelGeometry{Rows: 32, Columns: 32, SamplesPerPixel: 1, BitsAllocated: 8, BitsStored: 8}},
		{"8-bit rgb", PixelGeometry{Rows: 16, Columns: 16, SamplesPerPixel: 3, BitsAllocated: 8, BitsStored: 8}},
		{"1-bit segmentation", PixelGeometry{Rows: 16, Columns: 16, SamplesPerPixel: 1, BitsAllocated: 1, BitsStored: 1}},
		{"tiny frame", PixelGeometry{Rows: 4, Columns: 4, SamplesPerPixel: 1, BitsAllocated: 16, BitsStored: 12}},
	}
	c := openjpegCodec{ts: JPEG2000Lossless, canEncode: true}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			orig := syntheticFrame(tc.geom)
			encoded, err := c.Encode(orig, tc.geom)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			decoded, err := c.Decode(encoded, tc.geom)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if !bytes.Equal(decoded, orig) {
				for i := range orig {
					if decoded[i] != orig[i] {
						t.Fatalf("round-trip not pixel-exact at byte %d: got %d want %d", i, decoded[i], orig[i])
					}
				}
			}
		})
	}
}

// TestJPEG2000LossyIsDecodeOnly checks the .91 codec rejects encode with the typed
// ErrEncodeUnsupported (re-compressing as lossy is never offered).
func TestJPEG2000LossyIsDecodeOnly(t *testing.T) {
	c, ok := lookupCodec(JPEG2000)
	if !ok {
		t.Fatal("JPEG 2000 codec not registered")
	}
	geom := PixelGeometry{Rows: 8, Columns: 8, SamplesPerPixel: 1, BitsAllocated: 8, BitsStored: 8}
	_, err := c.Encode(make([]byte, geom.FrameLength()), geom)
	if !errors.Is(err, ErrEncodeUnsupported) {
		t.Errorf("Encode error = %v, want ErrEncodeUnsupported", err)
	}
}

// TestJPEG2000TranscodeFromLiver decodes liver_j2k.dcm, transcodes the frames to
// uncompressed Explicit VR Little Endian, and confirms the native frame count and
// length match the geometry, exercising the codec through the Transcode path.
func TestJPEG2000TranscodeFromLiver(t *testing.T) {
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "liver_j2k.dcm"))
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

// TestTranscodeNativeToJPEG2000AndBack round-trips a native frame through JPEG 2000
// Lossless and back, requiring pixel-exact frames, which exercises the encode path
// through Transcode that is unavailable without the tag.
func TestTranscodeNativeToJPEG2000AndBack(t *testing.T) {
	geom := PixelGeometry{
		Rows: 32, Columns: 32, SamplesPerPixel: 1, BitsAllocated: 16, BitsStored: 16,
		NumberOfFrames: 2, TransferSyntax: ExplicitVRLittleEndian,
	}
	frameLen := geom.FrameLength()
	buf := make([]byte, frameLen*2)
	for i := 0; i+1 < len(buf); i += 2 {
		v := uint16(i * 3)
		buf[i] = byte(v)
		buf[i+1] = byte(v >> 8)
	}
	src := newNativePixelData(geom, buf)

	encoded, err := Transcode(src, JPEG2000Lossless)
	if err != nil {
		t.Fatalf("Transcode to JPEG 2000: %v", err)
	}
	if !encoded.IsEncapsulated() {
		t.Fatal("JPEG 2000 target should be encapsulated")
	}

	back, err := Transcode(encoded, ExplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("Transcode back to native: %v", err)
	}
	var frames int
	for frame, err := range back.Frames() {
		if err != nil {
			t.Fatalf("native frame %d: %v", frames, err)
		}
		want := buf[frames*frameLen : (frames+1)*frameLen]
		if !bytes.Equal(frame.Pixels, want) {
			t.Errorf("frame %d not pixel-exact after JPEG 2000 round-trip", frames)
		}
		frames++
	}
	if frames != 2 {
		t.Errorf("decoded %d frames, want 2", frames)
	}
}

// syntheticFrame fills a frame of the geometry's FrameLength with deterministic,
// in-range sample values so a lossless round-trip is meaningful.
func syntheticFrame(geom PixelGeometry) []byte {
	out := make([]byte, geom.FrameLength())
	if geom.BitsAllocated >= 16 {
		stored := geom.BitsStored
		if stored == 0 || stored > 16 {
			stored = 16
		}
		mask := uint16(1)<<stored - 1
		for p := 0; p*2+1 < len(out); p++ {
			v := uint16(p*37+11) & mask
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
