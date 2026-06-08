//go:build cgo && (dicom_openjpeg || dicom_libjpeg || dicom_charls)

package dicom

import (
	"os"
	"testing"
)

// skipHostileUnderASAN skips a whole codec hostile-input test when the sanitizer
// build exports RADX_ASAN=1 (the env var the ASAN CI step sets). Feeding malformed
// codestreams to the vendored OpenJPEG/CharLS/libjpeg-turbo C decoders trips latent
// upstream out-of-bounds reads that ASAN turns into non-deterministic SIGSEGVs
// (tracked in #107), so the hostile-malformed-input corpora are exercised only in the
// non-ASAN passes (build/test/race), where they already assert clean rejection with a
// typed error. The ASAN pass instead covers the valid-data codec paths and the cgo
// boundary, which is deterministic and catches go-radx's own memory bugs.
func skipHostileUnderASAN(t *testing.T) {
	t.Helper()
	if os.Getenv("RADX_ASAN") != "" {
		t.Skip("hostile-malformed-input corpus is exercised in the non-ASAN passes; " +
			"excluded from ASAN because malformed input trips latent upstream OpenJPEG/CharLS OOBs " +
			"(tracked in #107). ASAN covers the valid-data codec paths and the cgo boundary.")
	}
}

// clone returns a copy of b. The hostile-input corpora derive their tampered cases
// from a real codestream and must not mutate the shared fixture buffer.
func clone(b []byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

// garbage returns n deterministic pseudo-random bytes (no crypto needed; the corpus
// just needs non-codestream bytes that vary). It is shared by the codec hostile-input
// tests across build tags.
func garbage(n int) []byte {
	out := make([]byte, n)
	x := uint32(0x9E3779B9)
	for i := range out {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		out[i] = byte(x)
	}
	return out
}
