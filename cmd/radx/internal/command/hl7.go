package command

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/hl7v2"
	"github.com/codeninja55/go-radx/logging"
)

// defaultMaxFrame is the default MLLP frame cap (1 MiB), the hostile-input bound shared by send and
// listen so a peer cannot exhaust memory with an unbounded frame (PRD §9.3).
const defaultMaxFrame = 1 << 20

// HL7Cmd groups the HL7 v2 over MLLP commands.
type HL7Cmd struct {
	Send   HL7SendCmd   `cmd:"" help:"Send a message and read the ACK."`
	Listen HL7ListenCmd `cmd:"" help:"Receive messages and reply with ACK/NAK."`
}

// HL7SendCmd reads an HL7 v2 message from a file or stdin, frames it over MLLP, and prints the
// parsed acknowledgement — including the MSA-1 acknowledgement code rendered by name (there is no
// NAK message; a negative ack is an ACK with a rejecting MSA-1). Message content is PHI and is not
// logged at default verbosity (docs/reference/cli.md hl7).
type HL7SendCmd struct {
	File string `arg:"" optional:"" name:"file" help:"Message file (omit or '-' to read stdin)."`

	Host     string        `name:"host" required:"" env:"RADX_MLLP_HOST" help:"MLLP host."`
	Port     int           `name:"port" default:"2575" env:"RADX_MLLP_PORT" help:"MLLP port."`
	Timeout  time.Duration `name:"timeout" default:"30s" help:"Read/write timeout."`
	MaxFrame int           `name:"max-frame" default:"1048576" help:"Maximum MLLP frame length (hostile-input cap)."`
}

// hl7AckResult is the canonical machine shape for send: the acknowledgement code by name, whether
// it was positive, and the control ID it acknowledged. It carries no message content (PHI).
type hl7AckResult struct {
	Status      string `json:"status"`
	AckCode     string `json:"ack_code"`
	Positive    bool   `json:"positive"`
	ControlID   string `json:"control_id,omitempty"`
	MessageType string `json:"message_type,omitempty"`
}

// Run reads the message, sends it over MLLP, and reports the ACK. A rejecting MSA-1 (AE/AR) is a
// non-success outcome: the result is emitted then the command exits non-zero, so a script never
// reads a negative ack as a success.
func (c *HL7SendCmd) Run(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return &exitcode.UsageErr{Message: "hl7 send does not support --format csv; use human or json"}
	}

	raw, err := readMessageInput(c.File, rc.Stdin)
	if err != nil {
		return err
	}
	msg, err := hl7v2.Parse(raw)
	if err != nil {
		return err
	}

	maxFrame := c.MaxFrame
	if maxFrame <= 0 {
		maxFrame = defaultMaxFrame
	}

	client, err := hl7v2.NewClient(net.JoinHostPort(c.Host, fmt.Sprintf("%d", c.Port)),
		hl7v2.WithClientDialTimeout(c.Timeout),
		hl7v2.WithClientReadTimeout(c.Timeout),
		hl7v2.WithClientMaxFrameSize(maxFrame),
	)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	log := logging.FromContext(rc.Ctx)
	// Diagnostics name the endpoint only, never the message content (PHI).
	log.Debug("hl7 send: sending message", zap.String("host", c.Host), zap.Int("port", c.Port))

	ack, err := client.Send(rc.Ctx, msg)
	if err != nil {
		return err
	}

	result, err := ackResult(ack)
	if err != nil {
		return err
	}
	if emitErr := c.emit(rc, result); emitErr != nil {
		return emitErr
	}
	if !result.Positive {
		// A rejecting acknowledgement is a peer "no" at the application level: the message parsed and
		// was framed and sent fine, and the peer rejected it (AE) or refused it (AR). That is a
		// protocol/peer failure, not a usage fault — classify it as a network/protocol error (exit 4),
		// mirroring how a non-success DIMSE status maps, so a script never reads an AE/AR as a flag
		// mistake (exit 2) or as success.
		return &exitcode.ProtocolErr{Message: fmt.Sprintf("peer returned a non-accept acknowledgement: %s", result.AckCode)}
	}
	return nil
}

// ackResult builds the machine shape from a parsed acknowledgement message, reading MSA-1 by name.
// A missing or malformed MSA is a parse failure (the reply was not a well-formed acknowledgement).
func ackResult(ack *hl7v2.Message) (hl7AckResult, error) {
	seg, ok := ack.Segment("MSA")
	if !ok {
		return hl7AckResult{}, &hl7v2.SegmentError{Segment: "MSA", Reason: "acknowledgement has no MSA segment"}
	}
	msa, err := hl7v2.ParseMSA(seg)
	if err != nil {
		return hl7AckResult{}, err
	}
	result := hl7AckResult{
		Status:    "success",
		AckCode:   string(msa.AckCode),
		Positive:  msa.AckCode.IsPositive(),
		ControlID: msa.ControlID,
	}
	if !result.Positive {
		result.Status = "failure"
	}
	if msh, ok := ack.MSH(); ok {
		result.MessageType = renderMessageType(msh)
	}
	return result, nil
}

// emit renders the ACK result in the resolved format.
func (c *HL7SendCmd) emit(rc *RunContext, r hl7AckResult) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSON(r)
	}
	_, err := fmt.Fprintf(rc.Out.Machine, "ACK %s (%s) control-id=%s\n", r.AckCode, r.Status, r.ControlID)
	return err
}

// HL7ListenCmd binds an MLLP listener (loopback by default), receives messages, and replies with an
// ACK built from the inbound MSH. It enforces a maximum frame length so a hostile peer cannot
// exhaust memory, and shuts down on SIGINT/SIGTERM. Message content is PHI and is not logged.
type HL7ListenCmd struct {
	Bind     string `name:"bind" default:"127.0.0.1" env:"RADX_BIND" help:"Listen address (loopback by default)."`
	Port     int    `name:"port" default:"2575" env:"RADX_MLLP_PORT" help:"Listen port."`
	Ack      string `name:"ack" enum:"AA,AE,AR" default:"AA" help:"Default acknowledgment code (AA|AE|AR)."`
	MaxFrame int    `name:"max-frame" default:"1048576" help:"Maximum MLLP frame length."`
}

// hl7ListenResult is the canonical machine shape emitted once the listener is bound: the bound
// address and default ack code. The listener then blocks serving until interrupted.
type hl7ListenResult struct {
	Status  string `json:"status"`
	Bind    string `json:"bind"`
	Port    int    `json:"port"`
	AckCode string `json:"ack_code"`
}

// Run binds the MLLP listener and serves until interrupted, replying to each inbound message with
// an acknowledgement carrying the configured code, built from the inbound MSH.
func (c *HL7ListenCmd) Run(rc *RunContext) error {
	if rc.Out.Format == cli.FormatCSV {
		return &exitcode.UsageErr{Message: "hl7 listen does not support --format csv; use human or json"}
	}
	ackCode, err := hl7v2.ParseAckCode(c.Ack)
	if err != nil {
		return &exitcode.UsageErr{Message: fmt.Sprintf("invalid --ack: %v", err)}
	}

	maxFrame := c.MaxFrame
	if maxFrame <= 0 {
		maxFrame = defaultMaxFrame
	}

	log := logging.FromContext(rc.Ctx)
	if !isLoopbackBind(c.Bind) {
		log.Warn("hl7 listen: binding a non-loopback address; the listener is reachable from the network",
			zap.String("bind", c.Bind))
	}

	var received atomic.Int64
	handler := hl7v2.HandlerFunc(func(_ context.Context, m *hl7v2.Message) (*hl7v2.Message, error) {
		received.Add(1)
		return m.BuildACK(ackCode)
	})
	srv := hl7v2.NewServer(handler, hl7v2.WithMaxFrameSize(maxFrame))

	sigCtx, stop := signal.NotifyContext(rc.Ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(sigCtx, net.JoinHostPort(c.Bind, fmt.Sprintf("%d", c.Port))) }()

	addr, err := waitForMLLPBind(srv, served)
	if err != nil {
		return err
	}

	result := hl7ListenResult{Status: "listening", Bind: c.Bind, Port: mllpAddrPort(addr, c.Port), AckCode: string(ackCode)}
	if emitErr := c.emit(rc, result); emitErr != nil {
		return emitErr
	}
	log.Info("hl7 listen: listening", zap.String("addr", addr.String()))

	// Wait on BOTH the serve goroutine and the signal: a post-startup MLLP listener failure returns its
	// error promptly instead of hanging until an interrupt, while a signal drains the server gracefully.
	if err := awaitListenerStop(sigCtx, served, srv.Shutdown); err != nil {
		return err
	}
	log.Info("hl7 listen: stopped", zap.Int64("received", received.Load()))
	return nil
}

// emit renders the listening result in the resolved format.
func (c *HL7ListenCmd) emit(rc *RunContext, r hl7ListenResult) error {
	if rc.Out.Format == cli.FormatJSON {
		return rc.Out.EmitJSON(r)
	}
	_, err := fmt.Fprintf(rc.Out.Machine, "MLLP listening on %s:%d, default ack %s\n", r.Bind, r.Port, r.AckCode)
	return err
}

// waitForMLLPBind blocks until the MLLP server publishes its bound address or the serve goroutine
// returns an early bind error.
func waitForMLLPBind(srv *hl7v2.Server, served <-chan error) (net.Addr, error) {
	for {
		if addr := srv.Addr(); addr != nil {
			return addr, nil
		}
		select {
		case err := <-served:
			if err != nil {
				return nil, err
			}
			return nil, nil
		default:
		}
	}
}

// mllpAddrPort returns the resolved TCP port from a bound address, falling back to the configured
// port when the address is not a TCP address.
func mllpAddrPort(addr net.Addr, fallback int) int {
	if tcp, ok := addr.(*net.TCPAddr); ok {
		return tcp.Port
	}
	return fallback
}

// readMessageInput reads a message body from a file path, or from stdin when the path is empty or
// "-". A read failure is a file-I/O fault (exit 5). The bytes may be PHI, so they are never logged.
func readMessageInput(path string, stdin io.Reader) ([]byte, error) {
	if path == "" || path == "-" {
		return io.ReadAll(stdin)
	}
	return os.ReadFile(path) // #nosec G304 -- reading the user-specified input file is the CLI's purpose
}

// renderMessageType formats an MSH-9 message type as "Code^TriggerEvent" (e.g. "ORU^R01"). It
// names the message structure only, never a patient value.
func renderMessageType(msh hl7v2.MSH) string {
	mt := msh.MessageType
	if mt.TriggerEvent == "" {
		return mt.Code
	}
	return mt.Code + "^" + mt.TriggerEvent
}
