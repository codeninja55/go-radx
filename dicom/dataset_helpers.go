// Package dicom provides helper methods for common DICOM dataset operations.
package dicom

import (
	"fmt"
	"strings"
	"time"

	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/uid"
	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
)

// SetPatientName sets the Patient's Name (0010,0010) in the dataset.
//
// Example:
//
//	ds.SetPatientName("Doe^John^A^^Dr.")
func (ds *DataSet) SetPatientName(name string) error {
	val, err := value.NewStringValue(vr.PersonName, []string{name})
	if err != nil {
		return fmt.Errorf("failed to create PatientName value: %w", err)
	}
	elem, err := element.NewElement(tag.PatientName, vr.PersonName, val)
	if err != nil {
		return fmt.Errorf("failed to create PatientName element: %w", err)
	}
	return ds.Add(elem)
}

// SetPatientID sets the Patient ID (0010,0020) in the dataset.
//
// Example:
//
//	ds.SetPatientID("123456789")
func (ds *DataSet) SetPatientID(id string) error {
	val, err := value.NewStringValue(vr.LongString, []string{id})
	if err != nil {
		return fmt.Errorf("failed to create PatientID value: %w", err)
	}
	elem, err := element.NewElement(tag.PatientID, vr.LongString, val)
	if err != nil {
		return fmt.Errorf("failed to create PatientID element: %w", err)
	}
	return ds.Add(elem)
}

// SetPatientBirthDate sets the Patient's Birth Date (0010,0030) in the dataset.
//
// The date should be in YYYYMMDD format.
//
// Example:
//
//	ds.SetPatientBirthDate("19800515")
func (ds *DataSet) SetPatientBirthDate(date string) error {
	// Validate date format (basic check)
	if len(date) != 8 && date != "" {
		return fmt.Errorf("birth date must be in YYYYMMDD format or empty, got: %s", date)
	}

	val, err := value.NewStringValue(vr.Date, []string{date})
	if err != nil {
		return fmt.Errorf("failed to create PatientBirthDate value: %w", err)
	}
	elem, err := element.NewElement(tag.PatientBirthDate, vr.Date, val)
	if err != nil {
		return fmt.Errorf("failed to create PatientBirthDate element: %w", err)
	}
	return ds.Add(elem)
}

// SetPatientAge sets the Patient's Age (0010,1010) in the dataset.
//
// Age format: nnnD, nnnW, nnnM, or nnnY (Days, Weeks, Months, Years)
//
// Example:
//
//	ds.SetPatientAge("045Y")
func (ds *DataSet) SetPatientAge(age string) error {
	// Basic validation
	if age != "" && len(age) < 4 {
		return fmt.Errorf("age must be in format nnnD/W/M/Y, got: %s", age)
	}

	val, err := value.NewStringValue(vr.AgeString, []string{age})
	if err != nil {
		return fmt.Errorf("failed to create PatientAge value: %w", err)
	}
	elem, err := element.NewElement(tag.PatientAge, vr.AgeString, val)
	if err != nil {
		return fmt.Errorf("failed to create PatientAge element: %w", err)
	}
	return ds.Add(elem)
}

// SetPatientSex sets the Patient's Sex (0010,0040) in the dataset.
//
// Valid values: "M" (Male), "F" (Female), "O" (Other), or "" (Unknown)
//
// Example:
//
//	ds.SetPatientSex("M")
func (ds *DataSet) SetPatientSex(sex string) error {
	// Validate sex value
	sex = strings.ToUpper(sex)
	if sex != "" && sex != "M" && sex != "F" && sex != "O" {
		return fmt.Errorf("sex must be M, F, O, or empty, got: %s", sex)
	}

	val, err := value.NewStringValue(vr.CodeString, []string{sex})
	if err != nil {
		return fmt.Errorf("failed to create PatientSex value: %w", err)
	}
	elem, err := element.NewElement(tag.PatientSex, vr.CodeString, val)
	if err != nil {
		return fmt.Errorf("failed to create PatientSex element: %w", err)
	}
	return ds.Add(elem)
}

// SetAccessionNumber sets the Accession Number (0008,0050) in the dataset.
//
// Example:
//
//	ds.SetAccessionNumber("ACC123456")
func (ds *DataSet) SetAccessionNumber(number string) error {
	val, err := value.NewStringValue(vr.ShortString, []string{number})
	if err != nil {
		return fmt.Errorf("failed to create AccessionNumber value: %w", err)
	}
	elem, err := element.NewElement(tag.AccessionNumber, vr.ShortString, val)
	if err != nil {
		return fmt.Errorf("failed to create AccessionNumber element: %w", err)
	}
	return ds.Add(elem)
}

// SetStudyInstanceUID sets the Study Instance UID (0020,000D) in the dataset.
//
// If an empty string is provided, a new UID will be generated.
//
// Example:
//
//	ds.SetStudyInstanceUID("1.2.840.113619.2.55.3.604688119.123.1234567890.123")
//	ds.SetStudyInstanceUID("") // Generates new UID
func (ds *DataSet) SetStudyInstanceUID(uidStr string) error {
	if uidStr == "" {
		uidStr = uid.Generate()
	}

	// Validate UID format
	if !uid.IsValid(uidStr) {
		return fmt.Errorf("invalid UID format: %s", uidStr)
	}

	val, err := value.NewStringValue(vr.UniqueIdentifier, []string{uidStr})
	if err != nil {
		return fmt.Errorf("failed to create StudyInstanceUID value: %w", err)
	}
	elem, err := element.NewElement(tag.StudyInstanceUID, vr.UniqueIdentifier, val)
	if err != nil {
		return fmt.Errorf("failed to create StudyInstanceUID element: %w", err)
	}
	return ds.Add(elem)
}

// SetSeriesInstanceUID sets the Series Instance UID (0020,000E) in the dataset.
//
// If an empty string is provided, a new UID will be generated.
//
// Example:
//
//	ds.SetSeriesInstanceUID("1.2.840.113619.2.55.3.604688119.456.1234567890.456")
//	ds.SetSeriesInstanceUID("") // Generates new UID
func (ds *DataSet) SetSeriesInstanceUID(uidStr string) error {
	if uidStr == "" {
		uidStr = uid.Generate()
	}

	// Validate UID format
	if !uid.IsValid(uidStr) {
		return fmt.Errorf("invalid UID format: %s", uidStr)
	}

	val, err := value.NewStringValue(vr.UniqueIdentifier, []string{uidStr})
	if err != nil {
		return fmt.Errorf("failed to create SeriesInstanceUID value: %w", err)
	}
	elem, err := element.NewElement(tag.SeriesInstanceUID, vr.UniqueIdentifier, val)
	if err != nil {
		return fmt.Errorf("failed to create SeriesInstanceUID element: %w", err)
	}
	return ds.Add(elem)
}

// SetSOPInstanceUID sets the SOP Instance UID (0008,0018) in the dataset.
//
// If an empty string is provided, a new UID will be generated.
//
// Example:
//
//	ds.SetSOPInstanceUID("1.2.840.113619.2.55.3.604688119.789.1234567890.789")
//	ds.SetSOPInstanceUID("") // Generates new UID
func (ds *DataSet) SetSOPInstanceUID(uidStr string) error {
	if uidStr == "" {
		uidStr = uid.Generate()
	}

	// Validate UID format
	if !uid.IsValid(uidStr) {
		return fmt.Errorf("invalid UID format: %s", uidStr)
	}

	val, err := value.NewStringValue(vr.UniqueIdentifier, []string{uidStr})
	if err != nil {
		return fmt.Errorf("failed to create SOPInstanceUID value: %w", err)
	}
	elem, err := element.NewElement(tag.SOPInstanceUID, vr.UniqueIdentifier, val)
	if err != nil {
		return fmt.Errorf("failed to create SOPInstanceUID element: %w", err)
	}
	return ds.Add(elem)
}

// GenerateNewUIDs generates new UIDs for Study, Series, and SOP Instance UIDs.
//
// This is useful for creating anonymized copies or new instances.
//
// Example:
//
//	if err := ds.GenerateNewUIDs(); err != nil {
//	    log.Fatal(err)
//	}
func (ds *DataSet) GenerateNewUIDs() error {
	// Generate new Study Instance UID
	if err := ds.SetStudyInstanceUID(""); err != nil {
		return fmt.Errorf("failed to generate Study Instance UID: %w", err)
	}

	// Generate new Series Instance UID
	if err := ds.SetSeriesInstanceUID(""); err != nil {
		return fmt.Errorf("failed to generate Series Instance UID: %w", err)
	}

	// Generate new SOP Instance UID
	if err := ds.SetSOPInstanceUID(""); err != nil {
		return fmt.Errorf("failed to generate SOP Instance UID: %w", err)
	}

	// Also update Media Storage SOP Instance UID if present in File Meta Information
	if ds.Contains(tag.MediaStorageSOPInstanceUID) {
		sopElem, _ := ds.Get(tag.SOPInstanceUID)
		sopUID := sopElem.Value().String()

		val, err := value.NewStringValue(vr.UniqueIdentifier, []string{sopUID})
		if err != nil {
			return fmt.Errorf("failed to create Media Storage SOP Instance UID value: %w", err)
		}
		elem, err := element.NewElement(tag.MediaStorageSOPInstanceUID, vr.UniqueIdentifier, val)
		if err != nil {
			return fmt.Errorf("failed to update Media Storage SOP Instance UID: %w", err)
		}
		if err := ds.Add(elem); err != nil {
			return fmt.Errorf("failed to add Media Storage SOP Instance UID: %w", err)
		}
	}

	return nil
}

// SetStudyDate sets the Study Date (0008,0020) in the dataset.
//
// The date should be in YYYYMMDD format.
//
// Example:
//
//	ds.SetStudyDate("20240315")
func (ds *DataSet) SetStudyDate(date string) error {
	// Validate date format (basic check)
	if len(date) != 8 && date != "" {
		return fmt.Errorf("study date must be in YYYYMMDD format or empty, got: %s", date)
	}

	val, err := value.NewStringValue(vr.Date, []string{date})
	if err != nil {
		return fmt.Errorf("failed to create StudyDate value: %w", err)
	}
	elem, err := element.NewElement(tag.StudyDate, vr.Date, val)
	if err != nil {
		return fmt.Errorf("failed to create StudyDate element: %w", err)
	}
	return ds.Add(elem)
}

// SetStudyTime sets the Study Time (0008,0030) in the dataset.
//
// The time should be in HHMMSS.FFFFFF format (fractional seconds optional).
//
// Example:
//
//	ds.SetStudyTime("143025")       // 14:30:25
//	ds.SetStudyTime("143025.123456") // 14:30:25.123456
func (ds *DataSet) SetStudyTime(timeStr string) error {
	val, err := value.NewStringValue(vr.Time, []string{timeStr})
	if err != nil {
		return fmt.Errorf("failed to create StudyTime value: %w", err)
	}
	elem, err := element.NewElement(tag.StudyTime, vr.Time, val)
	if err != nil {
		return fmt.Errorf("failed to create StudyTime element: %w", err)
	}
	return ds.Add(elem)
}

// SetSeriesNumber sets the Series Number (0020,0011) in the dataset.
//
// Example:
//
//	ds.SetSeriesNumber(1)
func (ds *DataSet) SetSeriesNumber(number int) error {
	val, err := value.NewStringValue(vr.IntegerString, []string{fmt.Sprintf("%d", number)})
	if err != nil {
		return fmt.Errorf("failed to create SeriesNumber value: %w", err)
	}
	elem, err := element.NewElement(tag.SeriesNumber, vr.IntegerString, val)
	if err != nil {
		return fmt.Errorf("failed to create SeriesNumber element: %w", err)
	}
	return ds.Add(elem)
}

// SetInstanceNumber sets the Instance Number (0020,0013) in the dataset.
//
// Example:
//
//	ds.SetInstanceNumber(1)
func (ds *DataSet) SetInstanceNumber(number int) error {
	val, err := value.NewStringValue(vr.IntegerString, []string{fmt.Sprintf("%d", number)})
	if err != nil {
		return fmt.Errorf("failed to create InstanceNumber value: %w", err)
	}
	elem, err := element.NewElement(tag.InstanceNumber, vr.IntegerString, val)
	if err != nil {
		return fmt.Errorf("failed to create InstanceNumber element: %w", err)
	}
	return ds.Add(elem)
}

// Walk iterates through all elements in the dataset, calling fn for each element.
//
// The function fn should return an error to stop iteration.
// If fn returns nil, iteration continues.
//
// Example:
//
//	ds.Walk(func(elem *element.Element) error {
//	    fmt.Printf("%s = %s\n", elem.Tag(), elem.Value())
//	    return nil
//	})
func (ds *DataSet) Walk(fn func(*element.Element) error) error {
	for _, elem := range ds.Elements() {
		if err := fn(elem); err != nil {
			return err
		}
	}
	return nil
}

// WalkFunc is a function type for walking through dataset elements.
//
// Return true to modify the element, false to keep it unchanged.
type WalkFunc func(elem *element.Element) (modified bool, err error)

// WalkModify iterates through all elements, allowing modification or removal.
//
// The function fn should return:
//   - modified=true, err=nil: Element was modified in place
//   - modified=false, err=nil: Keep element unchanged
//   - modified=false, err=ErrRemoveElement: Remove the element
//   - any other error: Stop iteration and return error
//
// Example:
//
//	ds.WalkModify(func(elem *element.Element) (bool, error) {
//	    if elem.VR() == vr.PersonName {
//	        // Anonymize person names
//	        newVal := value.NewStringValue(vr.PersonName, []string{"ANONYMOUS"})
//	        elem.SetValue(newVal)
//	        return true, nil
//	    }
//	    return false, nil
//	})
var ErrRemoveElement = fmt.Errorf("remove element")

func (ds *DataSet) WalkModify(fn WalkFunc) error {
	toRemove := []tag.Tag{}

	for t, elem := range ds.elements {
		modified, err := fn(elem)
		if err != nil {
			if err == ErrRemoveElement {
				toRemove = append(toRemove, t)
				continue
			}
			return err
		}
		if modified {
			// Element was modified in place
			ds.elements[t] = elem
		}
	}

	// Remove marked elements
	for _, t := range toRemove {
		delete(ds.elements, t)
	}

	return nil
}

// RemovePrivateTags removes all private tags from the dataset.
//
// Private tags are those with odd group numbers.
//
// Example:
//
//	if err := ds.RemovePrivateTags(); err != nil {
//	    log.Fatal(err)
//	}
func (ds *DataSet) RemovePrivateTags() error {
	toRemove := []tag.Tag{}

	for t := range ds.elements {
		// Private tags have odd group numbers
		if t.Group%2 == 1 {
			toRemove = append(toRemove, t)
		}
	}

	for _, t := range toRemove {
		delete(ds.elements, t)
	}

	return nil
}

// RemoveGroupTags removes all tags from a specific group.
//
// This is useful for removing entire groups like overlays (0x6000-0x60FF).
//
// Example:
//
//	// Remove curve data (group 0x5000)
//	if err := ds.RemoveGroupTags(0x5000); err != nil {
//	    log.Fatal(err)
//	}
func (ds *DataSet) RemoveGroupTags(group uint16) error {
	toRemove := []tag.Tag{}

	for t := range ds.elements {
		// Check if tag belongs to the specified group
		// For repeating groups (0x5000-0x50FF, 0x6000-0x60FF), check range
		if isInGroup(t.Group, group) {
			toRemove = append(toRemove, t)
		}
	}

	for _, t := range toRemove {
		delete(ds.elements, t)
	}

	return nil
}

// isInGroup checks if a tag group belongs to a logical group.
//
// This handles repeating groups like curves (0x5000-0x50FF) and overlays (0x6000-0x60FF).
func isInGroup(tagGroup, targetGroup uint16) bool {
	if tagGroup == targetGroup {
		return true
	}

	// Handle repeating groups
	// Curves: 0x5000-0x50FF
	if targetGroup == 0x5000 && tagGroup >= 0x5000 && tagGroup <= 0x50FF {
		return true
	}

	// Overlays: 0x6000-0x60FF
	if targetGroup == 0x6000 && tagGroup >= 0x6000 && tagGroup <= 0x60FF {
		return true
	}

	return false
}

// SetCurrentDateTime sets the current date and time in relevant DICOM tags.
//
// Updates:
//   - Instance Creation Date (0008,0012)
//   - Instance Creation Time (0008,0013)
//   - Content Date (0008,0023) - if exists
//   - Content Time (0008,0033) - if exists
//
// Example:
//
//	ds.SetCurrentDateTime()
func (ds *DataSet) SetCurrentDateTime() error {
	now := time.Now()

	// Format date as YYYYMMDD
	dateStr := now.Format("20060102")

	// Format time as HHMMSS.ffffff
	timeStr := now.Format("150405.000000")

	// Instance Creation Date (0008,0012)
	dateVal, err := value.NewStringValue(vr.Date, []string{dateStr})
	if err != nil {
		return fmt.Errorf("failed to create InstanceCreationDate value: %w", err)
	}
	dateElem, err := element.NewElement(tag.InstanceCreationDate, vr.Date, dateVal)
	if err != nil {
		return fmt.Errorf("failed to create InstanceCreationDate: %w", err)
	}
	if err := ds.Add(dateElem); err != nil {
		return err
	}

	// Instance Creation Time (0008,0013)
	timeVal, err := value.NewStringValue(vr.Time, []string{timeStr})
	if err != nil {
		return fmt.Errorf("failed to create InstanceCreationTime value: %w", err)
	}
	timeElem, err := element.NewElement(tag.InstanceCreationTime, vr.Time, timeVal)
	if err != nil {
		return fmt.Errorf("failed to create InstanceCreationTime: %w", err)
	}
	if err := ds.Add(timeElem); err != nil {
		return err
	}

	// Update Content Date if it exists
	if ds.Contains(tag.ContentDate) {
		contentDateElem, err := element.NewElement(tag.ContentDate, vr.Date, dateVal)
		if err != nil {
			return fmt.Errorf("failed to create ContentDate: %w", err)
		}
		if err := ds.Add(contentDateElem); err != nil {
			return err
		}
	}

	// Update Content Time if it exists
	if ds.Contains(tag.ContentTime) {
		contentTimeElem, err := element.NewElement(tag.ContentTime, vr.Time, timeVal)
		if err != nil {
			return fmt.Errorf("failed to create ContentTime: %w", err)
		}
		if err := ds.Add(contentTimeElem); err != nil {
			return err
		}
	}

	return nil
}

// AnonymizeBasic performs basic anonymization on the dataset.
//
// This is a simple anonymization that:
//   - Removes patient name, ID, birth date
//   - Removes institution names
//   - Removes private tags
//   - Generates new UIDs
//   - Updates dates/times to current
//
// For full DICOM PS3.15 compliant anonymization, use the anonymize package.
//
// Example:
//
//	if err := ds.AnonymizeBasic(); err != nil {
//	    log.Fatal(err)
//	}
func (ds *DataSet) AnonymizeBasic() error {
	// Set anonymous patient data
	if err := ds.SetPatientName("ANONYMOUS"); err != nil {
		return err
	}
	if err := ds.SetPatientID("ANON001"); err != nil {
		return err
	}
	if err := ds.SetPatientBirthDate(""); err != nil {
		return err
	}
	if err := ds.SetPatientAge(""); err != nil {
		return err
	}

	// Remove institution identifiers
	institutionTags := []tag.Tag{
		tag.InstitutionName,
		tag.InstitutionAddress,
		tag.InstitutionalDepartmentName,
		tag.ReferringPhysicianName,
		tag.PerformingPhysicianName,
		tag.OperatorsName,
	}

	for _, t := range institutionTags {
		if ds.Contains(t) {
			_ = ds.Remove(t)
		}
	}

	// Remove private tags
	if err := ds.RemovePrivateTags(); err != nil {
		return err
	}

	// Generate new UIDs
	if err := ds.GenerateNewUIDs(); err != nil {
		return err
	}

	// Update timestamps
	if err := ds.SetCurrentDateTime(); err != nil {
		return err
	}

	return nil
}
