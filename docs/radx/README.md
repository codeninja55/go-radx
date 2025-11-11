# RadX DICOM Utility CLI

RadX is a comprehensive command-line tool for working with DICOM medical imaging files. It provides utilities for
inspecting, transferring, modifying, and cataloguing DICOM datasets.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands](#commands)
- [Common Workflows](#common-workflows)
- [Database Schema](#database-schema)

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/codeninja55/go-radx.git
cd go-radx

# Build the radx CLI
cd cmd/radx
go build -o radx .

# Optionally, move to your PATH
mv radx /usr/local/bin/
```

### Using GoReleaser

Pre-built binaries are available for each release:

```bash
# Download from GitHub Releases
# Available for: darwin-arm64, darwin-amd64, linux-amd64, linux-arm64
```

## Quick Start

### Inspect a DICOM file

```bash
radx dicom dump file.dcm
```

### Filter specific tags

```bash
radx dicom dump file.dcm --tag PatientName --tag StudyDate
```

### Test DICOM connectivity

```bash
radx dicom echo --host localhost --port 11112
```

### Build a searchable catalogue

```bash
radx dicom catalogue /path/to/dicom/files --database my-files.db
```

### Query the catalogue

```bash
radx dicom catalogue --database my-files.db -q "Modality=CT"
radx dicom catalogue --database my-files.db --sql "SELECT patient_name, COUNT(*) FROM dicom_metadata GROUP BY patient_name"
```

## Commands

RadX provides eight main commands for working with DICOM files:

| Command | Description | Documentation |
|---------|-------------|---------------|
| `dump` | Inspect DICOM file contents | [dump.md](dump.md) |
| `echo` | Verify DICOM connectivity (C-ECHO) | [echo.md](echo.md) |
| `store` | Send DICOM files to server (C-STORE) | [store.md](store.md) |
| `modify` | Modify DICOM file tags | [modify.md](modify.md) |
| `organize` | Reorganize files by UID structure | [organize.md](organize.md) |
| `scp` | Run DICOM SCP server | [scp.md](scp.md) |
| `lookup` | Look up DICOM tag information | [lookup.md](lookup.md) |
| `catalogue` | Build and query DICOM file database | [catalogue.md](catalogue.md) |

## Common Workflows

### QA/QC Workflow

1. **Catalogue files** for quick searching:
   ```bash
   radx dicom catalogue /data/dicom --database qa.db
   ```

2. **Query for specific modalities**:
   ```bash
   radx dicom catalogue --database qa.db -q "Modality=CT"
   ```

3. **Inspect specific tags**:
   ```bash
   radx dicom dump suspicious-file.dcm -t PatientID -t StudyDate
   ```

### Archive Organization Workflow

1. **Organize files** by Study/Series/Instance hierarchy:
   ```bash
   radx dicom organize /incoming --output-dir /archive
   ```

2. **Catalogue organized** files:
   ```bash
   radx dicom catalogue /archive --database archive.db
   ```

3. **Query for studies** to verify:
   ```bash
   radx dicom catalogue --database archive.db --sql "SELECT DISTINCT study_instance_uid, study_description FROM dicom_metadata"
   ```

### PACS Integration Workflow

1. **Test connectivity**:
   ```bash
   radx dicom echo --host pacs.example.com --port 4242
   ```

2. **Send files with rate limiting**:
   ```bash
   radx dicom store --dir /data/to-send --host pacs.example.com --port 4242 --rate-limit 10
   ```

3. **Monitor transfer** (runs during store command automatically)

### Anonymization Preparation

1. **Catalogue files** to identify PHI tags:
   ```bash
   radx dicom catalogue /data/dicom --database phi-check.db
   ```

2. **Query for PHI presence**:
   ```bash
   radx dicom catalogue --database phi-check.db --sql "SELECT file_path, patient_name, patient_id FROM dicom_metadata WHERE patient_name != ''"
   ```

3. **Modify sensitive** tags:
   ```bash
   radx dicom modify file.dcm --output-dir /anonymized --insert "(0010,0010)=ANONYMOUS"
   ```

## Database Schema

The catalogue command creates a SQLite database with the following schema:

```sql
CREATE TABLE dicom_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path TEXT NOT NULL UNIQUE,
    file_name TEXT NOT NULL,
    file_size INTEGER,

    -- Patient Information
    patient_name TEXT,
    patient_id TEXT,
    patient_birth_date TEXT,
    patient_sex TEXT,

    -- Study Information
    study_instance_uid TEXT,
    study_date TEXT,
    study_description TEXT,

    -- Series Information
    series_instance_uid TEXT,
    series_number TEXT,
    series_description TEXT,

    -- Instance Information
    sop_instance_uid TEXT,
    sop_class_uid TEXT,
    instance_number TEXT,

    -- Equipment & Clinical
    modality TEXT,
    manufacturer TEXT,
    institution_name TEXT,
    accession_number TEXT,
    referring_physician_name TEXT,
    performing_physician_name TEXT,

    indexed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for fast queries
CREATE INDEX idx_patient_id ON dicom_metadata(patient_id);
CREATE INDEX idx_study_uid ON dicom_metadata(study_instance_uid);
CREATE INDEX idx_series_uid ON dicom_metadata(series_instance_uid);
CREATE INDEX idx_sop_uid ON dicom_metadata(sop_instance_uid);
CREATE INDEX idx_modality ON dicom_metadata(modality);
```

## Global Flags

All commands support these global flags:

- `--output <format>` - Output format: table, json, csv (default: table)
- `--log-level <level>` - Log level: trace, debug, info, warn, error, fatal (default: info)
- `--debug` - Enable debug mode (equivalent to --log-level debug)
- `--pretty` - Enable pretty formatting for logs (default: true)
- `--version` - Show version information

## Environment Variables

RadX respects the following environment variables:

- `RADX_LOG_LEVEL` - Default log level
- `RADX_OUTPUT_FORMAT` - Default output format
- `RADX_DATABASE` - Default catalogue database path

## Performance Considerations

### Large Datasets

- Use `--rate-limit` with store command to avoid overwhelming PACS systems
- Catalogue command skips already-indexed files by default
- SQL queries are limited to 1000 rows by default for safety

### Memory Usage

- Dump command processes files sequentially
- Catalogue command uses prepared statements for efficient batch inserts
- SCP server can handle concurrent connections

## Troubleshooting

### Connection Issues

```bash
# Test basic connectivity
radx dicom echo --host <hostname> --port <port>

# Check detailed logs
radx dicom echo --host <hostname> --port <port> --debug
```

### Database Issues

```bash
# Rebuild catalogue from scratch
radx dicom catalogue /path/to/files --database my.db --rebuild

# View database summary
radx dicom catalogue --database my.db
```

### File Access Issues

```bash
# Check file permissions
ls -la suspicious-file.dcm

# Validate DICOM file
radx dicom dump suspicious-file.dcm
```

## Contributing

Contributions are welcome! Please see the main repository README for contribution guidelines.

## License

See LICENSE file in the repository root.
