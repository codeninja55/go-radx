// Package dicom provides DICOM file parsing implementation.
package dicom

import (
	"compress/flate"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/value"
)

// Parser handles parsing of DICOM files.
//
// The parser reads DICOM files according to DICOM Part 10 File Format:
// 1. 128-byte preamble
// 2. "DICM" prefix (4 bytes)
// 3. File Meta Information (Group 0x0002, always Explicit VR Little Endian)
// 4. Dataset (encoding per Transfer Syntax UID)
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part10.html#sect_7
type Parser struct {
	reader       *Reader
	rawReader    io.Reader // Original io.Reader for decompression wrapping
	ts           *TransferSyntax
	bufferedElem *element.Element // Element read ahead during File Meta parsing
}

// ParseFile reads and parses a DICOM file from the filesystem.
//
// This is the main entry point for parsing DICOM files. It handles:
//   - Reading the file preamble and validating the DICM prefix
//   - Parsing File Meta Information to determine transfer syntax
//   - Parsing the main dataset with the appropriate encoding
//
// Returns a DataSet containing all parsed DICOM elements, or an error if parsing fails.
//
// Example:
//
//	ds, err := dicom.ParseFile("image.dcm")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Parsed %d elements\n", ds.Len())
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part10.html#sect_7
func ParseFile(path string) (*DataSet, error) {
	// Open file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Parse from reader
	return ParseReader(file)
}

// ParseReader reads and parses a DICOM file from an io.Reader.
//
// This allows parsing DICOM data from any source (files, network, memory, etc.).
// The reader must provide a complete DICOM file starting with the preamble.
//
// Returns a DataSet containing all parsed DICOM elements, or an error if parsing fails.
//
// Example:
//
//	file, _ := os.Open("image.dcm")
//	defer file.Close()
//	ds, err := dicom.ParseReader(file)
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part10.html#sect_7
func ParseReader(r io.Reader) (*DataSet, error) {
	// Create binary reader (File Meta is always Little Endian)
	reader := NewReader(r, binary.LittleEndian)

	// Create parser, keeping reference to original reader for potential decompression
	parser := &Parser{
		reader:    reader,
		rawReader: r,
	}

	// Step 1: Read and validate preamble + "DICM" prefix
	if err := parser.readPreamble(); err != nil {
		return nil, fmt.Errorf("invalid DICOM file: %w", err)
	}

	// Step 2: Read File Meta Information (Group 0x0002)
	metaInfo, err := parser.readFileMetaInformation()
	if err != nil {
		return nil, fmt.Errorf("failed to read File Meta Information: %w", err)
	}

	// Step 3: Detect and configure transfer syntax
	ts, err := parser.detectTransferSyntax(metaInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to detect transfer syntax: %w", err)
	}
	parser.ts = ts

	// Update reader byte order for main dataset
	parser.reader.SetByteOrder(ts.ByteOrder)

	// Step 3.5: Handle deflated transfer syntax
	// If the dataset is deflated, wrap the reader in a DEFLATE decompressor.
	// The File Meta Information is never compressed, so we apply decompression
	// only to the main dataset which follows. At this point, rawReader is positioned
	// right at the start of the compressed data.
	//
	// DICOM uses raw DEFLATE (RFC 1951) compression, not zlib format (RFC 1950).
	if ts.Deflated {
		// Create a flate reader that decompresses from the current position
		// flate.NewReader returns io.ReadCloser for raw DEFLATE streams
		flateReader := flate.NewReader(parser.rawReader)
		defer flateReader.Close()

		// Create a new Reader wrapping the decompressed stream
		// Keep the same byte order that was configured for the dataset
		decompressedReader := NewReader(flateReader, ts.ByteOrder)
		parser.reader = decompressedReader
	}

	// Step 4: Read main dataset
	mainDS, err := parser.readDataset()
	if err != nil {
		return nil, fmt.Errorf("failed to read dataset: %w", err)
	}

	// Merge File Meta and main dataset
	// Add all File Meta elements first
	for _, elem := range metaInfo.Elements() {
		mainDS.Add(elem)
	}

	return mainDS, nil
}

// readPreamble reads and validates the 128-byte preamble and "DICM" prefix.
//
// A valid DICOM file must:
//   - Start with exactly 128 bytes (preamble content is not validated)
//   - Followed by the ASCII string "DICM" (4 bytes)
//
// The preamble content is not specified by the standard and may contain
// application-specific data or be all null bytes.
//
// Returns ErrInvalidPreamble if the prefix is not "DICM" or if the file is truncated.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part10.html#sect_7.1
func (p *Parser) readPreamble() error {
	// Read 128-byte preamble (content doesn't matter, just skip it)
	_, err := p.reader.ReadBytes(128)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return fmt.Errorf("%w: file truncated before DICM prefix", ErrInvalidPreamble)
		}
		return fmt.Errorf("%w: failed to read preamble: %v", ErrInvalidPreamble, err)
	}

	// Read 4-byte DICM prefix
	prefix, err := p.reader.ReadString(4)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return fmt.Errorf("%w: file truncated at DICM prefix", ErrInvalidPreamble)
		}
		return fmt.Errorf("%w: failed to read DICM prefix: %v", ErrInvalidPreamble, err)
	}

	// Validate prefix is exactly "DICM"
	if prefix != "DICM" {
		return fmt.Errorf("%w: expected 'DICM', got %q", ErrInvalidPreamble, prefix)
	}

	return nil
}

// readFileMetaInformation reads the File Meta Information (Group 0x0002).
//
// File Meta Information is always encoded as Explicit VR Little Endian,
// regardless of the transfer syntax used for the main dataset.
//
// It contains critical metadata including:
//   - (0002,0000) File Meta Information Group Length
//   - (0002,0010) Transfer Syntax UID (required)
//   - Other metadata like Media Storage SOP Class UID, etc.
//
// Returns a DataSet containing all File Meta Information elements.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part10.html#sect_7.1
func (p *Parser) readFileMetaInformation() (*DataSet, error) {
	// File Meta is always Explicit VR Little Endian
	fileMetaTS := &TransferSyntax{
		ExplicitVR: true,
		ByteOrder:  binary.LittleEndian,
	}

	// Create element parser for File Meta
	elemParser := NewElementParser(p.reader, fileMetaTS)

	// Create dataset to store File Meta elements
	ds := NewDataSet()

	// Read first element which should be File Meta Information Group Length (0002,0000)
	firstElem, err := elemParser.ReadElement()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("unexpected EOF while reading File Meta Information")
		}
		return nil, fmt.Errorf("failed to read first File Meta element: %w", err)
	}

	ds.Add(firstElem)

	// Check if this is the Group Length element
	groupLengthTag := tag.New(0x0002, 0x0000)
	var fileMetaLength uint32
	hasGroupLength := false

	if firstElem.Tag().Equals(groupLengthTag) {
		// Extract group length value (should be UL - uint32)
		// Type assert to IntValue to access Ints() method
		if intVal, ok := firstElem.Value().(*value.IntValue); ok {
			intVals := intVal.Ints()
			if len(intVals) > 0 {
				fileMetaLength = uint32(intVals[0])
				hasGroupLength = true
			}
		}
	}

	// If we have a group length, use it to determine when to stop
	if hasGroupLength && fileMetaLength > 0 {
		// Track bytes read after the group length element
		// We need to read exactly fileMetaLength bytes
		bytesRead := uint32(0)
		startPos := p.reader.Position()

		for bytesRead < fileMetaLength {
			elem, err := elemParser.ReadElement()
			if err != nil {
				if err == io.EOF {
					// Unexpected EOF before reaching group length
					break
				}
				return nil, fmt.Errorf("failed to read File Meta element: %w", err)
			}

			ds.Add(elem)

			// Update bytes read
			currentPos := p.reader.Position()
			bytesRead = uint32(currentPos - startPos)
		}
	} else {
		// Fallback: read until we hit a tag outside Group 0x0002
		for {
			elem, err := elemParser.ReadElement()
			if err != nil {
				if err == io.EOF {
					// Unexpected EOF in File Meta Information
					return nil, fmt.Errorf("unexpected EOF while reading File Meta Information")
				}
				return nil, fmt.Errorf("failed to read File Meta element: %w", err)
			}

			// Check if we've moved past Group 0x0002
			if elem.Tag().Group != 0x0002 {
				// This element belongs to the main dataset, not File Meta
				// Buffer it for the main dataset parser to process
				p.bufferedElem = elem
				break
			}

			// Add element to dataset
			ds.Add(elem)
		}
	}

	return ds, nil
}

// detectTransferSyntax extracts the Transfer Syntax UID from File Meta Information
// and returns the corresponding TransferSyntax configuration.
//
// The Transfer Syntax UID (0002,0010) is required in File Meta Information and
// determines how the main dataset is encoded.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#chapter_10
func (p *Parser) detectTransferSyntax(metaInfo *DataSet) (*TransferSyntax, error) {
	// Get Transfer Syntax UID element (0002,0010)
	tsTag := tag.New(0x0002, 0x0010)
	elem, err := metaInfo.Get(tsTag)
	if err != nil {
		return nil, fmt.Errorf("%w: Transfer Syntax UID not found in File Meta Information", ErrMissingTransferSyntax)
	}

	// Extract UID string
	tsUID := elem.Value().String()
	if tsUID == "" {
		return nil, fmt.Errorf("%w: Transfer Syntax UID is empty", ErrMissingTransferSyntax)
	}

	// Map UID to TransferSyntax properties
	// For now, support the most common transfer syntaxes
	switch tsUID {
	case "1.2.840.10008.1.2":
		// Implicit VR Little Endian
		return &TransferSyntax{
			UID:        tsUID,
			ExplicitVR: false,
			ByteOrder:  binary.LittleEndian,
			Compressed: false,
			Deflated:   false,
		}, nil

	case "1.2.840.10008.1.2.1":
		// Explicit VR Little Endian (default)
		return &TransferSyntax{
			UID:        tsUID,
			ExplicitVR: true,
			ByteOrder:  binary.LittleEndian,
			Compressed: false,
			Deflated:   false,
		}, nil

	case "1.2.840.10008.1.2.2":
		// Explicit VR Big Endian (RETIRED)
		return &TransferSyntax{
			UID:        tsUID,
			ExplicitVR: true,
			ByteOrder:  binary.BigEndian,
			Compressed: false,
			Deflated:   false,
		}, nil

	case "1.2.840.10008.1.2.1.99":
		// Deflated Explicit VR Little Endian
		return &TransferSyntax{
			UID:        tsUID,
			ExplicitVR: true,
			ByteOrder:  binary.LittleEndian,
			Compressed: false,
			Deflated:   true,
		}, nil

	// Compressed transfer syntaxes
	// Pixel data remains as raw bytes until explicitly decompressed via pixel.Extract()

	case "1.2.840.10008.1.2.5":
		// RLE Lossless
		return &TransferSyntax{
			UID:        tsUID,
			ExplicitVR: true,
			ByteOrder:  binary.LittleEndian,
			Compressed: true,
			Deflated:   false,
		}, nil

	case "1.2.840.10008.1.2.4.50":
		// JPEG Baseline (Process 1)
		return &TransferSyntax{
			UID:        tsUID,
			ExplicitVR: true,
			ByteOrder:  binary.LittleEndian,
			Compressed: true,
			Deflated:   false,
		}, nil

	case "1.2.840.10008.1.2.4.51":
		// JPEG Baseline (Processes 2 & 4)
		return &TransferSyntax{
			UID:        tsUID,
			ExplicitVR: true,
			ByteOrder:  binary.LittleEndian,
			Compressed: true,
			Deflated:   false,
		}, nil

	case "1.2.840.10008.1.2.4.57":
		// JPEG Lossless, Non-Hierarchical, First-Order Prediction
		return &TransferSyntax{
			UID:        tsUID,
			ExplicitVR: true,
			ByteOrder:  binary.LittleEndian,
			Compressed: true,
			Deflated:   false,
		}, nil

	case "1.2.840.10008.1.2.4.70":
		// JPEG Lossless, Non-Hierarchical (Process 14)
		return &TransferSyntax{
			UID:        tsUID,
			ExplicitVR: true,
			ByteOrder:  binary.LittleEndian,
			Compressed: true,
			Deflated:   false,
		}, nil

	case "1.2.840.10008.1.2.4.90":
		// JPEG 2000 Image Compression (Lossless Only)
		return &TransferSyntax{
			UID:        tsUID,
			ExplicitVR: true,
			ByteOrder:  binary.LittleEndian,
			Compressed: true,
			Deflated:   false,
		}, nil

	case "1.2.840.10008.1.2.4.91":
		// JPEG 2000 Image Compression
		return &TransferSyntax{
			UID:        tsUID,
			ExplicitVR: true,
			ByteOrder:  binary.LittleEndian,
			Compressed: true,
			Deflated:   false,
		}, nil

	case "1.2.840.10008.1.2.4.201":
		// High-Throughput JPEG 2000 (HTJ2K) Lossless Only
		return &TransferSyntax{
			UID:        tsUID,
			ExplicitVR: true,
			ByteOrder:  binary.LittleEndian,
			Compressed: true,
			Deflated:   false,
		}, nil

	case "1.2.840.10008.1.2.4.203":
		// High-Throughput JPEG 2000 (HTJ2K) Lossless or Lossy
		return &TransferSyntax{
			UID:        tsUID,
			ExplicitVR: true,
			ByteOrder:  binary.LittleEndian,
			Compressed: true,
			Deflated:   false,
		}, nil

	default:
		// Unknown transfer syntax - return error
		return nil, fmt.Errorf("%w: Transfer Syntax UID %q not supported", ErrInvalidTransferSyntax, tsUID)
	}
}

// readDataset reads the main dataset elements using the detected transfer syntax.
//
// The main dataset follows the File Meta Information and uses the encoding
// specified by the Transfer Syntax UID.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part10.html#sect_7.1
func (p *Parser) readDataset() (*DataSet, error) {
	// Create element parser with detected transfer syntax
	elemParser := NewElementParser(p.reader, p.ts)

	// Create dataset to store elements
	ds := NewDataSet()

	// If we have a buffered element from File Meta parsing, add it first
	if p.bufferedElem != nil {
		ds.Add(p.bufferedElem)
		p.bufferedElem = nil
	}

	// Read elements until EOF
	for {
		elem, err := elemParser.ReadElement()
		if err != nil {
			if err == io.EOF {
				// Normal end of file
				break
			}
			// Check if this is an EOF wrapped in other errors (e.g., from sequence parsing)
			// In that case, treat it as end of dataset rather than failure
			if errors.Is(err, io.EOF) {
				// EOF encountered during parsing (e.g., in sequence skipping)
				// This might indicate a truncated file, but we can return what we've parsed so far
				break
			}
			return nil, fmt.Errorf("failed to read dataset element: %w", err)
		}

		// Add element to dataset
		ds.Add(elem)
	}

	return ds, nil
}

// TransferSyntax describes the encoding of a DICOM dataset.
// TODO: Move to transfer_syntax.go once implemented
type TransferSyntax struct {
	UID        string           // Transfer Syntax UID
	ExplicitVR bool             // true = Explicit VR, false = Implicit VR
	ByteOrder  binary.ByteOrder // Little or Big Endian
	Compressed bool             // true if pixel data is compressed
	Deflated   bool             // true for deflated transfer syntax
}
