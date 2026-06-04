//go:build cgo && dicom_charls

package dicom

import "testing"

// charlsDecodeFixtures are the vendored JPEG-LS Lossless and Near-Lossless fixtures the
// CharLS decode benchmark walks, paired with their transfer syntax.
var charlsDecodeFixtures = []struct {
	file string
	ts   TransferSyntax
}{
	{"MR_small_jpeg_ls_lossless.dcm", JPEGLSLossless},
	{"JPEGLSNearLossless_08.dcm", JPEGLSNearLossless},
	{"JPEGLSNearLossless_16.dcm", JPEGLSNearLossless},
	{"SC_rgb_jls_lossy_sample.dcm", JPEGLSNearLossless},
}

// BenchmarkCharLSCodecDecode measures the CharLS-backed frame decode hot path across the
// JPEG-LS fixtures. The encoded frames are read once outside the timed loop; only
// codec.Decode is timed.
func BenchmarkCharLSCodecDecode(b *testing.B) {
	for _, tc := range charlsDecodeFixtures {
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

// BenchmarkCharLSCodecEncode measures the CharLS JPEG-LS Lossless frame encode hot path. The
// lossless fixture is decoded once to native frames outside the timed loop; only codec.Encode
// is timed (Near-Lossless is decode-only and has no encode benchmark).
func BenchmarkCharLSCodecEncode(b *testing.B) {
	pd := readPixelDataBench(b, "MR_small_jpeg_ls_lossless.dcm")
	codec, ok := lookupCodec(JPEGLSLossless)
	if !ok || !codec.CanEncode() {
		b.Fatalf("JPEG-LS Lossless codec unavailable or decode-only")
	}
	native := decodeNativeFrames(b, pd)
	b.ReportAllocs()
	for b.Loop() {
		for _, frame := range native {
			if _, err := codec.Encode(frame, pd.Geometry); err != nil {
				b.Fatalf("encode: %v", err)
			}
		}
	}
}
