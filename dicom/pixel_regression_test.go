package dicom

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
)

// TestNativeDecodeSCRGBRegression is the named regression for native decode of an
// uncompressed fixture: SC_rgb_expb.dcm is uncompressed RGB (Explicit VR Big Endian),
// and its single frame is the contiguous 100x100x3 buffer.
func TestNativeDecodeSCRGBRegression(t *testing.T) {
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "SC_rgb_expb.dcm"))
	if err != nil {
		t.Fatalf("ReadPixelData: %v", err)
	}
	if pd.IsEncapsulated() {
		t.Fatal("SC_rgb_expb.dcm is uncompressed; IsEncapsulated() should be false")
	}

	var frames int
	for frame, err := range pd.Frames() {
		if err != nil {
			t.Fatalf("frame %d: %v", frames, err)
		}
		if len(frame.Pixels) != 100*100*3 {
			t.Errorf("frame %d length = %d, want %d", frames, len(frame.Pixels), 100*100*3)
		}
		frames++
	}
	if frames != 1 {
		t.Errorf("decoded %d frames, want 1", frames)
	}
}

// TestUnknownCompressedSyntaxReturnsCodecUnavailable checks the degradation for any
// compressed syntax with no registered codec, using a synthetic encapsulated stream
// under an unregistered JPEG-LS UID.
func TestUnknownCompressedSyntaxReturnsCodecUnavailable(t *testing.T) {
	const jpegLS TransferSyntax = "1.2.840.10008.1.2.4.80"

	ds := NewDataSet()
	ds.Set(Element{Tag: TagRows, VR: VRUS, Value: NewInts(VRUS, 2)})
	ds.Set(Element{Tag: TagColumns, VR: VRUS, Value: NewInts(VRUS, 2)})
	ds.Set(Element{Tag: TagBitsAllocated, VR: VRUS, Value: NewInts(VRUS, 8)})

	var s bytes.Buffer
	s.Write(itemHeader(0)) // empty BOT
	s.Write(itemHeader(4))
	s.Write([]byte{1, 2, 3, 4})
	s.Write(seqDelim())

	pd, err := NewEncapsulatedPixelData(ds, jpegLS, s.Bytes())
	if err != nil {
		t.Fatalf("NewEncapsulatedPixelData: %v", err)
	}
	var gotErr error
	for _, err := range pd.Frames() {
		if err != nil {
			gotErr = err
			break
		}
	}
	if !errors.Is(gotErr, ErrCodecUnavailable) {
		t.Errorf("error = %v, want ErrCodecUnavailable for an unregistered compressed syntax", gotErr)
	}
}
