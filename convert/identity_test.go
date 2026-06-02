package convert

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

// TestUIDIdentifierR5 is the absolute identity rule: a DICOM UID becomes a FHIR
// Identifier under urn:dicom:uid with an urn:oid value, never a Reference URL.
func TestUIDIdentifierR5(t *testing.T) {
	id := UIDIdentifierR5(dicom.UID("1.2.3"))

	if id.System == nil || *id.System != "urn:dicom:uid" {
		t.Errorf("Identifier.System = %v, want urn:dicom:uid", deref(id.System))
	}
	if id.Value == nil || *id.Value != "urn:oid:1.2.3" {
		t.Errorf("Identifier.Value = %v, want urn:oid:1.2.3", deref(id.Value))
	}
}

// TestUIDIdentifierR5RealStudyUID exercises a realistic Study Instance UID.
func TestUIDIdentifierR5RealStudyUID(t *testing.T) {
	const uid = "1.2.840.113619.2.55.3.604688.1"
	id := UIDIdentifierR5(dicom.UID(uid))

	if id.Value == nil || *id.Value != "urn:oid:"+uid {
		t.Errorf("Identifier.Value = %v, want urn:oid:%s", deref(id.Value), uid)
	}
}

// deref renders a *string for an error message without panicking on nil.
func deref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
