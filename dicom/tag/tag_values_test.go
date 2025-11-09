package tag_test

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/vr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagDict_LookupCommonTags(t *testing.T) {
	tests := []struct {
		name            string
		tagVar          tag.Tag
		expectedKeyword string
		expectedName    string
		expectedVM      string
		expectedRetired bool
	}{
		{
			name:            "PixelData",
			tagVar:          tag.PixelData,
			expectedKeyword: "PixelData",
			expectedName:    "Pixel Data",
			expectedVM:      "1",
			expectedRetired: false,
		},
		{
			name:            "PatientName",
			tagVar:          tag.PatientName,
			expectedKeyword: "PatientName",
			expectedName:    "Patient's Name",
			expectedVM:      "1",
			expectedRetired: false,
		},
		{
			name:            "StudyInstanceUID",
			tagVar:          tag.StudyInstanceUID,
			expectedKeyword: "StudyInstanceUID",
			expectedName:    "Study Instance UID",
			expectedVM:      "1",
			expectedRetired: false,
		},
		{
			name:            "Modality",
			tagVar:          tag.Modality,
			expectedKeyword: "Modality",
			expectedName:    "Modality",
			expectedVM:      "1",
			expectedRetired: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			info, ok := tag.TagDict[tc.tagVar]
			require.True(t, ok, "Tag should exist in TagDict")
			assert.Equal(t, tc.expectedKeyword, info.Keyword)
			assert.Equal(t, tc.expectedName, info.Name)
			assert.Equal(t, tc.expectedVM, info.VM)
			assert.Equal(t, tc.expectedRetired, info.Retired)
			assert.NotEmpty(t, info.VRs, "VRs should not be empty")
		})
	}
}

func TestTagDict_VRTypes(t *testing.T) {
	tests := []struct {
		name        string
		tagVar      tag.Tag
		expectedVRs []vr.VR
	}{
		{
			name:        "PixelData has OB or OW",
			tagVar:      tag.PixelData,
			expectedVRs: []vr.VR{vr.OtherByte, vr.OtherWord},
		},
		{
			name:        "PatientName has PN",
			tagVar:      tag.PatientName,
			expectedVRs: []vr.VR{vr.PersonName},
		},
		{
			name:        "Rows has US",
			tagVar:      tag.Rows,
			expectedVRs: []vr.VR{vr.UnsignedShort},
		},
		{
			name:        "StudyDate has DA",
			tagVar:      tag.StudyDate,
			expectedVRs: []vr.VR{vr.Date},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			info, ok := tag.TagDict[tc.tagVar]
			require.True(t, ok, "Tag should exist in TagDict")
			assert.Equal(t, tc.expectedVRs, info.VRs)
		})
	}
}

func TestTagDict_FileMetaInformation(t *testing.T) {
	tests := []struct {
		name   string
		tagVar tag.Tag
	}{
		{"FileMetaInformationGroupLength", tag.FileMetaInformationGroupLength},
		{"FileMetaInformationVersion", tag.FileMetaInformationVersion},
		{"MediaStorageSOPClassUID", tag.MediaStorageSOPClassUID},
		{"MediaStorageSOPInstanceUID", tag.MediaStorageSOPInstanceUID},
		{"TransferSyntaxUID", tag.TransferSyntaxUID},
		{"ImplementationClassUID", tag.ImplementationClassUID},
		{"ImplementationVersionName", tag.ImplementationVersionName},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := tag.TagDict[tc.tagVar]
			assert.True(t, ok, "Tag should exist in TagDict")
			assert.True(t, tc.tagVar.IsMetaElement(), "Tag should be a meta element")
		})
	}
}

func TestTagDict_ComprehensiveCoverage(t *testing.T) {
	// Verify we have a substantial number of tags
	assert.Greater(t, len(tag.TagDict), 5000, "Should have over 5000 DICOM tags")

	// Verify all entries have required fields
	for tagKey, info := range tag.TagDict {
		assert.True(t, tagKey.Equals(info.Tag), "TagDict key should match TagInfo.Tag")
		assert.NotEmpty(t, info.Name, "Name should not be empty")
		assert.NotEmpty(t, info.Keyword, "Keyword should not be empty")
		assert.NotEmpty(t, info.VM, "ValueMultiplicity should not be empty")
		assert.NotEmpty(t, info.VRs, "VRs should not be empty")
	}
}
