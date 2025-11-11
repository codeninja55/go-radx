# go-radx

[![Go Version](https://img.shields.io/badge/go-1.23+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

A comprehensive Go library and CLI toolkit for working with DICOM (Digital Imaging and Communications in Medicine) files,
implementing native DICOM parsing, DIMSE protocol support, and powerful utilities for medical imaging workflows.

## Features

### Core Library (`github.com/codeninja55/go-radx`)

- **Native DICOM Parsing**: Pure Go implementation without C dependencies
- **DIMSE Protocol**: Full support for DICOM network operations (C-ECHO, C-STORE, C-FIND, C-MOVE)
- **Transfer Syntax Support**: Multiple encodings including JPEG 2000, JPEG Lossless
- **Tag Dictionary**: Complete DICOM data dictionary with 6000+ standard tags
- **Value Representations**: Full support for all DICOM VRs (Value Representations)
- **Pixel Data Handling**: Native pixel data extraction and processing
- **SCP/SCU**: Both Service Class Provider and Service Class User implementations

### RadX CLI (`cmd/radx`)

A powerful command-line tool for DICOM file manipulation and analysis:

| Command | Description |
|---------|-------------|
| **dump** | Inspect DICOM file contents with tag filtering |
| **echo** | Verify DICOM connectivity (C-ECHO SCU) |
| **store** | Send files to PACS (C-STORE SCU) with rate limiting |
| **modify** | Modify DICOM tags and regenerate UIDs |
| **organize** | Reorganize files by Study/Series/Instance hierarchy |
| **scp** | Run DICOM SCP server (C-ECHO and C-STORE) |
| **lookup** | Look up DICOM tag information |
| **catalogue** | Build and query SQLite database of DICOM metadata |

## Quick Start

### Installation

```bash
# Install from source
git clone https://github.com/codeninja55/go-radx.git
cd go-radx/cmd/radx
go install

# Or build manually
go build -o radx .
```

### Basic Usage

```bash
# Inspect a DICOM file
radx dicom dump file.dcm

# Test PACS connectivity
radx dicom echo --host pacs.example.com --port 11112

# Send files to PACS with rate limiting
radx dicom store --dir /data/dicom --host pacs.example.com --rate-limit 10

# Build a searchable database
radx dicom catalogue /data/dicom --database archive.db

# Query the database
radx dicom catalogue --database archive.db -q "Modality=CT"
radx dicom catalogue --database archive.db --sql "SELECT patient_name, COUNT(*) FROM dicom_metadata GROUP BY patient_name"
```

## Documentation

- **CLI Documentation**: See [docs/radx/](../../docs/radx/) for detailed command documentation
  - [dump](../../docs/radx/dump.md) - Inspect DICOM file contents
  - [echo](../../docs/radx/echo.md) - Verify DICOM connectivity (C-ECHO)
  - [store](../../docs/radx/store.md) - Send files to PACS (C-STORE)
  - [modify](../../docs/radx/modify.md) - Modify DICOM tags and UIDs
  - [organize](../../docs/radx/organize.md) - Reorganize files by Study/Series/Instance
  - [scp](../../docs/radx/scp.md) - Run DICOM SCP server
  - [catalogue](../../docs/radx/catalogue.md) - Build SQLite database of metadata
  - [lookup](../../docs/radx/lookup.md) - Look up DICOM tag information
- **API Documentation**: Run `go doc github.com/codeninja55/go-radx/dicom`

## Key Features

### 1. Tag Filtering (dump command)

```bash
# Filter by tag ID, hex code, or keyword
radx dicom dump file.dcm --tag PatientName --tag "(0010,0020)" --tag 00080060

# Common filters
radx dicom dump file.dcm -t PatientName -t PatientID -t StudyDate -t Modality
```

### 2. SQLite Catalogue with SQL Queries

```bash
# Index DICOM files
radx dicom catalogue /data/dicom --database my-archive.db

# Keyword queries
radx dicom catalogue --database my-archive.db -q "PatientID=12345"

# Raw SQL queries (safe, SELECT-only)
radx dicom catalogue --database my-archive.db --sql "
  SELECT modality, COUNT(*) as count, AVG(file_size) as avg_size
  FROM dicom_metadata
  GROUP BY modality
  ORDER BY count DESC"
```

### 3. DICOM Network Operations

```bash
# C-ECHO verification
radx dicom echo --host pacs.example.com --port 11112

# C-STORE with rate limiting
radx dicom store --dir /data/to-send \
  --host pacs.example.com \
  --rate-limit 10 \
  --rate-limit-bytes 5.0

# Run SCP server
radx dicom scp --port 11112 --output-dir /received \
  --accept-echo \
  --auto-organize
```

## Safety and Security

### SQL Query Safety

The catalogue command implements multiple safety layers for SQL queries:

1. **SELECT-only**: Only SELECT statements are permitted
2. **Keyword filtering**: Blocks dangerous keywords (DROP, DELETE, INSERT, UPDATE, etc.)
3. **Result limiting**: Maximum 1000 rows returned
4. **No destructive operations**: Database is read-only for SQL queries

## Recent Additions

- ✅ Tag filtering in dump command
- ✅ SQL query support in catalogue command
- ✅ Tag lookup command with built-in dictionary
- ✅ Enhanced rate limiting (files/sec and MB/sec)
- ✅ SCP server with auto-organize
- ✅ Comprehensive CLI documentation

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: [docs/radx/](docs/radx/)
- **Issues**: [GitHub Issues](https://github.com/codeninja55/go-radx/issues)
