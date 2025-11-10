package main

import (
	"fmt"
	"os"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
)

// Version information (injected at build time via ldflags)
var (
	Version = "develop"
	Commit  = ""
	Date    = ""
)

func main() {
	if err := cli.Run(Version, Commit, Date); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
