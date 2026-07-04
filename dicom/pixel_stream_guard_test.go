package dicom

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
)

// buildEncapsulatedPart10 assembles a Part 10 stream like seedEncapsulatedPart10 but
// with a caller-controlled fragment stream after the empty Basic Offset Table, so
// hostile fragment layouts can be expressed.
func buildEncapsulatedPart10(ts TransferSyntax, fragments func(out *bytes.Buffer)) []byte {
	meta := &FileMeta{
		MediaStorageSOPClassUID:    "1.2.840.10008.5.1.4.1.1.7",
		MediaStorageSOPInstanceUID: "1.2.3.4.5.6.7.8.9",
		TransferSyntaxUID:          ts,
	}
	var out bytes.Buffer
	_ = writeFileMeta(&out, [128]byte{}, meta)

	geom := NewDataSet()
	geom.Set(Element{Tag: NewTag(0x0028, 0x0002), VR: VRUS, Value: NewInts(VRUS, 1)})
	geom.Set(Element{Tag: NewTag(0x0028, 0x0010), VR: VRUS, Value: NewInts(VRUS, 2)})
	geom.Set(Element{Tag: NewTag(0x0028, 0x0011), VR: VRUS, Value: NewInts(VRUS, 2)})
	geom.Set(Element{Tag: NewTag(0x0028, 0x0100), VR: VRUS, Value: NewInts(VRUS, 8)})
	_ = writeDataSet(&out, geom, ExplicitVRLittleEndian)

	out.Write(encapsulatedPixelHeader())
	out.Write(seedItemHeader(0)) // empty Basic Offset Table
	fragments(&out)
	return out.Bytes()
}

// TestReadEncapsulatedAggregateStreamCap pins the aggregate bound on the retained
// fragment stream: per-item bounds alone do not cap the total, so a hostile stream of
// many tiny fragments must fail with the package's typed limit error once the
// accumulated stream exceeds the same WithMaxElementLen cap that bounds a native
// (7FE0,0010) value — and the reader must stop consuming promptly, never buffering
// the whole hostile stream.
func TestReadEncapsulatedAggregateStreamCap(t *testing.T) {
	const cap = 1 << 10 // 1 KiB element cap
	frag := []byte{0xAB, 0xCD}
	data := buildEncapsulatedPart10(RLELossless, func(out *bytes.Buffer) {
		for i := 0; i < 4096; i++ { // ~40 KiB of fragment stream, far past the cap
			out.Write(seedItemHeader(uint32(len(frag))))
			out.Write(frag)
		}
		out.Write(seedSequenceDelimiter())
	})

	src := &countingReader{data: data}
	_, err := Read(src, WithMaxElementLen(cap))

	lim, ok := errors.AsType[*LimitExceededError](err)
	if !ok {
		t.Fatalf("Read = %v, want *LimitExceededError", err)
	}
	if lim.Kind != "element-length" {
		t.Errorf("LimitExceededError.Kind = %q, want %q", lim.Kind, "element-length")
	}
	if lim.Limit != cap {
		t.Errorf("LimitExceededError.Limit = %d, want %d", lim.Limit, cap)
	}
	if lim.Tag != TagPixelData {
		t.Errorf("LimitExceededError.Tag = %s, want %s", lim.Tag, TagPixelData)
	}
	// Boundedness: the reader must stop near the cap, not consume the whole hostile
	// stream. 4*cap is generous slack for the file-meta group, the geometry elements,
	// and the item headers counted alongside the fragment values.
	if src.read > 4*cap {
		t.Errorf("consumed %d source bytes after the cap fired, want <= %d", src.read, 4*cap)
	}
	if src.read >= len(data) {
		t.Error("the entire hostile stream was consumed; the aggregate cap did not bound the read")
	}
}

// TestReadEncapsulatedAggregateCapAcceptsStreamAtCap confirms a legitimate stream
// whose retained size stays within the element cap still reads: the aggregate bound
// must not reject valid compressed objects.
func TestReadEncapsulatedAggregateCapAcceptsStreamAtCap(t *testing.T) {
	frag := make([]byte, 64)
	data := buildEncapsulatedPart10(RLELossless, func(out *bytes.Buffer) {
		out.Write(seedItemHeader(uint32(len(frag))))
		out.Write(frag)
		out.Write(seedSequenceDelimiter())
	})
	f, err := Read(bytes.NewReader(data), WithMaxElementLen(1<<10))
	if err != nil {
		t.Fatalf("Read of an in-cap encapsulated stream: %v", err)
	}
	if _, ok := f.DataSet.Get(TagPixelData); !ok {
		t.Error("PixelData element was not retained")
	}
}

// TestReadEncapsulatedNonZeroDelimiterLength pins the structural rule on the read
// layer: a Sequence Delimitation Item must carry length zero (PS3.5 A.4). Silent
// acceptance of a non-zero length would desynchronise the outer element loop, so it
// is a typed error — never a dataset whose following elements were misparsed.
func TestReadEncapsulatedNonZeroDelimiterLength(t *testing.T) {
	data := buildEncapsulatedPart10(RLELossless, func(out *bytes.Buffer) {
		out.Write(seedItemHeader(4))
		out.Write([]byte{1, 2, 3, 4})
		delim := seedSequenceDelimiter()
		delim[4] = 0x04 // declare a non-zero delimiter length
		out.Write(delim)
		// Bytes a desynchronised parser would misread as the next element.
		out.Write([]byte{0xDE, 0xAD, 0xBE, 0xEF})
	})
	_, err := Read(bytes.NewReader(data))
	ve, ok := errors.AsType[*ValueError](err)
	if !ok {
		t.Fatalf("Read = %v, want *ValueError for a non-zero Sequence Delimitation Item length", err)
	}
	if ve.Tag != tagSequenceDelimit {
		t.Errorf("ValueError.Tag = %s, want %s", ve.Tag, tagSequenceDelimit)
	}
}

// TestReadEncapsulatedOddFragmentLength pins the even-length rule on the read layer:
// encapsulated item values are even (PS3.5 A.4), so an odd item length is a typed
// error at Read rather than a stream that parseEncapsulated later rejects.
func TestReadEncapsulatedOddFragmentLength(t *testing.T) {
	data := buildEncapsulatedPart10(RLELossless, func(out *bytes.Buffer) {
		out.Write(seedItemHeader(3))
		out.Write([]byte{1, 2, 3})
		out.Write(seedSequenceDelimiter())
	})
	_, err := Read(bytes.NewReader(data))
	if _, ok := errors.AsType[*ValueError](err); !ok {
		t.Fatalf("Read = %v, want *ValueError for an odd fragment item length", err)
	}
}

// TestEncapsulatedReadAndParseLayersRejectConsistently pins the two-layer contract:
// every malformed fragment stream the read layer rejects must also be rejected by
// parseEncapsulated (NewPixelData's validator), so a stream can never be accepted on
// read and then rejected on decode, or vice versa, for these structural classes. The
// read layer is deliberately laxer only about frame mapping, which needs the
// dataset's NumberOfFrames and stays in NewPixelData.
func TestEncapsulatedReadAndParseLayersRejectConsistently(t *testing.T) {
	cases := []struct {
		name   string
		stream func(out *bytes.Buffer) // fragment stream after the (7FE0,0010) header
	}{
		{
			name: "odd item length",
			stream: func(out *bytes.Buffer) {
				out.Write(seedItemHeader(0)) // empty Basic Offset Table
				out.Write(seedItemHeader(3))
				out.Write([]byte{1, 2, 3})
				out.Write(seedSequenceDelimiter())
			},
		},
		{
			name: "non-zero delimiter length",
			stream: func(out *bytes.Buffer) {
				out.Write(seedItemHeader(0))
				out.Write(seedItemHeader(2))
				out.Write([]byte{1, 2})
				delim := seedSequenceDelimiter()
				delim[4] = 0x02
				out.Write(delim)
			},
		},
		{
			name: "undefined item length",
			stream: func(out *bytes.Buffer) {
				out.Write(seedItemHeader(0))
				out.Write(seedItemHeader(undefinedLength))
				out.Write(seedSequenceDelimiter())
			},
		},
		{
			name: "foreign tag in the stream",
			stream: func(out *bytes.Buffer) {
				out.Write(seedItemHeader(0))
				foreign := make([]byte, 8)
				copy(foreign, []byte{0x08, 0x00, 0x16, 0x00, 0x00, 0x00, 0x00, 0x00})
				out.Write(foreign)
				out.Write(seedSequenceDelimiter())
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var value bytes.Buffer
			tc.stream(&value)
			if _, err := parseEncapsulated(value.Bytes(), 0); err == nil {
				t.Error("parseEncapsulated accepted a stream the read layer must reject")
			}

			var part10 bytes.Buffer
			tc.stream(&part10)
			data := buildEncapsulatedPart10(RLELossless, func(out *bytes.Buffer) {
				// buildEncapsulatedPart10 already wrote the empty Basic Offset Table
				// item; skip the stream's own leading BOT item (8 bytes).
				out.Write(part10.Bytes()[8:])
			})
			if _, err := Read(bytes.NewReader(data)); err == nil {
				t.Error("Read accepted a stream parseEncapsulated rejects")
			}
		})
	}
}

// TestSetPixelDataRemovesStaleEncapsulatedTotalLength pins the staleness rule for
// (7FE0,0003) Encapsulated Pixel Data Value Total Length: it describes the previous
// stream's byte length, so SetPixelData must drop it for a native target and for an
// encapsulated target alike.
func TestSetPixelDataRemovesStaleEncapsulatedTotalLength(t *testing.T) {
	for _, target := range []TransferSyntax{ExplicitVRLittleEndian, RLELossless} {
		t.Run(string(target), func(t *testing.T) {
			f, err := ReadFile(filepath.Join("..", "testdata", "dicom", "MR2_UNCI.dcm"))
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			pd, err := NewPixelData(f.DataSet, f.Meta.TransferSyntaxUID)
			if err != nil {
				t.Fatalf("NewPixelData: %v", err)
			}
			out, err := Transcode(pd, target)
			if err != nil {
				t.Fatalf("Transcode: %v", err)
			}
			f.DataSet.Set(Element{
				Tag: TagEncapsulatedPixelDataValueTotalLength,
				VR:  VRUV, Value: NewInts(VRUV, 12345),
			})
			if err := f.SetPixelData(out); err != nil {
				t.Fatalf("SetPixelData: %v", err)
			}
			if _, ok := f.DataSet.Get(TagEncapsulatedPixelDataValueTotalLength); ok {
				t.Error("stale EncapsulatedPixelDataValueTotalLength survived SetPixelData")
			}
		})
	}
}
