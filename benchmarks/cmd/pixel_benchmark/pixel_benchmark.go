package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/pixel"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/value"
)

// BenchmarkResult contains timing and hash information
type BenchmarkResult struct {
	Success        bool    `json:"success"`
	ParseTimeUs    float64 `json:"parse_time_us"`
	PixelTimeUs    float64 `json:"pixel_time_us"`
	TransferSyntax string  `json:"transfer_syntax"`
	PixelHash      string  `json:"pixel_hash"`
	ImageShape     []int   `json:"image_shape"`
	Error          string  `json:"error,omitempty"`
}

func hashPixels(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func benchmark(filePath string) BenchmarkResult {
	result := BenchmarkResult{
		Success: false,
	}

	// Time DICOM parsing
	parseStart := time.Now()
	ds, err := dicom.ParseFile(filePath)
	if err != nil {
		result.Error = fmt.Sprintf("parse DICOM: %v", err)
		return result
	}
	parseTime := time.Since(parseStart)
	result.ParseTimeUs = float64(parseTime.Microseconds())

	// Get transfer syntax from File Meta Information
	if fileMeta := ds.FileMetaInformation(); fileMeta != nil {
		if tsElem, err := fileMeta.Get(tag.TransferSyntaxUID); err == nil {
			if tsVal, ok := tsElem.Value().(*value.StringValue); ok {
				if strs := tsVal.Strings(); len(strs) > 0 {
					result.TransferSyntax = strs[0]
				}
			}
		}
	}

	// Time pixel extraction
	pixelStart := time.Now()
	pixelData, err := pixel.Extract(ds)
	if err != nil {
		result.Error = fmt.Sprintf("extract pixels: %v", err)
		return result
	}
	pixelTime := time.Since(pixelStart)
	result.PixelTimeUs = float64(pixelTime.Microseconds())

	// Get pixel data as bytes
	data := pixelData.RawBytes()

	// Compute hash
	result.PixelHash = hashPixels(data)

	// Get image shape
	numFrames := pixelData.NumberOfFrames
	rows := int(pixelData.Rows)
	cols := int(pixelData.Columns)
	samplesPerPixel := int(pixelData.SamplesPerPixel)

	if numFrames > 1 {
		if samplesPerPixel > 1 {
			result.ImageShape = []int{numFrames, rows, cols, samplesPerPixel}
		} else {
			result.ImageShape = []int{numFrames, rows, cols}
		}
	} else {
		if samplesPerPixel > 1 {
			result.ImageShape = []int{rows, cols, samplesPerPixel}
		} else {
			result.ImageShape = []int{rows, cols}
		}
	}

	result.Success = true
	return result
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <dicom-file>\n", os.Args[0])
		os.Exit(1)
	}

	filePath := os.Args[1]

	result := benchmark(filePath)

	// Output JSON
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "JSON encode error: %v\n", err)
		os.Exit(1)
	}
}
