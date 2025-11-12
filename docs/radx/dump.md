# radx dicom dump

Inspect DICOM file contents and display tag information.

## Synopsis

```bash
radx dicom dump [FILES...] [flags]
radx dicom dump --dir DIRECTORY [flags]
```

## Description

The dump command parses DICOM files and displays their tag information in a human-readable format. It supports multiple
output formats (table, JSON, CSV) and can filter specific tags for focused inspection.

## Flags

### Input (mutually exclusive)

| Flag | Type | Description |
|------|------|-------------|
| `FILES...` | positional | One or more DICOM files to dump |
| `--dir` | string | Directory containing DICOM files |

### Options

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--recursive` | `-R` | false | Recursively search directories |
| `--tag` | `-t` | | Filter specific tags (can be specified multiple times) |
| `--group` | `-g` | | Filter by group tags (can be specified multiple times) |
| `--process-pixel-data` | | false | Process pixel data elements |
| `--store-pixel-data` | | false | Extract and store pixel data to files |

## Filter Formats

### Tag Filters

The `--tag` flag accepts multiple formats:

| Format | Example | Description |
|--------|---------|-------------|
| (GGGG,EEEE) | `(0010,0010)` | Standard DICOM notation |
| GGGGEEEE | `00100010` | Compact hex notation |
| Keyword | `PatientName` | Tag keyword (case-insensitive) |

### Group Filters

The `--group` flag accepts multiple formats:

| Format | Example | Description |
|--------|---------|-------------|
| GGGG | `0010` | Group number in hex |
| Named | `patient` | Predefined group name |

#### Supported Group Names

| Name | Group | Description |
|------|-------|-------------|
| `patient` | 0010 | Patient Information |
| `study`, `series` | 0020 | Study/Series Information |
| `image` | 0028 | Image Information |
| `pixel` | 7FE0 | Pixel Data |
| `metadata`, `meta` | 0002 | File Meta Information |
| `overlay` | 6000 | Overlay Data |

## Usage Examples

### Basic Usage

Dump a single file:

```bash
radx dicom dump file.dcm
```

Dump multiple files:

```bash
radx dicom dump file1.dcm file2.dcm file3.dcm
```

Dump all files in a directory:

```bash
radx dicom dump --dir /data/dicom
```

Recursively dump files:

```bash
radx dicom dump --dir /data/dicom --recursive
```

### Tag Filtering

Filter by tag notation:

```bash
radx dicom dump file.dcm --tag "(0010,0010)" --tag "(0010,0020)"
```

Filter by hex code:

```bash
radx dicom dump file.dcm --tag 00100010 --tag 00100020
```

Filter by keyword (case-insensitive):

```bash
radx dicom dump file.dcm --tag PatientName --tag PatientID --tag StudyDate
```

Mix formats:

```bash
radx dicom dump file.dcm -t PatientName -t "(0020,000D)" -t 00080060
```

### Group Filtering

Filter by group number:

```bash
# Show all Patient Information tags (group 0010)
radx dicom dump file.dcm --group 0010
```

Filter by named group:

```bash
# Show all patient tags using named group
radx dicom dump file.dcm --group patient

# Show all study/series tags
radx dicom dump file.dcm --group study
```

Multiple groups:

```bash
# Show patient and study information
radx dicom dump file.dcm --group 0010,0020

# Using named groups
radx dicom dump file.dcm -g patient -g study
```

### Combined Filtering

Group and tag filtering (AND logic):

```bash
# Show only PatientName from Patient group
radx dicom dump file.dcm --group patient --tag PatientName

# Show specific tags from multiple groups
radx dicom dump file.dcm -g 0010 -g 0020 -t PatientName -t StudyDate
```

### Common Tag Filters

Patient information:

```bash
# Using tag filters
radx dicom dump file.dcm \
  -t PatientName \
  -t PatientID \
  -t PatientBirthDate \
  -t PatientSex

# Or use group filter for all patient tags
radx dicom dump file.dcm --group patient
```

Study information:

```bash
# Using tag filters
radx dicom dump file.dcm \
  -t StudyInstanceUID \
  -t StudyDate \
  -t StudyTime \
  -t StudyDescription

# Or use group filter for all study tags
radx dicom dump file.dcm --group study
```

Series information:

```bash
radx dicom dump file.dcm \
  -t SeriesInstanceUID \
  -t SeriesNumber \
  -t SeriesDescription \
  -t Modality
```

Instance information:

```bash
radx dicom dump file.dcm \
  -t SOPInstanceUID \
  -t InstanceNumber \
  -t SOPClassUID
```

Image dimensions:

```bash
# Using tag filters
radx dicom dump file.dcm \
  -t Rows \
  -t Columns \
  -t BitsAllocated \
  -t PhotometricInterpretation

# Or use group filter for all image tags
radx dicom dump file.dcm --group image
```

### Output Formats

Table format (default):

```bash
radx dicom dump file.dcm
```

Output:
```
┌──────────────┬─────┬────────────────────────┬──────────────┐
│ Tag          │ VR  │ Name                   │ Value        │
├──────────────┼─────┼────────────────────────┼──────────────┤
│ (0008,0005)  │ CS  │ SpecificCharacterSet   │ ISO_IR 100   │
│ (0008,0016)  │ UI  │ SOPClassUID            │ 1.2.840...   │
│ (0008,0018)  │ UI  │ SOPInstanceUID         │ 1.2.840...   │
│ (0010,0010)  │ PN  │ PatientName            │ DOE^JOHN     │
└──────────────┴─────┴────────────────────────┴──────────────┘
```

JSON format:

```bash
radx dicom dump file.dcm --output json
```

Output:
```json
[
  {
    "tag": "(0010,0010)",
    "vr": "PN",
    "name": "PatientName",
    "value": "DOE^JOHN",
    "file": ""
  }
]
```

CSV format:

```bash
radx dicom dump file.dcm --output csv > output.csv
```

### Multiple Files

Dump multiple files with file column:

```bash
radx dicom dump file1.dcm file2.dcm file3.dcm --output csv
```

Output includes filename:
```csv
Tag,VR,Name,Value,File
"(0010,0010)",PN,PatientName,DOE^JOHN,file1.dcm
"(0010,0010)",PN,PatientName,SMITH^JANE,file2.dcm
```

## Common Workflows

### Quality Assurance

Check patient demographics:

```bash
radx dicom dump *.dcm \
  -t PatientName \
  -t PatientID \
  -t PatientBirthDate \
  -t PatientSex \
  --output csv > demographics.csv
```

Verify study information:

```bash
radx dicom dump --dir /data/study \
  -t StudyInstanceUID \
  -t StudyDescription \
  -t StudyDate
```

Check modality and equipment:

```bash
radx dicom dump *.dcm \
  -t Modality \
  -t Manufacturer \
  -t ManufacturerModelName
```

### Anonymization Check

Identify PHI tags:

```bash
radx dicom dump file.dcm \
  -t PatientName \
  -t PatientID \
  -t PatientBirthDate \
  -t InstitutionName \
  -t ReferringPhysicianName
```

### Image Analysis Preparation

Extract image metadata:

```bash
radx dicom dump image.dcm \
  -t Rows \
  -t Columns \
  -t BitsAllocated \
  -t BitsStored \
  -t PixelRepresentation \
  -t PhotometricInterpretation \
  -t WindowCenter \
  -t WindowWidth
```

### UID Verification

Check all UIDs:

```bash
radx dicom dump file.dcm \
  -t StudyInstanceUID \
  -t SeriesInstanceUID \
  -t SOPInstanceUID \
  -t FrameOfReferenceUID
```

## Advanced Usage

### Pixel Data Extraction (Future)

Extract pixel data to separate files:

```bash
radx dicom dump file.dcm --process-pixel-data --store-pixel-data
```

This will create `file.raw` containing the pixel data.

### Combining with Other Tools

Pipe to grep for specific values:

```bash
radx dicom dump file.dcm --output csv | grep "PatientName"
```

Use with jq for JSON processing:

```bash
radx dicom dump file.dcm --output json | jq '.[] | select(.tag == "(0010,0010)")'
```

Export filtered data:

```bash
radx dicom dump --dir /data/dicom \
  -t PatientID \
  -t StudyDate \
  --output csv > study-dates.csv
```

## Output Details

### Table Format

- Tag: DICOM tag in (GGGG,EEEE) notation
- VR: Value Representation (data type)
- Name: Tag name from DICOM dictionary
- Value: Tag value (truncated to 60 characters in table view)

### JSON Format

Complete tag information in machine-readable format:

```json
{
  "tag": "(0010,0010)",
  "vr": "PN",
  "name": "PatientName",
  "value": "DOE^JOHN",
  "file": "example.dcm"
}
```

### CSV Format

Comma-separated values suitable for spreadsheet import:
- Includes header row
- Quoted strings for values containing commas
- File column only present when processing multiple files

## Performance

- **Single file**: Instant (< 1 second)
- **Directory**: ~100-200 files/second
- **Large files**: May be slower for files with large pixel data

Tips:
- Use tag filters (`-t`) to reduce output size
- Use `--recursive` carefully with large directory trees
- Consider using `catalogue` command for large-scale metadata extraction

## Troubleshooting

### Invalid DICOM file

```bash
# Error: file does not appear to be a valid DICOM file
# Check if file has DICM magic number at offset 128
hexdump -C file.dcm | head -20
```

### Tag not found

```bash
# If a tag filter matches nothing, no output is shown
# Verify tag exists in file first
radx dicom dump file.dcm | grep -i patient
```

### Permission denied

```bash
# Check file permissions
ls -la file.dcm

# Run with appropriate permissions
sudo radx dicom dump file.dcm
```

## See Also

- [catalogue](catalogue.md) - Build searchable database of DICOM metadata
- [lookup](lookup.md) - Look up DICOM tag definitions
- [modify](modify.md) - Modify DICOM tag values
