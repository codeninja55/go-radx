package ui

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
)

// ProgressBar represents a simple progress indicator.
type ProgressBar struct {
	total   int
	current int
	prefix  string
	writer  io.Writer
}

// NewProgressBar creates a new progress bar.
func NewProgressBar(total int, prefix string) *ProgressBar {
	return &ProgressBar{
		total:  total,
		prefix: prefix,
		writer: os.Stderr,
	}
}

// Increment increments the progress and optionally prints an update.
func (p *ProgressBar) Increment(message string) {
	p.current++
	p.Print(message)
}

// Print prints the current progress.
func (p *ProgressBar) Print(message string) {
	percentage := float64(p.current) / float64(p.total) * 100
	progressText := fmt.Sprintf("[%d/%d] %.1f%%", p.current, p.total, percentage)

	style := InfoStyle.Bold(true)
	if p.current == p.total {
		style = SuccessStyle
	}

	_, _ = fmt.Fprintf(p.writer, "\r%s %s - %s",
		p.prefix,
		style.Render(progressText),
		message)

	if p.current == p.total {
		_, _ = fmt.Fprintln(p.writer) // New line when complete
	}
}

// Complete marks the progress as complete and prints a final message.
func (p *ProgressBar) Complete(message string) {
	p.current = p.total
	p.Print(message)
}

// Spinner represents a simple text-based spinner.
type Spinner struct {
	frames []string
	index  int
	prefix string
	writer io.Writer
}

// NewSpinner creates a new spinner.
func NewSpinner(prefix string) *Spinner {
	return &Spinner{
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		prefix: prefix,
		writer: os.Stderr,
	}
}

// Tick advances the spinner and prints the current frame with a message.
func (s *Spinner) Tick(message string) {
	frame := s.frames[s.index%len(s.frames)]
	s.index++

	style := lipgloss.NewStyle().Foreground(TertiaryColor)
	_, _ = fmt.Fprintf(s.writer, "\r%s %s %s", s.prefix, style.Render(frame), message)
}

// Stop stops the spinner and clears the line.
func (s *Spinner) Stop() {
	_, _ = fmt.Fprintf(s.writer, "\r%s\r", "                                                  ")
}
