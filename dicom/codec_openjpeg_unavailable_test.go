//go:build !(cgo && dicom_openjpeg)

package dicom

import (
	"errors"
	"path/filepath"
	"testing"
)

// TestNoCodecForJPEG2000InPureGo asserts the JPEG 2000 family (classic and
// High-Throughput) has no registered codec without the dicom_openjpeg build tag. The
// companions TestJPEG2000CodecRegistered and TestHTJ2KCodecsRegistered assert the
// opposite once the codec is built in.
func TestNoCodecForJPEG2000InPureGo(t *testing.T) {
	for _, ts := range []TransferSyntax{
		JPEG2000Lossless, JPEG2000,
		HTJ2KLossless, HTJ2KLosslessRPCL, HTJ2K,
	} {
		if c, ok := lookupCodec(ts); ok {
			t.Fatalf("did not expect a %s codec in pure Go, got %T", ts.Name(), c)
		}
	}
}

// TestHTJ2KReturnsCodecUnavailable is the §7.3 degradation for High-Throughput JPEG
// 2000, asserted only in a build WITHOUT the dicom_openjpeg codec: an HTJ2K instance
// has no registered codec, so requesting frames yields a typed ErrCodecUnavailable
// naming the transfer syntax.
func TestHTJ2KReturnsCodecUnavailable(t *testing.T) {
	for _, f := range []struct {
		file string
		ts   TransferSyntax
	}{
		{"HTJ2KLossless_08_RGB.dcm", HTJ2KLossless},
		{"HTJ2K_08_RGB.dcm", HTJ2K},
	} {
		pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", f.file))
		if err != nil {
			t.Fatalf("%s: ReadPixelData: %v", f.file, err)
		}
		if pd.Geometry.TransferSyntax != f.ts {
			t.Fatalf("%s: transfer syntax = %s, want %s", f.file, pd.Geometry.TransferSyntax, f.ts)
		}
		var gotErr error
		for _, err := range pd.Frames() {
			if err != nil {
				gotErr = err
				break
			}
		}
		if !errors.Is(gotErr, ErrCodecUnavailable) {
			t.Errorf("%s: error = %v, want ErrCodecUnavailable", f.file, gotErr)
		}
	}
}

// TestJPEG2000ReturnsCodecUnavailable is the §7.3 degradation, asserted only in a
// build WITHOUT the dicom_openjpeg codec: a JPEG 2000 instance has no registered
// codec, so requesting frames yields a typed ErrCodecUnavailable naming the transfer
// syntax, never a build break or a partial image. liver_j2k.dcm is JPEG 2000
// Lossless (1.2.840.10008.1.2.4.90). The companion in codec_openjpeg_test.go asserts
// the opposite once the codec is built in.
func TestJPEG2000ReturnsCodecUnavailable(t *testing.T) {
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "liver_j2k.dcm"))
	if err != nil {
		t.Fatalf("ReadPixelData: %v", err)
	}
	if !pd.IsEncapsulated() {
		t.Fatal("liver_j2k.dcm is encapsulated; IsEncapsulated() should be true")
	}

	if pd.Geometry.TransferSyntax != JPEG2000Lossless {
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
	if cue.TransferSyntax != JPEG2000Lossless {
		t.Errorf("CodecUnavailableError names %s, want JPEG 2000 Lossless", cue.TransferSyntax)
	}
}

// TestTranscodeToJPEG2000Unsupported is the off-by-default boundary: re-encoding to a
// JPEG 2000 syntax with no codec built in returns ErrEncodeUnsupported, never a
// silent or corrupt result. The dicom_openjpeg build registers an encoder for the
// lossless syntax, so this assertion holds only without the tag.
func TestTranscodeToJPEG2000Unsupported(t *testing.T) {
	src := twoFrameNativePixelData(t)

	_, err := Transcode(src, JPEG2000Lossless)
	if err == nil {
		t.Fatal("expected ErrEncodeUnsupported transcoding to JPEG 2000 without the codec")
	}
	if !errors.Is(err, ErrEncodeUnsupported) {
		t.Errorf("error = %v, want ErrEncodeUnsupported", err)
	}
	var eu *EncodeUnsupportedError
	if !errors.As(err, &eu) {
		t.Fatalf("error %v is not an *EncodeUnsupportedError", err)
	}
	if eu.TransferSyntax != JPEG2000Lossless {
		t.Errorf("error names %s, want JPEG 2000 Lossless", eu.TransferSyntax)
	}
}

// TestTranscodeFromJPEG2000Unavailable transcoding a JPEG 2000 source without the
// codec built in fails to decode the source frames, surfacing ErrCodecUnavailable.
func TestTranscodeFromJPEG2000Unavailable(t *testing.T) {
	src, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "liver_j2k.dcm"))
	if err != nil {
		t.Fatalf("ReadPixelData: %v", err)
	}
	if _, err := Transcode(src, ExplicitVRLittleEndian); !errors.Is(err, ErrCodecUnavailable) {
		t.Errorf("error = %v, want ErrCodecUnavailable decoding a JPEG 2000 source", err)
	}
}
