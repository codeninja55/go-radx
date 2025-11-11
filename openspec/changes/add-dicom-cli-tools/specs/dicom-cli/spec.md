# DICOM CLI Capability Specification

## ADDED Requirements

### Requirement: CLI Build Information

The CLI MUST inject and display build metadata including version, commit hash, and build date.

#### Scenario: Version flag displays build info
- **WHEN** user runs `radx --version` or `radx -V`
- **THEN** the system displays:
  - Application name
  - Version (git tag or branch)
  - Commit hash (short)
  - Build date (RFC3339 format)
  - Go version
  - Platform (OS/Arch)
  - Exit with code 0

#### Scenario: Build info printed on command execution
- **WHEN** any dicom subcommand is executed
- **THEN** the system prints ASCII banner "GO RADX DICOM UTIL"
- **AND** displays brief build info (version and commit)
- **AND** continues with command execution

### Requirement: CLI Global Configuration

The CLI MUST support global configuration options available to all subcommands.

#### Scenario: Global log level configuration
- **WHEN** user specifies `--log-level debug`
- **THEN** all commands output debug-level logs
- **AND** logs are written to stderr
- **AND** command output remains on stdout

#### Scenario: Global output format configuration
- **WHEN** user specifies `--format json`
- **THEN** all commands that produce tabular output render as JSON
- **AND** JSON is written to stdout
- **AND** JSON is properly formatted and valid

#### Scenario: Global output directory configuration
- **WHEN** user specifies `--output-dir /path/to/output`
- **THEN** commands that write files use the specified directory
- **AND** the directory is created if it doesn't exist
- **AND** an error is returned if the directory cannot be created

### Requirement: C-ECHO Verification Service

The CLI MUST provide a C-ECHO command to verify DICOM network connectivity.

#### Scenario: Successful C-ECHO verification
- **WHEN** user runs `radx dicom cecho --host pacs.example.com --port 11112`
- **THEN** the system establishes DICOM association
- **AND** sends C-ECHO request
- **AND** receives C-ECHO response with status 0x0000 (Success)
- **AND** releases association
- **AND** displays "C-ECHO successful" message
- **AND** exits with code 0

#### Scenario: Failed C-ECHO due to network error
- **WHEN** user runs `radx dicom cecho --host unreachable.example.com --port 11112`
- **AND** the host is unreachable or connection times out
- **THEN** the system displays clear error message
- **AND** provides troubleshooting hint
- **AND** exits with code 4 (network error)

#### Scenario: C-ECHO with custom AE titles
- **WHEN** user runs `radx dicom cecho --host pacs.example.com --called-ae PACS --calling-ae RADX`
- **THEN** the association uses Called AE Title "PACS"
- **AND** the association uses Calling AE Title "RADX"
- **AND** C-ECHO proceeds normally

#### Scenario: C-ECHO with timeout
- **WHEN** user runs `radx dicom cecho --host slow.example.com --timeout 5s`
- **AND** the server doesn't respond within 5 seconds
- **THEN** the operation times out
- **AND** displays timeout error message
- **AND** exits with code 4

### Requirement: C-STORE Storage Service

The CLI MUST provide a C-STORE command to send DICOM files to a PACS.

#### Scenario: Store single DICOM file
- **WHEN** user runs `radx dicom cstore --host pacs.example.com file.dcm`
- **THEN** the system establishes DICOM association
- **AND** sends file.dcm via C-STORE
- **AND** receives C-STORE response with status 0x0000
- **AND** releases association
- **AND** displays success message
- **AND** exits with code 0

#### Scenario: Store directory of DICOM files with progress
- **WHEN** user runs `radx dicom cstore --host pacs.example.com --dir /path/to/studies/`
- **THEN** the system finds all DICOM files recursively
- **AND** establishes DICOM association
- **AND** sends each file via C-STORE
- **AND** displays progress bar showing current file and percentage
- **AND** releases association
- **AND** displays summary (success count, failure count)
- **AND** exits with code 0 if all succeeded, 1 if any failed

#### Scenario: Store with rate limiting by file count
- **WHEN** user runs `radx dicom cstore --host pacs.example.com --dir /studies/ --rate-limit 10`
- **THEN** the system limits transmission to maximum 10 files per second
- **AND** applies rate limiting between individual C-STORE operations
- **AND** displays progress with rate information
- **AND** completes all transfers successfully

#### Scenario: Store with rate limiting by bandwidth
- **WHEN** user runs `radx dicom cstore --host pacs.example.com --dir /studies/ --rate-limit-mb 50`
- **THEN** the system limits transmission to maximum 50 MB per second
- **AND** calculates rate based on file sizes
- **AND** throttles transmission accordingly
- **AND** displays bandwidth usage in progress indicator

#### Scenario: Store with burst size configuration
- **WHEN** user runs `radx dicom cstore --rate-limit 5 --burst-size 10 file1.dcm file2.dcm ... file15.dcm`
- **THEN** the system sends first 10 files immediately (burst)
- **AND** throttles remaining files at 5 files/second
- **AND** completes all transfers

#### Scenario: Rate limiting with zero limit (unlimited)
- **WHEN** user runs `radx dicom cstore --rate-limit 0 --dir /studies/`
- **THEN** the system sends files as fast as possible
- **AND** no rate limiting is applied
- **AND** transfers complete at maximum speed

#### Scenario: Store with transfer syntax selection
- **WHEN** user runs `radx dicom cstore --host pacs.example.com --transfer-syntax 1.2.840.10008.1.2.1 file.dcm`
- **THEN** the association negotiates Explicit VR Little Endian transfer syntax
- **AND** the file is transmitted with negotiated transfer syntax
- **AND** C-STORE succeeds

#### Scenario: Store with failed file
- **WHEN** user runs `radx dicom cstore --host pacs.example.com file1.dcm invalid.dcm file3.dcm`
- **AND** invalid.dcm is not a valid DICOM file
- **THEN** the system reports error for invalid.dcm
- **AND** continues processing file3.dcm
- **AND** displays summary with 2 successful, 1 failed
- **AND** exits with code 1

### Requirement: DICOM File Inspection (Dump)

The CLI MUST provide a dump command to inspect DICOM file contents.

#### Scenario: Dump single DICOM file as table
- **WHEN** user runs `radx dicom dump file.dcm`
- **THEN** the system parses the DICOM file
- **AND** displays tags in ASCII table format with columns: Tag, VR, Name, Value
- **AND** includes File Meta Information (group 0002)
- **AND** includes Dataset elements
- **AND** exits with code 0

#### Scenario: Dump single DICOM file as JSON
- **WHEN** user runs `radx dicom dump --format json file.dcm`
- **THEN** the system parses the DICOM file
- **AND** outputs valid JSON to stdout
- **AND** JSON includes all DICOM tags with tag, vr, name, value fields
- **AND** exits with code 0

#### Scenario: Dump directory of DICOM files
- **WHEN** user runs `radx dicom dump --dir /path/to/studies/ --recursive`
- **THEN** the system finds all .dcm files recursively
- **AND** parses each file
- **AND** displays metadata for each file
- **AND** separates files with clear visual divider
- **AND** exits with code 0

#### Scenario: Dump with pixel data processing
- **WHEN** user runs `radx dicom dump --process-pixel-data file.dcm`
- **THEN** the system parses pixel data element
- **AND** decompresses pixel data if compressed
- **AND** displays pixel data statistics (dimensions, depth, samples)
- **AND** does not extract pixel values by default

#### Scenario: Dump with pixel data extraction
- **WHEN** user runs `radx dicom dump --process-pixel-data --store-pixel-data --output-dir /tmp file.dcm`
- **THEN** the system extracts pixel data
- **AND** writes pixel data to /tmp/file.raw or /tmp/file.png
- **AND** displays confirmation message
- **AND** continues with metadata display

#### Scenario: Dump invalid DICOM file
- **WHEN** user runs `radx dicom dump invalid.dcm`
- **AND** invalid.dcm is not a valid DICOM file
- **THEN** the system displays error message
- **AND** provides diagnostic information (file size, first bytes)
- **AND** exits with code 3 (DICOM parsing error)

#### Scenario: Dump with CSV output
- **WHEN** user runs `radx dicom dump --format csv file1.dcm file2.dcm file3.dcm`
- **THEN** the system outputs CSV to stdout
- **AND** CSV has header row: Tag,VR,Name,Value,File
- **AND** each DICOM tag appears as a CSV row
- **AND** File column identifies source file
- **AND** CSV is properly quoted and escaped

### Requirement: DICOM File Modification

The CLI MUST provide a modify command to change DICOM tag values.

#### Scenario: Insert or update single tag
- **WHEN** user runs `radx dicom modify -i "0010,0010=Doe^John" input.dcm -o output.dcm`
- **THEN** the system parses input.dcm
- **AND** sets Patient Name (0010,0010) to "Doe^John"
- **AND** writes modified file to output.dcm
- **AND** preserves all other tags
- **AND** exits with code 0

#### Scenario: Delete single tag
- **WHEN** user runs `radx dicom modify -e "0010,0010" input.dcm -o output.dcm`
- **THEN** the system removes Patient Name (0010,0010) tag
- **AND** writes modified file without the tag
- **AND** preserves all other tags
- **AND** exits with code 0

#### Scenario: Delete tag group with wildcard
- **WHEN** user runs `radx dicom modify -e "0010,xxxx" input.dcm -o output.dcm`
- **AND** user confirms the operation (interactive prompt)
- **THEN** the system removes all tags in group 0010 (patient information)
- **AND** writes modified file
- **AND** displays count of removed tags
- **AND** exits with code 0

#### Scenario: Regenerate Study Instance UID
- **WHEN** user runs `radx dicom modify --regenerate-study-uid input.dcm -o output.dcm`
- **THEN** the system generates new Study Instance UID
- **AND** replaces (0020,000D) Study Instance UID with new value
- **AND** preserves all other tags
- **AND** writes modified file
- **AND** displays old and new UIDs
- **AND** exits with code 0

#### Scenario: Regenerate all UIDs
- **WHEN** user runs `radx dicom modify --regenerate-all-uids input.dcm -o output.dcm`
- **THEN** the system generates new Study Instance UID
- **AND** generates new Series Instance UID
- **AND** generates new SOP Instance UID
- **AND** writes modified file
- **AND** displays all old and new UIDs
- **AND** exits with code 0

#### Scenario: Batch modify directory
- **WHEN** user runs `radx dicom modify --dir /input/ --output-dir /output/ -i "0008,0070=ACME Medical"`
- **THEN** the system processes all DICOM files in /input/ recursively
- **AND** applies modification to each file
- **AND** writes modified files to /output/ preserving directory structure
- **AND** displays progress bar
- **AND** displays summary (success count, failure count)
- **AND** exits with code 0

#### Scenario: Modify with backup
- **WHEN** user runs `radx dicom modify --backup -i "0010,0010=Anonymous" file.dcm`
- **THEN** the system creates backup file.dcm.bak
- **AND** modifies file.dcm in place
- **AND** displays confirmation message
- **AND** exits with code 0

#### Scenario: Modify in place without output flag
- **WHEN** user runs `radx dicom modify -i "0010,0010=Doe^John" input.dcm`
- **AND** no --output or --output-dir flag is provided
- **THEN** the system prompts user to confirm in-place modification
- **AND** if confirmed, modifies input.dcm
- **AND** if declined, exits without changes
- **AND** returns appropriate exit code

#### Scenario: Multiple modifications in single command
- **WHEN** user runs `radx dicom modify -i "0010,0010=Doe^John" -i "0010,0030=19800101" -e "0010,0040" input.dcm -o output.dcm`
- **THEN** the system applies all modifications in order
- **AND** inserts/updates Patient Name
- **AND** inserts/updates Patient Birth Date
- **AND** removes Patient Sex
- **AND** writes modified file
- **AND** exits with code 0

### Requirement: DICOM SCP Server

The CLI MUST provide an SCP server command to receive DICOM objects.

#### Scenario: Start SCP server with defaults
- **WHEN** user runs `radx dicom scp --output-dir /dicom-data`
- **THEN** the system starts DICOM SCP server on port 11112
- **AND** uses AE Title "RADX_SCP"
- **AND** accepts only Secondary Capture SOP Classes by default
- **AND** displays server status (port, AE title, accepted SOP Classes)
- **AND** runs until SIGTERM or SIGINT received

#### Scenario: Receive DICOM object and organize by UID
- **WHEN** SCP server receives C-STORE request
- **AND** the SOP Class is accepted
- **THEN** the system receives DICOM dataset
- **AND** extracts Study Instance UID
- **AND** extracts Series Instance UID
- **AND** extracts SOP Instance UID
- **AND** creates directory structure: `/dicom-data/<study-uid>/<series-uid>/`
- **AND** writes file as `<sop-instance-uid>.dcm`
- **AND** sends C-STORE response with status 0x0000 (Success)
- **AND** logs receipt (timestamp, source AE, study/series/instance UIDs)

#### Scenario: Reject unsupported SOP Class
- **WHEN** SCP server receives C-STORE request
- **AND** the SOP Class is not in accepted list
- **THEN** the system rejects C-STORE
- **AND** sends C-STORE response with status 0xA500 (Refused - SOP Class Not Supported)
- **AND** logs rejection (timestamp, source AE, rejected SOP Class)
- **AND** does not store file

#### Scenario: SCP server with custom SOP Class filter
- **WHEN** user runs `radx dicom scp --output-dir /data --accept-sop-class 1.2.840.10008.5.1.4.1.1.2 --accept-sop-class 1.2.840.10008.5.1.4.1.1.4`
- **THEN** the system accepts CT Image Storage (1.2.840.10008.5.1.4.1.1.2)
- **AND** accepts MR Image Storage (1.2.840.10008.5.1.4.1.1.4)
- **AND** rejects all other SOP Classes
- **AND** logs accepted SOP Classes on startup

#### Scenario: SCP server accepts all SOP Classes
- **WHEN** user runs `radx dicom scp --output-dir /data --accept-all`
- **THEN** the system accepts any SOP Class
- **AND** does not filter by SOP Class
- **AND** logs "Accepting all SOP Classes" on startup

#### Scenario: SCP server with custom port and AE title
- **WHEN** user runs `radx dicom scp --port 104 --ae-title MY_SCP --output-dir /data`
- **THEN** the system listens on port 104
- **AND** uses AE Title "MY_SCP"
- **AND** accepts associations with Called AE Title "MY_SCP"
- **AND** rejects associations with different Called AE Title

#### Scenario: SCP server handles file collision
- **WHEN** SCP server receives DICOM object
- **AND** file `<sop-instance-uid>.dcm` already exists
- **THEN** the system appends sequential number to filename
- **AND** writes as `<sop-instance-uid>-1.dcm`
- **AND** logs collision and resolution
- **AND** sends C-STORE response with status 0x0000

#### Scenario: SCP server graceful shutdown
- **WHEN** SCP server receives SIGTERM or SIGINT
- **THEN** the system stops accepting new associations
- **AND** waits for active associations to complete (max 30 seconds)
- **AND** closes all connections
- **AND** logs shutdown message
- **AND** exits with code 0

#### Scenario: SCP server with association limit
- **WHEN** user runs `radx dicom scp --max-associations 10 --output-dir /data`
- **AND** 10 associations are already active
- **AND** new association request arrives
- **THEN** the system rejects new association
- **AND** sends A-ASSOCIATE-RJ (reject)
- **AND** logs rejection due to limit

### Requirement: DICOM Directory Organization

The CLI MUST provide an organize command to restructure DICOM files by UID.

#### Scenario: Organize directory with copy (default)
- **WHEN** user runs `radx dicom organize --input-dir /unorganized --output-dir /organized`
- **THEN** the system scans /unorganized recursively for DICOM files
- **AND** parses each file to extract Study/Series/Instance UIDs
- **AND** copies files to `/organized/<study-uid>/<series-uid>/<instance-uid>.dcm`
- **AND** displays progress bar
- **AND** leaves original files unchanged
- **AND** displays summary (files processed, files copied, errors)
- **AND** exits with code 0

#### Scenario: Organize directory with move
- **WHEN** user runs `radx dicom organize --input-dir /unorganized --output-dir /organized --move`
- **THEN** the system scans /unorganized recursively for DICOM files
- **AND** parses each file to extract Study/Series/Instance UIDs
- **AND** moves files to `/organized/<study-uid>/<series-uid>/<instance-uid>.dcm`
- **AND** removes original files after successful move
- **AND** displays progress bar
- **AND** displays summary
- **AND** exits with code 0

#### Scenario: Organize with file collision
- **WHEN** organizing directory
- **AND** target file `<instance-uid>.dcm` already exists
- **THEN** the system compares file content (MD5 or size)
- **AND** if identical, skips file and logs "duplicate"
- **AND** if different, appends sequential number: `<instance-uid>-1.dcm`
- **AND** logs collision resolution
- **AND** continues processing

#### Scenario: Organize with invalid DICOM file
- **WHEN** organizing directory
- **AND** non-DICOM file is encountered
- **THEN** the system logs error for invalid file
- **AND** continues processing remaining files
- **AND** includes error in summary
- **AND** exits with code 0 if other files succeeded

#### Scenario: Organize with missing UIDs
- **WHEN** organizing directory
- **AND** DICOM file is missing Study Instance UID
- **THEN** the system logs error "missing Study Instance UID"
- **AND** skips file
- **AND** continues processing
- **AND** includes error in summary

#### Scenario: Organize with dry-run
- **WHEN** user runs `radx dicom organize --input-dir /in --output-dir /out --dry-run`
- **THEN** the system scans and parses all files
- **AND** displays planned operations (copy/move paths)
- **AND** does NOT copy or move any files
- **AND** displays summary of what would be done
- **AND** exits with code 0

#### Scenario: Organize with JSON output
- **WHEN** user runs `radx dicom organize --input-dir /in --output-dir /out --format json`
- **THEN** the system outputs JSON to stdout
- **AND** JSON includes array of operations with source, target, status
- **AND** performs organization normally
- **AND** JSON is valid and properly formatted

### Requirement: Output Format Consistency

All commands that produce tabular data MUST support multiple output formats.

#### Scenario: Command supports JSON output
- **WHEN** user runs any command with `--format json`
- **THEN** the output is valid JSON
- **AND** JSON is written to stdout
- **AND** JSON is properly formatted with consistent structure
- **AND** exit code reflects operation success/failure, not format

#### Scenario: Command supports table output
- **WHEN** user runs any command with `--format table` or default
- **THEN** the output is ASCII table using simpletable
- **AND** table has clear headers and borders
- **AND** table columns are aligned
- **AND** table fits terminal width if possible

#### Scenario: Command supports CSV output
- **WHEN** user runs any command with `--format csv`
- **THEN** the output is valid CSV
- **AND** CSV has header row
- **AND** values are properly quoted and escaped
- **AND** CSV can be imported into spreadsheet applications

### Requirement: Error Handling and Exit Codes

The CLI MUST provide clear error messages and appropriate exit codes.

#### Scenario: Success exits with code 0
- **WHEN** any command completes successfully
- **THEN** the system exits with code 0

#### Scenario: Invalid arguments exit with code 2
- **WHEN** user provides invalid command-line arguments
- **THEN** the system displays error message
- **AND** displays usage hint or help
- **AND** exits with code 2

#### Scenario: DICOM parsing error exits with code 3
- **WHEN** command encounters invalid DICOM file
- **THEN** the system displays error message with file path
- **AND** provides diagnostic information
- **AND** exits with code 3

#### Scenario: Network error exits with code 4
- **WHEN** command encounters network error (connection failed, timeout, etc.)
- **THEN** the system displays clear error message
- **AND** provides troubleshooting hints
- **AND** exits with code 4

#### Scenario: File I/O error exits with code 5
- **WHEN** command encounters file read/write error
- **THEN** the system displays error with file path and OS error
- **AND** provides actionable guidance (check permissions, disk space)
- **AND** exits with code 5

#### Scenario: General error exits with code 1
- **WHEN** command encounters any other error
- **THEN** the system displays error message
- **AND** exits with code 1

### Requirement: Logging Configuration

The CLI MUST provide configurable structured logging.

#### Scenario: Default log level
- **WHEN** user runs any command without --log-level
- **THEN** the system logs at INFO level
- **AND** displays informational messages, warnings, and errors
- **AND** does not display debug or trace messages

#### Scenario: Debug log level
- **WHEN** user runs command with `--log-level debug`
- **THEN** the system logs at DEBUG level
- **AND** displays detailed diagnostic information
- **AND** includes function names and line numbers
- **AND** logs DICOM tag parsing details

#### Scenario: Error log level
- **WHEN** user runs command with `--log-level error`
- **THEN** the system logs only ERROR level and above
- **AND** does not display informational or debug messages
- **AND** suitable for production/automation

#### Scenario: Pretty logging format
- **WHEN** user runs command with `--pretty` (default)
- **THEN** logs are formatted with colors and icons
- **AND** timestamps are human-readable
- **AND** log levels are visually distinct

#### Scenario: JSON logging format
- **WHEN** user runs command with `--no-pretty`
- **THEN** logs are formatted as JSON
- **AND** each log entry is a single JSON object per line
- **AND** suitable for log aggregation systems

### Requirement: Help and Documentation

The CLI MUST provide comprehensive inline help.

#### Scenario: Global help
- **WHEN** user runs `radx --help` or `radx -h`
- **THEN** the system displays main help screen
- **AND** lists all available commands
- **AND** shows global flags
- **AND** exits with code 0

#### Scenario: Subcommand help
- **WHEN** user runs `radx dicom --help`
- **THEN** the system displays DICOM subcommand help
- **AND** lists all DICOM commands
- **AND** shows available flags
- **AND** exits with code 0

#### Scenario: Command-specific help
- **WHEN** user runs `radx dicom cstore --help`
- **THEN** the system displays cstore command help
- **AND** describes command purpose
- **AND** lists all flags with descriptions
- **AND** provides usage examples
- **AND** exits with code 0

#### Scenario: Invalid command shows help
- **WHEN** user runs `radx dicom invalid-command`
- **THEN** the system displays error "unknown command"
- **AND** suggests similar commands if available
- **AND** displays brief help or usage
- **AND** exits with code 2