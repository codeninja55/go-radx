package command

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/logging"
)

// MoveCmd retrieves instances to a destination AE with a C-MOVE (dcmtk's movescu). The peer
// resolves --move-destination, opens its own association to it, and C-STOREs each matched instance
// there; the CLI reports the terminal status and the sub-operation counts. A Warning/Failure
// terminal status (a partial-failure retrieve, or a "Move Destination Unknown") is reported
// faithfully and exits 4, never laundered into success (docs/reference/cli.md move; PRD §9.2).
type MoveCmd struct {
	Host            string        `name:"host" required:"" env:"RADX_HOST" help:"Remote host."`
	Port            int           `name:"port" default:"11112" env:"RADX_PORT" help:"Remote port."`
	CalledAE        string        `name:"called-ae" default:"${default_called_ae}" env:"RADX_CALLED_AE" help:"Called AE Title (the remote AE)."`
	CallingAE       string        `name:"calling-ae" default:"${default_calling_ae}" env:"RADX_CALLING_AE" help:"Calling AE Title (this client)."`
	MoveDestination string        `name:"move-destination" required:"" help:"Destination AE Title the SCP C-STOREs matched instances to."`
	Level           string        `name:"level" enum:"PATIENT,STUDY,SERIES,IMAGE" default:"SERIES" help:"Query/Retrieve Level."`
	Match           []string      `name:"match" help:"Identifier match key (key=value); repeat to add keys."`
	Timeout         time.Duration `name:"timeout" default:"5m" env:"RADX_TIMEOUT" help:"Operation timeout."`
	MaxPDU          uint32        `name:"max-pdu" default:"${default_max_pdu}" env:"RADX_MAX_PDU" help:"Maximum PDU length in bytes."`

	TLSFlags tlsFlags `embed:""`
}

// Run opens a C-MOVE association, runs the retrieve to the destination AE, and reports the terminal
// status and counts.
func (c *MoveCmd) Run(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return &exitcode.UsageErr{Message: "move does not support --format csv; use human or json"}
	}
	level, err := dimse.ParseQueryLevel(c.Level)
	if err != nil {
		return &exitcode.UsageErr{Message: fmt.Sprintf("invalid --level: %v", err)}
	}
	calling, called, err := parseAETitles(c.CallingAE, c.CalledAE)
	if err != nil {
		return err
	}
	dest, err := dimse.ParseAETitle(c.MoveDestination)
	if err != nil {
		return &exitcode.UsageErr{Message: fmt.Sprintf("invalid --move-destination: %v", err)}
	}
	identifier, err := buildIdentifier(c.Match)
	if err != nil {
		return err
	}

	log := logging.FromContext(rc.Ctx)
	log.Debug("move: opening retrieve association",
		zap.String("host", c.Host),
		zap.Int("port", c.Port),
		zap.String("called_ae", string(called)),
		zap.String("move_destination", string(dest)),
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

	assoc, err := ae.Associate(rc.Ctx, hostPort(c.Host, c.Port), called, dimse.QueryRetrieveContexts())
	if err != nil {
		return err
	}
	defer func() { _ = assoc.Release(rc.Ctx) }()

	var terminal dimse.Status
	for status := range assoc.Move(rc.Ctx, identifier, level, dest) {
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
	}
	if terminal.IsSuccess() {
		result.Status = "success"
		if emitErr := emitRetrieveResult(rc, result); emitErr != nil {
			return emitErr
		}
		return nil
	}

	result.Status = "failure"
	if emitErr := emitRetrieveResult(rc, result); emitErr != nil {
		return emitErr
	}
	return &exitcode.StatusError{Status: terminal}
}
