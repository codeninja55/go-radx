package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/uid"
	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
)

// generateNestedDICOM creates a nested directory structure with synthetic DICOM files
// similar to the CTC_2 structure but without any PHI.
//
// Structure created:
// testdata/dicom/nested/
//
//	├── series_1/ (2 files)
//	├── series_2/ (58 files)
//	├── series_3/ (56 files)
//	├── series_4/ (184 files)
//	├── series_5/ (69 files)
//	├── series_6/ (69 files)
//	├── series_7/ (688 files) - main test target
//	└── series_8/ (69 files)
//
// Total: ~1195 files across 8 series directories
func main() {
	baseDir := filepath.Join("dicom", "nested")

	// Define series structure: series name -> number of files
	seriesStructure := map[string]int{
		"series_1": 2,
		"series_2": 58,
		"series_3": 56,
		"series_4": 184,
		"series_5": 69,
		"series_6": 69,
		"series_7": 688, // Main test target - needs >100 files
		"series_8": 69,
	}

	fmt.Println("Generating synthetic nested DICOM test data...")

	for seriesName, numFiles := range seriesStructure {
		seriesDir := filepath.Join(baseDir, seriesName)

		// Create series directory
		if err := os.MkdirAll(seriesDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", seriesDir, err)
			os.Exit(1)
		}

		fmt.Printf("Creating %d files in %s...\n", numFiles, seriesName)

		// Generate DICOM files for this series
		for i := 1; i <= numFiles; i++ {
			if err := generateSyntheticDICOM(seriesDir, seriesName, i); err != nil {
				fmt.Fprintf(os.Stderr, "Error generating file %d in %s: %v\n", i, seriesName, err)
				os.Exit(1)
			}
		}

		fmt.Printf("  ✓ Created %d files in %s\n", numFiles, seriesName)
	}

	fmt.Printf("\n✓ Successfully generated synthetic DICOM test data in %s\n", baseDir)
	fmt.Println("  Total files created: ~1195 across 8 series directories")
}

// generateSyntheticDICOM creates a minimal synthetic DICOM file without any PHI
func generateSyntheticDICOM(seriesDir, seriesName string, instanceNum int) error {
	// Create synthetic dataset with minimal required elements
	ds := dicom.NewDataSet()

	// Generate synthetic UIDs using the UID generator (no PHI)
	studyUID := uid.Generate()       // One study UID for all series
	seriesUID := uid.Generate()      // One series UID per series
	sopInstanceUID := uid.Generate() // Unique instance UID per file

	// Set synthetic patient information (clearly marked as test data)
	_ = ds.SetPatientName(fmt.Sprintf("TEST^SYNTHETIC^DATA^%d", instanceNum))
	_ = ds.SetPatientID(fmt.Sprintf("SYNTHETIC_%s_%04d", seriesName, instanceNum))
	_ = ds.SetPatientBirthDate("20000101")
	_ = ds.SetPatientSex("O") // Other - clearly synthetic

	// Set SOP Class UID (required for DICOM files) - CT Image Storage
	sopClassVal, _ := value.NewStringValue(vr.UniqueIdentifier, []string{"1.2.840.10008.5.1.4.1.1.2"})
	sopClassElem, _ := element.NewElement(tag.SOPClassUID, vr.UniqueIdentifier, sopClassVal)
	_ = ds.Add(sopClassElem)

	// Set study/series/instance UIDs
	_ = ds.SetStudyInstanceUID(studyUID)
	_ = ds.SetSeriesInstanceUID(seriesUID)
	_ = ds.SetSOPInstanceUID(sopInstanceUID)

	// Set series and instance numbers
	seriesNum := 1
	switch seriesName {
	case "series_1":
		seriesNum = 1
	case "series_2":
		seriesNum = 2
	case "series_3":
		seriesNum = 3
	case "series_4":
		seriesNum = 4
	case "series_5":
		seriesNum = 5
	case "series_6":
		seriesNum = 6
	case "series_7":
		seriesNum = 7
	case "series_8":
		seriesNum = 8
	}

	_ = ds.SetSeriesNumber(seriesNum)
	_ = ds.SetInstanceNumber(instanceNum)

	// Set study/series descriptions (clearly synthetic)
	_ = ds.SetStudyDate("20240101")

	// Write DICOM file
	filename := filepath.Join(seriesDir, fmt.Sprintf("%s.%d.dcm", seriesName, instanceNum))
	if err := dicom.WriteFile(filename, ds); err != nil {
		return fmt.Errorf("failed to write DICOM file: %w", err)
	}

	return nil
}
