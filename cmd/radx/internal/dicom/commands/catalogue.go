package commands

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexeyco/simpletable"
	"github.com/charmbracelet/log"
	"github.com/codeninja55/go-radx/cmd/radx/internal/config"
	"github.com/codeninja55/go-radx/cmd/radx/internal/dicom/ui"
	"github.com/codeninja55/go-radx/dicom"
	_ "github.com/mattn/go-sqlite3"
)

// CatalogueCmd implements the DICOM catalogue command with SQLite database.
type CatalogueCmd struct {
	Dir       string   `arg:"" optional:"" type:"existingdir" help:"Directory containing DICOM files to catalogue"`
	Database  string   `name:"database" short:"d" default:"dicom-catalogue.db" help:"SQLite database path"`
	Rebuild   bool     `name:"rebuild" help:"Rebuild database from scratch"`
	Recursive bool     `name:"recursive" short:"R" help:"Recursively search directories" default:"true"`
	Query     []string `name:"query" short:"q" help:"Query tags (format: tag=(GGGG,EEEE), keyword=value, or text search)"`
	SQL       string   `name:"sql" help:"Execute raw SQL query (read-only, safe queries only)"`
}

// DICOMMetadata represents the key metadata stored for each DICOM file.
type DICOMMetadata struct {
	FilePath                string
	FileName                string
	FileSize                int64
	PatientName             string
	PatientID               string
	PatientBirthDate        string
	PatientSex              string
	StudyInstanceUID        string
	StudyDate               string
	StudyDescription        string
	SeriesInstanceUID       string
	SeriesNumber            string
	SeriesDescription       string
	SOPInstanceUID          string
	SOPClassUID             string
	InstanceNumber          string
	Modality                string
	Manufacturer            string
	InstitutionName         string
	AccessionNumber         string
	ReferringPhysicianName  string
	PerformingPhysicianName string
}

const (
	// SQL schema for DICOM metadata table
	createTableSQL = `
	CREATE TABLE IF NOT EXISTS dicom_metadata (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_path TEXT NOT NULL UNIQUE,
		file_name TEXT NOT NULL,
		file_size INTEGER,
		patient_name TEXT,
		patient_id TEXT,
		patient_birth_date TEXT,
		patient_sex TEXT,
		study_instance_uid TEXT,
		study_date TEXT,
		study_description TEXT,
		series_instance_uid TEXT,
		series_number TEXT,
		series_description TEXT,
		sop_instance_uid TEXT,
		sop_class_uid TEXT,
		instance_number TEXT,
		modality TEXT,
		manufacturer TEXT,
		institution_name TEXT,
		accession_number TEXT,
		referring_physician_name TEXT,
		performing_physician_name TEXT,
		indexed_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_patient_id ON dicom_metadata(patient_id);
	CREATE INDEX IF NOT EXISTS idx_study_uid ON dicom_metadata(study_instance_uid);
	CREATE INDEX IF NOT EXISTS idx_series_uid ON dicom_metadata(series_instance_uid);
	CREATE INDEX IF NOT EXISTS idx_sop_uid ON dicom_metadata(sop_instance_uid);
	CREATE INDEX IF NOT EXISTS idx_modality ON dicom_metadata(modality);
	`

	insertMetadataSQL = `
	INSERT OR REPLACE INTO dicom_metadata (
		file_path, file_name, file_size,
		patient_name, patient_id, patient_birth_date, patient_sex,
		study_instance_uid, study_date, study_description,
		series_instance_uid, series_number, series_description,
		sop_instance_uid, sop_class_uid, instance_number,
		modality, manufacturer, institution_name,
		accession_number, referring_physician_name, performing_physician_name
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
)

// Run executes the catalogue command.
func (c *CatalogueCmd) Run(cfg *config.GlobalConfig) error {
	// Print banner
	ui.PrintBanner()

	logger := log.Default()
	logger.Info("Starting DICOM catalogue operation")

	// Open/create database
	db, err := c.openDatabase(logger)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("Failed to close database", "error", err)
		}
	}()

	// If SQL query is provided, execute it and return
	if c.SQL != "" {
		logger.Info("Executing SQL query")
		return c.executeSQLQuery(db, logger)
	}

	// Validate directory is provided for indexing operations
	if c.Dir == "" {
		logger.Info("No directory provided, displaying database summary")
		c.displaySummary(db, logger)
		return nil
	}

	// Rebuild database if requested
	if c.Rebuild {
		logger.Info("Rebuilding database")
		if err := c.rebuildDatabase(db, logger); err != nil {
			return fmt.Errorf("failed to rebuild database: %w", err)
		}
	}

	// Scan directory and index files
	logger.Info("Scanning directory", "path", c.Dir)
	files, err := listDicomFiles(c.Dir, c.Recursive)
	if err != nil {
		return fmt.Errorf("failed to list DICOM files: %w", err)
	}

	if len(files) == 0 {
		logger.Warn("No DICOM files found")
		return nil
	}

	logger.Info("Found DICOM files", "count", len(files))

	// Index files
	indexed, skipped, failed := c.indexFiles(db, files, logger)
	logger.Info("Indexing complete",
		"indexed", indexed,
		"skipped", skipped,
		"failed", failed,
	)

	// Query and display results
	if len(c.Query) > 0 {
		logger.Info("Executing query", "filters", c.Query)
		results, err := c.queryDatabase(db, logger)
		if err != nil {
			return fmt.Errorf("failed to query database: %w", err)
		}

		c.displayResults(results, logger)
	} else {
		// Display summary
		c.displaySummary(db, logger)
	}

	return nil
}

// openDatabase opens or creates the SQLite database.
func (c *CatalogueCmd) openDatabase(logger *log.Logger) (*sql.DB, error) {
	logger.Debug("Opening database", "path", c.Database)

	db, err := sql.Open("sqlite3", c.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create tables
	if _, err := db.Exec(createTableSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return db, nil
}

// rebuildDatabase drops and recreates the database tables.
func (c *CatalogueCmd) rebuildDatabase(db *sql.DB, logger *log.Logger) error {
	logger.Info("Dropping existing tables")
	if _, err := db.Exec("DROP TABLE IF EXISTS dicom_metadata"); err != nil {
		return fmt.Errorf("failed to drop tables: %w", err)
	}

	logger.Info("Creating tables")
	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

// indexFiles indexes DICOM files into the database.
func (c *CatalogueCmd) indexFiles(db *sql.DB, files []DICOMFile, logger *log.Logger) (indexed, skipped, failed int) {
	progress := ui.NewProgressBar(len(files), "Indexing")

	stmt, err := db.Prepare(insertMetadataSQL)
	if err != nil {
		logger.Error("Failed to prepare statement", "error", err)
		return 0, 0, len(files)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			logger.Error("Failed to close statement", "error", err)
		}
	}()

	for _, file := range files {
		progress.Increment(fmt.Sprintf("Indexing %s", file.Name))

		// Check if already indexed
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM dicom_metadata WHERE file_path = ?", file.Path).Scan(&count)
		if err == nil && count > 0 {
			skipped++
			continue
		}

		// Extract metadata
		metadata, err := c.extractMetadata(file, logger)
		if err != nil {
			logger.Error("Failed to extract metadata", "file", file.Path, "error", err)
			failed++
			continue
		}

		// Insert into database
		if err := c.insertMetadata(stmt, metadata); err != nil {
			logger.Error("Failed to insert metadata", "file", file.Path, "error", err)
			failed++
			continue
		}

		indexed++
	}

	progress.Complete("Complete")
	return indexed, skipped, failed
}

// extractMetadata extracts metadata from a DICOM file.
func (c *CatalogueCmd) extractMetadata(file DICOMFile, logger *log.Logger) (*DICOMMetadata, error) {
	dataset, err := dicom.ParseFile(file.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DICOM file: %w", err)
	}

	metadata := &DICOMMetadata{
		FilePath: file.Path,
		FileName: file.Name,
		FileSize: file.Size,
	}

	// Extract relevant tags
	for _, elem := range dataset.Elements() {
		tag := elem.Tag()
		value := elem.Value().String()

		switch {
		case tag.Group == 0x0010 && tag.Element == 0x0010: // Patient Name
			metadata.PatientName = value
		case tag.Group == 0x0010 && tag.Element == 0x0020: // Patient ID
			metadata.PatientID = value
		case tag.Group == 0x0010 && tag.Element == 0x0030: // Patient Birth Date
			metadata.PatientBirthDate = value
		case tag.Group == 0x0010 && tag.Element == 0x0040: // Patient Sex
			metadata.PatientSex = value
		case tag.Group == 0x0020 && tag.Element == 0x000D: // Study Instance UID
			metadata.StudyInstanceUID = value
		case tag.Group == 0x0008 && tag.Element == 0x0020: // Study Date
			metadata.StudyDate = value
		case tag.Group == 0x0008 && tag.Element == 0x1030: // Study Description
			metadata.StudyDescription = value
		case tag.Group == 0x0020 && tag.Element == 0x000E: // Series Instance UID
			metadata.SeriesInstanceUID = value
		case tag.Group == 0x0020 && tag.Element == 0x0011: // Series Number
			metadata.SeriesNumber = value
		case tag.Group == 0x0008 && tag.Element == 0x103E: // Series Description
			metadata.SeriesDescription = value
		case tag.Group == 0x0008 && tag.Element == 0x0018: // SOP Instance UID
			metadata.SOPInstanceUID = value
		case tag.Group == 0x0008 && tag.Element == 0x0016: // SOP Class UID
			metadata.SOPClassUID = value
		case tag.Group == 0x0020 && tag.Element == 0x0013: // Instance Number
			metadata.InstanceNumber = value
		case tag.Group == 0x0008 && tag.Element == 0x0060: // Modality
			metadata.Modality = value
		case tag.Group == 0x0008 && tag.Element == 0x0070: // Manufacturer
			metadata.Manufacturer = value
		case tag.Group == 0x0008 && tag.Element == 0x0080: // Institution Name
			metadata.InstitutionName = value
		case tag.Group == 0x0008 && tag.Element == 0x0050: // Accession Number
			metadata.AccessionNumber = value
		case tag.Group == 0x0008 && tag.Element == 0x0090: // Referring Physician Name
			metadata.ReferringPhysicianName = value
		case tag.Group == 0x0008 && tag.Element == 0x1050: // Performing Physician Name
			metadata.PerformingPhysicianName = value
		}
	}

	return metadata, nil
}

// insertMetadata inserts metadata into the database.
func (c *CatalogueCmd) insertMetadata(stmt *sql.Stmt, metadata *DICOMMetadata) error {
	_, err := stmt.Exec(
		metadata.FilePath,
		metadata.FileName,
		metadata.FileSize,
		metadata.PatientName,
		metadata.PatientID,
		metadata.PatientBirthDate,
		metadata.PatientSex,
		metadata.StudyInstanceUID,
		metadata.StudyDate,
		metadata.StudyDescription,
		metadata.SeriesInstanceUID,
		metadata.SeriesNumber,
		metadata.SeriesDescription,
		metadata.SOPInstanceUID,
		metadata.SOPClassUID,
		metadata.InstanceNumber,
		metadata.Modality,
		metadata.Manufacturer,
		metadata.InstitutionName,
		metadata.AccessionNumber,
		metadata.ReferringPhysicianName,
		metadata.PerformingPhysicianName,
	)
	return err
}

// queryDatabase queries the database based on user filters.
func (c *CatalogueCmd) queryDatabase(db *sql.DB, logger *log.Logger) ([]*DICOMMetadata, error) {
	// Build query
	query := "SELECT * FROM dicom_metadata WHERE 1=1"
	args := make([]interface{}, 0)

	for _, filter := range c.Query {
		whereClause, filterArgs := c.buildWhereClause(filter)
		if whereClause != "" {
			query += " AND " + whereClause
			args = append(args, filterArgs...)
		}
	}

	query += " LIMIT 100" // Limit results for performance

	logger.Debug("Executing query", "sql", query, "args", args)

	// Execute query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			logger.Error("Failed to close rows", "error", err)
		}
	}()

	// Parse results
	results := make([]*DICOMMetadata, 0)
	for rows.Next() {
		var id int
		var indexedAt string
		metadata := &DICOMMetadata{}

		err := rows.Scan(
			&id,
			&metadata.FilePath,
			&metadata.FileName,
			&metadata.FileSize,
			&metadata.PatientName,
			&metadata.PatientID,
			&metadata.PatientBirthDate,
			&metadata.PatientSex,
			&metadata.StudyInstanceUID,
			&metadata.StudyDate,
			&metadata.StudyDescription,
			&metadata.SeriesInstanceUID,
			&metadata.SeriesNumber,
			&metadata.SeriesDescription,
			&metadata.SOPInstanceUID,
			&metadata.SOPClassUID,
			&metadata.InstanceNumber,
			&metadata.Modality,
			&metadata.Manufacturer,
			&metadata.InstitutionName,
			&metadata.AccessionNumber,
			&metadata.ReferringPhysicianName,
			&metadata.PerformingPhysicianName,
			&indexedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		results = append(results, metadata)
	}

	return results, nil
}

// executeSQLQuery executes a raw SQL query safely and displays results.
func (c *CatalogueCmd) executeSQLQuery(db *sql.DB, logger *log.Logger) error {
	// Validate query is read-only (only SELECT statements allowed)
	trimmedSQL := strings.TrimSpace(strings.ToUpper(c.SQL))
	if !strings.HasPrefix(trimmedSQL, "SELECT") {
		return fmt.Errorf("only SELECT queries are allowed for safety (got: %s)", strings.Fields(trimmedSQL)[0])
	}

	// Additional safety check: prevent dangerous keywords
	dangerousKeywords := []string{"DROP", "DELETE", "INSERT", "UPDATE", "ALTER", "CREATE", "TRUNCATE", "REPLACE"}
	for _, keyword := range dangerousKeywords {
		if strings.Contains(trimmedSQL, keyword) {
			return fmt.Errorf("query contains dangerous keyword: %s", keyword)
		}
	}

	logger.Debug("Executing SQL query", "sql", c.SQL)

	// Execute query
	rows, err := db.Query(c.SQL)
	if err != nil {
		return fmt.Errorf("failed to execute SQL query: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			logger.Error("Failed to close rows", "error", err)
		}
	}()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get column names: %w", err)
	}

	// Display results in table format
	fmt.Println()
	fmt.Printf("%s\n\n", ui.InfoStyle.Render("SQL Query Results"))

	table := ui.NewTable()

	// Set headers
	headerCells := make([]*simpletable.Cell, len(columns))
	for i, col := range columns {
		headerCells[i] = &simpletable.Cell{
			Align: simpletable.AlignLeft,
			Text:  col,
		}
	}
	table.Header = &simpletable.Header{Cells: headerCells}

	// Read and display rows
	rowCount := 0
	for rows.Next() {
		// Create slice of interface{} to hold each column value
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			logger.Warn("Failed to scan row", "error", err)
			continue
		}

		// Convert values to strings for display
		rowCells := make([]*simpletable.Cell, len(columns))
		for i, val := range values {
			var strVal string
			if val == nil {
				strVal = ui.SubtleStyle.Render("NULL")
			} else {
				strVal = fmt.Sprintf("%v", val)
				// Truncate long values
				if len(strVal) > 50 {
					strVal = strVal[:47] + "..."
				}
			}
			rowCells[i] = &simpletable.Cell{Text: strVal}
		}
		table.Body.Cells = append(table.Body.Cells, rowCells)
		rowCount++

		// Limit results to prevent overwhelming output
		if rowCount >= 1000 {
			logger.Warn("Result set limited to 1000 rows")
			break
		}
	}

	ui.PrintTable(table, os.Stdout)
	fmt.Println()
	fmt.Printf("%s %s\n\n", ui.SubtleStyle.Render("Rows returned:"), ui.InfoStyle.Render(fmt.Sprintf("%d", rowCount)))

	return nil
}

// buildWhereClause builds a WHERE clause for a query filter.
func (c *CatalogueCmd) buildWhereClause(filter string) (string, []interface{}) {
	// Parse filter format: tag=(value), keyword=value, or text search
	if strings.Contains(filter, "=") {
		parts := strings.SplitN(filter, "=", 2)
		field := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Map field names to database columns
		columnName := c.mapFieldToColumn(field)
		if columnName != "" {
			return fmt.Sprintf("%s LIKE ?", columnName), []interface{}{"%" + value + "%"}
		}
	}

	// Text search across multiple fields
	whereClause := `(
		patient_name LIKE ? OR
		patient_id LIKE ? OR
		study_description LIKE ? OR
		series_description LIKE ? OR
		modality LIKE ? OR
		manufacturer LIKE ? OR
		institution_name LIKE ?
	)`
	searchValue := "%" + filter + "%"
	return whereClause, []interface{}{
		searchValue, searchValue, searchValue, searchValue,
		searchValue, searchValue, searchValue,
	}
}

// mapFieldToColumn maps a field name to a database column.
func (c *CatalogueCmd) mapFieldToColumn(field string) string {
	normalized := strings.ToLower(strings.ReplaceAll(field, " ", "_"))

	mapping := map[string]string{
		"patientname":         "patient_name",
		"patient_name":        "patient_name",
		"patientid":           "patient_id",
		"patient_id":          "patient_id",
		"studyinstanceuid":    "study_instance_uid",
		"study_instance_uid":  "study_instance_uid",
		"studyuid":            "study_instance_uid",
		"seriesinstanceuid":   "series_instance_uid",
		"series_instance_uid": "series_instance_uid",
		"seriesuid":           "series_instance_uid",
		"sopinstanceuid":      "sop_instance_uid",
		"sop_instance_uid":    "sop_instance_uid",
		"instanceuid":         "sop_instance_uid",
		"modality":            "modality",
		"manufacturer":        "manufacturer",
		"institution":         "institution_name",
		"institution_name":    "institution_name",
		"accessionnumber":     "accession_number",
		"accession_number":    "accession_number",
		"studydescription":    "study_description",
		"study_description":   "study_description",
		"seriesdescription":   "series_description",
		"series_description":  "series_description",
	}

	if col, ok := mapping[normalized]; ok {
		return col
	}

	return ""
}

// displayResults displays query results in a table.
func (c *CatalogueCmd) displayResults(results []*DICOMMetadata, logger *log.Logger) {
	if len(results) == 0 {
		fmt.Println()
		fmt.Println(ui.WarnStyle.Render("No matching records found"))
		fmt.Println()
		return
	}

	fmt.Println()
	fmt.Printf("%s %s\n\n", ui.InfoStyle.Render("Found"), ui.SuccessStyle.Render(fmt.Sprintf("%d record(s)", len(results))))

	// Create table with most valuable columns
	table := ui.NewTable()
	table.Header = &simpletable.Header{
		Cells: []*simpletable.Cell{
			{Align: simpletable.AlignLeft, Text: "Patient"},
			{Align: simpletable.AlignLeft, Text: "Study"},
			{Align: simpletable.AlignCenter, Text: "Modality"},
			{Align: simpletable.AlignCenter, Text: "Series"},
			{Align: simpletable.AlignCenter, Text: "Instance"},
			{Align: simpletable.AlignLeft, Text: "File"},
		},
	}

	for _, meta := range results {
		// Truncate long values for display
		patientInfo := meta.PatientName
		if meta.PatientID != "" {
			patientInfo = fmt.Sprintf("%s (%s)", meta.PatientName, meta.PatientID)
		}
		if len(patientInfo) > 25 {
			patientInfo = patientInfo[:22] + "..."
		}

		studyInfo := meta.StudyDescription
		if meta.StudyDate != "" {
			studyInfo = fmt.Sprintf("%s [%s]", meta.StudyDescription, meta.StudyDate)
		}
		if len(studyInfo) > 30 {
			studyInfo = studyInfo[:27] + "..."
		}

		fileName := filepath.Base(meta.FilePath)
		if len(fileName) > 30 {
			fileName = fileName[:27] + "..."
		}

		table.Body.Cells = append(table.Body.Cells, []*simpletable.Cell{
			{Text: ui.InfoStyle.Render(patientInfo)},
			{Text: studyInfo},
			{Text: ui.SuccessStyle.Render(meta.Modality)},
			{Text: meta.SeriesNumber},
			{Text: meta.InstanceNumber},
			{Text: ui.SubtleStyle.Render(fileName)},
		})
	}

	ui.PrintTable(table, os.Stdout)
	fmt.Println()
}

// displaySummary displays a summary of the database contents.
func (c *CatalogueCmd) displaySummary(db *sql.DB, logger *log.Logger) {
	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("✓ Database catalogue complete"))
	fmt.Println()

	// Count total files
	var totalFiles int
	_ = db.QueryRow("SELECT COUNT(*) FROM dicom_metadata").Scan(&totalFiles)

	// Count unique patients
	var uniquePatients int
	_ = db.QueryRow("SELECT COUNT(DISTINCT patient_id) FROM dicom_metadata WHERE patient_id != ''").Scan(&uniquePatients)

	// Count unique studies
	var uniqueStudies int
	_ = db.QueryRow("SELECT COUNT(DISTINCT study_instance_uid) FROM dicom_metadata WHERE study_instance_uid != ''").Scan(&uniqueStudies)

	// Count unique series
	var uniqueSeries int
	_ = db.QueryRow("SELECT COUNT(DISTINCT series_instance_uid) FROM dicom_metadata WHERE series_instance_uid != ''").Scan(&uniqueSeries)

	// Get modality breakdown
	rows, err := db.Query("SELECT modality, COUNT(*) as count FROM dicom_metadata WHERE modality != '' GROUP BY modality ORDER BY count DESC LIMIT 10")
	if err != nil {
		logger.Warn("Failed to query modality breakdown", "error", err)
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			logger.Error("Failed to close rows", "error", err)
		}
	}()

	modalities := make(map[string]int)
	for rows.Next() {
		var modality string
		var count int
		if err := rows.Scan(&modality, &count); err != nil {
			logger.Warn("Failed to scan modality row", "error", err)
			continue
		}
		modalities[modality] = count
	}

	// Print summary
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Database:"), ui.InfoStyle.Render(c.Database))
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Total Files:"), ui.InfoStyle.Render(fmt.Sprintf("%d", totalFiles)))
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Unique Patients:"), ui.InfoStyle.Render(fmt.Sprintf("%d", uniquePatients)))
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Unique Studies:"), ui.InfoStyle.Render(fmt.Sprintf("%d", uniqueStudies)))
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Unique Series:"), ui.InfoStyle.Render(fmt.Sprintf("%d", uniqueSeries)))

	if len(modalities) > 0 {
		fmt.Println()
		fmt.Println(ui.SubtleStyle.Render("  Modality Breakdown:"))
		for modality, count := range modalities {
			fmt.Printf("    %s %s\n", ui.SuccessStyle.Render(modality+":"), ui.InfoStyle.Render(fmt.Sprintf("%d files", count)))
		}
	}

	fmt.Println()
	fmt.Println(ui.SubtleStyle.Render("  Use --query to search the catalogue"))
	fmt.Println()
}
