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

// TestJPEG2000ReturnsCodecUnavailable is the §7.3 degradation: a JPEG 2000 instance
// in a pure-Go build has no registered codec, so requesting frames yields a typed
// ErrCodecUnavailable naming the transfer syntax, never a build break or a partial
// image. liver_j2k.dcm is JPEG 2000 Lossless (1.2.840.10008.1.2.4.90).
func TestJPEG2000ReturnsCodecUnavailable(t *testing.T) {
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "liver_j2k.dcm"))
	if err != nil {
		t.Fatalf("ReadPixelData: %v", err)
	}
	if !pd.IsEncapsulated() {
		t.Fatal("liver_j2k.dcm is encapsulated; IsEncapsulated() should be true")
	}

	const jpeg2000Lossless TransferSyntax = "1.2.840.10008.1.2.4.90"
	if pd.Geometry.TransferSyntax != jpeg2000Lossless {
		t.Fatalf("transfer syntax = %s, want JPEG 2000 Lossless", pd.Geometry.TransferSyntax)
	}

	var gotErr error
	for _, err := range pd.Frames() {
		if err != nil {
			gotErr = err
			break
		}
	}
	if gotErr == nil {
		t.Fatal("expected ErrCodecUnavailable for a JPEG 2000 instance in a pure-Go build")
	}
	if !errors.Is(gotErr, ErrCodecUnavailable) {
		t.Errorf("error = %v, want it to match ErrCodecUnavailable", gotErr)
	}
	var cue *CodecUnavailableError
	if !errors.As(gotErr, &cue) {
		t.Fatalf("error %v is not a *CodecUnavailableError", gotErr)
	}
	if cue.TransferSyntax != jpeg2000Lossless {
		t.Errorf("CodecUnavailableError names %s, want JPEG 2000 Lossless", cue.TransferSyntax)
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
