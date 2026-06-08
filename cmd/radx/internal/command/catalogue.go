package command

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	_ "modernc.org/sqlite" // pure-Go SQLite driver; keeps the CLI build cgo-free.

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/logging"
	"github.com/codeninja55/go-radx/server"
)

// catalogueColumns is the catalogue's column set, mirroring the server package's index schema, used
// to render --schema and to drive the tag-filter query. It is the conformance subset the C-FIND and
// QIDO-RS models query, with patient_id and patient_name flagged as the PHI columns.
var catalogueColumns = []struct {
	column string
	tag    dicom.Tag
	phi    bool
}{
	{"sop_instance_uid", dicom.TagSOPInstanceUID, false},
	{"series_instance_uid", dicom.TagSeriesInstanceUID, false},
	{"study_instance_uid", dicom.TagStudyInstanceUID, false},
	{"sop_class_uid", dicom.TagSOPClassUID, false},
	{"modality", dicom.TagModality, false},
	{"study_date", dicom.TagStudyDate, false},
	{"accession_number", dicom.TagAccessionNumber, false},
	{"series_number", dicom.TagSeriesNumber, false},
	{"instance_number", dicom.TagInstanceNumber, false},
	{"patient_id", dicom.TagPatientID, true},
	{"patient_name", dicom.TagPatientName, true},
}

// CatalogueCmd indexes a directory of DICOM files into a local SQLite catalogue and queries it by
// tag filters or read-only SQL. The catalogue stores patient identifiers, so it is a PHI-bearing
// convenience store and is opt-in (RADX-007/008): creating a catalogue with PHI columns requires
// --confirm-phi, --redact indexes only structural fields, the database file is created 0600, and
// neither SQL text nor filter values are logged at default verbosity. Indexing that fails on some
// files exits non-zero unless --ignore-errors (RADX-013).
type CatalogueCmd struct {
	Dir string `arg:"" optional:"" name:"dir" help:"Directory to index (omit to query an existing catalogue)."`

	Database     string   `short:"d" name:"database" default:"dicom-catalogue.db" help:"Catalogue file path."`
	Rebuild      bool     `name:"rebuild" help:"Drop and rebuild from scratch."`
	Recursive    bool     `short:"R" name:"recursive" default:"true" negatable:"" help:"Descend into the source."`
	Query        []string `name:"query" help:"Tag filter (key=value); repeat to add filters."`
	SQL          string   `name:"sql" help:"Read-only SQL (SELECT only)."`
	Mode         string   `short:"m" name:"mode" enum:"table,csv,json,jsonl,list,markdown" default:"table" help:"SQL result rendering."`
	Schema       bool     `name:"schema" help:"Print the catalogue schema and exit."`
	ConfirmPHI   bool     `name:"confirm-phi" help:"Acknowledge that the catalogue stores PHI."`
	Redact       bool     `name:"redact" help:"Index structural fields only; omit PHI columns."`
	IgnoreErrors bool     `name:"ignore-errors" help:"Exit 0 even if some files failed to index."`
}

// catalogueIndexResult is the canonical machine shape for an index run: the catalogue path and the
// per-outcome tally of indexed and failed files. It names counts only, never patient values.
type catalogueIndexResult struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Indexed  int    `json:"indexed"`
	Failed   int    `json:"failed"`
}

// Run dispatches to the requested mode: print the schema, index a directory, or query an existing
// catalogue. The PHI gate is enforced before any index that would write patient columns.
func (c *CatalogueCmd) Run(rc *RunContext) error {
	if c.Schema {
		return c.printSchema(rc)
	}
	if c.SQL != "" {
		return c.runSQL(rc)
	}
	if len(c.Query) > 0 {
		return c.runQuery(rc)
	}
	if c.Dir != "" {
		return c.runIndex(rc)
	}
	return &exitcode.UsageErr{Message: "nothing to do: pass a directory to index, --query, --sql, or --schema"}
}

// runIndex indexes the source directory into the catalogue. It enforces the PHI gate, opens the
// catalogue (creating the file 0600), indexes each file, and tallies failures. A failed file makes
// the run exit non-zero unless --ignore-errors (RADX-013).
func (c *CatalogueCmd) runIndex(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return &exitcode.UsageErr{Message: "catalogue index does not support --format csv; use human or json"}
	}
	if !c.Redact && !c.ConfirmPHI {
		return &exitcode.UsageErr{Message: "indexing PHI columns requires --confirm-phi (or use --redact for a structural-only index)"}
	}

	info, err := os.Stat(c.Dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return &exitcode.UsageErr{Message: fmt.Sprintf("%s is not a directory", c.Dir)}
	}

	if c.Rebuild {
		if err := os.Remove(c.Database); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	cat, err := server.SQLiteCatalogue(rc.Ctx, c.Database, server.WithRedaction(c.Redact))
	if err != nil {
		return err
	}
	defer closeCatalogue(cat)
	// Harden the catalogue file to owner-only: it holds PHI, so it is never world-readable
	// (RADX-008). The chmod happens after the driver created the file.
	if err := os.Chmod(c.Database, 0o600); err != nil {
		return err
	}

	files, err := dicomSourceFiles(c.Dir, c.Recursive)
	if err != nil {
		return err
	}

	log := logging.FromContext(rc.Ctx)
	indexed, failed := 0, 0
	for _, path := range files {
		if err := indexFile(rc.Ctx, cat, path); err != nil {
			failed++
			// Diagnostics name the file and the structural reason, never a patient value.
			log.Warn("catalogue: failed to index file", zap.String("file", path), zap.Error(err))
			continue
		}
		indexed++
	}

	result := catalogueIndexResult{Database: c.Database, Indexed: indexed, Failed: failed}
	if failed == 0 {
		result.Status = "success"
	} else {
		result.Status = "failure"
	}
	if emitErr := c.emitIndex(rc, result); emitErr != nil {
		return emitErr
	}
	if failed > 0 && !c.IgnoreErrors {
		return &exitcode.UsageErr{Message: fmt.Sprintf("%d of %d files failed to index", failed, indexed+failed)}
	}
	return nil
}

// closeCatalogue closes the catalogue's backing database when the concrete implementation is an
// io.Closer (the SQLite catalogue is), so a CLI run does not leak the connection. The Catalogue
// interface itself exposes no Close, so the assertion is the documented way to release it.
func closeCatalogue(cat server.Catalogue) {
	if closer, ok := cat.(io.Closer); ok {
		_ = closer.Close()
	}
}

// indexFile reads one DICOM file and indexes it. A read/parse failure or an index failure is
// propagated so the caller tallies it; truncation is a parse failure, never a logged no-op
// (RADX-012/013).
func indexFile(ctx context.Context, cat server.Catalogue, path string) error {
	f, err := dicom.ReadFile(path)
	if err != nil {
		return err
	}
	return cat.Index(ctx, f.DataSet)
}

// runQuery answers a tag-filter query against an existing catalogue, streaming one result per match
// through the server catalogue's iterator. A backend fault terminates the stream with an error,
// never a laundered empty success (RADX-015).
func (c *CatalogueCmd) runQuery(rc *RunContext) error {
	match, err := buildCatalogueMatch(c.Query)
	if err != nil {
		return err
	}
	// A catalogue indexed with --redact stores PHI columns (PatientID/PatientName) as one-way
	// hashes, so an exact filter only matches when the backend hashes the query value the same way.
	// Opening the query with the redaction setting used at index time is therefore required: a
	// redacted catalogue queried without --redact compares cleartext against hashes and returns
	// nothing. Redaction state is not recorded in the database, so it must be re-specified here.
	cat, err := server.SQLiteCatalogue(rc.Ctx, c.Database, server.WithRedaction(c.Redact))
	if err != nil {
		return err
	}
	defer closeCatalogue(cat)

	em := &matchEmitter{out: rc.Out, columns: queryColumns(c.Query)}
	if err := em.start(); err != nil {
		return err
	}
	for ds, qErr := range cat.Query(rc.Ctx, server.CatalogueQuery{Level: dimse.QueryLevelImage, Match: match}) {
		if qErr != nil {
			return qErr
		}
		if emitErr := em.emit(datasetAttributes(ds)); emitErr != nil {
			return emitErr
		}
	}
	return em.finish()
}

// runSQL runs a read-only SELECT against the catalogue and renders the rows in --mode. The SQL is
// validated to be a non-empty SELECT before execution — empty or non-SELECT SQL is a clean usage
// error, never a panic (RADX-014) — and the database is opened read-only so a stray mutation cannot
// alter the PHI store.
func (c *CatalogueCmd) runSQL(rc *RunContext) error {
	stmt := strings.TrimSpace(c.SQL)
	if stmt == "" {
		return &exitcode.UsageErr{Message: "--sql is empty"}
	}
	if !isReadOnlySelect(stmt) {
		return &exitcode.UsageErr{Message: "--sql must be a single read-only SELECT statement"}
	}

	db, err := sql.Open("sqlite", "file:"+c.Database+"?mode=ro")
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	rows, err := db.QueryContext(rc.Ctx, stmt)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	records, err := scanAllRows(rows, cols)
	if err != nil {
		return err
	}
	// Check the iteration error AFTER the loop, so a row-scan fault is never reported as a clean
	// result (RADX-015).
	if err := rows.Err(); err != nil {
		return err
	}
	return renderSQLResult(rc, c.Mode, cols, records)
}

// printSchema prints the catalogue schema (the column set and its PHI flags) and exits. It opens no
// database, so it works before a catalogue exists.
func (c *CatalogueCmd) printSchema(rc *RunContext) error {
	type schemaColumn struct {
		Column string `json:"column"`
		Tag    string `json:"tag"`
		PHI    bool   `json:"phi"`
	}
	if rc.Out.Format == cli.FormatJSON {
		out := make([]schemaColumn, 0, len(catalogueColumns))
		for _, col := range catalogueColumns {
			out = append(out, schemaColumn{Column: col.column, Tag: col.tag.String(), PHI: col.phi})
		}
		return rc.Out.EmitJSON(out)
	}
	if rc.Out.Format == cli.FormatCSV {
		cw := newCSVWriter(rc.Out)
		if err := cw.write([]string{"column", "tag", "phi"}); err != nil {
			return err
		}
		for _, col := range catalogueColumns {
			if err := cw.write([]string{col.column, col.tag.String(), fmt.Sprintf("%t", col.phi)}); err != nil {
				return err
			}
		}
		return cw.flush()
	}
	for _, col := range catalogueColumns {
		phi := ""
		if col.phi {
			phi = "  [PHI]"
		}
		if _, err := fmt.Fprintf(rc.Out.Machine, "%-20s %s%s\n", col.column, col.tag.String(), phi); err != nil {
			return err
		}
	}
	return nil
}

// emitIndex renders the index-run tally in the resolved format.
func (c *CatalogueCmd) emitIndex(rc *RunContext, r catalogueIndexResult) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSON(r)
	}
	_, err := fmt.Fprintf(rc.Out.Machine, "indexed %d files into %s (%d failed)\n", r.Indexed, r.Database, r.Failed)
	return err
}

// buildCatalogueMatch turns the --query key=value pairs into a tag-keyed match map. An unparseable
// key is a usage error.
func buildCatalogueMatch(pairs []string) (map[dicom.Tag]string, error) {
	match := make(map[dicom.Tag]string, len(pairs))
	for _, raw := range pairs {
		key, value, ok := splitKeyValue(raw)
		if !ok {
			return nil, &exitcode.UsageErr{Message: fmt.Sprintf("invalid --query %q (use key=value)", raw)}
		}
		t, resolvable := parseTagSpec(key)
		if !resolvable {
			return nil, &exitcode.UsageErr{Message: fmt.Sprintf("invalid --query key %q", key)}
		}
		match[t] = value
	}
	return match, nil
}

// queryColumns returns the --query keys as the streamed-output columns, in input order.
func queryColumns(pairs []string) []string {
	cols := make([]string, 0, len(pairs))
	for _, raw := range pairs {
		key, _, ok := splitKeyValue(raw)
		if ok {
			cols = append(cols, key)
		}
	}
	return cols
}

// isReadOnlySelect reports whether stmt is a single read-only SELECT: it must begin with SELECT (or
// a WITH … SELECT CTE) and contain no statement separator that could smuggle a second, mutating
// statement. It is a conservative gate, not a full SQL parser; anything it is unsure of is rejected.
func isReadOnlySelect(stmt string) bool {
	lower := strings.ToLower(stmt)
	if !strings.HasPrefix(lower, "select") && !strings.HasPrefix(lower, "with") {
		return false
	}
	// Reject a trailing or embedded statement separator that could append a second statement, and
	// the mutating keywords, so a read-only intent cannot be subverted.
	trimmed := strings.TrimRight(stmt, "; \t\r\n")
	if strings.Contains(trimmed, ";") {
		return false
	}
	for _, kw := range []string{"insert ", "update ", "delete ", "drop ", "alter ", "create ", "attach ", "pragma "} {
		if strings.Contains(lower, kw) {
			return false
		}
	}
	return true
}

// scanAllRows reads every row into a string slice per row, rendering NULLs as the empty string so
// the output is total. It buffers the result, which is acceptable for a triage catalogue query.
func scanAllRows(rows *sql.Rows, cols []string) ([][]string, error) {
	var out [][]string
	for rows.Next() {
		cells := make([]sql.NullString, len(cols))
		dst := make([]any, len(cols))
		for i := range cells {
			dst[i] = &cells[i]
		}
		if err := rows.Scan(dst...); err != nil {
			return nil, err
		}
		row := make([]string, len(cols))
		for i, cell := range cells {
			if cell.Valid {
				row[i] = cell.String
			}
		}
		out = append(out, row)
	}
	return out, nil
}

// renderSQLResult renders the SQL rows in the requested --mode. The global --format is independent
// of --mode: a SQL query renders by its mode regardless of --format, since the result is tabular by
// nature.
func renderSQLResult(rc *RunContext, mode string, cols []string, rows [][]string) error {
	w := rc.Out.Machine
	switch mode {
	case "csv":
		cw := newCSVWriter(rc.Out)
		if err := cw.write(cols); err != nil {
			return err
		}
		for _, row := range rows {
			if err := cw.write(row); err != nil {
				return err
			}
		}
		return cw.flush()
	case "json":
		objs := make([]map[string]string, 0, len(rows))
		for _, row := range rows {
			objs = append(objs, rowObject(cols, row))
		}
		return rc.Out.EmitJSON(objs)
	case "jsonl":
		for _, row := range rows {
			b, err := json.Marshal(rowObject(cols, row))
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintln(w, string(b)); err != nil {
				return err
			}
		}
		return nil
	case "list":
		for _, row := range rows {
			if _, err := fmt.Fprintln(w, strings.Join(row, "|")); err != nil {
				return err
			}
		}
		return nil
	case "markdown":
		return renderMarkdownTable(w, cols, rows)
	default: // table
		return renderPlainTable(w, cols, rows)
	}
}

// rowObject pairs the columns with a row's cells into a string-keyed object for json output.
func rowObject(cols, row []string) map[string]string {
	obj := make(map[string]string, len(cols))
	for i, col := range cols {
		obj[col] = row[i]
	}
	return obj
}

// renderPlainTable writes a fixed-width text table with a header rule.
func renderPlainTable(w io.Writer, cols []string, rows [][]string) error {
	widths := columnWidths(cols, rows)
	if err := writeTableRow(w, cols, widths); err != nil {
		return err
	}
	rule := make([]string, len(cols))
	for i, width := range widths {
		rule[i] = strings.Repeat("-", width)
	}
	if err := writeTableRow(w, rule, widths); err != nil {
		return err
	}
	for _, row := range rows {
		if err := writeTableRow(w, row, widths); err != nil {
			return err
		}
	}
	return nil
}

// renderMarkdownTable writes a GitHub-flavoured markdown table.
func renderMarkdownTable(w io.Writer, cols []string, rows [][]string) error {
	if _, err := fmt.Fprintf(w, "| %s |\n", strings.Join(cols, " | ")); err != nil {
		return err
	}
	sep := make([]string, len(cols))
	for i := range sep {
		sep[i] = "---"
	}
	if _, err := fmt.Fprintf(w, "| %s |\n", strings.Join(sep, " | ")); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := fmt.Fprintf(w, "| %s |\n", strings.Join(row, " | ")); err != nil {
			return err
		}
	}
	return nil
}

// columnWidths computes the display width of each column for the plain table.
func columnWidths(cols []string, rows [][]string) []int {
	widths := make([]int, len(cols))
	for i, col := range cols {
		widths[i] = len(col)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	return widths
}

// writeTableRow writes one padded table row.
func writeTableRow(w io.Writer, cells []string, widths []int) error {
	parts := make([]string, len(cells))
	for i, cell := range cells {
		parts[i] = fmt.Sprintf("%-*s", widths[i], cell)
	}
	_, err := fmt.Fprintln(w, strings.Join(parts, "  "))
	return err
}
