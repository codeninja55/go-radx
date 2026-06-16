package dicom

import (
	"bytes"
	"testing"
)

// TestApplyColorLUT8Bit expands a tiny PALETTE COLOR frame with 8-bit LUT entries
// packed two-per-word (the common PS3.5 form) and asserts the exact RGB output. The
// LUTs map index i to a distinct per-channel value so a mis-wired channel is visible.
func TestApplyColorLUT8Bit(t *testing.T) {
	ds := NewDataSet()
	// 4 entries, first mapped value 0, 8 bits per entry.
	ds.Set(Element{Tag: TagRedPaletteColorLookupTableDescriptor, VR: VRUS, Value: NewInts(VRUS, 4, 0, 8)})
	ds.Set(Element{Tag: TagGreenPaletteColorLookupTableDescriptor, VR: VRUS, Value: NewInts(VRUS, 4, 0, 8)})
	ds.Set(Element{Tag: TagBluePaletteColorLookupTableDescriptor, VR: VRUS, Value: NewInts(VRUS, 4, 0, 8)})
	// Entries 0..3 packed two-per-16-bit-word, low byte first: value in low byte, 0 high.
	ds.Set(Element{Tag: TagRedPaletteColorLookupTableData, VR: VROW, Value: NewBytes(VROW, []byte{10, 0, 20, 0, 30, 0, 40, 0})})
	ds.Set(Element{Tag: TagGreenPaletteColorLookupTableData, VR: VROW, Value: NewBytes(VROW, []byte{50, 0, 60, 0, 70, 0, 80, 0})})
	ds.Set(Element{Tag: TagBluePaletteColorLookupTableData, VR: VROW, Value: NewBytes(VROW, []byte{90, 0, 100, 0, 110, 0, 120, 0})})

	geom := PixelGeometry{Rows: 1, Columns: 4, BitsAllocated: 8, PhotometricInterpretation: "PALETTE COLOR"}
	frame := Frame{Pixels: []byte{0, 1, 2, 3}}

	got, err := ApplyColorLUT(frame, ds, geom)
	if err != nil {
		t.Fatalf("ApplyColorLUT: %v", err)
	}
	want := []byte{
		10, 50, 90, // index 0
		20, 60, 100, // index 1
		30, 70, 110, // index 2
		40, 80, 120, // index 3
	}
	if !bytes.Equal(got, want) {
		t.Errorf("8-bit palette\n got=%v\nwant=%v", got, want)
	}
}

// TestApplyColorLUT16Bit uses 16-bit LUT entries and a non-zero first-mapped value,
// asserting the high-byte downscale and the descriptor offset.
func TestApplyColorLUT16Bit(t *testing.T) {
	ds := NewDataSet()
	// 3 entries, first mapped value 5, 16 bits per entry.
	ds.Set(Element{Tag: TagRedPaletteColorLookupTableDescriptor, VR: VRUS, Value: NewInts(VRUS, 3, 5, 16)})
	ds.Set(Element{Tag: TagGreenPaletteColorLookupTableDescriptor, VR: VRUS, Value: NewInts(VRUS, 3, 5, 16)})
	ds.Set(Element{Tag: TagBluePaletteColorLookupTableDescriptor, VR: VRUS, Value: NewInts(VRUS, 3, 5, 16)})
	// 16-bit entries little-endian: 0x1000->high 0x10=16, 0x2000->32, 0x3000->48.
	ds.Set(Element{Tag: TagRedPaletteColorLookupTableData, VR: VROW, Value: NewBytes(VROW, []byte{0x00, 0x10, 0x00, 0x20, 0x00, 0x30})})
	ds.Set(Element{Tag: TagGreenPaletteColorLookupTableData, VR: VROW, Value: NewBytes(VROW, []byte{0x00, 0x40, 0x00, 0x50, 0x00, 0x60})})
	ds.Set(Element{Tag: TagBluePaletteColorLookupTableData, VR: VROW, Value: NewBytes(VROW, []byte{0x00, 0x70, 0x00, 0x80, 0x00, 0x90})})

	geom := PixelGeometry{Rows: 1, Columns: 4, BitsAllocated: 16, PhotometricInterpretation: "PALETTE COLOR"}
	// Pixel indices (16-bit LE): 4 (below first-mapped -> clamp to entry 0),
	// 5 -> entry 0, 6 -> entry 1, 99 (above last -> clamp to entry 2).
	frame := Frame{Pixels: []byte{4, 0, 5, 0, 6, 0, 99, 0}}

	got, err := ApplyColorLUT(frame, ds, geom)
	if err != nil {
		t.Fatalf("ApplyColorLUT: %v", err)
	}
	want := []byte{
		0x10, 0x40, 0x70, // index 4 clamps to entry 0
		0x10, 0x40, 0x70, // index 5 -> entry 0
		0x20, 0x50, 0x80, // index 6 -> entry 1
		0x30, 0x60, 0x90, // index 99 clamps to entry 2
	}
	if !bytes.Equal(got, want) {
		t.Errorf("16-bit palette\n got=%v\nwant=%v", got, want)
	}
}

// TestApplyColorLUTSignedDescriptorCount verifies the descriptor[0] unsigned-fold
// quirk: a signed VR may store the 65536-entry count as -1 (or pydicom-style 0 means
// 65536). A count of 0 here resolves to a full 65536-entry table.
func TestApplyColorLUTZeroCountIs65536(t *testing.T) {
	lut, err := paletteLUTFromCount(t, 0)
	if err != nil {
		t.Fatalf("readPaletteLUT: %v", err)
	}
	if len(lut.entries) != 65536 {
		t.Errorf("count 0 should mean 65536 entries, got %d", len(lut.entries))
	}
}

func paletteLUTFromCount(t *testing.T, count int64) (paletteLUT, error) {
	t.Helper()
	ds := NewDataSet()
	ds.Set(Element{Tag: TagRedPaletteColorLookupTableDescriptor, VR: VRUS, Value: NewInts(VRUS, count, 0, 8)})
	ds.Set(Element{Tag: TagRedPaletteColorLookupTableData, VR: VROW, Value: NewBytes(VROW, make([]byte, 131072))})
	return readPaletteLUT(ds, TagRedPaletteColorLookupTableDescriptor, TagRedPaletteColorLookupTableData)
}

// TestYBRFullToRGBExact asserts the PS3.3 C.7.6.3.1.2 inverse equations pixel-exactly.
// Mid-gray (Y=128,Cb=128,Cr=128) must round-trip to neutral gray (128,128,128).
func TestYBRFullToRGBExact(t *testing.T) {
	geom := PixelGeometry{Rows: 1, Columns: 1, PlanarConfiguration: 0}
	frame := Frame{Pixels: []byte{128, 128, 128}}
	out, err := ConvertColorSpace(frame, ColorSpaceYBRFull, ColorSpaceRGB, geom)
	if err != nil {
		t.Fatalf("ConvertColorSpace: %v", err)
	}
	want := []byte{128, 128, 128}
	if !bytes.Equal(out.Pixels, want) {
		t.Errorf("YBR_FULL gray -> RGB\n got=%v\nwant=%v", out.Pixels, want)
	}
}

// TestRGBToYBRFullExact asserts the PS3.3 C.7.6.3.1.2 forward equations pixel-exactly
// for the primary colours. Hand-computed:
//
//	red  (255,0,0):   Y=floor(0.2990*255+0.5)=76,  Cb=floor(-0.1687*255+128+0.5)=85,  Cr=floor(0.5*255+128+0.5)=256->255
//	green(0,255,0):   Y=floor(0.5870*255+0.5)=150, Cb=floor(-0.3313*255+128+0.5)=44,  Cr=floor(-0.4187*255+128+0.5)=21
//	blue (0,0,255):   Y=floor(0.1140*255+0.5)=29,  Cb=floor(0.5*255+128+0.5)=256->255, Cr=floor(-0.0813*255+128+0.5)=107
func TestRGBToYBRFullExact(t *testing.T) {
	geom := PixelGeometry{Rows: 1, Columns: 3, PlanarConfiguration: 0}
	frame := Frame{Pixels: []byte{
		255, 0, 0,
		0, 255, 0,
		0, 0, 255,
	}}
	out, err := ConvertColorSpace(frame, ColorSpaceRGB, ColorSpaceYBRFull, geom)
	if err != nil {
		t.Fatalf("ConvertColorSpace: %v", err)
	}
	want := []byte{
		76, 85, 255,
		150, 44, 21,
		29, 255, 107,
	}
	if !bytes.Equal(out.Pixels, want) {
		t.Errorf("RGB primaries -> YBR_FULL\n got=%v\nwant=%v", out.Pixels, want)
	}
}

// TestConvertColorSpacePlanar verifies PlanarConfiguration 1 (RRR...GGG...BBB) is
// handled: a planar RGB frame converts to the same YBR_FULL as its interleaved twin.
func TestConvertColorSpacePlanar(t *testing.T) {
	geom0 := PixelGeometry{Rows: 1, Columns: 2, PlanarConfiguration: 0}
	geom1 := PixelGeometry{Rows: 1, Columns: 2, PlanarConfiguration: 1}
	interleaved := Frame{Pixels: []byte{255, 0, 0, 0, 255, 0}} // pixel0 red, pixel1 green
	planar := Frame{Pixels: []byte{255, 0, 0, 255, 0, 0}}      // R:255,0  G:0,255  B:0,0
	a, err := ConvertColorSpace(interleaved, ColorSpaceRGB, ColorSpaceYBRFull, geom0)
	if err != nil {
		t.Fatalf("interleaved: %v", err)
	}
	b, err := ConvertColorSpace(planar, ColorSpaceRGB, ColorSpaceYBRFull, geom1)
	if err != nil {
		t.Fatalf("planar: %v", err)
	}
	if !bytes.Equal(a.Pixels, b.Pixels) {
		t.Errorf("planar config 1 should match interleaved\n interleaved=%v\n planar=%v", a.Pixels, b.Pixels)
	}
}

// TestYBRFull422ToRGB verifies 4:2:2 horizontal chroma reconstruction. A 1x2 frame
// packs Y0,Y1,Cb,Cr; both pixels share the chroma. With Cb=Cr=128 the result is two
// gray pixels at their own luma.
func TestYBRFull422ToRGB(t *testing.T) {
	geom := PixelGeometry{Rows: 1, Columns: 2}
	// Y0=128, Y1=200, Cb=128, Cr=128 -> both neutral chroma, luma passes through.
	frame := Frame{Pixels: []byte{128, 200, 128, 128}}
	out, err := ConvertColorSpace(frame, ColorSpaceYBRFull422, ColorSpaceRGB, geom)
	if err != nil {
		t.Fatalf("ConvertColorSpace 422: %v", err)
	}
	want := []byte{128, 128, 128, 200, 200, 200}
	if !bytes.Equal(out.Pixels, want) {
		t.Errorf("YBR_FULL_422 -> RGB\n got=%v\nwant=%v", out.Pixels, want)
	}
}

// TestConvertColorSpaceUnsupported confirms out-of-scope pairs fail closed with a typed
// error rather than returning a silently wrong frame.
func TestConvertColorSpaceUnsupported(t *testing.T) {
	geom := PixelGeometry{Rows: 1, Columns: 1}
	frame := Frame{Pixels: []byte{1, 2, 3}}
	_, err := ConvertColorSpace(frame, ColorSpace("YBR_PARTIAL_420"), ColorSpaceRGB, geom)
	if err == nil {
		t.Fatal("expected unsupported-conversion error, got nil")
	}
}

// TestRGBYBRRoundTrip checks the forward/inverse pair is near-identity (within the
// quantisation tolerance the integer rounding allows) for a spread of colours.
func TestRGBYBRRoundTrip(t *testing.T) {
	geom := PixelGeometry{Rows: 1, Columns: 4}
	frame := Frame{Pixels: []byte{
		10, 20, 30,
		200, 100, 50,
		0, 0, 0,
		255, 255, 255,
	}}
	ybr, err := ConvertColorSpace(frame, ColorSpaceRGB, ColorSpaceYBRFull, geom)
	if err != nil {
		t.Fatalf("forward: %v", err)
	}
	back, err := ConvertColorSpace(ybr, ColorSpaceYBRFull, ColorSpaceRGB, geom)
	if err != nil {
		t.Fatalf("inverse: %v", err)
	}
	for i := range frame.Pixels {
		d := int(frame.Pixels[i]) - int(back.Pixels[i])
		if d < -2 || d > 2 {
			t.Errorf("round-trip drift at byte %d: in=%d out=%d", i, frame.Pixels[i], back.Pixels[i])
		}
	}
}
