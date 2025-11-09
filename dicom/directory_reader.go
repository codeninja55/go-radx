package dicom

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ParseDirectoryOptions configures directory parsing behavior.
type ParseDirectoryOptions struct {
	// Workers specifies the number of concurrent parsing workers.
	// Default: runtime.GOMAXPROCS(0)
	Workers int

	// Recursive enables recursive directory traversal.
	// If nil, defaults to true (recursive enabled).
	// Set to false to scan only the top-level directory.
	Recursive *bool

	// FilePattern specifies the glob pattern for file filtering.
	// Default: "*.dcm"
	FilePattern string

	// ProgressCallback is called after each file is processed.
	// Parameters: current file count, total file count, current file path
	ProgressCallback func(current, total int, path string)

	// ErrorCallback is called when a file fails to parse.
	// Return true to continue parsing, false to abort.
	// Parameters: file path, error
	ErrorCallback func(path string, err error) bool

	// FailFast stops parsing immediately on first error.
	// Default: false
	FailFast bool

	// Context allows cancellation of the parsing operation.
	// If nil, a background context will be used.
	Context context.Context
}

// ParseResult contains the results of a directory parsing operation.
type ParseResult struct {
	// Collection contains all successfully parsed datasets.
	Collection *DataSetCollection

	// Parsed is the count of successfully parsed files.
	Parsed int

	// Failed is the count of files that failed to parse.
	Failed int

	// Errors maps file paths to their parse errors.
	Errors map[string]error

	// Duration is the total time taken for the parsing operation.
	Duration time.Duration
}

// ParseDirectory recursively parses all DICOM files in the specified directory
// and returns a DataSetCollection containing all successfully parsed datasets.
//
// This is a convenience function that uses default options:
//   - Recursive traversal enabled
//   - Worker count: runtime.GOMAXPROCS(0)
//   - File pattern: "*.dcm"
//   - Continue on errors (no fail-fast)
//
// For more control over parsing behavior, use ParseDirectoryWithOptions.
func ParseDirectory(path string) (*ParseResult, error) {
	return ParseDirectoryWithOptions(path, ParseDirectoryOptions{})
}

// ParseDirectoryWithOptions parses all DICOM files in the specified directory
// using the provided options to configure parsing behavior.
//
// The function operates in three phases:
//  1. File Discovery: Recursively walks the directory tree to find matching files
//  2. Concurrent Parsing: Distributes files across worker goroutines for parsing
//  3. Result Aggregation: Collects parsed datasets into a DataSetCollection
//
// Error Handling:
//   - Individual file errors do not stop the entire operation by default
//   - Errors are collected in ParseResult.Errors map
//   - ErrorCallback can be used to handle errors or abort parsing
//   - FailFast option stops parsing immediately on first error
//
// Thread Safety:
//   - All operations are thread-safe
//   - DataSetCollection is safely built across concurrent workers
//   - Progress and error callbacks are called with proper synchronization
//
// Example:
//
//	opts := dicom.ParseDirectoryOptions{
//	    Workers: 4,
//	    Recursive: true,
//	    ProgressCallback: func(current, total int, path string) {
//	        fmt.Printf("Parsing %d/%d: %s\n", current, total, path)
//	    },
//	}
//	result, err := dicom.ParseDirectoryWithOptions("/path/to/dicom", opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Parsed %d files, %d failed\n", result.Parsed, result.Failed)
func ParseDirectoryWithOptions(path string, opts ParseDirectoryOptions) (*ParseResult, error) {
	startTime := time.Now()

	// Validate directory path
	if err := validateDirectory(path); err != nil {
		return nil, err
	}

	// Apply default options
	opts = applyDefaultOptions(opts)

	// Discover all matching files
	files, err := discoverFiles(path, opts)
	if err != nil {
		return nil, fmt.Errorf("file discovery failed: %w", err)
	}

	// Handle empty directory
	if len(files) == 0 {
		return &ParseResult{
			Collection: NewDataSetCollection(),
			Parsed:     0,
			Failed:     0,
			Errors:     make(map[string]error),
			Duration:   time.Since(startTime),
		}, nil
	}

	// Parse files concurrently
	result := parseFileConcurrently(files, opts)
	result.Duration = time.Since(startTime)

	return result, nil
}

// validateDirectory checks if the path exists and is a directory.
func validateDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", path)
		}
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied accessing directory: %s", path)
		}
		return fmt.Errorf("error accessing directory %s: %w", path, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	return nil
}

// applyDefaultOptions fills in missing options with sensible defaults.
func applyDefaultOptions(opts ParseDirectoryOptions) ParseDirectoryOptions {
	if opts.Workers <= 0 {
		opts.Workers = runtime.GOMAXPROCS(0)
	}
	if opts.FilePattern == "" {
		opts.FilePattern = "*.dcm"
	}
	if opts.Context == nil {
		opts.Context = context.Background()
	}
	if opts.Recursive == nil {
		recursive := true
		opts.Recursive = &recursive
	}

	return opts
}

// discoverFiles walks the directory tree and returns all matching file paths.
func discoverFiles(root string, opts ParseDirectoryOptions) ([]string, error) {
	var files []string
	var mu sync.Mutex

	// Convert to absolute path
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories/files with permission errors but continue walking
			return nil
		}

		// Skip directories
		if info.IsDir() {
			// For non-recursive mode, skip subdirectories
			if opts.Recursive != nil && !*opts.Recursive && path != absRoot {
				return filepath.SkipDir
			}
			return nil
		}

		// Match file pattern (case-insensitive on file name)
		matched, err := filepath.Match(strings.ToLower(opts.FilePattern), strings.ToLower(filepath.Base(path)))
		if err != nil {
			return fmt.Errorf("invalid file pattern %q: %w", opts.FilePattern, err)
		}

		if matched {
			mu.Lock()
			files = append(files, path)
			mu.Unlock()
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("directory walk failed: %w", err)
	}

	return files, nil
}

// parseFileConcurrently distributes file parsing across worker goroutines
// and aggregates results into a ParseResult.
func parseFileConcurrently(files []string, opts ParseDirectoryOptions) *ParseResult {
	collection := NewDataSetCollection()
	errors := make(map[string]error)
	var errorsMu sync.Mutex
	var progressMu sync.Mutex

	parsed := 0
	failed := 0
	current := 0
	total := len(files)

	// Create job and result channels
	jobs := make(chan string, len(files))
	results := make(chan parseFileResult, len(files))

	// Start worker pool
	var wg sync.WaitGroup
	for w := 0; w < opts.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			parseWorker(jobs, results, opts.Context)
		}()
	}

	// Send jobs to workers
	for _, filePath := range files {
		jobs <- filePath
	}
	close(jobs)

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for result := range results {
		current++

		if result.err != nil {
			failed++
			errorsMu.Lock()
			errors[result.path] = result.err
			errorsMu.Unlock()

			// Handle error callback
			if opts.ErrorCallback != nil {
				if !opts.ErrorCallback(result.path, result.err) {
					// Abort parsing if callback returns false
					break
				}
			}

			// Fail-fast mode
			if opts.FailFast {
				break
			}
		} else {
			// Add dataset to collection
			if err := collection.Add(result.dataset); err != nil {
				failed++
				errorsMu.Lock()
				errors[result.path] = fmt.Errorf("failed to add to collection: %w", err)
				errorsMu.Unlock()
			} else {
				parsed++
			}
		}

		// Progress callback
		if opts.ProgressCallback != nil {
			progressMu.Lock()
			opts.ProgressCallback(current, total, result.path)
			progressMu.Unlock()
		}
	}

	return &ParseResult{
		Collection: collection,
		Parsed:     parsed,
		Failed:     failed,
		Errors:     errors,
	}
}

// parseFileResult holds the result of parsing a single file.
type parseFileResult struct {
	path    string
	dataset *DataSet
	err     error
}

// parseWorker is a worker goroutine that parses files from the jobs channel
// and sends results to the results channel.
func parseWorker(jobs <-chan string, results chan<- parseFileResult, ctx context.Context) {
	for filePath := range jobs {
		// Check for cancellation
		select {
		case <-ctx.Done():
			results <- parseFileResult{
				path: filePath,
				err:  ctx.Err(),
			}
			return
		default:
		}

		// Parse the file
		dataset, err := ParseFile(filePath)
		results <- parseFileResult{
			path:    filePath,
			dataset: dataset,
			err:     err,
		}
	}
}
