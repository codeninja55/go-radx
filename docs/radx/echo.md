# radx dicom echo

Verify DICOM connectivity using C-ECHO (Verification SOP Class).

## Synopsis

```bash
radx dicom echo HOST [PORT] [flags]
```

## Description

The echo command implements the DICOM C-ECHO verification service. It sends a C-ECHO request to a
DICOM server to verify network connectivity, association negotiation, and basic DICOM
communication capabilities.

This is the DICOM equivalent of a network ping and is typically the first test performed when
troubleshooting PACS connectivity issues.

## Arguments

| Argument | Required | Default | Description |
|----------|----------|---------|-------------|
| `HOST` | Yes | | DICOM server hostname or IP address |
| `PORT` | No | 11112 | DICOM server port |

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--called-ae` | | ANY-SCP | Called AE Title (server) |
| `--calling-ae` | | RADX | Calling AE Title (client) |
| `--timeout` | | 30s | Connection timeout |
| `--max-pdu` | | 16384 | Maximum PDU size in bytes |

## Usage Examples

### Basic Usage

Test connectivity to PACS server:

```bash
radx dicom echo pacs.example.com
```

Test connectivity on custom port:

```bash
radx dicom echo pacs.example.com 11113
```

### Custom AE Titles

Some PACS servers require specific AE titles:

```bash
radx dicom echo pacs.example.com --called-ae PACS-SERVER --calling-ae MY-CLIENT
```

Short form:

```bash
radx dicom echo pacs.example.com --called-ae PACS-SERVER --calling-ae RADX-SCU
```

### Timeout Configuration

Test with shorter timeout:

```bash
radx dicom echo pacs.example.com --timeout 10s
```

Test with longer timeout for slow networks:

```bash
radx dicom echo pacs.example.com --timeout 60s
```

### Maximum PDU Size

Some older PACS systems require smaller PDU sizes:

```bash
radx dicom echo pacs.example.com --max-pdu 8192
```

Use larger PDU size for better performance:

```bash
radx dicom echo pacs.example.com --max-pdu 32768
```

## Output

### Successful Connection

```
✓ C-ECHO successful!

  Server:        pacs.example.com:11112
  Called AE:     ANY-SCP
  Calling AE:    RADX
  Response Time: 45ms
```

### Connection Failure

```
Error: failed to connect to server: connection refused
```

### Association Rejected

```
Error: C-ECHO failed: association rejected by server
```

### Timeout

```
Error: failed to connect to server: context deadline exceeded
```

## Common Use Cases

### Initial Connectivity Test

Before sending files, verify PACS is reachable:

```bash
radx dicom echo pacs.example.com
```

### Troubleshooting Workflow

1. Test basic network connectivity:

```bash
ping pacs.example.com
telnet pacs.example.com 11112
```

2. Test DICOM connectivity:

```bash
radx dicom echo pacs.example.com
```

3. Test with correct AE titles:

```bash
radx dicom echo pacs.example.com --called-ae CORRECT-AE
```

### Scripting Health Checks

Monitor PACS availability:

```bash
#!/bin/bash
if radx dicom echo pacs.example.com --timeout 10s; then
    echo "PACS is online"
else
    echo "PACS is offline"
    # Send alert
fi
```

Check multiple PACS servers:

```bash
#!/bin/bash
servers=(
    "pacs1.example.com"
    "pacs2.example.com"
    "pacs3.example.com"
)

for server in "${servers[@]}"; do
    echo "Testing $server..."
    radx dicom echo "$server" --timeout 5s
done
```

### Latency Measurement

Measure response time to PACS:

```bash
for i in {1..10}; do
    radx dicom echo pacs.example.com
    sleep 1
done
```

### Network Performance Testing

Test with different PDU sizes:

```bash
for pdu in 4096 8192 16384 32768; do
    echo "Testing PDU size: $pdu"
    radx dicom echo pacs.example.com --max-pdu $pdu
done
```

## Verification SOP Class

The echo command uses the DICOM Verification SOP Class UID:

- **SOP Class UID**: 1.2.840.10008.1.1
- **Service**: C-ECHO
- **Purpose**: Verify application-level communication

Supported transfer syntaxes:
- Implicit VR Little Endian (1.2.840.10008.1.2)
- Explicit VR Little Endian (1.2.840.10008.1.2.1)
- Explicit VR Big Endian (1.2.840.10008.1.2.2)

## Troubleshooting

### Connection Refused

```bash
Error: failed to connect to server: connection refused
```

**Causes**:
- PACS server is not running
- Wrong port number
- Firewall blocking connection

**Solutions**:
```bash
# Check if port is open
telnet pacs.example.com 11112

# Try default DICOM port
radx dicom echo pacs.example.com 11112

# Try alternative port
radx dicom echo pacs.example.com 104
```

### Timeout

```bash
Error: failed to connect to server: context deadline exceeded
```

**Causes**:
- Network latency
- Server under heavy load
- Firewall with delayed response

**Solutions**:
```bash
# Increase timeout
radx dicom echo pacs.example.com --timeout 60s

# Check network connectivity
ping pacs.example.com
traceroute pacs.example.com
```

### Association Rejected

```bash
Error: C-ECHO failed: association rejected
```

**Causes**:
- AE Title not recognized by server
- Client IP not whitelisted
- Maximum connections exceeded

**Solutions**:
```bash
# Try correct Called AE Title
radx dicom echo pacs.example.com --called-ae CORRECT-AE

# Try different Calling AE Title
radx dicom echo pacs.example.com --calling-ae WHITELISTED-AE

# Contact PACS administrator to:
# - Verify AE Title configuration
# - Whitelist client IP address
# - Check connection limits
```

### C-ECHO Failed

```bash
Error: C-ECHO failed: invalid response from server
```

**Causes**:
- Server doesn't support C-ECHO
- Protocol version mismatch
- Network packet corruption

**Solutions**:
```bash
# Try smaller PDU size
radx dicom echo pacs.example.com --max-pdu 8192

# Check PACS server logs
# Contact PACS administrator
```

### Name Resolution Failed

```bash
Error: failed to connect to server: no such host
```

**Solutions**:
```bash
# Check hostname
nslookup pacs.example.com

# Use IP address instead
radx dicom echo 192.168.1.100

# Check /etc/hosts or DNS configuration
```

## Performance

- **Connection Time**: Typically 10-100ms on local network
- **C-ECHO Time**: Typically < 50ms
- **Total Time**: Usually < 200ms for successful verification
- **Timeout Default**: 30 seconds (configurable)

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success - C-ECHO successful |
| 1 | Failure - Connection or C-ECHO failed |

## See Also

- [store](store.md) - Send DICOM files to PACS (C-STORE)
- [scp](scp.md) - Run DICOM SCP server
- [dump](dump.md) - Inspect DICOM files

## DICOM References

- [DICOM PS3.4 - Service Class Specifications](https://dicom.nema.org/medical/dicom/current/output/html/part04.html)
- [DICOM PS3.7 - Message Exchange](https://dicom.nema.org/medical/dicom/current/output/html/part07.html)
- Verification SOP Class specification in DICOM Standard Part 4, Annex A