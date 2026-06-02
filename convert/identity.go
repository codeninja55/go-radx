package convert

import (
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// dicomUIDSystem is the URN namespace every DICOM UID identifier carries as its
// FHIR Identifier.system. It is the registered DICOM UID namespace, fixed by the
// glossary (UBIQUITOUS_LANGUAGE.md cross-standard table).
const dicomUIDSystem = "urn:dicom:uid"

// dicomUIDValuePrefix is the OID URN prefix applied to the UID to form the
// Identifier.value. A UID is an ISO object identifier, so its URN form is
// "urn:oid:" + uid.
const dicomUIDValuePrefix = "urn:oid:"

// UIDIdentifierR5 turns a DICOM UID into a FHIR R5 Identifier. This is the single
// most important rule in cross-standard conversion: a DICOM UID is a globally
// unique ISO object identifier, not a server-resolvable pointer, so it ALWAYS
// becomes an Identifier (system "urn:dicom:uid", value "urn:oid:" + uid) and
// NEVER a Reference.reference URL (docs/reference/convert.md "Identity handling").
//
// For example dicom.UID("1.2.3") becomes
//
//	r5.Identifier{ System: "urn:dicom:uid", Value: "urn:oid:1.2.3" }
func UIDIdentifierR5(uid dicom.UID) r5.Identifier {
	system := dicomUIDSystem
	value := dicomUIDValuePrefix + string(uid)
	return r5.Identifier{
		System: &system,
		Value:  &value,
	}
}
