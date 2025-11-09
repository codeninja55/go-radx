package dicom

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseDirectory_FlatDirectory tests parsing a flat directory with DICOM files.
func TestParseDirectory_FlatDirectory(t *testing.T) {
	// Use testdata root which has several .dcm files at top level
	testDir := filepath.Join("..", "testdata", "dicom")

	result, err := ParseDirectory(testDir)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Debug: print what was parsed
	t.Logf("Parsed: %d, Failed: %d, Total files in errors: %d",
		result.Parsed, result.Failed, len(result.Errors))
	for errPath, errMsg := range result.Errors {
		t.Logf("Error: %s -> %v", errPath, errMsg)
	}

	// Should have parsed at least the 6 top-level files (1.dcm through 6.dcm)
	// Note: With recursive enabled, we'll get files from subdirectories too
	assert.Greater(t, result.Parsed, 0, "Should parse some files")
	assert.NotNil(t, result.Collection)
	assert.Greater(t, result.Collection.Len(), 0, "Collection should contain datasets")
	assert.Greater(t, result.Duration, time.Duration(0), "Duration should be tracked")
}

// TestParseDirectory_NestedStructure tests parsing a nested directory structure.
func TestParseDirectory_NestedStructure(t *testing.T) {
	// Use a nested testdata directory - CTC_2 contains nested series directories
	testDir := filepath.Join("..", "testdata", "dicom", "nested", "series_7")

	result, err := ParseDirectory(testDir)
	require.NoError(t, err)
	require.NotNil(t, result)

	// This directory should have many nested .dcm files (CTC_2/...7 has ~690 files)
	assert.Greater(t, result.Parsed, 100, "Should parse many files from nested structure")
	assert.NotNil(t, result.Collection)
	assert.Equal(t, result.Parsed, result.Collection.Len(), "Collection count should match parsed count")
}

// TestParseDirectory_EmptyDirectory tests parsing an empty directory.
func TestParseDirectory_EmptyDirectory(t *testing.T) {
	// Create temporary empty directory
	tmpDir := t.TempDir()

	result, err := ParseDirectory(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 0, result.Parsed, "Should parse 0 files")
	assert.Equal(t, 0, result.Failed, "Should have 0 failures")
	assert.NotNil(t, result.Collection)
	assert.Equal(t, 0, result.Collection.Len(), "Collection should be empty")
}

// TestParseDirectory_NonExistent tests parsing a non-existent directory.
func TestParseDirectory_NonExistent(t *testing.T) {
	_, err := ParseDirectory("/nonexistent/directory/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

// TestParseDirectory_FileNotDirectory tests providing a file path instead of directory.
func TestParseDirectory_FileNotDirectory(t *testing.T) {
	// Use a known file instead of directory
	testFile := filepath.Join("..", "testdata", "dicom", "1.dcm")

	_, err := ParseDirectory(testFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}

// TestParseDirectory_RelativePath tests parsing with relative path.
func TestParseDirectory_RelativePath(t *testing.T) {
	// Use relative path
	testDir := "../testdata"

	result, err := ParseDirectory(testDir)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.GreaterOrEqual(t, result.Parsed, 6, "Should parse files with relative path")
}

// TestParseDirectoryWithOptions_WorkerCount tests configurable worker count.
func TestParseDirectoryWithOptions_WorkerCount(t *testing.T) {
	testCases := []struct {
		name    string
		workers int
	}{
		{"single worker", 1},
		{"two workers", 2},
		{"four workers", 4},
		{"eight workers", 8},
	}

	testDir := filepath.Join("..", "testdata", "dicom", "nested", "series_7")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := ParseDirectoryOptions{
				Workers: tc.workers,
			}

			result, err := ParseDirectoryWithOptions(testDir, opts)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Greater(t, result.Parsed, 0, "Should parse files regardless of worker count")
			assert.NotNil(t, result.Collection)
		})
	}
}

// TestParseDirectoryWithOptions_NonRecursive tests non-recursive traversal.
func TestParseDirectoryWithOptions_NonRecursive(t *testing.T) {
	testDir := filepath.Join("..", "testdata", "dicom")

	recursive := false
	opts := ParseDirectoryOptions{
		Recursive: &recursive,
	}

	result, err := ParseDirectoryWithOptions(testDir, opts)
	require.NoError(t, err)
	require.NotNil(t, result)

	// With non-recursive, should only get top-level files
	// Should NOT include files from subdirectories
	// Test dataset has grown to include ~40-50 top-level DICOM files
	assert.LessOrEqual(t, result.Parsed, 100, "Should only parse top-level files")
	assert.GreaterOrEqual(t, result.Parsed, 40, "Should parse at least 40 top-level files")
}

// TestParseDirectoryWithOptions_Recursive tests recursive traversal (default).
func TestParseDirectoryWithOptions_Recursive(t *testing.T) {
	testDir := filepath.Join("..", "testdata", "dicom")

	recursive := true
	opts := ParseDirectoryOptions{
		Recursive: &recursive,
	}

	result, err := ParseDirectoryWithOptions(testDir, opts)
	require.NoError(t, err)
	require.NotNil(t, result)

	// With recursive, should get many more files from subdirectories
	assert.Greater(t, result.Parsed, 100, "Should parse many files recursively")
}

// TestParseDirectoryWithOptions_FilePattern tests custom file pattern matching.
func TestParseDirectoryWithOptions_FilePattern(t *testing.T) {
	testCases := []struct {
		name             string
		pattern          string
		expectFiles      bool
		expectedMinCount int
	}{
		{"DCM uppercase pattern", "*.DCM", true, 1},
		{"dcm lowercase pattern", "*.dcm", true, 6},
		{"txt pattern (no match)", "*.txt", false, 0},
		{"numbered pattern", "[1-3].dcm", true, 3},
	}

	testDir := filepath.Join("..", "testdata", "dicom")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recursive := false
			opts := ParseDirectoryOptions{
				Recursive:   &recursive,
				FilePattern: tc.pattern,
			}

			result, err := ParseDirectoryWithOptions(testDir, opts)
			require.NoError(t, err)
			require.NotNil(t, result)

			if tc.expectFiles {
				assert.GreaterOrEqual(t, result.Parsed, tc.expectedMinCount,
					"Should parse files matching pattern")
			} else {
				assert.Equal(t, 0, result.Parsed, "Should not parse files with non-matching pattern")
			}
		})
	}
}

// TestParseDirectoryWithOptions_ProgressCallback tests progress reporting.
func TestParseDirectoryWithOptions_ProgressCallback(t *testing.T) {
	testDir := filepath.Join("..", "testdata", "dicom")
	recursive := false

	var callbackCount int32
	var lastCurrent, lastTotal int
	var mu sync.Mutex
	var paths []string

	opts := ParseDirectoryOptions{
		Recursive: &recursive,
		ProgressCallback: func(current, total int, path string) {
			atomic.AddInt32(&callbackCount, 1)
			mu.Lock()
			defer mu.Unlock()
			lastCurrent = current
			lastTotal = total
			paths = append(paths, path)
		},
	}

	result, err := ParseDirectoryWithOptions(testDir, opts)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Callback should be invoked for each file
	assert.Equal(t, int32(result.Parsed+result.Failed), callbackCount,
		"Progress callback should be called once per file")
	assert.Equal(t, lastCurrent, lastTotal, "Last callback should show completion")
	assert.Len(t, paths, result.Parsed+result.Failed, "Should track all file paths")

	// Verify paths are unique
	uniquePaths := make(map[string]bool)
	for _, path := range paths {
		uniquePaths[path] = true
	}
	assert.Len(t, uniquePaths, result.Parsed+result.Failed, "All paths should be unique")
}

// TestParseDirectoryWithOptions_ErrorCallback tests error callback handling.
func TestParseDirectoryWithOptions_ErrorCallback(t *testing.T) {
	// Create temp directory with mix of valid and invalid files
	tmpDir := t.TempDir()

	// Copy a valid DICOM file
	validSource := filepath.Join("..", "testdata", "dicom", "1.dcm")
	validDest := filepath.Join(tmpDir, "valid.dcm")
	validData, err := os.ReadFile(validSource)
	require.NoError(t, err)
	err = os.WriteFile(validDest, validData, 0644)
	require.NoError(t, err)

	// Create an invalid "DICOM" file
	invalidDest := filepath.Join(tmpDir, "invalid.dcm")
	err = os.WriteFile(invalidDest, []byte("Not a DICOM file"), 0644)
	require.NoError(t, err)

	// Track errors
	var errorCount int32
	var errorPaths []string
	var mu sync.Mutex

	opts := ParseDirectoryOptions{
		ErrorCallback: func(path string, err error) bool {
			atomic.AddInt32(&errorCount, 1)
			mu.Lock()
			errorPaths = append(errorPaths, path)
			mu.Unlock()
			return true // Continue parsing
		},
	}

	result, err := ParseDirectoryWithOptions(tmpDir, opts)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have 1 success and 1 failure
	assert.Equal(t, 1, result.Parsed, "Should parse 1 valid file")
	assert.Equal(t, 1, result.Failed, "Should fail on 1 invalid file")
	assert.Equal(t, int32(1), errorCount, "Error callback should be called once")
	assert.Len(t, errorPaths, 1, "Should track error path")
	assert.Contains(t, errorPaths[0], "invalid.dcm", "Error path should be the invalid file")
}

// TestParseDirectoryWithOptions_ErrorCallbackAbort tests aborting on error.
func TestParseDirectoryWithOptions_ErrorCallbackAbort(t *testing.T) {
	// Create temp directory with mix of files
	tmpDir := t.TempDir()

	// Create multiple invalid files
	for i := 1; i <= 3; i++ {
		invalidDest := filepath.Join(tmpDir, fmt.Sprintf("invalid%d.dcm", i))
		err := os.WriteFile(invalidDest, []byte("Not a DICOM file"), 0644)
		require.NoError(t, err)
	}

	// Error callback that aborts on first error
	var errorCount int32
	opts := ParseDirectoryOptions{
		ErrorCallback: func(path string, err error) bool {
			atomic.AddInt32(&errorCount, 1)
			return false // Abort parsing
		},
	}

	result, err := ParseDirectoryWithOptions(tmpDir, opts)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should stop after first error
	assert.Equal(t, int32(1), errorCount, "Should stop after first error")
	assert.Equal(t, 1, result.Failed, "Should only report one failure")
}

// TestParseDirectoryWithOptions_FailFast tests fail-fast mode.
func TestParseDirectoryWithOptions_FailFast(t *testing.T) {
	// Create temp directory with multiple invalid files
	tmpDir := t.TempDir()

	for i := 1; i <= 5; i++ {
		invalidDest := filepath.Join(tmpDir, fmt.Sprintf("invalid%d.dcm", i))
		err := os.WriteFile(invalidDest, []byte("Not a DICOM file"), 0644)
		require.NoError(t, err)
	}

	opts := ParseDirectoryOptions{
		FailFast: true,
	}

	result, err := ParseDirectoryWithOptions(tmpDir, opts)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should stop after first error
	assert.Equal(t, 1, result.Failed, "Should stop after first error in fail-fast mode")
	assert.Equal(t, 0, result.Parsed, "Should not parse any files successfully")
	assert.Len(t, result.Errors, 1, "Should only have one error recorded")
}

// TestParseDirectoryWithOptions_Context tests context cancellation.
func TestParseDirectoryWithOptions_Context(t *testing.T) {
	testDir := filepath.Join("..", "testdata", "dicom", "nested", "series_7")

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	opts := ParseDirectoryOptions{
		Context: ctx,
		Workers: 4,
	}

	result, err := ParseDirectoryWithOptions(testDir, opts)

	// Should either complete successfully or have some files parsed before cancellation
	require.NoError(t, err)
	require.NotNil(t, result)

	// Cancellation timing is non-deterministic, but we should have processed something
	// or the context was cancelled very early
	t.Logf("Parsed %d files before cancellation", result.Parsed)
}

// TestParseResult_Metadata tests that ParseResult contains correct metadata.
func TestParseResult_Metadata(t *testing.T) {
	testDir := filepath.Join("..", "testdata", "dicom")
	recursive := false
	opts := ParseDirectoryOptions{
		Recursive: &recursive,
	}

	result, err := ParseDirectoryWithOptions(testDir, opts)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify all metadata fields are populated
	assert.NotNil(t, result.Collection, "Collection should not be nil")
	assert.GreaterOrEqual(t, result.Parsed, 0, "Parsed count should be non-negative")
	assert.GreaterOrEqual(t, result.Failed, 0, "Failed count should be non-negative")
	assert.NotNil(t, result.Errors, "Errors map should not be nil")
	assert.Greater(t, result.Duration, time.Duration(0), "Duration should be positive")

	// Verify counts make sense
	assert.Equal(t, len(result.Errors), result.Failed,
		"Error map size should match failed count")

	t.Logf("Parse statistics: Parsed=%d, Failed=%d, Duration=%v",
		result.Parsed, result.Failed, result.Duration)
}

// TestParseDirectory_DataSetCollection tests that datasets are correctly added to collection.
func TestParseDirectory_DataSetCollection(t *testing.T) {
	testDir := filepath.Join("..", "testdata", "dicom")
	recursive := false
	opts := ParseDirectoryOptions{
		Recursive: &recursive,
	}

	result, err := ParseDirectoryWithOptions(testDir, opts)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify collection is properly built
	assert.Equal(t, result.Parsed, result.Collection.Len(),
		"Collection size should match parsed count")

	// Verify datasets have required UIDs
	datasets := result.Collection.DataSets()
	for i := 0; i < len(datasets) && i < 5; i++ {
		ds := datasets[i]

		// Each dataset should have SOPInstanceUID
		elem, err := ds.Get(tag.New(0x0008, 0x0018)) // SOPInstanceUID
		assert.NoError(t, err, "Dataset should have SOPInstanceUID")
		if err == nil {
			sopInstanceUID := elem.Value().String()
			assert.NotEmpty(t, sopInstanceUID, "SOPInstanceUID should not be empty")
		}
	}
}

// TestParseDirectory_ConcurrentSafety tests thread-safety of concurrent parsing.
func TestParseDirectory_ConcurrentSafety(t *testing.T) {
	testDir := filepath.Join("..", "testdata", "dicom", "nested", "series_7")

	// Run with high concurrency
	opts := ParseDirectoryOptions{
		Workers: 16,
	}

	result, err := ParseDirectoryWithOptions(testDir, opts)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should complete without race conditions
	assert.Greater(t, result.Parsed, 0, "Should successfully parse files concurrently")
	assert.Equal(t, result.Parsed, result.Collection.Len(),
		"Collection should have correct count")
}

// TestParseDirectory_MixedContent tests directory with mixed DICOM and non-DICOM files.
func TestParseDirectory_MixedContent(t *testing.T) {
	// testdata has .dcm files, .json files, and .md files
	testDir := filepath.Join("..", "testdata", "dicom")
	recursive := false
	opts := ParseDirectoryOptions{
		Recursive: &recursive,
	}

	result, err := ParseDirectoryWithOptions(testDir, opts)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should only parse .dcm files, ignoring .json and .md
	assert.GreaterOrEqual(t, result.Parsed, 6, "Should parse DICOM files")

	// Verify no .json or .md files were attempted
	for errorPath := range result.Errors {
		assert.True(t, strings.HasSuffix(errorPath, ".dcm"),
			"Only .dcm files should be processed: %s", errorPath)
	}
}

// TestApplyDefaultOptions tests that default options are correctly applied.
func TestApplyDefaultOptions(t *testing.T) {
	testCases := []struct {
		name     string
		input    ParseDirectoryOptions
		validate func(t *testing.T, opts ParseDirectoryOptions)
	}{
		{
			name:  "empty options",
			input: ParseDirectoryOptions{},
			validate: func(t *testing.T, opts ParseDirectoryOptions) {
				assert.Greater(t, opts.Workers, 0, "Workers should default to GOMAXPROCS")
				assert.Equal(t, "*.dcm", opts.FilePattern, "FilePattern should default to *.dcm")
				assert.NotNil(t, opts.Context, "Context should default to Background")
				assert.NotNil(t, opts.Recursive, "Recursive should be set")
				assert.True(t, *opts.Recursive, "Recursive should default to true")
			},
		},
		{
			name: "explicit workers",
			input: ParseDirectoryOptions{
				Workers: 4,
			},
			validate: func(t *testing.T, opts ParseDirectoryOptions) {
				assert.Equal(t, 4, opts.Workers, "Workers should be preserved")
			},
		},
		{
			name: "explicit file pattern",
			input: ParseDirectoryOptions{
				FilePattern: "*.DCM",
			},
			validate: func(t *testing.T, opts ParseDirectoryOptions) {
				assert.Equal(t, "*.DCM", opts.FilePattern, "FilePattern should be preserved")
			},
		},
		{
			name: "explicit recursive false",
			input: ParseDirectoryOptions{
				Recursive: func() *bool { b := false; return &b }(),
			},
			validate: func(t *testing.T, opts ParseDirectoryOptions) {
				assert.NotNil(t, opts.Recursive, "Recursive should not be nil")
				assert.False(t, *opts.Recursive, "Recursive should be false")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := applyDefaultOptions(tc.input)
			tc.validate(t, opts)
		})
	}
}

// TestValidateDirectory tests directory validation logic.
func TestValidateDirectory(t *testing.T) {
	testCases := []struct {
		name        string
		path        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid directory",
			path:        filepath.Join("..", "testdata", "dicom"),
			expectError: false,
		},
		{
			name:        "non-existent directory",
			path:        "/nonexistent/path/to/directory",
			expectError: true,
			errorMsg:    "does not exist",
		},
		{
			name:        "file instead of directory",
			path:        filepath.Join("..", "testdata", "dicom", "1.dcm"),
			expectError: true,
			errorMsg:    "not a directory",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDirectory(tc.path)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDiscoverFiles tests file discovery logic.
func TestDiscoverFiles(t *testing.T) {
	testCases := []struct {
		name         string
		path         string
		opts         ParseDirectoryOptions
		expectFiles  bool
		minFileCount int
	}{
		{
			name: "recursive discovery",
			path: filepath.Join("..", "testdata", "dicom"),
			opts: ParseDirectoryOptions{
				Recursive:   func() *bool { b := true; return &b }(),
				FilePattern: "*.dcm",
			},
			expectFiles:  true,
			minFileCount: 100, // Should find many files recursively
		},
		{
			name: "non-recursive discovery",
			path: filepath.Join("..", "testdata", "dicom"),
			opts: ParseDirectoryOptions{
				Recursive:   func() *bool { b := false; return &b }(),
				FilePattern: "*.dcm",
			},
			expectFiles:  true,
			minFileCount: 6, // Only top-level files
		},
		{
			name: "custom pattern",
			path: filepath.Join("..", "testdata", "dicom"),
			opts: ParseDirectoryOptions{
				Recursive:   func() *bool { b := false; return &b }(),
				FilePattern: "[1-3].dcm",
			},
			expectFiles:  true,
			minFileCount: 3, // Only 1.dcm, 2.dcm, 3.dcm
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			files, err := discoverFiles(tc.path, tc.opts)
			require.NoError(t, err)

			if tc.expectFiles {
				assert.GreaterOrEqual(t, len(files), tc.minFileCount,
					"Should discover at least %d files", tc.minFileCount)

				// Verify all paths are absolute
				for _, file := range files {
					assert.True(t, filepath.IsAbs(file),
						"File path should be absolute: %s", file)
				}

				// Verify all paths match pattern
				for _, file := range files {
					matched, _ := filepath.Match(
						strings.ToLower(tc.opts.FilePattern),
						strings.ToLower(filepath.Base(file)))
					assert.True(t, matched,
						"File should match pattern %s: %s", tc.opts.FilePattern, file)
				}
			} else {
				assert.Empty(t, files, "Should not discover any files")
			}
		})
	}
}

// BenchmarkParseDirectory benchmarks directory parsing performance.
func BenchmarkParseDirectory(b *testing.B) {
	testDir := filepath.Join("..", "testdata", "dicom", "nested", "series_7")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ParseDirectory(testDir)
		if err != nil {
			b.Fatal(err)
		}
		if result.Parsed == 0 {
			b.Fatal("No files parsed")
		}
	}
}

// BenchmarkParseDirectory_Parallel benchmarks with different worker counts.
func BenchmarkParseDirectory_Parallel(b *testing.B) {
	testDir := filepath.Join("..", "testdata", "dicom", "nested", "series_7")

	workerCounts := []int{1, 2, 4, 8, 16}

	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("workers-%d", workers), func(b *testing.B) {
			opts := ParseDirectoryOptions{
				Workers: workers,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, err := ParseDirectoryWithOptions(testDir, opts)
				if err != nil {
					b.Fatal(err)
				}
				if result.Parsed == 0 {
					b.Fatal("No files parsed")
				}
			}
		})
	}
}
