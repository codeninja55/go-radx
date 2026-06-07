package convert

import (
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r4"
)

// UIDIdentifierR4 turns a DICOM UID into a FHIR R4 Identifier, the R4 twin of
// UIDIdentifierR5. The Identifier datatype is identical across R4 and R5, but the
// type lives in each release sub-package, so the rule is mirrored rather than
// shared: a DICOM UID is a globally unique ISO object identifier, not a
// server-resolvable pointer, so it ALWAYS becomes an Identifier (system
// "urn:dicom:uid", value "urn:oid:" + uid) and NEVER a Reference.reference URL
// (docs/reference/convert.md "Identity handling").
func UIDIdentifierR4(uid dicom.UID) r4.Identifier {
	system := dicomUIDSystem
	value := dicomUIDValuePrefix + string(uid)
	return r4.Identifier{
		System: &system,
		Value:  &value,
	}
}
