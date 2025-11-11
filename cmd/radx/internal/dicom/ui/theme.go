package ui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Color scheme for go-radx CLI
var (
	PrimaryColor   = lipgloss.Color("#5436bd")
	SecondaryColor = lipgloss.Color("#d6cfef")
	TertiaryColor  = lipgloss.Color("#40c3fe")
	SuccessColor   = lipgloss.Color("#69efad")
	WarnColor      = lipgloss.Color("#fffc7a")
	ErrorColor     = lipgloss.Color("#fe5151")
	DarkColor      = lipgloss.Color("#546d79")
)

// Common lipgloss styles
var (
	HeaderStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
			MarginBottom(1)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(SuccessColor).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	WarnStyle = lipgloss.NewStyle().
			Foreground(WarnColor).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(TertiaryColor)

	SubtleStyle = lipgloss.NewStyle().
			Foreground(DarkColor)
)

// ThemeRadx returns a custom huh theme for interactive prompts.
func ThemeRadx() *huh.Theme {
	theme := huh.ThemeBase()

	// Customize theme colors
	theme.Focused.Base = theme.Focused.Base.BorderForeground(PrimaryColor)
	theme.Focused.Title = theme.Focused.Title.Foreground(PrimaryColor).Bold(true)
	theme.Focused.SelectSelector = theme.Focused.SelectSelector.Foreground(TertiaryColor)
	theme.Focused.SelectedOption = theme.Focused.SelectedOption.Foreground(SuccessColor)

	theme.Blurred.Base = theme.Blurred.Base.BorderForeground(DarkColor)

	return theme
}
