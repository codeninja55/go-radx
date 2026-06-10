package command

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
)

// redactedMarker is the fixed value rendered in place of a PHI-sensitive element's
// real value when the user opts in with --redact. The element's structure — tag,
// keyword, VR — is always shown; only the value is masked.
const redactedMarker = "[redacted]"

// pixelDataTag is PixelData (7FE0,0010). Its value is read and rendered only when the
// caller passes --process-pixel-data; otherwise it is summarised structurally without
// touching the (potentially large) pixel bytes.
var pixelDataTag = dicom.NewTag(0x7FE0, 0x0010)

// DumpCmd inspects DICOM file contents, printing each element with its tag, keyword, VR, and a
// rendered value (dcmtk's dcmdump). A malformed or truncated file exits 3 (truncation is a
// failure, never a graceful end); a missing or unreadable file exits 5.
//
// Element values are shown by default: a dump is an explicit, authorized local inspection of a
// file the user already holds (the same posture as dcmtk's dcmdump). The no-PHI rule targets
// ambient logging, not a command the user deliberately ran on a local file, so it does not
// apply here. --redact opts in to masking the values of PHI-sensitive elements (the PS3.15
// confidentiality attributes) to a fixed marker; the structure — tag, keyword, VR, length — is
// always shown either way. Pixel data is summarised structurally unless --process-pixel-data is
// set. The test suite renders only synthetic, non-PHI fixtures (docs/reference/cli.md dump).
type DumpCmd struct {
	Paths []string `arg:"" name:"path" help:"DICOM files to inspect."`

	Recursive        bool     `short:"R" name:"recursive" help:"Descend into directory inputs for *.dcm files."`
	Tags             []string `short:"t" name:"tag" help:"Show only these tags ((GGGG,EEEE), GGGGEEEE, or keyword)."`
	Groups           []string `short:"g" name:"group" help:"Show only these groups (GGGG or a group name, e.g. \"patient\")."`
	ProcessPixelData bool     `name:"process-pixel-data" help:"Parse pixel-data element values (off by default)."`
	Redact           bool     `name:"redact" help:"Mask PHI-sensitive element values as [redacted]."`
	IgnoreErrors     bool     `name:"ignore-errors" help:"Exit 0 even if some inputs failed (exploratory)."`
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
//
// Under --format json, a single input emits one indented JSON object; multiple inputs emit a
// JSON Lines stream (one compact object per file per line) so the output is always parseable
// rather than several concatenated documents.
func (c *DumpCmd) Run(rc *RunContext) error {
	filt, err := c.buildFilter()
	if err != nil {
		return err
	}

	paths, err := c.resolvePaths()
	if err != nil {
		return err
	}

	if rc.Out.Format == cli.FormatCSV {
		return c.runCSV(rc, filt, paths)
	}

	jsonLines := rc.Out.Format == cli.FormatJSON && len(paths) > 1

	var firstErr error
	for _, path := range paths {
		df, inspectErr := c.inspect(path, filt)
		if emitErr := c.emit(rc, df, jsonLines); emitErr != nil {
			return emitErr
		}
		if inspectErr != nil && firstErr == nil {
			firstErr = inspectErr
		}
	}
	if c.IgnoreErrors {
		return nil
	}
	return firstErr
}

// resolvePaths expands the positional inputs into the concrete files to inspect. A directory
// input is descended for *.dcm files only when --recursive is set; without it a directory is a
// usage error (a directory is not a DICOM file). File inputs pass through unchanged so a
// missing file still surfaces its file-I/O error at read time.
func (c *DumpCmd) resolvePaths() ([]string, error) {
	var out []string
	for _, p := range c.Paths {
		info, err := os.Stat(p)
		if err != nil {
			// Let inspect surface the file-I/O error (exit 5) with the per-file machine shape.
			out = append(out, p)
			continue
		}
		if !info.IsDir() {
			out = append(out, p)
			continue
		}
		if !c.Recursive {
			return nil, &exitcode.UsageErr{Message: fmt.Sprintf("%s is a directory; pass -R/--recursive to descend into it", p)}
		}
		found, err := dicomFilesUnder(p)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}
	return out, nil
}

// dicomFilesUnder walks dir and returns every *.dcm file beneath it in lexical order, so a
// recursive dump is deterministic. The walk error is a file-I/O fault (exit 5).
func dicomFilesUnder(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".dcm") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// dumpFilter selects which elements a dump shows. An empty filter shows every element;
// otherwise an element is shown when its tag is in tags OR its group is in groups (an OR
// across the two flag families, so --tag and --group widen the selection together). The
// filter is built once per run from the validated flag values.
type dumpFilter struct {
	tags   map[dicom.Tag]bool
	groups map[uint16]bool
}

// active reports whether any filter is set. When false, every element passes.
func (f dumpFilter) active() bool { return len(f.tags) > 0 || len(f.groups) > 0 }

// shows reports whether t passes the filter.
func (f dumpFilter) shows(t dicom.Tag) bool {
	if !f.active() {
		return true
	}
	return f.tags[t] || f.groups[t.Group()]
}

// buildFilter parses and validates the --tag and --group flag values once. An unparseable
// tag or group is a usage error (a bad flag value), not a parse error: the fault is in the
// invocation, not in any DICOM input.
func (c *DumpCmd) buildFilter() (dumpFilter, error) {
	filt := dumpFilter{}
	for _, raw := range c.Tags {
		t, ok := parseTagSpec(raw)
		if !ok {
			return dumpFilter{}, &exitcode.UsageErr{Message: fmt.Sprintf("invalid --tag %q (use (GGGG,EEEE), GGGGEEEE, or a keyword)", raw)}
		}
		if filt.tags == nil {
			filt.tags = make(map[dicom.Tag]bool, len(c.Tags))
		}
		filt.tags[t] = true
	}
	for _, raw := range c.Groups {
		g, ok := parseGroupSpec(raw)
		if !ok {
			return dumpFilter{}, &exitcode.UsageErr{Message: fmt.Sprintf("invalid --group %q (use GGGG or a group name, e.g. \"patient\")", raw)}
		}
		if filt.groups == nil {
			filt.groups = make(map[uint16]bool, len(c.Groups))
		}
		filt.groups[g] = true
	}
	return filt, nil
}

// inspect reads one file and builds its machine shape. On a read or parse failure it returns a
// populated dumpFile with a "failure" status (so the per-file outcome is visible) AND the
// underlying error (so the runner classifies the exit), keeping the honest-failure contract: a
// failed parse is never reported as a clean dump. The filter selects which elements appear;
// PHI-sensitive values are masked only when --redact is set; pixel data is summarised unless
// --process-pixel-data is set.
func (c *DumpCmd) inspect(path string, filt dumpFilter) (dumpFile, error) {
	f, err := dicom.ReadFile(path)
	if err != nil {
		return dumpFile{File: path, Status: "failure", Error: structuralError(err)}, err
	}
	elements := make(map[string]dumpElement, f.DataSet.Len())
	for e := range f.DataSet.All() {
		if !filt.shows(e.Tag) {
			continue
		}
		key := tagKey(e.Tag)
		elements[key] = dumpElement{
			Tag:     e.Tag.String(),
			Keyword: keywordOf(e.Tag),
			VR:      e.VR.String(),
			Value:   c.elementValue(e),
		}
	}
	return dumpFile{File: path, Status: "success", Elements: elements}, nil
}

// elementValue renders an element's display value under the privacy and pixel-data rules.
// Values are shown by default; when --redact is set, a PHI-sensitive element (a PS3.15
// confidentiality attribute) is rendered as the fixed redacted marker so the caller can share a
// listing without patient values. Pixel data is summarised structurally unless
// --process-pixel-data is set, so the (large, never useful in a listing) pixel bytes stay out
// of the dump output by default.
func (c *DumpCmd) elementValue(e dicom.Element) string {
	if e.Tag == pixelDataTag && !c.ProcessPixelData {
		return fmt.Sprintf("<pixel data: %s, not processed>", e.VR)
	}
	if c.Redact && dicom.IsConfidential(e.Tag) {
		return redactedMarker
	}
	return renderValue(e)
}

// emit renders one file's result in the resolved format: an indented human listing or the
// canonical JSON shape. CSV is handled separately (runCSV) because it streams rows. Under
// jsonLines (set when multiple files are dumped as json) each file is one compact JSON object
// on its own line, so the stream stays parseable; a single file stays one indented object.
func (c *DumpCmd) emit(rc *RunContext, df dumpFile, jsonLines bool) error {
	if rc.Out.Format == cli.FormatJSON {
		if jsonLines {
			return rc.Out.EmitJSONLine(df)
		}
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
func (c *DumpCmd) runCSV(rc *RunContext, filt dumpFilter, paths []string) error {
	cw := rc.Out.CSVWriter()
	if err := cw.Write([]string{"file", "status", "tag", "keyword", "vr", "value"}); err != nil {
		return err
	}
	var firstErr error
	for _, path := range paths {
		df, err := c.inspect(path, filt)
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

// parseTagSpec resolves a --tag value to a Tag. It accepts the parenthesised "(GGGG,EEEE)"
// form, the bare 8-hex-digit "GGGGEEEE" form, and a dictionary keyword ("PatientName"). It
// returns ok == false for anything it cannot resolve, which the caller reports as a usage
// error.
func parseTagSpec(raw string) (dicom.Tag, bool) {
	s := strings.TrimSpace(raw)
	if t, ok := dicom.LookupKeyword(s); ok {
		return t, true
	}
	s = strings.TrimPrefix(strings.TrimSuffix(s, ")"), "(")
	s = strings.ReplaceAll(s, ",", "")
	if len(s) != 8 {
		return 0, false
	}
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, false
	}
	return dicom.NewTag(uint16(n>>16), uint16(n)), true // #nosec G115 -- ParseUint(s, 16, 32) bounds n to 32 bits; this splits its halves
}

// parseGroupSpec resolves a --group value to a 16-bit group number. It accepts a 4-hex-digit
// "GGGG" form and a small set of friendly group names (e.g. "patient"). It returns ok == false
// for anything it cannot resolve, which the caller reports as a usage error.
func parseGroupSpec(raw string) (uint16, bool) {
	s := strings.TrimSpace(raw)
	if g, ok := groupNames[strings.ToLower(s)]; ok {
		return g, true
	}
	if len(s) != 4 {
		return 0, false
	}
	n, err := strconv.ParseUint(s, 16, 16)
	if err != nil {
		return 0, false
	}
	return uint16(n), true
}

// groupNames maps the friendly group names accepted by --group to their DICOM group numbers.
// Only groups that correspond unambiguously to a single named group are listed; an unlisted
// name falls through to the 4-hex-digit group parse. Group 0x0008 is deliberately absent
// because it mixes SOP-common, study, and series identity, so no single label fits it.
var groupNames = map[string]uint16{
	"meta":         0x0002, // File Meta Information
	"patient":      0x0010, // Patient identification and demographics
	"acquisition":  0x0018, // Acquisition / equipment
	"relationship": 0x0020, // Study/Series/instance relationship
	"image":        0x0028, // Image presentation
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
