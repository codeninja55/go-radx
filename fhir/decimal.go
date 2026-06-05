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
// FHIR decimal production and preserving the source form verbatim. An empty or
// malformed input returns an error.
//
// The FHIR decimal production places no length limit on a value, so ParseDecimal
// uses the shared lexical parser without the DICOM DS 16-byte cap (PS3.5): a long
// but well-formed FHIR decimal is accepted here. The DS cap is a DICOM VR write
// constraint enforced by dicom.ParseDecimal at the DICOM value boundary, not a
// property of the shared lexical Decimal.
func ParseDecimal(s string) (Decimal, error) { return dicom.ParseDecimalLexical(s) }
