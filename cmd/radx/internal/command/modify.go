package command

import (
	"fmt"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/logging"
)

// ModifyCmd edits elements in DICOM files: insert or update a tag, delete a tag, or regenerate
// Study/Series/SOP Instance UIDs (dcmtk's dcmodify). This command really mutates the dataset — the
// single most important correction in the CLI. The prototype's modify logged "Would insert" and
// wrote unchanged files while reporting success (RADX-001/002); here each edit is applied to the
// in-memory dataset, the file is re-encoded, and the written file is what the flags asked for. If
// any requested edit cannot be applied, modify returns an error, exits 1, and writes no output for
// that file (fail-closed). Inserted values are PHI and are never logged at default verbosity.
type ModifyCmd struct {
	Paths []string `arg:"" name:"path" help:"DICOM files to edit."`

	OutputDir string   `name:"output-dir" help:"Write modified files here (required unless --in-place)."`
	InPlace   bool     `short:"i" name:"in-place" help:"Overwrite the originals in place."`
	Insert    []string `short:"I" name:"insert" help:"Insert or update a tag ((GGGG,EEEE)=value or keyword=value)."`
	Delete    []string `short:"D" name:"delete" help:"Delete a tag ((GGGG,EEEE) or keyword)."`
	Recursive bool     `short:"R" name:"recursive" help:"Descend into directories for *.dcm files."`

	RegenerateStudyUID    bool `name:"regenerate-study-uid" help:"New Study Instance UID (0020,000D)."`
	RegenerateSeriesUID   bool `name:"regenerate-series-uid" help:"New Series Instance UID (0020,000E)."`
	RegenerateInstanceUID bool `name:"regenerate-instance-uid" help:"New SOP Instance UID (0008,0018)."`
	RegenerateAllUIDs     bool `name:"regenerate-all-uids" help:"Regenerate all three, preserving the reference graph."`
}

// modifyResult is the canonical per-file machine shape: the source and written paths, the outcome
// status, the count of edits applied, and (on failure) a structural error. It names structure
// only, never the inserted patient values (RADX-007).
type modifyResult struct {
	File    string `json:"file"`
	Output  string `json:"output,omitempty"`
	Status  string `json:"status"`
	Edits   int    `json:"edits"`
	Error   string `json:"error,omitempty"`
	NewUIDs bool   `json:"regenerated_uids,omitempty"`
}

// modifyPlan is the validated, parsed set of edits to apply to every file, resolved once from the
// flags so an invalid tag or value is a usage error before any file is touched.
type modifyPlan struct {
	inserts   []insertEdit
	deletes   []dicom.Tag
	regenUIDs []dicom.Tag
}

// insertEdit is one validated insert/update: a resolved tag and the value to set.
type insertEdit struct {
	tag   dicom.Tag
	value string
}

// Run validates the edit plan, resolves the files, and applies the plan to each, writing the
// modified file or failing closed. A file that cannot be re-keyed produces no output and the
// command exits non-zero; modify never reports success on an unwritten edit.
func (c *ModifyCmd) Run(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return &exitcode.UsageErr{Message: "modify does not support --format csv; use human or json"}
	}
	if c.InPlace == (c.OutputDir != "") {
		return &exitcode.UsageErr{Message: "exactly one of --output-dir or --in-place is required"}
	}

	plan, err := c.buildPlan()
	if err != nil {
		return err
	}
	if len(plan.inserts) == 0 && len(plan.deletes) == 0 && len(plan.regenUIDs) == 0 {
		return &exitcode.UsageErr{Message: "no edits requested (use --insert, --delete, or a --regenerate-* flag)"}
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
		return &exitcode.UsageErr{Message: "no DICOM files to modify"}
	}

	log := logging.FromContext(rc.Ctx)
	gen := dicom.NewRandomUIDGenerator()

	var firstErr error
	for _, path := range files {
		result, modifyErr := c.modifyOne(path, plan, gen)
		// Diagnostics name structure only — the edit COUNT and tag identities, never the values.
		log.Debug("modify: processed file",
			zap.String("file", path),
			zap.Int("edits", result.Edits),
			zap.String("status", result.Status),
		)
		if emitErr := c.emit(rc, result); emitErr != nil {
			return emitErr
		}
		if modifyErr != nil && firstErr == nil {
			firstErr = modifyErr
		}
	}
	return firstErr
}

// modifyOne reads one file, applies the plan to its dataset, and writes the result. Read failures
// and write failures are fail-closed: the returned result carries a failure status AND the
// underlying error, and on a write failure NO partial output file is left for that input (the
// dataset is fully re-encoded to the destination only after every edit applied cleanly).
func (c *ModifyCmd) modifyOne(path string, plan modifyPlan, gen *dicom.UIDGenerator) (modifyResult, error) {
	f, err := dicom.ReadFile(path)
	if err != nil {
		return modifyResult{File: path, Status: "failure", Error: structuralError(err)}, err
	}

	edits := 0
	for _, del := range plan.deletes {
		if _, present := f.DataSet.Get(del); present {
			f.DataSet.Delete(del)
			edits++
		}
	}
	for _, ins := range plan.inserts {
		f.DataSet.SetString(ins.tag, ins.value)
		edits++
	}
	regenerated := false
	if len(plan.regenUIDs) > 0 {
		if err := applyUIDRegeneration(f, plan.regenUIDs, gen); err != nil {
			return modifyResult{File: path, Status: "failure", Error: structuralError(err)}, err
		}
		edits += len(plan.regenUIDs)
		regenerated = true
	}

	dest := c.destinationFor(path)
	ts := f.Meta.TransferSyntaxUID
	if err := f.DataSet.WriteFile(dest, ts); err != nil {
		// A write failure (an unwritable target, a malformed dataset) leaves no output: WriteFile
		// writes to a temp file and renames, so a failure removes the temp rather than leaving a
		// truncated file readable as complete. Report the failure and the error; write nothing.
		return modifyResult{File: path, Status: "failure", Error: structuralError(err)}, err
	}

	return modifyResult{
		File:    path,
		Output:  dest,
		Status:  "success",
		Edits:   edits,
		NewUIDs: regenerated,
	}, nil
}

// destinationFor resolves where a modified file is written: in place over the original, or under
// --output-dir keyed by the source's base name.
func (c *ModifyCmd) destinationFor(path string) string {
	if c.InPlace {
		return path
	}
	return filepath.Join(c.OutputDir, filepath.Base(path))
}

// buildPlan parses and validates the edit flags once, so an invalid tag or insert value is a usage
// error before any file is read. The regenerate flags resolve to the set of UID tags to mint anew;
// --regenerate-all-uids is the union of the three.
func (c *ModifyCmd) buildPlan() (modifyPlan, error) {
	var plan modifyPlan
	for _, raw := range c.Insert {
		key, value, ok := splitKeyValue(raw)
		if !ok {
			return modifyPlan{}, &exitcode.UsageErr{Message: fmt.Sprintf("invalid --insert %q (use tag=value)", raw)}
		}
		t, resolvable := parseTagSpec(key)
		if !resolvable {
			return modifyPlan{}, &exitcode.UsageErr{Message: fmt.Sprintf("invalid --insert tag %q (use (GGGG,EEEE), GGGGEEEE, or a keyword)", key)}
		}
		plan.inserts = append(plan.inserts, insertEdit{tag: t, value: value})
	}
	for _, raw := range c.Delete {
		t, resolvable := parseTagSpec(raw)
		if !resolvable {
			return modifyPlan{}, &exitcode.UsageErr{Message: fmt.Sprintf("invalid --delete tag %q (use (GGGG,EEEE), GGGGEEEE, or a keyword)", raw)}
		}
		plan.deletes = append(plan.deletes, t)
	}

	if c.RegenerateAllUIDs || c.RegenerateStudyUID {
		plan.regenUIDs = append(plan.regenUIDs, dicom.TagStudyInstanceUID)
	}
	if c.RegenerateAllUIDs || c.RegenerateSeriesUID {
		plan.regenUIDs = append(plan.regenUIDs, dicom.TagSeriesInstanceUID)
	}
	if c.RegenerateAllUIDs || c.RegenerateInstanceUID {
		plan.regenUIDs = append(plan.regenUIDs, dicom.TagSOPInstanceUID)
	}
	return plan, nil
}

// applyUIDRegeneration mints a fresh, validated UID for each requested UID tag, writes it to the
// dataset, and keeps the file's reference graph intact: a new SOP Instance UID is also written to
// the File Meta MediaStorageSOPInstanceUID (0002,0003) so the Part 10 header and the dataset agree.
// A minted UID is validated before it is written, so a regeneration never produces a non-conformant
// identifier (RADX-002: the prototype generated UIDs and logged them without ever writing them).
func applyUIDRegeneration(f *dicom.File, tags []dicom.Tag, gen *dicom.UIDGenerator) error {
	for _, tag := range tags {
		fresh := gen.Generate()
		if err := fresh.Validate(); err != nil {
			return fmt.Errorf("generated UID for %s is not conformant: %w", tag, err)
		}
		f.DataSet.SetString(tag, string(fresh))
		if tag == dicom.TagSOPInstanceUID && f.Meta != nil {
			f.Meta.MediaStorageSOPInstanceUID = dicom.SOPInstanceUID(fresh)
		}
	}
	return nil
}

// emit renders one file's modify result in the resolved format.
func (c *ModifyCmd) emit(rc *RunContext, r modifyResult) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSON(r)
	}
	if r.Status == "success" {
		_, err := fmt.Fprintf(rc.Out.Machine, "%s -> %s: %d edits applied\n", r.File, r.Output, r.Edits)
		return err
	}
	_, err := fmt.Fprintf(rc.Out.Machine, "%s: FAILED — %s\n", r.File, r.Error)
	return err
}
