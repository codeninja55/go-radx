package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/logging"
)

// TranscodeCmd rewrites a Part 10 file's transfer syntax (dcmtk's dcmconv/dcmcrle/dcmdrle family)
// through the library's dataset-level seam: Read -> NewPixelData -> Transcode -> SetPixelData ->
// Write. A pixel-less object passes through with a meta rewrite only, and an object already in the
// target syntax is written as stored, so nothing is silently altered. A target this build cannot
// encode fails closed per file with the library's typed error (ErrEncodeUnsupported /
// ErrCodecUnavailable, exit 3) and writes no output for that file; transcode never reports success
// on an unwritten object (the modify fail-closed posture).
type TranscodeCmd struct {
	Paths []string `arg:"" name:"path" help:"DICOM files to transcode."`

	To        string `name:"to" required:"" help:"Target transfer syntax: a UID or a dicom keyword (e.g. ExplicitVRLittleEndian, RLELossless). Pure-Go builds encode uncompressed and RLE targets only; JPEG-family sources decode only in a CGo codec build (dicom_libjpeg, dicom_charls, dicom_openjpeg tags)."`
	OutputDir string `name:"output-dir" help:"Write transcoded files here (required unless --in-place)."`
	InPlace   bool   `short:"i" name:"in-place" help:"Overwrite the originals in place."`
	Recursive bool   `short:"R" name:"recursive" help:"Descend into directories for *.dcm files."`
}

// transcodeResult is the canonical per-file machine shape: the source and written paths, the
// outcome status, the source and target transfer syntax UIDs, and (on failure) a structural error.
// It names structure only, never patient values.
type transcodeResult struct {
	File   string `json:"file"`
	Output string `json:"output,omitempty"`
	Status string `json:"status"`
	From   string `json:"from,omitempty"`
	To     string `json:"to,omitempty"`
	Error  string `json:"error,omitempty"`
}

// transferSyntaxKeywords maps the dicom package's exported transfer syntax identifiers to their
// UIDs, so --to accepts the keyword form a Go caller of the library already knows.
var transferSyntaxKeywords = map[string]dicom.TransferSyntax{
	"ImplicitVRLittleEndian":         dicom.ImplicitVRLittleEndian,
	"ExplicitVRLittleEndian":         dicom.ExplicitVRLittleEndian,
	"DeflatedExplicitVRLittleEndian": dicom.DeflatedExplicitVRLittleEndian,
	"ExplicitVRBigEndian":            dicom.ExplicitVRBigEndian,
	"RLELossless":                    dicom.RLELossless,
	"JPEGBaseline8Bit":               dicom.JPEGBaseline8Bit,
	"JPEGExtended12Bit":              dicom.JPEGExtended12Bit,
	"JPEGLossless":                   dicom.JPEGLossless,
	"JPEGLosslessSV1":                dicom.JPEGLosslessSV1,
	"JPEGLSLossless":                 dicom.JPEGLSLossless,
	"JPEGLSNearLossless":             dicom.JPEGLSNearLossless,
	"JPEG2000Lossless":               dicom.JPEG2000Lossless,
	"JPEG2000":                       dicom.JPEG2000,
	"HTJ2KLossless":                  dicom.HTJ2KLossless,
	"HTJ2KLosslessRPCL":              dicom.HTJ2KLosslessRPCL,
	"HTJ2K":                          dicom.HTJ2K,
}

// resolveTransferSyntax resolves a flag value to a transfer syntax: the dicom package's keyword
// form first, then any structurally valid UID. It returns ok == false for anything else, which the
// caller reports as a usage error.
func resolveTransferSyntax(raw string) (dicom.TransferSyntax, bool) {
	if ts, ok := transferSyntaxKeywords[raw]; ok {
		return ts, true
	}
	if err := dicom.UID(raw).Validate(); err == nil {
		return dicom.TransferSyntax(raw), true
	}
	return "", false
}

// Run validates the target and destination, resolves the files, and transcodes each, writing the
// result or failing closed per file. Processing continues past a failed file (the modify
// convention) and the first failure's typed error drives the exit class.
func (c *TranscodeCmd) Run(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return &exitcode.UsageErr{Message: "transcode does not support --format csv; use human or json"}
	}
	if c.InPlace == (c.OutputDir != "") {
		return &exitcode.UsageErr{Message: "exactly one of --output-dir or --in-place is required"}
	}
	target, ok := resolveTransferSyntax(c.To)
	if !ok {
		return &exitcode.UsageErr{Message: fmt.Sprintf("--to %q is neither a valid transfer syntax UID nor a known keyword", c.To)}
	}
	if !c.InPlace {
		if err := ensureDir(c.OutputDir); err != nil {
			return err
		}
	}

	files, err := resolveDICOMPaths(c.Paths, c.Recursive)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return &exitcode.UsageErr{Message: "no DICOM files to transcode"}
	}
	// Preflight every destination BEFORE any write: a batch where two inputs flatten to one output,
	// where two inputs collide only by basename case on a case-insensitive filesystem, or where one
	// input's output would clobber a DIFFERENT input's source, is rejected up front so the first
	// rename never destroys a file a later input still needs (the no-silent-data-loss rule).
	if !c.InPlace {
		if err := c.preflightDestinations(files); err != nil {
			return err
		}
	}

	log := logging.FromContext(rc.Ctx)
	var firstErr error
	for _, path := range files {
		result, transcodeErr := c.transcodeOne(path, target)
		// Diagnostics name structure only — paths and transfer syntaxes, never element values.
		log.Debug("transcode: processed file",
			zap.String("file", path),
			zap.String("to", string(target)),
			zap.String("status", result.Status),
		)
		if emitErr := c.emit(rc, result); emitErr != nil {
			return emitErr
		}
		if transcodeErr != nil && firstErr == nil {
			firstErr = transcodeErr
		}
	}
	return firstErr
}

// transcodeOne reads one file and produces its target-syntax output. When the file is already in
// the target transfer syntax, the source bytes are copied UNCHANGED (a raw byte copy, or a no-op
// under --in-place): re-encoding a same-syntax object would rebuild the File Meta and break any
// checksum or digital signature over the original bytes, so a passthrough must be byte-identical.
// A real transfer-syntax change re-encodes: a pixel-bearing object through the codec seam, a
// pixel-less object as a meta rewrite. Failures are fail-closed: no output file is left for a
// failed input (the temp-and-rename writer / atomic copy), and the returned result carries the
// failure status AND the underlying typed error so exitcode.Classify routes the run to its real
// class. Batch destination collisions are rejected up front by preflightDestinations.
func (c *TranscodeCmd) transcodeOne(path string, target dicom.TransferSyntax) (transcodeResult, error) {
	dest := c.destinationFor(path)

	f, err := dicom.ReadFile(path)
	if err != nil {
		return transcodeResult{File: path, Status: "failure", Error: structuralError(err)}, err
	}
	source := f.Meta.TransferSyntaxUID
	result := transcodeResult{File: path, From: string(source), To: string(target)}

	// Same-syntax passthrough: preserve the exact bytes rather than re-encoding. The batch preflight
	// has already rejected any destination that would clobber another input, so overwrite is safe here.
	if source == target {
		if !c.InPlace {
			if err := copyFileAtomic(path, dest, true); err != nil {
				result.Status, result.Error = "failure", structuralError(err)
				return result, err
			}
		}
		result.Output, result.Status = dest, "success"
		return result, nil
	}

	if _, hasPixels := f.DataSet.Get(dicom.TagPixelData); hasPixels {
		pd, err := dicom.NewPixelData(f.DataSet, source)
		if err != nil {
			result.Status, result.Error = "failure", structuralError(err)
			return result, err
		}
		out, err := dicom.Transcode(pd, target)
		if err != nil {
			result.Status, result.Error = "failure", structuralError(err)
			return result, err
		}
		if err := f.SetPixelData(out); err != nil {
			result.Status, result.Error = "failure", structuralError(err)
			return result, err
		}
	}

	if err := writeDataSetAtomic(dest, f.DataSet, target); err != nil {
		result.Status, result.Error = "failure", structuralError(err)
		return result, err
	}

	result.Output, result.Status = dest, "success"
	return result, nil
}

// preflightDestinations validates the whole batch's --output-dir destinations before any file is
// written, so the first rename never destroys a file a later input still needs. It rejects, as a
// usage error: two inputs whose outputs land on the same path, a case-only basename collision on a
// case-insensitive filesystem (disambiguated with os.SameFile so the identical physical input
// listed twice is not a false collision), and an input whose output would overwrite a DIFFERENT
// input's source. Paths are canonicalised with filepath.Abs so the comparison is independent of
// spelling.
func (c *TranscodeCmd) preflightDestinations(files []string) error {
	sources := make(map[string]string, len(files)) // abs source path -> input as given
	for _, path := range files {
		sources[absPath(path)] = path
	}
	byDest := make(map[string]string, len(files)) // abs dest -> input
	byFold := make(map[string]string, len(files)) // case-folded abs dest -> input
	for _, path := range files {
		dest := c.destinationFor(path)
		absDest := absPath(dest)
		if prior, clash := byDest[absDest]; clash {
			return &exitcode.UsageErr{Message: fmt.Sprintf(
				"output collision: %q and %q both write to %q; rename an input or use distinct output names", prior, path, dest)}
		}
		fold := strings.ToLower(absDest)
		if prior, clash := byFold[fold]; clash && !sameFilePath(prior, path) {
			return &exitcode.UsageErr{Message: fmt.Sprintf(
				"output collision (case-insensitive filesystem): %q and %q map to the same file %q", prior, path, dest)}
		}
		if other, clash := sources[absDest]; clash && absDest != absPath(path) {
			return &exitcode.UsageErr{Message: fmt.Sprintf(
				"output collision: transcoding %q would overwrite the input %q; use a distinct --output-dir", path, other)}
		}
		byDest[absDest] = path
		byFold[fold] = path
	}
	return nil
}

// absPath canonicalises a path, falling back to the original when Abs fails, so preflight
// comparisons are independent of how a path was spelled.
func absPath(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}

// sameFilePath reports whether two paths name the same existing file (os.SameFile), so a case-only
// destination collision that is actually the identical physical input is not flagged.
func sameFilePath(a, b string) bool {
	ai, err := os.Stat(a)
	if err != nil {
		return false
	}
	bi, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(ai, bi)
}

// destinationFor resolves where a transcoded file is written: in place over the original, or under
// --output-dir keyed by the source's base name.
func (c *TranscodeCmd) destinationFor(path string) string {
	if c.InPlace {
		return path
	}
	return filepath.Join(c.OutputDir, filepath.Base(path))
}

// emit renders one file's transcode result in the resolved format. Under JSON it writes one
// compact object per file (JSON Lines), consistent with the other per-item commands.
func (c *TranscodeCmd) emit(rc *RunContext, r transcodeResult) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSONLine(r)
	}
	if r.Status == "success" {
		_, err := fmt.Fprintf(rc.Out.Machine, "%s -> %s: %s -> %s\n",
			r.File, r.Output, dicom.TransferSyntax(r.From).Name(), dicom.TransferSyntax(r.To).Name())
		return err
	}
	_, err := fmt.Fprintf(rc.Out.Machine, "%s: FAILED — %s\n", r.File, r.Error)
	return err
}
