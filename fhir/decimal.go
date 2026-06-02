package fhir

import "github.com/codeninja55/go-radx/dicom"

// Decimal preserves the source lexical form of a FHIR decimal, so 1.20 and 1.2
// stay distinguishable and trailing zeros survive a round-trip. It is an alias
// for dicom.Decimal: the lexical-preserving numeric type is shared between the
// two standards (the glossary's single Decimal noun), so a value read on the
// DICOM side and one carried on the FHIR side are the same type with one
// implementation of the lexical rules. Conversion to a machine number is
// explicit; the type performs no in-place arithmetic. Its MarshalJSON emits the
// preserved lexical form as an unquoted JSON number.
type Decimal = dicom.Decimal

// ParseDecimal builds a Decimal from a lexical string, validating it against the
// decimal production and preserving the source form verbatim. An empty or
// malformed input returns an error.
//
// TODO(M6a): the shared dicom.ParseDecimal enforces the DICOM DS 16-byte cap
// (PS3.5), which is stricter than the FHIR decimal production. The dicom/fhir
// Decimal unification at M6a (see docs/plans walking-skeleton Open question 1)
// must split the length rule so a long-but-valid FHIR decimal is not rejected by
// the DS cap. M2 carries no FHIR decimal through any converter, so this edge is
// unreachable in the M2 slice.
func ParseDecimal(s string) (Decimal, error) { return dicom.ParseDecimal(s) }
