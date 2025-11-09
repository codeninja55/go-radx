package datetime

import "fmt"

// ParseError represents an error that occurred during parsing of a DICOM temporal value.
// It provides context about what was being parsed and why it failed.
type ParseError struct {
	// VR is the DICOM Value Representation being parsed (DA, TM, DT, or AS).
	VR string

	// Input is the string that failed to parse.
	Input string

	// Reason describes why parsing failed.
	Reason string
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	if e.Input == "" {
		return fmt.Sprintf("parse %s: empty input", e.VR)
	}
	return fmt.Sprintf("parse %s %q: %s", e.VR, e.Input, e.Reason)
}

// FormatError represents an error that occurred during formatting of a temporal value
// back to DICOM string format.
type FormatError struct {
	// VR is the DICOM Value Representation being formatted.
	VR string

	// Reason describes why formatting failed.
	Reason string
}

// Error implements the error interface.
func (e *FormatError) Error() string {
	return fmt.Sprintf("format %s: %s", e.VR, e.Reason)
}

// RangeError represents an error where a temporal component value is outside valid DICOM range.
type RangeError struct {
	// Component is the name of the temporal component (e.g., "month", "hour", "day").
	Component string

	// Value is the invalid value.
	Value int

	// Min is the minimum valid value (inclusive).
	Min int

	// Max is the maximum valid value (inclusive).
	Max int
}

// Error implements the error interface.
func (e *RangeError) Error() string {
	return fmt.Sprintf("%s %d out of range [%d, %d]", e.Component, e.Value, e.Min, e.Max)
}

// newParseError creates a new ParseError with the given VR, input, and reason.
func newParseError(vr, input, reason string) error {
	return &ParseError{
		VR:     vr,
		Input:  input,
		Reason: reason,
	}
}

// newFormatError creates a new FormatError with the given VR and reason.
func newFormatError(vr, reason string) error {
	return &FormatError{
		VR:     vr,
		Reason: reason,
	}
}

// newRangeError creates a new RangeError with the given component, value, and valid range.
func newRangeError(component string, value, min, max int) error {
	return &RangeError{
		Component: component,
		Value:     value,
		Min:       min,
		Max:       max,
	}
}
