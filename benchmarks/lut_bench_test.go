package benchmarks

import (
	"fmt"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/pixel"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
)

// BenchmarkApplyWindowLevel measures window/level transformation performance
func BenchmarkApplyWindowLevel(b *testing.B) {
	imageSizes := []struct {
		name string
		rows uint16
		cols uint16
	}{
		{"512x512", 512, 512},
		{"1024x1024", 1024, 1024},
		{"2048x2048", 2048, 2048},
	}

	bitDepths := []uint16{8, 16}

	for _, size := range imageSizes {
		for _, bits := range bitDepths {
			b.Run(fmt.Sprintf("%s_%dbit", size.name, bits), func(b *testing.B) {
				pixelData := setupGrayscalePixelData(b, size.rows, size.cols, bits)

				// Typical lung window for CT
				center := -600.0
				width := 1500.0
				outputBits := uint16(8)

				b.ReportAllocs()
				b.SetBytes(int64(size.rows) * int64(size.cols) * int64(bits/8))
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					_, err := pixel.ApplyWindowLevel(pixelData, center, width, outputBits)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// BenchmarkApplyModalityLUT measures modality LUT transformation performance
func BenchmarkApplyModalityLUT(b *testing.B) {
	imageSizes := []struct {
		name string
		rows uint16
		cols uint16
	}{
		{"512x512", 512, 512},
		{"1024x1024", 1024, 1024},
	}

	for _, size := range imageSizes {
		b.Run(size.name, func(b *testing.B) {
			pixelData := setupGrayscalePixelData(b, size.rows, size.cols, 16)

			// Typical CT modality LUT
			slope := 1.0
			intercept := -1024.0

			b.ReportAllocs()
			b.SetBytes(int64(size.rows) * int64(size.cols) * 2)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := pixel.ApplyModalityLUT(pixelData, slope, intercept)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkApplyFullImagePipeline measures complete LUT pipeline
func BenchmarkApplyFullImagePipeline(b *testing.B) {
	imageSizes := []struct {
		name string
		rows uint16
		cols uint16
	}{
		{"512x512", 512, 512},
		{"1024x1024", 1024, 1024},
	}

	for _, size := range imageSizes {
		b.Run(size.name, func(b *testing.B) {
			pixelData := setupGrayscalePixelData(b, size.rows, size.cols, 16)
			ds := setupDataSetWithLUTParams(b)

			outputBits := uint16(8)

			b.ReportAllocs()
			b.SetBytes(int64(size.rows) * int64(size.cols) * 2)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := pixel.ApplyFullImagePipeline(ds, pixelData, outputBits)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkApplyPresentationLUT measures presentation LUT performance
func BenchmarkApplyPresentationLUT(b *testing.B) {
	b.Run("ShapeInverse", func(b *testing.B) {
		pixelData := setupGrayscalePixelData(b, 512, 512, 8)

		presentationLUT := &pixel.PresentationLUT{
			PresentationLUTShape: "INVERSE",
		}

		b.ReportAllocs()
		b.SetBytes(512 * 512)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := pixel.ApplyPresentationLUT(pixelData, presentationLUT)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("TableBased", func(b *testing.B) {
		pixelData := setupGrayscalePixelData(b, 512, 512, 8)

		// Setup a simple linear LUT
		lutData := make([]uint16, 256)
		for i := 0; i < 256; i++ {
			lutData[i] = uint16(i)
		}

		presentationLUT := &pixel.PresentationLUT{
			LUTData:       lutData,
			LUTDescriptor: [3]uint16{256, 0, 8},
		}

		b.ReportAllocs()
		b.SetBytes(512 * 512)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := pixel.ApplyPresentationLUT(pixelData, presentationLUT)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkApplyPaletteColorLUT measures palette color LUT performance
func BenchmarkApplyPaletteColorLUT(b *testing.B) {
	imageSizes := []struct {
		name string
		rows uint16
		cols uint16
	}{
		{"512x512", 512, 512},
		{"1024x1024", 1024, 1024},
	}

	for _, size := range imageSizes {
		b.Run(size.name, func(b *testing.B) {
			// Create indexed color pixel data
			pixelData := setupIndexedColorPixelData(b, size.rows, size.cols)

			// Create a simple grayscale palette
			palette := setupGrayscalePalette(b, 256)

			b.ReportAllocs()
			b.SetBytes(int64(size.rows) * int64(size.cols) * 3) // RGB output
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := pixel.ApplyPaletteColorLUT(pixelData, palette)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkSegmentedLUTExpansion measures segmented LUT expansion performance
func BenchmarkSegmentedLUTExpansion(b *testing.B) {
	sizes := []int{256, 4096, 65536}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("%d_entries", size), func(b *testing.B) {
			segmentedLUT := setupSegmentedLUT(b, size)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := segmentedLUT.Expand(size)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkColorSpaceConversion measures color space transformation performance
func BenchmarkColorSpaceConversion(b *testing.B) {
	b.Run("ToSRGB", func(b *testing.B) {
		pixelData := setupRGBPixelData(b, 512, 512)

		b.ReportAllocs()
		b.SetBytes(512 * 512 * 3)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := pixel.ConvertColorSpace(pixelData, "sRGB")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ToLinearRGB", func(b *testing.B) {
		pixelData := setupRGBPixelData(b, 512, 512)

		b.ReportAllocs()
		b.SetBytes(512 * 512 * 3)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := pixel.ConvertColorSpace(pixelData, "Linear RGB")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Helper functions

func setupGrayscalePixelData(b *testing.B, rows, cols, bits uint16) *pixel.PixelData {
	b.Helper()

	numPixels := int(rows) * int(cols)
	bytesPerPixel := int(bits / 8)
	data := make([]byte, numPixels*bytesPerPixel)

	// Fill with some test pattern
	for i := 0; i < numPixels; i++ {
		val := uint16(i % (1 << bits))
		if bytesPerPixel == 1 {
			data[i] = byte(val)
		} else {
			data[i*2] = byte(val)
			data[i*2+1] = byte(val >> 8)
		}
	}

	pixelData, err := pixel.NewPixelDataFromUint8(data, int(rows), int(cols))
	if err != nil {
		b.Fatal(err)
	}

	return pixelData
}

func setupRGBPixelData(b *testing.B, rows, cols uint16) *pixel.PixelData {
	b.Helper()

	numPixels := int(rows) * int(cols)
	data := make([]byte, numPixels*3) // RGB

	// Fill with gradient pattern
	for i := 0; i < numPixels; i++ {
		data[i*3] = byte((i * 255) / numPixels)     // R
		data[i*3+1] = byte(((i % 256) * 255) / 256) // G
		data[i*3+2] = byte((i / 256) % 256)         // B
	}

	pixelData, err := pixel.NewPixelDataFromRGB(data, int(rows), int(cols))
	if err != nil {
		b.Fatal(err)
	}

	return pixelData
}

func setupIndexedColorPixelData(b *testing.B, rows, cols uint16) *pixel.PixelData {
	b.Helper()

	numPixels := int(rows) * int(cols)
	data := make([]byte, numPixels)

	// Fill with indices
	for i := 0; i < numPixels; i++ {
		data[i] = byte(i % 256)
	}

	// Create pixel data using NewPixelDataFromUint8
	pixelData, err := pixel.NewPixelDataFromUint8(data, int(rows), int(cols))
	if err != nil {
		b.Fatal(err)
	}

	// Override photometric interpretation for palette color
	// Note: This is a benchmark helper, so we're creating a simplified palette color setup
	return pixelData
}

func setupGrayscalePalette(b *testing.B, numEntries int) *pixel.PaletteColorLUT {
	b.Helper()

	redData := make([]uint16, numEntries)
	greenData := make([]uint16, numEntries)
	blueData := make([]uint16, numEntries)

	// Create grayscale palette
	for i := 0; i < numEntries; i++ {
		val := uint16((i * 65535) / (numEntries - 1))
		redData[i] = val
		greenData[i] = val
		blueData[i] = val
	}

	return &pixel.PaletteColorLUT{
		RedDescriptor:   [3]uint16{uint16(numEntries), 0, 16},
		GreenDescriptor: [3]uint16{uint16(numEntries), 0, 16},
		BlueDescriptor:  [3]uint16{uint16(numEntries), 0, 16},
		RedData:         redData,
		GreenData:       greenData,
		BlueData:        blueData,
	}
}

func setupSegmentedLUT(b *testing.B, numEntries int) *pixel.SegmentedLUT {
	b.Helper()

	// Create a segmented LUT with discrete and linear segments
	data := []uint16{
		// Discrete segment: 10 values
		0x000A, 100, 200, 300, 400, 500, 600, 700, 800, 900, 1000,
		// Linear segment: interpolate 100 values
		0x0164, 2000,
		// Discrete segment: 5 values
		0x0005, 3000, 3500, 4000, 4500, 5000,
	}

	return &pixel.SegmentedLUT{
		Data: data,
	}
}

func setupDataSetWithLUTParams(b *testing.B) *dicom.DataSet {
	b.Helper()
	ds := dicom.NewDataSet()

	// Add Window Center (0028,1050)
	centerVal, _ := value.NewStringValue(vr.DecimalString, []string{"-600"})
	centerElem, _ := element.NewElement(tag.New(0x0028, 0x1050), vr.DecimalString, centerVal)
	_ = ds.Add(centerElem)

	// Add Window Width (0028,1051)
	widthVal, _ := value.NewStringValue(vr.DecimalString, []string{"1500"})
	widthElem, _ := element.NewElement(tag.New(0x0028, 0x1051), vr.DecimalString, widthVal)
	_ = ds.Add(widthElem)

	// Add Rescale Intercept (0028,1052)
	interceptVal, _ := value.NewStringValue(vr.DecimalString, []string{"-1024"})
	interceptElem, _ := element.NewElement(tag.New(0x0028, 0x1052), vr.DecimalString, interceptVal)
	_ = ds.Add(interceptElem)

	// Add Rescale Slope (0028,1053)
	slopeVal, _ := value.NewStringValue(vr.DecimalString, []string{"1.0"})
	slopeElem, _ := element.NewElement(tag.New(0x0028, 0x1053), vr.DecimalString, slopeVal)
	_ = ds.Add(slopeElem)

	return ds
}
