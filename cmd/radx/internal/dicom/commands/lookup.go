package commands

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/alexeyco/simpletable"
	"github.com/charmbracelet/log"
	"github.com/codeninja55/go-radx/cmd/radx/internal/config"
	"github.com/codeninja55/go-radx/cmd/radx/internal/dicom/ui"
	"github.com/codeninja55/go-radx/dicom/tag"
)

// LookupCmd implements the DICOM tag lookup command.
type LookupCmd struct {
	Query []string `arg:"" required:"" help:"Tag ID ((GGGG,EEEE) or GGGGEEEE), keyword, or text to search for"`
}

// TagInfo represents information about a DICOM tag.
type TagInfo struct {
	Tag     string `json:"tag"`
	Name    string `json:"name"`
	Keyword string `json:"keyword"`
	VR      string `json:"vr"`
}

// Run executes the lookup command.
func (c *LookupCmd) Run(cfg *config.GlobalConfig) error {
	// Print banner
	ui.PrintBanner()

	logger := log.Default()
	logger.Info("Starting DICOM tag lookup")

	// Build tag dictionary
	logger.Debug("Building tag dictionary")
	tagDict := buildTagDictionary()
	logger.Debug("Tag dictionary built", "count", len(tagDict))

	// Search for tags
	results := make([]TagInfo, 0)
	for _, query := range c.Query {
		matches := searchTags(tagDict, query, logger)
		results = append(results, matches...)
	}

	// Remove duplicates
	results = removeDuplicateTagInfo(results)

	if len(results) == 0 {
		fmt.Println()
		fmt.Println(ui.WarnStyle.Render("No matching tags found"))
		fmt.Println()
		logger.Info("No matching tags found", "queries", c.Query)
		return nil
	}

	// Print results
	fmt.Println()
	fmt.Printf("%s %s\n\n", ui.InfoStyle.Render("Found"), ui.SuccessStyle.Render(fmt.Sprintf("%d matching tag(s)", len(results))))

	// Render as table by default, or as requested format
	if cfg.Format == config.FormatJSON {
		if err := renderTagInfoAsJSON(results, os.Stdout); err != nil {
			return fmt.Errorf("failed to render output: %w", err)
		}
	} else {
		renderTagInfoAsTable(results)
	}

	logger.Info("Lookup complete", "queries", c.Query, "results", len(results))

	return nil
}

// buildTagDictionary builds a dictionary of all known DICOM tags using reflection.
func buildTagDictionary() map[string]TagInfo {
	dict := make(map[string]TagInfo)

	// Use reflection to iterate through all exported variables in the tag package
	tagPkg := reflect.ValueOf(&tag.PatientName).Elem().Type().PkgPath()
	logger := log.Default()
	logger.Debug("Tag package path", "path", tagPkg)

	// Get common DICOM tags by creating instances
	// Note: This is a simplified implementation. A more complete version would need
	// to programmatically discover all tag constants.
	commonTags := []struct {
		tag     tag.Tag
		keyword string
	}{
		{tag.PatientName, "PatientName"},
		{tag.PatientID, "PatientID"},
		{tag.PatientBirthDate, "PatientBirthDate"},
		{tag.PatientSex, "PatientSex"},
		{tag.StudyInstanceUID, "StudyInstanceUID"},
		{tag.SeriesInstanceUID, "SeriesInstanceUID"},
		{tag.SOPInstanceUID, "SOPInstanceUID"},
		{tag.SOPClassUID, "SOPClassUID"},
		{tag.StudyDate, "StudyDate"},
		{tag.StudyTime, "StudyTime"},
		{tag.StudyDescription, "StudyDescription"},
		{tag.SeriesDescription, "SeriesDescription"},
		{tag.SeriesNumber, "SeriesNumber"},
		{tag.InstanceNumber, "InstanceNumber"},
		{tag.Modality, "Modality"},
		{tag.Manufacturer, "Manufacturer"},
		{tag.InstitutionName, "InstitutionName"},
		{tag.ReferringPhysicianName, "ReferringPhysicianName"},
		{tag.PerformingPhysicianName, "PerformingPhysicianName"},
		{tag.AccessionNumber, "AccessionNumber"},
		{tag.ImageType, "ImageType"},
		{tag.AcquisitionDate, "AcquisitionDate"},
		{tag.AcquisitionTime, "AcquisitionTime"},
		{tag.ContentDate, "ContentDate"},
		{tag.ContentTime, "ContentTime"},
		{tag.TransferSyntaxUID, "TransferSyntaxUID"},
		{tag.ImplementationClassUID, "ImplementationClassUID"},
		{tag.ImplementationVersionName, "ImplementationVersionName"},
		{tag.SpecificCharacterSet, "SpecificCharacterSet"},
		{tag.ImagePositionPatient, "ImagePositionPatient"},
		{tag.ImageOrientationPatient, "ImageOrientationPatient"},
		{tag.SliceLocation, "SliceLocation"},
		{tag.SliceThickness, "SliceThickness"},
		{tag.PixelSpacing, "PixelSpacing"},
		{tag.Rows, "Rows"},
		{tag.Columns, "Columns"},
		{tag.BitsAllocated, "BitsAllocated"},
		{tag.BitsStored, "BitsStored"},
		{tag.HighBit, "HighBit"},
		{tag.PixelRepresentation, "PixelRepresentation"},
		{tag.PixelData, "PixelData"},
		{tag.WindowCenter, "WindowCenter"},
		{tag.WindowWidth, "WindowWidth"},
		{tag.RescaleIntercept, "RescaleIntercept"},
		{tag.RescaleSlope, "RescaleSlope"},
		{tag.PhotometricInterpretation, "PhotometricInterpretation"},
		{tag.SamplesPerPixel, "SamplesPerPixel"},
		{tag.PlanarConfiguration, "PlanarConfiguration"},
		{tag.NumberOfFrames, "NumberOfFrames"},
	}

	for _, ct := range commonTags {
		tagStr := fmt.Sprintf("(%04X,%04X)", ct.tag.Group, ct.tag.Element)

		// Try to infer VR (this is simplified - actual DICOM tags have context-dependent VRs)
		vrStr := inferVR(ct.keyword)

		dict[tagStr] = TagInfo{
			Tag:     tagStr,
			Name:    ct.tag.String(),
			Keyword: ct.keyword,
			VR:      vrStr,
		}
	}

	return dict
}

// inferVR attempts to infer the VR for a tag based on its keyword.
// This is a simplified heuristic and not exhaustive.
func inferVR(keyword string) string {
	switch {
	case strings.Contains(keyword, "UID"):
		return "UI"
	case strings.Contains(keyword, "Date"):
		return "DA"
	case strings.Contains(keyword, "Time"):
		return "TM"
	case strings.Contains(keyword, "Name"):
		return "PN"
	case strings.Contains(keyword, "Description"):
		return "LO"
	case strings.Contains(keyword, "Number"):
		return "IS"
	case strings.Contains(keyword, "ID"):
		return "LO"
	case keyword == "PixelData":
		return "OB/OW"
	default:
		return "CS" // Code String as default
	}
}

// searchTags searches for tags matching the query.
func searchTags(dict map[string]TagInfo, query string, logger *log.Logger) []TagInfo {
	results := make([]TagInfo, 0)
	normalizedQuery := normalizeTagFilter(query)

	logger.Debug("Searching tags", "query", query, "normalized", normalizedQuery)

	for _, tagInfo := range dict {
		// Check if query matches tag notation, keyword, or name
		normalizedTag := normalizeTagFilter(tagInfo.Tag)
		normalizedKeyword := normalizeTagFilter(tagInfo.Keyword)
		normalizedName := normalizeTagFilter(tagInfo.Name)

		// Exact match on tag notation
		if normalizedTag == normalizedQuery {
			results = append(results, tagInfo)
			continue
		}

		// Exact match on keyword
		if normalizedKeyword == normalizedQuery {
			results = append(results, tagInfo)
			continue
		}

		// Text search (contains) on keyword or name
		if strings.Contains(normalizedKeyword, normalizedQuery) ||
			strings.Contains(normalizedName, normalizedQuery) {
			results = append(results, tagInfo)
		}
	}

	return results
}

// removeDuplicateTagInfo removes duplicate TagInfo entries.
func removeDuplicateTagInfo(tags []TagInfo) []TagInfo {
	seen := make(map[string]bool)
	result := make([]TagInfo, 0)

	for _, tag := range tags {
		if !seen[tag.Tag] {
			seen[tag.Tag] = true
			result = append(result, tag)
		}
	}

	return result
}

// renderTagInfoAsTable renders tag information as a formatted table.
func renderTagInfoAsTable(tags []TagInfo) {
	table := ui.NewTable()

	// Set table header
	table.Header = &simpletable.Header{
		Cells: []*simpletable.Cell{
			{Align: simpletable.AlignCenter, Text: "Tag"},
			{Align: simpletable.AlignCenter, Text: "VR"},
			{Align: simpletable.AlignLeft, Text: "Keyword"},
			{Align: simpletable.AlignLeft, Text: "Name"},
		},
	}

	// Add rows
	for _, tag := range tags {
		table.Body.Cells = append(table.Body.Cells, []*simpletable.Cell{
			{Text: ui.InfoStyle.Render(tag.Tag)},
			{Text: ui.SubtleStyle.Render(tag.VR)},
			{Text: ui.SuccessStyle.Render(tag.Keyword)},
			{Text: tag.Name},
		})
	}

	ui.PrintTable(table, os.Stdout)
}

// renderTagInfoAsJSON renders tag information as JSON.
func renderTagInfoAsJSON(tags []TagInfo, w *os.File) error {
	return renderAsJSON(convertTagInfoToDICOMTags(tags), w)
}

// convertTagInfoToDICOMTags converts TagInfo to DICOMTag for JSON rendering.
func convertTagInfoToDICOMTags(tags []TagInfo) []DICOMTag {
	result := make([]DICOMTag, len(tags))
	for i, tag := range tags {
		result[i] = DICOMTag{
			Tag:   tag.Tag,
			VR:    tag.VR,
			Name:  fmt.Sprintf("%s (%s)", tag.Name, tag.Keyword),
			Value: "", // No value for dictionary lookup
		}
	}
	return result
}
