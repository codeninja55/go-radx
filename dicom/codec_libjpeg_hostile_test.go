//go:build cgo && dicom_libjpeg

package dicom

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// jpegHostileWorkerEnv puts the test binary into single-decode worker mode for the
// JPEG hostile corpus. When set, the worker runs one hostile decode and exits, so
// the parent can bound it with a timeout (DCM-015: a hang must fail the test, never
// wedge CI).
const jpegHostileWorkerEnv = "GORADX_JPEG_HOSTILE_WORKER"

// jpegHostileGeom is the small fixed geometry the worker decodes against, so an
// oversized-dimension codestream is rejected by the dimension cap.
var jpegHostileGeom = PixelGeometry{
	Rows: 64, Columns: 64, SamplesPerPixel: 3, BitsAllocated: 8, BitsStored: 8,
	PhotometricInterpretation: "RGB",
}

// TestJPEGHostileInputs feeds malformed, truncated, and dimension-tampered JPEG
// codestreams to the decoder. Each case must fail with a typed *jpegError (wrapping
// ErrJPEG) and must not crash or hang; every decode runs in a subprocess with a
// timeout so a hang is a killed process and a test failure (DCM-015).
func TestJPEGHostileInputs(t *testing.T) {
	if os.Getenv(jpegHostileWorkerEnv) != "" {
		runJPEGHostileWorker() // never returns
		return
	}

	corpus := jpegHostileCorpus(t)
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("locate test binary: %v", err)
	}

	for name, payload := range corpus {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "case.jpg")
			if err := os.WriteFile(path, payload, 0o600); err != nil {
				t.Fatalf("write case: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, exe,
				"-test.run", "^TestJPEGHostileInputs$", "-test.timeout", "30s")
			cmd.Env = append(os.Environ(), jpegHostileWorkerEnv+"="+path)
			out, err := cmd.CombinedOutput()

			if ctx.Err() == context.DeadlineExceeded {
				t.Fatalf("decode hung on hostile input (killed after timeout); output:\n%s", out)
			}
			if err != nil {
				t.Fatalf("worker exited with error %v; output:\n%s", err, out)
			}
		})
	}
}

// runJPEGHostileWorker decodes the single hostile JPEG codestream named by the worker
// env var. It exits 0 if the decoder rejected the input with a typed *jpegError, and
// non-zero otherwise (including a clean decode, which would mean the bounds checks
// did not hold).
func runJPEGHostileWorker() {
	path := os.Getenv(jpegHostileWorkerEnv)
	payload, err := os.ReadFile(path)
	if err != nil {
		os.Stderr.WriteString("worker: read payload: " + err.Error() + "\n")
		os.Exit(2)
	}

	c := libjpegCodec{ts: JPEGBaseline8Bit}
	_, decErr := c.Decode(payload, jpegHostileGeom)
	if decErr == nil {
		os.Stderr.WriteString("worker: hostile input decoded without error\n")
		os.Exit(1)
	}
	var je *jpegError
	if !errors.As(decErr, &je) || !errors.Is(decErr, ErrJPEG) {
		os.Stderr.WriteString("worker: error is not a typed jpegError: " + decErr.Error() + "\n")
		os.Exit(1)
	}
	os.Exit(0)
}

// jpegHostileCorpus builds the malformed-input cases, deriving the truncated and
// dimension-tampered cases from a real baseline codestream so they are structurally
// close to a valid one.
func jpegHostileCorpus(t *testing.T) map[string][]byte {
	t.Helper()
	valid := validBaselineCodestream(t)

	return map[string][]byte{
		"empty":            {},
		"single_byte":      {0xFF},
		"soi_only":         {0xFF, 0xD8},
		"random_garbage":   garbage(4096),
		"zero_filled":      make([]byte, 8192),
		"truncated_header": clone(valid[:32]),
		"truncated_mid":    clone(valid[:len(valid)/2]),
		"truncated_tail":   clone(valid[:len(valid)-4]),
		"oversized_dims":   jpegOversizedDimensions(valid),
		"zero_dims":        jpegZeroDimensions(valid),
	}
}

// validBaselineCodestream returns the first frame's raw JPEG codestream from the
// baseline RGB fixture, the seed for the structurally-close hostile cases.
func validBaselineCodestream(t *testing.T) []byte {
	t.Helper()
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "SC_jpeg_no_color_transform.dcm"))
	if err != nil {
		t.Fatalf("ReadPixelData: %v", err)
	}
	encoded, err := pd.frameEncodedBytes()
	if err != nil {
		t.Fatalf("frameEncodedBytes: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatal("baseline fixture yielded no frames")
	}
	return clone(encoded[0])
}

// sofFrameDims tampers with the height (lines) and width (samples per line) fields of
// the first SOF0 (Start Of Frame, baseline DCT) marker in a JPEG codestream. The SOF0
// payload after the 0xFFC0 marker and 2-byte length is: precision (1), height (2),
// width (2). Returns the tampered copy, or the input unchanged if no SOF0 is found.
func sofFrameDims(valid []byte, height, width uint16) []byte {
	b := clone(valid)
	for i := 0; i+9 < len(b); i++ {
		if b[i] == 0xFF && b[i+1] == 0xC0 {
			// b[i+2:i+4] = segment length; b[i+4] = precision; height at i+5, width at i+7.
			binary.BigEndian.PutUint16(b[i+5:], height)
			binary.BigEndian.PutUint16(b[i+7:], width)
			return b
		}
	}
	return b
}

// jpegOversizedDimensions sets the SOF0 height/width larger than the worker cap so
// the dimension check rejects the header before any pixel buffer is allocated.
func jpegOversizedDimensions(valid []byte) []byte {
	return sofFrameDims(valid, 50000, 50000)
}

// jpegZeroDimensions sets the SOF0 height/width to zero to probe the non-positive
// dimension guard.
func jpegZeroDimensions(valid []byte) []byte {
	return sofFrameDims(valid, 0, 0)
}
