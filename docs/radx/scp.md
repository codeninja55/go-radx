# radx dicom scp

Run a DICOM SCP (Service Class Provider) server to receive DICOM files.

## Synopsis

```bash
radx dicom scp [PORT] [flags]
```

## Description

The scp command runs a DICOM SCP server that accepts incoming DICOM associations and handles
C-ECHO (verification) and C-STORE (storage) requests. It can receive DICOM files from SCU clients,
modalities, or PACS systems.

Features:
- C-ECHO support for connectivity verification
- C-STORE support for receiving DICOM files
- Automatic file organization by Study/Series/Instance UID
- Concurrent connection handling
- SOP Class filtering
- Progress statistics

Use cases:
- Receiving studies from modalities
- PACS integration testing
- DICOM router implementation
- Backup and archival workflows
- Quality assurance environments

## Arguments

| Argument | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | 11112 | Port to listen on |

## Flags

### Server Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `--aet` | RADX-SCP | Application Entity Title for this SCP |
| `--output-dir` | ./dicom-received | Directory to store received files |
| `--max-pdu` | 16384 | Maximum PDU size in bytes |
| `--max-conns` | 10 | Maximum concurrent connections |

### Functionality

| Flag | Default | Description |
|------|---------|-------------|
| `--organize` | true | Auto-organize files by Study/Series/Instance UID |
| `--accept-echo` | true | Accept C-ECHO verification requests |
| `--sop-class` | | Accepted SOP Class UIDs (default: all) |

## Usage Examples

### Basic Usage

Start server on default port:

```bash
radx dicom scp
```

Start server on custom port:

```bash
radx dicom scp 11113
```

Custom output directory:

```bash
radx dicom scp --output-dir /data/received
```

### Server Configuration

Custom AE Title:

```bash
radx dicom scp --aet MY-SCP
```

Custom port and AE Title:

```bash
radx dicom scp 11113 --aet RESEARCH-SCP
```

Maximum connections:

```bash
radx dicom scp --max-conns 20
```

Larger PDU size for better performance:

```bash
radx dicom scp --max-pdu 32768
```

### Organization Options

Disable auto-organization (flat structure):

```bash
radx dicom scp --organize=false --output-dir /data/flat
```

Custom organized output:

```bash
radx dicom scp --organize --output-dir /archive/2024
```

### Functionality Options

Disable C-ECHO (storage only):

```bash
radx dicom scp --accept-echo=false
```

Filter specific SOP Classes:

```bash
radx dicom scp \
  --sop-class 1.2.840.10008.5.1.4.1.1.2 \
  --sop-class 1.2.840.10008.5.1.4.1.1.4
```

### Production Configuration

Production server with enhanced limits:

```bash
radx dicom scp 11112 \
  --aet PRODUCTION-SCP \
  --output-dir /data/pacs-incoming \
  --max-pdu 65536 \
  --max-conns 50 \
  --organize
```

### Testing Configuration

Test server with logging:

```bash
radx dicom scp 11113 \
  --aet TEST-SCP \
  --output-dir /tmp/test-received \
  --max-conns 5
```

## File Organization

### With Organization (Default)

Files are organized in Study/Series/Instance hierarchy:

```
./dicom-received/
  ├── 1.2.840.113619.2.55.3.12345678/              # Study Instance UID
  │   ├── 1.2.840.113619.2.55.3.12345678.1/        # Series Instance UID
  │   │   ├── 1.2.840.113619.2.55.3.12345678.1.1.dcm
  │   │   ├── 1.2.840.113619.2.55.3.12345678.1.2.dcm
  │   │   └── 1.2.840.113619.2.55.3.12345678.1.3.dcm
  │   └── 1.2.840.113619.2.55.3.12345678.2/
  │       └── 1.2.840.113619.2.55.3.12345678.2.1.dcm
  └── 1.2.840.113619.2.55.3.87654321/
      └── 1.2.840.113619.2.55.3.87654321.1/
          └── 1.2.840.113619.2.55.3.87654321.1.1.dcm
```

### Without Organization

Flat structure using SOP Instance UIDs:

```
./dicom-received/
  ├── 1.2.840.113619.2.55.3.12345678.1.1.dcm
  ├── 1.2.840.113619.2.55.3.12345678.1.2.dcm
  ├── 1.2.840.113619.2.55.3.12345678.1.3.dcm
  └── 1.2.840.113619.2.55.3.87654321.1.1.dcm
```

## Output

### Server Started

```
✓ DICOM SCP server started

  Listen Address:    :11112
  AE Title:          RADX-SCP
  Output Directory:  ./dicom-received
  Organization:      Enabled (Study/Series/Instance)
  C-ECHO:           Enabled
  Max Connections:   10

Press Ctrl+C to stop the server...
```

### During Operation

The server logs incoming requests:

```
INFO: C-ECHO request received calling_ae=WORKSTATION count=1
INFO: C-STORE request received calling_ae=MODALITY sop_class=1.2.840.10008.5.1.4.1.1.2 count=1
INFO: Stored DICOM file path=/data/received/1.2.840.../1.2.840.../1.2.840....dcm
```

### Statistics (Every 30 seconds)

```
INFO: Server statistics echo_count=5 store_count=150 store_failures=0
```

### Shutdown

```
Server shutting down...

  C-ECHO Requests:    12
  C-STORE Requests:   245
  Successful Stores:  245
```

## Supported SOP Classes

### Default (All SOP Classes)

By default, the server accepts all SOP Classes with these transfer syntaxes:
- Implicit VR Little Endian (1.2.840.10008.1.2)
- Explicit VR Little Endian (1.2.840.10008.1.2.1)
- Explicit VR Big Endian (1.2.840.10008.1.2.2)
- JPEG 2000 Lossless (1.2.840.10008.1.2.4.90)
- JPEG 2000 (1.2.840.10008.1.2.4.91)

### Common SOP Classes (When Not Filtered)

- Secondary Capture (1.2.840.10008.5.1.4.1.1.7)
- CR Image Storage (1.2.840.10008.5.1.4.1.1.1)
- CT Image Storage (1.2.840.10008.5.1.4.1.1.2)
- MR Image Storage (1.2.840.10008.5.1.4.1.1.4)
- US Image Storage (1.2.840.10008.5.1.4.1.1.6.1)
- RT Image Storage (1.2.840.10008.5.1.4.1.1.481.1)

### Verification SOP Class

Always supported when `--accept-echo` is enabled:
- Verification (1.2.840.10008.1.1)

## Common Use Cases

### Receive from Modality

Configure modality to send to this SCP:

```bash
# Start SCP
radx dicom scp 11112 --aet ARCHIVE-SCP --output-dir /archive/incoming

# Configure modality:
# - Destination Host: <your-server-ip>
# - Destination Port: 11112
# - Destination AE: ARCHIVE-SCP
```

### PACS Testing

Test PACS C-STORE functionality:

```bash
# Start test SCP
radx dicom scp 11113 --aet TEST-SCP --output-dir /tmp/test

# Configure PACS to send test study to:
# - Host: <your-ip>
# - Port: 11113
# - AE: TEST-SCP
```

### Backup Server

Receive automatic backups from PACS:

```bash
radx dicom scp 11112 \
  --aet BACKUP-SCP \
  --output-dir /backup/dicom/$(date +%Y%m%d) \
  --organize \
  --max-conns 20
```

### Quality Assurance Workflow

Receive and automatically organize QA studies:

```bash
radx dicom scp 11114 \
  --aet QA-SCP \
  --output-dir /qa/received \
  --organize
```

Then process received files:

```bash
# Monitor for new studies
watch -n 10 "radx dicom catalogue /qa/received --database qa.db"

# Process each study
for study in /qa/received/*/; do
    # Run QA checks
    echo "QA check: $study"
done
```

### DICOM Router

Receive and forward to multiple destinations:

```bash
# Start receiver
radx dicom scp 11112 --aet ROUTER-SCP --output-dir /router/queue

# Forward to destinations
while true; do
    for file in /router/queue/*/*/*/*.dcm; do
        radx dicom store "$file" --host pacs1.example.com
        radx dicom store "$file" --host pacs2.example.com
        rm "$file"
    done
    sleep 10
done
```

### Research Archive

Receive research studies:

```bash
radx dicom scp 11115 \
  --aet RESEARCH-SCP \
  --output-dir /research/archive \
  --organize \
  --max-conns 30
```

## Testing the Server

### Test with C-ECHO

From another terminal or machine:

```bash
radx dicom echo localhost 11112 --called-ae RADX-SCP
```

### Test with C-STORE

Send test files:

```bash
radx dicom store test.dcm --host localhost --port 11112 --called-ae RADX-SCP
```

Send directory:

```bash
radx dicom store --dir /test-data --host localhost --port 11112 --called-ae RADX-SCP
```

### Monitor Received Files

```bash
# Watch output directory
watch -n 1 "find ./dicom-received -name '*.dcm' | wc -l"

# Tail server logs (if redirected)
tail -f scp.log

# Check database of received files
radx dicom catalogue ./dicom-received --database received.db
```

## Performance

### Throughput

Typical performance:
- **C-ECHO**: 100-1000 requests/second
- **C-STORE**: 10-100 files/second (depends on file size and organization)

### Concurrency

- **Max Connections**: Controls concurrent DICOM associations
- **Default**: 10 concurrent connections
- **Recommended**: 20-50 for production servers

### Optimization

Increase throughput:

```bash
radx dicom scp \
  --max-pdu 65536 \      # Larger PDU
  --max-conns 50 \       # More concurrent connections
  --organize=false       # Skip organization overhead (if acceptable)
```

## Troubleshooting

### Port Already in Use

```bash
Error: failed to start server: address already in use
```

**Solutions**:
```bash
# Check what's using the port
lsof -i :11112
netstat -an | grep 11112

# Use different port
radx dicom scp 11113

# Kill existing process
kill <pid>
```

### Permission Denied (Port < 1024)

```bash
Error: failed to start server: permission denied
```

**Solutions**:
```bash
# Use port >= 1024
radx dicom scp 11112

# Or run with sudo (not recommended)
sudo radx dicom scp 104
```

### Output Directory Permission Denied

```bash
Error: failed to create output directory: permission denied
```

**Solutions**:
```bash
# Use writable directory
radx dicom scp --output-dir $HOME/dicom-received

# Create directory first
mkdir -p /data/received
chmod 755 /data/received
radx dicom scp --output-dir /data/received
```

### Connection Refused from Remote

**Causes**:
- Firewall blocking port
- Server not listening on external interface

**Solutions**:
```bash
# Check firewall
sudo iptables -L | grep 11112

# Allow port in firewall
sudo ufw allow 11112
sudo firewall-cmd --add-port=11112/tcp

# Verify server is listening
netstat -an | grep 11112
```

### Association Rejected

**Causes**:
- Incorrect AE Title
- Unsupported SOP Class
- Max connections exceeded

**Check server logs** for specific error.

### Storage Failures

If files aren't being stored:

```bash
# Check disk space
df -h ./dicom-received

# Check file permissions
ls -la ./dicom-received

# Enable debug logging to see errors
# (future feature)
```

## Running as a Service

### systemd Service (Linux)

Create `/etc/systemd/system/radx-scp.service`:

```ini
[Unit]
Description=RadX DICOM SCP Server
After=network.target

[Service]
Type=simple
User=dicom
WorkingDirectory=/opt/radx
ExecStart=/usr/local/bin/radx dicom scp 11112 \
  --aet RADX-SCP \
  --output-dir /data/dicom-received \
  --max-conns 20
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable radx-scp
sudo systemctl start radx-scp
sudo systemctl status radx-scp
```

### Docker Container

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o radx ./cmd/radx

FROM alpine:latest
COPY --from=builder /app/radx /usr/local/bin/
EXPOSE 11112
VOLUME /data
CMD ["radx", "dicom", "scp", "11112", "--output-dir", "/data"]
```

Run:

```bash
docker build -t radx-scp .
docker run -d -p 11112:11112 -v /data/dicom:/data radx-scp
```

## Security Considerations

### Network Security

- **Firewall**: Only expose port to trusted networks
- **Authentication**: Consider using VPN or SSH tunnel for untrusted networks
- **Encryption**: DICOM protocol is not encrypted by default

### Access Control

- **AE Title Filtering**: Some PACS support AE-based access control
- **IP Whitelisting**: Use firewall rules to restrict source IPs

### Data Security

- **PHI Protection**: Received files may contain protected health information
- **Storage Security**: Use encrypted filesystem for output directory
- **Access Logging**: Monitor who sends data to your SCP

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success - clean shutdown |
| 1 | Failure - server error |

## See Also

- [echo](echo.md) - Test connectivity to SCP
- [store](store.md) - Send files to SCP
- [organize](organize.md) - Organize received files
- [catalogue](catalogue.md) - Index received files

## DICOM References

- [DICOM PS3.4 - Service Class Specifications](https://dicom.nema.org/medical/dicom/current/output/html/part04.html)
- [DICOM PS3.7 - Message Exchange](https://dicom.nema.org/medical/dicom/current/output/html/part07.html)
- [DICOM Conformance Statement Template](https://dicom.nema.org/medical/dicom/current/output/html/part02.html#chapter_K)