// Package datetime provides parsing and formatting for DICOM temporal Value Representations (VR).
//
// This package handles four DICOM temporal VR types:
//   - DA (Date): YYYYMMDD format with variable precision
//   - TM (Time): HHMMSS.FFFFFF format with microsecond precision
//   - DT (DateTime): Combined date/time with timezone support
//   - AS (Age String): nnnU format (D/W/M/Y units)
//
// All temporal types provide bidirectional conversion between DICOM strings and Go native
// types (time.Time, time.Duration), with precision tracking to preserve the original
// level of detail specified in the DICOM string.
//
// # Basic Usage
//
// Parse DICOM date strings:
//
//	date, err := datetime.ParseDate("20231015")  // Full date
//	date, err := datetime.ParseDate("202310")    // Year-month
//	date, err := datetime.ParseDate("2023")      // Year only
//
// Parse DICOM time strings:
//
//	time, err := datetime.ParseTime("143025.123456")  // With microseconds
//	time, err := datetime.ParseTime("143025")         // Seconds precision
//	time, err := datetime.ParseTime("1430")           // Minutes precision
//
// Parse DICOM datetime strings:
//
//	dt, err := datetime.ParseDateTime("20231015143025+1000")  // With timezone
//	dt, err := datetime.ParseDateTime("20231015143025")       // Without timezone
//
// Parse DICOM age strings:
//
//	age, err := datetime.ParseAge("042Y")     // 42 years
//	duration := age.Duration()                 // Convert to time.Duration
//
// # Precision Tracking
//
// All temporal types track the precision of the original DICOM string. This allows
// round-trip conversion (parse → format → parse) to preserve the original format:
//
//	date, _ := datetime.ParseDate("202310")   // Year-month precision
//	fmt.Println(date.Precision)                // PrecisionMonth
//	fmt.Println(date.DCM())                    // "202310" (not "20231001")
//
// # Formatting
//
// Each type provides two formatting methods:
//   - DCM(): Returns DICOM VR format string (respects precision)
//   - String(): Returns human-readable format
//
// Example:
//
//	date, _ := datetime.ParseDate("20231015")
//	fmt.Println(date.DCM())      // "20231015"
//	fmt.Println(date.String())   // "2023-10-15"
//
// # Timezone Handling
//
// DateTime (DT) values support timezone offsets in +HHMM or -HHMM format:
//
//	dt, _ := datetime.ParseDateTime("20231015143025+1000")
//	_, offset := dt.Time.Zone()  // offset = 36000 (10 hours in seconds)
//
// The NoOffset field distinguishes between "no timezone specified" and "UTC timezone":
//
//	dt1, _ := datetime.ParseDateTime("20231015143025")       // NoOffset=true
//	dt2, _ := datetime.ParseDateTime("20231015143025+0000")  // NoOffset=false
//
// # Age to Duration Conversion
//
// Age String (AS) values can be converted to time.Duration using medically accurate factors:
//   - Days: 24 hours
//   - Weeks: 7 days
//   - Months: 30.4375 days (365.25/12, accounting for leap years)
//   - Years: 365.25 days (accounting for leap years)
//
// Example:
//
//	age, _ := datetime.ParseAge("042Y")
//	duration := age.Duration()           // 42 * 365.25 * 24 * time.Hour
//	fmt.Println(duration.Hours() / 24)   // 15340.5 days
//
// # Error Handling
//
// All Parse functions return descriptive errors for invalid inputs:
//
//	_, err := datetime.ParseDate("20230231")  // Invalid: Feb 31
//	// Returns: parse DA "20230231": invalid day for month
//
//	_, err := datetime.ParseTime("245959")    // Invalid: hour 24
//	// Returns: parse TM "245959": hour 24 out of range [0, 23]
//
// # Integration with StringValue
//
// Convenience methods are available on value.StringValue for direct parsing:
//
//	dateVal, _ := value.NewStringValue(vr.Date, []string{"20231015"})
//	date, err := dateVal.AsDate()
//
//	ageVal, _ := value.NewStringValue(vr.AgeString, []string{"042Y"})
//	age, err := ageVal.AsAge()
//	duration := age.Duration()
//
// # Legacy Format Support
//
// The Date parser supports legacy NEMA-300 format (YYYY.MM.DD):
//
//	date, _ := datetime.ParseDate("2023.10.15")
//	fmt.Println(date.IsNEMA)  // true
//	fmt.Println(date.DCM())    // "2023.10.15" (preserves format)
//
// # Round-Trip Guarantee
//
// All temporal types guarantee that parse → format → parse preserves the original string:
//
//	original := "20231015143025.123456+1000"
//	dt1, _ := datetime.ParseDateTime(original)
//	formatted := dt1.DCM()
//	dt2, _ := datetime.ParseDateTime(formatted)
//	// original == formatted, and dt1 equals dt2
//
// # DICOM Standard Reference
//
// This package implements parsing according to DICOM Part 5: Data Structures and Encoding.
//
// See: https://dicom.nema.org/medical/dicom/current/output/html/part05.html
package datetime
