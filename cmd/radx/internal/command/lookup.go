package command

import (
	"fmt"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/dicom"
)

// LookupCmd resolves a DICOM tag or keyword against the standard data dictionary and prints the
// authoritative information (no standalone dcmtk tool; dcmdump ships the dictionaries this queries).
// Lookup is wired to the generated dictionary, not a hand-curated partial list with heuristic VR
// inference (RADX-019). A query that resolves to no dictionary entry is a parse failure (exit 3):
// the input named something the dictionary does not define, not a silent empty success.
type LookupCmd struct {
	Query []string `arg:"" name:"query" help:"Tag ((GGGG,EEEE) or GGGGEEEE) or keyword (PatientName)."`
}

// lookupRecord is the canonical machine shape for one resolved query: the tag, its canonical name
// and keyword, the VR and value multiplicity. It carries no patient value.
type lookupRecord struct {
	Query   string `json:"query"`
	Status  string `json:"status"`
	Tag     string `json:"tag,omitempty"`
	Keyword string `json:"keyword,omitempty"`
	Name    string `json:"name,omitempty"`
	VR      string `json:"vr,omitempty"`
	VM      string `json:"vm,omitempty"`
}

// Run resolves each query and emits its record. A single unresolved query makes the command exit 3,
// so a script can tell a confirmed dictionary entry from a name the standard does not define.
func (c *LookupCmd) Run(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return c.runCSV(rc)
	}

	jsonLines := rc.Out.Format == cli.FormatJSON && len(c.Query) > 1
	var unresolved bool
	for _, q := range c.Query {
		rec := c.resolve(q)
		if rec.Status != "found" {
			unresolved = true
		}
		if err := c.emit(rc, rec, jsonLines); err != nil {
			return err
		}
	}
	if unresolved {
		return errUnresolvedLookup()
	}
	return nil
}

// resolve resolves one query to a dictionary record. It accepts a parenthesised tag, a bare 8-hex
// tag, or a keyword (the same forms parseTagSpec accepts), then reads the authoritative TagInfo.
func (c *LookupCmd) resolve(query string) lookupRecord {
	tag, ok := parseTagSpec(query)
	if !ok {
		return lookupRecord{Query: query, Status: "not_found"}
	}
	info, ok := dicom.Lookup(tag)
	if !ok {
		return lookupRecord{Query: query, Status: "not_found", Tag: tag.String()}
	}
	return lookupRecord{
		Query:   query,
		Status:  "found",
		Tag:     tag.String(),
		Keyword: info.Keyword,
		Name:    info.Name,
		VR:      info.VR.String(),
		VM:      info.VM,
	}
}

// runCSV emits one CSV row per query. An unresolved query still produces a row (marked not_found)
// and drives the non-zero exit, so the tabular output names every query's outcome.
func (c *LookupCmd) runCSV(rc *RunContext) error {
	cw := newCSVWriter(rc.Out)
	if err := cw.write([]string{"query", "status", "tag", "keyword", "name", "vr", "vm"}); err != nil {
		return err
	}
	var unresolved bool
	for _, q := range c.Query {
		rec := c.resolve(q)
		if rec.Status != "found" {
			unresolved = true
		}
		if err := cw.write([]string{rec.Query, rec.Status, rec.Tag, rec.Keyword, rec.Name, rec.VR, rec.VM}); err != nil {
			return err
		}
	}
	if err := cw.flush(); err != nil {
		return err
	}
	if unresolved {
		return errUnresolvedLookup()
	}
	return nil
}

// errUnresolvedLookup is the error a lookup raises when a well-formed query names nothing the
// standard dictionary defines. It is a parse/validation failure, not a usage fault: the input is
// syntactically valid (a parseable tag or a plausible keyword) but resolves to no dictionary entry,
// the same "well-formed input the standard does not define" class as a malformed DICOM value, so it
// classifies to exit 3 (parse) rather than exit 2 (usage). A genuinely malformed query string is
// rejected earlier as a usage error by the argument parser. The message names no PHI (the query is a
// tag or keyword, never a patient value).
func errUnresolvedLookup() error {
	return &dicom.ValueError{
		VR:  dicom.VRUI,
		Msg: "one or more queries did not resolve to a dictionary entry",
	}
}

// emit renders one lookup record in the resolved format.
func (c *LookupCmd) emit(rc *RunContext, rec lookupRecord, jsonLines bool) error {
	if rc.Out.Format == cli.FormatJSON {
		if jsonLines {
			return rc.Out.EmitJSONLine(rec)
		}
		return rc.Out.EmitJSON(rec)
	}
	if rec.Status != "found" {
		_, err := fmt.Fprintf(rc.Out.Machine, "%s: not found in the dictionary\n", rec.Query)
		return err
	}
	_, err := fmt.Fprintf(rc.Out.Machine, "%s %s %s VR=%s VM=%s — %s\n",
		rec.Tag, rec.Keyword, rec.Query, rec.VR, rec.VM, rec.Name)
	return err
}
