package hl7v2

import (
	"bytes"
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

func TestParseXPNDegree(t *testing.T) {
	// Family^Given^Middle^Suffix^Prefix^Degree^NameTypeCode.
	r := parseRepetition([]byte("DOE^JOHN^A^JR^DR^PHD^L"), DefaultEncoding())
	xpn := parseXPN(r)
	if xpn.Degree != "PHD" {
		t.Errorf("XPN.Degree = %q, want PHD", xpn.Degree)
	}
	// NameTypeCode must stay at component 7, not shift when Degree is read at 6.
	if xpn.NameTypeCode != "L" {
		t.Errorf("XPN.NameTypeCode = %q, want L", xpn.NameTypeCode)
	}
	if xpn.Prefix != "DR" {
		t.Errorf("XPN.Prefix = %q, want DR", xpn.Prefix)
	}
}

func TestParseXAD(t *testing.T) {
	// Street^OtherDesignation^City^State^Zip^Country.
	r := parseRepetition([]byte("123 MAIN ST^APT 4^METROPOLIS^NY^10001^USA"), DefaultEncoding())
	xad := parseXAD(r)
	want := XAD{
		Street:           "123 MAIN ST",
		OtherDesignation: "APT 4",
		City:             "METROPOLIS",
		State:            "NY",
		Zip:              "10001",
		Country:          "USA",
	}
	if xad != want {
		t.Errorf("parseXAD = %+v, want %+v", xad, want)
	}

	// An absent address yields the zero XAD.
	empty := parseRepetition([]byte(""), DefaultEncoding())
	if got := parseXAD(empty); got != (XAD{}) {
		t.Errorf("parseXAD(empty) = %+v, want zero XAD", got)
	}
}

// renderRepetition renders a single Repetition with the given encoding, the same
// way MarshalText renders one repetition of a field.
func renderRepetition(r Repetition, enc EncodingCharacters) string {
	var buf bytes.Buffer
	r.render(&buf, enc)
	return buf.String()
}

func TestCompositeRepetitionRenderers(t *testing.T) {
	enc := DefaultEncoding()

	// A CX with gaps: CX-2 and CX-3 empty, CX-4 a nested HD, CX-5 the type code.
	cx := CX{
		ID:                 "PATID1234",
		AssigningAuthority: HD{NamespaceID: "HOSP"},
		IdentifierTypeCode: "MR",
	}
	if got := renderRepetition(cx.repetition(), enc); got != "PATID1234^^^HOSP^MR" {
		t.Errorf("CX render = %q, want PATID1234^^^HOSP^MR", got)
	}

	// A CWE with only its code renders with no trailing carets.
	if got := renderRepetition(CWE{Code: "NM"}.repetition(), enc); got != "NM" {
		t.Errorf("CWE render = %q, want NM", got)
	}

	// An HD with a full universal-ID triplet.
	if got := renderRepetition(HD{NamespaceID: "HOSP", UniversalID: "1.2.3", UniversalIDType: "ISO"}.repetition(), enc); got != "HOSP^1.2.3^ISO" {
		t.Errorf("HD render = %q, want HOSP^1.2.3^ISO", got)
	}

	// An XPN with Degree present at component 6 keeps NameTypeCode at 7.
	xpn := XPN{Family: "DOE", Given: "JOHN", Degree: "PHD", NameTypeCode: "L"}
	if got := renderRepetition(xpn.repetition(), enc); got != "DOE^JOHN^^^^PHD^L" {
		t.Errorf("XPN render = %q, want DOE^JOHN^^^^PHD^L", got)
	}

	// An XAD round-trips its postal components.
	xad := XAD{Street: "123 MAIN ST", City: "METROPOLIS", State: "NY", Zip: "10001", Country: "USA"}
	if got := renderRepetition(xad.repetition(), enc); got != "123 MAIN ST^^METROPOLIS^NY^10001^USA" {
		t.Errorf("XAD render = %q, want 123 MAIN ST^^METROPOLIS^NY^10001^USA", got)
	}

	// A wholly empty composite renders as the empty string (one empty component).
	if got := renderRepetition(CWE{}.repetition(), enc); got != "" {
		t.Errorf("empty CWE render = %q, want empty", got)
	}
}

func TestCompositeRoundTrip(t *testing.T) {
	enc := DefaultEncoding()

	// parseThenRender parses raw into a typed composite and renders it back; it
	// returns the rendered text so the test can assert render(parse(raw)) == raw.
	// reparse re-parses the rendered text so the test can assert that the value
	// survives parse → render → parse unchanged.
	tests := []struct {
		name     string
		raw      string
		render   func(string) string // render(parse(raw))
		stableID bool                // raw round-trips byte-exact (no lossy components)
	}{
		// HD
		{"HD/full", "HOSP^1.2.3^ISO", func(r string) string {
			return renderRepetition(parseHD(parseRepetition([]byte(r), enc)).repetition(), enc)
		}, true},
		{"HD/namespace-only", "HOSP", func(r string) string {
			return renderRepetition(parseHD(parseRepetition([]byte(r), enc)).repetition(), enc)
		}, true},
		{"HD/empty", "", func(r string) string {
			return renderRepetition(parseHD(parseRepetition([]byte(r), enc)).repetition(), enc)
		}, true},
		// CX (CX-3 is not modelled, so a value carrying CX-3 is not byte-stable;
		// the modelled-field cases below are.)
		{"CX/id-authority-type", "PATID1234^^^HOSP^MR", func(r string) string {
			return renderRepetition(parseCX(parseRepetition([]byte(r), enc)).repetition(), enc)
		}, true},
		{"CX/nested-hd", "9000^^^NS&1.2.3&ISO^MR", func(r string) string {
			return renderRepetition(parseCX(parseRepetition([]byte(r), enc)).repetition(), enc)
		}, true},
		{"CX/id-only", "PATID1234", func(r string) string {
			return renderRepetition(parseCX(parseRepetition([]byte(r), enc)).repetition(), enc)
		}, true},
		{"CX/empty", "", func(r string) string {
			return renderRepetition(parseCX(parseRepetition([]byte(r), enc)).repetition(), enc)
		}, true},
		// CWE
		{"CWE/six", "36643-5^CHEST XRAY^LN^36643^CHEST X-RAY^L", func(r string) string {
			return renderRepetition(parseCWE(parseRepetition([]byte(r), enc)).repetition(), enc)
		}, true},
		{"CWE/three", "24323-8^COMPREHENSIVE METABOLIC PANEL^LN", func(r string) string {
			return renderRepetition(parseCWE(parseRepetition([]byte(r), enc)).repetition(), enc)
		}, true},
		{"CWE/code-only", "NM", func(r string) string {
			return renderRepetition(parseCWE(parseRepetition([]byte(r), enc)).repetition(), enc)
		}, true},
		{"CWE/gappy", "NM^^L", func(r string) string {
			return renderRepetition(parseCWE(parseRepetition([]byte(r), enc)).repetition(), enc)
		}, true},
		{"CWE/empty", "", func(r string) string {
			return renderRepetition(parseCWE(parseRepetition([]byte(r), enc)).repetition(), enc)
		}, true},
		// XPN
		{"XPN/full", "DOE^JOHN^A^JR^DR^PHD^L", func(r string) string {
			return renderRepetition(parseXPN(parseRepetition([]byte(r), enc)).repetition(), enc)
		}, true},
		{"XPN/family-given", "DOE^JOHN", func(r string) string {
			return renderRepetition(parseXPN(parseRepetition([]byte(r), enc)).repetition(), enc)
		}, true},
		{"XPN/trailing-type", "DOE^JOHN^^^^^L", func(r string) string {
			return renderRepetition(parseXPN(parseRepetition([]byte(r), enc)).repetition(), enc)
		}, true},
		{"XPN/empty", "", func(r string) string {
			return renderRepetition(parseXPN(parseRepetition([]byte(r), enc)).repetition(), enc)
		}, true},
		// XAD
		{"XAD/full", "123 MAIN ST^APT 4^METROPOLIS^NY^10001^USA", func(r string) string {
			return renderRepetition(parseXAD(parseRepetition([]byte(r), enc)).repetition(), enc)
		}, true},
		{"XAD/street-city-state", "200 SAMPLE AVE^^METROPOLIS^NY", func(r string) string {
			return renderRepetition(parseXAD(parseRepetition([]byte(r), enc)).repetition(), enc)
		}, true},
		{"XAD/empty", "", func(r string) string {
			return renderRepetition(parseXAD(parseRepetition([]byte(r), enc)).repetition(), enc)
		}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.render(tc.raw)
			if tc.stableID && got != tc.raw {
				t.Errorf("render(parse(%q)) = %q, want byte-exact round-trip", tc.raw, got)
			}
			// parse → render → parse stability: re-rendering the already-rendered
			// form must reproduce it (the value is fixed under the round-trip).
			if again := tc.render(got); again != got {
				t.Errorf("render(parse(render(parse(%q)))) = %q, want %q (not idempotent)", tc.raw, again, got)
			}
		})
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
