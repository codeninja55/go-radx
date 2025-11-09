package datetime_test

import (
	"fmt"
	"time"

	"github.com/codeninja55/go-radx/dicom/datetime"
)

// Example demonstrates basic usage of all temporal parsers.
func Example() {
	// Parse a DICOM date
	date, _ := datetime.ParseDate("20231015")
	fmt.Println("Date:", date.String())

	// Parse a DICOM time
	tim, _ := datetime.ParseTime("143025.123456")
	fmt.Println("Time:", tim.String())

	// Parse a DICOM datetime with timezone
	dt, _ := datetime.ParseDateTime("20231015143025+1000")
	fmt.Println("DateTime:", dt.String())

	// Parse a DICOM age string
	age, _ := datetime.ParseAge("042Y")
	fmt.Println("Age:", age.String())

	// Output:
	// Date: 2023-10-15
	// Time: 14:30:25.123456
	// DateTime: 2023-10-15 14:30:25 +1000
	// Age: 42 years
}

// ExampleParseDate demonstrates parsing DICOM Date (DA) values
// with different precision levels.
func ExampleParseDate() {
	// Full date
	date1, _ := datetime.ParseDate("20231015")
	fmt.Println(date1.DCM(), "->", date1.String())

	// Year and month only
	date2, _ := datetime.ParseDate("202310")
	fmt.Println(date2.DCM(), "->", date2.String())

	// Year only
	date3, _ := datetime.ParseDate("2023")
	fmt.Println(date3.DCM(), "->", date3.String())

	// Legacy NEMA-300 format
	date4, _ := datetime.ParseDate("2023.10.15")
	fmt.Println(date4.DCM(), "->", date4.String())

	// Output:
	// 20231015 -> 2023-10-15
	// 202310 -> 2023-10
	// 2023 -> 2023
	// 2023.10.15 -> 2023-10-15
}

// ExampleParseTime demonstrates parsing DICOM Time (TM) values
// with different precision levels.
func ExampleParseTime() {
	// Full time with microseconds
	time1, _ := datetime.ParseTime("143025.123456")
	fmt.Println(time1.DCM(), "->", time1.String())

	// Seconds precision
	time2, _ := datetime.ParseTime("143025")
	fmt.Println(time2.DCM(), "->", time2.String())

	// Minutes precision
	time3, _ := datetime.ParseTime("1430")
	fmt.Println(time3.DCM(), "->", time3.String())

	// Hours precision
	time4, _ := datetime.ParseTime("14")
	fmt.Println(time4.DCM(), "->", time4.String())

	// Output:
	// 143025.123456 -> 14:30:25.123456
	// 143025 -> 14:30:25
	// 1430 -> 14:30
	// 14 -> 14
}

// ExampleParseDateTime demonstrates parsing DICOM DateTime (DT) values
// with and without timezone offsets.
func ExampleParseDateTime() {
	// Full datetime with timezone
	dt1, _ := datetime.ParseDateTime("20231015143025+1000")
	fmt.Println("With timezone:", dt1.String())

	// Full datetime without timezone
	dt2, _ := datetime.ParseDateTime("20231015143025")
	fmt.Println("Without timezone:", dt2.String())

	// Date only
	dt3, _ := datetime.ParseDateTime("20231015")
	fmt.Println("Date only:", dt3.String())

	// Output:
	// With timezone: 2023-10-15 14:30:25 +1000
	// Without timezone: 2023-10-15 14:30:25 UTC
	// Date only: 2023-10-15 UTC
}

// ExampleParseAge demonstrates parsing DICOM Age String (AS) values
// and converting them to time.Duration.
func ExampleParseAge() {
	// Parse different age units
	days, _ := datetime.ParseAge("007D")
	fmt.Printf("%s = %.0f hours\n", days.String(), days.Duration().Hours())

	weeks, _ := datetime.ParseAge("004W")
	fmt.Printf("%s = %.0f hours\n", weeks.String(), weeks.Duration().Hours())

	months, _ := datetime.ParseAge("006M")
	fmt.Printf("%s = %.0f days\n", months.String(), months.Duration().Hours()/24)

	years, _ := datetime.ParseAge("042Y")
	fmt.Printf("%s = %.1f days\n", years.String(), years.Duration().Hours()/24)

	// Output:
	// 7 days = 168 hours
	// 4 weeks = 672 hours
	// 6 months = 183 days
	// 42 years = 15340.5 days
}

// ExampleDate_DCM demonstrates formatting Date values back to DICOM format
// while preserving the original precision.
func ExampleDate_DCM() {
	// Parse with different precisions
	yearMonth, _ := datetime.ParseDate("202310")
	yearOnly, _ := datetime.ParseDate("2023")

	// Format back to DICOM - precision is preserved
	fmt.Println(yearMonth.DCM()) // Not "20231001"
	fmt.Println(yearOnly.DCM())  // Not "20230101"

	// Output:
	// 202310
	// 2023
}

// ExampleDateTime_NoOffset demonstrates the distinction between
// "no timezone" and "UTC timezone" in DICOM datetime values.
func ExampleDateTime_NoOffset() {
	// DateTime without timezone offset
	dt1, _ := datetime.ParseDateTime("20231015143025")
	fmt.Println("NoOffset =", dt1.NoOffset)

	// DateTime with explicit UTC offset
	dt2, _ := datetime.ParseDateTime("20231015143025+0000")
	fmt.Println("NoOffset =", dt2.NoOffset)

	// Both are stored in UTC internally
	fmt.Println("Both are UTC:", dt1.Time.Location() == time.UTC && dt2.Time.Location().String() == "+0000")

	// Output:
	// NoOffset = true
	// NoOffset = false
	// Both are UTC: true
}

// ExampleAge_Duration demonstrates converting Age String (AS) values
// to Go's time.Duration type using medically accurate conversion factors.
func ExampleAge_Duration() {
	age, _ := datetime.ParseAge("001Y")
	duration := age.Duration()

	// One year = 365.25 days (accounts for leap years)
	daysPerYear := duration.Hours() / 24
	fmt.Printf("1 year = %.2f days\n", daysPerYear)

	// One month = 30.4375 days (365.25/12)
	monthAge, _ := datetime.ParseAge("001M")
	daysPerMonth := monthAge.Duration().Hours() / 24
	fmt.Printf("1 month = %.4f days\n", daysPerMonth)

	// Output:
	// 1 year = 365.25 days
	// 1 month = 30.4375 days
}

// ExamplePrecisionLevel demonstrates working with precision tracking.
func ExamplePrecisionLevel() {
	// Parse dates with different precisions
	fullDate, _ := datetime.ParseDate("20231015")
	yearMonth, _ := datetime.ParseDate("202310")
	yearOnly, _ := datetime.ParseDate("2023")

	// Check precision
	fmt.Println("Full date:", fullDate.Precision.String())
	fmt.Println("Year-month:", yearMonth.Precision.String())
	fmt.Println("Year only:", yearOnly.Precision.String())

	// Output:
	// Full date: Day
	// Year-month: Month
	// Year only: Year
}
