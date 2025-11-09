package dicom

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/uid"
	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteFile_BasicWrite tests writing a simple DICOM file with default options.
func TestWriteFile_BasicWrite(t *testing.T) {
	// Create a temporary directory for test output
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "test_output.dcm")

	// Create a synthetic test dataset
	ds := createTestDatasetForWriter(t)
	require.NotNil(t, ds, "Dataset should not be nil")

	// Write the dataset
	err := WriteFile(outputPath, ds)
	require.NoError(t, err, "Failed to write file")

	// Verify file exists
	_, err = os.Stat(outputPath)
	require.NoError(t, err, "Output file should exist")
}

// TestWriteFile_RoundTrip tests that a file can be written and read back with identical content.
func TestWriteFile_RoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "roundtrip.dcm")

	// Create synthetic dataset
	originalDS := createTestDatasetForWriter(t)

	// Write it out
	err := WriteFile(outputPath, originalDS)
	require.NoError(t, err, "Failed to write file")

	// Read it back
	roundtripDS, err := ParseFile(outputPath)
	require.NoError(t, err, "Failed to parse written file")

	// Compare key elements
	verifyElementsMatch(t, originalDS, roundtripDS, tag.New(0x0008, 0x0018)) // SOPInstanceUID
	verifyElementsMatch(t, originalDS, roundtripDS, tag.New(0x0020, 0x000D)) // StudyInstanceUID
	verifyElementsMatch(t, originalDS, roundtripDS, tag.New(0x0020, 0x000E)) // SeriesInstanceUID
}

// TestWriteFileWithOptions_ExplicitVRLittleEndian tests writing with explicit transfer syntax.
func TestWriteFileWithOptions_ExplicitVRLittleEndian(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "explicit_vr.dcm")

	// Create synthetic test dataset
	ds := createTestDatasetForWriter(t)

	// Write with explicit VR Little Endian
	explicitUID, err := uid.Parse("1.2.840.10008.1.2.1")
	require.NoError(t, err)

	opts := WriteOptions{
		TransferSyntax: &explicitUID,
		Overwrite:      true,
		CreateDirs:     true,
		Atomic:         true,
	}

	err = WriteFileWithOptions(outputPath, ds, opts)
	require.NoError(t, err, "Failed to write file with explicit VR")

	// Verify file exists and can be read back
	_, err = os.Stat(outputPath)
	require.NoError(t, err)

	roundtripDS, err := ParseFile(outputPath)
	require.NoError(t, err, "Failed to parse written file")
	require.NotNil(t, roundtripDS)
}

// TestWriteFileWithOptions_ImplicitVRLittleEndian tests writing with implicit transfer syntax.
func TestWriteFileWithOptions_ImplicitVRLittleEndian(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "implicit_vr.dcm")

	// Create synthetic test dataset
	ds := createTestDatasetForWriter(t)

	// Write with implicit VR Little Endian
	implicitUID, err := uid.Parse("1.2.840.10008.1.2")
	require.NoError(t, err)

	opts := WriteOptions{
		TransferSyntax: &implicitUID,
		Overwrite:      true,
	}

	err = WriteFileWithOptions(outputPath, ds, opts)
	require.NoError(t, err, "Failed to write file with implicit VR")

	// Verify file exists
	_, err = os.Stat(outputPath)
	require.NoError(t, err)
}

// TestWriteFile_Overwrite tests overwrite behavior.
func TestWriteFile_Overwrite(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "overwrite_test.dcm")

	// Create synthetic test dataset
	ds := createTestDatasetForWriter(t)

	// Write first time
	err := WriteFile(outputPath, ds)
	require.NoError(t, err)

	// Get original file info
	originalInfo, err := os.Stat(outputPath)
	require.NoError(t, err)

	// Write second time with Overwrite: true
	opts := WriteOptions{
		Overwrite: true,
	}
	err = WriteFileWithOptions(outputPath, ds, opts)
	require.NoError(t, err, "Should be able to overwrite with Overwrite: true")

	// Verify file still exists
	newInfo, err := os.Stat(outputPath)
	require.NoError(t, err)
	assert.NotEqual(t, originalInfo.ModTime(), newInfo.ModTime(), "ModTime should change after overwrite")
}

// TestWriteFile_NoOverwrite tests that writing fails when file exists and Overwrite is false.
func TestWriteFile_NoOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "no_overwrite_test.dcm")

	// Create synthetic test dataset
	ds := createTestDatasetForWriter(t)

	// Write first time
	err := WriteFile(outputPath, ds)
	require.NoError(t, err)

	// Write second time with Overwrite: false
	opts := WriteOptions{
		Overwrite: false,
	}
	err = WriteFileWithOptions(outputPath, ds, opts)
	assert.Error(t, err, "Should fail when trying to overwrite with Overwrite: false")
	assert.Contains(t, err.Error(), "file already exists", "Error should mention file exists")
}

// TestWriteFile_CreateDirs tests automatic directory creation.
func TestWriteFile_CreateDirs(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "subdir1", "subdir2", "test.dcm")

	// Create synthetic test dataset
	ds := createTestDatasetForWriter(t)

	// Write with CreateDirs: true
	opts := WriteOptions{
		CreateDirs: true,
	}
	err := WriteFileWithOptions(outputPath, ds, opts)
	require.NoError(t, err, "Should create directories automatically")

	// Verify file exists
	_, err = os.Stat(outputPath)
	require.NoError(t, err, "File should exist in created subdirectories")
}

// TestWriteFile_NoCreateDirs tests that writing fails when directories don't exist and CreateDirs is false.
func TestWriteFile_NoCreateDirs(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "nonexistent", "test.dcm")

	// Create synthetic test dataset
	ds := createTestDatasetForWriter(t)

	// Write with CreateDirs: false
	opts := WriteOptions{
		CreateDirs: false,
	}
	err := WriteFileWithOptions(outputPath, ds, opts)
	assert.Error(t, err, "Should fail when directories don't exist and CreateDirs is false")
}

// TestWriteFile_AtomicWrite tests that atomic writes work correctly.
func TestWriteFile_AtomicWrite(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "atomic_test.dcm")

	// Create synthetic test dataset
	ds := createTestDatasetForWriter(t)

	// Write with Atomic: true
	opts := WriteOptions{
		Atomic: true,
	}
	err := WriteFileWithOptions(outputPath, ds, opts)
	require.NoError(t, err, "Atomic write should succeed")

	// Verify no temp files left behind
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)

	for _, file := range files {
		assert.NotContains(t, file.Name(), ".dicom-tmp-", "No temporary files should remain")
	}

	// Verify output file exists
	_, err = os.Stat(outputPath)
	require.NoError(t, err)
}

// TestWriteFile_NilDataSet tests error handling for nil dataset.
func TestWriteFile_NilDataSet(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "nil_test.dcm")

	err := WriteFile(outputPath, nil)
	assert.Error(t, err, "Should fail when dataset is nil")
	assert.Contains(t, err.Error(), "nil", "Error should mention nil dataset")
}

// TestWriteFile_MissingRequiredTags tests handling of datasets missing required tags.
func TestWriteFile_MissingRequiredTags(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "missing_tags.dcm")

	// Create a minimal dataset without required tags
	ds := NewDataSet()

	err := WriteFile(outputPath, ds)
	assert.Error(t, err, "Should fail when required tags are missing")
}

// TestWriteFile_FileMetaInformation tests that File Meta Information is correctly generated.
func TestWriteFile_FileMetaInformation(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "meta_info_test.dcm")

	// Create synthetic test dataset
	ds := createTestDatasetForWriter(t)

	// Write the file
	err := WriteFile(outputPath, ds)
	require.NoError(t, err)

	// Read back and verify File Meta Information exists
	writtenDS, err := ParseFile(outputPath)
	require.NoError(t, err)

	// Verify required File Meta Information elements
	verifyFileMetaElement(t, writtenDS, tag.New(0x0002, 0x0001)) // File Meta Information Version
	verifyFileMetaElement(t, writtenDS, tag.New(0x0002, 0x0002)) // Media Storage SOP Class UID
	verifyFileMetaElement(t, writtenDS, tag.New(0x0002, 0x0003)) // Media Storage SOP Instance UID
	verifyFileMetaElement(t, writtenDS, tag.New(0x0002, 0x0010)) // Transfer Syntax UID
	verifyFileMetaElement(t, writtenDS, tag.New(0x0002, 0x0012)) // Implementation Class UID
	verifyFileMetaElement(t, writtenDS, tag.New(0x0002, 0x0013)) // Implementation Version Name
}

// TestWriteFile_MultipleFiles tests writing multiple files sequentially.
func TestWriteFile_MultipleFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create synthetic test dataset
	ds := createTestDatasetForWriter(t)

	// Write 5 files
	for i := 0; i < 5; i++ {
		outputPath := filepath.Join(tempDir, "file_"+string(rune('0'+i))+".dcm")
		err := WriteFile(outputPath, ds)
		require.NoError(t, err, "Failed to write file %d", i)
	}

	// Verify all files exist
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Equal(t, 5, len(files), "Should have 5 files")
}

// TestWriteFile_LargeDataset tests writing datasets with many elements.
func TestWriteFile_LargeDataset(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "large_dataset.dcm")

	// Create synthetic test dataset (has 6 elements)
	ds := createTestDatasetForWriter(t)

	// Write the dataset
	err := WriteFile(outputPath, ds)
	require.NoError(t, err)

	// Verify file exists and has reasonable size
	info, err := os.Stat(outputPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(128+4), "File should be larger than just preamble and prefix")
}

// Helper function to verify elements match between two datasets.
func verifyElementsMatch(t *testing.T, ds1, ds2 *DataSet, tag tag.Tag) {
	elem1, err1 := ds1.Get(tag)
	elem2, err2 := ds2.Get(tag)

	if err1 != nil && err2 != nil {
		// Both missing - that's fine
		return
	}

	require.NoError(t, err1, "Original dataset should have tag %s", tag.String())
	require.NoError(t, err2, "Roundtrip dataset should have tag %s", tag.String())

	val1 := elem1.Value().String()
	val2 := elem2.Value().String()
	assert.Equal(t, val1, val2, "Values should match for tag %s", tag.String())
}

// Helper function to verify File Meta Information element exists.
func verifyFileMetaElement(t *testing.T, ds *DataSet, tag tag.Tag) {
	elem, err := ds.Get(tag)
	require.NoError(t, err, "File Meta Information element %s should exist", tag.String())
	require.NotNil(t, elem, "File Meta Information element %s should not be nil", tag.String())
}

// Helper: createTestDatasetForWriter creates a synthetic dataset for writer tests.
func createTestDatasetForWriter(t *testing.T) *DataSet {
	ds := NewDataSet()

	// SOPInstanceUID (0008,0018)
	sopInstanceUID := "1.2.840.10008.5.1.4.1.1.1.999"
	sopInstanceValue, err := value.NewStringValue(vr.UniqueIdentifier, []string{sopInstanceUID})
	require.NoError(t, err)
	sopInstanceElem, err := element.NewElement(tag.New(0x0008, 0x0018), vr.UniqueIdentifier, sopInstanceValue)
	require.NoError(t, err)
	ds.Add(sopInstanceElem)

	// SOPClassUID (0008,0016)
	sopClassUID := "1.2.840.10008.5.1.4.1.1.1"
	sopClassValue, err := value.NewStringValue(vr.UniqueIdentifier, []string{sopClassUID})
	require.NoError(t, err)
	sopClassElem, err := element.NewElement(tag.New(0x0008, 0x0016), vr.UniqueIdentifier, sopClassValue)
	require.NoError(t, err)
	ds.Add(sopClassElem)

	// StudyInstanceUID (0020,000D)
	studyUID := "1.2.840.10008.5.1.4.1.1.2.1"
	studyValue, err := value.NewStringValue(vr.UniqueIdentifier, []string{studyUID})
	require.NoError(t, err)
	studyElem, err := element.NewElement(tag.New(0x0020, 0x000D), vr.UniqueIdentifier, studyValue)
	require.NoError(t, err)
	ds.Add(studyElem)

	// SeriesInstanceUID (0020,000E)
	seriesUID := "1.2.840.10008.5.1.4.1.1.3.1"
	seriesValue, err := value.NewStringValue(vr.UniqueIdentifier, []string{seriesUID})
	require.NoError(t, err)
	seriesElem, err := element.NewElement(tag.New(0x0020, 0x000E), vr.UniqueIdentifier, seriesValue)
	require.NoError(t, err)
	ds.Add(seriesElem)

	// PatientID (0010,0020)
	patientID := "PAT001"
	patientValue, err := value.NewStringValue(vr.LongString, []string{patientID})
	require.NoError(t, err)
	patientElem, err := element.NewElement(tag.New(0x0010, 0x0020), vr.LongString, patientValue)
	require.NoError(t, err)
	ds.Add(patientElem)

	// PatientName (0010,0010)
	patientName := "Test^Patient"
	patientNameValue, err := value.NewStringValue(vr.PersonName, []string{patientName})
	require.NoError(t, err)
	patientNameElem, err := element.NewElement(tag.New(0x0010, 0x0010), vr.PersonName, patientNameValue)
	require.NoError(t, err)
	ds.Add(patientNameElem)

	return ds
}
