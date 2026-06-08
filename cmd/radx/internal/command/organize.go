package command

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/logging"
)

// OrganizeCmd lays out a flat or mixed directory of DICOM files into a Study/Series/SOP hierarchy
// by reading each file's UIDs. UIDs are validated and sanitised before they become path segments
// (the prototype's sanitizeUID was a no-op, RADX-018), and files are written with exclusive-create
// semantics unless --overwrite is set, so a duplicate or malformed UID cannot silently truncate an
// existing file. --dry-run performs no I/O (docs/reference/cli.md organize).
type OrganizeCmd struct {
	Dir string `arg:"" name:"dir" help:"Source directory."`

	OutputDir string `name:"output-dir" required:"" help:"Destination root."`
	Recursive bool   `short:"R" name:"recursive" default:"true" negatable:"" help:"Descend into the source."`
	Move      bool   `name:"move" help:"Move instead of copy."`
	DryRun    bool   `name:"dry-run" help:"Report the planned layout without touching files."`
	Overwrite bool   `name:"overwrite" help:"Allow overwriting an existing destination file."`
}

// organizeAction is one planned (or performed) relocation: the source file and the destination it
// is placed at under the Study/Series/SOP layout. It names paths, never patient values.
type organizeAction struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}

// Run resolves the source files, computes each file's destination from its validated UIDs, and
// either reports the plan (--dry-run) or performs the relocation. A file whose UID is malformed or
// whose relocation fails is reported and drives a non-zero exit; organize never reports success on
// a file it could not place.
func (c *OrganizeCmd) Run(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return &exitcode.UsageErr{Message: "organize does not support --format csv; use human or json"}
	}

	info, err := os.Stat(c.Dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return &exitcode.UsageErr{Message: fmt.Sprintf("%s is not a directory", c.Dir)}
	}

	files, err := dicomSourceFiles(c.Dir, c.Recursive)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return &exitcode.UsageErr{Message: "no DICOM files found under the source directory"}
	}

	if !c.DryRun {
		if err := ensureDir(c.OutputDir); err != nil {
			return err
		}
	}

	log := logging.FromContext(rc.Ctx)
	var firstErr error
	for _, path := range files {
		action, actErr := c.organizeOne(path)
		log.Debug("organize: processed file",
			zap.String("source", path),
			zap.String("status", action.Status),
		)
		if emitErr := c.emit(rc, action); emitErr != nil {
			return emitErr
		}
		if actErr != nil && firstErr == nil {
			firstErr = actErr
		}
	}
	return firstErr
}

// organizeOne reads one file's UIDs, computes its destination under the Study/Series/SOP layout,
// and either plans (dry-run) or performs the relocation. A malformed UID is a parse failure; a
// relocation I/O fault is a file error. Both are reported per-file and propagated.
func (c *OrganizeCmd) organizeOne(path string) (organizeAction, error) {
	f, err := dicom.ReadFile(path)
	if err != nil {
		return organizeAction{Source: path, Status: "failure", Error: structuralError(err)}, err
	}

	study, err := safeUIDSegment(f.DataSet, dicom.TagStudyInstanceUID, "StudyInstanceUID")
	if err != nil {
		return organizeAction{Source: path, Status: "failure", Error: structuralError(err)}, err
	}
	series, err := safeUIDSegment(f.DataSet, dicom.TagSeriesInstanceUID, "SeriesInstanceUID")
	if err != nil {
		return organizeAction{Source: path, Status: "failure", Error: structuralError(err)}, err
	}
	instance, err := safeUIDSegment(f.DataSet, dicom.TagSOPInstanceUID, "SOPInstanceUID")
	if err != nil {
		return organizeAction{Source: path, Status: "failure", Error: structuralError(err)}, err
	}

	dir := filepath.Join(c.OutputDir, study, series)
	dest := filepath.Join(dir, instance+".dcm")

	if c.DryRun {
		return organizeAction{Source: path, Destination: dest, Status: "planned"}, nil
	}

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return organizeAction{Source: path, Destination: dest, Status: "failure", Error: structuralError(err)}, err
	}
	if err := c.place(path, dest); err != nil {
		return organizeAction{Source: path, Destination: dest, Status: "failure", Error: structuralError(err)}, err
	}
	status := "copied"
	if c.Move {
		status = "moved"
	}
	return organizeAction{Source: path, Destination: dest, Status: status}, nil
}

// place relocates src to dest with exclusive-create semantics (unless --overwrite is set), so a
// duplicate or malformed destination cannot silently truncate an existing file (RADX-018). It
// moves with a rename when --move is set and the rename succeeds, falling back to a copy across
// filesystems; otherwise it copies and leaves the source intact.
func (c *OrganizeCmd) place(src, dest string) error {
	if !c.Overwrite {
		if _, err := os.Stat(dest); err == nil {
			return fmt.Errorf("destination %s already exists (pass --overwrite to replace)", dest)
		}
	}
	if c.Move {
		if err := os.Rename(src, dest); err == nil {
			return nil
		}
		// A cross-device rename fails; fall back to copy-then-remove so a move still works across
		// filesystems rather than failing the whole run.
		if err := copyFileExclusive(src, dest, c.Overwrite); err != nil {
			return err
		}
		return os.Remove(src)
	}
	return copyFileExclusive(src, dest, c.Overwrite)
}

// copyFileExclusive copies src to dest, creating dest exclusively (O_EXCL) unless overwrite is set,
// so a concurrent or duplicate write cannot truncate an existing file. A failure removes the
// partial destination so a copy never leaves a truncated file readable as complete.
func copyFileExclusive(src, dest string, overwrite bool) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if overwrite {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	out, err := os.OpenFile(dest, flags, 0o640)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(dest)
		return err
	}
	return out.Close()
}

// dicomSourceFiles lists the *.dcm files to organise. With recursive set it descends the whole
// tree; otherwise it lists only the top-level *.dcm files of dir.
func dicomSourceFiles(dir string, recursive bool) ([]string, error) {
	if recursive {
		return dicomFilesUnder(dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) == ".dcm" {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files, nil
}

// emit renders one organize action in the resolved format.
func (c *OrganizeCmd) emit(rc *RunContext, a organizeAction) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSONLine(a)
	}
	if a.Status == "failure" {
		_, err := fmt.Fprintf(rc.Out.Machine, "%s: FAILED — %s\n", a.Source, a.Error)
		return err
	}
	_, err := fmt.Fprintf(rc.Out.Machine, "%s: %s -> %s\n", a.Source, a.Status, a.Destination)
	return err
}
