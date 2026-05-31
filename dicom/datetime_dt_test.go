package dicom

import (
	"testing"
	"time"
)

func TestParseDTFullWithOffset(t *testing.T) {
	dt, err := ParseDT("20240229153045.123456+1000")
	if err != nil {
		t.Fatalf("ParseDT: %v", err)
	}
	if dt.String() != "20240229153045.123456+1000" {
		t.Errorf("String() = %q, want preserved input", dt.String())
	}
	got, ok := dt.Time()
	if !ok {
		t.Fatal("full DT should resolve to a time.Time")
	}
	if got.Year() != 2024 || got.Month() != time.February || got.Day() != 29 {
		t.Errorf("date = %v, want 2024-02-29", got)
	}
	if got.Hour() != 15 || got.Minute() != 30 || got.Second() != 45 || got.Nanosecond() != 123_456_000 {
		t.Errorf("time = %v, want 15:30:45.123456", got)
	}
	_, offset := got.Zone()
	if offset != 10*3600 {
		t.Errorf("zone offset = %d s, want %d (+1000)", offset, 10*3600)
	}
}

// TestParseDTNegativeOffset is part of the DCM-010 regression: DT must parse the
// signed +/-HHMM UTC offset and Time() must carry it.
func TestParseDTNegativeOffset(t *testing.T) {
	dt, err := ParseDT("20240101000000-0530")
	if err != nil {
		t.Fatalf("ParseDT: %v", err)
	}
	got, _ := dt.Time()
	_, offset := got.Zone()
	if offset != -(5*3600 + 30*60) {
		t.Errorf("zone offset = %d s, want %d (-0530)", offset, -(5*3600 + 30*60))
	}
}

func TestParseDTNoOffsetIsUTC(t *testing.T) {
	dt, err := ParseDT("20240101120000")
	if err != nil {
		t.Fatalf("ParseDT: %v", err)
	}
	if dt.HasOffset() {
		t.Error("a DT without &ZZXX must report HasOffset() == false")
	}
	got, _ := dt.Time()
	_, offset := got.Zone()
	if offset != 0 {
		t.Errorf("offsetless DT resolves to UTC, got offset %d", offset)
	}
}

// TestParseDTVariablePrecisionRoundTrip is the DCM-010 regression that a
// variable-precision DT round-trips its source form unchanged (no silent zero-fill).
func TestParseDTVariablePrecisionRoundTrip(t *testing.T) {
	cases := []string{
		"2024",
		"202402",
		"20240229",
		"2024022915",
		"202402291530",
		"20240229153045",
		"20240229153045.5",
		"20240229153045.123456",
		"2024+1000",
		"20240229153045.123-0500",
	}
	for _, in := range cases {
		dt, err := ParseDT(in)
		if err != nil {
			t.Errorf("ParseDT(%q): %v", in, err)
			continue
		}
		if dt.String() != in {
			t.Errorf("round-trip lost form: String() = %q, want %q", dt.String(), in)
		}
	}
}

func TestParseDTLeapSecond(t *testing.T) {
	dt, err := ParseDT("20240630235960")
	if err != nil {
		t.Fatalf("ParseDT(leap second): %v (DCM-010)", err)
	}
	if dt.String() != "20240630235960" {
		t.Errorf("String() = %q, want preserved leap second", dt.String())
	}
	got, _ := dt.Time()
	if got.Second() != 59 {
		t.Errorf("Time().Second() = %d, want 59 (leap second normalised)", got.Second())
	}
}

func TestParseDTPartialTimeIncomplete(t *testing.T) {
	// A date-only DT must not fabricate a time; Time() resolves midnight only when
	// the date is full, and a year-only DT reports incomplete via ok == false.
	dt, err := ParseDT("2024")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := dt.Time(); ok {
		t.Error("a year-precision DT must not resolve to a fabricated instant")
	}
}

func TestParseDTRejectsMalformed(t *testing.T) {
	cases := []string{
		"",                       // empty
		"202413",                 // month 13
		"20240230",               // Feb 30
		"2024022915304",          // odd time digit count
		"20240229153045.",        // trailing dot
		"20240229153045.1234567", // 7 fractional digits
		"20240229153045+10",      // short offset
		"20240229153045+1060",    // offset minute 60
		"20240229153045+2400",    // offset hour 24
		"20240229153045Z",        // bad offset sign
		"2024-02-29",             // separators
		"20240229246000",         // hour 24
		"20",                     // 2 digits (not a valid year)
	}
	for _, s := range cases {
		if _, err := ParseDT(s); err == nil {
			t.Errorf("ParseDT(%q): want error, got nil", s)
		}
	}
}
