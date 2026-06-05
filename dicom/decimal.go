package dicom

import (
	"fmt"
	"math/big"
	"strings"
)

// maxDSLen is the PS3.5 byte cap for a single DS value.
const maxDSLen = 16

// Decimal is the lexical-preserving numeric type shared by FHIR decimal and DICOM
// DS/IS. It carries the source string so a value read from a file serialises back
// byte-identically. It performs no in-place arithmetic; conversion to a Go numeric
// is explicit and may report inexactness.
type Decimal struct {
	lexical string     // preserved source form
	val     *big.Float // parsed once on construction
}

// ParseDecimal validates s as a DICOM DS lexical form and preserves it verbatim.
// DS is limited to 16 bytes per value (PS3.5), so the DICOM value codec and the
// DICOMweb JSON reader call this path to keep that VR constraint at the DICOM
// boundary.
func ParseDecimal(s string) (Decimal, error) {
	if len(s) > maxDSLen {
		return Decimal{}, &ValueError{VR: VRDS, Msg: fmt.Sprintf("DS value exceeds 16 bytes (%d)", len(s))}
	}
	return ParseDecimalLexical(s)
}

// ParseDecimalLexical validates s against the lexical decimal production and
// preserves it verbatim, without the DS 16-byte cap. The FHIR decimal production
// places no length limit on a value, so the FHIR side (fhir.ParseDecimal and the
// shared Decimal.UnmarshalJSON) parses through here; the DICOM DS 16-byte cap is a
// VR write constraint enforced by ParseDecimal at the DICOM boundary, not a
// property of the shared lexical type. A long-but-valid FHIR decimal is accepted
// here and rejected by ParseDecimal only when it must fit a DS value.
func ParseDecimalLexical(s string) (Decimal, error) {
	if s == "" {
		return Decimal{}, &ValueError{VR: VRDS, Msg: "decimal is empty"}
	}
	// big.Float parses the standard decimal/exponent forms; reject anything it cannot.
	bf, _, err := big.ParseFloat(s, 10, 256, big.ToNearestEven)
	if err != nil {
		return Decimal{}, &ValueError{VR: VRDS, Msg: fmt.Sprintf("not a decimal: %q", s)}
	}
	return Decimal{lexical: s, val: bf}, nil
}

// String returns the preserved lexical form.
func (d Decimal) String() string { return d.lexical }

// Float64 returns the value as a float64. ok is false only when the lexical form has
// no finite float64 representation; a representable-but-rounded value returns ok == true.
func (d Decimal) Float64() (float64, bool) {
	if d.val == nil {
		return 0, false
	}
	f, acc := d.val.Float64()
	if (f == 0 && acc == big.Below) || f != f { // NaN guard
		return 0, false
	}
	if f > 1.7976931348623157e308 || f < -1.7976931348623157e308 {
		return 0, false
	}
	return f, true
}

// Exact reports whether the float64 from Float64 represents d without rounding loss.
func (d Decimal) Exact() bool {
	if d.val == nil {
		return false
	}
	_, acc := d.val.Float64()
	return acc == big.Exact
}

// BigFloat returns a copy of the parsed *big.Float with precision sufficient for the
// lexical form, for callers that need exactness or their own rounding.
func (d Decimal) BigFloat() *big.Float {
	if d.val == nil {
		return new(big.Float)
	}
	return new(big.Float).Copy(d.val)
}

// Int64 returns the integral value. ok is false if d is not integral.
func (d Decimal) Int64() (int64, bool) {
	if d.val == nil || !d.val.IsInt() {
		return 0, false
	}
	n, acc := d.val.Int64()
	if acc != big.Exact {
		return 0, false
	}
	return n, true
}

// MarshalJSON emits the preserved lexical form, unquoted, for FHIR decimal.
func (d Decimal) MarshalJSON() ([]byte, error) {
	if d.lexical == "" {
		return []byte("null"), nil
	}
	return []byte(d.lexical), nil
}

// UnmarshalJSON preserves the raw token's lexical form (trimming any quotes a lenient
// producer added). It parses through the lexical production without the DS 16-byte
// cap so a long-but-valid FHIR decimal round-trips; the DS cap is a DICOM VR write
// constraint applied at the DICOM value boundary (ParseDecimal), not on JSON decode
// of the shared lexical type.
func (d *Decimal) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	parsed, err := ParseDecimalLexical(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}
