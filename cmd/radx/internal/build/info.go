package build

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/charmbracelet/lipgloss"
)

// Info contains build-time metadata about the radx CLI.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// Global instance set by SetBuildInfo
var info *Info

// SetBuildInfo initializes the global build info with values injected at build time.
func SetBuildInfo(version, commit, date string) {
	info = &Info{
		Version:   version,
		Commit:    commit,
		BuildDate: date,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// Get returns the current build info.
// Returns a default Info if SetBuildInfo was not called.
func Get() Info {
	if info == nil {
		return Info{
			Version:   "unknown",
			Commit:    "unknown",
			BuildDate: "unknown",
			GoVersion: runtime.Version(),
			Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		}
	}
	return *info
}

// String returns a human-readable build info string.
func (i Info) String() string {
	return fmt.Sprintf("radx version %s (commit: %s, built: %s, %s, %s)",
		i.Version, i.Commit, i.BuildDate, i.GoVersion, i.Platform)
}

// JSON returns build info as JSON string.
func (i Info) JSON() (string, error) {
	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal build info: %w", err)
	}
	return string(data), nil
}

// PrintBuildInfo prints build information to stdout in a formatted layout.
func PrintBuildInfo() {
	i := Get()

	// Define styles for output
	primaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5436bd"))
	secondaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#d6cfef"))

	// Truncate commit to first 6 characters
	commit := i.Commit
	if commit != "" && len(commit) > 6 {
		commit = commit[:6]
	}

	// Print formatted build info
	fmt.Printf("%s", primaryStyle.Render("version:         "))
	fmt.Println(secondaryStyle.Render(i.Version))

	fmt.Printf("%s", primaryStyle.Render("commit:          "))
	fmt.Println(secondaryStyle.Render(commit))

	fmt.Printf("%s", primaryStyle.Render("buildDate:       "))
	fmt.Println(secondaryStyle.Render(i.BuildDate))

	fmt.Printf("%s", primaryStyle.Render("goVersion:       "))
	fmt.Println(secondaryStyle.Render(i.GoVersion))

	fmt.Printf("%s", primaryStyle.Render("platform:        "))
	fmt.Println(secondaryStyle.Render(i.Platform))
	fmt.Println()
}

// PrintCompactBuildInfo prints a single-line build info for command output.
func PrintCompactBuildInfo() {
	i := Get()

	// Define styles for output
	primaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5436bd"))
	secondaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#d6cfef"))

	// Truncate commit to first 6 characters
	commit := i.Commit
	if commit != "" && len(commit) > 6 {
		commit = commit[:6]
	}

	// Print compact single-line build info
	fmt.Printf("%s %s %s %s %s\n",
		secondaryStyle.Render(i.Version),
		primaryStyle.Render("commit:"),
		secondaryStyle.Render(commit),
		primaryStyle.Render("buildDate:"),
		secondaryStyle.Render(i.BuildDate),
	)
}
