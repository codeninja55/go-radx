package command

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/logging"
)

// maxStoreMaxPDU is the explicit cap on --max-pdu for store: a high-throughput batch can ask for
// larger P-DATA-TF PDUs than the canonical default, but not without bound (docs/reference/cli.md
// store). A request above the cap is a usage error, not a silent clamp.
const maxStoreMaxPDU uint32 = 131072

// StoreCmd sends one or more DICOM objects to a remote Storage SCP over negotiated associations
// (dcmtk's storescu). Any failed transfer makes the command exit non-zero; --continue-on-error
// changes only whether the batch keeps going, never the final status (docs/reference/cli.md store,
// closing RADX-003).
type StoreCmd struct {
	Paths []string `arg:"" name:"path" help:"DICOM files or directories to send."`

	Host      string        `name:"host" required:"" env:"RADX_HOST" help:"Remote host."`
	Port      int           `name:"port" default:"11112" env:"RADX_PORT" help:"Remote port."`
	CalledAE  string        `name:"called-ae" default:"${default_called_ae}" env:"RADX_CALLED_AE" help:"Called AE Title (the remote AE)."`
	CallingAE string        `name:"calling-ae" default:"${default_calling_ae}" env:"RADX_CALLING_AE" help:"Calling AE Title (this client)."`
	Recursive bool          `short:"R" name:"recursive" help:"Descend into directories for *.dcm files."`
	Timeout   time.Duration `name:"timeout" default:"5m" env:"RADX_TIMEOUT" help:"Operation timeout."`
	MaxPDU    uint32        `name:"max-pdu" default:"${default_max_pdu}" env:"RADX_MAX_PDU" help:"Maximum PDU length in bytes (cap 131072)."`
	Workers   int           `name:"workers" default:"4" help:"Concurrent worker associations (1-128)."`

	TranscodeTo     string `name:"transcode-to" help:"Transcode to this transfer syntax before sending (default: send as stored)."`
	ContinueOnError bool   `name:"continue-on-error" help:"Keep processing after a failed object (final exit still non-zero)."`
}

// storeResult is the canonical per-object machine shape (one JSON Line per file). It names the
// file, the SOP Instance UID it carried, the outcome status, and — on a non-success — the named
// DIMSE status or a structural error. It carries no patient values.
type storeResult struct {
	File           string `json:"file"`
	SOPInstanceUID string `json:"sop_instance_uid,omitempty"`
	Status         string `json:"status"`
	DIMSEStatus    string `json:"dimse_status,omitempty"`
	Error          string `json:"error,omitempty"`
}

// storeSummary is the trailing JSON Line of a store run: the per-outcome tally so a consumer can
// branch on the batch result without re-counting the per-file lines.
type storeSummary struct {
	Status    string `json:"status"`
	Total     int    `json:"total"`
	Succeeded int    `json:"succeeded"`
	Failed    int    `json:"failed"`
}

// Run resolves the input files, opens a pool of worker associations, and sends each object,
// emitting one result line per file followed by a summary line. A single failed transfer yields a
// non-zero exit; --continue-on-error only governs whether the batch stops at the first failure.
func (c *StoreCmd) Run(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return &exitcode.UsageErr{Message: "store does not support --format csv; use human or json"}
	}
	if c.MaxPDU > maxStoreMaxPDU {
		return &exitcode.UsageErr{Message: fmt.Sprintf("--max-pdu %d exceeds the store cap of %d bytes", c.MaxPDU, maxStoreMaxPDU)}
	}
	if c.Workers < 1 || c.Workers > 128 {
		return &exitcode.UsageErr{Message: fmt.Sprintf("--workers %d out of range (1-128)", c.Workers)}
	}
	// Transcode-on-store is an explicit data-integrity operation that depends on the optional CGo
	// encode codecs. The library exposes whole-dataset re-encoding only at the pixel-data layer, not
	// through Store, so a --transcode-to request fails closed rather than silently sending as stored
	// (RADX-011: medical-image fidelity is never altered by a silent default, nor a silent no-op).
	if c.TranscodeTo != "" {
		return &exitcode.UsageErr{Message: "--transcode-to is not supported by this build (encode-side transcoding requires the CGo codec build); objects are sent as stored"}
	}

	calling, called, err := parseAETitles(c.CallingAE, c.CalledAE)
	if err != nil {
		return err
	}

	files, err := resolveDICOMPaths(c.Paths, c.Recursive)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return &exitcode.UsageErr{Message: "no DICOM files to send (did you mean -R/--recursive for a directory?)"}
	}

	log := logging.FromContext(rc.Ctx)
	log.Debug("store: starting batch",
		zap.String("host", c.Host),
		zap.Int("port", c.Port),
		zap.String("called_ae", string(called)),
		zap.Int("files", len(files)),
		zap.Int("workers", c.Workers),
	)

	results := c.transferAll(rc.Ctx, files, calling, called)

	succeeded, failed := 0, 0
	for _, r := range results {
		if emitErr := c.emit(rc, r); emitErr != nil {
			return emitErr
		}
		if r.Status == "success" {
			succeeded++
		} else {
			failed++
		}
	}

	summary := storeSummary{Total: succeeded + failed, Succeeded: succeeded, Failed: failed}
	if failed == 0 {
		summary.Status = "success"
	} else {
		summary.Status = "failure"
	}
	if emitErr := c.emitSummary(rc, summary); emitErr != nil {
		return emitErr
	}

	if failed > 0 {
		// Honest-failure: a partial batch is a failure. --continue-on-error governed only whether the
		// batch stopped early, never the final status (RADX-003).
		return &exitcode.UsageErr{Message: fmt.Sprintf("%d of %d objects failed to store", failed, summary.Total)}
	}
	return nil
}

// transferAll fans the files across the worker pool. Each worker owns its own association for its
// whole slice of work, so a reconnect replaces only that worker's client (RADX-009). Order of the
// returned results follows input order so the per-file output is deterministic for golden tests.
func (c *StoreCmd) transferAll(ctx context.Context, files []string, calling, called dimse.AETitle) []storeResult {
	results := make([]storeResult, len(files))

	type job struct {
		idx  int
		path string
	}
	jobs := make(chan job)

	// stopAll is closed when a worker hits a failure under the default (fail-fast on the batch
	// continuing) policy, so the remaining files are reported as skipped rather than sent.
	var stopOnce sync.Once
	stopAll := make(chan struct{})

	var wg sync.WaitGroup
	workers := c.Workers
	if workers > len(files) {
		workers = len(files)
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				select {
				case <-stopAll:
					results[j.idx] = storeResult{File: j.path, Status: "skipped", Error: "skipped after an earlier failure"}
					continue
				default:
				}
				r := c.transferOne(ctx, j.path, calling, called)
				results[j.idx] = r
				if r.Status != "success" && !c.ContinueOnError {
					stopOnce.Do(func() { close(stopAll) })
				}
			}
		}()
	}

	for i, path := range files {
		jobs <- job{idx: i, path: path}
	}
	close(jobs)
	wg.Wait()
	return results
}

// transferOne reads one file, opens a Storage association, and sends the object. A read or parse
// failure is reported per-file; an association or transport fault is a network failure; a
// non-success C-STORE status is promoted to a failure outcome so the batch never reads a peer's
// "no" as a success (PRD §9.2). The association is released cleanly on every path.
func (c *StoreCmd) transferOne(ctx context.Context, path string, calling, called dimse.AETitle) storeResult {
	f, err := dicom.ReadFile(path)
	if err != nil {
		return storeResult{File: path, Status: "failure", Error: structuralError(err)}
	}
	sopInstance, _ := f.DataSet.GetString(dicom.TagSOPInstanceUID)

	ae, err := dimse.NewAE(calling,
		dimse.WithMaxPDULength(dimse.MaxPDULength(c.MaxPDU)),
		dimse.WithACSETimeout(c.Timeout),
		dimse.WithDIMSETimeout(c.Timeout),
		dimse.WithConnectionTimeout(c.Timeout),
	)
	if err != nil {
		return storeResult{File: path, SOPInstanceUID: sopInstance, Status: "failure", Error: err.Error()}
	}

	assoc, err := ae.Associate(ctx, hostPort(c.Host, c.Port), called, dimse.StorageContexts())
	if err != nil {
		return storeResult{File: path, SOPInstanceUID: sopInstance, Status: "failure", Error: err.Error()}
	}
	defer func() { _ = assoc.Release(ctx) }()

	status, err := assoc.Store(ctx, f.DataSet)
	if err != nil {
		return storeResult{File: path, SOPInstanceUID: sopInstance, Status: "failure", Error: structuralError(err)}
	}
	if !status.IsSuccess() {
		return storeResult{
			File:           path,
			SOPInstanceUID: sopInstance,
			Status:         "failure",
			DIMSEStatus:    status.String(),
		}
	}
	return storeResult{
		File:           path,
		SOPInstanceUID: sopInstance,
		Status:         "success",
		DIMSEStatus:    status.String(),
	}
}

// emit renders one per-file result: a JSON Line in json format, or a human progress line on the
// machine sink. The diagnostic banner/progress contract is honoured — only the payload reaches the
// machine sink.
func (c *StoreCmd) emit(rc *RunContext, r storeResult) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSONLine(r)
	}
	if r.Status == "success" {
		_, err := fmt.Fprintf(rc.Out.Machine, "%s: stored (%s)\n", r.File, r.DIMSEStatus)
		return err
	}
	detail := r.Error
	if detail == "" {
		detail = r.DIMSEStatus
	}
	_, err := fmt.Fprintf(rc.Out.Machine, "%s: %s — %s\n", r.File, r.Status, detail)
	return err
}

// emitSummary renders the trailing tally: a JSON Line in json format, a one-line human total
// otherwise.
func (c *StoreCmd) emitSummary(rc *RunContext, s storeSummary) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSONLine(s)
	}
	_, err := fmt.Fprintf(rc.Out.Machine, "stored %d/%d objects (%d failed)\n", s.Succeeded, s.Total, s.Failed)
	return err
}
