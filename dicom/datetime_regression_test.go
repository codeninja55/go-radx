package dicom

import "testing"

// TestDCM010DateTimeStrictness is the named regression for Codex DCM-010: the
// prototype accepted DA as YYYY/YYYYMM unconditionally and rejected the leap-second
// SS=60 for TM/DT. The fix is strict-by-default DA, opt-in lenient partial forms, and
// leap-second acceptance with documented :59 normalisation for TM and DT.
func TestDCM010DateTimeStrictness(t *testing.T) {
	t.Run("DA rejects partial forms by default", func(t *testing.T) {
		for _, s := range []string{"2024", "202402"} {
			if _, err := ParseDA(s); err == nil {
				t.Errorf("ParseDA(%q) strict accepted a partial date (regression)", s)
			}
		}
	})

	t.Run("DA accepts partial forms only under lenient option", func(t *testing.T) {
		for _, s := range []string{"2024", "202402"} {
			if _, err := ParseDA(s, withLenient()); err != nil {
				t.Errorf("ParseDA(%q, lenient): %v", s, err)
			}
		}
	})

	t.Run("TM accepts leap second and normalises to 59", func(t *testing.T) {
		tm, err := ParseTM("235960")
		if err != nil {
			t.Fatalf("ParseTM(235960): %v", err)
		}
		if tm.String() != "235960" {
			t.Errorf("TM lexical = %q, want preserved 235960", tm.String())
		}
		if got, _ := tm.Time(); got.Second() != 59 {
			t.Errorf("TM Time().Second() = %d, want 59", got.Second())
		}
	})

	t.Run("DT accepts leap second and normalises to 59", func(t *testing.T) {
		dt, err := ParseDT("20240630235960")
		if err != nil {
			t.Fatalf("ParseDT leap second: %v", err)
		}
		if got, _ := dt.Time(); got.Second() != 59 {
			t.Errorf("DT Time().Second() = %d, want 59", got.Second())
		}
	})

	t.Run("DT parses signed timezone offset", func(t *testing.T) {
		for _, c := range []struct {
			in   string
			secs int
		}{
			{"20240101000000+1000", 10 * 3600},
			{"20240101000000-0530", -(5*3600 + 30*60)},
		} {
			dt, err := ParseDT(c.in)
			if err != nil {
				t.Fatalf("ParseDT(%q): %v", c.in, err)
			}
			if !dt.HasOffset() || dt.OffsetSeconds() != c.secs {
				t.Errorf("ParseDT(%q) offset = %d, want %d", c.in, dt.OffsetSeconds(), c.secs)
			}
		}
	})

	t.Run("variable-precision DT round-trips its source form", func(t *testing.T) {
		for _, in := range []string{"2024", "202402291530", "20240229153045.123+0930"} {
			dt, err := ParseDT(in)
			if err != nil {
				t.Fatalf("ParseDT(%q): %v", in, err)
			}
			if dt.String() != in {
				t.Errorf("DT round-trip = %q, want %q", dt.String(), in)
			}
		}
	})
}
