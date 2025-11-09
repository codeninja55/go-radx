package pixel

import (
	"errors"
	"fmt"
)

var (
	// ErrPixelDataNotFound indicates that the PixelData element (7FE0,0010) is missing from the dataset.
	ErrPixelDataNotFound = errors.New("pixel data element not found")

	// ErrUnsupportedTransferSyntax indicates that no decoder is registered for the transfer syntax.
	ErrUnsupportedTransferSyntax = errors.New("unsupported transfer syntax")

	// ErrInvalidPixelData indicates that pixel data is malformed or inconsistent with metadata.
	ErrInvalidPixelData = errors.New("invalid pixel data")

	// ErrMissingRequiredAttribute indicates that a required DICOM attribute is missing.
	ErrMissingRequiredAttribute = errors.New("missing required attribute")

	// ErrDecompressionFailed indicates that pixel data decompression failed.
	ErrDecompressionFailed = errors.New("decompression failed")
)

// TransferSyntaxError wraps ErrUnsupportedTransferSyntax with the specific UID.
type TransferSyntaxError struct {
	UID string
}

func (e *TransferSyntaxError) Error() string {
	return fmt.Sprintf("%s: %s", ErrUnsupportedTransferSyntax.Error(), e.UID)
}

func (e *TransferSyntaxError) Unwrap() error {
	return ErrUnsupportedTransferSyntax
}

// DecompressionError wraps ErrDecompressionFailed with details about the failure.
type DecompressionError struct {
	TransferSyntaxUID string
	Cause             error
}

func (e *DecompressionError) Error() string {
	return fmt.Sprintf("%s for transfer syntax %s: %v", ErrDecompressionFailed.Error(), e.TransferSyntaxUID, e.Cause)
}

func (e *DecompressionError) Unwrap() error {
	return ErrDecompressionFailed
}

// PixelDataError wraps ErrInvalidPixelData with details about what's invalid.
type PixelDataError struct {
	Field    string
	Expected interface{}
	Actual   interface{}
}

func (e *PixelDataError) Error() string {
	return fmt.Sprintf("%s: %s (expected: %v, actual: %v)", ErrInvalidPixelData.Error(), e.Field, e.Expected, e.Actual)
}

func (e *PixelDataError) Unwrap() error {
	return ErrInvalidPixelData
}

// MissingAttributeError wraps ErrMissingRequiredAttribute with the attribute name.
type MissingAttributeError struct {
	AttributeName string
	Tag           string
}

func (e *MissingAttributeError) Error() string {
	return fmt.Sprintf("%s: %s (%s)", ErrMissingRequiredAttribute.Error(), e.AttributeName, e.Tag)
}

func (e *MissingAttributeError) Unwrap() error {
	return ErrMissingRequiredAttribute
}
