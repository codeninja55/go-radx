package dicom

import (
	"errors"
	"testing"
)

func TestNativeCodecsRegistered(t *testing.T) {
	// The pure-Go pipeline always has the native (uncompressed) codecs available,
	// regardless of build tag.
	for _, ts := range []TransferSyntax{
		ImplicitVRLittleEndian,
		ExplicitVRLittleEndian,
		ExplicitVRBigEndian,
		DeflatedExplicitVRLittleEndian,
	} {
		if _, ok := lookupCodec(ts); !ok {
			t.Errorf("no codec registered for %s (%s)", ts.Name(), ts)
		}
	}
}

func TestNoCodecForJPEG2000InPureGo(t *testing.T) {
	// With no CGo codec build tag, the JPEG family has no registered codec.
	const jpeg2000Lossless TransferSyntax = "1.2.840.10008.1.2.4.90"
	if c, ok := lookupCodec(jpeg2000Lossless); ok {
		t.Fatalf("did not expect a JPEG 2000 codec in pure Go, got %T", c)
	}
}

func TestCodecUnavailableErrorWrapsSentinelAndNamesSyntax(t *testing.T) {
	const jpeg2000Lossless TransferSyntax = "1.2.840.10008.1.2.4.90"
	err := newCodecUnavailable(jpeg2000Lossless)

	if !errors.Is(err, ErrCodecUnavailable) {
		t.Errorf("error %v does not match ErrCodecUnavailable sentinel", err)
	}
	if !contains(err.Error(), string(jpeg2000Lossless)) {
		t.Errorf("error %q should name the transfer syntax %s", err.Error(), jpeg2000Lossless)
	}
}

func TestEncodeUnsupportedErrorWrapsSentinelAndNamesSyntax(t *testing.T) {
	const jpegBaseline TransferSyntax = "1.2.840.10008.1.2.4.50"
	err := newEncodeUnsupported(jpegBaseline)

	if !errors.Is(err, ErrEncodeUnsupported) {
		t.Errorf("error %v does not match ErrEncodeUnsupported sentinel", err)
	}
	if !contains(err.Error(), string(jpegBaseline)) {
		t.Errorf("error %q should name the transfer syntax %s", err.Error(), jpegBaseline)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
