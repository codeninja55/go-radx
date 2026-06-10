package command

import (
	"fmt"
	"iter"
	"sort"
	"time"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/logging"
)

// FindCmd queries a remote AE with a C-FIND (dcmtk's findscu), streaming one match per result.
// The streaming multi-response contract mirrors the underlying Association.Find iterator: each
// Pending match is one JSON Line, and a transport fault after the loop (Association.LastError) is a
// network failure (docs/reference/cli.md find). The -W flag switches to the Modality Worklist
// Information Model (findscu -W): it negotiates the worklist context and queries through
// Association.FindWorklist on the SPS-sequence skeleton.
type FindCmd struct {
	Host      string        `name:"host" required:"" env:"RADX_HOST" help:"Remote host."`
	Port      int           `name:"port" default:"11112" env:"RADX_PORT" help:"Remote port."`
	CalledAE  string        `name:"called-ae" default:"${default_called_ae}" env:"RADX_CALLED_AE" help:"Called AE Title (the remote AE)."`
	CallingAE string        `name:"calling-ae" default:"${default_calling_ae}" env:"RADX_CALLING_AE" help:"Calling AE Title (this client)."`
	Level     string        `name:"level" enum:"PATIENT,STUDY,SERIES,IMAGE" default:"STUDY" help:"Query/Retrieve Level."`
	Worklist  bool          `name:"worklist" short:"W" help:"Query the Modality Worklist Information Model (findscu -W). The worklist model is flat, so --level is ignored."`
	Match     []string      `name:"match" help:"Identifier match key (key=value); repeat to add keys."`
	Timeout   time.Duration `name:"timeout" default:"5m" env:"RADX_TIMEOUT" help:"Operation timeout."`
	MaxPDU    uint32        `name:"max-pdu" default:"${default_max_pdu}" env:"RADX_MAX_PDU" help:"Maximum PDU length in bytes."`
}

// findMatch is one C-FIND match: the tag-keyed identifier attributes the SCP returned. The keys
// are the lowercase "GGGG,EEEE" tag form used across the CLI's DICOM-JSON-style output. Match
// values may carry patient identifiers, but a find is an explicit query the operator ran, so the
// returned attributes are shown (the dump posture) and never logged.
type findMatch struct {
	Status     string            `json:"status"`
	Attributes map[string]string `json:"attributes"`
}

// Run opens a query association, runs the C-FIND, and streams matches. A pre-flight or transport
// fault yields a non-zero (network) exit via Association.LastError; a clean run with zero matches
// is a success with no match lines.
func (c *FindCmd) Run(rc *RunContext) error {
	level, err := dimse.ParseQueryLevel(c.Level)
	if err != nil {
		return &exitcode.UsageErr{Message: fmt.Sprintf("invalid --level: %v", err)}
	}
	calling, called, err := parseAETitles(c.CallingAE, c.CalledAE)
	if err != nil {
		return err
	}
	identifier, err := buildIdentifier(c.Match)
	if err != nil {
		return err
	}
	if c.Worklist {
		// A worklist identifier starts from the SPS-sequence skeleton (PS3.4 K.6.1.2): the empty
		// Scheduled Procedure Step Sequence item universal-matches every scheduled step, and the
		// --match keys are applied at the top level (nested sequence match keys are not expressible
		// through --match).
		wl := dimse.NewWorklistQuery()
		for e := range identifier.All() {
			wl.Set(e)
		}
		identifier = wl
	}

	log := logging.FromContext(rc.Ctx)
	log.Debug("find: opening query association",
		zap.String("host", c.Host),
		zap.Int("port", c.Port),
		zap.String("called_ae", string(called)),
		zap.String("level", level.String()),
	)

	ae, err := dimse.NewAE(calling,
		dimse.WithMaxPDULength(dimse.MaxPDULength(c.MaxPDU)),
		dimse.WithACSETimeout(c.Timeout),
		dimse.WithDIMSETimeout(c.Timeout),
		dimse.WithConnectionTimeout(c.Timeout),
	)
	if err != nil {
		return err
	}

	contexts := dimse.QueryRetrieveContexts()
	if c.Worklist {
		contexts = dimse.BasicWorklistContexts()
	}
	assoc, err := ae.Associate(rc.Ctx, hostPort(c.Host, c.Port), called, contexts)
	if err != nil {
		return err
	}
	defer func() { _ = assoc.Release(rc.Ctx) }()

	em := &matchEmitter{out: rc.Out, columns: csvColumnsFor(c.Match)}
	if err := em.start(); err != nil {
		return err
	}

	// FindWorklist targets the flat Modality Worklist model (no Query/Retrieve Level); the levelled
	// Find runs the Patient/Study Root models.
	var query iter.Seq2[dimse.Status, *dicom.DataSet]
	if c.Worklist {
		query = assoc.FindWorklist(rc.Ctx, identifier)
	} else {
		query = assoc.Find(rc.Ctx, identifier, level)
	}

	var terminal dimse.Status
	for status, ds := range query {
		if status.IsPending() {
			if emitErr := em.emit(datasetAttributes(ds)); emitErr != nil {
				return emitErr
			}
			continue
		}
		terminal = status
	}
	if err := em.finish(); err != nil {
		return err
	}

	// A transport or protocol fault that ended the iteration before a clean terminal status is
	// surfaced via LastError (PRD §8.1); a non-success terminal status is a peer "no" we exit 4 on.
	if lastErr := assoc.LastError(); lastErr != nil {
		return lastErr
	}
	if !terminal.IsSuccess() {
		return &exitcode.StatusError{Status: terminal}
	}
	return nil
}

// matchEmitter streams query matches in the resolved format, holding one CSV writer across the
// whole stream so the header and rows share a single writer (csv.NewWriter returns a fresh one
// each call). It is shared by the streaming query commands (find, qido). The csv columns are the
// requested match keys; json emits one JSON Line of the tag-keyed attributes; human emits an
// indented attribute block per match.
type matchEmitter struct {
	out     *cli.Output
	columns []string
	cw      *csvWriterHandle
}

// csvWriterHandle wraps the encoding/csv writer the emitter retains. It is a tiny alias so the
// emitter holds exactly one instance rather than re-creating one per row.
type csvWriterHandle = csvWriter

// start writes the CSV header when in csv mode; it is a no-op for json/human.
func (e *matchEmitter) start() error {
	if e.out.Format != cli.FormatCSV {
		return nil
	}
	e.cw = newCSVWriter(e.out)
	header := append([]string{"status"}, e.columns...)
	return e.cw.write(header)
}

// emit renders one match's tag-keyed attributes in the resolved format.
func (e *matchEmitter) emit(attrs map[string]string) error {
	switch e.out.Format {
	case cli.FormatCSV:
		row := make([]string, 0, len(e.columns)+1)
		row = append(row, "match")
		for _, col := range e.columns {
			t, _ := parseTagSpec(col)
			row = append(row, attrs[tagKey(t)])
		}
		return e.cw.write(row)
	case cli.FormatJSON:
		return e.out.EmitJSONLine(findMatch{Status: "match", Attributes: attrs})
	default:
		if _, err := fmt.Fprintln(e.out.Machine, "match:"); err != nil {
			return err
		}
		for _, key := range sortedStringKeys(attrs) {
			if _, err := fmt.Fprintf(e.out.Machine, "  %s %s\n", key, attrs[key]); err != nil {
				return err
			}
		}
		return nil
	}
}

// finish flushes the CSV writer and surfaces any deferred write error; it is a no-op otherwise.
func (e *matchEmitter) finish() error {
	if e.cw == nil {
		return nil
	}
	return e.cw.flush()
}

// buildIdentifier turns the --match pairs into a C-FIND identifier dataset. Each key resolves to a
// tag (keyword, parenthesised, or bare-hex form); an empty value is a universal return key. An
// unparseable key is a usage error.
func buildIdentifier(matches []string) (*dicom.DataSet, error) {
	ds := dicom.NewDataSet()
	for _, raw := range matches {
		key, value, ok := splitKeyValue(raw)
		if !ok {
			return nil, &exitcode.UsageErr{Message: fmt.Sprintf("invalid --match %q (use key=value)", raw)}
		}
		t, resolvable := parseTagSpec(key)
		if !resolvable {
			return nil, &exitcode.UsageErr{Message: fmt.Sprintf("invalid --match key %q (use a keyword, (GGGG,EEEE), or GGGGEEEE)", key)}
		}
		if value == "" {
			ds.SetEmpty(t)
			continue
		}
		ds.SetString(t, value)
	}
	return ds, nil
}

// csvColumnsFor returns the match keys as the CSV column tags, in input order, so a csv consumer
// gets a stable column set keyed by the attributes it asked to match/return.
func csvColumnsFor(matches []string) []string {
	cols := make([]string, 0, len(matches))
	for _, raw := range matches {
		key, _, ok := splitKeyValue(raw)
		if !ok {
			continue
		}
		cols = append(cols, key)
	}
	return cols
}

// datasetAttributes flattens a dataset into a tag-keyed string map (the lowercase "GGGG,EEEE" key
// form), rendering each value with the shared renderer so the output names structure, never raw
// bytes. Sequences and pixel data are summarised by the renderer.
func datasetAttributes(ds *dicom.DataSet) map[string]string {
	if ds == nil {
		return map[string]string{}
	}
	attrs := make(map[string]string, ds.Len())
	for e := range ds.All() {
		attrs[tagKey(e.Tag)] = renderValue(e)
	}
	return attrs
}

// sortedStringKeys returns the keys of a string map in ascending order for deterministic human
// output.
func sortedStringKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
