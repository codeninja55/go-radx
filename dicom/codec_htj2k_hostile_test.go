//go:build cgo && dicom_openjpeg

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

// htj2kHostileWorkerEnv puts the test binary into single-decode worker mode for the
// HTJ2K hostile corpus, mirroring the JPEG 2000 harness. The HTJ2K codec routes
// through the same OpenJPEG bridge (goradx_opj_decode) and dimension caps as classic
// JPEG 2000, so this confirms the safety posture (DCM-014/DCM-015) holds for the
// HTJ2K codestream variant too: a malformed or oversized codestream fails with a
// typed *jpeg2000Error in a timeout-guarded subprocess, never a crash or hang.
const htj2kHostileWorkerEnv = "GORADX_HTJ2K_HOSTILE_WORKER"

// htj2kHostileGeom is the small fixed geometry the worker decodes against, so an
// oversized-dimension codestream is rejected by the dimension cap.
var htj2kHostileGeom = PixelGeometry{
	Rows: 64, Columns: 64, SamplesPerPixel: 3, BitsAllocated: 8, BitsStored: 8,
}

// TestHTJ2KHostileInputs feeds malformed, truncated, and dimension-tampered HTJ2K
// codestreams to the decoder. Each case must fail with a typed *jpeg2000Error and
// must not crash or hang; every decode runs in a subprocess with a timeout so a hang
// is a killed process and a test failure (DCM-015).
func TestHTJ2KHostileInputs(t *testing.T) {
	if os.Getenv(htj2kHostileWorkerEnv) != "" {
		runHTJ2KHostileWorker() // never returns
		return
	}
	skipHostileUnderASAN(t)

	corpus := htj2kHostileCorpus(t)
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("locate test binary: %v", err)
	}

	for name, payload := range corpus {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "case.j2k")
			if err := os.WriteFile(path, payload, 0o600); err != nil {
				t.Fatalf("write case: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, exe,
				"-test.run", "^TestHTJ2KHostileInputs$", "-test.timeout", "30s")
			cmd.Env = append(os.Environ(), htj2kHostileWorkerEnv+"="+path)
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

// runHTJ2KHostileWorker decodes the single hostile HTJ2K codestream named by the
// worker env var. It exits 0 if the decoder rejected the input with a typed
// *jpeg2000Error, and non-zero otherwise (including a clean decode, which would mean
// the bounds checks did not hold).
func runHTJ2KHostileWorker() {
	path := os.Getenv(htj2kHostileWorkerEnv)
	payload, err := os.ReadFile(path)
	if err != nil {
		os.Stderr.WriteString("worker: read payload: " + err.Error() + "\n")
		os.Exit(2)
	}

	c := openjpegCodec{ts: HTJ2KLossless, canEncode: false}
	_, decErr := c.Decode(payload, htj2kHostileGeom)
	if decErr == nil {
		os.Stderr.WriteString("worker: hostile input decoded without error\n")
		os.Exit(1)
	}
	var je *jpeg2000Error
	if !errors.As(decErr, &je) || !errors.Is(decErr, ErrJPEG2000) {
		os.Stderr.WriteString("worker: error is not a typed jpeg2000Error: " + decErr.Error() + "\n")
		os.Exit(1)
	}
	os.Exit(0)
}

// htj2kHostileCorpus builds the malformed-input cases, deriving the truncated and
// dimension-tampered cases from a real HTJ2K codestream so they are structurally
// close to a valid one.
func htj2kHostileCorpus(t *testing.T) map[string][]byte {
	t.Helper()
	valid := validHTJ2KCodestream(t)

	return map[string][]byte{
		"empty":            {},
		"single_byte":      {0xFF},
		"random_garbage":   garbage(4096),
		"zero_filled":      make([]byte, 8192),
		"truncated_header": clone(valid[:16]),
		"truncated_mid":    clone(valid[:len(valid)/2]),
		"truncated_tail":   clone(valid[:len(valid)-4]),
		"oversized_dims":   htj2kOversizedDimensions(valid),
		"zero_dims":        htj2kZeroDimensions(valid),
	}
}

// validHTJ2KCodestream returns the first frame's raw HTJ2K codestream from the
// vendored lossless fixture, the seed for the structurally-close hostile cases.
func validHTJ2KCodestream(t *testing.T) []byte {
	t.Helper()
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "HTJ2KLossless_08_RGB.dcm"))
	if err != nil {
		t.Fatalf("ReadPixelData: %v", err)
	}
	encoded, err := pd.frameEncodedBytes()
	if err != nil {
		t.Fatalf("frameEncodedBytes: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatal("HTJ2K fixture yielded no frames")
	}
	return clone(encoded[0])
}

// htj2kOversizedDimensions sets Xsiz/Ysiz larger than the worker geometry's cap so
// the dimension check rejects the header before any pixel buffer is allocated. The
// SIZ Xsiz field sits at the same offset in an HTJ2K codestream as in classic J2K.
func htj2kOversizedDimensions(valid []byte) []byte {
	b := clone(valid)
	if len(b) >= sizXsizOffset+8 {
		binary.BigEndian.PutUint32(b[sizXsizOffset:], 50000)
		binary.BigEndian.PutUint32(b[sizXsizOffset+4:], 50000)
	}
	return b
}

// htj2kZeroDimensions sets Xsiz/Ysiz to zero to probe the non-positive-dimension guard.
func htj2kZeroDimensions(valid []byte) []byte {
	b := clone(valid)
	if len(b) >= sizXsizOffset+8 {
		binary.BigEndian.PutUint32(b[sizXsizOffset:], 0)
		binary.BigEndian.PutUint32(b[sizXsizOffset+4:], 0)
	}
	return b
}
