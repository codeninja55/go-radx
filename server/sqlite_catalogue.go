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
	"github.com/codeninja55/go-radx/dicomweb"
	"github.com/codeninja55/go-radx/dimse"
)

// indexedColumn is one queryable attribute the catalogue extracts and the column it maps to.
type indexedColumn struct {
	column string
	tag    dicom.Tag
	phi    bool
}

// catalogueConfig holds the resolved SQLiteCatalogue options.
type catalogueConfig struct {
	redact bool
}

// CatalogueOption configures a SQLiteCatalogue at construction.
type CatalogueOption func(*catalogueConfig)

// WithRedaction stores one-way hashes of the direct identifiers (PatientName, PatientID,
// AccessionNumber) instead of cleartext, for a non-clinical or shared-development catalogue. A
// redacted catalogue persists no cleartext for any direct identifier it stores, so it carries no
// reversible PHI. Off by default; redaction is a deliberate choice, not a hidden default (PRD §9.1).
func WithRedaction(enabled bool) CatalogueOption {
	return func(c *catalogueConfig) { c.redact = enabled }
}

// sqliteCatalogue is the default Catalogue: a SQLite database indexing the queryable attributes of
// stored objects. It indexes one row per instance, carrying the study/series/instance hierarchy and
// the queryable attributes the C-FIND and QIDO-RS models need. The direct-identifier columns
// (PatientName, PatientID, AccessionNumber) are stored cleartext by default and hashed under
// WithRedaction (PRD §9.1). The pure-Go modernc driver keeps the default build cgo-free.
type sqliteCatalogue struct {
	db     *sql.DB
	redact bool
}

// indexedColumns is the queryable attribute set the catalogue extracts and the column each maps to.
// It is the conformance subset the C-FIND and QIDO-RS models query at the study/series/instance
// levels. The PatientName, PatientID, and AccessionNumber columns hold direct identifiers (PHI) and
// are governed by the redaction option: AccessionNumber is an order/visit record locator that maps
// back to a patient (PS3.15 Annex E lists it among the identity attributes to remove), so a redacted
// catalogue must carry no cleartext for it any more than for the patient name or ID.
var indexedColumns = []indexedColumn{
	{"sop_instance_uid", dicom.TagSOPInstanceUID, false},
	{"series_instance_uid", dicom.TagSeriesInstanceUID, false},
	{"study_instance_uid", dicom.TagStudyInstanceUID, false},
	{"sop_class_uid", dicom.TagSOPClassUID, false},
	{"modality", dicom.TagModality, false},
	{"study_date", dicom.TagStudyDate, false},
	{"accession_number", dicom.TagAccessionNumber, true},
	{"series_number", dicom.TagSeriesNumber, false},
	{"instance_number", dicom.TagInstanceNumber, false},
	{"patient_id", dicom.TagPatientID, true},
	{"patient_name", dicom.TagPatientName, true},
}

// levelColumns projects the indexed columns onto the attributes a query at level identifies and
// returns, so the SELECT can distinct-collapse to one row per resource at that hierarchy level. A
// study-level query identifies a study (and its study-level attributes), a series-level query a
// study+series, and an instance-level query (the default) the full per-instance row. The order
// follows indexedColumns so the column list and scanRow stay aligned. A patient-level query has no
// catalogue patient table, so it is served at the study granularity, the coarsest the index keys.
func levelColumns(level dimse.QueryLevel) []indexedColumn {
	switch level {
	case dimse.QueryLevelPatient:
		return columnsByName("patient_id", "patient_name")
	case dimse.QueryLevelStudy:
		return columnsByName("study_instance_uid", "study_date", "accession_number", "patient_id", "patient_name")
	case dimse.QueryLevelSeries:
		return columnsByName("study_instance_uid", "series_instance_uid", "modality", "series_number",
			"accession_number", "patient_id", "patient_name")
	default:
		return indexedColumns
	}
}

// columnsByName selects the named indexed columns in indexedColumns order, so the projected SELECT
// column list and scanRow agree on column positions.
func columnsByName(names ...string) []indexedColumn {
	want := make(map[string]struct{}, len(names))
	for _, n := range names {
		want[n] = struct{}{}
	}
	out := make([]indexedColumn, 0, len(names))
	for _, ic := range indexedColumns {
		if _, ok := want[ic.column]; ok {
			out = append(out, ic)
		}
	}
	return out
}

// levelKeyTags lists the tags that uniquely identify a resource at level, so the collapser yields one
// row per study/series/instance rather than one per stored instance. A study collapses by Study UID, a
// series by Study+Series UID, an instance by SOP Instance UID, and a patient by the patient
// identifiers (the coarsest the index keys, since there is no separate patient table).
func levelKeyTags(level dimse.QueryLevel) []dicom.Tag {
	switch level {
	case dimse.QueryLevelPatient:
		return []dicom.Tag{dicom.TagPatientID, dicom.TagPatientName}
	case dimse.QueryLevelStudy:
		return []dicom.Tag{dicom.TagStudyInstanceUID}
	case dimse.QueryLevelSeries:
		return []dicom.Tag{dicom.TagStudyInstanceUID, dicom.TagSeriesInstanceUID}
	default:
		return []dicom.Tag{dicom.TagSOPInstanceUID}
	}
}

// levelCollapser collapses matched instance rows to one row per resource at a query level. Matching
// runs over the full per-instance row so an instance-level attribute the query constrains on is still
// visible; the collapse then projects the survivor to the level's identifying attributes and drops
// any later instance of an already-seen resource (the DISTINCT collapse moved AFTER the Go matcher).
type levelCollapser struct {
	keyTags []dicom.Tag
	project []indexedColumn
	retain  []dicom.Tag
	yielded map[string]struct{}
}

// newLevelCollapser builds a collapser for level, capturing the level's identifying key tags and the
// columns the projected row carries. retain names attributes the projected row must carry beyond the
// level's own projection — every match-key tag and any requested return field — so a downstream
// re-matcher or includefield projection sees the value rather than a row the level collapse dropped it
// from. A retained tag the row does not carry is simply absent from the projection.
func newLevelCollapser(level dimse.QueryLevel, retain []dicom.Tag) *levelCollapser {
	return &levelCollapser{
		keyTags: levelKeyTags(level),
		project: levelColumns(level),
		retain:  retain,
		yielded: make(map[string]struct{}),
	}
}

// collapse projects a matched instance row to the level's identifying attributes and reports whether it
// is the FIRST row for its resource: a true result is a new study/series/instance to yield, a false
// result is a duplicate to skip. The returned DataSet carries only the level's columns.
func (lc *levelCollapser) collapse(ds *dicom.DataSet) (*dicom.DataSet, bool) {
	key := lc.resourceKey(ds)
	if _, seen := lc.yielded[key]; seen {
		return nil, false
	}
	lc.yielded[key] = struct{}{}
	return lc.projectRow(ds), true
}

// resourceKey builds a collision-free identity for the row's resource at the query level by joining its
// key-tag values under a separator that cannot appear in a UID, so two distinct resources never share
// a key.
func (lc *levelCollapser) resourceKey(ds *dicom.DataSet) string {
	var b strings.Builder
	for _, tag := range lc.keyTags {
		v, _ := ds.GetString(tag)
		b.WriteString(v)
		b.WriteByte('\x1f')
	}
	return b.String()
}

// projectRow copies the level's projection columns from the matched instance row into a fresh DataSet,
// so a study-level row carries the study-level attributes and not the per-instance ones. The retained
// tags are carried alongside the projection so an attribute a downstream matcher or includefield
// projection needs — a match key outside the level's columns, a requested return field — survives the
// collapse rather than being dropped to the level's identifying set.
func (lc *levelCollapser) projectRow(ds *dicom.DataSet) *dicom.DataSet {
	out := dicom.NewDataSet()
	for _, ic := range lc.project {
		if v, ok := ds.GetString(ic.tag); ok && v != "" {
			out.SetString(ic.tag, v)
		}
	}
	for _, tag := range lc.retain {
		if _, ok := out.Get(tag); ok {
			continue
		}
		if e, ok := ds.Get(tag); ok {
			out.Set(cloneElement(e))
		}
	}
	return out
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
	stmt := "CREATE TABLE IF NOT EXISTS instances (" + strings.Join(cols, ", ") + ")" // #nosec G202 -- column names come only from the compile-time indexedColumns table
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
	// #nosec G201 -- column names come only from the compile-time indexedColumns table; all values bind via ? placeholders
	stmt := fmt.Sprintf("INSERT OR REPLACE INTO instances (%s) VALUES (%s)",
		strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	if _, err := c.db.ExecContext(ctx, stmt, args...); err != nil {
		return fmt.Errorf("server: index object: %w", err)
	}
	return nil
}

// Query answers a hierarchical query at the requested level. The SQL only NARROWS the candidate set
// — it pushes down a conservative, definitely-safe equality on an indexed column when the match value
// carries no DICOM matching syntax — and the authoritative DICOM matcher (PS3.4 Annex C: UID lists,
// DA/TM/DT ranges, wildcards, PN fuzzy) DECIDES which candidates match. A value with list, range, or
// wildcard syntax is never pushed to SQL (an exact comparison against the whole literal would return
// nothing), so it is left to the Go matcher. The full DICOM match runs at instance granularity, and
// the result is collapsed to the requested hierarchy level AFTER matching, so a study-level query
// returns one row per matching study while matching still sees instance-level attributes.
//
// A backend fault terminates the iterator with a typed error, never a laundered empty success
// (PRD §9.2). PHI columns are returned as stored (cleartext, or the redaction hash), so a redacted
// catalogue never re-materialises a cleartext identifier it does not hold.
func (c *sqliteCatalogue) Query(ctx context.Context, q CatalogueQuery) iter.Seq2[*dicom.DataSet, error] {
	return func(yield func(*dicom.DataSet, error) bool) {
		// Match and the level-collapse run over the full per-instance rows, so the matcher sees every
		// indexed attribute (a series-level Modality, an instance-level number) rather than a
		// level-projected subset that would drop the attribute the query constrains on.
		keys := c.matchKeys(q.Match)
		where, args := c.buildWhere(q.Match)
		stmt := c.buildSelect(indexedColumns) + where // #nosec G202 -- buildSelect/buildWhere interpolate only indexedColumns names; match values bind via ? placeholders

		rows, err := c.db.QueryContext(ctx, stmt, args...)
		if err != nil {
			yield(nil, fmt.Errorf("server: query catalogue: %w", err))
			return
		}
		defer func() { _ = rows.Close() }()

		// Limit and offset page the COLLAPSED, MATCHED result rows (QIDO-RS limit=/offset=), so paging
		// counts one row per resource at the requested level rather than the pre-collapse instance rows.
		out := newLevelCollapser(q.Level, q.Return)
		seen := 0
		for rows.Next() {
			ds, scanErr := scanRow(rows, indexedColumns)
			if scanErr != nil {
				yield(nil, scanErr)
				return
			}
			// The Go matcher decides: a candidate the conservative SQL did not fully constrain is still
			// tested against the full DICOM match rules before it is yielded.
			if !dicomweb.MatchDataSet(ds, keys, q.Fuzzy) {
				continue
			}
			projected, ok := out.collapse(ds)
			if !ok {
				// Already yielded a row for this study/series at the requested level; collapse the
				// duplicate so a study-level query yields one row per matching study.
				continue
			}
			if seen < q.Offset {
				seen++
				continue
			}
			seen++
			if !yield(projected, nil) {
				return
			}
			if q.Limit > 0 && seen-q.Offset >= q.Limit {
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

// buildSelect lists the projected columns in a stable order so scanRow maps the result columns back to
// their tags deterministically. It selects the full per-instance rows (no SQL-level collapse): matching
// runs over the instance attributes and the level-collapse to one row per study/series happens AFTER
// the Go matcher (PS3.4 hierarchical query), so a collapse cannot drop an instance the match needed.
func (c *sqliteCatalogue) buildSelect(cols []indexedColumn) string {
	names := make([]string, 0, len(cols))
	for _, ic := range cols {
		names = append(names, ic.column)
	}
	return "SELECT " + strings.Join(names, ", ") + " FROM instances"
}

// buildWhere builds a parameterised WHERE clause that NARROWS the candidate set without DECIDING the
// match: it pushes down only a conservative, definitely-safe equality on an indexed column when the
// match value carries no DICOM matching syntax — no backslash UID list, no DA/TM/DT range hyphen, no
// "*"/"?" wildcard. Each value is bound as a parameter so a hostile match value can never inject SQL
// (PRD §9.1 input validation). The authoritative DICOM matcher then decides every candidate, so the
// SQL clause is an optimisation that must never drop a row the matcher would keep: any value with
// list/range/wildcard syntax, and any unindexed key, is left entirely to the matcher rather than
// compared as a whole literal that would return zero rows. A universal match (empty value) is a return
// key, not a constraint, so it is skipped.
//
// Under redaction the PHI columns store a one-way hash of the identifier, never its cleartext, so a
// pushed-down equality on a redacted column hashes the value the same way before comparison —
// otherwise a query by the cleartext PatientID/PatientName would compare cleartext against a stored
// hash and never match. A redacted column supports only exact-identity matching: a wildcard or range
// match value cannot be satisfied against a one-way hash (the cleartext needed to test the pattern is
// gone), and the Go matcher cannot re-decide a hashed column either. Such a constraint therefore
// matches NOTHING — it is pushed down as an unsatisfiable predicate rather than dropped, because a
// dropped PHI constraint would silently return every row (a redacted catalogue must never over-match
// on a PHI filter it cannot honour).
func (c *sqliteCatalogue) buildWhere(match map[dicom.Tag]string) (string, []any) {
	column := columnByTag()
	var clauses []string
	var args []any
	for tag, value := range match {
		ic, ok := column[tag]
		if !ok || value == "" {
			continue
		}
		redacted := ic.phi && c.redact
		if redacted {
			// A redacted column cannot be re-decided by the Go matcher (it holds a hash), so the SQL must
			// honour it directly. Only exact equality is representable against a hash; a wildcard or range
			// match value matches nothing rather than being dropped (which would over-match every row).
			if !isSafeNarrowingValue(ic.tag, value) {
				clauses = append(clauses, "1 = 0")
				continue
			}
			clauses = append(clauses, ic.column+" = ?")
			args = append(args, hashIdentifier(value))
			continue
		}
		if !isSafeNarrowingValue(ic.tag, value) {
			// List, range, or wildcard syntax: leave it to the authoritative matcher rather than comparing
			// the whole literal as SQL equality (which would return zero rows).
			continue
		}
		clauses = append(clauses, ic.column+" = ?")
		args = append(args, value)
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// isSafeNarrowingValue reports whether value is a single token the catalogue may push down as exact
// SQL equality without risking dropping a candidate the DICOM matcher would keep. It is safe only when
// the value carries no DICOM matching syntax: no "*"/"?" wildcard, no backslash UID list, and — for a
// DA/TM/DT attribute, whose hyphen denotes a range — no hyphen. A hyphen in a non-temporal attribute
// is a literal character (a UID never contains one), so it does not bar narrowing there.
func isSafeNarrowingValue(tag dicom.Tag, value string) bool {
	if strings.ContainsAny(value, "*?\\") {
		return false
	}
	if isTemporalTag(tag) && strings.ContainsRune(value, '-') {
		return false
	}
	return true
}

// isTemporalTag reports whether tag's VR is DA, TM, or DT, the value representations whose match value
// may carry a range ("lo-hi"). The catalogue must not push down equality on a range value.
func isTemporalTag(tag dicom.Tag) bool {
	info, ok := dicom.Lookup(tag)
	if !ok {
		return false
	}
	switch info.VR {
	case dicom.VRDA, dicom.VRTM, dicom.VRDT:
		return true
	default:
		return false
	}
}

// matchKeys projects the catalogue's tag->value match map into the authoritative matcher's key slice,
// resolving each key's VR from the dictionary so the matcher applies the right rule (UID list, range,
// wildcard, single value). A universal (empty) value is dropped: it constrains nothing.
//
// Only keys the catalogue INDEXES are returned. A key on an attribute the catalogue does not store
// cannot be decided here — the candidate row never carries the attribute, so the matcher would reject
// every candidate. Such a key is left to the caller, which fetches the stored dataset from the
// ObjectStore and applies the full match against real attribute values (see the DICOMweb and DIMSE
// roles). A redacted PHI key is dropped for the same reason: the column holds a one-way hash, never the
// cleartext the matcher compares against, so the SQL clause already decided it — by hashed equality for
// an exact value, or by an unsatisfiable predicate for a wildcard/range a hash cannot honour — and the
// matcher must not re-test it against the hash.
func (c *sqliteCatalogue) matchKeys(match map[dicom.Tag]string) []dicomweb.MatchKey {
	column := columnByTag()
	keys := make([]dicomweb.MatchKey, 0, len(match))
	for tag, value := range match {
		if value == "" {
			continue
		}
		ic, ok := column[tag]
		if !ok {
			continue
		}
		if ic.phi && c.redact {
			continue
		}
		keys = append(keys, dicomweb.NewMatchKey(tag, value))
	}
	return keys
}

// scanRow reconstructs a DataSet from one result row over the projected columns, writing each
// non-empty column back under its DICOM tag. A NULL/empty column is omitted so the returned
// identifier carries only present attributes. The column slice MUST match the SELECT projection so
// the scan destinations and result columns align.
func scanRow(rows *sql.Rows, cols []indexedColumn) (*dicom.DataSet, error) {
	dest := make([]sql.NullString, len(cols))
	ptrs := make([]any, len(cols))
	for i := range dest {
		ptrs[i] = &dest[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, fmt.Errorf("server: scan catalogue row: %w", err)
	}
	ds := dicom.NewDataSet()
	for i, ic := range cols {
		if dest[i].Valid && dest[i].String != "" {
			ds.SetString(ic.tag, dest[i].String)
		}
	}
	return ds, nil
}

// columnByTag maps each indexed DICOM tag to its column metadata for WHERE-clause construction, so
// buildWhere reads both the column name and whether it is a redaction-governed PHI column.
func columnByTag() map[dicom.Tag]indexedColumn {
	m := make(map[dicom.Tag]indexedColumn, len(indexedColumns))
	for _, ic := range indexedColumns {
		m[ic.tag] = ic
	}
	return m
}

// hashIdentifier returns a stable one-way hash of a direct identifier for the redacted catalogue, so
// a non-clinical catalogue can still match on equality without storing the cleartext value.
func hashIdentifier(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}
