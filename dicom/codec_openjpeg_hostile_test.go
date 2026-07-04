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

// hostileWorkerEnv names the environment variable that puts the test binary into
// single-decode worker mode. When set, TestMain-less worker entry runs one hostile
// decode and exits, so the parent can bound it with a timeout (DCM-015: a hang must
// fail the test, never wedge CI).
const hostileWorkerEnv = "GORADX_OPJ_HOSTILE_WORKER"

// hostileWorkerGeom is the geometry the worker decodes against. It is small and
// fixed so an oversized-dimension codestream is rejected by the dimension cap.
var hostileWorkerGeom = PixelGeometry{
	Rows: 64, Columns: 64, SamplesPerPixel: 1, BitsAllocated: 16, BitsStored: 16,
}

// TestJPEG2000HostileInputs feeds a corpus of malformed, truncated, and
// oversized-dimension J2K codestreams to the decoder. Each case must fail with a
// typed *jpeg2000Error (wrapping ErrJPEG2000) and must not crash or hang. Every
// decode runs in a subprocess with a timeout, so a hang is a killed process and a
// test failure rather than a wedged run (PRD §9.3, §11.2; DCM-015).
func TestJPEG2000HostileInputs(t *testing.T) {
	if os.Getenv(hostileWorkerEnv) != "" {
		runHostileWorker() // never returns
		return
	}
	skipHostileUnderASAN(t)

	corpus := hostileCorpus(t)
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

			// A generous timeout: a correct decode of any of these rejects in well
			// under a second. Exceeding it means a hang, which must fail the test.
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, exe,
				"-test.run", "^TestJPEG2000HostileInputs$", "-test.timeout", "30s")
			cmd.Env = append(os.Environ(), hostileWorkerEnv+"="+path)
			out, err := cmd.CombinedOutput()

			if ctx.Err() == context.DeadlineExceeded {
				t.Fatalf("decode hung on hostile input (killed after timeout); output:\n%s", out)
			}
			// The worker exits 0 when the decoder rejected the input with a typed
			// error (the intended, safe outcome). A non-zero exit means a panic or an
			// unexpected success, both of which are failures.
			if err != nil {
				t.Fatalf("worker exited with error %v; output:\n%s", err, out)
			}
		})
	}
}

// runHostileWorker decodes the single hostile codestream named by the worker env
// var. It exits 0 if the decoder rejected the input with a typed *jpeg2000Error (or
// accepted a degenerate-but-valid input without crashing), and non-zero otherwise.
// A crash (panic / SIGSEGV from the C side) terminates the process abnormally, which
// the parent observes as a non-zero exit.
func runHostileWorker() {
	path := os.Getenv(hostileWorkerEnv)
	payload, err := os.ReadFile(path)
	if err != nil {
		os.Stderr.WriteString("worker: read payload: " + err.Error() + "\n")
		os.Exit(2)
	}

	c := openjpegCodec{ts: JPEG2000Lossless, canEncode: true}
	_, decErr := c.Decode(payload, hostileWorkerGeom)
	if decErr == nil {
		// A hostile codestream must not decode cleanly into a frame for this
		// geometry; if one does, the bounds checks did not hold.
		os.Stderr.WriteString("worker: hostile input decoded without error\n")
		os.Exit(1)
	}
	_, ok := errors.AsType[*jpeg2000Error](decErr)
	if !ok || !errors.Is(decErr, ErrJPEG2000) {
		os.Stderr.WriteString("worker: error is not a typed jpeg2000Error: " + decErr.Error() + "\n")
		os.Exit(1)
	}
	os.Exit(0)
}

// hostileCorpus builds the malformed-input cases. It derives the truncated and
// dimension-tampered cases from a real frame of the liver fixture so they are
// structurally close to a valid codestream (a tougher test than pure noise).
func hostileCorpus(t *testing.T) map[string][]byte {
	t.Helper()
	valid := validLiverCodestream(t)

	corpus := map[string][]byte{
		"empty":            {},
		"single_byte":      {0xFF},
		"soc_only":         {0xFF, 0x4F},
		"random_garbage":   garbage(4096),
		"zero_filled":      make([]byte, 8192),
		"truncated_header": clone(valid[:16]),
		"truncated_mid":    clone(valid[:len(valid)/2]),
		"truncated_tail":   clone(valid[:len(valid)-4]),
		"oversized_dims":   oversizedDimensions(valid),
		"zero_dims":        zeroDimensions(valid),
		"giant_dims":       giantDimensions(valid),
	}
	return corpus
}

// validLiverCodestream returns the first frame's raw J2K codestream from the liver
// fixture, the seed for the structurally-close hostile cases.
func validLiverCodestream(t *testing.T) []byte {
	t.Helper()
	pd, err := ReadPixelData(filepath.Join("..", "testdata", "dicom", "liver_j2k.dcm"))
	if err != nil {
		t.Fatalf("ReadPixelData: %v", err)
	}
	encoded, err := pd.frameEncodedBytes()
	if err != nil {
		t.Fatalf("frameEncodedBytes: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatal("liver fixture yielded no frames")
	}
	return clone(encoded[0])
}

// sizOffset is the byte offset of the SIZ marker payload's Xsiz field in a raw J2K
// codestream: SOC marker (2) + SIZ marker (2) + Lsiz (2) + Rsiz (2) = 8, then Xsiz.
const sizXsizOffset = 8

// oversizedDimensions sets Xsiz/Ysiz larger than the worker geometry's cap so the
// dimension check rejects the header before any pixel buffer is allocated.
func oversizedDimensions(valid []byte) []byte {
	b := clone(valid)
	if len(b) >= sizXsizOffset+8 {
		binary.BigEndian.PutUint32(b[sizXsizOffset:], 50000)   // Xsiz
		binary.BigEndian.PutUint32(b[sizXsizOffset+4:], 50000) // Ysiz
	}
	return b
}

// giantDimensions sets Xsiz/Ysiz to near-uint32-max to probe size-overflow handling.
func giantDimensions(valid []byte) []byte {
	b := clone(valid)
	if len(b) >= sizXsizOffset+8 {
		binary.BigEndian.PutUint32(b[sizXsizOffset:], 0xFFFFFFF0)
		binary.BigEndian.PutUint32(b[sizXsizOffset+4:], 0xFFFFFFF0)
	}
	return b
}

// zeroDimensions sets Xsiz/Ysiz to zero to probe the non-positive-dimension guard.
func zeroDimensions(valid []byte) []byte {
	b := clone(valid)
	if len(b) >= sizXsizOffset+8 {
		binary.BigEndian.PutUint32(b[sizXsizOffset:], 0)
		binary.BigEndian.PutUint32(b[sizXsizOffset+4:], 0)
	}
	return b
}
