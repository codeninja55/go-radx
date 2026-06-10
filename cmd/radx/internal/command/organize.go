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
		if err := copyFileAtomic(src, dest, c.Overwrite); err != nil {
			return err
		}
		return os.Remove(src)
	}
	return copyFileAtomic(src, dest, c.Overwrite)
}

// copyFileAtomic copies src to dest with successful-or-nothing semantics in both modes, mirroring
// writeDataSetAtomic for the raw-byte copy case. It copies into a sibling temp file, fsyncs and
// closes it, and commits the fully written temp into place only after the copy and close both succeed
// (a rename or link within one directory does not cross device boundaries). With overwrite the commit
// is an atomic rename that replaces any existing dest; without overwrite the commit is a hard link
// that fails if dest already exists, preserving exclusive-create semantics atomically (no
// probe-then-write gap a racing writer could exploit). On any failure the temp file is removed and an
// existing destination is left untouched, so a delayed writeback failure (a Close error on a full
// disk or NFS) leaves NO file at dest. The earlier non-overwrite path wrote straight into an O_EXCL
// handle and returned out.Close() last, so a Close failure left a truncated .dcm behind that a later
// run treated as an existing destination — violating successful-or-nothing for clinical data.
func copyFileAtomic(src, dest string, overwrite bool) error {
	in, err := os.Open(src) // #nosec G304 -- copying the user-specified source file is the CLI's purpose
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	dir := filepath.Dir(dest)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(dest)+".radx-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Any path out of this function that has not committed the temp removes it, so a failure never
	// leaves a stray partial file beside the destination. The exclusive route also removes the temp
	// after a successful link, since the linked dest is the durable copy and the temp is now redundant.
	committed := false
	defer func() {
		if !committed {
			_ = tmp.Close()
		}
		_ = os.Remove(tmpName)
	}()

	if _, err := io.Copy(tmp, in); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	committed = true
	if overwrite {
		return os.Rename(tmpName, dest)
	}
	return os.Link(tmpName, dest)
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
