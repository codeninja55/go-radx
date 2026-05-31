package dicom

import (
	"bytes"
	"testing"
)

func TestPackBitsRoundTrip(t *testing.T) {
	cases := [][]byte{
		{},
		{0x00},
		{0x01, 0x02, 0x03},
		bytes.Repeat([]byte{0xAA}, 200),                  // long replicate run, capped at 128
		append(bytes.Repeat([]byte{0x01}, 5), 0x02, 0x03), // replicate then literal
		{1, 2, 2, 2, 3, 4, 5, 5, 5, 5, 5, 6},
	}
	for i, src := range cases {
		packed := packBits(src)
		got, err := unpackBits(packed, len(src))
		if err != nil {
			t.Fatalf("case %d unpackBits: %v", i, err)
		}
		if !bytes.Equal(got, src) {
			t.Errorf("case %d round-trip = %v, want %v", i, got, src)
		}
	}
}

// TestRLERoundTripPixelExact is the verification-gate round-trip: encode a frame to
// RLE, decode it back, and require the pixels to be byte-identical. Covers 8-bit
// mono, 16-bit mono, and 8-bit RGB (three byte planes).
func TestRLERoundTripPixelExact(t *testing.T) {
	tests := []struct {
		name string
		geom PixelGeometry
		make func(n int) []byte
	}{
		{
			name: "8-bit mono",
			geom: PixelGeometry{Rows: 8, Columns: 8, SamplesPerPixel: 1, BitsAllocated: 8},
			make: func(n int) []byte {
				b := make([]byte, n)
				for i := range b {
					b[i] = byte(i % 17)
				}
				return b
			},
		},
		{
			name: "16-bit mono",
			geom: PixelGeometry{Rows: 4, Columns: 4, SamplesPerPixel: 1, BitsAllocated: 16},
			make: func(n int) []byte {
				b := make([]byte, n)
				for i := 0; i < n; i += 2 {
					b[i] = byte(i)        // LSB
					b[i+1] = byte(i / 2)  // MSB
				}
				return b
			},
		},
		{
			name: "8-bit RGB interleaved",
			geom: PixelGeometry{Rows: 4, Columns: 4, SamplesPerPixel: 3, BitsAllocated: 8},
			make: func(n int) []byte {
				b := make([]byte, n)
				for i := 0; i < n; i += 3 {
					b[i] = byte((i / 3) % 256)
					b[i+1] = byte((i/3 + 85) % 256)
					b[i+2] = byte(255 - (i/3)%256)
				}
				return b
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			orig := tc.make(tc.geom.FrameLength())
			encoded, err := encodeRLEFrame(orig, tc.geom)
			if err != nil {
				t.Fatalf("encodeRLEFrame: %v", err)
			}
			decoded, err := decodeRLEFrame(encoded, tc.geom)
			if err != nil {
				t.Fatalf("decodeRLEFrame: %v", err)
			}
			if !bytes.Equal(decoded, orig) {
				t.Errorf("round-trip not pixel-exact:\n got %v\nwant %v", decoded, orig)
			}
		})
	}
}

func TestRLEEncodeRejectsMismatchedFrameLength(t *testing.T) {
	geom := PixelGeometry{Rows: 4, Columns: 4, SamplesPerPixel: 1, BitsAllocated: 8}
	if _, err := encodeRLEFrame(make([]byte, 10), geom); err == nil {
		t.Fatal("expected an error for a buffer that does not match FrameLength")
	}
}

func TestRLEEncodeRejectsBadBitsAllocated(t *testing.T) {
	geom := PixelGeometry{Rows: 4, Columns: 4, SamplesPerPixel: 1, BitsAllocated: 32}
	if _, err := encodeRLEFrame(make([]byte, geom.FrameLength()), geom); err == nil {
		t.Fatal("expected an error for BitsAllocated 32")
	}
}

// TestRLECodecEncodeDecodeThroughInterface drives encode and decode through the
// registered codec, the way the transcode path does.
func TestRLECodecEncodeDecodeThroughInterface(t *testing.T) {
	c, ok := lookupCodec(RLELossless)
	if !ok {
		t.Fatal("RLE codec not registered")
	}
	geom := PixelGeometry{Rows: 6, Columns: 6, SamplesPerPixel: 1, BitsAllocated: 16}
	orig := make([]byte, geom.FrameLength())
	for i := range orig {
		orig[i] = byte((i * 7) % 251)
	}
	encoded, err := c.Encode(orig, geom)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	decoded, err := c.Decode(encoded, geom)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !bytes.Equal(decoded, orig) {
		t.Error("codec round-trip not pixel-exact")
	}
}
