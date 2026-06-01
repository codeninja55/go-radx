//go:build cgo && dicom_openjpeg

package dicom

import (
	"errors"
	"path/filepath"
	"testing"
)

// htj2kFixtures is the set of vendored High-Throughput JPEG 2000 fixtures with their
// expected transfer syntax. Both are 480x640 RGB 8-bit single-frame images from the
// pydicom-data store (see testdata/dicom/README.md).
var htj2kFixtures = []struct {
	file string
	ts   TransferSyntax
}{
	{"HTJ2KLossless_08_RGB.dcm", HTJ2KLossless},
	{"HTJ2K_08_RGB.dcm", HTJ2K},
}

// TestHTJ2KCodecsRegistered checks the dicom_openjpeg build registers a decode-only
// codec for each HTJ2K transfer syntax. OpenJPEG 2.5 decodes HTJ2K through the same
// OPJ_CODEC_J2K path as classic JPEG 2000; this build has no HT encoder, so every
// HTJ2K syntax is decode-only.
func TestHTJ2KCodecsRegistered(t *testing.T) {
	for _, ts := range []TransferSyntax{HTJ2KLossless, HTJ2KLosslessRPCL, HTJ2K} {
		c, ok := lookupCodec(ts)
		if !ok {
			t.Fatalf("no codec registered for %s (%s)", ts.Name(), ts)
		}
		if c.TransferSyntax() != ts {
			t.Errorf("codec for %s reports %s", ts, c.TransferSyntax())
		}
		if c.CanEncode() {
			t.Errorf("HTJ2K codec for %s should be decode-only", ts)
		}
	}
}

// TestHTJ2KDecodeFixtures is the named decode regression: with the codec built in,
// each HTJ2K fixture decodes and every frame is exactly the FrameLength the resolved
// PixelGeometry declares.
func TestHTJ2KDecodeFixtures(t *testing.T) {
	for _, tc := range htj2kFixtures {
		t.Run(tc.file, func(t *testing.T) {
			pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", tc.file))
			if err != nil {
				t.Fatalf("ReadPixelData: %v", err)
			}
			if pd.Geometry.TransferSyntax != tc.ts {
				t.Fatalf("transfer syntax = %s, want %s", pd.Geometry.TransferSyntax, tc.ts)
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
		})
	}
}

// TestHTJ2KTranscodeToNative decodes an HTJ2K fixture and transcodes it to
// uncompressed Explicit VR Little Endian, confirming the native frame count and
// length match the geometry through the Transcode path.
func TestHTJ2KTranscodeToNative(t *testing.T) {
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "HTJ2KLossless_08_RGB.dcm"))
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

// TestHTJ2KIsDecodeOnly checks each HTJ2K codec rejects encode with the typed
// ErrEncodeUnsupported (this OpenJPEG build has no HT encoder).
func TestHTJ2KIsDecodeOnly(t *testing.T) {
	geom := PixelGeometry{Rows: 8, Columns: 8, SamplesPerPixel: 1, BitsAllocated: 8, BitsStored: 8}
	for _, ts := range []TransferSyntax{HTJ2KLossless, HTJ2KLosslessRPCL, HTJ2K} {
		c, ok := lookupCodec(ts)
		if !ok {
			t.Fatalf("no codec registered for %s", ts)
		}
		if _, err := c.Encode(make([]byte, geom.FrameLength()), geom); !errors.Is(err, ErrEncodeUnsupported) {
			t.Errorf("%s Encode error = %v, want ErrEncodeUnsupported", ts, err)
		}
	}
}
