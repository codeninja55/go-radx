package cli

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/log"
	"github.com/codeninja55/go-radx/cmd/radx/internal/build"
	"github.com/codeninja55/go-radx/cmd/radx/internal/config"
	"github.com/codeninja55/go-radx/cmd/radx/internal/dicom/commands"
)

const (
	appName        = "radx"
	appDescription = "DICOM utility CLI for go-radx"
)

// CLI represents the root command structure.
type CLI struct {
	config.GlobalConfig

	// DICOM subcommand group
	Dicom DicomCmd `cmd:"" help:"DICOM utilities"`
}

// DicomCmd is the parent command for all DICOM utilities.
type DicomCmd struct {
	Dump      commands.DumpCmd      `cmd:"" name:"dump" help:"Inspect DICOM file contents"`
	Echo      commands.CEchoCmd     `cmd:"" name:"echo" help:"Verify DICOM connectivity (C-ECHO)"`
	Store     commands.CStoreCmd    `cmd:"" name:"store" help:"Send DICOM files to server (C-STORE)"`
	Modify    commands.ModifyCmd    `cmd:"" name:"modify" help:"Modify DICOM file tags"`
	Organize  commands.OrganizeCmd  `cmd:"" name:"organize" help:"Reorganize DICOM files by UID structure"`
	SCP       commands.SCPCmd       `cmd:"" name:"scp" help:"Run DICOM SCP server"`
	Lookup    commands.LookupCmd    `cmd:"" name:"lookup" help:"Look up DICOM tag information"`
	Catalogue commands.CatalogueCmd `cmd:"" name:"catalogue" help:"Build and query DICOM file catalogue database"`
}

// Run executes the radx CLI with the provided build info.
func Run(version, commit, date string) error {
	// Set build info
	build.SetBuildInfo(version, commit, date)

	// Parse command-line arguments
	cli := &CLI{}
	ctx := kong.Parse(cli,
		kong.Name(appName),
		kong.Description(appDescription),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{
			"version": version,
			"commit":  commit,
			"date":    date,
		},
	)

	// Setup logging
	logger := setupLogger(&cli.GlobalConfig)

	// Log build info
	logger.Debug("radx CLI starting",
		"version", version,
		"commit", commit,
		"build_date", date,
	)

	// Run the selected command
	err := ctx.Run(&cli.GlobalConfig)
	if err != nil {
		logger.Error("command failed", "error", err)
		return err
	}

	return nil
}

// setupLogger configures the global logger based on config.
func setupLogger(cfg *config.GlobalConfig) *log.Logger {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    cfg.Debug,
		ReportTimestamp: true,
		TimeFormat:      "15:04:05",
	})

	// Set log level
	switch cfg.LogLevel {
	case "trace":
		logger.SetLevel(log.DebugLevel) // log package doesn't have trace, use debug
	case "debug":
		logger.SetLevel(log.DebugLevel)
	case "info":
		logger.SetLevel(log.InfoLevel)
	case "warn":
		logger.SetLevel(log.WarnLevel)
	case "error":
		logger.SetLevel(log.ErrorLevel)
	case "fatal":
		logger.SetLevel(log.FatalLevel)
	default:
		logger.SetLevel(log.InfoLevel)
	}

	// Set output format (pretty vs JSON)
	if !cfg.Pretty {
		logger.SetFormatter(log.JSONFormatter)
	}

	// Set as default logger
	log.SetDefault(logger)

	return logger
}

// ParseArgs is a convenience function for testing.
// It parses arguments and returns the CLI struct and Kong context.
func ParseArgs(args []string, version, commit, date string) (*CLI, *kong.Context, error) {
	build.SetBuildInfo(version, commit, date)

	cli := &CLI{}
	parser, err := kong.New(cli,
		kong.Name(appName),
		kong.Description(appDescription),
		kong.Vars{
			"version": version,
			"commit":  commit,
			"date":    date,
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create parser: %w", err)
	}

	ctx, err := parser.Parse(args)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	return cli, ctx, nil
}
