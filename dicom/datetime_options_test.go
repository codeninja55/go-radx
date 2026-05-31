package dicom

import "testing"

func TestWithLenientDatesSetsReadConfig(t *testing.T) {
	if newReadConfig().lenientDates {
		t.Error("lenientDates should default false (strict parsing is the default)")
	}
	if !newReadConfig(WithLenientDates()).lenientDates {
		t.Error("WithLenientDates should set lenientDates true")
	}
}

func TestDateOptionLenientToggle(t *testing.T) {
	var strict dateConfig
	if strict.lenient {
		t.Error("a zero dateConfig must be strict")
	}
	var lenient dateConfig
	withLenient()(&lenient)
	if !lenient.lenient {
		t.Error("withLenient must set lenient true")
	}
}

func TestDateModeConstantsAreDistinct(t *testing.T) {
	if DateModeKeep == DateModeShift {
		t.Error("DateModeKeep and DateModeShift must be distinct values")
	}
	// DateModeKeep is the zero value so a default Profile retains dates verbatim
	// rather than shifting them when temporal retention is opted in.
	var zero DateMode
	if zero != DateModeKeep {
		t.Errorf("zero DateMode = %v, want DateModeKeep", zero)
	}
}
