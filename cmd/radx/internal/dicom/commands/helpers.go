package commands

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/codeninja55/go-radx/cmd/radx/internal/config"
	"github.com/codeninja55/go-radx/cmd/radx/internal/dicom/ui"
)

// DICOMFile represents a DICOM file with basic metadata.
type DICOMFile struct {
	Path string
	Name string
	Size int64
}

// DICOMTag represents a single DICOM tag with its metadata.
type DICOMTag struct {
	Tag   string `json:"tag"`
	VR    string `json:"vr"`
	Name  string `json:"name"`
	Value string `json:"value"`
	File  string `json:"file,omitempty"`
}

// listDicomFiles finds all DICOM files in a directory (recursively if requested).
func listDicomFiles(dirPath string, recursive bool) ([]DICOMFile, error) {
	var files []DICOMFile

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			if !recursive && path != dirPath {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file has .dcm extension or starts with DICM magic
		if strings.HasSuffix(strings.ToLower(path), ".dcm") || isDICOMFile(path) {
			files = append(files, DICOMFile{
				Path: path,
				Name: filepath.Base(path),
				Size: info.Size(),
			})
		}

		return nil
	}

	err := filepath.Walk(dirPath, walkFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return files, nil
}

// isDICOMFile checks if a file starts with the DICOM magic number "DICM".
func isDICOMFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	// Read first 132 bytes (128 byte preamble + "DICM")
	buf := make([]byte, 132)
	n, err := f.Read(buf)
	if err != nil || n < 132 {
		return false
	}

	// Check for "DICM" at offset 128
	return string(buf[128:132]) == "DICM"
}

// validateDicomFile performs basic validation on a DICOM file.
func validateDicomFile(path string) error {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("file does not exist: %w", err)
	}

	// Check if it's a file
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file")
	}

	// Check if file is empty
	if info.Size() == 0 {
		return fmt.Errorf("file is empty")
	}

	// Check DICOM magic number
	if !isDICOMFile(path) {
		return fmt.Errorf("file does not appear to be a valid DICOM file (missing DICM magic number)")
	}

	return nil
}

// formatDicomValue formats a DICOM value for display, truncating if necessary.
func formatDicomValue(value interface{}, maxLength int) string {
	var str string

	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	case time.Time:
		str = v.Format("2006-01-02 15:04:05")
	case int, int8, int16, int32, int64:
		str = fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		str = fmt.Sprintf("%d", v)
	case float32, float64:
		str = fmt.Sprintf("%.6f", v)
	case []interface{}:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			parts = append(parts, formatDicomValue(item, 0))
		}
		str = strings.Join(parts, ", ")
	default:
		str = fmt.Sprintf("%v", v)
	}

	// Truncate if necessary
	if maxLength > 0 && len(str) > maxLength {
		return str[:maxLength-3] + "..."
	}

	return str
}

// createOutputDirectory creates the output directory if it doesn't exist.
func createOutputDirectory(path string) error {
	if path == "." || path == "" {
		return nil // Current directory always exists
	}

	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("output path exists but is not a directory: %s", path)
		}
		return nil // Directory already exists
	}

	if os.IsNotExist(err) {
		err = os.MkdirAll(path, 0o755)
		if err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		return nil
	}

	return fmt.Errorf("failed to check output directory: %w", err)
}

// generateUID generates a DICOM UID using a timestamp-based approach.
// Format: 1.2.840.113619.2.5.<timestamp>.<random>
func generateUID() string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("1.2.840.113619.2.5.%d.%d", timestamp/1000000, timestamp%1000000)
}

// normalizeTagFilter normalizes a tag filter string for comparison.
// Supports formats: (GGGG,EEEE), GGGGEEEE, or keyword (e.g., PatientName)
func normalizeTagFilter(filter string) string {
	// Remove common formatting characters
	normalized := strings.ReplaceAll(filter, "(", "")
	normalized = strings.ReplaceAll(normalized, ")", "")
	normalized = strings.ReplaceAll(normalized, ",", "")
	normalized = strings.ReplaceAll(normalized, " ", "")

	// Convert to lowercase for case-insensitive matching
	normalized = strings.ToLower(normalized)

	return normalized
}

// Output formatting functions

// renderAsJSON renders DICOM tags as JSON to the given writer.
func renderAsJSON(tags []DICOMTag, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(tags)
}

// renderAsTable renders DICOM tags as an ASCII table to the given writer.
func renderAsTable(tags []DICOMTag, w io.Writer) error {
	table := ui.NewDICOMTagTable()

	for _, tag := range tags {
		// Truncate long values for table display
		displayValue := formatDicomValue(tag.Value, 60)
		ui.AddDICOMTagRow(table, tag.Tag, tag.VR, tag.Name, displayValue)
	}

	ui.PrintTable(table, w)
	return nil
}

// renderAsCSV renders DICOM tags as CSV to the given writer.
func renderAsCSV(tags []DICOMTag, w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write header
	header := []string{"Tag", "VR", "Name", "Value"}
	if len(tags) > 0 && tags[0].File != "" {
		header = append(header, "File")
	}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write rows
	for _, tag := range tags {
		row := []string{tag.Tag, tag.VR, tag.Name, tag.Value}
		if tag.File != "" {
			row = append(row, tag.File)
		}
		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// RenderOutput renders DICOM tags in the specified format.
func RenderOutput(tags []DICOMTag, format config.OutputFormat, w io.Writer) error {
	switch format {
	case config.FormatJSON:
		return renderAsJSON(tags, w)
	case config.FormatTable:
		return renderAsTable(tags, w)
	case config.FormatCSV:
		return renderAsCSV(tags, w)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
