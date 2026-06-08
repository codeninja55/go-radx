package server

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"iter"
	"strings"

	_ "modernc.org/sqlite" // pure-Go SQLite driver; keeps the default build cgo-free.

	"github.com/codeninja55/go-radx/dicom"
)

// catalogueConfig holds the resolved SQLiteCatalogue options.
type catalogueConfig struct {
	redact bool
}

// CatalogueOption configures a SQLiteCatalogue at construction.
type CatalogueOption func(*catalogueConfig)

// WithRedaction stores one-way hashes of direct identifiers (PatientName, PatientID) instead of
// cleartext, for a non-clinical or shared-development catalogue. Off by default; redaction is a
// deliberate choice, not a hidden default (PRD §9.1).
func WithRedaction(enabled bool) CatalogueOption {
	return func(c *catalogueConfig) { c.redact = enabled }
}

// sqliteCatalogue is the default Catalogue: a SQLite database indexing the queryable attributes of
// stored objects. It indexes one row per instance, carrying the study/series/instance hierarchy and
// the queryable attributes the C-FIND and QIDO-RS models need. PHI columns (PatientName, PatientID)
// are stored cleartext by default and hashed under WithRedaction (PRD §9.1). The pure-Go modernc
// driver keeps the default build cgo-free.
type sqliteCatalogue struct {
	db     *sql.DB
	redact bool
}

// indexedTags is the queryable attribute set the catalogue extracts and the column each maps to. It
// is the conformance subset the C-FIND and QIDO-RS models query at the study/series/instance levels.
// The PatientName and PatientID columns hold PHI and are governed by the redaction option.
var indexedColumns = []struct {
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

// SQLiteCatalogue is the default Catalogue: a SQLite database at dbPath indexing the queryable
// attributes of stored objects. dbPath is required and is never defaulted, because the catalogue
// holds PHI — no command or daemon silently creates a PHI-bearing catalogue at a default path
// (PRD §9.1; the prototype's accidental default-path PHI store is the defect this forbids). Pass
// ":memory:" for an ephemeral in-process catalogue (tests). It offers a redacted mode
// (WithRedaction) that hashes direct identifiers for non-clinical use.
func SQLiteCatalogue(ctx context.Context, dbPath string, opts ...CatalogueOption) (Catalogue, error) {
	if dbPath == "" {
		return nil, errors.New("server: catalogue path must be named explicitly (the catalogue holds PHI)")
	}
	cfg := catalogueConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("server: open catalogue: %w", err)
	}
	// SQLite serialises writers; a single connection avoids "database is locked" under the daemon's
	// concurrent handlers while keeping the contract concurrency-safe (PRD §9.4).
	db.SetMaxOpenConns(1)

	cat := &sqliteCatalogue{db: db, redact: cfg.redact}
	if err := cat.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return cat, nil
}

// migrate creates the instance index table if it does not exist. The schema is keyed on SOP Instance
// UID (the catalogue's primary key, matching ObjectStore), so Index is idempotent: re-indexing the
// same instance updates its row rather than duplicating it.
func (c *sqliteCatalogue) migrate(ctx context.Context) error {
	cols := make([]string, 0, len(indexedColumns))
	for _, ic := range indexedColumns {
		if ic.column == "sop_instance_uid" {
			cols = append(cols, ic.column+" TEXT PRIMARY KEY")
			continue
		}
		cols = append(cols, ic.column+" TEXT")
	}
	stmt := "CREATE TABLE IF NOT EXISTS instances (" + strings.Join(cols, ", ") + ")"
	if _, err := c.db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("server: migrate catalogue: %w", err)
	}
	return nil
}

// Index records or updates the queryable attributes of ds. PHI columns are hashed under redaction.
// An object with no SOP Instance UID cannot be keyed and is rejected rather than indexed under a
// placeholder (PRD §9.2).
func (c *sqliteCatalogue) Index(ctx context.Context, ds *dicom.DataSet) error {
	instance, ok := ds.GetString(dicom.TagSOPInstanceUID)
	if !ok || instance == "" {
		return errors.New("server: cannot index an object with no SOPInstanceUID")
	}

	cols := make([]string, 0, len(indexedColumns))
	placeholders := make([]string, 0, len(indexedColumns))
	args := make([]any, 0, len(indexedColumns))
	for _, ic := range indexedColumns {
		v, _ := ds.GetString(ic.tag)
		if ic.phi && c.redact && v != "" {
			v = hashIdentifier(v)
		}
		cols = append(cols, ic.column)
		placeholders = append(placeholders, "?")
		args = append(args, v)
	}
	stmt := fmt.Sprintf("INSERT OR REPLACE INTO instances (%s) VALUES (%s)",
		strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	if _, err := c.db.ExecContext(ctx, stmt, args...); err != nil {
		return fmt.Errorf("server: index object: %w", err)
	}
	return nil
}

// Query answers a hierarchical query at the requested level. It builds a parameterised SELECT from
// the match keys (so a match value can never inject SQL) and streams the rows as an iterator,
// reconstructing one DataSet per row. A backend fault terminates the iterator with a typed error,
// never a laundered empty success (PRD §9.2). PHI columns are returned as stored (cleartext, or the
// redaction hash), so a redacted catalogue never re-materialises a cleartext identifier it does not
// hold.
func (c *sqliteCatalogue) Query(ctx context.Context, q CatalogueQuery) iter.Seq2[*dicom.DataSet, error] {
	return func(yield func(*dicom.DataSet, error) bool) {
		where, args := c.buildWhere(q.Match)
		stmt := c.buildSelect() + where
		if q.Limit > 0 {
			stmt += fmt.Sprintf(" LIMIT %d", q.Limit)
		}
		if q.Offset > 0 {
			stmt += fmt.Sprintf(" OFFSET %d", q.Offset)
		}

		rows, err := c.db.QueryContext(ctx, stmt, args...)
		if err != nil {
			yield(nil, fmt.Errorf("server: query catalogue: %w", err))
			return
		}
		defer func() { _ = rows.Close() }()

		for rows.Next() {
			ds, scanErr := c.scanRow(rows)
			if scanErr != nil {
				yield(nil, scanErr)
				return
			}
			if !yield(ds, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("server: iterate catalogue: %w", err))
		}
	}
}

// Remove drops the index entry for one instance, returning ErrNotFound when no such row exists so a
// caller distinguishes a no-op from a successful removal.
func (c *sqliteCatalogue) Remove(ctx context.Context, instance dicom.SOPInstanceUID) error {
	res, err := c.db.ExecContext(ctx, "DELETE FROM instances WHERE sop_instance_uid = ?", string(instance))
	if err != nil {
		return fmt.Errorf("server: remove from catalogue: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("server: remove from catalogue: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("%w: instance not indexed", ErrNotFound)
	}
	return nil
}

// Close releases the database handle.
func (c *sqliteCatalogue) Close() error { return c.db.Close() }

// buildSelect lists every indexed column in a stable order so scanRow maps the result columns back to
// their tags deterministically.
func (c *sqliteCatalogue) buildSelect() string {
	cols := make([]string, 0, len(indexedColumns))
	for _, ic := range indexedColumns {
		cols = append(cols, ic.column)
	}
	return "SELECT " + strings.Join(cols, ", ") + " FROM instances"
}

// buildWhere builds a parameterised WHERE clause from the match keys: only tags the catalogue indexes
// constrain the query, and each value is bound as a parameter so a hostile match value can never
// inject SQL (PRD §9.1 input validation). A universal match (empty value) is a return key, not a
// constraint, so it is skipped. A "*" wildcard becomes a LIKE so a QIDO-RS wildcard match works.
func (c *sqliteCatalogue) buildWhere(match map[dicom.Tag]string) (string, []any) {
	column := columnByTag()
	var clauses []string
	var args []any
	for tag, value := range match {
		col, ok := column[tag]
		if !ok || value == "" {
			continue
		}
		if strings.ContainsAny(value, "*?") {
			clauses = append(clauses, col+" LIKE ?")
			args = append(args, wildcardToLike(value))
			continue
		}
		clauses = append(clauses, col+" = ?")
		args = append(args, value)
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// scanRow reconstructs a DataSet from one result row, writing each non-empty column back under its
// DICOM tag. A NULL/empty column is omitted so the returned identifier carries only present
// attributes.
func (c *sqliteCatalogue) scanRow(rows *sql.Rows) (*dicom.DataSet, error) {
	dest := make([]sql.NullString, len(indexedColumns))
	ptrs := make([]any, len(indexedColumns))
	for i := range dest {
		ptrs[i] = &dest[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, fmt.Errorf("server: scan catalogue row: %w", err)
	}
	ds := dicom.NewDataSet()
	for i, ic := range indexedColumns {
		if dest[i].Valid && dest[i].String != "" {
			ds.SetString(ic.tag, dest[i].String)
		}
	}
	return ds, nil
}

// columnByTag maps each indexed DICOM tag to its column name for WHERE-clause construction.
func columnByTag() map[dicom.Tag]string {
	m := make(map[dicom.Tag]string, len(indexedColumns))
	for _, ic := range indexedColumns {
		m[ic.tag] = ic.column
	}
	return m
}

// wildcardToLike translates a DICOM wildcard match value (PS3.4 C.2.2.2.4: "*" any, "?" one char) to
// a SQL LIKE pattern, escaping the SQL metacharacters so only the DICOM wildcards are special.
func wildcardToLike(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch r {
		case '*':
			b.WriteByte('%')
		case '?':
			b.WriteByte('_')
		case '%', '_':
			// A literal SQL wildcard in the value must not act as a pattern; there is no ESCAPE clause,
			// so collapse it to a single-char match rather than letting it widen the result silently.
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// hashIdentifier returns a stable one-way hash of a direct identifier for the redacted catalogue, so
// a non-clinical catalogue can still match on equality without storing the cleartext value.
func hashIdentifier(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}
