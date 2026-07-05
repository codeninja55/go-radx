package command

import (
	"fmt"
	"io"
	"os"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicomweb"
	"github.com/codeninja55/go-radx/logging"
)

// ComposeCmd builds a Part 10 file from a PS3.18 DICOM JSON document (dcmtk's json2dcm): the
// dicomweb codec decodes the Annex F shape (failing closed on an unknown VR, a malformed
// InlineBinary, or a truncated document), and the meta glue derives the File Meta from the
// dataset — SOP/Study/Series Instance UIDs are minted fresh ONLY where absent and honoured where
// present, and a dataset with no SOP Class UID is refused (no meta can be derived, exit 3). The
// input is the PS3.18 DICOM-JSON model, not radx dump's tag-keyed shape. The output transfer
// syntax defaults to Explicit VR LE; PS3.18 JSON carries native binary values, so only
// uncompressed syntaxes can be honoured (use radx transcode afterwards to compress).
type ComposeCmd struct {
	JSON   string `arg:"" name:"json" help:"PS3.18 DICOM JSON file (\"-\" = stdin)."`
	Output string `arg:"" name:"output" help:"Part 10 output path."`

	TransferSyntax string `name:"transfer-syntax" default:"ExplicitVRLittleEndian" help:"Output transfer syntax (UID or dicom keyword); uncompressed syntaxes only."`
	Overwrite      bool   `name:"overwrite" help:"Allow overwriting an existing output file."`
}

// composeResult is the canonical machine shape: the written path, the outcome status, the
// transfer syntax, the SOP Instance UID the file carries, and which instance UIDs were minted.
// It names identifiers and structure only, never patient values.
type composeResult struct {
	Output         string   `json:"output"`
	Status         string   `json:"status"`
	TransferSyntax string   `json:"transfer_syntax"`
	SOPInstanceUID string   `json:"sop_instance_uid,omitempty"`
	MintedUIDs     []string `json:"minted_uids,omitempty"`
}

// composeUIDTags are the instance-identity UIDs the meta glue mints when absent, in the order
// they are reported. SOP Class UID is deliberately not here: a SOP Class can never be invented.
var composeUIDTags = []struct {
	tag     dicom.Tag
	keyword string
}{
	{dicom.TagSOPInstanceUID, "SOPInstanceUID"},
	{dicom.TagStudyInstanceUID, "StudyInstanceUID"},
	{dicom.TagSeriesInstanceUID, "SeriesInstanceUID"},
}

// Run reads the PS3.18 JSON, decodes it fail-closed, applies the meta glue, and writes the Part
// 10 file atomically (successful-or-nothing; a failed compose leaves no output file).
func (c *ComposeCmd) Run(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return &exitcode.UsageErr{Message: "compose does not support --format csv; use human or json"}
	}
	target, ok := resolveTransferSyntax(c.TransferSyntax)
	if !ok {
		return &exitcode.UsageErr{Message: fmt.Sprintf("--transfer-syntax %q is neither a valid transfer syntax UID nor a known keyword", c.TransferSyntax)}
	}
	if target.IsEncapsulated() {
		return &exitcode.UsageErr{Message: fmt.Sprintf(
			"--transfer-syntax %s (%s) cannot be honoured: PS3.18 JSON carries native binary values, so only uncompressed syntaxes are supported (transcode the composed file to compress)",
			target.Name(), c.TransferSyntax)}
	}

	// Refuse to clobber an existing output unless --overwrite is set, so a compose never silently
	// replaces a Part 10 file (mirrors organize). Lstat catches a symlink target too.
	if !c.Overwrite {
		if _, statErr := os.Lstat(c.Output); statErr == nil {
			return &exitcode.UsageErr{Message: fmt.Sprintf("output %q already exists (pass --overwrite to replace)", c.Output)}
		}
	}

	data, err := c.readInput(rc)
	if err != nil {
		return err
	}
	ds, err := dicomweb.UnmarshalJSON(data)
	if err != nil {
		return err
	}

	// The File Meta derives from the dataset's SOP Class UID; without one there is nothing
	// conformant to write, so compose refuses rather than inventing a class (fail-closed).
	if sopClass, ok := ds.GetString(dicom.TagSOPClassUID); !ok || sopClass == "" {
		return &dicom.ValueError{Tag: dicom.TagSOPClassUID, VR: dicom.VRUI,
			Msg: "the JSON dataset carries no SOP Class UID; File Meta cannot be derived"}
	}

	log := logging.FromContext(rc.Ctx)
	// A UID the JSON carries is honoured verbatim (compose does not rewrite present identities), but
	// a non-conformant one is warned about so the operator is told the file carries an off-spec UID
	// rather than it passing unnoticed. The diagnostic names the keyword only, never a patient value.
	warnNonconformantUIDs(log, ds)

	minted, err := mintMissingUIDs(ds)
	if err != nil {
		return err
	}

	if err := writeDataSetAtomic(c.Output, ds, target); err != nil {
		return err
	}

	sopInstance, _ := ds.GetString(dicom.TagSOPInstanceUID)
	return c.emit(rc, composeResult{
		Output:         c.Output,
		Status:         "success",
		TransferSyntax: string(target),
		SOPInstanceUID: sopInstance,
		MintedUIDs:     minted,
	})
}

// readInput reads the JSON document from the named file, or from stdin when the path is "-"
// (the convention the stdin-reading commands share). A file that cannot be read keeps its
// file-I/O class (exit 5).
func (c *ComposeCmd) readInput(rc *RunContext) ([]byte, error) {
	if c.JSON == "-" {
		return io.ReadAll(rc.Stdin)
	}
	return os.ReadFile(c.JSON) // #nosec G304 -- the operator-named input file
}

// warnNonconformantUIDs logs a warning for each present SOP Class or instance-identity UID that is
// not a conformant DICOM UID. compose honours a present UID verbatim (it does not rewrite the
// caller's identities), so this is the operator's only signal that the JSON supplied an off-spec
// value. It names the keyword only, never a patient value.
func warnNonconformantUIDs(log *zap.Logger, ds *dicom.DataSet) {
	checks := append([]struct {
		tag     dicom.Tag
		keyword string
	}{{dicom.TagSOPClassUID, "SOPClassUID"}}, composeUIDTags...)
	for _, u := range checks {
		v, ok := ds.GetString(u.tag)
		if !ok || v == "" {
			continue
		}
		if err := dicom.UID(v).Validate(); err != nil {
			log.Warn("compose: present UID is not conformant; honouring it verbatim",
				zap.String("keyword", u.keyword))
		}
	}
}

// mintMissingUIDs mints a fresh, validated UID for each absent instance-identity tag, honouring
// every UID the document carries, and returns the keywords of the tags it minted. A minted UID
// is validated before it is written, so compose never produces a non-conformant identifier.
func mintMissingUIDs(ds *dicom.DataSet) ([]string, error) {
	gen := dicom.NewRandomUIDGenerator()
	var minted []string
	for _, u := range composeUIDTags {
		if v, ok := ds.GetString(u.tag); ok && v != "" {
			continue
		}
		fresh := gen.Generate()
		if err := fresh.Validate(); err != nil {
			return nil, fmt.Errorf("generated UID for %s is not conformant: %w", u.keyword, err)
		}
		ds.SetString(u.tag, string(fresh))
		minted = append(minted, u.keyword)
	}
	return minted, nil
}

// emit renders the compose result in the resolved format.
func (c *ComposeCmd) emit(rc *RunContext, r composeResult) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSON(r)
	}
	_, err := fmt.Fprintf(rc.Out.Machine, "%s: composed (%s)\n", r.Output, dicom.TransferSyntax(r.TransferSyntax).Name())
	return err
}
