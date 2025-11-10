package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/codeninja55/go-radx/cmd/radx/internal/config"
	"github.com/codeninja55/go-radx/cmd/radx/internal/dicom/ui"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/tag"
)

// ModifyCmd implements the DICOM modify command.
type ModifyCmd struct {
	Paths     []string `arg:"" optional:"" type:"existingfile" help:"DICOM files to modify" group:"Input"`
	Dir       string   `name:"dir" type:"existingdir" help:"Directory containing DICOM files" group:"Input" xor:"Input"`
	OutputDir string   `name:"output-dir" required:"" type:"path" help:"Output directory for modified files"`

	Recursive bool `name:"recursive" short:"R" help:"Recursively search directories"`
	InPlace   bool `name:"in-place" short:"i" help:"Modify files in-place (overwrite original)"`

	// Tag operations
	Insert []string `name:"insert" short:"I" help:"Insert or update tag (format: (GGGG,EEEE)=value)"`
	Delete []string `name:"delete" short:"D" help:"Delete tag (format: (GGGG,EEEE))"`

	// UID regeneration
	RegenerateStudyUID    bool `name:"regenerate-study-uid" help:"Generate new Study Instance UID"`
	RegenerateSeriesUID   bool `name:"regenerate-series-uid" help:"Generate new Series Instance UID"`
	RegenerateInstanceUID bool `name:"regenerate-instance-uid" help:"Generate new SOP Instance UID"`
	RegenerateAll         bool `name:"regenerate-all-uids" help:"Generate all new UIDs (Study, Series, Instance)"`
}

// tagModification represents a tag modification operation.
type tagModification struct {
	tag    tag.Tag
	value  string
	delete bool
}

// Run executes the modify command.
func (c *ModifyCmd) Run(cfg *config.GlobalConfig) error {
	// Print banner
	ui.PrintBanner()

	logger := log.Default()
	logger.Info("Starting DICOM modify operation")

	// Validate that at least one modification is specified
	if len(c.Insert) == 0 && len(c.Delete) == 0 &&
		!c.RegenerateStudyUID && !c.RegenerateSeriesUID &&
		!c.RegenerateInstanceUID && !c.RegenerateAll {
		return fmt.Errorf("no modifications specified (use --insert, --delete, or --regenerate-* flags)")
	}

	// Validate output directory unless in-place mode
	if !c.InPlace {
		if err := createOutputDirectory(c.OutputDir); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Parse tag modifications
	modifications, err := c.parseModifications()
	if err != nil {
		return fmt.Errorf("failed to parse modifications: %w", err)
	}

	// Collect DICOM files
	var files []DICOMFile

	if c.Dir != "" {
		logger.Debug("Scanning directory", "path", c.Dir, "recursive", c.Recursive)
		files, err = listDicomFiles(c.Dir, c.Recursive)
		if err != nil {
			return fmt.Errorf("failed to list DICOM files: %w", err)
		}
	} else if len(c.Paths) > 0 {
		logger.Debug("Processing files", "count", len(c.Paths))
		for _, path := range c.Paths {
			info, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("failed to stat file %s: %w", path, err)
			}
			files = append(files, DICOMFile{
				Path: path,
				Name: filepath.Base(path),
				Size: info.Size(),
			})
		}
	} else {
		return fmt.Errorf("no input files specified (use paths or --dir)")
	}

	if len(files) == 0 {
		logger.Warn("No DICOM files found")
		return nil
	}

	logger.Info("Found DICOM files", "count", len(files))
	logger.Debug("Modifications to apply",
		"inserts", len(c.Insert),
		"deletes", len(c.Delete),
		"regen_study", c.RegenerateStudyUID || c.RegenerateAll,
		"regen_series", c.RegenerateSeriesUID || c.RegenerateAll,
		"regen_instance", c.RegenerateInstanceUID || c.RegenerateAll,
	)

	// Process each file
	progress := ui.NewProgressBar(len(files), "Modifying")
	successCount := 0
	failCount := 0

	for _, file := range files {
		progress.Increment(fmt.Sprintf("Modifying %s", file.Name))

		// Parse DICOM file
		dataset, err := dicom.ParseFile(file.Path)
		if err != nil {
			logger.Error("Failed to parse DICOM file", "file", file.Path, "error", err)
			failCount++
			continue
		}

		// Apply modifications
		if err := c.applyModifications(dataset, modifications, logger); err != nil {
			logger.Error("Failed to apply modifications", "file", file.Path, "error", err)
			failCount++
			continue
		}

		// Regenerate UIDs if requested
		if err := c.regenerateUIDs(dataset, logger); err != nil {
			logger.Error("Failed to regenerate UIDs", "file", file.Path, "error", err)
			failCount++
			continue
		}

		// Determine output path
		outputPath := c.getOutputPath(file.Path)

		// Write modified file
		if err := dicom.WriteFile(outputPath, dataset); err != nil {
			logger.Error("Failed to write modified file", "file", outputPath, "error", err)
			failCount++
			continue
		}

		successCount++
		logger.Debug("Modified file", "input", file.Path, "output", outputPath)
	}

	progress.Complete("Complete")

	// Print summary
	fmt.Println()
	if failCount == 0 {
		fmt.Println(ui.SuccessStyle.Render("✓ All files modified successfully!"))
	} else {
		fmt.Println(ui.WarnStyle.Render(fmt.Sprintf("⚠ Modification completed with %d failures", failCount)))
	}
	fmt.Println()
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Total Files:"), ui.InfoStyle.Render(fmt.Sprintf("%d", len(files))))
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Successful:"), ui.SuccessStyle.Render(fmt.Sprintf("%d", successCount)))
	if failCount > 0 {
		fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Failed:"), ui.ErrorStyle.Render(fmt.Sprintf("%d", failCount)))
	}
	if c.InPlace {
		fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Mode:"), ui.WarnStyle.Render("In-place (original files overwritten)"))
	} else {
		fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Output Directory:"), ui.InfoStyle.Render(c.OutputDir))
	}
	fmt.Println()

	logger.Info("Modify operation complete",
		"total", len(files),
		"success", successCount,
		"failed", failCount,
	)

	if failCount > 0 {
		return fmt.Errorf("modify completed with %d failures", failCount)
	}

	return nil
}

// parseModifications parses insert and delete flags into tag modifications.
func (c *ModifyCmd) parseModifications() ([]tagModification, error) {
	modifications := make([]tagModification, 0)

	// Parse insert operations
	for _, insert := range c.Insert {
		parts := strings.SplitN(insert, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid insert format: %s (expected (GGGG,EEEE)=value)", insert)
		}

		tag, err := parseTagString(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid tag in insert: %s: %w", parts[0], err)
		}

		modifications = append(modifications, tagModification{
			tag:    tag,
			value:  parts[1],
			delete: false,
		})
	}

	// Parse delete operations
	for _, delete := range c.Delete {
		tag, err := parseTagString(delete)
		if err != nil {
			return nil, fmt.Errorf("invalid tag in delete: %s: %w", delete, err)
		}

		modifications = append(modifications, tagModification{
			tag:    tag,
			delete: true,
		})
	}

	return modifications, nil
}

// parseTagString parses a tag string like "(0010,0010)" into a tag.Tag.
func parseTagString(s string) (tag.Tag, error) {
	// Remove parentheses and spaces
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "()")

	// Split by comma
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return tag.Tag{}, fmt.Errorf("invalid tag format: %s (expected (GGGG,EEEE))", s)
	}

	// Parse group
	group, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 16, 16)
	if err != nil {
		return tag.Tag{}, fmt.Errorf("invalid group: %s: %w", parts[0], err)
	}

	// Parse element
	element, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 16, 16)
	if err != nil {
		return tag.Tag{}, fmt.Errorf("invalid element: %s: %w", parts[1], err)
	}

	return tag.Tag{Group: uint16(group), Element: uint16(element)}, nil
}

// applyModifications applies tag modifications to a dataset.
func (c *ModifyCmd) applyModifications(dataset *dicom.DataSet, modifications []tagModification, logger *log.Logger) error {
	// TODO: Implement tag insertion/update and deletion
	// This requires methods on DataSet to manipulate elements
	// For now, log the operations that would be performed

	for _, mod := range modifications {
		if mod.delete {
			logger.Debug("Would delete tag", "tag", fmt.Sprintf("(%04X,%04X)", mod.tag.Group, mod.tag.Element))
			// dataset.DeleteElement(mod.tag)
		} else {
			logger.Debug("Would insert/update tag",
				"tag", fmt.Sprintf("(%04X,%04X)", mod.tag.Group, mod.tag.Element),
				"value", mod.value,
			)
			// dataset.SetElement(mod.tag, mod.value)
		}
	}

	// TODO: Remove this once actual implementation is complete
	if len(modifications) > 0 {
		logger.Warn("Tag modification not yet fully implemented - file will be copied without changes")
	}

	return nil
}

// regenerateUIDs regenerates UIDs in the dataset.
func (c *ModifyCmd) regenerateUIDs(dataset *dicom.DataSet, logger *log.Logger) error {
	if c.RegenerateAll || c.RegenerateStudyUID {
		newUID := generateUID()
		logger.Debug("Generated new Study Instance UID", "uid", newUID)
		// TODO: Update tag (0020,000D) with newUID
	}

	if c.RegenerateAll || c.RegenerateSeriesUID {
		newUID := generateUID()
		logger.Debug("Generated new Series Instance UID", "uid", newUID)
		// TODO: Update tag (0020,000E) with newUID
	}

	if c.RegenerateAll || c.RegenerateInstanceUID {
		newUID := generateUID()
		logger.Debug("Generated new SOP Instance UID", "uid", newUID)
		// TODO: Update tag (0008,0018) with newUID
	}

	// TODO: Remove this once actual implementation is complete
	if c.RegenerateAll || c.RegenerateStudyUID || c.RegenerateSeriesUID || c.RegenerateInstanceUID {
		logger.Warn("UID regeneration not yet fully implemented - original UIDs will be preserved")
	}

	return nil
}

// getOutputPath determines the output path for a modified file.
func (c *ModifyCmd) getOutputPath(inputPath string) string {
	if c.InPlace {
		return inputPath
	}

	// Use output directory
	filename := filepath.Base(inputPath)
	return filepath.Join(c.OutputDir, filename)
}
