package dicom

import (
	"bufio"
	"io"
	"os"
)

// ReadPixelData opens path and returns its decoded-ready PixelData. Unlike Read, it
// accepts an encapsulated transfer syntax: it parses the main dataset for geometry,
// then reads the (7FE0,0010) element — a native OB/OW value or an encapsulated
// fragment stream — into a PixelData. The fragment stream is parsed as a bounded item
// stream so a malformed object fails with a typed error (Codex DCM-006).
func ReadPixelData(path string, opts ...ReadOption) (*PixelData, error) {
	f, err := os.Open(path) // #nosec G304 -- reading a caller-supplied path is this API's contract
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return ReadPixelDataFrom(bufio.NewReader(f), opts...)
}

// ReadPixelDataFrom reads a Part 10 stream and returns its PixelData. It shares the
// Read path: the full dataset is parsed (an encapsulated transfer syntax retains the
// verbatim fragment stream) and NewPixelData binds it to its geometry.
func ReadPixelDataFrom(r io.Reader, opts ...ReadOption) (*PixelData, error) {
	f, err := Read(r, opts...)
	if err != nil {
		return nil, err
	}
	return NewPixelData(f.DataSet, f.Meta.TransferSyntaxUID)
}

// readEncapsulatedValue copies the encapsulated fragment stream — every byte from the
// first item header through the Sequence Delimitation Item — into a buffer for
// parseEncapsulated to bound-check. The caller (readDataSet's element loop) has
// already consumed the (7FE0,0010) header, so the reader is positioned at the first
// item. Each item header is validated against the bytes remaining as it is read, so a
// hostile length cannot drive an allocation past the source (Codex DCM-006). The
// accumulated stream is one (7FE0,0010) element value, so it is also held to the same
// per-element cap (WithMaxElementLen) that bounds a native pixel value: per-item
// bounds alone would let many small fragments grow the retained stream without limit.
//
// Structural validation mirrors parseEncapsulated (NewPixelData's validator): item
// tags only, defined even item lengths, and a zero-length Sequence Delimitation Item,
// so the two layers accept and reject the same streams. parseEncapsulated reads the
// first item as the Basic Offset Table, which is positional — any stream this loop
// accepts begins with an item, so there is nothing extra to check here. The read
// layer is laxer only about frame mapping, which needs the dataset's NumberOfFrames
// and stays in NewPixelData.
func readEncapsulatedValue(br *boundedReader, ts TransferSyntax) ([]byte, error) {
	var out []byte
	for {
		tag, length, err := readDelimiterHeader(br, ts)
		if err != nil {
			return nil, err
		}
		out = appendDelimiterHeader(out, tag, length, ts)
		if tag == tagSequenceDelimit {
			if length != 0 {
				// A non-zero delimiter length must fail here: accepting it would leave
				// the outer element loop misreading the bytes that follow (PS3.5 A.4).
				return nil, &ValueError{Tag: tagSequenceDelimit, VR: VROBorOW, Msg: "Sequence Delimitation Item must have zero length"}
			}
			return out, nil
		}
		if tag != tagItem {
			return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "unexpected tag inside encapsulated pixel data"}
		}
		if length == undefinedLength {
			return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "fragment item has undefined length"}
		}
		if length%2 != 0 {
			return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "fragment item length is odd; encapsulated item values must be even"}
		}
		if total := uint64(len(out)) + uint64(length); total > uint64(br.maxLen) {
			return nil, &LimitExceededError{Tag: TagPixelData, Limit: uint64(br.maxLen), Actual: total, Kind: "element-length"}
		}
		body, err := br.readN(length)
		if err != nil {
			return nil, err
		}
		out = append(out, body...)
	}
}

// appendDelimiterHeader re-serialises an 8-byte item/delimiter header so the captured
// value stream is byte-identical to the source for parseEncapsulated to re-parse.
func appendDelimiterHeader(dst []byte, tag Tag, length uint32, ts TransferSyntax) []byte {
	bo := ts.byteOrder()
	var b [8]byte
	bo.PutUint16(b[0:2], tag.Group())
	bo.PutUint16(b[2:4], tag.Element())
	bo.PutUint32(b[4:8], length)
	return append(dst, b[:]...)
}
