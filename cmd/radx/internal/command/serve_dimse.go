package command

import (
	"errors"
	"fmt"
	"net"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/logging"
	"github.com/codeninja55/go-radx/server"
)

// ServeDIMSECmd runs the DIMSE Q/R archive reference daemon (dcmtk's dcmqrscp): the server
// package's DIMSERole over the shared filesystem object store and SQLite catalogue, serving
// C-ECHO, C-STORE, C-FIND, C-GET, and C-MOVE on one AE. C-MOVE destinations follow dcmqrscp's
// known-AE model: a static table from repeatable --move-destination flags; a destination outside
// it is answered with the 0xA801 "Move Destination Unknown" status. The bind defaults to
// loopback; a non-loopback bind requires authentication and is refused otherwise, exactly like
// serve dicomweb and serve fhir (ErrInsecureBind; servers.md "Bind policy"). The catalogue holds
// PHI, so its path is always explicit and never a default.
type ServeDIMSECmd struct {
	Bind             string   `name:"bind" default:"127.0.0.1" env:"RADX_BIND" help:"Listen address (loopback by default)."`
	Port             int      `name:"port" default:"11112" help:"Listen port."`
	AET              string   `name:"aet" default:"RADX-SCP" help:"This SCP's AE Title."`
	ObjectStore      string   `name:"object-store" required:"" help:"Filesystem object-store root."`
	Catalogue        string   `name:"catalogue" required:"" help:"SQLite catalogue path (PHI store; never a default path)."`
	MaxConns         int      `name:"max-conns" default:"10" help:"Maximum concurrent associations."`
	MoveDestinations []string `name:"move-destination" help:"Known C-MOVE destination as AET=host:port; repeat to add more. Unknown destinations are refused (0xA801)."`
}

// serveDIMSEStartedResult is the canonical machine shape emitted once the daemon is bound: the
// bound addresses, the AE title, and the count of configured move destinations (identifiers
// only, no PHI). The daemon then blocks serving until interrupted.
type serveDIMSEStartedResult struct {
	Status           string            `json:"status"`
	Bind             string            `json:"bind"`
	AET              string            `json:"aet"`
	MoveDestinations int               `json:"move_destinations"`
	Addrs            map[string]string `json:"addrs"`
}

// Run wires the DIMSE role over the shared backends and runs the daemon until interrupted. A
// non-loopback bind without authentication is refused with a clear usage error (ErrInsecureBind),
// never a silent unauthenticated exposure; an invalid --move-destination is a usage error before
// any bind.
func (c *ServeDIMSECmd) Run(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return &exitcode.UsageErr{Message: "serve dimse does not support --format csv; use human or json"}
	}
	aet, err := dimse.ParseAETitle(c.AET)
	if err != nil {
		return &exitcode.UsageErr{Message: fmt.Sprintf("invalid --aet: %v", err)}
	}
	dests, err := parseMoveDestinations(c.MoveDestinations)
	if err != nil {
		return err
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

	// serve dimse is the Q/R archive daemon (dcmqrscp), so it explicitly mounts the C-GET/C-MOVE
	// retrieve capability — the opt-in the library role requires so an embedder does not gain
	// archive-wide retrieve by accident.
	roleOpts := []server.DIMSERoleOption{server.WithDIMSEPort(c.Port), server.WithDIMSERetrieve()}
	if c.MaxConns > 0 {
		roleOpts = append(roleOpts, server.WithMaxAssociations(c.MaxConns))
	}
	if len(dests) > 0 {
		roleOpts = append(roleOpts, server.WithDIMSEMoveDestinations(dests))
	}
	role, err := server.NewDIMSERole(aet, store, cat, roleOpts...)
	if err != nil {
		return err
	}

	log := logging.FromContext(rc.Ctx)
	daemonOpts := []server.Option{
		server.WithLogger(log),
		server.WithDIMSE(role),
		server.WithBind(c.Bind),
	}
	if !isLoopbackBind(c.Bind) {
		// A non-loopback bind is an explicit opt-in the daemon refuses without an authenticator
		// (ErrInsecureBind, mapped below); the reference daemon enables AllowAll only on loopback.
		log.Warn("serve dimse: binding a non-loopback address requires authentication",
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

	result := serveDIMSEStartedResult{
		Status:           "listening",
		Bind:             c.Bind,
		AET:              string(aet),
		MoveDestinations: len(dests),
		Addrs:            stringAddrs(addrs),
	}
	if emitErr := c.emit(rc, result); emitErr != nil {
		return emitErr
	}
	log.Info("serve dimse: listening", zap.String("aet", string(aet)),
		zap.Int("move_destinations", len(dests)))

	return awaitDaemonStop(sigCtx, runErr, log, "serve dimse")
}

// parseMoveDestinations parses the repeatable AET=host:port flags into the role's known-AE table,
// validating each AE Title and address shape before any bind (a bad entry is a usage error, never
// a runtime surprise on the first C-MOVE).
func parseMoveDestinations(raw []string) (map[dimse.AETitle]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	dests := make(map[dimse.AETitle]string, len(raw))
	for _, entry := range raw {
		name, addr, ok := strings.Cut(entry, "=")
		if !ok || name == "" || addr == "" {
			return nil, &exitcode.UsageErr{Message: fmt.Sprintf("invalid --move-destination %q (use AET=host:port)", entry)}
		}
		aet, err := dimse.ParseAETitle(name)
		if err != nil {
			return nil, &exitcode.UsageErr{Message: fmt.Sprintf("invalid --move-destination AE title %q: %v", name, err)}
		}
		host, port, err := net.SplitHostPort(addr)
		if err != nil || host == "" {
			return nil, &exitcode.UsageErr{Message: fmt.Sprintf("invalid --move-destination address %q (use host:port)", addr)}
		}
		if n, perr := strconv.Atoi(port); perr != nil || n < 1 || n > 65535 {
			return nil, &exitcode.UsageErr{Message: fmt.Sprintf("invalid --move-destination port in %q (use a number 1-65535)", addr)}
		}
		if _, dup := dests[aet]; dup {
			return nil, &exitcode.UsageErr{Message: fmt.Sprintf("duplicate --move-destination AE title %q", name)}
		}
		dests[aet] = addr
	}
	return dests, nil
}

// emit renders the listening result in the resolved format.
func (c *ServeDIMSECmd) emit(rc *RunContext, r serveDIMSEStartedResult) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSON(r)
	}
	_, err := fmt.Fprintf(rc.Out.Machine, "DIMSE daemon %s listening on %s (%d move destinations)\n",
		r.AET, r.Bind, r.MoveDestinations)
	return err
}
