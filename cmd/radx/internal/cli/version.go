package cli

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// version, commit, and date are the build-stamp variables a release overrides with
// -ldflags "-X .../internal/cli.version=v1.2.3 -X ...commit=abc -X ...date=2026-01-01". When
// unset (a plain `go build` or `go install`), BuildInfo fills them from the embedded VCS
// stamp, so radx --version is coherent in both a release artefact and a from-source build.
var (
	version = ""
	commit  = ""
	date    = ""
)

// BuildInfo is the resolved build stamp radx --version prints. Every field is structural
// build metadata, never user or patient data.
type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"go_version"`
}

// ResolveBuildInfo merges the -ldflags overrides with the embedded VCS stamp from
// runtime/debug.ReadBuildInfo, preferring an explicit override and otherwise reading the
// stamp Go records at build time. A field with neither source resolves to "unknown" so the
// output is always coherent (never an empty field).
func ResolveBuildInfo() BuildInfo {
	bi := BuildInfo{
		Version:   version,
		Commit:    commit,
		Date:      date,
		GoVersion: runtime.Version(),
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		if bi.Version == "" && info.Main.Version != "" {
			bi.Version = info.Main.Version
		}
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if bi.Commit == "" {
					bi.Commit = setting.Value
				}
			case "vcs.time":
				if bi.Date == "" {
					bi.Date = setting.Value
				}
			}
		}
	}

	bi.Version = orUnknown(bi.Version)
	bi.Commit = orUnknown(bi.Commit)
	bi.Date = orUnknown(bi.Date)
	return bi
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

// String renders the build stamp as a single human line for radx --version.
func (b BuildInfo) String() string {
	return fmt.Sprintf("radx %s (commit %s, built %s, %s)", b.Version, b.Commit, b.Date, b.GoVersion)
}
