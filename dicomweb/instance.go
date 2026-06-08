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

// encodeRetrievedInstance renders a retrieved instance as a Part-10 application/dicom object
// for a WADO-RS response, honouring the transfer-syntax decision. The decision is taken to be
// acceptable; the caller has already answered 406 otherwise.
//
// A passthrough of a syntax go-radx cannot write (every encapsulated/compressed syntax) is
// served byte-exact from the instance's stored Encoded bytes, never re-encoded through
// dicom.Write, which emits only the four uncompressed syntaxes. When such a passthrough has no
// stored bytes the instance has no writable representation, reported as ErrNotAcceptable so the
// caller answers 406 rather than 500 from a doomed re-encode. A writable syntax (passthrough or
// transcode) is encoded from the DataSet as before.
func encodeRetrievedInstance(si RetrievedInstance, decision transferSyntaxDecision) ([]byte, error) {
	if decision.passthrough && decision.syntax.IsEncapsulated() {
		if len(si.Encoded) == 0 {
			return nil, fmt.Errorf("%w: stored instance is in encapsulated transfer syntax %s with no byte-exact "+
				"representation to pass through, and go-radx transcodes no pixel data in this slice",
				ErrNotAcceptable, decision.syntax.Name())
		}
		return si.Encoded, nil
	}
	return encodeInstance(si.DataSet, decision.syntax)
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

// decodeRetrievedInstance parses a Part 10 application/dicom object into a RetrievedInstance.
//
// captureBytes selects between the two retrieve paths and their memory profiles. When true (the
// object-returning variants — RetrieveInstanceObject, RetrieveSeriesObjects, RetrieveStudyObjects),
// the reader is teed into a buffer so Encoded holds the byte-exact Part 10 object and TransferSyntax
// reports the origin's syntax: the caller can write the object back unchanged rather than re-encoding
// (which would silently transcode), which matters for an encapsulated syntax go-radx cannot re-encode
// from a dataset. When false (the dataset-only variants — RetrieveInstance, RetrieveSeries,
// RetrieveStudy), no buffer is allocated and Encoded stays nil: the common path streams the part
// straight into the decoder, so a large study does not pay the doubled per-instance memory of teeing
// bytes the caller is about to discard.
//
// An encapsulated (compressed) transfer syntax is a fail-closed parse error on either path: go-radx
// reads only the four uncompressed syntaxes, so a compressed object cannot be decoded faithfully —
// the honest outcome is the reader's error, never a corrupted uncompressed file. A body that ends
// mid-object surfaces the same way (the multipart layer has already typed a short part as a
// TruncatedError).
func decodeRetrievedInstance(r io.Reader, captureBytes bool) (RetrievedInstance, error) {
	src := r
	var raw bytes.Buffer
	if captureBytes {
		src = io.TeeReader(r, &raw)
	}
	f, err := dicom.Read(src)
	if err != nil {
		return RetrievedInstance{}, fmt.Errorf("dicomweb: decode application/dicom part: %w", err)
	}
	si := RetrievedInstance{DataSet: f.DataSet}
	if captureBytes {
		si.Encoded = raw.Bytes()
	}
	if f.Meta != nil {
		si.TransferSyntax = f.Meta.TransferSyntaxUID
	}
	return si, nil
}
