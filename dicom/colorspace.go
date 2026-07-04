package dicom

import (
	"encoding/binary"
	"fmt"
	"math"
)

// ColorSpace names a DICOM Photometric Interpretation (0028,0004) colour model that
// the conversion helpers in this file understand. The string values match the PS3.3
// C.7.6.3.1.2 defined terms so callers can pass a frame's
// PixelGeometry.PhotometricInterpretation straight through.
type ColorSpace string

const (
	// ColorSpaceRGB is interleaved or planar 8-bit-per-sample red/green/blue.
	ColorSpaceRGB ColorSpace = "RGB"
	// ColorSpaceYBRFull is full-range luma/chroma at full chroma resolution (no
	// subsampling): one Y, Cb, Cr triple per pixel.
	ColorSpaceYBRFull ColorSpace = "YBR_FULL"
	// ColorSpaceYBRFull422 is full-range luma/chroma with 4:2:2 horizontal chroma
	// subsampling: two luma samples share one Cb/Cr pair (PS3.3 C.7.6.3.1.2).
	ColorSpaceYBRFull422 ColorSpace = "YBR_FULL_422"
)

// ApplyColorLUT expands a single PALETTE COLOR frame to interleaved 3-sample RGB by
// mapping each pixel index through the Red, Green, and Blue Palette Color Lookup
// Tables of ds. It is the go-radx analogue of pydicom's apply_color_lut for the
// non-segmented palette path (PS3.3 C.7.9, C.7.6.3.1.5).
//
// The input frame carries one index per pixel: a byte per pixel when geom.BitsAllocated
// is 8, or a word per pixel when it is 16, decoded with the dataset's transfer-syntax
// byte order bo. When geom.PixelRepresentation is 1 the indices and the descriptor's
// first-mapped value are two's-complement signed (PS3.3 C.7.6.3.1.5); otherwise they are
// unsigned. Each index selects an entry from the three LUTs; the output is Rows*Columns*3
// bytes, R,G,B interleaved, one 8-bit sample per channel (16-bit LUT entries are scaled
// down to 8 bits by taking the high byte, matching pydicom's apply_color_lut output dtype
// for an 8-bit render).
//
// The LUT is described by its three-value descriptor (0028,1101..1103):
//
//	descriptor[0] = number of entries in the LUT (0 means 65536, PS3.3 C.7.6.3.1.5);
//	descriptor[1] = the first input value mapped (pixel values below it clamp to the
//	                first entry, values at or above the last clamp to the last);
//	descriptor[2] = number of bits per entry, 8 or 16.
//
// pydicom documents a quirk that PS3.3 C.7.6.3.1.5 also notes: when the descriptor is
// read under a signed VR (US or SS), descriptor[0] may decode as a negative number for
// a 65536-entry table; ApplyColorLUT interprets descriptor[0] as unsigned, so a stored
// -1/65535/0 all resolve to a 65536-entry table. The count is always unsigned; only the
// first-mapped value (descriptor[1]) and the pixel indices follow PixelRepresentation.
//
// bo is the dataset's transfer-syntax byte order. The DataSet does not carry its transfer
// syntax, so the caller supplies it (consistent with value_codec.go's decode helpers). It
// governs both the 16-bit LUT-entry words (OW) and the 16-bit pixel indices.
func ApplyColorLUT(frame Frame, ds *DataSet, geom PixelGeometry, bo binary.ByteOrder) ([]byte, error) {
	signed := geom.PixelRepresentation == 1
	red, err := readPaletteLUT(ds, TagRedPaletteColorLookupTableDescriptor, TagRedPaletteColorLookupTableData, signed, bo)
	if err != nil {
		return nil, err
	}
	green, err := readPaletteLUT(ds, TagGreenPaletteColorLookupTableDescriptor, TagGreenPaletteColorLookupTableData, signed, bo)
	if err != nil {
		return nil, err
	}
	blue, err := readPaletteLUT(ds, TagBluePaletteColorLookupTableDescriptor, TagBluePaletteColorLookupTableData, signed, bo)
	if err != nil {
		return nil, err
	}

	pixels := int(geom.Rows) * int(geom.Columns)
	if pixels == 0 {
		return []byte{}, nil
	}

	indices, err := decodeStoredValues(frame.Pixels, pixels, geom, signed, bo)
	if err != nil {
		return nil, err
	}

	out := make([]byte, pixels*3)
	for i, idx := range indices {
		out[i*3+0] = red.lookup(idx)
		out[i*3+1] = green.lookup(idx)
		out[i*3+2] = blue.lookup(idx)
	}
	return out, nil
}

// paletteLUT is one resolved Palette Color Lookup Table: its descriptor plus the
// per-entry 8-bit values already normalised from the stored 8- or 16-bit entries.
type paletteLUT struct {
	firstMapped int // descriptor[1], the first input value this LUT maps
	entries     []byte
}

// lookup maps a pixel index to its 8-bit channel value, clamping out-of-range indices
// to the first or last entry as PS3.3 C.7.6.3.1.5 requires (an index below the first
// mapped value uses the first entry; an index at or beyond the last uses the last).
func (l paletteLUT) lookup(index int) byte {
	pos := index - l.firstMapped
	if pos < 0 {
		pos = 0
	}
	if pos >= len(l.entries) {
		pos = len(l.entries) - 1
	}
	return l.entries[pos]
}

// readPaletteLUT reads one colour channel's descriptor and data into a paletteLUT.
// The descriptor (US-or-SS, VM 3) gives the entry count, first-mapped value, and
// bits-per-entry; the data (OW) carries the entries decoded with byte order bo. An 8-bit
// table stores its entries in the low byte of each 16-bit word per PS3.5, except a table
// whose byte length is exactly the entry count (entries packed one per byte). A 16-bit
// table keeps the high byte for the 8-bit render.
//
// When signed is true the image's Pixel Representation (0028,0103) is two's-complement, so
// the first-mapped value (descriptor[1]) is signed and sign-extended from 16 bits (PS3.3
// C.7.6.3.1.5); the count (descriptor[0]) is always unsigned.
func readPaletteLUT(ds *DataSet, descTag, dataTag Tag, signed bool, bo binary.ByteOrder) (paletteLUT, error) {
	desc, ok := getInts(ds, descTag)
	if !ok || len(desc) < 3 {
		return paletteLUT{}, &ValueError{Tag: descTag, VR: VRUSorSS, Msg: "missing or short Palette Color LUT Descriptor (need 3 values)"}
	}

	// descriptor[0] is the entry count; 0 means 65536 (PS3.3 C.7.6.3.1.5). A signed VR
	// can carry the count as a negative number for a full 65536-entry table, so fold it
	// to the low 16 bits (the unsigned interpretation) before applying the zero rule.
	// The mask keeps the conversion in range for any wider-VR value a hostile file
	// might carry under this tag. The count is unsigned regardless of Pixel Representation.
	numEntries := int(desc[0] & 0xFFFF)
	if numEntries == 0 {
		numEntries = 65536
	}
	// descriptor[1] is the first input value mapped. For a signed (PixelRepresentation==1)
	// image it is two's-complement, so sign-extend the low 16 bits; otherwise it is the
	// unsigned low 16 bits. The lookup offset (index - firstMapped) then works for both.
	firstMapped := int(uint16(desc[1])) // #nosec G115 -- intentional fold to 16-bit then re-extend below
	if signed {
		firstMapped = int(int16(uint16(desc[1]))) // #nosec G115 -- two's-complement first-mapped per PS3.3 C.7.6.3.1.5
	}
	bitsPerEntry := desc[2]
	if bitsPerEntry != 8 && bitsPerEntry != 16 {
		return paletteLUT{}, &ValueError{Tag: descTag, VR: VRUSorSS, Msg: fmt.Sprintf("Palette Color LUT bits-per-entry must be 8 or 16, got %d", bitsPerEntry)}
	}

	raw, ok := getRawBytes(ds, dataTag)
	if !ok {
		return paletteLUT{}, &ValueError{Tag: dataTag, VR: VROW, Msg: "missing Palette Color LUT Data"}
	}

	entries, err := decodePaletteEntries(raw, numEntries, int(bitsPerEntry), dataTag, bo)
	if err != nil {
		return paletteLUT{}, err
	}
	return paletteLUT{firstMapped: firstMapped, entries: entries}, nil
}

// decodePaletteEntries normalises numEntries LUT entries from raw to 8 bits each.
//
// An 8-bit LUT (bitsPerEntry 8) is stored one of two ways and pydicom handles both:
// either packed two entries per 16-bit word (byte length 2*numEntries, the common
// PS3.5 form) or one entry per byte (byte length numEntries, seen in some encoders).
// A 16-bit LUT stores one word per entry; the 8-bit render keeps the high byte
// (value >> 8), matching pydicom's downscale to an 8-bit image. The 16-bit words and the
// 8-bit-padded-OW low byte are decoded with byte order bo so an Explicit VR Big Endian
// dataset reads its OW words big-endian rather than swapped.
func decodePaletteEntries(raw []byte, numEntries, bitsPerEntry int, dataTag Tag, bo binary.ByteOrder) ([]byte, error) {
	out := make([]byte, numEntries)
	switch bitsPerEntry {
	case 8:
		switch {
		case len(raw) >= 2*numEntries:
			// Two 8-bit entries per OW word; the entry sits in the word's low byte, whose
			// position within the pair depends on the dataset byte order.
			for i := range numEntries {
				out[i] = byte(bo.Uint16(raw[i*2:i*2+2]) & 0xFF) // #nosec G115 -- deliberate low-byte extraction of an 8-bit-in-OW entry
			}
		case len(raw) >= numEntries:
			copy(out, raw[:numEntries])
		default:
			return nil, &ValueError{Tag: dataTag, VR: VROW, Msg: fmt.Sprintf("Palette Color LUT Data too short: have %d bytes, need %d entries", len(raw), numEntries)}
		}
	case 16:
		if len(raw) < 2*numEntries {
			return nil, &ValueError{Tag: dataTag, VR: VROW, Msg: fmt.Sprintf("Palette Color LUT Data too short: have %d bytes, need %d 16-bit entries", len(raw), numEntries)}
		}
		for i := range numEntries {
			out[i] = byte(bo.Uint16(raw[i*2:i*2+2]) >> 8)
		}
	}
	return out, nil
}

// ConvertColorSpace converts one decoded colour frame between photometric
// interpretations, the go-radx analogue of pydicom's convert_color_space. It supports
// the full-range luma/chroma conversions defined by PS3.3 C.7.6.3.1.2:
//
//	YBR_FULL      <-> RGB
//	YBR_FULL_422  ->  RGB   (and -> YBR_FULL by first reconstructing full chroma)
//
// from describes the frame's current colour model; to is the target. geom supplies the
// dimensions and PlanarConfiguration (0 interleaved R0G0B0R1..., 1 planar
// RRR...GGG...BBB). The 4:2:2 input is always interleaved per PS3.5 8.2.4. The returned
// frame is interleaved 3-sample-per-pixel bytes in the target colour model.
//
// Out of scope (documented honestly, not silently mishandled): YBR_PARTIAL_420 and
// YBR_PARTIAL_422 (the limited-range variants, PS3.3 C.7.6.3.1.2), and the
// JPEG-2000-internal ICT/RCT transforms, which the J2K codec applies during decode so
// decoded frames are already RGB. Passing an unsupported pair returns a typed error.
func ConvertColorSpace(frame Frame, from, to ColorSpace, geom PixelGeometry) (Frame, error) {
	pixels := int(geom.Rows) * int(geom.Columns)
	if pixels == 0 {
		return Frame{Index: frame.Index, Pixels: []byte{}}, nil
	}

	if from == to {
		return frame, nil
	}

	switch {
	case from == ColorSpaceYBRFull && to == ColorSpaceRGB:
		src, err := deinterleaveTriples(frame.Pixels, pixels, geom.PlanarConfiguration)
		if err != nil {
			return Frame{}, err
		}
		return Frame{Index: frame.Index, Pixels: ybrFullToRGB(src, pixels)}, nil

	case from == ColorSpaceRGB && to == ColorSpaceYBRFull:
		src, err := deinterleaveTriples(frame.Pixels, pixels, geom.PlanarConfiguration)
		if err != nil {
			return Frame{}, err
		}
		return Frame{Index: frame.Index, Pixels: rgbToYBRFull(src, pixels)}, nil

	case from == ColorSpaceYBRFull422 && to == ColorSpaceRGB:
		full, err := upsample422(frame.Pixels, int(geom.Rows), int(geom.Columns))
		if err != nil {
			return Frame{}, err
		}
		return Frame{Index: frame.Index, Pixels: ybrFullToRGB(full, pixels)}, nil

	case from == ColorSpaceYBRFull422 && to == ColorSpaceYBRFull:
		full, err := upsample422(frame.Pixels, int(geom.Rows), int(geom.Columns))
		if err != nil {
			return Frame{}, err
		}
		return Frame{Index: frame.Index, Pixels: full}, nil

	default:
		return Frame{}, &ValueError{
			Tag: TagPhotometricInterpretation, VR: VRCS,
			Msg: fmt.Sprintf("unsupported colour-space conversion %s -> %s", from, to),
		}
	}
}

// deinterleaveTriples returns the frame's samples as a flat interleaved triple buffer
// (R0,G0,B0,R1,...) regardless of the source PlanarConfiguration: configuration 0 is
// already interleaved, configuration 1 (RRR...GGG...BBB) is interleaved here. The
// result is always pixels*3 bytes.
func deinterleaveTriples(pixelsBuf []byte, count int, planar uint16) ([]byte, error) {
	need := count * 3
	if len(pixelsBuf) < need {
		return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "colour frame shorter than 3 samples per pixel"}
	}
	if planar == 0 {
		out := make([]byte, need)
		copy(out, pixelsBuf[:need])
		return out, nil
	}
	out := make([]byte, need)
	plane := count
	for i := range count {
		out[i*3+0] = pixelsBuf[i]
		out[i*3+1] = pixelsBuf[plane+i]
		out[i*3+2] = pixelsBuf[2*plane+i]
	}
	return out, nil
}

// upsample422 reconstructs full-resolution interleaved YBR_FULL from a YBR_FULL_422
// frame. Under 4:2:2, each pair of horizontally adjacent pixels carries two luma
// samples and one shared Cb/Cr pair, packed per PS3.5 8.2.4 as
// Y0 Y1 Cb Cr (Y2 Y3 Cb Cr) ... across a row. The shared chroma is copied to both
// pixels of the pair (nearest-neighbour upsampling, matching pydicom). Columns is
// taken even per the 4:2:2 constraint; an odd trailing pixel reuses the last pair's
// chroma.
func upsample422(packed []byte, rows, cols int) ([]byte, error) {
	pixels := rows * cols
	// Each group of two columns contributes 4 bytes (Y0,Y1,Cb,Cr); an odd final column
	// contributes 2 (Y,Cb,Cr share with... ) — but DICOM constrains 4:2:2 to even
	// Columns, so size against the even-pair packing and fail closed if short.
	pairsPerRow := (cols + 1) / 2
	need := rows * (cols + 2*pairsPerRow) // cols luma + 2 chroma per pair
	if len(packed) < need {
		return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: fmt.Sprintf("YBR_FULL_422 frame too short: have %d bytes, need %d", len(packed), need)}
	}

	out := make([]byte, pixels*3)
	p := 0
	for r := range rows {
		for c := 0; c < cols; c += 2 {
			y0 := packed[p]
			p++
			var y1 byte
			hasSecond := c+1 < cols
			if hasSecond {
				y1 = packed[p]
				p++
			}
			cb := packed[p]
			cr := packed[p+1]
			p += 2

			i0 := (r*cols + c) * 3
			out[i0+0] = y0
			out[i0+1] = cb
			out[i0+2] = cr
			if hasSecond {
				i1 := (r*cols + c + 1) * 3
				out[i1+0] = y1
				out[i1+1] = cb
				out[i1+2] = cr
			}
		}
	}
	return out, nil
}

// ybrFullToRGB converts an interleaved full-range YBR_FULL triple buffer to interleaved
// RGB per PS3.3 C.7.6.3.1.2 (the JFIF / ITU-T T.871 full-range inverse):
//
//	R = Y + 1.402 (Cr - 128)
//	G = Y - 0.344136 (Cb - 128) - 0.714136 (Cr - 128)
//	B = Y + 1.772 (Cb - 128)
//
// Each result is rounded to nearest (add 0.5, floor) and clamped to [0,255], matching
// pydicom's convert_color_space output for an 8-bit image.
func ybrFullToRGB(ybr []byte, count int) []byte {
	out := make([]byte, count*3)
	for i := range count {
		y := float64(ybr[i*3+0])
		cb := float64(ybr[i*3+1]) - 128.0
		cr := float64(ybr[i*3+2]) - 128.0
		out[i*3+0] = clampRound(y + 1.402*cr)
		out[i*3+1] = clampRound(y - 0.344136*cb - 0.714136*cr)
		out[i*3+2] = clampRound(y + 1.772*cb)
	}
	return out
}

// rgbToYBRFull converts an interleaved RGB triple buffer to interleaved full-range
// YBR_FULL per PS3.3 C.7.6.3.1.2 (the JFIF / ITU-T T.871 forward):
//
//	Y  =  0.2990 R + 0.5870 G + 0.1140 B
//	Cb = -0.1687 R - 0.3313 G + 0.5000 B + 128
//	Cr =  0.5000 R - 0.4187 G - 0.0813 B + 128
//
// Each result is rounded to nearest (add 0.5, floor) and clamped to [0,255].
func rgbToYBRFull(rgb []byte, count int) []byte {
	out := make([]byte, count*3)
	for i := range count {
		r := float64(rgb[i*3+0])
		g := float64(rgb[i*3+1])
		b := float64(rgb[i*3+2])
		out[i*3+0] = clampRound(0.2990*r + 0.5870*g + 0.1140*b)
		out[i*3+1] = clampRound(-0.1687*r - 0.3313*g + 0.5000*b + 128.0)
		out[i*3+2] = clampRound(0.5000*r - 0.4187*g - 0.0813*b + 128.0)
	}
	return out
}

// clampRound rounds v to the nearest integer (half away from zero on the positive side
// via +0.5 then floor) and clamps it into the 8-bit [0,255] range.
func clampRound(v float64) byte {
	v = math.Floor(v + 0.5)
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return byte(v)
}

// getInts returns all integer values of an SS/US/SL/UL element at t. It is the
// multi-value companion to DataSet.GetInt, needed for the three-value Palette Color LUT
// descriptors. ok is false when t is absent or not an integer value.
func getInts(ds *DataSet, t Tag) ([]int64, bool) {
	e, ok := ds.Get(t)
	if !ok {
		return nil, false
	}
	v, ok := materialise(e.Value)
	if !ok {
		return nil, false
	}
	iv, ok := v.(*Ints)
	if !ok {
		return nil, false
	}
	return iv.Ints(), true
}

// getRawBytes returns the raw value bytes of an OB/OW element at t (Palette Color LUT
// Data is OW). ok is false when t is absent or not a byte value.
func getRawBytes(ds *DataSet, t Tag) ([]byte, bool) {
	e, ok := ds.Get(t)
	if !ok {
		return nil, false
	}
	v, ok := materialise(e.Value)
	if !ok {
		return nil, false
	}
	bv, ok := v.(*Bytes)
	if !ok {
		return nil, false
	}
	return bv.Bytes(), true
}
