// Package pixel provides functionality for extracting, creating, transforming, and processing DICOM pixel data.
//
// This package handles various DICOM transfer syntaxes including:
//   - Native (uncompressed) pixel data
//   - RLE Lossless (1.2.840.10008.1.2.5)
//   - JPEG Baseline (1.2.840.10008.1.2.4.50)
//   - JPEG Lossless (1.2.840.10008.1.2.4.70) via CGo
//   - JPEG 2000 (1.2.840.10008.1.2.4.90/91) via CGo
//   - High-Throughput JPEG 2000 / HTJ2K (1.2.840.10008.1.2.4.201/203) via CGo
//
// # Basic Usage
//
// Extract pixel data from a DICOM dataset:
//
//	ds, err := dicom.ParseFile("ct_image.dcm")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	pixelData, err := pixel.Extract(ds)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Access pixel values as typed array
//	pixels := pixelData.Array() // Returns []uint8, []uint16, or []int16
//
// # Creating Pixel Data
//
// Create new pixel data from raw arrays for image generation, AI model outputs, or reconstructions:
//
//	// From 8-bit grayscale
//	data := make([]uint8, 512*512)
//	pixelData, err := pixel.NewPixelDataFromUint8(data, 512, 512)
//
//	// From 16-bit signed (for CT Hounsfield Units)
//	huData := make([]int16, 512*512)
//	pixelData, err := pixel.NewPixelDataFromInt16(huData, 512, 512)
//
//	// From RGB color data
//	rgbData := make([]byte, 512*512*3)
//	pixelData, err := pixel.NewPixelDataFromRGB(rgbData, 512, 512)
//
// # Multi-Frame Support
//
// For multi-frame datasets, access individual frames:
//
//	frames := pixelData.Frames()
//	for i, frame := range frames {
//	    fmt.Printf("Frame %d: %dx%d\n", i, frame.Columns, frame.Rows)
//	    pixels := frame.Array()
//	    // Process frame pixels...
//	}
//
// # Image Transformations
//
// Convert between color spaces (photometric interpretations):
//
//	// RGB → YBR_FULL for compression
//	ybrData, err := pixel.ConvertPhotometricInterpretation(pixelData, "YBR_FULL")
//
//	// MONOCHROME1 → MONOCHROME2 (inversion)
//	mono2, err := pixel.ConvertPhotometricInterpretation(mono1Data, "MONOCHROME2")
//
// Convert between planar configurations:
//
//	// Planar → Interleaved (RRR...GGG...BBB... → RGBRGBRGB...)
//	interleaved, err := pixel.ConvertPlanarConfiguration(planarData, 0)
//
// # LUT Transformations
//
// Apply window/level (VOI LUT) for optimal tissue visualization:
//
//	// Lung window for CT: center=-600, width=1500, output 8-bit
//	displayPixels, err := pixel.ApplyWindowLevel(pixelData, -600, 1500, 8)
//
// Apply modality LUT to convert to physical units (e.g., Hounsfield Units):
//
//	// Convert to HU: HU = slope * pixel_value + intercept
//	huData, err := pixel.ApplyModalityLUT(pixelData, 1.0, -1024)
//
// Apply complete DICOM display pipeline (Modality LUT → VOI LUT):
//
//	// Automatically applies transformations based on DICOM tags
//	displayData, err := pixel.ApplyFullImagePipeline(ds, pixelData, 8)
//
//	// Convert to standard image format
//	img := displayData.Image() // Returns image.Image
//
// # Decoder Registry
//
// The package uses a pluggable decoder registry. Custom decoders can be registered for
// proprietary or unsupported transfer syntaxes:
//
//	pixel.RegisterDecoder("1.2.3.4.5.6.7", myCustomDecoder)
//
// # CGo Dependencies
//
// Some decoders require external C libraries:
//   - JPEG Lossless: libjpeg-turbo
//   - JPEG 2000, HTJ2K: OpenJPEG 2.5+
//
// See the project README for installation instructions.
package pixel
