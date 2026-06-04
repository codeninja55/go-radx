//go:build cgo && (dicom_openjpeg || dicom_charls)

package dicom

import "testing"

// decodeNativeFrames decodes every frame of pd into native pixel bytes for an encode
// benchmark that times only the codec. Only the encode-capable codecs (OpenJPEG JPEG 2000
// Lossless and CharLS JPEG-LS Lossless) use it, so it is compiled under their build tags.
func decodeNativeFrames(b *testing.B, pd *PixelData) [][]byte {
	b.Helper()
	var frames [][]byte
	for frame, err := range pd.Frames() {
		if err != nil {
			b.Fatalf("decode frame: %v", err)
		}
		frames = append(frames, frame.Pixels)
	}
	return frames
}
