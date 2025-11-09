// Package dicom provides DICOM file parsing and manipulation.
//
// This package implements a DICOM file parser following the DICOM standard Part 10.
// https://dicom.nema.org/medical/dicom/current/output/html/part10.html
package dicom

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Reader wraps an io.Reader and provides DICOM-specific binary reading operations.
// It supports both Little Endian and Big Endian byte ordering, which can be changed
// dynamically during parsing.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.3
type Reader struct {
	r         io.Reader
	byteOrder binary.ByteOrder
	position  int64 // Track bytes read for position tracking
}

// NewReader creates a new DICOM binary reader with the specified byte order.
//
// Parameters:
//   - r: The underlying io.Reader to read from
//   - byteOrder: The byte order to use (binary.LittleEndian or binary.BigEndian)
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.3
func NewReader(r io.Reader, byteOrder binary.ByteOrder) *Reader {
	return &Reader{
		r:         r,
		byteOrder: byteOrder,
	}
}

// ReadUint16 reads a 16-bit unsigned integer using the current byte order.
//
// Returns io.EOF if the end of the stream is reached.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.1.2
func (r *Reader) ReadUint16() (uint16, error) {
	buf := make([]byte, 2)
	n, err := io.ReadFull(r.r, buf)
	if err != nil {
		if err == io.EOF && n == 0 {
			return 0, io.EOF
		}
		if err == io.ErrUnexpectedEOF || (err == io.EOF && n > 0) {
			return 0, io.ErrUnexpectedEOF
		}
		return 0, fmt.Errorf("failed to read uint16: %w", err)
	}

	r.position += 2
	return r.byteOrder.Uint16(buf), nil
}

// ReadUint32 reads a 32-bit unsigned integer using the current byte order.
//
// Returns io.EOF if the end of the stream is reached.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.1.2
func (r *Reader) ReadUint32() (uint32, error) {
	buf := make([]byte, 4)
	n, err := io.ReadFull(r.r, buf)
	if err != nil {
		if err == io.EOF && n == 0 {
			return 0, io.EOF
		}
		if err == io.ErrUnexpectedEOF || (err == io.EOF && n > 0) {
			return 0, io.ErrUnexpectedEOF
		}
		return 0, fmt.Errorf("failed to read uint32: %w", err)
	}

	r.position += 4
	return r.byteOrder.Uint32(buf), nil
}

// ReadBytes reads exactly n bytes from the reader.
//
// Returns an error if fewer than n bytes are available.
// Returns an empty slice if n is 0.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_7.1.2
func (r *Reader) ReadBytes(n int) ([]byte, error) {
	if n == 0 {
		return []byte{}, nil
	}

	buf := make([]byte, n)
	read, err := io.ReadFull(r.r, buf)
	if err != nil {
		if err == io.EOF && read == 0 {
			return nil, io.EOF
		}
		if err == io.ErrUnexpectedEOF || (err == io.EOF && read > 0) {
			return nil, io.ErrUnexpectedEOF
		}
		return nil, fmt.Errorf("failed to read %d bytes: %w", n, err)
	}

	r.position += int64(n)
	return buf, nil
}

// ReadString reads exactly n bytes and returns them as a string.
//
// DICOM strings may contain null terminators or trailing spaces which are preserved.
// The caller is responsible for trimming if needed.
//
// Returns an error if fewer than n bytes are available.
// Returns an empty string if n is 0.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_6.2
func (r *Reader) ReadString(n int) (string, error) {
	if n == 0 {
		return "", nil
	}

	buf, err := r.ReadBytes(n)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

// SetByteOrder changes the byte order for subsequent read operations.
//
// This is used when switching between File Meta Information (always Little Endian)
// and the main dataset (which may use Big Endian depending on Transfer Syntax).
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part10.html#sect_7.1
func (r *Reader) SetByteOrder(order binary.ByteOrder) {
	r.byteOrder = order
}

// Position returns the current byte position in the stream.
//
// This tracks the total number of bytes read from the underlying reader,
// which is useful for parsing operations that need to know byte offsets.
func (r *Reader) Position() int64 {
	return r.position
}

// WrapReader replaces the underlying reader with a new one.
//
// This is used for applying transformations to the reader stream,
// such as wrapping it in a decompression reader for deflated transfer syntax.
// The position counter is preserved to maintain accurate position tracking
// relative to the original stream.
//
// Parameters:
//   - newReader: The new io.Reader to use for subsequent read operations
func (r *Reader) WrapReader(newReader io.Reader) {
	r.r = newReader
}
