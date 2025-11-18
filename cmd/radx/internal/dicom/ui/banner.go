package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/codeninja55/go-radx/cmd/radx/internal/build"
	"github.com/common-nighthawk/go-figure"
)

// BannerStyle defines the styling for the ASCII banner.
var BannerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#5436bd")).
	Bold(true)

// PrintBanner prints the "RadX" ASCII art banner and build info to stderr.
func PrintBanner() {
	banner := figure.NewFigure("RadX", "banner3", true)

	fmt.Fprintln(os.Stderr, BannerStyle.Render(banner.String()))
	build.PrintCompactBuildInfo()
	fmt.Fprintln(os.Stderr)
}
