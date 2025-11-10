# Design Document: DICOM CLI Tools

## Context

The go-radx library provides comprehensive DICOM and DIMSE networking capabilities as Go packages, but lacks
user-facing CLI tools. This design document outlines the technical architecture, patterns, and implementation decisions
for adding a complete DICOM utility CLI as a subcommand of the main `radx` tool.

**Background:**
- Existing DICOM core in `dicom/` package (parsing, writing, datasets)
- Existing DIMSE networking in `dimse/` package (C-ECHO, C-STORE, C-FIND, etc.)
- Reference implementation available at `/Users/Andru.Che@annalise.ai/annalise/nexus-dicom-util/`
- Target users: PACS administrators, radiology engineers, healthcare software developers

**Constraints:**
- Must follow go-radx project conventions (KISS, YAGNI, SOLID, 12-Factor App)
- Native Go implementation (no external binary dependencies)
- Cross-platform support (macOS, Linux, Windows/WSL2)
- Must integrate cleanly with existing go-radx packages
- No breaking changes to existing dicom or dimse packages

**Stakeholders:**
- Healthcare software developers (primary users)
- PACS administrators (operations and troubleshooting)
- Radiology workflow engineers (automation and integration)
- go-radx library maintainers (code quality and patterns)

## Goals / Non-Goals

### Goals
1. Provide production-ready CLI tools for common DICOM operations
2. Match or exceed capabilities of DCMTK utilities (dcmdump, dcmodify, echoscu, storescu)
3. Deliver excellent user experience with modern TUI elements
4. Follow reference implementation patterns for consistency
5. Enable automation through scriptable commands and JSON output
6. Support both interactive and non-interactive usage

### Non-Goals
- DICOM image viewer or renderer (out of scope)
- Full PACS server implementation (only SCP for receiving)
- HL7 integration (future work, not this change)
- Web interface or REST API (CLI only)
- DICOM C-FIND, C-GET, C-MOVE (future commands)

## Technical Decisions

### 1. CLI Framework: alecthomas/kong

**Decision:** Use `github.com/alecthomas/kong` for CLI argument parsing.

**Rationale:**
- Declarative struct-based API reduces boilerplate
- Automatic help generation and validation
- Support for complex flag types (enums, file paths, durations)
- Excellent error messages
- Used successfully in reference implementation
- Better than stdlib `flag` or `cobra` for complex CLIs

**Alternatives Considered:**
- `spf13/cobra` - More verbose, requires manual flag binding
- `urfave/cli` - Less type-safe, more runtime configuration
- stdlib `flag` - Insufficient for complex subcommand structure

**Example Usage:**
```go
type CLI struct {
    config.GlobalConfig
    Dump   commands.DumpCmd   `cmd:"" name:"dump" help:"Inspect DICOM files"`
    CStore commands.CStoreCmd `cmd:"" name:"cstore" help:"Send DICOM files to PACS"`
}

func main() {
    ctx := kong.Parse(&cli)
    err := ctx.Run(&cli.GlobalConfig)
    kong.FatalIfErrorf(err)
}
```

### 2. Build Info Injection: Linker Flags

**Decision:** Inject version, commit, and build date via Go linker flags (ld flags).

**Rationale:**
- Standard Go practice for version injection
- No runtime dependencies
- Works with all build systems
- Provides accurate build metadata
- Matches reference implementation pattern

**Implementation:**
```makefile
VERSION := $(shell git describe --tags --always --dirty)
COMMIT := $(shell git rev-parse HEAD)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w \
  -X main.Version=$(VERSION) \
  -X main.Commit=$(COMMIT) \
  -X main.Date=$(BUILD_DATE)

build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/radx ./cmd/radx
```

### 3. TUI Libraries: Charmbracelet Ecosystem

**Decision:** Use Charmbracelet libraries for terminal UI components.

**Rationale:**
- Modern, actively maintained Go TUI libraries
- Excellent developer experience and documentation
- Cross-platform terminal support
- Composable components
- Used widely in Go CLI tools
- Matches user requirements

**Components:**
- `lipgloss` - Styling and layout
- `log` - Structured logging with styled output
- `huh` - Interactive forms and prompts
- `bubbles` - Reusable TUI components (progress bars, spinners)

**Example:**
```go
// Styled banner
var bannerStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("#5436bd")).
    Bold(true)

// Structured logging
logger := log.New(os.Stderr)
logger.SetLevel(log.InfoLevel)
logger.Info("Starting DICOM server", "port", 11112, "ae", "RADX_SCP")
```

### 4. ASCII Art: go-figure

**Decision:** Use `github.com/common-nighthawk/go-figure` for ASCII art banner.

**Rationale:**
- Simple API for generating ASCII art text
- Multiple font options
- No external dependencies
- Lightweight (5KB)

**Example:**
```go
func PrintBanner() {
    fig := figure.NewFigure("GO RADX DICOM UTIL", "banner3", true)
    fig.Print()
}
```

### 5. Table Rendering: simpletable

**Decision:** Use `github.com/alexeyco/simpletable` for ASCII table rendering.

**Rationale:**
- Simple, focused library for table output
- Supports styling and alignment
- No external dependencies
- Matches user requirements
- Lightweight alternative to termui or tview

**Example:**
```go
table := simpletable.New()
table.Header = &simpletable.Header{
    Cells: []*simpletable.Cell{
        {Text: "Tag"},
        {Text: "VR"},
        {Text: "Value"},
    },
}
table.Body.Cells = append(table.Body.Cells, []*simpletable.Cell{
    {Text: "(0010,0010)"},
    {Text: "PN"},
    {Text: "Doe^John"},
})
table.Println()
```

### 6. Output Formats: JSON, Table, CSV

**Decision:** Support multiple output formats controlled by flags.

**Rationale:**
- JSON for machine-readable output (scripting, automation)
- Table for human-readable output (interactive use)
- CSV for spreadsheet import (data analysis)
- Industry standard formats
- Enables multiple use cases

**Implementation Pattern:**
```go
type OutputFormat string

const (
    OutputJSON  OutputFormat = "json"
    OutputTable OutputFormat = "table"
    OutputCSV   OutputFormat = "csv"
)

type GlobalConfig struct {
    Format OutputFormat `enum:"json,table,csv" default:"table" help:"Output format"`
}

func RenderMetadata(meta []DicomMetadata, format OutputFormat) error {
    switch format {
    case OutputJSON:
        return json.NewEncoder(os.Stdout).Encode(meta)
    case OutputTable:
        return renderTable(meta)
    case OutputCSV:
        return renderCSV(meta)
    }
}
```

### 7. DICOM Modify: Native Go Implementation

**Decision:** Implement modify command natively in Go, don't wrap DCMTK dcmodify.

**Rationale:**
- No external binary dependencies (pure Go)
- Better error handling and validation
- Consistent with go-radx philosophy
- Cross-platform without DCMTK installation
- Can leverage existing go-radx dicom.DataSet API
- Easier to extend with custom operations

**Operations Supported:**
```go
// Tag insertion/update
-i "0010,0010=Doe^John"           // Insert or update tag
-i "0010,0030=19800101"           // Date format

// Tag deletion
-e "0010,0010"                    // Remove patient name
-e "0010,xxxx"                    // Remove all 0010 group tags (with confirmation)

// UID regeneration
--regenerate-study-uid            // Generate new Study Instance UID
--regenerate-series-uid           // Generate new Series Instance UID
--regenerate-instance-uid         // Generate new SOP Instance UID
--regenerate-all-uids             // Regenerate all UIDs
```

**Alternative:** Wrapping DCMTK was considered but rejected due to external dependency requirement and platform
compatibility issues.

### 8. SCP Server: Configurable SOP Class Filters

**Decision:** Support configurable SOP Class filtering with sensible defaults.

**Rationale:**
- Secondary Capture is common for generated reports
- Different workflows need different SOP Classes
- Flexibility for various use cases
- Safety through defaults

**Implementation:**
```go
type SCPCmd struct {
    Port              int      `default:"11112" help:"DICOM port"`
    AETitle           string   `default:"RADX_SCP" help:"Application Entity title"`
    OutputDir         string   `required:"" type:"path" help:"Output directory"`
    AcceptedSOPClasses []string `help:"SOP Class UIDs to accept (default: Secondary Capture)"`
    AcceptAll         bool     `help:"Accept all SOP Classes"`
}

// Default SOP Classes if not specified
var defaultSOPClasses = []string{
    "1.2.840.10008.5.1.4.1.1.7",    // Secondary Capture
    "1.2.840.10008.5.1.4.1.1.7.1",  // Multi-frame Single Bit SC
    "1.2.840.10008.5.1.4.1.1.7.2",  // Multi-frame Grayscale Byte SC
}
```

### 9. File Organization: Study/Series/Instance Structure

**Decision:** Use UID-based directory structure for organized DICOM storage.

**Rationale:**
- Standard DICOM organizational pattern
- Matches PACS internal structure
- Prevents filename collisions
- Enables quick lookup by UID
- Supports multiple modalities

**Structure:**
```
<output-dir>/
└── <study-instance-uid>/
    └── <series-instance-uid>/
        └── <sop-instance-uid>.dcm
```

**Example:**
```
dicom-data/
└── 1.2.840.113619.2.5.1762583153.215519.978957063.78/
    ├── 1.2.840.113619.2.5.1762583153.215519.978957063.121/
    │   ├── 1.2.840.113619.2.5.1762583153.215519.978957063.122.dcm
    │   └── 1.2.840.113619.2.5.1762583153.215519.978957063.123.dcm
    └── 1.2.840.113619.2.5.1762583153.215519.978957063.124/
        └── 1.2.840.113619.2.5.1762583153.215519.978957063.125.dcm
```

**Collision Handling:** If file exists, append sequential number: `<uid>-1.dcm`, `<uid>-2.dcm`, etc.

### 10. Rate Limiting for C-STORE

**Decision:** Implement configurable rate limiting for C-STORE operations.

**Rationale:**
- Prevents overwhelming target PACS systems
- Controls network bandwidth usage
- Enables throttled batch uploads
- Industry requirement for large transfers
- Prevents connection timeouts from sustained high load

**Implementation:**
```go
type CStoreCmd struct {
    // ... other fields
    RateLimit     float64 `help:"Rate limit in files/second (0 = unlimited)" default:"0"`
    RateLimitMB   float64 `help:"Rate limit in MB/second (0 = unlimited)" default:"0"`
    BurstSize     int     `help:"Maximum burst size for rate limiting" default:"1"`
}

// Rate limiter using golang.org/x/time/rate
import "golang.org/x/time/rate"

func (c *CStoreCmd) createRateLimiter() *rate.Limiter {
    if c.RateLimit > 0 {
        // Files per second
        return rate.NewLimiter(rate.Limit(c.RateLimit), c.BurstSize)
    }
    if c.RateLimitMB > 0 {
        // Convert MB/s to bytes/s
        bytesPerSecond := c.RateLimitMB * 1024 * 1024
        return rate.NewLimiter(rate.Limit(bytesPerSecond), int(bytesPerSecond))
    }
    return nil // Unlimited
}
```

**Options:**
- `--rate-limit 10` - Limit to 10 files per second
- `--rate-limit-mb 50` - Limit to 50 MB per second
- `--burst-size 5` - Allow burst of 5 files at once

## Architecture

### Directory Structure

```
cmd/radx/
├── main.go                          # Entry point
├── go.mod                           # CLI dependencies
├── internal/
│   ├── build/
│   │   └── info.go                  # Build info management
│   ├── cli/
│   │   └── cli.go                   # Kong CLI setup
│   ├── config/
│   │   └── config.go                # Global configuration
│   └── dicom/
│       ├── commands/
│       │   ├── cecho.go             # C-ECHO command
│       │   ├── cstore.go            # C-STORE command (with rate limiting)
│       │   ├── dump.go              # Dump command
│       │   ├── modify.go            # Modify command
│       │   ├── scp.go               # SCP server command
│       │   ├── organize.go          # Organize command
│       │   └── helpers.go           # Shared helpers
│       └── ui/
│           ├── banner.go            # ASCII banner
│           ├── theme.go             # Charmbracelet theme
│           ├── progress.go          # Progress bars
│           └── table.go             # Table rendering
```

### Data Flow

**1. CLI Initialization:**
```
main.go
  ↓ (parse args)
cli.ParseArgs()
  ↓ (create context)
kong.Context
  ↓ (run command)
Command.Run(config.GlobalConfig)
```

**2. Command Execution (dump example):**
```
DumpCmd.Run()
  ↓
ui.PrintBanner()
  ↓
helpers.listDicomFiles(path)
  ↓
dicom.ParseFile() (go-radx package)
  ↓
helpers.formatDicomValue()
  ↓
ui.renderTable() / renderJSON() / renderCSV()
```

**3. Network Operations (cstore example):**
```
CStoreCmd.Run()
  ↓
ui.PrintBanner()
  ↓
dimse.NewClient() (go-radx package)
  ↓
client.Associate()
  ↓
rate.Limiter.Wait() (if rate limiting enabled)
  ↓
client.CStore(dataset)
  ↓
ui.updateProgress()
  ↓
client.Release()
```

### Error Handling Strategy

**1. User-Facing Errors:**
- Clear, actionable error messages
- Suggestions for resolution
- Exit codes for scripting

```go
if err != nil {
    return fmt.Errorf("failed to connect to DICOM server %s:%d: %w\n"+
        "  Hint: Check that the server is running and the AE title is correct",
        host, port, err)
}
```

**2. Exit Codes:**
- 0: Success
- 1: General error
- 2: Invalid command-line arguments
- 3: DICOM parsing error
- 4: Network error
- 5: File I/O error

**3. Logging:**
- Use charmbracelet/log for structured logging
- Configurable log levels (trace, debug, info, warn, error)
- Errors logged to stderr, output to stdout
- JSON logging mode for automation

### Concurrency Model

**1. Batch Operations:**
- Use worker pool pattern for parallel file processing
- Configurable worker count (default: NumCPU)
- Progress tracking with atomic counters

```go
type workerPool struct {
    workers   int
    tasks     chan task
    results   chan result
    wg        sync.WaitGroup
    progress  *progress.Model
}
```

**2. SCP Server:**
- One goroutine per association
- Graceful shutdown with context cancellation
- Connection pool limits to prevent resource exhaustion

```go
func (s *SCPServer) Run(ctx context.Context) error {
    listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
    defer listener.Close()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            conn, err := listener.Accept()
            if err != nil {
                continue
            }
            go s.handleAssociation(ctx, conn)
        }
    }
}
```

### Testing Strategy

**1. Unit Tests:**
- Test each command struct independently
- Mock DICOM datasets and network operations
- Table-driven tests for flag parsing
- Error condition coverage

**2. Integration Tests:**
- Use test DICOM files from `testdata/`
- Spin up test DICOM server for network tests
- Test output formatting (JSON, table, CSV)
- Test batch operations with multiple files

**3. Benchmark Tests:**
- Measure performance of batch operations
- Profile memory usage for large files
- Test rate limiting accuracy
- Compare with DCMTK utilities

## Risks / Trade-offs

### Risk 1: External Dependencies

**Risk:** Adding multiple TUI dependencies increases binary size and potential security vulnerabilities.

**Mitigation:**
- All dependencies are well-maintained, widely used, and permissive licenses
- Regular dependency updates via Dependabot
- Vulnerability scanning with govulncheck
- Total binary size estimated at 15-20 MB (acceptable for CLI tool)

### Risk 2: DCMTK Compatibility

**Risk:** Native Go implementation may not match DCMTK behavior exactly.

**Mitigation:**
- Follow DICOM standard strictly, not DCMTK implementation details
- Comprehensive testing with real-world DICOM files
- Document any intentional differences
- Provide migration guide for DCMTK users

### Risk 3: Rate Limiting Accuracy

**Risk:** Rate limiting may not be precise, especially for variable file sizes.

**Mitigation:**
- Use proven `golang.org/x/time/rate` package
- Support both file-count and byte-count rate limiting
- Document expected accuracy (±5% is acceptable)
- Allow burst size configuration for flexibility

### Risk 4: Cross-Platform Compatibility

**Risk:** TUI libraries may behave differently on Windows.

**Mitigation:**
- Primary development and testing on macOS and Linux
- Windows support via WSL2 (recommended)
- CI testing on multiple platforms
- Document platform-specific limitations

## Migration Plan

**Phase 1: Foundation (Week 1)**
- Set up project structure
- Implement build system
- Create core CLI infrastructure
- Implement banner and theming

**Phase 2: Core Commands (Week 2-3)**
- Implement dump command
- Implement cecho command
- Implement cstore command (with rate limiting)
- Comprehensive testing

**Phase 3: Advanced Commands (Week 4)**
- Implement modify command
- Implement organize command
- Implement scp server command

**Phase 4: Polish & Documentation (Week 5)**
- Complete documentation
- Integration testing
- Performance optimization
- User acceptance testing

**Rollout:**
- Alpha release for internal testing
- Beta release for community feedback
- v1.0.0 stable release

**No backward compatibility concerns** as this is the first CLI release.

## Open Questions

1. **Q:** Should we support DICOM TLS (Secure Transport Connection Profile)?
   **A:** Not in initial release. Add in future version if user demand.

2. **Q:** Should we support DICOM query (C-FIND)?
   **A:** Future command. Not in initial scope.

3. **Q:** Should we support interactive mode for modify command?
   **A:** Yes, using huh for interactive tag editing. Add after basic implementation works.

4. **Q:** Should rate limiting apply to the entire batch or per-file?
   **A:** Per-file for simplicity. Can enhance to support bandwidth-based limiting in future.

5. **Q:** Should we create separate binaries for each command or single radx CLI?
   **A:** Single radx CLI with subcommands for better UX and maintenance.