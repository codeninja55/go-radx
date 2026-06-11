package dicom

import (
	"bytes"
	"errors"
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

	var lim *LimitExceededError
	if !errors.As(err, &lim) {
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
