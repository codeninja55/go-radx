// Package testutil provides utility functions for FHIR testing.
package testutil

// StringPtr returns a pointer to the given string value.
// This is a test utility function for creating pointers to literal strings.
func StringPtr(s string) *string {
	return &s
}

// BoolPtr returns a pointer to the given bool value.
// This is a test utility function for creating pointers to literal bools.
func BoolPtr(b bool) *bool {
	return &b
}

// IntPtr returns a pointer to the given int value.
// This is a test utility function for creating pointers to literal ints.
func IntPtr(i int) *int {
	return &i
}

// Int64Ptr returns a pointer to the given int64 value.
// This is a test utility function for creating pointers to literal int64s.
func Int64Ptr(i int64) *int64 {
	return &i
}

// Float64Ptr returns a pointer to the given float64 value.
// This is a test utility function for creating pointers to literal float64s.
func Float64Ptr(f float64) *float64 {
	return &f
}
