package config

import (
	"github.com/alecthomas/kong"
	"github.com/codeninja55/go-radx/cmd/radx/internal/build"
)

// GlobalConfig contains global flags available to all commands.
type GlobalConfig struct {
	Version   VersionFlag  `short:"V" help:"Print version information and quit"`
	LogLevel  string       `short:"l" help:"Log output level" enum:"trace,debug,info,warn,error,fatal,panic" default:"info"`
	Pretty    bool         `name:"pretty" help:"Pretty print output" default:"true" negatable:""`
	Debug     bool         `help:"Switch on debug mode" default:"false"`
	OutputDir string       `name:"output" short:"o" type:"path" help:"Output directory" default:"."`
	Format    OutputFormat `name:"format" short:"f" help:"Output format" enum:"table,json,csv" default:"table"`
}

// OutputFormat represents the output format for commands.
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatCSV   OutputFormat = "csv"
)

// VersionFlag is a custom flag type that prints version info and exits.
type VersionFlag bool

// BeforeApply is called by Kong before the command runs.
// It prints build info and exits.
func (v VersionFlag) BeforeApply(app *kong.Kong, vars kong.Vars) error {
	if v {
		build.PrintBuildInfo()
		app.Exit(0)
	}
	return nil
}
