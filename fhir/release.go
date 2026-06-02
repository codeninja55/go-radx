package fhir

// Release identifies a generated FHIR release. The two v1 releases live in
// sibling packages (fhir/r4 and fhir/r5) as distinct type spaces; this constant
// names the exact HL7 FHIR release version the package was generated from, so a
// consumer can assert the release at runtime. The value is the FHIR release
// version string (for example "5.0.0"), not the go-radx library version.
//
// R6 is not generated in v1; the type is left open (a plain string) so a future
// release adds a constant without changing the type.
type Release string

const (
	// R4 is HL7 FHIR R4, release version 4.0.1.
	R4 Release = "4.0.1"

	// R5 is HL7 FHIR R5, release version 5.0.0.
	R5 Release = "5.0.0"
)

// String returns the FHIR release version string.
func (r Release) String() string { return string(r) }
