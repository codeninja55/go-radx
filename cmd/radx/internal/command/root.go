// Package command holds the radx Kong command tree and the runner that ties the global
// output contract, the context-injected logger, and the exit-code taxonomy together. Main is
// the single entry point the binary calls; every command is a leaf struct with a
// Run(*RunContext) error method, and the runner maps the returned error onto an exit code.
package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/alecthomas/kong"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
)

// banner is the one-line startup banner shown only in human format on an interactive stdout.
const banner = "radx — go-radx command-line interface"

// CLI is the root Kong grammar: the shared global flags, the --version flag, and the command
// tree. Every command is wired end to end against the library (docs/reference/cli.md).
type CLI struct {
	cli.Globals

	Version kong.VersionFlag `short:"V" name:"version" help:"Print build information and exit."`

	Echo EchoCmd `cmd:"" help:"Verify DICOM connectivity (C-ECHO)."`
	Dump DumpCmd `cmd:"" help:"Inspect DICOM file contents."`

	Store     StoreCmd     `cmd:"" help:"Send DICOM objects (C-STORE SCU)."`
	Find      FindCmd      `cmd:"" help:"Query a remote AE (C-FIND SCU)."`
	Get       GetCmd       `cmd:"" help:"Retrieve over the same association (C-GET)."`
	Move      MoveCmd      `cmd:"" help:"Retrieve to a destination AE (C-MOVE)."`
	Scp       ScpCmd       `cmd:"" help:"Run a Storage/Verification SCP."`
	Modify    ModifyCmd    `cmd:"" help:"Edit DICOM tags and regenerate UIDs."`
	Transcode TranscodeCmd `cmd:"" help:"Rewrite a file's transfer syntax."`
	Compose   ComposeCmd   `cmd:"" help:"Build a Part 10 file from PS3.18 DICOM JSON."`
	Organize  OrganizeCmd  `cmd:"" help:"Reorganise files by Study/Series/SOP UID."`
	Lookup    LookupCmd    `cmd:"" help:"Resolve DICOM tag dictionary information."`
	Catalogue CatalogueCmd `cmd:"" help:"Index and query a local DICOM catalogue."`

	HL7      HL7Cmd      `cmd:"" name:"hl7" help:"HL7 v2 messaging over MLLP."`
	DICOMweb DICOMwebCmd `cmd:"" name:"dicomweb" help:"DICOMweb clients."`
	Convert  ConvertCmd  `cmd:"" help:"Cross-standard conversion."`
	Serve    ServeCmd    `cmd:"" help:"Run a reference daemon (server package)."`
}

// RunContext is bound into every command's Run method. It carries the resolved output surface,
// the logger-bearing context, and the build info, so a command never reaches for a global: it
// reads everything it needs from here (PRD §9.4, no global mutable state).
type RunContext struct {
	// Ctx carries the context-injected zap logger (logging.FromContext) and cancellation.
	Ctx context.Context
	// Out is the resolved output surface enforcing the clean-stdout/diagnostics-to-stderr
	// split.
	Out *cli.Output
	// Build is the resolved build stamp, for commands that report it.
	Build cli.BuildInfo
	// Stdin is the process input stream, read by the commands that accept a message on stdin
	// (hl7 send, the convert HL7 mappers) when their path argument is "-" or absent.
	Stdin io.Reader
}

// Main parses args and runs the selected command, returning the process exit code. It is the
// single composition root: it wires Kong's writers to stderr (so Kong's own help and errors
// never touch machine stdout), resolves the output surface and logger from the global flags,
// runs the command, and classifies any error onto the exit-code taxonomy. It never calls
// os.Exit itself — the caller (main) does — so it is testable in-process.
func Main(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var root CLI

	// Resolve TTY-ness from the real stdout before Kong or any command writes, so the banner
	// decision reflects the actual terminal, not a buffer.
	stdoutTTY := cli.IsTTY(stdout)

	parser, err := kong.New(&root,
		kong.Name("radx"),
		kong.Description("go-radx command-line interface for DICOM, HL7 v2, and FHIR."),
		kong.UsageOnError(),
		// Kong's help, usage, and parse-error text are diagnostics: route them to stderr so
		// machine stdout is never polluted, even on a --help or a bad flag (RADX-004).
		kong.Writers(stderr, stderr),
		// Take ownership of process exit: Kong calls this for --help and --version (status 0)
		// and for a parse error (status 1). We translate Kong's status into our taxonomy
		// rather than letting Kong call os.Exit, so Main returns a code and stays testable.
		kong.Exit(func(int) { panic(kongExitPanic{}) }),
		kong.Vars{
			"version":            cli.ResolveBuildInfo().String(),
			"default_called_ae":  cli.DefaultCalledAE,
			"default_calling_ae": cli.DefaultCallingAE,
			"default_max_pdu":    strconv.Itoa(cli.DefaultMaxPDU),
		},
	)
	if err != nil {
		// A malformed grammar is a programming error, not user input; report it and exit 1.
		_, _ = fmt.Fprintln(stderr, "radx: internal error building the command parser:", err)
		return exitcode.GeneralFailure
	}

	kctx, parseErr := parseWithExitGuard(parser, args)
	if parseErr != nil {
		// Kong already printed usage to stderr (UsageOnError + Writers). A parse failure is a
		// usage error (exit 2); a --help/--version exit is handled by the guard returning a
		// sentinel we treat as success.
		if errors.Is(parseErr, errKongExited) {
			return exitcode.Success
		}
		return exitcode.Classify(parseErr)
	}

	out, cleanup, err := buildOutput(&root.Globals, stdout, stderr, stdoutTTY)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return exitcode.Classify(err)
	}
	defer cleanup()

	ctx := context.Background()
	loggerCtx, err := root.NewLoggerContext(ctx, stderr)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return exitcode.Classify(err)
	}

	out.Banner(banner)

	rctx := &RunContext{Ctx: loggerCtx, Out: out, Build: cli.ResolveBuildInfo(), Stdin: stdin}

	if runErr := kctx.Run(rctx); runErr != nil {
		// The command already wrote any partial machine output it chose to. Report the error
		// to stderr (never stdout) and map it onto the taxonomy.
		_, _ = fmt.Fprintln(stderr, "radx:", runErr)
		return exitcode.Classify(runErr)
	}
	return exitcode.Success
}

// kongExitPanic is the sentinel panic value our kong.Exit callback raises so a Kong-initiated
// exit (--help, --version, or a parse error after printing usage) unwinds back to
// parseWithExitGuard instead of killing the process. It carries no data.
type kongExitPanic struct{}

// errKongExited marks a parse phase that Kong terminated cleanly (--help or --version), so the
// caller maps it to a success exit rather than a usage error.
var errKongExited = errors.New("kong exited during parse")

// parseWithExitGuard runs parser.Parse but converts a Kong-initiated os.Exit (intercepted by
// the kong.Exit callback's panic) into a returned sentinel, so --help and --version do not
// terminate the process from inside a library call. A genuine parse error is returned as-is.
func parseWithExitGuard(parser *kong.Kong, args []string) (kctx *kong.Context, err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(kongExitPanic); ok {
				err = errKongExited
				return
			}
			panic(r)
		}
	}()
	kctx, err = parser.Parse(args)
	return kctx, err
}

// buildOutput resolves the machine sink (stdout or the --output file) and the diagnostic sink
// (stderr) into an *cli.Output, returning a cleanup that closes any opened file. Opening
// --output is a file-I/O operation, so a failure here is a *os.PathError that classifies to
// exit 5.
func buildOutput(g *cli.Globals, stdout, stderr io.Writer, stdoutTTY bool) (*cli.Output, func(), error) {
	if g.Output == "" || g.Output == "-" {
		return cli.NewOutput(stdout, stderr, g.Format, g.Quiet, g.NoColor, stdoutTTY), func() {}, nil
	}
	f, err := os.Create(g.Output)
	if err != nil {
		return nil, func() {}, err
	}
	// Writing to a file means stdout is not the machine sink, so the banner (which keys off
	// the interactive stdout) still follows the TTY rule via stdoutTTY.
	out := cli.NewOutput(f, stderr, g.Format, g.Quiet, g.NoColor, stdoutTTY)
	return out, func() { _ = f.Close() }, nil
}
