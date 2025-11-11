package build

import (
	"encoding/json"
	"fmt"
	"runtime"
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

// PrintBuildInfo prints build information to stdout.
func PrintBuildInfo() {
	i := Get()
	fmt.Println(i.String())
}
