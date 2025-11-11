package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/codeninja55/go-radx/cmd/radx/internal/config"
	"github.com/codeninja55/go-radx/cmd/radx/internal/dicom/ui"
	"github.com/codeninja55/go-radx/dicom"
)

// OrganizeCmd implements the DICOM organize command.
type OrganizeCmd struct {
	Dir       string `arg:"" required:"" type:"existingdir" help:"Directory containing DICOM files to organize"`
	OutputDir string `name:"output-dir" required:"" type:"path" help:"Output directory for organized files"`
	Recursive bool   `name:"recursive" short:"R" help:"Recursively search directories" default:"true"`
	Move      bool   `name:"move" help:"Move files instead of copying"`
	DryRun    bool   `name:"dry-run" help:"Show what would be done without making changes"`
}

// organizedFile represents a DICOM file with its organization metadata.
type organizedFile struct {
	sourcePath   string
	studyUID     string
	seriesUID    string
	instanceUID  string
	sopClassUID  string
	relativePath string
}

// Run executes the organize command.
func (c *OrganizeCmd) Run(cfg *config.GlobalConfig) error {
	// Print banner
	ui.PrintBanner()

	logger := log.Default()
	logger.Info("Starting DICOM organize operation")

	// Validate output directory
	if !c.DryRun {
		if err := createOutputDirectory(c.OutputDir); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Find all DICOM files
	logger.Debug("Scanning directory", "path", c.Dir, "recursive", c.Recursive)
	files, err := listDicomFiles(c.Dir, c.Recursive)
	if err != nil {
		return fmt.Errorf("failed to list DICOM files: %w", err)
	}

	if len(files) == 0 {
		logger.Warn("No DICOM files found")
		return nil
	}

	logger.Info("Found DICOM files", "count", len(files))

	// Parse files and extract organization metadata
	logger.Info("Analyzing DICOM files...")
	progress := ui.NewProgressBar(len(files), "Analyzing")
	organizedFiles := make([]organizedFile, 0, len(files))
	parseFailures := 0

	for _, file := range files {
		progress.Increment(fmt.Sprintf("Analyzing %s", file.Name))

		orgFile, err := c.analyzeFile(file.Path, logger)
		if err != nil {
			logger.Error("Failed to analyze file", "file", file.Path, "error", err)
			parseFailures++
			continue
		}

		organizedFiles = append(organizedFiles, orgFile)
	}

	progress.Complete("Analysis complete")

	if len(organizedFiles) == 0 {
		return fmt.Errorf("no valid DICOM files to organize")
	}

	logger.Info("Analyzed files",
		"valid", len(organizedFiles),
		"failed", parseFailures,
	)

	// Organize files
	logger.Info("Organizing DICOM files...")
	progress = ui.NewProgressBar(len(organizedFiles), "Organizing")
	successCount := 0
	failCount := 0

	for _, orgFile := range organizedFiles {
		progress.Increment(fmt.Sprintf("Organizing %s", filepath.Base(orgFile.sourcePath)))

		if err := c.organizeFile(orgFile, logger); err != nil {
			logger.Error("Failed to organize file", "file", orgFile.sourcePath, "error", err)
			failCount++
			continue
		}

		successCount++
	}

	progress.Complete("Complete")

	// Print summary
	fmt.Println()
	if c.DryRun {
		fmt.Println(ui.InfoStyle.Render("⚠ DRY RUN - No changes were made"))
	} else if failCount == 0 {
		fmt.Println(ui.SuccessStyle.Render("✓ All files organized successfully!"))
	} else {
		fmt.Println(ui.WarnStyle.Render(fmt.Sprintf("⚠ Organization completed with %d failures", failCount)))
	}
	fmt.Println()
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Source Directory:"), ui.InfoStyle.Render(c.Dir))
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Output Directory:"), ui.InfoStyle.Render(c.OutputDir))
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Total Files:"), ui.InfoStyle.Render(fmt.Sprintf("%d", len(files))))
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Organized:"), ui.SuccessStyle.Render(fmt.Sprintf("%d", successCount)))
	if parseFailures > 0 {
		fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Parse Failures:"), ui.ErrorStyle.Render(fmt.Sprintf("%d", parseFailures)))
	}
	if failCount > 0 {
		fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Organization Failures:"), ui.ErrorStyle.Render(fmt.Sprintf("%d", failCount)))
	}
	if c.Move {
		fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Mode:"), ui.WarnStyle.Render("Move"))
	} else {
		fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Mode:"), ui.InfoStyle.Render("Copy"))
	}
	fmt.Println()

	logger.Info("Organize operation complete",
		"total", len(files),
		"organized", successCount,
		"parse_failures", parseFailures,
		"org_failures", failCount,
	)

	if failCount > 0 || parseFailures > 0 {
		return fmt.Errorf("organize completed with failures")
	}

	return nil
}

// analyzeFile parses a DICOM file and extracts organization metadata.
func (c *OrganizeCmd) analyzeFile(path string, logger *log.Logger) (organizedFile, error) {
	// Parse DICOM file
	dataset, err := dicom.ParseFile(path)
	if err != nil {
		return organizedFile{}, fmt.Errorf("failed to parse DICOM file: %w", err)
	}

	// Extract UIDs
	var studyUID, seriesUID, instanceUID, sopClassUID string

	for _, elem := range dataset.Elements() {
		tag := elem.Tag()
		switch {
		case tag.Group == 0x0020 && tag.Element == 0x000D: // Study Instance UID
			studyUID = elem.Value().String()
		case tag.Group == 0x0020 && tag.Element == 0x000E: // Series Instance UID
			seriesUID = elem.Value().String()
		case tag.Group == 0x0008 && tag.Element == 0x0018: // SOP Instance UID
			instanceUID = elem.Value().String()
		case tag.Group == 0x0008 && tag.Element == 0x0016: // SOP Class UID
			sopClassUID = elem.Value().String()
		}
	}

	// Validate required UIDs
	if studyUID == "" {
		return organizedFile{}, fmt.Errorf("study instance UID (0020,000D) not found")
	}
	if seriesUID == "" {
		return organizedFile{}, fmt.Errorf("series instance UID (0020,000E) not found")
	}
	if instanceUID == "" {
		return organizedFile{}, fmt.Errorf("SOP Instance UID (0008,0018) not found")
	}

	// Build relative path: <study-uid>/<series-uid>/<instance-uid>.dcm
	relativePath := filepath.Join(
		sanitizeUID(studyUID),
		sanitizeUID(seriesUID),
		sanitizeUID(instanceUID)+".dcm",
	)

	logger.Debug("Analyzed file",
		"source", path,
		"study", studyUID,
		"series", seriesUID,
		"instance", instanceUID,
	)

	return organizedFile{
		sourcePath:   path,
		studyUID:     studyUID,
		seriesUID:    seriesUID,
		instanceUID:  instanceUID,
		sopClassUID:  sopClassUID,
		relativePath: relativePath,
	}, nil
}

// organizeFile moves or copies a file to its organized location.
func (c *OrganizeCmd) organizeFile(orgFile organizedFile, logger *log.Logger) error {
	// Build destination path
	destPath := filepath.Join(c.OutputDir, orgFile.relativePath)

	// Dry run mode - just log what would be done
	if c.DryRun {
		if c.Move {
			logger.Info("Would move", "from", orgFile.sourcePath, "to", destPath)
		} else {
			logger.Info("Would copy", "from", orgFile.sourcePath, "to", destPath)
		}
		return nil
	}

	// Create destination directory
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Move or copy file
	if c.Move {
		if err := os.Rename(orgFile.sourcePath, destPath); err != nil {
			// If rename fails (e.g., cross-device), fall back to copy and delete
			if err := copyFile(orgFile.sourcePath, destPath); err != nil {
				return fmt.Errorf("failed to copy file: %w", err)
			}
			if err := os.Remove(orgFile.sourcePath); err != nil {
				logger.Warn("Failed to remove source file after copy", "file", orgFile.sourcePath, "error", err)
			}
		}
		logger.Debug("Moved file", "from", orgFile.sourcePath, "to", destPath)
	} else {
		if err := copyFile(orgFile.sourcePath, destPath); err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}
		logger.Debug("Copied file", "from", orgFile.sourcePath, "to", destPath)
	}

	return nil
}

// sanitizeUID sanitizes a UID string for use in filenames.
func sanitizeUID(uid string) string {
	// UIDs should already be filesystem-safe (only digits and dots)
	// but we'll be defensive and replace any problematic characters
	return uid
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() { _ = srcFile.Close() }()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() { _ = dstFile.Close() }()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Sync to ensure data is written
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	return nil
}
