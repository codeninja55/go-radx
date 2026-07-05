package command

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/logging"
	"github.com/codeninja55/go-radx/server"
)

// ServeCmd groups the reference daemons. serve dicomweb wires the server package's DICOMweb role
// over the default filesystem object store and SQLite catalogue; serve dimse wires the DIMSE Q/R
// archive role over the same backends; serve fhir wires the FHIR REST role over the in-memory
// development repository (docs/reference/cli.md serve).
type ServeCmd struct {
	DICOMweb ServeDICOMwebCmd `cmd:"" name:"dicomweb" help:"Serve WADO-RS / STOW-RS / QIDO-RS."`
	DIMSE    ServeDIMSECmd    `cmd:"" name:"dimse" help:"Serve a DIMSE Q/R archive (C-ECHO/C-STORE/C-FIND/C-GET/C-MOVE)."`
	FHIR     ServeFHIRCmd     `cmd:"" name:"fhir" help:"Serve the FHIR REST API (in-memory development repository)."`
}

// ServeFHIRCmd runs the FHIR REST reference daemon: the server package's FHIRRole over the
// in-memory development repository, binding loopback by default. One process serves one FHIR
// release (--release); the repository holds resources in memory only — nothing is persisted, so no
// PHI ever lands on disk — which makes this a development and integration-test daemon, not an
// archive (servers.md; a production deployment supplies its own server.Repository). A non-loopback
// bind requires authentication and is refused otherwise, exactly like serve dicomweb
// (ErrInsecureBind; servers.md "Bind policy").
type ServeFHIRCmd struct {
	Bind            string `name:"bind" default:"127.0.0.1" env:"RADX_BIND" help:"Listen address (loopback by default)."`
	Port            int    `name:"port" default:"8080" help:"Listen port."`
	BasePath        string `name:"base-path" default:"/fhir" help:"FHIR REST base path."`
	Release         string `name:"release" default:"r5" enum:"r4,r5" help:"FHIR release to serve (r4 or r5)."`
	MaxRequestBytes int64  `name:"max-request-bytes" default:"0" help:"Request body cap (0 = library default)."`
}

// Run wires the FHIR role over the in-memory repository and runs the daemon until interrupted. A
// non-loopback bind without authentication is refused with a clear usage error (ErrInsecureBind),
// never a silent unauthenticated exposure.
func (c *ServeFHIRCmd) Run(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return &exitcode.UsageErr{Message: "serve fhir does not support --format csv; use human or json"}
	}

	release := fhir.R5
	if c.Release == "r4" {
		release = fhir.R4
	}
	repo, err := server.NewMemoryRepository(release)
	if err != nil {
		return err
	}

	roleOpts := []server.FHIRRoleOption{
		server.WithFHIRPort(c.Port),
		server.WithFHIRBasePath(c.BasePath),
		server.WithFHIRRelease(release),
	}
	if c.MaxRequestBytes > 0 {
		roleOpts = append(roleOpts, server.WithFHIRMaxRequestBytes(c.MaxRequestBytes))
	}
	role, err := server.NewFHIRRole(repo, roleOpts...)
	if err != nil {
		return err
	}

	log := logging.FromContext(rc.Ctx)
	daemonOpts := []server.Option{
		server.WithLogger(log),
		server.WithFHIR(role),
		server.WithBind(c.Bind),
	}
	if !isLoopbackBind(c.Bind) {
		// A non-loopback bind is an explicit opt-in the daemon refuses without an authenticator
		// (ErrInsecureBind, mapped below); the reference daemon enables AllowAll only on loopback.
		log.Warn("serve fhir: binding a non-loopback address requires authentication",
			zap.String("bind", c.Bind))
	}

	daemon, err := server.New(daemonOpts...)
	if err != nil {
		if errors.Is(err, server.ErrInsecureBind) {
			return &exitcode.UsageErr{Message: "a non-loopback --bind requires authentication; the reference daemon serves loopback only"}
		}
		return err
	}

	sigCtx, stop := signal.NotifyContext(rc.Ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	runErr := make(chan error, 1)
	go func() { runErr <- daemon.Run(sigCtx) }()

	addrs, err := waitForDaemon(daemon, runErr)
	if err != nil {
		return err
	}

	result := serveStartedResult{
		Status:   "listening",
		Bind:     c.Bind,
		BasePath: c.BasePath,
		Addrs:    stringAddrs(addrs),
	}
	if emitErr := c.emit(rc, result); emitErr != nil {
		return emitErr
	}
	log.Info("serve fhir: listening",
		zap.String("base_path", c.BasePath), zap.String("release", c.Release))

	return awaitDaemonStop(sigCtx, runErr, log, "serve fhir")
}

// emit renders the listening result in the resolved format.
func (c *ServeFHIRCmd) emit(rc *RunContext, r serveStartedResult) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSON(r)
	}
	_, err := fmt.Fprintf(rc.Out.Machine, "FHIR daemon listening on %s%s\n", r.Bind, r.BasePath)
	return err
}

// ServeDICOMwebCmd runs the DICOMweb reference daemon over the shared filesystem object store and
// SQLite catalogue, binding loopback by default. A non-loopback bind requires an explicit
// authenticator, surfaced through the server package's ErrInsecureBind as a clear usage error
// (docs/reference/cli.md serve; servers.md "Bind policy"). The catalogue holds PHI, so its path is
// always explicit and never a default.
type ServeDICOMwebCmd struct {
	Bind            string `name:"bind" default:"127.0.0.1" env:"RADX_BIND" help:"Listen address (loopback by default)."`
	Port            int    `name:"port" default:"8042" help:"Listen port."`
	BasePath        string `name:"base-path" default:"/dicom-web" help:"DICOMweb base path."`
	ObjectStore     string `name:"object-store" required:"" help:"Filesystem object-store root."`
	Catalogue       string `name:"catalogue" required:"" help:"SQLite catalogue path (PHI store; never a default path)."`
	MaxRequestBytes int64  `name:"max-request-bytes" default:"0" help:"Request body cap (0 = library default)."`
}

// serveStartedResult is the canonical machine shape emitted once the daemon is bound: the bound
// addresses and the base path. The daemon then blocks serving until interrupted.
type serveStartedResult struct {
	Status   string            `json:"status"`
	Bind     string            `json:"bind"`
	BasePath string            `json:"base_path"`
	Addrs    map[string]string `json:"addrs"`
}

// Run wires the DICOMweb role over the default backends and runs the daemon until interrupted. A
// non-loopback bind without authentication is refused with a clear usage error (ErrInsecureBind),
// never a silent unauthenticated exposure.
func (c *ServeDICOMwebCmd) Run(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return &exitcode.UsageErr{Message: "serve dicomweb does not support --format csv; use human or json"}
	}

	store, err := server.FileStore(c.ObjectStore)
	if err != nil {
		return err
	}
	cat, err := server.SQLiteCatalogue(rc.Ctx, c.Catalogue)
	if err != nil {
		return err
	}
	defer closeCatalogue(cat)
	// The catalogue holds PHI: harden the file to owner-only after the driver creates it (RADX-008).
	// An in-memory or URI-form DSN with no filesystem file is a no-op rather than a startup failure.
	if err := hardenCataloguePath(c.Catalogue); err != nil {
		return err
	}

	roleOpts := []server.DICOMwebRoleOption{
		server.WithDICOMwebPort(c.Port),
		server.WithDICOMwebBasePath(c.BasePath),
	}
	if c.MaxRequestBytes > 0 {
		roleOpts = append(roleOpts, server.WithMaxRequestBytes(c.MaxRequestBytes))
	}
	role, err := server.NewDICOMwebRole(store, cat, roleOpts...)
	if err != nil {
		return err
	}

	log := logging.FromContext(rc.Ctx)
	daemonOpts := []server.Option{
		server.WithLogger(log),
		server.WithDICOMweb(role),
		server.WithBind(c.Bind),
	}
	if !isLoopbackBind(c.Bind) {
		// A non-loopback bind is an explicit opt-in the daemon refuses without an authenticator
		// (ErrInsecureBind, mapped below); the reference daemon enables AllowAll only on loopback.
		log.Warn("serve dicomweb: binding a non-loopback address requires authentication",
			zap.String("bind", c.Bind))
	}

	daemon, err := server.New(daemonOpts...)
	if err != nil {
		// A non-loopback bind without an authenticator surfaces here as ErrInsecureBind; report it as
		// a clear usage error rather than a runtime failure.
		if errors.Is(err, server.ErrInsecureBind) {
			return &exitcode.UsageErr{Message: "a non-loopback --bind requires authentication; the reference daemon serves loopback only"}
		}
		return err
	}

	sigCtx, stop := signal.NotifyContext(rc.Ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	runErr := make(chan error, 1)
	go func() { runErr <- daemon.Run(sigCtx) }()

	addrs, err := waitForDaemon(daemon, runErr)
	if err != nil {
		return err
	}

	result := serveStartedResult{
		Status:   "listening",
		Bind:     c.Bind,
		BasePath: c.BasePath,
		Addrs:    stringAddrs(addrs),
	}
	if emitErr := c.emit(rc, result); emitErr != nil {
		return emitErr
	}
	log.Info("serve dicomweb: listening", zap.String("base_path", c.BasePath))

	return awaitDaemonStop(sigCtx, runErr, log, "serve dicomweb")
}

// awaitDaemonStop blocks after a successful startup until either an interrupt signal arrives or the
// daemon's Run goroutine returns. It selects on BOTH so a post-startup daemon failure (a listener or
// role that dies after reporting ready) is surfaced promptly with its error rather than hanging the
// CLI until a signal: if runErr fires first, that error is returned (and maps to its exit code); if
// the signal fires first, the daemon drains gracefully and Run's terminal error (nil on a clean
// stop) is returned. Either way Run is awaited once, so the daemon is fully drained before the
// command returns.
func awaitDaemonStop(sigCtx context.Context, runErr <-chan error, log *zap.Logger, name string) error {
	select {
	case serveErr := <-runErr:
		// The daemon stopped on its own after reporting ready: a listener or role failure. Surface it
		// rather than blocking on a signal that may never come.
		if serveErr != nil {
			return serveErr
		}
	case <-sigCtx.Done():
		// An interrupt: the signal cancels sigCtx, which wakes the daemon's Run; wait for it to finish
		// draining and surface any terminal error from the graceful shutdown.
		if serveErr := <-runErr; serveErr != nil {
			return serveErr
		}
	}
	log.Info(name + ": stopped")
	return nil
}

// awaitListenerStop blocks after a successful startup for the listener commands (scp, hl7 listen)
// whose server is run via ListenAndServe and stopped with an explicit Shutdown. It selects on BOTH
// the served channel and the signal context, mirroring awaitDaemonStop, so a post-startup listener
// failure (an accept loop that dies after reporting ready) is surfaced promptly with its error rather
// than hanging the CLI until a signal: if served fires first, that error is returned (and maps to its
// exit code); if the signal fires first, shutdown drains in-flight work against a fresh background
// context and the serve goroutine's terminal error is returned. Either way served is awaited once, so
// the server is fully drained before the command returns. A nil return is a clean stop; the caller
// logs its own stop message (with role-specific counters) afterwards.
func awaitListenerStop(sigCtx context.Context, served <-chan error, shutdown func(context.Context) error) error {
	select {
	case serveErr := <-served:
		// The server stopped on its own after reporting ready: a listener failure. Surface it rather
		// than blocking on a signal that may never come.
		return serveErr
	case <-sigCtx.Done():
		// An interrupt: drain in-flight associations against a background context the signal has not
		// cancelled, then collect the serve goroutine's terminal error.
		shutdownCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if shutErr := shutdown(shutdownCtx); shutErr != nil {
			return shutErr
		}
		return <-served
	}
}

// emit renders the listening result in the resolved format.
func (c *ServeDICOMwebCmd) emit(rc *RunContext, r serveStartedResult) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSON(r)
	}
	_, err := fmt.Fprintf(rc.Out.Machine, "DICOMweb daemon listening on %s%s\n", r.Bind, r.BasePath)
	return err
}

// waitForDaemon blocks until the daemon publishes its bound addresses or its Run goroutine returns
// an early startup error.
func waitForDaemon(daemon *server.Daemon, runErr <-chan error) (map[string]net.Addr, error) {
	for {
		if addrs := daemon.Addrs(); len(addrs) > 0 {
			return addrs, nil
		}
		select {
		case err := <-runErr:
			if err != nil {
				return nil, err
			}
			return nil, nil
		default:
		}
	}
}

// stringAddrs renders a role-keyed address map to strings for the machine output.
func stringAddrs(addrs map[string]net.Addr) map[string]string {
	out := make(map[string]string, len(addrs))
	for role, addr := range addrs {
		if addr != nil {
			out[role] = addr.String()
		}
	}
	return out
}
