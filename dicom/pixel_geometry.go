package dicom

// PixelGeometry is the resolved image-pixel module describing one instance's pixel
// layout. It is the parameter a Codec needs to decode or encode a frame: the codec
// must know the frame dimensions, sample count, and bit packing to lay decoded
// samples out correctly. The fields are read from the PS3.3 Image Pixel module
// attributes; absent optional attributes take their standard defaults.
type PixelGeometry struct {
	Rows                uint16 // (0028,0010)
	Columns             uint16 // (0028,0011)
	SamplesPerPixel     uint16 // (0028,0002), default 1
	BitsAllocated       uint16 // (0028,0100)
	BitsStored          uint16 // (0028,0101)
	HighBit             uint16 // (0028,0102)
	PixelRepresentation uint16 // (0028,0103), 0 unsigned / 1 two's-complement signed
	PlanarConfiguration uint16 // (0028,0006), 0 interleaved / 1 planar, default 0
	NumberOfFrames      int    // (0028,0008), default 1

	// PhotometricInterpretation (0028,0004) is the colour model defined term
	// (MONOCHROME2, RGB, YBR_FULL, ...). It is carried for codecs and callers that
	// need it; it does not affect FrameLength.
	PhotometricInterpretation string

	// TransferSyntax names the encoding the pixel data is carried in; it selects the
	// codec and, for native data, the sample byte order.
	TransferSyntax TransferSyntax
}

// FrameLength is the byte count of one decoded, contiguously packed frame. For
// BitsAllocated >= 8 it is the whole-byte product of the dimensions; for sub-byte
// BitsAllocated (a 1-bit segmentation bitmap) it is the bit count rounded up to the
// next whole byte (PS3.5 §8.1.1). It never overflows for in-range geometry because
// every term is widened to int before multiplying.
func (g PixelGeometry) FrameLength() int {
	pixels := int(g.Rows) * int(g.Columns) * int(g.SamplesPerPixel)
	bits := pixels * int(g.BitsAllocated)
	return (bits + 7) / 8
}

// ResolvePixelGeometry reads the Image Pixel module attributes from ds for the pixel
// data carried under ts. Rows, Columns, and BitsAllocated are required; the resolver
// rejects a dataset that omits them or declares a zero BitsAllocated, so a malformed
// header fails before any pixel allocation rather than producing a zero-length or
// undersized frame. SamplesPerPixel, PlanarConfiguration, and NumberOfFrames take
// their PS3.3 defaults (1, 0, 1) when absent.
func ResolvePixelGeometry(ds *DataSet, ts TransferSyntax) (PixelGeometry, error) {
	rows, okR := ds.GetInt(TagRows)
	cols, okC := ds.GetInt(TagColumns)
	if !okR || !okC {
		return PixelGeometry{}, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "missing Rows or Columns"}
	}
	bits, okB := ds.GetInt(TagBitsAllocated)
	if !okB || bits <= 0 {
		return PixelGeometry{}, &ValueError{Tag: TagBitsAllocated, VR: VRUS, Msg: "missing or zero BitsAllocated"}
	}

	geom := PixelGeometry{
		Rows:                uint16(rows),
		Columns:             uint16(cols),
		SamplesPerPixel:     1,
		BitsAllocated:       uint16(bits),
		NumberOfFrames:      1,
		PlanarConfiguration: 0,
		TransferSyntax:      ts,
	}
	if n, ok := ds.GetInt(TagSamplesPerPixel); ok && n > 0 {
		geom.SamplesPerPixel = uint16(n)
	}
	if n, ok := ds.GetInt(TagBitsStored); ok {
		geom.BitsStored = uint16(n)
	}
	if n, ok := ds.GetInt(TagHighBit); ok {
		geom.HighBit = uint16(n)
	}
	if n, ok := ds.GetInt(TagPixelRepresentation); ok {
		geom.PixelRepresentation = uint16(n)
	}
	if n, ok := ds.GetInt(TagPlanarConfiguration); ok {
		geom.PlanarConfiguration = uint16(n)
	}
	if n, ok := ds.GetInt(TagNumberOfFrames); ok && n > 0 {
		geom.NumberOfFrames = int(n)
	}
	if s, ok := ds.GetString(TagPhotometricInterpretation); ok {
		geom.PhotometricInterpretation = s
	}
	return geom, nil
}
