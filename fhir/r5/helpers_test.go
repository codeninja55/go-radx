package r5_test

// strptr and strPtr return a pointer to their argument, the pointer-to-scalar
// constructors the r5 package tests use to populate optional FHIR fields. Both
// spellings exist because the test files were authored against both; they are kept
// as a single shared definition so the helpers live in one place.
func strptr(s string) *string { return &s }
func strPtr(s string) *string { return &s }

// boolPtr returns a pointer to its argument, for optional boolean FHIR fields.
func boolPtr(b bool) *bool { return &b }
