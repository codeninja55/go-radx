package dicomweb

import (
	"bytes"
	"fmt"
	"io"

	"github.com/codeninja55/go-radx/dicom"
)

// defaultStorageTransferSyntax is the transfer syntax a dataset is encoded in when it
// carries none of its own: Explicit VR Little Endian, the universal uncompressed syntax
// every DICOMweb origin accepts. A STOW-RS application/dicom part is a complete Part 10
// object, so an encoding must always be chosen.
const defaultStorageTransferSyntax = dicom.ExplicitVRLittleEndian

// encodeInstance renders ds as a complete Part 10 application/dicom object: preamble,
// File Meta group, and the main dataset in ts. The File Meta SOP Class / SOP Instance
// UIDs are derived from the dataset's (0008,0016) and (0008,0018); a dataset missing
// either is rejected, because a STOW-RS part with no SOP identity cannot be referenced
// in the store response (PRD §9.2).
func encodeInstance(ds *dicom.DataSet, ts dicom.TransferSyntax) ([]byte, error) {
	if err := requireSOPIdentity(ds); err != nil {
		return nil, err
	}
	sopClass, _ := ds.GetString(dicom.TagSOPClassUID)
	sopInstance, _ := ds.GetString(dicom.TagSOPInstanceUID)
	if ts == "" {
		ts = defaultStorageTransferSyntax
	}
	f := &dicom.File{
		Meta: &dicom.FileMeta{
			MediaStorageSOPClassUID:    dicom.SOPClassUID(sopClass),
			MediaStorageSOPInstanceUID: dicom.SOPInstanceUID(sopInstance),
			TransferSyntaxUID:          ts,
		},
		DataSet: ds,
	}
	var buf bytes.Buffer
	if err := dicom.Write(&buf, f); err != nil {
		return nil, fmt.Errorf("dicomweb: encode application/dicom part: %w", err)
	}
	return buf.Bytes(), nil
}

// requireSOPIdentity rejects a dataset that lacks the SOP Class UID (0008,0016) or SOP
// Instance UID (0008,0018) a STOW-RS store response must reference. An instance with no
// SOP identity cannot be reported as accepted or failed, so it is rejected as an invalid
// resource (PRD §9.2). It carries no patient value.
func requireSOPIdentity(ds *dicom.DataSet) error {
	if ds == nil {
		return fmt.Errorf("%w: instance is nil", ErrInvalidResource)
	}
	if v, ok := ds.GetString(dicom.TagSOPClassUID); !ok || v == "" {
		return fmt.Errorf("%w: instance has no SOP Class UID (0008,0016)", ErrInvalidResource)
	}
	if v, ok := ds.GetString(dicom.TagSOPInstanceUID); !ok || v == "" {
		return fmt.Errorf("%w: instance has no SOP Instance UID (0008,0018)", ErrInvalidResource)
	}
	return nil
}

// decodeInstance parses a complete Part 10 application/dicom object from r into its main
// dataset. A body that ends mid-object surfaces as the dicom reader's error; the
// multipart layer has already converted a short part into a typed TruncatedError before
// these bytes are read.
func decodeInstance(r io.Reader) (*dicom.DataSet, error) {
	f, err := dicom.Read(r)
	if err != nil {
		return nil, fmt.Errorf("dicomweb: decode application/dicom part: %w", err)
	}
	return f.DataSet, nil
}
