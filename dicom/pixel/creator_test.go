package pixel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPixelDataBuilder(t *testing.T) {
	builder := NewPixelDataBuilder()
	require.NotNil(t, builder)

	// Check defaults
	assert.Equal(t, uint16(0), builder.pixelRepresentation)
	assert.Equal(t, uint16(1), builder.samplesPerPixel)
	assert.Equal(t, "MONOCHROME2", builder.photometricInterpretation)
	assert.Equal(t, uint16(0), builder.planarConfiguration)
	assert.Equal(t, 1, builder.numberOfFrames)
}

func TestPixelDataBuilder_Grayscale8bit(t *testing.T) {
	data := make([]byte, 256*256)
	for i := range data {
		data[i] = byte(i % 256)
	}

	pixelData, err := NewPixelDataBuilder().
		WithDimensions(256, 256).
		WithBitsAllocated(8).
		WithPhotometricInterpretation("MONOCHROME2").
		WithPixelData(data).
		Build()

	require.NoError(t, err)
	require.NotNil(t, pixelData)

	assert.Equal(t, uint16(256), pixelData.Rows)
	assert.Equal(t, uint16(256), pixelData.Columns)
	assert.Equal(t, uint16(8), pixelData.BitsAllocated)
	assert.Equal(t, uint16(8), pixelData.BitsStored)
	assert.Equal(t, uint16(7), pixelData.HighBit)
	assert.Equal(t, uint16(0), pixelData.PixelRepresentation)
	assert.Equal(t, uint16(1), pixelData.SamplesPerPixel)
	assert.Equal(t, "MONOCHROME2", pixelData.PhotometricInterpretation)
	assert.Equal(t, 1, pixelData.NumberOfFrames)
	assert.Len(t, pixelData.data, 256*256)
}

func TestPixelDataBuilder_Grayscale16bit(t *testing.T) {
	data := make([]byte, 512*512*2)

	pixelData, err := NewPixelDataBuilder().
		WithDimensions(512, 512).
		WithBitsAllocated(16).
		WithPixelRepresentation(1). // Signed
		WithPhotometricInterpretation("MONOCHROME2").
		WithPixelData(data).
		Build()

	require.NoError(t, err)
	require.NotNil(t, pixelData)

	assert.Equal(t, uint16(512), pixelData.Rows)
	assert.Equal(t, uint16(16), pixelData.BitsAllocated)
	assert.Equal(t, uint16(1), pixelData.PixelRepresentation)
}

func TestPixelDataBuilder_RGB(t *testing.T) {
	data := make([]byte, 256*256*3)

	pixelData, err := NewPixelDataBuilder().
		WithDimensions(256, 256).
		WithBitsAllocated(8).
		WithSamplesPerPixel(3).
		WithPhotometricInterpretation("RGB").
		WithPlanarConfiguration(0). // Interleaved
		WithPixelData(data).
		Build()

	require.NoError(t, err)
	require.NotNil(t, pixelData)

	assert.Equal(t, uint16(3), pixelData.SamplesPerPixel)
	assert.Equal(t, "RGB", pixelData.PhotometricInterpretation)
	assert.Equal(t, uint16(0), pixelData.PlanarConfiguration)
	assert.Len(t, pixelData.data, 256*256*3)
}

func TestPixelDataBuilder_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		builder func() *PixelDataBuilder
		wantErr string
	}{
		{
			name: "missing dimensions",
			builder: func() *PixelDataBuilder {
				return NewPixelDataBuilder().
					WithBitsAllocated(8).
					WithPixelData([]byte{1, 2, 3})
			},
			wantErr: "dimensions not set",
		},
		{
			name: "missing pixel data",
			builder: func() *PixelDataBuilder {
				return NewPixelDataBuilder().
					WithDimensions(10, 10).
					WithBitsAllocated(8)
			},
			wantErr: "pixel data not set",
		},
		{
			name: "missing bits allocated",
			builder: func() *PixelDataBuilder {
				return NewPixelDataBuilder().
					WithDimensions(10, 10).
					WithPixelData(make([]byte, 100))
			},
			wantErr: "bits allocated not set",
		},
		{
			name: "size mismatch",
			builder: func() *PixelDataBuilder {
				return NewPixelDataBuilder().
					WithDimensions(10, 10).
					WithBitsAllocated(8).
					WithPixelData([]byte{1, 2, 3}) // Too small
			},
			wantErr: "pixel data size mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.builder().Build()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestNewPixelDataFromUint8(t *testing.T) {
	data := make([]uint8, 100*100)
	for i := range data {
		data[i] = uint8(i % 256)
	}

	pixelData, err := NewPixelDataFromUint8(data, 100, 100)
	require.NoError(t, err)
	require.NotNil(t, pixelData)

	assert.Equal(t, uint16(100), pixelData.Rows)
	assert.Equal(t, uint16(100), pixelData.Columns)
	assert.Equal(t, uint16(8), pixelData.BitsAllocated)
	assert.Equal(t, uint16(0), pixelData.PixelRepresentation)
	assert.Equal(t, "MONOCHROME2", pixelData.PhotometricInterpretation)
}

func TestNewPixelDataFromUint16(t *testing.T) {
	data := make([]uint16, 100*100)
	for i := range data {
		data[i] = uint16(i % 65536)
	}

	pixelData, err := NewPixelDataFromUint16(data, 100, 100)
	require.NoError(t, err)
	require.NotNil(t, pixelData)

	assert.Equal(t, uint16(16), pixelData.BitsAllocated)
	assert.Equal(t, uint16(0), pixelData.PixelRepresentation)

	// Verify endianness
	array := pixelData.Array().([]uint16)
	assert.Equal(t, data[0], array[0])
	assert.Equal(t, data[len(data)-1], array[len(array)-1])
}

func TestNewPixelDataFromInt16(t *testing.T) {
	data := make([]int16, 100*100)
	for i := range data {
		data[i] = int16(i - 5000) // Include negative values
	}

	pixelData, err := NewPixelDataFromInt16(data, 100, 100)
	require.NoError(t, err)
	require.NotNil(t, pixelData)

	assert.Equal(t, uint16(16), pixelData.BitsAllocated)
	assert.Equal(t, uint16(1), pixelData.PixelRepresentation) // Signed

	// Verify signed values
	array := pixelData.Array().([]int16)
	assert.Equal(t, data[0], array[0])
	assert.Equal(t, data[len(data)-1], array[len(array)-1])
}

func TestNewPixelDataFromRGB(t *testing.T) {
	data := make([]byte, 100*100*3)
	for i := 0; i < len(data); i += 3 {
		data[i] = 255   // R
		data[i+1] = 128 // G
		data[i+2] = 64  // B
	}

	pixelData, err := NewPixelDataFromRGB(data, 100, 100)
	require.NoError(t, err)
	require.NotNil(t, pixelData)

	assert.Equal(t, uint16(3), pixelData.SamplesPerPixel)
	assert.Equal(t, "RGB", pixelData.PhotometricInterpretation)
	assert.Equal(t, uint16(0), pixelData.PlanarConfiguration)

	// Verify data
	assert.Equal(t, byte(255), pixelData.data[0])
	assert.Equal(t, byte(128), pixelData.data[1])
	assert.Equal(t, byte(64), pixelData.data[2])
}

func TestNewPixelDataFromRGBPlanar(t *testing.T) {
	size := 100 * 100
	r := make([]byte, size)
	g := make([]byte, size)
	b := make([]byte, size)

	for i := 0; i < size; i++ {
		r[i] = 255
		g[i] = 128
		b[i] = 64
	}

	pixelData, err := NewPixelDataFromRGBPlanar(r, g, b, 100, 100)
	require.NoError(t, err)
	require.NotNil(t, pixelData)

	assert.Equal(t, uint16(3), pixelData.SamplesPerPixel)
	assert.Equal(t, "RGB", pixelData.PhotometricInterpretation)
	assert.Equal(t, uint16(1), pixelData.PlanarConfiguration) // Planar

	// Verify planar organization: RRR...GGG...BBB...
	assert.Equal(t, byte(255), pixelData.data[0])     // First R
	assert.Equal(t, byte(128), pixelData.data[size])  // First G
	assert.Equal(t, byte(64), pixelData.data[2*size]) // First B
}

func TestNewPixelDataFromRGBPlanar_SizeMismatch(t *testing.T) {
	r := make([]byte, 100)
	g := make([]byte, 100)
	b := make([]byte, 50) // Wrong size

	_, err := NewPixelDataFromRGBPlanar(r, g, b, 10, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "channel size mismatch")
}
