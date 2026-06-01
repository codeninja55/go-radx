//go:build !(cgo && dicom_libjpeg)

package dicom

import (
	"errors"
	"path/filepath"
	"testing"
)

// TestNoCodecForBaselineJPEGInPureGo asserts the lossy JPEG syntaxes have no
// registered codec without the dicom_libjpeg build tag. The companion
// TestJPEGCodecsRegistered asserts the opposite once the codec is built in.
func TestNoCodecForBaselineJPEGInPureGo(t *testing.T) {
	for _, ts := range []TransferSyntax{JPEGBaseline8Bit, JPEGExtended12Bit} {
		if c, ok := lookupCodec(ts); ok {
			t.Fatalf("did not expect a %s codec without dicom_libjpeg, got %T", ts.Name(), c)
		}
	}
}

// TestBaselineJPEGReturnsCodecUnavailable is the §7.3 degradation, asserted only in a
// build WITHOUT the dicom_libjpeg codec: a JPEG Baseline instance has no registered
// codec, so requesting frames yields a typed ErrCodecUnavailable naming the transfer
// syntax, never a build break or a partial image.
func TestBaselineJPEGReturnsCodecUnavailable(t *testing.T) {
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "SC_jpeg_no_color_transform.dcm"))
	if err != nil {
		t.Fatalf("ReadPixelData: %v", err)
	}
	if pd.Geometry.TransferSyntax != JPEGBaseline8Bit {
		t.Fatalf("transfer syntax = %s, want JPEG Baseline", pd.Geometry.TransferSyntax)
	}

	var gotErr error
	for _, err := range pd.Frames() {
		if err != nil {
			gotErr = err
			break
		}
	}
	if !errors.Is(gotErr, ErrCodecUnavailable) {
		t.Errorf("error = %v, want ErrCodecUnavailable", gotErr)
	}
	var cue *CodecUnavailableError
	if !errors.As(gotErr, &cue) {
		t.Fatalf("error %v is not a *CodecUnavailableError", gotErr)
	}
	if cue.TransferSyntax != JPEGBaseline8Bit {
		t.Errorf("CodecUnavailableError names %s, want JPEG Baseline", cue.TransferSyntax)
	}
}

// TestExtendedJPEGReturnsCodecUnavailable mirrors the baseline degradation for the
// extended 12-bit fixture.
func TestExtendedJPEGReturnsCodecUnavailable(t *testing.T) {
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "JPGExtended.dcm"))
	if err != nil {
		t.Fatalf("ReadPixelData: %v", err)
	}
	if pd.Geometry.TransferSyntax != JPEGExtended12Bit {
		t.Fatalf("transfer syntax = %s, want JPEG Extended", pd.Geometry.TransferSyntax)
	}
	var gotErr error
	for _, err := range pd.Frames() {
		if err != nil {
			gotErr = err
			break
		}
	}
	if !errors.Is(gotErr, ErrCodecUnavailable) {
		t.Errorf("error = %v, want ErrCodecUnavailable", gotErr)
	}
}
