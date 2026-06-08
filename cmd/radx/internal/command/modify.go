package command

import (
	"bufio"
	"fmt"
	"os"
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
	Insert    []string `short:"I" name:"insert" sep:"none" help:"Insert or update a tag ((GGGG,EEEE)=value or keyword=value)."`
	Delete    []string `short:"D" name:"delete" sep:"none" help:"Delete a tag ((GGGG,EEEE) or keyword)."`
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

	// One old->new UID map per re-keyed grouping tag, shared across every file in this run. The first
	// file to carry a given old Study (or Series) UID mints one new UID for it; every later file with
	// the same old UID reuses that new UID, so a multi-file study or series is re-keyed as ONE object,
	// not split into N unrelated ones. SOP Instance UIDs are never mapped — each instance gets a
	// unique new UID — so the reference graph (study groups its series, series groups its instances)
	// survives the batch (the preserve-the-reference-graph contract).
	uidMaps := newUIDRemap()

	written := make(map[string]string, len(files))
	var firstErr error
	for _, path := range files {
		result, modifyErr := c.modifyOne(path, plan, gen, written, uidMaps)
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
// dataset is fully re-encoded to a sibling temp file and only renamed over the destination once it
// is complete on disk). The written map records each destination already produced in this run so a
// later input that would clobber an earlier output is refused rather than silently overwriting it.
func (c *ModifyCmd) modifyOne(path string, plan modifyPlan, gen *dicom.UIDGenerator, written map[string]string, uidMaps *uidRemap) (modifyResult, error) {
	dest := c.destinationFor(path)
	if prior, clash := written[dest]; clash {
		// Two inputs flatten to the same --output-dir path. Overwriting the earlier output would
		// destroy a file already reported successful, so fail closed: a collision is an error, never
		// a silent data loss (the no-silent-data-loss rule). The structural detail names paths only.
		err := &exitcode.UsageErr{Message: fmt.Sprintf(
			"output collision: %q and %q both write to %q; rename an input or use distinct output names", prior, path, dest)}
		return modifyResult{File: path, Status: "failure", Error: structuralError(err)}, err
	}

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
		if err := applyUIDRegeneration(f, plan.regenUIDs, gen, uidMaps); err != nil {
			return modifyResult{File: path, Status: "failure", Error: structuralError(err)}, err
		}
		edits += len(plan.regenUIDs)
		regenerated = true
	}

	ts := f.Meta.TransferSyntaxUID
	if err := writeDataSetAtomic(dest, f.DataSet, ts); err != nil {
		// A write failure (an unwritable target, a malformed dataset) leaves no output: the encode
		// goes to a sibling temp file that is removed on any error, so the destination — the source
		// itself under --in-place — is never truncated or left partially overwritten. Report the
		// failure and the error; write nothing.
		return modifyResult{File: path, Status: "failure", Error: structuralError(err)}, err
	}

	written[dest] = path
	return modifyResult{
		File:    path,
		Output:  dest,
		Status:  "success",
		Edits:   edits,
		NewUIDs: regenerated,
	}, nil
}

// writeDataSetAtomic re-encodes ds in ts to dest with successful-or-nothing semantics: it writes a
// Part 10 file to a temp file in dest's directory, fsyncs and closes it, and only renames it over
// dest once the whole file is durable. On any failure it removes the temp file and leaves dest (the
// source file itself under --in-place) byte-for-byte untouched. dicom.WriteFile truncates its target
// with os.Create before writing, so a failed write there would partially overwrite the original — a
// clinical-data-safety defect; the temp-and-rename here makes the replacement atomic on the same
// filesystem (a rename within one directory does not cross device boundaries). The File Meta is
// derived from the dataset's SOP Class/Instance UIDs, mirroring dicom.DataSet.WriteFile.
func writeDataSetAtomic(dest string, ds *dicom.DataSet, ts dicom.TransferSyntax) error {
	f, err := fileFromDataSet(ds, ts)
	if err != nil {
		return err
	}

	dir := filepath.Dir(dest)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(dest)+".radx-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Any path out of this function that has not renamed the temp into place removes it, so a failure
	// never leaves a stray partial file beside the original.
	committed := false
	defer func() {
		if !committed {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()

	bw := bufio.NewWriter(tmp)
	if err := dicom.Write(bw, f); err != nil {
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

// fileFromDataSet wraps ds in a Part 10 File whose meta is derived from the dataset's SOP Class
// (0008,0016) and SOP Instance (0008,0018) UIDs, the same derivation dicom.DataSet.WriteFile
// performs, so the atomic writer produces an identical header. A dataset missing either UID, or an
// unwritable transfer syntax, is rejected before any temp file is created.
func fileFromDataSet(ds *dicom.DataSet, ts dicom.TransferSyntax) (*dicom.File, error) {
	sopClass, ok := ds.GetString(dicom.TagSOPClassUID)
	if !ok || sopClass == "" {
		return nil, &dicom.ValueError{Tag: dicom.TagSOPClassUID, VR: dicom.VRUI, Msg: "dataset has no SOP Class UID to derive file meta from"}
	}
	sopInstance, ok := ds.GetString(dicom.TagSOPInstanceUID)
	if !ok || sopInstance == "" {
		return nil, &dicom.ValueError{Tag: dicom.TagSOPInstanceUID, VR: dicom.VRUI, Msg: "dataset has no SOP Instance UID to derive file meta from"}
	}
	return &dicom.File{
		Meta: &dicom.FileMeta{
			MediaStorageSOPClassUID:    dicom.SOPClassUID(sopClass),
			MediaStorageSOPInstanceUID: dicom.SOPInstanceUID(sopInstance),
			TransferSyntaxUID:          ts,
		},
		DataSet: ds,
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

// uidRemap is the run-level old->new UID map for the grouping tags (Study, Series). It maps a tag
// to a per-tag table of original UID -> minted UID, so every file in one invocation that shares an
// old Study or Series UID is re-keyed to the SAME new UID, preserving the study/series grouping
// across the batch. It is NOT used for SOP Instance UID, which must stay unique per instance.
type uidRemap struct {
	byTag map[dicom.Tag]map[string]dicom.UID
}

// newUIDRemap returns an empty run-level UID remap.
func newUIDRemap() *uidRemap {
	return &uidRemap{byTag: make(map[dicom.Tag]map[string]dicom.UID)}
}

// mintFor returns the new UID this run has assigned to old under tag, minting and recording one the
// first time old is seen so every later file sharing that old UID maps to the same new UID. A blank
// old UID is not pooled: a file missing a grouping UID gets a fresh UID of its own rather than being
// merged with every other file that also lacks one. The minted UID is validated before it is
// recorded, so a remap never yields a non-conformant identifier.
func (r *uidRemap) mintFor(tag dicom.Tag, old string, gen *dicom.UIDGenerator) (dicom.UID, error) {
	if old != "" {
		if table, ok := r.byTag[tag]; ok {
			if mapped, seen := table[old]; seen {
				return mapped, nil
			}
		}
	}
	fresh := gen.Generate()
	if err := fresh.Validate(); err != nil {
		return "", fmt.Errorf("generated UID for %s is not conformant: %w", tag, err)
	}
	if old != "" {
		table := r.byTag[tag]
		if table == nil {
			table = make(map[string]dicom.UID)
			r.byTag[tag] = table
		}
		table[old] = fresh
	}
	return fresh, nil
}

// applyUIDRegeneration mints a fresh, validated UID for each requested UID tag, writes it to the
// dataset, and keeps the file's reference graph intact in two ways. Within one file, a new SOP
// Instance UID is mirrored into the File Meta MediaStorageSOPInstanceUID (0002,0003) so the Part 10
// header and the dataset agree. Across the run, the grouping tags (Study, Series) are re-keyed
// through the shared run-level map so files that came in under one Study or Series UID stay grouped
// under one new UID, rather than each file becoming an unrelated object. The SOP Instance UID is
// always minted fresh per instance and never pooled. A minted UID is validated before it is written,
// so a regeneration never produces a non-conformant identifier (RADX-002: the prototype generated
// UIDs and logged them without ever writing them).
func applyUIDRegeneration(f *dicom.File, tags []dicom.Tag, gen *dicom.UIDGenerator, uidMaps *uidRemap) error {
	for _, tag := range tags {
		var fresh dicom.UID
		if tag == dicom.TagSOPInstanceUID {
			// Each instance is a distinct object: never share a SOP Instance UID across the batch.
			fresh = gen.Generate()
			if err := fresh.Validate(); err != nil {
				return fmt.Errorf("generated UID for %s is not conformant: %w", tag, err)
			}
		} else {
			old, _ := f.DataSet.GetString(tag)
			mapped, err := uidMaps.mintFor(tag, old, gen)
			if err != nil {
				return err
			}
			fresh = mapped
		}
		f.DataSet.SetString(tag, string(fresh))
		if tag == dicom.TagSOPInstanceUID && f.Meta != nil {
			f.Meta.MediaStorageSOPInstanceUID = dicom.SOPInstanceUID(fresh)
		}
	}
	return nil
}

// emit renders one file's modify result in the resolved format. Under JSON it writes one compact
// object per file (JSON Lines), so a batch over multiple files is a parseable stream consistent with
// the other per-item commands (store, qido) rather than concatenated indented documents.
func (c *ModifyCmd) emit(rc *RunContext, r modifyResult) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSONLine(r)
	}
	if r.Status == "success" {
		_, err := fmt.Fprintf(rc.Out.Machine, "%s -> %s: %d edits applied\n", r.File, r.Output, r.Edits)
		return err
	}
	_, err := fmt.Fprintf(rc.Out.Machine, "%s: FAILED — %s\n", r.File, r.Error)
	return err
}
