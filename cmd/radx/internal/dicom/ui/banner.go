package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"
)

// BannerStyle defines the styling for the ASCII banner.
var BannerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#5436bd")).
	Bold(true)

// PrintBanner prints the "RadX DICOM" ASCII art banner to stderr.
func PrintBanner() {
	banner := figure.NewFigure("RadX DICOM", "banner3", true)

	fmt.Fprintln(os.Stderr, BannerStyle.Render(banner.String()))
	fmt.Fprintln(os.Stderr)
}
