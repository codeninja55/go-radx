package dicom

import (
	"path/filepath"
	"testing"
)

// part10DecodeFixtures are the uncompressed Part 10 fixtures the decode benchmark walks.
// Each is decoded through the always-available pure-Go dataset codec, so the set is the
// uncompressed transfer syntaxes plus the byte-order and overlay variants; the encapsulated
// JPEG-family fixtures are benchmarked behind their CGo build tags. The byte size of each
// fixture is reported via b.SetBytes so the result is a throughput, and b.ReportAllocs
// surfaces the Part 10 decode allocation profile (PRD §9.3 minimise allocations in hot paths).
var part10DecodeFixtures = []string{
	"liver.dcm",
	"MR2_UNCI.dcm",
	"SC_rgb_expb.dcm",
	"MR-SIEMENS-DICOM-WithOverlays.dcm",
	"basic-text-sr.dcm",
}

// BenchmarkReadFile measures the full Part 10 decode hot path: preamble, file-meta group,
// and main dataset parsed off disk through ReadFile, across representative uncompressed
// fixtures. It is the end-to-end counterpart to BenchmarkDecodeDataSet, which isolates the
// bare-dataset decode.
func BenchmarkReadFile(b *testing.B) {
	for _, name := range part10DecodeFixtures {
		b.Run(name, func(b *testing.B) {
			path := filepath.Join("..", "testdata", "dicom", name)
			info := statFixture(b, path)
			b.SetBytes(info)
			b.ReportAllocs()
			for b.Loop() {
				if _, err := ReadFile(path); err != nil {
					b.Fatalf("read %s: %v", name, err)
				}
			}
		})
	}
}

// BenchmarkRLECodecDecode measures the pure-Go RLE Lossless frame decode hot path. RLE is
// the one encapsulated codec available without a CGo build tag, so it is benchmarked in the
// default build. The encoded frames are read once outside the timed loop; only Decode is timed.
func BenchmarkRLECodecDecode(b *testing.B) {
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "liver_rle.dcm"))
	if err != nil {
		b.Fatalf("ReadPixelData: %v", err)
	}
	frames := encodedFrames(b, pd)
	codec := rleCodec{}
	b.ReportAllocs()
	for b.Loop() {
		for _, enc := range frames {
			if _, err := codec.Decode(enc, pd.Geometry); err != nil {
				b.Fatalf("decode: %v", err)
			}
		}
	}
}

// rleEncodeGeometries are conformant RLE geometries the encode benchmark exercises. The
// liver_rle.dcm fixture is a 1-bit segmentation that RLE only round-trips on decode (the
// encoder requires BitsAllocated 8 or 16 per PS3.5 Annex G), so the encode path is driven by
// synthetic conformant frames across the byte-per-sample and word-per-sample byte planes.
var rleEncodeGeometries = []struct {
	name string
	geom PixelGeometry
}{
	{"8-bit mono 256x256", PixelGeometry{Rows: 256, Columns: 256, SamplesPerPixel: 1, BitsAllocated: 8, BitsStored: 8, TransferSyntax: RLELossless}},
	{"16-bit mono 256x256", PixelGeometry{Rows: 256, Columns: 256, SamplesPerPixel: 1, BitsAllocated: 16, BitsStored: 16, TransferSyntax: RLELossless}},
	{"8-bit rgb 256x256", PixelGeometry{Rows: 256, Columns: 256, SamplesPerPixel: 3, BitsAllocated: 8, BitsStored: 8, PhotometricInterpretation: "RGB", TransferSyntax: RLELossless}},
}

// BenchmarkRLECodecEncode measures the pure-Go RLE Lossless frame encode hot path across
// conformant geometries. The synthetic frame is built once outside the timed loop; only
// Encode is timed.
func BenchmarkRLECodecEncode(b *testing.B) {
	codec := rleCodec{}
	for _, tc := range rleEncodeGeometries {
		b.Run(tc.name, func(b *testing.B) {
			frame := syntheticFrameRLE(tc.geom)
			b.SetBytes(int64(len(frame)))
			b.ReportAllocs()
			for b.Loop() {
				if _, err := codec.Encode(frame, tc.geom); err != nil {
					b.Fatalf("encode: %v", err)
				}
			}
		})
	}
}

// syntheticFrameRLE fills a frame of the geometry's FrameLength with deterministic samples
// that mix runs and literals, so the encoder exercises both PackBits paths rather than a
// trivially compressible constant frame.
func syntheticFrameRLE(geom PixelGeometry) []byte {
	out := make([]byte, geom.FrameLength())
	for i := range out {
		out[i] = byte((i / 7) % 251)
	}
	return out
}

// statFixture returns the on-disk byte size of the fixture at path for b.SetBytes.
func statFixture(b *testing.B, path string) int64 {
	b.Helper()
	f, err := ReadFile(path)
	if err != nil {
		b.Fatalf("stat-read %s: %v", path, err)
	}
	var buf countingWriter
	if err := Write(&buf, f); err != nil {
		b.Fatalf("stat-encode %s: %v", path, err)
	}
	return int64(buf)
}

// countingWriter discards its input and counts the bytes written, sizing the throughput
// denominator without holding the encoded form.
type countingWriter int

func (c *countingWriter) Write(p []byte) (int, error) {
	*c += countingWriter(len(p))
	return len(p), nil
}

// encodedFrames returns each frame's encoded bytes for an encapsulated PixelData, for a
// decode benchmark that times only the codec.
func encodedFrames(b *testing.B, pd *PixelData) [][]byte {
	b.Helper()
	frames, err := pd.frameEncodedBytes()
	if err != nil {
		b.Fatalf("frame encoded bytes: %v", err)
	}
	return frames
}
