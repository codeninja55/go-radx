//go:build cgo && dicom_openjpeg

package dicom

import "testing"

// openjpegDecodeFixtures are the vendored JPEG 2000 and High-Throughput JPEG 2000 fixtures
// the OpenJPEG decode benchmark walks, paired with their transfer syntax.
var openjpegDecodeFixtures = []struct {
	file string
	ts   TransferSyntax
}{
	{"liver_j2k.dcm", JPEG2000Lossless},
	{"HTJ2KLossless_08_RGB.dcm", HTJ2KLossless},
	{"HTJ2K_08_RGB.dcm", HTJ2K},
}

// BenchmarkOpenJPEGCodecDecode measures the OpenJPEG-backed frame decode hot path across the
// JPEG 2000 / HTJ2K fixtures. The encoded frames are read once outside the timed loop; only
// codec.Decode is timed.
func BenchmarkOpenJPEGCodecDecode(b *testing.B) {
	for _, tc := range openjpegDecodeFixtures {
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

// BenchmarkOpenJPEGCodecEncode measures the OpenJPEG JPEG 2000 Lossless frame encode hot
// path. The lossless liver fixture is decoded once to native frames outside the timed loop;
// only codec.Encode is timed.
func BenchmarkOpenJPEGCodecEncode(b *testing.B) {
	pd := readPixelDataBench(b, "liver_j2k.dcm")
	codec, ok := lookupCodec(JPEG2000Lossless)
	if !ok || !codec.CanEncode() {
		b.Fatalf("JPEG 2000 Lossless codec unavailable or decode-only")
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
