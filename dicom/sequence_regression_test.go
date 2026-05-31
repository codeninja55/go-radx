package dicom

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// DCM-005: a fixture with nested undefined-length sequences is parsed STRUCTURALLY
// (navigable nested datasets), not dropped or held opaquely, and re-encodes
// byte-identically. liver.dcm's ReferencedSeriesSequence (0008,1115) holds an item
// whose ReferencedInstanceSequence (0008,114A) is itself a nested sequence of
// items carrying ReferencedSOPClassUID (0008,1150) and ReferencedSOPInstanceUID
// (0008,1155). The test navigates that two-level nesting.
func TestRegressionDCM005StructuredNestedSequence(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "testdata", "dicom", "liver.dcm"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	f, err := Read(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	// Navigate: (0008,1115) -> item[0] -> (0008,114A) -> item[0] -> SOP UIDs.
	outer, ok := f.DataSet.GetSequence(NewTag(0x0008, 0x1115))
	if !ok {
		t.Fatal("ReferencedSeriesSequence (0008,1115) is not a structured sequence (DCM-005)")
	}
	if outer.Len() == 0 {
		t.Fatal("ReferencedSeriesSequence has no items (DCM-005: sequence dropped/blanked)")
	}

	var foundInstance bool
	for series := range outer.Items() {
		inner, ok := series.DataSet.GetSequence(NewTag(0x0008, 0x114A))
		if !ok {
			continue
		}
		for inst := range inner.Items() {
			if uid, ok := inst.DataSet.GetUID(NewTag(0x0008, 0x1155)); ok && uid != "" {
				foundInstance = true
			}
		}
	}
	if !foundInstance {
		t.Error("could not navigate to a ReferencedSOPInstanceUID inside the nested sequence (DCM-005)")
	}

	// And the whole dataset re-encodes byte-identically, proving the structured model
	// preserves the source length forms (undefined-length items and sequences).
	br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
	if _, err := readPreamble(br); err != nil {
		t.Fatalf("readPreamble: %v", err)
	}
	h, _ := readElementHeader(br, ExplicitVRLittleEndian)
	gv, _ := decodeValue(br, h, encodingFor(ExplicitVRLittleEndian), nil)
	groupLen := gv.(*Ints).Ints()[0]
	mainStart := br.offset() + groupLen
	originalMain := raw[mainStart:]

	var out bytes.Buffer
	if err := writeDataSet(&out, f.DataSet, f.Meta.TransferSyntaxUID); err != nil {
		t.Fatalf("writeDataSet: %v", err)
	}
	if !bytes.Equal(out.Bytes(), originalMain) {
		t.Errorf("structured sequence re-encode not byte-identical: got %d bytes, want %d bytes (DCM-005)",
			out.Len(), len(originalMain))
	}
}

// DCM-005: MR2_UNCI.dcm (Explicit VR LE, undefined-length sequences) round-trips
// byte-identically through the structured sequence model.
func TestRegressionDCM005MR2ByteIdentical(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "testdata", "dicom", "MR2_UNCI.dcm"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	f, err := Read(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	br := newBoundedReader(bytes.NewReader(raw), defaultMaxElementLen)
	if _, err := readPreamble(br); err != nil {
		t.Fatalf("readPreamble: %v", err)
	}
	h, _ := readElementHeader(br, ExplicitVRLittleEndian)
	gv, _ := decodeValue(br, h, encodingFor(ExplicitVRLittleEndian), nil)
	groupLen := gv.(*Ints).Ints()[0]
	mainStart := br.offset() + groupLen
	originalMain := raw[mainStart:]

	var out bytes.Buffer
	if err := writeDataSet(&out, f.DataSet, f.Meta.TransferSyntaxUID); err != nil {
		t.Fatalf("writeDataSet: %v", err)
	}
	if !bytes.Equal(out.Bytes(), originalMain) {
		t.Errorf("MR2_UNCI.dcm main dataset re-encode not byte-identical: got %d bytes, want %d bytes (DCM-005)",
			out.Len(), len(originalMain))
	}
}

// The depth cap rejects a maliciously deep sequence with a typed LimitExceededError
// of Kind "sequence-depth" rather than overflowing the stack. The synthetic dataset
// nests undefined-length SQ elements one inside another past the configured cap.
func TestRegressionSequenceDepthCap(t *testing.T) {
	const cap = 8
	// Build a chain of `cap+2` nested undefined-length SQ headers, each opening one
	// item that contains the next SQ. The innermost item is empty. We never close the
	// sequences (the parser must reject on depth before reaching the delimiters).
	var buf bytes.Buffer
	sqTag := NewTag(0x0040, 0xA730) // ContentSequence (a real SQ tag)
	for i := 0; i < cap+2; i++ {
		buf.Write(sqHeaderUndefinedLE(sqTag))
		buf.Write(itemHeaderUndefinedLE())
	}

	br := newBoundedReader(bytes.NewReader(buf.Bytes()), defaultMaxElementLen)
	_, err := readDataSet(br, ExplicitVRLittleEndian, newReadConfig(WithMaxSequenceDepth(cap)))
	var le *LimitExceededError
	if !errors.As(err, &le) {
		t.Fatalf("deep sequence error = %v, want *LimitExceededError", err)
	}
	if le.Kind != "sequence-depth" {
		t.Errorf("LimitExceededError.Kind = %q, want %q", le.Kind, "sequence-depth")
	}
}

// A hostile undefined-length sequence that never emits its Sequence Delimitation
// Item is a truncation (io.ErrUnexpectedEOF), not an infinite loop or a silently
// accepted dataset (Codex DCM-003).
func TestRegressionHostileUndefinedLengthSequence(t *testing.T) {
	var buf bytes.Buffer
	buf.Write(sqHeaderUndefinedLE(NewTag(0x0040, 0xA730)))
	buf.Write(itemHeaderUndefinedLE())
	// One element inside the item, then the stream ends with no item or sequence
	// delimiter.
	buf.Write(buildElementLE(t, NewTag(0x0010, 0x0010), VRPN, NewStrings(VRPN, "Doe^Jane")))

	br := newBoundedReader(bytes.NewReader(buf.Bytes()), defaultMaxElementLen)
	_, err := readDataSet(br, ExplicitVRLittleEndian, newReadConfig())
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("unterminated undefined-length sequence = %v, want io.ErrUnexpectedEOF (DCM-003)", err)
	}
}

// sqHeaderUndefinedLE builds an Explicit VR LE SQ element header with the
// undefined-length sentinel.
func sqHeaderUndefinedLE(tag Tag) []byte {
	b := make([]byte, 0, 12)
	b = append(b, leTag(tag.Group(), tag.Element())...)
	b = append(b, 'S', 'Q', 0x00, 0x00) // VR + 2-byte reserved
	b = append(b, le32(undefinedLength)...)
	return b
}

// itemHeaderUndefinedLE builds an undefined-length item header (FFFE,E000).
func itemHeaderUndefinedLE() []byte {
	b := make([]byte, 0, 8)
	b = append(b, leTag(0xFFFE, 0xE000)...)
	b = append(b, le32(undefinedLength)...)
	return b
}

// buildElementLE encodes one element header+value in Explicit VR LE.
func buildElementLE(t *testing.T, tag Tag, vr VR, v Value) []byte {
	t.Helper()
	return buildElement(t, ExplicitVRLittleEndian, tag, vr, v)
}
