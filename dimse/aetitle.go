package dimse

import (
	"fmt"
	"strings"
)

// aeTitleMaxLength is the DICOM AE Title field width (PS3.5, VR AE): a value of at most
// 16 characters. Leading and trailing spaces are insignificant and not counted.
const aeTitleMaxLength = 16

// AETitle is a DICOM Application Entity Title: 1..16 characters of the DICOM default
// character repertoire, with no significant leading or trailing padding (PS3.5, VR AE).
// It is a named type, never a bare string, so a signature states the role (PRD §8.2).
type AETitle string

// ParseAETitle validates s as an AE Title and returns it with insignificant surrounding
// spaces trimmed. Per PS3.5 (VR AE) and the pynetdicom validator the title must be 1..16
// characters of ASCII excluding backslash (0x5C, the DICOM value delimiter) and all
// control characters; a value that is empty or all spaces, longer than 16 characters, or
// uses a character outside the repertoire is rejected with a typed *ValidationError. The
// error names the violated constraint, never any patient value (PRD §9.1).
func ParseAETitle(s string) (AETitle, error) {
	trimmed := strings.Trim(s, " ")
	if trimmed == "" {
		return "", &ValidationError{Detail: "AE Title is empty or all spaces (must be 1..16 characters)"}
	}
	if len(trimmed) > aeTitleMaxLength {
		return "", &ValidationError{
			Detail: fmt.Sprintf("AE Title length %d exceeds the %d-character maximum", len(trimmed), aeTitleMaxLength),
		}
	}
	if i, c, ok := firstDisallowedAEChar(trimmed); ok {
		return "", &ValidationError{
			Detail: fmt.Sprintf("AE Title has a disallowed character (code 0x%02X) at position %d", c, i),
		}
	}
	return AETitle(trimmed), nil
}

// firstDisallowedAEChar reports the first character of s that is not permitted in an AE
// Title — a non-ASCII byte, a control character (< 0x20 or 0x7F), or the backslash
// delimiter (0x5C) — returning its index, its byte value, and whether one was found.
func firstDisallowedAEChar(s string) (int, byte, bool) {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 0x80 || c < 0x20 || c == 0x7F || c == '\\' {
			return i, c, true
		}
	}
	return 0, 0, false
}

// String returns the AE Title text.
func (a AETitle) String() string { return string(a) }

// Valid reports whether a is a conformant AE Title (the condition ParseAETitle enforces).
func (a AETitle) Valid() bool {
	_, err := ParseAETitle(string(a))
	return err == nil
}
