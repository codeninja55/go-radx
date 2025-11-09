// Package value provides DICOM element value representations and operations.
//
// Values in DICOM can be strings, bytes, integers, floats, or sequences.
// Each value type corresponds to one or more Value Representations (VRs).
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
package value

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/codeninja55/go-radx/dicom/vr"
)

// Value represents a DICOM element value.
// Different VRs have different value representations (strings, bytes, integers, etc.).
type Value interface {
	// VR returns the Value Representation of this value
	VR() vr.VR

	// Bytes returns the raw byte encoding of this value
	Bytes() []byte

	// String returns a human-readable string representation
	String() string

	// Equals returns true if this value equals another value
	Equals(other Value) bool
}

// StringValue represents string-based DICOM values.
// Supports VRs: AE, AS, CS, DA, DS, DT, IS, LO, LT, PN, SH, ST, TM, UC, UI, UR, UT
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
type StringValue struct {
	vr     vr.VR
	values []string
}

// maxLengths defines maximum character lengths for string VRs per DICOM standard
var maxLengths = map[vr.VR]int{
	vr.ApplicationEntity:           16,    // AE
	vr.AgeString:                   4,     // AS
	vr.CodeString:                  16,    // CS
	vr.Date:                        8,     // DA
	vr.DecimalString:               16,    // DS
	vr.DateTime:                    26,    // DT
	vr.IntegerString:               12,    // IS
	vr.LongString:                  64,    // LO
	vr.LongText:                    10240, // LT
	vr.PersonName:                  64,    // PN (per component group)
	vr.ShortString:                 16,    // SH
	vr.ShortText:                   1024,  // ST
	vr.Time:                        14,    // TM
	vr.UnlimitedCharacters:         0,     // UC (unlimited)
	vr.UniqueIdentifier:            64,    // UI
	vr.UniversalResourceIdentifier: 0,     // UR (unlimited, but practical limit)
	vr.UnlimitedText:               0,     // UT (unlimited)
}

// isStringVR returns true if the VR is a string-based type
func isStringVR(v vr.VR) bool {
	_, ok := maxLengths[v]
	return ok
}

// NewStringValue creates a new StringValue with the specified VR and values.
// Returns an error if the VR is not a string type or if values exceed the maximum length.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func NewStringValue(v vr.VR, values []string) (*StringValue, error) {
	// Validate VR is a string type
	if !isStringVR(v) {
		return nil, fmt.Errorf("VR %s is not a string type", v.String())
	}

	// Validate lengths if there's a max length defined
	if maxLen, ok := maxLengths[v]; ok && maxLen > 0 {
		for _, val := range values {
			if len(val) > maxLen {
				return nil, fmt.Errorf("value %q exceeds maximum length %d for VR %s", val, maxLen, v.String())
			}
		}
	}

	return &StringValue{
		vr:     v,
		values: values,
	}, nil
}

// VR returns the Value Representation of this string value
func (s *StringValue) VR() vr.VR {
	return s.vr
}

// Strings return the string values as a slice
func (s *StringValue) Strings() []string {
	return s.values
}

// String returns a human-readable string representation.
// Multiple values are separated by backslash (\).
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func (s *StringValue) String() string {
	if len(s.values) == 0 {
		return ""
	}
	return strings.Join(s.values, "\\")
}

// Bytes returns the raw byte encoding of this value.
// Multiple values are separated by backslash (\).
// UI values are null-padded if they have odd length.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func (s *StringValue) Bytes() []byte {
	if len(s.values) == 0 {
		return []byte{}
	}

	// Join values with a backslash separator
	result := strings.Join(s.values, "\\")

	// UI values need null padding if odd length
	if s.vr == vr.UniqueIdentifier && len(result)%2 == 1 {
		result += "\x00"
	}

	return []byte(result)
}

// Equals returns true if this value equals another value.
// Compares VR and all string values for equality.
func (s *StringValue) Equals(other Value) bool {
	// Check if other is also a StringValue
	otherStr, ok := other.(*StringValue)
	if !ok {
		return false
	}

	// Compare VRs
	if s.vr != otherStr.vr {
		return false
	}

	// Compare lengths
	if len(s.values) != len(otherStr.values) {
		return false
	}

	// Compare all values
	for i := range s.values {
		if s.values[i] != otherStr.values[i] {
			return false
		}
	}

	return true
}

// Verify StringValue implements Value interface at compile time
var _ Value = (*StringValue)(nil)

// BytesValue represents binary DICOM values.
// Supports VRs: OB, OD, OF, OL, OV, OW, UN
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
type BytesValue struct {
	vr   vr.VR
	data []byte
}

// isBytesVR returns true if the VR is a binary data type
func isBytesVR(v vr.VR) bool {
	switch v {
	case vr.OtherByte, vr.OtherDouble, vr.OtherFloat,
		vr.OtherLong, vr.OtherVeryLong, vr.OtherWord,
		vr.SequenceOfItems, vr.Unknown:
		return true
	default:
		return false
	}
}

// NewBytesValue creates a new BytesValue with the specified VR and data.
// Returns an error if the VR is not a binary type.
// Nil data is treated as empty []byte.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func NewBytesValue(v vr.VR, data []byte) (*BytesValue, error) {
	// Validate VR is a bytes type
	if !isBytesVR(v) {
		return nil, fmt.Errorf("VR %s is not a binary type", v.String())
	}

	// Treat nil as empty
	if data == nil {
		data = []byte{}
	}

	return &BytesValue{
		vr:   v,
		data: data,
	}, nil
}

// VR returns the Value Representation of this byte value
func (b *BytesValue) VR() vr.VR {
	return b.vr
}

// Bytes returns the raw byte data with padding if needed.
// Odd-length byte arrays are null-padded per DICOM standard.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func (b *BytesValue) Bytes() []byte {
	// Return an empty slice for empty data
	if len(b.data) == 0 {
		return []byte{}
	}

	// Pad odd-length bytes with null byte
	if len(b.data)%2 == 1 {
		padded := make([]byte, len(b.data)+1)
		copy(padded, b.data)
		padded[len(b.data)] = 0x00
		return padded
	}

	return b.data
}

// String returns a human-readable hex representation of the bytes.
// Long arrays (>16 bytes) are truncated for readability.
func (b *BytesValue) String() string {
	const maxDisplayBytes = 16

	if len(b.data) == 0 {
		return "[]"
	}

	var result strings.Builder
	result.WriteString("[")

	displayLen := len(b.data)
	truncated := false
	if displayLen > maxDisplayBytes {
		displayLen = maxDisplayBytes
		truncated = true
	}

	for i := 0; i < displayLen; i++ {
		if i > 0 {
			result.WriteString(" ")
		}
		result.WriteString(fmt.Sprintf("%02X", b.data[i]))
	}

	if truncated {
		result.WriteString(fmt.Sprintf(" ... (%d bytes)", len(b.data)))
	}

	result.WriteString("]")
	return result.String()
}

// Equals returns true if this value equals another value.
// Compares VR and byte data for equality.
// Nil and empty byte slices are considered equal.
func (b *BytesValue) Equals(other Value) bool {
	// Check if other is also a BytesValue
	otherBytes, ok := other.(*BytesValue)
	if !ok {
		return false
	}

	// Compare VRs
	if b.vr != otherBytes.vr {
		return false
	}

	// Handle empty/nil cases
	if len(b.data) == 0 && len(otherBytes.data) == 0 {
		return true
	}

	// Compare lengths
	if len(b.data) != len(otherBytes.data) {
		return false
	}

	// Compare all bytes
	for i := range b.data {
		if b.data[i] != otherBytes.data[i] {
			return false
		}
	}

	return true
}

// Verify BytesValue implements Value interface at compile time
var _ Value = (*BytesValue)(nil)

// IntValue represents integer-based DICOM values.
// Supports VRs: SS, US, SL, UL, SV, UV, AT
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
type IntValue struct {
	vr     vr.VR
	values []int64
}

// isIntVR returns true if the VR is an integer type
func isIntVR(v vr.VR) bool {
	switch v {
	case vr.SignedShort, vr.UnsignedShort,
		vr.SignedLong, vr.UnsignedLong,
		vr.SignedVeryLong, vr.UnsignedVeryLong,
		vr.AttributeTag:
		return true
	default:
		return false
	}
}

// validateIntRange checks if a value is within the valid range for its VR
func validateIntRange(v vr.VR, value int64) error {
	switch v {
	case vr.SignedShort:
		// int16: -32768 to 32767
		if value < -32768 || value > 32767 {
			return fmt.Errorf("value %d out of range for SS (int16): [-32768, 32767]", value)
		}
	case vr.UnsignedShort:
		// uint16: 0 to 65535
		if value < 0 || value > 65535 {
			return fmt.Errorf("value %d out of range for US (uint16): [0, 65535]", value)
		}
	case vr.SignedLong:
		// int32: -2147483648 to 2147483647
		if value < -2147483648 || value > 2147483647 {
			return fmt.Errorf("value %d out of range for SL (int32): [-2147483648, 2147483647]", value)
		}
	case vr.UnsignedLong:
		// uint32: 0 to 4294967295
		if value < 0 || value > 4294967295 {
			return fmt.Errorf("value %d out of range for UL (uint32): [0, 4294967295]", value)
		}
	case vr.AttributeTag:
		// uint32: 0 to 4294967295
		if value < 0 || value > 4294967295 {
			return fmt.Errorf("value %d out of range for AT (uint32): [0, 4294967295]", value)
		}
	case vr.SignedVeryLong:
		// int64: all int64 values are valid
	case vr.UnsignedVeryLong:
		// uint64: negative values not allowed
		if value < 0 {
			return fmt.Errorf("value %d out of range for UV (uint64): must be non-negative", value)
		}
	}
	return nil
}

// NewIntValue creates a new IntValue with the specified VR and values.
// Returns an error if the VR is not an integer type or if values are out of range.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func NewIntValue(v vr.VR, values []int64) (*IntValue, error) {
	// Validate VR is an integer type
	if !isIntVR(v) {
		return nil, fmt.Errorf("VR %s is not an integer type", v.String())
	}

	// Validate all values are within range
	for _, val := range values {
		if err := validateIntRange(v, val); err != nil {
			return nil, err
		}
	}

	return &IntValue{
		vr:     v,
		values: values,
	}, nil
}

// VR returns the Value Representation of this integer value
func (i *IntValue) VR() vr.VR {
	return i.vr
}

// Ints returns the integer values as a slice
func (i *IntValue) Ints() []int64 {
	return i.values
}

// String returns a human-readable string representation.
// Multiple values are separated by backslash (\).
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func (i *IntValue) String() string {
	if len(i.values) == 0 {
		return ""
	}

	var parts []string
	for _, val := range i.values {
		parts = append(parts, fmt.Sprintf("%d", val))
	}
	return strings.Join(parts, "\\")
}

// Bytes return the little-endian byte encoding of the integer values.
// - SS/US: 2 bytes per value (int16/uint16)
// - SL/UL/AT: 4 bytes per value (int32/uint32)
// - SV/UV: 8 bytes per value (int64/uint64)
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func (i *IntValue) Bytes() []byte {
	if len(i.values) == 0 {
		return []byte{}
	}

	var bytesPerValue int
	switch i.vr {
	case vr.SignedShort, vr.UnsignedShort:
		bytesPerValue = 2
	case vr.SignedLong, vr.UnsignedLong, vr.AttributeTag:
		bytesPerValue = 4
	case vr.SignedVeryLong, vr.UnsignedVeryLong:
		bytesPerValue = 8
	}

	result := make([]byte, len(i.values)*bytesPerValue)
	offset := 0

	for _, val := range i.values {
		switch i.vr {
		case vr.SignedShort:
			binary.LittleEndian.PutUint16(result[offset:], uint16(int16(val)))
		case vr.UnsignedShort:
			binary.LittleEndian.PutUint16(result[offset:], uint16(val))
		case vr.SignedLong:
			binary.LittleEndian.PutUint32(result[offset:], uint32(int32(val)))
		case vr.UnsignedLong:
			binary.LittleEndian.PutUint32(result[offset:], uint32(val))
		case vr.AttributeTag:
			// AT is encoded as two uint16 values: group and element
			group := uint16((val >> 16) & 0xFFFF)
			element := uint16(val & 0xFFFF)
			binary.LittleEndian.PutUint16(result[offset:], group)
			binary.LittleEndian.PutUint16(result[offset+2:], element)
		case vr.SignedVeryLong:
			binary.LittleEndian.PutUint64(result[offset:], uint64(val))
		case vr.UnsignedVeryLong:
			binary.LittleEndian.PutUint64(result[offset:], uint64(val))
		}
		offset += bytesPerValue
	}

	return result
}

// Equals returns true if this value equals another value.
// Compares VR and all integer values for equality.
func (i *IntValue) Equals(other Value) bool {
	// Check if other is also an IntValue
	otherInt, ok := other.(*IntValue)
	if !ok {
		return false
	}

	// Compare VRs
	if i.vr != otherInt.vr {
		return false
	}

	// Compare lengths
	if len(i.values) != len(otherInt.values) {
		return false
	}

	// Compare all values
	for idx := range i.values {
		if i.values[idx] != otherInt.values[idx] {
			return false
		}
	}

	return true
}

// Verify IntValue implements Value interface at compile time
var _ Value = (*IntValue)(nil)

// FloatValue represents floating-point DICOM values.
// Supports VRs: FL, FD
//
// Special Values:
// DICOM fully supports IEEE 754 special values including NaN, +Infinity, -Infinity
// as these may be meaningful for representing computational results.
//
// Precision Note:
// FL (float32) values may lose precision when converting from float64.
// FL provides ~7 decimal digits of precision, FD provides ~15-16 digits.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
type FloatValue struct {
	vr     vr.VR
	values []float64
}

// isFloatVR returns true if the VR is a floating-point type
func isFloatVR(v vr.VR) bool {
	switch v {
	case vr.FloatingPointSingle, vr.FloatingPointDouble:
		return true
	default:
		return false
	}
}

// NewFloatValue creates a new FloatValue with the specified VR and values.
// Returns an error if the VR is not a float type.
// Supports special IEEE 754 values: NaN, +Infinity, -Infinity
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func NewFloatValue(v vr.VR, values []float64) (*FloatValue, error) {
	// Validate VR is a float type
	if !isFloatVR(v) {
		return nil, fmt.Errorf("VR %s is not a floating-point type", v.String())
	}

	return &FloatValue{
		vr:     v,
		values: values,
	}, nil
}

// VR returns the Value Representation of this float value
func (f *FloatValue) VR() vr.VR {
	return f.vr
}

// Floats returns the float values as a slice
func (f *FloatValue) Floats() []float64 {
	return f.values
}

// String returns a human-readable string representation.
// Multiple values are separated by backslash (\).
// Special values are formatted as: NaN, +Inf, -Inf
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func (f *FloatValue) String() string {
	if len(f.values) == 0 {
		return ""
	}

	parts := make([]string, 0, len(f.values))
	for _, val := range f.values {
		parts = append(parts, formatFloatValue(val))
	}
	return strings.Join(parts, "\\")
}

// formatFloatValue formats a float for string representation
// Handles special values: NaN, +Inf, -Inf
func formatFloatValue(val float64) string {
	switch {
	case math.IsNaN(val):
		return "NaN"
	case math.IsInf(val, 1):
		return "+Inf"
	case math.IsInf(val, -1):
		return "-Inf"
	default:
		// Use 'g' format for automatic scientific notation
		// -1 precision means use minimum digits needed
		return strconv.FormatFloat(val, 'g', -1, 64)
	}
}

// Bytes return the little-endian IEEE 754 byte encoding of the float values.
// - FL: 4 bytes per value (float32)
// - FD: 8 bytes per value (float64)
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func (f *FloatValue) Bytes() []byte {
	if len(f.values) == 0 {
		return []byte{}
	}

	var bytesPerValue int
	if f.vr == vr.FloatingPointSingle {
		bytesPerValue = 4
	} else { // VRFloatingPointDouble
		bytesPerValue = 8
	}

	result := make([]byte, len(f.values)*bytesPerValue)
	offset := 0

	for _, val := range f.values {
		if f.vr == vr.FloatingPointSingle {
			// Convert float64 to float32, then encode as IEEE 754 binary32
			binary.LittleEndian.PutUint32(result[offset:], math.Float32bits(float32(val)))
		} else { // VRFloatingPointDouble
			// Encode as IEEE 754 binary64
			binary.LittleEndian.PutUint64(result[offset:], math.Float64bits(val))
		}
		offset += bytesPerValue
	}

	return result
}

// Equals returns true if this value equals another value.
// Compares VR and all float values for equality.
//
// Note: Per IEEE 754, NaN != NaN. However, for DICOM value comparison purposes,
// we treat two NaN values as equal to enable meaningful equality checks.
func (f *FloatValue) Equals(other Value) bool {
	// Check if other is also a FloatValue
	otherFloat, ok := other.(*FloatValue)
	if !ok {
		return false
	}

	// Compare VRs
	if f.vr != otherFloat.vr {
		return false
	}

	// Compare lengths
	if len(f.values) != len(otherFloat.values) {
		return false
	}

	// Compare all values
	for i := range f.values {
		// Handle special case: treat NaN == NaN for comparison purposes
		if math.IsNaN(f.values[i]) && math.IsNaN(otherFloat.values[i]) {
			continue // Both NaN, consider equal
		}
		if f.values[i] != otherFloat.values[i] {
			return false
		}
	}

	return true
}

// Verify FloatValue implements Value interface at compile time
var _ Value = (*FloatValue)(nil)
