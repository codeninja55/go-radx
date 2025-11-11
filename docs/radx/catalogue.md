# radx dicom catalogue

Build and query a searchable SQLite database of DICOM file metadata.

## Synopsis

```bash
radx dicom catalogue [DIR] [flags]
```

## Description

The catalogue command creates a searchable SQLite database containing metadata from DICOM files. It indexes 22 key
DICOM tags for fast querying and provides three query methods: keyword-based filters, field-specific searches, and raw
SQL queries.

This is particularly useful for:
- Large DICOM archives requiring fast search
- Quality assurance and validation workflows
- Finding specific studies, series, or patients
- Generating reports and statistics
- Data exploration and analysis

## Flags

### Required
None (can run with just --database flag to view existing database)

### Optional

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--database` | `-d` | `dicom-catalogue.db` | SQLite database path |
| `--rebuild` | | false | Rebuild database from scratch |
| `--recursive` | `-R` | true | Recursively search directories |
| `--query` | `-q` | | Query filters (can be specified multiple times) |
| `--sql` | | | Execute raw SQL query (SELECT only) |

## Usage Examples

### Basic Indexing

Index all DICOM files in a directory:

```bash
radx dicom catalogue /data/dicom
```

Index with custom database name:

```bash
radx dicom catalogue /data/dicom --database my-archive.db
```

Rebuild existing database:

```bash
radx dicom catalogue /data/dicom --database my-archive.db --rebuild
```

### Viewing Database Summary

View statistics without indexing:

```bash
radx dicom catalogue --database my-archive.db
```

Output:
```
✓ Database catalogue complete

  Database:        my-archive.db
  Total Files:     1,245
  Unique Patients: 127
  Unique Studies:  342
  Unique Series:   891

  Modality Breakdown:
    CT: 567 files
    MR: 432 files
    CR: 246 files
```

### Keyword-Based Queries

Query by patient ID:

```bash
radx dicom catalogue --database my-archive.db -q "PatientID=12345"
```

Query by modality:

```bash
radx dicom catalogue --database my-archive.db -q "Modality=CT"
```

Query by study description (partial match):

```bash
radx dicom catalogue --database my-archive.db -q "StudyDescription=chest"
```

### Multiple Filters

Combine multiple query filters (AND logic):

```bash
radx dicom catalogue --database my-archive.db \
  -q "Modality=CT" \
  -q "Manufacturer=GE"
```

### Text Search

Search across multiple fields:

```bash
radx dicom catalogue --database my-archive.db -q "smith"
```

This searches in: patient_name, patient_id, study_description, series_description, modality, manufacturer,
institution_name

### Raw SQL Queries

Execute custom SQL queries (SELECT only for safety):

```bash
radx dicom catalogue --database my-archive.db \
  --sql "SELECT patient_name, COUNT(*) as num_files FROM dicom_metadata GROUP BY patient_name"
```

Find studies with most images:

```bash
radx dicom catalogue --database my-archive.db \
  --sql "SELECT study_instance_uid, study_description, COUNT(*) as images
         FROM dicom_metadata
         GROUP BY study_instance_uid
         ORDER BY images DESC
         LIMIT 10"
```

Find files by date range:

```bash
radx dicom catalogue --database my-archive.db \
  --sql "SELECT * FROM dicom_metadata
         WHERE study_date BETWEEN '20240101' AND '20241231'"
```

Get modality statistics:

```bash
radx dicom catalogue --database my-archive.db \
  --sql "SELECT modality, AVG(file_size) as avg_size, COUNT(*) as count
         FROM dicom_metadata
         GROUP BY modality"
```

## Indexed Fields

The catalogue indexes these DICOM tags:

### Patient Information
- `patient_name` (0010,0010)
- `patient_id` (0010,0020)
- `patient_birth_date` (0010,0030)
- `patient_sex` (0010,0040)

### Study Information
- `study_instance_uid` (0020,000D)
- `study_date` (0008,0020)
- `study_description` (0008,1030)

### Series Information
- `series_instance_uid` (0020,000E)
- `series_number` (0020,0011)
- `series_description` (0008,103E)

### Instance Information
- `sop_instance_uid` (0008,0018)
- `sop_class_uid` (0008,0016)
- `instance_number` (0020,0013)

### Equipment & Clinical
- `modality` (0008,0060)
- `manufacturer` (0008,0070)
- `institution_name` (0008,0080)
- `accession_number` (0008,0050)
- `referring_physician_name` (0008,0090)
- `performing_physician_name` (0008,1050)

### File Metadata
- `file_path` - Full path to DICOM file
- `file_name` - Filename only
- `file_size` - File size in bytes
- `indexed_at` - Timestamp when indexed

## Query Field Mapping

These field names can be used in `--query` filters:

| Field Name | Database Column |
|------------|-----------------|
| PatientName, patient_name | patient_name |
| PatientID, patient_id | patient_id |
| StudyInstanceUID, studyuid, study_instance_uid | study_instance_uid |
| SeriesInstanceUID, seriesuid, series_instance_uid | series_instance_uid |
| SOPInstanceUID, instanceuid, sop_instance_uid | sop_instance_uid |
| Modality, modality | modality |
| Manufacturer, manufacturer | manufacturer |
| Institution, institution_name | institution_name |
| AccessionNumber, accession_number | accession_number |
| StudyDescription, study_description | study_description |
| SeriesDescription, series_description | series_description |

## SQL Safety

Raw SQL queries have the following safety restrictions:

1. **Only SELECT queries allowed** - No INSERT, UPDATE, DELETE, DROP, etc.
2. **Dangerous keyword detection** - Queries containing DROP, DELETE, INSERT, UPDATE, ALTER, CREATE, TRUNCATE, REPLACE are rejected
3. **Result limit** - Maximum 1000 rows returned
4. **Read-only** - Database is opened in default mode (read-write) but only SELECT is permitted

Example rejected queries:

```bash
# This will be rejected (not SELECT)
radx dicom catalogue --database my.db --sql "DELETE FROM dicom_metadata"
# Error: only SELECT queries are allowed

# This will be rejected (contains dangerous keyword)
radx dicom catalogue --database my.db --sql "SELECT * FROM dicom_metadata; DROP TABLE dicom_metadata"
# Error: query contains dangerous keyword: DROP
```

## Performance

### Indexing Performance

- First-time indexing: ~100-200 files/second (varies by system)
- Subsequent runs: Only new files are indexed (skips existing)
- Database uses indexes on frequently queried fields

### Query Performance

- Indexed queries (patient_id, study_uid, series_uid, sop_uid, modality): Very fast (milliseconds)
- Full-text search: Moderate (sub-second for typical databases)
- Complex SQL joins: Depends on query complexity

### Optimization Tips

1. **Use indexes**: Query by patient_id, study_instance_uid, series_instance_uid, sop_instance_uid, or modality for best performance
2. **Limit results**: Use LIMIT in SQL queries for faster response
3. **Rebuild periodically**: Use `--rebuild` if database becomes fragmented
4. **Separate databases**: Use different databases for different archives

## Output Format

Query results are displayed in a table:

```
Found 23 record(s)

┌──────────────────────┬──────────────────────────┬─────────┬────────┬──────────┬──────────────┐
│ Patient              │ Study                    │ Modality│ Series │ Instance │ File         │
├──────────────────────┼──────────────────────────┼─────────┼────────┼──────────┼──────────────┤
│ SMITH^JOHN (123456)  │ CHEST CT [20240115]      │   CT    │   1    │    45    │ IMG0001.dcm  │
│ DOE^JANE (789012)    │ BRAIN MRI [20240116]     │   MR    │   2    │    12    │ IMG0002.dcm  │
└──────────────────────┴──────────────────────────┴─────────┴────────┴──────────┴──────────────┘
```

SQL query results show all selected columns:

```
SQL Query Results

┌──────────────┬────────────────────┬──────┐
│ modality     │ avg_size           │ count│
├──────────────┼────────────────────┼──────┤
│ CT           │ 524288            │  567  │
│ MR           │ 1048576           │  432  │
│ CR           │ 262144            │  246  │
└──────────────┴────────────────────┴──────┘

Rows returned: 3
```

## Troubleshooting

### Database locked error

```bash
# Another process is using the database
# Wait for other process to finish or close connections
```

### Missing files in catalogue

```bash
# Re-index with rebuild
radx dicom catalogue /data/dicom --database my.db --rebuild
```

### Slow queries

```bash
# Use indexed fields
radx dicom catalogue --database my.db -q "PatientID=12345"  # Fast (indexed)
radx dicom catalogue --database my.db -q "patient_name=John"  # Slower (not indexed)

# Limit SQL results
radx dicom catalogue --database my.db --sql "SELECT * FROM dicom_metadata LIMIT 100"
```

### SQL syntax errors

```bash
# Check your SQL syntax
radx dicom catalogue --database my.db --debug --sql "SELECT * FROM dicom_metadata"
```

## Advanced Examples

### Find duplicate studies

```bash
radx dicom catalogue --database my.db --sql "
  SELECT study_instance_uid, COUNT(*) as count
  FROM dicom_metadata
  GROUP BY study_instance_uid
  HAVING count > 1
  ORDER BY count DESC"
```

### Find studies missing descriptions

```bash
radx dicom catalogue --database my.db --sql "
  SELECT DISTINCT study_instance_uid, patient_name, study_date
  FROM dicom_metadata
  WHERE study_description = '' OR study_description IS NULL"
```

### Calculate storage by modality

```bash
radx dicom catalogue --database my.db --sql "
  SELECT modality,
         COUNT(*) as files,
         ROUND(SUM(file_size) / 1024.0 / 1024.0, 2) as size_mb
  FROM dicom_metadata
  GROUP BY modality
  ORDER BY size_mb DESC"
```

### Export to CSV

```bash
# Use sqlite3 directly for CSV export
sqlite3 -header -csv my.db "SELECT * FROM dicom_metadata" > export.csv
```

## See Also

- [dump](dump.md) - Inspect individual DICOM files
- [lookup](lookup.md) - Look up DICOM tag definitions
- [organize](organize.md) - Reorganize DICOM files
