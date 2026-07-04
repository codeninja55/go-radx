package dicom

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"image"
	"io"
)

// RenderFrame turns one decoded frame into an 8-bit consumer image using the
// presentation pipeline this package already ships. frame is one decoded frame from
// PixelData.Frames; ds supplies the presentation attributes (rescale, windows, LUTs,
// palettes); geom is the frame's PixelData.Geometry.
//
// The photometric interpretations rendered, per PS3.3 C.7.6.3.1.2:
//
//	MONOCHROME2/MONOCHROME1: ApplyModalityLUT, then a value-of-interest mapping to
//	  the 0..255 display range, into *image.Gray (MONOCHROME1 inverted: lowest
//	  value renders white).
//	PALETTE COLOR:           ApplyColorLUT into *image.NRGBA.
//	RGB / YBR_FULL / YBR_FULL_422: ConvertColorSpace to RGB into *image.NRGBA.
//
// The value-of-interest mapping follows ApplyVOILUT's precedence: the first VOI LUT
// Sequence table when present, normalised over the table's observed output range,
// else the first explicit Window Center/Width pair (through ApplyWindowing with the
// dataset's VOI LUT Function), else a min/max stretch of the frame that excludes
// pixel-padding values (the dcm2pnm min-max default). The stretch also serves when
// the declared window is unusable (e.g. width 0). Any unsupported photometric
// interpretation, bit depth, or short frame is a typed *ValueError, never a silently
// wrong image.
func RenderFrame(frame Frame, ds *DataSet, geom PixelGeometry) (image.Image, error) {
	switch geom.PhotometricInterpretation {
	case "MONOCHROME2":
		return renderMonochrome(frame, ds, geom, false)
	case "MONOCHROME1":
		return renderMonochrome(frame, ds, geom, true)
	case "PALETTE COLOR":
		rgb, err := ApplyColorLUT(frame, ds, geom, geom.TransferSyntax.byteOrder())
		if err != nil {
			return nil, err
		}
		return rgbImage(rgb, geom), nil
	case string(ColorSpaceRGB):
		if err := requireEightBitColor(geom); err != nil {
			return nil, err
		}
		count := int(geom.Rows) * int(geom.Columns)
		need := count * 3
		if len(frame.Pixels) < need {
			return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "colour frame shorter than 3 samples per pixel"}
		}
		// Interleaved data feeds the image directly; only planar layout reorders.
		rgb := frame.Pixels[:need]
		if geom.PlanarConfiguration != 0 {
			var err error
			rgb, err = deinterleaveTriples(frame.Pixels, count, geom.PlanarConfiguration)
			if err != nil {
				return nil, err
			}
		}
		return rgbImage(rgb, geom), nil
	case string(ColorSpaceYBRFull), string(ColorSpaceYBRFull422):
		if err := requireEightBitColor(geom); err != nil {
			return nil, err
		}
		converted, err := ConvertColorSpace(frame, ColorSpace(geom.PhotometricInterpretation), ColorSpaceRGB, geom)
		if err != nil {
			return nil, err
		}
		return rgbImage(converted.Pixels, geom), nil
	default:
		return nil, &ValueError{
			Tag: TagPhotometricInterpretation, VR: VRCS,
			Msg: fmt.Sprintf("unsupported Photometric Interpretation %q for rendering", geom.PhotometricInterpretation),
		}
	}
}

// requireEightBitColor rejects colour frames whose samples are not single bytes:
// every colour path here (YBR maths, triple deinterleaving, image packing) is defined
// on 8-bit samples, and slicing a 16-bit frame as bytes would render silently
// corrupt from the first third of its buffer.
func requireEightBitColor(geom PixelGeometry) error {
	if geom.BitsAllocated == 8 {
		return nil
	}
	return &ValueError{
		Tag: TagBitsAllocated, VR: VRUS,
		Msg: fmt.Sprintf("%s rendering supports 8 BitsAllocated, got %d", geom.PhotometricInterpretation, geom.BitsAllocated),
	}
}

// renderMonochrome runs the PS3.3 C.11 grayscale pipeline for one frame: stored
// samples -> Modality LUT -> value-of-interest mapping -> 8-bit gray, inverted for
// MONOCHROME1.
func renderMonochrome(frame Frame, ds *DataSet, geom PixelGeometry, invert bool) (*image.Gray, error) {
	samples, err := frameSamples(frame.Pixels, geom)
	if err != nil {
		return nil, err
	}
	modal, err := ApplyModalityLUT(samples, ds)
	if err != nil {
		return nil, err
	}
	display, err := displayValues(modal, ds, geom)
	if err != nil {
		return nil, err
	}

	img := image.NewGray(image.Rect(0, 0, int(geom.Columns), int(geom.Rows)))
	for i, v := range display {
		b := clampRound(v)
		if invert {
			b = 255 - b
		}
		img.Pix[i] = b
	}
	return img, nil
}

// displayValues maps modality-LUT output onto the 0..255 display range: the dataset's
// value-of-interest state through voiToDisplay when it is usable, else a min/max
// stretch that excludes pixel-padding values (PS3.3 §C.7.5.1.1.2). The stretch also
// serves as the fallback when a declared window is unusable (e.g. width 0), matching
// dcm2pnm's min-max default rather than refusing to render. in is consumed: the
// stretch path reuses it as the result buffer.
func displayValues(in []float64, ds *DataSet, geom PixelGeometry) ([]float64, error) {
	if out, ok, err := voiToDisplay(in, ds, 0, 0, 255); err != nil {
		return nil, err
	} else if ok {
		return out, nil
	}
	return autoStretch(in, ds, geom)
}

// autoStretch maps in onto 0..255 in place by a linear min/max stretch, excluding
// padding: pixels inside the Pixel Padding Value / Pixel Padding Range Limit interval
// carry no image information (PS3.3 §C.7.5.1.1.2), so they neither widen the stretch
// nor render above black. A frame that is all padding, or whose real values are flat,
// renders black.
func autoStretch(in []float64, ds *DataSet, geom PixelGeometry) ([]float64, error) {
	padLo, padHi, hasPad, err := paddingInterval(ds, geom)
	if err != nil {
		return nil, err
	}
	if !hasPad {
		lo, hi := minMax(in)
		stretchToDisplay(in, lo, hi, 0, 255)
		return in, nil
	}

	var lo, hi float64
	seen := false
	for _, v := range in {
		if v >= padLo && v <= padHi {
			continue
		}
		if !seen {
			lo, hi = v, v
			seen = true
			continue
		}
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	for i, v := range in {
		if (v >= padLo && v <= padHi) || hi <= lo {
			in[i] = 0
			continue
		}
		in[i] = (v - lo) * 255 / (hi - lo)
	}
	return in, nil
}

// paddingInterval resolves the modality-space interval that Pixel Padding Value
// (0028,0120) and Pixel Padding Range Limit (0028,0121) declare on stored values. The
// elements are US or SS per Pixel Representation, and Implicit VR materialises them
// as unsigned words, so signed pixel data reinterprets the low 16 bits (the same
// two's-complement fold lutDescriptor applies). Both bounds run through
// ApplyModalityLUT so a rescale (including a negative slope, which flips the
// interval) keeps the comparison in the frame's value space. hasPad is false when
// (0028,0120) is absent; an absent range limit degenerates the interval to the
// padding value alone.
func paddingInterval(ds *DataSet, geom PixelGeometry) (lo, hi float64, hasPad bool, err error) {
	pad, ok := ds.GetInt(TagPixelPaddingValue)
	if !ok {
		return 0, 0, false, nil
	}
	limit, hasLimit := ds.GetInt(TagPixelPaddingRangeLimit)
	if !hasLimit {
		limit = pad
	}
	signed := geom.PixelRepresentation == 1
	mapped, err := ApplyModalityLUT([]float64{storedScalar(pad, signed), storedScalar(limit, signed)}, ds)
	if err != nil {
		return 0, 0, false, err
	}
	lo, hi = mapped[0], mapped[1]
	if lo > hi {
		lo, hi = hi, lo
	}
	return lo, hi, true, nil
}

// storedScalar folds a US-or-SS attribute value into stored-value space: the low 16
// bits, two's-complement when the pixel data is signed.
func storedScalar(v int64, signed bool) float64 {
	if signed {
		return float64(int16(uint16(v))) // #nosec G115 -- two's-complement fold of a US-or-SS word
	}
	return float64(uint16(v)) // #nosec G115 -- unsigned 16-bit fold of a US-or-SS word
}

// frameSamples decodes one single-sample grayscale frame's stored bytes to float64
// values, signed values negative per PixelRepresentation (the input form the
// ApplyModalityLUT pipeline documents).
func frameSamples(pixels []byte, geom PixelGeometry) ([]float64, error) {
	if geom.SamplesPerPixel > 1 {
		return nil, &ValueError{
			Tag: TagSamplesPerPixel, VR: VRUS,
			Msg: fmt.Sprintf("grayscale rendering requires 1 sample per pixel, got %d", geom.SamplesPerPixel),
		}
	}
	count := int(geom.Rows) * int(geom.Columns)
	vals, err := decodeStoredValues(pixels, count, geom, geom.PixelRepresentation == 1, geom.TransferSyntax.byteOrder())
	if err != nil {
		return nil, err
	}
	out := make([]float64, len(vals))
	for i, v := range vals {
		out[i] = float64(v)
	}
	return out, nil
}

// decodeStoredValues reads count single-sample stored values from a decoded frame: a
// byte per value at 8 BitsAllocated, a word per value at 16 decoded with byte order
// bo. Grayscale samples and PALETTE COLOR indices share this layout; when signed the
// values are two's-complement (PS3.3 C.7.6.3.1.5). A geometry that declares a
// narrower stored width (BitsStored with HighBit, PS3.5 §8.1.1) has the stored bits
// extracted from below HighBit and, for signed data, sign-extended from BitsStored,
// so overlay planes packed in the unused high bits never corrupt the value; an
// incoherent BitsStored/HighBit pair falls back to the full allocated width rather
// than corrupting every sample with a bad shift. It fails closed on an unsupported
// bit depth or a frame shorter than the pixel count.
func decodeStoredValues(pixels []byte, count int, geom PixelGeometry, signed bool, bo binary.ByteOrder) ([]int, error) {
	bits := int(geom.BitsAllocated)
	stored := int(geom.BitsStored)
	high := int(geom.HighBit)
	extract := stored > 0 && stored < bits && high+1 >= stored && high < bits
	shift := 0
	if extract {
		shift = high + 1 - stored
	}

	toValue := func(raw uint16) int {
		if extract {
			v := int(raw>>shift) & (1<<stored - 1)
			if signed && v >= 1<<(stored-1) {
				v -= 1 << stored
			}
			return v
		}
		if signed {
			if bits == 8 {
				return int(int8(uint8(raw))) // #nosec G115 -- two's-complement stored value per PixelRepresentation
			}
			return int(int16(raw)) // #nosec G115 -- two's-complement stored value per PixelRepresentation
		}
		return int(raw)
	}

	out := make([]int, count)
	switch geom.BitsAllocated {
	case 8:
		if len(pixels) < count {
			return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "frame shorter than pixel count"}
		}
		for i := range count {
			out[i] = toValue(uint16(pixels[i]))
		}
	case 16:
		if len(pixels) < 2*count {
			return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "frame shorter than pixel count"}
		}
		for i := range count {
			out[i] = toValue(bo.Uint16(pixels[i*2 : i*2+2]))
		}
	default:
		return nil, &ValueError{
			Tag: TagBitsAllocated, VR: VRUS,
			Msg: fmt.Sprintf("stored-value decoding supports 8 or 16 BitsAllocated, got %d", geom.BitsAllocated),
		}
	}
	return out, nil
}

// rgbImage packs interleaved 8-bit R,G,B triples into an opaque *image.NRGBA.
func rgbImage(rgb []byte, geom PixelGeometry) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, int(geom.Columns), int(geom.Rows)))
	for i := 0; i < len(rgb)/3; i++ {
		img.Pix[i*4+0] = rgb[i*3+0]
		img.Pix[i*4+1] = rgb[i*3+1]
		img.Pix[i*4+2] = rgb[i*3+2]
		img.Pix[i*4+3] = 0xFF
	}
	return img
}

// EncodePPM writes img as binary PPM (P6, maxval 255). PPM has no grayscale or alpha
// form, so every pixel is written as an opaque 8-bit R,G,B triple. PNG needs no
// counterpart here: image/png encodes the returned images directly.
func EncodePPM(w io.Writer, img image.Image) error {
	b := img.Bounds()
	bw := bufio.NewWriter(w)
	if _, err := fmt.Fprintf(bw, "P6\n%d %d\n255\n", b.Dx(), b.Dy()); err != nil {
		return err
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			// RGBA() channels are 16-bit (0..65535), so >>8 is always in byte range.
			if _, err := bw.Write([]byte{byte(r >> 8), byte(g >> 8), byte(bl >> 8)}); err != nil { // #nosec G115 -- 16-bit channel narrowed to its high byte
				return err
			}
		}
	}
	return bw.Flush()
}
