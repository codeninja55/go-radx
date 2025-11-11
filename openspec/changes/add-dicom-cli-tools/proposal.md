# Add DICOM CLI Tools

## Why

The go-radx library currently provides comprehensive DICOM and DIMSE networking capabilities but lacks user-facing CLI
tools to leverage these features. Healthcare professionals, PACS administrators, and medical imaging engineers need
command-line utilities to perform common DICOM operations without writing custom code.

Common scenarios include:
- Verifying DICOM network connectivity between systems (C-ECHO)
- Inspecting DICOM file metadata for troubleshooting
- Sending DICOM studies to PACS systems (C-STORE)
- Modifying DICOM tags for anonymization or correction
- Running SCP servers to receive DICOM objects
- Organizing DICOM files into standardized directory structures

The reference implementation at `/Users/Andru.Che@annalise.ai/annalise/nexus-dicom-util/` demonstrates proven
patterns for CLI structure, build info management, and user experience that should be adapted for go-radx.

## What Changes

This proposal adds a comprehensive DICOM utility CLI as a subcommand of the main `radx` CLI tool. The implementation
will provide six core commands with a consistent, user-friendly interface built using modern TUI libraries.

### New CLI Commands

1. **radx dicom cecho** - DICOM verification service (C-ECHO)
   - Test network connectivity to DICOM servers
   - Verify association negotiation
   - Support for custom AE titles, timeouts

2. **radx dicom cstore** - DICOM storage service (C-STORE SCU)
   - Send DICOM files or directories to PACS
   - Support for batch operations with progress tracking
   - **Rate limiting support** (files/second or MB/second)
   - Configurable burst size for rate limiting
   - Configurable transfer syntaxes

3. **radx dicom dump** - Inspect DICOM file contents
   - Display DICOM tags in human-readable format
   - Multiple output formats (table, JSON, CSV)
   - Optional pixel data processing
   - Inspired by DCMTK dcmdump but with idiomatic Go flags

4. **radx dicom modify** - Modify DICOM file tags
   - Insert/update DICOM tags
   - Delete DICOM tags
   - Regenerate UIDs (Study, Series, Instance)
   - Batch processing support
   - Inspired by DCMTK dcmodify but with idiomatic Go flags
   - Native Go implementation using go-radx dicom package

5. **radx dicom scp** - Run DICOM SCP server
   - Accept incoming DICOM associations
   - Configurable SOP Class filters (defaults to Secondary Capture)
   - Auto-organize received files by Study/Series/Instance UID
   - Long-running daemon mode

6. **radx dicom organize** - Reorganize DICOM directory structure
   - Walk directory tree to find DICOM files
   - Reorganize into: `<study-uid>/<series-uid>/<instance-uid>.dcm`
   - Support both copy (default) and move operations
   - Preserve or flatten original structure

### User Experience Enhancements

- **ASCII Art Banner**: Display "GO RADX DICOM UTIL" on every command invocation
- **Rich TUI**: Use Charmbracelet libraries (huh, lipgloss, log) for interactive elements
- **Progress Tracking**: Visual progress bars for batch operations
- **Structured Logging**: Configurable log levels with pretty formatting
- **Table Output**: Use simpletable for human-readable tabular data
- **Multiple Formats**: Support JSON, table, and CSV output for all commands
- **Consistent Flags**: Idiomatic Go flag naming across all commands

### Architecture

- **CLI Framework**: alecthomas/kong for declarative CLI parsing (matching reference implementation)
- **Build Info**: Inject version, commit, and build date via linker flags
- **Global Config**: Shared configuration context passed to all commands
- **Native Implementation**: Use go-radx dicom and dimse packages directly
- **Error Handling**: Comprehensive error messages with actionable guidance

## Impact

### Affected Specs
- **ADDED**: `dicom-cli` - Complete DICOM CLI toolkit specification

### Affected Code

**New Files**:
- `cmd/radx/main.go` - CLI entry point with version injection
- `cmd/radx/internal/build/info.go` - Build information management
- `cmd/radx/internal/cli/cli.go` - CLI setup with Kong
- `cmd/radx/internal/config/config.go` - Global configuration context
- `cmd/radx/internal/dicom/commands/` - Individual command implementations:
  - `cecho.go` - C-ECHO verification command
  - `cstore.go` - C-STORE storage command
  - `dump.go` - DICOM dump command
  - `modify.go` - DICOM modification command
  - `scp.go` - SCP server command
  - `organize.go` - Directory organization command
  - `helpers.go` - Shared helper functions
- `cmd/radx/internal/dicom/ui/` - TUI components:
  - `banner.go` - ASCII art banner
  - `theme.go` - Charmbracelet theme configuration
  - `progress.go` - Progress bar utilities
  - `table.go` - Table rendering utilities

**Modified Files**:
- `mise.toml` - Add tasks for building and testing CLI tools
- `README.md` - Add CLI documentation and examples
- `docs/user-guide/cli/` - **NEW** - CLI user guide documentation
- `.github/workflows/ci.yml` - Add CLI build and test steps
- `.github/workflows/release.yml` - **NEW** - GoReleaser workflow for releases
- `.goreleaser.yml` - **NEW** - GoReleaser configuration

**Integration Points**:
- Uses existing `dicom.DataSet`, `dicom.ParseFile()`, `dicom.WriteFile()`
- Uses existing `dimse.Client` for networking operations
- Uses existing `dicom.ParseDirectory()` for batch operations
- No changes required to core dicom or dimse packages

### New Dependencies

**CLI Framework**:
- `github.com/alecthomas/kong` - CLI argument parsing

**TUI Libraries**:
- `github.com/charmbracelet/huh` - Interactive prompts and forms
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `github.com/charmbracelet/log` - Structured logging
- `github.com/charmbracelet/bubbles` - TUI components

**Output Formatting**:
- `github.com/alexeyco/simpletable` - ASCII table rendering
- `github.com/common-nighthawk/go-figure` - ASCII art text generation

All dependencies have permissive licenses (MIT/Apache-2.0) and are actively maintained.

### Benefits

- ✅ **Production Ready**: Provides immediately usable CLI tools for DICOM operations
- ✅ **User Experience**: Modern TUI with progress tracking and rich formatting
- ✅ **Developer Friendly**: Comprehensive documentation and examples
- ✅ **Integration Ready**: Natural integration point for go-radx library users
- ✅ **Standards Compliant**: Full DICOM and DIMSE protocol support
- ✅ **Extensible**: Clear patterns for adding new commands
- ✅ **Cross-Platform**: Works on macOS, Linux, and Windows (WSL2)

### Non-Goals

- ❌ **DICOM Viewer**: Not implementing image display or rendering
- ❌ **PACS Server**: Not a full PACS implementation (only SCP for receiving)
- ❌ **HL7 Integration**: Focused on DICOM only for this change
- ❌ **Web Interface**: CLI only, no web UI
- ❌ **Backwards Compatibility**: First CLI release, no compatibility constraints

###Release and Distribution

- **GoReleaser Integration**: Automated release process using GoReleaser
  - Cross-platform builds for macOS (Intel/ARM), Linux (x86_64/ARM64), Windows
  - Automated GitHub Releases with changelog generation
  - Checksums and signatures for release artifacts
  - Archive creation for easy distribution

### Risk Mitigation

1. **Dependency Management**: All dependencies are well-maintained, widely used, and permissive licenses
2. **Testing Strategy**: Comprehensive integration tests using test DICOM files in `testdata/`
3. **Documentation**: User guide, man pages, and inline help for all commands
4. **Error Handling**: Clear error messages with troubleshooting guidance
5. **Performance**: Progress tracking for long operations, cancellation support
6. **Security**: No credential storage, support for secure network configuration
7. **Release Automation**: GoReleaser ensures consistent, reproducible releases across platforms
