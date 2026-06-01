//go:build cgo && (dicom_openjpeg || dicom_libjpeg || dicom_charls)

package dicom

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
