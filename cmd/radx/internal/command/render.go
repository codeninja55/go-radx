package command

import (
	"bufio"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/logging"
)

// RenderCmd renders a DICOM image's pixel frames to 8-bit consumer images (dcmtk's dcm2pnm/dcm2img
// family) through the library's presentation pipeline: Read -> NewPixelData -> Frames -> RenderFrame
// -> PNG/PPM. The value-of-interest mapping (VOI LUT table, window, or padding-aware auto-stretch)
// and the photometric handling (MONOCHROME1/2, PALETTE COLOR, RGB, YBR) live in dicom.RenderFrame;
// this command only iterates frames and writes files. An object with no pixel data, an unsupported
// photometric interpretation, or (for an encapsulated syntax) a codec this build cannot decode fails
// closed per file with the library's typed error (exit 3) and writes no output for that file.
type RenderCmd struct {
	Paths []string `arg:"" name:"path" help:"DICOM image files to render."`

	OutputDir string `name:"output-dir" required:"" help:"Write rendered images here."`
	Format    string `name:"image-format" enum:"png,ppm" default:"png" help:"Output image format: png or ppm."`
	Frame     int    `name:"frame" default:"0" help:"Render this 0-based frame (ignored with --all-frames)."`
	AllFrames bool   `name:"all-frames" help:"Render every frame, suffixing the output name with the frame index."`
	Recursive bool   `short:"R" name:"recursive" help:"Descend into directories for *.dcm files."`
}

// renderResult is the canonical per-file machine shape: the source path, the outcome status, the
// written image paths, the frame count rendered, and (on failure) a structural error. It names
// structure only, never patient values.
type renderResult struct {
	File    string   `json:"file"`
	Status  string   `json:"status"`
	Outputs []string `json:"outputs,omitempty"`
	Frames  int      `json:"frames,omitempty"`
	Error   string   `json:"error,omitempty"`
}

// Run validates the destination and options, resolves the files, and renders each, failing closed
// per file. Processing continues past a failed file (the transcode/modify convention) and the first
// failure's typed error drives the exit class.
func (c *RenderCmd) Run(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return &exitcode.UsageErr{Message: "render does not support --format csv; use human or json"}
	}
	if c.Frame < 0 {
		return &exitcode.UsageErr{Message: "--frame must be zero or greater"}
	}
	if err := ensureDir(c.OutputDir); err != nil {
		return err
	}

	files, err := resolveDICOMPaths(c.Paths, c.Recursive)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return &exitcode.UsageErr{Message: "no DICOM files to render"}
	}

	log := logging.FromContext(rc.Ctx)
	var firstErr error
	for _, path := range files {
		result, renderErr := c.renderOne(path)
		// Diagnostics name structure only — paths and frame counts, never element values.
		log.Debug("render: processed file",
			zap.String("file", path),
			zap.String("status", result.Status),
			zap.Int("frames", result.Frames),
		)
		if emitErr := c.emit(rc, result); emitErr != nil {
			return emitErr
		}
		if renderErr != nil && firstErr == nil {
			firstErr = renderErr
		}
	}
	return firstErr
}

// renderOne reads one file and writes its rendered frame(s). Failures are fail-closed: a file that
// errors before or during any frame leaves no partial output for that frame, and the returned result
// carries the failure status AND the underlying typed error so exitcode.Classify routes the run.
func (c *RenderCmd) renderOne(path string) (renderResult, error) {
	f, err := dicom.ReadFile(path)
	if err != nil {
		return renderResult{File: path, Status: "failure", Error: structuralError(err)}, err
	}
	if _, ok := f.DataSet.Get(dicom.TagPixelData); !ok {
		e := &exitcode.UsageErr{Message: fmt.Sprintf("%s carries no pixel data to render", path)}
		return renderResult{File: path, Status: "failure", Error: "no pixel data"}, e
	}

	pd, err := dicom.NewPixelData(f.DataSet, f.Meta.TransferSyntaxUID)
	if err != nil {
		return renderResult{File: path, Status: "failure", Error: structuralError(err)}, err
	}

	result := renderResult{File: path}
	frameIdx := 0
	for frame, ferr := range pd.Frames() {
		if ferr != nil {
			result.Status, result.Error = "failure", structuralError(ferr)
			return result, ferr
		}
		render := c.AllFrames || frameIdx == c.Frame
		if render {
			img, rerr := dicom.RenderFrame(frame, f.DataSet, pd.Geometry)
			if rerr != nil {
				result.Status, result.Error = "failure", structuralError(rerr)
				return result, rerr
			}
			dest := c.destinationFor(path, frameIdx)
			if werr := c.writeImageAtomic(dest, img); werr != nil {
				result.Status, result.Error = "failure", structuralError(werr)
				return result, werr
			}
			result.Outputs = append(result.Outputs, dest)
			result.Frames++
		}
		frameIdx++
		if !c.AllFrames && frameIdx > c.Frame {
			break
		}
	}

	if result.Frames == 0 {
		// The requested single frame was past the end of the multi-frame stream.
		e := &exitcode.UsageErr{Message: fmt.Sprintf("%s has %d frame(s); --frame %d is out of range", path, frameIdx, c.Frame)}
		result.Status, result.Error = "failure", "frame index out of range"
		return result, e
	}
	result.Status = "success"
	return result, nil
}

// destinationFor resolves the output path for one rendered frame under --output-dir: the source's
// base name (extension stripped), a frame-index suffix when every frame is rendered, and the image
// format's extension.
func (c *RenderCmd) destinationFor(path string, frameIdx int) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if c.AllFrames {
		base = fmt.Sprintf("%s-%03d", base, frameIdx)
	}
	return filepath.Join(c.OutputDir, base+"."+c.Format)
}

// writeImageAtomic encodes img in the selected format to a temp file and renames it into place, so a
// failed encode never leaves a partial image beside a prior render.
func (c *RenderCmd) writeImageAtomic(dest string, img image.Image) error {
	dir := filepath.Dir(dest)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(dest)+".radx-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()

	bw := bufio.NewWriter(tmp)
	if err := encodeImage(bw, img, c.Format); err != nil {
		return err
	}
	if err := bw.Flush(); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, dest); err != nil {
		return err
	}
	committed = true
	return nil
}

// encodeImage writes img in the named format. PNG uses the standard library; PPM uses the dicom
// package's P6 writer (no standard-library encoder exists for it).
func encodeImage(w io.Writer, img image.Image, format string) error {
	switch format {
	case "ppm":
		return dicom.EncodePPM(w, img)
	default:
		return png.Encode(w, img)
	}
}

// emit renders one file's result in the resolved format: one compact JSON object per file (JSON
// Lines) under --format json, a human summary otherwise.
func (c *RenderCmd) emit(rc *RunContext, r renderResult) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSONLine(r)
	}
	if r.Status == "success" {
		_, err := fmt.Fprintf(rc.Out.Machine, "%s: rendered %d frame(s) -> %s\n",
			r.File, r.Frames, strings.Join(r.Outputs, ", "))
		return err
	}
	_, err := fmt.Fprintf(rc.Out.Machine, "%s: FAILED — %s\n", r.File, r.Error)
	return err
}
