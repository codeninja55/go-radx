package command

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/logging"
)

// GetCmd retrieves instances over the same association with a C-GET (dcmtk's getscu). The peer
// C-STOREs each matched instance back to this requestor as a sub-operation, so the CLI acts as the
// Storage SCP for those sub-operations: each received object is written to --output-dir. A
// partial-failure retrieve surfaces as a Warning/Failure terminal status, never laundered into
// success (docs/reference/cli.md get; PRD §9.2).
type GetCmd struct {
	Host      string        `name:"host" required:"" env:"RADX_HOST" help:"Remote host."`
	Port      int           `name:"port" default:"11112" env:"RADX_PORT" help:"Remote port."`
	CalledAE  string        `name:"called-ae" default:"${default_called_ae}" env:"RADX_CALLED_AE" help:"Called AE Title (the remote AE)."`
	CallingAE string        `name:"calling-ae" default:"${default_calling_ae}" env:"RADX_CALLING_AE" help:"Calling AE Title (this client)."`
	Level     string        `name:"level" enum:"PATIENT,STUDY,SERIES,IMAGE" default:"SERIES" help:"Query/Retrieve Level."`
	Match     []string      `name:"match" help:"Identifier match key (key=value); repeat to add keys."`
	OutputDir string        `name:"output-dir" default:"./dicom-received" help:"Where to write retrieved objects."`
	Timeout   time.Duration `name:"timeout" default:"5m" env:"RADX_TIMEOUT" help:"Operation timeout."`
	MaxPDU    uint32        `name:"max-pdu" default:"${default_max_pdu}" env:"RADX_MAX_PDU" help:"Maximum PDU length in bytes."`

	TLSFlags tlsFlags `embed:""`
}

// retrieveResult is the canonical machine shape for get/move: the terminal status, the
// sub-operation counts the peer reported, and (for get) the number of instances written. It
// carries no PHI — counts and a named status only.
type retrieveResult struct {
	Status      string `json:"status"`
	DIMSEStatus string `json:"dimse_status"`
	Completed   uint16 `json:"completed"`
	Failed      uint16 `json:"failed"`
	Warning     uint16 `json:"warning"`
	Stored      int    `json:"stored,omitempty"`
}

// Run opens a C-GET association proposing the Storage SCP role for the validated Storage classes,
// runs the retrieve, writes each received instance, and reports the terminal status and counts. A
// non-success terminal status exits 4; a transport fault (LastError) propagates as a network error.
func (c *GetCmd) Run(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return &exitcode.UsageErr{Message: "get does not support --format csv; use human or json"}
	}
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

	sink, err := newFileStoreSink(c.OutputDir)
	if err != nil {
		return err
	}

	log := logging.FromContext(rc.Ctx)
	log.Debug("get: opening retrieve association",
		zap.String("host", c.Host),
		zap.Int("port", c.Port),
		zap.String("called_ae", string(called)),
		zap.String("level", level.String()),
	)

	// The TLS material is loaded fail-closed before any dial; a nil config keeps plaintext, and a
	// disabled-verification run is warned loudly.
	tlsCfg, err := c.TLSFlags.resolveClientTLS(log)
	if err != nil {
		return err
	}
	ae, err := dimse.NewAE(calling, scuAEOptions(c.Timeout, c.MaxPDU, tlsCfg)...)
	if err != nil {
		return err
	}

	// A C-GET needs the GET model context to carry the request and Storage contexts (with the
	// Storage SCP role granted) for the sub-operation C-STOREs that arrive back on this association.
	assoc, err := ae.Associate(rc.Ctx, hostPort(c.Host, c.Port), called,
		dimse.QueryRetrieveWithStorageContexts(), storageSCPRoles()...)
	if err != nil {
		return err
	}
	defer func() { _ = assoc.Release(rc.Ctx) }()

	var terminal dimse.Status
	for status := range assoc.Get(rc.Ctx, identifier, level, sink) {
		terminal = status
	}
	if lastErr := assoc.LastError(); lastErr != nil {
		return lastErr
	}

	counts := assoc.SubOperationCounts()
	result := retrieveResult{
		DIMSEStatus: terminal.String(),
		Completed:   counts.Completed,
		Failed:      counts.Failed,
		Warning:     counts.Warning,
		Stored:      sink.count(),
	}
	if terminal.IsSuccess() {
		result.Status = "success"
		if emitErr := emitRetrieveResult(rc, result); emitErr != nil {
			return emitErr
		}
		return nil
	}

	// A Warning/Failure terminal status is a peer "no" or partial failure: report it then exit 4.
	result.Status = "failure"
	if emitErr := emitRetrieveResult(rc, result); emitErr != nil {
		return emitErr
	}
	return &exitcode.StatusError{Status: terminal}
}

// storageSCPRoles proposes the Storage SCP role for every Storage SOP Class in the validated
// Storage preset, so the C-GET SCP is granted permission to C-STORE matched instances back to this
// requestor. The classes are read from StorageContexts (their exported AbstractSyntax) rather than
// duplicated here, so the role proposal always tracks the preset.
func storageSCPRoles() []dimse.AssociateOption {
	contexts := dimse.StorageContexts()
	opts := make([]dimse.AssociateOption, 0, len(contexts))
	for _, pc := range contexts {
		opts = append(opts, dimse.WithRoleSelection(dimse.RoleSelection{
			SOPClassUID: pc.AbstractSyntax,
			SCURole:     true,
			SCPRole:     true,
		}))
	}
	return opts
}

// fileStoreSink is the C-GET sub-operation StoreHandler: it writes each received instance to a
// directory laid out by Study/Series/SOP UID, returning a store-success status only after the file
// is durably written (PRD §9.2 honest-failure). It counts the instances written for the result.
type fileStoreSink struct {
	root  string
	saved int
}

// newFileStoreSink prepares the output directory for the retrieve sink. A directory that cannot be
// created is a file-I/O fault (exit 5).
func newFileStoreSink(root string) (*fileStoreSink, error) {
	if err := ensureDir(root); err != nil {
		return nil, err
	}
	return &fileStoreSink{root: root}, nil
}

// Store writes one received dataset and reports the C-STORE status. A write failure is reported as
// a processing-failure status (0xA700) so the peer's sub-operation count reflects it, never a
// success on an object that did not land.
func (s *fileStoreSink) Store(_ context.Context, ds *dicom.DataSet, info dimse.OpInfo) dimse.Status {
	if err := writeReceivedInstance(s.root, ds, info.TransferSyntax); err != nil {
		return dimse.NewStatus(0xA700, dimse.ServiceClassStorage)
	}
	s.saved++
	return dimse.StatusStoreSuccess
}

// count reports how many instances the sink wrote.
func (s *fileStoreSink) count() int { return s.saved }

// emitRetrieveResult renders the shared get/move result in the resolved format.
func emitRetrieveResult(rc *RunContext, r retrieveResult) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSON(r)
	}
	_, err := fmt.Fprintf(rc.Out.Machine,
		"retrieve %s (%s): completed=%d failed=%d warning=%d stored=%d\n",
		r.Status, r.DIMSEStatus, r.Completed, r.Failed, r.Warning, r.Stored)
	return err
}
