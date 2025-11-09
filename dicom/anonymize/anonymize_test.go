package anonymize

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewAnonymizer tests creating anonymizers with different profiles
func TestNewAnonymizer(t *testing.T) {
	profiles := []Profile{
		ProfileBasic,
		ProfileClean,
		ProfileRetainUIDs,
		ProfileRetainDeviceIdentity,
		ProfileCustom,
	}

	for _, profile := range profiles {
		t.Run(string(rune(profile)), func(t *testing.T) {
			anonymizer := NewAnonymizer(profile)
			assert.NotNil(t, anonymizer)
			assert.Equal(t, profile, anonymizer.config.Profile)
			assert.NotNil(t, anonymizer.actions)
		})
	}
}

// TestNewAnonymizerWithConfig tests creating anonymizer with custom config
func TestNewAnonymizerWithConfig(t *testing.T) {
	config := Config{
		Profile:     ProfileBasic,
		PatientName: "TEST_PATIENT",
		PatientID:   "TEST_ID",
		Options: Options{
			RetainUIDs:        true,
			RemovePrivateTags: false,
		},
	}

	anonymizer := NewAnonymizerWithConfig(config)
	assert.NotNil(t, anonymizer)
	assert.Equal(t, "TEST_PATIENT", anonymizer.config.PatientName)
	assert.Equal(t, "TEST_ID", anonymizer.config.PatientID)
	assert.True(t, anonymizer.config.Options.RetainUIDs)
	assert.False(t, anonymizer.config.Options.RemovePrivateTags)
}

// TestAnonymizeBasicProfile tests basic profile anonymization
func TestAnonymizeBasicProfile(t *testing.T) {
	ds := setupTestDataSet(t)
	anonymizer := NewAnonymizer(ProfileBasic)

	result, err := anonymizer.Anonymize(ds)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify patient name was anonymized
	nameElem, err := result.Get(tag.PatientName)
	require.NoError(t, err)
	assert.NotEqual(t, "Smith^John^Robert^^Dr.", nameElem.Value().String())
	assert.Equal(t, "ANONYMOUS", nameElem.Value().String())

	// Verify patient ID was anonymized
	idElem, err := result.Get(tag.PatientID)
	require.NoError(t, err)
	assert.NotEqual(t, "PAT123456789", idElem.Value().String())

	// Verify birth date was emptied
	birthElem, err := result.Get(tag.PatientBirthDate)
	require.NoError(t, err)
	assert.Equal(t, "", birthElem.Value().String())

	// Verify UIDs were regenerated
	studyUIDElem, err := result.Get(tag.StudyInstanceUID)
	require.NoError(t, err)
	assert.NotEqual(t, "1.2.840.113619.2.55.3.604688119.123.1234567890.123",
		studyUIDElem.Value().String())

	// Verify original dataset unchanged
	origNameElem, _ := ds.Get(tag.PatientName)
	assert.Equal(t, "Smith^John^Robert^^Dr.", origNameElem.Value().String())
}

// TestAnonymizeRetainUIDs tests UID retention
func TestAnonymizeRetainUIDs(t *testing.T) {
	ds := setupTestDataSet(t)

	config := Config{
		Profile:     ProfileBasic,
		PatientName: "ANONYMOUS",
		PatientID:   "ANON001",
		Options: Options{
			RetainUIDs: true,
		},
	}

	anonymizer := NewAnonymizerWithConfig(config)
	result, err := anonymizer.Anonymize(ds)
	require.NoError(t, err)

	// Verify UIDs were retained
	origStudyUID, _ := ds.Get(tag.StudyInstanceUID)
	newStudyUID, _ := result.Get(tag.StudyInstanceUID)
	assert.Equal(t, origStudyUID.Value().String(), newStudyUID.Value().String())

	origSeriesUID, _ := ds.Get(tag.SeriesInstanceUID)
	newSeriesUID, _ := result.Get(tag.SeriesInstanceUID)
	assert.Equal(t, origSeriesUID.Value().String(), newSeriesUID.Value().String())

	// Verify patient data was still anonymized
	nameElem, _ := result.Get(tag.PatientName)
	assert.Equal(t, "ANONYMOUS", nameElem.Value().String())
}

// TestAnonymizeRetainPatientCharacteristics tests retaining patient characteristics
func TestAnonymizeRetainPatientCharacteristics(t *testing.T) {
	ds := setupTestDataSet(t)

	config := Config{
		Profile:     ProfileBasic,
		PatientName: "ANONYMOUS",
		PatientID:   "ANON001",
		Options: Options{
			RetainPatientCharacteristics: true,
		},
	}

	anonymizer := NewAnonymizerWithConfig(config)
	result, err := anonymizer.Anonymize(ds)
	require.NoError(t, err)

	// Verify patient characteristics retained
	sexElem, err := result.Get(tag.PatientSex)
	require.NoError(t, err)
	assert.Equal(t, "M", sexElem.Value().String())

	ageElem, err := result.Get(tag.PatientAge)
	require.NoError(t, err)
	assert.Equal(t, "048Y", ageElem.Value().String())

	// Verify identifying info still anonymized
	nameElem, _ := result.Get(tag.PatientName)
	assert.Equal(t, "ANONYMOUS", nameElem.Value().String())
}

// TestAnonymizeRemovePrivateTags tests private tag removal
func TestAnonymizeRemovePrivateTags(t *testing.T) {
	ds := setupTestDataSet(t)

	// Add private tags
	privateTag := tag.New(0x0009, 0x0010)
	val, _ := value.NewStringValue(vr.LongString, []string{"Private vendor data"})
	elem, _ := element.NewElement(privateTag, vr.LongString, val)
	_ = ds.Add(elem)

	assert.True(t, ds.Contains(privateTag))

	anonymizer := NewAnonymizer(ProfileBasic) // Basic profile removes private tags
	result, err := anonymizer.Anonymize(ds)
	require.NoError(t, err)

	// Verify private tag removed
	assert.False(t, result.Contains(privateTag))
}

// TestAnonymizeCustomActions tests custom action mapping
func TestAnonymizeCustomActions(t *testing.T) {
	ds := setupTestDataSet(t)

	customActions := map[tag.Tag]Action{
		tag.PatientName:      ActionHash,
		tag.PatientID:        ActionHash,
		tag.PatientBirthDate: ActionRemove,
	}

	config := Config{
		Profile:       ProfileCustom,
		CustomActions: customActions,
	}

	anonymizer := NewAnonymizerWithConfig(config)
	result, err := anonymizer.Anonymize(ds)
	require.NoError(t, err)

	// Verify patient name was hashed (starts with HASH_)
	nameElem, err := result.Get(tag.PatientName)
	require.NoError(t, err)
	assert.Contains(t, nameElem.Value().String(), "HASH_")

	// Verify patient ID was hashed
	idElem, err := result.Get(tag.PatientID)
	require.NoError(t, err)
	assert.Contains(t, idElem.Value().String(), "HASH_")

	// Verify birth date was removed
	assert.False(t, result.Contains(tag.PatientBirthDate))
}

// TestActionKeep tests the Keep action
func TestActionKeep(t *testing.T) {
	ds := setupTestDataSet(t)

	customActions := map[tag.Tag]Action{
		tag.PatientName: ActionKeep,
	}

	config := Config{
		Profile:       ProfileCustom,
		CustomActions: customActions,
	}

	anonymizer := NewAnonymizerWithConfig(config)
	result, err := anonymizer.Anonymize(ds)
	require.NoError(t, err)

	// Verify patient name unchanged
	origName, _ := ds.Get(tag.PatientName)
	newName, _ := result.Get(tag.PatientName)
	assert.Equal(t, origName.Value().String(), newName.Value().String())
}

// TestActionRemove tests the Remove action
func TestActionRemove(t *testing.T) {
	ds := setupTestDataSet(t)

	customActions := map[tag.Tag]Action{
		tag.PatientName: ActionRemove,
		tag.PatientID:   ActionRemove,
	}

	config := Config{
		Profile:       ProfileCustom,
		CustomActions: customActions,
	}

	anonymizer := NewAnonymizerWithConfig(config)
	result, err := anonymizer.Anonymize(ds)
	require.NoError(t, err)

	// Verify tags removed
	assert.False(t, result.Contains(tag.PatientName))
	assert.False(t, result.Contains(tag.PatientID))
}

// TestActionEmpty tests the Empty action
func TestActionEmpty(t *testing.T) {
	ds := setupTestDataSet(t)

	customActions := map[tag.Tag]Action{
		tag.PatientName:      ActionEmpty,
		tag.PatientBirthDate: ActionEmpty,
	}

	config := Config{
		Profile:       ProfileCustom,
		CustomActions: customActions,
	}

	anonymizer := NewAnonymizerWithConfig(config)
	result, err := anonymizer.Anonymize(ds)
	require.NoError(t, err)

	// Verify values are empty
	nameElem, err := result.Get(tag.PatientName)
	require.NoError(t, err)
	assert.Equal(t, "", nameElem.Value().String())

	birthElem, err := result.Get(tag.PatientBirthDate)
	require.NoError(t, err)
	assert.Equal(t, "", birthElem.Value().String())
}

// TestActionDummy tests the Dummy action
func TestActionDummy(t *testing.T) {
	ds := setupTestDataSet(t)

	config := Config{
		Profile:     ProfileBasic,
		PatientName: "DUMMY_NAME",
		PatientID:   "DUMMY_ID",
	}

	anonymizer := NewAnonymizerWithConfig(config)
	result, err := anonymizer.Anonymize(ds)
	require.NoError(t, err)

	// Verify dummy values were used
	nameElem, err := result.Get(tag.PatientName)
	require.NoError(t, err)
	assert.Equal(t, "DUMMY_NAME", nameElem.Value().String())

	idElem, err := result.Get(tag.PatientID)
	require.NoError(t, err)
	assert.Equal(t, "DUMMY_ID", idElem.Value().String())
}

// TestActionUID tests UID generation
func TestActionUID(t *testing.T) {
	ds := setupTestDataSet(t)

	customActions := map[tag.Tag]Action{
		tag.StudyInstanceUID:  ActionUID,
		tag.SeriesInstanceUID: ActionUID,
		tag.SOPInstanceUID:    ActionUID,
	}

	config := Config{
		Profile:       ProfileCustom,
		CustomActions: customActions,
	}

	anonymizer := NewAnonymizerWithConfig(config)
	result, err := anonymizer.Anonymize(ds)
	require.NoError(t, err)

	// Verify UIDs were regenerated
	origStudy, _ := ds.Get(tag.StudyInstanceUID)
	newStudy, _ := result.Get(tag.StudyInstanceUID)
	assert.NotEqual(t, origStudy.Value().String(), newStudy.Value().String())
	assert.NotEmpty(t, newStudy.Value().String())

	origSeries, _ := ds.Get(tag.SeriesInstanceUID)
	newSeries, _ := result.Get(tag.SeriesInstanceUID)
	assert.NotEqual(t, origSeries.Value().String(), newSeries.Value().String())
	assert.NotEmpty(t, newSeries.Value().String())

	// Verify UIDs are unique
	assert.NotEqual(t, newStudy.Value().String(), newSeries.Value().String())
}

// TestRemoveOverlays tests overlay removal
func TestRemoveOverlays(t *testing.T) {
	ds := setupTestDataSet(t)

	// Add overlay tag (group 0x6000)
	overlayTag := tag.New(0x6000, 0x0010)
	val, _ := value.NewIntValue(vr.UnsignedShort, []int64{512})
	elem, _ := element.NewElement(overlayTag, vr.UnsignedShort, val)
	_ = ds.Add(elem)

	assert.True(t, ds.Contains(overlayTag))

	config := Config{
		Profile: ProfileBasic,
		Options: Options{
			RemoveOverlays: true,
		},
	}

	anonymizer := NewAnonymizerWithConfig(config)
	result, err := anonymizer.Anonymize(ds)
	require.NoError(t, err)

	// Verify overlay removed
	assert.False(t, result.Contains(overlayTag))
}

// TestRemoveCurves tests curve data removal
func TestRemoveCurves(t *testing.T) {
	ds := setupTestDataSet(t)

	// Add curve tag (group 0x5000)
	curveTag := tag.New(0x5000, 0x0010)
	val, _ := value.NewIntValue(vr.UnsignedShort, []int64{256})
	elem, _ := element.NewElement(curveTag, vr.UnsignedShort, val)
	_ = ds.Add(elem)

	assert.True(t, ds.Contains(curveTag))

	config := Config{
		Profile: ProfileBasic,
		Options: Options{
			RemoveCurves: true,
		},
	}

	anonymizer := NewAnonymizerWithConfig(config)
	result, err := anonymizer.Anonymize(ds)
	require.NoError(t, err)

	// Verify curve removed
	assert.False(t, result.Contains(curveTag))
}

// TestAnonymizePreservesOriginal tests that original dataset is not modified
func TestAnonymizePreservesOriginal(t *testing.T) {
	ds := setupTestDataSet(t)

	// Get original values
	origName, _ := ds.Get(tag.PatientName)
	origID, _ := ds.Get(tag.PatientID)
	origNameStr := origName.Value().String()
	origIDStr := origID.Value().String()

	anonymizer := NewAnonymizer(ProfileBasic)
	result, err := anonymizer.Anonymize(ds)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify original dataset unchanged
	currentName, _ := ds.Get(tag.PatientName)
	currentID, _ := ds.Get(tag.PatientID)
	assert.Equal(t, origNameStr, currentName.Value().String())
	assert.Equal(t, origIDStr, currentID.Value().String())

	// Verify result is different
	resultName, _ := result.Get(tag.PatientName)
	assert.NotEqual(t, origNameStr, resultName.Value().String())
}

// TestCleanText tests text cleaning
func TestCleanText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Phone number", "Contact phone: 555-1234", "Contact REMOVED 555-1234"},
		{"Email", "contact@hospital.com", "CLEANED_TEXT"},
		{"Normal text", "This is normal text", "This is normal text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanText(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestHashString tests string hashing
func TestHashString(t *testing.T) {
	// Same input should produce same hash
	hash1 := hashString("test")
	hash2 := hashString("test")
	assert.Equal(t, hash1, hash2)

	// Different inputs should produce different hashes
	hash3 := hashString("different")
	assert.NotEqual(t, hash1, hash3)
}

// Helper functions

func setupTestDataSet(t *testing.T) *dicom.DataSet {
	t.Helper()
	ds := dicom.NewDataSet()

	// Patient identification
	_ = ds.SetPatientName("Smith^John^Robert^^Dr.")
	_ = ds.SetPatientID("PAT123456789")
	_ = ds.SetPatientBirthDate("19750315")
	_ = ds.SetPatientSex("M")
	_ = ds.SetPatientAge("048Y")

	// Institution
	val, _ := value.NewStringValue(vr.LongString, []string{"General Hospital"})
	elem, _ := element.NewElement(tag.InstitutionName, vr.LongString, val)
	_ = ds.Add(elem)

	// UIDs
	_ = ds.SetStudyInstanceUID("1.2.840.113619.2.55.3.604688119.123.1234567890.123")
	_ = ds.SetSeriesInstanceUID("1.2.840.113619.2.55.3.604688119.456.1234567890.456")
	_ = ds.SetSOPInstanceUID("1.2.840.113619.2.55.3.604688119.789.1234567890.789")

	return ds
}
