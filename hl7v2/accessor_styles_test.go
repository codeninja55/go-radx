package hl7v2

import "testing"

// TestParseAccessorRejectsMixedStyles verifies a key that mixes the numeric and
// prefixed level styles within one key is rejected: the style is fixed by the
// first level and every later level must match it.
func TestParseAccessorRejectsMixedStyles(t *testing.T) {
	for _, key := range []string{"PID.F5.1", "PID-5.R1", "PID.F5.R1.2", "PID-5-1.C2"} {
		if _, err := ParseAccessor(key); err == nil {
			t.Errorf("ParseAccessor(%q) = nil error, want a mixed-style AccessorError", key)
		}
	}
}

// TestAccessorStringSegmentOnlyRoundTrips verifies a segment-only accessor renders
// without a field suffix, so the canonical form round-trips through ParseAccessor
// and keeps addressing the segment (Field 0), never field 1.
func TestAccessorStringSegmentOnlyRoundTrips(t *testing.T) {
	for _, key := range []string{"PID", "PID2", "OBR3"} {
		a, err := ParseAccessor(key)
		if err != nil {
			t.Fatalf("ParseAccessor(%q) error = %v", key, err)
		}
		if got := a.String(); got != key {
			t.Errorf("ParseAccessor(%q).String() = %q, want %q", key, got, key)
		}
		round, err := ParseAccessor(a.String())
		if err != nil {
			t.Fatalf("re-parse %q error = %v", a.String(), err)
		}
		if round.Field != 0 {
			t.Errorf("round-tripped %q has Field = %d, want 0 (segment-only)", key, round.Field)
		}
	}
}
