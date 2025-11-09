package dicom

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/codeninja55/go-radx/dicom/tag"
)

// FileNamingStrategy controls how output files are named.
type FileNamingStrategy int

const (
	// FileNamingOriginal preserves original filenames from source paths.
	// Falls back to SOPInstanceUID if source path unavailable.
	FileNamingOriginal FileNamingStrategy = iota

	// FileNamingSOPInstanceUID names files as {SOPInstanceUID}.dcm.
	FileNamingSOPInstanceUID
)

// DirectoryWriteOptions configures directory writing behavior.
type DirectoryWriteOptions struct {
	// Workers specifies the number of concurrent writing workers.
	// Default: runtime.GOMAXPROCS(0)
	Workers int

	// Hierarchical enables Patient/Study/Series directory structure.
	// Default: false (flat structure)
	Hierarchical bool

	// FileNaming controls output filename strategy.
	// In hierarchical mode, SOPInstanceUID naming is enforced.
	// Default: FileNamingOriginal
	FileNaming FileNamingStrategy

	// PatientFolderNaming customizes patient folder names in hierarchical mode.
	// Receives dataset, should return folder name (e.g., "PatientID" or "PatientID_Name").
	// Default: uses PatientID, falls back to StudyInstanceUID
	PatientFolderNaming func(*DataSet) (string, error)

	// FallbackFlat enables fallback to _flat/ subdirectory for datasets missing required UIDs.
	// Only applies in hierarchical mode.
	// Default: true
	FallbackFlat bool

	// WriteOptions are passed to individual file writes.
	WriteOptions WriteOptions

	// ProgressCallback is called after each file write.
	ProgressCallback func(current, total int)
}

// DirectoryWriteResult contains the results of a directory write operation.
type DirectoryWriteResult struct {
	// Written is the count of successfully written files.
	Written int

	// Failed is the count of files that failed to write.
	Failed int

	// Errors maps SOPInstanceUID to write errors.
	Errors map[string]error

	// FallbackFiles lists SOPInstanceUIDs written to _flat/ due to missing UIDs.
	// Only populated in hierarchical mode with FallbackFlat enabled.
	FallbackFiles []string

	// Duration is the total time taken for the write operation.
	Duration time.Duration
}

// WriteDirectory writes a DataSetCollection to a directory in flat structure.
//
// Files are written with original filenames preserved (if available) or SOPInstanceUID.dcm as fallback.
//
// Example:
//
//	result, err := dicom.WriteDirectory("/output", collection)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Wrote %d files, %d failed\n", result.Written, result.Failed)
func WriteDirectory(dir string, collection *DataSetCollection) (*DirectoryWriteResult, error) {
	return WriteDirectoryWithOptions(dir, collection, DirectoryWriteOptions{})
}

// WriteDirectoryWithOptions writes a DataSetCollection with configurable options.
//
// Supports both flat and hierarchical directory structures:
//
// Flat mode (default):
//   - All files in single directory
//   - Filenames: original or {SOPInstanceUID}.dcm
//
// Hierarchical mode:
//   - Structure: Patient/Study/Series/{SOPInstanceUID}.dcm
//   - Patient folder: PatientID (or custom via PatientFolderNaming)
//   - Fallback to _flat/ for datasets missing required UIDs
//
// Example flat:
//
//	opts := dicom.DirectoryWriteOptions{
//	    Workers: 4,
//	    FileNaming: dicom.FileNamingOriginal,
//	}
//	result, err := dicom.WriteDirectoryWithOptions("/output", collection, opts)
//
// Example hierarchical:
//
//	opts := dicom.DirectoryWriteOptions{
//	    Hierarchical: true,
//	    FallbackFlat: true,
//	}
//	result, err := dicom.WriteDirectoryWithOptions("/output", collection, opts)
func WriteDirectoryWithOptions(dir string, collection *DataSetCollection, opts DirectoryWriteOptions) (*DirectoryWriteResult, error) {
	startTime := time.Now()

	if collection == nil {
		return nil, fmt.Errorf("cannot write nil collection")
	}

	// Apply defaults
	opts = applyDefaultDirectoryWriteOptions(opts)

	// Get all datasets
	datasets := collection.DataSets()

	// Handle empty collection
	if len(datasets) == 0 {
		return &DirectoryWriteResult{
			Written:       0,
			Failed:        0,
			Errors:        make(map[string]error),
			FallbackFiles: []string{},
			Duration:      time.Since(startTime),
		}, nil
	}

	// Write files concurrently
	result := writeFilesConcurrently(dir, datasets, opts)
	result.Duration = time.Since(startTime)

	return result, nil
}

// applyDefaultDirectoryWriteOptions fills in missing options with defaults.
func applyDefaultDirectoryWriteOptions(opts DirectoryWriteOptions) DirectoryWriteOptions {
	if opts.Workers <= 0 {
		opts.Workers = runtime.GOMAXPROCS(0)
	}

	if opts.PatientFolderNaming == nil {
		opts.PatientFolderNaming = defaultPatientFolderNaming
	}

	// Apply defaults for nested WriteOptions
	opts.WriteOptions = applyDefaultWriteOptions(opts.WriteOptions)

	// For directory operations, we need CreateDirs to be true to create
	// the hierarchical directory structure. Override if not explicitly set.
	if !opts.WriteOptions.CreateDirs {
		opts.WriteOptions.CreateDirs = true
	}

	// Default Atomic to true for safer writes
	if !opts.WriteOptions.Atomic {
		opts.WriteOptions.Atomic = true
	}

	return opts
}

// defaultPatientFolderNaming returns PatientID, falling back to StudyInstanceUID.
func defaultPatientFolderNaming(ds *DataSet) (string, error) {
	// Try PatientID first
	patientIDElem, err := ds.Get(tag.New(0x0010, 0x0020))
	if err == nil {
		patientID := strings.TrimSpace(patientIDElem.Value().String())
		if patientID != "" {
			return sanitizePathComponent(patientID), nil
		}
	}

	// Fallback to StudyInstanceUID
	studyUIDElem, err := ds.Get(tag.New(0x0020, 0x000D))
	if err == nil {
		studyUID := strings.TrimSpace(studyUIDElem.Value().String())
		if studyUID != "" {
			return sanitizePathComponent(studyUID), nil
		}
	}

	return "", fmt.Errorf("dataset missing both PatientID and StudyInstanceUID")
}

// writeFilesConcurrently distributes file writing across worker goroutines.
func writeFilesConcurrently(dir string, datasets []*DataSet, opts DirectoryWriteOptions) *DirectoryWriteResult {
	errors := make(map[string]error)
	var errorsMu sync.Mutex

	var fallbackFiles []string
	var fallbackMu sync.Mutex

	written := 0
	failed := 0
	var statsMu sync.Mutex

	current := 0
	var progressMu sync.Mutex

	total := len(datasets)

	// Create job and result channels
	jobs := make(chan *DataSet, len(datasets))
	results := make(chan writeFileResult, len(datasets))

	// Start worker pool
	var wg sync.WaitGroup
	for w := 0; w < opts.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			writeWorker(dir, jobs, results, opts)
		}()
	}

	// Send jobs
	for _, ds := range datasets {
		jobs <- ds
	}
	close(jobs)

	// Close results when all workers done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for result := range results {
		current++

		if result.err != nil {
			statsMu.Lock()
			failed++
			statsMu.Unlock()

			errorsMu.Lock()
			errors[result.sopInstanceUID] = result.err
			errorsMu.Unlock()
		} else {
			statsMu.Lock()
			written++
			statsMu.Unlock()

			if result.isFallback {
				fallbackMu.Lock()
				fallbackFiles = append(fallbackFiles, result.sopInstanceUID)
				fallbackMu.Unlock()
			}
		}

		// Progress callback
		if opts.ProgressCallback != nil {
			progressMu.Lock()
			opts.ProgressCallback(current, total)
			progressMu.Unlock()
		}
	}

	return &DirectoryWriteResult{
		Written:       written,
		Failed:        failed,
		Errors:        errors,
		FallbackFiles: fallbackFiles,
	}
}

// writeFileResult holds the result of writing a single file.
type writeFileResult struct {
	sopInstanceUID string
	err            error
	isFallback     bool // true if written to _flat/ subdirectory
}

// writeWorker processes write jobs from the jobs channel.
func writeWorker(baseDir string, jobs <-chan *DataSet, results chan<- writeFileResult, opts DirectoryWriteOptions) {
	for ds := range jobs {
		// Extract SOPInstanceUID for result tracking
		sopInstanceUID := extractSOPInstanceUID(ds)

		// Determine output path
		var outputPath string
		var isFallback bool
		var err error

		if opts.Hierarchical {
			outputPath, isFallback, err = generateHierarchicalPath(baseDir, ds, opts)
		} else {
			outputPath, err = generateFlatPath(baseDir, ds, opts)
		}

		if err != nil {
			results <- writeFileResult{
				sopInstanceUID: sopInstanceUID,
				err:            fmt.Errorf("failed to generate path: %w", err),
			}
			continue
		}

		// Write the file
		if err := WriteFileWithOptions(outputPath, ds, opts.WriteOptions); err != nil {
			results <- writeFileResult{
				sopInstanceUID: sopInstanceUID,
				err:            fmt.Errorf("failed to write file: %w", err),
			}
			continue
		}

		results <- writeFileResult{
			sopInstanceUID: sopInstanceUID,
			err:            nil,
			isFallback:     isFallback,
		}
	}
}

// extractSOPInstanceUID extracts SOPInstanceUID from dataset, returns "unknown" if missing.
func extractSOPInstanceUID(ds *DataSet) string {
	elem, err := ds.Get(tag.New(0x0008, 0x0018))
	if err != nil {
		return "unknown"
	}
	uid := strings.TrimSpace(elem.Value().String())
	if uid == "" {
		return "unknown"
	}
	return uid
}

// generateFlatPath generates output path for flat directory structure.
func generateFlatPath(baseDir string, ds *DataSet, opts DirectoryWriteOptions) (string, error) {
	// FileNamingOriginal: use original filename if available
	// TODO(human): Implement source path tracking
	// For now, both naming options use SOPInstanceUID
	sopInstanceUID := extractSOPInstanceUID(ds)
	if sopInstanceUID == "unknown" {
		return "", fmt.Errorf("missing SOPInstanceUID required for filename")
	}
	filename := sopInstanceUID + ".dcm"

	return filepath.Join(baseDir, filename), nil
}

// generateHierarchicalPath generates output path for Patient/Study/Series hierarchy.
func generateHierarchicalPath(baseDir string, ds *DataSet, opts DirectoryWriteOptions) (string, bool, error) {
	// Extract required UIDs
	studyUID, err := extractUID(ds, tag.New(0x0020, 0x000D))
	if err != nil && opts.FallbackFlat {
		return generateFallbackPath(baseDir, ds)
	} else if err != nil {
		return "", false, fmt.Errorf("missing StudyInstanceUID: %w", err)
	}

	seriesUID, err := extractUID(ds, tag.New(0x0020, 0x000E))
	if err != nil && opts.FallbackFlat {
		return generateFallbackPath(baseDir, ds)
	} else if err != nil {
		return "", false, fmt.Errorf("missing SeriesInstanceUID: %w", err)
	}

	sopInstanceUID := extractSOPInstanceUID(ds)
	if sopInstanceUID == "unknown" {
		return "", false, fmt.Errorf("missing SOPInstanceUID")
	}

	// Get patient folder name
	patientFolder, err := opts.PatientFolderNaming(ds)
	if err != nil && opts.FallbackFlat {
		return generateFallbackPath(baseDir, ds)
	} else if err != nil {
		return "", false, fmt.Errorf("failed to determine patient folder: %w", err)
	}

	// Build hierarchical path
	path := filepath.Join(
		baseDir,
		patientFolder,
		sanitizePathComponent(studyUID),
		sanitizePathComponent(seriesUID),
		sopInstanceUID+".dcm",
	)

	return path, false, nil
}

// generateFallbackPath generates path in _flat/ subdirectory for datasets missing required UIDs.
func generateFallbackPath(baseDir string, ds *DataSet) (string, bool, error) {
	sopInstanceUID := extractSOPInstanceUID(ds)
	if sopInstanceUID == "unknown" {
		return "", false, fmt.Errorf("missing SOPInstanceUID even for fallback")
	}

	path := filepath.Join(baseDir, "_flat", sopInstanceUID+".dcm")
	return path, true, nil
}

// extractUID extracts a UID from dataset.
func extractUID(ds *DataSet, t tag.Tag) (string, error) {
	elem, err := ds.Get(t)
	if err != nil {
		return "", err
	}
	uid := strings.TrimSpace(elem.Value().String())
	if uid == "" {
		return "", fmt.Errorf("UID is empty")
	}
	return uid, nil
}

// sanitizePathComponent removes or replaces characters unsafe for filesystem paths.
func sanitizePathComponent(s string) string {
	// Replace path separators and other unsafe characters
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "*", "_")
	s = strings.ReplaceAll(s, "?", "_")
	s = strings.ReplaceAll(s, "\"", "_")
	s = strings.ReplaceAll(s, "<", "_")
	s = strings.ReplaceAll(s, ">", "_")
	s = strings.ReplaceAll(s, "|", "_")

	// Trim leading/trailing spaces and dots
	s = strings.Trim(s, " .")

	// Truncate if too long (255 char limit for most filesystems)
	if len(s) > 200 {
		s = s[:200]
	}

	return s
}
