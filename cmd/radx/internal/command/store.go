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

// storeJob is one unit of work in the store pool: a file and its index in the input order, so a
// worker can record its result index-aligned for deterministic per-file output.
type storeJob struct {
	idx  int
	path string
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
	// Transcode-on-store is an explicit data-integrity operation (RADX-011: fidelity is
	// never altered by a silent default). The store transport encodes each dataset in the
	// negotiated uncompressed transfer syntax (dimse.DefaultTransferSyntaxes), so the
	// honourable targets are the uncompressed syntaxes — decompress-before-send. An
	// encapsulated or malformed target fails closed here as a usage error rather than
	// surprising the operator per file at encode time.
	if c.TranscodeTo != "" {
		target := dicom.TransferSyntax(c.TranscodeTo)
		if err := dicom.UID(target).Validate(); err != nil {
			return &exitcode.UsageErr{Message: fmt.Sprintf("--transcode-to %q is not a valid transfer syntax UID", c.TranscodeTo)}
		}
		if target.IsEncapsulated() {
			return &exitcode.UsageErr{Message: fmt.Sprintf(
				"--transcode-to %s (%s) cannot be honoured: store sends datasets in the negotiated uncompressed transfer syntax, so only uncompressed targets are supported",
				target.Name(), c.TranscodeTo)}
		}
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

	results, firstErr := c.transferAll(rc.Ctx, files, calling, called)

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
		// batch stopped early, never the final status (RADX-003). Return the first transfer's underlying
		// typed error so exitcode.Classify routes it to its real class — a missing input file to exit 5,
		// a refused association or non-success C-STORE status to exit 4, a malformed object to exit 3 —
		// rather than collapsing every runtime failure into a usage error (exit 2). UsageError is for
		// CLI misuse only; the exit-code taxonomy is the contract operators branch on.
		if firstErr != nil {
			return firstErr
		}
		return &exitcode.UsageErr{Message: fmt.Sprintf("%d of %d objects failed to store", failed, summary.Total)}
	}
	return nil
}

// transferAll fans the files across the worker pool. Each worker opens ONE Storage association and
// sends all of its assigned objects over that held association, releasing it only when its queue
// drains. A study sent with --workers=N therefore negotiates ~N associations, not one per object:
// re-associating per file is heavy ACSE overhead and risks a PACS's association-rate/negotiation
// limits, so the association is amortised across the worker's whole slice of work (RADX-009). A
// reconnect replaces only that worker's association. Order of the returned results follows input
// order so the per-file output is deterministic for golden tests. It also returns the underlying
// typed error of the first failed file in input order (or nil when every transfer succeeded), so Run
// can surface the real failure class through exitcode.Classify rather than a flattened usage error.
// Per-file errors are stored index-aligned and selected after the pool drains, keeping the choice
// deterministic regardless of which worker finished first.
func (c *StoreCmd) transferAll(ctx context.Context, files []string, calling, called dimse.AETitle) ([]storeResult, error) {
	results := make([]storeResult, len(files))
	errs := make([]error, len(files))

	jobs := make(chan storeJob)

	// stopAll is closed when a worker hits a failure under the default (fail-fast on the batch
	// continuing) policy, so the remaining files are reported as skipped rather than sent.
	var stopOnce sync.Once
	stopAll := make(chan struct{})

	record := func(idx int, r storeResult, err error) {
		results[idx] = r
		errs[idx] = err
		if r.Status != "success" && !c.ContinueOnError {
			stopOnce.Do(func() { close(stopAll) })
		}
	}

	var wg sync.WaitGroup
	workers := c.Workers
	if workers > len(files) {
		workers = len(files)
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.runWorker(ctx, calling, called, jobs, stopAll, func(int, int) {}, record)
		}()
	}

	for i, path := range files {
		jobs <- storeJob{idx: i, path: path}
	}
	close(jobs)
	wg.Wait()

	var firstErr error
	for _, err := range errs {
		if err != nil {
			firstErr = err
			break
		}
	}
	return results, firstErr
}

// runWorker drives one worker: it lazily opens a single Storage association on its first sendable
// job and sends every subsequent job over that same held association, releasing it when the worker's
// queue drains or its association fails. A file read or parse error is recorded per-file and does NOT
// tear down the association — the worker skips that file and continues with the next, so a single
// bad object never costs a re-association. Only a transport/association fault (a failed Associate or a
// C-STORE error from the held association) ends the association: its remaining queued jobs are then
// recorded as failures rather than silently dropped, and a fresh association is opened for the next
// job if the batch is still running. The typed error of every failure is preserved through record so
// exitcode.Classify routes the run to its real class.
//
// onSend is invoked with (jobIndex, associationGeneration) immediately before each C-STORE over a
// live association, so a test can observe how many distinct associations a worker opened; production
// passes a no-op.
func (c *StoreCmd) runWorker(
	ctx context.Context,
	calling, called dimse.AETitle,
	jobs <-chan storeJob,
	stopAll <-chan struct{},
	onSend func(idx, generation int),
	record func(idx int, r storeResult, err error),
) {
	var assoc *dimse.Association
	generation := 0
	releaseAssoc := func() {
		if assoc != nil {
			_ = assoc.Release(ctx)
			assoc = nil
		}
	}
	defer releaseAssoc()

	for j := range jobs {
		select {
		case <-stopAll:
			record(j.idx, storeResult{File: j.path, Status: "skipped", Error: "skipped after an earlier failure"}, nil)
			continue
		default:
		}

		// Read and parse before touching the association: a structural fault is a per-file failure that
		// must not tear down the worker's association (RADX-012/013). Skip the file, record its typed
		// error, keep the held association for the next job.
		f, err := dicom.ReadFile(j.path)
		if err != nil {
			record(j.idx, storeResult{File: j.path, Status: "failure", Error: structuralError(err)}, err)
			continue
		}
		if err := prepareForStore(f, dicom.TransferSyntax(c.TranscodeTo)); err != nil {
			record(j.idx, storeResult{File: j.path, Status: "failure", Error: structuralError(err)}, err)
			continue
		}
		sopInstance, _ := f.DataSet.GetString(dicom.TagSOPInstanceUID)

		if assoc == nil {
			a, aerr := c.associate(ctx, calling, called)
			if aerr != nil {
				// The association could not be opened. This file fails on the transport, and because there
				// is no live association the next job will attempt a fresh one.
				record(j.idx, storeResult{File: j.path, SOPInstanceUID: sopInstance, Status: "failure", Error: aerr.Error()}, aerr)
				continue
			}
			assoc = a
			generation++
		}

		onSend(j.idx, generation)
		status, serr := assoc.Store(ctx, f.DataSet)
		if serr != nil {
			// A transport/association fault on the held association: this file fails and the association is
			// no longer usable, so release it and let the next job re-associate.
			record(j.idx, storeResult{File: j.path, SOPInstanceUID: sopInstance, Status: "failure", Error: structuralError(serr)}, serr)
			releaseAssoc()
			continue
		}
		if !status.IsSuccess() {
			// The conversation reached the peer and the peer answered with a non-success terminal status.
			// Promote it to a *exitcode.StatusError so Classify routes the run to NetworkError (exit 4),
			// the class for "the peer reported it could not perform the work" (PRD §9.2). The association
			// itself is intact, so it is kept for the next job.
			record(j.idx, storeResult{
				File:           j.path,
				SOPInstanceUID: sopInstance,
				Status:         "failure",
				DIMSEStatus:    status.String(),
			}, &exitcode.StatusError{Status: status})
			continue
		}
		record(j.idx, storeResult{
			File:           j.path,
			SOPInstanceUID: sopInstance,
			Status:         "success",
			DIMSEStatus:    status.String(),
		}, nil)
	}
}

// prepareForStore makes f sendable over the negotiated uncompressed transfer syntaxes. With a
// --transcode-to target, a compressed object is decoded and re-encoded through the library's
// dataset-level seam (NewPixelData -> Transcode -> SetPixelData), which also reconciles the
// Image Pixel attributes with the decoded bytes (interleaved layout, decoded colour model,
// frame count) and keeps the lossy bookkeeping (0028,2110) for a lossy source; a pixel-less
// object and an already-uncompressed object pass through unchanged, so nothing is silently
// altered. Without a
// target, a compressed pixel-bearing object is a per-file failure naming the flag: the transport
// cannot carry encapsulated pixel data and never silently decompresses it (RADX-011).
func prepareForStore(f *dicom.File, target dicom.TransferSyntax) error {
	if !f.Meta.TransferSyntaxUID.IsEncapsulated() {
		return nil
	}
	if _, ok := f.DataSet.Get(dicom.TagPixelData); !ok {
		return nil
	}
	if target == "" {
		return fmt.Errorf("object is compressed (%s) and the store transport is uncompressed; pass --transcode-to to decompress on send",
			f.Meta.TransferSyntaxUID.Name())
	}
	pd, err := dicom.NewPixelData(f.DataSet, f.Meta.TransferSyntaxUID)
	if err != nil {
		return err
	}
	out, err := dicom.Transcode(pd, target)
	if err != nil {
		return err
	}
	return f.SetPixelData(out)
}

// associate opens one Storage association for a worker. The AE is built per association so each
// worker's client is independent and a reconnect replaces only that worker's association.
func (c *StoreCmd) associate(ctx context.Context, calling, called dimse.AETitle) (*dimse.Association, error) {
	ae, err := dimse.NewAE(calling,
		dimse.WithMaxPDULength(dimse.MaxPDULength(c.MaxPDU)),
		dimse.WithACSETimeout(c.Timeout),
		dimse.WithDIMSETimeout(c.Timeout),
		dimse.WithConnectionTimeout(c.Timeout),
	)
	if err != nil {
		return nil, err
	}
	return ae.Associate(ctx, hostPort(c.Host, c.Port), called, dimse.StorageContexts())
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
