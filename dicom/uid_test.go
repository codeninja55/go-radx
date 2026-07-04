package dicom

import (
	"errors"
	"strings"
	"testing"
)

func TestParseUIDValid(t *testing.T) {
	valid := []string{
		"1.2.840.10008.1.2.1",
		"1.2.840.10008.5.1.4.1.1.2",
		"0",     // single zero component is allowed
		"1.0.2", // single zero between dots is allowed
		"2.25.123456789",
	}
	for _, s := range valid {
		if _, err := ParseUID(s); err != nil {
			t.Errorf("ParseUID(%q) unexpected error: %v", s, err)
		}
	}
}

func TestParseUIDInvalid(t *testing.T) {
	invalid := []string{
		"",                      // empty
		"1..2",                  // empty component (DCM-009)
		"1.02",                  // leading zero in multi-digit component (DCM-009)
		"1.2.",                  // trailing dot
		".1.2",                  // leading dot
		"1.2.a",                 // non-numeric
		strings.Repeat("1", 65), // over 64 characters
	}
	for _, s := range invalid {
		if _, err := ParseUID(s); err == nil {
			t.Errorf("ParseUID(%q) = nil error, want rejection", s)
		}
	}
}

func TestUIDName(t *testing.T) {
	if got := UID("1.2.840.10008.1.2.1").Name(); got != "Explicit VR Little Endian" {
		t.Errorf("Name() = %q, want Explicit VR Little Endian", got)
	}
	// Unregistered UID returns itself.
	if got := UID("1.2.3.4.5").Name(); got != "1.2.3.4.5" {
		t.Errorf("Name() = %q, want the UID itself", got)
	}
}

func TestUIDValidateErrorIsTyped(t *testing.T) {
	_, err := ParseUID("1..2")
	if err == nil {
		t.Fatal("want error")
	}
	if _, ok := errors.AsType[*ValueError](err); !ok {
		t.Errorf("want *ValueError, got %T", err)
	}
}

func TestSOPIdentifierTypesInheritValidation(t *testing.T) {
	sc := SOPClassUID("1.2.840.10008.5.1.4.1.1.2")
	if err := UID(sc).Validate(); err != nil {
		t.Errorf("SOPClassUID should validate through UID conversion: %v", err)
	}
	if UID(sc).Name() != "CT Image Storage" {
		t.Errorf("Name() = %q, want CT Image Storage", UID(sc).Name())
	}
	si := SOPInstanceUID("1..2") // invalid
	if UID(si).IsValid() {
		t.Error("invalid SOPInstanceUID should not validate")
	}
}
