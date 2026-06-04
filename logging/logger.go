package logging

import (
	"context"
	"fmt"
	"io"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Format selects the encoder a constructed logger uses.
type Format int

const (
	// FormatJSON emits machine-structured JSON, one object per entry.
	FormatJSON Format = iota
	// FormatConsole emits a human-readable, level-coloured line per entry.
	FormatConsole
)

// Config describes how NewLogger builds its logger. The zero value is valid and
// yields a JSON logger at InfoLevel writing to the provided sink.
type Config struct {
	// Level is the minimum level an entry must meet to be emitted. The zero
	// value (zapcore.InfoLevel) keeps default verbosity at info, where only
	// structural identifiers — never patient values — are logged.
	Level zapcore.Level
	// Format selects JSON or console encoding.
	Format Format
}

// NewLogger constructs a *zap.Logger writing to sink according to cfg. It is the
// single construction point a composition root calls once; the result is then
// threaded through context.Context rather than stored in a package global.
//
// The caller owns sink's lifetime. Callers should defer logger.Sync() before the
// process exits to flush buffered entries.
func NewLogger(sink io.Writer, cfg Config) (*zap.Logger, error) {
	if sink == nil {
		return nil, fmt.Errorf("logging: sink must not be nil")
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	switch cfg.Format {
	case FormatConsole:
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	case FormatJSON:
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	default:
		return nil, fmt.Errorf("logging: unknown format %d", cfg.Format)
	}

	// Lock the sink: the logger is threaded through context and used
	// concurrently, and a bare io.Writer offers no write serialization, so
	// unlocked writes could interleave and corrupt records.
	core := zapcore.NewCore(encoder, zapcore.Lock(zapcore.AddSync(sink)), cfg.Level)
	return zap.New(core), nil
}

// loggerKey is an unexported context key so only this package can set or read the
// injected logger, preventing collisions with other packages' context values.
type loggerKey struct{}

// WithContext returns a child context carrying logger. A nil logger is replaced
// with the no-op logger so FromContext never returns nil.
func WithContext(ctx context.Context, logger *zap.Logger) context.Context {
	if logger == nil {
		logger = zap.NewNop()
	}
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext returns the logger carried by ctx, or a no-op logger when none is
// present. It never returns nil, so callers can log unconditionally without a
// nil check.
func FromContext(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*zap.Logger); ok {
		return logger
	}
	return zap.NewNop()
}
