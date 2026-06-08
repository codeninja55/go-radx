package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"go.uber.org/zap/zapcore"

	"github.com/codeninja55/go-radx/logging"
)

// Globals holds the flags every command shares. Kong embeds it in the root grammar, so each
// flag is parsed once and is available to every command through the bound *Globals. The env
// tags bind each operational setting to a RADX_* variable; Kong's precedence is
// flags > environment > defaults (docs/reference/cli.md "Environment configuration").
type Globals struct {
	Format    Format `short:"f" name:"format" enum:"human,json,csv" default:"human" env:"RADX_FORMAT" help:"Output format: human | json | csv."`
	Output    string `short:"o" name:"output" default:"-" help:"Write machine output to a file (\"-\" = stdout)."`
	Quiet     bool   `short:"q" name:"quiet" help:"Suppress the banner and progress on stderr."`
	NoColor   bool   `name:"no-color" help:"Disable ANSI colour in human output."`
	LogLevel  string `short:"l" name:"log-level" enum:"trace,debug,info,warn,error" default:"info" env:"RADX_LOG_LEVEL" help:"Log verbosity: trace|debug|info|warn|error."`
	LogFormat string `name:"log-format" enum:"text,json" default:"text" env:"RADX_LOG_FORMAT" help:"Log encoding: text | json."`
}

// logLevel maps the --log-level enum onto a zapcore level. trace has no zap equivalent, so it
// maps to DebugLevel (the most verbose zap offers); every other value maps one to one.
func (g *Globals) logLevel() zapcore.Level {
	switch g.LogLevel {
	case "trace", "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// logFormat maps the --log-format enum onto the logging package's encoder selector. The
// console encoder is the human "text" default; "json" emits one JSON object per line.
func (g *Globals) logFormat() logging.Format {
	if g.LogFormat == "json" {
		return logging.FormatJSON
	}
	return logging.FormatConsole
}

// NewLoggerContext builds the zap logger from the global flags and injects it into a child of
// ctx, so a command reads it with logging.FromContext rather than a package global (no global
// mutable logger, docs/reference/cli.md "Logging"). The logger writes to diagnostic (stderr),
// never to machine stdout, so a log line can never pollute machine output. Default verbosity
// logs structural identifiers only — never PHI (PRD §9.1).
func (g *Globals) NewLoggerContext(ctx context.Context, diagnostic io.Writer) (context.Context, error) {
	logger, err := logging.NewLogger(diagnostic, logging.Config{
		Level:  g.logLevel(),
		Format: g.logFormat(),
	})
	if err != nil {
		return ctx, fmt.Errorf("radx: configure logging: %w", err)
	}
	return logging.WithContext(ctx, logger), nil
}

// EnvDefaults are the resolved RADX_* operational defaults a DICOM command reads for values
// not exposed as a positional argument. Kong binds the global format/log flags directly, but
// the connection settings (host, port, AE titles, timeout, max-PDU) are per-command flags
// whose env fallback Kong resolves through the env tag on each command's flag; this struct
// documents the canonical default values shared across the DICOM commands.
type EnvDefaults struct{}

// DefaultTimeout is the canonical operation-timeout default for connectionless inspection and
// the echo command (docs/reference/cli.md echo flags).
const DefaultTimeout = 30 * time.Second

// DefaultMaxPDU is the one canonical maximum-PDU default across the CLI, matching
// dimse.MaxPDULength's default; every command that opens an association defaults --max-pdu to
// this value and RADX_MAX_PDU binds the same (docs/reference/cli.md "Environment
// configuration").
const DefaultMaxPDU = 16382

// DefaultCalledAE and DefaultCallingAE are the AE-title defaults the echo/store commands use
// when neither a flag nor RADX_CALLED_AE / RADX_CALLING_AE is set.
const (
	DefaultCalledAE  = "ANY-SCP"
	DefaultCallingAE = "RADX"
)
