// Package vr defines DICOM Value Representations (VRs) and their properties.
//
// Value Representations specify the data type and format of DICOM element values.
// Each VR has specific encoding rules, padding requirements, and length constraints.
//
// See DICOM Part 5, Section 6.2:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
package vr

import (
	"fmt"
)

// VR represents a DICOM Value Representation type.
// Each VR defines how element values are encoded and interpreted.
type VR uint8

// Standard DICOM Value Representations as defined in Part 5, Section 6.2.
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
const (
	// ApplicationEntity (AE) - Application Entity title (string, max 16 chars, space-padded)
	ApplicationEntity VR = iota + 1
	// AgeString (AS) - Age in format nnnW, nnnM, nnnY (string, fixed 4 chars, space-padded)
	AgeString
	// AttributeTag (AT) - Tag (4 bytes, group-element pair)
	AttributeTag
	// CodeString (CS) - Code value (string, max 16 chars, space-padded, uppercase)
	CodeString
	// Date (DA) - Date in format YYYYMMDD (string, 8 chars, space-padded)
	Date
	// DecimalString (DS) - Decimal number as string (string, max 16 chars, space-padded)
	DecimalString
	// DateTime (DT) - Date and time (string, max 26 chars, space-padded)
	DateTime
	// FloatingPointDouble (FD) - 64-bit floating point (8 bytes)
	FloatingPointDouble
	// FloatingPointSingle (FL) - 32-bit floating point (4 bytes)
	FloatingPointSingle
	// IntegerString (IS) - Integer as string (string, max 12 chars, space-padded)
	IntegerString
	// LongString (LO) - Character string (string, max 64 chars, space-padded)
	LongString
	// LongText (LT) - Text (string, max 10240 chars, space-padded)
	LongText
	// OtherByte (OB) - Byte string (binary, variable length, null-padded)
	OtherByte
	// OtherDouble (OD) - 64-bit floating point array (binary, variable length, null-padded)
	OtherDouble
	// OtherFloat (OF) - 32-bit floating point array (binary, variable length, null-padded)
	OtherFloat
	// OtherLong (OL) - 32-bit integer array (binary, variable length, null-padded)
	OtherLong
	// OtherVeryLong (OV) - 64-bit integer array (binary, variable length, null-padded)
	OtherVeryLong
	// OtherWord (OW) - 16-bit integer array (binary, variable length, null-padded)
	OtherWord
	// PersonName (PN) - Person's name in format Last^First^Middle^Prefix^Suffix (string, max 324 chars, space-padded)
	PersonName
	// ShortString (SH) - Short character string (string, max 16 chars, space-padded)
	ShortString
	// SignedLong (SL) - Signed 32-bit integer (4 bytes)
	SignedLong
	// SequenceOfItems (SQ) - Sequence containing nested datasets (structured data)
	SequenceOfItems
	// SignedShort (SS) - Signed 16-bit integer (2 bytes)
	SignedShort
	// ShortText (ST) - Short text (string, max 1024 chars, space-padded)
	ShortText
	// SignedVeryLong (SV) - Signed 64-bit integer (8 bytes)
	SignedVeryLong
	// Time (TM) - Time in format HHMMSS.FFFFFF (string, max 14 chars, space-padded)
	Time
	// UnlimitedCharacters (UC) - Unlimited length character string (string, unlimited, space-padded)
	UnlimitedCharacters
	// UniqueIdentifier (UI) - UID in dotted notation (string, max 64 chars, null-padded)
	UniqueIdentifier
	// UnsignedLong (UL) - Unsigned 32-bit integer (4 bytes)
	UnsignedLong
	// Unknown (UN) - Unknown value type (binary, variable length, null-padded)
	Unknown
	// UniversalResourceIdentifier (UR) - URI or URL (string, unlimited, space-padded)
	UniversalResourceIdentifier
	// UnsignedShort (US) - Unsigned 16-bit integer (2 bytes)
	UnsignedShort
	// UnlimitedText (UT) - Unlimited length text (string, unlimited, space-padded)
	UnlimitedText
	// UnsignedVeryLong (UV) - Unsigned 64-bit integer (8 bytes)
	UnsignedVeryLong
)

// vrStrings maps VR constants to their string representations.
var vrStrings = map[VR]string{
	ApplicationEntity: "AE", AgeString: "AS", AttributeTag: "AT", CodeString: "CS",
	Date: "DA", DecimalString: "DS", DateTime: "DT", FloatingPointDouble: "FD",
	FloatingPointSingle: "FL", IntegerString: "IS", LongString: "LO", LongText: "LT",
	OtherByte: "OB", OtherDouble: "OD", OtherFloat: "OF", OtherLong: "OL",
	OtherVeryLong: "OV", OtherWord: "OW", PersonName: "PN", ShortString: "SH",
	SignedLong: "SL", SequenceOfItems: "SQ", SignedShort: "SS", ShortText: "ST",
	SignedVeryLong: "SV", Time: "TM", UnlimitedCharacters: "UC", UniqueIdentifier: "UI",
	UnsignedLong: "UL", Unknown: "UN", UniversalResourceIdentifier: "UR", UnsignedShort: "US",
	UnlimitedText: "UT", UnsignedVeryLong: "UV",
}

// stringToVR maps string representations to VR constants.
var stringToVR = map[string]VR{
	"AE": ApplicationEntity, "AS": AgeString, "AT": AttributeTag, "CS": CodeString,
	"DA": Date, "DS": DecimalString, "DT": DateTime, "FD": FloatingPointDouble,
	"FL": FloatingPointSingle, "IS": IntegerString, "LO": LongString, "LT": LongText,
	"OB": OtherByte, "OD": OtherDouble, "OF": OtherFloat, "OL": OtherLong,
	"OV": OtherVeryLong, "OW": OtherWord, "PN": PersonName, "SH": ShortString,
	"SL": SignedLong, "SQ": SequenceOfItems, "SS": SignedShort, "ST": ShortText,
	"SV": SignedVeryLong, "TM": Time, "UC": UnlimitedCharacters, "UI": UniqueIdentifier,
	"UL": UnsignedLong, "UN": Unknown, "UR": UniversalResourceIdentifier, "US": UnsignedShort,
	"UT": UnlimitedText, "UV": UnsignedVeryLong,
}

// String returns the two-character string representation of the VR.
func (v VR) String() string {
	if s, ok := vrStrings[v]; ok {
		return s
	}
	return "UN"
}

// IsValid returns true if the given string is a valid VR identifier.
func IsValid(s string) bool {
	_, ok := stringToVR[s]
	return ok
}

// Parse parses a two-character VR string and returns the corresponding VR constant.
func Parse(s string) (VR, error) {
	if v, ok := stringToVR[s]; ok {
		return v, nil
	}
	return 0, fmt.Errorf("invalid VR: %q", s)
}

// UsesExplicitLength32 returns true if this VR requires a 32-bit value length field
// in explicit VR encoding, as opposed to the standard 16-bit length.
//
// See DICOM Part 5, Section 7.1.2:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.1.2
func (v VR) UsesExplicitLength32() bool {
	switch v {
	case OtherByte, OtherDouble, OtherFloat, OtherLong, OtherVeryLong, OtherWord,
		SequenceOfItems, UnlimitedCharacters, Unknown, UniversalResourceIdentifier, UnlimitedText:
		return true
	default:
		return false
	}
}

// PaddingByte returns the byte used for padding odd-length values for this VR.
// String VRs use space (0x20) padding, while binary VRs use null (0x00) padding.
//
// See DICOM Part 5, Section 6.2:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func (v VR) PaddingByte() byte {
	switch v {
	case UniqueIdentifier, OtherByte, OtherDouble, OtherFloat, OtherLong, OtherVeryLong, OtherWord, Unknown:
		return 0x00
	default:
		return ' '
	}
}

// MaxLength returns the maximum allowed length in bytes for this VR.
// Returns 0 for VRs with unlimited length.
//
// See DICOM Part 5, Section 6.2:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func (v VR) MaxLength() int {
	switch v {
	case ApplicationEntity:
		return 16
	case AgeString:
		return 4
	case CodeString:
		return 16
	case Date:
		return 8
	case DecimalString:
		return 16
	case DateTime:
		return 26
	case IntegerString:
		return 12
	case LongString:
		return 64
	case LongText:
		return 10240
	case PersonName:
		return 324
	case ShortString:
		return 16
	case ShortText:
		return 1024
	case Time:
		return 14
	case UniqueIdentifier:
		return 64
	case UnlimitedCharacters, UniversalResourceIdentifier, UnlimitedText,
		OtherByte, OtherDouble, OtherFloat, OtherLong, OtherVeryLong, OtherWord,
		SequenceOfItems, Unknown:
		return 0 // unlimited
	default:
		return 0
	}
}

// AllowsBackslash returns true if this VR allows backslash characters within its value.
// Person Name (PN) uses backslash as a component separator.
//
// See DICOM Part 5, Section 6.2.1:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2.1
func (v VR) AllowsBackslash() bool {
	return v == PersonName
}

// IsStringType returns true if this VR represents character string data.
func (v VR) IsStringType() bool {
	switch v {
	case ApplicationEntity, AgeString, CodeString, Date, DecimalString, DateTime,
		IntegerString, LongString, LongText, PersonName, ShortString, ShortText,
		Time, UnlimitedCharacters, UniqueIdentifier, UniversalResourceIdentifier, UnlimitedText:
		return true
	default:
		return false
	}
}

// IsBinaryType returns true if this VR represents binary data.
func (v VR) IsBinaryType() bool {
	switch v {
	case OtherByte, OtherDouble, OtherFloat, OtherLong, OtherVeryLong, OtherWord, Unknown:
		return true
	default:
		return false
	}
}

// IsNumericType returns true if this VR represents numeric data (integers or floats).
func (v VR) IsNumericType() bool {
	switch v {
	case SignedShort, UnsignedShort, SignedLong, UnsignedLong,
		SignedVeryLong, UnsignedVeryLong, FloatingPointSingle, FloatingPointDouble:
		return true
	default:
		return false
	}
}
