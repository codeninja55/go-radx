// Package cli holds the shared output/runner harness every radx command builds on: the
// output-format contract, the Kong global flags, the RADX_* environment binding, and the
// context-injected zap logger. It enforces the three binding rules of the output contract
// (docs/reference/cli.md "Global flags and output contract"): machine output is clean,
// diagnostics go to stderr, and each command emits one canonical machine shape.
package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Format is the output encoding a command renders its result in. It is the value of the
// global -f/--format flag (and RADX_FORMAT).
type Format string

const (
	// FormatHuman is the default: a practitioner-readable rendering on stdout, with the
	// banner and progress on stderr (suppressed off a TTY or under --quiet).
	FormatHuman Format = "human"
	// FormatJSON emits the command's canonical machine shape as a JSON document (or JSON
	// Lines for streaming results) on stdout, and nothing else there.
	FormatJSON Format = "json"
	// FormatCSV emits RFC 4180 CSV on stdout for the tabular commands; it is a usage error
	// elsewhere.
	FormatCSV Format = "csv"
)

// IsMachine reports whether the format is a machine encoding (json or csv). Under a machine
// format the banner is always suppressed and stdout carries only the payload.
func (f Format) IsMachine() bool { return f == FormatJSON || f == FormatCSV }

// Output is the resolved output surface a command writes through. It separates the machine
// sink (stdout, or the --output file) from the diagnostic sink (always stderr), so a command
// physically cannot leak a banner or a log line into machine stdout: the two are different
// io.Writers. Construct one with NewOutput.
type Output struct {
	// Machine is where the canonical payload goes: stdout by default, or the --output file.
	Machine io.Writer
	// Diagnostic is where the banner, progress, warnings, and logs go: always stderr.
	Diagnostic io.Writer
	// Format is the resolved output format.
	Format Format
	// Quiet suppresses the banner and progress even in human format on a TTY.
	Quiet bool
	// NoColor disables ANSI colour in human output.
	NoColor bool
	// stdoutTTY records whether the original stdout is an interactive terminal, so the
	// banner shows only there (and never when stdout is redirected or piped).
	stdoutTTY bool
}

// NewOutput resolves the output surface from the global flags. machineSink is stdout (or the
// opened --output file); diagnostic is stderr; stdoutIsTTY reports whether the process's
// stdout is an interactive terminal (computed once by the caller, who owns the os.File).
func NewOutput(machineSink, diagnostic io.Writer, format Format, quiet, noColor, stdoutIsTTY bool) *Output {
	return &Output{
		Machine:    machineSink,
		Diagnostic: diagnostic,
		Format:     format,
		Quiet:      quiet,
		NoColor:    noColor,
		stdoutTTY:  stdoutIsTTY,
	}
}

// ShowBanner reports whether the startup banner should be printed. The banner is shown only
// in human format, on an interactive stdout, and when --quiet is not set; it is always
// suppressed under a machine format so stdout stays clean (docs/reference/cli.md "Diagnostics
// go to stderr").
func (o *Output) ShowBanner() bool {
	if o.Format.IsMachine() || o.Quiet {
		return false
	}
	return o.stdoutTTY
}

// Banner writes the one-line startup banner to the diagnostic sink (stderr) when ShowBanner
// allows it. It is a no-op otherwise, so machine stdout is never touched.
func (o *Output) Banner(text string) {
	if !o.ShowBanner() {
		return
	}
	_, _ = fmt.Fprintln(o.Diagnostic, text)
}

// EmitJSON writes v as a single indented JSON document to the machine sink, followed by a
// newline. It is the canonical single-result emitter (echo, dump-as-json) and writes nothing
// but the payload.
func (o *Output) EmitJSON(v any) error {
	enc := json.NewEncoder(o.Machine)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// EmitJSONLine writes v as one compact JSON object followed by a newline — a single line of a
// JSON Lines stream, for commands with per-item results. Each call is one complete line, so a
// consumer can process results as they arrive.
func (o *Output) EmitJSONLine(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(o.Machine, string(b))
	return err
}

// CSVWriter returns an RFC 4180 csv.Writer over the machine sink. The caller writes header
// and rows then calls Flush; the writer touches only the machine sink.
func (o *Output) CSVWriter() *csv.Writer { return csv.NewWriter(o.Machine) }

// IsTTY reports whether w is an interactive terminal. It inspects the file mode rather than
// pulling in a terminal-detection dependency: a character device that is not a regular file,
// pipe, or socket is treated as a TTY. A non-*os.File writer (a bytes.Buffer in tests) is
// never a TTY, so tests see banner-free machine output by construction.
func IsTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
