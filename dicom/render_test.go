package dicom

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"path/filepath"
	"testing"
)

// le16Frame packs 16-bit stored values as little-endian frame bytes.
func le16Frame(vals ...uint16) []byte {
	out := make([]byte, len(vals)*2)
	for i, v := range vals {
		out[i*2] = byte(v)
		out[i*2+1] = byte(v >> 8)
	}
	return out
}

// grayPix returns the Gray image's pixel bytes row-major, failing the test for a
// non-Gray image.
func grayPix(t *testing.T, img image.Image) []byte {
	t.Helper()
	g, ok := img.(*image.Gray)
	if !ok {
		t.Fatalf("image is %T, want *image.Gray", img)
	}
	return g.Pix
}

func TestRenderFrameMonochrome2Windowing(t *testing.T) {
	// PS3.3 C.11.2.1.2.1 LINEAR c=128 w=256 over 0..255 output: stored 0 clamps to
	// 0, 128 maps to 128, 255 maps to 255, 256 clamps to 255 (the same vectors as
	// TestApplyWindowingLinear, carried through to gray bytes).
	ds := NewDataSet()
	setDS(t, ds, TagWindowCenter, "128")
	setDS(t, ds, TagWindowWidth, "256")
	geom := PixelGeometry{
		Rows: 1, Columns: 4, SamplesPerPixel: 1, BitsAllocated: 16,
		PhotometricInterpretation: "MONOCHROME2", TransferSyntax: ExplicitVRLittleEndian,
	}
	img, err := RenderFrame(Frame{Pixels: le16Frame(0, 128, 255, 256)}, ds, geom)
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	want := []byte{0, 128, 255, 255}
	if got := grayPix(t, img); !bytes.Equal(got, want) {
		t.Errorf("MONOCHROME2 windowed\n got=%v\nwant=%v", got, want)
	}
}

func TestRenderFrameMonochrome1Inverts(t *testing.T) {
	// MONOCHROME1: lowest display value is white (PS3.3 C.7.6.3.1.2), so the gray
	// bytes are the MONOCHROME2 render inverted.
	ds := NewDataSet()
	setDS(t, ds, TagWindowCenter, "128")
	setDS(t, ds, TagWindowWidth, "256")
	geom := PixelGeometry{
		Rows: 1, Columns: 4, SamplesPerPixel: 1, BitsAllocated: 16,
		PhotometricInterpretation: "MONOCHROME1", TransferSyntax: ExplicitVRLittleEndian,
	}
	img, err := RenderFrame(Frame{Pixels: le16Frame(0, 128, 255, 256)}, ds, geom)
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	want := []byte{255, 127, 0, 0}
	if got := grayPix(t, img); !bytes.Equal(got, want) {
		t.Errorf("MONOCHROME1 inverted\n got=%v\nwant=%v", got, want)
	}
}

func TestRenderFrameModalityRescaleThenWindow(t *testing.T) {
	// CT-style pipeline ordering (PS3.3 C.11): rescale to HU first, then window.
	// Stored 0 -> -1024 HU sits at the LINEAR clamp edge (yMin); stored 1024 -> 0 HU
	// maps to ((0+0.5)/2047+0.5)*255 = 127.56 -> 128.
	ds := NewDataSet()
	setDS(t, ds, TagRescaleSlope, "1")
	setDS(t, ds, TagRescaleIntercept, "-1024")
	setDS(t, ds, TagWindowCenter, "0")
	setDS(t, ds, TagWindowWidth, "2048")
	geom := PixelGeometry{
		Rows: 1, Columns: 2, SamplesPerPixel: 1, BitsAllocated: 16,
		PhotometricInterpretation: "MONOCHROME2", TransferSyntax: ExplicitVRLittleEndian,
	}
	img, err := RenderFrame(Frame{Pixels: le16Frame(0, 1024)}, ds, geom)
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	want := []byte{0, 128}
	if got := grayPix(t, img); !bytes.Equal(got, want) {
		t.Errorf("rescale+window\n got=%v\nwant=%v", got, want)
	}
}

func TestRenderFrameAutoWindowMinMax(t *testing.T) {
	// No Window Center/Width and no VOI LUT Sequence: the full stored range is
	// stretched linearly, min -> 0 and max -> 255 (50/100*255 = 127.5 -> 128).
	ds := NewDataSet()
	geom := PixelGeometry{
		Rows: 1, Columns: 3, SamplesPerPixel: 1, BitsAllocated: 8,
		PhotometricInterpretation: "MONOCHROME2", TransferSyntax: ExplicitVRLittleEndian,
	}
	img, err := RenderFrame(Frame{Pixels: []byte{0, 50, 100}}, ds, geom)
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	want := []byte{0, 128, 255}
	if got := grayPix(t, img); !bytes.Equal(got, want) {
		t.Errorf("auto window\n got=%v\nwant=%v", got, want)
	}
}

func TestRenderFrameAutoWindowUniformFrameIsBlack(t *testing.T) {
	// A flat frame has no range to stretch; it renders black rather than dividing
	// by zero.
	ds := NewDataSet()
	geom := PixelGeometry{
		Rows: 1, Columns: 2, SamplesPerPixel: 1, BitsAllocated: 8,
		PhotometricInterpretation: "MONOCHROME2", TransferSyntax: ExplicitVRLittleEndian,
	}
	img, err := RenderFrame(Frame{Pixels: []byte{7, 7}}, ds, geom)
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	want := []byte{0, 0}
	if got := grayPix(t, img); !bytes.Equal(got, want) {
		t.Errorf("uniform frame\n got=%v\nwant=%v", got, want)
	}
}

func TestRenderFrameSignedPixelsAutoWindow(t *testing.T) {
	// PixelRepresentation 1: stored words are two's-complement, so -100 is the
	// stretch minimum and +100 the maximum.
	ds := NewDataSet()
	geom := PixelGeometry{
		Rows: 1, Columns: 2, SamplesPerPixel: 1, BitsAllocated: 16,
		PixelRepresentation:       1,
		PhotometricInterpretation: "MONOCHROME2", TransferSyntax: ExplicitVRLittleEndian,
	}
	pixels := le16Frame(uint16(0xFF9C), 100) // int16(-100), int16(100)
	img, err := RenderFrame(Frame{Pixels: pixels}, ds, geom)
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	want := []byte{0, 255}
	if got := grayPix(t, img); !bytes.Equal(got, want) {
		t.Errorf("signed auto window\n got=%v\nwant=%v", got, want)
	}
}

func TestRenderFrameVOILUTSequenceNormalized(t *testing.T) {
	// No Window Center/Width: the VOI LUT Sequence table applies and its output is
	// normalised over the table's own range (10..40 here), so the four entries land
	// on 0, 85, 170, 255 exactly.
	ds := NewDataSet()
	seq := NewSequence(lutItem([]int64{4, 0, 8}, []int64{10, 20, 30, 40}))
	ds.Set(Element{Tag: TagVOILUTSequence, VR: VRSQ, Value: NewSequenceValue(seq)})
	geom := PixelGeometry{
		Rows: 1, Columns: 4, SamplesPerPixel: 1, BitsAllocated: 8,
		PhotometricInterpretation: "MONOCHROME2", TransferSyntax: ExplicitVRLittleEndian,
	}
	img, err := RenderFrame(Frame{Pixels: []byte{0, 1, 2, 3}}, ds, geom)
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	want := []byte{0, 85, 170, 255}
	if got := grayPix(t, img); !bytes.Equal(got, want) {
		t.Errorf("VOI LUT table\n got=%v\nwant=%v", got, want)
	}
}

func TestRenderFrameVOILUTTakesPrecedenceOverWindow(t *testing.T) {
	// The VOI LUT Sequence table wins over an explicit Window Center/Width pair,
	// matching ApplyVOILUT's precedence. The table maps stored 0 and 3 to its range
	// extremes (0 and 255 after normalisation); the c=128/w=256 window would leave
	// them near black (0 and 3).
	ds := NewDataSet()
	setDS(t, ds, TagWindowCenter, "128")
	setDS(t, ds, TagWindowWidth, "256")
	seq := NewSequence(lutItem([]int64{4, 0, 8}, []int64{10, 20, 30, 40}))
	ds.Set(Element{Tag: TagVOILUTSequence, VR: VRSQ, Value: NewSequenceValue(seq)})
	geom := PixelGeometry{
		Rows: 1, Columns: 2, SamplesPerPixel: 1, BitsAllocated: 16,
		PhotometricInterpretation: "MONOCHROME2", TransferSyntax: ExplicitVRLittleEndian,
	}
	img, err := RenderFrame(Frame{Pixels: le16Frame(0, 3)}, ds, geom)
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	want := []byte{0, 255}
	if got := grayPix(t, img); !bytes.Equal(got, want) {
		t.Errorf("VOI LUT precedence\n got=%v\nwant=%v", got, want)
	}
}

func TestRenderFramePaletteColor(t *testing.T) {
	// The 8-bit palette from TestApplyColorLUT8Bit, carried through to NRGBA pixels.
	ds := NewDataSet()
	ds.Set(Element{Tag: TagRedPaletteColorLookupTableDescriptor, VR: VRUS, Value: NewInts(VRUS, 4, 0, 8)})
	ds.Set(Element{Tag: TagGreenPaletteColorLookupTableDescriptor, VR: VRUS, Value: NewInts(VRUS, 4, 0, 8)})
	ds.Set(Element{Tag: TagBluePaletteColorLookupTableDescriptor, VR: VRUS, Value: NewInts(VRUS, 4, 0, 8)})
	ds.Set(Element{Tag: TagRedPaletteColorLookupTableData, VR: VROW, Value: NewBytes(VROW, []byte{10, 0, 20, 0, 30, 0, 40, 0})})
	ds.Set(Element{Tag: TagGreenPaletteColorLookupTableData, VR: VROW, Value: NewBytes(VROW, []byte{50, 0, 60, 0, 70, 0, 80, 0})})
	ds.Set(Element{Tag: TagBluePaletteColorLookupTableData, VR: VROW, Value: NewBytes(VROW, []byte{90, 0, 100, 0, 110, 0, 120, 0})})
	geom := PixelGeometry{
		Rows: 1, Columns: 4, SamplesPerPixel: 1, BitsAllocated: 8,
		PhotometricInterpretation: "PALETTE COLOR", TransferSyntax: ExplicitVRLittleEndian,
	}
	img, err := RenderFrame(Frame{Pixels: []byte{0, 1, 2, 3}}, ds, geom)
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	rgba, ok := img.(*image.NRGBA)
	if !ok {
		t.Fatalf("image is %T, want *image.NRGBA", img)
	}
	want := []color.NRGBA{
		{10, 50, 90, 255}, {20, 60, 100, 255}, {30, 70, 110, 255}, {40, 80, 120, 255},
	}
	for x, w := range want {
		if got := rgba.NRGBAAt(x, 0); got != w {
			t.Errorf("pixel %d = %v, want %v", x, got, w)
		}
	}
}

func TestRenderFrameRGBInterleaved(t *testing.T) {
	ds := NewDataSet()
	geom := PixelGeometry{
		Rows: 1, Columns: 2, SamplesPerPixel: 3, BitsAllocated: 8,
		PhotometricInterpretation: "RGB", TransferSyntax: ExplicitVRLittleEndian,
	}
	img, err := RenderFrame(Frame{Pixels: []byte{255, 0, 0, 0, 255, 0}}, ds, geom)
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	rgba, ok := img.(*image.NRGBA)
	if !ok {
		t.Fatalf("image is %T, want *image.NRGBA", img)
	}
	if got := rgba.NRGBAAt(0, 0); got != (color.NRGBA{255, 0, 0, 255}) {
		t.Errorf("pixel 0 = %v, want red", got)
	}
	if got := rgba.NRGBAAt(1, 0); got != (color.NRGBA{0, 255, 0, 255}) {
		t.Errorf("pixel 1 = %v, want green", got)
	}
}

func TestRenderFrameRGBPlanar(t *testing.T) {
	// PlanarConfiguration 1 (RRR...GGG...BBB) renders the same pixels as its
	// interleaved twin.
	ds := NewDataSet()
	geom := PixelGeometry{
		Rows: 1, Columns: 2, SamplesPerPixel: 3, BitsAllocated: 8,
		PlanarConfiguration:       1,
		PhotometricInterpretation: "RGB", TransferSyntax: ExplicitVRLittleEndian,
	}
	img, err := RenderFrame(Frame{Pixels: []byte{255, 0, 0, 255, 0, 0}}, ds, geom)
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	rgba := img.(*image.NRGBA)
	if got := rgba.NRGBAAt(0, 0); got != (color.NRGBA{255, 0, 0, 255}) {
		t.Errorf("pixel 0 = %v, want red", got)
	}
	if got := rgba.NRGBAAt(1, 0); got != (color.NRGBA{0, 255, 0, 255}) {
		t.Errorf("pixel 1 = %v, want green", got)
	}
}

func TestRenderFrameYBRFull(t *testing.T) {
	// Neutral chroma (Cb=Cr=128) passes luma through: mid-gray in, mid-gray out.
	ds := NewDataSet()
	geom := PixelGeometry{
		Rows: 1, Columns: 1, SamplesPerPixel: 3, BitsAllocated: 8,
		PhotometricInterpretation: "YBR_FULL", TransferSyntax: ExplicitVRLittleEndian,
	}
	img, err := RenderFrame(Frame{Pixels: []byte{128, 128, 128}}, ds, geom)
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	rgba := img.(*image.NRGBA)
	if got := rgba.NRGBAAt(0, 0); got != (color.NRGBA{128, 128, 128, 255}) {
		t.Errorf("pixel = %v, want gray 128", got)
	}
}

func TestRenderFrameYBRFull422(t *testing.T) {
	// 4:2:2 packing Y0,Y1,Cb,Cr with neutral chroma: two gray pixels at their own
	// luma (the TestYBRFull422ToRGB vectors as image pixels).
	ds := NewDataSet()
	geom := PixelGeometry{
		Rows: 1, Columns: 2, SamplesPerPixel: 3, BitsAllocated: 8,
		PhotometricInterpretation: "YBR_FULL_422", TransferSyntax: ExplicitVRLittleEndian,
	}
	img, err := RenderFrame(Frame{Pixels: []byte{128, 200, 128, 128}}, ds, geom)
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	rgba := img.(*image.NRGBA)
	if got := rgba.NRGBAAt(0, 0); got != (color.NRGBA{128, 128, 128, 255}) {
		t.Errorf("pixel 0 = %v, want gray 128", got)
	}
	if got := rgba.NRGBAAt(1, 0); got != (color.NRGBA{200, 200, 200, 255}) {
		t.Errorf("pixel 1 = %v, want gray 200", got)
	}
}

func TestRenderFrameUnsupportedPhotometricInterpretation(t *testing.T) {
	ds := NewDataSet()
	for _, pi := range []string{"YBR_PARTIAL_420", "HSV", ""} {
		geom := PixelGeometry{
			Rows: 1, Columns: 1, SamplesPerPixel: 3, BitsAllocated: 8,
			PhotometricInterpretation: pi, TransferSyntax: ExplicitVRLittleEndian,
		}
		_, err := RenderFrame(Frame{Pixels: []byte{1, 2, 3}}, ds, geom)
		if err == nil {
			t.Fatalf("RenderFrame(%q) = nil error, want typed error", pi)
		}
		if _, ok := errors.AsType[*ValueError](err); !ok {
			t.Errorf("RenderFrame(%q) error is %T, want *ValueError", pi, err)
		}
	}
}

func TestRenderFrameMonoUnsupportedBitsAllocated(t *testing.T) {
	ds := NewDataSet()
	for _, bits := range []uint16{1, 32} {
		geom := PixelGeometry{
			Rows: 1, Columns: 8, SamplesPerPixel: 1, BitsAllocated: bits,
			PhotometricInterpretation: "MONOCHROME2", TransferSyntax: ExplicitVRLittleEndian,
		}
		_, err := RenderFrame(Frame{Pixels: make([]byte, 32)}, ds, geom)
		if err == nil {
			t.Fatalf("RenderFrame(BitsAllocated=%d) = nil error, want typed error", bits)
		}
		if _, ok := errors.AsType[*ValueError](err); !ok {
			t.Errorf("BitsAllocated=%d error is %T, want *ValueError", bits, err)
		}
	}
}

func TestRenderFrameShortMonoFrame(t *testing.T) {
	ds := NewDataSet()
	geom := PixelGeometry{
		Rows: 2, Columns: 2, SamplesPerPixel: 1, BitsAllocated: 16,
		PhotometricInterpretation: "MONOCHROME2", TransferSyntax: ExplicitVRLittleEndian,
	}
	_, err := RenderFrame(Frame{Pixels: []byte{0, 0}}, ds, geom)
	if err == nil {
		t.Fatal("short frame = nil error, want typed error")
	}
	if _, ok := errors.AsType[*ValueError](err); !ok {
		t.Errorf("short frame error is %T, want *ValueError", err)
	}
}

func TestRenderFrameColorRejectsNon8Bit(t *testing.T) {
	// A 16-bit colour frame sliced as bytes would render silently corrupt from the
	// first third of its buffer, so every colour path requires 8 BitsAllocated.
	ds := NewDataSet()
	for _, pi := range []string{"RGB", "YBR_FULL", "YBR_FULL_422"} {
		geom := PixelGeometry{
			Rows: 1, Columns: 2, SamplesPerPixel: 3, BitsAllocated: 16,
			PhotometricInterpretation: pi, TransferSyntax: ExplicitVRLittleEndian,
		}
		// 12 bytes satisfy a 3-bytes-per-pixel length check, so only the bit-depth
		// guard can reject the frame.
		_, err := RenderFrame(Frame{Pixels: make([]byte, 12)}, ds, geom)
		if err == nil {
			t.Fatalf("RenderFrame(%s, 16-bit) = nil error, want typed error", pi)
		}
		if _, ok := errors.AsType[*ValueError](err); !ok {
			t.Errorf("RenderFrame(%s, 16-bit) error is %T, want *ValueError", pi, err)
		}
	}
}

func TestRenderFrameUnusableWindowFallsBackToStretch(t *testing.T) {
	// Window Width 0 has no defined output (PS3.3 C.11.2.1.2.1). Rendering degrades
	// to the min/max stretch instead of refusing the frame.
	ds := NewDataSet()
	setDS(t, ds, TagWindowCenter, "40")
	setDS(t, ds, TagWindowWidth, "0")
	geom := PixelGeometry{
		Rows: 1, Columns: 3, SamplesPerPixel: 1, BitsAllocated: 8,
		PhotometricInterpretation: "MONOCHROME2", TransferSyntax: ExplicitVRLittleEndian,
	}
	img, err := RenderFrame(Frame{Pixels: []byte{0, 50, 100}}, ds, geom)
	if err != nil {
		t.Fatalf("RenderFrame with width-0 window: %v", err)
	}
	want := []byte{0, 128, 255}
	if got := grayPix(t, img); !bytes.Equal(got, want) {
		t.Errorf("degenerate window fallback\n got=%v\nwant=%v", got, want)
	}
}

func TestFrameSamplesMasksOverlayBits(t *testing.T) {
	// 12 bits stored in 16 allocated (HighBit 11): a legacy overlay plane packed in
	// bit 15 must not leak into the sample value (PS3.5 §8.1.1).
	geom := PixelGeometry{
		Rows: 1, Columns: 2, SamplesPerPixel: 1, BitsAllocated: 16,
		BitsStored: 12, HighBit: 11, TransferSyntax: ExplicitVRLittleEndian,
	}
	got, err := frameSamples(le16Frame(0x8123, 0x0123), geom)
	if err != nil {
		t.Fatalf("frameSamples: %v", err)
	}
	for i, want := range []float64{291, 291} { // 0x123 with and without the overlay bit
		if got[i] != want {
			t.Errorf("sample %d = %v, want %v", i, got[i], want)
		}
	}
}

func TestFrameSamplesSignExtendsFromBitsStored(t *testing.T) {
	// Signed 12-in-16: 0x0FFF is -1 in 12-bit two's-complement, not +4095; 0x07FF
	// stays the maximum positive 12-bit value.
	geom := PixelGeometry{
		Rows: 1, Columns: 2, SamplesPerPixel: 1, BitsAllocated: 16,
		BitsStored: 12, HighBit: 11, PixelRepresentation: 1,
		TransferSyntax: ExplicitVRLittleEndian,
	}
	got, err := frameSamples(le16Frame(0x0FFF, 0x07FF), geom)
	if err != nil {
		t.Fatalf("frameSamples: %v", err)
	}
	for i, want := range []float64{-1, 2047} {
		if got[i] != want {
			t.Errorf("sample %d = %v, want %v", i, got[i], want)
		}
	}
}

func TestRenderFrameAutoStretchExcludesPixelPadding(t *testing.T) {
	// PS3.3 C.7.5.1.1.2: stored values inside [Pixel Padding Value, Pixel Padding
	// Range Limit] carry no image information. Stored 0 and 5 are padding here; the
	// tissue range -100..300 HU (after the -1024 rescale) spans the full 0..255
	// output and padded pixels render black. Without the exclusion the stretch would
	// span -1024..300 and cram the tissue into the top of the range.
	ds := NewDataSet()
	setDS(t, ds, TagRescaleSlope, "1")
	setDS(t, ds, TagRescaleIntercept, "-1024")
	ds.Set(Element{Tag: TagPixelPaddingValue, VR: VRUS, Value: NewInts(VRUS, 0)})
	ds.Set(Element{Tag: TagPixelPaddingRangeLimit, VR: VRUS, Value: NewInts(VRUS, 10)})
	geom := PixelGeometry{
		Rows: 1, Columns: 5, SamplesPerPixel: 1, BitsAllocated: 16,
		PhotometricInterpretation: "MONOCHROME2", TransferSyntax: ExplicitVRLittleEndian,
	}
	img, err := RenderFrame(Frame{Pixels: le16Frame(0, 5, 924, 1124, 1324)}, ds, geom)
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	want := []byte{0, 0, 0, 128, 255}
	if got := grayPix(t, img); !bytes.Equal(got, want) {
		t.Errorf("padding exclusion\n got=%v\nwant=%v", got, want)
	}
}

func TestEncodePPM(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{255, 0, 0, 255})
	img.SetNRGBA(1, 0, color.NRGBA{0, 255, 0, 255})
	var buf bytes.Buffer
	if err := EncodePPM(&buf, img); err != nil {
		t.Fatalf("EncodePPM: %v", err)
	}
	want := append([]byte("P6\n2 1\n255\n"), 255, 0, 0, 0, 255, 0)
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("PPM bytes\n got=%q\nwant=%q", buf.Bytes(), want)
	}
}

func TestEncodePPMGray(t *testing.T) {
	// A gray image writes equal R,G,B triples: PPM P6 carries no grayscale form.
	img := image.NewGray(image.Rect(0, 0, 1, 1))
	img.SetGray(0, 0, color.Gray{Y: 128})
	var buf bytes.Buffer
	if err := EncodePPM(&buf, img); err != nil {
		t.Fatalf("EncodePPM: %v", err)
	}
	want := append([]byte("P6\n1 1\n255\n"), 128, 128, 128)
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("gray PPM bytes\n got=%q\nwant=%q", buf.Bytes(), want)
	}
}

func TestRenderFrameSCRGBFixture(t *testing.T) {
	// End-to-end: decode the uncompressed RGB fixture (100x100, ten 10-row colour
	// strips, Explicit VR Big Endian) and spot-check one pixel per strip. The
	// expected colours were read from the decoded native buffer of this vendored
	// pydicom fixture.
	f, err := ReadFile(filepath.Join("..", "testdata", "dicom", "SC_rgb_expb.dcm"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	pd, err := NewPixelData(f.DataSet, f.Meta.TransferSyntaxUID)
	if err != nil {
		t.Fatalf("NewPixelData: %v", err)
	}
	var img image.Image
	for frame, err := range pd.Frames() {
		if err != nil {
			t.Fatalf("frame: %v", err)
		}
		img, err = RenderFrame(frame, f.DataSet, pd.Geometry)
		if err != nil {
			t.Fatalf("RenderFrame: %v", err)
		}
		break
	}
	rgba, ok := img.(*image.NRGBA)
	if !ok {
		t.Fatalf("image is %T, want *image.NRGBA", img)
	}
	want := []color.NRGBA{
		{255, 0, 0, 255}, {255, 128, 128, 255},
		{0, 255, 0, 255}, {128, 255, 128, 255},
		{0, 0, 255, 255}, {128, 128, 255, 255},
		{0, 0, 0, 255}, {64, 64, 64, 255},
		{192, 192, 192, 255}, {255, 255, 255, 255},
	}
	for i, w := range want {
		y := 5 + i*10 // one sample row per strip
		if got := rgba.NRGBAAt(50, y); got != w {
			t.Errorf("strip %d (row %d) = %v, want %v", i, y, got, w)
		}
	}
}
