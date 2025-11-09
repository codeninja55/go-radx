package pixel

import (
	"bytes"
	"fmt"
	"image/color"
	"math"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/tag"
)

// ICCProfile represents an ICC color profile embedded in DICOM.
//
// ICC profiles enable device-independent color representation by defining
// the relationship between image color values and a standard color space (PCS).
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part03.html#sect_C.11.15
//
// ICC Specification:
// https://www.color.org/specification/ICC.2-2023.pdf
type ICCProfile struct {
	// ProfileData is the raw ICC profile data
	ProfileData []byte

	// Parsed header information
	ProfileSize        uint32
	PreferredCMMType   [4]byte
	ProfileVersion     [4]byte
	ProfileClass       [4]byte
	ColorSpaceType     [4]byte
	PCSType            [4]byte // Profile Connection Space
	CreationDateTime   [12]byte
	ProfileSignature   [4]byte
	PlatformSignature  [4]byte
	ProfileFlags       uint32
	DeviceManufacturer [4]byte
	DeviceModel        [4]byte
	RenderingIntent    uint32
}

// ColorSpaceTransform defines color space transformation parameters.
type ColorSpaceTransform struct {
	SourceColorSpace string
	TargetColorSpace string
	ICCProfile       *ICCProfile
	RenderingIntent  RenderingIntent
}

// RenderingIntent specifies how to handle out-of-gamut colors during transformation.
type RenderingIntent int

const (
	// RenderingIntentPerceptual preserves visual relationship between colors (photography).
	RenderingIntentPerceptual RenderingIntent = 0

	// RenderingIntentRelativeColorimetric preserves in-gamut colors exactly (proofing).
	RenderingIntentRelativeColorimetric RenderingIntent = 1

	// RenderingIntentSaturation preserves saturation (business graphics).
	RenderingIntentSaturation RenderingIntent = 2

	// RenderingIntentAbsoluteColorimetric preserves absolute colors (color matching).
	RenderingIntentAbsoluteColorimetric RenderingIntent = 3
)

// ICC profile header offsets and constants
const (
	iccHeaderSize            = 128
	iccProfileSizeOffset     = 0
	iccCMMTypeOffset         = 4
	iccVersionOffset         = 8
	iccClassOffset           = 12
	iccColorSpaceOffset      = 16
	iccPCSOffset             = 20
	iccDateTimeOffset        = 24
	iccSignatureOffset       = 36
	iccPlatformOffset        = 40
	iccFlagsOffset           = 44
	iccManufacturerOffset    = 48
	iccModelOffset           = 52
	iccRenderingIntentOffset = 64

	iccSignature = "acsp" // ICC profile signature
)

// NewICCProfile creates an ICC profile from raw profile data.
//
// Parameters:
//   - data: Raw ICC profile data
//
// Returns parsed ICC profile with header information extracted.
//
// Example:
//
//	iccData, err := pixel.ExtractICCProfileFromDataSet(ds)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	profile, err := pixel.NewICCProfile(iccData)
func NewICCProfile(data []byte) (*ICCProfile, error) {
	if len(data) < iccHeaderSize {
		return nil, fmt.Errorf("ICC profile data too short: %d bytes (minimum %d)",
			len(data), iccHeaderSize)
	}

	profile := &ICCProfile{
		ProfileData: data,
	}

	// Parse header
	if err := profile.parseHeader(); err != nil {
		return nil, fmt.Errorf("failed to parse ICC profile header: %w", err)
	}

	// Validate signature
	if string(profile.ProfileSignature[:]) != iccSignature {
		return nil, fmt.Errorf("invalid ICC profile signature: %s (expected %s)",
			string(profile.ProfileSignature[:]), iccSignature)
	}

	return profile, nil
}

// parseHeader parses the ICC profile header.
func (p *ICCProfile) parseHeader() error {
	data := p.ProfileData

	// Profile size (big-endian uint32)
	p.ProfileSize = uint32(data[iccProfileSizeOffset])<<24 |
		uint32(data[iccProfileSizeOffset+1])<<16 |
		uint32(data[iccProfileSizeOffset+2])<<8 |
		uint32(data[iccProfileSizeOffset+3])

	// Verify size matches actual data length
	if p.ProfileSize != uint32(len(data)) {
		return fmt.Errorf("profile size mismatch: header says %d, actual %d",
			p.ProfileSize, len(data))
	}

	// CMM Type
	copy(p.PreferredCMMType[:], data[iccCMMTypeOffset:iccCMMTypeOffset+4])

	// Version
	copy(p.ProfileVersion[:], data[iccVersionOffset:iccVersionOffset+4])

	// Profile Class
	copy(p.ProfileClass[:], data[iccClassOffset:iccClassOffset+4])

	// Color Space Type
	copy(p.ColorSpaceType[:], data[iccColorSpaceOffset:iccColorSpaceOffset+4])

	// PCS Type
	copy(p.PCSType[:], data[iccPCSOffset:iccPCSOffset+4])

	// Creation DateTime
	copy(p.CreationDateTime[:], data[iccDateTimeOffset:iccDateTimeOffset+12])

	// Profile Signature
	copy(p.ProfileSignature[:], data[iccSignatureOffset:iccSignatureOffset+4])

	// Platform Signature
	copy(p.PlatformSignature[:], data[iccPlatformOffset:iccPlatformOffset+4])

	// Profile Flags
	p.ProfileFlags = uint32(data[iccFlagsOffset])<<24 |
		uint32(data[iccFlagsOffset+1])<<16 |
		uint32(data[iccFlagsOffset+2])<<8 |
		uint32(data[iccFlagsOffset+3])

	// Device Manufacturer
	copy(p.DeviceManufacturer[:], data[iccManufacturerOffset:iccManufacturerOffset+4])

	// Device Model
	copy(p.DeviceModel[:], data[iccModelOffset:iccModelOffset+4])

	// Rendering Intent
	p.RenderingIntent = uint32(data[iccRenderingIntentOffset])<<24 |
		uint32(data[iccRenderingIntentOffset+1])<<16 |
		uint32(data[iccRenderingIntentOffset+2])<<8 |
		uint32(data[iccRenderingIntentOffset+3])

	return nil
}

// ColorSpace returns the color space type as a string.
func (p *ICCProfile) ColorSpace() string {
	return string(p.ColorSpaceType[:])
}

// PCS returns the Profile Connection Space type as a string.
func (p *ICCProfile) PCS() string {
	return string(p.PCSType[:])
}

// Class returns the profile class as a string.
func (p *ICCProfile) Class() string {
	return string(p.ProfileClass[:])
}

// Version returns the ICC profile version as a string.
func (p *ICCProfile) Version() string {
	major := p.ProfileVersion[0]
	minor := p.ProfileVersion[1] >> 4
	bugfix := p.ProfileVersion[1] & 0x0F
	return fmt.Sprintf("%d.%d.%d", major, minor, bugfix)
}

// ApplyICCProfile applies ICC profile transformation to pixel data.
//
// This function applies the embedded ICC profile to transform pixel colors
// to a standard color space (sRGB by default).
//
// Parameters:
//   - p: Source pixel data
//   - profile: ICC profile to apply
//   - targetColorSpace: Target color space (e.g., "sRGB", "Adobe RGB")
//
// Returns transformed pixel data in target color space.
//
// Example:
//
//	profile, err := pixel.ExtractICCProfileFromDataSet(ds)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	srgb, err := pixel.ApplyICCProfile(pixelData, profile, "sRGB")
func ApplyICCProfile(p *PixelData, profile *ICCProfile, targetColorSpace string) (*PixelData, error) {
	if profile == nil {
		return nil, fmt.Errorf("ICC profile cannot be nil")
	}

	if p.SamplesPerPixel != 3 {
		return nil, fmt.Errorf("ICC profile transform requires RGB data (SamplesPerPixel=3), got %d",
			p.SamplesPerPixel)
	}

	// For now, provide a placeholder implementation
	// Full ICC profile transformation requires a complete ICC engine
	// which would involve lookup tables, matrix math, and interpolation
	//
	// Production implementation would use:
	// - Little CMS (lcms2) via CGo
	// - Pure Go ICC library (when available)

	return nil, fmt.Errorf("ICC profile transformation not yet fully implemented")
}

// ExtractICCProfileFromDataSet extracts an ICC profile from a DICOM DataSet.
//
// Reads:
//   - ICC Profile (0028,2000) - Raw ICC profile data
//
// Returns the ICC profile data or an error if not present.
//
// Example:
//
//	profileData, err := pixel.ExtractICCProfileFromDataSet(ds)
//	if err != nil {
//	    log.Printf("No ICC profile: %v", err)
//	} else {
//	    profile, err := pixel.NewICCProfile(profileData)
//	}
func ExtractICCProfileFromDataSet(ds *dicom.DataSet) ([]byte, error) {
	// ICC Profile tag (0028,2000)
	elem, err := ds.Get(tag.New(0x0028, 0x2000))
	if err != nil {
		return nil, fmt.Errorf("ICC profile not found: %w", err)
	}

	// ICC profile is stored as OB (Other Byte String)
	// Extract the raw bytes
	profileData := elem.Value().Bytes()
	if len(profileData) == 0 {
		return nil, fmt.Errorf("ICC profile is empty")
	}

	return profileData, nil
}

// HasICCProfile checks if a DICOM DataSet contains an ICC profile.
func HasICCProfile(ds *dicom.DataSet) bool {
	return ds.Contains(tag.New(0x0028, 0x2000))
}

// ConvertColorSpace converts pixel data from one color space to another.
//
// Supported conversions:
//   - RGB ↔ sRGB (with gamma correction)
//   - RGB → Linear RGB
//   - Linear RGB → sRGB
//
// For ICC profile-based conversions, use ApplyICCProfile.
//
// Parameters:
//   - p: Source pixel data
//   - targetColorSpace: Target color space
//
// Example:
//
//	// Convert RGB to sRGB (apply gamma correction)
//	srgb, err := pixel.ConvertColorSpace(rgbData, "sRGB")
func ConvertColorSpace(p *PixelData, targetColorSpace string) (*PixelData, error) {
	if p.SamplesPerPixel != 3 {
		return nil, fmt.Errorf("color space conversion requires RGB data (SamplesPerPixel=3), got %d",
			p.SamplesPerPixel)
	}

	switch targetColorSpace {
	case "sRGB":
		return convertToSRGB(p)
	case "Linear RGB":
		return convertToLinearRGB(p)
	default:
		return nil, fmt.Errorf("unsupported target color space: %s", targetColorSpace)
	}
}

// convertToSRGB applies sRGB gamma correction.
func convertToSRGB(p *PixelData) (*PixelData, error) {
	data := make([]byte, len(p.data))

	numPixels := len(p.data) / 3

	for i := 0; i < numPixels; i++ {
		r := linearToSRGB(float64(p.data[i*3]) / 255.0)
		g := linearToSRGB(float64(p.data[i*3+1]) / 255.0)
		b := linearToSRGB(float64(p.data[i*3+2]) / 255.0)

		data[i*3] = uint8(r * 255.0)
		data[i*3+1] = uint8(g * 255.0)
		data[i*3+2] = uint8(b * 255.0)
	}

	result := &PixelData{
		Rows:                      p.Rows,
		Columns:                   p.Columns,
		BitsAllocated:             p.BitsAllocated,
		BitsStored:                p.BitsStored,
		HighBit:                   p.HighBit,
		PixelRepresentation:       p.PixelRepresentation,
		SamplesPerPixel:           p.SamplesPerPixel,
		PhotometricInterpretation: "sRGB",
		PlanarConfiguration:       p.PlanarConfiguration,
		NumberOfFrames:            p.NumberOfFrames,
		data:                      data,
		TransferSyntaxUID:         p.TransferSyntaxUID,
	}

	return result, nil
}

// convertToLinearRGB removes gamma correction.
func convertToLinearRGB(p *PixelData) (*PixelData, error) {
	data := make([]byte, len(p.data))

	numPixels := len(p.data) / 3

	for i := 0; i < numPixels; i++ {
		r := srgbToLinear(float64(p.data[i*3]) / 255.0)
		g := srgbToLinear(float64(p.data[i*3+1]) / 255.0)
		b := srgbToLinear(float64(p.data[i*3+2]) / 255.0)

		data[i*3] = uint8(r * 255.0)
		data[i*3+1] = uint8(g * 255.0)
		data[i*3+2] = uint8(b * 255.0)
	}

	result := &PixelData{
		Rows:                      p.Rows,
		Columns:                   p.Columns,
		BitsAllocated:             p.BitsAllocated,
		BitsStored:                p.BitsStored,
		HighBit:                   p.HighBit,
		PixelRepresentation:       p.PixelRepresentation,
		SamplesPerPixel:           p.SamplesPerPixel,
		PhotometricInterpretation: "RGB",
		PlanarConfiguration:       p.PlanarConfiguration,
		NumberOfFrames:            p.NumberOfFrames,
		data:                      data,
		TransferSyntaxUID:         p.TransferSyntaxUID,
	}

	return result, nil
}

// linearToSRGB converts linear RGB to sRGB using the sRGB gamma curve.
func linearToSRGB(linear float64) float64 {
	if linear <= 0.0031308 {
		return 12.92 * linear
	}
	return 1.055*math.Pow(linear, 1.0/2.4) - 0.055
}

// srgbToLinear converts sRGB to linear RGB.
func srgbToLinear(srgb float64) float64 {
	if srgb <= 0.04045 {
		return srgb / 12.92
	}
	return math.Pow((srgb+0.055)/1.055, 2.4)
}

// ToColor converts a pixel value to a color.Color.
//
// This is useful for integration with Go's image processing packages.
func ToColor(r, g, b uint8) color.Color {
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

// String returns a human-readable description of the ICC profile.
func (p *ICCProfile) String() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("ICC Profile v%s\n", p.Version()))
	buf.WriteString(fmt.Sprintf("  Class: %s\n", p.Class()))
	buf.WriteString(fmt.Sprintf("  Color Space: %s\n", p.ColorSpace()))
	buf.WriteString(fmt.Sprintf("  PCS: %s\n", p.PCS()))
	buf.WriteString(fmt.Sprintf("  Size: %d bytes\n", p.ProfileSize))
	buf.WriteString(fmt.Sprintf("  Rendering Intent: %d\n", p.RenderingIntent))
	return buf.String()
}
