# radx dicom store

Send DICOM files to a PACS server using C-STORE (Storage SCU).

## Synopsis

```bash
radx dicom store [FILES...] [flags]
radx dicom store --dir DIRECTORY [flags]
```

## Description

The store command implements the DICOM C-STORE service class user (SCU). It sends DICOM files to a
PACS server or DICOM storage SCP, with support for rate limiting, multiple transfer syntaxes, and
automatic SOP Class negotiation.

This command is used for:
- Uploading studies to PACS
- Migrating DICOM data between systems
- Backup and archival operations
- Integration testing and QA workflows

## Flags

### Input (mutually exclusive)

| Flag | Type | Description |
|------|------|-------------|
| `FILES...` | positional | One or more DICOM files to store |
| `--dir` | string | Directory containing DICOM files |

### Connection

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | *required* | DICOM server hostname or IP address |
| `--port` | 11112 | DICOM server port |
| `--called-ae` | ANY-SCP | Called AE Title (server) |
| `--calling-ae` | RADX | Calling AE Title (client) |
| `--timeout` | 5m | Operation timeout |
| `--max-pdu` | 16384 | Maximum PDU size in bytes |

### Options

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--recursive` | `-R` | false | Recursively search directories |
| `--rate-limit` | | 0 | Rate limit in files/second (0 = unlimited) |
| `--rate-limit-bytes` | | 0 | Rate limit in MB/second (0 = unlimited) |
| `--burst` | | 10 | Burst size for rate limiting |

## Usage Examples

### Basic Usage

Store a single file:

```bash
radx dicom store image.dcm --host pacs.example.com
```

Store multiple files:

```bash
radx dicom store file1.dcm file2.dcm file3.dcm --host pacs.example.com
```

Store all files in directory:

```bash
radx dicom store --dir /data/dicom --host pacs.example.com
```

Recursively store files:

```bash
radx dicom store --dir /data/dicom --recursive --host pacs.example.com
```

### Connection Configuration

Custom port and AE titles:

```bash
radx dicom store --dir /data/dicom \
  --host pacs.example.com \
  --port 11113 \
  --called-ae PACS-SERVER \
  --calling-ae RADX-CLIENT
```

Longer timeout for large transfers:

```bash
radx dicom store --dir /data/large-study \
  --host pacs.example.com \
  --timeout 30m
```

### Rate Limiting

Limit to 10 files per second:

```bash
radx dicom store --dir /data/dicom \
  --host pacs.example.com \
  --rate-limit 10
```

Limit bandwidth to 5 MB/second:

```bash
radx dicom store --dir /data/dicom \
  --host pacs.example.com \
  --rate-limit-bytes 5.0
```

Combine file and byte rate limits:

```bash
radx dicom store --dir /data/dicom \
  --host pacs.example.com \
  --rate-limit 20 \
  --rate-limit-bytes 10.0
```

Custom burst size for bursty uploads:

```bash
radx dicom store --dir /data/dicom \
  --host pacs.example.com \
  --rate-limit 5 \
  --burst 20
```

### Large Transfers

Store large study with progress tracking:

```bash
radx dicom store --dir /data/large-study \
  --host pacs.example.com \
  --timeout 60m \
  --rate-limit-bytes 20.0
```

Store with conservative rate limiting for production PACS:

```bash
radx dicom store --dir /data/archive \
  --host production-pacs.example.com \
  --recursive \
  --rate-limit 5 \
  --rate-limit-bytes 2.0
```

## Output

### Successful Transfer

```
✓ All files stored successfully!

  Server:      pacs.example.com:11112
  Total Files: 150
  Successful:  150
  Duration:    45.2s
  Throughput:  3.32 files/sec
```

### Partial Failure

```
⚠ Storage completed with 5 failures

  Server:      pacs.example.com:11112
  Total Files: 150
  Successful:  145
  Failed:      5
  Duration:    52.1s
  Throughput:  2.78 files/sec
```

## Transfer Syntaxes

The store command automatically negotiates these transfer syntaxes:

- Implicit VR Little Endian (1.2.840.10008.1.2)
- Explicit VR Little Endian (1.2.840.10008.1.2.1)
- Explicit VR Big Endian (1.2.840.10008.1.2.2)
- JPEG 2000 Lossless (1.2.840.10008.1.2.4.90)
- JPEG 2000 (1.2.840.10008.1.2.4.91)

The command automatically detects SOP Class UIDs from files and builds appropriate presentation
contexts.

## Common Workflows

### PACS Upload

Upload new study to PACS:

```bash
radx dicom store --dir /data/new-study \
  --host pacs.example.com \
  --called-ae PACS-SERVER
```

### Data Migration

Migrate data from one PACS to another:

```bash
# Extract from source
radx dicom store --dir /archive/study-2024 \
  --host new-pacs.example.com \
  --recursive \
  --rate-limit 10
```

### Backup Verification

Send backup to archive server:

```bash
radx dicom store --dir /backup/daily \
  --host archive.example.com \
  --called-ae ARCHIVE-SCP \
  --rate-limit-bytes 50.0
```

### Integration Testing

Test PACS with sample data:

```bash
radx dicom store --dir /test-data/samples \
  --host test-pacs.example.com \
  --called-ae TEST-PACS
```

### Production Upload

Controlled production upload:

```bash
radx dicom store --dir /data/production-upload \
  --host production-pacs.example.com \
  --called-ae PROD-PACS \
  --rate-limit 5 \
  --rate-limit-bytes 3.0 \
  --timeout 120m
```

## Rate Limiting

### File-Based Rate Limiting

Controls the number of files sent per second:

```bash
--rate-limit 10  # Max 10 files/second
```

Uses token bucket algorithm with configurable burst:

```bash
--rate-limit 5 --burst 20  # Average 5/sec, burst up to 20
```

### Bandwidth Rate Limiting

Controls total data throughput in MB/second:

```bash
--rate-limit-bytes 5.0  # Max 5 MB/second
```

### Combined Limits

Both limits can be used together - the more restrictive limit applies:

```bash
radx dicom store --dir /data \
  --host pacs.example.com \
  --rate-limit 10 \        # Max 10 files/sec
  --rate-limit-bytes 5.0   # Max 5 MB/sec
```

### When to Use Rate Limiting

**File-based limiting** is best for:
- Controlling PACS server load
- Preventing database lock contention
- Managing concurrent association limits

**Bandwidth limiting** is best for:
- Network congestion control
- Shared network environments
- WAN transfers

**Both together** for:
- Production systems
- Shared PACS environments
- Critical infrastructure

## Performance

### Throughput

Typical performance on local network:
- Small files (< 1 MB): 10-50 files/sec
- Medium files (1-10 MB): 5-20 files/sec
- Large files (> 10 MB): 1-10 files/sec

Factors affecting performance:
- Network bandwidth and latency
- PACS server capacity
- File size distribution
- Transfer syntax (compression)

### Optimization Tips

1. **Use rate limiting** to prevent overwhelming the PACS server

2. **Increase PDU size** for large files:
```bash
--max-pdu 32768
```

3. **Adjust timeout** for large studies:
```bash
--timeout 60m
```

4. **Consider batch size** - very large batches may benefit from splitting

## Troubleshooting

### Connection Issues

```bash
Error: failed to connect to server: connection refused
```

**Solutions**:
```bash
# Verify connectivity first
radx dicom echo pacs.example.com

# Check port
radx dicom store file.dcm --host pacs.example.com --port 104
```

### Association Rejected

```bash
Error: failed to connect to server: association rejected
```

**Causes**:
- Incorrect AE titles
- IP not whitelisted
- Maximum connections exceeded

**Solutions**:
```bash
# Verify AE titles
radx dicom store file.dcm \
  --host pacs.example.com \
  --called-ae CORRECT-AE \
  --calling-ae WHITELISTED-CLIENT
```

### C-STORE Failures

```bash
⚠ Storage completed with failures
```

**Common causes**:
- Unsupported SOP Class
- Unsupported Transfer Syntax
- Duplicate SOP Instance UID
- PACS storage full

**Debug approach**:
```bash
# Check DICOM file validity
radx dicom dump file.dcm -t SOPClassUID -t SOPInstanceUID

# Try single file to isolate issue
radx dicom store problematic.dcm --host pacs.example.com

# Check PACS server logs
```

### Timeout Errors

```bash
Error: context deadline exceeded
```

**Solutions**:
```bash
# Increase timeout
radx dicom store --dir /data \
  --host pacs.example.com \
  --timeout 120m

# Add rate limiting to reduce server load
radx dicom store --dir /data \
  --host pacs.example.com \
  --rate-limit 5
```

### Slow Transfer

**Possible causes**:
- Network latency
- PACS server overloaded
- Small PDU size

**Solutions**:
```bash
# Increase PDU size
radx dicom store --dir /data \
  --host pacs.example.com \
  --max-pdu 65536

# Reduce concurrent load
radx dicom store --dir /data \
  --host pacs.example.com \
  --rate-limit 5

# Check network performance
ping pacs.example.com
iperf3 -c pacs.example.com
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success - all files stored |
| 1 | Failure - one or more files failed |

## See Also

- [echo](echo.md) - Test PACS connectivity before storing
- [dump](dump.md) - Inspect DICOM files before sending
- [scp](scp.md) - Run receiving SCP server
- [organize](organize.md) - Organize files before storing

## DICOM References

- [DICOM PS3.4 - Storage Service Class](https://dicom.nema.org/medical/dicom/current/output/html/part04.html#chapter_B)
- [DICOM PS3.7 - DIMSE Services](https://dicom.nema.org/medical/dicom/current/output/html/part07.html)