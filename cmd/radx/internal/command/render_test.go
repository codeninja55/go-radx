package command

import (
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
)

// writeMonochromeDICOM writes a synthetic MONOCHROME2 Part 10 file with native pixel data and
// frames frames, so a test can drive the render pipeline. Pixels ramp 0..255 across each frame, so
// the min/max auto-stretch has a known expectation (first pixel 0 -> black, last -> white).
// Identifiers are synthetic, never PHI.
func writeMonochromeDICOM(t *testing.T, dir, sopInstanceUID string, frames int) string {
	t.Helper()
	const rows, cols = 4, 4
	perFrame := rows * cols
	pixels := make([]byte, perFrame*frames)
	for i := range pixels {
		pixels[i] = byte((i % perFrame) * 17) // 0..255 ramp per frame
	}

	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.7") // Secondary Capture
	ds.SetString(dicom.TagSOPInstanceUID, sopInstanceUID)
	ds.SetString(dicom.TagStudyInstanceUID, "1.2.3.4.6.1")
	ds.SetString(dicom.TagSeriesInstanceUID, "1.2.3.4.6.2")
	ds.SetString(dicom.TagPhotometricInterpretation, "MONOCHROME2")
	ds.Set(dicom.Element{Tag: dicom.TagRows, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, rows)})
	ds.Set(dicom.Element{Tag: dicom.TagColumns, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, cols)})
	ds.Set(dicom.Element{Tag: dicom.TagBitsAllocated, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 8)})
	ds.Set(dicom.Element{Tag: dicom.TagBitsStored, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 8)})
	ds.Set(dicom.Element{Tag: dicom.TagHighBit, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 7)})
	ds.Set(dicom.Element{Tag: dicom.TagSamplesPerPixel, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, 1)})
	if frames > 1 {
		ds.SetString(dicom.TagNumberOfFrames, itoa(frames))
	}
	ds.Set(dicom.Element{Tag: dicom.TagPixelData, VR: dicom.VROB, Value: dicom.NewBytes(dicom.VROB, pixels)})

	path := filepath.Join(dir, strings.ReplaceAll(sopInstanceUID, ".", "_")+".dcm")
	if err := ds.WriteFile(path, dicom.ExplicitVRLittleEndian); err != nil {
		t.Fatalf("write monochrome DICOM: %v", err)
	}
	return path
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// TestRenderSingleFramePNG renders a MONOCHROME2 image's first frame to PNG and checks the output is
// a decodable image of the right size whose stretched corners are black and white.
func TestRenderSingleFramePNG(t *testing.T) {
	src := writeMonochromeDICOM(t, t.TempDir(), "1.2.3.4.6.10", 1)
	outDir := filepath.Join(t.TempDir(), "out")

	stdout, stderr, code := runRadx(t, "render", "--format", "json", "--output-dir", outDir, src)
	if code != exitcode.Success {
		t.Fatalf("render exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}
	var r renderResult
	if err := json.Unmarshal([]byte(stdout), &r); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	if r.Status != "success" || r.Frames != 1 {
		t.Fatalf("status=%q frames=%d, want success/1 (error=%q)", r.Status, r.Frames, r.Error)
	}

	out := filepath.Join(outDir, "1_2_3_4_6_10.png")
	if len(r.Outputs) != 1 || r.Outputs[0] != out {
		t.Fatalf("outputs = %v, want [%s]", r.Outputs, out)
	}
	fh, err := os.Open(out)
	if err != nil {
		t.Fatalf("open rendered png: %v", err)
	}
	defer fh.Close()
	img, err := png.Decode(fh)
	if err != nil {
		t.Fatalf("decode rendered png: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 4 || b.Dy() != 4 {
		t.Fatalf("image bounds = %v, want 4x4", b)
	}
	// Pixel 0 is the ramp minimum (stretched to black); the last pixel is the maximum (white).
	if r0, _, _, _ := img.At(0, 0).RGBA(); r0 != 0 {
		t.Errorf("top-left luma = %d, want 0 (black)", r0>>8)
	}
	if r15, _, _, _ := img.At(3, 3).RGBA(); r15>>8 != 255 {
		t.Errorf("bottom-right luma = %d, want 255 (white)", r15>>8)
	}
}

// TestRenderAllFramesPPM renders every frame of a 3-frame image to PPM with index-suffixed names.
func TestRenderAllFramesPPM(t *testing.T) {
	src := writeMonochromeDICOM(t, t.TempDir(), "1.2.3.4.6.11", 3)
	outDir := filepath.Join(t.TempDir(), "out")

	stdout, stderr, code := runRadx(t, "render", "--format", "json",
		"--image-format", "ppm", "--all-frames", "--output-dir", outDir, src)
	if code != exitcode.Success {
		t.Fatalf("render exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}
	var r renderResult
	if err := json.Unmarshal([]byte(stdout), &r); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	if r.Frames != 3 || len(r.Outputs) != 3 {
		t.Fatalf("frames=%d outputs=%d, want 3/3", r.Frames, len(r.Outputs))
	}
	for i := 0; i < 3; i++ {
		want := filepath.Join(outDir, "1_2_3_4_6_11-00"+itoa(i)+".ppm")
		if r.Outputs[i] != want {
			t.Errorf("output[%d] = %q, want %q", i, r.Outputs[i], want)
		}
		data, err := os.ReadFile(want)
		if err != nil {
			t.Fatalf("read ppm %d: %v", i, err)
		}
		if !strings.HasPrefix(string(data), "P6\n") {
			t.Errorf("output[%d] is not a P6 PPM (prefix %q)", i, string(data[:min(3, len(data))]))
		}
	}
}

// TestRenderNoPixelDataFailsClosed renders a dataset with no pixel data: it fails closed (exit 3)
// and writes nothing.
func TestRenderNoPixelDataFailsClosed(t *testing.T) {
	dir := t.TempDir()
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.7")
	ds.SetString(dicom.TagSOPInstanceUID, "1.2.3.4.6.12")
	src := filepath.Join(dir, "nopixels.dcm")
	if err := ds.WriteFile(src, dicom.ExplicitVRLittleEndian); err != nil {
		t.Fatalf("write no-pixel DICOM: %v", err)
	}
	outDir := filepath.Join(t.TempDir(), "out")

	_, _, code := runRadx(t, "render", "--output-dir", outDir, src)
	if code == exitcode.Success {
		t.Fatal("render of a pixel-less object should fail, got success")
	}
	entries, _ := os.ReadDir(outDir)
	if len(entries) != 0 {
		t.Errorf("render wrote %d output(s) for a failed input, want 0", len(entries))
	}
}

// TestRenderFrameOutOfRangeFailsClosed selects a frame past the end of a single-frame image.
func TestRenderFrameOutOfRangeFailsClosed(t *testing.T) {
	src := writeMonochromeDICOM(t, t.TempDir(), "1.2.3.4.6.13", 1)
	outDir := filepath.Join(t.TempDir(), "out")

	_, _, code := runRadx(t, "render", "--frame", "5", "--output-dir", outDir, src)
	if code == exitcode.Success {
		t.Fatal("render of an out-of-range frame should fail, got success")
	}
	entries, _ := os.ReadDir(outDir)
	if len(entries) != 0 {
		t.Errorf("render wrote %d output(s) for an out-of-range frame, want 0", len(entries))
	}
}
