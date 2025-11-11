# Implementation Tasks

## 1. Project Setup

- [ ] 1.1 Create `cmd/radx/` directory structure
- [ ] 1.2 Initialize `cmd/radx/go.mod` with dependencies
- [ ] 1.3 Add CLI dependencies to go.mod:
  - [ ] kong for CLI parsing
  - [ ] charmbracelet/* for TUI
  - [ ] simpletable for tables
  - [ ] go-figure for ASCII art
- [ ] 1.4 Update root `Makefile` with CLI build targets
- [ ] 1.5 Update `mise.toml` with CLI tasks

## 2. Core CLI Infrastructure

- [ ] 2.1 Implement `cmd/radx/main.go` with version injection variables
- [ ] 2.2 Implement `cmd/radx/internal/build/info.go`:
  - [ ] Build info struct with all metadata fields
  - [ ] SetBuildInfo() function
  - [ ] PrintBuildInfo() function
  - [ ] JSON serialization support
- [ ] 2.3 Implement `cmd/radx/internal/cli/cli.go`:
  - [ ] Kong-based CLI struct with embedded global config
  - [ ] ParseArgs() function returning config and context
  - [ ] Run() function with error handling
  - [ ] Command registration for all DICOM commands
- [ ] 2.4 Implement `cmd/radx/internal/config/config.go`:
  - [ ] DicomUtilContext struct with global flags
  - [ ] Version flag handler
  - [ ] Log level configuration
  - [ ] Output directory configuration
  - [ ] DICOM validation options

## 3. TUI Components

- [ ] 3.1 Implement `cmd/radx/internal/dicom/ui/banner.go`:
  - [ ] ASCII art generation for "GO RADX DICOM UTIL"
  - [ ] Styled output using lipgloss
  - [ ] Conditional display based on context
- [ ] 3.2 Implement `cmd/radx/internal/dicom/ui/theme.go`:
  - [ ] Custom color scheme (matching reference or go-radx branding)
  - [ ] huh.Theme configuration
  - [ ] lipgloss Style definitions
- [ ] 3.3 Implement `cmd/radx/internal/dicom/ui/progress.go`:
  - [ ] Progress bar for batch operations
  - [ ] Spinner for long operations
  - [ ] Status messages
- [ ] 3.4 Implement `cmd/radx/internal/dicom/ui/table.go`:
  - [ ] simpletable wrapper functions
  - [ ] DICOM metadata table rendering
  - [ ] Styled table output

## 4. Shared Helpers

- [ ] 4.1 Implement `cmd/radx/internal/dicom/commands/helpers.go`:
  - [ ] parseDicom() - Parse single DICOM file
  - [ ] listDicomFiles() - Find DICOM files in directory
  - [ ] validateDicomFile() - Validate DICOM file structure
  - [ ] formatDicomValue() - Format DICOM values for display
  - [ ] createOutputDirectory() - Ensure output directory exists
  - [ ] generateUID() - Generate DICOM UIDs
- [ ] 4.2 Implement output formatting utilities:
  - [ ] renderAsJSON() - JSON output
  - [ ] renderAsTable() - Table output
  - [ ] renderAsCSV() - CSV output

## 5. Command: cecho (DICOM Verification)

- [ ] 5.1 Implement `cmd/radx/internal/dicom/commands/cecho.go`:
  - [ ] Command struct with Kong tags
  - [ ] Run() method implementation
  - [ ] Connection parameters (host, port, AE titles)
  - [ ] Timeout configuration
- [ ] 5.2 Integrate with dimse.Client for C-ECHO operation
- [ ] 5.3 Add progress feedback and status display
- [ ] 5.4 Implement comprehensive error handling
- [ ] 5.5 Add unit tests for cecho command
- [ ] 5.6 Add integration tests with test DICOM server

## 6. Command: cstore (DICOM Storage)

- [ ] 6.1 Implement `cmd/radx/internal/dicom/commands/cstore.go`:
  - [ ] Command struct with file/directory input
  - [ ] Run() method implementation
  - [ ] Connection parameters
  - [ ] Transfer syntax selection
  - [ ] Batch operation support
  - [ ] Rate limiting flags (rate-limit, rate-limit-mb, burst-size)
- [ ] 6.2 Integrate with dimse.Client for C-STORE operation
- [ ] 6.3 Implement rate limiting using golang.org/x/time/rate:
  - [ ] File-count based rate limiting (files/second)
  - [ ] Bandwidth-based rate limiting (MB/second)
  - [ ] Configurable burst size
  - [ ] Rate limiter integration with C-STORE loop
- [ ] 6.4 Add progress tracking for batch uploads:
  - [ ] Current file count and total
  - [ ] Current bandwidth usage (if rate limiting enabled)
  - [ ] Estimated time remaining
- [ ] 6.5 Implement retry logic for failed transfers
- [ ] 6.6 Add validation before sending
- [ ] 6.7 Add unit tests for cstore command:
  - [ ] Test rate limiter configuration
  - [ ] Test rate limiting accuracy
  - [ ] Test burst behavior
- [ ] 6.8 Add integration tests with test DICOM server
- [ ] 6.9 Add benchmark tests for rate limiting performance

## 7. Command: dump (DICOM Inspection)

- [ ] 7.1 Implement `cmd/radx/internal/dicom/commands/dump.go`:
  - [ ] Command struct with file/directory input
  - [ ] Run() method implementation
  - [ ] Output format selection (JSON, table, CSV)
  - [ ] Process pixel data flag
  - [ ] Store pixel data flag
  - [ ] Recursive directory traversal
- [ ] 7.2 Use dicom.ParseFile() for parsing
- [ ] 7.3 Implement tag extraction and formatting
- [ ] 7.4 Add table rendering with simpletable
- [ ] 7.5 Add JSON output support
- [ ] 7.6 Add CSV output support
- [ ] 7.7 Add unit tests for dump command
- [ ] 7.8 Add integration tests with test DICOM files

## 8. Command: modify (DICOM Modification)

- [ ] 8.1 Implement `cmd/radx/internal/dicom/commands/modify.go`:
  - [ ] Command struct with file/directory input
  - [ ] Run() method implementation
  - [ ] Tag insertion/update operations
  - [ ] Tag deletion operations
  - [ ] UID regeneration options
  - [ ] Batch operation support
  - [ ] Backup creation option
- [ ] 8.2 Use dicom.DataSet for modifications
- [ ] 8.3 Implement tag parsing from CLI flags
- [ ] 8.4 Add validation after modifications
- [ ] 8.5 Implement UID regeneration logic
- [ ] 8.6 Add progress tracking for batch operations
- [ ] 8.7 Add unit tests for modify command
- [ ] 8.8 Add integration tests with test DICOM files

## 9. Command: scp (DICOM SCP Server)

- [ ] 9.1 Implement `cmd/radx/internal/dicom/commands/scp.go`:
  - [ ] Command struct with server configuration
  - [ ] Run() method implementation
  - [ ] Port and AE title configuration
  - [ ] SOP Class filter configuration (defaults to Secondary Capture)
  - [ ] Output directory configuration
  - [ ] Daemon mode support
  - [ ] Graceful shutdown handling
- [ ] 9.2 Integrate with dimse/scp for server implementation
- [ ] 9.3 Implement file organization by Study/Series/Instance UID
- [ ] 9.4 Add logging for received DICOM objects
- [ ] 9.5 Implement association management
- [ ] 9.6 Add signal handling (SIGTERM, SIGINT)
- [ ] 9.7 Add unit tests for scp command
- [ ] 9.8 Add integration tests with test DICOM client

## 10. Command: organize (Directory Organization)

- [ ] 10.1 Implement `cmd/radx/internal/dicom/commands/organize.go`:
  - [ ] Command struct with input/output directories
  - [ ] Run() method implementation
  - [ ] Recursive directory traversal
  - [ ] Copy vs move operation flag
  - [ ] Conflict resolution strategy
  - [ ] Progress tracking
- [ ] 10.2 Use dicom.ParseDirectory() for batch parsing
- [ ] 10.3 Implement UID extraction logic
- [ ] 10.4 Create directory structure: `<study>/<series>/<instance>.dcm`
- [ ] 10.5 Add file operation (copy/move) logic
- [ ] 10.6 Implement collision handling
- [ ] 10.7 Add dry-run mode
- [ ] 10.8 Add unit tests for organize command
- [ ] 10.9 Add integration tests with test DICOM files

## 11. Build System

- [ ] 11.1 Update `Makefile`:
  - [ ] Add `build-cli` target with version injection
  - [ ] Add `install-cli` target
  - [ ] Add `clean-cli` target
  - [ ] Set LDFLAGS for version, commit, date injection
- [ ] 11.2 Update `mise.toml`:
  - [ ] Add `radx:build` task
  - [ ] Add `radx:test` task
  - [ ] Add `radx:install` task
- [ ] 11.3 Add `.goreleaser.yml` for release automation
- [ ] 11.4 Test build on multiple platforms (macOS, Linux)

## 12. Documentation

- [ ] 12.1 Create `docs/user-guide/cli/` directory
- [ ] 12.2 Write `docs/user-guide/cli/overview.md`:
  - [ ] Introduction to radx DICOM CLI
  - [ ] Installation instructions
  - [ ] Common usage patterns
- [ ] 12.3 Write command-specific guides:
  - [ ] `docs/user-guide/cli/cecho.md`
  - [ ] `docs/user-guide/cli/cstore.md`
  - [ ] `docs/user-guide/cli/dump.md`
  - [ ] `docs/user-guide/cli/modify.md`
  - [ ] `docs/user-guide/cli/scp.md`
  - [ ] `docs/user-guide/cli/organize.md`
- [ ] 12.4 Add examples to each command guide
- [ ] 12.5 Update `README.md` with CLI section
- [ ] 12.6 Add inline help text to all commands and flags
- [ ] 12.7 Generate man pages (optional)

## 13. Testing

- [ ] 13.1 Unit tests for all command implementations
- [ ] 13.2 Integration tests with test DICOM files from `testdata/`
- [ ] 13.3 Integration tests for networking commands (requires test server)
- [ ] 13.4 Table-driven tests for edge cases
- [ ] 13.5 Test error handling paths
- [ ] 13.6 Test output formatting (JSON, table, CSV)
- [ ] 13.7 Benchmark tests for performance-critical operations
- [ ] 13.8 Ensure 80%+ code coverage

## 14. Quality Assurance

- [ ] 14.1 Run `mise fmt` and ensure clean formatting
- [ ] 14.2 Run `mise lint` and fix all issues
- [ ] 14.3 Run `mise test` and ensure all tests pass
- [ ] 14.4 Run `mise test:coverage` and verify coverage targets
- [ ] 14.5 Test CLI on macOS (local)
- [ ] 14.6 Test CLI on Linux (CI or container)
- [ ] 14.7 Test with real DICOM files from various modalities
- [ ] 14.8 Verify build info injection works correctly
- [ ] 14.9 Test all output formats
- [ ] 14.10 Verify ASCII banner displays correctly on different terminals

## 15. CI/CD with GoReleaser

- [ ] 15.1 Update `.github/workflows/ci.yml`:
  - [ ] Add CLI build step
  - [ ] Add CLI test step
  - [ ] Add artifact upload for binaries
- [ ] 15.2 Add `.goreleaser.yml` configuration:
  - [ ] Configure builds for multiple platforms (macOS, Linux, Windows)
  - [ ] Configure archives with checksums
  - [ ] Configure changelog generation
  - [ ] Configure GitHub release creation
- [ ] 15.3 Create `.github/workflows/release.yml`:
  - [ ] Trigger on git tags (v*.*.*)
  - [ ] Use GoReleaser action
  - [ ] Upload release assets
  - [ ] Generate release notes
- [ ] 15.4 Configure cross-platform builds (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64)
- [ ] 15.5 Test release workflow with pre-release tag

## 16. Final Review

- [ ] 16.1 Code review by team
- [ ] 16.2 Documentation review
- [ ] 16.3 User acceptance testing
- [ ] 16.4 Performance testing with large DICOM datasets
- [ ] 16.5 Security review (input validation, error handling)
- [ ] 16.6 Update CHANGELOG.md
- [ ] 16.7 Tag release version