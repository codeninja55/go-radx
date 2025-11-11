# radx dicom modify

Modify DICOM tags and regenerate UIDs in DICOM files.

## Synopsis

```bash
radx dicom modify [FILES...] --output-dir DIR [flags]
radx dicom modify --dir DIRECTORY --output-dir DIR [flags]
```

## Description

The modify command allows you to update DICOM tags and regenerate UIDs (Unique Identifiers) in
DICOM files. This is useful for anonymization workflows, data correction, and creating test
datasets with new identifiers.

**Note**: Tag modification is currently in development. UID regeneration features are planned.
See the [Implementation Status](#implementation-status) section for details.

## Flags

### Input (mutually exclusive)

| Flag | Type | Description |
|------|------|-------------|
| `FILES...` | positional | One or more DICOM files to modify |
| `--dir` | string | Directory containing DICOM files |

### Output

| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--output-dir` | | Yes* | Output directory for modified files |
| `--in-place` | `-i` | No | Modify files in-place (overwrite original) |

*Not required when using `--in-place`

### Options

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--recursive` | `-R` | false | Recursively search directories |

### Tag Operations

| Flag | Short | Description |
|------|-------|-------------|
| `--insert` | `-I` | Insert or update tag (format: (GGGG,EEEE)=value) |
| `--delete` | `-D` | Delete tag (format: (GGGG,EEEE)) |

### UID Regeneration

| Flag | Description |
|------|-------------|
| `--regenerate-study-uid` | Generate new Study Instance UID |
| `--regenerate-series-uid` | Generate new Series Instance UID |
| `--regenerate-instance-uid` | Generate new SOP Instance UID |
| `--regenerate-all-uids` | Generate all new UIDs |

## Usage Examples

### Tag Insertion/Update

Insert patient name:

```bash
radx dicom modify file.dcm --output-dir /output \
  --insert "(0010,0010)=DOE^JANE"
```

Update multiple tags:

```bash
radx dicom modify file.dcm --output-dir /output \
  --insert "(0010,0010)=DOE^JANE" \
  --insert "(0010,0020)=12345" \
  --insert "(0010,0030)=19900101"
```

Short form with `-I`:

```bash
radx dicom modify file.dcm --output-dir /output \
  -I "(0010,0010)=DOE^JANE" \
  -I "(0010,0020)=12345"
```

### Tag Deletion

Delete patient birth date:

```bash
radx dicom modify file.dcm --output-dir /output \
  --delete "(0010,0030)"
```

Delete multiple tags:

```bash
radx dicom modify file.dcm --output-dir /output \
  --delete "(0010,0030)" \
  --delete "(0010,0032)" \
  --delete "(0010,1040)"
```

Short form with `-D`:

```bash
radx dicom modify file.dcm --output-dir /output \
  -D "(0010,0030)" \
  -D "(0010,1040)"
```

### UID Regeneration

Generate new Study Instance UID:

```bash
radx dicom modify --dir /data/study --output-dir /output \
  --regenerate-study-uid
```

Generate new Series Instance UID:

```bash
radx dicom modify --dir /data/series --output-dir /output \
  --regenerate-series-uid
```

Generate new SOP Instance UID:

```bash
radx dicom modify file.dcm --output-dir /output \
  --regenerate-instance-uid
```

Regenerate all UIDs:

```bash
radx dicom modify --dir /data/study --output-dir /output \
  --regenerate-all-uids
```

### Combined Operations

Modify tags and regenerate UIDs:

```bash
radx dicom modify --dir /data/study --output-dir /output \
  --insert "(0010,0010)=ANONYMOUS" \
  --insert "(0010,0020)=ANON001" \
  --delete "(0010,0030)" \
  --regenerate-all-uids
```

### In-Place Modification

Modify original files (use with caution):

```bash
radx dicom modify file.dcm --in-place \
  --insert "(0010,0010)=DOE^JANE"
```

Short form:

```bash
radx dicom modify file.dcm -i \
  -I "(0010,0010)=DOE^JANE"
```

### Directory Operations

Modify all files in directory:

```bash
radx dicom modify --dir /data/dicom --output-dir /output \
  --insert "(0008,0070)=ACME Corp"
```

Recursive modification:

```bash
radx dicom modify --dir /data/archive --output-dir /output \
  --recursive \
  --insert "(0008,0080)=New Hospital"
```

## Tag Format

Tags must be specified in DICOM notation:

```
(GGGG,EEEE)=VALUE
```

Where:
- `GGGG` = Group number (4 hex digits)
- `EEEE` = Element number (4 hex digits)
- `VALUE` = New value for the tag

### Examples

```bash
# Patient Name (0010,0010)
--insert "(0010,0010)=DOE^JOHN"

# Patient ID (0010,0020)
--insert "(0010,0020)=12345"

# Study Date (0008,0020)
--insert "(0008,0020)=20240115"

# Manufacturer (0008,0070)
--insert "(0008,0070)=ACME Medical Systems"
```

## Common Use Cases

### Anonymization

Remove patient identifiers:

```bash
radx dicom modify --dir /data/study --output-dir /anonymized \
  --insert "(0010,0010)=ANONYMOUS" \
  --insert "(0010,0020)=ANON$(date +%s)" \
  --delete "(0010,0030)" \
  --delete "(0010,1040)" \
  --delete "(0010,4000)" \
  --regenerate-all-uids
```

### Test Data Creation

Create test dataset with new UIDs:

```bash
radx dicom modify --dir /original-study --output-dir /test-study \
  --insert "(0010,0010)=TEST^PATIENT" \
  --insert "(0010,0020)=TEST001" \
  --regenerate-all-uids
```

### Data Correction

Fix manufacturer information:

```bash
radx dicom modify --dir /data/study --output-dir /corrected \
  --insert "(0008,0070)=Correct Manufacturer" \
  --insert "(0008,1090)=Correct Model"
```

### Institution Change

Update institution name:

```bash
radx dicom modify --dir /data/archive --output-dir /updated \
  --recursive \
  --insert "(0008,0080)=New Hospital Name" \
  --insert "(0008,0081)=123 Medical Plaza"
```

### Referring Physician Update

Change referring physician:

```bash
radx dicom modify --dir /data/study --output-dir /updated \
  --insert "(0008,0090)=SMITH^JOHN^DR"
```

## Output

### Successful Modification

```
✓ All files modified successfully!

  Total Files:      150
  Successful:       150
  Output Directory: /output
```

### In-Place Mode Warning

```
✓ All files modified successfully!

  Total Files: 150
  Successful:  150
  Mode:        In-place (original files overwritten)
```

### Partial Failure

```
⚠ Modification completed with 5 failures

  Total Files: 150
  Successful:  145
  Failed:      5
  Output Directory: /output
```

## Implementation Status

### Currently Implemented

- ✅ Command structure and argument parsing
- ✅ File scanning and collection
- ✅ Tag modification syntax validation
- ✅ UID generation logic
- ✅ Output directory management
- ✅ In-place modification mode
- ✅ Progress tracking

### In Development

- ⚠️ Tag insertion/update application to DataSet
- ⚠️ Tag deletion from DataSet
- ⚠️ UID replacement in DataSet

When tag modification is not fully implemented, files are copied without changes and a warning is
logged:

```
WARN: Tag modification not yet fully implemented - file will be copied without changes
WARN: UID regeneration not yet fully implemented - original UIDs will be preserved
```

## Safety Features

### Output Directory Validation

The command validates that:
- Output directory can be created
- Output directory is not the same as input (unless --in-place)
- Sufficient disk space is available

### In-Place Mode Warnings

When using `--in-place`, the command:
- Displays a warning in the output
- Marks the mode clearly in summary
- Recommends backing up original files

### File Preservation

Unless `--in-place` is used:
- Original files are never modified
- Modified files are written to separate directory
- Operation can be safely repeated

## Performance

- **Processing Speed**: ~100-200 files/second
- **Memory Usage**: Minimal (processes one file at a time)
- **Disk I/O**: One read + one write per file

## Troubleshooting

### Invalid Tag Format

```bash
Error: invalid insert format: 0010,0010=value (expected (GGGG,EEEE)=value)
```

**Solution**: Use correct format with parentheses:
```bash
--insert "(0010,0010)=value"
```

### Output Directory Exists

The command will use existing directory and may overwrite files.

**Recommendation**: Use unique output directories:
```bash
--output-dir /output/modified-$(date +%Y%m%d-%H%M%S)
```

### Permission Denied

```bash
Error: failed to create output directory: permission denied
```

**Solutions**:
```bash
# Check permissions
ls -la /output

# Create directory first
mkdir -p /output
chmod 755 /output

# Use writable location
--output-dir $HOME/dicom-output
```

### No Modifications Specified

```bash
Error: no modifications specified
```

**Solution**: Specify at least one modification:
```bash
--insert "(0010,0010)=value"
# or
--delete "(0010,0030)"
# or
--regenerate-study-uid
```

## Tag Reference

Common tags for modification:

### Patient Module
- (0010,0010) - Patient's Name
- (0010,0020) - Patient ID
- (0010,0030) - Patient's Birth Date
- (0010,0040) - Patient's Sex
- (0010,1040) - Patient's Address

### Study Module
- (0008,0020) - Study Date
- (0008,0030) - Study Time
- (0008,0050) - Accession Number
- (0008,0090) - Referring Physician's Name
- (0008,1030) - Study Description

### Equipment Module
- (0008,0070) - Manufacturer
- (0008,0080) - Institution Name
- (0008,1090) - Manufacturer's Model Name

### UIDs
- (0020,000D) - Study Instance UID
- (0020,000E) - Series Instance UID
- (0008,0018) - SOP Instance UID

## See Also

- [dump](dump.md) - Inspect tags before modifying
- [lookup](lookup.md) - Look up tag definitions
- [organize](organize.md) - Organize modified files

## DICOM References

- [DICOM PS3.3 - Information Object Definitions](https://dicom.nema.org/medical/dicom/current/output/html/part03.html)
- [DICOM PS3.6 - Data Dictionary](https://dicom.nema.org/medical/dicom/current/output/html/part06.html)
- [DICOM Anonymization Guidelines](https://dicom.nema.org/medical/dicom/current/output/html/part15.html#chapter_E)
