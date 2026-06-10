package dicom

import (
	"encoding/binary"
	"fmt"
	"math"
)

// RLE Lossless (PS3.5 Annex G). Each frame is a 64-byte header of sixteen
// little-endian uint32 values — the first is the segment count (1..15), the next
// fifteen are byte offsets of each segment from the start of the frame — followed by
// the PackBits-encoded segments. A segment carries one byte plane: for an N-byte
// sample there are N segments per sample, most-significant byte first, so the segment
// for sample c and little-endian byte b is c*bytesPerSample + (bytesPerSample-1-b).

const (
	rleHeaderLen   = 64
	rleMaxSegments = 15
)

// rleSegmentCount is the number of byte planes a frame's geometry produces:
// SamplesPerPixel * bytesPerSample. RLE requires BitsAllocated to be 8 or 16, so
// bytesPerSample is 1 or 2 and the count never exceeds 6. It is used by the encoder,
// which derives the segment layout from the geometry; the decoder reads the segment
// count from the codestream header instead (it is authoritative per PS3.5 Annex G).
func rleSegmentCount(geom PixelGeometry) (segments, bytesPerSample int, err error) {
	switch geom.BitsAllocated {
	case 8:
		bytesPerSample = 1
	case 16:
		bytesPerSample = 2
	default:
		return 0, 0, &ValueError{
			Tag: TagBitsAllocated, VR: VRUS,
			Msg: fmt.Sprintf("RLE Lossless requires BitsAllocated 8 or 16, got %d", geom.BitsAllocated),
		}
	}
	samples := int(geom.SamplesPerPixel)
	if samples < 1 {
		samples = 1
	}
	segments = samples * bytesPerSample
	if segments < 1 || segments > rleMaxSegments {
		return 0, 0, &ValueError{
			Tag: TagPixelData, VR: VROBorOW,
			Msg: fmt.Sprintf("RLE segment count %d out of range 1..15", segments),
		}
	}
	return segments, bytesPerSample, nil
}

// rleByteInterleave reports whether geom's BitsAllocated and SamplesPerPixel match a
// segment count of segments under the PS3.5 Annex G byte-plane mapping (samples *
// bytesPerSample). When they do, the decoder scatters each segment into the
// interleaved little-endian output; when they do not (a non-conformant codestream
// such as a 1-bit segmentation re-encoded as RLE), the decoder concatenates each
// segment as a contiguous plane instead, so the codestream is still decoded faithfully
// rather than rejected. bytesPerSample is meaningful only when interleave is true.
func rleByteInterleave(geom PixelGeometry, segments int) (interleave bool, bytesPerSample, samples int) {
	switch geom.BitsAllocated {
	case 8:
		bytesPerSample = 1
	case 16:
		bytesPerSample = 2
	default:
		return false, 0, 0
	}
	samples = int(geom.SamplesPerPixel)
	if samples < 1 {
		samples = 1
	}
	if samples*bytesPerSample != segments {
		return false, 0, 0
	}
	return true, bytesPerSample, samples
}

// decodeRLEFrame expands one RLE-encoded frame into contiguously packed pixel bytes
// laid out per geom. Every offset and length read from the header is bounds-checked
// against the frame before use, so a malformed header or a segment that decodes to
// the wrong length is a typed error, never a panic or a partial image (PS3.5 Annex G;
// PRD §9.3 bounds-checked allocations).
func decodeRLEFrame(frame []byte, geom PixelGeometry) ([]byte, error) {
	if len(frame) < rleHeaderLen {
		return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "RLE frame shorter than the 64-byte header"}
	}

	// The codestream header is authoritative for the segment count (PS3.5 Annex G).
	segments := int(binary.LittleEndian.Uint32(frame[0:4]))
	if segments < 1 || segments > rleMaxSegments {
		return nil, &ValueError{
			Tag: TagPixelData, VR: VROBorOW,
			Msg: fmt.Sprintf("RLE header segment count %d out of range 1..15", segments),
		}
	}

	offsets := make([]int, segments)
	for i := 0; i < segments; i++ {
		offsets[i] = int(binary.LittleEndian.Uint32(frame[4+i*4 : 8+i*4]))
		if offsets[i] < rleHeaderLen || offsets[i] > len(frame) {
			return nil, &ValueError{
				Tag: TagPixelData, VR: VROBorOW,
				Msg: fmt.Sprintf("RLE segment %d offset %d outside frame", i, offsets[i]),
			}
		}
	}

	pixelsPerSegment := int(geom.Rows) * int(geom.Columns)
	if pixelsPerSegment <= 0 {
		return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "RLE frame geometry has zero pixels"}
	}

	planes := make([][]byte, segments)
	for seg := 0; seg < segments; seg++ {
		end := len(frame)
		if seg+1 < segments {
			end = offsets[seg+1]
		}
		if end < offsets[seg] {
			return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "RLE segment offsets not monotonic"}
		}
		plane, err := unpackBits(frame[offsets[seg]:end], pixelsPerSegment)
		if err != nil {
			return nil, err
		}
		if len(plane) != pixelsPerSegment {
			return nil, &ValueError{
				Tag: TagPixelData, VR: VROBorOW,
				Msg: fmt.Sprintf("RLE segment %d decoded to %d bytes, expected %d", seg, len(plane), pixelsPerSegment),
			}
		}
		planes[seg] = plane
	}

	interleave, bytesPerSample, samples := rleByteInterleave(geom, segments)
	out := make([]byte, segments*pixelsPerSegment)
	if !interleave {
		// Non-conformant geometry: emit each byte plane contiguously. A single-segment
		// frame (the common 8-bit case the byte interleave also handles) decodes to one
		// plane regardless, so this path stays correct for it.
		for seg := 0; seg < segments; seg++ {
			copy(out[seg*pixelsPerSegment:], planes[seg])
		}
		return out, nil
	}

	// Scatter each byte plane into the interleaved little-endian output. Segment index
	// seg = sample*bytesPerSample + bytePlane, where bytePlane 0 is the most
	// significant byte; the destination little-endian byte index is therefore
	// (bytesPerSample-1-bytePlane).
	stride := samples * bytesPerSample
	for seg := 0; seg < segments; seg++ {
		sample := seg / bytesPerSample
		bytePlane := seg % bytesPerSample
		destByte := bytesPerSample - 1 - bytePlane
		dst := sample*bytesPerSample + destByte
		plane := planes[seg]
		for p := 0; p < pixelsPerSegment; p++ {
			out[dst] = plane[p]
			dst += stride
		}
	}
	return out, nil
}

// unpackBits decodes a PackBits (PS3.5 G.3) byte stream, stopping once want bytes are
// produced or the input is exhausted. A literal or replicate run that would read past
// the input is a typed truncation error rather than an out-of-bounds read.
func unpackBits(src []byte, want int) ([]byte, error) {
	out := make([]byte, 0, want)
	i := 0
	for i < len(src) && len(out) < want {
		n := int8(src[i]) // #nosec G115 -- same-width reinterpretation: the PackBits control byte is signed per PS3.5 G.3
		i++
		switch {
		case n >= 0:
			count := int(n) + 1
			if i+count > len(src) {
				return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "RLE literal run runs past segment end"}
			}
			out = append(out, src[i:i+count]...)
			i += count
		case n == -128:
			// No-op per PS3.5 G.3.
		default:
			count := 1 - int(n)
			if i >= len(src) {
				return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "RLE replicate run missing its byte"}
			}
			v := src[i] // #nosec G602 -- guarded by the i >= len(src) check immediately above
			i++
			for k := 0; k < count; k++ {
				out = append(out, v)
			}
		}
	}
	return out, nil
}

// encodeRLEFrame packs one contiguous frame into an RLE-encoded frame: a 64-byte
// header followed by one PackBits segment per byte plane. The input length must match
// the geometry's FrameLength, so a mismatched buffer is rejected before encoding.
func encodeRLEFrame(frame []byte, geom PixelGeometry) ([]byte, error) {
	segments, bytesPerSample, err := rleSegmentCount(geom)
	if err != nil {
		return nil, err
	}
	if len(frame) != geom.FrameLength() {
		return nil, &ValueError{
			Tag: TagPixelData, VR: VROBorOW,
			Msg: fmt.Sprintf("frame is %d bytes, geometry needs %d", len(frame), geom.FrameLength()),
		}
	}

	pixelsPerSegment := int(geom.Rows) * int(geom.Columns)
	samples := segments / bytesPerSample
	stride := samples * bytesPerSample

	header := make([]byte, rleHeaderLen)
	binary.LittleEndian.PutUint32(header[0:4], uint32(segments)) // #nosec G115 -- rleSegmentCount bounds segments to 1..15

	var body []byte
	for seg := 0; seg < segments; seg++ {
		// The header records each segment's offset from the start of the frame; the
		// first segment begins immediately after the 64-byte header. Offsets are
		// 32-bit fields (PS3.5 Annex G), so a frame whose encoding grows past that
		// cannot be represented in RLE at all.
		if int64(rleHeaderLen)+int64(len(body)) > math.MaxUint32 {
			return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "RLE segment offset exceeds the 32-bit header field"}
		}
		binary.LittleEndian.PutUint32(header[4+seg*4:8+seg*4], uint32(rleHeaderLen+len(body))) // #nosec G115 -- bounded by the MaxUint32 check above

		sample := seg / bytesPerSample
		bytePlane := seg % bytesPerSample
		srcByte := bytesPerSample - 1 - bytePlane

		plane := make([]byte, pixelsPerSegment)
		src := sample*bytesPerSample + srcByte
		for p := 0; p < pixelsPerSegment; p++ {
			plane[p] = frame[src]
			src += stride
		}
		seg := packBits(plane)
		// Each segment must be even-padded so the next begins on an even boundary
		// (PS3.5 Annex G: segments are byte-aligned and the frame is even-length).
		if len(seg)%2 == 1 {
			seg = append(seg, 0x00)
		}
		body = append(body, seg...)
	}

	out := make([]byte, 0, rleHeaderLen+len(body))
	out = append(out, header...)
	out = append(out, body...)
	return out, nil
}

// packBits encodes src as a PackBits (PS3.5 G.3) stream: replicate runs for three or
// more equal bytes, literal runs otherwise, each run capped at 128 bytes.
func packBits(src []byte) []byte {
	var out []byte
	i := 0
	for i < len(src) {
		runLen := 1
		for i+runLen < len(src) && runLen < 128 && src[i+runLen] == src[i] {
			runLen++
		}
		if runLen >= 3 {
			// #nosec G115 -- same-width reinterpretation: runLen is 3..128 so 1-runLen fits int8 (PS3.5 G.3)
			out = append(out, byte(int8(1-runLen)), src[i])
			i += runLen
			continue
		}
		// Gather a literal run, breaking before a replicable run of three.
		start := i
		i++
		for i < len(src) && i-start < 128 {
			if i+2 < len(src) && src[i] == src[i+1] && src[i] == src[i+2] {
				break
			}
			i++
		}
		litLen := i - start
		out = append(out, byte(int8(litLen-1))) // #nosec G115 -- same-width reinterpretation: litLen is 1..128 so litLen-1 fits int8 (PS3.5 G.3)
		out = append(out, src[start:i]...)
	}
	return out
}
