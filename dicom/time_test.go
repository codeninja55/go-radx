package dicom

import (
	"testing"
	"time"
)

func TestParseTMVariablePrecision(t *testing.T) {
	cases := []struct {
		in    string
		h, m  int
		s, ns int
	}{
		{"07", 7, 0, 0, 0},
		{"0730", 7, 30, 0, 0},
		{"073045", 7, 30, 45, 0},
		{"073045.5", 7, 30, 45, 500_000_000},
		{"073045.123456", 7, 30, 45, 123_456_000},
		{"235959.999999", 23, 59, 59, 999_999_000},
	}
	for _, c := range cases {
		tm, err := ParseTM(c.in)
		if err != nil {
			t.Errorf("ParseTM(%q): %v", c.in, err)
			continue
		}
		if tm.String() != c.in {
			t.Errorf("String() = %q, want preserved %q", tm.String(), c.in)
		}
		got, ok := tm.Time()
		if !ok {
			t.Errorf("ParseTM(%q).Time() ok == false", c.in)
			continue
		}
		if got.Hour() != c.h || got.Minute() != c.m || got.Second() != c.s || got.Nanosecond() != c.ns {
			t.Errorf("ParseTM(%q).Time() = %02d:%02d:%02d.%09d, want %02d:%02d:%02d.%09d",
				c.in, got.Hour(), got.Minute(), got.Second(), got.Nanosecond(), c.h, c.m, c.s, c.ns)
		}
	}
}

// TestParseTMLeapSecondNormalises is the DCM-010 regression for TM: SS=60 is valid
// in DICOM but not in Go's time package, so ParseTM accepts it, the preserved string
// keeps 60, and Time() normalises the second to 59.
func TestParseTMLeapSecondNormalises(t *testing.T) {
	tm, err := ParseTM("235960")
	if err != nil {
		t.Fatalf("ParseTM(235960): want acceptance, got %v (DCM-010)", err)
	}
	if tm.String() != "235960" {
		t.Errorf("String() = %q, want preserved 235960", tm.String())
	}
	got, ok := tm.Time()
	if !ok {
		t.Fatal("leap-second TM should still resolve to a time.Time")
	}
	if got.Second() != 59 {
		t.Errorf("Time().Second() = %d, want 59 (leap second normalised)", got.Second())
	}
	if got.Hour() != 23 || got.Minute() != 59 {
		t.Errorf("Time() = %02d:%02d, want 23:59", got.Hour(), got.Minute())
	}
}

func TestParseTMLeapSecondWithFraction(t *testing.T) {
	tm, err := ParseTM("235960.500000")
	if err != nil {
		t.Fatalf("ParseTM(235960.500000): %v", err)
	}
	got, _ := tm.Time()
	if got.Second() != 59 || got.Nanosecond() != 500_000_000 {
		t.Errorf("Time() = :%d.%09d, want :59.500000000", got.Second(), got.Nanosecond())
	}
}

func TestTMPrecisionReported(t *testing.T) {
	cases := []struct {
		in   string
		prec TimePrecision
	}{
		{"07", TimePrecisionHour},
		{"0730", TimePrecisionMinute},
		{"073045", TimePrecisionSecond},
		{"073045.5", TimePrecisionFraction},
	}
	for _, c := range cases {
		tm, err := ParseTM(c.in)
		if err != nil {
			t.Fatalf("ParseTM(%q): %v", c.in, err)
		}
		if tm.Precision() != c.prec {
			t.Errorf("ParseTM(%q).Precision() = %v, want %v", c.in, tm.Precision(), c.prec)
		}
	}
}

func TestParseTMRejectsMalformed(t *testing.T) {
	cases := []string{
		"",               // empty
		"7",              // odd, single digit
		"0760",           // minute 60
		"076145",         // minute 61
		"073061",         // second 61 (60 is the only allowed overflow)
		"2400",           // hour 24
		"073045.",        // trailing dot, no fraction
		"073045.1234567", // 7 fractional digits
		"07:30:45",       // separators
		"07304",          // 5 digits (odd)
		"abcdef",         // non-digit
	}
	for _, s := range cases {
		if _, err := ParseTM(s); err == nil {
			t.Errorf("ParseTM(%q): want error, got nil", s)
		}
	}
}

func TestTMFractionalPrecisionPreserved(t *testing.T) {
	// A 3-digit fraction must not be zero-filled to 6 digits in the lexical form.
	tm, err := ParseTM("073045.123")
	if err != nil {
		t.Fatal(err)
	}
	if tm.String() != "073045.123" {
		t.Errorf("String() = %q, want 073045.123 (no zero-fill)", tm.String())
	}
	got, _ := tm.Time()
	if got.Nanosecond() != 123_000_000 {
		t.Errorf("Time().Nanosecond() = %d, want 123000000", got.Nanosecond())
	}
	_ = time.Second
}
