package value

import (
	"fmt"

	"github.com/codeninja55/go-radx/dicom/datetime"
	"github.com/codeninja55/go-radx/dicom/vr"
)

// AsDate parses the StringValue as a DICOM Date (DA) Value Representation.
//
// Returns an error if:
//   - The VR is not DA
//   - The value is empty or has multiple values
//   - The date string is invalid
//
// Example:
//
//	val, _ := NewStringValue(vr.Date, []string{"20231015"})
//	date, err := val.AsDate()  // datetime.Date{2023-10-15}
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func (s *StringValue) AsDate() (datetime.Date, error) {
	// Validate VR
	if s.vr != vr.Date {
		return datetime.Date{}, fmt.Errorf("cannot parse VR %s as Date (expected DA)", s.vr.String())
	}

	// Validate single value
	if len(s.values) == 0 {
		return datetime.Date{}, fmt.Errorf("cannot parse empty Date value")
	}
	if len(s.values) > 1 {
		return datetime.Date{}, fmt.Errorf("cannot parse Date with multiple values (got %d)", len(s.values))
	}

	// Parse date
	return datetime.ParseDate(s.values[0])
}

// AsTime parses the StringValue as a DICOM Time (TM) Value Representation.
//
// Returns an error if:
//   - The VR is not TM
//   - The value is empty or has multiple values
//   - The time string is invalid
//
// Example:
//
//	val, _ := NewStringValue(vr.Time, []string{"143025.123456"})
//	time, err := val.AsTime()  // datetime.Time{14:30:25.123456}
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func (s *StringValue) AsTime() (datetime.Time, error) {
	// Validate VR
	if s.vr != vr.Time {
		return datetime.Time{}, fmt.Errorf("cannot parse VR %s as Time (expected TM)", s.vr.String())
	}

	// Validate single value
	if len(s.values) == 0 {
		return datetime.Time{}, fmt.Errorf("cannot parse empty Time value")
	}
	if len(s.values) > 1 {
		return datetime.Time{}, fmt.Errorf("cannot parse Time with multiple values (got %d)", len(s.values))
	}

	// Parse time
	return datetime.ParseTime(s.values[0])
}

// AsDateTime parses the StringValue as a DICOM DateTime (DT) Value Representation.
//
// Returns an error if:
//   - The VR is not DT
//   - The value is empty or has multiple values
//   - The datetime string is invalid
//
// Example:
//
//	val, _ := NewStringValue(vr.DateTime, []string{"20231015143025+1000"})
//	dt, err := val.AsDateTime()  // datetime.DateTime{2023-10-15 14:30:25 +1000}
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func (s *StringValue) AsDateTime() (datetime.DateTime, error) {
	// Validate VR
	if s.vr != vr.DateTime {
		return datetime.DateTime{}, fmt.Errorf("cannot parse VR %s as DateTime (expected DT)", s.vr.String())
	}

	// Validate single value
	if len(s.values) == 0 {
		return datetime.DateTime{}, fmt.Errorf("cannot parse empty DateTime value")
	}
	if len(s.values) > 1 {
		return datetime.DateTime{}, fmt.Errorf("cannot parse DateTime with multiple values (got %d)", len(s.values))
	}

	// Parse datetime
	return datetime.ParseDateTime(s.values[0])
}

// AsAge parses the StringValue as a DICOM Age String (AS) Value Representation.
//
// Returns an error if:
//   - The VR is not AS
//   - The value is empty or has multiple values
//   - The age string is invalid
//
// Example:
//
//	val, _ := NewStringValue(vr.AgeString, []string{"042Y"})
//	age, err := val.AsAge()  // datetime.Age{Value: 42, Unit: Years}
//	duration := age.Duration()  // time.Duration (42 years in nanoseconds)
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func (s *StringValue) AsAge() (datetime.Age, error) {
	// Validate VR
	if s.vr != vr.AgeString {
		return datetime.Age{}, fmt.Errorf("cannot parse VR %s as Age (expected AS)", s.vr.String())
	}

	// Validate single value
	if len(s.values) == 0 {
		return datetime.Age{}, fmt.Errorf("cannot parse empty Age value")
	}
	if len(s.values) > 1 {
		return datetime.Age{}, fmt.Errorf("cannot parse Age with multiple values (got %d)", len(s.values))
	}

	// Parse age
	return datetime.ParseAge(s.values[0])
}
