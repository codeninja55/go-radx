package uid_test

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom/uid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUID_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		uid   string
		valid bool
	}{
		{
			name:  "valid transfer syntax UID",
			uid:   "1.2.840.10008.1.2",
			valid: true,
		},
		{
			name:  "valid SOP class UID",
			uid:   "1.2.840.10008.5.1.4.1.1.1",
			valid: true,
		},
		{
			name:  "valid private UID",
			uid:   "1.2.840.123456.1.2.3.4.5",
			valid: true,
		},
		{
			name:  "valid single digit components",
			uid:   "1.2.3",
			valid: true,
		},
		{
			name:  "empty string",
			uid:   "",
			valid: false,
		},
		{
			name:  "contains letters",
			uid:   "1.2.abc.4",
			valid: false,
		},
		{
			name:  "contains spaces",
			uid:   "1.2.840. 10008.1.2",
			valid: false,
		},
		{
			name:  "starts with dot",
			uid:   ".1.2.840.10008.1.2",
			valid: false,
		},
		{
			name:  "ends with dot",
			uid:   "1.2.840.10008.1.2.",
			valid: false,
		},
		{
			name:  "consecutive dots",
			uid:   "1.2..840.10008",
			valid: false,
		},
		{
			name:  "leading zero in component",
			uid:   "1.02.840.10008",
			valid: false,
		},
		{
			name:  "too long (>64 chars)",
			uid:   "1.2.3.4.5.6.7.8.9.10.11.12.13.14.15.16.17.18.19.20.21.22.23.24.25",
			valid: false,
		},
		{
			name:  "exactly 64 characters",
			uid:   "1.2.840.10008.5.1.4.1.1.1.2.3.4.5.6.7.8.9.10.11.12.13.14.15",
			valid: true,
		},
		{
			name:  "component with only zero",
			uid:   "1.2.0.10008",
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, uid.IsValid(tt.uid))
		})
	}
}

func TestUID_Parse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "valid UID",
			input:   "1.2.840.10008.1.2",
			want:    "1.2.840.10008.1.2",
			wantErr: false,
		},
		{
			name:    "invalid UID",
			input:   "1.2.abc.4",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := uid.Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.String())
		})
	}
}

func TestUID_String(t *testing.T) {
	u, err := uid.Parse("1.2.840.10008.1.2")
	require.NoError(t, err)
	assert.Equal(t, "1.2.840.10008.1.2", u.String())
}

func TestUID_Equals(t *testing.T) {
	u1, err := uid.Parse("1.2.840.10008.1.2")
	require.NoError(t, err)

	u2, err := uid.Parse("1.2.840.10008.1.2")
	require.NoError(t, err)

	u3, err := uid.Parse("1.2.840.10008.1.2.1")
	require.NoError(t, err)

	assert.True(t, u1.Equals(u2))
	assert.False(t, u1.Equals(u3))
}

// Test well-known Transfer Syntax UIDs
func TestUID_TransferSyntaxUIDs(t *testing.T) {
	tests := []struct {
		name string
		uid  uid.UID
		want string
	}{
		{
			name: "Implicit VR Little Endian",
			uid:  uid.ImplicitVRLittleEndian,
			want: "1.2.840.10008.1.2",
		},
		{
			name: "Explicit VR Little Endian",
			uid:  uid.ExplicitVRLittleEndian,
			want: "1.2.840.10008.1.2.1",
		},
		{
			name: "Explicit VR Big Endian",
			uid:  uid.ExplicitVRBigEndian,
			want: "1.2.840.10008.1.2.2",
		},
		{
			name: "Deflated Explicit VR Little Endian",
			uid:  uid.DeflatedExplicitVRLittleEndian,
			want: "1.2.840.10008.1.2.1.99",
		},
		{
			name: "JPEG Baseline Process 1",
			uid:  uid.JPEGBaselineProcess1,
			want: "1.2.840.10008.1.2.4.50",
		},
		{
			name: "JPEG Extended Process 2 & 4",
			uid:  uid.JPEGExtendedProcess2And4,
			want: "1.2.840.10008.1.2.4.51",
		},
		{
			name: "JPEG Lossless",
			uid:  uid.JPEGLosslessNonHierarchicalProcess14,
			want: "1.2.840.10008.1.2.4.57",
		},
		{
			name: "JPEG Lossless Non-Hierarchical First-Order Prediction",
			uid:  uid.JPEGLosslessNonHierarchicalFirstOrderPredictionProcess14SelectionValue1,
			want: "1.2.840.10008.1.2.4.70",
		},
		{
			name: "JPEG-LS Lossless",
			uid:  uid.JPEGLsLosslessImageCompression,
			want: "1.2.840.10008.1.2.4.80",
		},
		{
			name: "JPEG-LS Near-Lossless",
			uid:  uid.JPEGLsLossyNearLosslessImageCompression,
			want: "1.2.840.10008.1.2.4.81",
		},
		{
			name: "JPEG 2000 Lossless",
			uid:  uid.JPEG2000ImageCompressionLosslessOnly,
			want: "1.2.840.10008.1.2.4.90",
		},
		{
			name: "JPEG 2000",
			uid:  uid.JPEG2000ImageCompression,
			want: "1.2.840.10008.1.2.4.91",
		},
		{
			name: "RLE Lossless",
			uid:  uid.RLELossless,
			want: "1.2.840.10008.1.2.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.uid.String())
			assert.True(t, uid.IsValid(tt.uid.String()))
		})
	}
}

// Test some well-known SOP Class UIDs
func TestUID_SOPClassUIDs(t *testing.T) {
	tests := []struct {
		name string
		uid  uid.UID
		want string
	}{
		{
			name: "CR Image Storage",
			uid:  uid.ComputedRadiographyImageStorage,
			want: "1.2.840.10008.5.1.4.1.1.1",
		},
		{
			name: "CT Image Storage",
			uid:  uid.CTImageStorage,
			want: "1.2.840.10008.5.1.4.1.1.2",
		},
		{
			name: "MR Image Storage",
			uid:  uid.MRImageStorage,
			want: "1.2.840.10008.5.1.4.1.1.4",
		},
		{
			name: "Ultrasound Image Storage",
			uid:  uid.UltrasoundImageStorage,
			want: "1.2.840.10008.5.1.4.1.1.6",
		},
		{
			name: "Secondary Capture Image Storage",
			uid:  uid.SecondaryCaptureImageStorage,
			want: "1.2.840.10008.5.1.4.1.1.7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.uid.String())
			assert.True(t, uid.IsValid(tt.uid.String()))
		})
	}
}
