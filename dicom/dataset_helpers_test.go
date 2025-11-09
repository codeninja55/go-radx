package dicom

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetPatientName tests setting patient name
func TestSetPatientName(t *testing.T) {
	ds := NewDataSet()

	// Test setting patient name
	err := ds.SetPatientName("Doe^John^A^^Dr.")
	require.NoError(t, err)

	// Verify it was set correctly
	elem, err := ds.Get(tag.PatientName)
	require.NoError(t, err)
	assert.Equal(t, "Doe^John^A^^Dr.", elem.Value().String())
	assert.Equal(t, vr.PersonName, elem.VR())

	// Test updating patient name
	err = ds.SetPatientName("Smith^Jane")
	require.NoError(t, err)

	elem, err = ds.Get(tag.PatientName)
	require.NoError(t, err)
	assert.Equal(t, "Smith^Jane", elem.Value().String())
}

// TestSetPatientID tests setting patient ID
func TestSetPatientID(t *testing.T) {
	ds := NewDataSet()

	// Test setting patient ID
	err := ds.SetPatientID("123456789")
	require.NoError(t, err)

	elem, err := ds.Get(tag.PatientID)
	require.NoError(t, err)
	assert.Equal(t, "123456789", elem.Value().String())
	assert.Equal(t, vr.LongString, elem.VR())

	// Test empty patient ID
	err = ds.SetPatientID("")
	require.NoError(t, err)

	elem, err = ds.Get(tag.PatientID)
	require.NoError(t, err)
	assert.Equal(t, "", elem.Value().String())
}

// TestSetPatientBirthDate tests setting patient birth date
func TestSetPatientBirthDate(t *testing.T) {
	tests := []struct {
		name      string
		date      string
		expectErr bool
	}{
		{"Valid date", "19800515", false},
		{"Empty date", "", false},
		{"Invalid format - too short", "1980", true},
		{"Invalid format - too long", "198005151234", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := NewDataSet()
			err := ds.SetPatientBirthDate(tt.date)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				elem, err := ds.Get(tag.PatientBirthDate)
				require.NoError(t, err)
				assert.Equal(t, tt.date, elem.Value().String())
				assert.Equal(t, vr.Date, elem.VR())
			}
		})
	}
}

// TestSetPatientAge tests setting patient age
func TestSetPatientAge(t *testing.T) {
	tests := []struct {
		name      string
		age       string
		expectErr bool
	}{
		{"Years", "045Y", false},
		{"Months", "006M", false},
		{"Weeks", "012W", false},
		{"Days", "030D", false},
		{"Empty", "", false},
		{"Invalid - too short", "45Y", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := NewDataSet()
			err := ds.SetPatientAge(tt.age)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.age != "" {
					elem, err := ds.Get(tag.PatientAge)
					require.NoError(t, err)
					assert.Equal(t, tt.age, elem.Value().String())
					assert.Equal(t, vr.AgeString, elem.VR())
				}
			}
		})
	}
}

// TestSetPatientSex tests setting patient sex
func TestSetPatientSex(t *testing.T) {
	tests := []struct {
		name      string
		sex       string
		expected  string
		expectErr bool
	}{
		{"Male", "M", "M", false},
		{"Female", "F", "F", false},
		{"Other", "O", "O", false},
		{"Unknown", "", "", false},
		{"Lowercase male", "m", "M", false},
		{"Invalid", "X", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := NewDataSet()
			err := ds.SetPatientSex(tt.sex)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				elem, err := ds.Get(tag.PatientSex)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, elem.Value().String())
			}
		})
	}
}

// TestSetAccessionNumber tests setting accession number
func TestSetAccessionNumber(t *testing.T) {
	ds := NewDataSet()

	err := ds.SetAccessionNumber("ACC123456")
	require.NoError(t, err)

	elem, err := ds.Get(tag.AccessionNumber)
	require.NoError(t, err)
	assert.Equal(t, "ACC123456", elem.Value().String())
	assert.Equal(t, vr.ShortString, elem.VR())
}

// TestSetStudyInstanceUID tests setting study instance UID
func TestSetStudyInstanceUID(t *testing.T) {
	ds := NewDataSet()

	// Test with explicit UID
	uid := "1.2.840.113619.2.55.3.604688119.123.1234567890.123"
	err := ds.SetStudyInstanceUID(uid)
	require.NoError(t, err)

	elem, err := ds.Get(tag.StudyInstanceUID)
	require.NoError(t, err)
	assert.Equal(t, uid, elem.Value().String())

	// Test with auto-generated UID
	err = ds.SetStudyInstanceUID("")
	require.NoError(t, err)

	elem, err = ds.Get(tag.StudyInstanceUID)
	require.NoError(t, err)
	assert.NotEmpty(t, elem.Value().String())
	assert.NotEqual(t, uid, elem.Value().String())

	// Test with invalid UID
	err = ds.SetStudyInstanceUID("invalid-uid")
	assert.Error(t, err)
}

// TestSetSeriesInstanceUID tests setting series instance UID
func TestSetSeriesInstanceUID(t *testing.T) {
	ds := NewDataSet()

	uid := "1.2.840.113619.2.55.3.604688119.456.1234567890.456"
	err := ds.SetSeriesInstanceUID(uid)
	require.NoError(t, err)

	elem, err := ds.Get(tag.SeriesInstanceUID)
	require.NoError(t, err)
	assert.Equal(t, uid, elem.Value().String())
}

// TestSetSOPInstanceUID tests setting SOP instance UID
func TestSetSOPInstanceUID(t *testing.T) {
	ds := NewDataSet()

	uid := "1.2.840.113619.2.55.3.604688119.789.1234567890.789"
	err := ds.SetSOPInstanceUID(uid)
	require.NoError(t, err)

	elem, err := ds.Get(tag.SOPInstanceUID)
	require.NoError(t, err)
	assert.Equal(t, uid, elem.Value().String())
}

// TestGenerateNewUIDs tests generating new UIDs for Study, Series, and SOP
func TestGenerateNewUIDs(t *testing.T) {
	ds := NewDataSet()

	// Set initial UIDs
	_ = ds.SetStudyInstanceUID("1.2.3.4.5")
	_ = ds.SetSeriesInstanceUID("1.2.3.4.6")
	_ = ds.SetSOPInstanceUID("1.2.3.4.7")

	// Generate new UIDs
	err := ds.GenerateNewUIDs()
	require.NoError(t, err)

	// Verify all UIDs were changed
	studyElem, _ := ds.Get(tag.StudyInstanceUID)
	seriesElem, _ := ds.Get(tag.SeriesInstanceUID)
	sopElem, _ := ds.Get(tag.SOPInstanceUID)

	assert.NotEqual(t, "1.2.3.4.5", studyElem.Value().String())
	assert.NotEqual(t, "1.2.3.4.6", seriesElem.Value().String())
	assert.NotEqual(t, "1.2.3.4.7", sopElem.Value().String())

	// Verify UIDs are valid (not empty)
	assert.NotEmpty(t, studyElem.Value().String())
	assert.NotEmpty(t, seriesElem.Value().String())
	assert.NotEmpty(t, sopElem.Value().String())

	// Verify all UIDs are different
	assert.NotEqual(t, studyElem.Value().String(), seriesElem.Value().String())
	assert.NotEqual(t, seriesElem.Value().String(), sopElem.Value().String())
}

// TestWalk tests iterating through dataset elements
func TestWalk(t *testing.T) {
	ds := NewDataSet()

	// Add multiple elements
	_ = ds.SetPatientName("Doe^John")
	_ = ds.SetPatientID("123456")
	_ = ds.SetAccessionNumber("ACC123")

	// Walk and count elements
	count := 0
	err := ds.Walk(func(elem *element.Element) error {
		count++
		assert.NotNil(t, elem)
		assert.NotNil(t, elem.Value())
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

// TestWalkWithError tests Walk error propagation
func TestWalkWithError(t *testing.T) {
	ds := NewDataSet()
	_ = ds.SetPatientName("Doe^John")
	_ = ds.SetPatientID("123456")

	// Walk with error on second element
	count := 0
	err := ds.Walk(func(elem *element.Element) error {
		count++
		if count == 2 {
			return assert.AnError
		}
		return nil
	})

	assert.Error(t, err)
	assert.Equal(t, 2, count)
}

// TestWalkModify tests modifying elements during iteration
func TestWalkModify(t *testing.T) {
	ds := NewDataSet()

	// Add elements
	_ = ds.SetPatientName("Doe^John")
	_ = ds.SetPatientID("123456")
	_ = ds.SetAccessionNumber("ACC123")

	// Modify all PersonName elements
	err := ds.WalkModify(func(elem *element.Element) (bool, error) {
		if elem.VR() == vr.PersonName {
			newVal, _ := value.NewStringValue(vr.PersonName, []string{"ANONYMOUS"})
			_ = elem.SetValue(newVal)
			return true, nil
		}
		return false, nil
	})

	require.NoError(t, err)

	// Verify modification
	elem, _ := ds.Get(tag.PatientName)
	assert.Equal(t, "ANONYMOUS", elem.Value().String())

	// Verify other elements unchanged
	elem, _ = ds.Get(tag.PatientID)
	assert.Equal(t, "123456", elem.Value().String())
}

// TestWalkModifyRemove tests removing elements during iteration
func TestWalkModifyRemove(t *testing.T) {
	ds := NewDataSet()

	// Add elements
	_ = ds.SetPatientName("Doe^John")
	_ = ds.SetPatientID("123456")
	_ = ds.SetAccessionNumber("ACC123")

	initialCount := len(ds.Elements())

	// Remove PatientID
	err := ds.WalkModify(func(elem *element.Element) (bool, error) {
		if elem.Tag() == tag.PatientID {
			return false, ErrRemoveElement
		}
		return false, nil
	})

	require.NoError(t, err)

	// Verify removal
	assert.False(t, ds.Contains(tag.PatientID))
	assert.Equal(t, initialCount-1, len(ds.Elements()))

	// Verify other elements still present
	assert.True(t, ds.Contains(tag.PatientName))
	assert.True(t, ds.Contains(tag.AccessionNumber))
}

// TestRemovePrivateTags tests removing private tags
func TestRemovePrivateTags(t *testing.T) {
	ds := NewDataSet()

	// Add public tags
	_ = ds.SetPatientName("Doe^John")
	_ = ds.SetPatientID("123456")

	// Add private tags (odd group numbers)
	privateTag1 := tag.New(0x0009, 0x0010)
	privateTag2 := tag.New(0x0011, 0x0020)

	val, _ := value.NewStringValue(vr.LongString, []string{"Private Data"})
	elem1, _ := element.NewElement(privateTag1, vr.LongString, val)
	elem2, _ := element.NewElement(privateTag2, vr.LongString, val)

	_ = ds.Add(elem1)
	_ = ds.Add(elem2)

	assert.Equal(t, 4, len(ds.Elements()))

	// Remove private tags
	err := ds.RemovePrivateTags()
	require.NoError(t, err)

	// Verify private tags removed
	assert.False(t, ds.Contains(privateTag1))
	assert.False(t, ds.Contains(privateTag2))

	// Verify public tags still present
	assert.True(t, ds.Contains(tag.PatientName))
	assert.True(t, ds.Contains(tag.PatientID))
	assert.Equal(t, 2, len(ds.Elements()))
}

// TestRemoveGroupTags tests removing specific tag groups
func TestRemoveGroupTags(t *testing.T) {
	ds := NewDataSet()

	// Add patient group (0x0010) tags
	_ = ds.SetPatientName("Doe^John")
	_ = ds.SetPatientID("123456")

	// Add study group (0x0020) tags
	_ = ds.SetStudyInstanceUID("1.2.3.4.5")

	assert.Equal(t, 3, len(ds.Elements()))

	// Remove patient group
	err := ds.RemoveGroupTags(0x0010)
	require.NoError(t, err)

	// Verify patient group tags removed
	assert.False(t, ds.Contains(tag.PatientName))
	assert.False(t, ds.Contains(tag.PatientID))

	// Verify study group tag still present
	assert.True(t, ds.Contains(tag.StudyInstanceUID))
	assert.Equal(t, 1, len(ds.Elements()))
}

// TestAnonymizeBasic tests basic anonymization
func TestAnonymizeBasic(t *testing.T) {
	ds := NewDataSet()

	// Add identifying information
	_ = ds.SetPatientName("Doe^John^Robert")
	_ = ds.SetPatientID("123456789")
	_ = ds.SetPatientBirthDate("19800515")
	_ = ds.SetPatientAge("043Y")
	_ = ds.SetStudyInstanceUID("1.2.3.4.5")

	// Add private tags
	privateTag := tag.New(0x0009, 0x0010)
	val, _ := value.NewStringValue(vr.LongString, []string{"Private Data"})
	elem, _ := element.NewElement(privateTag, vr.LongString, val)
	_ = ds.Add(elem)

	// Anonymize
	err := ds.AnonymizeBasic()
	require.NoError(t, err)

	// Verify anonymization
	nameElem, _ := ds.Get(tag.PatientName)
	assert.Equal(t, "ANONYMOUS", nameElem.Value().String())

	idElem, _ := ds.Get(tag.PatientID)
	assert.Equal(t, "ANON001", idElem.Value().String())

	birthDateElem, _ := ds.Get(tag.PatientBirthDate)
	assert.Equal(t, "", birthDateElem.Value().String())

	ageElem, _ := ds.Get(tag.PatientAge)
	assert.Equal(t, "", ageElem.Value().String())

	// Verify UID was regenerated
	uidElem, _ := ds.Get(tag.StudyInstanceUID)
	assert.NotEqual(t, "1.2.3.4.5", uidElem.Value().String())

	// Verify private tags removed
	assert.False(t, ds.Contains(privateTag))
}

// TestSetCurrentDateTime tests setting current date/time
func TestSetCurrentDateTime(t *testing.T) {
	ds := NewDataSet()

	err := ds.SetCurrentDateTime()
	require.NoError(t, err)

	// Verify Instance Creation Date and Time were set
	dateElem, err := ds.Get(tag.InstanceCreationDate)
	require.NoError(t, err)
	assert.NotEmpty(t, dateElem.Value().String())
	assert.Len(t, dateElem.Value().String(), 8) // YYYYMMDD

	timeElem, err := ds.Get(tag.InstanceCreationTime)
	require.NoError(t, err)
	assert.NotEmpty(t, timeElem.Value().String())
}

// TestSetStudyDate tests setting study date
func TestSetStudyDate(t *testing.T) {
	ds := NewDataSet()

	err := ds.SetStudyDate("20240315")
	require.NoError(t, err)

	elem, err := ds.Get(tag.StudyDate)
	require.NoError(t, err)
	assert.Equal(t, "20240315", elem.Value().String())

	// Test invalid format
	err = ds.SetStudyDate("2024")
	assert.Error(t, err)
}

// TestSetStudyTime tests setting study time
func TestSetStudyTime(t *testing.T) {
	ds := NewDataSet()

	tests := []struct {
		name string
		time string
	}{
		{"HHMMSS", "143025"},
		{"HHMMSS.ffffff", "143025.123456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ds.SetStudyTime(tt.time)
			require.NoError(t, err)

			elem, err := ds.Get(tag.StudyTime)
			require.NoError(t, err)
			assert.Equal(t, tt.time, elem.Value().String())
		})
	}
}

// TestSetSeriesNumber tests setting series number
func TestSetSeriesNumber(t *testing.T) {
	ds := NewDataSet()

	err := ds.SetSeriesNumber(5)
	require.NoError(t, err)

	elem, err := ds.Get(tag.SeriesNumber)
	require.NoError(t, err)

	// IntegerString (IS) VR stores integers as strings
	strVal, ok := elem.Value().(*value.StringValue)
	require.True(t, ok)
	strs := strVal.Strings()
	require.Len(t, strs, 1)
	assert.Equal(t, "5", strs[0])
}

// TestSetInstanceNumber tests setting instance number
func TestSetInstanceNumber(t *testing.T) {
	ds := NewDataSet()

	err := ds.SetInstanceNumber(42)
	require.NoError(t, err)

	elem, err := ds.Get(tag.InstanceNumber)
	require.NoError(t, err)

	// IntegerString (IS) VR stores integers as strings
	strVal, ok := elem.Value().(*value.StringValue)
	require.True(t, ok)
	strs := strVal.Strings()
	require.Len(t, strs, 1)
	assert.Equal(t, "42", strs[0])
}
