package command

import (
	"context"
	"fmt"
	"net"
	"os/signal"
	"sync/atomic"
	"syscall"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/logging"
)

// ScpCmd runs a Storage/Verification SCP (dcmtk's storescp), accepting C-STORE (and, by default,
// C-ECHO) and writing received objects to a directory. The bind defaults to loopback; a
// non-loopback bind is an explicit, logged opt-in (docs/reference/cli.md scp, closing RADX-017).
type ScpCmd struct {
	Bind       string `name:"bind" default:"127.0.0.1" env:"RADX_BIND" help:"Listen address (loopback by default)."`
	Port       int    `name:"port" default:"11112" help:"Listen port."`
	AET        string `name:"aet" default:"RADX-SCP" help:"This SCP's AE Title."`
	OutputDir  string `name:"output-dir" default:"./dicom-received" help:"Where to write received objects."`
	AcceptEcho bool   `name:"accept-echo" default:"true" negatable:"" help:"Accept C-ECHO."`
	MaxPDU     uint32 `name:"max-pdu" default:"${default_max_pdu}" env:"RADX_MAX_PDU" help:"Maximum PDU length in bytes."`
	MaxConns   int    `name:"max-conns" default:"10" help:"Maximum concurrent associations."`

	TLSCert string `name:"tls-cert" help:"PEM server certificate; with --tls-key, the listener terminates TLS (1.2+)."`
	TLSKey  string `name:"tls-key" help:"PEM private key for --tls-cert."`
}

// scpStartedResult is the canonical machine shape emitted once the SCP has bound: the bound address
// and AE title, so a json consumer (or a test) can learn where the server is listening. The SCP
// then blocks serving until interrupted; per-object outcomes are logged to stderr, never stdout.
type scpStartedResult struct {
	Status   string `json:"status"`
	Bind     string `json:"bind"`
	Port     int    `json:"port"`
	AET      string `json:"aet"`
	Output   string `json:"output_dir"`
	Loopback bool   `json:"loopback"`
}

// scpHandler answers inbound C-ECHO and C-STORE: it writes each stored object to the output
// directory (with sender-controlled UIDs validated against path traversal) and counts the
// received objects. It implements both EchoHandler and StoreHandler.
type scpHandler struct {
	root       string
	acceptEcho bool
	log        *zap.Logger
	received   atomic.Int64
}

// Echo answers a C-ECHO. When --accept-echo is off the SCP reports the verification SOP class is
// not supported, so a peer sees an explicit refusal rather than a silent accept.
func (h *scpHandler) Echo(_ context.Context, _ dimse.OpInfo) dimse.Status {
	if !h.acceptEcho {
		return dimse.NewStatus(0x0122, dimse.ServiceClassVerification) // SOP Class Not Supported
	}
	return dimse.StatusEchoSuccess
}

// Store writes one received object and reports the C-STORE status. A write failure is a
// processing-failure status (never a success on an object that did not land); the per-object log
// names protocol identifiers only (no PHI).
func (h *scpHandler) Store(_ context.Context, ds *dicom.DataSet, info dimse.OpInfo) dimse.Status {
	if err := writeReceivedInstance(h.root, ds, info.TransferSyntax); err != nil {
		h.log.Warn("scp: failed to write received object",
			zap.String("calling_ae", string(info.CallingAETitle)),
			zap.String("sop_class", string(info.SOPClassUID)),
			zap.Error(err),
		)
		return dimse.NewStatus(0xA700, dimse.ServiceClassStorage)
	}
	h.received.Add(1)
	h.log.Info("scp: stored object",
		zap.String("calling_ae", string(info.CallingAETitle)),
		zap.String("sop_class", string(info.SOPClassUID)),
	)
	return dimse.StatusStoreSuccess
}

// Run binds the SCP and serves until SIGINT/SIGTERM, draining in-flight associations on shutdown.
// A non-loopback bind is logged as an explicit opt-in; the bind itself happens through the dimse
// server, which also defaults a hostless address to loopback.
func (c *ScpCmd) Run(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return &exitcode.UsageErr{Message: "scp does not support --format csv; use human or json"}
	}
	aet, err := dimse.ParseAETitle(c.AET)
	if err != nil {
		return &exitcode.UsageErr{Message: fmt.Sprintf("invalid --aet: %v", err)}
	}
	if err := ensureDir(c.OutputDir); err != nil {
		return err
	}

	log := logging.FromContext(rc.Ctx)
	loopback := isLoopbackBind(c.Bind)
	if !loopback {
		log.Warn("scp: binding a non-loopback address; the server is reachable from the network",
			zap.String("bind", c.Bind))
	}

	aeOpts := []dimse.AEOption{dimse.WithMaxPDULength(dimse.MaxPDULength(c.MaxPDU))}
	// Both halves of the certificate pair are required together; the material is loaded
	// fail-closed at startup, before any bind, and its paths are never logged at info level.
	if (c.TLSCert == "") != (c.TLSKey == "") {
		return &exitcode.UsageErr{Message: "--tls-cert and --tls-key must be provided together"}
	}
	if c.TLSCert != "" {
		tlsCfg, err := serverTLSConfig(c.TLSCert, c.TLSKey)
		if err != nil {
			return err
		}
		aeOpts = append(aeOpts, dimse.WithTLS(tlsCfg))
		log.Info("scp: TLS listener enabled (TLS 1.2+)")
	}
	ae, err := dimse.NewAE(aet, aeOpts...)
	if err != nil {
		return err
	}

	handler := &scpHandler{root: c.OutputDir, acceptEcho: c.AcceptEcho, log: log}
	contexts := append(dimse.StorageContexts(), dimse.VerificationContexts()...)
	srv := dimse.NewServer(ae, contexts, handler, dimse.WithMaxAssociations(c.MaxConns))

	// Serve until an OS interrupt; the signal context drives a graceful drain.
	sigCtx, stop := signal.NotifyContext(rc.Ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(sigCtx, hostPort(c.Bind, c.Port)) }()

	// Wait for the bind to publish its address before declaring "started" so the machine output
	// reports the real (possibly port-0-resolved) address.
	addr, err := waitForBind(srv, served)
	if err != nil {
		return err
	}

	result := scpStartedResult{
		Status:   "listening",
		Bind:     c.Bind,
		Port:     addrPort(addr, c.Port),
		AET:      string(aet),
		Output:   c.OutputDir,
		Loopback: loopback,
	}
	if emitErr := c.emit(rc, result); emitErr != nil {
		return emitErr
	}
	log.Info("scp: listening", zap.String("addr", addr.String()), zap.String("aet", string(aet)))

	// Wait on BOTH the serve goroutine and the signal: a post-startup accept-loop failure returns its
	// error promptly instead of hanging until an interrupt, while a signal drains the server gracefully.
	if err := awaitListenerStop(sigCtx, served, srv.Shutdown); err != nil {
		return err
	}
	log.Info("scp: stopped", zap.Int64("received", handler.received.Load()))
	return nil
}

// emit renders the listening result in the resolved format.
func (c *ScpCmd) emit(rc *RunContext, r scpStartedResult) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSON(r)
	}
	_, err := fmt.Fprintf(rc.Out.Machine, "SCP %s listening on %s:%d, writing to %s\n",
		r.AET, r.Bind, r.Port, r.Output)
	return err
}

// waitForBind blocks until the server publishes its bound address or the serve goroutine returns
// an early bind error. It is the synchronous "the server is up" handshake the CLI reports against.
func waitForBind(srv *dimse.Server, served <-chan error) (net.Addr, error) {
	for {
		if addr := srv.Addr(); addr != nil {
			return addr, nil
		}
		select {
		case err := <-served:
			if err != nil {
				return nil, err
			}
			// The server returned nil before binding (an immediate shutdown); treat the absence of an
			// address as a clean, if unusual, stop.
			return nil, nil
		default:
		}
	}
}

// addrPort returns the resolved TCP port from a bound address, falling back to the configured port
// when the address is not a TCP address (it always is here, but the fallback keeps it total).
func addrPort(addr net.Addr, fallback int) int {
	if tcp, ok := addr.(*net.TCPAddr); ok {
		return tcp.Port
	}
	return fallback
}

// isLoopbackBind reports whether bind names a loopback host, so a non-loopback bind can be logged
// as an explicit opt-in. An empty or hostless bind is loopback (the server defaults it so).
func isLoopbackBind(bind string) bool {
	if bind == "" {
		return true
	}
	ip := net.ParseIP(bind)
	if ip == nil {
		// A hostname (not an IP literal): treat the conventional loopback names as loopback and
		// anything else as a potential network bind, erring toward warning.
		return bind == "localhost"
	}
	return ip.IsLoopback()
}
