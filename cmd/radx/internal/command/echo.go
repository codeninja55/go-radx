package command

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/logging"
)

// EchoCmd verifies DICOM connectivity by opening an association to a remote AE, sending a
// C-ECHO (Verification), and reporting the result (dcmtk's echoscu). A refused connection,
// a rejected or aborted association, or a non-success C-ECHO status exits 4
// (docs/reference/cli.md echo).
type EchoCmd struct {
	Host string `arg:"" name:"host" help:"Remote host." env:"RADX_HOST"`
	Port int    `arg:"" name:"port" help:"Remote port." env:"RADX_PORT"`

	CalledAE  string        `name:"called-ae" default:"${default_called_ae}" env:"RADX_CALLED_AE" help:"Called AE Title (the remote AE)."`
	CallingAE string        `name:"calling-ae" default:"${default_calling_ae}" env:"RADX_CALLING_AE" help:"Calling AE Title (this client)."`
	Timeout   time.Duration `name:"timeout" default:"30s" env:"RADX_TIMEOUT" help:"Connection and operation timeout."`
	MaxPDU    uint32        `name:"max-pdu" default:"${default_max_pdu}" env:"RADX_MAX_PDU" help:"Maximum PDU length in bytes."`
}

// echoResult is the canonical machine shape for echo: a single JSON object with a stable
// status field, so automation can branch on outcome (docs/reference/cli.md "One canonical
// machine shape per command"). It carries no PHI — host, AE title, and timing only.
type echoResult struct {
	Status    string `json:"status"`
	ElapsedMS int64  `json:"elapsed_ms"`
	CalledAE  string `json:"called_ae"`
	CallingAE string `json:"calling_ae"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	// DIMSEStatus is the C-ECHO response status rendered by name (never raw hex), present on
	// both success and a non-success outcome so a consumer sees what the peer reported.
	DIMSEStatus string `json:"dimse_status"`
}

// Run opens the association, issues the C-ECHO, and emits the result. The AE titles are
// validated before any dial (an over-long title is a usage-class fault), the association is
// opened and released cleanly, and a non-success C-ECHO status is promoted to a *StatusError
// so the command exits 4 rather than reporting success on a peer that said no.
func (c *EchoCmd) Run(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return &exitcode.UsageErr{Message: "echo does not support --format csv; use human or json"}
	}

	calling, err := dimse.ParseAETitle(c.CallingAE)
	if err != nil {
		return &exitcode.UsageErr{Message: fmt.Sprintf("invalid --calling-ae: %v", err)}
	}
	called, err := dimse.ParseAETitle(c.CalledAE)
	if err != nil {
		return &exitcode.UsageErr{Message: fmt.Sprintf("invalid --called-ae: %v", err)}
	}

	log := logging.FromContext(rc.Ctx)
	// Structural diagnostics only: host, port, AE titles are operational identifiers, not PHI.
	log.Debug("echo: opening association",
		zap.String("host", c.Host),
		zap.Int("port", c.Port),
		zap.String("called_ae", string(called)),
		zap.String("calling_ae", string(calling)),
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

	addr := net.JoinHostPort(c.Host, strconv.Itoa(c.Port))

	ctx := rc.Ctx
	start := time.Now()
	assoc, err := ae.Associate(ctx, addr, called, dimse.VerificationContexts())
	if err != nil {
		return err
	}
	defer func() { _ = assoc.Release(ctx) }()

	status, err := assoc.Echo(ctx)
	elapsed := time.Since(start)
	if err != nil {
		return err
	}

	result := echoResult{
		ElapsedMS:   elapsed.Milliseconds(),
		CalledAE:    string(called),
		CallingAE:   string(calling),
		Host:        c.Host,
		Port:        c.Port,
		DIMSEStatus: status.String(),
	}

	if !status.IsSuccess() {
		// The conversation succeeded; the peer answered and said no. Emit the machine shape
		// with a failure status (so json consumers see the outcome) THEN return the typed
		// status error so the command exits 4 — never a zero exit on a failed verification.
		result.Status = "failure"
		if emitErr := c.emit(rc, result); emitErr != nil {
			return emitErr
		}
		return &exitcode.StatusError{Status: status}
	}

	result.Status = "success"
	return c.emit(rc, result)
}

// emit renders the echo result in the resolved format: a one-line human summary on stdout, or
// the canonical JSON object. Diagnostics never enter this path — only the payload reaches the
// machine sink.
func (c *EchoCmd) emit(rc *RunContext, result echoResult) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSON(result)
	}
	_, err := fmt.Fprintf(rc.Out.Machine, "C-ECHO to %s@%s:%d: %s (%s, %dms)\n",
		result.CalledAE, result.Host, result.Port, result.Status, result.DIMSEStatus, result.ElapsedMS)
	return err
}
