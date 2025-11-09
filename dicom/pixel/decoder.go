package pixel

import (
	"fmt"
	"sync"
)

// Decoder defines the interface for decompressing pixel data from a specific transfer syntax.
//
// Implementations must be safe for concurrent use.
type Decoder interface {
	// Decode decompresses encapsulated pixel data.
	//
	// Parameters:
	//   - encapsulated: Raw compressed pixel data bytes
	//   - info: Metadata about the pixel data structure
	//
	// Returns:
	//   - Decompressed pixel data bytes
	//   - Error if decompression fails
	Decode(encapsulated []byte, info *PixelInfo) ([]byte, error)

	// TransferSyntaxUID returns the transfer syntax UID this decoder handles.
	TransferSyntaxUID() string
}

// PixelInfo contains metadata needed for pixel data decompression.
type PixelInfo struct {
	Rows                      uint16
	Columns                   uint16
	BitsAllocated             uint16
	BitsStored                uint16
	HighBit                   uint16
	PixelRepresentation       uint16
	SamplesPerPixel           uint16
	PhotometricInterpretation string
	PlanarConfiguration       uint16
	NumberOfFrames            int
	TransferSyntaxUID         string
}

// decoderRegistry manages registered pixel data decoders.
var (
	decoderRegistry   = make(map[string]Decoder)
	decoderRegistryMu sync.RWMutex
)

// RegisterDecoder registers a decoder for a specific transfer syntax UID.
//
// If a decoder is already registered for the UID, it will be replaced.
// This function is safe for concurrent use.
//
// Example:
//
//	pixel.RegisterDecoder("1.2.840.10008.1.2.5", rleDecoder)
func RegisterDecoder(transferSyntaxUID string, decoder Decoder) {
	decoderRegistryMu.Lock()
	defer decoderRegistryMu.Unlock()
	decoderRegistry[transferSyntaxUID] = decoder
}

// GetDecoder retrieves the decoder for a specific transfer syntax UID.
//
// Returns an error if no decoder is registered for the UID.
// This function is safe for concurrent use.
func GetDecoder(transferSyntaxUID string) (Decoder, error) {
	decoderRegistryMu.RLock()
	defer decoderRegistryMu.RUnlock()

	decoder, ok := decoderRegistry[transferSyntaxUID]
	if !ok {
		return nil, &TransferSyntaxError{UID: transferSyntaxUID}
	}
	return decoder, nil
}

// UnregisterDecoder removes a decoder for a specific transfer syntax UID.
//
// This is primarily useful for testing. Most applications should not need to unregister decoders.
// This function is safe for concurrent use.
func UnregisterDecoder(transferSyntaxUID string) {
	decoderRegistryMu.Lock()
	defer decoderRegistryMu.Unlock()
	delete(decoderRegistry, transferSyntaxUID)
}

// ListDecoders returns a list of all registered transfer syntax UIDs.
//
// This function is safe for concurrent use.
func ListDecoders() []string {
	decoderRegistryMu.RLock()
	defer decoderRegistryMu.RUnlock()

	uids := make([]string, 0, len(decoderRegistry))
	for uid := range decoderRegistry {
		uids = append(uids, uid)
	}
	return uids
}

// NativeDecoder handles uncompressed (native) pixel data.
//
// This is a no-op decoder that returns the input data unchanged.
type NativeDecoder struct{}

// Decode returns the input data unchanged (no decompression needed).
func (d *NativeDecoder) Decode(encapsulated []byte, info *PixelInfo) ([]byte, error) {
	return encapsulated, nil
}

// TransferSyntaxUID returns an empty string since native data doesn't require a specific UID.
func (d *NativeDecoder) TransferSyntaxUID() string {
	return ""
}

// CalculateExpectedSize calculates the expected size in bytes for pixel data based on metadata.
//
// Formula: Rows × Columns × SamplesPerPixel × NumberOfFrames × (BitsAllocated / 8)
func CalculateExpectedSize(info *PixelInfo) int {
	bytesPerSample := (int(info.BitsAllocated) + 7) / 8
	return int(info.Rows) * int(info.Columns) * int(info.SamplesPerPixel) * info.NumberOfFrames * bytesPerSample
}

// ValidatePixelData validates that pixel data size matches expected size based on metadata.
func ValidatePixelData(data []byte, info *PixelInfo) error {
	expected := CalculateExpectedSize(info)
	actual := len(data)

	if actual != expected {
		return &PixelDataError{
			Field:    "PixelData length",
			Expected: fmt.Sprintf("%d bytes", expected),
			Actual:   fmt.Sprintf("%d bytes", actual),
		}
	}
	return nil
}

func init() {
	// Register native decoder for uncompressed transfer syntaxes
	nativeDecoder := &NativeDecoder{}

	// Implicit VR Little Endian (default)
	RegisterDecoder("1.2.840.10008.1.2", nativeDecoder)

	// Explicit VR Little Endian
	RegisterDecoder("1.2.840.10008.1.2.1", nativeDecoder)

	// Explicit VR Big Endian (retired but still in use)
	RegisterDecoder("1.2.840.10008.1.2.2", nativeDecoder)

	// Deflated Explicit VR Little Endian
	RegisterDecoder("1.2.840.10008.1.2.1.99", nativeDecoder)
}
