package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/codeninja55/go-radx/cmd/radx/internal/config"
	"github.com/codeninja55/go-radx/cmd/radx/internal/dicom/ui"
	"github.com/codeninja55/go-radx/dicom"
)

// DumpCmd implements the DICOM dump command.
type DumpCmd struct {
	Paths            []string `arg:"" optional:"" type:"existingfile" help:"DICOM files to dump" group:"Input"`
	Dir              string   `name:"dir" type:"existingdir" help:"Directory containing DICOM files" group:"Input" xor:"Input"`
	Recursive        bool     `name:"recursive" short:"R" help:"Recursively search directories"`
	ProcessPixelData bool     `name:"process-pixel-data" help:"Process pixel data elements"`
	StorePixelData   bool     `name:"store-pixel-data" help:"Extract and store pixel data to files"`
}

// Run executes the dump command.
func (c *DumpCmd) Run(cfg *config.GlobalConfig) error {
	// Print banner
	ui.PrintBanner()

	logger := log.Default()
	logger.Info("Starting DICOM dump")

	// Collect DICOM files
	var files []DICOMFile
	var err error

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

	// Process each file
	allTags := make([]DICOMTag, 0)
	progress := ui.NewProgressBar(len(files), "Processing")

	for i, file := range files {
		progress.Increment(fmt.Sprintf("Parsing %s", file.Name))

		// Validate file
		if err := validateDicomFile(file.Path); err != nil {
			logger.Error("Invalid DICOM file", "file", file.Path, "error", err)
			continue
		}

		// Parse DICOM file
		tags, err := c.parseDicomFile(file, logger)
		if err != nil {
			logger.Error("Failed to parse DICOM file", "file", file.Path, "error", err)
			continue
		}

		// Add file information if processing multiple files
		if len(files) > 1 {
			for i := range tags {
				tags[i].File = file.Name
			}
		}

		allTags = append(allTags, tags...)

		// Extract pixel data if requested
		if c.StorePixelData && c.ProcessPixelData {
			if err := c.extractPixelData(file, cfg.OutputDir, logger); err != nil {
				logger.Warn("Failed to extract pixel data", "file", file.Path, "error", err)
			}
		}

		logger.Debug("Processed file", "file", file.Name, "tags", len(tags))

		// Add separator for table format when processing multiple files
		if cfg.Format == config.FormatTable && i < len(files)-1 {
			fmt.Fprintln(os.Stdout, "\n"+ui.SubtleStyle.Render("---"))
		}
	}

	progress.Complete("Complete")

	// Render output
	logger.Debug("Rendering output", "format", cfg.Format, "tags", len(allTags))

	if err := RenderOutput(allTags, cfg.Format, os.Stdout); err != nil {
		return fmt.Errorf("failed to render output: %w", err)
	}

	logger.Info("Dump complete", "files", len(files), "tags", len(allTags))

	return nil
}

// parseDicomFile parses a DICOM file and extracts tags.
func (c *DumpCmd) parseDicomFile(file DICOMFile, logger *log.Logger) ([]DICOMTag, error) {
	// Parse DICOM file using go-radx
	dataset, err := dicom.ParseFile(file.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DICOM file: %w", err)
	}

	// Extract tags from dataset
	tags := make([]DICOMTag, 0)

	// Iterate through all elements in the dataset
	for _, elem := range dataset.Elements() {
		elemTag := elem.Tag()
		tagStr := fmt.Sprintf("(%04X,%04X)", elemTag.Group, elemTag.Element)
		vr := elem.VR().String()

		// Get tag name from dictionary
		name := elemTag.String()

		// Format value - use the String() method which gives human-readable output
		valueStr := elem.Value().String()

		tags = append(tags, DICOMTag{
			Tag:   tagStr,
			VR:    vr,
			Name:  name,
			Value: valueStr,
		})
	}

	logger.Debug("Extracted tags", "file", file.Name, "count", len(tags))

	return tags, nil
}

// extractPixelData extracts pixel data from a DICOM file to a separate file.
func (c *DumpCmd) extractPixelData(file DICOMFile, outputDir string, logger *log.Logger) error {
	// Create output directory
	if err := createOutputDirectory(outputDir); err != nil {
		return err
	}

	// Generate output filename
	baseFilename := filepath.Base(file.Name)
	ext := filepath.Ext(baseFilename)
	nameWithoutExt := baseFilename[:len(baseFilename)-len(ext)]
	outputPath := filepath.Join(outputDir, nameWithoutExt+".raw")

	logger.Debug("Extracting pixel data", "input", file.Path, "output", outputPath)

	// TODO: Implement pixel data extraction using go-radx pixel package
	// For now, just log a placeholder
	logger.Warn("Pixel data extraction not yet implemented", "file", file.Name)

	return fmt.Errorf("pixel data extraction not implemented")
}
