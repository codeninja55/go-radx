# radx dicom lookup

Look up DICOM tag information from the built-in dictionary.

## Synopsis

```bash
radx dicom lookup QUERY [QUERY...] [flags]
```

## Description

The lookup command searches the built-in DICOM tag dictionary for tags matching your query. It supports searching by
tag ID, keyword, or text, making it easy to find tag information without consulting external references.

## Arguments

| Argument | Description |
|----------|-------------|
| `QUERY` | One or more search terms (tag ID, keyword, or text) |

## Query Formats

| Format | Example | Description |
|--------|---------|-------------|
| (GGGG,EEEE) | `(0010,0010)` | Standard DICOM notation |
| GGGGEEEE | `00100010` | Compact hex notation |
| Keyword | `PatientName` | Exact keyword match |
| Text | `patient` | Text search in keywords and names |

## Usage Examples

### Lookup by Tag ID

```bash
radx dicom lookup "(0010,0010)"
```

Output:
```
Found 1 matching tag(s)

┌──────────────┬─────┬─────────────┬─────────────────────┐
│ Tag          │ VR  │ Keyword     │ Name                │
├──────────────┼─────┼─────────────┼─────────────────────┤
│ (0010,0010)  │ PN  │ PatientName │ Patient's Name      │
└──────────────┴─────┴─────────────┴─────────────────────┘
```

### Lookup by Keyword

```bash
radx dicom lookup PatientID
```

```bash
radx dicom lookup StudyInstanceUID
```

### Text Search

Search for all patient-related tags:

```bash
radx dicom lookup patient
```

Output:
```
Found 4 matching tag(s)

┌──────────────┬─────┬───────────────────┬──────────────────────────┐
│ Tag          │ VR  │ Keyword           │ Name                     │
├──────────────┼─────┼───────────────────┼──────────────────────────┤
│ (0010,0010)  │ PN  │ PatientName       │ Patient's Name           │
│ (0010,0020)  │ LO  │ PatientID         │ Patient ID               │
│ (0010,0030)  │ DA  │ PatientBirthDate  │ Patient's Birth Date     │
│ (0010,0040)  │ CS  │ PatientSex        │ Patient's Sex            │
└──────────────┴─────┴───────────────────┴──────────────────────────┘
```

Search for UID tags:

```bash
radx dicom lookup UID
```

### Multiple Queries

Look up multiple tags at once:

```bash
radx dicom lookup PatientName StudyDate Modality
```

```bash
radx dicom lookup "(0010,0010)" "(0008,0020)" "(0008,0060)"
```

Mix query formats:

```bash
radx dicom lookup PatientID "(0020,000D)" study
```

## Output Format

### Table Format (Default)

```
┌──────────────┬─────┬─────────────────┬──────────────────────┐
│ Tag          │ VR  │ Keyword         │ Name                 │
├──────────────┼─────┼─────────────────┼──────────────────────┤
│ (0010,0010)  │ PN  │ PatientName     │ Patient's Name       │
└──────────────┴─────┴─────────────────┴──────────────────────┘
```

### JSON Format

```bash
radx dicom lookup PatientName --output json
```

Output:
```json
[
  {
    "tag": "(0010,0010)",
    "vr": "PN",
    "name": "Patient's Name (PatientName)"
  }
]
```

## Common Use Cases

### Before Using dump Command

Find the correct tag keyword:

```bash
# Want to filter patient information in dump?
radx dicom lookup patient

# Use the keywords in dump command
radx dicom dump file.dcm -t PatientName -t PatientID
```

### Verify Tag Format

Check the correct tag notation:

```bash
radx dicom lookup "0010,0010"
# Shows: (0010,0010) - PatientName
```

### Discover Related Tags

Find all study-related tags:

```bash
radx dicom lookup study
```

Find all UID tags:

```bash
radx dicom lookup UID
```

### Quick Reference

Look up VR (Value Representation) for a tag:

```bash
radx dicom lookup PatientName
# VR: PN (Person Name)

radx dicom lookup StudyDate
# VR: DA (Date)
```

## Built-in Dictionary

The lookup command includes 50+ common DICOM tags:

### Patient Information
- PatientName (0010,0010)
- PatientID (0010,0020)
- PatientBirthDate (0010,0030)
- PatientSex (0010,0040)

### Study Information
- StudyInstanceUID (0020,000D)
- StudyDate (0008,0020)
- StudyTime (0008,0030)
- StudyDescription (0008,1030)
- AccessionNumber (0008,0050)

### Series Information
- SeriesInstanceUID (0020,000E)
- SeriesNumber (0020,0011)
- SeriesDescription (0008,103E)
- Modality (0008,0060)

### Instance Information
- SOPInstanceUID (0008,0018)
- SOPClassUID (0008,0016)
- InstanceNumber (0020,0013)

### Image Information
- Rows (0028,0010)
- Columns (0028,0011)
- BitsAllocated (0028,0100)
- BitsStored (0028,0101)
- PixelData (7FE0,0010)

### And more...

## Advanced Examples

### Integration with dump

```bash
# 1. Find tag keyword
radx dicom lookup manufacturer

# 2. Use keyword in dump
radx dicom dump file.dcm -t Manufacturer
```

### Integration with modify

```bash
# 1. Find tag to modify
radx dicom lookup patient name

# 2. Use tag in modify command
radx dicom modify file.dcm --output-dir /out --insert "(0010,0010)=DOE^JANE"
```

### Export Tag Dictionary

```bash
# Export all tags to JSON
radx dicom lookup patient study series instance --output json > tags.json
```

## Troubleshooting

### No matches found

```bash
radx dicom lookup xyz
# No matching tags found

# Try broader search
radx dicom lookup patient
```

### Case Sensitivity

Tag keywords are case-insensitive:

```bash
radx dicom lookup patientname  # Works
radx dicom lookup PatientName  # Works
radx dicom lookup PATIENTNAME  # Works
```

## See Also

- [dump](dump.md) - Inspect DICOM files using tag keywords
- [catalogue](catalogue.md) - Database uses same tag keywords
- [modify](modify.md) - Modify tags using tag IDs or keywords

## Online Resources

For complete DICOM tag reference, see:
- [DICOM Standard Part 6](https://dicom.nema.org/medical/dicom/current/output/html/part06.html)
- [Innolitics DICOM Browser](https://dicom.innolitics.com/ciods)