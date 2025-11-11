# radx dicom organize

Reorganize DICOM files into hierarchical directory structure by Study/Series/Instance UID.

## Synopsis

```bash
radx dicom organize DIR --output-dir OUTPUT [flags]
```

## Description

The organize command restructures DICOM files into a standardized hierarchical directory
organization based on DICOM UIDs. It creates a three-level hierarchy:

```
<study-instance-uid>/
  <series-instance-uid>/
    <sop-instance-uid>.dcm
```

This organization:
- Groups files by study and series
- Prevents filename conflicts using unique UIDs
- Enables efficient archival and retrieval
- Matches common PACS storage patterns
- Simplifies study management

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `DIR` | Yes | Source directory containing DICOM files |
| `--output-dir` | Yes | Destination directory for organized files |

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--recursive` | `-R` | true | Recursively search source directory |
| `--move` | | false | Move files instead of copying |
| `--dry-run` | | false | Show what would be done without making changes |

## Usage Examples

### Basic Usage

Organize files with copy:

```bash
radx dicom organize /data/unorganized --output-dir /data/organized
```

Organize with move (faster, but modifies source):

```bash
radx dicom organize /data/unorganized --output-dir /data/organized --move
```

Preview changes without executing:

```bash
radx dicom organize /data/unorganized --output-dir /data/organized --dry-run
```

### Non-Recursive Mode

Organize only files in top-level directory:

```bash
radx dicom organize /data/dicom --output-dir /data/organized --recursive=false
```

### Production Workflow

Safe production organization with dry run first:

```bash
# 1. Preview changes
radx dicom organize /data/raw --output-dir /data/archive --dry-run

# 2. Execute with copy (preserves originals)
radx dicom organize /data/raw --output-dir /data/archive

# 3. Verify organized files
radx dicom catalogue /data/archive --database verify.db

# 4. Remove originals if verification successful
rm -rf /data/raw
```

### Large Archive Organization

Organize large archive with progress tracking:

```bash
radx dicom organize /archive/2024 --output-dir /archive/organized \
  --recursive
```

## Directory Structure

### Before Organization

```
/data/unorganized/
  ├── IMG0001.dcm
  ├── IMG0002.dcm
  ├── scan_001.dcm
  ├── scan_002.dcm
  └── subdir/
      ├── image1.dcm
      └── image2.dcm
```

### After Organization

```
/data/organized/
  ├── 1.2.840.113619.2.55.3.12345678/              # Study Instance UID
  │   ├── 1.2.840.113619.2.55.3.12345678.1/        # Series Instance UID
  │   │   ├── 1.2.840.113619.2.55.3.12345678.1.1.dcm
  │   │   ├── 1.2.840.113619.2.55.3.12345678.1.2.dcm
  │   │   └── 1.2.840.113619.2.55.3.12345678.1.3.dcm
  │   └── 1.2.840.113619.2.55.3.12345678.2/
  │       ├── 1.2.840.113619.2.55.3.12345678.2.1.dcm
  │       └── 1.2.840.113619.2.55.3.12345678.2.2.dcm
  └── 1.2.840.113619.2.55.3.87654321/
      └── 1.2.840.113619.2.55.3.87654321.1/
          └── 1.2.840.113619.2.55.3.87654321.1.1.dcm
```

## Output

### Successful Organization

```
✓ All files organized successfully!

  Source Directory: /data/unorganized
  Output Directory: /data/organized
  Total Files:      245
  Organized:        245
  Mode:            Copy
```

### Dry Run Mode

```
⚠ DRY RUN - No changes were made

  Source Directory: /data/unorganized
  Output Directory: /data/organized
  Total Files:      245
  Organized:        245
  Mode:            Copy
```

### Move Mode

```
✓ All files organized successfully!

  Source Directory: /data/unorganized
  Output Directory: /data/organized
  Total Files:      245
  Organized:        245
  Mode:            Move
```

### Partial Failure

```
⚠ Organization completed with failures

  Source Directory:       /data/unorganized
  Output Directory:       /data/organized
  Total Files:            250
  Organized:              245
  Parse Failures:         3
  Organization Failures:  2
```

## Common Use Cases

### Archive Organization

Organize raw DICOM archive:

```bash
radx dicom organize /archive/raw-data --output-dir /archive/organized \
  --recursive
```

### Pre-PACS Organization

Organize before uploading to PACS:

```bash
# 1. Organize files
radx dicom organize /incoming --output-dir /ready-to-send

# 2. Verify organization
radx dicom catalogue /ready-to-send --database check.db

# 3. Send to PACS
radx dicom store --dir /ready-to-send --host pacs.example.com
```

### Study Isolation

Organize mixed directory to separate studies:

```bash
radx dicom organize /mixed-data --output-dir /separated-studies
```

Each subdirectory in output will contain one study.

### Series Organization

After organizing, each series is in its own directory:

```bash
# Find all series in a study
ls /organized/1.2.840.113619.2.55.3.12345678/

# Process each series independently
for series in /organized/1.2.840.113619.2.55.3.12345678/*/; do
    echo "Processing series: $(basename $series)"
    # ... series-level operations
done
```

### Backup Preparation

Organize before backup for efficient retrieval:

```bash
radx dicom organize /data/clinical --output-dir /backup/organized \
  --recursive
```

Benefits:
- Easy to restore individual studies
- Efficient deduplication
- Clear study boundaries

### Data Quality Check

Organize and identify incomplete series:

```bash
# 1. Organize files
radx dicom organize /data/raw --output-dir /data/organized

# 2. Check series instance counts
find /data/organized -type d -name "1.*" | while read series; do
    count=$(ls "$series"/*.dcm 2>/dev/null | wc -l)
    if [ $count -lt 10 ]; then
        echo "Incomplete series ($count files): $series"
    fi
done
```

## Copy vs Move Modes

### Copy Mode (Default)

**Behavior**:
- Original files remain unchanged
- Files are copied to output directory
- Requires double disk space
- Safe for production use

**When to use**:
- Preserving original data is critical
- Testing organization before committing
- Creating separate organized archive

```bash
radx dicom organize /data/raw --output-dir /data/organized
```

### Move Mode

**Behavior**:
- Original files are moved (deleted from source)
- Faster than copy mode
- No additional disk space required
- Cannot be easily undone

**When to use**:
- Limited disk space
- Source data will be deleted anyway
- Final organization step

```bash
radx dicom organize /data/raw --output-dir /data/organized --move
```

**Important**: Always test with `--dry-run` first when using move mode!

## UID Requirements

The organize command requires these UIDs to be present in each file:

### Required UIDs

- **Study Instance UID** (0020,000D) - Required
- **Series Instance UID** (0020,000E) - Required
- **SOP Instance UID** (0008,0018) - Required

Files missing any required UID will fail to organize and be reported in parse failures.

### Verify UIDs Before Organizing

```bash
# Check if files have required UIDs
radx dicom dump file.dcm \
  -t StudyInstanceUID \
  -t SeriesInstanceUID \
  -t SOPInstanceUID
```

## Performance

### Speed

- **Copy Mode**: ~100-200 files/second
- **Move Mode**: ~500-1000 files/second (same filesystem)
- **Cross-device Move**: Falls back to copy + delete

### Factors Affecting Performance

1. **Filesystem**:
   - Same filesystem: Fast (move is rename)
   - Different filesystem: Slower (copy required)

2. **File Size**:
   - Small files: Higher files/second
   - Large files: Lower files/second

3. **Directory Depth**:
   - Recursive scanning adds overhead

### Optimization Tips

1. **Use move mode** when possible (same filesystem):
```bash
--move
```

2. **Disable recursion** if files are in top-level:
```bash
--recursive=false
```

3. **Use SSD** for both source and destination

## Troubleshooting

### Missing UIDs

```
Error: study instance UID (0020,000D) not found
```

**Causes**:
- Invalid DICOM file
- Non-standard DICOM implementation
- File corruption

**Solutions**:
```bash
# Verify file is valid DICOM
radx dicom dump problematic.dcm

# Check if UIDs exist
radx dicom dump problematic.dcm -t StudyInstanceUID -t SeriesInstanceUID -t SOPInstanceUID
```

### Permission Denied

```
Error: failed to create destination directory: permission denied
```

**Solutions**:
```bash
# Check output directory permissions
ls -la /data/organized

# Create directory first with correct permissions
mkdir -p /data/organized
chmod 755 /data/organized

# Use writable location
radx dicom organize /data/raw --output-dir $HOME/organized
```

### Disk Space

```
Error: failed to copy file: no space left on device
```

**Solutions**:
```bash
# Check available space
df -h /data/organized

# Use move mode instead of copy (if acceptable)
radx dicom organize /data/raw --output-dir /data/organized --move

# Clean up space or use different output directory
```

### Filename Conflicts

Filename conflicts are impossible with UID-based organization since SOP Instance UIDs are unique.

### Cross-Device Move

When source and output are on different filesystems:

```bash
# Move command automatically falls back to copy + delete
radx dicom organize /mnt/source --output-dir /mnt/dest --move
```

This is slower but works correctly.

## Dry Run Workflow

Always test with dry run first:

```bash
# 1. See what would happen
radx dicom organize /data/raw --output-dir /data/organized --dry-run

# 2. Review the plan

# 3. Execute if everything looks good
radx dicom organize /data/raw --output-dir /data/organized

# 4. Verify results
radx dicom catalogue /data/organized --database verify.db
```

## Integration with Other Commands

### With Catalogue

Build searchable database of organized files:

```bash
# 1. Organize
radx dicom organize /data/raw --output-dir /data/organized

# 2. Catalogue
radx dicom catalogue /data/organized --database archive.db

# 3. Query organized structure
radx dicom catalogue --database archive.db -q "Modality=CT"
```

### With Store

Organize then send to PACS:

```bash
# 1. Organize
radx dicom organize /data/raw --output-dir /data/organized

# 2. Send to PACS
radx dicom store --dir /data/organized --host pacs.example.com
```

### With Dump

Inspect organized structure:

```bash
# Organize
radx dicom organize /data/raw --output-dir /data/organized

# Inspect each study
for study in /data/organized/*/; do
    echo "Study: $(basename $study)"
    radx dicom dump "$study" -t PatientName -t StudyDate -t Modality
done
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success - all files organized |
| 1 | Failure - one or more files failed |

## See Also

- [catalogue](catalogue.md) - Index organized files in database
- [store](store.md) - Send organized files to PACS
- [dump](dump.md) - Inspect organized files
- [modify](modify.md) - Modify files before organizing

## DICOM References

- [DICOM PS3.3 - UIDs](https://dicom.nema.org/medical/dicom/current/output/html/part03.html#sect_C.7.2)
- [DICOM PS3.10 - File Format](https://dicom.nema.org/medical/dicom/current/output/html/part10.html)