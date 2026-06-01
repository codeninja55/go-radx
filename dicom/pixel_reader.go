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
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return ReadPixelDataFrom(bufio.NewReader(f), opts...)
}

// ReadPixelDataFrom reads a Part 10 stream and returns its PixelData. The main
// dataset must be carried in one of the uncompressed encodings (only the pixel data
// is compressed under an encapsulated transfer syntax), which is always the case for
// a conformant Part 10 object.
func ReadPixelDataFrom(r io.Reader, opts ...ReadOption) (*PixelData, error) {
	cfg := newReadConfig(opts...)
	br := newBoundedReader(r, cfg.maxElementLen)

	if _, err := readPreamble(br); err != nil {
		return nil, err
	}
	meta, err := readFileMeta(br)
	if err != nil {
		return nil, err
	}
	ts := meta.TransferSyntaxUID
	if ts == "" {
		return nil, &ValueError{Tag: tagTransferSyntax, VR: VRUI, Msg: "empty transfer syntax"}
	}

	if !ts.IsEncapsulated() {
		// The dataset reader handles native pixel data as a normal OB/OW value.
		ds, err := readDataSet(br, ts, cfg)
		if err != nil {
			return nil, err
		}
		return NewPixelData(ds, ts)
	}

	// Encapsulated: stop the dataset reader at the pixel-data element. It consumes the
	// (7FE0,0010) undefined-length header and leaves the reader positioned at the first
	// fragment item, which readEncapsulatedValue copies out as a bounded item stream.
	cfg.stopAtPixelData = true
	ds, err := readDataSet(br, ts, cfg)
	if err != nil {
		return nil, err
	}
	value, err := readEncapsulatedValue(br, ts)
	if err != nil {
		return nil, err
	}
	return NewEncapsulatedPixelData(ds, ts, value)
}

// readEncapsulatedValue copies the encapsulated fragment stream — every byte from the
// first item header through the Sequence Delimitation Item — into a buffer for
// parseEncapsulated to bound-check. The dataset reader has already consumed the
// (7FE0,0010) header (stopAtPixelData), so the reader is positioned at the first item.
// Each item header is validated against the bytes remaining as it is read, so a
// hostile length cannot drive an allocation past the source (Codex DCM-006).
func readEncapsulatedValue(br *boundedReader, ts TransferSyntax) ([]byte, error) {
	var out []byte
	for {
		tag, length, err := readDelimiterHeader(br, ts)
		if err != nil {
			return nil, err
		}
		out = appendDelimiterHeader(out, tag, length, ts)
		if tag == tagSequenceDelimit {
			return out, nil
		}
		if tag != tagItem {
			return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "unexpected tag inside encapsulated pixel data"}
		}
		if length == undefinedLength {
			return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "fragment item has undefined length"}
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
