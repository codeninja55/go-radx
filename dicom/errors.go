// Package dicom provides DICOM file parsing and manipulation.
package dicom

import "errors"

// ErrInvalidPreamble indicates the file doesn't have a valid DICOM preamble.
// A valid DICOM file must have 128 bytes followed by "DICM" (ASCII).
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part10.html#sect_7.1
var ErrInvalidPreamble = errors.New("invalid DICOM preamble: missing or invalid DICM prefix")

// ErrInvalidVR indicates an invalid or unknown VR was encountered.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
var ErrInvalidVR = errors.New("invalid or unknown VR")

// ErrInvalidTag indicates a malformed tag was encountered.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.1
var ErrInvalidTag = errors.New("invalid or malformed tag")

// ErrInvalidTransferSyntax indicates an unsupported or invalid transfer syntax.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#chapter_10
var ErrInvalidTransferSyntax = errors.New("invalid or unsupported transfer syntax")

// ErrMissingTransferSyntax indicates the Transfer Syntax UID was not found in File Meta Information.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part10.html#sect_7.1
var ErrMissingTransferSyntax = errors.New("missing Transfer Syntax UID in File Meta Information")

// ErrInvalidLength indicates an invalid value length was encountered.
var ErrInvalidLength = errors.New("invalid value length")

// ErrUndefinedLength indicates an undefined length (0xFFFFFFFF) was encountered.
// This is valid for sequences but requires special handling.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.5
var ErrUndefinedLength = errors.New("undefined length encountered")
