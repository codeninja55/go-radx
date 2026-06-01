//go:build cgo && dicom_libjpeg

package dicom

import (
	"bytes"
	"testing"
)

// syntheticSamples builds a deterministic, pseudo-random sample frame that exercises
// the full precision range. A prime stride spreads values so the codestream is not a
// flat field a predictor could trivially collapse. The byte layout matches what the
// decode path emits: one byte per sample for precision <= 8, little-endian uint16
// otherwise.
func syntheticSamples(width, height, numcomps, precision int) []byte {
	n := width * height * numcomps
	mod := 1 << precision
	if precision <= 8 {
		s := make([]byte, n)
		for i := range s {
			s[i] = byte((i*37 + 11) % mod)
		}
		return s
	}
	s := make([]byte, n*2)
	for i := 0; i < n; i++ {
		v := uint16((i*1103 + 17) % mod)
		s[i*2] = byte(v)
		s[i*2+1] = byte(v >> 8)
	}
	return s
}

// TestJPEGLosslessRoundTrip is the pixel-exact correctness proof for the lossless
// decode path at every supported precision. Because libjpeg-turbo's lossless process
// is reversible, decode(encode(x)) must equal x byte-for-byte; any divergence is a
// codec defect. This oracle needs no external reference library and covers the
// 16-bit decompress path no vendored fixture exercises, plus both the SV1 (.70) and
// general Process 14 (.57) predictor forms.
func TestJPEGLosslessRoundTrip(t *testing.T) {
	cases := []struct {
		name      string
		width     int
		height    int
		numcomps  int
		precision int
		psv       int // 1 = .70 (SV1); 2..7 = .57 (general Process 14)
	}{
		{"8bit_mono_sv1", 16, 12, 1, 8, 1},
		{"8bit_mono_psv5", 16, 12, 1, 8, 5},
		{"12bit_mono_sv1", 17, 9, 1, 12, 1},
		{"16bit_mono_sv1", 13, 7, 1, 16, 1},
		{"16bit_mono_psv6", 13, 7, 1, 16, 6},
		{"8bit_rgb_sv1", 10, 8, 3, 8, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			samples := syntheticSamples(tc.width, tc.height, tc.numcomps, tc.precision)
			codestream, err := encodeLosslessJPEG(samples, tc.width, tc.height, tc.numcomps, tc.precision, tc.psv)
			if err != nil {
				t.Fatalf("encodeLosslessJPEG: %v", err)
			}

			bitsAllocated := uint16(8)
			if tc.precision > 8 {
				bitsAllocated = 16
			}
			pi := "MONOCHROME2"
			if tc.numcomps == 3 {
				pi = "RGB"
			}
			geom := PixelGeometry{
				Rows:                      uint16(tc.height),
				Columns:                   uint16(tc.width),
				SamplesPerPixel:           uint16(tc.numcomps),
				BitsAllocated:             bitsAllocated,
				BitsStored:                uint16(tc.precision),
				HighBit:                   uint16(tc.precision - 1),
				NumberOfFrames:            1,
				PhotometricInterpretation: pi,
				TransferSyntax:            JPEGLosslessSV1,
			}

			codec := libjpegCodec{ts: JPEGLosslessSV1}
			got, err := codec.Decode(codestream, geom)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if len(got) != geom.FrameLength() {
				t.Fatalf("decoded length = %d, want FrameLength %d", len(got), geom.FrameLength())
			}
			if !bytes.Equal(got, samples) {
				t.Errorf("lossless round-trip is not byte-identical (precision %d, psv %d, %d bytes)",
					tc.precision, tc.psv, len(samples))
			}
		})
	}
}
