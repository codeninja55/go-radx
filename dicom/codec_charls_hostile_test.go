//go:build cgo && dicom_charls

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

// jpeglsHostileWorkerEnv puts the test binary into single-decode worker mode for the
// JPEG-LS hostile corpus. When set, the worker runs one hostile decode and exits, so
// the parent can bound it with a timeout (DCM-015: a hang must fail the test, never
// wedge CI).
const jpeglsHostileWorkerEnv = "GORADX_JPEGLS_HOSTILE_WORKER"

// jpeglsHostileGeom is the small fixed geometry the worker decodes against, so an
// oversized-dimension codestream is rejected by the dimension cap.
var jpeglsHostileGeom = PixelGeometry{
	Rows: 64, Columns: 64, SamplesPerPixel: 1, BitsAllocated: 16, BitsStored: 16,
}

// TestJPEGLSHostileInputs feeds malformed, truncated, and dimension-tampered JPEG-LS
// codestreams to the decoder. Each case must fail with a typed *jpeglsError (wrapping
// ErrJPEGLS) and must not crash or hang; every decode runs in a subprocess with a
// timeout so a hang is a killed process and a test failure (DCM-015).
func TestJPEGLSHostileInputs(t *testing.T) {
	if os.Getenv(jpeglsHostileWorkerEnv) != "" {
		runJPEGLSHostileWorker() // never returns
		return
	}

	corpus := jpeglsHostileCorpus(t)
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("locate test binary: %v", err)
	}

	for name, payload := range corpus {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "case.jls")
			if err := os.WriteFile(path, payload, 0o600); err != nil {
				t.Fatalf("write case: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, exe,
				"-test.run", "^TestJPEGLSHostileInputs$", "-test.timeout", "30s")
			cmd.Env = append(os.Environ(), jpeglsHostileWorkerEnv+"="+path)
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

// runJPEGLSHostileWorker decodes the single hostile JPEG-LS codestream named by the
// worker env var. It exits 0 if the decoder rejected the input with a typed
// *jpeglsError, and non-zero otherwise (including a clean decode, which would mean
// the bounds checks did not hold).
func runJPEGLSHostileWorker() {
	path := os.Getenv(jpeglsHostileWorkerEnv)
	payload, err := os.ReadFile(path)
	if err != nil {
		os.Stderr.WriteString("worker: read payload: " + err.Error() + "\n")
		os.Exit(2)
	}

	c := charlsCodec{ts: JPEGLSLossless, canEncode: true}
	_, decErr := c.Decode(payload, jpeglsHostileGeom)
	if decErr == nil {
		os.Stderr.WriteString("worker: hostile input decoded without error\n")
		os.Exit(1)
	}
	var je *jpeglsError
	if !errors.As(decErr, &je) || !errors.Is(decErr, ErrJPEGLS) {
		os.Stderr.WriteString("worker: error is not a typed jpeglsError: " + decErr.Error() + "\n")
		os.Exit(1)
	}
	os.Exit(0)
}

// jpeglsHostileCorpus builds the malformed-input cases, deriving the truncated and
// dimension-tampered cases from a real JPEG-LS codestream so they are structurally
// close to a valid one.
func jpeglsHostileCorpus(t *testing.T) map[string][]byte {
	t.Helper()
	valid := validJPEGLSCodestream(t)

	return map[string][]byte{
		"empty":            {},
		"single_byte":      {0xFF},
		"soi_only":         {0xFF, 0xD8},
		"random_garbage":   garbage(4096),
		"zero_filled":      make([]byte, 8192),
		"truncated_header": clone(valid[:32]),
		"truncated_mid":    clone(valid[:len(valid)/2]),
		"truncated_tail":   clone(valid[:len(valid)-4]),
		"oversized_dims":   jpeglsFrameDims(valid, 50000, 50000),
		"zero_dims":        jpeglsFrameDims(valid, 0, 0),
	}
}

// validJPEGLSCodestream returns the first frame's raw JPEG-LS codestream from the
// lossless fixture, the seed for the structurally-close hostile cases.
func validJPEGLSCodestream(t *testing.T) []byte {
	t.Helper()
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "MR_small_jpeg_ls_lossless.dcm"))
	if err != nil {
		t.Fatalf("ReadPixelData: %v", err)
	}
	encoded, err := pd.frameEncodedBytes()
	if err != nil {
		t.Fatalf("frameEncodedBytes: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatal("JPEG-LS fixture yielded no frames")
	}
	return clone(encoded[0])
}

// jpeglsFrameDims tampers with the height (lines) and width (samples per line) fields
// of the JPEG-LS Start Of Frame marker (SOF55, 0xFFF7). The SOF payload after the
// marker and 2-byte length is: precision (1), height (2), width (2). Returns the
// tampered copy, or the input unchanged if no SOF55 is found.
func jpeglsFrameDims(valid []byte, height, width uint16) []byte {
	b := clone(valid)
	for i := 0; i+9 < len(b); i++ {
		if b[i] == 0xFF && b[i+1] == 0xF7 {
			binary.BigEndian.PutUint16(b[i+5:], height)
			binary.BigEndian.PutUint16(b[i+7:], width)
			return b
		}
	}
	return b
}
