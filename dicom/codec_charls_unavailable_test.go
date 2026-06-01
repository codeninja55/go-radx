//go:build !(cgo && dicom_charls)

package dicom

import (
	"errors"
	"path/filepath"
	"testing"
)

// TestNoCodecForJPEGLSInPureGo asserts the JPEG-LS syntaxes have no registered codec
// without the dicom_charls build tag. The companion TestJPEGLSCodecsRegistered
// asserts the opposite once the codec is built in.
func TestNoCodecForJPEGLSInPureGo(t *testing.T) {
	for _, ts := range []TransferSyntax{JPEGLSLossless, JPEGLSNearLossless} {
		if c, ok := lookupCodec(ts); ok {
			t.Fatalf("did not expect a %s codec without dicom_charls, got %T", ts.Name(), c)
		}
	}
}

// TestJPEGLSReturnsCodecUnavailable is the §7.3 degradation, asserted only in a build
// WITHOUT the dicom_charls codec: a JPEG-LS instance has no registered codec, so
// requesting frames yields a typed ErrCodecUnavailable naming the transfer syntax,
// never a build break or a partial image.
func TestJPEGLSReturnsCodecUnavailable(t *testing.T) {
	for _, f := range []struct {
		file string
		ts   TransferSyntax
	}{
		{"MR_small_jpeg_ls_lossless.dcm", JPEGLSLossless},
		{"JPEGLSNearLossless_08.dcm", JPEGLSNearLossless},
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
		var cue *CodecUnavailableError
		if !errors.As(gotErr, &cue) {
			t.Fatalf("%s: error %v is not a *CodecUnavailableError", f.file, gotErr)
		}
	}
}

// TestTranscodeToJPEGLSUnsupported is the off-by-default boundary: re-encoding to
// JPEG-LS Lossless with no codec built in returns ErrEncodeUnsupported. The
// dicom_charls build registers an encoder for the lossless syntax, so this assertion
// holds only without the tag.
func TestTranscodeToJPEGLSUnsupported(t *testing.T) {
	src := twoFrameNativePixelData(t)

	_, err := Transcode(src, JPEGLSLossless)
	if err == nil {
		t.Fatal("expected ErrEncodeUnsupported transcoding to JPEG-LS without the codec")
	}
	if !errors.Is(err, ErrEncodeUnsupported) {
		t.Errorf("error = %v, want ErrEncodeUnsupported", err)
	}
}
