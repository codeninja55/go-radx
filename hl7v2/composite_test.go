package hl7v2

import (
	"testing"
	"time"
)

func TestParseDTMPrecision(t *testing.T) {
	tests := []struct {
		in        string
		wantPrec  Precision
		wantTime  time.Time
		wantZero  bool
		wantError bool
	}{
		{in: "", wantZero: true},
		{in: "2026", wantPrec: PrecisionYear, wantTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{in: "202605", wantPrec: PrecisionMonth, wantTime: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)},
		{in: "20260531", wantPrec: PrecisionDay, wantTime: time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)},
		{in: "2026053112", wantPrec: PrecisionHour, wantTime: time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)},
		{in: "202605311230", wantPrec: PrecisionMinute, wantTime: time.Date(2026, 5, 31, 12, 30, 0, 0, time.UTC)},
		{in: "20260531123045", wantPrec: PrecisionSecond, wantTime: time.Date(2026, 5, 31, 12, 30, 45, 0, time.UTC)},
		{in: "20260531123045.5", wantPrec: PrecisionFraction, wantTime: time.Date(2026, 5, 31, 12, 30, 45, 0, time.UTC)},
		{in: "202605311230+1000", wantPrec: PrecisionMinute, wantTime: time.Date(2026, 5, 31, 12, 30, 0, 0, time.UTC)}, // TZ offset preserved, not applied
		{in: "202605311230-0500", wantPrec: PrecisionMinute, wantTime: time.Date(2026, 5, 31, 12, 30, 0, 0, time.UTC)},
		{in: "2026053", wantError: true},           // 7 digits is not a valid precision
		{in: "abcd", wantError: true},              // non-numeric
		{in: "20261331", wantError: true},          // month 13
		{in: "20260231", wantError: true},          // Feb 31 must not normalise to Mar 3
		{in: "20260229", wantError: true},          // 2026 is not a leap year
		{in: "20260531123045.", wantError: true},   // empty fractional tail
		{in: "20260531123045.ab", wantError: true}, // non-numeric fraction
		{in: "202605311230.5", wantError: true},    // fraction without seconds precision
		{in: "20260531123060", wantError: true},    // second 60 must not normalise to the next minute
		{in: "20260531126030", wantError: true},    // minute 60 must not normalise
		{in: "20260531253045", wantError: true},    // hour 25 must not normalise
		{in: "202605311230+ABCD", wantError: true}, // non-numeric timezone
		{in: "202605311230+", wantError: true},     // bare timezone sign
		{in: "202605311230+100", wantError: true},  // timezone offset too short
	}

	for _, tc := range tests {
		got, err := ParseDTM(tc.in)
		if tc.wantError {
			if err == nil {
				t.Errorf("ParseDTM(%q) expected error, got none", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseDTM(%q) error = %v", tc.in, err)
			continue
		}
		if got.IsZero() != tc.wantZero {
			t.Errorf("ParseDTM(%q).IsZero() = %v, want %v", tc.in, got.IsZero(), tc.wantZero)
		}
		if tc.wantZero {
			continue
		}
		if got.Precision() != tc.wantPrec {
			t.Errorf("ParseDTM(%q).Precision() = %d, want %d", tc.in, got.Precision(), tc.wantPrec)
		}
		resolved, _, ok := got.Time()
		if !ok || !resolved.Equal(tc.wantTime) {
			t.Errorf("ParseDTM(%q).Time() = %v (ok=%v), want %v", tc.in, resolved, ok, tc.wantTime)
		}
		if got.String() != tc.in {
			t.Errorf("ParseDTM(%q).String() = %q, want lexical round-trip", tc.in, got.String())
		}
	}
}

func TestParseCXAndHD(t *testing.T) {
	// PID-3 of the canonical ORM: 555-44-4444^^^HOSP^MR
	msg, err := Parse([]byte(canonicalORM))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	pid, _ := msg.Segment("PID")
	cx := parseCX(pid.field(3).Repetitions[0])

	if cx.ID != "555-44-4444" {
		t.Errorf("CX.ID = %q, want 555-44-4444", cx.ID)
	}
	if cx.AssigningAuthority.NamespaceID != "HOSP" {
		t.Errorf("CX.AssigningAuthority.NamespaceID = %q, want HOSP", cx.AssigningAuthority.NamespaceID)
	}
	if cx.IdentifierTypeCode != "MR" {
		t.Errorf("CX.IdentifierTypeCode = %q, want MR", cx.IdentifierTypeCode)
	}
}

func TestParseCWESixComponents(t *testing.T) {
	// A six-component CWE: code^text^system^altcode^alttext^altsystem.
	full := parseRepetition([]byte("36643-5^CHEST XRAY^LN^36643^CHEST X-RAY^L"), DefaultEncoding())
	cwe := parseCWE(full)
	want := CWE{
		Code:            "36643-5",
		Text:            "CHEST XRAY",
		CodingSystem:    "LN",
		AltCode:         "36643",
		AltText:         "CHEST X-RAY",
		AltCodingSystem: "L",
	}
	if cwe != want {
		t.Errorf("parseCWE(full) = %+v, want %+v", cwe, want)
	}

	// A three-component CWE leaves the alternate fields empty (absence is empty,
	// not error).
	partial := parseRepetition([]byte("24323-8^COMPREHENSIVE METABOLIC PANEL^LN"), DefaultEncoding())
	cwe = parseCWE(partial)
	if cwe.AltCode != "" || cwe.AltText != "" || cwe.AltCodingSystem != "" {
		t.Errorf("parseCWE(partial) alternate fields = %q/%q/%q, want all empty",
			cwe.AltCode, cwe.AltText, cwe.AltCodingSystem)
	}
}

func TestParseCXNestedHDInSubcomponents(t *testing.T) {
	// CX-4 may carry the HD as subcomponents: ID^^^NS&UID&ISO^MR
	msg, err := Parse([]byte(
		"MSH|^~\\&|A|B|C|D|202605311230||ORM^O01|M1|P|2.4\r" +
			"PID|||9000^^^NS&1.2.3&ISO^MR\r"))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	pid, _ := msg.Segment("PID")
	cx := parseCX(pid.field(3).Repetitions[0])
	if cx.AssigningAuthority.NamespaceID != "NS" || cx.AssigningAuthority.UniversalID != "1.2.3" || cx.AssigningAuthority.UniversalIDType != "ISO" {
		t.Errorf("CX-4 HD = %+v, want {NS 1.2.3 ISO}", cx.AssigningAuthority)
	}
}
