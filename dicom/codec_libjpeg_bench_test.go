//go:build cgo && dicom_libjpeg

package dicom

import "testing"

// libjpegDecodeFixtures are the vendored JPEG Baseline, Extended, and Lossless fixtures the
// libjpeg-turbo decode benchmark walks, paired with their transfer syntax. The libjpeg
// codecs are all decode-only, so there is no encode benchmark for this tag.
var libjpegDecodeFixtures = []struct {
	file string
	ts   TransferSyntax
}{
	{"SC_jpeg_no_color_transform.dcm", JPEGBaseline8Bit},
	{"JPGExtended.dcm", JPEGExtended12Bit},
	{"JPGLosslessP14SV1_1s_1f_8b.dcm", JPEGLosslessSV1},
}

// BenchmarkLibJPEGCodecDecode measures the libjpeg-turbo-backed frame decode hot path across
// the JPEG Baseline / Extended / Lossless fixtures. The encoded frames are read once outside
// the timed loop; only codec.Decode is timed.
func BenchmarkLibJPEGCodecDecode(b *testing.B) {
	for _, tc := range libjpegDecodeFixtures {
		b.Run(tc.file, func(b *testing.B) {
			pd := readPixelDataBench(b, tc.file)
			if pd.Geometry.TransferSyntax != tc.ts {
				b.Fatalf("transfer syntax = %s, want %s", pd.Geometry.TransferSyntax, tc.ts)
			}
			codec, ok := lookupCodec(tc.ts)
			if !ok {
				b.Fatalf("no codec registered for %s", tc.ts)
			}
			frames := encodedFrames(b, pd)
			b.ReportAllocs()
			for b.Loop() {
				for _, enc := range frames {
					if _, err := codec.Decode(enc, pd.Geometry); err != nil {
						b.Fatalf("decode: %v", err)
					}
				}
			}
		})
	}
}
