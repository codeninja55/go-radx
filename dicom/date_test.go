package dicom

import (
	"errors"
	"testing"
	"time"
)

func TestParseDAStrictAcceptsEightDigits(t *testing.T) {
	d, err := ParseDA("20240229") // leap day, valid
	if err != nil {
		t.Fatalf("ParseDA(20240229): %v", err)
	}
	if d.String() != "20240229" {
		t.Errorf("String() = %q, want preserved 20240229", d.String())
	}
	if d.Precision() != DatePrecisionDay {
		t.Errorf("Precision() = %v, want DatePrecisionDay", d.Precision())
	}
	tm, ok := d.Time()
	if !ok {
		t.Fatal("a full date should resolve to a time.Time")
	}
	if tm.Year() != 2024 || tm.Month() != time.February || tm.Day() != 29 {
		t.Errorf("Time() = %v, want 2024-02-29", tm)
	}
}

// TestParseDARejectsPartialFormsByDefault is the DCM-010 regression: strict DA
// requires 8 digits, rejecting the legacy YYYY and YYYYMM forms.
func TestParseDARejectsPartialFormsByDefault(t *testing.T) {
	for _, s := range []string{"2024", "202402"} {
		if _, err := ParseDA(s); err == nil {
			t.Errorf("ParseDA(%q) strict: want error, got nil (DCM-010)", s)
		} else {
			var ve *ValueError
			if !errors.As(err, &ve) {
				t.Errorf("ParseDA(%q) error = %T, want *ValueError", s, err)
			}
		}
	}
}

func TestParseDALenientAcceptsPartialForms(t *testing.T) {
	cases := []struct {
		in   string
		prec DatePrecision
	}{
		{"2024", DatePrecisionYear},
		{"202402", DatePrecisionMonth},
		{"20240229", DatePrecisionDay},
	}
	for _, c := range cases {
		d, err := ParseDA(c.in, withLenient())
		if err != nil {
			t.Errorf("ParseDA(%q, lenient): %v", c.in, err)
			continue
		}
		if d.String() != c.in {
			t.Errorf("String() = %q, want preserved %q", d.String(), c.in)
		}
		if d.Precision() != c.prec {
			t.Errorf("ParseDA(%q) precision = %v, want %v", c.in, d.Precision(), c.prec)
		}
	}
}

// TestDAPartialTimeReportsIncomplete: a partial date must not fabricate the
// missing month/day, so Time() reports ok == false (the reference forbids
// silently zero-filling).
func TestDAPartialTimeReportsIncomplete(t *testing.T) {
	d, err := ParseDA("202402", withLenient())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := d.Time(); ok {
		t.Error("a month-precision date must not resolve to a fabricated full date")
	}
	if d.Year() != 2024 || d.Month() != 2 {
		t.Errorf("Year/Month = %d/%d, want 2024/2", d.Year(), d.Month())
	}
}

func TestParseDARejectsMalformed(t *testing.T) {
	cases := []string{
		"",          // empty
		"2024022",   // 7 digits
		"202402290", // 9 digits
		"2024-02-29", // separators
		"20241301",  // month 13
		"20240230",  // Feb 30 (invalid calendar day)
		"20230229",  // 2023 not a leap year
		"2024023x",  // non-digit
		"00000000",  // year 0 / month 0 / day 0
	}
	for _, s := range cases {
		if _, err := ParseDA(s); err == nil {
			t.Errorf("ParseDA(%q): want error, got nil", s)
		}
	}
}

func TestParseDARejectsMalformedEvenLenient(t *testing.T) {
	// Lenient relaxes only the length-3 partial forms; it never accepts a
	// non-conformant 8-digit or separator-laden string.
	for _, s := range []string{"20241301", "2024-02", "20240230", "00"} {
		if _, err := ParseDA(s, withLenient()); err == nil {
			t.Errorf("ParseDA(%q, lenient): want error, got nil", s)
		}
	}
}
