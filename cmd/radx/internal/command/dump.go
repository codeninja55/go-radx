package command

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/dicom"
)

// DumpCmd inspects DICOM file contents, printing each element with its tag, keyword, VR, and a
// rendered value (dcmtk's dcmdump). A malformed or truncated file exits 3 (truncation is a
// failure, never a graceful end); a missing or unreadable file exits 5. The listing names
// structure — tag, keyword, VR, length — and renders only synthetic, non-PHI fixtures in the
// test suite (docs/reference/cli.md dump).
type DumpCmd struct {
	Paths []string `arg:"" name:"path" help:"DICOM files to inspect."`

	IgnoreErrors bool `name:"ignore-errors" help:"Exit 0 even if some inputs failed (exploratory)."`
}

// dumpFile is the canonical machine shape for one inspected file: the path, an outcome status,
// and either the tag-keyed element list (on success) or a structural error string (on
// failure). The error names structure only, never patient values.
type dumpFile struct {
	File     string                 `json:"file"`
	Status   string                 `json:"status"`
	Error    string                 `json:"error,omitempty"`
	Elements map[string]dumpElement `json:"elements,omitempty"`
}

// dumpElement is one element in the machine shape: its keyword, VR, value count, and a
// rendered value. Pixel data is summarised by length, never rendered as bytes.
type dumpElement struct {
	Tag     string `json:"tag"`
	Keyword string `json:"keyword"`
	VR      string `json:"vr"`
	Value   string `json:"value"`
}

// Run inspects each path, emits the per-file machine shape, and decides the exit. A parse
// failure on any file is recorded in that file's machine output and makes the command return
// the first such error (exit 3) unless --ignore-errors is set; a clean run returns nil.
func (c *DumpCmd) Run(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return c.runCSV(rc)
	}

	var firstErr error
	for _, path := range c.Paths {
		df, err := c.inspect(path)
		if emitErr := c.emit(rc, df); emitErr != nil {
			return emitErr
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if c.IgnoreErrors {
		return nil
	}
	return firstErr
}

// inspect reads one file and builds its machine shape. On a read or parse failure it returns a
// populated dumpFile with a "failure" status (so the per-file outcome is visible) AND the
// underlying error (so the runner classifies the exit), keeping the honest-failure contract: a
// failed parse is never reported as a clean dump.
func (c *DumpCmd) inspect(path string) (dumpFile, error) {
	f, err := dicom.ReadFile(path)
	if err != nil {
		return dumpFile{File: path, Status: "failure", Error: structuralError(err)}, err
	}
	elements := make(map[string]dumpElement, f.DataSet.Len())
	for e := range f.DataSet.All() {
		key := tagKey(e.Tag)
		elements[key] = dumpElement{
			Tag:     e.Tag.String(),
			Keyword: keywordOf(e.Tag),
			VR:      e.VR.String(),
			Value:   renderValue(e),
		}
	}
	return dumpFile{File: path, Status: "success", Elements: elements}, nil
}

// emit renders one file's result in the resolved format: an indented human listing or the
// canonical JSON object. CSV is handled separately (runCSV) because it streams rows.
func (c *DumpCmd) emit(rc *RunContext, df dumpFile) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSON(df)
	}
	return c.emitHuman(rc, df)
}

// emitHuman writes the indented, human-readable element listing for one file.
func (c *DumpCmd) emitHuman(rc *RunContext, df dumpFile) error {
	w := rc.Out.Machine
	if df.Status != "success" {
		_, err := fmt.Fprintf(w, "# %s: FAILED — %s\n", df.File, df.Error)
		return err
	}
	if _, err := fmt.Fprintf(w, "# %s\n", df.File); err != nil {
		return err
	}
	for _, key := range sortedKeys(df.Elements) {
		e := df.Elements[key]
		if _, err := fmt.Fprintf(w, "%s %-16s %s  %s\n", e.Tag, e.Keyword, e.VR, e.Value); err != nil {
			return err
		}
	}
	return nil
}

// runCSV streams one row per element across all files as RFC 4180 CSV. A parse failure is one
// row marking the file failed, and still drives a non-zero exit unless --ignore-errors is set.
func (c *DumpCmd) runCSV(rc *RunContext) error {
	cw := rc.Out.CSVWriter()
	if err := cw.Write([]string{"file", "status", "tag", "keyword", "vr", "value"}); err != nil {
		return err
	}
	var firstErr error
	for _, path := range c.Paths {
		df, err := c.inspect(path)
		if err != nil {
			if writeErr := cw.Write([]string{path, "failure", "", "", "", structuralError(err)}); writeErr != nil {
				return writeErr
			}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for _, key := range sortedKeys(df.Elements) {
			e := df.Elements[key]
			if writeErr := cw.Write([]string{path, "success", e.Tag, e.Keyword, e.VR, e.Value}); writeErr != nil {
				return writeErr
			}
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return err
	}
	if c.IgnoreErrors {
		return nil
	}
	return firstErr
}

// tagKey renders a tag as the lowercase "gggg,eeee" key used in the JSON map, matching the
// PS3.18 DICOM-JSON convention so a consumer can address an element by tag.
func tagKey(t dicom.Tag) string {
	return fmt.Sprintf("%04X,%04X", t.Group(), t.Element())
}

// keywordOf resolves a tag's dictionary keyword, falling back to a fixed placeholder for an
// unknown (private or unregistered) tag. It never returns a patient value.
func keywordOf(t dicom.Tag) string {
	if info, ok := dicom.Lookup(t); ok && info.Keyword != "" {
		return info.Keyword
	}
	return "Unknown"
}

// sortedKeys returns the element keys in ascending order so the human and CSV output is
// deterministic (the JSON map is unordered by definition, but the rendered listings are
// stable for golden comparison).
func sortedKeys(m map[string]dumpElement) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// structuralError renders an error as a structural diagnostic string for the machine shape. It
// is the error's own message, which the library guarantees names structure (tags, offsets,
// constraints) and never a patient value (PRD §9.1).
func structuralError(err error) string { return err.Error() }

// renderValue produces a display string for an element's value, type-switching on the
// exported concrete Value implementations. Sequences, pixel data, and other binary VRs are
// summarised structurally (by VR or byte length) rather than dumped as bytes, so the listing
// names structure, not raw content (docs/reference/cli.md: "the listing names structure, not
// patient values"). A nil value renders as an empty string.
func renderValue(e dicom.Element) string {
	switch v := e.Value.(type) {
	case *dicom.Strings:
		return strings.Join(v.Strings(), `\`)
	case *dicom.Ints:
		parts := make([]string, 0, len(v.Ints()))
		for _, n := range v.Ints() {
			parts = append(parts, strconv.FormatInt(n, 10))
		}
		return strings.Join(parts, `\`)
	case *dicom.Floats:
		parts := make([]string, 0, len(v.Floats()))
		for _, f := range v.Floats() {
			parts = append(parts, strconv.FormatFloat(f, 'g', -1, 64))
		}
		return strings.Join(parts, `\`)
	case *dicom.Decimals:
		parts := make([]string, 0, len(v.Decimals()))
		for _, d := range v.Decimals() {
			parts = append(parts, d.String())
		}
		return strings.Join(parts, `\`)
	case *dicom.Tags:
		parts := make([]string, 0, len(v.Tags()))
		for _, t := range v.Tags() {
			parts = append(parts, t.String())
		}
		return strings.Join(parts, `\`)
	case *dicom.Bytes:
		return fmt.Sprintf("<%d bytes>", len(v.Bytes()))
	case nil:
		return ""
	default:
		// Sequences (SQ), pixel data, and any other Value type the reader produces are
		// summarised by VR rather than rendered as bytes: the listing names structure, never
		// raw or patient content.
		return fmt.Sprintf("<%s>", e.VR)
	}
}
