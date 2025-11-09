package uid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookup(t *testing.T) {
	tests := []struct {
		name      string
		uid       string
		wantFound bool
		wantInfo  Info
	}{
		{
			name:      "valid transfer syntax",
			uid:       "1.2.840.10008.1.2",
			wantFound: true,
			wantInfo: Info{
				UID:     "1.2.840.10008.1.2",
				Name:    "Implicit VR Little Endian",
				Type:    TypeTransferSyntax,
				Info:    "Default Transfer Syntax for DICOM",
				Retired: false,
			},
		},
		{
			name:      "valid SOP class",
			uid:       "1.2.840.10008.5.1.4.1.1.2",
			wantFound: true,
			wantInfo: Info{
				UID:     "1.2.840.10008.5.1.4.1.1.2",
				Name:    "CT Image Storage",
				Type:    TypeSOPClass,
				Info:    "",
				Retired: false,
			},
		},
		{
			name:      "retired UID",
			uid:       "1.2.840.10008.1.2.2",
			wantFound: true,
			wantInfo: Info{
				UID:     "1.2.840.10008.1.2.2",
				Name:    "Explicit VR Big Endian",
				Type:    TypeTransferSyntax,
				Info:    "",
				Retired: true,
			},
		},
		{
			name:      "unknown UID",
			uid:       "1.2.3.4.5.6.7.8.9",
			wantFound: false,
			wantInfo:  Info{},
		},
		{
			name:      "empty string",
			uid:       "",
			wantFound: false,
			wantInfo:  Info{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, found := Lookup(tt.uid)
			assert.Equal(t, tt.wantFound, found)
			if tt.wantFound {
				assert.Equal(t, tt.wantInfo, info)
			}
		})
	}
}

func TestName(t *testing.T) {
	tests := []struct {
		name     string
		uid      string
		wantName string
	}{
		{
			name:     "transfer syntax",
			uid:      "1.2.840.10008.1.2.1",
			wantName: "Explicit VR Little Endian",
		},
		{
			name:     "SOP class",
			uid:      "1.2.840.10008.5.1.4.1.1.4",
			wantName: "MR Image Storage",
		},
		{
			name:     "verification SOP class",
			uid:      "1.2.840.10008.1.1",
			wantName: "Verification SOP Class",
		},
		{
			name:     "unknown UID",
			uid:      "1.2.3.4.5",
			wantName: "",
		},
		{
			name:     "empty string",
			uid:      "",
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := Name(tt.uid)
			assert.Equal(t, tt.wantName, name)
		})
	}
}

func TestIsRetired(t *testing.T) {
	tests := []struct {
		name        string
		uid         string
		wantRetired bool
	}{
		{
			name:        "retired transfer syntax",
			uid:         "1.2.840.10008.1.2.2", // Explicit VR Big Endian
			wantRetired: true,
		},
		{
			name:        "active transfer syntax",
			uid:         "1.2.840.10008.1.2",
			wantRetired: false,
		},
		{
			name:        "active SOP class",
			uid:         "1.2.840.10008.5.1.4.1.1.2",
			wantRetired: false,
		},
		{
			name:        "unknown UID",
			uid:         "1.2.3.4.5",
			wantRetired: false,
		},
		{
			name:        "empty string",
			uid:         "",
			wantRetired: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retired := IsRetired(tt.uid)
			assert.Equal(t, tt.wantRetired, retired)
		})
	}
}

func TestGetType(t *testing.T) {
	tests := []struct {
		name     string
		uid      string
		wantType Type
	}{
		{
			name:     "transfer syntax",
			uid:      "1.2.840.10008.1.2",
			wantType: TypeTransferSyntax,
		},
		{
			name:     "SOP class",
			uid:      "1.2.840.10008.5.1.4.1.1.2",
			wantType: TypeSOPClass,
		},
		{
			name:     "coding scheme",
			uid:      "1.2.840.10008.2.16.4", // DICOM Controlled Terminology
			wantType: TypeCodingScheme,
		},
		{
			name:     "unknown UID",
			uid:      "1.2.3.4.5",
			wantType: "",
		},
		{
			name:     "empty string",
			uid:      "",
			wantType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uidType := GetType(tt.uid)
			assert.Equal(t, tt.wantType, uidType)
		})
	}
}

func TestIsTransferSyntax(t *testing.T) {
	tests := []struct {
		name           string
		uid            string
		wantTransferSx bool
	}{
		{
			name:           "implicit VR little endian",
			uid:            "1.2.840.10008.1.2",
			wantTransferSx: true,
		},
		{
			name:           "explicit VR little endian",
			uid:            "1.2.840.10008.1.2.1",
			wantTransferSx: true,
		},
		{
			name:           "JPEG baseline",
			uid:            "1.2.840.10008.1.2.4.50",
			wantTransferSx: true,
		},
		{
			name:           "RLE lossless",
			uid:            "1.2.840.10008.1.2.5",
			wantTransferSx: true,
		},
		{
			name:           "SOP class (not transfer syntax)",
			uid:            "1.2.840.10008.5.1.4.1.1.2",
			wantTransferSx: false,
		},
		{
			name:           "coding scheme (not transfer syntax)",
			uid:            "1.2.840.10008.2.16.4",
			wantTransferSx: false,
		},
		{
			name:           "unknown UID",
			uid:            "1.2.3.4.5",
			wantTransferSx: false,
		},
		{
			name:           "empty string",
			uid:            "",
			wantTransferSx: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isTS := IsTransferSyntax(tt.uid)
			assert.Equal(t, tt.wantTransferSx, isTS)
		})
	}
}

func TestIsSOPClass(t *testing.T) {
	tests := []struct {
		name         string
		uid          string
		wantSOPClass bool
	}{
		{
			name:         "CT image storage",
			uid:          "1.2.840.10008.5.1.4.1.1.2",
			wantSOPClass: true,
		},
		{
			name:         "MR image storage",
			uid:          "1.2.840.10008.5.1.4.1.1.4",
			wantSOPClass: true,
		},
		{
			name:         "verification SOP class",
			uid:          "1.2.840.10008.1.1",
			wantSOPClass: true,
		},
		{
			name:         "patient root QR find",
			uid:          "1.2.840.10008.5.1.4.1.2.1.1",
			wantSOPClass: true,
		},
		{
			name:         "transfer syntax (not SOP class)",
			uid:          "1.2.840.10008.1.2",
			wantSOPClass: false,
		},
		{
			name:         "coding scheme (not SOP class)",
			uid:          "1.2.840.10008.2.16.4",
			wantSOPClass: false,
		},
		{
			name:         "unknown UID",
			uid:          "1.2.3.4.5",
			wantSOPClass: false,
		},
		{
			name:         "empty string",
			uid:          "",
			wantSOPClass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isSOPC := IsSOPClass(tt.uid)
			assert.Equal(t, tt.wantSOPClass, isSOPC)
		})
	}
}

// TestUIDMapCompleteness verifies that all exported UID constants are in uidMap
func TestUIDMapCompleteness(t *testing.T) {
	exportedUIDs := []struct {
		name string
		uid  UID
	}{
		{"ImplicitVRLittleEndian", ImplicitVRLittleEndian},
		{"ExplicitVRLittleEndian", ExplicitVRLittleEndian},
		{"ExplicitVRBigEndian", ExplicitVRBigEndian},
		{"DeflatedExplicitVRLittleEndian", DeflatedExplicitVRLittleEndian},
		{"JPEGBaselineProcess1", JPEGBaselineProcess1},
		{"JPEGExtendedProcess2And4", JPEGExtendedProcess2And4},
		{"JPEGLosslessNonHierarchicalProcess14", JPEGLosslessNonHierarchicalProcess14},
		{"JPEGLosslessNonHierarchicalFirstOrderPredictionProcess14SelectionValue1", JPEGLosslessNonHierarchicalFirstOrderPredictionProcess14SelectionValue1},
		{"JPEGLsLosslessImageCompression", JPEGLsLosslessImageCompression},
		{"JPEGLsLossyNearLosslessImageCompression", JPEGLsLossyNearLosslessImageCompression},
		{"JPEG2000ImageCompressionLosslessOnly", JPEG2000ImageCompressionLosslessOnly},
		{"JPEG2000ImageCompression", JPEG2000ImageCompression},
		{"RLELossless", RLELossless},
		{"VerificationSOPClass", VerificationSOPClass},
		{"ComputedRadiographyImageStorage", ComputedRadiographyImageStorage},
		{"CTImageStorage", CTImageStorage},
		{"MRImageStorage", MRImageStorage},
		{"SecondaryCaptureImageStorage", SecondaryCaptureImageStorage},
		{"PatientRootQueryRetrieveInformationModelFind", PatientRootQueryRetrieveInformationModelFind},
		{"PatientRootQueryRetrieveInformationModelMove", PatientRootQueryRetrieveInformationModelMove},
		{"PatientRootQueryRetrieveInformationModelGet", PatientRootQueryRetrieveInformationModelGet},
		{"StudyRootQueryRetrieveInformationModelFind", StudyRootQueryRetrieveInformationModelFind},
		{"StudyRootQueryRetrieveInformationModelMove", StudyRootQueryRetrieveInformationModelMove},
		{"StudyRootQueryRetrieveInformationModelGet", StudyRootQueryRetrieveInformationModelGet},
		{"ModalityWorklistInformationModelFind", ModalityWorklistInformationModelFind},
	}

	for _, tt := range exportedUIDs {
		t.Run(tt.name, func(t *testing.T) {
			_, found := Lookup(tt.uid.String())
			assert.True(t, found, "exported UID %s not found in uidMap", tt.name)
		})
	}
}

// TestUIDMapStatistics verifies the basic statistics of the uidMap
func TestUIDMapStatistics(t *testing.T) {
	assert.Greater(t, len(uidMap), 400, "uidMap should contain at least 400 entries")

	var transferSyntaxCount, sopClassCount, retiredCount int

	for _, info := range uidMap {
		switch info.Type {
		case TypeTransferSyntax:
			transferSyntaxCount++
		case TypeSOPClass, TypeMetaSOPClass:
			sopClassCount++
		}
		if info.Retired {
			retiredCount++
		}
	}

	assert.Greater(t, transferSyntaxCount, 50, "should have at least 50 transfer syntaxes")
	assert.Greater(t, sopClassCount, 200, "should have at least 200 SOP classes")
	assert.Greater(t, retiredCount, 50, "should have at least 50 retired UIDs")
}

func TestFind(t *testing.T) {
	tests := []struct {
		name     string
		uid      string
		wantErr  bool
		wantInfo Info
	}{
		{
			name:    "valid transfer syntax",
			uid:     "1.2.840.10008.1.2",
			wantErr: false,
			wantInfo: Info{
				UID:     "1.2.840.10008.1.2",
				Name:    "Implicit VR Little Endian",
				Type:    TypeTransferSyntax,
				Info:    "Default Transfer Syntax for DICOM",
				Retired: false,
			},
		},
		{
			name:    "valid SOP class",
			uid:     "1.2.840.10008.5.1.4.1.1.2",
			wantErr: false,
			wantInfo: Info{
				UID:     "1.2.840.10008.5.1.4.1.1.2",
				Name:    "CT Image Storage",
				Type:    TypeSOPClass,
				Info:    "",
				Retired: false,
			},
		},
		{
			name:     "unknown UID",
			uid:      "1.2.3.4.5.6.7.8.9",
			wantErr:  true,
			wantInfo: Info{},
		},
		{
			name:     "empty string",
			uid:      "",
			wantErr:  true,
			wantInfo: Info{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := Find(tt.uid)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantInfo, info)
			}
		})
	}
}

func TestFindByName(t *testing.T) {
	tests := []struct {
		name     string
		uidName  string
		wantErr  bool
		wantUID  string
		wantType Type
	}{
		{
			name:     "transfer syntax",
			uidName:  "Implicit VR Little Endian",
			wantErr:  false,
			wantUID:  "1.2.840.10008.1.2",
			wantType: TypeTransferSyntax,
		},
		{
			name:     "SOP class",
			uidName:  "CT Image Storage",
			wantErr:  false,
			wantUID:  "1.2.840.10008.5.1.4.1.1.2",
			wantType: TypeSOPClass,
		},
		{
			name:     "verification SOP class",
			uidName:  "Verification SOP Class",
			wantErr:  false,
			wantUID:  "1.2.840.10008.1.1",
			wantType: TypeSOPClass,
		},
		{
			name:    "unknown name",
			uidName: "Nonexistent UID Name",
			wantErr: true,
		},
		{
			name:    "empty string",
			uidName: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := FindByName(tt.uidName)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantUID, info.UID)
				assert.Equal(t, tt.wantType, info.Type)
				assert.Equal(t, tt.uidName, info.Name)
			}
		})
	}
}

func TestFindAllByType(t *testing.T) {
	tests := []struct {
		name    string
		uidType Type
		wantMin int
	}{
		{
			name:    "transfer syntaxes",
			uidType: TypeTransferSyntax,
			wantMin: 50,
		},
		{
			name:    "SOP classes",
			uidType: TypeSOPClass,
			wantMin: 200,
		},
		{
			name:    "coding schemes",
			uidType: TypeCodingScheme,
			wantMin: 5,
		},
		{
			name:    "nonexistent type",
			uidType: Type("Nonexistent Type"),
			wantMin: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := FindAllByType(tt.uidType)
			assert.GreaterOrEqual(t, len(results), tt.wantMin,
				"expected at least %d UIDs of type %s, got %d", tt.wantMin, tt.uidType, len(results))

			// Verify all returned UIDs match the requested type
			for _, info := range results {
				assert.Equal(t, tt.uidType, info.Type,
					"UID %s has type %s, expected %s", info.UID, info.Type, tt.uidType)
			}
		})
	}
}
